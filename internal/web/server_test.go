package web

import (
	"github.com/levitateos/sodaos/internal/config"
	"net/http/httptest"
	"testing"
)

func TestHome(t *testing.T) {
	s := New(config.Config{})
	w := httptest.NewRecorder()
	s.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	if w.Code != 200 {
		t.Fatal(w.Code)
	}
	if w.Header().Get("Content-Security-Policy") == "" {
		t.Fatal("missing CSP")
	}
}
