package forgejo

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/identity"
)

type nativeFixture struct {
	scope     string
	userID    int64
	verified  bool
	active    bool
	failure   string
	exchanges int
	refreshes int
	verifier  string
}

func fixtureProvider(t *testing.T) (*Provider, *nativeFixture) {
	t.Helper()
	f := &nativeFixture{scope: requiredScopes, userID: 7, verified: true, active: true}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == f.failure {
			w.WriteHeader(502)
			_, _ = fmt.Fprint(w, "private native diagnostic")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/login/oauth/access_token":
			if err := r.ParseForm(); err != nil {
				t.Error(err)
				return
			}
			if r.Form.Get("client_id") != "broker-client" || r.Form.Get("client_secret") != "fixture-secret" {
				t.Error("wrong native application")
			}
			if r.Form.Get("grant_type") == "authorization_code" {
				f.exchanges++
				f.verifier = r.Form.Get("code_verifier")
				if r.Form.Get("code") != "fixture-code" || r.Form.Get("redirect_uri") != "https://forgejo.example.test/-/soda/identity/callback" {
					t.Error("wrong native code exchange")
				}
			} else {
				f.refreshes++
				if r.Form.Get("grant_type") != "refresh_token" || r.Form.Get("refresh_token") != "fixture-refresh" {
					t.Error("wrong native refresh")
				}
			}
			_, _ = fmt.Fprint(w, `{"access_token":"fixture-access","refresh_token":"rotated-refresh","token_type":"bearer","expires_in":3600}`)
		case "/login/oauth/introspect":
			user, secret, ok := r.BasicAuth()
			if !ok || user != "broker-client" || secret != "fixture-secret" {
				t.Error("wrong introspection application")
			}
			if err := r.ParseForm(); err != nil {
				t.Error(err)
				return
			}
			if r.Form.Get("token") != "fixture-access" {
				t.Error("wrong introspected access")
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"active": f.active, "scope": f.scope, "sub": "7", "aud": []string{"broker-client"}})
		case "/api/v1/user":
			if r.Header.Get("Authorization") != "token fixture-access" {
				t.Error("wrong native actor credential")
			}
			_, _ = fmt.Fprintf(w, `{"id":%d,"login":"soda-tester"}`, f.userID)
		case "/api/v1/user/emails":
			_, _ = fmt.Fprintf(w, `[{"email":"soda-tester@example.test","verified":%t,"primary":true}]`, f.verified)
		default:
			t.Error("unexpected native route", r.URL.Path)
			w.WriteHeader(404)
		}
	}))
	t.Cleanup(server.Close)
	p, err := New(Config{Base: server.URL, ClientID: "broker-client", RedirectURL: "https://forgejo.example.test/-/soda/identity/callback"}, "fixture-secret")
	if err != nil {
		t.Fatal(err)
	}
	return p, f
}

func startSession(t *testing.T, p *Provider) (*Session, url.Values) {
	t.Helper()
	enrollment, err := p.Start(t.Context(), 7)
	if err != nil {
		t.Fatal(err)
	}
	s := enrollment.(*Session)
	t.Cleanup(func() { _ = s.Close() })
	u, err := url.Parse(s.Snapshot().VerificationURL)
	if err != nil {
		t.Fatal(err)
	}
	return s, u.Query()
}

func TestEnrollmentBindsNativeOwnerStatePKCEAndOneUse(t *testing.T) {
	p, native := fixtureProvider(t)
	s, query := startSession(t, p)
	if query.Get("scope") != requiredScopes || query.Get("client_id") != "broker-client" || query.Get("code_challenge_method") != "S256" || query.Get("state") == "" {
		t.Fatal("native authorization request incomplete")
	}
	if err := s.Complete(t.Context(), "foreign-state", "fixture-code"); !errors.Is(err, identity.ErrDenied) || native.exchanges != 0 || s.Snapshot().State != "pending" {
		t.Fatal("wrong state consumed or exchanged pending enrollment")
	}
	if err := s.Complete(t.Context(), query.Get("state"), "fixture-code"); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256([]byte(native.verifier))
	if query.Get("code_challenge") != base64.RawURLEncoding.EncodeToString(digest[:]) || len(native.verifier) < 43 {
		t.Fatal("native PKCE exchange mismatch")
	}
	if err := s.Complete(t.Context(), query.Get("state"), "fixture-code"); !errors.Is(err, identity.ErrDenied) || native.exchanges != 1 {
		t.Fatal("authorization code was replayed")
	}
	if snapshot := s.Snapshot(); snapshot.State != "completed" || snapshot.Connection != nil || snapshot.VerificationURL != "" {
		t.Fatal("completion exposed unretained connection or state")
	}
	connection, data, err := s.Finish(t.Context())
	if err != nil || connection.ProviderID != identity.Forgejo || connection.OwnerID != 7 || connection.Email != "soda-tester@example.test" || connection.Plan != "" {
		t.Fatal("verified connection metadata missing")
	}
	credential, err := Decode(data, 7)
	if err != nil || credential.Refresh != "rotated-refresh" || credential.Expiry <= time.Now().Unix() {
		t.Fatal("native credential was not retained")
	}
	if _, _, err := s.Finish(t.Context()); !errors.Is(err, identity.ErrDenied) {
		t.Fatal("finished seed was retained in session")
	}
}

