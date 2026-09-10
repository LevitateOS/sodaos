package web

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/levitateos/sodaos/internal/projectos"
	"io"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"
)

func testCreationProfile() projectos.Profile {
	return projectos.Profile{ID: projectos.RockyHeadless, Distribution: "rocky", Version: "10.2", Interface: "headless", Architecture: runtime.GOARCH, Image: "sha256:" + strings.Repeat("a", 64), Revision: strings.Repeat("b", 40)}
}
func profileTestResponse() *http.Response {
	raw, _ := json.Marshal(testCreationProfile())
	return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(string(raw)))}
}

func TestProfileReadKeepsOwnerAndRequestGuards(t *testing.T) {
	s := grantedTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/user" {
			if strings.Contains(r.Header.Get("Authorization"), "bob") {
				fmt.Fprint(w, `{"id":2,"login":"bob","is_admin":true}`)
			} else {
				fmt.Fprint(w, `{"id":1,"login":"alice"}`)
			}
		} else {
			fmt.Fprint(w, `{"id":7,"name":"demo","full_name":"alice/demo","owner":{"id":1,"login":"alice"}}`)
		}
	})
	s.Config.OperatorID = 2
	calls := 0
	s.Host.HTTP = &http.Client{Transport: roundTrip(func(r *http.Request) (*http.Response, error) {
		calls++
		raw, _ := io.ReadAll(r.Body)
		if r.Method != "POST" || r.URL.Path != "/profile" || strings.TrimSpace(string(raw)) != "{}" {
			t.Error("unbounded native profile request")
		}
		return profileTestResponse(), nil
	})}
	for _, tc := range []struct {
		path, login string
		want        int
	}{
		{"/api/repositories/7/profiles", "alice", 200},
		{"/api/repositories/7/profiles", "bob", 403},
		{"/api/repositories/07/profiles", "alice", 400},
		{"/api/repositories/7/profiles?image=other", "alice", 400},
		{"/api/repositories/7/profiles?", "alice", 400},
	} {
		before := calls
		w := httptest.NewRecorder()
		s.ServeHTTP(w, apiTestRequest("GET", tc.path, "", tc.login))
		if w.Code != tc.want || (tc.want != 200 && calls != before) {
			t.Fatal(tc, w.Code, calls)
		}
	}
}

func TestProfilePreflightNeverReservesOnUnavailableOrLogout(t *testing.T) {
	for _, kind := range []string{"unavailable", "logout", "transfer", "wrong architecture response", "wrong profile"} {
		t.Run(kind, func(t *testing.T) {
			owner := 1
			s := grantedTestServer(t, func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/api/v1/user" {
					fmt.Fprint(w, `{"id":1,"login":"alice"}`)
				} else {
					fmt.Fprintf(w, `{"id":7,"name":"demo","full_name":"alice/demo","owner":{"id":%d,"login":"alice"}}`, owner)
				}
			})
			s.Host.HTTP = &http.Client{Transport: roundTrip(func(r *http.Request) (*http.Response, error) {
				if r.URL.Path != "/profile" {
					t.Error("unexpected mutation", r.URL.Path)
				}
				switch kind {
				case "unavailable":
					return nil, errors.New("not installed")
				case "logout":
					if err := s.Store.DeleteSession(t.Context(), "session-alice"); err != nil {
						t.Error(err)
					}
				case "transfer":
					owner = 2
				case "wrong architecture response":
					return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"id":"rocky-headless","architecture":"other"}`))}, nil
				}
				return profileTestResponse(), nil
			})}
			body := `{"repository_id":"7"}`
			if kind == "wrong profile" {
				body = `{"repository_id":"7","profile_id":"fedora-kde"}`
			}
			w := httptest.NewRecorder()
			s.ServeHTTP(w, apiTestRequest("POST", "/api/environments", body, "alice"))
			if w.Code < 400 {
				t.Fatal(w.Code, w.Body.String())
			}
			if _, err := s.Store.ProjectByRepository(t.Context(), 7); err == nil {
				t.Fatal("preflight failure reserved a project")
			}
		})
	}
}
