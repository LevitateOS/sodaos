package web

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	extensions "forgejo.org/extension-sdk"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
	"github.com/levitateos/sodaos/internal/tailnet"
)

func TestManagedCreateReviewsPolicyWithoutChangingLegacyDefaults(t *testing.T) {
	for _, kind := range []string{"legacy", "off", "managed", "stale", "closed", "transfer", "native-failure", "network-failure", "binding-changed", "transfer-after-create"} {
		t.Run(kind, func(t *testing.T) {
			owner := "alice"
			s := apiTestServer(t)
			callback := func(in extensions.CallbackRequest) extensions.CallbackResponse {
				if in.Operation == extensions.OperationOrganizationOwner {
					owner := false
					return extensions.CallbackResponse{Owner: &owner}
				}
				if in.Operation != extensions.OperationRepository || in.RepositoryID != "7" {
					t.Error("unexpected native callback", in.Operation, in.RepositoryID)
					return extensions.CallbackResponse{ErrorCode: "invalid_operation"}
				}
				return extensions.CallbackResponse{Repository: &extensions.Repository{ID: "7", Owner: owner, Name: "demo", Permission: "write"}}
			}
			if kind == "transfer-after-create" {
				s.Config.OperatorID = 99
			}
			options, creates, networks := 0, 0, 0
			s.Host.HTTP.Transport = roundTrip(func(r *http.Request) (*http.Response, error) {
				if r.URL.Path == "/profile" {
					return profileTestResponse(), nil
				}
				var value any
				status := 200
				switch r.URL.Path {
				case "/tailnet/options":
					options++
					revision := strings.Repeat("a", 32)
					if kind == "stale" {
						revision = strings.Repeat("c", 32)
					}
					value = tailnet.ProjectOptions{Revision: revision, Binding: strings.Repeat("b", 32), Tailnet: "soda.example.test", Available: kind != "closed", Default: kind != "closed"}
					if kind == "transfer" {
						owner = "bob"
					}
				case "/create":
					creates++
					var in project.Create
					if json.NewDecoder(r.Body).Decode(&in) != nil {
						t.Fatal("invalid helper creation")
					}
					value = project.Environment{ID: in.ID, Running: true, IP: "10.89.0.2", Profile: in.Profile}
					if kind == "native-failure" {
						status = 500
					}
					if kind == "transfer-after-create" {
						owner = "bob"
					}
				case "/tailnet/project":
					networks++
					var in tailnet.ProjectRequest
					if json.NewDecoder(r.Body).Decode(&in) != nil || in.Action != "enable" || in.Revision != "0" || in.Binding != strings.Repeat("b", 32) || in.ConfirmID != in.Project {
						t.Fatal("reviewed selection lost")
					}
					p, e := s.Store.Project(t.Context(), in.Project)
					if e != nil || !p.Ready || creates != 1 {
						t.Fatal("network blocked stored provisioning", e)
					}
					value = tailnet.ProjectView{Project: in.Project, Saved: true, Enabled: true, Revision: strings.Repeat("c", 32), Binding: in.Binding, State: "unconfirmed", Outcome: "queued"}
					if kind == "network-failure" {
						status = 502
					}
					if kind == "binding-changed" {
						status = 409
					}
				default:
					t.Fatal("unexpected helper operation", r.URL.Path)
				}
				b, _ := json.Marshal(value)
				return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(string(b))), Header: make(http.Header)}, nil
			})
			body := `{"repository_id":"7"}`
			if kind == "off" {
				body = `{"repository_id":"7","tailnet":{"enabled":false}}`
			} else if kind != "legacy" {
				body = `{"repository_id":"7","tailnet":{"enabled":true,"revision":"` + strings.Repeat("a", 32) + `","binding":"` + strings.Repeat("b", 32) + `"}}`
			}
			w := httptest.NewRecorder()
			nativeAPIServeWithCallback(t, s, w, apiTestRequest("POST", "/api/environments", body, "alice"), callback)
			stored, e := s.Store.ProjectByRepository(t.Context(), 7)
			switch kind {
			case "stale", "closed", "transfer":
				if w.Code < 400 || creates != 0 || e == nil {
					t.Fatal("preflight failure reserved/created", kind, w.Code, creates, e)
				}
			case "transfer-after-create":
				if (w.Code != 403 && w.Code != 503) || creates != 1 || networks != 0 || e != nil || !stored.Ready {
					t.Fatal("lost authority enrolled or lost provisioned stored", w.Code, e)
				}
			case "native-failure":
				if w.Code != 502 || creates != 1 || networks != 0 || e != nil || stored.Ready {
					t.Fatal("failed reservation lost", w.Code, creates, e)
				}
			default:
				if w.Code != 201 || creates != 1 || e != nil || !stored.Ready {
					t.Fatal("creation failed", kind, w.Code, creates, e)
				}
			}
			expected := 1
			if kind == "legacy" || kind == "off" {
				expected = 0
			}
			if options != expected {
				t.Fatal("implicit policy lookup", options)
			}
			wantNetwork := kind == "managed" || kind == "network-failure" || kind == "binding-changed"
			if (networks == 1) != wantNetwork {
				t.Fatal("implicit or missing enrollment", networks)
			}
			if kind == "network-failure" || kind == "binding-changed" {
				if !strings.Contains(w.Body.String(), `"tailnet_outcome":"unconfirmed"`) {
					t.Fatal("network failure concealed")
				}
				again := httptest.NewRecorder()
				nativeAPIServeWithCallback(t, s, again, apiTestRequest("POST", "/api/environments", body, "alice"), callback)
				if again.Code != 409 || creates != 1 || networks != 1 {
					t.Fatal("network failure recreated stored")
				}
			}
		})
	}
}