func TestEnrollmentRejectsUnverifiedNativeAuthorityWithoutReplay(t *testing.T) {
	for _, tc := range []struct {
		name, scope, failure string
		userID               int64
		verified, active     bool
	}{
		{"default all", "all", "", 7, true, true},
		{"read-only repository", "read:user read:repository", "", 7, true, true},
		{"implicit user read", "write:user write:repository", "", 7, true, true},
		{"extra permissions", requiredScopes + " write:organization", "", 7, true, true},
		{"foreign owner", requiredScopes, "", 8, true, true},
		{"unverified primary", requiredScopes, "", 7, false, true},
		{"inactive consent", requiredScopes, "", 7, true, false},
		{"native exchange failed", requiredScopes, "/login/oauth/access_token", 7, true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p, native := fixtureProvider(t)
			native.scope, native.failure, native.userID, native.verified, native.active = tc.scope, tc.failure, tc.userID, tc.verified, tc.active
			s, query := startSession(t, p)
			if err := s.Complete(t.Context(), query.Get("state"), "fixture-code"); !errors.Is(err, identity.ErrDenied) {
				t.Fatal("unverified native grant accepted")
			}
			if s.Snapshot().State != "failed" {
				t.Fatal("uncertain enrollment retained")
			}
			calls := native.exchanges
			if err := s.Complete(t.Context(), query.Get("state"), "fixture-code"); !errors.Is(err, identity.ErrDenied) || native.exchanges != calls {
				t.Fatal("failed exchange was replayed")
			}
			if _, data, err := s.Finish(t.Context()); err == nil || data != nil {
				t.Fatal("failed enrollment returned credential")
			}
		})
	}
}

func TestEnrollmentExpiryCancellationAndInvalidOwner(t *testing.T) {
	p, native := fixtureProvider(t)
	if _, err := p.Start(t.Context(), 0); !errors.Is(err, identity.ErrDenied) {
		t.Fatal("invalid owner accepted")
	}
	s, query := startSession(t, p)
	s.expires = time.Now().Add(-time.Second)
	if s.Snapshot().State != "expired" {
		t.Fatal("expired enrollment stayed pending")
	}
	if err := s.Complete(t.Context(), query.Get("state"), "fixture-code"); !errors.Is(err, identity.ErrDenied) || native.exchanges != 0 {
		t.Fatal("expired enrollment exchanged")
	}
	s, query = startSession(t, p)
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	if err := s.Complete(t.Context(), query.Get("state"), "fixture-code"); !errors.Is(err, identity.ErrDenied) || native.exchanges != 0 {
		t.Fatal("canceled enrollment exchanged")
	}
}

func TestCompletedEnrollmentExpiresWithoutPolling(t *testing.T) {
	p, _ := fixtureProvider(t)
	s, query := startSession(t, p)
	if s.timer == nil {
		t.Fatal("enrollment has no expiry timer")
	}
	if err := s.Complete(t.Context(), query.Get("state"), "fixture-code"); err != nil {
		t.Fatal(err)
	}
	s.mu.Lock()
	s.expires = time.Now().Add(-time.Second)
	s.mu.Unlock()
	s.expireEnrollment()
	if s.credential != (Credential{}) || s.nonce != "" || s.verifier != "" || s.timer != nil {
		t.Fatal("expiry callback retained enrollment secrets or timer")
	}
	if _, data, err := s.Finish(t.Context()); !errors.Is(err, identity.ErrDenied) || data != nil {
		t.Fatal("expired completion returned a credential")
	}
	if s.Snapshot().State != "expired" {
		t.Fatal("completed enrollment did not expire")
	}
}

