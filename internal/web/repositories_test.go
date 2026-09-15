package web

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/store"
)

func TestRepositoryPickerAuthorityAndReservations(t *testing.T) {
	for _, kind := range []string{"new", "ready", "incomplete", "transfer", "hidden", "foreign-search", "wrong-actor", "unavailable"} {
		t.Run(kind, func(t *testing.T) {
			s := grantedTestServer(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("Authorization") != "token acting-alice" {
					t.Error("borrowed authority")
				}
				switch r.URL.Path {
				case "/api/v1/user":
					if kind == "wrong-actor" {
						fmt.Fprint(w, `{"id":2,"login":"bob"}`)
					} else {
						fmt.Fprint(w, `{"id":1,"login":"alice"}`)
					}
				case "/api/v1/repos/search":
					if kind == "unavailable" {
						w.WriteHeader(503)
						return
					}
					owner := 1
					if kind == "foreign-search" {
						owner = 99
					}
					fmt.Fprintf(w, `{"ok":true,"data":[{"id":7,"name":"repo","full_name":"alice/repo","owner":{"id":%d,"login":"alice"}}]}`, owner)
				case "/api/v1/repositories/7":
					if kind == "hidden" {
						w.WriteHeader(404)
						return
					}
					owner := 1
					if kind == "transfer" {
						owner = 2
					}
					fmt.Fprintf(w, `{"id":7,"name":"renamed","full_name":"alice/renamed","owner":{"id":%d,"login":"alice"}}`, owner)
				default:
					t.Error("unexpected provider request", r.URL)
					w.WriteHeader(500)
				}
			})
			if kind == "ready" || kind == "incomplete" {
				if err := s.Store.CreateProject(t.Context(), store.Project{ID: "p0123456789abcdef01234567", RepositoryID: 7, OwnerID: 1, Name: "repo", Repository: "alice/repo"}); err != nil {
					t.Fatal(err)
				}
				if kind == "ready" {
					if err := s.Store.MarkReady(t.Context(), "p0123456789abcdef01234567", "10.89.0.2"); err != nil {
						t.Fatal(err)
					}
				}
			}
			w := httptest.NewRecorder()
			s.ServeHTTP(w, apiTestRequest("GET", "/api/repositories?q=&page=1", "", "alice"))
			body := w.Body.String()
			switch kind {
			case "foreign-search", "wrong-actor", "unavailable":
				if w.Code == 200 || strings.Contains(body, "renamed") {
					t.Fatal(w.Code, body)
				}
			case "transfer", "hidden":
				if w.Code != 200 || !strings.Contains(body, `"items":[]`) {
					t.Fatal(w.Code, body)
				}
			default:
				if w.Code != 200 || !strings.Contains(body, `"name":"renamed"`) {
					t.Fatal(w.Code, body)
				}
				if kind == "new" && !strings.Contains(body, `"can_create":true,"project":null`) {
					t.Fatal(body)
				}
				if kind != "new" && !strings.Contains(body, `"can_create":false,"project":{"id":"p0123456789abcdef01234567","provisioned":`+fmt.Sprint(kind == "ready")) {
					t.Fatal(body)
				}
			}
		})
	}
}
func TestRepositoryPickerLogoutWinsPublication(t *testing.T) {
	var s *Server
	s = grantedTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/user" {
			fmt.Fprint(w, `{"id":1,"login":"alice"}`)
			return
		}
		if r.URL.Path != "/api/v1/repos/search" {
			t.Error("unexpected request", r.URL)
		}
		if err := s.Store.DeleteSession(t.Context(), "session-alice"); err != nil {
			t.Error(err)
		}
		fmt.Fprint(w, `{"ok":true,"data":[]}`)
	})
	w := httptest.NewRecorder()
	s.ServeHTTP(w, apiTestRequest("GET", "/api/repositories?q=&page=1", "", "alice"))
	if w.Code != 401 || strings.Contains(w.Body.String(), `"items"`) {
		t.Fatal(w.Code, w.Body.String())
	}
}

func TestRepositoryPickerReadConsentAndPageBoundary(t *testing.T) {
	s := grantedTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/user":
			fmt.Fprint(w, `{"id":1,"login":"alice"}`)
		case "/api/v1/repos/search":
			fmt.Fprint(w, `{"ok":true,"data":[`)
			for n := 1; n <= 12; n++ {
				if n > 1 {
					fmt.Fprint(w, ",")
				}
				fmt.Fprintf(w, `{"id":%d,"name":"repo","full_name":"alice/repo","owner":{"id":1,"login":"alice"}}`, n)
			}
			fmt.Fprint(w, `]}`)
		default:
			var id int
			if _, err := fmt.Sscanf(r.URL.Path, "/api/v1/repositories/%d", &id); err != nil {
				t.Error(err)
			}
			fmt.Fprintf(w, `{"id":%d,"name":"repo","full_name":"alice/repo","owner":{"id":1,"login":"alice"}}`, id)
		}
	})
	grant, err := s.Store.Grant(t.Context(), "session-alice", 1)
	if err != nil {
		t.Fatal(err)
	}
	for _, scopes := range []string{"read:user read:repository", "read:user"} {
		if err = s.Store.DeleteSession(t.Context(), "session-alice"); err != nil {
			t.Fatal(err)
		}
		grant.Scopes = scopes
		if err = s.Store.CreateGrantedSession(t.Context(), "session-alice", 1, "csrf-alice", grant); err != nil {
			t.Fatal(err)
		}
		w := httptest.NewRecorder()
		s.ServeHTTP(w, apiTestRequest("GET", "/api/repositories?q=&page=100", "", "alice"))
		if scopes == "read:user" {
			if w.Code == 200 {
				t.Fatal("missing consent accepted")
			}
			continue
		}
		if w.Code != 200 || !strings.Contains(w.Body.String(), `"more":false,"limited":true`) {
			t.Fatal(w.Code, w.Body.String())
		}
	}
}

func TestRepositoryPickerQueryAndAdmission(t *testing.T) {
	s := grantedTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("invalid request reached Forgejo")
		w.WriteHeader(500)
	})
	for _, query := range []string{"", "?q=", "?q=&page=0", "?q=&page=01", "?q=&page=101", "?q=&page=1&page=1", "?q=&page=1&uid=2", "?q=%0a&page=1", "?q=" + strings.Repeat("x", 201) + "&page=1"} {
		w := httptest.NewRecorder()
		s.ServeHTTP(w, apiTestRequest("GET", "/api/repositories"+query, "", "alice"))
		if w.Code != 400 {
			t.Fatal(query, w.Code, w.Body.String())
		}
	}
	for range cap(s.API.RepositorySlots) {
		s.API.RepositorySlots <- struct{}{}
	}
	w := httptest.NewRecorder()
	s.ServeHTTP(w, apiTestRequest("GET", "/api/repositories?q=&page=1", "", "alice"))
	if w.Code != 503 {
		t.Fatal(w.Code)
	}
	for range cap(s.API.RepositorySlots) {
		<-s.API.RepositorySlots
	}
	r := apiTestRequest("GET", "/api/repositories?q=&page=1", "", "alice")
	r.Header.Set("X-Soda-Expected-User-ID", "2")
	w = httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != 403 {
		t.Fatal(w.Code, w.Body.String())
	}
}
