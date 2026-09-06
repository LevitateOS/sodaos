package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRepositorySettingsCannotRenameTransferOrTouchProjects(t *testing.T) {
	calls := 0
	s := grantedTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != "PATCH" || r.URL.Path != "/api/v1/repos/alice/demo" || r.Header.Get("Authorization") != "token acting-bob" {
			t.Error("wrong settings actor/target")
		}
		var body map[string]json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if len(body) != 1 || string(body["description"]) != `"Updated"` {
			t.Error("unchanged settings were submitted")
		}
		fmt.Fprint(w, `{"id":10,"name":"demo","full_name":"alice/demo","owner":{"id":1,"login":"alice"},"description":"Updated","default_branch":"main"}`)
	})
	s.Host.HTTP = &http.Client{Transport: roundTrip(func(r *http.Request) (*http.Response, error) {
		t.Error("repository settings touched project helper")
		return nil, fmt.Errorf("forbidden")
	})}
	path := "/api/forgejo/repos/alice/demo/settings"
	for _, body := range []string{`{"name":"renamed"}`, `{"new_owner":"bob"}`, `{"archived":true}`, `{"website":"javascript:alert(1)"}`} {
		w := httptest.NewRecorder()
		s.ServeHTTP(w, apiTestRequest("PATCH", path, body, "bob"))
		if w.Code != 400 && w.Code != 422 {
			t.Fatal("unsupported ownership/destructive/unsafe change accepted", w.Code)
		}
	}
	if calls != 0 {
		t.Fatal("unsupported settings reached native provider")
	}
	w := httptest.NewRecorder()
	s.ServeHTTP(w, apiTestRequest("PATCH", path, `{"description":"Updated"}`, "bob"))
	if w.Code != 200 || calls != 1 {
		t.Fatal(w.Code)
	}
}
func TestCollaboratorAdministrationUsesNativeDenialNotSodaAuthority(t *testing.T) {
	calls := 0
	s := grantedTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Header.Get("Authorization") != "token acting-alice" || r.URL.Path != "/api/v1/repos/bob/demo/collaborators/alice" {
			t.Error("collaborator actor changed")
		}
		w.WriteHeader(403)
	})
	w := httptest.NewRecorder()
	s.ServeHTTP(w, apiTestRequest("PUT", "/api/forgejo/repos/bob/demo/collaborators/alice", `{"permission":"admin"}`, "alice"))
	if w.Code != 403 || calls != 1 {
		t.Fatal("Soda operator overrode native collaborator denial", w.Code)
	}
}
func TestDeployKeysRejectPrivateInputAndRequireExplicitMode(t *testing.T) {
	calls := 0
	s := grantedTestServer(t, func(w http.ResponseWriter, r *http.Request) { calls++; w.WriteHeader(403) })
	path := "/api/forgejo/repos/alice/demo/deploy-keys"
	w := httptest.NewRecorder()
	s.ServeHTTP(w, apiTestRequest("POST", path, `{"title":"Not a public key","key":"-----BEGIN PRIVATE KEY-----\ninvalid fixture","read_only":true}`, "bob"))
	if w.Code != 422 || calls != 0 {
		t.Fatal("private-key-shaped input reached provider")
	}
	w = httptest.NewRecorder()
	s.ServeHTTP(w, apiTestRequest("DELETE", path+"/9", `{}`, "bob"))
	if w.Code != 403 || calls != 1 {
		t.Fatal("native deploy-key denial hidden or retried", w.Code)
	}
}
