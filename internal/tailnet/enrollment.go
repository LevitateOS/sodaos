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

var authKeyPattern = regexp.MustCompile(`^tskey-auth-[A-Za-z0-9_-]{8,512}$`) // slop-audit-allow: production validation pattern for real Tailscale-shaped auth keys

// RunTarget is resolved by the host's native incarnation validator, never decoded
// from a browser request. Run hashes boot, process and user/network namespace IDs.
type RunTarget struct{ Project, Container, Run string }

func (t RunTarget) valid() bool {
	return ValidProject(t.Project) && containerPattern.MatchString(t.Container) && containerPattern.MatchString(t.Run)
}

// keyTransport admits exactly one key POST, with an operation-owned bearer. The
// SDK never sees a reusable credential, redirect, uncapped body or raw error body.
// No SDK Auth wrapper/background token refresh and no automatic POST replay.
type keyTransport struct {
	base        http.RoundTripper
	path, token string
	used        bool
}

func validKeyURL(u *url.URL, path string) bool {
	return u.Scheme == "https" && u.Host == "api.tailscale.com" && u.User == nil && u.Path == path && u.RawQuery == "" && u.Fragment == "" && u.RawPath == ""
}

func validKeyRequest(t *keyTransport, r *http.Request) bool {
	return !t.used && r.Context().Err() == nil && r.Method == "POST" && validKeyURL(r.URL, t.path)
}

func keyResponseBody(res *http.Response) ([]byte, error) {
	if res == nil || res.Body == nil {
		return nil, ErrUnconfirmed
	}
	if res.StatusCode != 200 {
		res.Body.Close()
		return nil, ErrUnconfirmed
	}
	defer res.Body.Close()
	b, e := io.ReadAll(io.LimitReader(res.Body, responseLimit+1))
	if e != nil || len(b) > responseLimit {
		return nil, ErrUnconfirmed
	}
	return b, nil
}

func validKeyCreateCapabilities(b []byte) error {
	var raw map[string]json.RawMessage
	if nativeObject(b, &raw, "id", "key", "created", "expires", "capabilities") != nil {
		return ErrUnconfirmed
	}
	var caps, devices, create map[string]json.RawMessage
	if nativeObject(raw["capabilities"], &caps, "devices") != nil || nativeObject(caps["devices"], &devices, "create") != nil || nativeObject(devices["create"], &create, "reusable", "ephemeral", "tags", "preauthorized") != nil {
		return ErrUnconfirmed
	}
	for _, k := range []string{"reusable", "ephemeral", "preauthorized"} {
		if string(create[k]) != "true" && string(create[k]) != "false" {
			return ErrUnconfirmed
		}
	}
	return nil
}

