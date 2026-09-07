package web

import (
	"net/http/httptest"
	"testing"

	"github.com/levitateos/sodaos/internal/config"
)

func TestRootOnlyRedirectsToConfiguredForgejo(t *testing.T) {
	for _, base := range []string{"https://forgejo.example.test", "https://forgejo.example.test/"} {
		s := New(config.Config{ForgejoURL: base}, nil)
		for _, target := range []string{"/", "/?return_to=https://evil.example"} {
			w := httptest.NewRecorder()
			s.ServeHTTP(w, httptest.NewRequest("GET", target, nil))
			want := base
			if want[len(want)-1] != '/' {
				want += "/"
			}
			if w.Code != 303 || w.Header().Get("Location") != want || w.Header().Get("Cache-Control") != "no-store" {
				t.Fatal(w.Code, w.Header())
			}
			if w.Header().Get("Content-Security-Policy") == "" {
				t.Fatal("missing CSP")
			}
		}
	}
}
func TestNoNativeDestinationDoesNotLoopOrRender(t *testing.T) {
	for _, base := range []string{"", "//evil.example", "javascript:alert(1)", "https://user:password@example.test", "https://forgejo.example.test?redirect=evil", "https://forgejo.example.test/native/", "http://forgejo.example.test"} {
		s := New(config.Config{ForgejoURL: base}, nil)
		w := httptest.NewRecorder()
		s.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
		if w.Code != 503 || w.Header().Get("Location") != "" {
			t.Fatal(w.Code, w.Header())
		}
		w = httptest.NewRecorder()
		s.ServeHTTP(w, httptest.NewRequest("GET", "/healthz", nil))
		if w.Code != 200 || w.Body.String() != "ok\n" {
			t.Fatal("health check depends on a frontend")
		}
	}
}
