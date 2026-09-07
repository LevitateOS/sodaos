package web

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/config"
	"github.com/levitateos/sodaos/internal/store"
)

func TestSodaNamespaceDoesNotAliasNativeOrLegacyPaths(t *testing.T) {
	s := apiTestServer(t)
	for _, target := range []string{
		"/api/session", "/login", "/oauth/callback", "/-/soda",
		"/-/sodax/api/session", "/-/fetch-redirect", "/user/login", "/assets/logo.svg",
		"/-/soda/healthz", "/-/soda//api/session", "/-/soda/api/../api/session",
		"/-/soda/api/%73ession", "/-/soda/api%2fsession", "/%2d/soda/api/session",
		"/-/soda/api/..%2fsession", "/-/soda/api/session/", "/-/soda/api/session%5c",
	} {
		t.Run(target, func(t *testing.T) {
			for _, method := range []string{"GET", "POST"} {
				r := httptest.NewRequest(method, target, nil)
				r.AddCookie(&http.Cookie{Name: sessionCookie, Value: "session-alice"})
				w := httptest.NewRecorder()
				s.ServeHTTP(w, r)
				if w.Code != 404 || w.Header().Get("Location") != "" {
					t.Fatalf("%s %s: %d %s", method, target, w.Code, w.Body.String())
				}
				if r.URL.RequestURI() != target {
					t.Fatal("mount changed caller's URL")
				}
			}
		})
	}
	w := httptest.NewRecorder()
	s.ServeHTTP(w, apiTestRequest("GET", "/api/session", "", "alice"))
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
}

func TestSodaOnlyAcceptsOneScopedSessionCookie(t *testing.T) {
	s := apiTestServer(t)
	for _, cookies := range []string{
		"soda_session=session-alice", "i_like_gitea=session-alice",
		sessionCookie + "=session-alice; " + sessionCookie + "=session-bob",
		sessionCookie + "=session-alice; " + sessionCookie + "=session-alice",
		sessionCookie + "=", sessionCookie + "=" + strings.Repeat("a", 129),
	} {
		r := httptest.NewRequest("GET", config.SodaPath+"/api/session", nil)
		r.Header.Set("Cookie", cookies)
		w := httptest.NewRecorder()
		s.ServeHTTP(w, r)
		if w.Code != 401 {
			t.Fatal("invalid/foreign cookie selected an actor", w.Code)
		}
	}
	// Native cookies may arrive on the shared origin, but never select Soda's actor.
	r := apiTestRequest("GET", "/api/session", "", "alice")
	r.AddCookie(&http.Cookie{Name: "i_like_gitea", Value: "session-bob"})
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"id":"1"`) {
		t.Fatal(w.Code, w.Body.String())
	}
}

func TestSodaLoginAndLogoutUseScopedSecureCookies(t *testing.T) {
	s := apiTestServer(t)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, httptest.NewRequest("GET", config.SodaPath+"/login", nil))
	location, err := url.Parse(w.Header().Get("Location"))
	if err != nil || w.Code != 302 || location.Query().Get("redirect_uri") != s.Config.OAuthCallbackURL() {
		t.Fatal("incorrect callback route", w.Code, err)
	}
	cookies := w.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != oauthCookie || cookies[0].MaxAge != 600 {
		t.Fatal("incorrect OAuth cookie")
	}
	for _, cookie := range cookies {
		if cookie.Path != config.SodaPath+"/" || cookie.Domain != "" || !cookie.HttpOnly || !cookie.Secure || cookie.SameSite != http.SameSiteLaxMode {
			t.Fatal("OAuth cookie boundary lost")
		}
	}
	// Moving onto Forgejo's origin does not allow the former Soda origin or a
	// same-site different port to make a mutation, even with valid credentials.
	for _, origin := range []string{"https://soda.example.test", "https://forgejo.example.test:444"} {
		r := apiTestRequest("POST", "/api/session/logout", "{}", "alice")
		r.Header.Set("Origin", origin)
		w = httptest.NewRecorder()
		s.ServeHTTP(w, r)
		if w.Code != 403 {
			t.Fatal("foreign origin accepted", w.Code)
		}
	}
	w = httptest.NewRecorder()
	s.ServeHTTP(w, apiTestRequest("POST", "/api/session/logout", "{}", "alice"))
	cookies = w.Result().Cookies()
	if w.Code != 204 || len(cookies) != 2 || cookies[0].Name != sessionCookie || cookies[0].MaxAge != -1 || cookies[0].Path != config.SodaPath+"/" || cookies[0].Domain != "" || !cookies[0].Secure || !cookies[0].HttpOnly || cookies[0].SameSite != http.SameSiteLaxMode {
		t.Fatal("logout did not expire only the scoped Soda cookie")
	}
}

func TestOAuthRejectsLegacyAndDuplicateCookiesBeforeExchange(t *testing.T) {
	for _, cookies := range []string{
		"soda_oauth=pending", "i_like_gitea=pending",
		oauthCookie + "=pending; " + oauthCookie + "=pending",
		oauthCookie + "=pending; " + sessionCookie + "=session-alice; " + sessionCookie + "=session-bob",
	} {
		s := grantedTestServer(t, func(w http.ResponseWriter, r *http.Request) {
			t.Error("invalid cookies reached provider")
			w.WriteHeader(500)
		})
		if err := s.Store.BeginOAuth(t.Context(), "pending", store.OAuthLogin{Verifier: "verifier"}, "", ""); err != nil {
			t.Fatal(err)
		}
		r := httptest.NewRequest("GET", config.SodaPath+"/oauth/callback?state=pending&code=test", nil)
		r.Header.Set("Cookie", cookies)
		w := httptest.NewRecorder()
		s.ServeHTTP(w, r)
		if w.Code != 400 || w.Header().Get("Location") != "" {
			t.Fatal("invalid cookies accepted", w.Code)
		}
		if _, err := s.Store.ConsumeOAuth(t.Context(), "pending", ""); err != nil {
			t.Fatal("invalid cookies consumed state", err)
		}
	}
}