func tailnetOffSettings() tailnet.SettingsView {
	return tailnet.SettingsView{HostUnavailable: true, Enrollment: tailnet.EnrollmentView{Revision: "0", Tags: []string{}}}
}

func TestTailnetOperatorAdmissionAndNativeFailureSecrecy(t *testing.T) {
	calls := 0
	fail := false
	s := stubbedHostWebFixture(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if fail {
			w.WriteHeader(409)
			_, _ = fmt.Fprint(w, "synthetic-secret-provider-body")
			return
		}
		if r.URL.Path == "/tailnet/settings" {
			_ = json.NewEncoder(w).Encode(tailnetOffSettings())
			return
		}
		t.Error("unexpected native dispatch", r.URL.Path)
	})
	for _, route := range []string{"/api/settings/tailnet", "/api/settings/tailnet/host", "/api/settings/tailnet/enrollment"} {
		method := "POST"
		if route == "/api/settings/tailnet" {
			method = "GET"
		}
		w := httptest.NewRecorder()
		nativeAPIServe(t, s, w, apiTestRequest(method, route, `{"secret":`, "bob"))
		if w.Code != 403 || calls != 0 {
			t.Fatal("site-admin acquired appliance access", w.Code, calls)
		}
	}
	w := httptest.NewRecorder()
	nativeAPIServe(t, s, w, apiTestRequest("GET", "/api/settings/tailnet", "", "alice"))
	if w.Code != 200 || calls != 1 || w.Header().Get("Referrer-Policy") != "no-referrer" {
		t.Fatal(w.Code, w.Body.String(), calls)
	}
	fail = true
	w = httptest.NewRecorder()
	nativeAPIServe(t, s, w, apiTestRequest("GET", "/api/settings/tailnet", "", "alice"))
	if w.Code != 409 || calls != 2 || strings.Contains(w.Body.String(), "synthetic-secret") {
		t.Fatal(w.Code, w.Body.String(), calls)
	}
}

