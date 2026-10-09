package web

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	extensions "forgejo.org/extension-sdk"
	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
)

func lifecycleWebFixture(t *testing.T) *Server {
	t.Helper()
	s := apiTestServer(t)
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
			project.RoleCoder: {Harness: project.FactoryHarnessCodex, HarnessVers: "0.157.1", Model: "test"}, project.RoleReviewer: {Harness: project.FactoryHarnessCodex, HarnessVers: "0.157.1", Model: "test"},
		},
		Checks: []string{"native-ci/build"}, MergeMethod: factory.MergeFastForward,
		Publish:       factory.ActorBindingRef{TokenID: 11, ActorID: 12, Kind: factory.OpRefPublish},
		Create:        factory.ActorBindingRef{TokenID: 11, ActorID: 12, Kind: factory.OpPRCreate},
		Review:        factory.ActorBindingRef{TokenID: 13, ActorID: 14, Kind: factory.OpReviewSubmit},
		Merge:         factory.ActorBindingRef{TokenID: 15, ActorID: 16, Kind: factory.OpMerge},
		AttemptLimits: factory.DefaultAttemptLimits(),
		MaxConcurrent: 2,
	}
	if err := s.Store.SaveRepositoryPolicy(ctx, policy); err != nil {
		t.Fatal(err)
	}
	if err := s.Store.SaveOperatorGrant(ctx, factory.OperatorGrant{Repository: repository, GrantedBy: 1, Active: true, MaxConcurrent: 2}); err != nil {
		t.Fatal(err)
	}
	if err := s.Store.SaveCapacity(ctx, factory.Capacity{UpdatedBy: 1, MaxConcurrentRuns: 2, MaxQueued: 4}); err != nil {
		t.Fatal(err)
	}
	if err := s.Store.SaveSponsorship(ctx, factory.Sponsorship{Repository: repository, GrantedBy: 1, ActorID: 1, ProjectID: webTerminalProject, Connection: "conn-1", GrantID: "grant-1", Generation: 1, Roles: []string{project.RoleCoder}, AllowanceMinutes: 60, MaxConcurrent: 1, Active: true}); err != nil {
		t.Fatal(err)
	}
	if err := s.Store.SaveConnectionUsageBudget(ctx, factory.ConnectionUsageBudget{Connection: "conn-1", RollingMinutes: factory.DefaultConnectionUsageBudgetMinutes}); err != nil {
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
		stored, _, err := s.Store.AdmitPreparation(ctx, project.StoredPreparation{Preparation: project.Preparation{
			ID: "f0123456789abcdef0123456" + string(rune('0'+i)), Project: webTerminalProject, Role: role,
			Requirements: project.RequirementAcceptance{ID: "d0123456789abcdef01234567", Revision: 1, Approver: 1, SourceCommit: strings.Repeat("a", 40), Digest: strings.Repeat("b", 64)},
			Approval:     project.AdminApproval{ID: "d123456789abcdef012345678", Revision: 1, Approver: 1, EffectsDigest: strings.Repeat("b", 64)},
			SourceCommit: strings.Repeat("a", 40), SetupDigest: strings.Repeat("b", 64), Tools: []string{"python3"},
		}})
		if err != nil {
			t.Fatal(err)
		}
		stored.State.Ready = true
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
	s := lifecycleWebFixture(t)
	lifecycleWebGrants(t, s, 8, false)
	pause := factory.NewID()
	w := lifecycleAction(t, s, "POST", "/api/repositories/8/factory/actions", `{"command_id":"`+pause+`","action":"pause"}`, "alice")
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	var receipt factory.PauseReceipt
	decodeBody(t, w, &receipt)
	if !receipt.Paused || receipt.Withdrawal.Cause != factory.CauseControlPaused || len(receipt.Runs) != 0 {
		t.Fatalf("pause: %+v", receipt)
	}
	assertNoSecrets(t, w.Body.String())
	replay := lifecycleAction(t, s, "POST", "/api/repositories/8/factory/actions", `{"command_id":"`+pause+`","action":"pause"}`, "alice")
	if replay.Code != 200 {
		t.Fatal(replay.Code, replay.Body.String())
	}
	resume := lifecycleAction(t, s, "POST", "/api/repositories/8/factory/actions", `{"command_id":"`+factory.NewID()+`","action":"resume"}`, "alice")
	if resume.Code != 200 {
		t.Fatal(resume.Code, resume.Body.String())
	}
	var reopened factory.ResumeReceipt
	decodeBody(t, resume, &reopened)
	if !reopened.Reopened || !reopened.Effective.Effective {
		t.Fatalf("resume: %+v", reopened)
	}
	read := lifecycleAction(t, s, "GET", "/api/factory/commands/"+pause, "", "alice")
	if read.Code != 200 {
		t.Fatal(read.Code, read.Body.String())
	}
	var cmd factory.Command
	decodeBody(t, read, &cmd)
	if cmd.ID != pause || cmd.Finished == "" {
		t.Fatalf("command read: %+v", cmd)
	}
	assertNoSecrets(t, read.Body.String())
}

