package forgejo

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRepositoryByNameBindsNativeRemoteIdentity(t *testing.T) {
	response := `{"id":7,"name":"repo","full_name":"soda-tester/repo","owner":{"id":1,"login":"soda-tester"},"permissions":{"pull":true}}`
	calls := 0
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path != "/api/v1/repos/soda-tester/repo" || r.Header.Get("Authorization") != "token synthetic" {
			t.Error("native repository lookup lost route or account")
		}
		_, _ = fmt.Fprint(w, response)
	}))
	t.Cleanup(s.Close)
	c := New(s.URL)
	if repo, err := c.RepositoryByName(t.Context(), "synthetic", "soda-tester", "repo"); err != nil || repo.ID != 7 {
		t.Fatal("native identity not resolved", err)
	}
	for _, owner := range []string{"", "..", "other/repo", "soda-tester\n"} {
		if _, err := c.RepositoryByName(t.Context(), "synthetic", owner, "repo"); err == nil {
			t.Fatal("invalid native owner admitted")
		}
	}
	if calls != 1 {
		t.Fatal("invalid path reached native API")
	}
	response = `{"id":8,"name":"other","full_name":"soda-tester/other","owner":{"id":1,"login":"soda-tester"}}`
	if _, err := c.RepositoryByName(t.Context(), "synthetic", "soda-tester", "repo"); err == nil {
		t.Fatal("another repository silently replaced requested remote")
	}
}
