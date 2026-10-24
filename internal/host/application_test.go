package host

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/application"
	"github.com/levitateos/sodaos/internal/project"
)

func hostApplicationSpec() application.Spec {
	return application.Spec{ID: "a" + strings.Repeat("1", 24), Revision: 1, Name: "Example", ArtifactSHA256: strings.Repeat("2", 64), BundleSHA256: strings.Repeat("3", 64), ToolsSHA256: strings.Repeat("4", 64), RuntimeImage: "sha256:" + strings.Repeat("5", 64), Image: "sha256:" + strings.Repeat("6", 64), Data: "a" + strings.Repeat("7", 24), OwnerTeam: "12345678-1234-1234-1234-123456789abc", Origin: "https://app.example.test", Port: 20001}
}

func TestApplicationNativeClient(t *testing.T) {
	spec := hostApplicationSpec()
	container := strings.Repeat("a", 64)
	for _, test := range []struct {
		name   string
		action string
		change func(map[string]any)
		ok     bool
	}{
		{"running", "start", func(map[string]any) {}, true},
		{"stopped", "stop", func(v map[string]any) { v["running"] = false; v["ready"] = false }, true},
		{"wrong_image", "inspect", func(v map[string]any) { v["image"] = "sha256:" + strings.Repeat("b", 64) }, false},
		{"wrong_container", "start", func(v map[string]any) { v["container"] = strings.Repeat("b", 64) }, false},
		{"missing_running", "stop", func(v map[string]any) { delete(v, "running"); v["ready"] = false }, false},
		{"wrong_revision", "inspect", func(v map[string]any) { v["revision"] = 2 }, false},
		{"stop_unconfirmed", "stop", func(map[string]any) {}, false},
		{"unexpected_field", "inspect", func(v map[string]any) { v["token"] = "unexpected" }, false},
		{"replace_same_container", "replace", func(v map[string]any) { v["running"] = false; v["ready"] = false }, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" || r.URL.Path != "/application" {
					t.Error("wrong native route")
					w.WriteHeader(500)
					return
				}
				var request application.Request
				if err := json.NewDecoder(r.Body).Decode(&request); err != nil || request.Application != spec || request.ExpectedContainer != container || request.Action != test.action {
					t.Errorf("wrong native request %+v %v", request, err)
				}
				result := map[string]any{"id": spec.ID, "revision": 1, "container": container, "image": spec.Image, "running": true, "ready": true, "origin": spec.Origin}
				test.change(result)
				_ = json.NewEncoder(w).Encode(result)
			}))
			defer server.Close()
			client := &Client{HTTP: &http.Client{Transport: applicationTestTransport{server.URL}}}
			_, err := client.Application(context.Background(), application.Request{Application: spec, Action: test.action, ExpectedContainer: container})
			if (err == nil) != test.ok {
				t.Fatalf("success=%v err=%v", test.ok, err)
			}
		})
	}
	t.Run("candidate_worker_deadline_survives_short_rpc", func(t *testing.T) {
		digest, commit := strings.Repeat("a", 64), strings.Repeat("b", 40)
		prep := project.Preparation{
			ID: "f" + strings.Repeat("1", 24), Project: "p" + strings.Repeat("2", 24), Role: project.RoleReviewer,
			Requirements: project.RequirementAcceptance{ID: "d" + strings.Repeat("3", 24), Approver: 1, SourceCommit: commit, Digest: digest},
			Approval:     project.AdminApproval{ID: "d" + strings.Repeat("4", 24), Approver: 1, EffectsDigest: digest},
			SourceCommit: commit, SetupDigest: digest, Tools: []string{"git", "can"},
		}
		admittedDeadline := time.Now().Add(10 * time.Minute).UTC()
		input := project.FactoryCandidate{Preparation: prep, SourcePreparation: "f" + strings.Repeat("5", 24), Bundle: []byte("source"), Deadline: admittedDeadline}
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "POST" || r.URL.Path != "/prepare-candidate" {
				t.Error("wrong candidate route")
				w.WriteHeader(500)
				return
			}
			var wire project.FactoryCandidate
			if err := json.NewDecoder(r.Body).Decode(&wire); err != nil {
				t.Error(err)
				return
			}
			if !wire.Deadline.Equal(admittedDeadline) {
				t.Errorf("async worker deadline became RPC deadline: got %v want %v", wire.Deadline, admittedDeadline)
			}
			_ = json.NewEncoder(w).Encode(project.PrepareState{ID: prep.ID, Project: prep.Project, Role: prep.Role, Container: container, SourceCommit: commit, SetupDigest: digest, Phase: project.PrepareRunning})
		}))
		defer server.Close()
		client := &Client{HTTP: &http.Client{Transport: applicationTestTransport{server.URL}}}
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if state, err := client.PrepareCandidate(ctx, input); err != nil || state.Phase != project.PrepareRunning {
			t.Fatalf("bounded candidate admission: %+v %v", state, err)
		}
		cancel()
		if _, err := client.PrepareCandidate(ctx, input); err != context.Canceled {
			t.Fatalf("expired RPC was not refused: %v", err)
		}
	})
}

type applicationTestTransport struct{ target string }

func (t applicationTestTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	copy := r.Clone(r.Context())
	u := *r.URL
	copy.URL = &u
	copy.URL.Scheme = "http"
	copy.URL.Host = strings.TrimPrefix(t.target, "http://")
	return http.DefaultTransport.RoundTrip(copy)
}
