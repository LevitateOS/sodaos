package web

import (
	"context"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/config"
)

func TestCSRFAndOrigin(t *testing.T) {
	s := New(config.Config{PublicURL: "https://soda.example.test"}, nil)
	for _, tc := range []struct {
		origin, token string
		ok            bool
	}{{"https://soda.example.test", "secret", true}, {"https://evil.example", "secret", false}, {"", "wrong", false}} {
		r := httptest.NewRequest("POST", "/profile", strings.NewReader(url.Values{"csrf": {tc.token}}.Encode()))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		r.Header.Set("Origin", tc.origin)
		r.ParseForm()
		if s.validMutation(r, "secret") != tc.ok {
			t.Fatalf("origin %q", tc.origin)
		}
	}
}
func TestOAuthReturnPathIsStoredRatherThanTakenFromCallback(t *testing.T) {
	s := apiTestServer(t)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, httptest.NewRequest("GET", "/login?return_to=%2Fapp%2F", nil))
	if w.Code != 302 {
		t.Fatal(w.Code)
	}
	location, err := url.Parse(w.Header().Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	oauth, err := s.Store.ConsumeOAuth(context.Background(), location.Query().Get("state"))
	if err != nil || oauth.ReturnPath != "/app/" || oauth.Verifier == "" {
		t.Fatal(oauth.ReturnPath, err)
	}
	if location.Query().Get("redirect_uri") != s.Config.PublicURL+"/oauth/callback" {
		t.Fatal("upstream callback changed")
	}
	for _, target := range []string{"//evil.example/", "https://evil.example/", "/people", "/app/../people"} {
		w = httptest.NewRecorder()
		s.ServeHTTP(w, httptest.NewRequest("GET", "/login?"+url.Values{"return_to": {target}}.Encode(), nil))
		if w.Code != 400 || w.Header().Get("Set-Cookie") != "" {
			t.Fatal("unsafe redirect accepted", target)
		}
	}
}

func TestOperatorBoundary(t *testing.T) {
	s := New(config.Config{}, nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, httptest.NewRequest("GET", "/people", nil))
	if w.Code != 303 {
		t.Fatal(w.Code)
	}
}
