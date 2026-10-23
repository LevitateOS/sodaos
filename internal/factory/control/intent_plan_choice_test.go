package control

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	sdk "forgejo.org/extension-sdk"
	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
)

// choicePlanIDs pins the intent, reviewed draft and accepted successor
// for the recorded-plan choice route.
var choicePlanIDs = struct {
	intent   string
	draft    string
	accepted string
}{"d100000000000000000000000", "d200000000000000000000000", "d300000000000000000000000"}

// choicePlanDraft is the reviewed draft behind the accepted revision:
// the options-supported objective with attributed effective scope.
func choicePlanDraft(repo int64) factory.IntentRevision {
	idea := func(id, desc, consequence string) factory.ProductIdea {
		return factory.ProductIdea{ID: id, Description: desc, Consequence: consequence, Decision: "accept", Source: "requested", Provenance: "human", Actor: 7}
	}
	return factory.IntentRevision{
		ID: choicePlanIDs.draft, Intent: choicePlanIDs.intent, Owner: 7, Repository: repo,
		Objective:         "Build a leave approval workflow so managers can review time-off requests",
		State:             "draft",
		MaxIssues:         10,
		PlanningCalls:     12,
		CreatedUnix:       1,
		EffectiveOutcomes: []string{"d555555555555555555555555", "d666666666666666666666666"},
		Ideas: []factory.ProductIdea{
			idea("d555555555555555555555555", "Capture leave requests", "Requests can be recorded"),
			idea("d666666666666666666666666", "Managers review requests", "Decisions are visible"),
		},
	}
}

// choicePlanAccepted carries the reviewed draft as its predecessor,
// preserving the reviewed substantive scope exactly: the objective,
// repository and product choices match the draft, while the authored
// tasks differ from any planner candidate so adoption proves the
// choice came from the recorded answer.
func choicePlanAccepted(t *testing.T, repo int64) factory.IntentRevision {
	t.Helper()
	r := broadJoinAcceptedRevision()
	r.ID, r.Intent, r.Repository, r.Predecessor = choicePlanIDs.accepted, choicePlanIDs.intent, repo, choicePlanIDs.draft
	r.Objective = choicePlanDraft(repo).Objective
	if err := r.Validate(); err != nil {
		t.Fatalf("accepted head invalid: %v", err)
	}
	return r
}

// choicePlanCoordinator wires funded comparative judgments behind the
// counting stub; the graph stays unwired unless the test drives the
// full generate tail.
func choicePlanCoordinator(t *testing.T, db *store.Store, fx dispatchFixture, stub ProductJudgment) *Coordinator {
	t.Helper()
	c := &Coordinator{Store: db, Host: fx.host}
	if err := c.Store.SavePlanningAllowance(context.Background(), factory.PlanningAllowance{ID: "d999999999999999999999999", Intent: choicePlanIDs.intent, Owner: 7, Amount: 32, Unit: factory.PlanningAllowanceUnit, Ceiling: 32, CreatedUnix: 1}); err != nil {
		t.Fatal(err)
	}
	c.Planning = &PlanningJudgments{Journal: c.Store, Funding: c.Store, Chooser: stub}
	return c
}

func choicePlanAnswer(t *testing.T, draft factory.IntentRevision, status string, keys ...string) string {
	t.Helper()
	out := choicePlannerOutput(draft, keys...)
	return fencedResult(t, factory.PhaseResult{
		Version: factory.PhaseResultVersion, Phase: project.ExecutionDecompose, Status: status,
		Summary: "recorded choice ready", Output: &factory.PhaseOutput{Decomposition: &out},
	})
}

// A single recorded candidate resolves with zero comparative calls:
// the funded chooser never evaluates.
func TestResolveAcceptedPlanChoiceSingleNeedsNoComparativeCall(t *testing.T) {
	ctx := context.Background()
	db, _ := dispatchTestDB(t)
	fx := dispatchSeed(t, db)
	draft := choicePlanDraft(fx.repo)
	if _, ok := settlePlanAnswerWithOperation(t, ctx, db, fx, draft, factory.IntentPlanningOptions, choicePlanAnswer(t, draft, "completed", "cand-a")); !ok {
		t.Fatal("recorded plan did not settle")
	}
	accepted := choicePlanAccepted(t, fx.repo)
	if err := db.SaveIntentRevision(ctx, accepted); err != nil {
		t.Fatal(err)
	}
	stub := &candidateChoiceStub{selected: "cand-a", confidence: 1}
	c := choicePlanCoordinator(t, db, fx, stub)
	chosen, wait, err := c.ResolveAcceptedPlanChoice(ctx, 7, accepted)
	if err != nil || wait != nil {
		t.Fatalf("resolve err=%v wait=%+v", err, wait)
	}
	if chosen.Key != "cand-a" || chosen.Intent != choicePlanIDs.intent || chosen.Revision != choicePlanIDs.draft || stub.calls != 0 {
		t.Fatalf("choice=%+v fundedCalls=%d", chosen, stub.calls)
	}
}

