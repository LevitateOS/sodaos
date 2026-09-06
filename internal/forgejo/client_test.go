package forgejo

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCurrent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/user" || r.Header.Get("Authorization") != "token test" {
			t.Error("wrong native request")
		}
		fmt.Fprint(w, `{"id":12,"login":"alice"}`)
	}))
	defer server.Close()
	u, err := New(server.URL).Current(context.Background(), "test")
	if err != nil || u.ID != 12 {
		t.Fatalf("%+v %v", u, err)
	}
}
func TestNoCredentialRedirect(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, "http://untrusted.invalid", 302) }))
	defer s.Close()
	if _, err := New(s.URL).Current(context.Background(), "secret"); err == nil {
		t.Fatal("redirect accepted")
	}
}