func (t *keyTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	if !validKeyRequest(t, r) {
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
	b, e := keyResponseBody(res)
	if e != nil {
		return nil, e
	}
	if e = validKeyCreateCapabilities(b); e != nil {
		return nil, e
	}
	res.Body = io.NopCloser(strings.NewReader(string(b)))
	return res, nil
}

func tokenFromEnrollment(ctx context.Context, provider *http.Client, p enrollmentPolicy, c credential) (*oauth2.Token, error) {
	tokenCtx := context.WithValue(ctx, oauth2.HTTPClient, provider)
	cfg := clientcredentials.Config{ClientID: c.ClientID, ClientSecret: c.Secret, TokenURL: "https://api.tailscale.com/api/v2/oauth/token", Scopes: []string{"auth_keys"}, EndpointParams: url.Values{"tags": {strings.Join(p.Tags, " ")}}, AuthStyle: oauth2.AuthStyleInHeader}
	token, e := cfg.Token(tokenCtx)
	if e != nil || token == nil || token.AccessToken == "" || strings.ContainsAny(token.AccessToken, "\r\n\x00") || !strings.EqualFold(token.TokenType, "Bearer") || !token.Expiry.After(time.Now()) {
		return nil, ErrUnconfirmed
	}
	return token, nil
}

func validAuthKeyIdentity(key *ts.Key) bool {
	return key != nil && authKeyPattern.MatchString(key.Key) && key.ID != "" && len(key.ID) <= 128 && !key.Invalid && key.Revoked.IsZero()
}

func validAuthKeyCapabilities(key *ts.Key, tags []string, preauthorized bool) bool {
	caps := key.Capabilities.Devices.Create
	return !caps.Reusable && caps.Ephemeral && caps.Preauthorized == preauthorized && slices.Equal(caps.Tags, tags)
}

func validAuthKeyLifetime(key *ts.Key, before time.Time) bool {
	return !key.Created.Before(before.Add(-time.Minute)) && !key.Created.After(time.Now().Add(time.Minute)) && key.Expires.After(time.Now()) && !key.Expires.After(before.Add(projectKeyLifetime+time.Minute))
}

func validCreatedAuthKey(key *ts.Key, tags []string, preauthorized bool, before time.Time) bool {
	return validAuthKeyIdentity(key) && validAuthKeyCapabilities(key, tags, preauthorized) && validAuthKeyLifetime(key, before)
}

func (m *Management) createProjectAuthKey(ctx context.Context, p enrollmentPolicy, token string) (string, error) {
	base := m.keyHTTP
	if base == nil {
		base = http.DefaultTransport
	}
	transport := &keyTransport{base: base, path: "/api/v2/tailnet/" + p.Tailnet + "/keys", token: token}
	defer func() { transport.token = "" }()
	client := ts.Client{Tailnet: p.Tailnet, HTTP: &http.Client{Timeout: 10 * time.Second, Transport: transport, CheckRedirect: func(*http.Request, []*http.Request) error { return ErrUnconfirmed }}}
	in := ts.CreateKeyRequest{ExpirySeconds: int64(projectKeyLifetime / time.Second), Description: "Soda ephemeral project run"}
	in.Capabilities.Devices.Create.Ephemeral = true
	in.Capabilities.Devices.Create.Tags = slices.Clone(p.Tags)
	in.Capabilities.Devices.Create.Preauthorized = p.Preauthorized
	before := time.Now()
	key, e := client.Keys().CreateAuthKey(ctx, in)
	if e != nil || !validCreatedAuthKey(key, p.Tags, p.Preauthorized, before) {
		if key != nil {
			key.Key = ""
		}
		return "", ErrUnconfirmed
	}
	value := key.Key
	key.Key = ""
	return value, nil
}

func (m *Management) projectKey(ctx context.Context, p enrollmentPolicy, c credential) (string, error) {
	req := EnrollmentRequest{Action: "save", Revision: p.Revision, Tailnet: p.Tailnet, Tags: p.Tags, Preauthorized: &p.Preauthorized, ClientID: c.ClientID, ClientSecret: c.Secret}
	if req.Validate() != nil {
		return "", ErrUnavailable
	}
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	token, e := tokenFromEnrollment(ctx, m.provider, p, c)
	c.Secret = ""
	req.ClientSecret = ""
	if e != nil {
		return "", e
	}
	defer func() { token.AccessToken = "" }()
	return m.createProjectAuthKey(ctx, p, token.AccessToken)
}

// EnrollRun is a root-native operation, not an HTTP credential/key endpoint.
// validate rechecks the exact incarnation; consume writes only the single-use key
// into the validated companion input and invokes its fixed CLI. Neither callback
// receives the OAuth secret/token. The existing policy lock fences rotation/Off;
// this must run outside the host's global Create/lifecycle admission gate.
// The native caller holds its runtime lock and observes the actual daemon before
// calling: an existing node or unfinished exec is not a new enrollment. A failed
// operation leaves no permanent attempt veto; a later explicit retry is permitted.
func admitEnrollRun(policy enrollmentPolicy, project projectPolicy, validate func(context.Context) error, ctx context.Context) error {
	if !project.Enabled || !policy.Admission || project.Binding != policy.Binding || policy.Revision == "0" {
		return ErrConflict
	}
	if validate(ctx) != nil || ctx.Err() != nil {
		return ErrConflict
	}
	return nil
}

func consumeEnrollKey(ctx context.Context, key string, validate func(context.Context) error, consume func(context.Context, string) error) error {
	if validate(ctx) != nil || ctx.Err() != nil {
		return ErrUnconfirmed
	}
	return consume(ctx, key)
}

// EnrollRun is a root-native operation, not an HTTP credential/key endpoint.
// validate rechecks the exact incarnation; consume writes only the single-use key
// into the validated companion input and invokes its fixed CLI. Neither callback
// receives the OAuth secret/token. The existing policy lock fences rotation/Off;
// this must run outside the host's global Create/lifecycle admission gate.
// The native caller holds its runtime lock and observes the actual daemon before
// calling: an existing node or unfinished exec is not a new enrollment. A failed
// operation leaves no permanent attempt veto; a later explicit retry is permitted.
func confirmEnrollSubmission(ctx context.Context, e error, validate func(context.Context) error) error {
	if e != nil || validate(ctx) != nil || ctx.Err() != nil {
		return ErrUnconfirmed
	}
	return nil
}

func (m *Management) enrollLocked(ctx context.Context, root *os.Root, target RunTarget, validate func(context.Context) error, consume func(context.Context, string) error) error {
	policy, e := m.policy.load(root)
	if e != nil {
		return e
	}
	project, e := m.policy.loadProject(root, target.Project, target.Container)
	if e != nil {
		return e
	}
	if e = admitEnrollRun(policy, project, validate, ctx); e != nil {
		return e
	}
	key, e := m.projectKey(ctx, policy, policy.Credential)
	policy.Credential.Secret = ""
	if e == nil {
		e = consumeEnrollKey(ctx, key, validate, consume)
	}
	key = ""
	return confirmEnrollSubmission(ctx, e, validate) // Submission is not enrollment/approval/reachability confirmation.
}

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
	return m.enrollLocked(ctx, root, target, validate, consume)
}
