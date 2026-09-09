package web

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSpacesHTMLSessionAuthorityAndBoundedException(t *testing.T) {
	for _, mode := range []string{"anonymous", "authorized", "mismatch", "unavailable"} {
		t.Run(mode, func(t *testing.T) {
			calls := 0
			s := grantedTestServer(t, func(w http.ResponseWriter, r *http.Request) {
				calls++
				if mode == "unavailable" {
					w.WriteHeader(503)
					return
				}
				id := 1
				if mode == "mismatch" {
					id = 2
				}
				fmt.Fprintf(w, `{"id":%d,"login":"alice"}`, id)
			})
			r := apiTestRequest("GET", "/spaces", "", "alice")
			if mode == "anonymous" {
				r.Header.Del("Cookie")
			}
			w := httptest.NewRecorder()
			s.ServeHTTP(w, r)
			body := w.Body.String()
			expected := 200
			if mode == "mismatch" || mode == "unavailable" {
				expected = 503
			}
			if w.Code != expected || !strings.HasPrefix(w.Header().Get("Content-Type"), "text/html") || w.Header().Get("Referrer-Policy") != "no-referrer" || w.Header().Get("Cache-Control") != "private, no-store" {
				t.Fatal(w.Code, w.Header())
			}
			if strings.Contains(body, `src="/assets/sodaspaces-page.js"`) != (mode == "authorized") {
				t.Fatal("unauthorized workspace bootstrap")
			}
			if mode == "anonymous" && calls != 0 {
				t.Fatal("anonymous page contacted provider")
			}
			for _, forbidden := range []string{"csrf-alice", "session-alice", "callback-access", "window.config", "/app/", "<iframe"} {
				if strings.Contains(body, forbidden) {
					t.Fatal("copied authority or credentials", forbidden)
				}
			}
			if strings.Contains(w.Header().Get("Content-Security-Policy"), "script-src 'self' 'unsafe-inline'") {
				t.Fatal("inline script permission")
			}
		})
	}
	s := apiTestServer(t)
	for _, path := range []string{"/spaces?", "/spaces?destination=spaces"} {
		w := terminalAPI(t, s, s.Config.ForgejoURL, "GET", path, nil)
		if w.Code != 400 {
			t.Fatal("page query alias")
		}
	}
	for _, path := range []string{"/app/", "/projects", "/spaces/"} {
		w := terminalAPI(t, s, s.Config.ForgejoURL, "GET", path, nil)
		if w.Code != 404 {
			t.Fatal("retired alias", path, w.Code)
		}
	}
	w := terminalAPI(t, s, s.Config.ForgejoURL, "GET", "/api/session", nil)
	if strings.Contains(w.Header().Get("Content-Security-Policy"), "script-src") {
		t.Fatal("page CSP leaked to API")
	}
}
