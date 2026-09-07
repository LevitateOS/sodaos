package web

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/levitateos/sodaos/internal/host"
	"github.com/levitateos/sodaos/internal/store"
)

func TestRepositoryDenialBlocksDiscoveryDirectReadsAndNewAccounts(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
	}{
		{"hidden", 404}, {"forbidden", 403}, {"unavailable", 503}, {"wrong repository", 503},
		{"malformed", 503}, {"oversized", 413}, {"wrong subject", 401}, {"no grant", 401},
		{"no consent", 403}, {"timeout", 503}, {"site admin", 404}, {"Soda operator", 404},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls atomic.Int32
			s := grantedTestServer(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("Authorization") != "token acting-bob" {
					t.Error("wrong authority")
				}
				if r.URL.Path == "/api/v1/user" {
					uid := 2
					if tc.name == "wrong subject" {
						uid = 1
					}
					fmt.Fprintf(w, `{"id":%d,"login":"bob","is_admin":true}`, uid)
					return
				}
				if r.URL.Path != "/api/v1/repositories/42" {
					t.Error("wrong repository lookup", r.URL.Path)
				}
				switch tc.name {
				case "wrong repository":
					fmt.Fprint(w, `{"id":43,"name":"private","full_name":"alice/private","owner":{"id":1,"login":"alice"}}`)
				case "malformed":
					fmt.Fprint(w, `{"id":42}`)
				case "oversized":
					fmt.Fprint(w, strings.Repeat("x", (2<<20)+1))
				case "unavailable":
					w.WriteHeader(503)
				case "forbidden":
					w.WriteHeader(403)
				default:
					w.WriteHeader(404)
				}
			})
			s.Config.OperatorID = 99
			if tc.name == "Soda operator" {
				s.Config.OperatorID = 2
			}
			id := "p0123456789abcdef01234567"
			if err := s.Store.CreateProject(t.Context(), store.Project{ID: id, RepositoryID: 42, OwnerID: 1, Name: "private", Repository: "alice/private"}); err != nil {
				t.Fatal(err)
			}
			if err := s.Store.MarkReady(t.Context(), id, "10.89.0.2"); err != nil {
				t.Fatal(err)
			}
			if err := s.Store.AddKey(t.Context(), 2, "public-key-double", "fingerprint"); err != nil {
				t.Fatal(err)
			}
			if tc.name == "no grant" {
				if err := s.Store.DeleteGrant(t.Context(), "session-bob"); err != nil {
					t.Fatal(err)
				}
			}
			if tc.name == "no consent" {
				g, err := s.Store.Grant(t.Context(), "session-bob", 2)
				if err != nil {
					t.Fatal(err)
				}
				g.Scopes = "read:user"
				if err = s.Store.ReplaceGrant(t.Context(), "session-bob", 2, g); err != nil {
					t.Fatal(err)
				}
			}
			if tc.name == "timeout" {
				s.Forgejo.HTTP.Transport = roundTrip(func(*http.Request) (*http.Response, error) { return nil, context.DeadlineExceeded })
			}
			s.Host.HTTP = &http.Client{Transport: roundTrip(func(*http.Request) (*http.Response, error) {
				calls.Add(1)
				return nil, errors.New("must not reach helper")
			})}
			for _, path := range []string{"/api/environments?repository_id=42", "/api/environments/" + id, "/api/environments/" + id + "/members", "/api/environments/" + id + "/join"} {
				method := "GET"
				if strings.HasSuffix(path, "/join") {
					method = "POST"
				}
				if tc.name == "Soda operator" && method != "POST" {
					continue
				} // operator inspection is selected, not a join bypass
				w := httptest.NewRecorder()
				s.ServeHTTP(w, apiTestRequest(method, path, "{}", "bob"))
				if w.Code != tc.status || strings.Contains(w.Body.String(), "alice/private") || strings.Contains(w.Body.String(), id) {
					t.Fatal(path, w.Code, w.Body.String())
				}
			}
			if calls.Load() != 0 {
				t.Fatal("denial reached helper")
			}
			if _, err := s.Store.MemberLogin(t.Context(), id, 2); !errors.Is(err, sql.ErrNoRows) {
				t.Fatal("denial wrote membership", err)
			}
		})
	}
}