// Multiple recorded alternatives resolve through exactly one funded
// comparative choice, and a second resolution replays cached without
// spending further.
func TestResolveAcceptedPlanChoiceMultiUsesOneFundedChoice(t *testing.T) {
	ctx := context.Background()
	db, _ := dispatchTestDB(t)
	fx := dispatchSeed(t, db)
	draft := choicePlanDraft(fx.repo)
	if _, ok := settlePlanAnswerWithOperation(t, ctx, db, fx, draft, factory.IntentPlanningOptions, choicePlanAnswer(t, draft, "completed", "cand-a", "cand-b")); !ok {
		t.Fatal("recorded plan did not settle")
	}
	accepted := choicePlanAccepted(t, fx.repo)
	if err := db.SaveIntentRevision(ctx, accepted); err != nil {
		t.Fatal(err)
	}
	stub := &candidateChoiceStub{selected: "cand-b", confidence: 0.9}
	c := choicePlanCoordinator(t, db, fx, stub)
	chosen, wait, err := c.ResolveAcceptedPlanChoice(ctx, 7, accepted)
	if err != nil || wait != nil {
		t.Fatalf("resolve err=%v wait=%+v", err, wait)
	}
	if chosen.Key != "cand-b" || stub.calls != 1 {
		t.Fatalf("choice=%+v fundedCalls=%d", chosen, stub.calls)
	}
	again, wait, err := c.ResolveAcceptedPlanChoice(ctx, 7, accepted)
	if err != nil || wait != nil || again.Key != "cand-b" || stub.calls != 1 {
		t.Fatalf("replay choice=%+v err=%v wait=%+v fundedCalls=%d", again, err, wait, stub.calls)
	}
}

// A still-running recorded plan waits truthfully instead of refusing
// or funding a choice around the outstanding run.
func TestResolveAcceptedPlanChoicePendingWaits(t *testing.T) {
	ctx := context.Background()
	db, _ := dispatchTestDB(t)
	fx := dispatchSeed(t, db)
	draft := choicePlanDraft(fx.repo)
	launchPlanRun(t, ctx, db, fx, draft, factory.IntentPlanningOptions)
	accepted := choicePlanAccepted(t, fx.repo)
	if err := db.SaveIntentRevision(ctx, accepted); err != nil {
		t.Fatal(err)
	}
	stub := &candidateChoiceStub{selected: "cand-a", confidence: 1}
	c := choicePlanCoordinator(t, db, fx, stub)
	chosen, wait, err := c.ResolveAcceptedPlanChoice(ctx, 7, accepted)
	if err != nil || wait == nil {
		t.Fatalf("pending plan must wait: choice=%+v wait=%+v err=%v", chosen, wait, err)
	}
	if stub.calls != 0 {
		t.Fatalf("pending plan funded %d comparative calls", stub.calls)
	}
}

// An accepted direction whose reviewed draft never planned refuses
// without funding: there is no completed result to consume.
func TestResolveAcceptedPlanChoiceMissingRefuses(t *testing.T) {
	ctx := context.Background()
	db, _ := dispatchTestDB(t)
	fx := dispatchSeed(t, db)
	draft := choicePlanDraft(fx.repo)
	if err := db.SaveIntentRevision(ctx, draft); err != nil {
		t.Fatal(err)
	}
	accepted := choicePlanAccepted(t, fx.repo)
	if err := db.SaveIntentRevision(ctx, accepted); err != nil {
		t.Fatal(err)
	}
	stub := &candidateChoiceStub{selected: "cand-a", confidence: 1}
	c := choicePlanCoordinator(t, db, fx, stub)
	if _, _, err := c.ResolveAcceptedPlanChoice(ctx, 7, accepted); err == nil ||
		errors.Is(err, ErrPlanChoiceStale) || errors.Is(err, ErrPlanChoiceAttention) {
		t.Fatalf("missing plan must refuse plainly: %v", err)
	}
	if stub.calls != 0 {
		t.Fatalf("missing plan funded %d comparative calls", stub.calls)
	}
}

// An accepted direction with no reviewed draft refuses: Generate never
// invents a plan the review never saw.
func TestResolveAcceptedPlanChoiceWithoutPredecessorRefuses(t *testing.T) {
	ctx := context.Background()
	db, _ := dispatchTestDB(t)
	fx := dispatchSeed(t, db)
	accepted := choicePlanAccepted(t, fx.repo)
	accepted.Predecessor = ""
	if err := db.SaveIntentRevision(ctx, accepted); err != nil {
		t.Fatal(err)
	}
	stub := &candidateChoiceStub{selected: "cand-a", confidence: 1}
	c := choicePlanCoordinator(t, db, fx, stub)
	if _, _, err := c.ResolveAcceptedPlanChoice(ctx, 7, accepted); err == nil {
		t.Fatal("predecessor-free acceptance admitted a plan choice")
	}
	if stub.calls != 0 {
		t.Fatalf("predecessor-free acceptance funded %d comparative calls", stub.calls)
	}
}

