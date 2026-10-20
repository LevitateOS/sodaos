// Helpers retained for the kept integration tests after unit-test removal.
package control

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
)

// deliveryChooser is the deterministic FW21 adapter: scripted answers, no
// live Jev consultation.
type deliveryChooser struct {
	calls  int
	answer func(fingerprint string) (factory.DeliveryAnswer, error)
}

func (f *deliveryChooser) SelectDelivery(_ context.Context, state any, _ string, _ map[string]any) (factory.DeliveryAnswer, error) {
	f.calls++
	in, ok := state.(factory.DeliveryFingerprintInput)
	if !ok {
		return factory.DeliveryAnswer{}, errors.New("delivery state is not its evidence input")
	}
	fingerprint, err := factory.DeliveryFingerprint(in)
	if err != nil {
		return factory.DeliveryAnswer{}, err
	}
	return f.answer(fingerprint)
}

func readyDeliveryAnswer(fingerprint string) (factory.DeliveryAnswer, error) {
	return factory.DeliveryAnswer{
		Probabilities: map[string]float64{factory.DeliveryReviewAgain: 0.1, factory.DeliveryReadyToMerge: 0.9},
		Fingerprint:   fingerprint,
		Selected:      factory.DeliveryReadyToMerge,
		Model:         "jev-test",
		Probability:   0.9,
		Confidence:    0.85,
	}, nil
}

func reviewDeliveryAnswer(fingerprint string) (factory.DeliveryAnswer, error) {
	return factory.DeliveryAnswer{
		Probabilities: map[string]float64{factory.DeliveryReviewAgain: 0.9, factory.DeliveryReadyToMerge: 0.1},
		Fingerprint:   fingerprint,
		Selected:      factory.DeliveryReviewAgain,
		Model:         "jev-test",
		Probability:   0.9,
		Confidence:    0.9,
	}, nil
}

// deliveryFunding mirrors the store funding contract in memory:
// attempt-root subledger plus a global ceiling, reserved in one step,
// with fingerprint-keyed receipts.
type deliveryFunding struct {
	subledgerLimit int
	globalBudget   int
	subledger      []string
	requestOwner   map[string]string
	spent          int
	reserved       map[string]bool
	receipts       map[string]factory.DeliveryReceipt
	reserveCalls   int
}

func newDeliveryFunding(subledgerLimit, globalBudget int) *deliveryFunding {
	return &deliveryFunding{
		subledgerLimit: subledgerLimit,
		globalBudget:   globalBudget,
		requestOwner:   map[string]string{},
		reserved:       map[string]bool{},
		receipts:       map[string]factory.DeliveryReceipt{},
	}
}

func (f *deliveryFunding) DeliveryJudgment(_ context.Context, fingerprint string) (factory.DeliveryReceipt, error) {
	r, ok := f.receipts[fingerprint]
	if !ok {
		return factory.DeliveryReceipt{}, store.ErrNotFound
	}
	return r, nil
}

func (f *deliveryFunding) ReserveDeliveryJudgment(_ context.Context, _, _ int64, _, request, fingerprint string) (bool, error) {
	f.reserveCalls++
	if !factory.ValidID(request) || !factory.ValidDigest(fingerprint) {
		return false, errors.New("invalid delivery reservation")
	}
	if f.reserved[fingerprint] {
		return false, nil
	}
	if owner, seen := f.requestOwner[request]; seen && owner != fingerprint {
		return false, store.ErrCommandConflict
	}
	for _, seen := range f.subledger {
		if seen == request {
			return false, store.ErrCommandConflict
		}
	}
	if len(f.subledger) >= f.subledgerLimit || f.spent >= f.globalBudget {
		return false, nil
	}
	f.subledger = append(append([]string(nil), f.subledger...), request)
	f.requestOwner[request] = fingerprint
	f.spent++
	f.reserved[fingerprint] = true
	return true, nil
}

func (f *deliveryFunding) RecordDeliveryJudgment(_ context.Context, r factory.DeliveryReceipt) error {
	if err := r.Validate(); err != nil {
		return err
	}
	if !f.reserved[r.Fingerprint] {
		return store.ErrCommandConflict
	}
	if _, done := f.receipts[r.Fingerprint]; done {
		return store.ErrCommandConflict
	}
	f.receipts[r.Fingerprint] = r
	return nil
}

func deliveryTestInput() factory.DeliveryFingerprintInput {
	return factory.DeliveryFingerprintInput{
		Reviews:        []string{strings.Repeat("1", 32)},
		Conversations:  []string{strings.Repeat("2", 64)},
		Constraints:    []string{"protected-files:docs/public"},
		AttemptRoot:    strings.Repeat("3", 32),
		PublicationID:  strings.Repeat("4", 32),
		ScopeDigest:    strings.Repeat("5", 64),
		FixDigest:      strings.Repeat("6", 64),
		DiffDigest:     strings.Repeat("7", 64),
		CheckDigest:    strings.Repeat("8", 64),
		TreeDigest:     strings.Repeat("9", 64),
		Head:           strings.Repeat("a", 40),
		Base:           strings.Repeat("b", 40),
		Model:          "jev-test",
		Rubric:         "ready_to_merge only with full evidence cover",
		PolicyRevision: 3,
		Round:          1,
	}
}

func deliveryTestAdvice(funding *deliveryFunding, chooser *deliveryChooser, in factory.DeliveryFingerprintInput, reviewsFit bool) DeliveryAdvice {
	return DeliveryAdvice{
		Input:                  in,
		Repository:             7,
		Issue:                  3,
		AttemptRoot:            strings.Repeat("3", 32),
		RequestID:              factory.NewID(),
		ReviewsFit:             reviewsFit,
		ObservedNativeRevision: 41,
		Chooser:                chooser,
		Funding:                funding,
	}
}

