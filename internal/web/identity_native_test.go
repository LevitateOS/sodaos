package web

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	extensions "forgejo.org/extension-sdk"
	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/identity"
	"github.com/levitateos/sodaos/internal/store"
	"github.com/levitateos/sodaos/internal/web/api"
)

func TestNativeCommentIntakeUsesHumanIssueAccessWithoutFactorySetup(t *testing.T) {
	s := apiTestServer(t)
	reads := 0
	proxy := nativeProductProxyForActorWithCallback(t, s, extensions.Contribution{}, "alice", func(in extensions.CallbackRequest) extensions.CallbackResponse {
		switch in.Operation {
		case extensions.OperationRepository:
			return extensions.CallbackResponse{Repository: &extensions.Repository{ID: "7", Owner: "alice", Name: "private-app", Permission: "read"}}
		case extensions.OperationRepositoryIssues:
			reads++
			if in.Authority.Actor.ID != "1" || in.RepositoryID != "7" {
				t.Error("issue read lost human or repository scope")
			}
			page := &extensions.RepositoryIssuePage{RepositoryID: "7", Items: []extensions.RepositoryIssue{}}
			switch {
			case in.Query == "9":
				page.Items = append(page.Items, extensions.RepositoryIssue{Index: "9", Title: "Earlier login flow", Body: "Previous objective", Closed: true})
			case in.Cursor == "":
				page.Items = append(page.Items, extensions.RepositoryIssue{Index: "41", Title: "Login flow", Body: "An existing issue without Factory work"})
				page.NextCursor = "41"
			case in.Cursor == "41":
				page.Items = append(page.Items, extensions.RepositoryIssue{Index: "42", Title: "Other work", Body: "A separate objective"})
			default:
				t.Error("unexpected issue selector")
			}
			return extensions.CallbackResponse{Issues: page}
		}
		return extensions.CallbackResponse{}
	})
	response := nativeProductRequest(t, proxy, s.Config.ForgejoURL, "POST", "/factory/feed/posts",
		[]byte(`{"repository":7,"issue":9,"body":"Please discuss the login flow.","idempotency_key":"human-context-intake"}`))
	if response.StatusCode != 200 {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("intake status=%d body=%s", response.StatusCode, body)
	}
	var saved struct {
		SourcePost  string `json:"post_id"`
		Revision int64  `json:"revision"`
	}
	if err := json.NewDecoder(response.Body).Decode(&saved); err != nil {
		t.Fatal(err)
	}
	if saved.SourcePost == "" {
		t.Fatal("intake did not return a comment")
	}
	candidates, err := s.Store.PostIssueContext(t.Context(), saved.SourcePost, saved.Revision)
	if err != nil || len(candidates) != 3 || reads != 3 {
		t.Fatal("human issue context was not saved", len(candidates), reads, err)
	}
	if candidates[0].Issue != 41 || candidates[0].Title != "Login flow" || candidates[2].Issue != 9 || candidates[2].Status != "closed" {
		t.Fatal("issue meaning or closed selection was lost", candidates)
	}
	pipe, err := s.Store.PostIntake(t.Context(), saved.SourcePost)
	if err != nil || pipe.Judgment != factory.PostJudgmentPending || pipe.Attempts != 0 {
		t.Fatal("intake spent or executed work", pipe, err)
	}

	// A post blocked by the old service-account path can be refreshed by its
	// author without creating a duplicate or configuring Factory.
	id, revision, err := s.Store.SaveFeedPost(t.Context(), factory.PostRequest{Repository: 7, Body: "A request from before the fix", IdempotencyKey: "old-context-intake"}, "1", "alice", 1)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Store.RecordPostIntake(t.Context(), id, revision); err != nil {
		t.Fatal(err)
	}
	if err := s.Store.NotePostStall(t.Context(), id, "factory"); err != nil {
		t.Fatal(err)
	}
	response = nativeProductRequest(t, proxy, s.Config.ForgejoURL, "POST", "/factory/feed/posts/"+id+"/retry", []byte(`{"expected_revision":0}`))
	if response.StatusCode != 200 {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("refresh status=%d body=%s", response.StatusCode, body)
	}
	refreshed, err := s.Store.FeedPost(t.Context(), id)
	if err != nil || refreshed.Record.Revision != revision+1 {
		t.Fatal("refresh lost the original post", refreshed.Record, err)
	}
	if _, err := s.Store.PostIssueContext(t.Context(), id, refreshed.Record.Revision); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Store.PostIssueContext(t.Context(), id, revision); err == nil {
		t.Fatal("old revision retained current issue authority")
	}
}

