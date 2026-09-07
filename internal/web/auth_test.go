package web

import (
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestOAuthHasNoCallerSelectedReturnOrAdministrationConsent(t *testing.T) {
	s := apiTestServer(t)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, httptest.NewRequest("GET", "/-/soda/login?administration=1", nil))
	if w.Code != 302 {
		t.Fatal(w.Code)
	}
	location, err := url.Parse(w.Header().Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	verifier, err := s.Store.ConsumeOAuth(t.Context(), location.Query().Get("state"))
	if err != nil || verifier == "" {
		t.Fatal(err)
	}
	q := location.Query()
	if q.Get("redirect_uri") != s.Config.OAuthCallbackURL() || q.Get("scope") != "read:user read:repository read:organization" || q.Get("code_challenge_method") != "S256" || q.Get("code_challenge") == "" {
		t.Fatal("OAuth contract changed")
	}
	for _, target := range []string{"//evil.example/", "https://evil.example/", "/", "/projects", "/people", "/app/", "/app/../people"} {
		w = httptest.NewRecorder()
		s.ServeHTTP(w, httptest.NewRequest("GET", "/-/soda/login?"+url.Values{"return_to": {target}}.Encode(), nil))
		if w.Code != 400 || w.Header().Get("Set-Cookie") != "" || w.Header().Get("Location") != "" {
			t.Fatal("caller redirect accepted", target)
		}
		if w.Header().Get("Content-Type") != "text/plain; charset=utf-8" {
			t.Fatal("OAuth error requires removed templates")
		}
	}
}