type fakeReviewer struct {
	observed     factory.ReviewObservation
	outcome      factory.OperationOutcome
	adopted      factory.ReviewOutcome
	works        []factory.ReviewWork
	submitted    []factory.ReviewWork
	lookups      []string
	results      map[string]factory.OperationOutcome
	submitErr    error
	beforeSubmit func(factory.ReviewWork)
	cancelled    []string
}

func (f *fakeReviewer) ObserveReview(ctx context.Context, w factory.ReviewWork) (factory.ReviewObservation, error) {
	f.works = append(f.works, w)
	return f.observed, nil
}

func (f *fakeReviewer) SubmitReview(ctx context.Context, w factory.ReviewWork) (factory.OperationOutcome, error) {
	f.submitted = append(f.submitted, w)
	if f.beforeSubmit != nil {
		f.beforeSubmit(w)
	}
	outcome := f.outcome
	outcome.OperationID, outcome.InstallationID, outcome.Kind = w.OperationID, "test-installation", factory.OpReviewSubmit
	outcome.ActorID, outcome.RepositoryID = w.ActorID, w.Repository
	if f.results == nil {
		f.results = make(map[string]factory.OperationOutcome)
	}
	f.results[w.OperationID] = outcome
	return outcome, f.submitErr
}

func (f *fakeReviewer) LookupOp(ctx context.Context, operationID string) (factory.OperationOutcome, error) {
	f.lookups = append(f.lookups, operationID)
	if outcome, ok := f.results[operationID]; ok {
		return outcome, nil
	}
	return factory.OperationOutcome{NotObserved: true}, nil
}

func (f *fakeReviewer) CancelOp(ctx context.Context, operationID string) (factory.OperationOutcome, error) {
	f.cancelled = append(f.cancelled, operationID)
	outcome, ok := f.results[operationID]
	if !ok {
		return factory.OperationOutcome{NotObserved: true}, nil
	}
	outcome.Effect, outcome.Cancellation = factory.OpEffectNotCommitted, factory.OpCancelCancelled
	f.results[operationID] = outcome
	return outcome, nil
}

func (f *fakeReviewer) AdoptReview(w factory.ReviewWork, outcome factory.OperationOutcome) (factory.ReviewOutcome, error) {
	adopted := f.adopted
	adopted.Operation = outcome
	return adopted, nil
}

func reviewRunSeed(t *testing.T, settled bool) (*publishFixture, factory.Assignment, factory.Publication, factory.Run) {
	t.Helper()
	ctx := context.Background()
	fx := publishSeed(t)
	policy, err := fx.db.RepositoryPolicy(ctx, fx.seed.repo)
	if err != nil {
		t.Fatal(err)
	}
	policy.Review = factory.ActorBindingRef{TokenID: 2, ActorID: 6, Kind: factory.OpReviewSubmit}
	if err := fx.db.SaveRepositoryPolicy(ctx, policy); err != nil {
		t.Fatal(err)
	}
	sponsorship, err := fx.db.Sponsorship(ctx, fx.seed.repo, "conn")
	if err != nil {
		t.Fatal(err)
	}
	sponsorship.Roles = append(sponsorship.Roles, project.RoleReviewer)
	if err := fx.db.SaveSponsorship(ctx, sponsorship); err != nil {
		t.Fatal(err)
	}
	a := fx.finishReported(t, 14)
	fx.wire(happyPublisher())
	fx.pass(t)
	p, err := fx.db.PublicationByAssignment(ctx, a.ID)
	if err != nil {
		t.Fatal(err)
	}
	_, run := fx.childRun(t, a, &p, project.RoleReviewer, p.Candidate, "f222222222222222222222222", settled)
	return fx, a, p, run
}

func phasedDecisionID(n int) string { return fmt.Sprintf("d%024x", 0xe000+n) }
func phaseTestRepairAssignment(t *testing.T, phase, role string, subject factory.ExecutionSubject) factory.Assignment {
	t.Helper()
	id, run := factory.NewID(), factory.NewID()
	prompt := []byte("prompt bytes for " + phase)
	sum := sha256.Sum256(prompt)
	a := factory.Assignment{
		ID: id, AttemptRoot: id,
		Subject: subject, Phase: phase,
		ProjectID: "p" + strings.Repeat("1", 24), Role: role,
		Repository: 7, NativeRev: 1,
		Preparation: "f" + strings.Repeat("d", 24),
		Harness:     "codex-1.2.3", HarnessVers: "1.2.3",
		Model: "test-model", Connection: "conn",
		SourceCommit: phaseTestCommit(), Prompt: prompt, PromptSHA: hex.EncodeToString(sum[:]),
		Run: run, RunHistory: []string{run}, Stage: factory.AssignmentAssigned,
		Attempts: 1, CreatedUnix: 1700000000, ActorID: 7,
	}
	if err := a.Validate(); err != nil {
		t.Fatalf("repair assignment fixture invalid: %v", err)
	}
	return a
}
func phasedDeliverySubject() factory.ExecutionSubject {
	return factory.ExecutionSubject{
		Kind: factory.SubjectDelivery,
		Delivery: &factory.DeliverySubject{
			DeliveryID: strings.Repeat("7", 32), IntentID: phasedDecisionID(20),
			AcceptedRevision: phasedDecisionID(21), ScopeDigest: phaseTestSHA("scope"),
			SourceCommit: phaseTestCommit(), PreparationDigest: phaseTestSHA("prep"),
			VerificationManifestDigest: phaseTestSHA("manifest"),
		},
	}
}

func phaseTestSHA(seed string) string {
	sum := sha256.Sum256([]byte(seed))
	return hex.EncodeToString(sum[:])
}

func phaseTestCommit() string { return strings.Repeat("c", 40) }