// A recorded attention answer surfaces its product decision instead of
// funding a choice past it.
func TestResolveAcceptedPlanChoiceAttentionRefuses(t *testing.T) {
	ctx := context.Background()
	db, _ := dispatchTestDB(t)
	fx := dispatchSeed(t, db)
	draft := choicePlanDraft(fx.repo)
	attention := "Decide whether managers review in the same queue before issues can split."
	answer := fencedResult(t, factory.PhaseResult{
		Version: factory.PhaseResultVersion, Phase: project.ExecutionDecompose, Status: "blocked",
		Summary: "waiting on product decision",
		Output: &factory.PhaseOutput{Decomposition: &factory.SemanticPlannerOutput{
			Intent: draft.Intent, Revision: draft.ID, Operation: factory.IntentPlanningOptions, Attention: attention,
		}},
	})
	if _, ok := settlePlanAnswerWithOperation(t, ctx, db, fx, draft, factory.IntentPlanningOptions, answer); !ok {
		t.Fatal("attention answer did not settle")
	}
	accepted := choicePlanAccepted(t, fx.repo)
	if err := db.SaveIntentRevision(ctx, accepted); err != nil {
		t.Fatal(err)
	}
	stub := &candidateChoiceStub{selected: "cand-a", confidence: 1}
	c := choicePlanCoordinator(t, db, fx, stub)
	_, _, err := c.ResolveAcceptedPlanChoice(ctx, 7, accepted)
	if !errors.Is(err, ErrPlanChoiceAttention) || !strings.Contains(err.Error(), attention) {
		t.Fatalf("attention must surface its decision: %v", err)
	}
	if stub.calls != 0 {
		t.Fatalf("attention funded %d comparative calls", stub.calls)
	}
}

// A failed recorded run refuses with its recorded summary: failure is
// retried at the plan, never papered over by a choice.
func TestResolveAcceptedPlanChoiceFailedRefuses(t *testing.T) {
	ctx := context.Background()
	db, _ := dispatchTestDB(t)
	fx := dispatchSeed(t, db)
	draft := choicePlanDraft(fx.repo)
	run := launchPlanRun(t, ctx, db, fx, draft, factory.IntentPlanningOptions)
	run.Outcome, run.Summary, run.Reconciled = factory.Failed, "provider failed", true
	if err := db.SaveFactoryRun(ctx, run); err != nil {
		t.Fatal(err)
	}
	finished, ok := AccountSettledRun(ctx, db, run, "", time.Now())
	if !ok || finished.Outcome != factory.Failed {
		t.Fatalf("failed run not charged: %+v %v", finished, ok)
	}
	accepted := choicePlanAccepted(t, fx.repo)
	if err := db.SaveIntentRevision(ctx, accepted); err != nil {
		t.Fatal(err)
	}
	stub := &candidateChoiceStub{selected: "cand-a", confidence: 1}
	c := choicePlanCoordinator(t, db, fx, stub)
	_, _, err := c.ResolveAcceptedPlanChoice(ctx, 7, accepted)
	if err == nil || !strings.Contains(err.Error(), "failed") {
		t.Fatalf("failed plan must refuse: %v", err)
	}
	if stub.calls != 0 {
		t.Fatalf("failed plan funded %d comparative calls", stub.calls)
	}
}

// A source move after settlement refuses as stale: the recorded plan
// inspects a commit the current preparation no longer carries.
func TestResolveAcceptedPlanChoiceStaleSourceRefuses(t *testing.T) {
	ctx := context.Background()
	db, _ := dispatchTestDB(t)
	fx := dispatchSeed(t, db)
	draft := choicePlanDraft(fx.repo)
	if _, ok := settlePlanAnswerWithOperation(t, ctx, db, fx, draft, factory.IntentPlanningOptions, choicePlanAnswer(t, draft, "completed", "cand-a")); !ok {
		t.Fatal("recorded plan did not settle")
	}
	accepted := choicePlanAccepted(t, fx.repo)
	if err := db.SaveIntentRevision(ctx, accepted); err != nil {
		t.Fatal(err)
	}
	rotateChoicePlanSource(t, ctx, db, fx, strings.Repeat("d", 40))
	stub := &candidateChoiceStub{selected: "cand-a", confidence: 1}
	c := choicePlanCoordinator(t, db, fx, stub)
	_, _, err := c.ResolveAcceptedPlanChoice(ctx, 7, accepted)
	if !errors.Is(err, ErrPlanChoiceStale) {
		t.Fatalf("moved source must refuse as stale: %v", err)
	}
	if stub.calls != 0 {
		t.Fatalf("stale source funded %d comparative calls", stub.calls)
	}
}

// rotateChoicePlanSource admits a fresh requirement, approval and ready
// coder preparation on the given commit so the recorded plan goes stale.
func rotateChoicePlanSource(t *testing.T, ctx context.Context, db *store.Store, fx dispatchFixture, commit string) {
	t.Helper()
	digest := strings.Repeat("e", 64)
	if err := db.AdmitRequirementDecision(ctx, project.RequirementDecision{
		ID: "d444444444444444444444444", Project: fx.proj, Approver: 7,
		Predecessor:  "d111111111111111111111111",
		SourceCommit: commit, SetupDigest: digest, InputsDigest: digest,
	}); err != nil {
		t.Fatal(err)
	}
	if err := db.AdmitApprovalDecision(ctx, project.ApprovalDecision{
		ID: "d555555555555555555555555", Project: fx.proj, Requirement: "d444444444444444444444444",
		Predecessor: "d222222222222222222222222",
		Approver:    7, EffectsDigest: digest, ReadinessDigest: digest, Verified: true,
	}); err != nil {
		t.Fatal(err)
	}
	prep := project.StoredPreparation{Preparation: project.Preparation{
		ID: "f333333333333333333333333", Project: fx.proj, Role: project.RoleCoder,
		Requirements: project.RequirementAcceptance{ID: "d444444444444444444444444", Revision: 2, Approver: 7, SourceCommit: commit, Digest: digest},
		Approval:     project.AdminApproval{ID: "d555555555555555555555555", Revision: 2, Approver: 7, EffectsDigest: digest},
		SourceCommit: commit, SetupDigest: digest, Tools: []string{"python3"},
	}}
	prep.State = project.PrepareState{ID: prep.Preparation.ID, Project: fx.proj, Role: project.RoleCoder, Phase: project.PrepareReady, Ready: true}
	if _, _, err := db.AdmitPreparation(ctx, prep); err != nil {
		t.Fatal(err)
	}
}

