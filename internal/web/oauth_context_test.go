package web

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/config"
)

func TestOAuthLoginBindsBoundedContextToPKCEState(t *testing.T) {
	s := apiTestServer(t)
	for _, query := range []string{
		"destination=", "destination=other", "destination=spaces&destination=spaces", "destination=spaces&repository_id=7", "destination=spaces&repository_id=", "destination=https://elsewhere.test",
		"repository_id=0", "repository_id=-1", "repository_id=01", "repository_id=",
		"repository_id=9223372036854775808", "repository_id=1&repository_id=2",
		"expected_user_id=0", "expected_user_id=alice", "expected_user_id=%2b1",
		"expected_user_id=1&expected_user_id=1", "repository_id=%zz",
		"return_to=&return_to=https://evil.example", "repository_id=1;expected_user_id=2",
		"ignored=" + strings.Repeat("a", 8192),
	} {
		w := httptest.NewRecorder()
		s.ServeHTTP(w, httptest.NewRequest("GET", config.SodaPath+"/login?"+query, nil))
		if w.Code != 400 || w.Header().Get("Set-Cookie") != "" || w.Header().Get("Location") != "" {
			t.Fatal("invalid login context accepted", w.Code)
		}
	}
	w := httptest.NewRecorder()
	s.ServeHTTP(w, httptest.NewRequest("GET", config.SodaPath+"/login?repository_id=42&expected_user_id=1", nil))
	location, err := url.Parse(w.Header().Get("Location"))
	if err != nil || w.Code != 302 {
		t.Fatal(w.Code, err)
	}
	query := location.Query()
	if query.Get("repository_id") != "" || query.Get("expected_user_id") != "" || w.Header().Get("Referrer-Policy") != "no-referrer" {
		t.Fatal("context leaked into provider request/referrer")
	}
	pending, err := s.Store.ConsumeOAuth(t.Context(), query.Get("state"), "")
	if err != nil || pending.RepositoryID != 42 || pending.ExpectedUserID != 1 {
		t.Fatal(pending, err)
	}
	challenge := sha256.Sum256([]byte(pending.Verifier))
	if query.Get("code_challenge") != base64.RawURLEncoding.EncodeToString(challenge[:]) {
		t.Fatal("context lost PKCE binding")
	}
}

func TestOAuthRepositoryReturnUsesOnlyStoredIDsAndActingGrant(t *testing.T) {
	for _, tc := range []struct {
		name       string
		context    string
		scopes     string
		repoStatus int
		repo       string
		wantPath   string
		wantCalls  int
	}{
		{"rename and transfer", "repository_id=42&expected_user_id=1", "read:user read:repository", 200, `{"id":42,"name":"renamed","full_name":"stale/ignored","html_url":"https://evil.example/","owner":{"id":7,"login":"current"}}`, "/current/renamed#sodaspaces", 4},
		{"anonymous start", "repository_id=42", "read:user read:repository", 200, `{"id":42,"name":"demo","full_name":"alice/demo","owner":{"id":1,"login":"alice"}}`, "/alice/demo#sodaspaces", 4},
		{"escaped names", "repository_id=42&expected_user_id=1", "read:user read:repository", 200, `{"id":42,"name":"demo?#","full_name":"ignored","owner":{"id":1,"login":"alice"}}`, "/alice/demo%3F%23#sodaspaces", 4},
		{"fixed Spaces", "destination=spaces&expected_user_id=1", "read:user read:repository", 0, "", "/-/soda/spaces", 3},
		{"no context", "", "read:user read:repository", 0, "", "/", 3},
		{"insufficient actual consent", "repository_id=42&expected_user_id=1", "read:user", 0, "", "/", 3},
		{"inaccessible", "repository_id=42&expected_user_id=1", "read:user read:repository", 404, `{}`, "/", 4},
		{"unavailable", "repository_id=42&expected_user_id=1", "read:user read:repository", 503, `{}`, "/", 4},
		{"wrong repository", "repository_id=42&expected_user_id=1", "read:user read:repository", 200, `{"id":43,"name":"demo","full_name":"alice/demo","owner":{"id":1,"login":"alice"}}`, "/", 4},
		{"missing fields", "repository_id=42&expected_user_id=1", "read:user read:repository", 200, `{"id":42}`, "/", 4},
		{"unsafe owner path", "repository_id=42&expected_user_id=1", "read:user read:repository", 200, `{"id":42,"name":"demo","full_name":"ignored","owner":{"id":1,"login":".."}}`, "/", 4},
		{"unsafe repository path", "repository_id=42&expected_user_id=1", "read:user read:repository", 200, `{"id":42,"name":"//evil.example/","full_name":"ignored","owner":{"id":1,"login":"alice"}}`, "/", 4},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			s := grantedTestServer(t, func(w http.ResponseWriter, r *http.Request) {
				calls++
				switch r.URL.Path {
				case "/login/oauth/access_token":
					if err := r.ParseForm(); err != nil || r.Form.Get("redirect_uri") != "https://forgejo.example.test/-/soda/oauth/callback" {
						t.Error("wrong callback binding")
					}
					fmt.Fprint(w, `{"access_token":"callback-access","refresh_token":"callback-refresh","token_type":"bearer","expires_in":3600}`)
				case "/api/v1/user":
					fmt.Fprint(w, `{"id":1,"login":"alice"}`)
				case "/login/oauth/introspect":
					fmt.Fprintf(w, `{"active":true,"scope":%q,"sub":"1","aud":["client"]}`, tc.scopes)
				case "/api/v1/repositories/42":
					if tc.repoStatus == 0 || r.Header.Get("Authorization") != "token callback-access" {
						t.Error("wrong or unnecessary repository request")
						w.WriteHeader(500)
						return
					}
					w.WriteHeader(tc.repoStatus)
					fmt.Fprint(w, tc.repo)
				default:
					t.Error("unexpected provider request", r.URL.Path)
					w.WriteHeader(500)
				}
			})
			start := httptest.NewRecorder()
			s.ServeHTTP(start, apiTestRequest("GET", "/login?"+tc.context, "", "alice"))
			location, err := url.Parse(start.Header().Get("Location"))
			if err != nil || start.Code != 302 {
				t.Fatal(start.Code, err)
			}
			query := url.Values{"state": {location.Query().Get("state")}, "code": {"fixture-code"}, "repository_id": {"99"}, "expected_user_id": {"2"}, "return_to": {"https://evil.example/"}}
			callback := httptest.NewRequest("GET", config.SodaPath+"/oauth/callback?"+query.Encode(), nil)
			for _, c := range start.Result().Cookies() {
				callback.AddCookie(c)
			}
			callback.AddCookie(&http.Cookie{Name: sessionCookie, Value: "session-alice"})
			w := httptest.NewRecorder()
			s.ServeHTTP(w, callback)
			if w.Code != 303 || w.Header().Get("Location") != s.Config.ForgejoURL+tc.wantPath || calls != tc.wantCalls || w.Header().Get("Referrer-Policy") != "no-referrer" {
				t.Fatal(w.Code, w.Header().Get("Location"), calls, w.Body.String())
			}
			var session string
			for _, c := range w.Result().Cookies() {
				if c.Name == sessionCookie {
					session = c.Value
				}
			}
			if session == "" {
				t.Fatal("successful sign-in lost session")
			}
			grant, err := s.Store.Grant(t.Context(), session, 1)
			if err != nil || grant.Scopes != tc.scopes {
				t.Fatal("actual consent was not preserved", err)
			}
			w = httptest.NewRecorder()
			s.ServeHTTP(w, callback)
			if w.Code != 400 || calls != tc.wantCalls {
				t.Fatal("context replay reached provider", w.Code, calls)
			}
		})
	}
}

