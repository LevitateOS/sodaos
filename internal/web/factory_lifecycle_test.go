package web

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/levitateos/factory/pkg/factory/execution"

	extensions "forgejo.org/extension-sdk"
	"github.com/levitateos/factory/pkg/collaboration"
	"github.com/levitateos/factory/pkg/factory"
	"github.com/levitateos/factory/pkg/project"
	"github.com/levitateos/factory/pkg/store"
	"github.com/levitateos/sodaos/internal/config"
)

func lifecycleWebFixture(t *testing.T) *Server {
	t.Helper()
	s := apiTestServer(t)
	// Baseline lifecycle journeys retain equal native/canonical repository
	// keys; canonical_execution_actor exercises distinct scoped identities.
	for native := int64(1); native <= 8; native++ {
		if _, err := s.API.FountainIdentities.LocalRepository(t.Context(), native); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.Store.CreateProject(t.Context(), store.Project{ID: webTerminalProject, Name: "factory", RepositoryID: 7, OwnerID: 1, Repository: "alice/factory"}); err != nil {
		t.Fatal(err)
	}
	if err := s.Store.MarkReady(t.Context(), webTerminalProject, "10.89.0.2"); err != nil {
		t.Fatal(err)
	}
	s.Host.HTTP = &http.Client{Transport: roundTrip(func(r *http.Request) (*http.Response, error) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		var response string
		switch r.URL.Path {
		case "/factory-stop", "/factory-inspect":
			response = `{"id":"` + body["id"].(string) + `","project":"` + body["project"].(string) + `","phase":"stopped","retirement":"confirmed","reason":"stopped"}`
		case "/factory-takeover":
			id := body["id"].(string)
			response = `{"id":"` + id + `","project":"` + body["project"].(string) + `","member":"` + body["member"].(string) +
				`","destination":"/home/` + body["member"].(string) + `/factory-takeover/` + id + `"}`
		case "/prepare-hold":
			hold, _ := body["hold"].(bool)
			response = `{"active":` + strconv.FormatBool(hold) + `,"revision":1}`
		case "/lifecycle":
			action, _ := body["action"].(string)
			running := strconv.FormatBool(action != "stop")
			response = `{"environment":{"id":"` + webTerminalProject + `","running":` + running + `},"boot_enabled":` + running + `}`
		default:
			return &http.Response{StatusCode: 500, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("unexpected helper call"))}, nil
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(response))}, nil
	})}
	return s
}

func lifecycleWebRun(t *testing.T, s *Server) factory.Run {
	t.Helper()
	now := time.Now()
	r := factory.Run{ID: factory.NewID(), ProjectID: webTerminalProject, Role: "coder", InputSHA: strings.Repeat("a", 40), Started: now, Deadline: now.Add(time.Hour), Image: "sha256:" + strings.Repeat("b", 64), Harness: "codex-0.157.1", Model: "test"}
	if err := s.Store.RecordFactoryRun(t.Context(), r); err != nil {
		t.Fatal(err)
	}
	return r
}

