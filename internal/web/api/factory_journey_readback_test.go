package api

// Proof boundary (lane-api-readback/70): these tests prove the
// REGISTERED HTTP read path over real persisted records seeded through
// public store writers and the real tables. The production-entrypoint
// partial producer->HTTP case (dispatch/admission entrypoints driving
// a live attempt chain into this read) is NOT included: its fixtures
// (mergeSeed, driveReassessmentBranch, happyMerger, twoPageEnumerator)
// live in control package _test files, which api tests cannot import.
// The missing dependency is the actual F1/H2/H4 production joints
// plus producer->HTTP/WorkSpaces/native proof evidence — explicitly
// NOT an exported production test hook (rejected per 041; no
// manufactured authority enters production seams to satisfy tests).
// Until those joints exist the full-journey gate stays OPEN; nothing
// here injects an accepted merge, check verdict or synthesized graph.
//
// Amendment (lane-api-phases/72, SUPERVISOR-041): the phase section
// proves the 040 correction — one CURRENT state per canonical phase
// for the selected owner AttemptRoot with exact contributing
// assignment/run/root refs in stable canonical order. The old
// tail-eight prefix rendering is gone: overflow, sibling roots and
// invalid phases are covered below, and F1/H2/H4 producer joints stay
// missing/named rather than closed by test hooks.

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	extensions "forgejo.org/extension-sdk"
	"github.com/levitateos/factory/pkg/collaboration"
	"github.com/levitateos/factory/pkg/factory"
	"github.com/levitateos/factory/pkg/factory/control"
	"github.com/levitateos/factory/pkg/factory/execution"
	factoryhttpapi "github.com/levitateos/factory/pkg/factory/httpapi"
	factoryservice "github.com/levitateos/factory/pkg/factory/service"
	"github.com/levitateos/factory/pkg/project"
	"github.com/levitateos/factory/pkg/store"
	"github.com/levitateos/sodaos/internal/config"
	"github.com/levitateos/sodaos/internal/forgejo"
	"github.com/levitateos/sodaos/internal/host"
)

// TestQualificationJourneyLostHTTPResponse connects the maintained qualification
// command client to real registered HTTP handlers and PostgreSQL. Only native
// callback authentication uses journeyMux's existing local test authority. The
// first mutation's socket closes AFTER its durable normal-API effect; no product
// command outcome or native retirement is synthesized by this check.
// Pause uses the real HTTP route and PostgreSQL gate; repeated requests
// return current state without creating or looking up command receipts.
func TestFactoryPauseUsesCurrentStateHTTP(t *testing.T) {
	fx, _ := journeyMux(t, nil)
	server := httptest.NewServer(fx.mux)
	defer server.Close()
	authority, err := json.Marshal(extensions.Authority{
		ExtensionID: "soda", InstanceID: "journey-http-test", SessionGeneration: "generation-1",
		Contribution: extensions.Contribution{ID: "spaces", Kind: "page", Scope: "global", Action: "post"},
		Actor:        extensions.Actor{ID: "1", Username: "alice"},
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	for i := 0; i < 2; i++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, server.URL+"/api/repositories/7/factory/actions", strings.NewReader(`{"action":"pause"}`))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set(extensions.ContextHeader, string(authority))
		req.Header.Set(extensions.AdmissionHeader, "test-admission")
		req.Header.Set(extensions.SessionGenerationHeader, "generation-1")
		req.Header.Set("Origin", "https://soda.test")
		req.Header.Set("Content-Type", "application/json")
		response, err := server.Client().Do(req)
		if err != nil {
			t.Fatal(err)
		}
		var result factory.PauseResult
		err = json.NewDecoder(response.Body).Decode(&result)
		response.Body.Close()
		if err != nil || response.StatusCode != 200 || !result.Paused || result.Withdrawal.Revision != 1 || result.Withdrawal.ClosedBy != "native:1" {
			t.Fatalf("pause %d: status=%d result=%+v err=%v", i, response.StatusCode, result, err)
		}
	}
	open, revision, _, err := fx.db.DispatchState(ctx, journeyRepo)
	if err != nil || open || revision != 1 {
		t.Fatalf("dispatch gate: open=%t revision=%d err=%v", open, revision, err)
	}
	var commands, posts int
	if err = fx.raw.QueryRowContext(ctx, `SELECT (SELECT count(*) FROM factory_commands),(SELECT count(*) FROM factory_feed_posts WHERE repository=7 AND data->>'status'='dispatch:paused')`).Scan(&commands, &posts); err != nil || commands != 0 || posts != 1 {
		t.Fatalf("pause storage: commands=%d posts=%d err=%v", commands, posts, err)
	}
}

const (
	journeyRepo  = 7
	journeyIssue = 3
)

// journeyAcceptanceSource is a deterministic test-only AcceptanceSource:
// it brackets the admitted decision's exact digests as visible native
// evidence, so AcceptanceStatus assesses without a native provider. It
// spends nothing and records nothing.
type journeyAcceptanceSource struct {
	decision factory.Acceptance
}

func (f *journeyAcceptanceSource) ReadAcceptanceEvidence(_ context.Context, _, _ string, _ []string) (control.AcceptanceEvidence, error) {
	return control.AcceptanceEvidence{
		Issue: control.AcceptanceIssueView{
			Index: f.decision.IssueIndex, TitleDigest: f.decision.TitleDigest,
			TitleOccurrence: f.decision.TitleOccurrence, ContentDigest: f.decision.ContentDigest,
			ContentVer: f.decision.ContentVersion, Visible: true,
		},
		Revision: f.decision.NativeRev,
	}, nil
}

type journeyHTTPFixture struct {
	api        *API
	db         *store.Store
	raw        *sql.DB
	mux        *http.ServeMux
	permission *atomic.Value
	dsn        string
}

// journeyMux builds the real registered product mux on an ephemeral PG
// store with a fake native callback authority and a deterministic
// acceptance-evidence source. It mirrors the existing q05 auth fixture.
func journeyMux(t *testing.T, fake *journeyAcceptanceSource) (*journeyHTTPFixture, func(method, path, body string) (int, []byte)) {
	t.Helper()
	suffix := "journey-" + strconv.Itoa(os.Getpid())
	permission := &atomic.Value{}
	permission.Store("admin")
	callbackPath := "@soda-journey-" + suffix
	callbackListener, err := net.Listen("unix", callbackPath)
	if err != nil {
		t.Fatal(err)
	}
	callback := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var in extensions.CallbackRequest
		if r.URL.Path != extensions.NativeCallbackPath || r.Header.Get(extensions.AdmissionHeader) != "test-admission" || json.NewDecoder(r.Body).Decode(&in) != nil {
			http.Error(w, "denied", http.StatusForbidden)
			return
		}
		switch in.Operation {
		case extensions.OperationCurrentActor:
			_ = json.NewEncoder(w).Encode(extensions.CallbackResponse{Actor: &extensions.Actor{ID: "1", Username: "alice"}})
		case extensions.OperationRepository:
			if in.RepositoryID != "7" {
				http.Error(w, "not visible", http.StatusNotFound)
				return
			}
			_ = json.NewEncoder(w).Encode(extensions.CallbackResponse{Repository: &extensions.Repository{ID: "7", Owner: "alice", Name: "Alpha", Permission: permission.Load().(string)}})
		default:
			http.Error(w, "denied", http.StatusForbidden)
		}
	}))
	callback.Listener.Close()
	callback.Listener = callbackListener
	callback.Start()
	t.Cleanup(callback.Close)
	t.Setenv(extensions.CallbackEnv, callbackPath)

	admin, ok := store.TestFixtureDSN()
	if !ok {
		t.Skip("SODA_PG_* fixture unavailable")
	}
	s, dsn, cleanup, err := store.OpenEphemeral(context.Background(), admin)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	raw, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = raw.Close() })
	product := &API{
		Config:      &config.Config{ForgejoURL: "https://soda.test"},
		Store:       s,
		Coordinator: &control.Coordinator{Store: s, AcceptanceReads: fake},
	}
	// This native peer declares the equal-number Fountain fixture namespace.
	// Adopt it before seeding records; production never infers this from IDs.
	host, err := s.BindConfiguredFountainLedger(t.Context(), product.Config.ForgejoURL)
	if err != nil {
		t.Fatal(err)
	}
	product.FountainIdentities, err = forgejo.NewFountainIdentities(t.Context(), s, host.ID)
	if err != nil {
		t.Fatal(err)
	}
	for native := int64(1); native <= 8; native++ {
		if _, err := product.FountainIdentities.LocalRepository(t.Context(), native); err != nil {
			t.Fatal(err)
		}
	}
	for native := int64(1); native <= 16; native++ {
		if _, err := product.FountainIdentities.LocalActor(t.Context(), strconv.FormatInt(native, 10)); err != nil {
			t.Fatal(err)
		}
	}
	mux := http.NewServeMux()
	product.registerExtensionProductRoutes(mux)

	call := func(method, path, body string) (int, []byte) {
		t.Helper()
		return journeyCall(t, mux, method, path, body, "generation-1", "generation-1", true)
	}
	return &journeyHTTPFixture{api: product, db: s, raw: raw, mux: mux, permission: permission, dsn: dsn}, call
}

func journeyCall(t *testing.T, mux http.Handler, method, path, body, authorityGen, sessionGen string, withAuthority bool) (int, []byte) {
	t.Helper()
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	if withAuthority {
		authority, err := json.Marshal(extensions.Authority{
			ExtensionID: "soda", InstanceID: "journey-test", SessionGeneration: authorityGen,
			Contribution: extensions.Contribution{ID: "spaces", Kind: "page", Scope: "global", Action: strings.ToLower(method)},
			Actor:        extensions.Actor{ID: "1", Username: "alice"},
		})
		if err != nil {
			t.Fatal(err)
		}
		r.Header.Set(extensions.ContextHeader, string(authority))
		r.Header.Set(extensions.AdmissionHeader, "test-admission")
		r.Header.Set(extensions.SessionGenerationHeader, sessionGen)
	}
	if method != http.MethodGet && method != http.MethodHead {
		r.Header.Set("Origin", "https://soda.test")
	}
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	return w.Code, w.Body.Bytes()
}

func journeyErrorCode(t *testing.T, body []byte) string {
	t.Helper()
	var decoded struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("error body: %s: %v", body, err)
	}
	return decoded.Error.Code
}

func journeyIssueView(t *testing.T, body []byte) factoryservice.IssueView {
	t.Helper()
	var view factoryservice.IssueView
	if err := json.Unmarshal(body, &view); err != nil {
		t.Fatalf("issue view: %s: %v", body, err)
	}
	if view.Journey == nil {
		t.Fatalf("issue view lacks its journey subsection: %s", body)
	}
	return view
}

// admitJourneyAcceptance records one objective-only acceptance through
// the real admission writer and returns its decision identity.
func admitJourneyAcceptance(t *testing.T, s *store.Store, n int, predecessor string) string {
	t.Helper()
	id := fmt.Sprintf("d%024x", n)
	decision := factory.Acceptance{
		ID: id, Predecessor: predecessor, Repository: journeyRepo, IssueIndex: strconv.Itoa(journeyIssue),
		Approver: 1, NativeRev: 1,
		TitleDigest: strings.Repeat("1", 64), TitleOccurrence: "1",
		ContentDigest: strings.Repeat("2", 64), ContentVersion: 0, Initial: true,
	}
	if err := s.AdmitAcceptanceDecision(context.Background(), decision); err != nil {
		t.Fatal(err)
	}
	return id
}

func journeyAcceptanceFor(id string) factory.Acceptance {
	return factory.Acceptance{
		ID: id, Repository: journeyRepo, IssueIndex: strconv.Itoa(journeyIssue),
		Approver: 1, NativeRev: 1,
		TitleDigest: strings.Repeat("1", 64), TitleOccurrence: "1",
		ContentDigest: strings.Repeat("2", 64), ContentVersion: 0, Initial: true,
	}
}

// insertJourneyRow stores one assignment with routing columns mirroring
// its JSON, matching production FinishAssignment.
func insertJourneyRow(t *testing.T, raw *sql.DB, a factory.Assignment, stage string) {
	t.Helper()
	data, err := json.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := raw.Exec(`INSERT INTO factory_assignments(id,repository,issue,run,stage,revision,data) VALUES($1,$2,$3,$4,$5,$6,$7)`,
		a.ID, a.Repository, a.Issue, a.Run, stage, 1, string(data)); err != nil {
		t.Fatal(err)
	}
}

func journeyOwner(t *testing.T, id, acceptance, phase, stage string, outcome factory.Outcome, reason string) factory.Assignment {
	t.Helper()
	return factory.Assignment{
		ID: id, AttemptRoot: id, PublicationAssignment: id,
		ProjectID: "p" + strings.Repeat("1", 24), Role: project.RoleCoder,
		Phase: phase, Stage: stage, Outcome: outcome, Reason: reason,
		Acceptance: acceptance, Preparation: "f" + strings.Repeat("b", 24),
		SourceCommit: strings.Repeat("c3", 20), Run: strings.Repeat("d4", 16),
		RunHistory: []string{strings.Repeat("d4", 16)},
		Repository: journeyRepo, Issue: journeyIssue, CreatedUnix: 1700000000,
		ActorID: 1,
	}
}

func journeyChild(t *testing.T, id, root, acceptance, phase, stage string, outcome factory.Outcome) factory.Assignment {
	t.Helper()
	child := journeyOwner(t, id, acceptance, phase, stage, outcome, "")
	child.AttemptRoot, child.ParentAssignment, child.PublicationAssignment = root, root, root
	child.Run, child.RunHistory = strings.Repeat("e5", 16), []string{strings.Repeat("e5", 16)}
	return child
}

var (
	journeyCandidate  = strings.Repeat("a1", 20)
	journeyBase       = strings.Repeat("b2", 20)
	journeyCorrection = strings.Repeat("e5", 20)
)

// journeyPublicationFixture builds one recorded publication: the exact
// candidate with its recorded creation base and PR linkage, one
// committed correction carrying its candidate, and one journaled
// finding. Correction effect and finding severity are parameters so
// malformed-row tests can poison exactly one of them.
func journeyPublicationFixture(root factory.Assignment, effect, severity string) factory.Publication {
	return factory.Publication{
		ID: strings.Repeat("9", 32), AssignmentID: root.ID,
		ProjectID: root.ProjectID, Role: project.RoleCoder, Acceptance: root.Acceptance,
		Preparation: root.Preparation, Run: root.Run,
		Candidate: journeyCandidate, BaseSHA: journeyBase, TargetBranch: "refs/heads/main",
		Tuple: &factory.CandidateTuple{Head: journeyCandidate, Base: journeyBase, DiffBase: journeyBase},
		Publish: factory.PublicationOperation{
			Kind: factory.OpRefPublish, OperationID: "soda-" + root.ID[:8] + "-publish-1",
			Effect: factory.OpEffectCommitted, Completion: factory.OpCompletionComplete,
		},
		PRCreate: factory.PublicationOperation{
			Kind: factory.OpPRCreate, OperationID: "soda-" + root.ID[:8] + "-prcreate-1",
			Effect: factory.OpEffectCommitted, Completion: factory.OpCompletionComplete,
			BaseOID: journeyBase,
		},
		Corrections: []factory.PublicationOperation{{
			Kind: factory.OpRefPublish, OperationID: "soda-" + root.ID[:8] + "-correction-1",
			Work:   &factory.PublicationIntent{OperationID: "soda-" + root.ID[:8] + "-correction-1", Candidate: journeyCorrection},
			Effect: effect, Completion: factory.OpCompletionComplete, Reason: "native_correction",
		}},
		CorrectionJournals: []factory.CorrectionJournal{{
			CycleID: strings.Repeat("7", 32), RunID: strings.Repeat("8", 32),
			Report: factory.CorrectionReport{NewCandidate: journeyCorrection, Findings: []factory.Finding{journeyFindingWithSeverity(severity)}},
		}},
		Repository: journeyRepo, Issue: journeyIssue, Stage: factory.PublicationOpen,
		PRNumber: 12, PRID: 34, CreatedUnix: 1700000000,
	}
}

func journeyFindingWithSeverity(severity string) factory.Finding {
	return factory.Finding{
		Key: strings.Repeat("f", 32), OriginRun: strings.Repeat("0", 32),
		Severity: severity, Defect: "candidate lacks a check",
		Evidence: "diff hunk without coverage", RequiredOutcome: "add the missing check",
		FixKind: factory.FindingFixCode, Actionable: true,
	}
}

func insertJourneyPublication(t *testing.T, raw *sql.DB, p factory.Publication) {
	t.Helper()
	data, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := raw.Exec(`INSERT INTO factory_publications(assignment,repository,issue,run,stage,revision,data) VALUES($1,$2,$3,$4,$5,$6,$7)`,
		p.AssignmentID, p.Repository, p.Issue, p.Run, p.Stage, p.Revision, string(data)); err != nil {
		t.Fatal(err)
	}
}

// seedJourneyPlan persists one admitted plan revision with receipts
// through the real planning writers: design complete, build running,
// docs pending.
func seedJourneyPlan(t *testing.T, s *store.Store, root string) {
	t.Helper()
	planID := strings.Repeat("1", 32)
	commit := strings.Repeat("a", 40)
	plan := factory.InvestigationPlan{
		ID: planID, AttemptRoot: root,
		ScopeDigest: strings.Repeat("3", 64), InputDigest: strings.Repeat("4", 64),
		SourceCommit: commit, IntegrationCommit: commit,
		Preparation:        "f" + strings.Repeat("d", 24),
		Approach:           "record the candidate behind checks",
		AcceptanceCoverage: []string{"accept-1"},
		Tasks: []factory.PlanTask{
			{ID: "design", Objective: "settle the approach", Outputs: []string{"note"}, AcceptanceRefs: []string{"accept-1"}, Lane: "main", CheckIDs: []string{"unit"}},
			{ID: "build", Objective: "implement the change", Outputs: []string{"diff"}, AcceptanceRefs: []string{"accept-1"}, Lane: "main", CheckIDs: []string{"unit"}},
			{ID: "docs", Objective: "record the contract", Outputs: []string{"delta"}, AcceptanceRefs: []string{"accept-1"}, Lane: "side", CheckIDs: []string{"unit"}},
		},
		Lanes:  []factory.PlanLane{{ID: "main", TaskIDs: []string{"design", "build"}}, {ID: "side", TaskIDs: []string{"docs"}}},
		Checks: []factory.CheckSpec{{ID: "unit", Argv: []string{"go", "test", "./..."}, Cwd: ".", TimeoutSeconds: 300, Scope: factory.CheckScopeTask, Required: true}},
	}
	digest, err := factory.PlanDigest(plan)
	if err != nil {
		t.Fatal(err)
	}
	plan.Digest = digest
	ctx := context.Background()
	if err := s.SaveInvestigationPlan(ctx, plan); err != nil {
		t.Fatal(err)
	}
	if err := s.AdvancePlanHead(ctx, root, "", planID); err != nil {
		t.Fatal(err)
	}
	advance := func(taskID, state string, revision int64, output bool, reason string) {
		receipt := store.PlanTaskReceipt{
			AttemptRoot: root, PlanID: planID, TaskID: taskID, Revision: revision,
			State: state, Checks: []string{"unit"},
		}
		if state != store.PlanTaskPending {
			receipt.Child = "child-" + taskID
		}
		if output {
			receipt.OutputDigest = strings.Repeat("5", 64)
		}
		if reason != "" {
			receipt.Reason = reason
		}
		if err := s.SavePlanTaskReceipt(ctx, receipt); err != nil {
			t.Fatalf("receipt %s/%s: %v", taskID, state, err)
		}
	}
	advance("design", store.PlanTaskPending, 0, false, "")
	advance("design", store.PlanTaskRunning, 0, false, "")
	advance("design", store.PlanTaskOutputReady, 1, true, "")
	advance("design", store.PlanTaskCompleted, 2, true, "implemented and verified")
	advance("build", store.PlanTaskPending, 0, false, "")
	advance("build", store.PlanTaskRunning, 0, false, "")
}

// seedJourneyCycle records one decided reassessment cycle with its
// funded decision receipt: the decision selects reassess at 0.87 under
// the test model while the cycle waits for its coder run.
func seedJourneyCycle(t *testing.T, fx *journeyHTTPFixture, acceptance, cycleID string, generation int64, waitReason string, answers ...factory.PriorityAnswer) {
	t.Helper()
	ctx := context.Background()
	answer := factory.PriorityAnswer{
		Selected: factory.ReassessmentOptionReassess, Confidence: 0.87,
		Probabilities: map[string]float64{factory.ReassessmentOptionReassess: 0.87, factory.ReassessmentOptionContinue: 0.13},
		Model:         "test-reassessment-model",
	}
	if len(answers) == 1 {
		answer = answers[0]
	}
	eventID := strings.Repeat("b", 32)
	completion := factory.CompletionEvent{
		ID: eventID, PublicationID: fmt.Sprintf("%032x", 7000+generation),
		Repository: journeyRepo, Issue: journeyIssue, Generation: generation,
		AcceptanceID: acceptance, MergedCommit: strings.Repeat("c", 40),
		ClosingOccurrence: fmt.Sprintf("journey-%d", generation),
	}
	if _, err := fx.raw.Exec(`INSERT INTO factory_completions(repository,generation,event,publication,occurrence,data) VALUES($1,$2,$3,$4,$5,$6)
		ON CONFLICT(repository,generation) DO NOTHING`,
		journeyRepo, generation, completion.ID,
		completion.PublicationID, completion.ClosingOccurrence, `{}`); err != nil {
		t.Fatal(err)
	}
	record := store.ReassessmentRecord{
		Cycle: factory.ReassessmentCycle{
			EventID: eventID, CycleID: cycleID,
			State: factory.ReassessmentAttention, Decision: factory.ReassessmentOptionReassess,
			Repository: journeyRepo, Generation: generation,
		},
		Completion: completion, Local: completionTestLocal(), Issue: journeyIssue,
		Evidence: []factory.CompletionIssueEvidence{}, EvidenceRevision: 41,
		InventoryDigest: factory.ReassessmentInventoryDigest(journeyRepo, generation, 41, [32]byte{}, nil),
		WaitReason:      waitReason,
	}
	record.EvidenceFingerprint = control.SealEvidenceFingerprint(record.InventoryDigest, eventID, cycleID)
	if !factory.ValidReassessmentDecision(answer.Selected, answer.Confidence) {
		record.Cycle.Decision, record.WaitReason = "", "decision_not_actionable"
	}
	// The production receipt keys the complete decision state, purpose,
	// rubric and options; the sealed evidence fingerprint is only one input.
	fingerprint, err := control.DecisionFingerprintFromRecord(record)
	if err != nil || fingerprint == record.EvidenceFingerprint {
		t.Fatal("decision receipt must have its own canonical fingerprint", err)
	}
	data, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fx.raw.Exec(`INSERT INTO factory_reassessments(cycle,repository,generation,event,state,data) VALUES($1,$2,$3,$4,$5,$6)`,
		cycleID, journeyRepo, generation, record.Cycle.EventID, record.Cycle.State, string(data)); err != nil {
		t.Fatal(err)
	}

	receipt := factory.PriorityReceipt{
		Snapshot:    fingerprint,
		Calls:       []factory.PriorityCall{{Answer: answer}},
		CreatedUnix: 1700000000,
	}
	if err := fx.db.RecordPriorityJudgment(ctx, receipt); err != nil {
		t.Fatal(err)
	}
}

func journeyReasons(view factoryservice.IssueView) map[string]bool {
	reasons := make(map[string]bool, len(view.Journey.Reasons))
	for _, reason := range view.Journey.Reasons {
		reasons[reason] = true
	}
	return reasons
}

