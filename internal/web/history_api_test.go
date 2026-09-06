package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFileUpdatesBindLoadedSHAAndPreserveNativeConflict(t *testing.T) {
	calls := 0
	sha := strings.Repeat("a", 40)
	s := grantedTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path != "/api/v1/repos/alice/demo/contents/docs/hello world.txt" || r.Method != "PUT" || r.Header.Get("Authorization") != "token acting-bob" {
			t.Error("wrong native file request")
		}
		var input struct {
			SHA     string `json:"sha"`
			Branch  string `json:"branch"`
			Content string `json:"content"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			t.Error(err)
		}
		if input.SHA != sha || input.Branch != "feature/one" || input.Content != "bmV3" {
			t.Error("lost write precondition")
		}
		w.WriteHeader(409)
		fmt.Fprint(w, `{"message":"private stale file detail"}`)
	})
	path := "/api/forgejo/repos/alice/demo/files?ref=feature%2Fone&path=docs%2Fhello+world.txt"
	w := httptest.NewRecorder()
	s.ServeHTTP(w, apiTestRequest("PUT", path, fmt.Sprintf(`{"sha":%q,"content":"bmV3","message":"Update"}`, sha), "bob"))
	if w.Code != 409 || calls != 1 || strings.Contains(w.Body.String(), "private stale") {
		t.Fatal("conflict ignored/retried", w.Code)
	}
	w = httptest.NewRecorder()
	s.ServeHTTP(w, apiTestRequest("PUT", path, `{"content":"bmV3","message":"Update"}`, "bob"))
	if w.Code != 422 || calls != 1 {
		t.Fatal("missing precondition reached native")
	}
}
func TestProtectedBranchDenialIsNotRetried(t *testing.T) {
	calls := 0
	s := grantedTestServer(t, func(w http.ResponseWriter, r *http.Request) { calls++; w.WriteHeader(403) })
	w := httptest.NewRecorder()
	s.ServeHTTP(w, apiTestRequest("POST", "/api/forgejo/repos/alice/demo/files?ref=main&path=new.txt", `{"content":"bmV3","message":"Create"}`, "alice"))
	if w.Code != 403 || calls != 1 {
		t.Fatal("native protection bypassed", w.Code)
	}
}
func TestImportOwnerCannotBeSelected(t *testing.T) {
	calls := 0
	s := grantedTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path != "/api/v1/repos/migrate" || r.Header.Get("Authorization") != "token acting-bob" {
			t.Error("invalid import request")
		}
		var input map[string]any
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			t.Error(err)
		}
		if input["repo_owner"] != "bob" || input["service"] != "git" || input["mirror"] != nil {
			t.Error("import ownership/behavior changed")
		}
		fmt.Fprint(w, `{"id":8,"name":"copy","owner":{"id":2,"login":"bob"}}`)
	})
	for _, body := range []string{`{"url":"https://git.example/repo.git","name":"copy","owner":"alice"}`, `{"url":"https://secret:password@git.example/repo.git","name":"copy"}`} {
		w := httptest.NewRecorder()
		s.ServeHTTP(w, apiTestRequest("POST", "/api/forgejo/repositories/import", body, "bob"))
		if w.Code != 400 && w.Code != 422 {
			t.Fatal("invalid import accepted", w.Code)
		}
	}
	if calls != 0 {
		t.Fatal("invalid import reached provider")
	}
	w := httptest.NewRecorder()
	s.ServeHTTP(w, apiTestRequest("POST", "/api/forgejo/repositories/import", `{"url":"https://git.example/repo.git","name":"copy","private":true}`, "bob"))
	if w.Code != 201 || calls != 1 {
		t.Fatal(w.Code, w.Body.String())
	}
}