func testMaterialLocal() factory.CompletionLocal {
	acceptance := factory.Acceptance{
		ID: "d" + strings.Repeat("0", 24), Repository: 7, IssueIndex: "3",
		Approver: 5, NativeRev: 41,
		TitleDigest: strings.Repeat("a", 64), TitleOccurrence: factory.OriginalOccurrence,
		ContentDigest: strings.Repeat("b", 64), ContentVersion: 2,
		Sources: []factory.SelectedSource{{ID: "11", ContentVersion: 0, Digest: strings.Repeat("c", 64)}},
	}
	material := factory.CompletionMaterial{
		Intents:     []factory.CompletionIntentScope{},
		Acceptances: []factory.Acceptance{acceptance},
		Withdrawals: []factory.CompletionWithdrawal{},
		Controls: []factory.IssueControl{{
			Repository: 7, Issue: 7,
			Readiness: factory.ReadinessQueued, Reason: factory.ReasonEligible,
			Fingerprint: strings.Repeat("1", 64), Authority: strings.Repeat("2", 64),
		}},
		Preparations: []factory.CompletionPreparation{},
		Plans:        []factory.InvestigationPlan{},
		PlanHeads:    []factory.CompletionPlanHead{},
		PlanTasks:    []factory.CompletionPlanTask{},
		Assignments:  []factory.CompletionAssignment{},
		Publications: []factory.CompletionPublication{},
		Merges:       []factory.CompletionMerge{},
		Checks:       []factory.CompletionCheck{},
		Authority: factory.CompletionAuthority{
			Effective: factory.EffectiveAuthority{
				Missing: []string{}, Effective: true, DispatchOpen: true,
				Authority: factory.AuthorityRef{
					RequirementsID: "d" + strings.Repeat("c", 24),
					ApprovalID:     "d" + strings.Repeat("e", 24),
				},
			},
			ProjectID:               "p" + strings.Repeat("d", 24),
			RequirementSourceCommit: strings.Repeat("c", 40),
			RequirementSetupDigest:  strings.Repeat("e", 64),
			RequirementInputsDigest: strings.Repeat("f", 64),
			ApprovalEffectsDigest:   strings.Repeat("0", 64),
			ApprovalReadinessDigest: strings.Repeat("1", 64),
			ApprovalVerified:        false,
			TargetBranch:            "refs/heads/main",
			Roles:                   []string{"soda-coder", "soda-reviewer"},
			ReassessmentMinutes:     factory.DefaultReassessmentMinutes,
		},
		Repository: 7,
	}
	digest, err := material.Digest()
	if err != nil {
		panic(err)
	}
	return factory.CompletionLocal{
		Material:             &material,
		SourceCommit:         strings.Repeat("a", 40),
		PreparationDigest:    strings.Repeat("b", 64),
		ScopeDigest:          digest,
		AuthorityFingerprint: "authority-fingerprint",
	}
}

func testLocal() factory.CompletionLocal {
	return testMaterialLocal()
}

// digestOnlyLocal is the pre-077 shape: valid four bindings but no
// material body. Every consumer must refuse it before any spend.
func digestOnlyLocal() factory.CompletionLocal {
	return factory.CompletionLocal{
		SourceCommit:         strings.Repeat("a", 40),
		PreparationDigest:    strings.Repeat("b", 64),
		ScopeDigest:          strings.Repeat("c", 64),
		AuthorityFingerprint: "authority-fingerprint",
	}
}

// fixtureEnumerator serves scripted pages with a stable revision
// bracket unless told to move it.
type fixtureEnumerator struct {
	pages    map[string]NativeInventoryPage
	revision int64
	idle     bool
	calls    int
	observes int
	moveRev  func(call int) int64
	err      error
}

func (f *fixtureEnumerator) ObserveRevision(context.Context) (int64, bool, error) {
	f.observes++
	if f.err != nil {
		return 0, false, f.err
	}
	revision := f.revision
	if f.moveRev != nil {
		revision = f.moveRev(f.calls)
	}
	idle := f.idle
	if !idle {
		return revision, false, nil
	}
	return revision, true, nil
}

func (f *fixtureEnumerator) EnumerateIssues(_ context.Context, _ int64, cursor string) (NativeInventoryPage, error) {
	f.calls++
	if f.err != nil {
		return NativeInventoryPage{}, f.err
	}
	page, ok := f.pages[cursor]
	if !ok {
		return NativeInventoryPage{}, errors.New("unexpected cursor")
	}
	revision := f.revision
	if f.moveRev != nil {
		revision = f.moveRev(f.calls)
	}
	page.Revision = revision
	page.Idle = f.idle
	return page, nil
}

func issueDigest(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:])
}

// inventoryIssue builds one canonical sealed payload through the real
// producer encoding, so the fixtures prove the exact bytes the seal
// persists and the decision assesses.
func inventoryIssue(t *testing.T, index int64) NativeInventoryIssue {
	t.Helper()
	title, body := fmt.Sprintf("title %d", index), fmt.Sprintf("body %d", index)
	data, err := factory.CompletionIssueEvidence{
		Index: index, Title: title, Body: body,
		TitleDigest: issueDigest(title), ContentDigest: issueDigest(body),
	}.CanonicalData()
	if err != nil {
		t.Fatal(err)
	}
	return NativeInventoryIssue{Index: index, Data: data}
}

func twoPageEnumerator(t *testing.T) *fixtureEnumerator {
	t.Helper()
	return &fixtureEnumerator{
		revision: 41,
		idle:     true,
		pages: map[string]NativeInventoryPage{
			"":  {Issues: []NativeInventoryIssue{inventoryIssue(t, 1), inventoryIssue(t, 2)}, NextCursor: "2"},
			"2": {Issues: []NativeInventoryIssue{inventoryIssue(t, 3)}, Complete: true},
		},
	}
}