func TestNativeJEVProviderRoutesThroughProxy(t *testing.T) {
	s := apiTestServer(t)
	keyPath := filepath.Join(t.TempDir(), "typesafe.key")
	s.API.Config.TypeSafeAPIKeyFile = keyPath
	panel := extensions.Contribution{Kind: "panel", ID: "providers", Scope: "panel"}
	for _, login := range []string{"alice", "bob"} {
		t.Run(login, func(t *testing.T) {
			proxy := nativeProductProxyForActor(t, s, panel, login)
			request := func(method, path, body string, status int) map[string]any {
				t.Helper()
				r, err := http.NewRequestWithContext(t.Context(), method, proxy.URL+path, strings.NewReader(body))
				if err != nil {
					t.Fatal(err)
				}
				r.Header = nativeProductHeadersForActor(method, s.Config.ForgejoURL, panel, login)
				r.Header.Set("Content-Type", "application/json")
				response, err := proxy.Client().Do(r)
				if err != nil {
					t.Fatal(err)
				}
				defer response.Body.Close()
				if response.StatusCode != status || !strings.HasPrefix(response.Header.Get("Content-Type"), "application/json") {
					t.Fatalf("%s %s: status=%d content-type=%q, want JSON status %d", method, path, response.StatusCode, response.Header.Get("Content-Type"), status)
				}
				var value map[string]any
				if err := json.NewDecoder(response.Body).Decode(&value); err != nil {
					t.Fatal(err)
				}
				return value
			}
			if err := os.Remove(keyPath); err != nil && !os.IsNotExist(err) {
				t.Fatal(err)
			}
			canManage := login == "alice"
			status := request("GET", "/identity/jev", "", 200)
			if status["connected"] != false || status["can_manage"] != canManage {
				t.Fatal("incorrect disconnected status", status)
			}
			if err := os.WriteFile(keyPath, []byte("test-key-not-a-real-credential"), 0o600); err != nil {
				t.Fatal(err)
			}
			status = request("GET", "/identity/jev", "", 200)
			if status["connected"] != true || status["can_manage"] != canManage {
				t.Fatal("incorrect connected status", status)
			}
			if canManage {
				// An empty key reaches validation without making a provider call.
				request("POST", "/identity/jev", `{"key":""}`, 400)
				status = request("POST", "/identity/jev/disconnect", "{}", 200)
				if status["connected"] != false || status["can_manage"] != true {
					t.Fatal("incorrect disconnect status", status)
				}
				if _, err := os.Stat(keyPath); !os.IsNotExist(err) {
					t.Fatal("disconnect did not remove the stored key", err)
				}
				request("POST", "/identity/jev/disconnect", "{}", 200)
			} else {
				request("POST", "/identity/jev", `{"key":""}`, 403)
				request("POST", "/identity/jev/disconnect", "{}", 403)
				if _, err := os.Stat(keyPath); err != nil {
					t.Fatal("non-operator changed the stored key", err)
				}
			}
		})
	}
}

type nativeIdentityFake struct {
	api.IdentityClient
	owner      int64
	calls      int
	grant      identity.GrantRequest
	endedLease string
}

func (f *nativeIdentityFake) Connections(_ context.Context, owner int64) ([]identity.Connection, error) {
	f.owner, f.calls = owner, f.calls+1
	return []identity.Connection{{ID: "connection", OwnerID: owner, Label: "personal", State: identity.Ready}}, nil
}

func (f *nativeIdentityFake) StartEnrollment(_ context.Context, owner int64, providerID, _ string) (identity.Enrollment, error) {
	f.owner, f.calls = owner, f.calls+1
	return identity.Enrollment{ID: "enrollment", ProviderID: providerID, State: "pending"}, nil
}

