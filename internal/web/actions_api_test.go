package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestActionRunsUseNativeCapActorAndLosslessIDs(t *testing.T) {
	calls := 0
	s := grantedTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Header.Get("Authorization") != "token acting-bob" {
			t.Error("wrong acting grant")
		}
		switch r.URL.Path {
		case "/api/v1/settings/api":
			fmt.Fprint(w, `{"max_response_items":2}`)
		case "/api/v1/repos/alice/demo/actions/runs":
			if r.URL.Query().Get("limit") != "2" || r.URL.Query().Get("page") != "2" {
				t.Error("native cap not respected")
			}
			fmt.Fprint(w, `{"total_count":5,"workflow_runs":[{"id":9007199254740993,"index_in_repo":3,"status":"running","event_payload":"fixture-private-input"},{"id":4,"index_in_repo":4}]}`)
		default:
			t.Error("unexpected native target")
			w.WriteHeader(404)
		}
	})
	w := httptest.NewRecorder()
	s.ServeHTTP(w, apiTestRequest("GET", "/api/forgejo/repos/alice/demo/actions/runs?page=2", "", "bob"))
	if w.Code != 200 || calls != 2 || !strings.Contains(w.Body.String(), `"id":"9007199254740993"`) || !strings.Contains(w.Body.String(), `"next_page":3`) || strings.Contains(w.Body.String(), "fixture-private-input") {
		t.Fatal(w.Code, w.Body.String())
	}
}
func TestActionCrossRepositoryDenialNeverRetries(t *testing.T) {
	calls := 0
	s := grantedTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path != "/api/v1/repos/alice/other/actions/runs/17" || r.Header.Get("Authorization") != "token acting-bob" {
			t.Error("run target or actor changed")
		}
		w.WriteHeader(404)
		fmt.Fprint(w, `{"message":"private-run-payload"}`)
	})
	w := httptest.NewRecorder()
	s.ServeHTTP(w, apiTestRequest("GET", "/api/forgejo/repos/alice/other/actions/runs/17", "", "bob"))
	if w.Code != 404 || calls != 1 || strings.Contains(w.Body.String(), "private-run") {
		t.Fatal("denial leaked or retried", w.Code)
	}
}
func TestActionSecretsAreWriteOnlyAndScopeIsRouteBound(t *testing.T) {
	calls := 0
	s := grantedTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Header.Get("Authorization") != "token acting-bob" {
			t.Error("wrong actor")
		}
		switch r.Method {
		case "GET":
			if r.URL.Path != "/api/v1/orgs/team/actions/secrets" {
				t.Error("organization scope changed")
			}
			fmt.Fprint(w, `[{"name":"FIXTURE","created_at":"2026-01-01T00:00:00Z","data":"fixture-hidden-secret","token":"fixture-hidden-token"}]`)
		case "PUT":
			if r.URL.Path != "/api/v1/repos/alice/demo/actions/secrets/FIXTURE" {
				t.Error("query changed repository scope")
			}
			var input map[string]string
			if err := json.NewDecoder(r.Body).Decode(&input); err != nil || input["data"] != "fixture-write-only" || len(input) != 1 {
				t.Error("wrong native secret payload")
			}
			w.WriteHeader(201)
		default:
			t.Error("unexpected operation")
		}
	})
	w := httptest.NewRecorder()
	s.ServeHTTP(w, apiTestRequest("GET", "/api/forgejo/organizations/team/actions/secrets", "", "bob"))
	if w.Code != 200 || strings.Contains(w.Body.String(), "fixture-hidden") {
		t.Fatal("secret readback", w.Code)
	}
	w = httptest.NewRecorder()
	s.ServeHTTP(w, apiTestRequest("PUT", "/api/forgejo/repos/alice/demo/actions/secrets/FIXTURE?org=team", `{"data":"fixture-write-only"}`, "bob"))
	if w.Code != 204 || w.Body.Len() != 0 || calls != 2 {
		t.Fatal("secret write not acknowledged without readback", w.Code)
	}
	for _, body := range []string{`{}`, `{"data":null}`, `{"data":"x","admin_token":"not-allowed"}`} {
		w = httptest.NewRecorder()
		s.ServeHTTP(w, apiTestRequest("PUT", "/api/forgejo/repos/alice/demo/actions/secrets/FIXTURE", body, "bob"))
		if w.Code < 400 || calls != 2 {
			t.Fatal("implicit clear or credential override accepted")
		}
	}
}
func TestActionVariableCreateAndUpdateUseNativeMethods(t *testing.T) {
	calls := 0
	s := grantedTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		want := "POST"
		if calls == 2 {
			want = "PUT"
		}
		if r.Method != want || r.URL.Path != "/api/v1/orgs/team/actions/variables/MODE" {
			t.Error("native variable semantics changed")
		}
		w.WriteHeader(204)
	})
	for _, method := range []string{"POST", "PUT"} {
		w := httptest.NewRecorder()
		s.ServeHTTP(w, apiTestRequest(method, "/api/forgejo/organizations/team/actions/variables/MODE", `{"value":"test"}`, "bob"))
		if w.Code != 204 {
			t.Fatal(w.Code, w.Body.String())
		}
	}
}
func TestWorkflowDispatchUsesExplicitRefAndNeverRetriesFailure(t *testing.T) {
	calls := 0
	s := grantedTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path != "/api/v1/repos/alice/demo/actions/workflows/test.yml/dispatches" || r.Method != "POST" || r.Header.Get("Authorization") != "token acting-bob" {
			t.Error("wrong dispatch authority")
		}
		var input struct {
			Ref        string            `json:"ref"`
			Inputs     map[string]string `json:"inputs"`
			ReturnInfo bool              `json:"return_run_info"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil || input.Ref != "refs/heads/topic" || input.Inputs["target"] != "test" || !input.ReturnInfo {
			t.Error("dispatch input changed")
		}
		w.WriteHeader(500)
		fmt.Fprint(w, `{"message":"fixture-input-private"}`)
	})
	w := httptest.NewRecorder()
	s.ServeHTTP(w, apiTestRequest("POST", "/api/forgejo/repos/alice/demo/actions/workflows/test.yml/dispatch", `{"ref":"refs/heads/topic","inputs":{"target":"test"}}`, "bob"))
	if w.Code < 500 || calls != 1 || strings.Contains(w.Body.String(), "fixture-input-private") {
		t.Fatal("failed dispatch retried or leaked")
	}
}
func TestWorkflowDiscoveryKeepsNativeDirectoryAndNestedNames(t *testing.T) {
	s := grantedTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/repos/alice/demo/contents":
			fmt.Fprint(w, `[{"name":".forgejo","type":"dir"}]`)
		case "/api/v1/repos/alice/demo/contents/.forgejo":
			fmt.Fprint(w, `[{"name":"workflows","type":"dir","sha":"abc"}]`)
		case "/api/v1/repos/alice/demo/git/trees/abc":
			if r.URL.Query().Get("recursive") != "true" {
				t.Error("nested workflows not requested")
			}
			fmt.Fprint(w, `{"page":1,"truncated":false,"tree":[{"path":"nested/test.yml","type":"blob","sha":"def"}]}`)
		default:
			t.Error("native directory selection widened", r.URL.Path)
			w.WriteHeader(404)
		}
	})
	w := httptest.NewRecorder()
	s.ServeHTTP(w, apiTestRequest("GET", "/api/forgejo/repos/alice/demo/actions/workflows?ref=main", "", "bob"))
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"name":"nested/test.yml"`) {
		t.Fatal(w.Code, w.Body.String())
	}
}
