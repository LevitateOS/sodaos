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
	"fmt"
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
	"github.com/levitateos/sodaos/internal/config"
	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/factory/control"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
)

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
	s, dsn, cleanup, err := store.OpenEphemeral(context.Background(), admin, nil)
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
	mux := http.NewServeMux()
	product.registerExtensionProductRoutes(mux)

	call := func(method, path, body string) (int, []byte) {
		t.Helper()
		return journeyCall(t, mux, method, path, body, "generation-1", "generation-1", true)
	}
	return &journeyHTTPFixture{api: product, db: s, raw: raw, mux: mux, permission: permission}, call
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

func journeyIssueView(t *testing.T, body []byte) factoryIssueView {
	t.Helper()
	var view factoryIssueView
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

func journeyFindingFixture() factory.Finding {
	return journeyFindingWithSeverity(factory.FindingSeverityMajor)
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
	if ok, err := fx.db.ReservePriorityJudgment(ctx, fingerprint, 1); err != nil || !ok {
		t.Fatalf("reserve judgment: %v %v", ok, err)
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

func journeyReasons(view factoryIssueView) map[string]bool {
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
	// Registered owner-route authentication and immutable lost-reply recovery.
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
	resolution := factory.ContextCurrentResolution{CommandID: factory.NewID(), RunID: run.ID, AssignmentID: capability.AssignmentID, Fingerprint: request.Fingerprint, ScopeDigest: request.ScopeDigest, RequiredDigest: request.RequiredDigest, Bundle: factory.ContextBundleCurrent, RecordedUnix: time.Now().Unix()}
	capability.ContextCurrentResolution = &resolution
	if err := fx.db.StageMCPCapability(ctx, capability); err != nil {
		t.Fatal(err)
	}
	payload, _ := json.Marshal(request)
	command := factory.Command{ID: resolution.CommandID, Type: factory.CommandContextCurrent, Target: "run/" + run.ID, Principal: "native:1", Payload: string(payload)}
	command.Digest = factory.SettingsDigest(command.Type, command.Target, command.Payload)
	if _, _, err := fx.db.RecordFactoryCommand(ctx, command, time.Now()); err != nil {
		t.Fatal(err)
	}
	outcome, _ := json.Marshal(resolution)
	if err := fx.db.FinishFactoryCommand(ctx, command.ID, string(outcome), time.Now()); err != nil {
		t.Fatal(err)
	}
	actionPath := "/api/factory/runs/" + run.ID + "/actions"
	body, _ := json.Marshal(factoryRunActionsRequest{CommandID: command.ID, Action: factory.ActionContextCurrent, Context: &request})
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
			t.Fatalf("context owner exact receipt recovery: %d %s", code, reply)
		}
	}
	changed := request
	changed.ScopeDigest = strings.Repeat("0", 64)
	conflict, _ := json.Marshal(factoryRunActionsRequest{CommandID: command.ID, Action: factory.ActionContextCurrent, Context: &changed})
	if code, reply := call(http.MethodPost, actionPath, string(conflict)); code != 409 || journeyErrorCode(t, reply) != "command_conflict" {
		t.Fatalf("context owner changed scope replay: %d %s", code, reply)
	}
	var commands, choices int
	if err := fx.raw.QueryRow(`SELECT count(*) FROM factory_commands`).Scan(&commands); err != nil {
		t.Fatal(err)
	}
	if err := fx.raw.QueryRow(`SELECT count(*) FROM factory_priority_judgments`).Scan(&choices); err != nil || commands != 1 || choices != 0 {
		t.Fatal("HTTP context recovery/refusals refilled work or choices", commands, choices, err)
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
	completionBody, _ := json.Marshal(factoryActionsRequest{CommandID: completionCommand.ID, Action: factory.ActionCompletionRetry, Completion: &completion})
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
	changedBody, _ := json.Marshal(factoryActionsRequest{CommandID: completionCommand.ID, Action: factory.ActionCompletionRetry, Completion: &changedCompletion})
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
	if err := fx.raw.QueryRow(`SELECT count(*) FROM factory_priority_judgments`).Scan(&choices); err != nil || cycles != 2 || commands != 2 || choices != 1 {
		t.Fatal("completion retry replay refilled cycles or judgments", cycles, commands, choices, err)
	}
}

func TestFactoryIssueJourneyNoAttempt(t *testing.T) {
	_, call := journeyMux(t, &journeyAcceptanceSource{})
	code, body := call(http.MethodGet, "/api/repositories/7/factory/issues/3", "")
	if code != http.StatusOK {
		t.Fatalf("no-attempt status=%d body=%s", code, body)
	}
	view := journeyIssueView(t, body)
	journey := view.Journey
	if journey.Available || journey.Complete {
		t.Fatalf("no-attempt flags: %+v", journey)
	}
	if len(journey.Reasons) != 1 || journey.Reasons[0] != journeyReasonNoAttempt {
		t.Fatalf("no-attempt reasons: %+v", journey.Reasons)
	}
	if journey.Repository != "7" || journey.Issue != "3" || journey.ObservedUnix <= 0 {
		t.Fatalf("no-attempt identity: %+v", journey)
	}
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
	if len(journey.Reasons) != 2 || journey.Reasons[0] != journeyReasonAdoptionsUnavailable || journey.Reasons[1] != journeyReasonDeliveryUnavailable {
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
	if len(journey.Phases) != 2 || journey.Phases[0].Phase != project.ExecutionImplement || journey.Phases[0].State != journeyComplete ||
		journey.Phases[1].Phase != project.ExecutionReview || journey.Phases[1].State != journeyAttention {
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
	if len(journey.Tasks) != 3 || journey.Tasks[0].Key != "design" || journey.Tasks[0].State != journeyComplete ||
		journey.Tasks[1].Key != "build" || journey.Tasks[1].State != journeyActive ||
		journey.Tasks[2].Key != "docs" || journey.Tasks[2].State != journeyPending {
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
	if len(journey.Judgments) != 1 || journey.Judgments[0].Kind != journeyJudgmentReassessDecision ||
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
	if reserved, err := fx.db.ReserveReassessmentCall(ctx, successorID, factory.ReassessmentDeciding,
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
	if ok, err := fx.db.ReservePriorityJudgment(ctx, strings.Repeat("b", 64), 1); err != nil || !ok {
		t.Fatalf("reserve: %v %v", ok, err)
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
		journeyReasonCorrectionsInvalidRow, journeyReasonFindingsInvalidRow,
		journeyReasonJudgmentsInvalidRow, journeyReasonGatesUnrenderable,
		journeyReasonAdoptionsUnavailable, journeyReasonDeliveryUnavailable,
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
	state := journeyComplete
	if at%7 == 6 {
		state = journeyAttention
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
func journeyExpectCurrentPhases(t *testing.T, journey *factoryIssueJourneyView, root factory.Assignment, n int) {
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
	if reasons[journeyReasonChainPageBound] {
		t.Fatalf("tail hit the page bound early: %+v", journey.Reasons)
	}
	// 301 valid rows on one root: every canonical phase renders its
	// newest state with exact refs — no old capped prefix stands in
	// for current, and no phase repeats or displaces another.
	journeyExpectCurrentPhases(t, journey, root, n)
	for _, unwanted := range []string{
		journeyReasonPhasesInvalidRow, journeyReasonPhasesTruncated,
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
	const n = 4*journeyChainPageLimit + 10
	root := seedJourneyChain(t, fx, acceptance, n)

	code, body := call(http.MethodGet, "/api/repositories/7/factory/issues/3", "")
	if code != http.StatusOK {
		t.Fatalf("bound status=%d body=%s", code, body)
	}
	journey := journeyIssueView(t, body).Journey
	if !journey.Available || journey.Complete {
		t.Fatalf("bound flags: %+v", journey)
	}
	if reasons := journeyReasons(journeyIssueView(t, body)); !reasons[journeyReasonChainPageBound] {
		t.Fatalf("bound missing page reason: %+v", journey.Reasons)
	}
	// The page bound cuts only the retained cursor trail: the phase
	// section still reads newest-per-phase across the whole scope, so
	// overflow past the 1024-row chain prefix cannot mask a newer
	// failed or current state.
	journeyExpectCurrentPhases(t, journey, root, n)
	if journey.Lineage.NextCursor == "" || len(journey.Lineage.ChainCursors) != journeyChainMaxPages {
		t.Fatalf("bound resume refs: %+v", journey.Lineage)
	}
}

func journeyStoreCounts(t *testing.T, raw *sql.DB) (map[string]int64, string) {
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
	var spent int64
	if err := raw.QueryRow(`SELECT COALESCE(sum(spent),0) FROM factory_judgment_budget`).Scan(&spent); err != nil {
		t.Fatalf("budget spent: %v", err)
	}
	return counts, fmt.Sprintf("%d", spent)
}

func TestFactoryIssueJourneyReadsChangeNothing(t *testing.T) {
	fake := &journeyAcceptanceSource{}
	fx, call := journeyMux(t, fake)
	seedRichJourney(t, fx, fake)
	path := "/api/repositories/7/factory/issues/3"
	if code, body := call(http.MethodGet, path, ""); code != http.StatusOK {
		t.Fatalf("warmup status=%d body=%s", code, body)
	}
	before, spentBefore := journeyStoreCounts(t, fx.raw)
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
	after, spentAfter := journeyStoreCounts(t, fx.raw)
	for table, count := range before {
		if after[table] != count {
			t.Fatalf("table %s moved %d -> %d", table, count, after[table])
		}
	}
	if spentAfter != spentBefore {
		t.Fatalf("judgment budget moved %s -> %s", spentBefore, spentAfter)
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
	if got.Phase != project.ExecutionImplement || got.State != journeyAttention {
		t.Fatalf("newest-failed state: %+v", got)
	}
	if got.Assignment != newest.ID || got.Run != newest.Run || got.AttemptRoot != root.ID {
		t.Fatalf("newest-failed refs: %+v want %s/%s/%s", got, newest.ID, newest.Run, root.ID)
	}
	if len(journey.Reasons) != 2 || journey.Reasons[0] != journeyReasonAdoptionsUnavailable || journey.Reasons[1] != journeyReasonDeliveryUnavailable {
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
	if got.Phase != project.ExecutionImplement || got.State != journeyAttention {
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
	if got.Phase != project.ExecutionCorrect || got.State != journeyComplete {
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
	if reasons := journeyReasons(view); !reasons[journeyReasonPhasesInvalidRow] {
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
	if reasons := journeyReasons(view); !reasons[journeyReasonPhasesInvalidRow] {
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
	receipt, err := factory.NewDeliveryReceipt(in, fingerprint, answer, true, 41)
	if err != nil {
		t.Fatal(err)
	}
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
// asked, and one intent adoption binding.
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
	if withReceipt {
		seedJourneyDeliveryReceipt(t, fx, root)
	}
	return intent, revision, root
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
		journey.Adoptions[0].Revision != revision || journey.Adoptions[0].State != journeyAdoptionAdopted {
		t.Fatalf("evidence adoptions: %+v", journey.Adoptions)
	}
	if len(journey.Judgments) != 2 || journey.Judgments[0].Kind != journeyJudgmentDelivery ||
		journey.Judgments[0].Selected != journeyJudgmentDeliveryReady ||
		journey.Judgments[0].Model != "test-delivery-model" || journey.Judgments[0].Probability != 0.9 ||
		journey.Judgments[1].Kind != journeyJudgmentReassessDecision {
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
	if len(journey.Reasons) != 1 || journey.Reasons[0] != journeyReasonDeliveryUnavailable {
		t.Fatalf("missing-receipt reasons: %+v", journey.Reasons)
	}
	if len(journey.Judgments) != 1 || journey.Judgments[0].Kind != journeyJudgmentReassessDecision {
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
	if len(journey.Judgments) != 1 || journey.Judgments[0].Kind != journeyJudgmentReassessDecision {
		t.Fatalf("clean-delivery judgments: %+v", journey.Judgments)
	}
}