func TestFactoryActionsRequireCurrentCodeWrite(t *testing.T) {
	s := lifecycleWebFixture(t)
	readOnly := func(in extensions.CallbackRequest) extensions.CallbackResponse {
		return extensions.CallbackResponse{Repository: &extensions.Repository{ID: in.RepositoryID, Owner: "alice", Name: "repo", Permission: "read"}}
	}
	w := httptest.NewRecorder()
	nativeAPIServeWithCallback(t, s, w, apiTestRequest("POST", "/api/repositories/7/factory/actions", `{"command_id":"`+factory.NewID()+`","action":"pause"}`, "bob"), readOnly)
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

func TestFactoryRunStopAndRetry(t *testing.T) {
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
	r.Outcome, r.Summary, r.Reconciled = factory.Cancelled, "stopped", true
	if err := s.Store.SaveFactoryRun(t.Context(), r); err != nil {
		t.Fatal(err)
	}
	retry := lifecycleAction(t, s, "POST", "/api/factory/runs/"+r.ID+"/actions", `{"command_id":"`+factory.NewID()+`","action":"retry"}`, "alice")
	if retry.Code != 200 {
		t.Fatal(retry.Code, retry.Body.String())
	}
	var decision factory.RetryDecision
	decodeBody(t, retry, &decision)
	if !decision.Queued || decision.Prior != r.ID {
		t.Fatalf("retry: %+v", decision)
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
	pending := lifecycleAction(t, s, "POST", "/api/factory/runs/"+r.ID+"/actions", `{"command_id":"`+factory.NewID()+`","action":"takeover"}`, "alice")
	if pending.Code != 409 {
		t.Fatal(pending.Code, pending.Body.String())
	}
	r.Outcome, r.Summary, r.Reconciled = factory.Cancelled, "stopped", true
	if err := s.Store.SaveFactoryRun(t.Context(), r); err != nil {
		t.Fatal(err)
	}
	outsider := lifecycleAction(t, s, "POST", "/api/factory/runs/"+r.ID+"/actions", `{"command_id":"`+factory.NewID()+`","action":"takeover"}`, "bob")
	if outsider.Code != 403 {
		t.Fatal(outsider.Code, outsider.Body.String())
	}
	takeover := lifecycleAction(t, s, "POST", "/api/factory/runs/"+r.ID+"/actions", `{"command_id":"`+factory.NewID()+`","action":"takeover"}`, "alice")
	if takeover.Code != 200 {
		t.Fatal(takeover.Code, takeover.Body.String())
	}
	var record factory.TakeoverRecord
	decodeBody(t, takeover, &record)
	if record.Run != r.ID || record.Member != "alice" || record.Dest != project.TakeoverDestination("alice", r.ID) {
		t.Fatalf("takeover: %+v", record)
	}
	assertNoSecrets(t, takeover.Body.String())
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
