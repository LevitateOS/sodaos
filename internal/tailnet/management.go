package tailnet

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/levitateos/sodaos/internal/strictjson"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

// ManagementCLIRelease is the reviewed management protocol baseline, not an
// installer upgrade instruction. Other releases fail closed until reviewed.
const ManagementCLIRelease = "1.102.4"
const hostSocket = "/var/run/tailscale/tailscaled.sock"
const responseLimit = 65536

type Management struct {
	policy   policyStore
	local    *http.Client
	provider *http.Client
	command  func(context.Context, string, ...string) ([]byte, error)
}

func NewManagement() *Management {
	return &Management{policy: policyStore{parent: policyParent, uid: 0}, local: &http.Client{
		Timeout: 8 * time.Second,
		Transport: &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "unix", hostSocket)
		}},
		CheckRedirect: func(*http.Request, []*http.Request) error { return ErrUnavailable },
	}, provider: &http.Client{Timeout: 10 * time.Second, Transport: boundedProviderTransport{http.DefaultTransport}, CheckRedirect: func(*http.Request, []*http.Request) error { return ErrUnavailable }}, command: managementCommand}
}

// A streaming cap errors on overflow, even when a valid JSON prefix fits. Never
// silently truncate a body into an apparently successful provider response.
type boundedBody struct {
	io.ReadCloser
	remaining int64
}

func (b *boundedBody) Read(p []byte) (int, error) {
	if int64(len(p)) > b.remaining+1 {
		p = p[:b.remaining+1]
	}
	n, e := b.ReadCloser.Read(p)
	b.remaining -= int64(n)
	if b.remaining < 0 {
		return 0, ErrUnavailable
	}
	return n, e
}

type boundedProviderTransport struct{ base http.RoundTripper }

func (t boundedProviderTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	if r.URL.Scheme != "https" || r.URL.Host != "api.tailscale.com" || r.URL.User != nil || r.URL.Path != "/api/v2/oauth/token" || r.URL.RawQuery != "" || r.Method != "POST" {
		return nil, ErrInvalid
	}
	response, err := t.base.RoundTrip(r)
	if err != nil {
		return nil, ErrUnavailable
	}
	response.Body = &boundedBody{response.Body, responseLimit}
	if response.StatusCode >= 300 && response.StatusCode < 400 {
		response.Body.Close()
		return nil, ErrUnavailable
	}
	return response, nil
}

// Do not embed bytes.Buffer: its promoted ReadFrom would bypass Write's cap
// when os/exec copies a pipe using io.Copy.
type boundedOutput struct{ buffer bytes.Buffer }