func TestFactoryIssueJourneyRefusals(t *testing.T) {
	fx, call := journeyMux(t, &journeyAcceptanceSource{})
	path := "/api/repositories/7/factory/issues/3"
	mux := fx.mux

	if code, body := journeyCall(t, mux, http.MethodGet, path, "", "", "", false); code != http.StatusForbidden || journeyErrorCode(t, body) != "native_authority_unavailable" {
		t.Fatalf("unauthorized status=%d body=%s", code, body)
	}
	if code, body := journeyCall(t, mux, http.MethodGet, path, "", "generation-1", "generation-2", true); code != http.StatusConflict || journeyErrorCode(t, body) != "extension_session_changed" {
		t.Fatalf("stale session status=%d body=%s", code, body)
	}
	if code, body := call(http.MethodGet, "/api/repositories/8/factory/issues/3", ""); code != http.StatusNotFound {
		t.Fatalf("wrong repository status=%d body=%s", code, body)
	}
	if code, body := call(http.MethodGet, "/api/repositories/7/factory/issues/abc", ""); code != http.StatusNotFound {
		t.Fatalf("malformed issue status=%d body=%s", code, body)
	}
	if code, body := call(http.MethodGet, path+"?limit=1", ""); code != http.StatusBadRequest || journeyErrorCode(t, body) != "invalid_request" {
		t.Fatalf("query status=%d body=%s", code, body)
	}
	// Registered owner-route authentication and the canonical capability choice.
	// The completed decision below is an explicit persisted fixture; fresh
	// complete-Builder admission is exercised by the controller integration.
	ctx := context.Background()
	if err := fx.db.UpsertUser(ctx, store.User{ID: 1, Login: "alice"}); err != nil {
		t.Fatal(err)
	}
	capability := fakeCatalogCapability()
	capability.ID, capability.RunID, capability.AssignmentID = factory.NewID(), factory.NewID(), factory.NewID()
	capability.ID = capability.RunID
	capability.State = factory.MCPCapabilityStaged
	capability.ContextSelectionSnapshot = strings.Repeat("e", 64)
	if err := fx.db.CreateProject(ctx, store.Project{ID: capability.Project, Name: "Held context", RepositoryID: 7, OwnerID: 1, Repository: "alice/Alpha"}); err != nil {
		t.Fatal(err)
	}
	run := factory.Run{ID: capability.RunID, ProjectID: capability.Project, Role: project.RoleCoder, ContextPending: true, Deadline: time.Unix(capability.DeadlineUnix, 0)}
	raw, _ := json.Marshal(run)
	if _, err := fx.raw.Exec(`INSERT INTO factory_runs(id,active,settled,data) VALUES($1,true,false,$2)`, run.ID, string(raw)); err != nil {
		t.Fatal(err)
	}
	request := factory.ContextCurrentRequest{Fingerprint: capability.ContextSelectionSnapshot, ScopeDigest: capability.ScopeDigest(), RequiredDigest: strings.Repeat("f", 64)}
	resolution := factory.ContextCurrentResolution{Principal: "native:1", RunID: run.ID, AssignmentID: capability.AssignmentID, Fingerprint: request.Fingerprint, ScopeDigest: request.ScopeDigest, RequiredDigest: request.RequiredDigest, Bundle: factory.ContextBundleCurrent, RecordedUnix: time.Now().Unix()}
	capability.ContextCurrentResolution = &resolution
	if err := fx.db.StageMCPCapability(ctx, capability); err != nil {
		t.Fatal(err)
	}
	actionPath := "/api/factory/runs/" + run.ID + "/actions"
	body, _ := json.Marshal(factoryservice.RunCommand{Action: factory.ActionContextCurrent, Context: &request})
	if code, _ := journeyCall(t, mux, http.MethodPost, actionPath, string(body), "", "", false); code != 403 {
		t.Fatal("context owner action accepted missing native authority", code)
	}
	if code, _ := journeyCall(t, mux, http.MethodPost, actionPath, string(body), "generation-1", "generation-2", true); code != 409 {
		t.Fatal("context owner action accepted a stale session", code)
	}
	foreign := actorCall(t, fx.api, "2", "bob")
	if code, _ := foreign(http.MethodPost, actionPath, string(body)); code < 400 {
		t.Fatal("foreign actor recovered an owner context decision")
	}
	fx.permission.Store("read")
	if code, reply := call(http.MethodPost, actionPath, string(body)); code != 403 || journeyErrorCode(t, reply) != "repository_write_required" {
		t.Fatalf("context owner action accepted read-only repository authority: %d %s", code, reply)
	}
	fx.permission.Store("admin")
	unknown := strings.TrimSuffix(string(body), "}") + `,"unknown":true}`
	if code, _ := call(http.MethodPost, actionPath, unknown); code != 400 {
		t.Fatal("context owner action accepted unknown JSON fields", code)
	}
	unknownContext := strings.Replace(string(body), `"context":{`, `"context":{"unknown":true,`, 1)
	if code, _ := call(http.MethodPost, actionPath, unknownContext); code != 400 {
		t.Fatal("context owner action accepted unknown context fields", code)
	}
	for i := 0; i < 2; i++ {
		code, reply := call(http.MethodPost, actionPath, string(body))
		var observed factory.ContextCurrentResolution
		if code != 200 || json.Unmarshal(reply, &observed) != nil || observed != resolution {
			t.Fatalf("canonical context choice: %d %s", code, reply)
		}
	}
	changed := request
	changed.ScopeDigest = strings.Repeat("0", 64)
	conflict, _ := json.Marshal(factoryservice.RunCommand{Action: factory.ActionContextCurrent, Context: &changed})
	if code, reply := call(http.MethodPost, actionPath, string(conflict)); code != 409 || journeyErrorCode(t, reply) != "context_resolution_ineligible" {
		t.Fatalf("context owner changed scope: %d %s", code, reply)
	}
	var commands, choices int
	if err := fx.raw.QueryRow(`SELECT count(*) FROM factory_commands`).Scan(&commands); err != nil {
		t.Fatal(err)
	}
	if err := fx.raw.QueryRow(`SELECT count(*) FROM factory_priority_judgments`).Scan(&choices); err != nil || commands != 0 || choices != 0 {
		t.Fatal("HTTP context choices wrote a duplicate command or consulted again", commands, choices, err)
	}

	// The completion Retry has the same current native maintainer/session
	// boundary. Its positive HTTP fixture recovers a canonical atomic writer
	// result; fresh admission through the coordinator is covered by the real
	// PG settled-completion integration.
	priorCycle := factory.NewID()
	seedJourneyCycle(t, fx, "daaaaaaaaaaaaaaaaaaaaaaa3", priorCycle, 1, "decision_not_actionable")
	completion := factory.CompletionRetryRequest{EventID: strings.Repeat("b", 32), ExpectedPriorCycle: priorCycle}
	completionCommand := factory.Command{ID: factory.NewID(), Type: factory.CommandCompletionRetry, Principal: "native:1", Target: "repository/7/completion/" + completion.EventID}
	completionPayload, _ := json.Marshal(completion)
	completionCommand.Payload = string(completionPayload)
	completionCommand.Digest = factory.SettingsDigest(completionCommand.Type, completionCommand.Target, completionCommand.Payload)
	completionDecision, err := fx.db.RecordCompletionRetryCommand(ctx, completionCommand, 7, completion, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	completionPath := "/api/repositories/7/factory/actions"
	completionBody, _ := json.Marshal(factoryservice.RepositoryCommand{CommandID: completionCommand.ID, Action: factory.ActionCompletionRetry, Completion: &completion})
	if code, _ := journeyCall(t, mux, http.MethodPost, completionPath, string(completionBody), "", "", false); code != 403 {
		t.Fatal("completion retry accepted missing native authority", code)
	}
	if code, _ := journeyCall(t, mux, http.MethodPost, completionPath, string(completionBody), "generation-1", "generation-2", true); code != 409 {
		t.Fatal("completion retry accepted stale session", code)
	}
	fx.permission.Store("read")
	if code, reply := call(http.MethodPost, completionPath, string(completionBody)); code != 403 || journeyErrorCode(t, reply) != "repository_write_required" {
		t.Fatalf("completion retry accepted read-only authority: %d %s", code, reply)
	}
	fx.permission.Store("admin")
	if code, _ := foreign(http.MethodPost, completionPath, string(completionBody)); code < 400 {
		t.Fatal("foreign actor recovered completion retry")
	}
	completionUnknown := strings.Replace(string(completionBody), `"completion":{`, `"completion":{"continue":true,`, 1)
	if code, _ := call(http.MethodPost, completionPath, completionUnknown); code != 400 {
		t.Fatal("completion retry admitted an unknown/override field", code)
	}
	if code, _ := call(http.MethodPost, completionPath, `{"command_id":"`+factory.NewID()+`","action":"retry_completion"}`); code != 400 {
		t.Fatal("completion retry admitted missing exact scope", code)
	}
	for i := 0; i < 2; i++ {
		code, reply := call(http.MethodPost, completionPath, string(completionBody))
		var observed factory.CompletionRetryDecision
		if code != 200 || json.Unmarshal(reply, &observed) != nil || observed != completionDecision {
			t.Fatalf("completion retry immutable replay: %d %s", code, reply)
		}
	}
	changedCompletion := completion
	changedCompletion.EventID = factory.NewID()
	changedBody, _ := json.Marshal(factoryservice.RepositoryCommand{CommandID: completionCommand.ID, Action: factory.ActionCompletionRetry, Completion: &changedCompletion})
	if code, reply := call(http.MethodPost, completionPath, string(changedBody)); code != 409 || journeyErrorCode(t, reply) != "command_conflict" {
		t.Fatalf("completion retry changed scope did not conflict: %d %s", code, reply)
	}
	var cycles int
	if err := fx.raw.QueryRow(`SELECT count(*) FROM factory_reassessments`).Scan(&cycles); err != nil {
		t.Fatal(err)
	}
	if err := fx.raw.QueryRow(`SELECT count(*) FROM factory_commands`).Scan(&commands); err != nil {
		t.Fatal(err)
	}
	if err := fx.raw.QueryRow(`SELECT count(*) FROM factory_priority_judgments`).Scan(&choices); err != nil || cycles != 2 || commands != 1 || choices != 1 {
		t.Fatal("completion retry replay refilled cycles or judgments", cycles, commands, choices, err)
	}
	// Registered run reads distinguish an issue-zero verifier from a Product
	// Lead and never turn a live/unaccounted run into a Retry affordance.
	native := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.NotFound(w, r) }))
	defer native.Close()
	fx.api.Host = &host.Client{HTTP: &http.Client{Transport: applicationNativeTransport{native.URL}}}
	verifier := factory.Assignment{ID: factory.NewID(), Run: factory.NewID(), ProjectID: capability.Project, Repository: 7, Role: project.RoleReviewer, Phase: project.ExecutionVerify, Stage: factory.AssignmentAssigned, Subject: factory.ExecutionSubject{Kind: factory.SubjectDelivery, Delivery: &factory.DeliverySubject{DeliveryID: factory.NewID(), IntentID: "d" + strings.Repeat("3", 24), AcceptedRevision: "d" + strings.Repeat("4", 24)}}}
	verifier.AttemptRoot = verifier.ID
	verifierRun := factory.Run{ID: verifier.Run, ProjectID: capability.Project, Role: project.RoleReviewer, InputSHA: strings.Repeat("a", 40)}
	raw, _ = json.Marshal(verifierRun)
	if _, err := fx.raw.Exec(`INSERT INTO factory_runs(id,active,settled,data) VALUES($1,true,false,$2)`, verifierRun.ID, string(raw)); err != nil {
		t.Fatal(err)
	}
	raw, _ = json.Marshal(verifier)
	if _, err := fx.raw.Exec(`INSERT INTO factory_assignments(id,repository,issue,run,stage,revision,data) VALUES($1,7,0,$2,'assigned',0,$3)`, verifier.ID, verifier.Run, string(raw)); err != nil {
		t.Fatal(err)
	}
	if _, _, err := fx.db.RecordFactoryRunView(ctx, factory.RunView{RunID: verifier.Run, Repository: 7, Issue: 0, Attempt: verifier.ID}); err != nil {
		t.Fatal(err)
	}
	readPath := "/api/factory/runs/" + verifier.Run
	if code, reply := call(http.MethodGet, readPath, ""); code != 200 || !strings.Contains(string(reply), `"verification_retry":{"available":false}`) || !strings.Contains(string(reply), `"delivery_repair":{"available":false}`) {
		t.Fatalf("unsettled verifier Retry projection: %d %s", code, reply)
	}
	fx.permission.Store("read")
	if code, _ := call(http.MethodGet, readPath, ""); code != 403 {
		t.Fatal("read-only actor saw owner verification controls", code)
	}
	fx.permission.Store("admin")
	if code, _ := journeyCall(t, mux, http.MethodGet, readPath, "", "generation-1", "generation-2", true); code != 409 {
		t.Fatal("stale session read owner verification controls", code)
	}
	verifier.Phase, verifier.Role, verifier.Subject = project.ExecutionDecompose, project.RoleCoder, factory.ExecutionSubject{Kind: factory.SubjectIntent, Intent: &factory.IntentSubject{IntentID: "d" + strings.Repeat("3", 24), OwnerID: 1}}
	raw, _ = json.Marshal(verifier)
	if _, err := fx.raw.Exec(`UPDATE factory_assignments SET data=$1 WHERE id=$2`, string(raw), verifier.ID); err != nil {
		t.Fatal(err)
	}
	if code, reply := call(http.MethodGet, readPath, ""); code != 200 || strings.Contains(string(reply), "verification_retry") || strings.Contains(string(reply), "delivery_repair") {
		t.Fatalf("Product Lead acquired verification Retry: %d %s", code, reply)
	}
	// Delivery repair requires actual failed-verification evidence. A command
	// receipt cannot substitute for the unavailable native owner below.
	repairBody, _ := json.Marshal(factoryservice.RunCommand{Action: factory.ActionDeliveryRepair})
	if code, _ := journeyCall(t, mux, http.MethodPost, actionPath, string(repairBody), "", "", false); code != 403 {
		t.Fatal("delivery repair accepted missing native authority", code)
	}
	if code, _ := journeyCall(t, mux, http.MethodPost, actionPath, string(repairBody), "generation-1", "generation-2", true); code != 409 {
		t.Fatal("delivery repair accepted stale session", code)
	}
	if code, _ := foreign(http.MethodPost, actionPath, string(repairBody)); code < 400 {
		t.Fatal("foreign actor recovered delivery repair")
	}
	fx.permission.Store("read")
	if code, reply := call(http.MethodPost, actionPath, string(repairBody)); code != 403 || journeyErrorCode(t, reply) != "repository_write_required" {
		t.Fatalf("delivery repair accepted read-only authority: %d %s", code, reply)
	}
	fx.permission.Store("admin")
	repairWithContext, _ := json.Marshal(factoryservice.RunCommand{Action: factory.ActionDeliveryRepair, Context: &request})
	if code, _ := call(http.MethodPost, actionPath, string(repairWithContext)); code != 400 {
		t.Fatal("delivery repair accepted context override", code)
	}
	if code, _ := call(http.MethodPost, actionPath, strings.TrimSuffix(string(repairBody), "}")+`,"unknown":true}`); code != 400 {
		t.Fatal("delivery repair accepted unknown field", code)
	}
	for range 2 {
		if code, reply := call(http.MethodPost, actionPath, string(repairBody)); code < 400 {
			t.Fatalf("repair admitted without native failed-verification evidence: %d %s", code, reply)
		}
	}
	if err := fx.raw.QueryRow(`SELECT count(*) FROM factory_commands`).Scan(&commands); err != nil || commands != 1 {
		t.Fatal("refused repair created another command", commands, err)
	}
	var repairs, allowances, usage int
	if err := fx.raw.QueryRow(`SELECT (SELECT count(*) FROM factory_commands WHERE type=$1),(SELECT count(*) FROM factory_attempt_allowances),(SELECT count(*) FROM factory_usage)`, factory.CommandDeliveryRepair).Scan(&repairs, &allowances, &usage); err != nil || repairs != 0 || allowances != 0 || usage != 0 {
		t.Fatal("receipt recovery admitted or spent repair work", repairs, allowances, usage, err)
	}
	// Existing internal-journal snapshots exercise only the registered owner
	// read projection. Fresh cause admission, worker dispatch, judgment and
	// protected adoption remain the separate composed controller PG journey.
	intent, revision := seedJourneyAdoption(t, fx.db, "d"+strings.Repeat("0", 24))
	accepted, err := fx.db.IntentRevision(ctx, intent, 1)
	if err != nil {
		t.Fatal(err)
	}
	delivery := factory.DeliverySubject{DeliveryID: factory.NewID(), IntentID: intent, AcceptedRevision: revision, ScopeDigest: accepted.Digest(), SourceCommit: verifierRun.InputSHA, PreparationDigest: strings.Repeat("b", 64), VerificationManifestDigest: strings.Repeat("c", 64)}
	verifier.Phase, verifier.Role, verifier.Subject = project.ExecutionVerify, project.RoleReviewer, factory.ExecutionSubject{Kind: factory.SubjectDelivery, Delivery: &delivery}
	raw, _ = json.Marshal(verifier)
	if _, err = fx.raw.Exec(`UPDATE factory_assignments SET data=$1 WHERE id=$2`, string(raw), verifier.ID); err != nil {
		t.Fatal(err)
	}
	verifierRun.Reconciled, verifierRun.CredentialReturned, verifierRun.Outcome = true, true, factory.Succeeded
	raw, _ = json.Marshal(verifierRun)
	if _, err = fx.raw.Exec(`UPDATE factory_runs SET active=false,settled=true,data=$1 WHERE id=$2`, string(raw), verifierRun.ID); err != nil {
		t.Fatal(err)
	}
	report := factory.AppVerificationReport{DeliveryID: delivery.DeliveryID, InputDigest: strings.Repeat("d", 64), Outcome: factory.AppVerificationFailed, CheckReceipts: []factory.VerificationCheckReceipt{{CheckID: factory.ApprovedAppCheckID, Outcome: factory.CheckReceiptPassed, ArtifactDigest: strings.Repeat("e", 64), LogExcerpt: "approved check completed"}}, Failures: []factory.CheckFailure{{CheckRef: "priority-state", Detail: "Priority did not persist after the shared-state update."}}}
	cause := factory.DeliveryRepairCause{Repository: 7, ProjectID: verifier.ProjectID, FailedAssignmentID: verifier.ID, FailedVerificationRunID: verifier.Run, Delivery: delivery, Report: report, FailureDigest: factory.DeliveryFailureDigest(report)}
	record := store.DeliveryRepairRecord{Cause: cause, Subject: factory.DeliveryRepairSubject{DeliveryID: delivery.DeliveryID, FailedVerificationRunID: verifier.Run, FailureDigest: cause.FailureDigest, CycleID: factory.DeliveryRepairCycleID(verifier.Run)}, Cycle: factory.StallRepairCycle{ID: factory.DeliveryRepairCycleID(verifier.Run), Number: 1, State: factory.RepairCollecting}}
	journal := factory.Command{ID: factory.DeliveryRepairCauseID(verifier.Run), Type: factory.CommandDeliveryRepair, Target: "repository/7/delivery-repair/" + verifier.Run, Principal: "factory-delivery-repair", Payload: `{"failure_digest":"` + cause.FailureDigest + `"}`}
	journal.Digest = factory.SettingsDigest(journal.Type, journal.Target, journal.Payload)
	if _, err = fx.raw.Exec(`INSERT INTO factory_commands(id,type,target,principal,digest,payload,created) VALUES($1,$2,$3,$4,$5,$6,$7)`, journal.ID, journal.Type, journal.Target, journal.Principal, journal.Digest, journal.Payload, time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
		t.Fatal(err)
	}
	for _, state := range []string{factory.RepairCollecting, factory.RepairInvestigating, factory.RepairAttention, factory.RepairAdopted} {
		record.Cycle.State = state
		if state != factory.RepairCollecting {
			record.Subject.InventoryDigest = strings.Repeat("f", 64)
			record.Cycle.InventoryDigest, record.Cycle.NativeRevision = record.Subject.InventoryDigest, 1
			record.Cycle.Local, record.Cycle.Issues = completionTestLocal(), []factory.CompletionIssueEvidence{}
			record.Cycle.AssignmentID, record.Cycle.RunID = record.Cycle.ID, factory.RepairChargeID(journal.ID, factory.RepairChargeRun, 1)
		}
		if state == factory.RepairAttention || state == factory.RepairAdopted {
			record.Cycle.ProposalOption, record.Cycle.ProposalProbability, record.Cycle.ProposalConfidence = factory.ReassessmentOptionAccept, 0.87, 0.24
			record.Cycle.Reason = "proposal_needs_attention"
			if state == factory.RepairAdopted {
				record.Cycle.ProposalConfidence = 0.88
				record.Cycle.Reason = "repair_graph_adopted_awaiting_fresh_readiness"
				record.Cycle.AdoptionProgress = json.RawMessage(`{"private_packet":"not a public progress field"}`)
			}
		}
		if err := record.Validate(); err != nil {
			t.Fatal("invalid read projection fixture", state, err)
		}
		raw, _ = json.Marshal(record)
		if _, err = fx.raw.Exec(`UPDATE factory_commands SET outcome=$1 WHERE id=$2`, string(raw), journal.ID); err != nil {
			t.Fatal(err)
		}
		code, reply := call(http.MethodGet, readPath, "")
		var observed factoryservice.RunStatusView
		if code != 200 || json.Unmarshal(reply, &observed) != nil || observed.DeliveryRepair == nil || observed.DeliveryRepair.Available || observed.DeliveryRepair.Progress == nil {
			t.Fatalf("owner repair progress %s: %d %s", state, code, reply)
		}
		progress := observed.DeliveryRepair.Progress
		if progress.State != state || progress.CauseID != journal.ID || progress.CycleID != record.Cycle.ID || progress.FailedRun != verifier.Run || progress.SourceCommit != delivery.SourceCommit || progress.RunID != record.Cycle.RunID || progress.FailureCount != 1 || len(progress.Failures) != 1 || progress.Failures[0] != report.Failures[0] {
			t.Fatalf("repair progress lost canonical bindings: %+v", progress)
		}
		if state == factory.RepairAttention || state == factory.RepairAdopted {
			if progress.Judgment == nil || progress.Judgment.Option != record.Cycle.ProposalOption || progress.Judgment.Probability != 0.87 || progress.Judgment.Confidence != record.Cycle.ProposalConfidence || progress.Reason != record.Cycle.Reason {
				t.Fatalf("repair judgment/Attention was omitted or promoted: %+v", progress)
			}
		}
		if strings.Contains(string(reply), "private_packet") || strings.Contains(string(reply), `"local"`) || strings.Contains(string(reply), `"adoption_progress"`) || strings.Contains(string(reply), `"check_receipts"`) {
			t.Fatal("repair read exposed private journal inputs", string(reply))
		}
		var retained string
		if err = fx.raw.QueryRow(`SELECT outcome FROM factory_commands WHERE id=$1`, journal.ID).Scan(&retained); err != nil || retained != string(raw) {
			t.Fatal("repair progress GET mutated the journal", err)
		}
	}
	if code, reply := foreign(http.MethodGet, readPath, ""); code == 200 && strings.Contains(string(reply), `"progress"`) {
		t.Fatal("another actor saw owner-private repair progress")
	}
	fx.permission.Store("read")
	if code, _ := call(http.MethodGet, readPath, ""); code != 403 {
		t.Fatal("read-only actor saw repair progress", code)
	}
	fx.permission.Store("admin")
	if code, _ := journeyCall(t, mux, http.MethodGet, readPath, "", "generation-1", "generation-2", true); code != 409 {
		t.Fatal("stale session saw repair progress", code)
	}
	if err = fx.raw.QueryRow(`SELECT (SELECT count(*) FROM factory_commands),(SELECT count(*) FROM factory_attempt_allowances),(SELECT count(*) FROM factory_usage)`).Scan(&commands, &allowances, &usage); err != nil || commands != 2 || allowances != 0 || usage != 0 {
		t.Fatal("repair progress reads admitted or spent work", commands, allowances, usage, err)
	}
	// A settled run needs no command identity or separate outcome record to stop.
	// The native stop itself is exercised by the controller integration tests.
	if _, err := fx.raw.Exec(`UPDATE factory_runs SET active=false,settled=true,
		data=jsonb_set(jsonb_set(data,'{reconciled}','true'),'{outcome}','"cancelled"') WHERE id=$1`, run.ID); err != nil {
		t.Fatal(err)
	}
	fx.permission.Store("read")
	if code, _ := call(http.MethodPost, actionPath, `{"action":"stop"}`); code != 403 {
		t.Fatal("read-only actor stopped a run", code)
	}
	fx.permission.Store("admin")
	for range 2 {
		code, reply := call(http.MethodPost, actionPath, `{"action":"stop"}`)
		var result control.StopResult
		if code != 200 || json.Unmarshal(reply, &result) != nil || result.RunID != run.ID || !result.Confirmed || result.Uncertain || result.Outcome != string(factory.Cancelled) {
			t.Fatalf("direct stop result: %d %s", code, reply)
		}
	}
	if err := fx.raw.QueryRow(`SELECT count(*) FROM factory_commands`).Scan(&commands); err != nil || commands != 2 {
		t.Fatal("stop recorded a separate command receipt", commands, err)
	}
}

func TestFactoryIssueJourneyNoAttempt(t *testing.T) {
	fx, call := journeyMux(t, &journeyAcceptanceSource{})
	code, body := call(http.MethodGet, "/api/repositories/7/factory/issues/3", "")
	if code != http.StatusOK {
		t.Fatalf("no-attempt status=%d body=%s", code, body)
	}
	view := journeyIssueView(t, body)
	journey := view.Journey
	if journey.Available || journey.Complete {
		t.Fatalf("no-attempt flags: %+v", journey)
	}
	if len(journey.Reasons) != 1 || journey.Reasons[0] != "no_attempt" {
		t.Fatalf("no-attempt reasons: %+v", journey.Reasons)
	}
	if journey.Repository != "7" || journey.Issue != "3" || journey.ObservedUnix <= 0 {
		t.Fatalf("no-attempt identity: %+v", journey)
	}
	fx.permission.Store("read")
	code, body = call(http.MethodGet, "/api/repositories/7/factory/issues/3", "")
	if code != http.StatusOK {
		t.Fatalf("read-only work status=%d body=%s", code, body)
	}
	readOnly := journeyIssueView(t, body).Capabilities
	if !readOnly.Watch.Available || readOnly.Stop.Available || readOnly.Retry.Available || readOnly.Takeover.Available || readOnly.Pause.Available || readOnly.Resume.Available {
		t.Fatalf("read-only user cannot watch without gaining mutations: %+v", readOnly)
	}
	fx.permission.Store("admin")
	for name, section := range map[string]int{
		"phases": len(journey.Phases), "tasks": len(journey.Tasks), "commits": len(journey.Commits),
		"findings": len(journey.Findings), "corrections": len(journey.Corrections),
		"judgments": len(journey.Judgments), "gates": len(journey.Gates), "adoptions": len(journey.Adoptions),
	} {
		if section != 0 {
			t.Fatalf("no-attempt section %s rendered %d entries", name, section)
		}
	}
	if view.Attempt != nil || view.Publication != nil {
		t.Fatalf("no-attempt top-level sections present: %+v", body)
	}
	// A genuine serving dependency refusal is observable through the existing
	// registered owner status GET, without collecting any run/provider body.
	fx.api.Coordinator.Dispatch(context.Background())
	statusCode, statusBody := call(http.MethodGet, "/api/repositories/7/factory", "")
	var status factoryservice.RepositoryStatusView
	if statusCode != http.StatusOK || json.Unmarshal(statusBody, &status) != nil || status.DispatchObservation == nil || len(status.DispatchObservation.Errors) != 1 || status.DispatchObservation.Errors[0].Reason != control.DispatchErrStore {
		t.Fatalf("owner dispatch observation unavailable: %d %s", statusCode, statusBody)
	}
	if strings.Contains(string(statusBody), "detail") || strings.Contains(string(statusBody), "provider") {
		t.Fatal("status diagnostic disclosed raw report material")
	}
	if rows, err := fx.db.IssueControls(context.Background(), 7, store.MaxIssueControls); err != nil || len(rows) != 0 {
		t.Fatalf("diagnostic read changed queue: %+v %v", rows, err)
	}
}