func lifecycleWebGrants(t *testing.T, s *Server, repository int64, paused bool) {
	t.Helper()
	ctx := context.Background()
	policy := factory.RepositoryPolicy{
		Repository: repository, GrantedBy: 1, Enabled: true, Paused: paused,
		TargetBranch: "refs/heads/main",
		Roles: map[string]factory.RoleSelection{
			project.RoleCoder: {Harness: project.FactoryHarnessMuse, HarnessVers: "0.157.1", Model: "test", Account: "conn-1"}, project.RoleReviewer: {Harness: project.FactoryHarnessMuse, HarnessVers: "0.157.1", Model: "test", Account: "conn-1"},
		},
		Checks: []string{"native-ci/build"}, MergeMethod: factory.MergeFastForward,
		Publish:             factory.ActorBindingRef{TokenID: 11, ActorID: 12, Kind: factory.OpRefPublish},
		Create:              factory.ActorBindingRef{TokenID: 11, ActorID: 12, Kind: factory.OpPRCreate},
		Review:              factory.ActorBindingRef{TokenID: 13, ActorID: 14, Kind: factory.OpReviewSubmit},
		Merge:               factory.ActorBindingRef{TokenID: 15, ActorID: 16, Kind: factory.OpMerge},
		AttemptLimits:       factory.DefaultAttemptLimits(),
		ReassessmentMinutes: factory.DefaultReassessmentMinutes,
	}
	if err := s.Store.SaveRepositoryPolicy(ctx, policy); err != nil {
		t.Fatal(err)
	}
	if err := s.Store.SaveOperatorGrant(ctx, factory.OperatorGrant{Repository: repository, GrantedBy: 1, Active: true}); err != nil {
		t.Fatal(err)
	}
	if err := s.Store.SaveCapacity(ctx, factory.Capacity{UpdatedBy: 1, MaxConcurrentRuns: 2}); err != nil {
		t.Fatal(err)
	}
	if err := s.Store.SaveSponsorship(ctx, factory.Sponsorship{Repository: repository, GrantedBy: 1, ActorID: 1, ProjectID: webTerminalProject, Connection: "conn-1", GrantID: "grant-1", Generation: 1, Roles: []string{project.RoleCoder}, AllowanceMinutes: 60, Active: true}); err != nil {
		t.Fatal(err)
	}
	profile := &project.Profile{ID: project.RockyHeadless, Distribution: "rocky", Version: "9.6", Interface: "headless", Architecture: "amd64", Image: "sha256:" + strings.Repeat("b", 64), Revision: strings.Repeat("c", 40)}
	if err := s.Store.SaveEnvironmentGrant(ctx, project.EnvironmentGrant{Repository: repository, Owner: 1, Profile: profile, Active: true}); err != nil {
		t.Fatal(err)
	}
}

func lifecycleWebPreparations(t *testing.T, s *Server) {
	t.Helper()
	ctx := context.Background()
	for i, role := range []string{project.RoleCoder, project.RoleReviewer} {
		stored, _, err := s.Store.AdmitPreparation(ctx, execution.PreparationRecord{Preparation: project.Preparation{
			ID: "f0123456789abcdef0123456" + string(rune('0'+i)), Project: webTerminalProject, Role: role,
			Requirements: project.RequirementAcceptance{ID: "d0123456789abcdef01234567", Revision: 1, Approver: 1, SourceCommit: strings.Repeat("a", 40), Digest: strings.Repeat("b", 64)},
			Approval:     project.AdminApproval{ID: "d123456789abcdef012345678", Revision: 1, Approver: 1, EffectsDigest: strings.Repeat("b", 64)},
			SourceCommit: strings.Repeat("a", 40), SetupDigest: strings.Repeat("b", 64), Tools: []string{"python3"},
		}})
		if err != nil {
			t.Fatal(err)
		}
		stored.State = execution.PreparationState{
			ID: stored.Preparation.ID, Project: stored.Preparation.Project, Role: role,
			Phase: project.PrepareReady, Ready: true, Container: strings.Repeat("c", 64),
			SourceCommit: stored.Preparation.SourceCommit, SetupDigest: stored.Preparation.SetupDigest,
		}
		if err = s.Store.ObservePreparation(ctx, stored); err != nil {
			t.Fatal(err)
		}
	}
}

func lifecycleAction(t *testing.T, s *Server, method, path, body, login string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	nativeAPIServe(t, s, w, apiTestRequest(method, path, body, login))
	return w
}

