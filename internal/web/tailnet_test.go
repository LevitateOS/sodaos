package web

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/config"
	"github.com/levitateos/sodaos/internal/store"
	"github.com/levitateos/sodaos/internal/tailnet"
)

func tailnetOffSettings() tailnet.SettingsView {
	return tailnet.SettingsView{HostUnavailable: true, Enrollment: tailnet.EnrollmentView{Revision: "0", Tags: []string{}}}
}
func TestTailnetOperatorAdmissionAndNativeFailureSecrecy(t *testing.T) {
	calls := 0
	fail := false
	s := runnerWebFixture(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if fail {
			w.WriteHeader(409)
			fmt.Fprint(w, "synthetic-secret-provider-body")
			return
		}
		if r.URL.Path == "/tailnet/settings" {
			json.NewEncoder(w).Encode(tailnetOffSettings())
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
		s.ServeHTTP(w, apiTestRequest(method, route, `{"secret":`, "bob"))
		if w.Code != 403 || calls != 0 {
			t.Fatal("site-admin acquired appliance access", w.Code, calls)
		}
	}
	w := httptest.NewRecorder()
	s.ServeHTTP(w, apiTestRequest("GET", "/api/settings/tailnet", "", "alice"))
	if w.Code != 200 || calls != 1 || w.Header().Get("Referrer-Policy") != "no-referrer" {
		t.Fatal(w.Code, w.Body.String(), calls)
	}
	fail = true
	w = httptest.NewRecorder()
	s.ServeHTTP(w, apiTestRequest("GET", "/api/settings/tailnet", "", "alice"))
	if w.Code != 409 || calls != 2 || strings.Contains(w.Body.String(), "synthetic-secret") {
		t.Fatal(w.Code, w.Body.String(), calls)
	}
}
func TestTailnetMutationStrictFieldsAndRequestGuards(t *testing.T) {
	calls := 0
	s := runnerWebFixture(t, func(w http.ResponseWriter, r *http.Request) { calls++; w.WriteHeader(422) })
	valid := `{"action":"signin","revision":"` + strings.Repeat("a", 64) + `"}`
	for _, header := range []string{"X-Soda-Expected-User-ID", "X-CSRF-Token", "Origin"} {
		r := apiTestRequest("POST", "/api/settings/tailnet/host", valid, "alice")
		r.Header.Del(header)
		w := httptest.NewRecorder()
		s.ServeHTTP(w, r)
		if w.Code < 400 || calls != 0 {
			t.Fatal(header, w.Code, calls)
		}
	}
	for _, body := range []string{`{}`, `null`, strings.TrimSuffix(valid, "}") + `,"socket":"/host.sock"}`, strings.TrimSuffix(valid, "}") + `,"action":"logout"}`, strings.Replace(valid, "signin", "logout", 1)} {
		w := httptest.NewRecorder()
		s.ServeHTTP(w, apiTestRequest("POST", "/api/settings/tailnet/host", body, "alice"))
		if w.Code != 400 || calls != 0 {
			t.Fatal(w.Code, w.Body.String(), calls)
		}
	}
	w := httptest.NewRecorder()
	s.ServeHTTP(w, apiTestRequest("POST", "/api/settings/tailnet/host", valid, "alice"))
	if w.Code != 422 || calls != 1 {
		t.Fatal(w.Code, calls)
	}
}
func TestTailnetEnrollmentNeverEchoesInputAndRejectsEndpointOverride(t *testing.T) {
	calls := 0
	s := runnerWebFixture(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path != "/tailnet/enrollment" {
			t.Error(r.URL.Path)
		}
		var in tailnet.EnrollmentRequest
		if json.NewDecoder(r.Body).Decode(&in) != nil || in.ClientSecret != "tskey-client-synthetic-credential" {
			t.Error("missing protected input")
		}
		json.NewEncoder(w).Encode(tailnet.EnrollmentResult{Outcome: "confirmed", CredentialChecked: true, Enrollment: tailnetOffSettings().Enrollment})
	})
	body := `{"action":"check","revision":"0","tailnet":"soda.example.test","tags":["tag:soda-project"],"preauthorized":false,"client_id":"synthetic-client","client_secret":"tskey-client-synthetic-credential"}`
	w := httptest.NewRecorder()
	s.ServeHTTP(w, apiTestRequest("POST", "/api/settings/tailnet/enrollment", body, "alice"))
	if w.Code != 200 || calls != 1 || strings.Contains(w.Body.String(), "synthetic-credential") || w.Header().Get("Cache-Control") != "no-store" {
		t.Fatal(w.Code, w.Body.String(), calls)
	}
	for _, bad := range []string{strings.Replace(body, "synthetic-credential", "synthetic-credential?baseURL=https://evil.test", 1), strings.Replace(body, `"soda.example.test"`, `"-"`, 1), strings.TrimSuffix(body, "}") + `,"credential_path":"/etc/shadow"}`} {
		w = httptest.NewRecorder()
		s.ServeHTTP(w, apiTestRequest("POST", "/api/settings/tailnet/enrollment", bad, "alice"))
		if w.Code != 400 || calls != 1 {
			t.Fatal(w.Code, calls)
		}
	}
}
func TestTailnetContextChangeSuppressesReadsAndDispatch(t *testing.T) {
	for _, phase := range []string{"provider", "native"} {
		t.Run(phase, func(t *testing.T) {
			var s *Server
			calls := 0
			s = grantedTestServer(t, func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprint(w, `{"id":1,"login":"alice"}`)
			})
			// Use the request's actual fixture session, not a guessed cookie name/value.
			request := apiTestRequest("GET", "/api/settings/tailnet", "", "alice")
			cookie, _ := requestCookie(request, sessionCookie)
			if phase == "provider" {
				s.Forgejo.HTTP.Transport = roundTrip(func(r *http.Request) (*http.Response, error) {
					v, e := s.Store.Session(t.Context(), cookie.Value)
					if e != nil {
						t.Fatal(e)
					}
					if e = s.Store.EndLoginContext(t.Context(), v.ContextID); e != nil {
						t.Fatal(e)
					}
					return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"id":1,"login":"alice"}`)), Header: make(http.Header)}, nil
				})
			}
			s.Host.HTTP.Transport = roundTrip(func(r *http.Request) (*http.Response, error) {
				calls++
				v, e := s.Store.Session(t.Context(), cookie.Value)
				if e != nil {
					t.Fatal(e)
				}
				if e = s.Store.EndLoginContext(t.Context(), v.ContextID); e != nil {
					t.Fatal(e)
				}
				b, _ := json.Marshal(tailnetOffSettings())
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(string(b))), Header: make(http.Header)}, nil
			})
			w := httptest.NewRecorder()
			s.ServeHTTP(w, request)
			if w.Code < 400 || (phase == "provider" && calls != 0) || (phase == "native" && calls != 1) || strings.Contains(w.Body.String(), "host_unavailable") {
				t.Fatal(w.Code, w.Body.String(), calls)
			}
		})
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
			s := grantedTestServer(t, func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/api/v1/user":
					fmt.Fprint(w, `{"id":1,"login":"renamed-alice","is_admin":true}`)
				case "/api/v1/repositories/7":
					fmt.Fprintf(w, `{"id":7,"name":"renamed","full_name":"current/renamed","owner":{"id":%d,"login":"current"}}`, tc.owner)
				case "/api/v1/users/renamed-alice/orgs/current/permissions":
					fmt.Fprintf(w, `{"is_owner":%t,"is_admin":true}`, tc.orgOwner)
				default:
					t.Error("unexpected provider path", r.URL.Path)
				}
			})
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
				json.NewDecoder(r.Body).Decode(&in)
				if in.Project != id {
					t.Error("wrong project")
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
			s.ServeHTTP(w, apiTestRequest(tc.method, "/api/environments/"+id+"/tailnet", body, "alice"))
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
			s := grantedTestServer(t, func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/api/v1/user" {
					fmt.Fprint(w, `{"id":1,"login":"alice","is_admin":true}`)
					return
				}
				if r.URL.Path != "/api/v1/repositories/7" {
					t.Error(r.URL.Path)
				}
				fmt.Fprintf(w, `{"id":7,"name":"project","full_name":"current/project","owner":{"id":%d,"login":"current"}}`, owner)
			})
			s.Config.OperatorID = 777
			s.Host.HTTP.Transport = roundTrip(func(r *http.Request) (*http.Response, error) {
				calls++
				if r.URL.Path != "/tailnet/options" {
					t.Error(r.URL.Path)
				}
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"revision":"0","available":false,"default":false}`)), Header: make(http.Header)}, nil
			})
			w := httptest.NewRecorder()
			s.ServeHTTP(w, apiTestRequest("GET", "/api/repositories/7/tailnet-options", "", "alice"))
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
func TestTailnetFixedBookmarkAndOAuthReturn(t *testing.T) {
	s := apiTestServer(t)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, httptest.NewRequest("GET", config.SodaPath+"/settings/tailnet", nil))
	if w.Code != 303 || !strings.Contains(w.Header().Get("Location"), "redirect_to=%2F%3Fsoda-view%3Dtailnet") {
		t.Fatal(w.Code, w.Header().Get("Location"))
	}
	for _, query := range []string{"destination=tailnet&repository_id=7", "destination=tailnet&destination=runners", "destination=tailnet&return_to=https://evil.test"} {
		w = httptest.NewRecorder()
		s.ServeHTTP(w, httptest.NewRequest("GET", config.SodaPath+"/login?"+query, nil))
		if w.Code != 400 {
			t.Fatal(query, w.Code)
		}
	}
	w = httptest.NewRecorder()
	s.ServeHTTP(w, httptest.NewRequest("GET", config.SodaPath+"/login?destination=tailnet&expected_user_id=1", nil))
	location, e := url.Parse(w.Header().Get("Location"))
	if e != nil || w.Code != 302 {
		t.Fatal(w.Code, e)
	}
	a, e := s.Store.ConsumeOAuth(t.Context(), location.Query().Get("state"), "")
	if e != nil || a.SettingsReturn != "tailnet" || a.ExpectedUserID != 1 || s.nativeOAuthReturn(a.OAuthLogin) != s.Config.ForgejoURL+"/?soda-view=tailnet" {
		t.Fatal(a, e)
	}
}