// seedRichJourney records one full lineage: acceptance, owner attempt
// with a newer child, publication with correction and journal,
// admitted plan with receipts, and one decided reassessment cycle with
// its funded receipt.
func seedRichJourney(t *testing.T, fx *journeyHTTPFixture, fake *journeyAcceptanceSource, answers ...factory.PriorityAnswer) (string, factory.Assignment, factory.Assignment) {
	t.Helper()
	acceptance := admitJourneyAcceptance(t, fx.db, 1, "")
	fake.decision = journeyAcceptanceFor(acceptance)
	root := journeyOwner(t, strings.Repeat("7", 32), acceptance, project.ExecutionImplement, factory.AssignmentFinished, factory.Succeeded, "")
	child := journeyChild(t, strings.Repeat("8", 32), root.ID, acceptance, project.ExecutionReview, factory.AssignmentFinished, factory.Failed)
	insertJourneyRow(t, fx.raw, root, factory.AssignmentFinished)
	insertJourneyRow(t, fx.raw, child, factory.AssignmentFinished)
	insertJourneyPublication(t, fx.raw, journeyPublicationFixture(root, factory.OpEffectCommitted, factory.FindingSeverityMajor))
	seedJourneyPlan(t, fx.db, root.ID)
	seedJourneyCycle(t, fx, acceptance, strings.Repeat("a", 32), 1, "awaiting_coder", answers...)
	return acceptance, root, child
}

func TestFactoryIssueChecksUseRecordedPullRequest(t *testing.T) {
	ctx := t.Context()
	fake := &journeyAcceptanceSource{}
	fx, call := journeyMux(t, fake)
	_, root, _ := seedRichJourney(t, fx, fake)
	publication, err := fx.db.PublicationByAssignment(ctx, root.ID)
	if err != nil {
		t.Fatal(err)
	}
	if publication.PRNumber == journeyIssue {
		t.Fatal("fixture needs distinct issue/PR numbers")
	}
	for _, pr := range []int64{journeyIssue, publication.PRNumber} {
		assessment := factory.CheckAssessment{Repository: 7, PRNumber: pr, PRID: publication.PRID, HeadOID: publication.Candidate, BaseOID: publication.BaseSHA, Verdict: factory.CheckPass, Reason: factory.CheckReasonPass, Results: []factory.CheckResult{}}
		raw, _ := json.Marshal(assessment)
		if _, err := fx.raw.ExecContext(ctx, `INSERT INTO factory_check_assessments(repository,pr,revision,data) VALUES(7,$1,1,$2)`, pr, string(raw)); err != nil {
			t.Fatal(err)
		}
	}
	code, body := call(http.MethodGet, "/api/repositories/7/factory/issues/3", "")
	if code != 200 {
		t.Fatalf("issue checks read: %d %s", code, body)
	}
	view := journeyIssueView(t, body)
	if view.Checks == nil || view.Checks.PRNumber != strconv.FormatInt(publication.PRNumber, 10) {
		t.Fatalf("issue number selected unrelated PR checks: %+v", view.Checks)
	}
	if _, err := fx.raw.ExecContext(ctx, `UPDATE factory_check_assessments SET data=jsonb_set(data,'{head_oid}',to_jsonb($2::text)) WHERE repository=7 AND pr=$1`, publication.PRNumber, strings.Repeat("0", 40)); err != nil {
		t.Fatal(err)
	}
	code, body = call(http.MethodGet, "/api/repositories/7/factory/issues/3", "")
	if code != 200 || !journeyIssueView(t, body).Checks.Stale {
		t.Fatal("check outside the candidate history was not stale", code, string(body))
	}
	if _, err := fx.raw.ExecContext(ctx, `DELETE FROM factory_check_assessments WHERE repository=7 AND pr=$1`, publication.PRNumber); err != nil {
		t.Fatal(err)
	}
	code, body = call(http.MethodGet, "/api/repositories/7/factory/issues/3", "")
	if code != 200 || journeyIssueView(t, body).Checks != nil {
		t.Fatal("missing linked PR checks fell back to issue-number checks", code, string(body))
	}
}

func TestFactoryIssueJourneyRecorded(t *testing.T) {
	fake := &journeyAcceptanceSource{}
	fx, call := journeyMux(t, fake)
	acceptance, root, child := seedRichJourney(t, fx, fake, factory.PriorityAnswer{
		Selected: factory.ReassessmentOptionContinue, Confidence: 0.28, Model: "test-prior-low-confidence",
		Probabilities: map[string]float64{factory.ReassessmentOptionContinue: 0.9, factory.ReassessmentOptionReassess: 0.1},
	})

	code, body := call(http.MethodGet, "/api/repositories/7/factory/issues/3", "")
	if code != http.StatusOK {
		t.Fatalf("recorded status=%d body=%s", code, body)
	}
	view := journeyIssueView(t, body)
	journey := view.Journey
	if !journey.Available || journey.Complete {
		t.Fatalf("recorded flags: %+v", journey)
	}
	// Producer-absent sections keep the journey honestly incomplete.
	if len(journey.Reasons) != 2 || journey.Reasons[0] != "adoptions_unavailable" || journey.Reasons[1] != "delivery_unavailable" {
		t.Fatalf("recorded reasons: %+v", journey.Reasons)
	}
	lineage := journey.Lineage
	if lineage.Head != acceptance || lineage.Acceptance != acceptance || lineage.Stale {
		t.Fatalf("recorded head scope: %+v", lineage)
	}
	if lineage.AttemptRoot != root.ID || lineage.RootAssignment != root.ID || lineage.LatestAssignment != child.ID {
		t.Fatalf("recorded root scope: %+v", lineage)
	}
	if lineage.Run != root.Run || lineage.LatestRun != child.Run {
		t.Fatalf("recorded run refs: %+v", lineage)
	}
	if lineage.SourceCommit != root.SourceCommit || lineage.Candidate != journeyCandidate {
		t.Fatalf("recorded source/candidate: %+v", lineage)
	}
	if lineage.TupleHead != journeyCandidate || lineage.TupleBase != journeyBase || lineage.TupleDiffBase != journeyBase {
		t.Fatalf("recorded tuple: %+v", lineage)
	}
	if lineage.PublishOp == "" || lineage.PRCreateOp == "" || lineage.PRNumber != "12" || lineage.PRID != "34" {
		t.Fatalf("recorded operation refs: %+v", lineage)
	}
	if len(lineage.RunHistory) != 1 || lineage.RunHistory[0] != root.Run {
		t.Fatalf("recorded history: %+v", lineage)
	}
	if len(journey.Phases) != 2 || journey.Phases[0].Phase != project.ExecutionImplement || journey.Phases[0].State != "complete" ||
		journey.Phases[1].Phase != project.ExecutionReview || journey.Phases[1].State != "attention" {
		t.Fatalf("recorded phases: %+v", journey.Phases)
	}
	// Each current phase names its exact contributing assignment, run
	// and AttemptRoot: implement behind the owner, review behind the
	// newer child with its own run.
	if journey.Phases[0].Assignment != root.ID || journey.Phases[0].Run != root.Run || journey.Phases[0].AttemptRoot != root.ID {
		t.Fatalf("implement refs: %+v", journey.Phases[0])
	}
	if journey.Phases[1].Assignment != child.ID || journey.Phases[1].Run != child.Run || journey.Phases[1].AttemptRoot != root.ID {
		t.Fatalf("review refs: %+v", journey.Phases[1])
	}
	if len(journey.Tasks) != 3 || journey.Tasks[0].Key != "design" || journey.Tasks[0].State != "complete" ||
		journey.Tasks[1].Key != "build" || journey.Tasks[1].State != "active" ||
		journey.Tasks[2].Key != "docs" || journey.Tasks[2].State != "pending" {
		t.Fatalf("recorded tasks: %+v", journey.Tasks)
	}
	if len(journey.Commits) != 1 || journey.Commits[0].Commit != journeyCandidate || journey.Commits[0].Base != journeyBase ||
		journey.Commits[0].PRNumber != "12" || journey.Commits[0].PRID != "34" {
		t.Fatalf("recorded commits: %+v", journey.Commits)
	}
	if len(journey.Corrections) != 1 || journey.Corrections[0].Candidate != journeyCorrection ||
		journey.Corrections[0].Effect != factory.OpEffectCommitted || journey.Corrections[0].Completion != factory.OpCompletionComplete {
		t.Fatalf("recorded corrections: %+v", journey.Corrections)
	}
	if len(journey.Findings) != 1 || journey.Findings[0].Key != strings.Repeat("f", 32) {
		t.Fatalf("recorded findings: %+v", journey.Findings)
	}
	if len(journey.Judgments) != 1 || journey.Judgments[0].Kind != "reassessment-decision" ||
		journey.Judgments[0].Selected != factory.ReassessmentOptionContinue || !journey.Judgments[0].Wait ||
		journey.Judgments[0].WaitReason != "decision_not_actionable" || journey.Judgments[0].Confidence != 0.28 || journey.Judgments[0].Probability != 0.9 {
		t.Fatalf("recorded judgments: %+v", journey.Judgments)
	}
	if len(journey.Gates) != 1 || journey.Gates[0].CycleID != strings.Repeat("a", 32) ||
		journey.Gates[0].State != factory.ReassessmentAttention || !journey.Gates[0].Retryable {
		t.Fatalf("recorded gates: %+v", journey.Gates)
	}
	if len(journey.Adoptions) != 0 {
		t.Fatalf("recorded adoptions: %+v", journey.Adoptions)
	}
	// The top-level attempt/publication read the same lineage.
	if view.Attempt == nil || view.Attempt.ID != root.ID || view.Publication == nil || view.Publication.Candidate != journeyCandidate {
		t.Fatalf("top-level lineage drift: %+v", body)
	}

	// An explicit successor keeps the original receipt while the current
	// read renders its own low-confidence answer without treating continue
	// as permission to release. The cached decision path must spend nothing.
	ctx := context.Background()
	prior, err := fx.db.ReassessmentRecord(ctx, strings.Repeat("a", 32))
	if err != nil {
		t.Fatal(err)
	}
	successorID := factory.NewID()
	if _, err := fx.db.StartReassessmentRetry(ctx, factory.ReassessmentRetry{
		EventID: prior.Cycle.EventID, ExpectedPriorCycle: prior.Cycle.CycleID,
		CommandID: successorID, SuccessorCycleID: successorID,
	}); err != nil {
		t.Fatal(err)
	}
	next, err := fx.db.ReassessmentRecord(ctx, successorID)
	if err != nil {
		t.Fatal(err)
	}
	next.Local, next.Evidence = prior.Local, prior.Evidence
	next.InventoryDigest, next.EvidenceRevision = prior.InventoryDigest, prior.EvidenceRevision
	next.EvidenceIssues = prior.EvidenceIssues
	next.EvidenceFingerprint = control.SealEvidenceFingerprint(next.InventoryDigest, next.Cycle.EventID, successorID)
	next.Cycle.State = factory.ReassessmentDeciding
	if err := fx.db.CasReassessmentRecord(ctx, successorID, factory.ReassessmentCollecting, next); err != nil {
		t.Fatal(err)
	}
	fingerprint, err := control.DecisionFingerprintFromRecord(next)
	if err != nil || fingerprint == next.EvidenceFingerprint {
		t.Fatal("successor decision lost its distinct full-state identity", err)
	}
	if reserved, err := fx.db.AdmitReassessmentCall(ctx, successorID, factory.ReassessmentDeciding,
		factory.NewID(), factory.ReassessmentDecisionPurpose, "", fingerprint); err != nil || !reserved {
		t.Fatalf("reserve successor decision: %v %v", reserved, err)
	}
	if err := fx.db.RecordPriorityJudgment(ctx, factory.PriorityReceipt{
		Snapshot: fingerprint, Purpose: factory.JudgmentPurpose(factory.ReassessmentDecisionPurpose),
		Outcome: control.ReassessmentDecisionOutcome, CreatedUnix: 1700000001,
		Calls: []factory.PriorityCall{{Answer: factory.PriorityAnswer{
			Selected: factory.ReassessmentOptionContinue, Confidence: 0.24, Model: "test-low-confidence",
			Probabilities: map[string]float64{factory.ReassessmentOptionContinue: 0.9, factory.ReassessmentOptionReassess: 0.1},
		}}},
	}); err != nil {
		t.Fatal(err)
	}
	state, err := control.DecisionStateFromRecord(next)
	if err != nil {
		t.Fatal(err)
	}
	coord := &control.Coordinator{Store: fx.db, Planning: &control.PlanningJudgments{Chooser: q05RefusingChooser{}}}
	held, decisionErr := coord.DecideReassessment(ctx, next, state)
	if decisionErr == nil || held.WaitReason != "decision_not_actionable" || held.Cycle.Decision != "" {
		t.Fatalf("cached low-confidence continue became actionable: %+v %v", held.Cycle, decisionErr)
	}
	held.Cycle.State = factory.ReassessmentAttention
	if err := fx.db.CasReassessmentRecord(ctx, successorID, factory.ReassessmentDeciding, held); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		code, body := call(http.MethodGet, "/api/repositories/7/factory/issues/3", "")
		if code != http.StatusOK {
			t.Fatalf("successor readback status=%d body=%s", code, body)
		}
		journey := journeyIssueView(t, body).Journey
		if len(journey.Judgments) != 2 || journey.Judgments[0].CycleID != prior.Cycle.CycleID ||
			journey.Judgments[0].CycleState != factory.ReassessmentSuperseded || journey.Judgments[0].Confidence != 0.28 ||
			journey.Judgments[0].Wait || journey.Judgments[0].WaitReason != "" ||
			journey.Judgments[1].CycleID != successorID || journey.Judgments[1].CycleState != factory.ReassessmentAttention ||
			journey.Judgments[1].Selected != factory.ReassessmentOptionContinue || journey.Judgments[1].Confidence != 0.24 ||
			journey.Judgments[1].Probability != 0.9 || !journey.Judgments[1].Wait || journey.Judgments[1].WaitReason != "decision_not_actionable" ||
			len(journey.Gates) != 2 || journey.Gates[0].State != factory.ReassessmentSuperseded || journey.Gates[0].Retryable ||
			journey.Gates[1].CycleID != successorID || journey.Gates[1].State != factory.ReassessmentAttention {
			t.Fatalf("successor judgment lost its exact held receipt: %+v", journey)
		}
	}
	var receipts int
	if err := fx.raw.QueryRow(`SELECT count(*) FROM factory_priority_judgments`).Scan(&receipts); err != nil || receipts != 2 {
		t.Fatalf("readback changed retained decision receipts: %d %v", receipts, err)
	}
	retained, err := fx.db.ReassessmentRecord(ctx, successorID)
	if err != nil || len(retained.Cycle.Ledger.DecisionCalls) != 1 || retained.Cycle.State != factory.ReassessmentAttention {
		t.Fatalf("readback changed successor allowance/state: %+v %v", retained.Cycle, err)
	}
}

func TestFactoryIssueJourneyStaleHead(t *testing.T) {
	fake := &journeyAcceptanceSource{}
	fx, call := journeyMux(t, fake)
	acc1, _, _ := seedRichJourney(t, fx, fake)
	acc2 := admitJourneyAcceptance(t, fx.db, 2, acc1)
	fake.decision = journeyAcceptanceFor(acc2)

	code, body := call(http.MethodGet, "/api/repositories/7/factory/issues/3", "")
	if code != http.StatusOK {
		t.Fatalf("stale status=%d body=%s", code, body)
	}
	journey := journeyIssueView(t, body).Journey
	if !journey.Available || !journey.Lineage.Stale {
		t.Fatalf("stale flags: %+v", journey)
	}
	if journey.Lineage.Head != acc2 || journey.Lineage.Acceptance != acc1 {
		t.Fatalf("stale scope: %+v", journey.Lineage)
	}
	// The recorded chain still renders behind the moved head.
	if len(journey.Phases) != 2 || len(journey.Commits) != 1 {
		t.Fatalf("stale sections: %+v", journey)
	}
}

func TestFactoryIssueJourneyMalformedRows(t *testing.T) {
	fake := &journeyAcceptanceSource{}
	fx, call := journeyMux(t, fake)
	acceptance := admitJourneyAcceptance(t, fx.db, 1, "")
	fake.decision = journeyAcceptanceFor(acceptance)
	root := journeyOwner(t, strings.Repeat("7", 32), acceptance, project.ExecutionImplement, factory.AssignmentFinished, factory.Succeeded, "")
	insertJourneyRow(t, fx.raw, root, factory.AssignmentFinished)
	// One poisoned correction, one poisoned finding; the valid
	// publication shell (candidate, PR linkage) still renders.
	publication := journeyPublicationFixture(root, "bogus_effect", "cosmic")
	insertJourneyPublication(t, fx.raw, publication)
	// A funded receipt whose answer names no probability, on an
	// attention cycle without its wait reason: judgment and gate both
	// name their rows instead of dropping them silently.
	malformedCompletion := factory.CompletionEvent{
		ID: strings.Repeat("b", 32), PublicationID: fmt.Sprintf("%032x", 7001),
		Repository: journeyRepo, Issue: journeyIssue, Generation: 1,
		AcceptanceID: acceptance, MergedCommit: strings.Repeat("c", 40),
		ClosingOccurrence: "malformed-1",
	}
	record := store.ReassessmentRecord{
		Cycle: factory.ReassessmentCycle{
			EventID: strings.Repeat("b", 32), CycleID: strings.Repeat("a", 32),
			State: factory.ReassessmentAttention, Decision: factory.ReassessmentOptionReassess,
			Repository: journeyRepo, Generation: 1,
		},
		Completion:          malformedCompletion,
		Issue:               journeyIssue,
		EvidenceFingerprint: strings.Repeat("b", 64),
	}
	data, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if _, err := fx.raw.Exec(`INSERT INTO factory_completions(repository,generation,event,publication,occurrence,data) VALUES($1,$2,$3,$4,$5,$6)
		ON CONFLICT(repository,generation) DO NOTHING`,
		journeyRepo, 1, malformedCompletion.ID,
		malformedCompletion.PublicationID, malformedCompletion.ClosingOccurrence, `{}`); err != nil {
		t.Fatal(err)
	}
	if _, err := fx.raw.Exec(`INSERT INTO factory_reassessments(cycle,repository,generation,event,state,data) VALUES($1,$2,$3,$4,$5,$6)`,
		record.Cycle.CycleID, journeyRepo, 1, record.Cycle.EventID, record.Cycle.State, string(data)); err != nil {
		t.Fatal(err)
	}

	if err := fx.db.RecordPriorityJudgment(ctx, factory.PriorityReceipt{
		Snapshot:    strings.Repeat("b", 64),
		Calls:       []factory.PriorityCall{{Answer: factory.PriorityAnswer{Selected: factory.ReassessmentOptionReassess, Confidence: 0.87, Model: "m"}}},
		CreatedUnix: 1700000000,
	}); err != nil {
		t.Fatal(err)
	}

	code, body := call(http.MethodGet, "/api/repositories/7/factory/issues/3", "")
	if code != http.StatusOK {
		t.Fatalf("malformed status=%d body=%s", code, body)
	}
	view := journeyIssueView(t, body)
	reasons := journeyReasons(view)
	for _, want := range []string{
		"corrections_invalid_row", "findings_invalid_row",
		"judgments_invalid_row", "gates_unrenderable",
		"adoptions_unavailable", "delivery_unavailable",
	} {
		if !reasons[want] {
			t.Fatalf("missing reason %s: %+v", want, view.Journey.Reasons)
		}
	}
	if view.Journey.Complete {
		t.Fatalf("malformed journey reads complete: %+v", view.Journey)
	}
	if len(view.Journey.Corrections) != 0 || len(view.Journey.Findings) != 0 ||
		len(view.Journey.Judgments) != 0 || len(view.Journey.Gates) != 0 {
		t.Fatalf("malformed rows rendered: %+v", view.Journey)
	}
	if len(view.Journey.Commits) != 1 || len(view.Journey.Phases) != 1 {
		t.Fatalf("valid rows dropped: %+v", view.Journey)
	}
}

func TestFactoryIssueJourneyCorruptRowIs503(t *testing.T) {
	fake := &journeyAcceptanceSource{}
	fx, call := journeyMux(t, fake)
	acceptance := admitJourneyAcceptance(t, fx.db, 1, "")
	fake.decision = journeyAcceptanceFor(acceptance)
	older := journeyOwner(t, strings.Repeat("6", 32), acceptance, project.ExecutionImplement, factory.AssignmentFinished, factory.Succeeded, "")
	newer := journeyOwner(t, strings.Repeat("7", 32), acceptance, project.ExecutionImplement, factory.AssignmentFinished, factory.Succeeded, "")
	insertJourneyRow(t, fx.raw, older, factory.AssignmentFinished)
	insertJourneyRow(t, fx.raw, newer, factory.AssignmentFinished)
	// Valid JSONB that still names this lineage but cannot decode:
	// the scope filter selects it, the reader fails, and the journey
	// reports an explicit storage failure, never a silent cut.
	if _, err := fx.raw.Exec(`UPDATE factory_assignments SET data=$1 WHERE id=$2`,
		`{"acceptance": "`+acceptance+`", "repository": "seven"}`, older.ID); err != nil {
		t.Fatal(err)
	}

	code, body := call(http.MethodGet, "/api/repositories/7/factory/issues/3", "")
	if code != http.StatusServiceUnavailable || journeyErrorCode(t, body) != "store_unavailable" {
		t.Fatalf("corrupt status=%d body=%s", code, body)
	}
}

func TestFactoryIssueJourneyOldAcceptanceExcluded(t *testing.T) {
	fake := &journeyAcceptanceSource{}
	fx, call := journeyMux(t, fake)
	oldAcc := fmt.Sprintf("d%024x", 90)
	acceptance := admitJourneyAcceptance(t, fx.db, 1, "")
	fake.decision = journeyAcceptanceFor(acceptance)
	for i := 0; i < 5; i++ {
		stale := journeyOwner(t, fmt.Sprintf("%032x", 100+i), oldAcc, project.ExecutionDecompose, factory.AssignmentFinished, factory.Succeeded, "")
		insertJourneyRow(t, fx.raw, stale, factory.AssignmentFinished)
	}
	root := journeyOwner(t, strings.Repeat("7", 32), acceptance, project.ExecutionImplement, factory.AssignmentFinished, factory.Succeeded, "")
	insertJourneyRow(t, fx.raw, root, factory.AssignmentFinished)

	code, body := call(http.MethodGet, "/api/repositories/7/factory/issues/3", "")
	if code != http.StatusOK {
		t.Fatalf("scoped status=%d body=%s", code, body)
	}
	journey := journeyIssueView(t, body).Journey
	if len(journey.Phases) != 1 || journey.Phases[0].Phase != project.ExecutionImplement {
		t.Fatalf("old acceptance leaked: %+v", journey.Phases)
	}
	if journey.Lineage.Acceptance != acceptance {
		t.Fatalf("scoped lineage: %+v", journey.Lineage)
	}
}

var journeyBulkPhases = []string{
	project.ExecutionDecompose, project.ExecutionInvestigate, project.ExecutionImplement,
	project.ExecutionReview, project.ExecutionCorrect, project.ExecutionReassess,
}

// seedJourneyChain records one owner plus n child rows on one
// acceptance after other-issue and old-acceptance rows, so seq order
// cannot leak across scopes. Every row is finished; every seventh
// fails, so the tail mixes complete and attention states.
func seedJourneyChain(t *testing.T, fx *journeyHTTPFixture, acceptance string, n int) factory.Assignment {
	t.Helper()
	for i := 0; i < 20; i++ {
		other := journeyOwner(t, fmt.Sprintf("%032x", 5000+i), acceptance, project.ExecutionDecompose, factory.AssignmentFinished, factory.Succeeded, "")
		other.Issue = 888
		insertJourneyRow(t, fx.raw, other, factory.AssignmentFinished)
	}
	oldAcc := fmt.Sprintf("d%024x", 90)
	for i := 0; i < 10; i++ {
		stale := journeyOwner(t, fmt.Sprintf("%032x", 6000+i), oldAcc, project.ExecutionDecompose, factory.AssignmentFinished, factory.Succeeded, "")
		stale.PublicationAssignment, stale.AttemptRoot = stale.ID, stale.ID
		insertJourneyRow(t, fx.raw, stale, factory.AssignmentFinished)
	}
	root := journeyOwner(t, strings.Repeat("7", 32), acceptance, project.ExecutionImplement, factory.AssignmentFinished, factory.Succeeded, "")
	insertJourneyRow(t, fx.raw, root, factory.AssignmentFinished)
	for i := 0; i < n; i++ {
		outcome := factory.Succeeded
		if i%7 == 6 {
			outcome = factory.Failed
		}
		child := journeyChild(t, fmt.Sprintf("%032x", 10000+i), root.ID, acceptance, journeyBulkPhases[i%len(journeyBulkPhases)], factory.AssignmentFinished, outcome)
		insertJourneyRow(t, fx.raw, child, factory.AssignmentFinished)
	}
	return root
}

func journeyBulkExpect(at int) (string, string) {
	phase := journeyBulkPhases[at%len(journeyBulkPhases)]
	state := "complete"
	if at%7 == 6 {
		state = "attention"
	}
	return phase, state
}

// journeyNewestChildPerPhase returns, for each bulk phase, the newest
// child index carrying it among n cycled children. The phase section
// must render exactly these newest states with their refs, whatever
// older rows repeat the same phases.
func journeyNewestChildPerPhase(n int) map[string]int {
	newest := make(map[string]int, len(journeyBulkPhases))
	for i := n - 1; i >= 0; i-- {
		phase := journeyBulkPhases[i%len(journeyBulkPhases)]
		if _, ok := newest[phase]; !ok {
			newest[phase] = i
		}
	}
	return newest
}

