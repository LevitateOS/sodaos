package forgejo

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMyRepositoriesUsesBoundedActingUserPages(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != "GET" || r.URL.Path != "/api/v1/user/repos" || r.URL.Query().Get("limit") != "50" || r.Header.Get("Authorization") != "token acting-alice" {
			t.Error("wrong actor/request")
		}
		switch r.URL.Query().Get("page") {
		case "1":
			// Smaller native page limit; only metadata, never follow this URL.
			w.Header().Set("Link", `<https://ignored.invalid/?page=2>; rel="next"`)
			fmt.Fprint(w, `[{"id":1,"full_name":"alice/one","owner":{"id":7}}]`)
		case "2":
			fmt.Fprint(w, `[{"id":2,"full_name":"alice/two","owner":{"id":7}}]`)
		default:
			t.Error("unexpected page")
			w.WriteHeader(500)
		}
	}))
	defer server.Close()
	client := New(server.URL)
	repos, page, err := client.MyRepositories(t.Context(), "acting-alice", 1)
	if err != nil || calls != 1 || len(repos) != 1 || page.NextPage == nil || *page.NextPage != 2 {
		t.Fatal(repos, page, calls, err)
	}
	repos, page, err = client.MyRepositories(t.Context(), "acting-alice", *page.NextPage)
	if err != nil || calls != 2 || len(repos) != 1 || page.NextPage != nil || repos[0].FullName != "alice/two" || repos[0].Owner.ID != 7 {
		t.Fatal(repos, page, calls, err)
	}
}

func TestMyRepositoriesRejectsFailedPage(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		body   string
	}{
		{"provider failure", 503, "unavailable"}, {"malformed response", 200, "{"}, {"redirect", 302, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Location", "https://untrusted.invalid")
				w.WriteHeader(tc.status)
				fmt.Fprint(w, tc.body)
			}))
			defer server.Close()
			repos, _, err := New(server.URL).MyRepositories(t.Context(), "acting-alice", 2)
			if err == nil || repos != nil {
				t.Fatal("failed page looked complete", repos, err)
			}
		})
	}
}

func TestMyRepositoriesHonorsCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { t.Error("canceled listing contacted Forgejo") }))
	defer server.Close()
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, _, err := New(server.URL).MyRepositories(ctx, "acting-alice", 1); err == nil {
		t.Fatal("canceled listing succeeded")
	}
}
