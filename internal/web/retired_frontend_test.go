package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRetiredFrontendAndForgeAdaptersAreNotRoutes(t *testing.T) {
	s := grantedTestServer(t, func(http.ResponseWriter, *http.Request) { t.Error("retired route called Forgejo") })
	s.Host.HTTP = &http.Client{Transport: roundTrip(func(*http.Request) (*http.Response, error) {
		t.Error("retired route called native helper")
		return nil, nil
	})}
	for _, path := range []string{
		"/app/", "/app/assets/stale.js", "/app/repositories/alice/demo",
		"/profile", "/people", "/keys", "/projects", "/projects/p0123456789abcdef01234567", "/projects/p0123456789abcdef01234567/join", "/logout",
		"/static/htmx.min.js", "/static/soda.css", "/assets/branding/source/logo.svg",
		"/api/forgejo/repositories", "/api/forgejo/admin/users", "/api/forgejo/me/settings", "/api/forgejo/me/git-keys",
		"/api/forgejo/repos/alice/demo", "/api/forgejo/repos/alice/demo/contents", "/api/forgejo/repos/alice/demo/download",
		"/api/forgejo/repos/alice/demo/issues", "/api/forgejo/repos/alice/demo/pulls", "/api/forgejo/repos/alice/demo/actions/runs",
		"/api/forgejo/organizations", "/api/forgejo/notifications",
	} {
		for _, method := range []string{"GET", "POST"} {
			t.Run(method+path, func(t *testing.T) {
				w := httptest.NewRecorder()
				s.ServeHTTP(w, apiTestRequest(method, path, `{}`, "alice"))
				if w.Code != 404 {
					t.Fatal("retired route survived", w.Code, w.Body.String())
				}
				if strings.HasPrefix(path, "/api/") && !strings.HasPrefix(w.Header().Get("Content-Type"), "application/json") {
					t.Fatal("retired API returned HTML")
				}
			})
		}
	}
}

func TestSodaAPIsRemainWithoutEitherFrontend(t *testing.T) {
	s := apiTestServer(t)
	for _, path := range []string{"/api/session", "/api/environments", "/api/me/development-keys"} {
		w := httptest.NewRecorder()
		s.ServeHTTP(w, apiTestRequest("GET", path, "", "alice"))
		want := 200
		if path == "/api/environments" {
			want = 400
		} // repository context is now required
		if w.Code != want || !strings.HasPrefix(w.Header().Get("Content-Type"), "application/json") {
			t.Fatal(path, w.Code, w.Body.String())
		}
	}
}
