package web

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/host"
	"github.com/levitateos/sodaos/internal/store"
)

func TestBrowserOnlyJoinAndExplicitSavedSSHChoice(t *testing.T) {
	for _, tc := range []struct {
		name, body       string
		saved, fail      bool
		wantKeys, status int
	}{
		{"new browser-only", `{"ssh_keys":"none"}`, false, false, 0, 200},
		{"browser-only ignores saved", `{"ssh_keys":"none"}`, true, false, 0, 200},
		{"explicit saved", `{"ssh_keys":"saved"}`, true, false, 1, 200},
		{"legacy empty", `{}`, false, false, 0, 200},
		{"legacy saved", `{}`, true, false, 1, 200},
		{"account-only native failure", `{"ssh_keys":"none"}`, false, true, 0, 502},
		{"invalid selection", `{"ssh_keys":"all-provider-keys"}`, true, false, 0, 400},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := grantedTestServer(t, func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/api/v1/user":
					fmt.Fprint(w, `{"id":2,"login":"bob"}`)
				case "/api/v1/repositories/7":
					fmt.Fprint(w, `{"id":7,"name":"demo","full_name":"alice/demo","owner":{"id":1,"login":"alice"}}`)
				default:
					t.Error(r.URL.Path)
					w.WriteHeader(503)
				}
			})
			id := "p0123456789abcdef01234567"
			if err := s.Store.CreateProject(t.Context(), store.Project{ID: id, RepositoryID: 7, OwnerID: 1}); err != nil {
				t.Fatal(err)
			}
			if err := s.Store.MarkReady(t.Context(), id, "10.89.0.2"); err != nil {
				t.Fatal(err)
			}
			if tc.saved {
				if err := s.Store.AddKey(t.Context(), 2, "saved-key-double", "fingerprint"); err != nil {
					t.Fatal(err)
				}
			}
			calls := 0
			s.Host.HTTP = &http.Client{Transport: roundTrip(func(r *http.Request) (*http.Response, error) {
				calls++
				var in host.Account
				if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
					t.Fatal(err)
				}
				if r.URL.Path != "/account" || in.Login != "bob" || in.Identity != 2 || in.Project != id || len(in.Keys) != tc.wantKeys || in.Keys == nil {
					t.Fatal("wrong real account request", in)
				}
				status := 200
				if tc.fail {
					status = 500
				}
				return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(`{"ok":true}`)), Header: make(http.Header)}, nil
			})}
			w := httptest.NewRecorder()
			s.ServeHTTP(w, apiTestRequest("POST", "/api/environments/"+id+"/join", tc.body, "bob"))
			if w.Code != tc.status {
				t.Fatal(w.Code, w.Body.String())
			}
			login, err := s.Store.MemberLogin(t.Context(), id, 2)
			if tc.status == 200 {
				if err != nil || login != "bob" || calls != 1 {
					t.Fatal(login, err, calls)
				}
				// Existing membership returns the original login without provisioning again,
				// changing keys or making renamed provider identity the native account.
				w = httptest.NewRecorder()
				s.ServeHTTP(w, apiTestRequest("POST", "/api/environments/"+id+"/join", `{"ssh_keys":"saved"}`, "bob"))
				if w.Code != 200 || calls != 1 {
					t.Fatal("join replayed account")
				}
			} else if !errors.Is(err, sql.ErrNoRows) {
				t.Fatal("unconfirmed membership recorded", err)
			}
			if tc.status == 400 && calls != 0 {
				t.Fatal("bad choice reached native")
			}
		})
	}
}