func (b *boundedOutput) Write(p []byte) (int, error) {
	if b.buffer.Len()+len(p) > responseLimit {
		return 0, ErrUnavailable
	}
	return b.buffer.Write(p)
}
func managementCommand(ctx context.Context, path string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, path, args...)
	var out boundedOutput
	cmd.Stdout = &out
	cmd.Stderr = io.Discard
	cmd.WaitDelay = time.Second
	err := cmd.Run()
	if err != nil {
		return nil, ErrUnconfirmed
	}
	return out.buffer.Bytes(), nil
}
func (m *Management) request(ctx context.Context, method, path string, value any) ([]byte, error) {
	var body []byte
	if value != nil {
		body, _ = json.Marshal(value)
	}
	req, err := http.NewRequestWithContext(ctx, method, "http://local-tailscaled.sock/localapi/v0/"+path, bytes.NewReader(body))
	if err != nil {
		return nil, ErrUnavailable
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := m.local.Do(req)
	if err != nil {
		return nil, ErrUnavailable
	}
	defer res.Body.Close()
	data, err := io.ReadAll(io.LimitReader(res.Body, responseLimit+1))
	if err != nil || len(data) > responseLimit || (res.StatusCode != 200 && res.StatusCode != 204) {
		return nil, ErrUnavailable
	}
	return data, nil
}

type nativePeer struct {
	ID             string
	DNSName        string
	TailscaleIPs   []string
	Online         bool
	ExitNodeOption bool
	Expired        bool
}
type nativeStatus struct {
	CurrentTailnet *struct {
		Name            string
		MagicDNSEnabled bool
	}
	Version      string
	BackendState string
	HaveNodeKey  bool
	AuthURL      string
	Self         *nativePeer
	Peer         map[string]nativePeer
	Health       []string
}
type nativePrefs struct {
	WantRunning            bool
	ExitNodeID             string
	ExitNodeIP             string
	ExitNodeAllowLANAccess bool
	AdvertiseRoutes        []string
}

func addresses(v []string) ([]string, error) {
	if len(v) > 16 {
		return nil, ErrUnavailable
	}
	out := make([]string, 0, len(v))
	for _, s := range v {
		a, e := netip.ParseAddr(s)
		if e != nil || !a.IsGlobalUnicast() || a.String() != s {
			return nil, ErrUnavailable
		}
		out = append(out, s)
	}
	return out, nil
}
func peerView(p nativePeer) (Peer, error) {
	if len(p.ID) > 128 || strings.ContainsAny(p.ID, "\r\n\x00") {
		return Peer{}, ErrUnavailable
	}
	var name string
	if p.DNSName != "" {
		var e error
		name, e = CanonicalMagicDNSName(p.DNSName)
		if e != nil {
			return Peer{}, ErrUnavailable
		}
	}
	ip, e := addresses(p.TailscaleIPs)
	return Peer{p.ID, name, ip, p.Online, p.ExitNodeOption, p.Expired}, e
}
func nativeObject(data []byte, out any, required ...string) error {
	var fields map[string]json.RawMessage
	if strictjson.Decode(bytes.NewReader(data), &fields) != nil || fields == nil {
		return ErrUnavailable
	}
	for _, key := range required {
		value, ok := fields[key]
		if !ok || (bytes.Equal(value, []byte("null")) && key != "AdvertiseRoutes") {
			return ErrUnavailable
		}
	}
	if json.Unmarshal(data, out) != nil {
		return ErrUnavailable
	}
	return nil
}
func (m *Management) observe(ctx context.Context) (HostView, string, error) {
	var s nativeStatus
	var p nativePrefs
	b, err := m.request(ctx, "GET", "status", nil)
	if err != nil || nativeObject(b, &s, "Version", "BackendState", "HaveNodeKey") != nil || s.BackendState == "" || len(s.Peer) > 128 || len(s.Health) > 128 {
		return HostView{}, "", ErrUnavailable
	}
	if strings.SplitN(s.Version, "-", 2)[0] != ManagementCLIRelease {
		return HostView{}, "", ErrUnsupported
	}
	switch s.BackendState {
	case "NoState", "InUseOtherUser", "NeedsLogin", "NeedsMachineAuth", "Stopped", "Starting", "Running":
	default:
		return HostView{}, "", ErrUnavailable
	}
	if s.BackendState == "Running" && (!s.HaveNodeKey || s.Self == nil || s.Self.ID == "" || len(s.Self.TailscaleIPs) == 0) {
		return HostView{}, "", ErrUnavailable
	}
	b, err = m.request(ctx, "GET", "prefs", nil)
	if err != nil || nativeObject(b, &p, "WantRunning", "ExitNodeID", "ExitNodeIP", "ExitNodeAllowLANAccess", "AdvertiseRoutes") != nil || len(p.AdvertiseRoutes) > 128 || len(p.ExitNodeID) > 128 {
		return HostView{}, "", ErrUnavailable
	}
	if p.ExitNodeIP != "" {
		if _, e := netip.ParseAddr(p.ExitNodeIP); e != nil {
			return HostView{}, "", ErrUnavailable
		}
	}
	view := HostView{State: s.BackendState, HaveNodeKey: s.HaveNodeKey, Peers: []Peer{}, Addresses: []string{}, HealthIssues: len(s.Health), Preferences: HostPreferences{WantRunning: p.WantRunning, ExitNodeID: p.ExitNodeID, ExitNodeIP: p.ExitNodeIP, AllowLAN: p.ExitNodeAllowLANAccess}}
	if s.CurrentTailnet != nil {
		view.Tailnet = s.CurrentTailnet.Name
		view.MagicDNSEnabled = s.CurrentTailnet.MagicDNSEnabled
	}
	for _, route := range p.AdvertiseRoutes {
		if _, e := netip.ParsePrefix(route); e != nil {
			return HostView{}, "", ErrUnavailable
		}
		if route == "0.0.0.0/0" || route == "::/0" {
			view.Preferences.AdvertiseExitNode = true
		}
	}
	if s.Self != nil {
		self, e := peerView(*s.Self)
		if e != nil {
			return HostView{}, "", e
		}
		view.DNSName, view.Addresses, view.Expired = self.DNSName, self.Addresses, self.Expired
	}
	for _, p := range s.Peer {
		peer, e := peerView(p)
		if e != nil {
			return HostView{}, "", e
		}
		view.Peers = append(view.Peers, peer)
	}
	sort.Slice(view.Peers, func(i, j int) bool { return view.Peers[i].ID < view.Peers[j].ID })
	// Revision covers the host's identity/state/preferences, not volatile peer or
	// health polling. Native CLI writers remain outside Soda's advisory lock.
	selfID := ""
	if s.Self != nil {
		selfID = s.Self.ID
	}
	revision, _ := json.Marshal(struct {
		State                string
		HaveNodeKey, Expired bool
		DNSName              string
		ID                   string
		Addresses            []string
		Prefs                nativePrefs
	}{s.BackendState, s.HaveNodeKey, view.Expired, view.DNSName, selfID, view.Addresses, p})
	digest := sha256.Sum256(revision)
	view.Revision = hex.EncodeToString(digest[:])
	if view.Validate() != nil {
		return HostView{}, "", ErrUnavailable
	}
	return view, s.AuthURL, nil
}
func authenticationURL(raw string) string {
	if len(raw) > 2048 {
		return ""
	}
	u, e := url.Parse(raw)
	if e != nil || u.Scheme != "https" || u.Host != "login.tailscale.com" || u.User != nil || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || u.RawPath != "" || !strings.HasPrefix(u.Path, "/a/") || !clientPattern.MatchString(strings.TrimPrefix(u.Path, "/a/")) {
		return ""
	}
	return u.String()
}
func (m *Management) Settings(ctx context.Context) (SettingsView, error) {
	enrollment, err := m.policy.enrollment(ctx)
	if err != nil {
		return SettingsView{}, err
	}
	result := SettingsView{Enrollment: enrollment}
	host, _, err := m.observe(ctx)
	if err != nil {
		result.HostUnavailable = true
	} else {
		result.Host = &host
	}
	return result, nil
}
func (m *Management) HostAction(ctx context.Context, r HostRequest) (HostResult, error) {
	if r.Validate() != nil {
		return HostResult{}, ErrInvalid
	}
	root, lock, err := m.policy.lock(ctx, r.Action != "authentication")
	// Authentication observation doesn't initialize native policy storage.
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return HostResult{}, err
	}
	if root != nil {
		defer root.Close()
		defer lock.Close()
	}
	before, auth, err := m.observe(ctx)
	if err != nil {
		return HostResult{}, err
	}
	if before.Revision != r.Revision {
		return HostResult{}, ErrConflict
	}
	if r.Action == "authentication" {
		return HostResult{Outcome: "observed", Host: &before, AuthURL: authenticationURL(auth)}, nil
	}
	if err = ctx.Err(); err != nil {
		return HostResult{}, ErrUnconfirmed
	}
	if r.Action == "exit-node" || r.Action == "advertise-exit-node" || r.Action == "refresh-forgejo" || (r.Action == "signin" && !before.HaveNodeKey) {
		if err = m.verifyCLI(ctx); err != nil {
			return HostResult{}, err
		}
	}
	selectedExitNodeID := ""
	switch r.Action {
	case "signin":
		if before.HaveNodeKey {
			_, err = m.request(ctx, "PATCH", "prefs", map[string]bool{"WantRunning": true, "WantRunningSet": true})
			if err == nil && (before.State == "NeedsLogin" || before.Expired) {
				_, err = m.request(ctx, "POST", "login-interactive", nil)
			}
		} else {
			var data []byte
			wait, cancel := context.WithTimeout(ctx, 8*time.Second)
			data, err = m.command(wait, DefaultCLI, "--socket="+hostSocket, "up", "--json", "--timeout=5s")
			cancel()
			// Decode complete bounded notifications; never emit native Error text.
			if err == nil {
				d := json.NewDecoder(bytes.NewReader(data))
				for {
					var message struct {
						AuthURL string
						Error   string
					}
					e := d.Decode(&message)
					if e == io.EOF {
						break
					}
					if e != nil || message.Error != "" {
						err = ErrUnconfirmed
						break
					}
				}
			}
		}
	case "logout":
		_, err = m.request(ctx, "POST", "logout", nil)
	case "exit-node":
		if *r.ExitNode != "" {
			available := false
			for _, peer := range before.Peers {
				if peer.ExitNode && peer.Online && !peer.Expired {
					for _, ip := range peer.Addresses {
						if ip == *r.ExitNode {
							available = true
							selectedExitNodeID = peer.ID
						}
					}
				}
			}
			if !available {
				return HostResult{}, ErrConflict
			}
		}
		err = m.run(ctx, "set", "--exit-node="+*r.ExitNode, "--exit-node-allow-lan-access="+boolString(*r.AllowLAN))
	case "advertise-exit-node":
		err = m.run(ctx, "set", "--advertise-exit-node="+boolString(*r.Advertise))
	case "refresh-forgejo":
		_, err = m.command(ctx, "/usr/local/libexec/soda/soda-forgejo-tailnet")
	}
	result := HostResult{Outcome: "confirmed"}
	after, auth, readErr := m.observe(ctx)
	if readErr != nil {
		result.ReadbackUnavailable = true
	} else {
		result.Host = &after
	}
	if err != nil {
		result.Outcome = "unconfirmed"
	}
	if readErr == nil && err == nil {
		switch r.Action {
		case "signin":
			if !after.Preferences.WantRunning {
				result.Outcome = "unconfirmed"
			}
		case "logout":
			if after.State != "NeedsLogin" {
				result.Outcome = "unconfirmed"
			}
		case "exit-node":
			// Native resolveExitNodeIPLocked upgrades the selected IP to its
			// stable ID and clears ExitNodeIP. Accept either native form; an
			// empty requested IP must clear both, never match a retained ID.
			matched := after.Preferences.ExitNodeID == selectedExitNodeID && after.Preferences.ExitNodeIP == ""
			if *r.ExitNode != "" && after.Preferences.ExitNodeIP == *r.ExitNode {
				matched = true
			}
			if !matched || after.Preferences.AllowLAN != *r.AllowLAN {
				result.Outcome = "unconfirmed"
			}
		case "advertise-exit-node":
			if after.Preferences.AdvertiseExitNode != *r.Advertise {
				result.Outcome = "unconfirmed"
			}
		}
	}
	if r.Action == "signin" && readErr == nil {
		result.AuthURL = authenticationURL(auth)
		if result.AuthURL != "" {
			result.Outcome = "pending"
		}
	}
	return result, nil
}
func boolString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
func (m *Management) verifyCLI(ctx context.Context) error {
	data, err := m.command(ctx, DefaultCLI, "version", "--json")
	if err != nil {
		return ErrUnavailable
	}
	var version struct {
		Short string `json:"short"`
	}
	if nativeObject(data, &version, "short") != nil || version.Short != ManagementCLIRelease {
		return ErrUnsupported
	}
	return nil
}
func (m *Management) run(ctx context.Context, args ...string) error {
	_, e := m.command(ctx, DefaultCLI, append([]string{"--socket=" + hostSocket}, args...)...)
	return e
}
func (m *Management) checkCredential(ctx context.Context, r EnrollmentRequest) error {
	ctx = context.WithValue(ctx, oauth2.HTTPClient, m.provider)
	c := clientcredentials.Config{ClientID: r.ClientID, ClientSecret: r.ClientSecret, TokenURL: "https://api.tailscale.com/api/v2/oauth/token", Scopes: []string{"auth_keys"}, EndpointParams: url.Values{"tags": {strings.Join(r.Tags, " ")}}, AuthStyle: oauth2.AuthStyleInHeader}
	token, err := c.Token(ctx)
	if err != nil || token == nil || token.AccessToken == "" || !strings.EqualFold(token.TokenType, "Bearer") || !token.Expiry.After(time.Now()) {
		return ErrUnavailable
	}
	token.AccessToken = ""
	return nil
}
func (m *Management) Enrollment(ctx context.Context, r EnrollmentRequest) (EnrollmentResult, error) {
	return m.policy.update(ctx, r, m.checkCredential)
}
func (m *Management) Options(ctx context.Context) (ProjectOptions, error) {
	v, e := m.policy.enrollment(ctx)
	return ProjectOptions{Revision: v.Revision, Binding: v.Binding, Tailnet: v.Tailnet, Available: false, Default: false}, e
}
func (m *Management) Project(ctx context.Context, r ProjectRequest, cid string) (ProjectView, error) {
	return m.policy.project(ctx, r, cid)
}