// journeyExpectCurrentPhases asserts one current state per canonical
// phase in stable canonical order, each naming the exact newest
// contributing child assignment, run and AttemptRoot. Children share
// one run identity; the owner root carries its own.
func journeyExpectCurrentPhases(t *testing.T, journey *factoryservice.IssueJourney, root factory.Assignment, n int) {
	t.Helper()
	childRun := strings.Repeat("e5", 16)
	newest := journeyNewestChildPerPhase(n)
	if len(journey.Phases) != len(journeyBulkPhases) {
		t.Fatalf("current phases: %+v", journey.Phases)
	}
	for i, phase := range journeyBulkPhases {
		at, ok := newest[phase]
		if !ok {
			t.Fatalf("phase %s has no newest child", phase)
		}
		wantPhase, wantState := journeyBulkExpect(at)
		got := journey.Phases[i]
		if got.Phase != wantPhase || got.State != wantState {
			t.Fatalf("current phase %d: %+v want %s/%s", i, got, wantPhase, wantState)
		}
		if want, wantRun := fmt.Sprintf("%032x", 10000+at), childRun; got.Assignment != want || got.Run != wantRun || got.AttemptRoot != root.ID {
			t.Fatalf("current phase %d refs: %+v want %s/%s/%s", i, got, want, wantRun, root.ID)
		}
	}
}

func TestFactoryIssueJourneyChainTail(t *testing.T) {
	fake := &journeyAcceptanceSource{}
	fx, call := journeyMux(t, fake)
	acceptance := admitJourneyAcceptance(t, fx.db, 1, "")
	fake.decision = journeyAcceptanceFor(acceptance)
	const n = 300
	root := seedJourneyChain(t, fx, acceptance, n)

	code, body := call(http.MethodGet, "/api/repositories/7/factory/issues/3", "")
	if code != http.StatusOK {
		t.Fatalf("tail status=%d body=%s", code, body)
	}
	journey := journeyIssueView(t, body).Journey
	if !journey.Available || journey.Complete {
		t.Fatalf("tail flags: %+v", journey)
	}
	reasons := journeyReasons(journeyIssueView(t, body))
	if reasons["chain_page_bound"] {
		t.Fatalf("tail hit the page bound early: %+v", journey.Reasons)
	}
	// 301 valid rows on one root: every canonical phase renders its
	// newest state with exact refs — no old capped prefix stands in
	// for current, and no phase repeats or displaces another.
	journeyExpectCurrentPhases(t, journey, root, n)
	for _, unwanted := range []string{
		"phases_invalid_row", "phases_truncated",
	} {
		if reasons[unwanted] {
			t.Fatalf("tail names %s without cause: %+v", unwanted, journey.Reasons)
		}
	}
	// Two pages fetched (256 plus 45): the cursor trail is retained
	// lineage history, and nothing was cut so no resume cursor names
	// a remainder.
	if journey.Lineage.NextCursor != "" || len(journey.Lineage.ChainCursors) != 2 {
		t.Fatalf("tail resume refs: %+v", journey.Lineage)
	}
	if journey.Lineage.LatestAssignment != fmt.Sprintf("%032x", 10000+n-1) {
		t.Fatalf("tail latest: %+v", journey.Lineage)
	}
}

func TestFactoryIssueJourneyChainPageBound(t *testing.T) {
	fake := &journeyAcceptanceSource{}
	fx, call := journeyMux(t, fake)
	acceptance := admitJourneyAcceptance(t, fx.db, 1, "")
	fake.decision = journeyAcceptanceFor(acceptance)
	const n = 4*256 + 10
	root := seedJourneyChain(t, fx, acceptance, n)

	code, body := call(http.MethodGet, "/api/repositories/7/factory/issues/3", "")
	if code != http.StatusOK {
		t.Fatalf("bound status=%d body=%s", code, body)
	}
	journey := journeyIssueView(t, body).Journey
	if !journey.Available || journey.Complete {
		t.Fatalf("bound flags: %+v", journey)
	}
	if reasons := journeyReasons(journeyIssueView(t, body)); !reasons["chain_page_bound"] {
		t.Fatalf("bound missing page reason: %+v", journey.Reasons)
	}
	// The page bound cuts only the retained cursor trail: the phase
	// section still reads newest-per-phase across the whole scope, so
	// overflow past the 1024-row chain prefix cannot mask a newer
	// failed or current state.
	journeyExpectCurrentPhases(t, journey, root, n)
	if journey.Lineage.NextCursor == "" || len(journey.Lineage.ChainCursors) != 4 {
		t.Fatalf("bound resume refs: %+v", journey.Lineage)
	}
}

func journeyStoreCounts(t *testing.T, raw *sql.DB) map[string]int64 {
	t.Helper()
	counts := make(map[string]int64)
	for _, table := range []string{
		"factory_assignments", "factory_publications", "factory_runs",
		"factory_plan_revisions", "factory_plan_tasks", "factory_priority_judgments",
		"factory_reassessments",
	} {
		var count int64
		if err := raw.QueryRow("SELECT count(*) FROM " + table).Scan(&count); err != nil {
			t.Fatalf("count %s: %v", table, err)
		}
		counts[table] = count
	}
	return counts
}

func TestSharedRunStatusRead(t *testing.T) {
	ctx := t.Context()
	fx, _ := journeyMux(t, nil)
	const actor collaboration.ActorID = 9007199254740993
	const repository collaboration.RepositoryID = 9223372036854775807
	const workspace = "p123456789012345678901234"
	now := time.Now().UTC()
	run := factory.Run{ID: strings.Repeat("a", 32), ProjectID: workspace, Role: "coder", Started: now, Deadline: now.Add(time.Hour),
		InputSHA: strings.Repeat("a", 40), Image: "sha256:" + strings.Repeat("b", 64), Harness: "muse", Model: "declared-fixture"}
	if err := fx.db.RecordFactoryRun(ctx, run); err != nil {
		t.Fatal(err)
	}
	wanted := run.ID
	checks, denyAt, inspections := 0, 0, 0
	denied := errors.New("declared current run authority denied")
	var scopeErr error
	observationErr := execution.ErrRunNotObserved
	var observation execution.RunObservation
	var cancelDuringInspect context.CancelFunc
	reader := factoryservice.RunStatus{Store: fx.db,
		WriteAccess: func(ctx context.Context, admitted collaboration.ActorID, repo collaboration.RepositoryID) error {
			if admitted != actor || repo != repository {
				t.Fatal("run status changed canonical authority subject", admitted, repo)
			}
			checks++
			if checks == denyAt {
				return denied
			}
			return nil
		},
		RunRepository: func(ctx context.Context, admitted collaboration.ActorID, selected factory.Run) (collaboration.RepositoryID, error) {
			if admitted != actor || selected.ID != wanted || selected.ProjectID != workspace {
				t.Fatal("run status changed runtime scope", admitted, selected.ID)
			}
			return repository, scopeErr
		},
		Inspect: func(ctx context.Context, admitted collaboration.ActorID, selected factory.Run) (execution.RunObservation, error) {
			if admitted != actor || selected.ID != wanted || selected.ProjectID != workspace {
				t.Fatal("run status inspected another runtime", admitted, selected.ID)
			}
			inspections++
			if cancelDuringInspect != nil {
				cancelDuringInspect()
			}
			return observation, observationErr
		},
	}
	read := func() *factoryservice.RunStatusView {
		t.Helper()
		checks = 0
		view, err := reader.Read(ctx, actor, wanted)
		if err != nil {
			t.Fatal(err)
		}
		return view
	}
	if view := read(); view.State != nil || view.View != nil || view.Run.Reconciled {
		t.Fatal("unobserved manual run became bound or retired", view)
	}
	if _, _, err := fx.db.RecordFactoryRunView(ctx, factory.RunView{RunID: run.ID, Repository: int64(repository), Issue: 3}); err != nil {
		t.Fatal(err)
	}
	observation = execution.RunObservation{Run: run.ID, Workspace: workspace, Kind: collaboration.Appliance, Phase: project.FactoryRunning,
		Retirement: execution.Running, Live: true, Appliance: &execution.ApplianceRunFacts{Invocation: execution.ApplianceInvocation{Project: workspace, Container: "declared-container", Unit: "declared-unit", Invocation: strings.Repeat("c", 32)}}}
	observationErr = nil
	view := read()
	if view.State == nil || !view.State.Live || view.State.Terminal || view.State.Invocation != strings.Repeat("c", 32) || view.View.Repository != strconv.FormatInt(int64(repository), 10) {
		t.Fatal("typed appliance observation lost exact binding", view)
	}
	observation.Phase, observation.Live, observation.Retirement, observation.Output = project.FactoryCompleted, false, execution.Retired, strings.Repeat("x", 20000)
	view = read()
	if !view.State.Terminal || view.State.Live || !view.State.OutputTruncated || len(view.State.Output) != 16*1024 || view.State.Retirement != "confirmed" {
		t.Fatal("terminal excerpt or retirement projection", view.State)
	}
	// Native facts are declared runtime observations, not a launched process.
	// The recorded tool/process/custody metadata must never become UI payload.
	native := run
	native.ID, native.Image, native.Harness = strings.Repeat("b", 32), "", "muse-1.0.0"
	native.Native = &execution.NativeRecord{Tool: execution.NativeTool{Harness: "muse", Version: "1.0.0", SHA256: strings.Repeat("e", 64), Executable: "/home/vince/declared-tools/muse"},
		Retirement: execution.Running, Provenance: execution.NativeProvenance{TrustMode: "same_user_trusted_host", Profile: collaboration.ProfileID(factory.NewID()),
			InstanceID: factory.NewID(), WorktreeID: "declared-private-worktree", Custody: execution.NativeCustody{State: execution.NotAcquired},
			Process: &execution.NativeProcess{InvocationID: strings.Repeat("d", 32), SystemInstance: "declared-boot", StartIdentity: "declared-start", PID: 123,
				StartedAt: now, Executable: "/home/vince/declared-tools/muse", ExecutableSHA256: strings.Repeat("e", 64)}}}
	if err := fx.db.RecordFactoryRun(ctx, native); err != nil {
		t.Fatal("declared native run record", err)
	}
	wanted = native.ID
	observation = execution.RunObservation{Run: native.ID, Workspace: workspace, Kind: collaboration.Native, Phase: project.FactoryRunning,
		Retirement: execution.Running, Live: true, Native: &native.Native.Provenance}
	view = read()
	if view.State == nil || !view.State.Live || view.State.Invocation != strings.Repeat("d", 32) || view.State.Container != "" || view.State.Unit != "" {
		t.Fatal("native observation acquired an appliance binding", view.State)
	}
	encoded, _ := json.Marshal(view)
	for _, private := range []string{"declared-tools", "declared-private-worktree", "declared-start", "executable_sha256", "custody", "system_instance"} {
		if strings.Contains(string(encoded), private) {
			t.Fatal("private native evidence entered run status", private)
		}
	}
	valid := observation
	foreign := *observation.Native
	process := *foreign.Process
	process.StartIdentity = "another-start-for-reused-pid"
	foreign.Process, observation.Native = &process, &foreign
	if view, err := reader.Read(ctx, actor, wanted); view != nil || !errors.Is(err, execution.ErrRunChanged) {
		t.Fatal("reused PID/start mismatch accepted", view, err)
	}
	observation = valid
	observation.Run = run.ID
	if view, err := reader.Read(ctx, actor, wanted); view != nil || !errors.Is(err, factoryservice.ErrRunStatusObservation) {
		t.Fatal("foreign run observation accepted", view, err)
	}
	observation = valid
	observation.Workspace = "another-workspace"
	if view, err := reader.Read(ctx, actor, wanted); view != nil || !errors.Is(err, factoryservice.ErrRunStatusObservation) {
		t.Fatal("foreign workspace observation accepted", view, err)
	}
	observation = valid
	observationErr = errors.New("declared runtime observation is unknown")
	if view, err := reader.Read(ctx, actor, wanted); view != nil || !errors.Is(err, observationErr) {
		t.Fatal("partial observation returned on runtime error", view, err)
	}
	observationErr = nil
	scopeErr = errors.New("declared runtime scope is unknown")
	if view, err := reader.Read(ctx, actor, wanted); view != nil || !errors.Is(err, scopeErr) {
		t.Fatal("unknown runtime scope returned status", view, err)
	}
	scopeErr = nil
	checks, denyAt = 0, 2
	if view, err := reader.Read(ctx, actor, wanted); view != nil || !errors.Is(err, denied) {
		t.Fatal("late write revocation exposed status", view, err)
	}
	checks, denyAt = 0, 1
	beforeInspect := inspections
	if view, err := reader.Read(ctx, actor, wanted); view != nil || !errors.Is(err, denied) || inspections != beforeInspect {
		t.Fatal("denied reader reached runtime", view, err)
	}
	denyAt = 0
	cancelled, cancel := context.WithCancel(ctx)
	cancelDuringInspect = cancel
	if view, err := reader.Read(cancelled, actor, wanted); view != nil || !errors.Is(err, context.Canceled) {
		t.Fatal("late observation cancellation returned status", view, err)
	}
	cancelDuringInspect = nil
	missing := reader
	missing.Inspect = nil
	if view, err := missing.Read(ctx, actor, wanted); view != nil || !errors.Is(err, factoryservice.ErrRunStatusUnavailable) {
		t.Fatal("missing runtime observation became absence", view, err)
	}
	// Record a conflicting display scope: no observation can repair or adopt it.
	if _, _, err := fx.db.RecordFactoryRunView(ctx, factory.RunView{RunID: native.ID, Repository: int64(repository - 1)}); err != nil {
		t.Fatal(err)
	}
	beforeInspect = inspections
	if view, err := reader.Read(ctx, actor, wanted); view != nil || !errors.Is(err, factoryservice.ErrRunStatusScope) || inspections != beforeInspect {
		t.Fatal("foreign display binding reached runtime", view, err)
	}
	wanted, observationErr = run.ID, execution.ErrRunNotObserved
	for _, expected := range []factory.Run{run, native} {
		stored, err := fx.db.FactoryRun(ctx, expected.ID)
		beforeJSON, _ := json.Marshal(expected)
		afterJSON, _ := json.Marshal(stored)
		if err != nil || string(beforeJSON) != string(afterJSON) {
			t.Fatal("status reads changed captured run/process/custody facts", expected.ID, err)
		}
	}
	var commands, usage int
	if err := fx.raw.QueryRowContext(ctx, `SELECT (SELECT count(*) FROM factory_commands),(SELECT count(*) FROM factory_usage)`).Scan(&commands, &usage); err != nil || commands != 0 || usage != 0 {
		t.Fatal("status reads wrote commands or charged execution", commands, usage, err)
	}
	snapshot := func() string {
		t.Helper()
		var data string
		if err := fx.raw.QueryRowContext(ctx, `SELECT jsonb_build_array(
		 (SELECT jsonb_agg(to_jsonb(r) ORDER BY id) FROM factory_runs r),
		 (SELECT jsonb_agg(to_jsonb(v) ORDER BY run) FROM factory_run_views v),
		 (SELECT count(*) FROM factory_commands),(SELECT count(*) FROM projects),(SELECT count(*) FROM factory_usage))::text`).Scan(&data); err != nil {
			t.Fatal(err)
		}
		return data
	}
	before := snapshot()
	retained := read()
	beforeJSON, _ := json.Marshal(retained)
	if err := fx.db.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := store.Open(fx.dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = reopened.Close() })
	reader.Store = reopened
	retained = read()
	afterJSON, _ := json.Marshal(retained)
	if string(beforeJSON) != string(afterJSON) || snapshot() != before {
		t.Fatal("run read/reopen changed retained status or records")
	}
	// A declared host serves the reopened service over real HTTP. No native
	// session, Project, process launch or replacement run journal is involved.
	var admitted atomic.Bool
	admitted.Store(true)
	var currentCalls, loseAt atomic.Int64
	transport := factoryhttpapi.Runs{Store: reopened, Status: reader,
		Host: fx.api.FountainIdentities.HostID(), Actor: actor,
		Current: func(w http.ResponseWriter, r *http.Request) bool {
			call := currentCalls.Add(1)
			if !admitted.Load() || call == loseAt.Load() || r.Header.Get("Authorization") != "Bearer declared-run-reader" {
				w.WriteHeader(http.StatusUnauthorized)
				return false
			}
			return true
		},
		AuthorityError: func(w http.ResponseWriter, err error) { w.WriteHeader(http.StatusForbidden) },
	}
	server := httptest.NewServer(transport.Handler())
	t.Cleanup(server.Close)
	call := func(target *httptest.Server, method, path string, expected int) []byte {
		t.Helper()
		currentCalls.Store(0)
		req, err := http.NewRequestWithContext(ctx, method, target.URL+path, nil)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Authorization", "Bearer declared-run-reader")
		res, err := target.Client().Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		body, err := io.ReadAll(res.Body)
		if err != nil || res.StatusCode != expected || expected == 200 && res.Header.Get("Cache-Control") != "no-store" {
			t.Fatal("shared run HTTP", method, path, res.StatusCode, string(body), err)
		}
		return body
	}
	path := "/api/factory/runs/" + run.ID
	body := call(server, http.MethodGet, path, 200)
	if strings.TrimSpace(string(body)) != string(beforeJSON) {
		t.Fatal("HTTP changed retained run projection", string(body))
	}
	call(server, http.MethodGet, path+"?cursor=1", 400)
	call(server, http.MethodGet, path+"?", 400)
	call(server, http.MethodHead, path, 405)
	call(server, http.MethodGet, "/api/factory/runs/short", 404)
	call(server, http.MethodGet, "/api/factory/runs/"+factory.NewID(), 404)
	wanted = native.ID
	call(server, http.MethodGet, "/api/factory/runs/"+native.ID, 409)
	wanted = run.ID
	checks, denyAt = 0, 1
	beforeInspect = inspections
	call(server, http.MethodGet, path, 403)
	if inspections != beforeInspect {
		t.Fatal("denied HTTP reader reached runtime inspection")
	}
	denyAt = 0
	observationErr = errors.New("private declared runtime outage")
	if body := call(server, http.MethodGet, path, 503); strings.Contains(string(body), "private declared") {
		t.Fatal("run transport exposed runtime failure detail", string(body))
	}
	observationErr = execution.ErrRunNotObserved
	loseAt.Store(2) // The account changes after service read, before disclosure.
	if body := call(server, http.MethodGet, path, 401); len(body) != 0 {
		t.Fatal("late host withdrawal disclosed run status", string(body))
	}
	loseAt.Store(0)
	admitted.Store(false)
	beforeInspect = inspections
	call(server, http.MethodGet, path, 401)
	if inspections != beforeInspect {
		t.Fatal("withdrawn host admission reached runtime")
	}
	admitted.Store(true)
	missingAdmission := transport
	missingAdmission.Current = nil
	missingServer := httptest.NewServer(missingAdmission.Handler())
	t.Cleanup(missingServer.Close)
	call(missingServer, http.MethodGet, path, 503)
	mixed := transport
	mixed.Status.Store = fx.db // Closed original handle is not the selected store.
	mixedServer := httptest.NewServer(mixed.Handler())
	t.Cleanup(mixedServer.Close)
	call(mixedServer, http.MethodGet, path, 503)
	if snapshot() != before {
		t.Fatal("shared run HTTP changed retained runs, bindings, commands or usage")
	}
	var projects int
	if err := fx.raw.QueryRowContext(ctx, `SELECT count(*) FROM projects`).Scan(&projects); err != nil || projects != 0 {
		t.Fatal("shared run status required a native Project", projects, err)
	}
}