func authorizeMergeFixture(t *testing.T, ctx context.Context, db *store.Store, coord *Coordinator, p factory.Publication) {
	t.Helper()
	authorizeMergeFixtureRound(t, ctx, db, coord, p, 1)
}

func phaseTestPlan(t *testing.T, root, source, prep string) factory.InvestigationPlan {
	t.Helper()
	plan := factory.InvestigationPlan{
		ID: factory.NewID(), AttemptRoot: root,
		ScopeDigest: phaseTestSHA("scope"), InputDigest: phaseTestSHA("input"),
		SourceCommit: source, IntegrationCommit: source,
		Preparation:        prep,
		Approach:           "inspect the selected source, then plan behind checks",
		AcceptanceCoverage: []string{"accept-one"},
		Tasks: []factory.PlanTask{{
			ID: "task-one", Objective: "inspect the selected source",
			Outputs: []string{"notes"}, AcceptanceRefs: []string{"accept-one"},
			Lane: "lane-one", OwnedPaths: []string{"internal/x.go"},
		}},
		Lanes: []factory.PlanLane{{ID: "lane-one", TaskIDs: []string{"task-one"}}},
		Checks: []factory.CheckSpec{{
			ID: "ci", Argv: []string{"ci"}, Cwd: ".",
			TimeoutSeconds: 60, Scope: factory.CheckScopeTask, Required: true,
		}},
	}
	digest, err := factory.PlanDigest(plan)
	if err != nil {
		t.Fatal(err)
	}
	plan.Digest = digest
	if err := plan.Validate(); err != nil {
		t.Fatalf("plan fixture invalid: %v", err)
	}
	if err := plan.Executable(); err != nil {
		t.Fatalf("plan fixture not executable: %v", err)
	}
	return plan
}

func phaseTestAssignment(t *testing.T, phase, role string, validate bool) factory.Assignment {
	t.Helper()
	id, run := factory.NewID(), factory.NewID()
	prompt := []byte("prompt bytes for " + phase)
	sum := sha256.Sum256(prompt)
	acceptance := "d" + strings.Repeat("a", 24)
	a := factory.Assignment{
		ID: id, AttemptRoot: id, PublicationAssignment: id,
		Subject: factory.ExecutionSubject{
			Kind:  factory.SubjectIssue,
			Issue: &factory.IssueSubject{Issue: 7, Acceptance: acceptance},
		},
		Phase:     phase,
		ProjectID: "p" + strings.Repeat("1", 24), Role: role,
		Repository: 7, Issue: 7, NativeRev: 1,
		Acceptance: acceptance, Preparation: "f" + strings.Repeat("d", 24),
		Harness: "codex-1.2.3", HarnessVers: "1.2.3",
		Model: "test-model", Connection: "conn",
		SourceCommit: phaseTestCommit(), Prompt: prompt, PromptSHA: hex.EncodeToString(sum[:]),
		Run: run, RunHistory: []string{run}, Stage: factory.AssignmentAssigned,
		Attempts: 1, CreatedUnix: 1700000000, ActorID: 7,
	}
	if validate {
		if err := a.Validate(); err != nil {
			t.Fatalf("assignment fixture invalid: %v", err)
		}
	}
	return a
}

func fencedResult(t *testing.T, v any) string {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return "work log\n```" + factory.ResultFence + "\n" + string(data) + "\n```\ntrailer\n"
}

type producerFlowHost struct {
	*fakePublishHost
	launches       []project.FactoryLaunch
	preparations   []project.FactoryCandidate
	prepInspects   []project.PrepareInspect
	prepPhase      string
	stopRetirement string
	prepStops      []project.PrepareStop
	contextReads   []project.FactoryPreparationContextRequest
	contextRoles   []string
}

func (h *producerFlowHost) FactoryLaunch(_ context.Context, in project.FactoryLaunch) (project.FactoryState, error) {
	h.launches = append(h.launches, in)
	return project.FactoryState{
		ID: in.Run.ID, Project: in.Run.Project, Role: in.Run.Role,
		Phase: project.FactoryCompleted, Container: strings.Repeat("c", 64),
	}, nil
}

func (h *producerFlowHost) ReadPreparationContext(_ context.Context, in project.FactoryPreparationContextRequest, role string) (project.FactoryPreparationContext, error) {
	h.contextReads = append(h.contextReads, in)
	h.contextRoles = append(h.contextRoles, role)
	return fixturePreparationContext(in, role), nil
}

func (*producerFlowHost) FactoryHarness(_ context.Context, family string) (project.FactoryHarnessPin, error) {
	return project.FactoryHarnessPin{Harness: family, Version: "1.2.3", SHA256: strings.Repeat("a", 64), Image: "sha256:" + strings.Repeat("b", 64)}, nil
}

func (h *producerFlowHost) PrepareCandidate(_ context.Context, in project.FactoryCandidate) (project.PrepareState, error) {
	h.preparations = append(h.preparations, in)
	phase := h.prepPhase
	if phase == "" {
		phase = project.PrepareReady
	}
	return project.PrepareState{
		ID: in.Preparation.ID, Project: in.Preparation.Project, Role: in.Preparation.Role,
		Phase: phase, Container: strings.Repeat("d", 64), SourceCommit: in.Preparation.SourceCommit,
		SetupDigest: in.Preparation.SetupDigest, Ready: phase == project.PrepareReady,
	}, nil
}

func (h *producerFlowHost) InspectPreparation(_ context.Context, in project.PrepareInspect) (project.PrepareState, error) {
	h.prepInspects = append(h.prepInspects, in)
	return project.PrepareState{}, project.ErrPreparationNotFound
}

func (h *producerFlowHost) StopPreparation(_ context.Context, in project.PrepareStop) (project.PrepareState, error) {
	h.prepStops = append(h.prepStops, in)
	retirement := h.stopRetirement
	if retirement == "" {
		retirement = "confirmed"
	}
	return project.PrepareState{
		ID: in.ID, Project: in.Project, Phase: project.PrepareStopped,
		Stopped: true, Retirement: retirement,
	}, nil
}

