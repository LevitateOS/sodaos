package web

import (
	"net/http/httptest"
	"testing"

	"github.com/levitateos/sodaos/internal/config"
)

func TestHome(t *testing.T) {
	s := New(config.Config{}, nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	if w.Code != 200 {
		t.Fatal(w.Code)
	}
	if w.Header().Get("Content-Security-Policy") == "" {
		t.Fatal("missing CSP")
	}
}