func TestSharedFactoryInterventions(t *testing.T) {
	ctx := t.Context()
	fx, _ := journeyMux(t, nil)
	const actor collaboration.ActorID = 9007199254740993
	const repository collaboration.RepositoryID = 9223372036854775807
	const principal = "profile:maintainer"
	const workspace = "p123456789012345678901234"
	allowed, checks, denyAt, memberCalls := true, 0, 0, 0
	denied := errors.New("declared current runtime write authority denied")
	var scopeErr error
	member := ""
	var nativeTarget *execution.NativeCheckoutRef
	var cancelInMembership context.CancelFunc
	application := factoryservice.Interventions{Store: fx.db, Writer: &control.Coordinator{Store: fx.db},
		WriteAccess: func(ctx context.Context, admitted collaboration.ActorID, repo collaboration.RepositoryID) error {
			if admitted != actor || repo != repository {
				t.Fatal("intervention authority changed canonical subject", admitted, repo)
			}
			checks++
			if !allowed || checks == denyAt {
				return denied
			}
			return nil
		},
		RunRepository: func(ctx context.Context, admitted collaboration.ActorID, run factory.Run) (collaboration.RepositoryID, error) {
			if admitted != actor || run.ProjectID != workspace {
				t.Fatal("run runtime scope changed", admitted, run.ProjectID)
			}
			return repository, scopeErr
		},
		RecoveryTarget: func(ctx context.Context, admitted collaboration.ActorID, repo collaboration.RepositoryID, run factory.Run) (execution.RecoveryTarget, error) {
			if admitted != actor || repo != repository || run.ProjectID != workspace {
				t.Fatal("takeover membership changed subject", admitted, repo, run.ProjectID)
			}
			memberCalls++
			if cancelInMembership != nil {
				cancelInMembership()
			}
			return execution.RecoveryTarget{Member: member, Native: nativeTarget}, nil
		},
	}
	pause := factoryservice.RepositoryCommand{Action: factory.ActionPause}
	result, err := application.Repository(ctx, actor, principal, repository, pause)
	if err != nil {
		t.Fatal(err)
	}
	receipt, ok := result.(factory.PauseResult)
	if !ok || !receipt.Paused || receipt.Withdrawal.ClosedBy != principal || receipt.Withdrawal.Repository != int64(repository) {
		t.Fatal("pause changed scope or principal", result)
	}
	if result, err := application.Repository(ctx, actor, principal, repository, factoryservice.RepositoryCommand{Action: factory.ActionResume}); !errors.Is(err, control.ErrIneffectiveAuthority) {
		t.Fatal("resume invented effective authority", result, err)
	}
	now := time.Now().UTC()
	for _, id := range []string{strings.Repeat("a", 32), strings.Repeat("b", 32)} {
		run := factory.Run{ID: id, ProjectID: workspace, Role: "coder", InputSHA: strings.Repeat("a", 40), Started: now, Deadline: now.Add(time.Hour),
			Image: "sha256:" + strings.Repeat("b", 64), Harness: "muse", Model: "declared-fixture"}
		if err := fx.db.RecordFactoryRun(ctx, run); err != nil {
			t.Fatal(err)
		}
		run.Reconciled, run.Outcome, run.Summary = true, factory.Cancelled, "stopped"
		if err := fx.db.SaveFactoryRun(ctx, run); err != nil {
			t.Fatal(err)
		}
	}
	runID := strings.Repeat("a", 32)
	stop := factoryservice.RunCommand{Action: factory.ActionStop}
	result, err = application.Run(ctx, actor, principal, runID, stop)
	stopped, ok := result.(control.StopResult)
	if err != nil || !ok || !stopped.Confirmed || stopped.Uncertain || stopped.RunID != runID || stopped.Outcome != string(factory.Cancelled) || memberCalls != 0 {
		t.Fatal("settled manual run stop changed its runtime or required membership", result, err)
	}
	if _, _, err := fx.db.RecordFactoryRunView(ctx, factory.RunView{RunID: strings.Repeat("b", 32), Repository: int64(repository - 1)}); err != nil {
		t.Fatal(err)
	}
	if _, err := application.Run(ctx, actor, principal, strings.Repeat("b", 32), stop); !errors.Is(err, factoryservice.ErrWorkJourneyScope) {
		t.Fatal("run intervention accepted a conflicting display/runtime repository", err)
	}
	snapshot := func() string {
		t.Helper()
		var data string
		if err := fx.raw.QueryRowContext(ctx, `SELECT jsonb_build_array(
		 (SELECT jsonb_agg(to_jsonb(r) ORDER BY id) FROM factory_runs r),
		 (SELECT jsonb_agg(to_jsonb(d) ORDER BY repository) FROM factory_dispatch d),
		 (SELECT jsonb_agg(to_jsonb(c) ORDER BY id) FROM factory_commands c),
		 (SELECT jsonb_agg(to_jsonb(v) ORDER BY run) FROM factory_run_views v),
		 (SELECT count(*) FROM projects))::text`).Scan(&data); err != nil {
			t.Fatal(err)
		}
		return data
	}
	before := snapshot()
	allowed = false
	if _, err := application.Repository(ctx, actor, principal, repository, pause); !errors.Is(err, denied) {
		t.Fatal("saved pause bypassed current authority", err)
	}
	if _, err := application.Run(ctx, actor, principal, runID, stop); !errors.Is(err, denied) {
		t.Fatal("settled run bypassed current authority", err)
	}
	allowed = true
	scopeErr = errors.New("declared unknown runtime scope")
	if _, err := application.Run(ctx, actor, principal, runID, stop); !errors.Is(err, scopeErr) {
		t.Fatal("unknown runtime scope admitted intervention", err)
	}
	scopeErr = nil
	takeover := factoryservice.RunCommand{Action: factory.ActionTakeover}
	if _, err := application.Run(ctx, actor, principal, runID, takeover); !errors.Is(err, factoryservice.ErrInterventionRecoveryTarget) {
		t.Fatal("takeover omitted current membership", err)
	}
	nativeTarget = &execution.NativeCheckoutRef{Profile: collaboration.ProfileID(factory.NewID()), CheckoutID: factory.NewID()}
	if _, err := application.Run(ctx, actor, principal, runID, takeover); !errors.Is(err, execution.ErrRecovery) {
		t.Fatal("native destination was lost or admitted for an appliance run", err)
	}
	nativeTarget = nil
	member, checks, denyAt = "declared-member", 0, 2
	if _, err := application.Run(ctx, actor, principal, runID, takeover); !errors.Is(err, denied) {
		t.Fatal("late write revocation reached takeover", err)
	}
	denyAt = 0
	cancelled, cancel := context.WithCancel(ctx)
	cancelInMembership = cancel
	if _, err := application.Run(cancelled, actor, principal, runID, takeover); !errors.Is(err, context.Canceled) {
		t.Fatal("late membership cancellation reached takeover", err)
	}
	cancelInMembership = nil
	cancelled, cancel = context.WithCancel(ctx)
	cancel()
	if _, err := application.Run(cancelled, actor, principal, runID, stop); !errors.Is(err, context.Canceled) {
		t.Fatal("cancelled run intervention", err)
	}
	if _, err := application.Repository(cancelled, actor, principal, repository, pause); !errors.Is(err, context.Canceled) {
		t.Fatal("cancelled repository intervention", err)
	}
	if _, err := application.Repository(ctx, actor, principal, repository, factoryservice.RepositoryCommand{Action: factory.ActionPause, CommandID: factory.NewID()}); !errors.Is(err, factoryservice.ErrInterventionPayload) {
		t.Fatal("pause accepted an unrelated command identity", err)
	}
	if _, err := application.Repository(ctx, actor, principal, repository, factoryservice.RepositoryCommand{Action: "launch"}); !errors.Is(err, factoryservice.ErrInterventionAction) {
		t.Fatal("unknown action admitted", err)
	}
	if _, err := application.Run(ctx, actor, principal, runID, factoryservice.RunCommand{Action: factory.ActionRetry}); !errors.Is(err, factoryservice.ErrInterventionCommand) {
		t.Fatal("retry omitted durable command identity", err)
	}
	if _, err := application.Run(ctx, actor, principal, runID, factoryservice.RunCommand{Action: factory.ActionContextCurrent}); !errors.Is(err, factoryservice.ErrInterventionPayload) {
		t.Fatal("context selection omitted exact scope", err)
	}
	if _, err := application.Run(ctx, actor, principal, factory.NewID(), stop); !errors.Is(err, factory.ErrNotFound) {
		t.Fatal("missing run disclosed or guessed", err)
	}
	missing := application
	missing.Writer = nil
	if _, err := missing.Run(ctx, actor, principal, runID, stop); !errors.Is(err, factoryservice.ErrInterventionUnavailable) {
		t.Fatal("missing coordinator admitted run action", err)
	}
	if _, err := missing.Repository(ctx, actor, principal, repository, pause); !errors.Is(err, factoryservice.ErrInterventionUnavailable) {
		t.Fatal("missing coordinator admitted repository action", err)
	}
	missing = application
	missing.WriteAccess = nil
	if _, err := missing.Repository(ctx, actor, principal, repository, pause); !errors.Is(err, factoryservice.ErrInterventionUnavailable) {
		t.Fatal("missing authority admitted pause", err)
	}
	if snapshot() != before {
		t.Fatal("refused interventions changed runs, dispatch, bindings or command journals")
	}
	if err := fx.db.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := store.Open(fx.dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = reopened.Close() })
	application.Store, application.Writer = reopened, &control.Coordinator{Store: reopened}
	result, err = application.Repository(ctx, actor, principal, repository, pause)
	again, ok := result.(factory.PauseResult)
	if err != nil || !ok || again.Withdrawal.Revision != receipt.Withdrawal.Revision || again.Withdrawal.ClosedBy != principal {
		t.Fatal("store reopen lost recorded pause", result, err)
	}
	allowed = false
	if _, err := application.Run(ctx, actor, principal, runID, stop); !errors.Is(err, denied) {
		t.Fatal("reopen bypassed current runtime admission", err)
	}
	allowed = true
	var admitted atomic.Bool
	admitted.Store(true)
	var currentCalls, loseAt atomic.Int64
	transport := factoryhttpapi.Runs{Store: reopened, Commands: application,
		Host: fx.api.FountainIdentities.HostID(), Actor: actor, Principal: principal,
		Visibility: func(ctx context.Context, selected collaboration.ActorID, repo collaboration.RepositoryID) error {
			if selected != actor || repo != repository {
				t.Fatal("HTTP action changed bound scope", selected, repo)
			}
			if !allowed {
				return denied
			}
			return ctx.Err()
		},
		Current: func(w http.ResponseWriter, r *http.Request) bool {
			call := currentCalls.Add(1)
			if !admitted.Load() || call == loseAt.Load() || r.Header.Get("Authorization") != "Bearer declared-intervention-client" {
				w.WriteHeader(http.StatusUnauthorized)
				return false
			}
			return true
		},
		AuthorityError:  func(w http.ResponseWriter, err error) { w.WriteHeader(http.StatusForbidden) },
		VisibilityError: func(w http.ResponseWriter, err error) { w.WriteHeader(http.StatusForbidden) },
	}
	server := httptest.NewServer(transport.Handler())
	t.Cleanup(server.Close)
	call := func(target *httptest.Server, method, path, data string, expected int) []byte {
		t.Helper()
		currentCalls.Store(0)
		req, err := http.NewRequestWithContext(ctx, method, target.URL+path, strings.NewReader(data))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Authorization", "Bearer declared-intervention-client")
		req.Header.Set("Content-Type", "application/json")
		res, err := target.Client().Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		body, err := io.ReadAll(res.Body)
		if err != nil || res.StatusCode != expected || expected == 200 && res.Header.Get("Cache-Control") != "no-store" {
			t.Fatal("shared intervention HTTP", method, path, res.StatusCode, string(body), err)
		}
		return body
	}
	repoPath := "/api/repositories/9223372036854775807/factory/actions"
	runPath := "/api/factory/runs/" + runID + "/actions"
	pauseJSON := `{"action":"pause"}`
	expectedWithdrawal, _ := json.Marshal(again.Withdrawal)
	for range 2 {
		body := call(server, http.MethodPost, repoPath, pauseJSON, 200)
		var saved factory.PauseResult
		if err := json.Unmarshal(body, &saved); err != nil {
			t.Fatal(err)
		}
		actualWithdrawal, _ := json.Marshal(saved.Withdrawal)
		if string(actualWithdrawal) != string(expectedWithdrawal) || !saved.Paused {
			t.Fatal("HTTP lost retained pause or bound principal", string(body))
		}
	}
	var confirmed control.StopResult
	body := call(server, http.MethodPost, runPath, `{"action":"stop"}`, 200)
	if json.Unmarshal(body, &confirmed) != nil || confirmed != stopped {
		t.Fatal("HTTP changed retained stop outcome", string(body))
	}
	call(server, http.MethodPost, repoPath, `{"action":"resume"}`, 409)
	call(server, http.MethodPost, repoPath, `{"action":"launch"}`, 400)
	call(server, http.MethodPost, repoPath, `{"action":"pause","principal":"other"}`, 400)
	call(server, http.MethodPost, repoPath, pauseJSON+pauseJSON, 400)
	call(server, http.MethodPost, repoPath, `{"action":"`+strings.Repeat("x", 65536)+`"}`, 413)
	call(server, http.MethodPost, "/api/repositories/09223372036854775807/factory/actions", pauseJSON, 404)
	call(server, http.MethodGet, repoPath, "", 405)
	call(server, http.MethodPost, runPath, `{"action":"retry"}`, 400)
	member = "" // The declared current peer withdraws the recovery destination.
	call(server, http.MethodPost, runPath, `{"action":"takeover"}`, 403)
	call(server, http.MethodPost, "/api/factory/runs/"+strings.Repeat("b", 32)+"/actions", `{"action":"stop"}`, 503)
	call(server, http.MethodPost, "/api/factory/runs/"+factory.NewID()+"/actions", `{"action":"stop"}`, 404)
	allowed = false
	call(server, http.MethodPost, repoPath, pauseJSON, 403)
	call(server, http.MethodPost, runPath, `{"action":"stop"}`, 403)
	allowed = true
	loseAt.Store(2) // Account withdrawal after decoding precedes any command.
	call(server, http.MethodPost, repoPath, pauseJSON, 401)
	call(server, http.MethodPost, runPath, `{"action":"stop"}`, 401)
	loseAt.Store(0)
	admitted.Store(false)
	call(server, http.MethodPost, repoPath, pauseJSON, 401)
	admitted.Store(true)
	missingPrincipal := transport
	missingPrincipal.Principal = ""
	missingServer := httptest.NewServer(missingPrincipal.Handler())
	t.Cleanup(missingServer.Close)
	call(missingServer, http.MethodPost, repoPath, pauseJSON, 503)
	if snapshot() != before {
		t.Fatal("HTTP interventions changed retained runs/dispatch/commands or created a Project")
	}
	var projects int
	if err := fx.raw.QueryRowContext(ctx, `SELECT count(*) FROM projects`).Scan(&projects); err != nil || projects != 0 {
		t.Fatal("shared intervention required or provisioned native Project", projects, err)
	}
}

func TestSharedRepositoryAccessRead(t *testing.T) {
	ctx := t.Context()
	fx, call := journeyMux(t, nil)
	const actor collaboration.ActorID = 9007199254740993
	const repository collaboration.RepositoryID = 7
	for index, connection := range []string{"foreign", "owned", "unused"} {
		sponsorship := factory.Sponsorship{Repository: int64(repository), Connection: connection, GrantedBy: int64(actor), ActorID: 3,
			Generation: 1, GrantID: "declared-" + connection, ProjectID: "p" + strings.Repeat("7", 24),
			Roles: []string{project.RoleCoder}, AllowanceMinutes: 60 + index, Active: true}
		if err := fx.db.SaveSponsorship(ctx, sponsorship); err != nil {
			t.Fatal(err)
		}
	}
	started := time.Unix(1700000000, 0)
	for index, scope := range []struct {
		repository int64
		connection string
		minutes    int
	}{
		{7, "owned", 3}, {8, "owned", 9}, {7, "foreign", 4},
	} {
		usage := factory.Usage{RunID: strings.Repeat(strconv.Itoa(index+1), 32), Repository: scope.repository, Connection: scope.connection,
			Minutes: scope.minutes, StartedAt: started, EndedAt: started.Add(time.Duration(scope.minutes) * time.Minute)}
		if err := fx.db.RecordRunUsage(ctx, usage); err != nil {
			t.Fatal(err)
		}
	}
	snapshot := func() string {
		t.Helper()
		var data string
		if err := fx.raw.QueryRowContext(ctx, `SELECT jsonb_build_array(
		 (SELECT jsonb_agg(to_jsonb(s) ORDER BY repository,connection) FROM factory_sponsorships s),
		 (SELECT jsonb_agg(to_jsonb(u) ORDER BY run) FROM factory_usage u),
		 (SELECT count(*) FROM factory_runs),(SELECT count(*) FROM projects))::text`).Scan(&data); err != nil {
			t.Fatal(err)
		}
		return data
	}
	before := snapshot()
	viewer := factoryservice.RepositoryViewer{CodeWrite: true, OwnerOrAdmin: true, EnvironmentOwner: true, SessionCurrent: true}
	owned := map[string]bool{"owned": true, "unused": true}
	identityDown := errors.New("declared current ownership is unavailable")
	var viewerErr, ownershipErr, preparationErr error
	ownershipErr = identityDown
	var preparation *factoryservice.RepositoryPreparation
	checks, denyAt := 0, 0
	denied := errors.New("declared repository access denied")
	runtime := &control.Coordinator{Store: fx.db}
	assertScope := func(admitted collaboration.ActorID, repo collaboration.RepositoryID) {
		if admitted != actor || repo != repository {
			t.Fatal("repository access changed canonical subject", admitted, repo)
		}
	}
	reader := factoryservice.RepositoryAccess{Store: fx.db,
		Visibility: func(ctx context.Context, admitted collaboration.ActorID, repo collaboration.RepositoryID) error {
			assertScope(admitted, repo)
			checks++
			if checks == denyAt {
				return denied
			}
			return nil
		},
		Viewer: func(ctx context.Context, admitted collaboration.ActorID, repo collaboration.RepositoryID) (factoryservice.RepositoryViewer, error) {
			assertScope(admitted, repo)
			return viewer, viewerErr
		},
		Connections: func(ctx context.Context, admitted collaboration.ActorID, repo collaboration.RepositoryID) (map[string]bool, error) {
			assertScope(admitted, repo)
			return owned, ownershipErr
		},
		Preparation: func(ctx context.Context, admitted collaboration.ActorID, repo collaboration.RepositoryID) (*factoryservice.RepositoryPreparation, error) {
			assertScope(admitted, repo)
			return preparation, preparationErr
		},
		Authority: func(ctx context.Context, admitted collaboration.ActorID, repo collaboration.RepositoryID) (factory.EffectiveAuthority, error) {
			assertScope(admitted, repo)
			return runtime.EffectiveAuthority(ctx, int64(repo))
		},
	}
	read := func() *factoryservice.RepositoryAccessView {
		t.Helper()
		checks = 0
		view, err := reader.Read(ctx, actor, repository)
		if err != nil {
			t.Fatal(err)
		}
		return view
	}
	view := read()
	for _, usage := range view.Usage {
		if usage.Status != "identity_unavailable" || usage.UsedMinutes != nil || usage.BudgetMinutes != 0 {
			t.Fatal("partial ownership facts exposed private usage", usage)
		}
	}
	code, body := call(http.MethodGet, "/api/repositories/7/factory/access", "")
	if code != 200 {
		t.Fatalf("native access: %d %s", code, body)
	}
	var native factoryservice.RepositoryAccessView
	if err := json.Unmarshal(body, &native); err != nil {
		t.Fatal(err)
	}
	view.ObservedUnix, native.ObservedUnix = 0, 0
	serviceJSON, _ := json.Marshal(view)
	nativeJSON, _ := json.Marshal(native)
	if string(serviceJSON) != string(nativeJSON) {
		t.Fatalf("native/shared access diverged: shared=%s native=%s", serviceJSON, nativeJSON)
	}
	if code, _ := call(http.MethodGet, "/api/repositories/7/factory/access?extra=1", ""); code != 400 {
		t.Fatal("access query accepted", code)
	}
	if code, _ := call(http.MethodGet, "/api/repositories/8/factory/access", ""); code != 404 {
		t.Fatal("foreign access scope accepted", code)
	}
	ownershipErr = nil
	view = read()
	if len(view.Usage) != 3 || len(view.Editable.Sponsorships) != 3 {
		t.Fatal("missing sponsored connections", view)
	}
	for index, usage := range view.Usage {
		if usage.Connection == "foreign" {
			if usage.Status != "connection_owner_only" || usage.UsedMinutes != nil || usage.BudgetMinutes != 0 || view.Editable.Sponsorships[index].Editable {
				t.Fatal("canonical sponsorship was mistaken for current connection ownership", usage)
			}
			continue
		}
		want := 3
		if usage.Connection == "unused" {
			want = 0
		}
		if usage.Status != "" || usage.UsedMinutes == nil || *usage.UsedMinutes != want || usage.BudgetMinutes < 60 || !view.Editable.Sponsorships[index].Editable {
			t.Fatal("owned usage lost exact scope or confirmed zero", usage)
		}
	}
	// Exercise actual SQL failure: missing usage data must not render as zero.
	if _, err := fx.raw.ExecContext(ctx, `ALTER TABLE factory_usage RENAME TO repository_access_unavailable_usage`); err != nil {
		t.Fatal(err)
	}
	view = read()
	for _, usage := range view.Usage {
		if usage.Connection != "foreign" && (usage.Status != "usage_unavailable" || usage.UsedMinutes != nil || usage.BudgetMinutes != 0) {
			t.Fatal("unknown SQL usage rendered as a zero or budget", usage)
		}
	}
	if _, err := fx.raw.ExecContext(ctx, `ALTER TABLE repository_access_unavailable_usage RENAME TO factory_usage`); err != nil {
		t.Fatal(err)
	}
	viewer.OwnerOrAdmin, viewer.EnvironmentOwner, viewer.Operator = false, false, true
	view = read()
	if view.Editable.Policy.Editable || view.Editable.EnvironmentGrant.Editable || !view.Editable.Capacity.Editable || !view.Editable.OperatorGrant.Editable {
		t.Fatal("runtime operation conferred repository ownership", view.Editable)
	}
	viewer.SessionCurrent = false
	view = read()
	if view.Editable.Capacity.Editable || view.Editable.Capacity.Reason != "session_changed" || view.Editable.Sponsorships[1].Editable || view.Editable.Sponsorships[1].Reason != "session_changed" {
		t.Fatal("stale session retained editability", view.Editable)
	}
	viewer.SessionCurrent = true
	preparation = &factoryservice.RepositoryPreparation{Project: "declared-runtime", Ready: false}
	if view = read(); view.Preparation == nil || view.Preparation.Ready {
		t.Fatal("known incomplete runtime preparation became ready", view.Preparation)
	}
	preparationErr = errors.New("declared preparation read failed")
	if view, err := reader.Read(ctx, actor, repository); view != nil || !errors.Is(err, preparationErr) {
		t.Fatal("unknown preparation returned partial access", view, err)
	}
	preparationErr, preparation = nil, nil
	viewerErr = errors.New("declared current viewer facts failed")
	if view, err := reader.Read(ctx, actor, repository); view != nil || !errors.Is(err, viewerErr) {
		t.Fatal("unknown viewer facts returned access", view, err)
	}
	viewerErr = nil
	checks, denyAt = 0, 2
	if view, err := reader.Read(ctx, actor, repository); view != nil || !errors.Is(err, denied) {
		t.Fatal("late visibility denial returned private access", view, err)
	}
	denyAt = 0
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if view, err := reader.Read(cancelled, actor, repository); view != nil || !errors.Is(err, context.Canceled) {
		t.Fatal("cancelled access read", view, err)
	}
	missing := reader
	missing.Authority = nil
	if view, err := missing.Read(ctx, actor, repository); view != nil || !errors.Is(err, factoryservice.ErrRepositoryAccessUnavailable) {
		t.Fatal("missing runtime authority supplied access", view, err)
	}
	if view, err := reader.Read(ctx, 0, repository); view != nil || !errors.Is(err, factoryservice.ErrRepositoryAccessInvalid) {
		t.Fatal("invalid canonical viewer accepted", view, err)
	}
	if snapshot() != before {
		t.Fatal("access reads changed recorded usage, sponsorships, runs or Projects")
	}
	retained := read()
	retained.ObservedUnix = 0
	beforeJSON, _ := json.Marshal(retained)
	if err := fx.db.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := store.Open(fx.dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = reopened.Close() })
	reader.Store, runtime.Store = reopened, reopened
	retained = read()
	retained.ObservedUnix = 0
	afterJSON, _ := json.Marshal(retained)
	if string(beforeJSON) != string(afterJSON) || snapshot() != before {
		t.Fatal("store reopen changed access or recorded facts")
	}
	// The standalone-facing read path consumes these exact reopened services.
	status := factoryservice.RepositoryStatus{Store: reopened, Visibility: reader.Visibility, Preparation: reader.Preparation, Authority: reader.Authority,
		Bindings: func(ctx context.Context, admitted collaboration.ActorID, repo collaboration.RepositoryID) ([]factory.ActorBindingRef, error) {
			assertScope(admitted, repo)
			// This declared adapter has no enrolled provider operation bindings.
			return []factory.ActorBindingRef{}, ctx.Err()
		}}
	transport := factoryhttpapi.Repositories{Store: reopened, Status: status, Access: reader,
		Host: fx.api.FountainIdentities.HostID(), Actor: actor, Visibility: reader.Visibility,
		Current: func(w http.ResponseWriter, r *http.Request) bool {
			if r.Header.Get("Authorization") != "Bearer declared-access-client" {
				http.Error(w, "declared account admission unavailable", 401)
				return false
			}
			return true
		}}
	server := httptest.NewServer(transport.Handler())
	defer server.Close()
	base := "/api/repositories/7/factory"
	for _, path := range []string{base, base + "/access"} {
		r, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+path, nil)
		if err != nil {
			t.Fatal(err)
		}
		r.Header.Set("Authorization", "Bearer declared-access-client")
		response, err := server.Client().Do(r)
		if err != nil {
			t.Fatal(err)
		}
		body, readErr := io.ReadAll(response.Body)
		_ = response.Body.Close()
		if readErr != nil || response.StatusCode != 200 || response.Header.Get("Cache-Control") != "no-store" {
			t.Fatal("direct repository read failed", path, response.StatusCode, string(body), readErr)
		}
		if path == base+"/access" {
			var served factoryservice.RepositoryAccessView
			if err := json.Unmarshal(body, &served); err != nil {
				t.Fatal(err)
			}
			served.ObservedUnix = 0
			encoded, _ := json.Marshal(served)
			if string(encoded) != string(beforeJSON) {
				t.Fatal("direct HTTP changed canonical repository access", string(encoded), string(beforeJSON))
			}
		}
	}
	if snapshot() != before {
		t.Fatal("direct repository HTTP read changed retained records")
	}
}

func TestRepositoryAccessUsesEnvironmentOwner(t *testing.T) {
	_, call := journeyMux(t, nil)
	code, body := call(http.MethodGet, "/api/repositories/7/factory/access", "")
	if code != 200 {
		t.Fatalf("repository access: %d %s", code, body)
	}
	var wire struct {
		Required []struct {
			Code      string `json:"code"`
			ResolveBy string `json:"resolve_by"`
			Setting   string `json:"setting"`
		} `json:"required_actions"`
	}
	if err := json.Unmarshal(body, &wire); err != nil {
		t.Fatal(err)
	}
	for _, action := range wire.Required {
		if action.Code == factory.MissingEnvironment {
			if action.ResolveBy != "repository_owner" || action.Setting != "environment_grant" {
				t.Fatalf("environment permission points at someone who cannot save it: %+v", action)
			}
			return
		}
	}
	t.Fatal("missing environment permission was not reported", string(body))
}

func TestRepositoryStatusCountsRecordedDispatchWork(t *testing.T) {
	ctx := t.Context()
	fx, call := journeyMux(t, nil)
	active := journeyOwner(t, strings.Repeat("3", 32), "declared-acceptance", project.ExecutionImplement, factory.AssignmentAssigned, "", "")
	active.Issue = 4
	insertJourneyRow(t, fx.raw, active, factory.AssignmentAssigned)
	foreign := active
	foreign.ID, foreign.AttemptRoot, foreign.PublicationAssignment, foreign.Repository = strings.Repeat("4", 32), strings.Repeat("4", 32), strings.Repeat("4", 32), 8
	insertJourneyRow(t, fx.raw, foreign, factory.AssignmentAssigned)
	if _, err := fx.raw.ExecContext(ctx, `INSERT INTO issue_controls(repository,issue,revision,data)
 SELECT 7,n,1,jsonb_build_object('repository','7','issue',n::text,'readiness','queued') FROM generate_series(1000,3059) n`); err != nil {
		t.Fatal(err)
	}
	code, body := call(http.MethodGet, "/api/repositories/7/factory", "")
	if code != 200 {
		t.Fatalf("repository status: %d %s", code, body)
	}
	var view factoryservice.RepositoryStatusView
	if err := json.Unmarshal(body, &view); err != nil {
		t.Fatal(err)
	}
	if view.Queue.Active != 1 || view.Queue.Queued != 2060 {
		t.Fatalf("repository dispatch counts lost recorded work or included another scope: %+v", view.Queue)
	}
	const sponsor collaboration.ActorID = 9007199254740993
	viewer, wanted := sponsor, collaboration.RepositoryID(7)
	checks, denyAt := 0, 0
	denied := errors.New("declared current repository status visibility denied")
	var preparationErr, bindingErr, diagnosticErr error
	var bindings []factory.ActorBindingRef
	var preparation *factoryservice.RepositoryPreparation
	admin := false
	runtime := &control.Coordinator{Store: fx.db}
	assertScope := func(actor collaboration.ActorID, repository collaboration.RepositoryID) {
		if actor != viewer || repository != wanted {
			t.Fatal("repository status authority subject changed", actor, repository)
		}
	}
	reader := factoryservice.RepositoryStatus{Store: fx.db,
		Visibility: func(ctx context.Context, actor collaboration.ActorID, repository collaboration.RepositoryID) error {
			assertScope(actor, repository)
			checks++
			if checks == denyAt {
				return denied
			}
			return nil
		},
		Bindings: func(ctx context.Context, actor collaboration.ActorID, repository collaboration.RepositoryID) ([]factory.ActorBindingRef, error) {
			assertScope(actor, repository)
			return bindings, bindingErr
		},
		Preparation: func(ctx context.Context, actor collaboration.ActorID, repository collaboration.RepositoryID) (*factoryservice.RepositoryPreparation, error) {
			assertScope(actor, repository)
			return preparation, preparationErr
		},
		Authority: func(ctx context.Context, actor collaboration.ActorID, repository collaboration.RepositoryID) (factory.EffectiveAuthority, error) {
			assertScope(actor, repository)
			return runtime.EffectiveAuthority(ctx, int64(repository))
		},
		Diagnostic: func(ctx context.Context, actor collaboration.ActorID, repository collaboration.RepositoryID) (*control.DispatchObservation, error) {
			assertScope(actor, repository)
			if !admin {
				return nil, nil
			}
			return &control.DispatchObservation{Stage: "declared-owner-stage", Waits: []control.DispatchObservationReason{{Repository: int64(repository), Issue: 4, Reason: "declared-owner-wait"}}}, diagnosticErr
		},
	}
	direct, err := reader.Read(ctx, viewer, 7)
	if err != nil {
		t.Fatal(err)
	}
	nativeJSON, _ := json.Marshal(view)
	directJSON, _ := json.Marshal(direct)
	if string(nativeJSON) != string(directJSON) {
		t.Fatalf("direct/native status differs: %s\n%s", directJSON, nativeJSON)
	}
	sp := factory.Sponsorship{Repository: 7, Connection: "declared-sponsor", GrantedBy: int64(sponsor), ActorID: int64(sponsor + 2), Generation: 1, GrantID: "declared-grant", ProjectID: "p" + strings.Repeat("1", 24), Roles: []string{project.RoleCoder}, AllowanceMinutes: 78}
	if err := fx.db.SaveSponsorship(ctx, sp); err != nil {
		t.Fatal(err)
	}
	ownerView, err := reader.Read(ctx, viewer, 7)
	if err != nil || len(ownerView.Sponsorships) != 1 || ownerView.Sponsorships[0].AllowanceMinutes != 78 {
		t.Fatal("canonical sponsor lost its private allowance", ownerView, err)
	}
	viewer = sponsor + 1
	otherView, err := reader.Read(ctx, viewer, 7)
	if err != nil || len(otherView.Sponsorships) != 1 || otherView.Sponsorships[0].AllowanceMinutes != 0 || otherView.Sponsorships[0].Connection != sp.Connection {
		t.Fatal("other viewer received sponsor allowance or lost its reference", otherView, err)
	}
	viewer = sponsor
	bindings = []factory.ActorBindingRef{{TokenID: 31, ActorID: int64(sponsor + 2), Kind: factory.OpRefPublish}}
	preparation = &factoryservice.RepositoryPreparation{Project: "declared-runtime-workspace", Ready: false}
	admin = true
	adminView, err := reader.Read(ctx, viewer, 7)
	if err != nil || len(adminView.NativeBindings) != 1 || adminView.NativeBindings[0].ActorID != int64(sponsor+2) || adminView.Preparation.Project != preparation.Project || adminView.DispatchObservation == nil {
		t.Fatal("runtime status ports lost their canonical facts", adminView, err)
	}
	diagnosticErr = errors.New("declared diagnostic authority unknown")
	unknownDiagnostic, err := reader.Read(ctx, viewer, 7)
	if err != nil || unknownDiagnostic.DispatchObservation != nil {
		t.Fatal("unknown diagnostic authority exposed private observation", unknownDiagnostic, err)
	}
	diagnosticErr = nil
	admin = false
	nonAdmin, err := reader.Read(ctx, viewer, 7)
	if err != nil || nonAdmin.DispatchObservation != nil {
		t.Fatal("ordinary readable status exposed admin observation", nonAdmin, err)
	}
	preparationErr = errors.New("declared runtime preparation unavailable")
	if got, err := reader.Read(ctx, viewer, 7); !errors.Is(err, preparationErr) || got != nil {
		t.Fatal("unknown runtime became absent/ready", got, err)
	}
	preparationErr = nil
	bindingErr = errors.New("declared binding identity unavailable")
	if got, err := reader.Read(ctx, viewer, 7); !errors.Is(err, bindingErr) || got != nil {
		t.Fatal("unknown bindings fabricated a status", got, err)
	}
	bindingErr = nil
	denyAt = checks + 2
	if got, err := reader.Read(ctx, viewer, 7); !errors.Is(err, denied) || got != nil {
		t.Fatal("late visibility denial exposed status", got, err)
	}
	denyAt = 0
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if got, err := reader.Read(cancelled, viewer, 7); !errors.Is(err, context.Canceled) || got != nil {
		t.Fatal("cancelled status exposed facts", got, err)
	}
	if _, err := (factoryservice.RepositoryStatus{Store: fx.db}).Read(ctx, viewer, 7); !errors.Is(err, factoryservice.ErrRepositoryStatusUnavailable) {
		t.Fatal("unwired runtime status was inferred", err)
	}
	wanted = 9223372036854775807
	preparation = nil
	empty, err := reader.Read(ctx, viewer, wanted)
	if err != nil || empty.Repository != "9223372036854775807" || empty.Queue.Active != 0 || empty.Queue.Queued != 0 {
		t.Fatal("canonical empty repository status", empty, err)
	}
	wanted = 7
	stable, err := reader.Read(ctx, viewer, 7)
	if err != nil {
		t.Fatal(err)
	}
	baseline, _ := json.Marshal(stable)
	before := journeyStoreCounts(t, fx.raw)
	if err := fx.db.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := store.Open(fx.dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = reopened.Close() })
	reader.Store, runtime.Store = reopened, reopened
	stable, err = reader.Read(ctx, viewer, 7)
	recovered, _ := json.Marshal(stable)
	if err != nil || string(recovered) != string(baseline) {
		t.Fatalf("reopened repository status changed: %s\n%s\n%v", baseline, recovered, err)
	}
	after := journeyStoreCounts(t, fx.raw)
	for table, count := range before {
		if after[table] != count {
			t.Fatalf("status read changed %s: %d -> %d", table, count, after[table])
		}
	}
	var projects, runs int
	if err := fx.raw.QueryRowContext(ctx, `SELECT (SELECT count(*) FROM projects),(SELECT count(*) FROM factory_runs)`).Scan(&projects, &runs); err != nil || projects != 0 || runs != 0 {
		t.Fatal("status required native Project or launched work", projects, runs, err)
	}
}