func TestTailnetMutationStrictFieldsAndRequestGuards(t *testing.T) {
	calls := 0
	s := stubbedHostWebFixture(t, func(w http.ResponseWriter, r *http.Request) { calls++; w.WriteHeader(422) })
	valid := `{"action":"signin","revision":"` + strings.Repeat("a", 64) + `"}`
	for _, header := range []string{"actor", "generation", "origin"} {
		r := apiTestRequest("POST", "/api/settings/tailnet/host", valid, "alice")
		w := httptest.NewRecorder()
		nativeAPIServeWithOptions(t, s, w, r, nil, func(h http.Header) {
			switch header {
			case "actor":
				h.Set(extensions.ContextHeader, strings.Replace(h.Get(extensions.ContextHeader), `"id":"1"`, `"id":"2"`, 1))
			case "generation":
				h.Set(extensions.SessionGenerationHeader, "expired")
			case "origin":
				h.Set("Origin", "https://other.invalid")
			}
		})
		if w.Code < 400 || calls != 0 {
			t.Fatal(header, w.Code, calls)
		}
	}
	for _, body := range []string{`{}`, `null`, strings.TrimSuffix(valid, "}") + `,"socket":"/host.sock"}`, strings.TrimSuffix(valid, "}") + `,"action":"logout"}`, strings.Replace(valid, "signin", "logout", 1)} {
		w := httptest.NewRecorder()
		nativeAPIServe(t, s, w, apiTestRequest("POST", "/api/settings/tailnet/host", body, "alice"))
		if w.Code != 400 || calls != 0 {
			t.Fatal(w.Code, w.Body.String(), calls)
		}
	}
	w := httptest.NewRecorder()
	nativeAPIServe(t, s, w, apiTestRequest("POST", "/api/settings/tailnet/host", valid, "alice"))
	if w.Code != 422 || calls != 1 {
		t.Fatal(w.Code, calls)
	}
}

func TestTailnetEnrollmentNeverEchoesInputAndRejectsEndpointOverride(t *testing.T) {
	calls := 0
	s := stubbedHostWebFixture(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path != "/tailnet/enrollment" {
			t.Error(r.URL.Path)
		}
		var in tailnet.EnrollmentRequest
		if json.NewDecoder(r.Body).Decode(&in) != nil || in.ClientSecret != "tskey-client-synthetic-credential" {
			t.Error("missing protected input")
		}
		_ = json.NewEncoder(w).Encode(tailnet.EnrollmentResult{Outcome: "confirmed", CredentialChecked: true, Enrollment: tailnetOffSettings().Enrollment})
	})
	body := `{"action":"check","revision":"0","tailnet":"soda.example.test","tags":["tag:soda-stored"],"preauthorized":false,"client_id":"synthetic-client","client_secret":"tskey-client-synthetic-credential"}`
	w := httptest.NewRecorder()
	nativeAPIServe(t, s, w, apiTestRequest("POST", "/api/settings/tailnet/enrollment", body, "alice"))
	if w.Code != 200 || calls != 1 || strings.Contains(w.Body.String(), "synthetic-credential") || w.Header().Get("Cache-Control") != "no-store" {
		t.Fatal(w.Code, w.Body.String(), calls)
	}
	for _, bad := range []string{strings.Replace(body, "synthetic-credential", "synthetic-credential?baseURL=https://evil.test", 1), strings.Replace(body, `"soda.example.test"`, `"-"`, 1), strings.TrimSuffix(body, "}") + `,"credential_path":"/etc/shadow"}`} {
		w = httptest.NewRecorder()
		nativeAPIServe(t, s, w, apiTestRequest("POST", "/api/settings/tailnet/enrollment", bad, "alice"))
		if w.Code != 400 || calls != 1 {
			t.Fatal(w.Code, calls)
		}
	}
}

