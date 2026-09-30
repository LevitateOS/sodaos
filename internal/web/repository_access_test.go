package web

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	extensions "forgejo.org/extension-sdk"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
)

func TestRepositoryDenialBlocksDiscoveryDirectReadsAndNewAccounts(t *testing.T) {
	for _, tc := range []struct {
		name     string
		response func(string) extensions.CallbackResponse
	}{
		{"hidden", func(string) extensions.CallbackResponse { return extensions.CallbackResponse{ErrorCode: "not_found"} }},
		{"unavailable", func(string) extensions.CallbackResponse { return extensions.CallbackResponse{ErrorCode: "unavailable"} }},
		{"wrong repository", func(string) extensions.CallbackResponse {
			return extensions.CallbackResponse{Repository: &extensions.Repository{ID: "43", Owner: "alice", Name: "private", Permission: "write"}}
		}},
		{"malformed", func(id string) extensions.CallbackResponse {
			return extensions.CallbackResponse{Repository: &extensions.Repository{ID: id, Owner: "", Name: "private", Permission: "write"}}
		}},
		{"no permission", func(id string) extensions.CallbackResponse {
			return extensions.CallbackResponse{Repository: &extensions.Repository{ID: id, Owner: "alice", Name: "private", Permission: "none"}}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := apiTestServer(t)
			s.Config.OperatorID = 99
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
			var calls atomic.Int32
			s.Host.HTTP = &http.Client{Transport: roundTrip(func(*http.Request) (*http.Response, error) {
				calls.Add(1)
				return nil, errors.New("must not reach helper")
			})}
			callback := func(in extensions.CallbackRequest) extensions.CallbackResponse {
				switch in.Operation {
				case extensions.OperationRepository:
					return tc.response(in.RepositoryID)
				case extensions.OperationOwnedRepositories:
					return extensions.CallbackResponse{ErrorCode: "unavailable"}
				default:
					t.Error("unexpected callback", in.Operation)
					return extensions.CallbackResponse{ErrorCode: "unavailable"}
				}
			}
			for _, path := range []string{"/api/environments?repository_id=42", "/api/environments/" + id, "/api/environments/" + id + "/members", "/api/environments/" + id + "/join", "/api/environments"} {
				method, body := "GET", "{}"
				if strings.HasSuffix(path, "/join") || path == "/api/environments" {
					method = "POST"
				}
				if path == "/api/environments" {
					body = `{"repository_id":"42"}`
				}
				w := httptest.NewRecorder()
				nativeAPIServeWithCallback(t, s, w, apiTestRequest(method, path, body, "bob"), callback)
				if w.Code < 400 || strings.Contains(w.Body.String(), "alice/private") || strings.Contains(w.Body.String(), id) {
					t.Fatal(path, w.Code, w.Body.String())
				}
			}
			if calls.Load() != 0 {
				t.Fatal("denial reached helper")
			}
			if _, err := s.Store.MemberLogin(t.Context(), id, 2); !errors.Is(err, store.ErrNotFound) {
				t.Fatal("denial wrote membership", err)
			}
		})
	}
}

func TestRepositoryLookupAndJoinRecheckNativeAccess(t *testing.T) {
	allowed := true
	var lookups, accounts atomic.Int32
	var last project.Account
	s := apiTestServer(t)
	s.Config.OperatorID = 99
	callback := func(in extensions.CallbackRequest) extensions.CallbackResponse {
		if in.Operation != extensions.OperationRepository || in.RepositoryID != "42" {
			t.Error("unexpected native callback", in.Operation, in.RepositoryID)
			return extensions.CallbackResponse{ErrorCode: "invalid_operation"}
		}
		lookups.Add(1)
		if !allowed {
			return extensions.CallbackResponse{ErrorCode: "not_found"}
		}
		return extensions.CallbackResponse{Repository: &extensions.Repository{ID: "42", Owner: "bob", Name: "renamed", Permission: "write"}}
	}
	for _, query := range []string{"", "?repository_id=0", "?repository_id=01", "?repository_id=42&repository_id=42", "?repository_id=%zz", "?repository_id=42&owner=alice", "?repository_id=" + strings.Repeat("1", 8192)} {
		w := httptest.NewRecorder()
		nativeAPIServeWithCallback(t, s, w, apiTestRequest("GET", "/api/environments"+query, "", "bob"), callback)
		if w.Code != 400 {
			t.Fatal("invalid context accepted", w.Code)
		}
	}
	if lookups.Load() != 0 {
		t.Fatal("invalid input reached repository")
	}
	read := func() *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		nativeAPIServeWithCallback(t, s, w, apiTestRequest("GET", "/api/environments?repository_id=42", "", "bob"), callback)
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
		nativeAPIServeWithCallback(t, s, w, apiTestRequest("POST", "/api/environments/"+id+"/join", "{}", "bob"), callback)
		return w
	}
	allowed = false
	if w := join(); w.Code < 400 || accounts.Load() != 0 {
		t.Fatal("prior read authorized join", w.Code)
	}
	allowed = true
	if w := join(); w.Code != 200 || accounts.Load() != 1 || last.Identity != 2 || last.Login != "bob" || last.Project != id {
		t.Fatal("incorrect new account", w.Code, last)
	}
	before := lookups.Load()
	allowed = false
	if w := join(); w.Code < 400 || accounts.Load() != 1 || lookups.Load() != before+1 {
		t.Fatal("existing join reprovisioned/revoked", w.Code)
	}
	p, err := s.Store.Project(context.Background(), id)
	if err != nil || p.OwnerID != 1 || p.Repository != "alice/old" {
		t.Fatal("remapped retained state")
	}
}