func TestWorkReadPriorityMasksEverySelection(t *testing.T) {
	ctx := t.Context()
	fx, call := journeyMux(t, nil)
	const hidden = "hidden-from-private-repository"
	const visible = "visible-repository-candidate"
	receipt := factory.PriorityReceipt{Snapshot: strings.Repeat("f", 64), CreatedUnix: 100, Outcome: "selected", Selected: hidden,
		Candidates: []factory.PriorityCandidate{
			{ID: visible, Control: factory.IssueControl{Repository: 7, Issue: 3}},
			{ID: hidden, Control: factory.IssueControl{Repository: 8, Issue: 9}},
		},
		Calls: []factory.PriorityCall{{Answer: factory.PriorityAnswer{Selected: hidden, Model: "declared-fixture", Confidence: 0.8, Probabilities: map[string]float64{visible: 0.2, hidden: 0.8}}}},
	}
	raw, err := json.Marshal(receipt)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fx.raw.ExecContext(ctx, `INSERT INTO factory_priority_judgments(snapshot,max_calls,data) VALUES($1,1,$2)`, receipt.Snapshot, string(raw)); err != nil {
		t.Fatal(err)
	}
	code, body := call(http.MethodGet, "/api/repositories/7/factory/issues/3/plan", "")
	if code != 200 {
		t.Fatalf("priority comparison read: %d %s", code, body)
	}
	if strings.Contains(string(body), hidden) {
		t.Fatalf("hidden candidate escaped the comparison through a call selection: %s", body)
	}
	older := receipt
	older.Snapshot, older.CreatedUnix, older.Selected = strings.Repeat("e", 64), 90, visible
	older.Calls = []factory.PriorityCall{{Answer: factory.PriorityAnswer{Selected: visible, Model: "declared-fixture", Confidence: 0.8, Probabilities: map[string]float64{visible: 0.8, hidden: 0.2}}}}
	raw, _ = json.Marshal(older)
	if _, err := fx.raw.ExecContext(ctx, `INSERT INTO factory_priority_judgments(snapshot,max_calls,data) VALUES($1,1,$2)`, older.Snapshot, string(raw)); err != nil {
		t.Fatal(err)
	}
	const actor collaboration.ActorID = 9007199254740993
	wanted := collaboration.RepositoryID(7)
	checks, denyAt := 0, 0
	allowed := true
	denied := errors.New("declared recorded work visibility revoked")
	reader := factoryservice.WorkReads{Store: fx.db, Visibility: func(ctx context.Context, owner collaboration.ActorID, repository collaboration.RepositoryID) error {
		checks++
		if owner != actor || repository != wanted {
			t.Fatal("work read scope changed", owner, repository)
		}
		if !allowed || checks == denyAt {
			return denied
		}
		return nil
	}}
	first, err := reader.Priority(ctx, actor, 7, 3, store.PriorityComparisonCursor{}, 1)
	if err != nil || len(first.Comparisons) != 1 || first.NextBeforeUnix != 100 || first.NextBeforeSnapshot != receipt.Snapshot || first.Comparisons[0].Selected != "" || first.Comparisons[0].Calls[0].Selected != "" || first.Comparisons[0].OtherCount != 1 || first.Comparisons[0].CandidatesTotal != 2 || first.Comparisons[0].ProbabilityN != 2 || first.Comparisons[0].ProbabilitiesFull {
		t.Fatal("private comparison/page metadata", first, err)
	}
	cursor := store.PriorityComparisonCursor{BeforeUnix: first.NextBeforeUnix, BeforeSnapshot: first.NextBeforeSnapshot}
	next, err := reader.Priority(ctx, actor, 7, 3, cursor, 1)
	if err != nil || len(next.Comparisons) != 1 || next.Comparisons[0].Snapshot != older.Snapshot || next.Comparisons[0].Selected != visible || next.Comparisons[0].Calls[0].Selected != visible || next.NextBeforeSnapshot != "" {
		t.Fatal("journal continuation or visible choice changed", next, err)
	}
	for _, limit := range []int{0, store.MaxPriorityComparisons + 1} {
		if _, err := reader.Priority(ctx, actor, 7, 3, cursor, limit); !errors.Is(err, factoryservice.ErrWorkReadQuery) {
			t.Fatal("unbounded priority read", err)
		}
	}
	if _, err := reader.Priority(ctx, actor, 7, 3, store.PriorityComparisonCursor{BeforeUnix: -1}, 1); !errors.Is(err, factoryservice.ErrWorkReadQuery) {
		t.Fatal("invalid cursor became a fresh page", err)
	}
	wanted = 9223372036854775807
	empty, err := reader.Priority(ctx, actor, wanted, 9223372036854775807, store.PriorityComparisonCursor{}, 1)
	if err != nil || empty.Repository != "9223372036854775807" || empty.Issue != "9223372036854775807" || len(empty.Comparisons) != 0 {
		t.Fatal("canonical empty comparison scope", empty, err)
	}
	wanted = 7
	if _, err := reader.Assignment(ctx, actor, 7, 3); !errors.Is(err, store.ErrNotFound) {
		t.Fatal("missing assignment was fabricated", err)
	}
	_, root, _ := seedRichJourney(t, fx, &journeyAcceptanceSource{})
	root.Prompt, root.PromptSHA = []byte("Recorded accepted fixture context"), strings.Repeat("7", 64)
	root.Result = &factory.AssignmentResult{AssignmentID: root.ID, RunID: root.Run, Status: "passed", Summary: "Recorded fixture outcome", Candidate: "refs/heads/fixture", Findings: []string{"Recorded finding"}, Reported: true, RecordedUnix: 123}
	raw, _ = json.Marshal(root)
	if _, err := fx.raw.ExecContext(ctx, `UPDATE factory_assignments SET data=$2 WHERE id=$1`, root.ID, string(raw)); err != nil {
		t.Fatal(err)
	}
	held := factory.Reservation{AssignmentID: root.ID, Repository: 7, Connection: "declared-capacity", State: "held", PlannedMinutes: 3, Revision: 1}
	raw, _ = json.Marshal(held)
	if _, err := fx.raw.ExecContext(ctx, `INSERT INTO factory_reservations(assignment,repository,connection,state,data) VALUES($1,7,$2,'held',$3)`, root.ID, held.Connection, string(raw)); err != nil {
		t.Fatal(err)
	}
	before := journeyStoreCounts(t, fx.raw)
	assignment, err := reader.Assignment(ctx, actor, 7, 3)
	if err != nil || assignment.ID != root.ID || assignment.Prompt != string(root.Prompt) || assignment.Result == nil || assignment.Result.Summary != root.Result.Summary || assignment.Reservation == nil || assignment.Reservation.PlannedMinutes != 3 || assignment.Publication == nil || assignment.Publication.PRNumber != "12" {
		t.Fatal("exact recorded assignment view", assignment, err)
	}
	assignmentBaseline, _ := json.Marshal(assignment)
	code, body = call(http.MethodGet, "/api/repositories/7/factory/issues/3/assignment", "")
	if code != 200 {
		t.Fatalf("assignment HTTP: %d %s", code, body)
	}
	var servedAssignment factoryservice.AssignmentView
	if err := json.Unmarshal(body, &servedAssignment); err != nil {
		t.Fatal(err)
	}
	servedBytes, _ := json.Marshal(servedAssignment)
	if string(servedBytes) != string(assignmentBaseline) {
		t.Fatalf("assignment direct/HTTP differs: %s\n%s", assignmentBaseline, servedBytes)
	}
	if _, err := fx.raw.ExecContext(ctx, `UPDATE factory_reservations SET data=jsonb_set(data,'{repository}','"8"') WHERE assignment=$1`, root.ID); err != nil {
		t.Fatal(err)
	}
	if got, err := reader.Assignment(ctx, actor, 7, 3); !errors.Is(err, factoryservice.ErrWorkJourneyScope) || got != nil {
		t.Fatal("foreign capacity joined by assignment alone", got, err)
	}
	raw, _ = json.Marshal(held)
	if _, err := fx.raw.ExecContext(ctx, `UPDATE factory_reservations SET data=$2 WHERE assignment=$1`, root.ID, string(raw)); err != nil {
		t.Fatal(err)
	}
	allowed = false
	if got, err := reader.Assignment(ctx, actor, 7, 3); !errors.Is(err, denied) || got != nil {
		t.Fatal("denied read exposed recorded prompt", got, err)
	}
	if got, err := reader.Priority(ctx, actor, 7, 3, cursor, 1); !errors.Is(err, denied) || got != nil {
		t.Fatal("denied read exposed comparisons", got, err)
	}
	allowed, denyAt = true, checks+2
	if got, err := reader.Assignment(ctx, actor, 7, 3); !errors.Is(err, denied) || got != nil {
		t.Fatal("late denial exposed assignment", got, err)
	}
	denyAt = checks + 2
	if got, err := reader.Priority(ctx, actor, 7, 3, cursor, 1); !errors.Is(err, denied) || got != nil {
		t.Fatal("late denial exposed comparisons", got, err)
	}
	denyAt = 0
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := reader.Priority(cancelled, actor, 7, 3, cursor, 1); !errors.Is(err, context.Canceled) {
		t.Fatal("cancelled priority read", err)
	}
	if _, err := (factoryservice.WorkReads{Store: fx.db}).Assignment(ctx, actor, 7, 3); !errors.Is(err, factoryservice.ErrWorkReadUnavailable) {
		t.Fatal("unwired prompt visibility", err)
	}
	retainedPriority, err := reader.Priority(ctx, actor, 7, 3, cursor, 1)
	if err != nil {
		t.Fatal(err)
	}
	retainedPriority.ObservedUnix = 0
	priorityBaseline, _ := json.Marshal(retainedPriority)
	if err := fx.db.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := store.Open(fx.dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = reopened.Close() })
	reader.Store = reopened
	assignment, err = reader.Assignment(ctx, actor, 7, 3)
	recovered, _ := json.Marshal(assignment)
	if err != nil || string(recovered) != string(assignmentBaseline) {
		t.Fatal("reopened assignment changed", assignment, err)
	}
	retainedPriority, err = reader.Priority(ctx, actor, 7, 3, cursor, 1)
	if err != nil {
		t.Fatal(err)
	}
	retainedPriority.ObservedUnix = 0
	recovered, _ = json.Marshal(retainedPriority)
	if string(recovered) != string(priorityBaseline) {
		t.Fatalf("reopened priority page changed: %s\n%s", priorityBaseline, recovered)
	}
	after := journeyStoreCounts(t, fx.raw)
	for table, count := range before {
		if after[table] != count {
			t.Fatalf("recorded work reads changed %s: %d -> %d", table, count, after[table])
		}
	}
}

func TestSharedWorkControlCapabilities(t *testing.T) {
	ctx := t.Context()
	fake := &journeyAcceptanceSource{}
	fx, _ := journeyMux(t, fake)
	const actor collaboration.ActorID = 9007199254740993
	wanted := collaboration.RepositoryID(7)
	checks, denyAt := 0, 0
	member := true
	authority := factoryservice.WorkAuthority{SessionCurrent: true}
	denied := errors.New("declared current work visibility denied")
	reader := factoryservice.WorkControls{Store: fx.db,
		Visibility: func(ctx context.Context, owner collaboration.ActorID, repository collaboration.RepositoryID) error {
			checks++
			if owner != actor || repository != wanted {
				t.Fatal("control visibility subject changed", owner, repository)
			}
			if checks == denyAt {
				return denied
			}
			return nil
		},
		Authority: func(ctx context.Context, owner collaboration.ActorID, repository collaboration.RepositoryID) (factoryservice.WorkAuthority, error) {
			if owner != actor || repository != wanted {
				t.Fatal("control authority subject changed", owner, repository)
			}
			return authority, nil
		},
		Membership: func(ctx context.Context, owner collaboration.ActorID, repository collaboration.RepositoryID, runtimeProject string) (bool, error) {
			if owner != actor || repository != wanted || runtimeProject != "p"+strings.Repeat("1", 24) {
				t.Fatal("runtime membership subject changed", owner, repository, runtimeProject)
			}
			return member, nil
		},
	}
	if _, err := (factoryservice.WorkControls{Store: fx.db}).Read(ctx, actor, 7, 3); !errors.Is(err, factoryservice.ErrWorkControlUnavailable) {
		t.Fatal("unwired runtime was treated as admitted", err)
	}
	view, err := reader.Read(ctx, actor, 7, 3)
	if err != nil || !view.Capabilities.Watch.Available || view.Capabilities.Stop.Available || view.Capabilities.Retry.Reason != "no_run" {
		t.Fatal("no-attempt control view", view, err)
	}
	wanted = 9223372036854775807
	if _, err := reader.Read(ctx, actor, wanted, 9223372036854775807); err != nil {
		t.Fatal("canonical maximum-int64 control scope", err)
	}
	wanted = 7
	acceptance, root, _ := seedRichJourney(t, fx, fake)
	run := factory.Run{ID: root.Run, ProjectID: root.ProjectID, Role: root.Role}
	raw, _ := json.Marshal(run)
	if _, err := fx.raw.ExecContext(ctx, `INSERT INTO factory_runs(id,active,settled,data) VALUES($1,true,false,$2)`, run.ID, string(raw)); err != nil {
		t.Fatal(err)
	}
	view, err = reader.Read(ctx, actor, 7, 3)
	if err != nil || !view.Capabilities.Stop.Available || view.Capabilities.Retry.Available || view.Capabilities.Takeover.Available || view.Capabilities.Retry.Reason != "run_unsettled" {
		t.Fatal("live run control gates", view, err)
	}
	run.Reconciled, run.Outcome = true, factory.Failed
	raw, _ = json.Marshal(run)
	if _, err := fx.raw.ExecContext(ctx, `UPDATE factory_runs SET active=false,settled=true,data=$2 WHERE id=$1`, run.ID, string(raw)); err != nil {
		t.Fatal(err)
	}
	view, err = reader.Read(ctx, actor, 7, 3)
	if err != nil || view.Capabilities.Stop.Available || !view.Capabilities.Retry.Available || !view.Capabilities.Takeover.Available {
		t.Fatal("settled member controls", view, err)
	}
	member = false
	view, err = reader.Read(ctx, actor, 7, 3)
	if err != nil || view.Capabilities.Takeover.Available || view.Capabilities.Takeover.Reason != "membership_required" {
		t.Fatal("non-member takeover gate", view, err)
	}
	member = true
	authority.Reason = factoryservice.WorkWriteRequired
	view, err = reader.Read(ctx, actor, 7, 3)
	if err != nil || !view.Capabilities.Watch.Available || view.Capabilities.Stop.Available || view.Capabilities.Retry.Available || view.Capabilities.Takeover.Available || view.Capabilities.Pause.Available || view.Capabilities.Resume.Available {
		t.Fatal("read-only watching acquired writes or was denied", view, err)
	}
	authority.SessionCurrent = false
	view, err = reader.Read(ctx, actor, 7, 3)
	if err != nil || view.Capabilities.Watch.Available || view.Capabilities.Watch.Reason != "session_changed" {
		t.Fatal("stale session could watch", view, err)
	}
	authority = factoryservice.WorkAuthority{SessionCurrent: true}
	// Declared persisted retry facts exercise readback only. Actual retry
	// command admission/dispatch remains with its existing lifecycle owner.
	root.Outcome = factory.Failed
	raw, _ = json.Marshal(root)
	if _, err := fx.raw.ExecContext(ctx, `UPDATE factory_assignments SET data=$2 WHERE id=$1`, root.ID, string(raw)); err != nil {
		t.Fatal(err)
	}
	controlRow := factory.IssueControl{Repository: 7, Issue: 3, Acceptance: acceptance}
	raw, _ = json.Marshal(controlRow)
	if _, err := fx.raw.ExecContext(ctx, `INSERT INTO issue_controls(repository,issue,revision,data) VALUES(7,3,1,$1) ON CONFLICT(repository,issue) DO UPDATE SET data=excluded.data`, string(raw)); err != nil {
		t.Fatal(err)
	}
	if _, _, err := fx.db.RecordFactoryRunView(ctx, factory.RunView{RunID: run.ID, Repository: 7, Issue: 3, Attempt: root.ID}); err != nil {
		t.Fatal(err)
	}
	retryID := factory.NewID()
	payload, _ := json.Marshal(map[string]string{"prior_run": run.ID, "prior_outcome": string(factory.Failed)})
	if _, err := fx.raw.ExecContext(ctx, `INSERT INTO factory_commands(id,type,target,principal,digest,payload,created,finished) VALUES($1,$2,$3,'profile:owner',$4,$5,'2026-11-01T00:00:00Z','2026-11-01T00:00:00Z')`, retryID, factory.CommandRetry, "run/"+run.ID, factory.SettingsDigest(factory.CommandRetry, "run/"+run.ID, string(payload)), string(payload)); err != nil {
		t.Fatal(err)
	}
	before := journeyStoreCounts(t, fx.raw)
	view, err = reader.Read(ctx, actor, 7, 3)
	if err != nil || view.Requested == nil || view.Requested.Retry.CommandID != retryID || view.Requested.Retry.Prior != run.ID || !view.Requested.Retry.Queued || view.Capabilities.Retry.Available || view.Capabilities.Retry.Reason != "retry_queued" {
		t.Fatal("queued retry was hidden or confirmed as execution", view, err)
	}
	if _, err := fx.db.WithdrawDispatch(ctx, 7, "declared pause", "profile:owner"); err != nil {
		t.Fatal(err)
	}
	view, err = reader.Read(ctx, actor, 7, 3)
	if err != nil || view.Capabilities.Pause.Available || !view.Capabilities.Resume.Available {
		t.Fatal("closed dispatch controls", view, err)
	}
	denyAt = checks + 2
	if view, err := reader.Read(ctx, actor, 7, 3); !errors.Is(err, denied) || view.Requested != nil || view.Capabilities.Watch.Available {
		t.Fatal("late denial leaked controls", view, err)
	}
	denyAt = 0
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := reader.Read(cancelled, actor, 7, 3); !errors.Is(err, context.Canceled) {
		t.Fatal("cancelled capability read", err)
	}
	for table, count := range before {
		if after := journeyStoreCounts(t, fx.raw)[table]; after != count {
			t.Fatalf("capability reads changed %s: %d -> %d", table, count, after)
		}
	}
}

