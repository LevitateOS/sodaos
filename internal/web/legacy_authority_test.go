package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestLegacyPeopleDelegatesNativeAuthorityAndValidation(t *testing.T) {
	for _, tc := range []struct {
		name, actor, login string
		native, want       int
	}{
		{"operator is not site admin", "alice", "person", 403, 403},
		{"site admin is not operator", "bob", "Native.Name", 201, 303},
		{"native account rejection", "bob", "root", 422, 422},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			s := grantedTestServer(t, func(w http.ResponseWriter, r *http.Request) {
				calls++
				if r.Method != "POST" || r.URL.Path != "/api/v1/admin/users" || r.Header.Get("Authorization") != "token acting-"+tc.actor {
					t.Error("wrong native authority")
				}
				var body map[string]any
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
				}
				if body["username"] != tc.login || body["password"] != "short" || body["must_change_password"] != true {
					t.Error("independent account policy or altered native request")
				}
				w.WriteHeader(tc.native)
				fmt.Fprint(w, `{"id":55,"login":"Native.Name","message":"private-provider-error"}`)
			})
			form := url.Values{"csrf": {"csrf-" + tc.actor}, "login": {tc.login}, "email": {"person@example.test"}, "password": {"short"}}
			r := httptest.NewRequest("POST", "/people", strings.NewReader(form.Encode()))
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			r.Header.Set("Origin", s.Config.PublicURL)
			r.AddCookie(&http.Cookie{Name: "soda_session", Value: "session-" + tc.actor})
			w := httptest.NewRecorder()
			s.ServeHTTP(w, r)
			if w.Code != tc.want || calls != 1 || strings.Contains(w.Body.String(), "private-provider-error") {
				t.Fatalf("%d calls=%d %s", w.Code, calls, w.Body.String())
			}
			if _, err := s.Store.User(t.Context(), 55); err == nil {
				t.Fatal("native account copied into Soda before OAuth")
			}
		})
	}
}

func TestLegacyPeopleReadsNativePageAndPreservesCSRF(t *testing.T) {
	calls := 0
	s := grantedTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != "GET" || r.URL.Path != "/api/v1/admin/users" || r.URL.Query().Get("page") != "2" || r.Header.Get("Authorization") != "token acting-bob" {
			t.Error("wrong people request")
		}
		w.Header().Set("Link", `<https://ignored.example/?page=3>; rel="next"`)
		fmt.Fprint(w, `[{"id":55,"login":"Native.Name"}]`)
	})
	w := httptest.NewRecorder()
	s.ServeHTTP(w, apiTestRequest("GET", "/people?page=2", "", "bob"))
	if w.Code != 200 || !strings.Contains(w.Body.String(), "Native.Name") || !strings.Contains(w.Body.String(), "/people?page=3") {
		t.Fatal(w.Code, w.Body.String())
	}
	for _, removed := range []string{"minlength=", "pattern=", "Lowercase"} {
		if strings.Contains(w.Body.String(), removed) {
			t.Fatal("independent account rule retained")
		}
	}
	r := httptest.NewRequest("POST", "/people", strings.NewReader("csrf=wrong&login=person&email=person%40example.test&password=short"))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.AddCookie(&http.Cookie{Name: "soda_session", Value: "session-bob"})
	w = httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != 403 || calls != 1 {
		t.Fatal("CSRF denial called provider")
	}
}

func TestLegacyProjectProviderDenialDoesNotProvision(t *testing.T) {
	calls := 0
	s := grantedTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Header.Get("Authorization") != "token acting-alice" {
			t.Error("wrong authority")
		}
		w.WriteHeader(403)
	})
	s.Host.HTTP = &http.Client{Transport: roundTrip(func(*http.Request) (*http.Response, error) { t.Fatal("denial reached host"); return nil, nil })}
	r := httptest.NewRequest("POST", "/projects", strings.NewReader("csrf=csrf-alice&repository=alice%2Fprivate"))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.AddCookie(&http.Cookie{Name: "soda_session", Value: "session-alice"})
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	projects, err := s.Store.Projects(t.Context())
	if w.Code != 403 || calls != 1 || err != nil || len(projects) != 0 {
		t.Fatal("denial created reservation", w.Code, err)
	}
}
