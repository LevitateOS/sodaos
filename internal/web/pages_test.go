package web

import (
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPageGuardsCSPAndBufferedRendering(t *testing.T) {
	for _, page := range []struct {
		path        string
		template    **template.Template
		inlineStyle bool
		renderError string
	}{
		{"/spaces", &spacesTemplate, true, "Could not render Spaces."},
		{"/settings/runners", &runnersTemplate, false, "Cannot render settings."},
		{"/repositories/7/settings/spaces", &repositorySpacesTemplate, false, "Cannot render settings."},
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
						status = 503
					}
					w := httptest.NewRecorder()
					s.ServeHTTP(w, r)
					csp := w.Header().Get("Content-Security-Policy")
					if w.Code != status || w.Header().Get("Cache-Control") != "private, no-store" || w.Header().Get("Referrer-Policy") != "no-referrer" {
						t.Fatal(mode, w.Code, w.Header())
					}
					if strings.Contains(csp, "style-src 'self' 'unsafe-inline'") != page.inlineStyle || !strings.Contains(csp, "script-src 'self';") || !strings.Contains(csp, "frame-ancestors 'none'") {
						t.Fatal("page CSP changed", csp)
					}
					if strings.Contains(w.Body.String(), "data-actor=") {
						t.Fatal("unauthorized actor bootstrap")
					}
				})
			}
			t.Run("render failure", func(t *testing.T) {
				original := *page.template
				t.Cleanup(func() { *page.template = original })
				*page.template = template.Must(template.New("failure").Funcs(template.FuncMap{
					"fail": func() (string, error) { return "", errors.New("private-render-detail") },
				}).Parse(`partial-authorized-html{{fail}}`))
				w := httptest.NewRecorder()
				renderServer := grantedTestServer(t, func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path == "/api/v1/user" {
						fmt.Fprint(w, `{"id":1,"login":"alice"}`)
					} else {
						fmt.Fprint(w, `{"id":7,"name":"demo","full_name":"alice/demo","owner":{"id":1,"login":"alice"},"permissions":{"admin":true}}`)
					}
				})
				renderServer.ServeHTTP(w, apiTestRequest("GET", page.path, "", "alice"))
				if w.Code != 500 || w.Body.String() != page.renderError+"\n" || !strings.HasPrefix(w.Header().Get("Content-Type"), "text/plain") {
					t.Fatal("partial HTML or internal render error escaped", w.Code, w.Body.String())
				}
			})
		})
	}
}
