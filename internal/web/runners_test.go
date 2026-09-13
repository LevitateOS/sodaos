package web

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/host"
	"github.com/levitateos/sodaos/internal/runners"
)

func runnerWebFixture(t *testing.T, native http.HandlerFunc) *Server {
	t.Helper()
	s := grantedTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/user" {
			t.Error("unexpected provider request", r.URL.Path)
		}
		if strings.Contains(r.Header.Get("Authorization"), "bob") {
			fmt.Fprint(w, `{"id":2,"login":"bob","is_admin":true}`)
		} else {
			fmt.Fprint(w, `{"id":1,"login":"alice","is_admin":false}`)
		}
	})
	peer := httptest.NewServer(native)
	t.Cleanup(peer.Close)
	s.Host = &host.Client{HTTP: peer.Client()}
	transport := peer.Client().Transport
	s.Host.HTTP.Transport = roundTrip(func(r *http.Request) (*http.Response, error) {
		r.URL.Scheme = "http"
		r.URL.Host = strings.TrimPrefix(peer.URL, "http://")
		return transport.RoundTrip(r)
	})
	return s
}
func TestRunnerOperatorGatesBeforeNativeAndDecode(t *testing.T) {
	calls := 0
	s := runnerWebFixture(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		fmt.Fprint(w, `{"runners":[],"unavailable":[]}`)
	})
	for _, path := range []string{"/api/settings/runners", "/api/settings/runners/one/remove"} {
		method := "GET"
		if strings.HasSuffix(path, "remove") {
			method = "POST"
		}
		w := httptest.NewRecorder()
		s.ServeHTTP(w, apiTestRequest(method, path, `{"bad":`, "bob"))
		if w.Code != 403 || calls != 0 || !strings.Contains(w.Body.String(), "operator_required") {
			t.Fatal(path, w.Code, w.Body.String(), calls)
		}
	}
	w := httptest.NewRecorder()
	s.ServeHTTP(w, apiTestRequest("GET", "/api/settings/runners", "", "alice"))
	if w.Code != 200 || calls != 1 {
		t.Fatal(w.Code, w.Body.String(), calls)
	}
}