// rotatingGraphFixture rotates the recorded source once during the first
// external ownership read, proving late source movement between choice
// resolution and native submission refuses before any native effect.
type rotatingGraphFixture struct {
	inner   *generatedGraphFixture
	rotated bool
	rotate  func()
}

func (f *rotatingGraphFixture) CheckIntentOwner(ctx context.Context, repo, owner, actor int64) (int64, error) {
	if !f.rotated {
		f.rotated = true
		f.rotate()
	}
	return f.inner.CheckIntentOwner(ctx, repo, owner, actor)
}

func (f *rotatingGraphFixture) SubmitGraph(ctx context.Context, g factory.IntentGeneration, repo int64) (sdk.OperationRecord, error) {
	return f.inner.SubmitGraph(ctx, g, repo)
}

func (f *rotatingGraphFixture) LookupGraph(ctx context.Context, g factory.IntentGeneration, repo int64) (sdk.OperationLookup, error) {
	return f.inner.LookupGraph(ctx, g, repo)
}

func (f *rotatingGraphFixture) CancelGraph(ctx context.Context, id string) (sdk.OperationRecord, error) {
	return f.inner.CancelGraph(ctx, id)
}

func (f *rotatingGraphFixture) ReadGeneratedIssue(ctx context.Context, repo, index, actor int64) (sdk.NativeSnapshot, int64, error) {
	return f.inner.ReadGeneratedIssue(ctx, repo, index, actor)
}

func (f *rotatingGraphFixture) ReadAcceptanceEvidence(ctx context.Context, revision, issue string, edges []string) (AcceptanceEvidence, error) {
	return f.inner.ReadAcceptanceEvidence(ctx, revision, issue, edges)
}

// A source rotation landing during the first external ownership read —
// after choice resolution but before native submission — refuses as
// stale with zero native submissions: the pre-submit boundary
// re-proves the stamped binding after the slow external reads.
func TestRecordedPlanSourceChangesBeforeNativeSubmit(t *testing.T) {
	ctx := context.Background()
	db, _ := dispatchTestDB(t)
	fx := dispatchSeed(t, db)
	draft := choicePlanDraft(fx.repo)
	if _, ok := settlePlanAnswerWithOperation(t, ctx, db, fx, draft, factory.IntentPlanningOptions, choicePlanAnswer(t, draft, "completed", "cand-a")); !ok {
		t.Fatal("recorded plan did not settle")
	}
	accepted := choicePlanAccepted(t, fx.repo)
	if err := db.SaveIntentRevision(ctx, accepted); err != nil {
		t.Fatal(err)
	}
	stub := &candidateChoiceStub{selected: "cand-a", confidence: 1}
	c := choicePlanCoordinator(t, db, fx, stub)
	inner := &generatedGraphFixture{revision: 10, actor: 91, owner: true, records: map[string]sdk.OperationRecord{}, issues: map[int64]sdk.NativeSnapshot{}}
	graph := &rotatingGraphFixture{inner: inner, rotate: func() {
		rotateChoicePlanSource(t, ctx, db, fx, strings.Repeat("d", 40))
	}}
	c.Graph, c.GraphActorID, c.AcceptanceReads = graph, 91, graph
	_, _, err := c.GenerateAcceptedIntentFromRecordedPlan(ctx, choicePlanIDs.intent, 7)
	if err == nil || (!strings.Contains(err.Error(), "source") && !strings.Contains(err.Error(), "preparation")) {
		t.Fatalf("late source movement must refuse as stale: %v", err)
	}
	if inner.submissions != 0 {
		t.Fatalf("late source movement submitted %d native operations", inner.submissions)
	}
	if stub.calls != 0 {
		t.Fatalf("late source movement funded %d comparative calls", stub.calls)
	}
}