func TestFactoryPauseAndResumeJourney(t *testing.T) {
	t.Run("canonical_execution_actor", checkMappedApplianceLaunch)
	s := lifecycleWebFixture(t)
	lifecycleWebGrants(t, s, 8, false)
	w := lifecycleAction(t, s, "POST", "/api/repositories/8/factory/actions", `{"action":"pause"}`, "alice")
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	var receipt factory.PauseResult
	decodeBody(t, w, &receipt)
	if !receipt.Paused || receipt.Withdrawal.Cause != factory.CauseControlPaused || len(receipt.Runs) != 0 {
		t.Fatalf("pause: %+v", receipt)
	}
	assertNoSecrets(t, w.Body.String())
	replay := lifecycleAction(t, s, "POST", "/api/repositories/8/factory/actions", `{"action":"pause"}`, "alice")
	if replay.Code != 200 {
		t.Fatal(replay.Code, replay.Body.String())
	}
	grant, err := s.Store.OperatorGrant(t.Context(), 8)
	if err != nil {
		t.Fatal(err)
	}
	grant.Active = false
	if err := s.Store.SaveOperatorGrant(t.Context(), grant); err != nil {
		t.Fatal(err)
	}
	refused := lifecycleAction(t, s, "POST", "/api/repositories/8/factory/actions", `{"action":"resume"}`, "alice")
	if refused.Code != 409 || !strings.Contains(refused.Body.String(), `"authority_ineffective"`) {
		t.Fatal("resume bypassed withdrawn authority", refused.Code, refused.Body.String())
	}
	if open, _, _, err := s.Store.DispatchState(t.Context(), 8); err != nil || open {
		t.Fatal("refused resume opened dispatch", open, err)
	}
	grant, err = s.Store.OperatorGrant(t.Context(), 8)
	if err != nil {
		t.Fatal(err)
	}
	grant.Active = true
	if err := s.Store.SaveOperatorGrant(t.Context(), grant); err != nil {
		t.Fatal(err)
	}
	resume := lifecycleAction(t, s, "POST", "/api/repositories/8/factory/actions", `{"action":"resume"}`, "alice")
	if resume.Code != 200 {
		t.Fatal(resume.Code, resume.Body.String())
	}
	var reopened factory.ResumeResult
	decodeBody(t, resume, &reopened)
	if !reopened.Reopened || !reopened.Effective.Effective {
		t.Fatalf("resume: %+v", reopened)
	}
	var repeated factory.PauseResult
	decodeBody(t, replay, &repeated)
	if repeated.Withdrawal.Revision != receipt.Withdrawal.Revision {
		t.Fatal("repeated pause changed its gate revision")
	}
	again := lifecycleAction(t, s, "POST", "/api/repositories/8/factory/actions", `{"action":"resume"}`, "alice")
	if again.Code != 200 {
		t.Fatal(again.Code, again.Body.String())
	}
}

