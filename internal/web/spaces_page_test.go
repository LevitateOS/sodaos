package web

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
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

func TestSpacesWorkspaceShell(t *testing.T) {
	header, err := os.ReadFile(filepath.Join("..", "..", "appliance", "forgejo", "templates", "custom", "header.tmpl"))
	if err != nil {
		t.Fatal(err)
	}
	revision := regexp.MustCompile(`name="soda-presentation-revision" content="([a-zA-Z0-9.-]+)"`).FindSubmatch(header)
	if len(revision) != 2 {
		t.Fatal("missing presentation revision")
	}
	s := grantedTestServer(t, func(http.ResponseWriter, *http.Request) { t.Fatal("shell must not query Forgejo") })
	w := httptest.NewRecorder()
	s.ServeHTTP(w, apiTestRequest("GET", "/workspace", "", "alice"))
	body := w.Body.String()
	if w.Code != 200 || w.Header().Get("Content-Type") != "text/html; charset=utf-8" {
		t.Fatal(w.Code, w.Header())
	}
	csp := w.Header().Get("Content-Security-Policy")
	if !strings.Contains(csp, "frame-src 'self'") || !strings.Contains(csp, "frame-ancestors 'none'") || strings.Contains(csp, "unsafe-inline") {
		t.Fatal(csp)
	}
	if !strings.Contains(body, `<iframe id="soda-forgejo-frame" title="Forgejo" src="/"></iframe>`) {
		t.Fatal("missing same-origin Forgejo frame")
	}
	if !strings.Contains(body, `data-actor="1"`) || !strings.Contains(body, "/assets/sodaspaces-shell.js?v="+string(revision[1])) {
		t.Fatal(body)
	}
	for _, forbidden := range []string{"csrf-alice", "session-alice", "callback-access", "window.config", "/app/", "unsafe-inline"} {
		if strings.Contains(body, forbidden) {
			t.Fatal("copied authority or credentials", forbidden)
		}
	}
}