// A pre-binding pending operation behind a recorded plan resumes
// through recovery with its durable original binding stamped: the
// resumed submission validates the same recorded assignment and
// completes exactly once.
func TestRecoverIntentGraphsPreservesOriginalPendingPlan(t *testing.T) {
	ctx := context.Background()
	db, _ := dispatchTestDB(t)
	fx := dispatchSeed(t, db)
	draft := choicePlanDraft(fx.repo)
	if _, ok := settlePlanAnswerWithOperation(t, ctx, db, fx, draft, factory.IntentPlanningOptions, choicePlanAnswer(t, draft, "completed", "cand-a")); !ok {
		t.Fatal("recorded plan did not settle")
	}
	accepted := choicePlanAccepted(t, fx.repo)
	if err := db.SaveIntentRevision(ctx, accepted); err != nil {
		t.Fatal(err)
	}
	inner := &generatedGraphFixture{revision: 10, actor: 91, owner: true, records: map[string]sdk.OperationRecord{}, issues: map[int64]sdk.NativeSnapshot{}}
	submitted := false
	inner.beforeSubmit = func(factory.IntentGeneration) error {
		if !submitted {
			submitted = true
			return errors.New("transient native failure")
		}
		return nil
	}
	stub := &candidateChoiceStub{selected: "cand-a", confidence: 1}
	c := choicePlanCoordinator(t, db, fx, stub)
	c.Graph, c.GraphActorID, c.AcceptanceReads = inner, 91, inner
	if _, _, err := c.GenerateAcceptedIntentFromRecordedPlan(ctx, choicePlanIDs.intent, 7); err == nil {
		t.Fatal("expected transient native failure")
	}
	frozen, err := db.IntentGeneration(ctx, accepted.ID)
	if err != nil || frozen.State != "pending" || frozen.PlanAssignment == "" {
		t.Fatalf("legacy row not frozen pending and unbound: %+v %v", frozen, err)
	}
	if err := c.RecoverIntentGraphs(ctx); err != nil {
		t.Fatalf("recovery bind failed: %v", err)
	}
	resumed, err := db.IntentGeneration(ctx, accepted.ID)
	if err != nil || resumed.State != "completed" {
		t.Fatalf("resumed=%+v err=%v", resumed, err)
	}
	attempt, err := db.IntentPlanAttempt(ctx, choicePlanIDs.intent, choicePlanIDs.draft)
	if err != nil {
		t.Fatal(err)
	}
	if resumed.PlanAssignment != attempt.ID {
		t.Fatalf("resumed binding=%q want recorded attempt %q", resumed.PlanAssignment, attempt.ID)
	}
	if inner.submissions != 1 {
		t.Fatalf("resumed submissions=%d want exactly one native effect", inner.submissions)
	}
}

// A pre-binding pending operation with no recorded plan behind its
// accepted head refuses through recovery instead of submitting: the
// unprovable operation is cancelled and reported, never observed.
func TestRecoverIntentGraphsRefusesUnboundPlanlessLegacy(t *testing.T) {
	ctx := context.Background()
	db, _ := dispatchTestDB(t)
	fx := dispatchSeed(t, db)
	draft := choicePlanDraft(fx.repo)
	if err := db.SaveIntentRevision(ctx, draft); err != nil {
		t.Fatal(err)
	}
	accepted := choicePlanAccepted(t, fx.repo)
	if err := db.SaveIntentRevision(ctx, accepted); err != nil {
		t.Fatal(err)
	}
	inner := &generatedGraphFixture{revision: 10, actor: 91, owner: true, records: map[string]sdk.OperationRecord{}, issues: map[int64]sdk.NativeSnapshot{}}
	submitted := false
	inner.beforeSubmit = func(factory.IntentGeneration) error {
		if !submitted {
			submitted = true
			return errors.New("transient native failure")
		}
		return nil
	}
	stub := &candidateChoiceStub{selected: "cand-a", confidence: 1}
	c := choicePlanCoordinator(t, db, fx, stub)
	c.Graph, c.GraphActorID, c.AcceptanceReads = inner, 91, inner
	seedUnboundGeneration(t, c, accepted)
	if err := c.RecoverIntentGraphs(ctx); err == nil || !strings.Contains(err.Error(), "no recorded plan binding") {
		t.Fatalf("unbound operation must refuse: %v", err)
	}
	if inner.submissions != 0 {
		t.Fatalf("plan-less legacy submitted %d native operations", inner.submissions)
	}
	refused, err := db.IntentGeneration(ctx, accepted.ID)
	if err != nil || refused.State != "refused" {
		t.Fatalf("plan-less row=%+v err=%v want terminal refusal", refused, err)
	}
}

// Resolve through generate adopts exactly the recorded single-candidate
// graph with no comparative call: the bound connector runs the full
// serving tail behind the resolved choice.
func TestGenerateFromResolvedPlanChoiceAdoptsExactly(t *testing.T) {
	ctx := context.Background()
	db, _ := dispatchTestDB(t)
	fx := dispatchSeed(t, db)
	draft := choicePlanDraft(fx.repo)
	if _, ok := settlePlanAnswerWithOperation(t, ctx, db, fx, draft, factory.IntentPlanningOptions, choicePlanAnswer(t, draft, "completed", "cand-a")); !ok {
		t.Fatal("recorded plan did not settle")
	}
	accepted := choicePlanAccepted(t, fx.repo)
	if err := db.SaveIntentRevision(ctx, accepted); err != nil {
		t.Fatal(err)
	}
	stub := &candidateChoiceStub{selected: "cand-a", confidence: 1}
	c := choicePlanCoordinator(t, db, fx, stub)
	f := &generatedGraphFixture{revision: 10, actor: 91, owner: true, records: map[string]sdk.OperationRecord{}, issues: map[int64]sdk.NativeSnapshot{}}
	c.Graph, c.GraphActorID, c.AcceptanceReads = f, 91, f
	g, wait, err := c.GenerateAcceptedIntentFromRecordedPlan(ctx, choicePlanIDs.intent, 7)
	if err != nil || wait != nil {
		t.Fatalf("generate err=%v wait=%+v", err, wait)
	}
	if g.State != "completed" || stub.calls != 0 {
		t.Fatalf("state=%s fundedCalls=%d", g.State, stub.calls)
	}
	if keys := submittedKeys(g.SubmittedTasks); len(keys) == 0 || keys[0] != choiceTaskKey(1) {
		t.Fatalf("submitted=%v want recorded candidate tasks", keys)
	}
}