func TestSharedWorkJourneyRead(t *testing.T) {
	ctx := t.Context()
	fake := &journeyAcceptanceSource{}
	fx, call := journeyMux(t, fake)
	_, root, _ := seedRichJourney(t, fx, fake)
	const actor collaboration.ActorID = 9007199254740993
	wanted := collaboration.RepositoryID(7)
	checks, denyAt := 0, 0
	allowed := true
	denied := errors.New("declared current issue-work visibility withdrawn")
	reader := factoryservice.WorkJourneys{Store: fx.db, Visibility: func(ctx context.Context, owner collaboration.ActorID, repository collaboration.RepositoryID) error {
		checks++
		if owner != actor || repository != wanted {
			t.Fatal("journey authority subject changed", owner, repository)
		}
		if !allowed || checks == denyAt {
			return denied
		}
		return nil
	}}
	statusReader := &control.Coordinator{Store: fx.db, AcceptanceReads: fake}
	var statusErr error
	foreignStatus := false
	issueReader := factoryservice.Issues{Store: fx.db, Visibility: reader.Visibility,
		Status: func(ctx context.Context, owner collaboration.ActorID, repository collaboration.RepositoryID, issue int64) (control.AcceptanceValidity, error) {
			if owner != actor || repository != wanted {
				t.Fatal("current status scope changed", owner, repository)
			}
			if statusErr != nil {
				return control.AcceptanceValidity{}, statusErr
			}
			status, err := statusReader.AcceptanceStatus(ctx, int64(repository), strconv.FormatInt(issue, 10))
			if foreignStatus && status.Acceptance != nil {
				copied := *status.Acceptance
				copied.Repository = 8
				status.Acceptance = &copied
			}
			return status, err
		},
		Authority: func(ctx context.Context, owner collaboration.ActorID, repository collaboration.RepositoryID) (factoryservice.WorkAuthority, error) {
			if owner != actor || repository != wanted {
				t.Fatal("issue execution authority scope changed", owner, repository)
			}
			return factoryservice.WorkAuthority{SessionCurrent: true}, nil
		},
		Membership: func(ctx context.Context, owner collaboration.ActorID, repository collaboration.RepositoryID, runtimeProject string) (bool, error) {
			if owner != actor || repository != wanted || runtimeProject != root.ProjectID {
				t.Fatal("issue runtime scope changed", owner, repository, runtimeProject)
			}
			return false, nil
		},
		Navigation: func(ctx context.Context, owner collaboration.ActorID, repository collaboration.RepositoryID, issue, pr int64) (*factoryservice.IssueLinks, error) {
			if owner != actor || repository != wanted || issue != 3 || pr != 12 {
				t.Fatal("navigation lost recorded issue/PR scope", owner, repository, issue, pr)
			}
			return &factoryservice.IssueLinks{Issue: "/alice/Alpha/issues/3", Pull: "/alice/Alpha/pulls/12"}, nil
		},
	}
	before := journeyStoreCounts(t, fx.raw)
	view, err := reader.Read(ctx, actor, 7, 3)
	if err != nil || view == nil || !view.Available || view.Lineage.AttemptRoot != root.AttemptRoot || checks != 2 {
		t.Fatalf("shared recorded journey: %+v %v checks=%d", view, err, checks)
	}
	view.ObservedUnix = 0
	baseline, err := json.Marshal(view)
	if err != nil {
		t.Fatal(err)
	}
	code, body := call(http.MethodGet, "/api/repositories/7/factory/issues/3", "")
	if code != 200 {
		t.Fatalf("shared HTTP journey: %d %s", code, body)
	}
	served := journeyIssueView(t, body).Journey
	served.ObservedUnix = 0
	encoded, err := json.Marshal(served)
	if err != nil || string(encoded) != string(baseline) {
		t.Fatalf("direct/HTTP journeys differ: %s\n%s\n%v", baseline, encoded, err)
	}
	whole, err := issueReader.Read(ctx, actor, 7, 3)
	if err != nil {
		t.Fatal("shared complete issue read", err)
	}
	whole.ObservedUnix, whole.Journey.ObservedUnix = 0, 0
	fullBaseline, err := json.Marshal(whole)
	if err != nil {
		t.Fatal(err)
	}
	servedWhole := journeyIssueView(t, body)
	servedWhole.ObservedUnix, servedWhole.Journey.ObservedUnix = 0, 0
	fullHTTP, err := json.Marshal(servedWhole)
	if err != nil || string(fullBaseline) != string(fullHTTP) {
		t.Fatalf("complete direct/HTTP issue differs: %s\n%s\n%v", fullBaseline, fullHTTP, err)
	}
	statusErr = errors.New("declared current acceptance evidence unavailable")
	if got, err := issueReader.Read(ctx, actor, 7, 3); !errors.Is(err, statusErr) || !errors.Is(err, factoryservice.ErrIssueStatus) || got != nil {
		t.Fatal("unknown current evidence fabricated issue status", got, err)
	}
	statusErr = nil
	foreignStatus = true
	if got, err := issueReader.Read(ctx, actor, 7, 3); !errors.Is(err, factoryservice.ErrWorkJourneyScope) || got != nil {
		t.Fatal("foreign acceptance escaped status port", got, err)
	}
	foreignStatus = false
	allowed = false
	if got, err := reader.Read(ctx, actor, 7, 3); !errors.Is(err, denied) || got != nil {
		t.Fatal("denied journey leaked facts", got, err)
	}
	if got, err := issueReader.Read(ctx, actor, 7, 3); !errors.Is(err, denied) || got != nil {
		t.Fatal("denied composite issue leaked facts", got, err)
	}
	allowed, denyAt = true, checks+2
	if got, err := reader.Read(ctx, actor, 7, 3); !errors.Is(err, denied) || got != nil {
		t.Fatal("late denial leaked assembled journey", got, err)
	}
	denyAt = checks + 2
	if got, err := issueReader.Read(ctx, actor, 7, 3); !errors.Is(err, denied) || got != nil {
		t.Fatal("late denial leaked composite issue", got, err)
	}
	denyAt = 0
	var originalPublication []byte
	if err := fx.raw.QueryRowContext(ctx, `SELECT data FROM factory_publications WHERE assignment=$1`, root.ID).Scan(&originalPublication); err != nil {
		t.Fatal(err)
	}
	if _, err := fx.raw.ExecContext(ctx, `UPDATE factory_publications SET data=jsonb_set(data,'{repository}','"8"') WHERE assignment=$1`, root.ID); err != nil {
		t.Fatal(err)
	}
	if got, err := reader.Read(ctx, actor, 7, 3); !errors.Is(err, factoryservice.ErrWorkJourneyScope) || got != nil {
		t.Fatal("foreign publication joined by assignment alone", got, err)
	}
	if got, err := issueReader.Read(ctx, actor, 7, 3); !errors.Is(err, factoryservice.ErrWorkJourneyScope) || got != nil {
		t.Fatal("composite issue joined foreign publication", got, err)
	}
	if _, err := fx.raw.ExecContext(ctx, `UPDATE factory_publications SET data=$2 WHERE assignment=$1`, root.ID, string(originalPublication)); err != nil {
		t.Fatal(err)
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if got, err := reader.Read(cancelled, actor, 7, 3); !errors.Is(err, context.Canceled) || got != nil {
		t.Fatal("cancelled journey returned facts", got, err)
	}
	if _, err := (factoryservice.WorkJourneys{Store: fx.db}).Read(ctx, actor, 7, 3); !errors.Is(err, factoryservice.ErrWorkJourneyUnavailable) {
		t.Fatal("unwired journey admission", err)
	}
	wanted = 9223372036854775807
	noAttempt, err := reader.Read(ctx, actor, wanted, 9223372036854775807)
	if err != nil || noAttempt.Repository != "9223372036854775807" || noAttempt.Issue != "9223372036854775807" || noAttempt.Available || noAttempt.Complete || len(noAttempt.Reasons) != 1 || noAttempt.Reasons[0] != "no_attempt" {
		t.Fatalf("canonical empty journey: %+v %v", noAttempt, err)
	}
	wanted = 7
	if err := fx.db.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := store.Open(fx.dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = reopened.Close() })
	reader.Store = reopened
	issueReader.Store, statusReader.Store = reopened, reopened
	view, err = reader.Read(ctx, actor, 7, 3)
	if err != nil {
		t.Fatal(err)
	}
	view.ObservedUnix = 0
	encoded, err = json.Marshal(view)
	if err != nil || string(encoded) != string(baseline) {
		t.Fatalf("reopened journey changed: %s\n%s\n%v", baseline, encoded, err)
	}
	whole, err = issueReader.Read(ctx, actor, 7, 3)
	if err != nil {
		t.Fatal(err)
	}
	whole.ObservedUnix, whole.Journey.ObservedUnix = 0, 0
	encoded, err = json.Marshal(whole)
	if err != nil || string(encoded) != string(fullBaseline) {
		t.Fatalf("reopened composite issue changed: %s\n%s\n%v", fullBaseline, encoded, err)
	}
	// Direct HTTP composition needs only canonical services and declared host
	// admission, with no native session or API object in the transport.
	var inventoryBefore int
	if err := fx.raw.QueryRowContext(ctx, `SELECT count(*) FROM projects`).Scan(&inventoryBefore); err != nil {
		t.Fatal(err)
	}
	var accountChecks, revokeAt atomic.Int32
	work := factoryhttpapi.Work{Reads: factoryservice.WorkReads{Store: reopened, Visibility: reader.Visibility}, Issues: issueReader,
		Host: fx.api.FountainIdentities.HostID(), Actor: actor,
		Current: func(w http.ResponseWriter, r *http.Request) bool {
			count := accountChecks.Add(1)
			if r.Header.Get("Authorization") != "Bearer declared-work-client" || revokeAt.Load() > 0 && count >= revokeAt.Load() {
				http.Error(w, "declared account admission unavailable", 401)
				return false
			}
			return true
		},
		VisibilityError: func(w http.ResponseWriter, _ error) { http.Error(w, "declared repository visibility unavailable", 404) },
	}
	server := httptest.NewServer(work.Handler())
	defer server.Close()
	direct := func(path string, status int) []byte {
		t.Helper()
		r, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+path, nil)
		if err != nil {
			t.Fatal(err)
		}
		r.Header.Set("Authorization", "Bearer declared-work-client")
		response, err := server.Client().Do(r)
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		body, err := io.ReadAll(response.Body)
		if err != nil || response.StatusCode != status {
			t.Fatalf("direct work HTTP %s: %d %s %v", path, response.StatusCode, body, err)
		}
		if status == 200 && response.Header.Get("Cache-Control") != "no-store" {
			t.Fatal("recorded work response was cacheable")
		}
		return body
	}
	base := "/api/repositories/7/factory/issues/3"
	directIssue := journeyIssueView(t, direct(base, 200))
	directIssue.ObservedUnix, directIssue.Journey.ObservedUnix = 0, 0
	encoded, err = json.Marshal(directIssue)
	if err != nil || string(encoded) != string(fullBaseline) {
		t.Fatal("direct host changed canonical issue/journey projection", string(encoded), err)
	}
	assignment, err := work.Reads.Assignment(ctx, actor, 7, 3)
	if err != nil {
		t.Fatal(err)
	}
	assignmentBytes, _ := json.Marshal(assignment)
	if served := direct(base+"/assignment", 200); strings.TrimSpace(string(served)) != string(assignmentBytes) {
		t.Fatal("direct host changed recorded assignment projection", string(served))
	}
	priority, err := work.Reads.Priority(ctx, actor, 7, 3, store.PriorityComparisonCursor{}, 1)
	if err != nil {
		t.Fatal(err)
	}
	var servedPriority factoryservice.IssuePriorityView
	if err := json.Unmarshal(direct(base+"/plan?limit=1", 200), &servedPriority); err != nil {
		t.Fatal(err)
	}
	priority.ObservedUnix, servedPriority.ObservedUnix = 0, 0
	a, _ := json.Marshal(priority)
	b, _ := json.Marshal(servedPriority)
	if string(a) != string(b) {
		t.Fatal("direct host changed priority projection", string(a), string(b))
	}
	direct(base+"?unexpected=1", 400)
	direct(base+"/assignment?unexpected=1", 400)
	direct(base+"/plan?limit=0", 400)
	direct(base+"/plan?before_unix=1", 400)
	direct("/api/repositories/07/factory/issues/3", 404)
	allowed = false
	if body := direct(base, 404); strings.Contains(string(body), root.ID) {
		t.Fatal("withdrawn visibility disclosed issue work", string(body))
	}
	allowed = true
	accountChecks.Store(0)
	revokeAt.Store(2)
	if body := direct(base+"/assignment", 401); strings.Contains(string(body), root.ID) || strings.Contains(string(body), "prompt") {
		t.Fatal("revoked host admission disclosed recorded assignment", string(body))
	}
	revokeAt.Store(0)
	var inventoryAfter int
	if err := fx.raw.QueryRowContext(ctx, `SELECT count(*) FROM projects`).Scan(&inventoryAfter); err != nil || inventoryAfter != inventoryBefore {
		t.Fatal("work HTTP created runtime inventory", inventoryBefore, inventoryAfter, err)
	}
	for table, count := range before {
		if after := journeyStoreCounts(t, fx.raw)[table]; after != count {
			t.Fatalf("journey read changed %s: %d -> %d", table, count, after)
		}
	}
}

func TestFactoryIssueJourneyReadsChangeNothing(t *testing.T) {
	fake := &journeyAcceptanceSource{}
	fx, call := journeyMux(t, fake)
	seedRichJourney(t, fx, fake)
	path := "/api/repositories/7/factory/issues/3"
	if code, body := call(http.MethodGet, path, ""); code != http.StatusOK {
		t.Fatalf("warmup status=%d body=%s", code, body)
	}
	before := journeyStoreCounts(t, fx.raw)
	var baseline []byte
	for i := 0; i < 3; i++ {
		code, body := call(http.MethodGet, path, "")
		if code != http.StatusOK {
			t.Fatalf("repeat %d status=%d body=%s", i, code, body)
		}
		view := journeyIssueView(t, body)
		view.Journey.ObservedUnix = 0
		encoded, err := json.Marshal(view.Journey)
		if err != nil {
			t.Fatal(err)
		}
		if baseline == nil {
			baseline = encoded
		} else if string(encoded) != string(baseline) {
			t.Fatalf("repeat %d changed the journey:\n%s\n%s", i, baseline, encoded)
		}
	}
	after := journeyStoreCounts(t, fx.raw)
	for table, count := range before {
		if after[table] != count {
			t.Fatalf("table %s moved %d -> %d", table, count, after[table])
		}
	}
}

// TestFactoryIssueJourneyPhasesRetainNewestFailed repeats one phase
// past the old eight-row cap: eleven implement attempts on one root
// whose newest fails. The current implement state stays attention
// behind the newest failed child — an older success never masks it.
func TestFactoryIssueJourneyPhasesRetainNewestFailed(t *testing.T) {
	fake := &journeyAcceptanceSource{}
	fx, call := journeyMux(t, fake)
	acceptance := admitJourneyAcceptance(t, fx.db, 1, "")
	fake.decision = journeyAcceptanceFor(acceptance)
	root := journeyOwner(t, strings.Repeat("7", 32), acceptance, project.ExecutionImplement, factory.AssignmentFinished, factory.Succeeded, "")
	insertJourneyRow(t, fx.raw, root, factory.AssignmentFinished)
	var newest factory.Assignment
	for i := 0; i < 10; i++ {
		outcome := factory.Succeeded
		if i == 9 {
			outcome = factory.Failed
		}
		newest = journeyChild(t, fmt.Sprintf("%032x", 20000+i), root.ID, acceptance, project.ExecutionImplement, factory.AssignmentFinished, outcome)
		insertJourneyRow(t, fx.raw, newest, factory.AssignmentFinished)
	}

	code, body := call(http.MethodGet, "/api/repositories/7/factory/issues/3", "")
	if code != http.StatusOK {
		t.Fatalf("newest-failed status=%d body=%s", code, body)
	}
	journey := journeyIssueView(t, body).Journey
	if len(journey.Phases) != 1 {
		t.Fatalf("newest-failed phases: %+v", journey.Phases)
	}
	got := journey.Phases[0]
	if got.Phase != project.ExecutionImplement || got.State != "attention" {
		t.Fatalf("newest-failed state: %+v", got)
	}
	if got.Assignment != newest.ID || got.Run != newest.Run || got.AttemptRoot != root.ID {
		t.Fatalf("newest-failed refs: %+v want %s/%s/%s", got, newest.ID, newest.Run, root.ID)
	}
	if len(journey.Reasons) != 2 || journey.Reasons[0] != "adoptions_unavailable" || journey.Reasons[1] != "delivery_unavailable" {
		t.Fatalf("newest-failed reasons: %+v", journey.Reasons)
	}
}

// TestFactoryIssueJourneyPhasesNeedsHumanAttention proves a finished
// implement assignment with Outcome needs-human (Reason
// harness_no_report, the normal dispatch_result outcome for missing
// typed reports) renders attention, never complete, over the
// registered HTTP path.
func TestFactoryIssueJourneyPhasesNeedsHumanAttention(t *testing.T) {
	fake := &journeyAcceptanceSource{}
	fx, call := journeyMux(t, fake)
	acceptance := admitJourneyAcceptance(t, fx.db, 1, "")
	fake.decision = journeyAcceptanceFor(acceptance)
	root := journeyOwner(t, strings.Repeat("7", 32), acceptance, project.ExecutionImplement, factory.AssignmentFinished, factory.Succeeded, "")
	insertJourneyRow(t, fx.raw, root, factory.AssignmentFinished)
	needy := journeyChild(t, fmt.Sprintf("%032x", 30000), root.ID, acceptance, project.ExecutionImplement, factory.AssignmentFinished, factory.NeedsHuman)
	needy.Reason = factory.AssignReasonNoReport
	insertJourneyRow(t, fx.raw, needy, factory.AssignmentFinished)

	code, body := call(http.MethodGet, "/api/repositories/7/factory/issues/3", "")
	if code != http.StatusOK {
		t.Fatalf("needs-human status=%d body=%s", code, body)
	}
	journey := journeyIssueView(t, body).Journey
	if len(journey.Phases) != 1 {
		t.Fatalf("needs-human phases: %+v", journey.Phases)
	}
	got := journey.Phases[0]
	if got.Phase != project.ExecutionImplement || got.State != "attention" {
		t.Fatalf("needs-human state: %+v, want implement/attention", got)
	}
	if got.Assignment != needy.ID || got.Run != needy.Run || got.AttemptRoot != root.ID {
		t.Fatalf("needs-human refs: %+v want %s/%s/%s", got, needy.ID, needy.Run, root.ID)
	}
}

// TestFactoryIssueJourneyPhasesIsolateAttemptRoots shares one
// acceptance between two owner roots: the selected (newest owner)
// root renders alone, and the sibling root's phases never mix in.
func TestFactoryIssueJourneyPhasesIsolateAttemptRoots(t *testing.T) {
	fake := &journeyAcceptanceSource{}
	fx, call := journeyMux(t, fake)
	acceptance := admitJourneyAcceptance(t, fx.db, 1, "")
	fake.decision = journeyAcceptanceFor(acceptance)
	rootA := journeyOwner(t, strings.Repeat("a", 32), acceptance, project.ExecutionImplement, factory.AssignmentFinished, factory.Succeeded, "")
	insertJourneyRow(t, fx.raw, rootA, factory.AssignmentFinished)
	childA := journeyChild(t, strings.Repeat("c", 32), rootA.ID, acceptance, project.ExecutionReview, factory.AssignmentFinished, factory.Succeeded)
	insertJourneyRow(t, fx.raw, childA, factory.AssignmentFinished)
	rootB := journeyOwner(t, strings.Repeat("b", 32), acceptance, project.ExecutionCorrect, factory.AssignmentFinished, factory.Succeeded, "")
	insertJourneyRow(t, fx.raw, rootB, factory.AssignmentFinished)

	code, body := call(http.MethodGet, "/api/repositories/7/factory/issues/3", "")
	if code != http.StatusOK {
		t.Fatalf("isolated status=%d body=%s", code, body)
	}
	journey := journeyIssueView(t, body).Journey
	if journey.Lineage.RootAssignment != rootB.ID || journey.Lineage.AttemptRoot != rootB.ID {
		t.Fatalf("isolated lineage: %+v", journey.Lineage)
	}
	if len(journey.Phases) != 1 {
		t.Fatalf("isolated phases: %+v", journey.Phases)
	}
	got := journey.Phases[0]
	if got.Phase != project.ExecutionCorrect || got.State != "complete" {
		t.Fatalf("isolated state: %+v", got)
	}
	if got.Assignment != rootB.ID || got.Run != rootB.Run || got.AttemptRoot != rootB.ID {
		t.Fatalf("isolated refs: %+v", got)
	}
}

// TestFactoryIssueJourneyInvalidPhaseExplicit records one unknown
// nonempty phase beside a valid one: the valid phase still renders
// current, the unknown value stays out, and the journey names
// phases_invalid_row instead of dropping the row silently.
func TestFactoryIssueJourneyInvalidPhaseExplicit(t *testing.T) {
	fake := &journeyAcceptanceSource{}
	fx, call := journeyMux(t, fake)
	acceptance := admitJourneyAcceptance(t, fx.db, 1, "")
	fake.decision = journeyAcceptanceFor(acceptance)
	root := journeyOwner(t, strings.Repeat("7", 32), acceptance, project.ExecutionImplement, factory.AssignmentFinished, factory.Succeeded, "")
	insertJourneyRow(t, fx.raw, root, factory.AssignmentFinished)
	junk := journeyChild(t, strings.Repeat("e", 32), root.ID, acceptance, "publish", factory.AssignmentFinished, factory.Succeeded)
	insertJourneyRow(t, fx.raw, junk, factory.AssignmentFinished)

	code, body := call(http.MethodGet, "/api/repositories/7/factory/issues/3", "")
	if code != http.StatusOK {
		t.Fatalf("invalid-phase status=%d body=%s", code, body)
	}
	view := journeyIssueView(t, body)
	if len(view.Journey.Phases) != 1 || view.Journey.Phases[0].Phase != project.ExecutionImplement {
		t.Fatalf("invalid-phase phases: %+v", view.Journey.Phases)
	}
	if view.Journey.Phases[0].Assignment != root.ID {
		t.Fatalf("invalid-phase refs: %+v", view.Journey.Phases[0])
	}
	if reasons := journeyReasons(view); !reasons["phases_invalid_row"] {
		t.Fatalf("invalid-phase not named: %+v", view.Journey.Reasons)
	}
	if view.Journey.Complete {
		t.Fatalf("invalid-phase journey reads complete: %+v", view.Journey)
	}
}

// TestFactoryIssueJourneyPhaselessRowInvalid records one row without a
// phase beside the owner's implement attempt: no phase is fabricated
// from it, implement still renders current, and the journey names
// phases_invalid_row.
func TestFactoryIssueJourneyPhaselessRowInvalid(t *testing.T) {
	fake := &journeyAcceptanceSource{}
	fx, call := journeyMux(t, fake)
	acceptance := admitJourneyAcceptance(t, fx.db, 1, "")
	fake.decision = journeyAcceptanceFor(acceptance)
	root := journeyOwner(t, strings.Repeat("7", 32), acceptance, project.ExecutionImplement, factory.AssignmentFinished, factory.Succeeded, "")
	insertJourneyRow(t, fx.raw, root, factory.AssignmentFinished)
	pre := journeyOwner(t, strings.Repeat("d", 32), acceptance, "", factory.AssignmentFinished, factory.Succeeded, "")
	pre.AttemptRoot, pre.PublicationAssignment = root.ID, root.ID
	insertJourneyRow(t, fx.raw, pre, factory.AssignmentFinished)

	code, body := call(http.MethodGet, "/api/repositories/7/factory/issues/3", "")
	if code != http.StatusOK {
		t.Fatalf("phaseless status=%d body=%s", code, body)
	}
	view := journeyIssueView(t, body)
	if len(view.Journey.Phases) != 1 || view.Journey.Phases[0].Phase != project.ExecutionImplement {
		t.Fatalf("phaseless phases: %+v", view.Journey.Phases)
	}
	if view.Journey.Phases[0].Assignment != root.ID {
		t.Fatalf("phaseless refs: %+v", view.Journey.Phases[0])
	}
	if reasons := journeyReasons(view); !reasons["phases_invalid_row"] {
		t.Fatalf("phaseless row not named invalid: %+v", view.Journey.Reasons)
	}
}

// seedJourneyAdoption records one accepted intent revision and binds
// its task to the journey issue under the journey acceptance, through
// the real intent writers. The binding is the canonical adoption
// record the journey evidence section joins on.
func seedJourneyAdoption(t *testing.T, s *store.Store, acceptance string) (string, string) {
	t.Helper()
	revision := factory.IntentRevision{
		ID: "d111111111111111111111111", Intent: "d222222222222222222222222",
		Owner: 1, Repository: journeyRepo, Objective: "Build a request form",
		State: "accepted", MaxIssues: 4, CreatedUnix: 1,
		Ideas: []factory.ProductIdea{{ID: "d333333333333333333333333", Description: "Request form", Consequence: "Requests are recorded", Decision: "accept", Source: "requested", Provenance: "human", Actor: 1}},
		Tasks: []factory.IntentTask{{Key: "d444444444444444444444444", Objective: "Build request form", Outcomes: []string{"Submit a request"}, IdeaIDs: []string{"d333333333333333333333333"}}},
	}
	ctx := context.Background()
	if err := s.SaveIntentRevision(ctx, revision); err != nil {
		t.Fatal(err)
	}
	binding := factory.IntentTaskBinding{
		RevisionID: revision.ID, TaskKey: "d444444444444444444444444",
		IssueID: 99, IssueIndex: journeyIssue, AcceptanceID: acceptance, CreatedUnix: 1,
	}
	if err := s.BindIntentTaskAcceptance(ctx, binding); err != nil {
		t.Fatal(err)
	}
	return revision.Intent, revision.ID
}

// seedJourneySuccessorAdoption records a successor intent revision that
// rebinds the unchanged task to the same native issue and acceptance.
// Both historical bindings stay visible as adoption history; the
// delivered section must show the shared completion once.
func seedJourneySuccessorAdoption(t *testing.T, s *store.Store, intent, predecessor, acceptance string) string {
	t.Helper()
	revision := factory.IntentRevision{
		ID: "d666666666666666666666666", Intent: intent,
		Owner: 1, Repository: journeyRepo, Objective: "Build a request form",
		State: "accepted", MaxIssues: 4, CreatedUnix: 2, Predecessor: predecessor,
		Ideas: []factory.ProductIdea{{ID: "d333333333333333333333333", Description: "Request form", Consequence: "Requests are recorded", Decision: "accept", Source: "requested", Provenance: "human", Actor: 1}},
		Tasks: []factory.IntentTask{{Key: "d444444444444444444444444", Objective: "Build request form", Outcomes: []string{"Submit a request"}, IdeaIDs: []string{"d333333333333333333333333"}}},
	}
	ctx := context.Background()
	if err := s.SaveIntentRevision(ctx, revision); err != nil {
		t.Fatal(err)
	}
	binding := factory.IntentTaskBinding{
		RevisionID: revision.ID, TaskKey: "d444444444444444444444444",
		IssueID: 99, IssueIndex: journeyIssue, AcceptanceID: acceptance, CreatedUnix: 2,
	}
	if err := s.BindIntentTaskAcceptance(ctx, binding); err != nil {
		t.Fatal(err)
	}
	return revision.ID
}

// journeyDeliveryAcceptanceFixture builds one valid journaled delivery
// acceptance over the fixture publication's committed correction tip.
func journeyDeliveryAcceptanceFixture(root factory.Assignment, kind string) factory.DeliveryAcceptance {
	publicationID := strings.Repeat("9", 32)
	acceptance := factory.DeliveryAcceptance{
		Tuple:         factory.CandidateTuple{Head: journeyCorrection, Base: journeyBase, DiffBase: journeyBase},
		ID:            factory.DeliveryAcceptanceID(publicationID, root.ID, 1, kind),
		PublicationID: publicationID, AttemptRoot: root.ID,
		ScopeDigest: strings.Repeat("3", 64), FindingDigest: strings.Repeat("4", 64),
		CoverageDigest: strings.Repeat("5", 64), CheckDigest: strings.Repeat("6", 64),
		PolicyRevision: "policy-1", OriginReceipt: strings.Repeat("6", 32),
		Kind: kind, Round: 1,
	}
	if kind == factory.AcceptanceJevAcceptance {
		acceptance.DecisionID = strings.Repeat("5", 32)
	}
	return acceptance
}

// seedJourneyDeliveryReceipt records the decided funded delivery
// receipt whose fingerprinted input binds the fixture publication,
// attempt root, round and correction tuple.
func seedJourneyDeliveryReceipt(t *testing.T, fx *journeyHTTPFixture, root factory.Assignment) {
	t.Helper()
	in := factory.DeliveryFingerprintInput{
		Reviews:     []string{strings.Repeat("6", 32)},
		AttemptRoot: root.ID, PublicationID: strings.Repeat("9", 32),
		ScopeDigest: strings.Repeat("3", 64), FixDigest: strings.Repeat("7", 64),
		DiffDigest: strings.Repeat("8", 64), CheckDigest: strings.Repeat("6", 64),
		TreeDigest: strings.Repeat("9", 64),
		Head:       journeyCorrection, Base: journeyBase,
		Model: "test-delivery-model", Rubric: factory.DeliveryRubric,
		PolicyRevision: 1, Round: 1,
	}
	fingerprint, err := factory.DeliveryFingerprint(in)
	if err != nil {
		t.Fatal(err)
	}
	answer := factory.DeliveryAnswer{
		Selected: factory.DeliveryReadyToMerge, Probability: 0.9, Confidence: 0.95,
		Probabilities: map[string]float64{factory.DeliveryReadyToMerge: 0.9, factory.DeliveryReviewAgain: 0.1},
		Model:         "test-delivery-model", Fingerprint: fingerprint,
	}
	receipt := factory.NewDeliveryReceipt(in, fingerprint, answer, true, 41)
	raw, err := json.Marshal(receipt)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fx.raw.Exec(`INSERT INTO factory_priority_judgments(snapshot,max_calls,data) VALUES($1,1,$2)`,
		fingerprint, string(raw)); err != nil {
		t.Fatal(err)
	}
}

