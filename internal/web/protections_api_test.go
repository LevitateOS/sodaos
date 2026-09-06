package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestProtectionPatchDoesNotClearOmittedNativeRules(t *testing.T) {
	calls := 0
	s := grantedTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != "PATCH" || r.Header.Get("Authorization") != "token acting-bob" || !strings.HasSuffix(r.URL.EscapedPath(), "/branch_protections/release%2F%2A") {
			t.Error("wrong protection target/actor")
		}
		var body map[string]json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if len(body) != 1 || string(body["required_approvals"]) != "2" {
			t.Error("omitted native protection was cleared")
		}
		fmt.Fprint(w, `{"rule_name":"release/*","required_approvals":2}`)
	})
	path := "/api/forgejo/repos/alice/demo/branch-protections/release%2F%2A"
	w := httptest.NewRecorder()
	s.ServeHTTP(w, apiTestRequest("PATCH", path, `{"required_approvals":2}`, "bob"))
	if w.Code != 200 || calls != 1 {
		t.Fatal(w.Code, calls)
	}
	w = httptest.NewRecorder()
	s.ServeHTTP(w, apiTestRequest("PATCH", path, `{"enable_push_whitelist":true}`, "bob"))
	if w.Code != 422 || calls != 1 {
		t.Fatal("otherwise-ignored native dependency accepted")
	}
}
func TestTagProtectionPreservesOpaqueIDAndNativeDenial(t *testing.T) {
	calls := 0
	s := grantedTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method == "GET" {
			fmt.Fprint(w, `[{"id":9007199254740993,"name_pattern":"v*","whitelist_usernames":["alice"],"whitelist_teams":[]}]`)
			return
		}
		w.WriteHeader(403)
	})
	path := "/api/forgejo/repos/alice/demo/tag-protections"
	w := httptest.NewRecorder()
	s.ServeHTTP(w, apiTestRequest("GET", path, "", "bob"))
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"id":"9007199254740993"`) {
		t.Fatal("tag ID lost")
	}
	w = httptest.NewRecorder()
	s.ServeHTTP(w, apiTestRequest("PATCH", path+"/9007199254740993", `{"whitelist_usernames":["bob"]}`, "bob"))
	if w.Code != 403 || calls != 2 {
		t.Fatal("native protection denial hidden or retried")
	}
}