// Real PostgreSQL owns the adopted namespace; an actual private Unix HTTP peer
// captures the production host wire. Ungranted launches must send no native
// request; historical observation and retirement remain available. The peer
// supplies declared metadata, not provider execution or credential acquisition.
func checkMappedApplianceLaunch(t *testing.T) {
	ctx := t.Context()
	admin, ok := store.TestFixtureDSN()
	if !ok {
		t.Skip("SODA_PG_* fixture unavailable")
	}
	db, dsn, cleanup, err := store.OpenEphemeral(ctx, admin)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	raw, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = raw.Close() })
	directory, err := os.MkdirTemp("", "appliance-actor-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(directory) })
	socket := filepath.Join(directory, "host.sock")
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	if err := os.Chmod(socket, 0o600); err != nil {
		t.Fatal(err)
	}
	s, err := NewFactory(config.Config{ForgejoURL: "https://forgejo.example.test", HostSocket: socket}, db)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := raw.ExecContext(ctx, `SELECT setval(pg_get_serial_sequence('collaboration_actors','id'),9007199254740991,true)`); err != nil {
		t.Fatal(err)
	}
	foreignHost := collaboration.Host{ID: collaboration.HostID(collaboration.NewID()), Provider: collaboration.GitHub, APIBase: "https://api.github-fixture.test"}
	if _, err := db.PutCollaborationHost(ctx, foreignHost); err != nil {
		t.Fatal(err)
	}
	foreign, err := db.ObserveCollaborationActor(ctx, foreignHost.ID, "7", "same-name")
	if err != nil {
		t.Fatal(err)
	}
	actor, err := s.API.FountainIdentities.LocalActor(ctx, "7")
	if err != nil || actor <= 9007199254740991 || actor == foreign.ID {
		t.Fatal("fixture conflated canonical/native execution actors", actor, foreign, err)
	}
	if err := db.UpsertUser(ctx, store.User{ID: 7, Login: "same-name", Name: "same-name"}); err != nil {
		t.Fatal(err)
	}
	if err := db.CreateProject(ctx, store.Project{ID: webTerminalProject, Name: "Factory", RepositoryID: 7, OwnerID: 7, Repository: "same-name/factory"}); err != nil {
		t.Fatal(err)
	}
	prompt := []byte("captured appliance actor fixture")
	request := execution.LaunchRequest{
		Kind: collaboration.Appliance,
		Input: project.FactoryLaunch{
			Run: project.FactoryRun{
				ID: factory.NewID(), Project: webTerminalProject, Role: project.RoleCoder,
				Preparation: "f" + strings.Repeat("a", 24), Harness: project.FactoryHarnessMuse,
				HarnessVers: "fixture-1", Model: "fixture-model", Assignment: project.FactoryPromptDigest(prompt),
				SourceCommit: strings.Repeat("b", 40), Connection: "fixture-connection", Actor: int64(actor),
				Deadline: time.Now().Add(time.Minute), ExecutionPhase: project.ExecutionInvestigate,
			},
			Prompt: prompt, HarnessSHA256: strings.Repeat("c", 64),
		},
		Appliance: &execution.ApplianceLaunch{
			Profile: project.Profile{ID: project.RockyHeadless, Distribution: "rocky", Version: "10.2", Interface: "headless", Architecture: "amd64", Image: "sha256:" + strings.Repeat("d", 64), Revision: strings.Repeat("e", 40)},
			Tool:    execution.ApplianceTool{Harness: project.FactoryHarnessMuse, Version: "fixture-1", SHA256: strings.Repeat("c", 64), Image: "sha256:" + strings.Repeat("d", 64)},
		},
	}
	request.Input.MCP = &project.FactoryMCPBinding{Capability: request.Input.Run.ID, Secret: strings.Repeat("f", 64)}
	if err := request.Validate(); err != nil {
		t.Fatal("invalid captured fixture", err)
	}
	captured, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	var calls, mode atomic.Int64
	peer := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/factory-launch" {
			t.Error("ungranted launch reached native owner")
			w.WriteHeader(403)
			return
		} else if r.URL.Path == "/factory-inspect" || r.URL.Path == "/factory-stop" {
			var address project.FactoryInspect
			if err := json.NewDecoder(r.Body).Decode(&address); err != nil || address.ID != request.Input.Run.ID || address.Project != webTerminalProject {
				t.Error("composition changed the native run address", err)
				w.WriteHeader(400)
				return
			}
		} else {
			t.Error("unexpected host path", r.URL.Path)
			w.WriteHeader(404)
			return
		}
		if mode.Load() == 2 {
			w.WriteHeader(503)
			return
		}
		state := project.FactoryState{ID: request.Input.Run.ID, Project: webTerminalProject, Role: project.RoleCoder, Phase: project.FactoryRunning, Live: true, Container: strings.Repeat("d", 64), Unit: "fixture-launch.service", Invocation: strings.Repeat("a", 32), Generation: 3, UID: 1001, GID: 1002}
		if mode.Load() == 1 {
			state.Role = project.RoleReviewer
		}
		if r.URL.Path == "/factory-stop" {
			state.Phase, state.Live, state.Retirement = "stopped", false, "confirmed"
		}
		_ = json.NewEncoder(w).Encode(state)
	})}
	done := make(chan error, 1)
	go func() { done <- peer.Serve(listener) }()
	t.Cleanup(func() {
		_ = peer.Close()
		if err := <-done; !errors.Is(err, http.ErrServerClosed) {
			t.Error("private host shutdown", err)
		}
	})
	observed, err := s.Coordinator.Host.LaunchRun(ctx, request)
	if !errors.Is(err, execution.ErrLaunch) || calls.Load() != 0 {
		t.Fatalf("ungranted launch was not refused before native I/O: %+v %v calls=%d", observed, err, calls.Load())
	}
	for _, wrong := range []int64{7, int64(foreign.ID), int64(actor) + 99, 0} {
		invalid := request
		invalid.Input.Run.Actor = wrong
		if _, err := s.Coordinator.Host.LaunchRun(ctx, invalid); err == nil || calls.Load() != 0 {
			t.Fatal("unmapped/foreign/invalid actor reached native credentials", wrong, err, calls.Load())
		}
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := s.Coordinator.Host.LaunchRun(cancelled, request); !errors.Is(err, execution.ErrLaunch) || calls.Load() != 0 {
		t.Fatal("cancelled launch reached native credentials", err, calls.Load())
	}
	for _, refusal := range []int64{2} {
		mode.Store(refusal)
		if _, err := s.Coordinator.Host.InspectRun(ctx, request.Address()); err == nil {
			t.Fatal("conflicting/unavailable historical observation became successful", refusal)
		}
	}
	mode.Store(0)
	if _, err := s.Coordinator.Host.InspectRun(ctx, request.Address()); err != nil {
		t.Fatal("native inspection delegation failed", err)
	}
	if stopped, err := s.Coordinator.Host.RetireRun(ctx, request.Address()); err != nil || stopped.Live || stopped.Retirement != execution.Retired {
		t.Fatal("native retirement delegation failed", stopped, err)
	}
	after, err := json.Marshal(request)
	if err != nil || !bytes.Equal(after, captured) {
		t.Fatal("native adapter rewrote the captured canonical launch", err)
	}
	var count, lastID int64
	if err := raw.QueryRowContext(ctx, `SELECT count(*) FROM collaboration_actors`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if err := raw.QueryRowContext(ctx, `SELECT last_value FROM collaboration_actors_id_seq`).Scan(&lastID); err != nil {
		t.Fatal(err)
	}
	if count != 2 || lastID != int64(actor) {
		t.Fatal("runtime translation allocated identity aliases", count, lastID)
	}
	nativeProject, err := db.Project(ctx, webTerminalProject)
	if err != nil || nativeProject.OwnerID != 7 || nativeProject.RepositoryID != 7 {
		t.Fatal("runtime translation rewrote native Project ownership", nativeProject, err)
	}
}

func TestFactoryActionsRequireCurrentCodeWrite(t *testing.T) {
	s := lifecycleWebFixture(t)
	readOnly := func(in extensions.CallbackRequest) extensions.CallbackResponse {
		return extensions.CallbackResponse{Repository: &extensions.Repository{ID: in.RepositoryID, Owner: "alice", Name: "repo", Permission: "read"}}
	}
	w := httptest.NewRecorder()
	nativeAPIServeWithCallback(t, s, w, apiTestRequest("POST", "/api/repositories/7/factory/actions", `{"action":"pause"}`, "bob"), readOnly)
	if w.Code != 403 {
		t.Fatal(w.Code, w.Body.String())
	}
	r := lifecycleWebRun(t, s)
	stop := httptest.NewRecorder()
	nativeAPIServeWithCallback(t, s, stop, apiTestRequest("POST", "/api/factory/runs/"+r.ID+"/actions", `{"command_id":"`+factory.NewID()+`","action":"stop"}`, "bob"), readOnly)
	if stop.Code != 403 {
		t.Fatal(stop.Code, stop.Body.String())
	}
}

func TestFactoryUncertainStopAndUnownedRunRetryRefusal(t *testing.T) {
	s := lifecycleWebFixture(t)
	lifecycleWebGrants(t, s, 7, false)
	lifecycleWebPreparations(t, s)
	r := lifecycleWebRun(t, s)
	early := lifecycleAction(t, s, "POST", "/api/factory/runs/"+r.ID+"/actions", `{"command_id":"`+factory.NewID()+`","action":"retry"}`, "alice")
	if early.Code != 409 {
		t.Fatal(early.Code, early.Body.String())
	}
	stop := lifecycleAction(t, s, "POST", "/api/factory/runs/"+r.ID+"/actions", `{"command_id":"`+factory.NewID()+`","action":"stop"}`, "alice")
	if stop.Code != 200 {
		t.Fatal(stop.Code, stop.Body.String())
	}
	var stopped struct {
		Confirmed bool `json:"confirmed"`
		Uncertain bool `json:"uncertain"`
	}
	decodeBody(t, stop, &stopped)
	// The test broker socket is dead, so the stop fences instead of settling.
	if stopped.Confirmed || !stopped.Uncertain {
		t.Fatalf("stop: %+v", stopped)
	}
	resume := lifecycleAction(t, s, "POST", "/api/repositories/7/factory/actions", `{"action":"resume"}`, "alice")
	if resume.Code != 409 || !strings.Contains(resume.Body.String(), `"pending_runs"`) {
		t.Fatal("resume bypassed unsettled native work", resume.Code, resume.Body.String())
	}
	// Even a settled standalone run has no accepted assignment/root authority.
	// This is a refusal subject, not a substitute for successful native Stop.
	r.Outcome, r.Summary, r.Reconciled = factory.Cancelled, "stopped", true
	if err := s.Store.SaveFactoryRun(t.Context(), r); err != nil {
		t.Fatal(err)
	}
	command := factory.NewID()
	retry := lifecycleAction(t, s, "POST", "/api/factory/runs/"+r.ID+"/actions", `{"command_id":"`+command+`","action":"retry"}`, "alice")
	if retry.Code != 409 || !strings.Contains(retry.Body.String(), `"retry_ineligible"`) {
		t.Fatal(retry.Code, retry.Body.String())
	}
	if _, err := s.Store.FactoryCommand(t.Context(), command); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("unowned run recorded a retry command: %v", err)
	}
	assertNoSecrets(t, retry.Body.String())
	bad := lifecycleAction(t, s, "POST", "/api/factory/runs/"+r.ID+"/actions", `{"command_id":"`+factory.NewID()+`","action":"launch"}`, "alice")
	if bad.Code != 400 {
		t.Fatal(bad.Code, bad.Body.String())
	}
	missing := lifecycleAction(t, s, "POST", "/api/factory/runs/"+factory.NewID()+"/actions", `{"command_id":"`+factory.NewID()+`","action":"stop"}`, "alice")
	if missing.Code != 404 {
		t.Fatal(missing.Code, missing.Body.String())
	}
}

