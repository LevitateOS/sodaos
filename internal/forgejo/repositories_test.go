package forgejo

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSearchOwnedRepositories(t *testing.T) {
	valid := `{"ok":true,"data":[{"id":7,"name":"repo","full_name":"alice/repo","owner":{"id":1,"login":"alice"}}]}`
	for _, tc := range []struct {
		name, body string
		accept     bool
	}{
		{"owned", valid, true}, {"empty", `{"ok":true,"data":[]}`, true},
		{"missing", `{}`, false}, {"null", `{"ok":true,"data":null}`, false}, {"false", `{"ok":false,"data":[]}`, false},
		{"foreign owner", strings.Replace(valid, `"id":1,`, `"id":2,`, 1), false},
		{"wrong label", strings.Replace(valid, "alice/repo", "other/repo", 1), false},
		{"oversized", strings.Repeat(" ", 2<<20) + valid, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				q := r.URL.Query()
				if r.URL.Path != "/api/v1/repos/search" || r.Header.Get("Authorization") != "token actor" || q.Get("q") != "x & y" || q.Get("uid") != "1" || q.Get("exclusive") != "true" || q.Get("private") != "true" || q.Get("page") != "2" || q.Get("limit") != "12" {
					t.Error("wrong native search", r.URL)
				}
				fmt.Fprint(w, tc.body)
			}))
			defer server.Close()
			_, err := New(server.URL).SearchOwnedRepositories(t.Context(), "actor", 1, "x & y", 2)
			if (err == nil) != tc.accept {
				t.Fatal(err)
			}
		})
	}
}
func TestSearchRepositoryBounds(t *testing.T) {
	c := New("http://must-not-contact.invalid")
	for _, page := range []int{-1, 0, 101} {
		if _, err := c.SearchOwnedRepositories(t.Context(), "", 1, "", page); err == nil {
			t.Fatal("page accepted")
		}
	}
	for _, query := range []string{strings.Repeat("x", 201), "x\n", "\xff"} {
		if _, err := c.SearchOwnedRepositories(t.Context(), "", 1, query, 1); err == nil {
			t.Fatal("query accepted")
		}
	}
}