func TestEnrollmentTimerStopsOnRetentionAndClose(t *testing.T) {
	p, _ := fixtureProvider(t)
	s, query := startSession(t, p)
	if err := s.Complete(t.Context(), query.Get("state"), "fixture-code"); err != nil {
		t.Fatal(err)
	}
	timer := s.timer
	if _, _, err := s.Finish(t.Context()); err != nil {
		t.Fatal(err)
	}
	if s.timer != nil || timer.Stop() {
		t.Fatal("retention left expiry timer active")
	}
	s.expires = time.Now().Add(-time.Second)
	s.expireEnrollment()
	if s.Snapshot().State != "completed" {
		t.Fatal("late callback changed retained completion")
	}
	s, _ = startSession(t, p)
	timer = s.timer
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	if s.timer != nil || timer.Stop() || s.nonce != "" || s.verifier != "" {
		t.Fatal("close left pending enrollment active")
	}
	s.expires = time.Now().Add(-time.Second)
	s.expireEnrollment()
	if s.Snapshot().State != "canceled" {
		t.Fatal("late callback changed canceled enrollment")
	}
	s, query = startSession(t, p)
	if err := s.Complete(t.Context(), query.Get("state"), "fixture-code"); err != nil {
		t.Fatal(err)
	}
	timer = s.timer
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	if s.timer != nil || timer.Stop() || s.credential != (Credential{}) {
		t.Fatal("close retained completed credential or timer")
	}
}

func TestRefreshReverifiesScopeOwnerAndEmailWithoutOldSeed(t *testing.T) {
	old, _ := json.Marshal(Credential{Access: "old-access", Refresh: "fixture-refresh", Expiry: 1, UserID: 7, Scopes: requiredScopes})
	p, native := fixtureProvider(t)
	connection, data, err := p.Refresh(t.Context(), 7, old)
	if err != nil || connection.Email != "soda-tester@example.test" || native.refreshes != 1 {
		t.Fatal("native refresh failed")
	}
	credential, err := Decode(data, 7)
	if err != nil || credential.Access != "fixture-access" || credential.Refresh != "rotated-refresh" {
		t.Fatal("rotated native credential missing")
	}
	for _, failure := range []string{"/login/oauth/access_token", "/login/oauth/introspect", "/api/v1/user/emails"} {
		native.failure = failure
		_, data, err := p.Refresh(t.Context(), 7, old)
		if !errors.Is(err, identity.ErrUncertain) || data != nil || strings.Contains(err.Error(), "private native diagnostic") {
			t.Fatal("uncertain renewal returned credential or native diagnostic")
		}
	}
	native.failure = ""
	native.scope = "all"
	if _, data, err := p.Refresh(t.Context(), 7, old); !errors.Is(err, identity.ErrUncertain) || data != nil {
		t.Fatal("renewed broad consent accepted")
	}
	native.scope = requiredScopes
	native.userID = 8
	if _, data, err := p.Refresh(t.Context(), 7, old); !errors.Is(err, identity.ErrUncertain) || data != nil {
		t.Fatal("renewed foreign owner accepted")
	}
}

func TestDecodeRejectsChangedOwnerDuplicateAndUnknownFields(t *testing.T) {
	valid := `{"access":"fixture-access","refresh":"fixture-refresh","expiry":1,"user_id":"7","scopes":"read:user write:repository"}`
	for _, input := range []string{strings.Replace(valid, `"7"`, `"8"`, 1), strings.Replace(valid, `"access":`, `"access":"duplicate","access":`, 1), strings.TrimSuffix(valid, "}") + `,"unexpected":true}`, strings.Replace(valid, requiredScopes, "all", 1)} {
		if _, err := Decode([]byte(input), 7); !errors.Is(err, identity.ErrDenied) {
			t.Fatal("invalid custody data accepted")
		}
	}
	if _, err := Decode([]byte(valid), 7); err != nil {
		t.Fatal(err)
	}
	if _, err := Decode([]byte(valid), 0); !errors.Is(err, identity.ErrDenied) {
		t.Fatal("absent expected owner accepted")
	}
	for _, value := range []string{"", "all", "read:user", "read:user read:user"} {
		if explicitScopes(value) {
			t.Fatal("implicit or duplicate scopes accepted")
		}
	}
	if !explicitScopes("write:repository read:user") {
		t.Fatal("equivalent scope order rejected")
	}
}

func TestConfigRejectsUnsafeIssuerWithoutEchoingPrivateInput(t *testing.T) {
	for _, base := range []string{"", "http://forgejo.example.test", "https://private-input@forgejo.example.test", "https://forgejo.example.test?private-input", "https://forgejo.example.test/#private-input"} {
		_, err := New(Config{Base: base, ClientID: "broker-client", RedirectURL: "https://forgejo.example.test/callback"}, "fixture-secret")
		if err == nil || strings.Contains(err.Error(), "private-input") || strings.Contains(err.Error(), "fixture-secret") {
			t.Fatal("unsafe configuration accepted or reflected in error")
		}
	}
}