func TestFactoryTakeoverRequiresMembershipAndSettledRun(t *testing.T) {
	s := lifecycleWebFixture(t)
	r := lifecycleWebRun(t, s)
	if err := s.Store.Join(t.Context(), webTerminalProject, 1, "alice"); err != nil {
		t.Fatal(err)
	}
	pending := lifecycleAction(t, s, "POST", "/api/factory/runs/"+r.ID+"/actions", `{"action":"takeover"}`, "alice")
	if pending.Code != 409 {
		t.Fatal(pending.Code, pending.Body.String())
	}
	r.Outcome, r.Summary, r.Reconciled = factory.Cancelled, "stopped", true
	if err := s.Store.SaveFactoryRun(t.Context(), r); err != nil {
		t.Fatal(err)
	}
	outsider := lifecycleAction(t, s, "POST", "/api/factory/runs/"+r.ID+"/actions", `{"action":"takeover"}`, "bob")
	if outsider.Code != 403 {
		t.Fatal(outsider.Code, outsider.Body.String())
	}
	takeover := lifecycleAction(t, s, "POST", "/api/factory/runs/"+r.ID+"/actions", `{"action":"takeover"}`, "alice")
	if takeover.Code != 200 {
		t.Fatal(takeover.Code, takeover.Body.String())
	}
	var record factory.TakeoverRecord
	decodeBody(t, takeover, &record)
	if record.Run != r.ID || record.Member != "alice" || record.Dest != project.TakeoverDestination("alice", r.ID) {
		t.Fatalf("takeover: %+v", record)
	}
	assertNoSecrets(t, takeover.Body.String())
	repeated := lifecycleAction(t, s, "POST", "/api/factory/runs/"+r.ID+"/actions", `{"action":"takeover"}`, "alice")
	if repeated.Code != 200 {
		t.Fatal(repeated.Code, repeated.Body.String())
	}
	var again factory.TakeoverRecord
	decodeBody(t, repeated, &again)
	if again != record {
		t.Fatalf("takeover post result changed: %+v", again)
	}
}