// A revised objective after review refuses before any funded choice:
// changing leave approvals to publishing payroll is another product
// direction, not planning currency the connector may spend on.
func TestResolveAcceptedPlanChoiceRevisedObjectiveRefuses(t *testing.T) {
	ctx := context.Background()
	db, _ := dispatchTestDB(t)
	fx := dispatchSeed(t, db)
	draft := choicePlanDraft(fx.repo)
	if _, ok := settlePlanAnswerWithOperation(t, ctx, db, fx, draft, factory.IntentPlanningOptions, choicePlanAnswer(t, draft, "completed", "cand-a", "cand-b")); !ok {
		t.Fatal("recorded plan did not settle")
	}
	accepted := choicePlanAccepted(t, fx.repo)
	accepted.Objective = "Publish payroll figures for every department each month"
	if err := db.SaveIntentRevision(ctx, accepted); err != nil {
		t.Fatal(err)
	}
	stub := &candidateChoiceStub{selected: "cand-b", confidence: 0.9}
	c := choicePlanCoordinator(t, db, fx, stub)
	_, _, err := c.ResolveAcceptedPlanChoice(ctx, 7, accepted)
	if !errors.Is(err, ErrPlanChoiceStale) || !strings.Contains(err.Error(), "reviewed objective") {
		t.Fatalf("revised objective must refuse as stale: %v", err)
	}
	if stub.calls != 0 {
		t.Fatalf("revised objective funded %d comparative calls", stub.calls)
	}
}

// A replacement preparation at the same commit still refuses: the
// recorded plan answers the exact preparation identity it settled,
// not merely any preparation carrying the same source commit.
func TestResolveAcceptedPlanChoiceReplacedPreparationRefuses(t *testing.T) {
	ctx := context.Background()
	db, _ := dispatchTestDB(t)
	fx := dispatchSeed(t, db)
	draft := choicePlanDraft(fx.repo)
	if _, ok := settlePlanAnswerWithOperation(t, ctx, db, fx, draft, factory.IntentPlanningOptions, choicePlanAnswer(t, draft, "completed", "cand-a")); !ok {
		t.Fatal("recorded plan did not settle")
	}
	accepted := choicePlanAccepted(t, fx.repo)
	if err := db.SaveIntentRevision(ctx, accepted); err != nil {
		t.Fatal(err)
	}
	rotateChoicePlanSource(t, ctx, db, fx, strings.Repeat("c", 40))
	stub := &candidateChoiceStub{selected: "cand-a", confidence: 1}
	c := choicePlanCoordinator(t, db, fx, stub)
	_, _, err := c.ResolveAcceptedPlanChoice(ctx, 7, accepted)
	if !errors.Is(err, ErrPlanChoiceStale) || !strings.Contains(err.Error(), "another preparation") {
		t.Fatalf("replaced preparation must refuse as stale: %v", err)
	}
	if stub.calls != 0 {
		t.Fatalf("replaced preparation funded %d comparative calls", stub.calls)
	}
}

// A head that moves during the funded comparison refuses after it:
// the choice resolved against a superseded direction never reaches
// native effects.
func TestResolveAcceptedPlanChoiceMovedHeadDuringChoiceRefuses(t *testing.T) {
	ctx := context.Background()
	db, _ := dispatchTestDB(t)
	fx := dispatchSeed(t, db)
	draft := choicePlanDraft(fx.repo)
	if _, ok := settlePlanAnswerWithOperation(t, ctx, db, fx, draft, factory.IntentPlanningOptions, choicePlanAnswer(t, draft, "completed", "cand-a", "cand-b")); !ok {
		t.Fatal("recorded plan did not settle")
	}
	accepted := choicePlanAccepted(t, fx.repo)
	if err := db.SaveIntentRevision(ctx, accepted); err != nil {
		t.Fatal(err)
	}
	stub := &candidateChoiceStub{selected: "cand-b", confidence: 0.9}
	c := choicePlanCoordinator(t, db, fx, &movingChoiceStub{T: t, store: db, accepted: accepted, inner: stub})
	_, _, err := c.ResolveAcceptedPlanChoice(ctx, 7, accepted)
	if !errors.Is(err, ErrPlanChoiceStale) {
		t.Fatalf("moved head must refuse as stale: %v", err)
	}
	if stub.calls != 1 {
		t.Fatalf("comparison ran %d times, want exactly the refused one", stub.calls)
	}
	if _, err := db.IntentGeneration(ctx, accepted.ID); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("moved head recorded a generation: %v", err)
	}
}

// movingChoiceStub delegates the comparison to the counting stub, then
// supersedes the accepted head before returning the answer — proving
// the post-comparison currency recheck observes the move.
type movingChoiceStub struct {
	T        *testing.T
	store    *store.Store
	accepted factory.IntentRevision
	inner    *candidateChoiceStub
}

