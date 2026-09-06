package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHookResponseNeverReturnsNativeCredentials(t *testing.T) {
	s := grantedTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/repos/alice/demo/hooks" || r.Header.Get("Authorization") != "token acting-bob" {
			t.Error("unexpected hook request")
		}
		fmt.Fprint(w, `[{"id":9007199254740993,"type":"forgejo","active":true,"events":["push"],"url":"https://hooks.example/path?credential=fixture-query","authorization_header":"fixture-header","config":{"url":"fixture-url","secret":"fixture-signing-secret"}}]`)
	})
	w := httptest.NewRecorder()
	s.ServeHTTP(w, apiTestRequest("GET", "/api/forgejo/repos/alice/demo/hooks", "", "bob"))
	if w.Code != 200 || strings.Contains(w.Body.String(), "fixture-") || strings.Contains(w.Body.String(), "https://hooks") || !strings.Contains(w.Body.String(), `"has_authorization":true`) || !strings.Contains(w.Body.String(), `"id":"9007199254740993"`) {
		t.Fatal("hook metadata leaked credentials or lost identifier")
	}
}
func TestHookPatchPreservesOmittedSecretsAndNativeSettings(t *testing.T) {
	writes := 0
	s := grantedTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/repos/alice/demo/hooks/9" {
			t.Error("Soda called a hook target or wrong provider resource")
		}
		if r.Method == "GET" {
			fmt.Fprint(w, `{"id":9,"type":"forgejo","active":true,"events":["push","package"],"branch_filter":"main","url":"https://hooks.example/path?credential=fixture-query","authorization_header":"fixture-header"}`)
			return
		}
		writes++
		var input struct {
			Authorization string            `json:"authorization_header"`
			Events        []string          `json:"events"`
			Branch        string            `json:"branch_filter"`
			Active        bool              `json:"active"`
			Config        map[string]string `json:"config"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			t.Error(err)
		}
		if input.Authorization != "fixture-header" || input.Branch != "main" || len(input.Events) != 2 || input.Active || !strings.Contains(input.Config["url"], "fixture-query") {
			t.Error("native PATCH silently reset omitted fields")
		}
		if _, present := input.Config["secret"]; present {
			t.Error("pretended to rotate an unsupported signing secret")
		}
		fmt.Fprint(w, `{"id":9,"type":"forgejo","active":false,"events":["push","package"],"authorization_header":"fixture-header"}`)
	})
	w := httptest.NewRecorder()
	s.ServeHTTP(w, apiTestRequest("PATCH", "/api/forgejo/repos/alice/demo/hooks/9", `{"active":false}`, "bob"))
	if w.Code != 200 || writes != 1 || strings.Contains(w.Body.String(), "fixture-") {
		t.Fatal("hook preservation failed", w.Code)
	}
	w = httptest.NewRecorder()
	s.ServeHTTP(w, apiTestRequest("PATCH", "/api/forgejo/repos/alice/demo/hooks/9", `{"events":["push"]}`, "bob"))
	if w.Code != 422 || writes != 1 {
		t.Fatal("unsupported native package flag edit was reported as applied")
	}
}
func TestHookRejectsUnsafeOrPrivilegedConfigurationBeforeNativeMutation(t *testing.T) {
	calls := 0
	s := grantedTestServer(t, func(w http.ResponseWriter, r *http.Request) { calls++; w.WriteHeader(403) })
	for _, body := range []string{`{"url":"javascript:alert(1)","events":["push"]}`, `{"url":"https://user:password@example.test/","events":["push"]}`, `{"url":"https://example.test/","events":["push"],"is_system_webhook":true}`, `{"url":"https://example.test/","events":["push"],"authorization":"header\r\ninjected"}`} {
		w := httptest.NewRecorder()
		s.ServeHTTP(w, apiTestRequest("POST", "/api/forgejo/repos/alice/demo/hooks", body, "bob"))
		if w.Code != 400 && w.Code != 422 {
			t.Fatal("unsafe hook input accepted", w.Code)
		}
	}
	if calls != 0 {
		t.Fatal("invalid configuration reached upstream")
	}
	w := httptest.NewRecorder()
	s.ServeHTTP(w, apiTestRequest("POST", "/api/forgejo/repos/alice/demo/hooks", `{"url":"https://example.test/","events":["push"],"active":false}`, "bob"))
	if w.Code != 403 || calls != 1 {
		t.Fatal("native hook denial hidden or retried", w.Code)
	}
}
