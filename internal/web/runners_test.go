package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	extensions "forgejo.org/extension-sdk"
	"github.com/levitateos/sodaos/internal/host"
	"github.com/levitateos/sodaos/internal/runners"
)

func runnerWebFixture(t *testing.T, native http.HandlerFunc) *Server {
	t.Helper()
	s := apiTestServer(t)
	peer := httptest.NewServer(native)
	t.Cleanup(peer.Close)
	s.SetHost(&host.Client{HTTP: peer.Client()})
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
		_, _ = fmt.Fprint(w, `{"runners":[],"unavailable":[]}`)
	})
	for _, path := range []string{"/api/settings/runners", "/api/settings/runners/one/remove"} {
		method := "GET"
		if strings.HasSuffix(path, "remove") {
			method = "POST"
		}
		w := httptest.NewRecorder()
		nativeAPIServe(t, s, w, apiTestRequest(method, path, `{"bad":`, "bob"))
		if w.Code != 403 || calls != 0 || !strings.Contains(w.Body.String(), "operator_required") {
			t.Fatal(path, w.Code, w.Body.String(), calls)
		}
	}
	w := httptest.NewRecorder()
	nativeAPIServe(t, s, w, apiTestRequest("GET", "/api/settings/runners", "", "alice"))
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
			nativeAPIServe(t, s, w, apiTestRequest("POST", "/api/settings/runners", string(input), "alice"))
			if w.Code != 503 || calls != 0 {
				t.Fatal("unsupported registration reached native code", w.Code, calls)
			}
			w = httptest.NewRecorder()
			nativeAPIServe(t, s, w, apiTestRequest("GET", "/api/settings/runners", "", "alice"))
			if w.Code != 503 || strings.Contains(w.Body.String(), "soda-runner-legacy") {
				t.Fatal("unsupported inventory was exposed", w.Code, w.Body.String())
			}
		})
	}
}

func TestRunnerExecutionUnavailableBeforeNativeDispatch(t *testing.T) {
	calls := 0
	s := runnerWebFixture(t, func(http.ResponseWriter, *http.Request) { calls++ })
	for _, operation := range runnerAPIRequests {
		if operation.name != "create" && operation.name != "start" && operation.name != "restart" {
			continue
		}
		w := httptest.NewRecorder()
		nativeAPIServe(t, s, w, apiTestRequest(operation.method, operation.path, operation.body, "alice"))
		if w.Code != 503 || calls != 0 || !strings.Contains(w.Body.String(), "runner_execution_unavailable") || strings.Contains(w.Body.String(), "synthetic-runner-secret") {
			t.Fatal(w.Code, w.Body.String(), calls)
		}
	}
}

func TestRunnerLifecycleConfirmationActorAndGeneration(t *testing.T) {
	calls := 0
	s := runnerWebFixture(t, func(w http.ResponseWriter, r *http.Request) { calls++; _, _ = fmt.Fprint(w, `{"ok":true}`) })
	for _, action := range []string{"stop", "remove"} {
		for _, body := range []string{`{}`, `{"confirm_id":""}`, `{"confirm_id":"other"}`, `{"confirm_id":"one"}`, `{"unit":"sshd"}`} {
			w := httptest.NewRecorder()
			nativeAPIServe(t, s, w, apiTestRequest("POST", "/api/settings/runners/one/"+action, body, "alice"))
			want := 400
			if action == "remove" && body == `{"confirm_id":"one"}` || action != "remove" && body == `{}` {
				want = 200
			}
			if w.Code != want {
				t.Fatal(action, w.Code)
			}
		}
	}
	if calls != 2 {
		t.Fatal(calls)
	}
	for _, action := range []string{"stop", "remove"} {
		for _, header := range []string{"actor", "generation", "origin"} {
			body := `{}`
			if action == "remove" {
				body = `{"confirm_id":"one"}`
			}
			r := apiTestRequest("POST", "/api/settings/runners/one/"+action, body, "alice")
			w := httptest.NewRecorder()
			nativeAPIServeWithOptions(t, s, w, r, nil, func(h http.Header) {
				switch header {
				case "actor":
					h.Set(extensions.ContextHeader, strings.Replace(h.Get(extensions.ContextHeader), `"id":"1"`, `"id":"2"`, 1))
				case "generation":
					h.Set(extensions.SessionGenerationHeader, "changed")
				case "origin":
					h.Del("Origin")
				}
			})
			if w.Code < 400 || calls != 2 {
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
	nativeAPIServe(t, s, w, apiTestRequest("GET", "/api/settings/runners", "", "alice"))
	if w.Code != 200 || strings.Contains(w.Body.String(), "internal-only") || !strings.Contains(w.Body.String(), `"active_listeners":1`) {
		t.Fatal(w.Code, w.Body.String())
	}
	fail = true
	w = httptest.NewRecorder()
	nativeAPIServe(t, s, w, apiTestRequest("GET", "/api/settings/runners", "", "alice"))
	if w.Code != 503 || strings.Contains(w.Body.String(), `"runners":[]`) {
		t.Fatal(w.Code, w.Body.String())
	}
}

func TestRunnerPartialInventoryKeepsValidatedRowsAndQualifiesCounts(t *testing.T) {
	inventory := runners.Inventory{Runners: []runners.RunnerView{{Descriptor: runners.Descriptor{ID: "one", Provider: runners.ProviderForgejo, Account: "soda-runner-one"}, Capacity: 1}}, Unavailable: []string{"two"}}
	s := runnerWebFixture(t, func(w http.ResponseWriter, r *http.Request) { _ = json.NewEncoder(w).Encode(inventory) })
	w := httptest.NewRecorder()
	nativeAPIServe(t, s, w, apiTestRequest("GET", "/api/settings/runners", "", "alice"))
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
		nativeAPIServe(t, s, w, apiTestRequest("GET", "/api/settings/runners", "", "alice"))
		if w.Code != 503 {
			t.Fatal(w.Code, w.Body.String())
		}
	}
}

func TestRunnerAuthorizationRejectsExpiredGenerationBeforeInventory(t *testing.T) {
	calls := 0
	s := runnerWebFixture(t, func(w http.ResponseWriter, r *http.Request) { calls++; _, _ = fmt.Fprint(w, `[]`) })
	w := httptest.NewRecorder()
	nativeAPIServeWithOptions(t, s, w, apiTestRequest("GET", "/api/settings/runners", "", "alice"), nil, func(h http.Header) {
		h.Set(extensions.SessionGenerationHeader, "expired")
	})
	if w.Code != 409 || calls != 0 {
		t.Fatal(w.Code, calls)
	}
}