func (s *movingChoiceStub) Select(ctx context.Context, state any, rubric string, choices map[string]any) (factory.PriorityAnswer, error) {
	answer, err := s.inner.Select(ctx, state, rubric, choices)
	if err != nil {
		return answer, err
	}
	next := s.accepted
	next.ID = "d333333333333333333333333"
	next.Predecessor = s.accepted.ID
	if err := s.store.SaveIntentRevision(ctx, next); err != nil {
		s.T.Fatalf("superseding head not recorded: %v", err)
	}
	return answer, nil
}

// A genuine assessed proposal can supply the recorded graph for an authored
// objective outside the canned alternatives, under the same source/head guards.
func TestResolveAcceptedProposalAdoptsRecordedGraph(t *testing.T) {
	ctx := context.Background()
	db, dsn := dispatchTestDB(t)
	fx := dispatchSeed(t, db)
	draft := choicePlanDraft(fx.repo)
	draft.Objective = "Build a private team task board with saved priority"
	draft.Ideas = draft.Ideas[:1]
	draft.Ideas[0].Description, draft.Ideas[0].Consequence = "Store team tasks and priority", "Team task state persists"
	draft.EffectiveOutcomes = []string{draft.Ideas[0].ID}
	draft.MaxIssues = 1
	if factory.DescribeAlternatives(draft.Objective).Supported {
		t.Fatal("authored task-board scope must not acquire a canned workflow")
	}
	answer := fencedResult(t, factory.PhaseResult{
		Version: factory.PhaseResultVersion, Phase: project.ExecutionDecompose, Status: "completed",
		Summary: "propose ready",
		Output: &factory.PhaseOutput{Decomposition: &factory.SemanticPlannerOutput{
			Intent: draft.Intent, Revision: draft.ID, Operation: factory.IntentPlanningPropose,
			Candidates: []factory.SemanticCandidate{{
				Key: "keep", Rationale: "Keep the recorded draft as the single review baseline.",
				Tasks: []factory.IntentTask{{
					Key: "d999999999999999999999999", Objective: draft.Objective,
					Outcomes: []string{"Team task state and priority persist"}, IdeaIDs: []string{draft.Ideas[0].ID},
				}},
			}},
		}},
	})
	if _, ok := settlePlanAnswerWithOperation(t, ctx, db, fx, draft, factory.IntentPlanningPropose, answer); !ok {
		t.Fatal("recorded propose plan did not settle")
	}
	accepted := draft
	accepted.ID, accepted.Predecessor, accepted.State = choicePlanIDs.accepted, draft.ID, "accepted"
	accepted.Tasks = []factory.IntentTask{{Key: "d888888888888888888888888", Objective: draft.Objective, Outcomes: []string{"Team task state persists"}, IdeaIDs: []string{draft.Ideas[0].ID}}}
	if err := db.SaveIntentRevision(ctx, accepted); err != nil {
		t.Fatal(err)
	}
	stub := &candidateChoiceStub{selected: "keep", confidence: 1}
	c := choicePlanCoordinator(t, db, fx, stub)
	f := &generatedGraphFixture{revision: 10, actor: 91, owner: true, records: map[string]sdk.OperationRecord{}, issues: map[int64]sdk.NativeSnapshot{}}
	c.Graph, c.GraphActorID, c.AcceptanceReads = f, 91, f
	generated, wait, err := c.GenerateAcceptedIntentFromRecordedPlan(ctx, draft.Intent, draft.Owner)
	if err != nil || wait != nil || generated.State != "completed" || len(generated.SubmittedTasks) != 1 || generated.SubmittedTasks[0].Key != "d999999999999999999999999" {
		t.Fatalf("assessed proposal did not generate its exact recorded graph: %+v wait=%+v err=%v", generated, wait, err)
	}
	if stub.calls != 0 {
		t.Fatalf("propose answer funded %d comparative calls", stub.calls)
	}
	bindings, err := db.IntentTaskBindings(ctx, accepted.ID)
	if err != nil || len(bindings) != 1 || bindings[0].TaskKey != generated.SubmittedTasks[0].Key {
		t.Fatalf("recorded graph adoption is unbound: %+v %v", bindings, err)
	}
	if candidates, err := db.DeliveredIntentCandidates(ctx, 16); err != nil || len(candidates) != 0 {
		t.Fatalf("unmerged generated graph admitted delivery: %+v %v", candidates, err)
	}
	sqlDB, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	binding := bindings[0]
	event := factory.CompletionEvent{ID: factory.NewID(), Repository: fx.repo, Issue: binding.IssueIndex, Generation: 1, PublicationID: factory.NewID(), AcceptanceID: binding.AcceptanceID, MergedCommit: fx.tip, ClosingOccurrence: "1"}
	raw, _ := json.Marshal(event)
	if _, err = sqlDB.Exec(`INSERT INTO factory_completions(repository,generation,event,publication,occurrence,data) VALUES($1,1,$2,$3,'1',$4)`, fx.repo, event.ID, event.PublicationID, string(raw)); err != nil {
		t.Fatal(err)
	}
	candidates, err := db.DeliveredIntentCandidates(ctx, 16)
	if err != nil || len(candidates) != 1 || candidates[0].Revision.ID != accepted.ID || candidates[0].FirstIssue != binding.IssueIndex {
		t.Fatalf("merged recorded graph lost application verification: %+v %v", candidates, err)
	}
	for _, invalid := range []string{"pending", "stale-scope", "uncovered-outcome"} {
		changed := generated
		switch invalid {
		case "pending":
			changed.State = "pending"
		case "stale-scope":
			changed.IntentDigest = strings.Repeat("f", 64)
		case "uncovered-outcome":
			changed.SubmittedTasks = append([]factory.IntentTask(nil), generated.SubmittedTasks...)
			changed.SubmittedTasks[0].IdeaIDs = nil
		}
		raw, _ := json.Marshal(changed)
		if _, err := sqlDB.Exec(`UPDATE factory_intent_generations SET data=$2 WHERE revision=$1`, accepted.ID, string(raw)); err != nil {
			t.Fatal(err)
		}
		if candidates, err := db.DeliveredIntentCandidates(ctx, 16); err != nil || len(candidates) != 0 {
			t.Fatalf("%s graph admitted application verification: %+v %v", invalid, candidates, err)
		}
	}
	raw, _ = json.Marshal(generated)
	if _, err := sqlDB.Exec(`UPDATE factory_intent_generations SET data=$2 WHERE revision=$1`, accepted.ID, string(raw)); err != nil {
		t.Fatal(err)
	}
	changedBinding := binding
	changedBinding.TaskKey = accepted.Tasks[0].Key
	raw, _ = json.Marshal(changedBinding)
	if _, err := sqlDB.Exec(`UPDATE factory_intent_tasks SET data=$3 WHERE revision=$1 AND task=$2`, accepted.ID, binding.TaskKey, string(raw)); err != nil {
		t.Fatal(err)
	}
	if candidates, err := db.DeliveredIntentCandidates(ctx, 16); err != nil || len(candidates) != 0 {
		t.Fatalf("mismatched native task binding admitted application verification: %+v %v", candidates, err)
	}
}

