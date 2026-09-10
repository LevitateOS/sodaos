package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPageEntryGuards(t *testing.T) {
	for _, page := range []struct{ path string }{
		{"/spaces"}, {"/settings/runners"}, {"/repositories/7/settings/spaces"},
	} {
		t.Run(page.path, func(t *testing.T) {
			s := apiTestServer(t)
			for _, mode := range []string{"anonymous", "expired", "duplicate cookie", "query", "empty query", "http origin", "invalid origin", "missing store"} {
				t.Run(mode, func(t *testing.T) {
					r := apiTestRequest("GET", page.path, "", "")
					origin, storage := s.Config.ForgejoURL, s.Store
					t.Cleanup(func() { s.Config.ForgejoURL, s.Store = origin, storage })
					status := 303
					switch mode {
					case "expired":
						r.AddCookie(&http.Cookie{Name: sessionCookie, Value: "expired"})
					case "duplicate cookie":
						r.AddCookie(&http.Cookie{Name: sessionCookie, Value: "session-alice"})
						r.AddCookie(&http.Cookie{Name: sessionCookie, Value: "session-alice"})
						status = 400
					case "query":
						r.URL.RawQuery = "actor=2"
						status = 400
					case "empty query":
						r.URL.ForceQuery = true
						status = 400
					case "http origin":
						s.Config.ForgejoURL = "http://forgejo.example.test"
						status = 503
					case "invalid origin":
						s.Config.ForgejoURL = "https://forgejo.example.test/?bad=1"
						status = 503
					case "missing store":
						r.AddCookie(&http.Cookie{Name: sessionCookie, Value: "session-alice"})
						s.Store = nil
					}
					w := httptest.NewRecorder()
					s.ServeHTTP(w, r)
					if w.Code != status || w.Header().Get("Cache-Control") != "private, no-store" || w.Header().Get("Referrer-Policy") != "no-referrer" {
						t.Fatal(mode, w.Code, w.Header())
					}
					if strings.Contains(w.Body.String(), "data-actor=") {
						t.Fatal("unauthorized actor bootstrap")
					}
				})
			}
		})
	}
}