func TestLifecycleStopAndStartCoordinate(t *testing.T) {
	s := lifecycleWebFixture(t)
	r := lifecycleWebRun(t, s)
	stop := lifecycleAction(t, s, "POST", "/api/environments/"+webTerminalProject+"/lifecycle", `{"action":"stop","confirm_stop":true}`, "alice")
	if stop.Code != 200 {
		t.Fatal(stop.Code, stop.Body.String())
	}
	var stopped struct {
		Environment struct {
			Running bool `json:"running"`
		} `json:"environment"`
		BootEnabled bool `json:"boot_enabled"`
		Control     struct {
			Withdrawal struct {
				Cause string `json:"cause"`
			} `json:"withdrawal"`
			Runs       []factory.RunStopOutcome `json:"runs"`
			Hold       bool                     `json:"hold"`
			HoldSynced bool                     `json:"hold_synced"`
		} `json:"control"`
	}
	decodeBody(t, stop, &stopped)
	if stopped.Environment.Running || stopped.BootEnabled {
		t.Fatalf("stop: %+v", stopped)
	}
	if stopped.Control.Withdrawal.Cause != factory.CauseProjectStop || len(stopped.Control.Runs) != 1 || !stopped.Control.Runs[0].Uncertain {
		t.Fatalf("control: %+v", stopped.Control)
	}
	if !stopped.Control.Hold || !stopped.Control.HoldSynced {
		t.Fatalf("hold: %+v", stopped.Control)
	}
	open, _, _, err := s.Store.DispatchState(t.Context(), 7)
	if err != nil || open {
		t.Fatal("project stop left dispatch open")
	}
	held, err := s.Store.MaintenanceHold(t.Context(), webTerminalProject)
	if err != nil || !held.Hold {
		t.Fatal("project stop left preparation admitted")
	}
	start := lifecycleAction(t, s, "POST", "/api/environments/"+webTerminalProject+"/lifecycle", `{"action":"start"}`, "alice")
	if start.Code != 200 {
		t.Fatal(start.Code, start.Body.String())
	}
	var started struct {
		Verification factory.StartVerification `json:"verification"`
	}
	decodeBody(t, start, &started)
	if !started.Verification.Started || len(started.Verification.Revived) != 0 {
		t.Fatalf("verification: %+v", started.Verification)
	}
	if len(started.Verification.Unverified) != 1 || started.Verification.Unverified[0] != r.ID {
		t.Fatalf("unverified: %+v", started.Verification)
	}
	if !started.Verification.Hold {
		t.Fatalf("hold released by start: %+v", started.Verification)
	}
	open, _, withdrawal, err := s.Store.DispatchState(t.Context(), 7)
	if err != nil || open || len(withdrawal.ActiveCauses) != 1 || withdrawal.ActiveCauses[0] != factory.CauseProjectStop {
		t.Fatalf("uncertain start cleared Project stop: %v %+v %v", open, withdrawal, err)
	}
}

