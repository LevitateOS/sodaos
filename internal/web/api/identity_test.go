package api

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/config"
	"github.com/levitateos/sodaos/internal/forgejo"
	"github.com/levitateos/sodaos/internal/identity"
	"github.com/levitateos/sodaos/internal/store"
	"github.com/levitateos/sodaos/internal/web/auth"
)

type identityFake struct {
	IdentityClient
	owner int64
	calls int
	grant identity.GrantRequest
	after func()
}

func (f *identityFake) Connections(_ context.Context, owner int64) ([]identity.Connection, error) {
	f.owner, f.calls = owner, f.calls+1
	if f.after != nil {
		f.after()
	}
	return []identity.Connection{{ID: "connection", OwnerID: owner, Label: "personal", State: identity.Ready}}, nil
}

func (f *identityFake) CreateGrant(_ context.Context, owner int64, in identity.GrantRequest) (identity.Grant, error) {
	f.owner, f.calls, f.grant = owner, f.calls+1, in
	return identity.Grant{ID: "grant", ConnectionID: in.ConnectionID, UserID: in.UserID, ProjectID: in.ProjectID}, nil
}

func (f *identityFake) Revoke(_ context.Context, owner int64, _ string) error {
	f.owner, f.calls = owner, f.calls+1
	return nil
}

func (f *identityFake) StartEnrollment(_ context.Context, owner int64, _, _ string) (identity.Enrollment, error) {
	f.owner, f.calls = owner, f.calls+1
	return identity.Enrollment{ID: "enrollment", State: "pending"}, nil
}

func (f *identityFake) Available(_ context.Context, owner int64, _ string) ([]identity.Connection, error) {
	f.owner, f.calls = owner, f.calls+1
	return []identity.Connection{}, nil
}

func identityFixture(t *testing.T, push bool) (*API, *http.ServeMux, *identityFake) {
	t.Helper()
	db, err := store.OpenEncrypted(filepath.Join(t.TempDir(), "identity-web.db"), bytes.Repeat([]byte{1}, 32))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Error(err)
		}
	})
	for _, user := range []store.User{{ID: 1, Login: "soda-owner"}, {ID: 2, Login: "soda-member"}} {
		if err := db.UpsertUser(t.Context(), user); err != nil {
			t.Fatal(err)
		}
	}
	if err := db.CreateGrantedSession(t.Context(), "session", 1, "csrf", store.Grant{Access: "acting", Refresh: "refresh", Scopes: "read:user read:repository", Expires: time.Now().Add(time.Hour).Unix()}); err != nil {
		t.Fatal(err)
	}
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/user":
			_, _ = fmt.Fprint(w, `{"id":1,"login":"soda-owner"}`)
		case "/api/v1/repositories/7":
			_, _ = fmt.Fprintf(w, `{"id":7,"name":"repo","full_name":"soda-owner/repo","owner":{"id":1,"login":"soda-owner"},"permissions":{"push":%t}}`, push)
		default:
			t.Errorf("unexpected provider route %s", r.URL.Path)
			w.WriteHeader(500)
		}
	}))
	t.Cleanup(provider.Close)
	cfg := &config.Config{ForgejoURL: "https://forgejo.example.test"}
	client := forgejo.New(provider.URL)
	s := New(cfg, db, client, nil, auth.New(cfg, db, client))
	fake := &identityFake{}
	s.Identity = fake
	mux := http.NewServeMux()
	s.mux = mux
	s.identityRoutes()
	return s, mux, fake
}

func identityRequest(method, path, body string) *http.Request {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	r.AddCookie(&http.Cookie{Name: auth.SessionCookie, Value: "session"})
	r.Header.Set(auth.ExpectedUserHeader, "1")
	r.Header.Set("Origin", "https://forgejo.example.test")
	r.Header.Set("X-CSRF-Token", "csrf")
	r.Header.Set("Content-Type", "application/json")
	return r
}

func TestIdentityBrowserGuardsAndOwnerBinding(t *testing.T) {
	_, mux, fake := identityFixture(t, true)
	for _, tc := range []struct {
		name, header, value string
		status              int
	}{
		{"different actor", auth.ExpectedUserHeader, "2", 403},
		{"missing actor", auth.ExpectedUserHeader, "", 400},
		{"wrong csrf", "X-CSRF-Token", "wrong", 403},
		{"wrong origin", "Origin", "https://other.example.test", 403},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := identityRequest("POST", "/api/identity/connections/connection/revoke", `{}`)
			if tc.value == "" {
				r.Header.Del(tc.header)
			} else {
				r.Header.Set(tc.header, tc.value)
			}
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, r)
			if w.Code != tc.status || fake.calls != 0 {
				t.Fatalf("status %d calls %d", w.Code, fake.calls)
			}
		})
	}
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, identityRequest("GET", "/api/identity/connections", ""))
	if w.Code != 200 || fake.owner != 1 || !strings.Contains(w.Body.String(), `"owner_id":"1"`) {
		t.Fatalf("%d %s owner %d", w.Code, w.Body.String(), fake.owner)
	}
	if w.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("identity metadata was cacheable")
	}
}

