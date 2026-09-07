package web

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestLogoutCancelsClaimedOAuthBeforeAndAfterCallbackCommit(t *testing.T) {
	for _, callbackFirst := range []bool{false, true} {
		t.Run(fmt.Sprint(callbackFirst), func(t *testing.T) {
			entered, release := make(chan struct{}), make(chan struct{})
			var once sync.Once
			unblock := func() { once.Do(func() { close(release) }) }
			defer unblock()
			s := grantedTestServer(t, func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/login/oauth/access_token":
					fmt.Fprint(w, `{"access_token":"review-access","refresh_token":"review-refresh","token_type":"bearer","expires_in":3600}`)
				case "/api/v1/user":
					fmt.Fprint(w, `{"id":1,"login":"changed-alice"}`)
				case "/login/oauth/introspect":
					fmt.Fprint(w, `{"active":true,"scope":"read:user read:repository","sub":"1","aud":["client"]}`)
				case "/api/v1/repositories/42":
					if !callbackFirst {
						close(entered)
						<-release
					}
					fmt.Fprint(w, `{"id":42,"name":"demo","full_name":"alice/demo","owner":{"id":1,"login":"alice"}}`)
				default:
					t.Error("unexpected provider request")
					w.WriteHeader(500)
				}
			})
			start := httptest.NewRecorder()
			s.ServeHTTP(start, apiTestRequest("GET", "/login?repository_id=42&expected_user_id=1", "", "alice"))
			u, err := url.Parse(start.Header().Get("Location"))
			if err != nil || start.Code != 302 {
				t.Fatal("login failed")
			}
			callback := apiTestRequest("GET", "/oauth/callback?"+url.Values{"state": {u.Query().Get("state")}, "code": {"fixture-code"}}.Encode(), "", "alice")
			for _, c := range start.Result().Cookies() {
				callback.AddCookie(c)
			}
			logout := apiTestRequest("POST", "/api/session/logout", "{}", "alice")
			// Blocking body consumption pauses logout AFTER actor/session/CSRF validation,
			// exercising the old-token/new-context rotation order without production hooks.
			if callbackFirst {
				logout.Body = &pausedLogoutBody{entered: entered, release: release, Reader: strings.NewReader("{}")}
			}
			finished := make(chan *httptest.ResponseRecorder, 1)
			go func() {
				w := httptest.NewRecorder()
				if callbackFirst {
					s.ServeHTTP(w, logout)
				} else {
					s.ServeHTTP(w, callback)
				}
				finished <- w
			}()
			select {
			case <-entered:
			case <-time.After(10 * time.Second):
				t.Fatal("barrier not reached")
			}
			w := httptest.NewRecorder()
			var callbackResult, logoutResult *httptest.ResponseRecorder
			if callbackFirst {
				s.ServeHTTP(w, callback)
				callbackResult = w
			} else {
				s.ServeHTTP(w, logout)
				logoutResult = w
			}
			unblock()
			if callbackFirst {
				logoutResult = <-finished
			} else {
				callbackResult = <-finished
			}
			if logoutResult.Code != 204 {
				t.Fatal("logout failed", logoutResult.Code)
			}
			if callbackFirst && callbackResult.Code != 303 {
				t.Fatal(callbackResult.Code, callbackResult.Body.String())
			}
			if !callbackFirst && (callbackResult.Code != 409 || callbackResult.Header().Get("Set-Cookie") != "") {
				t.Fatal("cancelled callback wrote cookies", callbackResult.Code)
			}
			for _, c := range callbackResult.Result().Cookies() {
				if c.Name != sessionCookie || c.Value == "" {
					continue
				}
				// Simulate applying a late successful callback's cookie AFTER logout.
				req := apiTestRequest("GET", "/api/session", "", "")
				req.AddCookie(c)
				denied := httptest.NewRecorder()
				s.ServeHTTP(denied, req)
				if denied.Code != 401 {
					t.Fatal("late cookie resurrected session", denied.Code)
				}
				if _, err := s.Store.Grant(t.Context(), c.Value, 1); err == nil {
					t.Fatal("grant survived")
				}
			}
			if _, err := s.Store.Session(t.Context(), "session-alice"); err == nil {
				t.Fatal("old session survived")
			}
			if _, err := s.Store.Grant(t.Context(), "session-bob", 2); err != nil {
				t.Fatal("unrelated browser affected", err)
			}
			user, err := s.Store.User(t.Context(), 1)
			if err != nil || (!callbackFirst && user.Login != "alice") {
				t.Fatal("cancelled callback changed profile")
			}
			// A genuinely new anonymous sign-in after logout can start normally.
			fresh := httptest.NewRecorder()
			s.ServeHTTP(fresh, apiTestRequest("GET", "/login", "", ""))
			if fresh.Code != 302 {
				t.Fatal(fresh.Code)
			}
		})
	}
}

type pausedLogoutBody struct {
	io.Reader
	entered, release chan struct{}
	once             sync.Once
}

func (b *pausedLogoutBody) Read(p []byte) (int, error) {
	b.once.Do(func() { close(b.entered); <-b.release })
	return b.Reader.Read(p)
}
func (b *pausedLogoutBody) Close() error { return nil }

func TestLoginRejectsStaleAndAmbiguousCookiesWithoutAnonymousFallback(t *testing.T) {
	s := apiTestServer(t)
	for _, cookie := range []string{sessionCookie + "=expired", oauthCookie + "=expired", sessionCookie + "=session-alice; " + sessionCookie + "=session-bob"} {
		r := apiTestRequest("GET", "/login", "", "")
		r.Header.Set("Cookie", cookie)
		w := httptest.NewRecorder()
		s.ServeHTTP(w, r)
		if w.Code != 409 && w.Code != 400 {
			t.Fatal("stale login accepted", w.Code)
		}
		if w.Header().Get("Location") != "" {
			t.Fatal("silently restarted login")
		}
	}
	if _, err := s.Store.Session(t.Context(), "session-alice"); err != nil {
		t.Fatal(errors.New("cookie rejection deleted session"))
	}
}