func TestRepositoryLookupAndJoinRecheckNativeAccess(t *testing.T) {
	allowed := true
	var lookups, accounts atomic.Int32
	var last host.Account
	s := grantedTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/user":
			fmt.Fprint(w, `{"id":2,"login":"bob-now"}`)
		case "/api/v1/repositories/42":
			lookups.Add(1)
			if !allowed {
				w.WriteHeader(404)
				return
			}
			fmt.Fprint(w, `{"id":42,"name":"renamed","full_name":"current/renamed","owner":{"id":2,"login":"current"}}`)
		default:
			t.Error("unexpected provider call", r.URL.Path)
			w.WriteHeader(500)
		}
	})
	s.Config.OperatorID = 99
	for _, query := range []string{"", "?repository_id=0", "?repository_id=01", "?repository_id=42&repository_id=42", "?repository_id=%zz", "?repository_id=42&owner=alice", "?repository_id=" + strings.Repeat("1", 8192)} {
		w := httptest.NewRecorder()
		s.ServeHTTP(w, apiTestRequest("GET", "/api/environments"+query, "", "bob"))
		if w.Code != 400 {
			t.Fatal("invalid context accepted", w.Code)
		}
	}
	if lookups.Load() != 0 {
		t.Fatal("invalid input reached repository")
	}
	read := func() *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		s.ServeHTTP(w, apiTestRequest("GET", "/api/environments?repository_id=42", "", "bob"))
		return w
	}
	absent := read()
	if absent.Code != 200 || !strings.Contains(absent.Body.String(), `"items":[]`) || !strings.Contains(absent.Body.String(), `"can_create":true`) {
		t.Fatal(absent.Code, absent.Body.String())
	}
	id := "p0123456789abcdef01234567"
	if err := s.Store.CreateProject(t.Context(), store.Project{ID: id, RepositoryID: 42, OwnerID: 1, Name: "old", Repository: "alice/old"}); err != nil {
		t.Fatal(err)
	}
	if err := s.Store.CreateProject(t.Context(), store.Project{ID: "punrelated", RepositoryID: 99, OwnerID: 1, Name: "secret", Repository: "alice/secret"}); err != nil {
		t.Fatal(err)
	}
	existing := read()
	if existing.Code != 200 || !strings.Contains(existing.Body.String(), `"name":"renamed"`) || strings.Contains(existing.Body.String(), "secret") || !strings.Contains(existing.Body.String(), `"can_create":false`) {
		t.Fatal(existing.Code, existing.Body.String())
	}
	if err := s.Store.MarkReady(t.Context(), id, "10.89.0.2"); err != nil {
		t.Fatal(err)
	}
	if err := s.Store.AddKey(t.Context(), 2, "public-key-double", "fingerprint"); err != nil {
		t.Fatal(err)
	}
	s.Host.HTTP = &http.Client{Transport: roundTrip(func(r *http.Request) (*http.Response, error) {
		accounts.Add(1)
		if r.URL.Path != "/account" {
			t.Error("unexpected native request")
		}
		if err := json.NewDecoder(r.Body).Decode(&last); err != nil {
			t.Fatal(err)
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"ok":true}`)), Header: make(http.Header)}, nil
	})}
	join := func() *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		s.ServeHTTP(w, apiTestRequest("POST", "/api/environments/"+id+"/join", "{}", "bob"))
		return w
	}
	allowed = false
	if w := join(); w.Code != 404 || accounts.Load() != 0 {
		t.Fatal("prior read authorized join", w.Code)
	}
	allowed = true
	if w := join(); w.Code != 200 || accounts.Load() != 1 || last.Identity != 2 || last.Login != "bob-now" || last.Project != id {
		t.Fatal("incorrect new account", w.Code, last)
	}
	before := lookups.Load()
	allowed = false
	if w := join(); w.Code != 200 || accounts.Load() != 1 || lookups.Load() != before || !strings.Contains(w.Body.String(), "bob-now") {
		t.Fatal("existing join reprovisioned/revoked", w.Code)
	}
	p, err := s.Store.Project(t.Context(), id)
	if err != nil || p.OwnerID != 1 || p.Repository != "alice/old" {
		t.Fatal("remapped retained state")
	}
}