func TestIdentityResponseRejectsRetiredSession(t *testing.T) {
	s, mux, fake := identityFixture(t, true)
	fake.after = func() {
		if err := s.Store.DeleteSession(t.Context(), "session"); err != nil {
			t.Fatal(err)
		}
	}
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, identityRequest("GET", "/api/identity/connections", ""))
	if w.Code != 401 || strings.Contains(w.Body.String(), `"label"`) {
		t.Fatalf("%d %s", w.Code, w.Body.String())
	}
}

func TestIdentityEnrollmentRequiresExplicitCredentialTrust(t *testing.T) {
	_, mux, fake := identityFixture(t, true)
	for _, body := range []string{`{"provider_id":"codex","label":"personal"}`, `{"provider_id":"codex","label":"personal","confirm_credential_exposure":false}`, `{"provider_id":"codex","label":"personal","confirm_credential_exposure":true,"owner_id":"2"}`} {
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, identityRequest("POST", "/api/identity/enrollments", body))
		if w.Code != 400 || fake.calls != 0 {
			t.Fatalf("%d %s calls %d", w.Code, w.Body.String(), fake.calls)
		}
	}
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, identityRequest("POST", "/api/identity/enrollments", `{"provider_id":"codex","label":"personal","confirm_credential_exposure":true}`))
	if w.Code != 200 || fake.owner != 1 || fake.calls != 1 {
		t.Fatalf("%d %s owner %d", w.Code, w.Body.String(), fake.owner)
	}
}

func TestIdentityGrantRequiresConfirmationsProvisionedMembersAndCurrentWrite(t *testing.T) {
	for _, tc := range []struct {
		name                                                 string
		ready, actorMember, recipientMember, push, confirmed bool
		status                                               int
	}{
		{"confirmed named member", true, true, true, true, true, 200},
		{"no confirmation", true, true, true, true, false, 400},
		{"unprovisioned", false, true, true, true, true, 409},
		{"requester not member", true, false, true, true, true, 403},
		{"recipient not member", true, true, false, true, true, 403},
		{"write removed", true, true, true, false, true, 403},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, mux, fake := identityFixture(t, tc.push)
			project := "p0123456789abcdef01234567"
			if err := s.Store.CreateProject(t.Context(), store.Project{ID: project, RepositoryID: 7, OwnerID: 1, Ready: tc.ready}); err != nil {
				t.Fatal(err)
			}
			if tc.ready {
				if err := s.Store.MarkReady(t.Context(), project, "10.0.0.2"); err != nil {
					t.Fatal(err)
				}
			}
			if tc.actorMember {
				if err := s.Store.Join(t.Context(), project, 1, "soda-owner"); err != nil {
					t.Fatal(err)
				}
			}
			if tc.recipientMember {
				if err := s.Store.Join(t.Context(), project, 2, "soda-member"); err != nil {
					t.Fatal(err)
				}
			}
			body := fmt.Sprintf(`{"connection_id":"connection","user_id":"2","confirm_subscription":%t,"confirm_credential_exposure":true}`, tc.confirmed)
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, identityRequest("POST", "/api/environments/"+project+"/identity/grants", body))
			if w.Code != tc.status {
				t.Fatalf("%d %s", w.Code, w.Body.String())
			}
			if tc.status != 200 && fake.calls != 0 {
				t.Fatal("unauthorized request reached identity service")
			}
			if tc.status == 200 && (fake.owner != 1 || fake.grant.UserID != 2 || fake.grant.ProjectID != project) {
				t.Fatal("caller authority replaced by recipient or project input")
			}
			if tc.confirmed {
				fake.calls = 0
				w = httptest.NewRecorder()
				mux.ServeHTTP(w, identityRequest("GET", "/api/environments/"+project+"/identity/connections", ""))
				want := tc.status
				if tc.name == "recipient not member" {
					want = 200
				}
				if w.Code != want {
					t.Fatalf("available status %d %s", w.Code, w.Body.String())
				}
				if want != 200 && fake.calls != 0 {
					t.Fatal("available connections exposed without current project authority")
				}
			}
		})
	}
}
