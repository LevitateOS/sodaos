package web

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSpacesBookmarkEntry(t *testing.T) {
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
			if w.Code != 303 || calls != 0 || !strings.Contains(w.Header().Get("Location"), "redirect_to=%2F%3Fsoda-view%3Dspaces") {
				t.Fatal(w.Code, w.Header(), calls)
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
