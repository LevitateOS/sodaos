package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const pullHead = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
const pullBase = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
const pullAncestor = "cccccccccccccccccccccccccccccccccccccccc"

func TestMergeRequiresDisplayedHeadAndNeverForcesOrTouchesEnvironment(t *testing.T) {
	calls := 0
	s := grantedTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Header.Get("Authorization") != "token acting-bob" {
			t.Error("merge actor changed")
		}
		if r.Method == "GET" {
			fmt.Fprintf(w, `{"head":{"sha":%q},"base":{"sha":%q},"merge_base":%q}`, pullHead, pullBase, pullAncestor)
			return
		}
		if r.URL.Path != "/api/v1/repos/alice/demo/pulls/12/merge" {
			t.Error("unexpected native mutation")
		}
		var body struct {
			Head     string `json:"head_commit_id"`
			Force    bool   `json:"force_merge"`
			Delete   bool   `json:"delete_branch_after_merge"`
			Auto     bool   `json:"merge_when_checks_succeed"`
			Strategy string `json:"Do"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if body.Head != pullHead || body.Force || body.Delete || body.Auto || body.Strategy != "merge" {
			t.Error("merge precondition or forbidden side effect changed")
		}
		w.WriteHeader(405)
		fmt.Fprint(w, `{"message":"sensitive protected branch detail"}`)
	})
	s.Host.HTTP = &http.Client{Transport: roundTrip(func(r *http.Request) (*http.Response, error) {
		t.Error("merge touched environment state")
		return nil, fmt.Errorf("forbidden")
	})}
	path := "/api/forgejo/repos/alice/demo/pulls/12/merge"
	body := fmt.Sprintf(`{"head":%q,"base":%q,"merge_base":%q,"strategy":"merge","force_merge":true}`, pullHead, pullBase, pullAncestor)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, apiTestRequest("POST", path, body, "bob"))
	if w.Code != 400 || calls != 0 {
		t.Fatal("force input accepted")
	}
	body = fmt.Sprintf(`{"head":%q,"base":%q,"merge_base":%q,"strategy":"merge"}`, pullHead, pullBase, pullAncestor)
	w = httptest.NewRecorder()
	s.ServeHTTP(w, apiTestRequest("POST", path, body, "bob"))
	if w.Code != 409 || calls != 2 || strings.Contains(w.Body.String(), "sensitive") {
		t.Fatal("native protection denial hidden or retried", w.Code, calls)
	}
}
func TestReviewRejectsChangedSnapshotAndPinsExplicitCommit(t *testing.T) {
	changed := true
	writes := 0
	s := grantedTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			head := pullHead
			if changed {
				head = pullAncestor
			}
			fmt.Fprintf(w, `{"head":{"sha":%q},"base":{"sha":%q},"merge_base":%q}`, head, pullBase, pullAncestor)
			return
		}
		writes++
		var body struct {
			Commit   string `json:"commit_id"`
			Event    string `json:"event"`
			Comments []struct {
				Path    string `json:"path"`
				Line    int64  `json:"new_position"`
				OldLine int64  `json:"old_position"`
			} `json:"comments"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if body.Commit != pullHead || body.Event != "COMMENT" || len(body.Comments) != 1 || body.Comments[0].Path != "src/main.go" || body.Comments[0].Line != 42 || body.Comments[0].OldLine != 0 {
			t.Error("review revision or native new-side mapping changed")
		}
		fmt.Fprintf(w, `{"id":9007199254740993,"commit_id":%q,"state":"COMMENT","user":{"id":2,"login":"bob"}}`, pullHead)
	})
	path := "/api/forgejo/repos/alice/demo/pulls/12/reviews"
	body := fmt.Sprintf(`{"head":%q,"base":%q,"merge_base":%q,"event":"COMMENT","body":"Review","comments":[{"path":"src/main.go","line":42,"body":"Why?"}]}`, pullHead, pullBase, pullAncestor)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, apiTestRequest("POST", path, body, "bob"))
	if w.Code != 409 || writes != 0 {
		t.Fatal("stale review mutated native state", w.Code)
	}
	changed = false
	w = httptest.NewRecorder()
	s.ServeHTTP(w, apiTestRequest("POST", path, body, "bob"))
	if w.Code != 201 || writes != 1 || !strings.Contains(w.Body.String(), `"id":"9007199254740993"`) {
		t.Fatal(w.Code, w.Body.String())
	}
}
func TestPullDiffNeverReturnsARevisionChangedDuringRead(t *testing.T) {
	reads := 0
	s := grantedTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, ".diff") {
			fmt.Fprint(w, "untrusted diff from racing head")
			return
		}
		reads++
		head := pullHead
		if reads > 1 {
			head = pullAncestor
		}
		fmt.Fprintf(w, `{"head":{"sha":%q},"base":{"sha":%q},"merge_base":%q}`, head, pullBase, pullAncestor)
	})
	path := "/api/forgejo/repos/alice/demo/pulls/12/diff?head=" + pullHead + "&base=" + pullBase + "&merge_base=" + pullAncestor
	w := httptest.NewRecorder()
	s.ServeHTTP(w, apiTestRequest("GET", path, "", "bob"))
	if w.Code != 409 || strings.Contains(w.Body.String(), "untrusted diff") {
		t.Fatal("mixed revision diff returned", w.Code)
	}
}
