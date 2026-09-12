package tailnet

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"slices"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
	ts "tailscale.com/client/tailscale/v2"
)

const projectKeyLifetime = 5 * time.Minute

var authKeyPattern = regexp.MustCompile(`^tskey-auth-[A-Za-z0-9_-]{8,512}$`)

// RunTarget is resolved by the host's native incarnation validator, never decoded
// from a browser request. Run hashes boot, process and user/network namespace IDs.
type RunTarget struct{ Project, Container, Run string }

func (t RunTarget) valid() bool {
	return ValidProject(t.Project) && containerPattern.MatchString(t.Container) && containerPattern.MatchString(t.Run)
}

type runAttempt struct {
	Version   int    `json:"version"`
	Project   string `json:"project"`
	Container string `json:"container"`
	Run       string `json:"run"`
	Binding   string `json:"binding"`
	Phase     string `json:"phase"`
}

// keyTransport admits exactly one key POST, with an operation-owned bearer. The
// SDK never sees a reusable credential, redirect, uncapped body or raw error body.
// No SDK Auth wrapper/background token refresh and no automatic POST replay.
type keyTransport struct {
	base        http.RoundTripper
	path, token string
	used        bool
}

func (t *keyTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	if t.used || r.Context().Err() != nil || r.Method != "POST" || r.URL.Scheme != "https" || r.URL.Host != "api.tailscale.com" || r.URL.User != nil || r.URL.Path != t.path || r.URL.RawQuery != "" || r.URL.Fragment != "" || r.URL.RawPath != "" {
		return nil, ErrInvalid
	}
	t.used = true
	req := r.Clone(r.Context())
	req.Header = r.Header.Clone()
	req.Header.Set("Authorization", "Bearer "+t.token)
	res, e := t.base.RoundTrip(req)
	if e != nil {
		return nil, ErrUnconfirmed
	}
	if res == nil || res.Body == nil {
		return nil, ErrUnconfirmed
	}
	if res.StatusCode != 200 {
		res.Body.Close()
		return nil, ErrUnconfirmed
	}
	// Reject duplicate/null/incomplete JSON before SDK decoding. Its response reader
	// otherwise accepts zero values for missing capability bits (notably reusable).
	defer res.Body.Close()
	b, e := io.ReadAll(io.LimitReader(res.Body, responseLimit+1))
	if e != nil || len(b) > responseLimit {
		return nil, ErrUnconfirmed
	}
	var raw map[string]json.RawMessage
	if nativeObject(b, &raw, "id", "key", "created", "expires", "capabilities") != nil {
		return nil, ErrUnconfirmed
	}
	var caps, devices, create map[string]json.RawMessage
	if nativeObject(raw["capabilities"], &caps, "devices") != nil || nativeObject(caps["devices"], &devices, "create") != nil || nativeObject(devices["create"], &create, "reusable", "ephemeral", "tags", "preauthorized") != nil {
		return nil, ErrUnconfirmed
	}
	for _, k := range []string{"reusable", "ephemeral", "preauthorized"} {
		if string(create[k]) != "true" && string(create[k]) != "false" {
			return nil, ErrUnconfirmed
		}
	}
	res.Body = io.NopCloser(strings.NewReader(string(b)))
	return res, nil
}

func (m *Management) projectKey(ctx context.Context, p enrollmentPolicy, c credential) (string, error) {
	req := EnrollmentRequest{Action: "save", Revision: p.Revision, Tailnet: p.Tailnet, Tags: p.Tags, Preauthorized: &p.Preauthorized, ClientID: c.ClientID, ClientSecret: c.Secret}
	if req.Validate() != nil {
		return "", ErrUnavailable
	}
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	tokenCtx := context.WithValue(ctx, oauth2.HTTPClient, m.provider)
	cfg := clientcredentials.Config{ClientID: c.ClientID, ClientSecret: c.Secret, TokenURL: "https://api.tailscale.com/api/v2/oauth/token", Scopes: []string{"auth_keys"}, EndpointParams: url.Values{"tags": {strings.Join(p.Tags, " ")}}, AuthStyle: oauth2.AuthStyleInHeader}
	token, e := cfg.Token(tokenCtx)
	cfg.ClientSecret = ""
	c.Secret = ""
	req.ClientSecret = ""
	if e != nil || token == nil || token.AccessToken == "" || strings.ContainsAny(token.AccessToken, "\r\n\x00") || !strings.EqualFold(token.TokenType, "Bearer") || !token.Expiry.After(time.Now()) {
		return "", ErrUnconfirmed
	}
	base := m.keyHTTP
	if base == nil {
		base = http.DefaultTransport
	}
	transport := &keyTransport{base: base, path: "/api/v2/tailnet/" + p.Tailnet + "/keys", token: token.AccessToken}
	defer func() { token.AccessToken = ""; transport.token = "" }()
	client := ts.Client{Tailnet: p.Tailnet, HTTP: &http.Client{Timeout: 10 * time.Second, Transport: transport, CheckRedirect: func(*http.Request, []*http.Request) error { return ErrUnconfirmed }}}
	in := ts.CreateKeyRequest{ExpirySeconds: int64(projectKeyLifetime / time.Second), Description: "Soda ephemeral project run"}
	in.Capabilities.Devices.Create.Ephemeral = true
	in.Capabilities.Devices.Create.Tags = slices.Clone(p.Tags)
	in.Capabilities.Devices.Create.Preauthorized = p.Preauthorized
	before := time.Now()
	key, e := client.Keys().CreateAuthKey(ctx, in)
	if e != nil || key == nil {
		return "", ErrUnconfirmed
	}
	caps := key.Capabilities.Devices.Create
	if !authKeyPattern.MatchString(key.Key) || key.ID == "" || len(key.ID) > 128 || key.Invalid || !key.Revoked.IsZero() || caps.Reusable || !caps.Ephemeral || caps.Preauthorized != p.Preauthorized || !slices.Equal(caps.Tags, p.Tags) || key.Created.Before(before.Add(-time.Minute)) || key.Created.After(time.Now().Add(time.Minute)) || !key.Expires.After(time.Now()) || key.Expires.After(before.Add(projectKeyLifetime+time.Minute)) {
		key.Key = ""
		return "", ErrUnconfirmed
	}
	value := key.Key
	key.Key = ""
	return value, nil
}

