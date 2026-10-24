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
	t.Run("admitted_factory_verification_receipt", func(t *testing.T) {
		request := project.FactoryVerificationChecks{Project: "p" + strings.Repeat("1", 24), Run: strings.Repeat("2", 32), Incarnation: strings.Repeat("3", 32), ManifestDigest: strings.Repeat("4", 64)}
		zero := 0
		good := project.FactoryVerificationChecksState{Project: request.Project, Run: request.Run, Incarnation: request.Incarnation, ManifestDigest: request.ManifestDigest, Phase: "passed", SourceCommit: strings.Repeat("5", 40), Preparation: "f" + strings.Repeat("6", 24), CheckDigest: strings.Repeat("7", 64), ExitCode: &zero, OutputDigest: strings.Repeat("8", 64), Retirement: "confirmed"}
		fault := ""
		calls := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls++
			var got project.FactoryVerificationChecks
			if r.Method != http.MethodPost || r.URL.Path != "/factory-verification-checks" || json.NewDecoder(r.Body).Decode(&got) != nil || got != request {
				t.Error("native verification request changed admitted identity")
				w.WriteHeader(400)
				return
			}
			state := good
			switch fault {
			case "run":
				state.Run = strings.Repeat("9", 32)
			case "invocation":
				state.Incarnation = strings.Repeat("9", 32)
			case "manifest":
				state.ManifestDigest = strings.Repeat("9", 64)
			case "custody":
				state.Retirement = "uncertain"
			case "output":
				state.OutputDigest = ""
			case "pending":
				state.Phase = "running"
				state.ExitCode = nil
				state.OutputDigest = ""
				state.Retirement = ""
			}
			_ = json.NewEncoder(w).Encode(state)
		}))
		defer server.Close()
		client := &Client{HTTP: &http.Client{Transport: applicationTestTransport{server.URL}}}
		for _, variant := range []string{"", "run", "invocation", "manifest", "custody", "output", "pending"} {
			fault = variant
			state, err := client.FactoryVerificationChecks(context.Background(), request)
			if (err == nil) != (variant == "" || variant == "pending") {
				t.Fatalf("verification response %s accepted=%v", variant, err == nil)
			}
			if variant == "pending" && state.Phase != "running" {
				t.Fatal("pending native checks were promoted to a pass")
			}
		}
		before := calls
		invalid := request
		invalid.Incarnation = ""
		if _, err := client.FactoryVerificationChecks(context.Background(), invalid); err == nil || calls != before {
			t.Fatal("unbound verification reached native operation")
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