func TestRunnerAPIRejectsUnsupportedProviders(t *testing.T) {
	for _, provider := range []runners.Provider{"github", "unknown"} {
		t.Run(string(provider), func(t *testing.T) {
			calls := 0
			s := runnerWebFixture(t, func(w http.ResponseWriter, r *http.Request) {
				calls++
				_ = json.NewEncoder(w).Encode(runners.Inventory{Runners: []runners.RunnerView{{Descriptor: runners.Descriptor{ID: "legacy", Provider: provider, Account: "soda-runner-legacy"}, Capacity: 1}}, Unavailable: []string{}})
			})
			input, err := json.Marshal(runners.CreateRequest{ID: "one", Provider: provider, RegistrationURL: "https://external.example.test/repo", RegistrationID: "33834eef-e758-48c4-a676-1745426747aa", Labels: "soda:host", RegistrationToken: "synthetic-input"})
			if err != nil {
				t.Fatal(err)
			}
			w := httptest.NewRecorder()
			s.ServeHTTP(w, apiTestRequest("POST", "/api/settings/runners", string(input), "alice"))
			if w.Code != 400 || calls != 0 {
				t.Fatal("unsupported registration reached native code", w.Code, calls)
			}
			w = httptest.NewRecorder()
			s.ServeHTTP(w, apiTestRequest("GET", "/api/settings/runners", "", "alice"))
			if w.Code != 503 || strings.Contains(w.Body.String(), "soda-runner-legacy") {
				t.Fatal("unsupported inventory was exposed", w.Code, w.Body.String())
			}
		})
	}
}
func TestRunnerRegistrationFixedOriginAndSanitizedFailure(t *testing.T) {
	calls := 0
	fail := false
	s := runnerWebFixture(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path != "/runners/create" {
			t.Error(r.URL.Path)
		}
		var in runners.CreateRequest
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			t.Error(err)
		}
		if in.RegistrationURL == "https://attacker.test" || in.RegistrationToken != "synthetic-registration-secret" {
			t.Error("incorrect native input")
		}
		if fail {
			w.WriteHeader(500)
			fmt.Fprint(w, "synthetic-registration-secret")
			return
		}
		fmt.Fprint(w, `{"ok":true}`)
	})
	body := `{"id":"one","provider":"forgejo","registration_url":"https://attacker.test","registration_id":"33834eef-e758-48c4-a676-1745426747aa","labels":"native:host","registration_token":"synthetic-registration-secret"}`
	for _, failure := range []bool{false, true} {
		fail = failure
		w := httptest.NewRecorder()
		s.ServeHTTP(w, apiTestRequest("POST", "/api/settings/runners", body, "alice"))
		want := 200
		if fail {
			want = 502
		}
		if w.Code != want || strings.Contains(w.Body.String(), "synthetic-registration-secret") || w.Header().Get("Cache-Control") != "no-store" {
			t.Fatal(w.Code, w.Body.String())
		}
	}
	if calls != 2 {
		t.Fatal("unexpected retry", calls)
	}
	for _, mutation := range []string{strings.Replace(body, "native:host", "native:docker://image", 1), strings.Replace(body, `"forgejo"`, `"evil"`, 1), strings.Replace(body, `"id":"one"`, `"id":"../one"`, 1)} {
		w := httptest.NewRecorder()
		s.ServeHTTP(w, apiTestRequest("POST", "/api/settings/runners", mutation, "alice"))
		if w.Code != 400 || calls != 2 {
			t.Fatal(w.Code, calls)
		}
	}
}
func TestRunnerLifecycleConfirmationActorAndCSRF(t *testing.T) {
	calls := 0
	s := runnerWebFixture(t, func(w http.ResponseWriter, r *http.Request) { calls++; fmt.Fprint(w, `{"ok":true}`) })
	for _, action := range []string{"start", "stop", "restart", "remove"} {
		for _, body := range []string{`{}`, `{"confirm_id":""}`, `{"confirm_id":"other"}`, `{"confirm_id":"one"}`, `{"unit":"sshd"}`} {
			w := httptest.NewRecorder()
			s.ServeHTTP(w, apiTestRequest("POST", "/api/settings/runners/one/"+action, body, "alice"))
			want := 400
			if action == "remove" && body == `{"confirm_id":"one"}` || action != "remove" && body == `{}` {
				want = 200
			}
			if w.Code != want {
				t.Fatal(action, w.Code)
			}
		}
	}
	if calls != 4 {
		t.Fatal(calls)
	}
	for _, action := range []string{"start", "stop", "restart", "remove"} {
		for _, header := range []string{"X-Soda-Expected-User-ID", "X-CSRF-Token", "Origin"} {
			body := `{}`
			if action == "remove" {
				body = `{"confirm_id":"one"}`
			}
			r := apiTestRequest("POST", "/api/settings/runners/one/"+action, body, "alice")
			r.Header.Del(header)
			w := httptest.NewRecorder()
			s.ServeHTTP(w, r)
			if w.Code < 400 || calls != 4 {
				t.Fatal(action, header, w.Code, calls)
			}
		}
	}
}
func TestRunnerListPublicOriginAndUnavailableNotEmpty(t *testing.T) {
	fail := false
	s := runnerWebFixture(t, func(w http.ResponseWriter, r *http.Request) {
		if fail {
			w.WriteHeader(503)
			return
		}
		_ = json.NewEncoder(w).Encode(runners.Inventory{Runners: []runners.RunnerView{{Descriptor: runners.Descriptor{ID: "legacy", Provider: runners.ProviderForgejo, RegistrationURL: "http://internal-only:3000", Account: "soda-runner-legacy", Architecture: "x86-64"}, Version: "runner", Capacity: 1, Service: &runners.ServiceState{Load: "loaded", Active: "active", Sub: "running", Enabled: "enabled"}}}, Unavailable: []string{}})
	})
	w := httptest.NewRecorder()
	s.ServeHTTP(w, apiTestRequest("GET", "/api/settings/runners", "", "alice"))
	if w.Code != 200 || strings.Contains(w.Body.String(), "internal-only") || !strings.Contains(w.Body.String(), `"active_listeners":1`) {
		t.Fatal(w.Code, w.Body.String())
	}
	fail = true
	w = httptest.NewRecorder()
	s.ServeHTTP(w, apiTestRequest("GET", "/api/settings/runners", "", "alice"))
	if w.Code != 503 || strings.Contains(w.Body.String(), `"runners":[]`) {
		t.Fatal(w.Code, w.Body.String())
	}
}
func TestRunnerPartialInventoryKeepsValidatedRowsAndQualifiesCounts(t *testing.T) {
	inventory := runners.Inventory{Runners: []runners.RunnerView{{Descriptor: runners.Descriptor{ID: "one", Provider: runners.ProviderForgejo, Account: "soda-runner-one"}, Capacity: 1}}, Unavailable: []string{"two"}}
	s := runnerWebFixture(t, func(w http.ResponseWriter, r *http.Request) { _ = json.NewEncoder(w).Encode(inventory) })
	w := httptest.NewRecorder()
	s.ServeHTTP(w, apiTestRequest("GET", "/api/settings/runners", "", "alice"))
	var result runners.ListResponse
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if w.Code != 200 || result.Complete || result.RunnerCount != 1 || result.TotalCapacity != 1 || result.ActiveListeners != 0 || len(result.Unavailable) != 1 || result.Runners[0].Service != nil {
		t.Fatal(w.Code, w.Body.String())
	}
	for _, unavailable := range [][]string{{"one"}, {"../two"}, {"two", "two"}} {
		inventory.Unavailable = unavailable
		w = httptest.NewRecorder()
		s.ServeHTTP(w, apiTestRequest("GET", "/api/settings/runners", "", "alice"))
		if w.Code != 503 {
			t.Fatal(w.Code, w.Body.String())
		}
	}
}

func TestRunnerAuthorizationLogoutDuringProviderCheck(t *testing.T) {
	calls := 0
	s := runnerWebFixture(t, func(w http.ResponseWriter, r *http.Request) { calls++; fmt.Fprint(w, `[]`) })
	s.Forgejo.HTTP = &http.Client{Transport: roundTrip(func(r *http.Request) (*http.Response, error) {
		v, err := s.Store.Session(context.Background(), "session-alice")
		if err != nil {
			t.Fatal(err)
		}
		if err = s.Store.EndLoginContext(context.Background(), v.ContextID); err != nil {
			t.Fatal(err)
		}
		response := httptest.NewRecorder()
		response.WriteHeader(200)
		_, _ = response.Write(bytes.NewBufferString(`{"id":1,"login":"alice"}`).Bytes())
		return response.Result(), nil
	})}
	w := httptest.NewRecorder()
	s.ServeHTTP(w, apiTestRequest("GET", "/api/settings/runners", "", "alice"))
	if w.Code != 401 || calls != 0 {
		t.Fatal(w.Code, calls)
	}
}