func publishedChildProducerSeed(t *testing.T, issue int64, preparationPhase ...string) (*publishFixture, *producerFlowHost, factory.Assignment, factory.Publication) {
	t.Helper()
	ctx := context.Background()
	fx := publishSeed(t)
	policy, err := fx.db.RepositoryPolicy(ctx, fx.seed.repo)
	if err != nil {
		t.Fatal(err)
	}
	policy.Review = factory.ActorBindingRef{TokenID: 2, ActorID: 6, Kind: factory.OpReviewSubmit}
	if err := fx.db.SaveRepositoryPolicy(ctx, policy); err != nil {
		t.Fatal(err)
	}
	sponsorship, err := fx.db.Sponsorship(ctx, fx.seed.repo, "conn")
	if err != nil {
		t.Fatal(err)
	}
	sponsorship.Roles = append(sponsorship.Roles, project.RoleReviewer)
	if err := fx.db.SaveSponsorship(ctx, sponsorship); err != nil {
		t.Fatal(err)
	}
	host := &producerFlowHost{fakePublishHost: fx.host}
	if len(preparationPhase) > 0 {
		host.prepPhase = preparationPhase[0]
	}
	fx.coord.Host = host
	fx.coord.DispatchReads = fx.seed.reads
	fx.coord.Reviews = &fakeReviewer{}
	owner := fx.finishReported(t, issue)
	fx.wire(happyPublisher())
	if report := fx.coord.PublishPass(ctx); len(report.Errors) != 0 {
		t.Fatalf("initial production publication pass: %+v", report)
	}
	p, err := fx.db.PublicationByAssignment(ctx, owner.ID)
	if err != nil || p.Stage != factory.PublicationPublished {
		t.Fatalf("publication not linked: %+v %v", p, err)
	}
	return fx, host, owner, p
}

// producedReviewCycleID returns the review cycle that admitted the child
// behind the given production run.
func producedReviewCycleID(t *testing.T, fx *publishFixture, runID string) string {
	t.Helper()
	ctx := context.Background()
	view, err := fx.db.FactoryRunView(ctx, runID)
	if err != nil {
		t.Fatal(err)
	}
	assignment, err := fx.db.Assignment(ctx, view.Attempt)
	if err != nil {
		t.Fatal(err)
	}
	p, err := fx.db.PublicationByAssignment(ctx, assignment.PublicationAssignment)
	if err != nil {
		t.Fatal(err)
	}
	cycleID := reviewerCycleID(p, assignment)
	if cycleID == "" {
		t.Fatalf("produced child admitted no review cycle: child=%s", assignment.ID)
	}
	return cycleID
}

func settleProducedChild(t *testing.T, fx *publishFixture, runID, output string) (factory.Assignment, factory.Run) {
	t.Helper()
	ctx := context.Background()
	run, err := fx.db.FactoryRun(ctx, runID)
	if err != nil {
		t.Fatal(err)
	}
	run.Outcome, run.Summary, run.Reconciled = factory.Succeeded, "child result reconciled", true
	if err := fx.db.SaveFactoryRun(ctx, run); err != nil {
		t.Fatal(err)
	}
	assignment, accounted := AccountSettledRun(ctx, fx.db, run, output, time.Now())
	if !accounted || assignment.Stage != factory.AssignmentFinished {
		t.Fatalf("produced child result was not accounted: %+v", assignment)
	}
	return assignment, run
}