// A recorded exclusion the acceptance drops refuses: flipping a
// reviewed accept to decline revises the reviewed product choice, so
// the recorded coverage no longer answers the accepted direction.
func TestResolveAcceptedPlanChoiceDroppedExclusionRefuses(t *testing.T) {
	ctx := context.Background()
	db, _ := dispatchTestDB(t)
	fx := dispatchSeed(t, db)
	draft := choicePlanDraft(fx.repo)
	if _, ok := settlePlanAnswerWithOperation(t, ctx, db, fx, draft, factory.IntentPlanningOptions, choicePlanAnswer(t, draft, "completed", "cand-a")); !ok {
		t.Fatal("recorded plan did not settle")
	}
	accepted := choicePlanAccepted(t, fx.repo)
	accepted.Ideas[1].Decision = "decline"
	accepted.Tasks = []factory.IntentTask{{
		Key: choiceTaskKey(9), Objective: "Capture leave requests",
		Outcomes: []string{"Requests can be recorded"}, IdeaIDs: []string{"d555555555555555555555555"},
	}}
	if err := db.SaveIntentRevision(ctx, accepted); err != nil {
		t.Fatal(err)
	}
	stub := &candidateChoiceStub{selected: "cand-a", confidence: 1}
	c := choicePlanCoordinator(t, db, fx, stub)
	_, _, err := c.ResolveAcceptedPlanChoice(ctx, 7, accepted)
	if !errors.Is(err, ErrPlanChoiceStale) || !strings.Contains(err.Error(), "reviewed product choice") {
		t.Fatalf("dropped exclusion must refuse as stale: %v", err)
	}
	if stub.calls != 0 {
		t.Fatalf("dropped exclusion funded %d comparative calls", stub.calls)
	}
}

// An acceptance that narrows the authorized outcomes refuses when the
// recorded candidates exceed them: coverage is proven before funding,
// not only at downstream graph validation.
func TestResolveAcceptedPlanChoiceNarrowedScopeRefuses(t *testing.T) {
	ctx := context.Background()
	db, _ := dispatchTestDB(t)
	fx := dispatchSeed(t, db)
	draft := choicePlanDraft(fx.repo)
	if _, ok := settlePlanAnswerWithOperation(t, ctx, db, fx, draft, factory.IntentPlanningOptions, choicePlanAnswer(t, draft, "completed", "cand-a")); !ok {
		t.Fatal("recorded plan did not settle")
	}
	accepted := choicePlanAccepted(t, fx.repo)
	accepted.EffectiveOutcomes = []string{"d555555555555555555555555"}
	accepted.Tasks = []factory.IntentTask{{
		Key: choiceTaskKey(9), Objective: "Capture leave requests",
		Outcomes: []string{"Requests can be recorded"}, IdeaIDs: []string{"d555555555555555555555555"},
	}}
	if err := db.SaveIntentRevision(ctx, accepted); err != nil {
		t.Fatal(err)
	}
	stub := &candidateChoiceStub{selected: "cand-a", confidence: 1}
	c := choicePlanCoordinator(t, db, fx, stub)
	_, _, err := c.ResolveAcceptedPlanChoice(ctx, 7, accepted)
	if !errors.Is(err, ErrPlanChoiceStale) || !strings.Contains(err.Error(), "exceeds accepted scope") {
		t.Fatalf("narrowed scope must refuse as stale: %v", err)
	}
	if stub.calls != 0 {
		t.Fatalf("narrowed scope funded %d comparative calls", stub.calls)
	}
}
