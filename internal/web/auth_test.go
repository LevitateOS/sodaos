package web

import (
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/config"
)

func TestCSRFAndOrigin(t *testing.T) {
	s := New(config.Config{PublicURL: "https://soda.example.test"}, nil)
	for _, tc := range []struct {
		origin, token string
		ok            bool
	}{{"https://soda.example.test", "secret", true}, {"https://evil.example", "secret", false}, {"", "wrong", false}} {
		r := httptest.NewRequest("POST", "/profile", strings.NewReader(url.Values{"csrf": {tc.token}}.Encode()))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		r.Header.Set("Origin", tc.origin)
		r.ParseForm()
		if s.validMutation(r, "secret") != tc.ok {
			t.Fatalf("origin %q", tc.origin)
		}
	}
}
func TestOperatorBoundary(t *testing.T) {
	s := New(config.Config{}, nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, httptest.NewRequest("GET", "/people", nil))
	if w.Code != 303 {
		t.Fatal(w.Code)
	}
}
