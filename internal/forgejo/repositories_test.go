package forgejo

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestUserRepositoriesPaginatesForRequestedUser(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.Method != "GET" || r.URL.Path != "/api/v1/users/alice/repos" || r.URL.Query().Get("limit") != "50" || r.Header.Get("Authorization") != "token operator" {
			t.Error("repository listing must query the requested user with the server credential")
		}
		switch r.URL.Query().Get("page") {
		case "1":
			// Model a server that caps pages below our requested limit.
			fmt.Fprint(w, `[{"id":1,"full_name":"alice/one","owner":{"id":7}},{"id":2,"full_name":"alice/two","owner":{"id":7}}]`)
		case "2":
			fmt.Fprint(w, `[{"id":3,"full_name":"alice/three","owner":{"id":7}}]`)
		case "3":
			fmt.Fprint(w, `[]`)
		default:
			t.Error("unexpected page", r.URL.Query().Get("page"))
			http.Error(w, "unexpected page", 500)
		}
	}))
	defer server.Close()
	repositories, err := New(server.URL).UserRepositories(t.Context(), "operator", "alice")
	if err != nil || requests != 3 || len(repositories) != 3 {
		t.Fatalf("repositories=%+v requests=%d err=%v", repositories, requests, err)
	}
	if repositories[2].FullName != "alice/three" || repositories[2].Owner.ID != 7 {
		t.Fatal("lost repository identity", repositories[2])
	}
}

func TestUserRepositoriesDoesNotReturnPartialListAfterFailure(t *testing.T) {
	for _, test := range []struct {
		name   string
		status int
		body   string
	}{
		{"provider failure", 503, "unavailable"},
		{"malformed response", 200, "{"},
		{"redirect", 302, ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Query().Get("page") == "1" {
					fmt.Fprint(w, `[{"id":1,"full_name":"alice/one"}]`)
					return
				}
				w.Header().Set("Location", "https://untrusted.invalid")
				w.WriteHeader(test.status)
				fmt.Fprint(w, test.body)
			}))
			defer server.Close()
			repositories, err := New(server.URL).UserRepositories(t.Context(), "operator", "alice")
			if err == nil || repositories != nil {
				t.Fatalf("partial listing must not look complete: %+v, %v", repositories, err)
			}
		})
	}
}

func TestUserRepositoriesHonorsCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("canceled listing contacted Forgejo")
	}))
	defer server.Close()
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := New(server.URL).UserRepositories(ctx, "operator", "alice"); err == nil {
		t.Fatal("canceled listing succeeded")
	}
}