// seedJourneyWithDelivery records the rich lineage with one journaled
// delivery acceptance of the given kind, its funded receipt when
// asked, one intent adoption binding, the canonical completion behind
// that binding, and the F10 verified application behind the adopted
// revision. The full adopted to work to verified chain stays closed,
// so the complete-chain tests prove every evidence join together.
func seedJourneyWithDelivery(t *testing.T, fx *journeyHTTPFixture, fake *journeyAcceptanceSource, kind string, withReceipt bool) (string, string, factory.Assignment) {
	t.Helper()
	acceptance := admitJourneyAcceptance(t, fx.db, 1, "")
	fake.decision = journeyAcceptanceFor(acceptance)
	root := journeyOwner(t, strings.Repeat("7", 32), acceptance, project.ExecutionImplement, factory.AssignmentFinished, factory.Succeeded, "")
	child := journeyChild(t, strings.Repeat("8", 32), root.ID, acceptance, project.ExecutionReview, factory.AssignmentFinished, factory.Failed)
	insertJourneyRow(t, fx.raw, root, factory.AssignmentFinished)
	insertJourneyRow(t, fx.raw, child, factory.AssignmentFinished)
	publication := journeyPublicationFixture(root, factory.OpEffectCommitted, factory.FindingSeverityMajor)
	publication.DeliveryAcceptances = []factory.DeliveryAcceptance{journeyDeliveryAcceptanceFixture(root, kind)}
	insertJourneyPublication(t, fx.raw, publication)
	seedJourneyPlan(t, fx.db, root.ID)
	seedJourneyCycle(t, fx, acceptance, strings.Repeat("a", 32), 1, "awaiting_coder")
	intent, revision := seedJourneyAdoption(t, fx.db, acceptance)
	seedJourneyDeliveredCompletion(t, fx.raw, acceptance)
	seedJourneyVerifiedApp(t, fx.raw, intent, revision)
	if withReceipt {
		seedJourneyDeliveryReceipt(t, fx, root)
	}
	return intent, revision, root
}

// seedJourneyDeliveredCompletion records the canonical completion
// event behind the adopted binding: the exact delivered-task evidence
// the journey read joins on native issue and acceptance. Generation 2
// is free in this lineage (the reassessment cycle seeds generation 1
// with placeholder bytes), and the latest generation wins the read.
func seedJourneyDeliveredCompletion(t *testing.T, raw *sql.DB, acceptance string) {
	t.Helper()
	event := factory.CompletionEvent{
		ID: strings.Repeat("d", 32), PublicationID: strings.Repeat("9", 32),
		MergedCommit: journeyCorrection, ClosingOccurrence: "journey-delivered",
		AcceptanceID: acceptance,
		Repository:   journeyRepo, Issue: journeyIssue, Generation: 2,
	}
	if err := event.Validate(); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(event)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := raw.Exec(`INSERT INTO factory_completions(repository,generation,event,publication,occurrence,data) VALUES($1,$2,$3,$4,$5,$6)`,
		event.Repository, event.Generation, event.ID, event.PublicationID, event.ClosingOccurrence, string(data)); err != nil {
		t.Fatal(err)
	}
}

// seedJourneyVerifiedApp records the finished verify assignment and its
// F10 verified feed event behind the adopted intent revision under the
// canonical identity, outcome and scope the delivery admission owns:
// finished verify, succeeded outcome, trusted verification outcome
// verified, exact intent, revision and scope digest, with the F10 event
// bound to both the delivery and the assignment. Only this trusted
// check renders; the observation and receipt seeds above never stand
// in for it.
func seedJourneyVerifiedApp(t *testing.T, raw *sql.DB, intent, revision string) {
	t.Helper()
	var revisionRaw []byte
	if err := raw.QueryRow(`SELECT data FROM factory_intent_revisions WHERE id=$1`, revision).Scan(&revisionRaw); err != nil {
		t.Fatal(err)
	}
	var adopted factory.IntentRevision
	if err := json.Unmarshal(revisionRaw, &adopted); err != nil {
		t.Fatal(err)
	}
	if adopted.ID != revision || adopted.Intent != intent {
		t.Fatalf("verified seed revision mismatch: %+v", adopted)
	}
	manifest := strings.Repeat("7", 64)
	subject := factory.DeliverySubject{
		IntentID: intent, AcceptedRevision: revision,
		ScopeDigest: adopted.Digest(), SourceCommit: journeyCorrection,
		PreparationDigest: strings.Repeat("4", 64), VerificationManifestDigest: manifest,
	}
	subject.DeliveryID = factory.DeliveryID(subject)
	if err := subject.Validate(); err != nil {
		t.Fatal(err)
	}
	report := factory.AppVerificationReport{
		DeliveryID: subject.DeliveryID, InputDigest: strings.Repeat("9", 64),
		Outcome:       factory.AppVerificationVerified,
		CheckReceipts: []factory.VerificationCheckReceipt{{CheckID: factory.ApprovedAppCheckID, Outcome: factory.CheckReceiptPassed, ArtifactDigest: strings.Repeat("e", 64), LogExcerpt: "approved check passed"}},
	}
	if err := report.Validate(); err != nil {
		t.Fatal(err)
	}
	verify := factory.Assignment{
		ID: strings.Repeat("5", 32), AttemptRoot: strings.Repeat("5", 32),
		ProjectID: "p" + strings.Repeat("1", 24), Role: project.RoleReviewer,
		Phase: project.ExecutionVerify, Stage: factory.AssignmentFinished, Outcome: factory.Succeeded,
		Subject:      factory.ExecutionSubject{Kind: factory.SubjectDelivery, Delivery: &subject},
		Preparation:  "f" + strings.Repeat("b", 24),
		SourceCommit: journeyCorrection, Run: strings.Repeat("6", 32),
		RunHistory: []string{strings.Repeat("6", 32)},
		Repository: journeyRepo, CreatedUnix: 1700000000, ActorID: 1,
		Result: &factory.AssignmentResult{
			AssignmentID: strings.Repeat("5", 32), RunID: strings.Repeat("6", 32),
			Phase: project.ExecutionVerify, Status: "completed",
			Summary: "Application outcomes verified", Reported: true, RecordedUnix: 1700000000,
			PhaseResult: &factory.PhaseResult{
				Version: factory.PhaseResultVersion, Phase: project.ExecutionVerify,
				Status: "completed", Summary: "Application outcomes verified",
				Output: &factory.PhaseOutput{Verification: &report},
			},
		},
	}
	insertJourneyRow(t, raw, verify, factory.AssignmentFinished)
	content := factory.FeedCommunication{
		ComponentID: "F10", Title: "Application outcomes verified",
		Summary:      "All accepted outcomes passed the approved application check.",
		EvidenceRefs: []string{journeyCorrection, manifest, strings.Repeat("8", 64)},
	}
	feed, err := json.Marshal(map[string]any{
		"actor": "factory", "author_name": "Factory", "issue": 0,
		"assignment": verify.ID, "at": 1700000000, "status": "application:verified",
		"communication": content, "event_identity": subject.DeliveryID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := raw.Exec(`INSERT INTO factory_feed_posts(id,repository,revision,author,body,idempotency,data) VALUES($1,$2,0,'factory',$3,$4,$5)`,
		subject.DeliveryID, journeyRepo,
		content.Title+"\n\n"+content.Summary, subject.DeliveryID, string(feed)); err != nil {
		t.Fatal(err)
	}
}

func TestFactoryIssueJourneyEvidenceJoinsComplete(t *testing.T) {
	fake := &journeyAcceptanceSource{}
	fx, call := journeyMux(t, fake)
	intent, revision, _ := seedJourneyWithDelivery(t, fx, fake, factory.AcceptanceJevAcceptance, true)

	code, body := call(http.MethodGet, "/api/repositories/7/factory/issues/3", "")
	if code != http.StatusOK {
		t.Fatalf("evidence status=%d body=%s", code, body)
	}
	journey := journeyIssueView(t, body).Journey
	if !journey.Available || !journey.Complete || len(journey.Reasons) != 0 {
		t.Fatalf("evidence flags: %+v", journey)
	}
	if len(journey.Adoptions) != 1 || journey.Adoptions[0].Intent != intent ||
		journey.Adoptions[0].Revision != revision || journey.Adoptions[0].State != "adopted" {
		t.Fatalf("evidence adoptions: %+v", journey.Adoptions)
	}
	if len(journey.Judgments) != 2 || journey.Judgments[0].Kind != "delivery" ||
		journey.Judgments[0].Selected != "ready_to_merge" ||
		journey.Judgments[0].Model != "test-delivery-model" || journey.Judgments[0].Probability != 0.9 ||
		journey.Judgments[1].Kind != "reassessment-decision" {
		t.Fatalf("evidence judgments: %+v", journey.Judgments)
	}
}

func TestFactoryIssueJourneyDeliveryMissingReceipt(t *testing.T) {
	fake := &journeyAcceptanceSource{}
	fx, call := journeyMux(t, fake)
	seedJourneyWithDelivery(t, fx, fake, factory.AcceptanceJevAcceptance, false)

	code, body := call(http.MethodGet, "/api/repositories/7/factory/issues/3", "")
	if code != http.StatusOK {
		t.Fatalf("missing-receipt status=%d body=%s", code, body)
	}
	journey := journeyIssueView(t, body).Journey
	// Adoption proof renders; the unlinked Jev acceptance keeps the
	// delivery section honestly unavailable with no fabricated judgment.
	if len(journey.Adoptions) != 1 || journey.Complete {
		t.Fatalf("missing-receipt adoption/flags: %+v", journey)
	}
	if len(journey.Reasons) != 1 || journey.Reasons[0] != "delivery_unavailable" {
		t.Fatalf("missing-receipt reasons: %+v", journey.Reasons)
	}
	if len(journey.Judgments) != 1 || journey.Judgments[0].Kind != "reassessment-decision" {
		t.Fatalf("missing-receipt judgments: %+v", journey.Judgments)
	}
}

func TestFactoryIssueJourneyCleanDeliveryEvidence(t *testing.T) {
	fake := &journeyAcceptanceSource{}
	fx, call := journeyMux(t, fake)
	seedJourneyWithDelivery(t, fx, fake, factory.AcceptanceCleanReview, false)

	code, body := call(http.MethodGet, "/api/repositories/7/factory/issues/3", "")
	if code != http.StatusOK {
		t.Fatalf("clean-delivery status=%d body=%s", code, body)
	}
	journey := journeyIssueView(t, body).Journey
	// A clean-review acceptance is its own delivery evidence: the join
	// closes without inventing a funded judgment.
	if !journey.Complete || len(journey.Reasons) != 0 {
		t.Fatalf("clean-delivery flags: %+v", journey)
	}
	if len(journey.Judgments) != 1 || journey.Judgments[0].Kind != "reassessment-decision" {
		t.Fatalf("clean-delivery judgments: %+v", journey.Judgments)
	}
}

func TestFactoryIssueJourneyDeliveredDedupOnUnchangedRebind(t *testing.T) {
	fake := &journeyAcceptanceSource{}
	fx, call := journeyMux(t, fake)
	intent, revision, root := seedJourneyWithDelivery(t, fx, fake, factory.AcceptanceJevAcceptance, true)
	seedJourneySuccessorAdoption(t, fx.db, intent, revision, root.Acceptance)

	code, body := call(http.MethodGet, "/api/repositories/7/factory/issues/3", "")
	if code != http.StatusOK {
		t.Fatalf("dedup status=%d body=%s", code, body)
	}
	journey := journeyIssueView(t, body).Journey
	if len(journey.Adoptions) != 2 {
		t.Fatalf("dedup preserved history: %+v", journey.Adoptions)
	}
	if len(journey.Delivered) != 1 {
		t.Fatalf("dedup delivered section: %+v", journey.Delivered)
	}
	first, second := journey.Adoptions[0], journey.Adoptions[1]
	if first.Task != second.Task || first.IssueID != second.IssueID || first.Acceptance != second.Acceptance {
		t.Fatalf("rebind changed task identity: %+v %+v", first, second)
	}
	delivered := journey.Delivered[0]
	if delivered.Task != first.Task || delivered.Issue != first.IssueID || delivered.Acceptance != first.Acceptance {
		t.Fatalf("delivered identity: %+v adoptions %+v %+v", delivered, first, second)
	}
	for _, reason := range journey.Reasons {
		if reason == "deliveries_unavailable" || reason == "deliveries_invalid_row" {
			t.Fatalf("dedup falsely unavailable: %+v", journey)
		}
	}
}

func journeyVerifiedDeliveryID(t *testing.T, raw *sql.DB) string {
	t.Helper()
	var delivery string
	if err := raw.QueryRow(`SELECT data->'subject'->'delivery'->>'delivery_id' FROM factory_assignments WHERE id=$1`, strings.Repeat("5", 32)).Scan(&delivery); err != nil {
		t.Fatal(err)
	}
	return delivery
}

func TestFactoryIssueJourneyVerifiedRefusesWrongAssignment(t *testing.T) {
	fake := &journeyAcceptanceSource{}
	fx, call := journeyMux(t, fake)
	seedJourneyWithDelivery(t, fx, fake, factory.AcceptanceJevAcceptance, true)
	delivery := journeyVerifiedDeliveryID(t, fx.raw)
	if _, err := fx.raw.Exec(`UPDATE factory_feed_posts SET data=jsonb_set(data,'{assignment}',$1) WHERE id=$2`,
		`"00000000000000000000000000000000"`, delivery); err != nil {
		t.Fatal(err)
	}
	code, body := call(http.MethodGet, "/api/repositories/7/factory/issues/3", "")
	if code != http.StatusOK {
		t.Fatalf("wrong-assignment status=%d body=%s", code, body)
	}
	view := journeyIssueView(t, body)
	journey := view.Journey
	if len(journey.Verifications) != 0 {
		t.Fatalf("wrong-assignment verifications: %+v", journey.Verifications)
	}
	if !journeyReasons(view)["verifications_unavailable"] {
		t.Fatalf("wrong-assignment reasons: %+v", journey.Reasons)
	}
}

func TestFactoryIssueJourneyVerifiedRefusesWrongScope(t *testing.T) {
	fake := &journeyAcceptanceSource{}
	fx, call := journeyMux(t, fake)
	seedJourneyWithDelivery(t, fx, fake, factory.AcceptanceJevAcceptance, true)
	old := journeyVerifiedDeliveryID(t, fx.raw)
	var data []byte
	if err := fx.raw.QueryRow(`SELECT data FROM factory_assignments WHERE id=$1`, strings.Repeat("5", 32)).Scan(&data); err != nil {
		t.Fatal(err)
	}
	var stored factory.Assignment
	if err := json.Unmarshal(data, &stored); err != nil {
		t.Fatal(err)
	}
	stored.Subject.Delivery.ScopeDigest = strings.Repeat("0", 64)
	stored.Subject.Delivery.DeliveryID = factory.DeliveryID(*stored.Subject.Delivery)
	patched, err := json.Marshal(stored)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fx.raw.Exec(`UPDATE factory_assignments SET data=$1 WHERE id=$2`, string(patched), stored.ID); err != nil {
		t.Fatal(err)
	}
	// Keep the event join valid: the F10 event follows the tampered
	// delivery identity, so only the scope guard can refuse.
	res, err := fx.raw.Exec(`UPDATE factory_feed_posts SET data=jsonb_set(data,'{event_identity}',$1) WHERE id=$2`,
		`"`+stored.Subject.Delivery.DeliveryID+`"`, old)
	if err != nil {
		t.Fatal(err)
	}
	if n, _ := res.RowsAffected(); n != 1 {
		t.Fatalf("F10 repoint affected %d rows, join not preserved", n)
	}
	code, body := call(http.MethodGet, "/api/repositories/7/factory/issues/3", "")
	if code != http.StatusOK {
		t.Fatalf("wrong-scope status=%d body=%s", code, body)
	}
	view := journeyIssueView(t, body)
	journey := view.Journey
	if len(journey.Verifications) != 0 {
		t.Fatalf("wrong-scope verifications: %+v", journey.Verifications)
	}
	if !journeyReasons(view)["verifications_unavailable"] {
		t.Fatalf("wrong-scope reasons: %+v", journey.Reasons)
	}
}

func TestFactoryIssueJourneyVerifiedRefusesManifestMismatch(t *testing.T) {
	fake := &journeyAcceptanceSource{}
	fx, call := journeyMux(t, fake)
	seedJourneyWithDelivery(t, fx, fake, factory.AcceptanceJevAcceptance, true)
	delivery := journeyVerifiedDeliveryID(t, fx.raw)
	// Keep the event join valid (same delivery, same assignment, valid
	// F10) and vary only the observed manifest: a well-formed but
	// foreign digest is an explicit storage failure, never a verified
	// application.
	res, err := fx.raw.Exec(`UPDATE factory_feed_posts SET data=jsonb_set(data,'{communication,evidence_refs,1}',$1) WHERE id=$2`,
		`"`+strings.Repeat("2", 64)+`"`, delivery)
	if err != nil {
		t.Fatal(err)
	}
	if n, _ := res.RowsAffected(); n != 1 {
		t.Fatalf("F10 manifest tamper affected %d rows, join not preserved", n)
	}
	code, body := call(http.MethodGet, "/api/repositories/7/factory/issues/3", "")
	if code != http.StatusServiceUnavailable || journeyErrorCode(t, body) != "store_unavailable" {
		t.Fatalf("manifest-mismatch status=%d body=%s", code, body)
	}
}

func TestFactoryIssueJourneyVerifiedRefusesNonVerifiedResult(t *testing.T) {
	fake := &journeyAcceptanceSource{}
	fx, call := journeyMux(t, fake)
	seedJourneyWithDelivery(t, fx, fake, factory.AcceptanceJevAcceptance, true)
	var data []byte
	if err := fx.raw.QueryRow(`SELECT data FROM factory_assignments WHERE id=$1`, strings.Repeat("5", 32)).Scan(&data); err != nil {
		t.Fatal(err)
	}
	var stored factory.Assignment
	if err := json.Unmarshal(data, &stored); err != nil {
		t.Fatal(err)
	}
	stored.Result.PhaseResult.Output.Verification.Outcome = factory.AppVerificationFailed
	stored.Result.PhaseResult.Output.Verification.CheckReceipts = []factory.VerificationCheckReceipt{{CheckID: factory.ApprovedAppCheckID, Outcome: factory.CheckReceiptPassed, ArtifactDigest: strings.Repeat("e", 64), LogExcerpt: "x"}}
	stored.Result.PhaseResult.Output.Verification.Failures = []factory.CheckFailure{{CheckRef: "outcome", Detail: "failed"}}
	patched, err := json.Marshal(stored)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fx.raw.Exec(`UPDATE factory_assignments SET data=$1 WHERE id=$2`, string(patched), stored.ID); err != nil {
		t.Fatal(err)
	}
	code, body := call(http.MethodGet, "/api/repositories/7/factory/issues/3", "")
	if code != http.StatusOK {
		t.Fatalf("non-verified status=%d body=%s", code, body)
	}
	view := journeyIssueView(t, body)
	journey := view.Journey
	if len(journey.Verifications) != 0 {
		t.Fatalf("non-verified verifications: %+v", journey.Verifications)
	}
	if !journeyReasons(view)["verifications_unavailable"] {
		t.Fatalf("non-verified reasons: %+v", journey.Reasons)
	}
}

func TestFactoryIssueJourneyVerifiedRefusesMissingF10(t *testing.T) {
	fake := &journeyAcceptanceSource{}
	fx, call := journeyMux(t, fake)
	seedJourneyWithDelivery(t, fx, fake, factory.AcceptanceJevAcceptance, true)
	delivery := journeyVerifiedDeliveryID(t, fx.raw)
	if _, err := fx.raw.Exec(`DELETE FROM factory_feed_posts WHERE id=$1`, delivery); err != nil {
		t.Fatal(err)
	}
	code, body := call(http.MethodGet, "/api/repositories/7/factory/issues/3", "")
	if code != http.StatusOK {
		t.Fatalf("missing-F10 status=%d body=%s", code, body)
	}
	view := journeyIssueView(t, body)
	journey := view.Journey
	if len(journey.Verifications) != 0 {
		t.Fatalf("missing-F10 verifications: %+v", journey.Verifications)
	}
	if !journeyReasons(view)["verifications_unavailable"] {
		t.Fatalf("missing-F10 reasons: %+v", journey.Reasons)
	}
}

func TestFactoryIssueJourneyVerifiedIgnoresUnrelatedNewer(t *testing.T) {
	fake := &journeyAcceptanceSource{}
	fx, call := journeyMux(t, fake)
	intent, revision, _ := seedJourneyWithDelivery(t, fx, fake, factory.AcceptanceJevAcceptance, true)
	second := "d777777777777777777777777"
	other := factory.IntentRevision{
		ID: second, Intent: intent,
		Owner: 1, Repository: journeyRepo, Objective: "Build a request form",
		State: "accepted", MaxIssues: 4, CreatedUnix: 2, Predecessor: revision,
		Ideas: []factory.ProductIdea{{ID: "d333333333333333333333333", Description: "Request form", Consequence: "Requests are recorded", Decision: "accept", Source: "requested", Provenance: "human", Actor: 1}},
		Tasks: []factory.IntentTask{{Key: "d555555555555555555555555", Objective: "Other task", Outcomes: []string{"Other"}, IdeaIDs: []string{"d333333333333333333333333"}}},
	}
	if err := fx.db.SaveIntentRevision(context.Background(), other); err != nil {
		t.Fatal(err)
	}
	var revisionRaw []byte
	if err := fx.raw.QueryRow(`SELECT data FROM factory_intent_revisions WHERE id=$1`, second).Scan(&revisionRaw); err != nil {
		t.Fatal(err)
	}
	var adopted factory.IntentRevision
	if err := json.Unmarshal(revisionRaw, &adopted); err != nil {
		t.Fatal(err)
	}
	unrelated := factory.DeliverySubject{
		IntentID: intent, AcceptedRevision: second,
		ScopeDigest: adopted.Digest(), SourceCommit: strings.Repeat("a", 40),
		PreparationDigest: strings.Repeat("4", 64), VerificationManifestDigest: strings.Repeat("1", 64),
	}
	unrelated.DeliveryID = factory.DeliveryID(unrelated)
	report := factory.AppVerificationReport{
		DeliveryID: unrelated.DeliveryID, InputDigest: strings.Repeat("9", 64),
		Outcome:       factory.AppVerificationVerified,
		CheckReceipts: []factory.VerificationCheckReceipt{{CheckID: factory.ApprovedAppCheckID, Outcome: factory.CheckReceiptPassed, ArtifactDigest: strings.Repeat("e", 64), LogExcerpt: "ok"}},
	}
	late := factory.Assignment{
		ID: strings.Repeat("9", 32), AttemptRoot: strings.Repeat("9", 32),
		ProjectID: "p" + strings.Repeat("1", 24), Role: project.RoleReviewer,
		Phase: project.ExecutionVerify, Stage: factory.AssignmentFinished, Outcome: factory.Succeeded,
		Subject:      factory.ExecutionSubject{Kind: factory.SubjectDelivery, Delivery: &unrelated},
		Preparation:  "f" + strings.Repeat("b", 24),
		SourceCommit: unrelated.SourceCommit, Run: strings.Repeat("1", 32),
		RunHistory: []string{strings.Repeat("1", 32)},
		Repository: journeyRepo, CreatedUnix: 1700000001, ActorID: 1,
		Result: &factory.AssignmentResult{
			AssignmentID: strings.Repeat("9", 32), RunID: strings.Repeat("1", 32),
			Phase: project.ExecutionVerify, Status: "completed",
			Summary: "ok", Reported: true, RecordedUnix: 1700000001,
			PhaseResult: &factory.PhaseResult{
				Version: factory.PhaseResultVersion, Phase: project.ExecutionVerify,
				Status: "completed", Summary: "ok",
				Output: &factory.PhaseOutput{Verification: &report},
			},
		},
	}
	insertJourneyRow(t, fx.raw, late, factory.AssignmentFinished)
	content := factory.FeedCommunication{
		ComponentID: "F10", Title: "Application outcomes verified",
		Summary:      "Other revision verified.",
		EvidenceRefs: []string{unrelated.SourceCommit, unrelated.VerificationManifestDigest, strings.Repeat("8", 64)},
	}
	feed, err := json.Marshal(map[string]any{
		"actor": "factory", "author_name": "Factory", "issue": 0,
		"assignment": late.ID, "at": 1700000001, "status": "application:verified",
		"communication": content, "event_identity": unrelated.DeliveryID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fx.raw.Exec(`INSERT INTO factory_feed_posts(id,repository,revision,author,body,idempotency,data) VALUES($1,$2,0,'factory',$3,$4,$5)`,
		unrelated.DeliveryID, journeyRepo, content.Title+"\n\n"+content.Summary, unrelated.DeliveryID, string(feed)); err != nil {
		t.Fatal(err)
	}
	code, body := call(http.MethodGet, "/api/repositories/7/factory/issues/3", "")
	if code != http.StatusOK {
		t.Fatalf("unrelated-newer status=%d body=%s", code, body)
	}
	journey := journeyIssueView(t, body).Journey
	if len(journey.Verifications) != 1 {
		t.Fatalf("unrelated-newer verifications: %+v", journey.Verifications)
	}
}