// RunAttempt is a passive native observation. It does not create directories,
// repair missing journals, fetch tokens or expose credentials. A same-run durable
// marker without its matching journal is explicitly unconfirmed.
func (m *Management) RunAttempt(ctx context.Context, target RunTarget) (string, error) {
	if !target.valid() {
		return "", ErrInvalid
	}
	root, lock, e := m.policy.lock(ctx, false)
	if errors.Is(e, os.ErrNotExist) {
		return "none", nil
	}
	if e != nil {
		return "", e
	}
	defer root.Close()
	defer lock.Close()
	project, e := m.policy.loadProject(root, target.Project, target.Container)
	if e != nil {
		return "", e
	}
	var attempt runAttempt
	e = m.policy.read(root, "attempt-"+target.Project+"-"+target.Run+".json", &attempt)
	if errors.Is(e, os.ErrNotExist) {
		if project.ActiveRun == target.Run {
			return "unconfirmed", nil
		}
		return "none", nil
	}
	if e != nil {
		return "", e
	}
	if attempt.Version != 1 || attempt.Project != target.Project || attempt.Container != target.Container || attempt.Run != target.Run || attempt.Binding != project.Binding || project.ActiveRun != target.Run {
		return "", ErrConflict
	}
	switch attempt.Phase {
	case "key-requested", "submitted", "unconfirmed":
		return attempt.Phase, nil
	}
	return "", ErrUnavailable
}

// EnrollRun is a root-native operation, not an HTTP credential/key endpoint.
// validate rechecks the exact incarnation; consume writes only the single-use key
// into the validated companion input and invokes its fixed CLI. Neither callback
// receives the OAuth secret/token. The existing policy lock fences rotation/Off;
// this must run outside the host's global Create/lifecycle admission gate.
// Any previous attempt in this run is observed/refused, never replayed. A missing
// journal with a matching durable ActiveRun marker is uncertainty, not first use.
func (m *Management) EnrollRun(ctx context.Context, target RunTarget, validate func(context.Context) error, consume func(context.Context, string) error) error {
	if !target.valid() || validate == nil || consume == nil {
		return ErrInvalid
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	root, lock, e := m.policy.lock(ctx, false)
	if errors.Is(e, os.ErrNotExist) {
		return ErrConflict
	}
	if e != nil {
		return e
	}
	defer root.Close()
	defer lock.Close()
	policy, e := m.policy.load(root)
	if e != nil {
		return e
	}
	project, e := m.policy.loadProject(root, target.Project, target.Container)
	if e != nil {
		return e
	}
	if !project.Enabled || !policy.Admission || project.Binding != policy.Binding || policy.Revision == "0" {
		return ErrConflict
	}
	if project.ActiveRun == target.Run {
		return ErrConflict
	}
	name := "attempt-" + target.Project + "-" + target.Run + ".json"
	if _, e = root.Lstat(name); !errors.Is(e, os.ErrNotExist) {
		return ErrConflict
	}
	if validate(ctx) != nil || ctx.Err() != nil {
		return ErrConflict
	}
	// Publish the durable incarnation marker first: even a missing/lost journal or
	// directory-fsync uncertainty cannot cause a second automatic key request.
	project.ActiveRun = target.Run
	if e = m.policy.publish(root, lock, "project-"+target.Project+".json", project); e != nil {
		return e
	}
	attempt := runAttempt{Version: 1, Project: target.Project, Container: target.Container, Run: target.Run, Binding: project.Binding, Phase: "key-requested"}
	if e = m.policy.publish(root, lock, name, attempt); e != nil {
		return e
	}
	var credential credential
	if m.policy.read(root, "credential-"+policy.Credential+".json", &credential) != nil {
		return ErrUnavailable
	}
	key, e := m.projectKey(ctx, policy, credential)
	credential.Secret = ""
	if e == nil {
		if validate(ctx) != nil || ctx.Err() != nil {
			e = ErrUnconfirmed
		} else {
			e = consume(ctx, key)
		}
	}
	key = ""
	attempt.Phase = "submitted"
	if e != nil {
		attempt.Phase = "unconfirmed"
	}
	if m.policy.publish(root, lock, name, attempt) != nil {
		return ErrUnconfirmed
	}
	if e != nil || validate(ctx) != nil || ctx.Err() != nil {
		return ErrUnconfirmed
	}
	return nil // Submission is not enrollment/approval/reachability confirmation.
}