func (f *nativeIdentityFake) CreateGrant(_ context.Context, owner int64, in identity.GrantRequest) (identity.Grant, error) {
	f.owner, f.calls, f.grant = owner, f.calls+1, in
	return identity.Grant{ID: "grant", ConnectionID: in.ConnectionID, UserID: in.UserID, ProjectID: in.ProjectID}, nil
}

func (f *nativeIdentityFake) EndLease(_ context.Context, owner int64, id string) error {
	f.owner, f.calls, f.endedLease = owner, f.calls+1, id
	return nil
}

func TestNativeIdentityActorAndEnrollmentTrust(t *testing.T) {
	s := apiTestServer(t)
	fake := &nativeIdentityFake{}
	s.API.Identity = fake
	for _, login := range []string{"alice", "bob"} {
		w := httptest.NewRecorder()
		nativeAPIServe(t, s, w, apiTestRequest("GET", "/api/identity/connections", "", login))
		wantOwner := int64(1)
		if login == "bob" {
			wantOwner = 2
		}
		if w.Code != 200 || fake.owner != wantOwner || !strings.Contains(w.Body.String(), `"owner_id":"`+nativeTestActorID(login)+`"`) || w.Header().Get("Cache-Control") != "no-store" {
			t.Fatal("native actor was not bound to identity metadata", login, w.Code, w.Body.String(), fake.owner)
		}
	}
	before := fake.calls
	for _, body := range []string{`{"provider_id":"codex","label":"personal"}`, `{"provider_id":"codex","label":"personal","confirm_credential_exposure":false}`, `{"provider_id":"codex","label":"personal","confirm_credential_exposure":true,"owner_id":"2"}`} {
		w := httptest.NewRecorder()
		nativeAPIServe(t, s, w, apiTestRequest("POST", "/api/identity/enrollments", body, "alice"))
		if w.Code != 400 || fake.calls != before {
			t.Fatal("unconfirmed credential custody reached identity service", w.Code, w.Body.String())
		}
	}
	w := httptest.NewRecorder()
	nativeAPIServe(t, s, w, apiTestRequest("POST", "/api/identity/enrollments", `{"provider_id":"codex","label":"personal","confirm_credential_exposure":true}`, "alice"))
	if w.Code != 200 || fake.owner != 1 || fake.calls != before+1 {
		t.Fatal("confirmed enrollment lost native actor", w.Code, fake.owner, fake.calls)
	}
	for _, changed := range []string{"actor", "generation", "origin"} {
		w = httptest.NewRecorder()
		nativeAPIServeWithOptions(t, s, w, apiTestRequest("POST", "/api/identity/enrollments", `{"provider_id":"codex","label":"personal","confirm_credential_exposure":true}`, "alice"), nil, func(h http.Header) {
			switch changed {
			case "actor":
				h.Set(extensions.ContextHeader, strings.Replace(h.Get(extensions.ContextHeader), `"id":"1"`, `"id":"2"`, 1))
			case "generation":
				h.Set(extensions.SessionGenerationHeader, "expired")
			case "origin":
				h.Set("Origin", "https://other.invalid")
			}
		})
		if w.Code < 400 || fake.calls != before+1 {
			t.Fatal("stale native identity mutation reached service", changed, w.Code, fake.calls)
		}
	}
}

