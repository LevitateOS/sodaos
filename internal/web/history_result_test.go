package web

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSameRefComparisonResolvesOnceAndAcceptsEmptyNativeResult(t *testing.T) {
	sha := strings.Repeat("a", 40)
	calls := 0
	s := grantedTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			fmt.Fprintf(w, `{"sha":%q}`, sha)
			return
		}
		if r.URL.Path != "/api/v1/repos/alice/demo/compare/"+sha+"..."+sha {
			t.Error("same ref resolved twice or compared mutable name")
		}
		fmt.Fprint(w, `{"commits":[],"files":[],"total_commits":0}`)
	})
	w := httptest.NewRecorder()
	s.ServeHTTP(w, apiTestRequest("GET", "/api/forgejo/repos/alice/demo/compare?base=main&head=main", "", "bob"))
	if w.Code != 200 || calls != 2 {
		t.Fatalf("same ref: %d %d", w.Code, calls)
	}
}

func TestHistoryRejectsWrongCommitAndMalformedRefCreationResults(t *testing.T) {
	for _, request := range []struct{ method, path, body string }{
		{"GET", "/api/forgejo/repos/alice/demo/commits/" + strings.Repeat("a", 40), ""},
		{"POST", "/api/forgejo/repos/alice/demo/branches", `{"name":"feature/one","ref":"main"}`},
		{"POST", "/api/forgejo/repos/alice/demo/tags", `{"name":"v1","target":"main"}`},
	} {
		t.Run(request.path, func(t *testing.T) {
			calls := 0
			s := grantedTestServer(t, func(w http.ResponseWriter, r *http.Request) {
				calls++
				fmt.Fprintf(w, `{"sha":%q}`, strings.Repeat("b", 40))
			})
			w := httptest.NewRecorder()
			s.ServeHTTP(w, apiTestRequest(request.method, request.path, request.body, "bob"))
			if w.Code != 503 || calls != 1 {
				t.Fatalf("bad native result accepted/retried: %d %d", w.Code, calls)
			}
		})
	}
}