func TestTailnetProjectPrivacyAndCurrentOwnership(t *testing.T) {
	for _, tc := range []struct {
		name             string
		owner            int64
		member, orgOwner bool
		method           string
		want             int
	}{
		{"current owner", 1, false, false, "POST", 200},
		{"renamed member", 2, true, false, "GET", 200},
		{"public reader", 2, false, false, "GET", 403},
		{"transferred creator", 2, false, false, "POST", 403},
		{"member cannot configure", 2, true, false, "POST", 403},
		{"organization owner", 99, false, true, "POST", 200},
		{"organization admin not owner", 99, false, false, "POST", 403},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			s := apiTestServer(t)
			owner := "current"
			if tc.owner == 1 {
				owner = "alice"
			}
			callback := func(in extensions.CallbackRequest) extensions.CallbackResponse {
				switch in.Operation {
				case extensions.OperationRepository:
					return extensions.CallbackResponse{Repository: &extensions.Repository{ID: in.RepositoryID, Owner: owner, Name: "renamed", Permission: "write"}}
				case extensions.OperationOrganizationOwner:
					return extensions.CallbackResponse{Owner: &tc.orgOwner}
				default:
					t.Error("unexpected native callback", in.Operation)
					return extensions.CallbackResponse{ErrorCode: "invalid_operation"}
				}
			}
			s.Config.OperatorID = 777
			id := webTerminalProject
			if e := s.Store.CreateProject(t.Context(), store.Project{ID: id, RepositoryID: 7, OwnerID: 1}); e != nil {
				t.Fatal(e)
			}
			if e := s.Store.MarkReady(t.Context(), id, "10.89.0.2"); e != nil {
				t.Fatal(e)
			}
			if tc.member {
				if e := s.Store.Join(t.Context(), id, 1, "original-linux-login"); e != nil {
					t.Fatal(e)
				}
			}
			s.Host.HTTP.Transport = roundTrip(func(r *http.Request) (*http.Response, error) {
				calls++
				if r.URL.Path != "/tailnet/project" {
					t.Error(r.URL.Path)
				}
				var in tailnet.ProjectRequest
				_ = json.NewDecoder(r.Body).Decode(&in)
				if in.Project != id {
					t.Error("wrong stored")
				}
				outcome := "observed"
				if in.Action == "disable" {
					outcome = "disconnect-unconfirmed"
				}
				revision := "0"
				if in.Action == "disable" {
					revision = strings.Repeat("b", 32)
				}
				b, _ := json.Marshal(tailnet.ProjectView{Saved: in.Action == "disable", Project: id, Revision: revision, State: "runtime-unsupported", Outcome: outcome})
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(string(b))), Header: make(http.Header)}, nil
			})
			body := fmt.Sprintf(`{"action":"disable","revision":"0","confirm_id":%q}`, id)
			w := httptest.NewRecorder()
			nativeAPIServeWithCallback(t, s, w, apiTestRequest(tc.method, "/api/environments/"+id+"/tailnet", body, "alice"), callback)
			if w.Code != tc.want || (calls > 0) != (tc.want == 200) {
				t.Fatal(w.Code, w.Body.String(), calls)
			}
		})
	}
}

func TestTailnetCreationOptionsRequireCurrentHumanOwner(t *testing.T) {
	for _, owner := range []int64{1, 2} {
		t.Run(fmt.Sprint(owner), func(t *testing.T) {
			calls := 0
			s := apiTestServer(t)
			callback := func(in extensions.CallbackRequest) extensions.CallbackResponse {
				if in.Operation != extensions.OperationRepository || in.RepositoryID != "7" {
					t.Error("unexpected native callback", in.Operation, in.RepositoryID)
					return extensions.CallbackResponse{ErrorCode: "invalid_operation"}
				}
				login := "current"
				if owner == 1 {
					login = "alice"
				}
				return extensions.CallbackResponse{Repository: &extensions.Repository{ID: "7", Owner: login, Name: "project", Permission: "write"}}
			}
			s.Config.OperatorID = 777
			s.Host.HTTP.Transport = roundTrip(func(r *http.Request) (*http.Response, error) {
				calls++
				if r.URL.Path != "/tailnet/options" {
					t.Error(r.URL.Path)
				}
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"revision":"0","available":false,"default":false}`)), Header: make(http.Header)}, nil
			})
			w := httptest.NewRecorder()
			nativeAPIServeWithCallback(t, s, w, apiTestRequest("GET", "/api/repositories/7/tailnet-options", "", "alice"), callback)
			want := 403
			if owner == 1 {
				want = 200
			}
			if w.Code != want || (calls > 0) != (owner == 1) || strings.Contains(w.Body.String(), "credential") || strings.Contains(w.Body.String(), "peers") {
				t.Fatal(w.Code, w.Body.String(), calls)
			}
		})
	}
}
