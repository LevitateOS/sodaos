package web

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/store"
	"golang.org/x/crypto/ssh"
)

func managementWebFixture(t *testing.T) (*Server, *[]string) {
	t.Helper()
	s := grantedTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		uid := 1
		if r.Header.Get("Authorization") == "token acting-bob" {
			uid = 2
		}
		switch r.URL.Path {
		case "/api/v1/user":
			fmt.Fprintf(w, `{"id":%d,"login":"current-login"}`, uid)
		case "/api/v1/repositories/7":
			fmt.Fprint(w, `{"id":7,"name":"repo","full_name":"alice/repo","owner":{"id":1,"login":"alice"}}`)
		case "/api/v1/users/current-login/orgs/alice/permissions":
			fmt.Fprint(w, `{"is_owner":false}`)
		default:
			t.Error("unexpected provider operation")
			w.WriteHeader(500)
		}
	})
	s.Config.OperatorID = 777
	if err := s.Store.CreateProject(t.Context(), store.Project{ID: webTerminalProject, RepositoryID: 7, OwnerID: 1}); err != nil {
		t.Fatal(err)
	}
	if err := s.Store.MarkReady(t.Context(), webTerminalProject, "10.89.0.2"); err != nil {
		t.Fatal(err)
	}
	for _, user := range []struct {
		id    int64
		login string
	}{{1, "original-alice"}, {2, "original-bob"}} {
		if err := s.Store.Join(t.Context(), webTerminalProject, user.id, user.login); err != nil {
			t.Fatal(err)
		}
	}
	calls := []string{}
	s.Host.HTTP = &http.Client{Transport: roundTrip(func(r *http.Request) (*http.Response, error) {
		calls = append(calls, r.URL.Path)
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		response := `{"revision":"` + strings.Repeat("a", 64) + `","keys":[]}`
		switch r.URL.Path {
		case "/lifecycle":
			action := body["action"].(string)
			running := action != "stop"
			if body["project"] != webTerminalProject {
				t.Error("untrusted lifecycle target")
			}
			response = fmt.Sprintf(`{"environment":{"id":%q,"running":%t},"boot_enabled":%t}`, webTerminalProject, running, running)
		case "/access-keys":
			if body["identity"] != float64(1) || body["login"] != "original-alice" {
				t.Error("key update remapped native identity")
			}
			if body["apply"] == true {
				keys := body["keys"]
				if keys == nil {
					keys = []string{}
				}
				b, _ := json.Marshal(keys)
				response = `{"revision":"` + strings.Repeat("a", 64) + `","keys":` + string(b) + `}`
			}
		default:
			t.Error("unexpected helper call")
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(response))}, nil
	})}
	return s, &calls
}
func TestLifecycleAuthorizationAndExplicitStop(t *testing.T) {
	for _, tc := range []struct {
		login, body string
		code        int
	}{{"bob", `{"action":"stop","confirm_stop":true}`, 403}, {"alice", `{"action":"stop"}`, 400}, {"alice", `{"action":"destroy","confirm_stop":true}`, 400}, {"alice", `{"action":"start","project":"other"}`, 400}, {"alice", `{"action":"start"}`, 200}, {"alice", `{"action":"stop","confirm_stop":true}`, 200}} {
		t.Run(tc.login+tc.body, func(t *testing.T) {
			s, calls := managementWebFixture(t)
			w := httptest.NewRecorder()
			s.ServeHTTP(w, apiTestRequest("POST", "/api/environments/"+webTerminalProject+"/lifecycle", tc.body, tc.login))
			if w.Code != tc.code {
				t.Fatal(w.Code, w.Body.String())
			}
			if (len(*calls) > 0) != (tc.code == 200) {
				t.Fatal("denial reached native mutation")
			}
		})
	}
}
func TestSavedKeyRemovalIsOwnOnlyAndNeverNativeRevocation(t *testing.T) {
	s, calls := managementWebFixture(t)
	if err := s.Store.AddKey(t.Context(), 1, "synthetic-public", "synthetic-fingerprint"); err != nil {
		t.Fatal(err)
	}
	keys, _ := s.Store.Keys(t.Context(), 1)
	path := fmt.Sprintf("/api/me/development-keys/%d", keys[0].ID)
	for _, login := range []string{"bob", "alice"} {
		w := httptest.NewRecorder()
		s.ServeHTTP(w, apiTestRequest("DELETE", path, `{}`, login))
		want := 200
		if login == "bob" {
			want = 404
		}
		if w.Code != want {
			t.Fatal(w.Code, w.Body.String())
		}
		if login == "alice" && !strings.Contains(w.Body.String(), `"existing_project_access_changed":false`) {
			t.Fatal("preference deletion claimed revocation")
		}
	}
	if len(*calls) != 0 {
		t.Fatal("saved key removal changed native access")
	}
}
func TestOperatorCannotManageAnotherAccountsKeysAndCSRFStillApplies(t *testing.T) {
	s, calls := managementWebFixture(t)
	other := "pabcdef0123456789abcdef01"
	if err := s.Store.CreateProject(t.Context(), store.Project{ID: other, RepositoryID: 8, OwnerID: 1}); err != nil {
		t.Fatal(err)
	}
	if err := s.Store.MarkReady(t.Context(), other, "10.89.0.3"); err != nil {
		t.Fatal(err)
	}
	s.Config.OperatorID = 1
	w := httptest.NewRecorder()
	s.ServeHTTP(w, apiTestRequest("GET", "/api/environments/"+other+"/access-keys", "", "alice"))
	if w.Code != 403 || len(*calls) != 0 {
		t.Fatal("operator bypassed own membership")
	}
	for _, path := range []string{"/api/environments/" + webTerminalProject + "/access-keys", "/api/environments/" + webTerminalProject + "/lifecycle"} {
		r := apiTestRequest("POST", path, `{}`, "alice")
		r.Header.Set("X-CSRF-Token", "wrong")
		w := httptest.NewRecorder()
		s.ServeHTTP(w, r)
		if w.Code != 403 || len(*calls) != 0 {
			t.Fatal("new control bypassed CSRF")
		}
	}
}

