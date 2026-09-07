package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestComparisonUsesResolvedSHAsAndRetainsPerCommitFiles(t *testing.T) {
	base, head := strings.Repeat("a", 40), strings.Repeat("b", 40)
	calls := 0
	s := grantedTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Header.Get("Authorization") != "token acting-bob" {
			t.Error("wrong comparison actor")
		}
		switch calls {
		case 1, 2:
			ref, sha := "feature/one", base
			if calls == 2 {
				ref, sha = "release/two", head
			}
			if r.URL.Path != "/api/v1/repos/alice/demo/git/commits/"+ref || r.URL.Query().Get("files") != "false" {
				t.Error("ref resolution lost identity or requested full files")
			}
			fmt.Fprintf(w, `{"sha":%q}`, sha)
		case 3:
			if r.URL.Path != "/api/v1/repos/alice/demo/compare/"+base+"..."+head {
				t.Error("comparison used moving refs")
			}
			fmt.Fprintf(w, `{"commits":[{"sha":%q}],"total_commits":1,"files":[{"filename":"same.txt","status":"modified"},{"filename":"same.txt","status":"modified"}]}`, head)
		default:
			t.Error("unexpected retry")
		}
	})
	w := httptest.NewRecorder()
	s.ServeHTTP(w, apiTestRequest("GET", "/api/forgejo/repos/alice/demo/compare?base=feature%2Fone&head=release%2Ftwo", "", "bob"))
	if w.Code != 200 || calls != 3 {
		t.Fatalf("comparison: %d, %d calls", w.Code, calls)
	}
	var got struct {
		Base  string `json:"base_sha"`
		Head  string `json:"head_sha"`
		Files []any  `json:"files"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Base != base || got.Head != head || len(got.Files) != 2 {
		t.Fatal("lost pinned identity or deduplicated native per-commit entries")
	}
}

func TestComparisonRejectsMalformedOrIncompleteNativeResults(t *testing.T) {
	sha := strings.Repeat("a", 40)
	for _, body := range []string{`{"commits":[],"files":[],"total_commits":1}`, `{"commits":null,"files":[],"total_commits":0}`, `{"commits":[{"sha":"bad"}],"files":[],"total_commits":1}`} {
		t.Run(body, func(t *testing.T) {
			calls := 0
			s := grantedTestServer(t, func(w http.ResponseWriter, r *http.Request) {
				calls++
				if calls < 3 {
					fmt.Fprintf(w, `{"sha":%q}`, sha)
				} else {
					fmt.Fprint(w, body)
				}
			})
			w := httptest.NewRecorder()
			s.ServeHTTP(w, apiTestRequest("GET", "/api/forgejo/repos/alice/demo/compare?base=main&head=other", "", "bob"))
			if w.Code != 503 || calls != 3 {
				t.Fatalf("incomplete comparison accepted/retried: %d %d", w.Code, calls)
			}
		})
	}
}

func TestComparisonStopsAfterDeniedOrInvalidResolution(t *testing.T) {
	for _, status := range []int{200, 403, 404} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			calls := 0
			s := grantedTestServer(t, func(w http.ResponseWriter, r *http.Request) {
				calls++
				w.WriteHeader(status)
				fmt.Fprint(w, `{"sha":"not-a-commit","message":"private upstream detail"}`)
			})
			w := httptest.NewRecorder()
			s.ServeHTTP(w, apiTestRequest("GET", "/api/forgejo/repos/alice/demo/compare?base=main&head=other", "", "bob"))
			want := status
			if status == 200 {
				want = 503
			}
			if w.Code != want || calls != 1 || strings.Contains(w.Body.String(), "private upstream") {
				t.Fatal("invalid/denied resolution escaped its boundary")
			}
		})
	}
}

func TestRefReadsNeedOnlyReadScopeButWritesStillNeedWriteScope(t *testing.T) {
	for _, kind := range []string{"branches", "tags"} {
		t.Run(kind, func(t *testing.T) {
			calls := 0
			s := grantedTestServer(t, func(w http.ResponseWriter, r *http.Request) { calls++; fmt.Fprint(w, `[]`) })
			grant, err := s.Store.Grant(t.Context(), "session-bob", 2)
			if err != nil {
				t.Fatal(err)
			}
			grant.Scopes = "read:repository"
			if err = s.Store.ReplaceGrant(t.Context(), "session-bob", 2, grant); err != nil {
				t.Fatal(err)
			}
			path := "/api/forgejo/repos/alice/demo/" + kind
			w := httptest.NewRecorder()
			s.ServeHTTP(w, apiTestRequest("GET", path, "", "bob"))
			if w.Code != 200 || calls != 1 {
				t.Fatalf("read scope rejected: %d", w.Code)
			}
			w = httptest.NewRecorder()
			s.ServeHTTP(w, apiTestRequest("POST", path, `{"name":"new","ref":"main"}`, "bob"))
			if w.Code == 201 || calls != 1 {
				t.Fatal("read grant reached a native mutation")
			}
		})
	}
}
