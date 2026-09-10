package web

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/store"
)

func cancellationRequest(method, state, csrf string) *http.Request {
	r := apiTestRequest(method, "/api/login/cancel", "{}", "")
	r.Header.Set("X-Soda-Logout", "1")
	r.Header.Set(expectedUserHeader, "1")
	r.Header.Set("Sec-Fetch-Site", "same-origin")
	r.Header.Set("Origin", "https://forgejo.example.test")
	r.Header.Set("Content-Type", "application/json")
	if csrf != "" {
		r.Header.Set("X-CSRF-Token", csrf)
	}
	if state != "" {
		r.AddCookie(&http.Cookie{Name: oauthCookie, Value: state})
	}
	return r
}

func cancellationCSRF(t *testing.T, s *Server, state string) string {
	t.Helper()
	w := httptest.NewRecorder()
	s.ServeHTTP(w, cancellationRequest("GET", state, ""))
	var body struct {
		CSRF string `json:"csrf_token"`
	}
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &body) != nil || body.CSRF == "" || w.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("bootstrap failed", w.Code)
	}
	return body.CSRF
}

func TestAnonymousCancellationBeforeAfterCallbackAndLateCookie(t *testing.T) {
	for _, first := range []bool{false, true} {
		t.Run(map[bool]string{false: "cancel first", true: "callback first"}[first], func(t *testing.T) {
			s := grantedTestServer(t, func(w http.ResponseWriter, r *http.Request) { t.Error("unexpected provider request") })
			if err := s.Store.BeginOAuth(t.Context(), "pending", store.OAuthLogin{Verifier: "v", ExpectedUserID: 1}, "", ""); err != nil {
				t.Fatal(err)
			}
			csrf := cancellationCSRF(t, s, "pending")
			a, err := s.Store.ConsumeOAuth(t.Context(), "pending", "")
			if err != nil {
				t.Fatal(err)
			}
			finish := func() error {
				return s.Store.FinishOAuth(t.Context(), a, store.User{ID: 1, Login: "alice"}, "late-cookie", "csrf", store.Grant{Access: "fixture", Refresh: "fixture-refresh", Scopes: "read:user", Expires: time.Now().Add(time.Hour).Unix()})
			}
			if first {
				if err := finish(); err != nil {
					t.Fatal(err)
				}
			}
			w := httptest.NewRecorder()
			s.ServeHTTP(w, cancellationRequest("POST", "pending", csrf))
			if w.Code != 204 {
				t.Fatal("cancellation failed", w.Code, w.Body.String())
			}
			if !first && finish() == nil {
				t.Fatal("cancelled callback committed")
			}
			if _, err := s.Store.Session(t.Context(), "late-cookie"); err == nil {
				t.Fatal("late cookie resurrected session")
			}
			if _, err := s.Store.Session(t.Context(), "session-bob"); err != nil {
				t.Fatal("other browser lost", err)
			}
		})
	}
}

func TestCancellationRefusesRotationForeignActorAndInvalidRequests(t *testing.T) {
	s := apiTestServer(t)
	if err := s.Store.BeginOAuth(t.Context(), "pending", store.OAuthLogin{Verifier: "v"}, "session-alice", ""); err != nil {
		t.Fatal(err)
	}
	csrf := cancellationCSRF(t, s, "pending")
	for _, mode := range []string{"origin", "metadata", "csrf", "duplicate cookie", "duplicate actor", "actor", "body", "query", "header", "foreign session"} {
		r := cancellationRequest("POST", "pending", csrf)
		switch mode {
		case "origin":
			r.Header.Set("Origin", "https://elsewhere.test")
		case "metadata":
			r.Header.Set("Sec-Fetch-Site", "cross-site")
		case "csrf":
			r.Header.Set("X-CSRF-Token", "wrong")
		case "duplicate cookie":
			r.AddCookie(&http.Cookie{Name: oauthCookie, Value: "pending"})
		case "duplicate actor":
			r.Header.Add(expectedUserHeader, "1")
		case "actor":
			r.Header.Set(expectedUserHeader, "2")
		case "body":
			r.Body = http.NoBody
		case "query":
			r.URL.RawQuery = "context=chosen"
		case "header":
			r.Header.Del("X-Soda-Logout")
		case "foreign session":
			r.AddCookie(&http.Cookie{Name: sessionCookie, Value: "session-bob"})
		}
		w := httptest.NewRecorder()
		s.ServeHTTP(w, r)
		if w.Code < 400 {
			t.Fatal(mode, "accepted")
		}
		if _, err := s.Store.Session(t.Context(), "session-alice"); err != nil {
			t.Fatal(mode, err)
		}
	}
	if err := s.Store.BeginOAuth(t.Context(), "rotated", store.OAuthLogin{Verifier: "new"}, "session-alice", ""); err != nil {
		t.Fatal(err)
	}
	for _, state := range []string{"pending", "rotated"} {
		w := httptest.NewRecorder()
		s.ServeHTTP(w, cancellationRequest("POST", state, csrf))
		if w.Code < 400 {
			t.Fatal("stale bootstrap/cookie cancelled rotation")
		}
	}
	if _, err := s.Store.Session(t.Context(), "session-alice"); err != nil {
		t.Fatal(err)
	}
	for _, method := range []string{"PUT", "DELETE"} {
		w := httptest.NewRecorder()
		s.ServeHTTP(w, cancellationRequest(method, "pending", csrf))
		if w.Code != 405 {
			t.Fatal(method, w.Code)
		}
	}
	w := httptest.NewRecorder()
	r := cancellationRequest("POST", "rotated", cancellationCSRF(t, s, "rotated"))
	r.Body = io.NopCloser(strings.NewReader(`{"context":"foreign"}`))
	s.ServeHTTP(w, r)
	if w.Code != 400 {
		t.Fatal(w.Code)
	}
}

func TestCancellationBootstrapRefusesAmbiguousSessionsAndCrossOrigin(t *testing.T) {
	s := apiTestServer(t)
	for _, mode := range []string{"duplicate session", "duplicate OAuth", "cross origin", "missing metadata", "missing actor"} {
		r := cancellationRequest("GET", "", "")
		switch mode {
		case "duplicate session":
			r.AddCookie(&http.Cookie{Name: sessionCookie, Value: "session-alice"})
			r.AddCookie(&http.Cookie{Name: sessionCookie, Value: "session-bob"})
		case "duplicate OAuth":
			r.AddCookie(&http.Cookie{Name: oauthCookie, Value: "one"})
			r.AddCookie(&http.Cookie{Name: oauthCookie, Value: "two"})
		case "cross origin":
			r.Header.Set("Origin", "https://elsewhere.test")
		case "missing metadata":
			r.Header.Del("Sec-Fetch-Site")
		case "missing actor":
			r.Header.Del(expectedUserHeader)
		}
		w := httptest.NewRecorder()
		s.ServeHTTP(w, r)
		if w.Code < 400 {
			t.Fatal(mode, "accepted")
		}
	}
}
