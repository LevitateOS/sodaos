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

	"github.com/levitateos/sodaos/internal/installlayout"
	"github.com/levitateos/sodaos/internal/strictjson"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

const (
	hostSocket    = "/var/run/tailscale/tailscaled.sock"
	responseLimit = 65536
)

type Control struct {
	policy   policyStore
	local    *http.Client
	provider *http.Client
	keyHTTP  http.RoundTripper // operation-owned SDK transport; nil uses the native default
	command  func(context.Context, string, ...string) ([]byte, error)
}

func NewControl() *Control {
	return &Control{policy: policyStore{parent: policyParent, uid: 0}, local: &http.Client{
		Timeout: 8 * time.Second,
		Transport: &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "unix", hostSocket)
		}},
		CheckRedirect: func(*http.Request, []*http.Request) error { return ErrUnavailable },
	}, provider: &http.Client{Timeout: 10 * time.Second, Transport: boundedProviderTransport{http.DefaultTransport}, CheckRedirect: func(*http.Request, []*http.Request) error { return ErrUnavailable }}, command: controlCommand}
}

// NewProjectControl is used only by the root runtime entrypoint/helper when
// its immutable companion image is explicitly configured. No installed default
// turns this on, and construction performs no provider or native operations.
func NewProjectControl() *Control {
	m := NewControl()
	m.policy.runtime = true
	return m
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

func validOAuthTokenRequest(r *http.Request) bool {
	return r.URL.Scheme == "https" && r.URL.Host == "api.tailscale.com" && r.URL.User == nil && r.URL.Path == "/api/v2/oauth/token" && r.URL.RawQuery == "" && r.Method == "POST"
}

func (t boundedProviderTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	if !validOAuthTokenRequest(r) {
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

func controlCommand(ctx context.Context, path string, args ...string) ([]byte, error) {
	return RunNative(ctx, path, args...)
}

// RunNative bounds stdout and discards private native diagnostics. Callers supply
// fixed executables/argument recipes, never a request-selected command or shell.
func RunNative(ctx context.Context, path string, args ...string) ([]byte, error) {
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

func (m *Control) request(ctx context.Context, method, path string, value any) ([]byte, error) {
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

func haveNodeKeyOmitted(fields map[string]json.RawMessage, key string) bool {
	// Tailscale 1.102.4 ipnstate.Status marks HaveNodeKey omitempty:
	// an absent field is false on a fresh, unenrolled daemon. Keep explicit
	// null/type errors and case-aliased fields fail-closed.
	if key != "HaveNodeKey" {
		return false
	}
	for name := range fields {
		if strings.EqualFold(name, key) {
			return false
		}
	}
	return true
}

func requiredNativeField(fields map[string]json.RawMessage, key string) bool {
	value, ok := fields[key]
	if !ok {
		return haveNodeKeyOmitted(fields, key)
	}
	return !bytes.Equal(value, []byte("null")) || key == "AdvertiseRoutes"
}

func nativeObject(data []byte, out any, required ...string) error {
	var fields map[string]json.RawMessage
	if strictjson.Decode(bytes.NewReader(data), &fields) != nil || fields == nil {
		return ErrUnavailable
	}
	for _, key := range required {
		if !requiredNativeField(fields, key) {
			return ErrUnavailable
		}
	}
	if json.Unmarshal(data, out) != nil {
		return ErrUnavailable
	}
	return nil
}

func validateNativeBackendState(s nativeStatus) error {
	switch s.BackendState {
	case "NoState", "InUseOtherUser", "NeedsLogin", "NeedsMachineAuth", "Stopped", "Starting":
		return nil
	case "Running":
		if !s.HaveNodeKey || s.Self == nil || s.Self.ID == "" || len(s.Self.TailscaleIPs) == 0 {
			return ErrUnavailable
		}
		return nil
	default:
		return ErrUnavailable
	}
}

func (m *Control) fetchNativeStatus(ctx context.Context) (nativeStatus, error) {
	var s nativeStatus
	b, err := m.request(ctx, "GET", "status", nil)
	if err != nil || nativeObject(b, &s, "BackendState", "HaveNodeKey") != nil {
		return nativeStatus{}, ErrUnavailable
	}
	if s.BackendState == "" || len(s.Peer) > 128 || len(s.Health) > 128 {
		return nativeStatus{}, ErrUnavailable
	}
	if err := validateNativeBackendState(s); err != nil {
		return nativeStatus{}, err
	}
	return s, nil
}

func (m *Control) fetchNativePrefs(ctx context.Context) (nativePrefs, error) {
	var p nativePrefs
	b, err := m.request(ctx, "GET", "prefs", nil)
	if err != nil || nativeObject(b, &p, "WantRunning", "ExitNodeID", "ExitNodeIP", "ExitNodeAllowLANAccess", "AdvertiseRoutes") != nil {
		return nativePrefs{}, ErrUnavailable
	}
	if len(p.AdvertiseRoutes) > 128 || len(p.ExitNodeID) > 128 {
		return nativePrefs{}, ErrUnavailable
	}
	if p.ExitNodeIP != "" {
		if _, err := netip.ParseAddr(p.ExitNodeIP); err != nil {
			return nativePrefs{}, ErrUnavailable
		}
	}
	return p, nil
}

func populateHostPreferences(p nativePrefs) (HostPreferences, error) {
	prefs := HostPreferences{
		WantRunning: p.WantRunning,
		ExitNodeID:  p.ExitNodeID,
		ExitNodeIP:  p.ExitNodeIP,
		AllowLAN:    p.ExitNodeAllowLANAccess,
	}
	for _, route := range p.AdvertiseRoutes {
		if _, err := netip.ParsePrefix(route); err != nil {
			return HostPreferences{}, ErrUnavailable
		}
		if route == "0.0.0.0/0" || route == "::/0" {
			prefs.AdvertiseExitNode = true
		}
	}
	return prefs, nil
}

func applySelfPeer(view *HostView, self *nativePeer) error {
	if self == nil {
		return nil
	}
	p, err := peerView(*self)
	if err != nil {
		return err
	}
	view.DNSName, view.Addresses, view.Expired = p.DNSName, p.Addresses, p.Expired
	return nil
}

func populatePeers(peers map[string]nativePeer) ([]Peer, error) {
	out := make([]Peer, 0, len(peers))
	for _, p := range peers {
		peer, err := peerView(p)
		if err != nil {
			return nil, err
		}
		out = append(out, peer)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func computeHostRevision(s nativeStatus, view HostView, p nativePrefs) string {
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
	return hex.EncodeToString(digest[:])
}

func (m *Control) observe(ctx context.Context) (HostView, string, error) {
	s, err := m.fetchNativeStatus(ctx)
	if err != nil {
		return HostView{}, "", err
	}
	p, err := m.fetchNativePrefs(ctx)
	if err != nil {
		return HostView{}, "", err
	}
	prefs, err := populateHostPreferences(p)
	if err != nil {
		return HostView{}, "", err
	}
	view := HostView{
		State:        s.BackendState,
		HaveNodeKey:  s.HaveNodeKey,
		Peers:        []Peer{},
		Addresses:    []string{},
		HealthIssues: len(s.Health),
		Preferences:  prefs,
	}
	if s.CurrentTailnet != nil {
		view.Tailnet = s.CurrentTailnet.Name
		view.MagicDNSEnabled = s.CurrentTailnet.MagicDNSEnabled
	}
	if err := applySelfPeer(&view, s.Self); err != nil {
		return HostView{}, "", err
	}
	view.Peers, err = populatePeers(s.Peer)
	if err != nil {
		return HostView{}, "", err
	}
	// Revision covers the host's identity/state/preferences, not volatile peer or
	// health polling. Native CLI writers remain outside Soda's advisory lock.
	view.Revision = computeHostRevision(s, view, p)
	if view.Validate() != nil {
		return HostView{}, "", ErrUnavailable
	}
	return view, s.AuthURL, nil
}

func validTailscaleAuthURL(u *url.URL) bool {
	return u.Scheme == "https" && u.Host == "login.tailscale.com" && u.User == nil && u.RawQuery == "" && !u.ForceQuery && u.Fragment == "" && u.RawPath == "" && strings.HasPrefix(u.Path, "/a/") && clientPattern.MatchString(strings.TrimPrefix(u.Path, "/a/"))
}

func authenticationURL(raw string) string {
	if len(raw) > 2048 {
		return ""
	}
	u, e := url.Parse(raw)
	if e != nil || !validTailscaleAuthURL(u) {
		return ""
	}
	return u.String()
}

func (m *Control) Settings(ctx context.Context) (SettingsView, error) {
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

func (m *Control) HostAction(ctx context.Context, r HostRequest) (HostResult, error) {
	if r.Validate() != nil {
		return HostResult{}, ErrInvalid
	}
	unlock, err := m.lockPolicy(ctx, r.Action != "authentication")
	if err != nil {
		return HostResult{}, err
	}
	if unlock != nil {
		defer unlock()
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
	selectedExitNodeID, err := m.executeHostAction(ctx, r, before)
	if errors.Is(err, ErrConflict) {
		return HostResult{}, ErrConflict
	}
	return m.readbackHostAction(ctx, r, selectedExitNodeID, err)
}

func (m *Control) lockPolicy(ctx context.Context, write bool) (func(), error) {
	root, lock, err := m.policy.lock(ctx, write)
	// Authentication observation doesn't initialize native policy storage.
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	if root == nil {
		return nil, nil
	}
	return func() {
		root.Close()
		lock.Close()
	}, nil
}

func (m *Control) executeHostAction(ctx context.Context, r HostRequest, before HostView) (string, error) {
	switch r.Action {
	case "signin":
		return "", m.executeSignin(ctx, before)
	case "logout":
		return "", m.executeLogout(ctx)
	case "exit-node":
		return m.executeExitNode(ctx, r, before)
	case "advertise-exit-node":
		return "", m.executeAdvertiseExitNode(ctx, r)
	case "refresh-forgejo":
		return "", m.executeRefreshForgejo(ctx)
	default:
		return "", nil
	}
}

func (m *Control) executeSignin(ctx context.Context, before HostView) error {
	if before.HaveNodeKey {
		_, err := m.request(ctx, "PATCH", "prefs", map[string]bool{"WantRunning": true, "WantRunningSet": true})
		if err == nil && (before.State == "NeedsLogin" || before.Expired) {
			_, err = m.request(ctx, "POST", "login-interactive", nil)
		}
		return err
	}
	wait, cancel := context.WithTimeout(ctx, 8*time.Second)
	data, err := m.command(wait, DefaultCLI, "--socket="+hostSocket, "up", "--json", "--timeout=5s")
	cancel()
	if err != nil {
		return err
	}
	return decodeUpNotifications(data)
}

// Decode complete bounded notifications; never emit native Error text.
func decodeUpNotifications(data []byte) error {
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
			return ErrUnconfirmed
		}
	}
	return nil
}

func (m *Control) executeLogout(ctx context.Context) error {
	_, err := m.request(ctx, "POST", "logout", nil)
	return err
}

func findAvailableExitNode(peers []Peer, targetIP string) (string, bool) {
	for _, peer := range peers {
		if peer.ExitNode && peer.Online && !peer.Expired {
			for _, ip := range peer.Addresses {
				if ip == targetIP {
					return peer.ID, true
				}
			}
		}
	}
	return "", false
}

func (m *Control) executeExitNode(ctx context.Context, r HostRequest, before HostView) (string, error) {
	selectedExitNodeID := ""
	if *r.ExitNode != "" {
		peerID, ok := findAvailableExitNode(before.Peers, *r.ExitNode)
		if !ok {
			return "", ErrConflict
		}
		selectedExitNodeID = peerID
	}
	err := m.run(ctx, "set", "--exit-node="+*r.ExitNode, "--exit-node-allow-lan-access="+boolString(*r.AllowLAN))
	return selectedExitNodeID, err
}

func (m *Control) executeAdvertiseExitNode(ctx context.Context, r HostRequest) error {
	return m.run(ctx, "set", "--advertise-exit-node="+boolString(*r.Advertise))
}

func (m *Control) executeRefreshForgejo(ctx context.Context) error {
	_, err := m.command(ctx, installlayout.Libexec+"/soda-forgejo-tailnet")
	return err
}

func (m *Control) readbackHostAction(ctx context.Context, r HostRequest, selectedExitNodeID string, actionErr error) (HostResult, error) {
	result := HostResult{Outcome: "confirmed"}
	after, auth, readErr := m.observe(ctx)
	if readErr != nil {
		result.ReadbackUnavailable = true
	} else {
		result.Host = &after
	}
	if actionErr != nil {
		result.Outcome = "unconfirmed"
	}
	if readErr == nil && actionErr == nil {
		if !verifyHostActionOutcome(r, after, selectedExitNodeID) {
			result.Outcome = "unconfirmed"
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

func verifyHostActionOutcome(r HostRequest, after HostView, selectedExitNodeID string) bool {
	switch r.Action {
	case "signin":
		return after.Preferences.WantRunning
	case "logout":
		return after.State == "NeedsLogin"
	case "exit-node":
		return verifyExitNode(r, after, selectedExitNodeID)
	case "advertise-exit-node":
		return after.Preferences.AdvertiseExitNode == *r.Advertise
	default:
		return true
	}
}

func verifyExitNode(r HostRequest, after HostView, selectedExitNodeID string) bool {
	// Native resolveExitNodeIPLocked upgrades the selected IP to its
	// stable ID and clears ExitNodeIP. Accept either native form; an
	// empty requested IP must clear both, never match a retained ID.
	matched := after.Preferences.ExitNodeID == selectedExitNodeID && after.Preferences.ExitNodeIP == ""
	if *r.ExitNode != "" && after.Preferences.ExitNodeIP == *r.ExitNode {
		matched = true
	}
	return matched && after.Preferences.AllowLAN == *r.AllowLAN
}

func boolString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func (m *Control) run(ctx context.Context, args ...string) error {
	_, e := m.command(ctx, DefaultCLI, append([]string{"--socket=" + hostSocket}, args...)...)
	return e
}

func (m *Control) checkCredential(ctx context.Context, r EnrollmentRequest) error {
	ctx = context.WithValue(ctx, oauth2.HTTPClient, m.provider)
	c := clientcredentials.Config{ClientID: r.ClientID, ClientSecret: r.ClientSecret, TokenURL: "https://api.tailscale.com/api/v2/oauth/token", Scopes: []string{"auth_keys"}, EndpointParams: url.Values{"tags": {strings.Join(r.Tags, " ")}}, AuthStyle: oauth2.AuthStyleInHeader}
	token, err := c.Token(ctx)
	if err != nil || token == nil || token.AccessToken == "" || !strings.EqualFold(token.TokenType, "Bearer") || !token.Expiry.After(time.Now()) {
		return ErrUnavailable
	}
	token.AccessToken = ""
	return nil
}

func (m *Control) Enrollment(ctx context.Context, r EnrollmentRequest) (EnrollmentResult, error) {
	return m.policy.update(ctx, r, m.checkCredential)
}

func (m *Control) Options(ctx context.Context) (ProjectOptions, error) {
	v, e := m.policy.enrollment(ctx)
	return ProjectOptions{Revision: v.Revision, Binding: v.Binding, Tailnet: v.Tailnet, Available: v.RuntimeSupported && v.Configured && v.Admission, Default: v.RuntimeSupported && v.Admission && v.Default}, e
}

func (m *Control) Project(ctx context.Context, r ProjectRequest, cid string) (ProjectView, error) {
	return m.policy.project(ctx, r, cid)
}