func TestAccessKeyPreviewApplyAndLastKeyConfirmation(t *testing.T) {
	s, calls := managementWebFixture(t)
	path := "/api/environments/" + webTerminalProject + "/access-keys"
	public, _, _ := ed25519.GenerateKey(rand.Reader)
	key, _ := ssh.NewPublicKey(public)
	canonical := string(ssh.MarshalAuthorizedKey(key))
	fingerprint := ssh.FingerprintSHA256(key)
	if err := s.Store.AddKey(t.Context(), 1, canonical, fingerprint); err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	s.ServeHTTP(w, apiTestRequest("GET", path, "", "alice"))
	if w.Code != 200 || !strings.Contains(w.Body.String(), "original-alice") {
		t.Fatal(w.Code, w.Body.String())
	}
	good := fmt.Sprintf(`{"revision":%q,"saved_fingerprints":[%q]}`, strings.Repeat("a", 64), fingerprint)
	for _, body := range []string{strings.Replace(good, fingerprint, "changed", 1), strings.Replace(good, "saved_fingerprints", "keys", 1)} {
		n := len(*calls)
		w := httptest.NewRecorder()
		s.ServeHTTP(w, apiTestRequest("POST", path, body, "alice"))
		if w.Code != 400 && w.Code != 409 {
			t.Fatal(w.Code, w.Body.String())
		}
		if len(*calls) != n {
			t.Fatal("stale/caller-selected key set reached native")
		}
	}
	w = httptest.NewRecorder()
	s.ServeHTTP(w, apiTestRequest("POST", path, good, "alice"))
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	keys, _ := s.Store.Keys(t.Context(), 1)
	_, _ = s.Store.RemoveKey(t.Context(), 1, keys[0].ID)
	for _, confirm := range []bool{false, true} {
		n := len(*calls)
		body := fmt.Sprintf(`{"revision":%q,"saved_fingerprints":[],"confirm_empty":%t}`, strings.Repeat("a", 64), confirm)
		w := httptest.NewRecorder()
		s.ServeHTTP(w, apiTestRequest("POST", path, body, "alice"))
		want := 400
		if confirm {
			want = 200
		}
		if w.Code != want {
			t.Fatal(w.Code, w.Body.String())
		}
		if !confirm && n != len(*calls) {
			t.Fatal("last key removed without confirmation")
		}
	}
}
