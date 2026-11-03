package host

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/levitateos/factory/pkg/application"
	"github.com/levitateos/factory/pkg/collaboration"
	"github.com/levitateos/factory/pkg/factory/execution"
	"github.com/levitateos/factory/pkg/identity"
	"github.com/levitateos/factory/pkg/project"
	"github.com/levitateos/factory/pkg/store"
	identityclient "github.com/levitateos/sodaos/internal/identity/client"
)

// Exercise the production host and broker clients over a real private Unix
// socket, plus the actual Rust broker and PostgreSQL across a process restart.
// The host endpoint supplies wire metadata; this is not an agent execution or
// a claim that the privileged runtime retired a real agent.
func TestFactoryObservationPrivateUnixBoundary(t *testing.T) {
	admin, available := store.TestFixtureDSN()
	binary := os.Getenv("SODA_FACTORY_OBSERVATION_BROKER_BIN")
	if !available || binary == "" {
		t.Skip("native PostgreSQL and the actual broker executable are required")
	}
	db, brokerDSN, cleanup, err := store.OpenEphemeral(context.Background(), admin)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	address := execution.RunAddress{Run: strings.Repeat("a", 32), Workspace: "p" + strings.Repeat("b", 24)}
	invocation := strings.Repeat("c", 32)
	preparationAddress := execution.PreparationAddress{Workspace: address.Workspace, ID: "f" + strings.Repeat("a", 24)}
	setup := project.ApprovedSetup{Files: map[string][]byte{project.FactorySetupEntry: []byte("fixture setup bytes"), project.FactoryCheckEntry: []byte("fixture check bytes")}, Bundle: []byte("fixture source bundle")}
	recipe := project.Preparation{ID: preparationAddress.ID, Project: address.Workspace, Role: project.RoleCoder, Requirements: project.RequirementAcceptance{ID: "d" + strings.Repeat("1", 24), Revision: 1, Approver: 7, SourceCommit: strings.Repeat("a", 40), Digest: strings.Repeat("b", 64)}, Approval: project.AdminApproval{ID: "d" + strings.Repeat("2", 24), Revision: 1, Approver: 7, EffectsDigest: strings.Repeat("c", 64)}, SourceCommit: strings.Repeat("a", 40), SetupDigest: project.SetupDigestOf(setup.Files), Tools: []string{"git"}}
	standing := execution.StandingPreparationRequest{Input: project.Prepare{Preparation: recipe, Setup: setup}}
	candidate := execution.CandidatePreparationRequest{Input: project.FactoryCandidate{Preparation: recipe, SourcePreparation: "f" + strings.Repeat("b", 24), Deadline: time.Now().Add(time.Minute)}}
	contextRequest := execution.PreparationContextRequest{Input: project.FactoryPreparationContextRequest{Project: recipe.Project, ID: recipe.ID, SourceCommit: recipe.SourceCommit, ApprovedBase: recipe.SourceCommit, DiffBase: strings.Repeat("b", 40), Candidate: strings.Repeat("c", 40), Paths: []string{"src/main.go"}, NotAfter: time.Now().Add(time.Minute)}, Role: recipe.Role}
	launchPrompt := []byte("immutable launch fixture")
	launchRequest := execution.LaunchRequest{Kind: collaboration.Appliance, Input: project.FactoryLaunch{Run: project.FactoryRun{ID: address.Run, Project: address.Workspace, Role: recipe.Role, Preparation: recipe.ID, Harness: project.FactoryHarnessMuse, HarnessVers: "fixture-1", Model: "fixture-model", Assignment: project.FactoryPromptDigest(launchPrompt), SourceCommit: recipe.SourceCommit, Connection: "fixture-connection", Actor: 7, Deadline: time.Now().Add(time.Minute), ExecutionPhase: project.ExecutionInvestigate}, Prompt: launchPrompt, HarnessSHA256: strings.Repeat("a", 64)}, Appliance: &execution.ApplianceLaunch{Profile: project.Profile{ID: project.RockyHeadless, Distribution: "rocky", Version: "10.2", Interface: "headless", Architecture: "amd64", Image: "sha256:" + strings.Repeat("b", 64), Revision: strings.Repeat("c", 40)}, Tool: execution.ApplianceTool{Harness: project.FactoryHarnessMuse, Version: "fixture-1", SHA256: strings.Repeat("a", 64), Image: "sha256:" + strings.Repeat("d", 64)}}}
	var mode, calls atomic.Int64
	var sealed atomic.Bool
	directory, err := os.MkdirTemp("", "run-observation-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(directory) })
	socket := filepath.Join(directory, "private.sock")
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(socket, 0o600); err != nil {
		t.Fatal(err)
	}
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/factory-launch":
			var request project.FactoryLaunch
			decoder := json.NewDecoder(r.Body)
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&request); err != nil || request.Validate() != nil || request.Run.ID != address.Run || request.Run.Project != address.Workspace || request.Run.Role != recipe.Role || request.Run.Preparation != recipe.ID || !request.Run.Deadline.Equal(launchRequest.Input.Run.Deadline) || !bytes.Equal(request.Prompt, launchPrompt) || request.HarnessSHA256 != launchRequest.Input.HarnessSHA256 {
				t.Error("launch changed captured wire inputs", err)
				w.WriteHeader(400)
				return
			}
			wire, err := json.Marshal(request)
			captured, captureErr := json.Marshal(launchRequest.Input)
			if err != nil || captureErr != nil || !bytes.Equal(wire, captured) {
				t.Error("appliance launch wire payload changed", err, captureErr)
				w.WriteHeader(400)
				return
			}
			result := project.FactoryState{ID: address.Run, Project: address.Workspace, Role: recipe.Role, Phase: project.FactoryRunning, Live: true, Container: strings.Repeat("d", 64), Unit: "fixture-launch.service", Invocation: invocation, Generation: 3, UID: 1001, GID: 1002}
			switch mode.Load() {
			case 57:
				result.Role = project.RoleReviewer
			case 58:
				result.ID = strings.Repeat("e", 32)
			case 59:
				result.Project = "p" + strings.Repeat("e", 24)
			case 60:
				result.Invocation = ""
			case 61:
				w.WriteHeader(503)
				return
			case 62:
				result.Role = ""
			}
			_ = json.NewEncoder(w).Encode(result)
		case "/prepare-context":
			var request project.FactoryPreparationContextRequest
			decoder := json.NewDecoder(r.Body)
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&request); err != nil || request.Validate() != nil || request.Project != recipe.Project || request.ID != recipe.ID || request.SourceCommit != contextRequest.Input.SourceCommit || request.ApprovedBase != contextRequest.Input.ApprovedBase || request.DiffBase != contextRequest.Input.DiffBase || request.Candidate != contextRequest.Input.Candidate || len(request.Paths) != 1 || request.Paths[0] != "src/main.go" || request.NotAfter.After(contextRequest.Input.NotAfter) || !request.NotAfter.After(time.Now()) {
				t.Error("context read changed the captured wire selection/deadline", err)
				w.WriteHeader(400)
				return
			}
			result := project.FactoryPreparationContext{Project: request.Project, ID: request.ID, Role: recipe.Role, SourceCommit: request.SourceCommit, ApprovedBase: request.ApprovedBase, DiffBase: request.DiffBase, Candidate: request.Candidate, Setup: []byte("approved setup fixture"), Check: []byte("approved check fixture"), Files: []project.FactoryContextFile{{Path: "src/main.go", Content: []byte("approved source fixture")}}, Diff: []byte("exact diff fixture")}
			switch mode.Load() {
			case 41:
				result.Project = "p" + strings.Repeat("e", 24)
			case 42:
				result.Role = project.RoleReviewer
			case 43:
				result.SourceCommit = strings.Repeat("e", 40)
			case 44:
				result.ApprovedBase = strings.Repeat("e", 40)
			case 45:
				result.DiffBase = strings.Repeat("e", 40)
			case 46:
				result.Candidate = strings.Repeat("e", 40)
			case 47:
				result.Check = nil
			case 48:
				result.Files[0].Content = bytes.Repeat([]byte("x"), project.MaxFactoryContextFileBytes+1)
			case 49:
				result.Diff = bytes.Repeat([]byte("x"), project.MaxFactoryContextBytes+1)
			case 50:
				result.Setup = []byte{0xff}
			case 51:
				w.WriteHeader(413)
				return
			case 52:
				w.WriteHeader(503)
				return
			case 53:
				w.WriteHeader(404)
				return
			case 54:
				_, _ = w.Write([]byte(`{"invalid":`))
				return
			case 55:
				result.ID = "f" + strings.Repeat("e", 24)
			case 56:
				_, _ = w.Write(bytes.Repeat([]byte("x"), factoryContextResponseLimit+1))
				return
			}
			_ = json.NewEncoder(w).Encode(result)
		case "/prepare", "/prepare-candidate", "/prepare-inspect", "/prepare-stop":
			switch r.URL.Path {
			case "/prepare":
				var request project.Prepare
				if err := json.NewDecoder(r.Body).Decode(&request); err != nil || request.Validate() != nil || request.Preparation.ID != recipe.ID || request.Preparation.SetupDigest != recipe.SetupDigest {
					t.Error("standing preparation changed approved inputs", err)
					w.WriteHeader(400)
					return
				}
			case "/prepare-candidate":
				var request project.FactoryCandidate
				if err := json.NewDecoder(r.Body).Decode(&request); err != nil || request.Validate() != nil || request.Preparation.ID != recipe.ID || request.SourcePreparation != candidate.Input.SourcePreparation || !request.Deadline.Equal(candidate.Input.Deadline) {
					t.Error("candidate preparation changed captured source/deadline", err)
					w.WriteHeader(400)
					return
				}
			default:
				var request project.PrepareInspect
				if err := json.NewDecoder(r.Body).Decode(&request); err != nil || request.Project != recipe.Project || request.ID != recipe.ID {
					t.Error("preparation observation changed scope", err)
					w.WriteHeader(400)
					return
				}
			}
			if mode.Load() == 36 {
				w.WriteHeader(404)
				return
			}
			if mode.Load() == 37 {
				w.WriteHeader(503)
				return
			}
			zero := 0
			result := project.PrepareState{ID: recipe.ID, Project: recipe.Project, Role: recipe.Role, Phase: project.PrepareReady, Ready: true, Container: strings.Repeat("d", 64), SourceCommit: recipe.SourceCommit, SetupDigest: recipe.SetupDigest, Tools: []project.ResolvedTool{{Name: "git", Path: "/usr/bin/git", Version: "fixture"}}, SetupExit: &zero, CheckExit: &zero, Output: "fixture completed checks"}
			if r.URL.Path == "/prepare-stop" {
				result.Phase, result.Ready, result.Stopped, result.Retirement = project.PrepareStopped, false, true, "confirmed"
			}
			switch mode.Load() {
			case 30:
				result.ID = "f" + strings.Repeat("c", 24)
			case 31:
				result.Project = "p" + strings.Repeat("e", 24)
			case 32:
				result.Role = project.RoleReviewer
			case 33:
				result.SourceCommit = strings.Repeat("b", 40)
			case 34:
				result.SetupDigest = strings.Repeat("c", 64)
			case 35:
				result.Container = ""
			case 38:
				result.Retirement = "uncertain"
			case 39:
				result = project.PrepareState{ID: recipe.ID, Project: recipe.Project, Phase: project.PrepareStopped, Stopped: true, Retirement: "confirmed"}
			case 40:
				result.Retirement = "future-unknown"
			}
			_ = json.NewEncoder(w).Encode(result)
		case "/factory-takeover":
			var request project.FactoryTakeover
			decoder := json.NewDecoder(r.Body)
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&request); err != nil || request.Validate() != nil || request.Project != address.Workspace || request.ID != address.Run || request.Member != "alice" {
				t.Error("recovery changed captured run/member", err)
				w.WriteHeader(400)
				return
			}
			result := project.TakeoverResult{ID: request.ID, Project: request.Project, Member: request.Member, Destination: project.TakeoverDestination(request.Member, request.ID), Reused: mode.Load() == 27}
			switch mode.Load() {
			case 23:
				result.Project = "p" + strings.Repeat("f", 24)
			case 24:
				result.Member = "bob"
				result.Destination = project.TakeoverDestination("bob", request.ID)
			case 25:
				result.Destination = "/home/alice/another-copy"
			case 26:
				w.WriteHeader(503)
				return
			}
			_ = json.NewEncoder(w).Encode(result)
		case "/factory-task-checkpoint", "/factory-final-candidate", "/factory-final-candidate-receipt":
			final := r.URL.Path != "/factory-task-checkpoint"
			var request project.FactoryTaskCheckpoint
			decoder := json.NewDecoder(r.Body)
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&request); err != nil || request.Validate(final) != nil || request.Project != address.Workspace || request.ID != address.Run || request.Invocation != invocation || request.Container != strings.Repeat("d", 64) || request.ExpectedHead != strings.Repeat("a", 40) || request.ManifestDigest != strings.Repeat("e", 64) {
				t.Error("checkpoint changed its captured wire binding", err)
				w.WriteHeader(400)
				return
			}
			result := project.FactoryTaskCheckpointState{Request: request, FinalCandidate: final, Status: "committed", Candidate: strings.Repeat("b", 40), Tree: strings.Repeat("c", 40), ReceiptDigest: strings.Repeat("d", 64)}
			if final {
				result.Candidate = request.ExpectedHead
			}
			switch mode.Load() {
			case 18:
				result.Request.Invocation = strings.Repeat("f", 32)
			case 19:
				result.Request.ExpectedHead = strings.Repeat("f", 40)
			case 20:
				result.Status = "future-unknown-state"
			case 21:
				result.ReceiptDigest = "malformed"
			case 22:
				w.WriteHeader(503)
				return
			}
			_ = json.NewEncoder(w).Encode(result)
		case "/factory-harness":
			var request struct {
				Harness string `json:"harness"`
			}
			decoder := json.NewDecoder(r.Body)
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&request); err != nil || request.Harness != project.FactoryHarnessMuse {
				t.Error("tool observation changed requested family", err)
				w.WriteHeader(400)
				return
			}
			pin := project.FactoryHarnessPin{Harness: request.Harness, Version: "fixture-1", SHA256: strings.Repeat("a", 64), Image: "sha256:" + strings.Repeat("b", 64)}
			switch mode.Load() {
			case 14:
				pin.Harness = "foreign-family"
			case 15:
				pin.SHA256 = ""
			case 16:
				pin.Image = ""
			case 17:
				w.WriteHeader(503)
				return
			}
			_ = json.NewEncoder(w).Encode(pin)
		case "/factory-export":
			var request project.FactoryExport
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil || request.Validate() != nil || request.ID != address.Run || request.Project != address.Workspace {
				t.Error("candidate export escaped its captured address", err)
				w.WriteHeader(400)
				return
			}
			result := project.FactoryExportState{ID: request.ID, Project: request.Project, Candidate: request.Candidate, Phase: project.FactoryCompleted, Bundle: base64.StdEncoding.EncodeToString([]byte("fixture bundle bytes"))}
			if mode.Load() == 12 {
				result.Bundle = "invalid base64"
			} else if mode.Load() == 13 {
				result.Candidate = strings.Repeat("f", 40)
			}
			_ = json.NewEncoder(w).Encode(result)
		case "/factory-inspect", "/factory-stop":
			var request project.FactoryInspect
			decoder := json.NewDecoder(r.Body)
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&request); err != nil || request.ID != address.Run || request.Project != address.Workspace {
				t.Error("run identity or private request changed", err)
				w.WriteHeader(400)
				return
			}
			if mode.Load() == 5 || mode.Load() == 6 {
				status := http.StatusNotFound
				if mode.Load() == 6 {
					status = http.StatusInternalServerError
				}
				w.WriteHeader(status)
				return
			}
			state := project.FactoryState{
				ID: address.Run, Project: address.Workspace, Role: project.RoleCoder, Phase: project.FactoryRunning, Live: true,
				Container: strings.Repeat("d", 64), Unit: "factory-fixture.service", Invocation: invocation, UID: 1001, GID: 1002, Login: "factory-fixture", Generation: 3, LeaseID: "retained-lease", Delivered: true,
				VerificationChecks: &project.FactoryVerificationChecksState{Project: address.Workspace, Run: address.Run, Incarnation: invocation, ManifestDigest: strings.Repeat("e", 64), Phase: "running"},
			}
			switch mode.Load() {
			case 1:
				code := 0
				state.Phase, state.Live, state.ExitCode = project.FactoryCompleted, false, &code
			case 2, 11:
				state.Phase, state.Live, state.Retirement = project.FactoryStopped, false, "confirmed"
				if mode.Load() == 11 {
					state.Retirement = "future-unknown-proof"
				}
			case 3:
				state = project.FactoryState{ID: address.Run, Project: address.Workspace, Phase: project.FactoryStopped, Retirement: "confirmed"}
			case 4:
				state.ID = strings.Repeat("f", 32)
			}
			_ = json.NewEncoder(w).Encode(state)
		case "/execution/get", "/execution/close":
			var request identity.Request
			decoder := json.NewDecoder(r.Body)
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&request); err != nil || request.Kind != identity.Factory || request.ExecutionID != address.Run {
				t.Error("broker request escaped its factory execution identity", err)
				w.WriteHeader(400)
				return
			}
			if r.URL.Path == "/execution/close" {
				sealed.Store(true)
				_, _ = w.Write([]byte(`{}`))
				return
			}
			if mode.Load() == 10 {
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte("missing"))
				return
			}
			metadata := identity.Execution{
				Kind: identity.Factory, ExecutionID: address.Run, State: identity.ExecutionLive, Digest: strings.Repeat("a", 64), LeaseID: "retained-lease",
				Binding: &identity.Binding{Kind: identity.Factory, ID: address.Run, Project: strings.Repeat("d", 64), Generation: 3, InvocationID: invocation, UID: 1001, GID: 1002, Login: "factory-fixture", CredentialRoot: "/fixture/private-root", Scope: "factory", ChildID: "child-1"},
			}
			if sealed.Load() {
				metadata.State = identity.ExecutionTerminal
			}
			switch mode.Load() {
			case 3:
				metadata = identity.Execution{Kind: identity.Factory, ExecutionID: address.Run, State: identity.ExecutionTerminal}
			case 7:
				metadata.State, metadata.Binding, metadata.LeaseID = identity.ExecutionPending, nil, ""
			case 8:
				metadata.Binding.ID = strings.Repeat("f", 32)
			case 9:
				metadata.State = "future-unknown-state"
			}
			_ = json.NewEncoder(w).Encode(metadata)
		default:
			t.Error("unexpected runtime or credential delivery route", r.URL.Path)
			w.WriteHeader(404)
		}
	})}
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() { _ = server.Close() })
	hostClient, brokerClient := NewClient(socket), identityclient.New(socket)
	t.Cleanup(hostClient.HTTP.CloseIdleConnections)
	t.Cleanup(brokerClient.HTTP.CloseIdleConnections)
	observer := NewFactoryObserver(hostClient, brokerClient)
	ctx := context.Background()
	launched, err := hostClient.LaunchRun(ctx, launchRequest)
	if err != nil || !launchRequest.MatchesObservation(launched) || launched.Appliance.Invocation.Invocation != invocation || launched.Native != nil {
		t.Fatal("launch lost captured scope or actual process observation", err)
	}
	for invalid := int64(57); invalid <= 62; invalid++ {
		mode.Store(invalid)
		if _, err := hostClient.LaunchRun(ctx, launchRequest); err == nil {
			t.Fatal("foreign/unavailable launch observation accepted", invalid)
		}
	}
	mode.Store(0)
	nativeLaunch := launchRequest
	nativeLaunch.Kind, nativeLaunch.Appliance = collaboration.Native, nil
	nativeLaunch.Native = &execution.NativeLaunch{Workspace: execution.NativeWorkspaceRef{Profile: collaboration.ProfileID(strings.Repeat("a", 32)), InstanceID: strings.Repeat("b", 32), WorktreeID: "owned-launch-fixture"}, Tool: execution.NativeTool{Harness: project.FactoryHarnessMuse, Version: "fixture-1", SHA256: strings.Repeat("a", 64), Executable: "/home/fixture/bin/muse"}, Runtime: collaboration.RuntimeID(strings.Repeat("e", 32)), PlacementGeneration: 7}
	if err := nativeLaunch.Validate(); err != nil {
		t.Fatal("native launch required appliance facts", err)
	}
	beforeLaunch := calls.Load()
	if _, err := hostClient.LaunchRun(ctx, nativeLaunch); err == nil || calls.Load() != beforeLaunch {
		t.Fatal("native launch crossed appliance transport", err)
	}
	mixedLaunch := nativeLaunch
	mixedLaunch.Appliance = launchRequest.Appliance
	if mixedLaunch.Validate() == nil {
		t.Fatal("mixed runtime launch accepted")
	}
	expiredLaunch := launchRequest
	expiredLaunch.Input.Run.Deadline = time.Now().Add(-time.Second)
	if _, err := hostClient.LaunchRun(ctx, expiredLaunch); !errors.Is(err, context.DeadlineExceeded) || calls.Load() != beforeLaunch {
		t.Fatal("expired launch reached appliance runtime", err)
	}
	nativeLaunchJSON, err := json.Marshal(nativeLaunch)
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{`"image"`, `"uid"`, `"gid"`, `"container"`, `"appliance"`} {
		if bytes.Contains(nativeLaunchJSON, []byte(field)) {
			t.Fatal("native launch invented appliance inputs", field)
		}
	}
	contextMaterial, err := hostClient.ReadWorkspaceContext(ctx, contextRequest)
	if err != nil || contextMaterial.ValidateFor(contextRequest) != nil || string(contextMaterial.Material.Files[0].Content) != "approved source fixture" || string(contextMaterial.Material.Diff) != "exact diff fixture" || contextMaterial.Native != nil {
		t.Fatal("context translation changed approved bytes or scope", err)
	}
	for invalid := int64(41); invalid <= 56; invalid++ {
		mode.Store(invalid)
		_, err := hostClient.ReadWorkspaceContext(ctx, contextRequest)
		if err == nil || errors.Is(err, execution.ErrPreparationContextBounds) != (invalid == 51) || errors.Is(err, execution.ErrPreparationNotObserved) {
			t.Fatal("context identity/bounds/transport refusal was misclassified", invalid, err)
		}
	}
	mode.Store(0)
	nativeContext := contextRequest
	ownerRef := execution.NativeWorkspaceRef{Profile: collaboration.ProfileID(strings.Repeat("a", 32)), InstanceID: strings.Repeat("b", 32), WorktreeID: "owned-context-checkout"}
	nativeContext.Native = &ownerRef
	beforeContext := calls.Load()
	if _, err := hostClient.ReadWorkspaceContext(ctx, nativeContext); err == nil || calls.Load() != beforeContext {
		t.Fatal("native context crossed appliance transport", err)
	}
	expired := contextRequest
	expired.Input.NotAfter = time.Now().Add(-time.Second)
	if _, err := hostClient.ReadWorkspaceContext(ctx, expired); !errors.Is(err, context.DeadlineExceeded) || calls.Load() != beforeContext {
		t.Fatal("expired context selection reached the runtime", err)
	}
	invalidSelection := contextRequest
	invalidSelection.Input.Paths = []string{"../outside"}
	if _, err := hostClient.ReadWorkspaceContext(ctx, invalidSelection); err == nil || calls.Load() != beforeContext {
		t.Fatal("context path escaped its relative selection", err)
	}
	nativeMaterial := contextMaterial
	nativeMaterial.Native = &ownerRef
	if nativeMaterial.ValidateFor(nativeContext) != nil || nativeMaterial.ValidateFor(contextRequest) == nil || contextMaterial.ValidateFor(nativeContext) == nil {
		t.Fatal("context variants lost captured native ownership")
	}
	foreignContext := nativeContext
	foreignRef := ownerRef
	foreignRef.WorktreeID = "another-context-checkout"
	foreignContext.Native = &foreignRef
	if nativeMaterial.ValidateFor(foreignContext) == nil {
		t.Fatal("context response changed native worktree ownership")
	}
	tool, err := hostClient.InspectTool(ctx, project.FactoryHarnessMuse)
	if err != nil || tool.Kind != collaboration.Appliance || tool.Native != nil || tool.Appliance == nil || tool.Appliance.Version != "fixture-1" || tool.Appliance.SHA256 != strings.Repeat("a", 64) || tool.Appliance.Image != "sha256:"+strings.Repeat("b", 64) {
		t.Fatal("tool observation changed captured appliance bytes", tool, err)
	}
	for _, invalid := range []int64{14, 15, 16, 17} {
		mode.Store(invalid)
		if _, err := hostClient.InspectTool(ctx, project.FactoryHarnessMuse); err == nil {
			t.Fatal("invalid/unavailable appliance tool was accepted", invalid)
		}
	}
	beforeTool := calls.Load()
	if _, err := hostClient.InspectTool(ctx, "foreign-family"); err == nil || calls.Load() != beforeTool {
		t.Fatal("unsupported tool selector reached the private endpoint", err)
	}
	mode.Store(0)
	if standing.Validate() != nil || candidate.Validate() != nil {
		t.Fatal("invalid preparation fixture")
	}
	prepared, err := hostClient.PrepareWorkspace(ctx, standing)
	if err != nil || !prepared.Ready || prepared.Native != nil || prepared.Container != strings.Repeat("d", 64) || prepared.SetupDigest != recipe.SetupDigest || prepared.Tools[0].Path != "/usr/bin/git" || prepared.CheckExit == nil || *prepared.CheckExit != 0 || prepared.RetirementState() != execution.Uncertain {
		t.Fatal("preparation translation changed input/readiness facts", prepared, err)
	}
	preparedCandidate, err := hostClient.PrepareWorkspaceCandidate(ctx, candidate)
	if err != nil || preparedCandidate.ValidateFor(preparationAddress) != nil || !preparedCandidate.Ready {
		t.Fatal("candidate preparation lost approved scope", preparedCandidate, err)
	}
	observedPreparation, err := hostClient.InspectWorkspacePreparation(ctx, preparationAddress)
	if err != nil || observedPreparation.SourceCommit != recipe.SourceCommit {
		t.Fatal("preparation inspection lost source", observedPreparation, err)
	}
	wirePreparation, err := hostClient.InspectPreparation(ctx, project.PrepareInspect{Project: recipe.Project, ID: recipe.ID})
	if err != nil {
		t.Fatal(err)
	}
	retainedPreparation := execution.PreparationRecord{Preparation: recipe, State: observedPreparation}
	retainedJSON, err := json.Marshal(retainedPreparation)
	if err != nil || retainedPreparation.Validate() != nil {
		t.Fatal("invalid translated preparation record", err)
	}
	wireJSON, err := json.Marshal(struct {
		Preparation project.Preparation  `json:"preparation"`
		State       project.PrepareState `json:"state,omitempty"`
	}{Preparation: recipe, State: wirePreparation})
	if err != nil || !bytes.Equal(retainedJSON, wireJSON) {
		t.Fatal("appliance preparation retained encoding changed", err)
	}
	var retainedDecoded execution.PreparationRecord
	if err := json.Unmarshal(wireJSON, &retainedDecoded); err != nil || retainedDecoded.Validate() != nil {
		t.Fatal("appliance retained record cannot be read", err)
	}
	emptyJSON, err := json.Marshal(execution.PreparationRecord{Preparation: recipe})
	if err != nil {
		t.Fatal(err)
	}
	wireEmptyJSON, err := json.Marshal(struct {
		Preparation project.Preparation  `json:"preparation"`
		State       project.PrepareState `json:"state,omitempty"`
	}{Preparation: recipe})
	if err != nil || !bytes.Equal(emptyJSON, wireEmptyJSON) {
		t.Fatal("unobserved appliance preparation encoding changed", err)
	}
	for _, invalid := range []int64{30, 31, 32, 33, 34, 35, 37} {
		mode.Store(invalid)
		if _, err := hostClient.PrepareWorkspace(ctx, standing); err == nil {
			t.Fatal("foreign/missing/unavailable preparation accepted", invalid)
		}
	}
	mode.Store(36)
	if _, err := hostClient.InspectWorkspacePreparation(ctx, preparationAddress); !errors.Is(err, execution.ErrPreparationNotObserved) {
		t.Fatal("preparation absence was not typed", err)
	}
	mode.Store(37)
	if _, err := hostClient.InspectWorkspacePreparation(ctx, preparationAddress); err == nil || errors.Is(err, execution.ErrPreparationNotObserved) {
		t.Fatal("preparation transport failure became absence", err)
	}
	mode.Store(38)
	retiredPreparation, err := hostClient.RetireWorkspacePreparation(ctx, preparationAddress)
	if err != nil || retiredPreparation.RetirementState() != execution.Uncertain {
		t.Fatal("uncertain preparation stop became retired", retiredPreparation, err)
	}
	mode.Store(39)
	retiredPreparation, err = hostClient.RetireWorkspacePreparation(ctx, preparationAddress)
	if err != nil || retiredPreparation.RetirementState() != execution.Retired || retiredPreparation.Container != "" || retiredPreparation.Role != "" || retiredPreparation.SourceCommit != "" {
		t.Fatal("stop tombstone invented preparation facts", retiredPreparation, err)
	}
	mode.Store(40)
	if _, err := hostClient.RetireWorkspacePreparation(ctx, preparationAddress); err == nil {
		t.Fatal("unknown preparation retirement accepted as confirmed")
	}
	nativeRef := execution.NativeWorkspaceRef{Profile: collaboration.ProfileID(strings.Repeat("a", 32)), InstanceID: strings.Repeat("b", 32), WorktreeID: "owned-worktree"}
	nativeAddress := preparationAddress
	nativeAddress.Native = &nativeRef
	nativeStanding, nativeCandidate := standing, candidate
	nativeStanding.Native, nativeCandidate.Native = &nativeRef, &nativeRef
	if nativeStanding.Validate() != nil || nativeCandidate.Validate() != nil {
		t.Fatal("native request required an appliance image or UID")
	}
	beforePreparation := calls.Load()
	if _, err := hostClient.PrepareWorkspace(ctx, nativeStanding); err == nil {
		t.Fatal("native standing preparation used appliance transport")
	}
	if _, err := hostClient.PrepareWorkspaceCandidate(ctx, nativeCandidate); err == nil {
		t.Fatal("native candidate preparation used appliance transport")
	}
	if _, err := hostClient.InspectWorkspacePreparation(ctx, nativeAddress); err == nil {
		t.Fatal("native preparation inspection used appliance transport")
	}
	if _, err := hostClient.RetireWorkspacePreparation(ctx, nativeAddress); err == nil || calls.Load() != beforePreparation {
		t.Fatal("native preparation invented an appliance boundary", err)
	}
	nativePrepared := prepared
	nativePrepared.Native, nativePrepared.Container = &nativeRef, ""
	nativeRecord := execution.PreparationRecord{Preparation: recipe, State: nativePrepared, Native: &nativeRef}
	if nativeRecord.Validate() != nil {
		t.Fatal("native ready preparation required a container")
	}
	nativeJSON, err := json.Marshal(nativeRecord)
	if err != nil {
		t.Fatal(err)
	}
	var nativeDecoded execution.PreparationRecord
	if err := json.Unmarshal(nativeJSON, &nativeDecoded); err != nil || nativeDecoded.Validate() != nil || bytes.Contains(nativeJSON, []byte(`"container"`)) {
		t.Fatal("native preparation record invented a container", err)
	}
	unobservedNativeJSON, err := json.Marshal(execution.PreparationRecord{Preparation: recipe, Native: &nativeRef})
	if err != nil {
		t.Fatal(err)
	}
	var unobservedNative execution.PreparationRecord
	if err := json.Unmarshal(unobservedNativeJSON, &unobservedNative); err != nil || unobservedNative.Validate() != nil || bytes.Contains(unobservedNativeJSON, []byte(`"container"`)) || bytes.Contains(unobservedNativeJSON, []byte(`"state"`)) {
		t.Fatal("unobserved native preparation invented runtime state", err)
	}
	mixed := nativePrepared
	mixed.Container = strings.Repeat("d", 64)
	if mixed.ValidateFor(nativeAddress) == nil {
		t.Fatal("mixed preparation variants accepted")
	}
	foreignNative := nativeRef
	foreignNative.WorktreeID = "another-worktree"
	if nativePrepared.ValidateFor(execution.PreparationAddress{Workspace: recipe.Project, ID: recipe.ID, Native: &foreignNative}) == nil {
		t.Fatal("native preparation changed captured worktree")
	}
	mode.Store(0)
	recovery := execution.WorkRecoveryRequest{Address: address, Member: "alice"}
	copied, err := hostClient.RecoverWork(ctx, recovery)
	if err != nil || copied.ValidateFor(recovery) != nil || copied.Reused || copied.Destination != project.TakeoverDestination("alice", address.Run) {
		t.Fatal("appliance recovery translation changed destination", copied, err)
	}
	mode.Store(27)
	copied, err = hostClient.RecoverWork(ctx, recovery)
	if err != nil || !copied.Reused {
		t.Fatal("recovery lost native reuse acknowledgement", copied, err)
	}
	for _, invalid := range []int64{23, 24, 25, 26} {
		mode.Store(invalid)
		if _, err := hostClient.RecoverWork(ctx, recovery); err == nil {
			t.Fatal("foreign/unavailable recovery was accepted", invalid)
		}
	}
	nativeRecovery := execution.WorkRecoveryRequest{Address: address, Native: &execution.NativeRecovery{Source: execution.NativeWorkspaceRef{Profile: collaboration.ProfileID(strings.Repeat("b", 32)), InstanceID: strings.Repeat("c", 32), WorktreeID: "owned-worktree"}, Target: execution.NativeCheckoutRef{Profile: collaboration.ProfileID(strings.Repeat("b", 32)), CheckoutID: strings.Repeat("d", 32)}}}
	if err := nativeRecovery.Validate(); err != nil {
		t.Fatal(err)
	}
	beforeRecovery := calls.Load()
	if _, err := hostClient.RecoverWork(ctx, nativeRecovery); err == nil || calls.Load() != beforeRecovery {
		t.Fatal("native recovery fabricated an appliance member", err)
	}
	mode.Store(0)
	checkpoint := execution.CheckpointRequest{Project: address.Workspace, ID: address.Run, Container: strings.Repeat("d", 64), Invocation: invocation, ManifestDigest: strings.Repeat("e", 64), TaskKey: "task-one", OutputDigest: strings.Repeat("f", 64), ExpectedHead: strings.Repeat("a", 40)}
	confirmed, err := hostClient.CheckpointTask(ctx, checkpoint)
	if err != nil || !confirmed.Request.Equal(checkpoint) || confirmed.Candidate != strings.Repeat("b", 40) {
		t.Fatal("task checkpoint translation changed captured binding", confirmed, err)
	}
	checkpoint.TaskKey, checkpoint.OutputDigest = "", ""
	confirmed, err = hostClient.CheckpointFinal(ctx, checkpoint)
	if err != nil || !confirmed.Request.Equal(checkpoint) || !confirmed.FinalCandidate || confirmed.Candidate != checkpoint.ExpectedHead {
		t.Fatal("final checkpoint translation changed captured binding", confirmed, err)
	}
	readback, err := hostClient.ReadFinalCheckpoint(ctx, checkpoint)
	if err != nil || !readback.Request.Equal(confirmed.Request) || readback.ReceiptDigest != confirmed.ReceiptDigest {
		t.Fatal("read-only checkpoint recovery changed its receipt", readback, err)
	}
	for _, invalid := range []int64{18, 19, 20, 21, 22} {
		mode.Store(invalid)
		if _, err := hostClient.ReadFinalCheckpoint(ctx, checkpoint); err == nil {
			t.Fatal("foreign/malformed checkpoint accepted", invalid)
		}
	}
	nativeCheckpoint := checkpoint
	nativeCheckpoint.Container = ""
	nativeCheckpoint.Native = &execution.NativeCheckpointAnchor{Profile: collaboration.ProfileID(strings.Repeat("b", 32)), InstanceID: strings.Repeat("c", 32), WorktreeID: "owned-fixture", SystemInstance: "boot", StartIdentity: "start", PID: 1001, ExecutableSHA256: strings.Repeat("e", 64)}
	if err := nativeCheckpoint.Validate(true); err != nil {
		t.Fatal(err)
	}
	beforeCheckpoint := calls.Load()
	if _, err := hostClient.CheckpointFinal(ctx, nativeCheckpoint); err == nil {
		t.Fatal("native final checkpoint reached appliance transport")
	}
	if _, err := hostClient.ReadFinalCheckpoint(ctx, nativeCheckpoint); err == nil {
		t.Fatal("native checkpoint read reached appliance transport")
	}
	nativeCheckpoint.TaskKey, nativeCheckpoint.OutputDigest = "task-one", strings.Repeat("f", 64)
	if _, err := hostClient.CheckpointTask(ctx, nativeCheckpoint); err == nil || calls.Load() != beforeCheckpoint {
		t.Fatal("native checkpoint fabricated an appliance binding", err)
	}
	mode.Store(0)
	state, err := observer.InspectRun(ctx, address)
	if err != nil || state.Kind != collaboration.Appliance || state.Native != nil || state.Appliance == nil || state.Retirement != execution.Running || !state.Live {
		t.Fatal("live appliance facts changed", state, err)
	}
	boundary := state.Appliance.Invocation
	if boundary.Invocation != invocation || boundary.UID != 1001 || boundary.GID != 1002 || boundary.Generation != 3 || boundary.Container != strings.Repeat("d", 64) || state.Appliance.LeaseID != "retained-lease" || !state.Appliance.CredentialDelivered || state.Appliance.CredentialReturned || state.VerificationChecks == nil || state.VerificationChecks.ManifestDigest != strings.Repeat("e", 64) {
		t.Fatal("original run boundary or manifest facts changed")
	}
	custody, err := observer.ReadCustody(ctx, address)
	if err != nil || custody.Kind != collaboration.Appliance || custody.Native != nil || custody.Appliance == nil || custody.Appliance.State != execution.Held || custody.Sealed || custody.Appliance.Binding == nil || custody.Appliance.Binding.Invocation != invocation || custody.Appliance.Binding.CredentialRoot != "/fixture/private-root" || custody.Appliance.Binding.ChildID != "child-1" || custody.Appliance.Binding.Boundary != strings.Repeat("d", 64) || custody.Workspace != "" {
		t.Fatal("broker acquisition metadata changed", custody, err)
	}
	mode.Store(1)
	state, err = observer.InspectRun(ctx, address)
	if err != nil || state.Retirement != execution.Exited || state.ExitCode == nil || *state.ExitCode != 0 || state.Appliance.CredentialReturned {
		t.Fatal("exit became retirement or credential-return proof", state, err)
	}
	mode.Store(2)
	state, err = observer.RetireRun(ctx, address)
	if err != nil || state.Retirement != execution.Retired || state.Appliance.CredentialReturned {
		t.Fatal("retirement invented credential return", state, err)
	}
	if err := observer.SealCustody(ctx, address); err != nil {
		t.Fatal(err)
	}
	custody, err = observer.ReadCustody(ctx, address)
	if err != nil || !custody.Sealed || custody.Appliance.State != execution.CustodyUncertain || custody.Appliance.Binding == nil {
		t.Fatal("sealing a bound execution invented credential return", custody, err)
	}
	mode.Store(3)
	state, err = observer.RetireRun(ctx, address)
	if err != nil || state.Retirement != execution.Retired || state.Appliance.Invocation.Invocation != "" || state.Appliance.Invocation.UID != 0 || state.Appliance.LeaseID != "" || state.Appliance.CredentialReturned {
		t.Fatal("pre-launch stop invented process or credential facts", state, err)
	}
	custody, err = observer.ReadCustody(ctx, address)
	if err != nil || !custody.Sealed || custody.Appliance.State != execution.Released || custody.Workspace != "" || custody.Appliance.Binding != nil || custody.Appliance.LeaseID != "" {
		t.Fatal("unbound seal gained invented provenance", custody, err)
	}
	for _, invalid := range []int64{4, 5, 6} {
		mode.Store(invalid)
		_, err := observer.InspectRun(ctx, address)
		if err == nil || errors.Is(err, execution.ErrRunNotObserved) != (invalid == 5) {
			t.Fatal("foreign/unavailable run became an absent observation", invalid, err)
		}
	}
	mode.Store(7)
	custody, err = observer.ReadCustody(ctx, address)
	if err != nil || custody.Appliance.State != execution.CustodyUncertain || custody.Sealed {
		t.Fatal("pending acquisition became release evidence", custody, err)
	}
	for _, invalid := range []int64{8, 9, 10} {
		mode.Store(invalid)
		_, err := observer.ReadCustody(ctx, address)
		if err == nil || errors.Is(err, execution.ErrCustodyNotObserved) != (invalid == 10) {
			t.Fatal("foreign/unknown custody became absence", invalid, err)
		}
	}
	mode.Store(11)
	state, err = observer.RetireRun(ctx, address)
	if err != nil || state.Retirement != execution.Uncertain {
		t.Fatal("unknown retirement proof became confirmation", state, err)
	}
	before := calls.Load()
	if _, err := observer.InspectRun(ctx, execution.RunAddress{Run: "invalid", Workspace: address.Workspace}); err == nil || calls.Load() != before {
		t.Fatal("invalid native address reached the runtime", err)
	}
	request := execution.CandidateRequest{Address: address, Role: project.RoleCoder, Preparation: "f" + strings.Repeat("1", 24), Candidate: strings.Repeat("a", 40)}
	bundle, err := observer.ExportCandidate(ctx, request)
	if err != nil || bundle.Address != address || bundle.Candidate != request.Candidate || string(bundle.Bundle) != "fixture bundle bytes" {
		t.Fatal("candidate wire decoding changed scope or bytes", bundle, err)
	}
	mode.Store(12)
	if _, err := observer.ExportCandidate(ctx, request); !errors.Is(err, execution.ErrCandidateBounds) {
		t.Fatal("malformed candidate bytes were accepted", err)
	}
	mode.Store(13)
	if _, err := observer.ExportCandidate(ctx, request); !errors.Is(err, execution.ErrObservation) {
		t.Fatal("another candidate was attributed to the requested commit", err)
	}
	// The actual broker requires a configured provider adapter and private
	// tmpfs enrollment root even for metadata. A private mount/user namespace
	// supplies that root without any enrollment, helper or model invocation.
	dsnPath, keyPath := filepath.Join(directory, "broker.dsn"), filepath.Join(directory, "broker.key")
	if err := os.WriteFile(dsnPath, []byte(brokerDSN), 0o600); err != nil {
		t.Fatal(err)
	}
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keyPath, []byte(base64.StdEncoding.EncodeToString(key)), 0o600); err != nil {
		t.Fatal(err)
	}
	adminPath, runtimePath := filepath.Join(directory, "broker.sock"), filepath.Join(directory, "runtime.sock")
	enrollmentRoot := filepath.Join(directory, "enrollment")
	if err := os.Mkdir(enrollmentRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	config, err := json.Marshal(map[string]any{"database_dsn_file": dsnPath, "key_file": keyPath, "admin_socket": adminPath, "runtime_socket": runtimePath, "host_socket": filepath.Join(directory, "unused-host.sock"), "muse": map[string]any{"store": true, "root": enrollmentRoot}})
	if err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(directory, "broker.json")
	if err := os.WriteFile(configPath, config, 0o600); err != nil {
		t.Fatal(err)
	}
	start := func() (*FactoryObserver, func()) {
		command := exec.Command("unshare", "--user", "--map-current-user", "--mount", "--keep-caps", "sh", "-c",
			`mount -n -t tmpfs -o "mode=0700,size=1m,uid=$4,gid=$5" tmpfs "$1" && exec "$2" --config "$3"`,
			"broker-fixture", enrollmentRoot, binary, configPath, fmt.Sprint(os.Getuid()), fmt.Sprint(os.Getgid()))
		var diagnostics bytes.Buffer
		command.Stderr = &diagnostics
		if err := command.Start(); err != nil {
			t.Fatal("start actual identity broker", err)
		}
		done := make(chan error, 1)
		go func() { done <- command.Wait() }()
		stopped := false
		stop := func() {
			if stopped {
				return
			}
			stopped = true
			_ = command.Process.Signal(syscall.SIGTERM)
			select {
			case err := <-done:
				if err != nil {
					t.Error("identity broker did not stop cleanly", err)
				}
			case <-time.After(5 * time.Second):
				_ = command.Process.Kill()
				<-done
				t.Error("identity broker required forced shutdown")
			}
			for _, path := range []string{adminPath, runtimePath} {
				if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
					t.Error("remove owned broker socket after process exit", err)
				}
			}
		}
		t.Cleanup(stop)
		client := identityclient.New(adminPath)
		t.Cleanup(client.HTTP.CloseIdleConnections)
		native := NewFactoryObserver(nil, client)
		deadline := time.Now().Add(15 * time.Second)
		for time.Now().Before(deadline) {
			bounded, cancel := context.WithTimeout(ctx, 250*time.Millisecond)
			_, err := native.ReadCustody(bounded, address)
			cancel()
			if err == nil || errors.Is(err, execution.ErrCustodyNotObserved) {
				return native, stop
			}
			select {
			case err := <-done:
				stopped = true
				message := strings.ReplaceAll(diagnostics.String(), brokerDSN, "[private DSN]")
				message = strings.ReplaceAll(message, base64.StdEncoding.EncodeToString(key), "[private key]")
				t.Fatal("identity broker exited during startup", err, message)
			default:
			}
			time.Sleep(20 * time.Millisecond)
		}
		t.Fatal("actual identity broker did not become ready")
		return nil, stop
	}
	native, stopBroker := start()
	if _, err := native.ReadCustody(ctx, address); !errors.Is(err, execution.ErrCustodyNotObserved) {
		t.Fatal("a new private broker database did not observe absence", err)
	}
	if err := native.SealCustody(ctx, address); err != nil {
		t.Fatal("actual broker seal", err)
	}
	custody, err = native.ReadCustody(ctx, address)
	if err != nil || !custody.Sealed || custody.Appliance == nil || custody.Appliance.State != execution.Released || custody.Appliance.Binding != nil || custody.Workspace != "" {
		t.Fatal("actual pre-acquisition seal was not retained", custody, err)
	}
	stopBroker()
	native, stopBroker = start()
	custody, err = native.ReadCustody(ctx, address)
	if err != nil || !custody.Sealed || custody.Appliance.State != execution.Released || custody.Run != address.Run || custody.Appliance.Binding != nil || custody.Appliance.LeaseID != "" || custody.Workspace != "" {
		t.Fatal("broker restart lost or changed its PostgreSQL tombstone", custody, err)
	}
	stopBroker()
	if _, err := native.ReadCustody(ctx, address); err == nil || errors.Is(err, execution.ErrCustodyNotObserved) {
		t.Fatal("stopped broker was reported as absent custody", err)
	}
}

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