func TestNativeIdentityGrantRequiresConfirmationsMembersAndWrite(t *testing.T) {
	for _, tc := range []struct {
		name                                                  string
		ready, actorMember, recipientMember, write, confirmed bool
		status                                                int
	}{
		{"confirmed named member", true, true, true, true, true, 200},
		{"no confirmation", true, true, true, true, false, 400},
		{"unprovisioned", false, true, true, true, true, 409},
		{"requester not member", true, false, true, true, true, 403},
		{"recipient not member", true, true, false, true, true, 403},
		{"write removed", true, true, true, false, true, 403},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := apiTestServer(t)
			fake := &nativeIdentityFake{}
			s.API.Identity = fake
			projectID := "p0123456789abcdef01234567"
			if err := s.Store.CreateProject(t.Context(), store.Project{ID: projectID, RepositoryID: 7, OwnerID: 1}); err != nil {
				t.Fatal(err)
			}
			if tc.ready {
				if err := s.Store.MarkReady(t.Context(), projectID, "10.0.0.2"); err != nil {
					t.Fatal(err)
				}
			}
			if tc.actorMember {
				if err := s.Store.Join(t.Context(), projectID, 1, "alice"); err != nil {
					t.Fatal(err)
				}
			}
			if tc.recipientMember {
				if err := s.Store.Join(t.Context(), projectID, 2, "bob"); err != nil {
					t.Fatal(err)
				}
			}
			permission := "read"
			if tc.write {
				permission = "write"
			}
			callback := func(in extensions.CallbackRequest) extensions.CallbackResponse {
				if in.Operation != extensions.OperationRepository || in.RepositoryID != "7" {
					t.Error("unexpected native callback", in.Operation, in.RepositoryID)
					return extensions.CallbackResponse{ErrorCode: "invalid_operation"}
				}
				return extensions.CallbackResponse{Repository: &extensions.Repository{ID: "7", Owner: "alice", Name: "repo", Permission: permission}}
			}
			body := `{"connection_id":"connection","user_id":"2","confirm_subscription":true,"confirm_credential_exposure":true}`
			if !tc.confirmed {
				body = strings.Replace(body, `"confirm_subscription":true`, `"confirm_subscription":false`, 1)
			}
			w := httptest.NewRecorder()
			nativeAPIServeWithCallback(t, s, w, apiTestRequest("POST", "/api/environments/"+projectID+"/identity/grants", body, "alice"), callback)
			if w.Code != tc.status || (fake.calls == 1) != (tc.status == 200) {
				t.Fatal(w.Code, w.Body.String(), fake.calls)
			}
			if tc.status == 200 && (fake.owner != 1 || fake.grant.UserID != 2 || fake.grant.ProjectID != projectID) {
				t.Fatal("caller authority replaced by recipient input", fake.grant)
			}
		})
	}
}

func TestNativeIdentityLaunchCompensatesGenerationChange(t *testing.T) {
	for _, change := range []string{"current", "generation"} {
		t.Run(change, func(t *testing.T) {
			s := apiTestServer(t)
			fake := &nativeIdentityFake{}
			s.API.Identity = fake
			projectID := "p0123456789abcdef01234567"
			if err := s.Store.CreateProject(t.Context(), store.Project{ID: projectID, RepositoryID: 7, OwnerID: 1}); err != nil {
				t.Fatal(err)
			}
			if err := s.Store.MarkReady(t.Context(), projectID, "10.0.0.2"); err != nil {
				t.Fatal(err)
			}
			if err := s.Store.Join(t.Context(), projectID, 1, "alice"); err != nil {
				t.Fatal(err)
			}
			r := apiTestRequest("POST", "/api/environments/"+projectID+"/identity/launch", `{"connection_id":"connection","cols":80,"rows":24}`, "alice")
			calls := 0
			s.Host.HTTP = &http.Client{Transport: roundTrip(func(req *http.Request) (*http.Response, error) {
				calls++
				if req.URL.Path != "/identity/launch" {
					t.Error("unexpected native request", req.URL.Path)
				}
				var in identity.TerminalStart
				if err := json.NewDecoder(req.Body).Decode(&in); err != nil || in.ActorID != 1 || in.ProjectID != projectID || in.Login != "alice" || len(in.Scope) != 64 {
					t.Fatal("wrong trusted launch actor", in, err)
				}
				if change == "generation" {
					r.Header.Set(extensions.ContextHeader, strings.Replace(r.Header.Get(extensions.ContextHeader), nativeProductGeneration, "expired", 1))
				}
				return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"id":"lease","execution_id":"` + strings.Repeat("a", 32) + `"}`))}, nil
			})}
			w := httptest.NewRecorder()
			nativeAPIServe(t, s, w, r)
			if calls != 1 {
				t.Fatal("native launch count", calls)
			}
			if change == "generation" {
				if w.Code != 401 || fake.calls != 1 || fake.owner != 1 || fake.endedLease != "lease" {
					t.Fatal("stale launch did not compensate lease", w.Code, fake.calls, fake.endedLease, w.Body.String())
				}
			} else if w.Code != 200 || !strings.Contains(w.Body.String(), `"terminal_id"`) || fake.calls != 0 {
				t.Fatal("current launch failed", w.Code, w.Body.String())
			}
		})
	}
}