func TestLifecycleStartClearsProjectStopAfterQuiescence(t *testing.T) {
	s := lifecycleWebFixture(t)
	if _, err := s.Store.WithdrawDispatch(t.Context(), 7, factory.CauseProjectStop, "native:1"); err != nil {
		t.Fatal(err)
	}
	started := lifecycleAction(t, s, "POST", "/api/environments/"+webTerminalProject+"/lifecycle", `{"action":"start"}`, "alice")
	if started.Code != 200 {
		t.Fatal(started.Code, started.Body.String())
	}
	var view struct {
		Verification factory.StartVerification `json:"verification"`
	}
	decodeBody(t, started, &view)
	if !view.Verification.DispatchOpen || !view.Verification.Started {
		t.Fatalf("start did not report reopened dispatch: %+v", view)
	}
	open, _, withdrawal, err := s.Store.DispatchState(t.Context(), 7)
	if err != nil || !open || len(withdrawal.ActiveCauses) != 0 || withdrawal.Cause != factory.CauseProjectStop {
		t.Fatalf("Project start did not clear only active cause: %v %+v %v", open, withdrawal, err)
	}
}

func TestSpacesShowsFactoryControl(t *testing.T) {
	s, _, _ := spacesFixture(t, true)
	lifecycleWebGrants(t, s, 7, true)
	if _, err := s.Store.WithdrawDispatch(t.Context(), 7, factory.CauseControlPaused, "native:1"); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	if err := s.Store.RecordFactoryRun(t.Context(), factory.Run{ID: factory.NewID(), ProjectID: webTerminalProject, Role: "coder", InputSHA: strings.Repeat("a", 40), Started: now, Deadline: now.Add(time.Hour), Image: "sha256:" + strings.Repeat("b", 64), Harness: "codex-0.157.1", Model: "test"}); err != nil {
		t.Fatal(err)
	}
	view := readSpaces(t, s, spacesCallback(200, new(atomic.Int32)))
	if len(view.Items) != 1 {
		t.Fatalf("items: %d", len(view.Items))
	}
	control := view.Items[0].FactoryControl
	if control == nil || control.DispatchOpen || control.WithdrawalCause != factory.CauseControlPaused || !control.Paused || control.UnsettledRuns != 1 {
		t.Fatalf("control: %+v", control)
	}
	var raw bytes.Buffer
	_ = json.NewEncoder(&raw).Encode(view)
	assertNoSecrets(t, raw.String())
}