func TestOAuthExpectedUserMismatchPreservesExistingSodaState(t *testing.T) {
	calls := 0
	s := grantedTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		switch r.URL.Path {
		case "/login/oauth/access_token":
			fmt.Fprint(w, `{"access_token":"wrong-user-access","refresh_token":"wrong-user-refresh","token_type":"bearer","expires_in":3600}`)
		case "/api/v1/user":
			fmt.Fprint(w, `{"id":2,"login":"changed-bob"}`)
		default:
			t.Error("mismatch reached another provider operation")
			w.WriteHeader(500)
		}
	})
	start := httptest.NewRecorder()
	s.ServeHTTP(start, apiTestRequest("GET", "/login?repository_id=42&expected_user_id=1", "", "alice"))
	location, err := url.Parse(start.Header().Get("Location"))
	if err != nil || start.Code != 302 {
		t.Fatal(start.Code, err)
	}
	r := httptest.NewRequest("GET", config.SodaPath+"/oauth/callback?"+url.Values{"state": {location.Query().Get("state")}, "code": {"fixture-code"}, "expected_user_id": {"2"}}.Encode(), nil)
	for _, c := range start.Result().Cookies() {
		r.AddCookie(c)
	}
	r.AddCookie(&http.Cookie{Name: sessionCookie, Value: "session-alice"})
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != 403 || calls != 2 || w.Header().Get("Location") != "" {
		t.Fatal(w.Code, calls, w.Body.String())
	}
	for _, c := range w.Result().Cookies() {
		if c.Name == sessionCookie {
			t.Fatal("mismatch replaced browser session")
		}
	}
	for _, login := range []string{"alice", "bob"} {
		session, err := s.Store.Session(t.Context(), "session-"+login)
		if err != nil || session.User.Login != login {
			t.Fatal("profile/session changed", err)
		}
		grant, err := s.Store.Grant(t.Context(), "session-"+login, session.User.ID)
		if err != nil || grant.Access != "acting-"+login {
			t.Fatal("grant changed", err)
		}
	}
	w = httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != 400 || calls != 2 {
		t.Fatal("mismatch state replayed")
	}
}

func TestOAuthRejectsMalformedCallbackBeforeProviderCalls(t *testing.T) {
	s := grantedTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("invalid callback reached provider")
		w.WriteHeader(500)
	})
	for _, query := range []string{
		"state=missing&state=missing&code=x", "state=missing&code=x&code=y",
		"state=missing&code=%zz", "state=missing&code=x;y", "state=missing&code=" + strings.Repeat("a", 4097),
		"state=" + strings.Repeat("a", 129) + "&code=x", "state=missing&code=x&extra=" + strings.Repeat("a", 8192),
		"state=missing&code=x",
	} {
		r := httptest.NewRequest("GET", config.SodaPath+"/oauth/callback?"+query, nil)
		r.AddCookie(&http.Cookie{Name: oauthCookie, Value: "missing"})
		w := httptest.NewRecorder()
		s.ServeHTTP(w, r)
		if w.Code != 400 || w.Header().Get("Location") != "" {
			t.Fatal("bad callback accepted", w.Code)
		}
	}
}
