package web

import (
	"net/http/httptest"
	"strings"
	"testing"

	extensions "forgejo.org/extension-sdk"
)

func TestProductAPIsRequireNativeExtensionRoute(t *testing.T) {
	s := apiTestServer(t)
	proxy := nativeProductProxy(t, s, extensions.Contribution{})
	for _, path := range []string{"/api/session", "/api/environments", "/api/me/development-keys"} {
		w := httptest.NewRecorder()
		s.ServeHTTP(w, apiTestRequest("GET", path, "", "alice"))
		if w.Code != 404 {
			t.Fatal("product API remains on public dashboard route", path, w.Code)
		}
		response := nativeProductRequest(t, proxy, s.Config.ForgejoURL, "GET", path, nil)
		want := 200
		if path == "/api/environments" {
			want = 400
		}
		if response.StatusCode != want || !strings.HasPrefix(response.Header.Get("Content-Type"), "application/json") {
			t.Fatal(path, response.StatusCode)
		}
	}
}