func authorizeMergeFixtureRound(t *testing.T, ctx context.Context, db *store.Store, coord *Coordinator, p factory.Publication, round int) {
	t.Helper()
	m, err := db.MergeByPublication(ctx, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if m.Stage != factory.MergeOpen || m.Authorization.Bound() {
		t.Fatalf("merge row not awaiting authorization: %+v", m)
	}
	owner, err := db.Assignment(ctx, p.AssignmentID)
	if err != nil {
		t.Fatal(err)
	}
	emptyContent, err := factory.FindingContentDigest(nil)
	if err != nil {
		t.Fatal(err)
	}
	// Phased fixtures record real review cycles before the merge round:
	// derive the round from the stored publication like production instead
	// of reusing the hardcoded first round.
	stored, err := db.PublicationByAssignment(ctx, p.AssignmentID)
	if err != nil {
		t.Fatal(err)
	}
	round = maxReviewCycleRound(stored) + 1
	reviewer := factory.NewID()
	cycle := factory.ReviewCycle{
		PublicationID: p.ID, AttemptRoot: owner.AttemptRoot,
		ID: factory.ReviewCycleID(p.ID, owner.AttemptRoot, round, "review"), Phase: "review",
		TriggerDecision: factory.DeliveryReviewAgain, TriggerReason: "fixture round needs its independent inspection",
		Head: p.Candidate, BaseOID: p.PRCreate.BaseOID, FindingDigest: emptyContent,
		ReviewerAssignment: reviewer, Round: round,
	}
	if err := cycle.Validate(); err != nil {
		t.Fatal(err)
	}
	cycled := stored
	cycled.ReviewCycles = append(cycled.ReviewCycles, cycle)
	cycled.Revision = stored.Revision + 1
	if err := db.UpdatePublication(ctx, cycled); err != nil {
		t.Fatal(err)
	}
	assessment, err := db.CheckAssessment(ctx, p.Repository, p.PRNumber)
	if err != nil {
		t.Fatal(err)
	}

	policy, err := db.RepositoryPolicy(ctx, p.Repository)
	if err != nil {
		t.Fatal(err)
	}
	journaled, acceptance, err := AdmitClearReview(ClearAdmissionInput{
		Publication: cycled, Cycle: cycle,
		Reviewer:       clearTestReviewer(t, reviewer, cycle.Head, cycle.BaseOID, cycle, nil, nil),
		Carried:        nil,
		ScopeDigest:    dispatchDigest("merge fixture scope"),
		PolicyRevision: strconv.FormatInt(policy.Revision, 10),
		CheckDigest:    assessment.ChecksDigest,
		DiffBase:       p.PRCreate.BaseOID,
	})
	if err != nil {
		t.Fatal(err)
	}
	journaled.Revision = cycled.Revision + 1
	if err := db.UpdatePublication(ctx, journaled); err != nil {
		t.Fatal(err)
	}
	if err := coord.AuthorizeMerge(ctx, p.ID, servingUnionBinding(t, m, acceptance, factory.DeliveryDecision{})); err != nil {
		t.Fatal(err)
	}
}

func testMCPCapability() factory.MCPCapability {
	digest := strings.Repeat("a", 64)
	return factory.MCPCapability{
		ID: factory.NewID(), AssignmentID: factory.NewID(),
		RunID: "run-1", Generation: 1,
		Profile: factory.MCPProfileBuilder, Subject: "implement:42", Repository: 7,
		Project:           "p" + strings.Repeat("d", 24),
		PreparationDigest: digest, SourceDigest: "abc", AcceptedRevision: "rev-1",
		Authority: digest, DeadlineUnix: time.Now().Add(time.Hour).Unix(),
		State: factory.MCPCapabilityActive,
	}
}

func testMCPObserved(capability factory.MCPCapability) MCPRunObservation {
	return MCPRunObservation{
		RunID: capability.RunID, Incarnation: 3,
		AssignmentID: capability.AssignmentID, Authority: capability.Authority,
	}
}

func testMCPSubmission(capability factory.MCPCapability, tool factory.MCPTool, id string, payload []byte) factory.MCPSubmission {
	return factory.MCPSubmission{
		AssignmentID: capability.AssignmentID, SubmissionID: id,
		CapabilityID: capability.ID, Tool: tool,
		ExpectedInputDigest: capability.ScopeDigest(),
		PayloadHash:         factory.MCPInputDigest(payload), Payload: payload,
	}
}

func unionSeedForBind(t *testing.T, ctx context.Context, issue int64) (*mergeFixture, factory.Publication, factory.Merge, factory.DeliveryAcceptance, factory.DeliveryDecision, string) {
	t.Helper()
	fx, p := mergeSeed(t, issue)
	db := fx.publish.db
	storedPub, acceptance, decision, fingerprint := unionPgAdmitJev(t, ctx, db, p, strings.Repeat("4", 32), nil)
	setup := MergeReport{Merged: []MergeLink{}}
	fx.publish.coord.mergeOne(ctx, storedPub, &setup)
	if len(setup.Errors) != 0 {
		t.Fatalf("merge row open errored: %+v", setup)
	}
	m, err := db.MergeByPublication(ctx, p.ID)
	if err != nil {
		t.Fatalf("merge row unopened: %v", err)
	}
	return fx, storedPub, m, acceptance, decision, fingerprint
}

// unionAuthForBind builds one union authorization for an open row behind
// a journaled acceptance.
func unionAuthForBind(t *testing.T, m factory.Merge, acceptance factory.DeliveryAcceptance, decision factory.DeliveryDecision, fingerprint string, confirmed bool) factory.MergeAuthorization {
	t.Helper()
	auth := factory.MergeAuthorization{
		Generation: factory.MergeGeneration{
			Authority: m.Authority, PublicationID: m.PublicationID,
			DecisionID:         acceptance.DecisionID,
			NativeOperationID:  factory.MergeGenerationOperationID(m.PublicationID, 1),
			Fingerprint:        fingerprint,
			Head:               m.HeadOID,
			Base:               m.BaseOID,
			DiffBase:           acceptance.Tuple.DiffBase,
			ChecksDigest:       strings.Repeat("c", 64),
			AcceptanceKind:     acceptance.Kind,
			AcceptanceID:       acceptance.ID,
			AttemptRoot:        acceptance.AttemptRoot,
			OriginReceipt:      acceptance.OriginReceipt,
			Generation:         1,
			ReviewRound:        acceptance.Round,
			AssessmentRevision: 2, ObservedNativeRev: 12, NotAfter: 1300,
		},
		Decision: decision, ResolutionsConfirmed: confirmed,
	}
	return auth
}

func correctionTestKey(n int) string { return fmt.Sprintf("%032x", 0xa0+n) }

func correctionTestFinding(n int, actionable bool, fixKind string) factory.Finding {
	return factory.Finding{
		Key: correctionTestKey(n), OriginRun: strings.Repeat("1", 32),
		Severity: factory.FindingSeverityMajor,
		Defect:   "missing bound", Evidence: "line 42 lacks a check", RequiredOutcome: "add the check",
		FixKind: fixKind, Actionable: actionable,
	}
}

func correctionTestDigest(t *testing.T, keys ...string) string {
	t.Helper()
	digest, err := factory.FindingSetDigest(keys)
	if err != nil {
		t.Fatal(err)
	}
	return digest
}

func correctionTestCycle(t *testing.T, publicationID string, head, base string, round int, digest string) factory.ReviewCycle {
	t.Helper()
	attemptRoot := strings.Repeat("b", 32)
	return factory.ReviewCycle{
		PublicationID: publicationID, AttemptRoot: attemptRoot,
		ID:                 factory.ReviewCycleID(publicationID, attemptRoot, round, "review"),
		Phase:              "review",
		TriggerDecision:    factory.DeliveryReviewAgain,
		TriggerReason:      "correction set needs another inspection",
		Head:               head,
		BaseOID:            base,
		FindingDigest:      digest,
		ReviewerAssignment: strings.Repeat("4", 32),
		Round:              round,
	}
}

func correctionTestReport(candidate string, findings []factory.Finding, dispositions []factory.CorrectionDisposition) factory.CorrectionReport {
	keys := make([]string, 0, len(findings))
	for _, finding := range findings {
		keys = append(keys, finding.Key)
	}
	digest, err := factory.FindingSetDigest(keys)
	if err != nil {
		return factory.CorrectionReport{}
	}
	return factory.CorrectionReport{SetDigest: digest, NewCandidate: candidate, Findings: findings, Dispositions: dispositions}
}

func correctionTestHeads() (head, base, candidate string) {
	return strings.Repeat("c", 40), strings.Repeat("b", 40), strings.Repeat("d", 40)
}

func correctionTestChecks(head, base string) *factory.CheckAssessment {
	checks := []string{"ci"}
	return &factory.CheckAssessment{
		Results:          []factory.CheckResult{{Context: "ci", State: factory.CheckStateSuccess, Passed: true}},
		Checks:           checks,
		Repository:       7,
		PRNumber:         3,
		PRID:             5,
		IssueID:          9,
		PolicyRevision:   1,
		NativeRev:        2,
		Revision:         1,
		AssessedUnix:     time.Now().Unix(),
		ObservedContexts: 1,
		HeadRef:          "refs/heads/fix",
		BaseRef:          "refs/heads/main",
		HeadOID:          head,
		BaseOID:          base,
		ChecksDigest:     factory.ChecksDigest(checks),
		Verdict:          factory.CheckPass,
		Reason:           factory.CheckReasonPass,
	}
}

func clearTestReviewer(t *testing.T, reviewer, head, base string, cycle factory.ReviewCycle, assessments []factory.FindingAssessment, carried []factory.Finding) factory.Assignment {
	t.Helper()
	content, err := factory.FindingContentDigest(carried)
	if err != nil {
		t.Fatal(err)
	}
	coverage, err := factory.ClearCoverageDigest(head, base, content)
	if err != nil {
		t.Fatal(err)
	}
	a := phaseTestAssignment(t, project.ExecutionReview, project.RoleReviewer, true)
	a.ID, a.SourceCommit = reviewer, head
	// The reviewer is a child of the attempt root: its lineage binds
	// the recorded publication owner and cycle attempt root.
	a.ParentAssignment = cycle.AttemptRoot
	a.AttemptRoot, a.PublicationAssignment = cycle.AttemptRoot, cycle.AttemptRoot
	a.Stage, a.Outcome, a.Reason, a.FinishedUnix = factory.AssignmentFinished, factory.Succeeded, factory.AssignReasonReported, 1700000100
	a.Result = &factory.AssignmentResult{
		AssignmentID: reviewer, RunID: a.Run, Phase: project.ExecutionReview,
		Status: "completed", Summary: "verdict recorded", Reported: true, RecordedUnix: 1700000100,
		PhaseResult: &factory.PhaseResult{
			Version: factory.PhaseResultVersion, Phase: project.ExecutionReview, Status: "completed",
			Summary: "verdict recorded",
			Output: &factory.PhaseOutput{Clear: &factory.ClearReviewReport{
				Assessments: assessments, CycleID: cycle.ID, Conclusion: factory.ClearReviewConclusion,
				Head: head, BaseOID: base, CoverageDigest: coverage,
			}},
		},
	}
	if err := a.Validate(); err != nil {
		t.Fatalf("clear reviewer invalid: %v", err)
	}
	return a
}

func unionPgAdmitJev(t *testing.T, ctx context.Context, db *store.Store, p factory.Publication, reviewer string, intents []string) (factory.Publication, factory.DeliveryAcceptance, factory.DeliveryDecision, string) {
	t.Helper()
	owner, err := db.Assignment(ctx, p.AssignmentID)
	if err != nil {
		t.Fatal(err)
	}
	finding := correctionTestFinding(1, true, factory.FindingFixVerification)
	digest := correctionTestDigest(t, finding.Key)
	cycle := factory.ReviewCycle{
		PublicationID: p.ID, AttemptRoot: owner.AttemptRoot,
		ID:                 factory.ReviewCycleID(p.ID, owner.AttemptRoot, 1, "review"),
		Phase:              "review",
		TriggerDecision:    "initial-review",
		TriggerReason:      "first review of the published head",
		Head:               p.Candidate,
		BaseOID:            p.PRCreate.BaseOID,
		FindingDigest:      digest,
		ReviewerAssignment: reviewer,
		Round:              1,
	}
	p.ReviewCycles = append(p.ReviewCycles, cycle)
	scope, err := factory.FindingContentDigest(nil)
	if err != nil {
		t.Fatal(err)
	}
	fingerprintInput := factory.DeliveryFingerprintInput{
		Reviews:       []string{reviewer},
		Conversations: []string{strings.Repeat("2", 64)},
		AttemptRoot:   owner.AttemptRoot,
		PublicationID: p.ID,
		ScopeDigest:   scope,
		FixDigest:     scope,
		DiffDigest:    scope,
		CheckDigest:   scope,
		TreeDigest:    scope,
		Head:          p.Candidate,
		Base:          p.PRCreate.BaseOID,
		Model:         "jev-test",
		Rubric:        factory.DeliveryRubric,
		Round:         1,
	}
	fingerprint, err := factory.DeliveryFingerprint(fingerprintInput)
	if err != nil {
		t.Fatal(err)
	}
	answer, err := readyDeliveryAnswer(fingerprint)
	if err != nil {
		t.Fatal(err)
	}
	decision := factory.DeliveryDecision{Answer: answer, Effective: factory.DeliveryEffectiveReady, ReviewsFit: true}
	if err := decision.Validate(); err != nil {
		t.Fatal(err)
	}
	published, acceptance, err := AdmitJevAcceptance(JevAdmissionInput{
		Publication:       p,
		Cycle:             cycle,
		Carried:           []factory.Finding{finding},
		FingerprintInput:  fingerprintInput,
		Fingerprint:       fingerprint,
		Decision:          decision,
		DecisionID:        strings.Repeat("9", 32),
		ScopeDigest:       scope,
		PolicyRevision:    "policy-revision-9",
		CheckDigest:       scope,
		DiffBase:          p.PRCreate.BaseOID,
		ResolutionIntents: intents,
	})
	if err != nil {
		t.Fatal(err)
	}
	published.Revision = p.Revision + 1
	if err := db.UpdatePublication(ctx, published); err != nil {
		t.Fatal(err)
	}
	return published, acceptance, decision, fingerprint
}

func servingUnionBinding(t *testing.T, m factory.Merge, acceptance factory.DeliveryAcceptance, decision factory.DeliveryDecision) factory.MergeAuthorization {
	t.Helper()
	fingerprint := strings.Repeat("e", 64)
	decisionID := ""
	if acceptance.Kind == factory.AcceptanceJevAcceptance {
		fingerprint, decisionID = decision.Answer.Fingerprint, acceptance.DecisionID
	}
	return factory.MergeAuthorization{
		Generation: factory.MergeGeneration{
			Authority: m.Authority, PublicationID: m.PublicationID,
			DecisionID:        decisionID,
			NativeOperationID: factory.MergeGenerationOperationID(m.PublicationID, 1),
			Fingerprint:       fingerprint, Head: m.HeadOID, Base: m.BaseOID, DiffBase: m.BaseOID,
			ChecksDigest:   strings.Repeat("c", 64),
			AcceptanceKind: acceptance.Kind, AcceptanceID: acceptance.ID,
			AttemptRoot: acceptance.AttemptRoot, OriginReceipt: acceptance.OriginReceipt,
			Generation:         1,
			ReviewRound:        acceptance.Round,
			AssessmentRevision: 2, ObservedNativeRev: 12, NotAfter: 1300,
		},
		Decision:             decision,
		ResolutionsConfirmed: true,
	}
}

func grantPolicy() factory.RepositoryPolicy {
	return factory.RepositoryPolicy{
		Repository: 42, GrantedBy: 7, Enabled: true,
		TargetBranch: "refs/heads/main",
		Roles: map[string]factory.RoleSelection{
			project.RoleCoder:    {Harness: project.FactoryHarnessCodex, HarnessVers: "0.157.1", Model: "test"},
			project.RoleReviewer: {Harness: project.FactoryHarnessCodex, HarnessVers: "0.157.1", Model: "test"},
		},
		Checks:              []string{"native-ci/build"},
		MergeMethod:         factory.MergeFastForward,
		Publish:             factory.ActorBindingRef{TokenID: 11, ActorID: 12, Kind: factory.OpRefPublish},
		Create:              factory.ActorBindingRef{TokenID: 11, ActorID: 12, Kind: factory.OpPRCreate},
		Review:              factory.ActorBindingRef{TokenID: 13, ActorID: 14, Kind: factory.OpReviewSubmit},
		Merge:               factory.ActorBindingRef{TokenID: 15, ActorID: 16, Kind: factory.OpMerge},
		AttemptLimits:       factory.DefaultAttemptLimits(),
		ReassessmentMinutes: factory.DefaultReassessmentMinutes,
		MaxConcurrent:       2,
	}
}

func grantProfile() *project.Profile {
	return &project.Profile{
		ID: project.RockyHeadless, Distribution: "rocky", Version: "9.6", Interface: "headless",
		Architecture: "amd64", Image: "sha256:" + strings.Repeat("b", 64), Revision: strings.Repeat("c", 40),
	}
}

func grantProject(t *testing.T, c *Coordinator, id string) {
	t.Helper()
	ctx := context.Background()
	if err := c.Store.UpsertUser(ctx, store.User{ID: 7, Login: "soda-tester"}); err != nil {
		t.Fatal(err)
	}
	if err := c.Store.CreateProject(ctx, store.Project{ID: id, Name: "repo", RepositoryID: 42, OwnerID: 7, Repository: "soda-tester/repo"}); err != nil {
		t.Fatal(err)
	}
}

func grantFullAuthority(t *testing.T, c *Coordinator) {
	t.Helper()
	ctx := context.Background()
	if _, err := c.ApplyPolicy(ctx, factory.NewID(), "native:7", 0, grantPolicy()); err != nil {
		t.Fatal(err)
	}
	if _, err := c.ApplyOperatorGrant(ctx, factory.NewID(), "os-uid:0", 0,
		factory.OperatorGrant{Repository: 42, GrantedBy: 1, Active: true, MaxConcurrent: 2}); err != nil {
		t.Fatal(err)
	}
	if _, err := c.ApplyCapacity(ctx, factory.NewID(), "os-uid:0", 0,
		factory.Capacity{UpdatedBy: 1, MaxConcurrentRuns: 2, MaxQueued: 4}); err != nil {
		t.Fatal(err)
	}
	if _, err := c.ApplySponsorship(ctx, factory.NewID(), "native:9", 0,
		factory.Sponsorship{
			Repository: 42, GrantedBy: 9, ActorID: 9, ProjectID: "p" + strings.Repeat("a", 24), Connection: "conn-1", GrantID: "grant-1",
			Generation: 1, Roles: []string{project.RoleCoder}, AllowanceMinutes: 60, MaxConcurrent: 1, Active: true,
		}); err != nil {
		t.Fatal(err)
	}
	if _, err := c.ApplyEnvironmentGrant(ctx, factory.NewID(), "native:7", 0,
		project.EnvironmentGrant{Repository: 42, Owner: 7, Profile: grantProfile(), Active: true}); err != nil {
		t.Fatal(err)
	}
}
