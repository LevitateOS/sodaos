package control

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
)

type fakeMerger struct {
	ledger  map[string]factory.OperationOutcome
	observe func(factory.MergeWork) (factory.MergeObservation, error)
	submit  func(factory.MergeWork) (factory.OperationOutcome, error)
	lookup  func(string) (factory.OperationOutcome, error)
	cancel  func(string) (factory.OperationOutcome, error)
	adopt   func(factory.MergeWork, factory.OperationOutcome) (factory.MergeOutcome, error)
	confirm func(factory.MergeWork) (factory.MergeConfirmation, error)

	observes []factory.MergeWork
	submits  []factory.MergeWork
	lookups  []string
	cancels  []string
	confirms []factory.MergeWork
}

func (f *fakeMerger) ObserveMerge(ctx context.Context, work factory.MergeWork) (factory.MergeObservation, error) {
	f.observes = append(f.observes, work)
	return f.observe(work)
}

func (f *fakeMerger) SubmitMerge(ctx context.Context, work factory.MergeWork) (factory.OperationOutcome, error) {
	f.submits = append(f.submits, work)
	outcome, err := f.submit(work)
	if err == nil && !outcome.NotObserved {
		outcome.OperationID, outcome.InstallationID, outcome.Kind = work.OperationID, "test-installation", factory.OpMerge
		outcome.ActorID, outcome.RepositoryID = work.ActorID, work.Repository
		f.ledger[work.OperationID] = outcome
	}
	return outcome, err
}

func (f *fakeMerger) LookupOp(ctx context.Context, operationID string) (factory.OperationOutcome, error) {
	f.lookups = append(f.lookups, operationID)
	if f.lookup != nil {
		return f.lookup(operationID)
	}
	outcome, ok := f.ledger[operationID]
	if !ok {
		return factory.OperationOutcome{NotObserved: true}, nil
	}
	return outcome, nil
}

func (f *fakeMerger) CancelOp(ctx context.Context, operationID string) (factory.OperationOutcome, error) {
	f.cancels = append(f.cancels, operationID)
	if f.cancel != nil {
		return f.cancel(operationID)
	}
	outcome := f.ledger[operationID]
	if outcome.Effect == factory.OpEffectCommitted {
		outcome.Cancellation = factory.OpCancelTooLate
	} else {
		outcome.Effect, outcome.Cancellation, outcome.Reason = factory.OpEffectNotCommitted, factory.OpCancelCancelled, "cancelled_before_admission"
	}
	outcome.OperationID, outcome.InstallationID = operationID, "test-installation"
	f.ledger[operationID] = outcome
	return outcome, nil
}

func (f *fakeMerger) AdoptMerge(work factory.MergeWork, outcome factory.OperationOutcome) (factory.MergeOutcome, error) {
	return f.adopt(work, outcome)
}

func (f *fakeMerger) ObserveCompletion(ctx context.Context, work factory.MergeWork) (factory.MergeConfirmation, error) {
	f.confirms = append(f.confirms, work)
	return f.confirm(work)
}

func happyMerger() *fakeMerger {
	return &fakeMerger{
		ledger: map[string]factory.OperationOutcome{},
		observe: func(w factory.MergeWork) (factory.MergeObservation, error) {
			return factory.MergeObservation{
				Checks: factory.ObservedChecks{
					Checks:    []factory.ObservedCheck{{Context: "ci", State: factory.CheckStateSuccess}},
					NativeRev: 9, ObservedContexts: 1, HeadTip: w.HeadOID, BaseTip: w.BaseOID, Complete: true,
				},
				NativeRev: 9, ObservedUnix: time.Now().Unix(),
				PRID: w.PRID, PRNumber: w.PRNumber, IssueID: w.IssueID, PRAuthorID: w.PRAuthorID,
				HeadRef: w.HeadRef, BaseRef: w.BaseRef, HeadOID: w.HeadOID, BaseOID: w.BaseOID,
				ReviewerID: w.ReviewerID, ReviewID: 31,
			}, nil
		},
		submit: func(w factory.MergeWork) (factory.OperationOutcome, error) {
			return committedOutcome(`{"pr_number":9}`), nil
		},
		adopt: func(w factory.MergeWork, o factory.OperationOutcome) (factory.MergeOutcome, error) {
			return factory.MergeOutcome{
				Operation: o, HeadRef: w.HeadRef, BaseRef: w.BaseRef, HeadOID: w.HeadOID, BaseOID: w.BaseOID,
				MergedCommit: w.HeadOID, PRNumber: w.PRNumber, PRID: w.PRID, IssueID: w.IssueID, ActorID: w.ActorID,
			}, nil
		},
		confirm: func(w factory.MergeWork) (factory.MergeConfirmation, error) {
			now := time.Now().Unix()
			return factory.MergeConfirmation{MergedCommit: w.HeadOID, BaseTip: w.HeadOID, MergerID: w.ActorID, MergedUnix: now, ClosedUnix: now, NativeRev: 9, ObservedUnix: now, IssueClosed: true}, nil
		},
	}
}

// pendingMerger observes pending live checks: MergePass opens its row
// but reconcile waits, leaving the row open and undriven for tests
// that regress the stored evidence afterwards.
func pendingMerger() *fakeMerger {
	exec := happyMerger()
	observe := exec.observe
	exec.observe = func(w factory.MergeWork) (factory.MergeObservation, error) {
		observation, err := observe(w)
		if err != nil {
			return observation, err
		}
		observation.Checks.Checks = nil
		observation.Checks.ObservedContexts = 0
		return observation, nil
	}
	return exec
}

type mergeFixture struct {
	publish *publishFixture
	exec    *fakeMerger
}

func mergeSeed(t *testing.T, issue int64) (*mergeFixture, factory.Publication) {
	t.Helper()
	fx := publishSeed(t)
	ctx := context.Background()
	policy, err := fx.db.RepositoryPolicy(ctx, fx.seed.repo)
	if err != nil {
		t.Fatal(err)
	}
	// The merge reviewer must be independent of the PR author.
	policy.Review = factory.ActorBindingRef{TokenID: 2, ActorID: 6, Kind: factory.OpReviewSubmit}
	if err := fx.db.SaveRepositoryPolicy(ctx, policy); err != nil {
		t.Fatal(err)
	}
	a := fx.finishReported(t, issue)
	fx.wire(happyPublisher())
	fx.pass(t)
	p, err := fx.db.PublicationByAssignment(ctx, a.ID)
	if err != nil {
		t.Fatal(err)
	}
	if p.Stage != factory.PublicationPublished {
		t.Fatalf("merge seed publication not published: %+v", p)
	}
	current, err := fx.db.RepositoryPolicy(ctx, fx.seed.repo)
	if err != nil {
		t.Fatal(err)
	}
	assessment := factory.CheckAssessment{
		Results: []factory.CheckResult{{Context: "ci", State: factory.CheckStateSuccess, Passed: true}},
		Checks:  []string{"ci"}, Repository: p.Repository, PRNumber: p.PRNumber, PRID: p.PRID, IssueID: p.PRCreate.IssueID,
		PolicyRevision: current.Revision, NativeRev: 9, AssessedUnix: time.Now().Unix(), ObservedContexts: 1,
		HeadRef: p.PRCreate.HeadRef, BaseRef: p.PRCreate.BaseRef, HeadOID: p.PRCreate.HeadOID, BaseOID: p.PRCreate.BaseOID,
		ChecksDigest: factory.ChecksDigest([]string{"ci"}), Verdict: factory.CheckPass, Reason: factory.CheckReasonPass,
	}
	if _, err := fx.db.RecordCheckAssessment(ctx, assessment); err != nil {
		t.Fatal(err)
	}
	return &mergeFixture{publish: fx}, p
}

func (fx *mergeFixture) wire(exec *fakeMerger) {
	fx.exec = exec
	fx.publish.coord.Merges = exec
}

func (fx *mergeFixture) pass(t *testing.T) MergeReport {
	t.Helper()
	report := fx.publish.coord.MergePass(context.Background())
	if len(report.Errors) != 0 {
		t.Fatalf("merge pass: %+v", report)
	}
	return report
}

func (fx *mergeFixture) merge(t *testing.T, p factory.Publication) factory.Merge {
	t.Helper()
	m, err := fx.publish.db.MergeByPublication(context.Background(), p.ID)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func TestMergePassMergesOneExactPR(t *testing.T) {
	fx, p := mergeSeed(t, 3)
	exec := happyMerger()
	fx.wire(exec)
	first := fx.pass(t)
	m := fx.merge(t, p)
	if len(first.Merged) != 1 || m.Stage != factory.MergeMerged || m.MergedCommit != p.PRCreate.HeadOID || m.Operation.Effect != factory.OpEffectCommitted {
		t.Fatalf("merge linkage: %+v %+v", first, m)
	}
	if m.Operation.Completion != factory.OpCompletionComplete || m.MergedUnix == 0 || m.ClosedUnix == 0 {
		t.Fatalf("completion unconfirmed: %+v", m)
	}
	if m.Operation.Work == nil || len(exec.observes) != 1 || len(exec.submits) != 1 || len(exec.confirms) != 1 {
		t.Fatalf("single observed intent: %+v", m)
	}
	fx.pass(t)
	if len(exec.submits) != 1 || len(exec.confirms) != 1 {
		t.Fatalf("replayed mutations: %+v", exec)
	}
	control, err := fx.publish.db.IssueControl(context.Background(), p.Repository, p.Issue)
	if err != nil {
		t.Fatalf("dependant trigger left no assessment: %v", err)
	}
	if control.Readiness != factory.ReadinessQueued {
		t.Fatalf("completed issue not runnable: %+v", control)
	}
}

func TestMergePassReconcilesLostSubmitReply(t *testing.T) {
	fx, p := mergeSeed(t, 3)
	exec := happyMerger()
	calls := 0
	exec.submit = func(w factory.MergeWork) (factory.OperationOutcome, error) {
		calls++
		if calls == 1 {
			// The native side committed, but the reply never arrived.
			exec.ledger[w.OperationID] = factory.OperationOutcome{
				OperationID: w.OperationID, InstallationID: "test-installation", Kind: factory.OpMerge,
				ActorID: w.ActorID, RepositoryID: w.Repository,
				Effect: factory.OpEffectCommitted, Cancellation: factory.OpCancelNone,
				Completion: factory.OpCompletionComplete, Receipt: []byte(`{"pr_number":9}`),
			}
			return factory.OperationOutcome{NotObserved: true}, nil
		}
		return committedOutcome(`{"pr_number":9}`), nil
	}
	fx.wire(exec)
	first := fx.pass(t)
	m := fx.merge(t, p)
	if m.Stage != factory.MergeOpen || len(first.Waits) != 1 || first.Waits[0].Reason != "submit_unconfirmed" {
		t.Fatalf("lost reply misadvanced: %+v %+v", first, m)
	}
	intent := *m.Operation.Work
	second := fx.pass(t)
	m = fx.merge(t, p)
	if m.Stage != factory.MergeMerged || len(second.Merged) != 1 {
		t.Fatalf("lost reply unrecovered: %+v %+v", second, m)
	}
	if calls != 1 || *m.Operation.Work != intent {
		t.Fatal("lost reply resubmitted or rewrote its intent")
	}
}

func TestMergePassSkipsRowWithoutCurrentPass(t *testing.T) {
	fx, p := mergeSeed(t, 3)
	ctx := context.Background()
	stored, err := fx.publish.db.CheckAssessment(ctx, p.Repository, p.PRNumber)
	if err != nil {
		t.Fatal(err)
	}
	stored.Verdict, stored.Reason = factory.CheckFailed, factory.CheckReasonFailed
	stored.Results[0].Passed, stored.Results[0].State = false, "failure"
	stored.Revision = 0
	if _, err := fx.publish.db.RecordCheckAssessment(ctx, stored); err != nil {
		t.Fatal(err)
	}
	fx.wire(happyMerger())
	if report := fx.pass(t); len(report.Merged) != 0 {
		t.Fatalf("failing head merged: %+v", report)
	}
	if _, err := fx.publish.db.MergeByPublication(ctx, p.ID); err == nil {
		t.Fatal("failing head opened a merge row")
	}
	stored.Verdict, stored.Reason = factory.CheckPass, factory.CheckReasonPass
	stored.Results[0].Passed, stored.Results[0].State = true, factory.CheckStateSuccess
	stored.Revision = 0
	if _, err := fx.publish.db.RecordCheckAssessment(ctx, stored); err != nil {
		t.Fatal(err)
	}
	fx.pass(t)
	m := fx.merge(t, p)
	if m.Stage != factory.MergeMerged {
		t.Fatalf("passing head unmerged: %+v", m)
	}
}

func TestMergePassWaitsForStaleCheckEvidence(t *testing.T) {
	fx, p := mergeSeed(t, 3)
	ctx := context.Background()
	// MergePass only opens rows behind a current pass, so open the
	// reconciled row first (pending live evidence leaves it open and
	// undriven), then regress the stored evidence.
	fx.wire(pendingMerger())
	setup := MergeReport{Merged: []MergeLink{}}
	fx.publish.coord.mergeOne(ctx, p, &setup)
	if len(setup.Errors) != 0 {
		t.Fatalf("setup merge open: %+v", setup)
	}
	stored, err := fx.publish.db.CheckAssessment(ctx, p.Repository, p.PRNumber)
	if err != nil {
		t.Fatal(err)
	}
	stored.HeadOID = strings.Repeat("c", 40)
	stored.Revision = 0
	if _, err := fx.publish.db.RecordCheckAssessment(ctx, stored); err != nil {
		t.Fatal(err)
	}
	fx.wire(happyMerger())
	first := fx.pass(t)
	m := fx.merge(t, p)
	if m.Stage != factory.MergeOpen || len(first.Waits) != 1 || first.Waits[0].Reason != "check_evidence_stale" {
		t.Fatalf("stale evidence misadvanced: %+v %+v", first, m)
	}
	stored.HeadOID = p.PRCreate.HeadOID
	stored.Revision = 0
	if _, err := fx.publish.db.RecordCheckAssessment(ctx, stored); err != nil {
		t.Fatal(err)
	}
	second := fx.pass(t)
	m = fx.merge(t, p)
	if m.Stage != factory.MergeMerged || len(second.Merged) != 1 {
		t.Fatalf("fresh evidence unsettled: %+v %+v", second, m)
	}
}

func TestMergePassRefusesFailedCheckEvidence(t *testing.T) {
	fx, p := mergeSeed(t, 3)
	ctx := context.Background()
	// Open the reconciled row behind the seeded current pass first;
	// the failure below must refuse an already-open row, since
	// MergePass never opens rows for failing heads.
	fx.wire(pendingMerger())
	setup := MergeReport{Merged: []MergeLink{}}
	fx.publish.coord.mergeOne(ctx, p, &setup)
	if len(setup.Errors) != 0 {
		t.Fatalf("setup merge open: %+v", setup)
	}
	stored, err := fx.publish.db.CheckAssessment(ctx, p.Repository, p.PRNumber)
	if err != nil {
		t.Fatal(err)
	}
	stored.Verdict, stored.Reason = factory.CheckFailed, factory.CheckReasonFailed
	stored.Results[0].Passed, stored.Results[0].State = false, "failure"
	stored.Revision = 0
	if _, err := fx.publish.db.RecordCheckAssessment(ctx, stored); err != nil {
		t.Fatal(err)
	}
	fx.wire(happyMerger())
	fx.pass(t)
	m := fx.merge(t, p)
	if m.Stage != factory.MergeFailed || m.Reason != factory.MergeReasonInvalid {
		t.Fatalf("failed checks merged: %+v", m)
	}
}

func TestMergePassWaitsForPendingChecks(t *testing.T) {
	fx, p := mergeSeed(t, 3)
	exec := happyMerger()
	observe := exec.observe
	pending := true
	exec.observe = func(w factory.MergeWork) (factory.MergeObservation, error) {
		observation, err := observe(w)
		if err != nil {
			return observation, err
		}
		if pending {
			observation.Checks.Checks = nil
			observation.Checks.ObservedContexts = 0
		}
		return observation, nil
	}
	fx.wire(exec)
	first := fx.pass(t)
	m := fx.merge(t, p)
	if m.Stage != factory.MergeOpen || len(first.Waits) != 1 || first.Waits[0].Reason != "check_evidence_pending" {
		t.Fatalf("pending checks misadvanced: %+v %+v", first, m)
	}
	pending = false
	second := fx.pass(t)
	m = fx.merge(t, p)
	if m.Stage != factory.MergeMerged || len(second.Merged) != 1 || len(exec.submits) != 1 {
		t.Fatalf("settled checks unmerged: %+v %+v", second, m)
	}
}

func TestMergePassRefusesRegressedChecks(t *testing.T) {
	fx, p := mergeSeed(t, 3)
	exec := happyMerger()
	observe := exec.observe
	exec.observe = func(w factory.MergeWork) (factory.MergeObservation, error) {
		observation, err := observe(w)
		if err != nil {
			return observation, err
		}
		observation.Checks.Checks[0].State = "failure"
		return observation, nil
	}
	fx.wire(exec)
	fx.pass(t)
	m := fx.merge(t, p)
	if m.Stage != factory.MergeFailed || m.Reason != factory.MergeReasonInvalid || len(exec.submits) != 0 {
		t.Fatalf("regressed checks merged: %+v", m)
	}
}

func TestMergePassWaitsForNativeCompletion(t *testing.T) {
	fx, p := mergeSeed(t, 3)
	exec := happyMerger()
	exec.submit = func(w factory.MergeWork) (factory.OperationOutcome, error) {
		return factory.OperationOutcome{Effect: factory.OpEffectCommitted, Cancellation: factory.OpCancelNone, Completion: factory.OpCompletionPending, Receipt: []byte(`{"pr_number":9}`)}, nil
	}
	fx.wire(exec)
	first := fx.pass(t)
	m := fx.merge(t, p)
	// A committed ref without finished bookkeeping never finishes.
	if m.Stage != factory.MergeOpen || len(first.Waits) != 1 || first.Waits[0].Reason != "native_completion_pending" {
		t.Fatalf("incomplete merge finished: %+v %+v", first, m)
	}
	for id, outcome := range exec.ledger {
		outcome.Completion = factory.OpCompletionComplete
		exec.ledger[id] = outcome
	}
	second := fx.pass(t)
	m = fx.merge(t, p)
	if m.Stage != factory.MergeMerged || len(second.Merged) != 1 || len(exec.submits) != 1 {
		t.Fatalf("completed merge unsettled: %+v %+v", second, m)
	}
}

func TestMergePassFencesUnconfirmedCompletion(t *testing.T) {
	fx, p := mergeSeed(t, 3)
	exec := happyMerger()
	exec.confirm = func(w factory.MergeWork) (factory.MergeConfirmation, error) {
		return factory.MergeConfirmation{}, &factory.PublicationRefusal{Reason: "completion_unconfirmed"}
	}
	fx.wire(exec)
	fx.pass(t)
	m := fx.merge(t, p)
	if m.Stage != factory.MergeFenced || m.Reason != factory.MergeReasonUnattributed {
		t.Fatalf("unconfirmed completion merged: %+v", m)
	}
}

func TestMergePassFencesIndeterminateEffect(t *testing.T) {
	fx, p := mergeSeed(t, 3)
	exec := happyMerger()
	exec.submit = func(w factory.MergeWork) (factory.OperationOutcome, error) {
		return factory.OperationOutcome{Effect: factory.OpEffectIndeterminate, Cancellation: factory.OpCancelNone}, nil
	}
	fx.wire(exec)
	fx.pass(t)
	m := fx.merge(t, p)
	if m.Stage != factory.MergeFenced || m.Reason != factory.MergeReasonFenced {
		t.Fatalf("indeterminate effect unfinished: %+v", m)
	}
}

func TestMergePassWaitsForApproval(t *testing.T) {
	fx, p := mergeSeed(t, 3)
	exec := happyMerger()
	observe := exec.observe
	approved := false
	exec.observe = func(w factory.MergeWork) (factory.MergeObservation, error) {
		if !approved {
			return factory.MergeObservation{}, &factory.PublicationWait{Reason: "approval_missing"}
		}
		return observe(w)
	}
	fx.wire(exec)
	first := fx.pass(t)
	m := fx.merge(t, p)
	if m.Stage != factory.MergeOpen || len(first.Waits) != 1 || first.Waits[0].Reason != "approval_missing" || len(exec.submits) != 0 {
		t.Fatalf("unapproved candidate misadvanced: %+v %+v", first, m)
	}
	approved = true
	second := fx.pass(t)
	m = fx.merge(t, p)
	if m.Stage != factory.MergeMerged || len(second.Merged) != 1 || len(exec.submits) != 1 {
		t.Fatalf("approved candidate unmerged: %+v %+v", second, m)
	}
}

func TestMergePassRefusesChangeRequest(t *testing.T) {
	fx, p := mergeSeed(t, 3)
	exec := happyMerger()
	exec.observe = func(w factory.MergeWork) (factory.MergeObservation, error) {
		return factory.MergeObservation{}, &factory.PublicationRefusal{Reason: "changes_requested"}
	}
	fx.wire(exec)
	fx.pass(t)
	m := fx.merge(t, p)
	if m.Stage != factory.MergeFailed || m.Reason != factory.MergeReasonInvalid || len(exec.submits) != 0 {
		t.Fatalf("rejected candidate merged: %+v", m)
	}
}

func TestMergePassRequiresCurrentAuthority(t *testing.T) {
	fx, p := mergeSeed(t, 3)
	fx.wire(happyMerger())
	fx.pass(t)
	m := fx.merge(t, p)
	if m.Stage != factory.MergeMerged {
		t.Fatalf("seed merge unsettled: %+v", m)
	}
	// A later policy change strands nothing: new merges bind the new
	// authority, and changed authority waits instead of submitting.
	a := fx.publish.finishReported(t, 4)
	fx.publish.wire(happyPublisher())
	fx.publish.pass(t)
	second, err := fx.publish.db.PublicationByAssignment(context.Background(), a.ID)
	if err != nil {
		t.Fatal(err)
	}
	policy, err := fx.publish.db.RepositoryPolicy(context.Background(), second.Repository)
	if err != nil {
		t.Fatal(err)
	}
	policy.Paused = true
	if err := fx.publish.db.SaveRepositoryPolicy(context.Background(), policy); err != nil {
		t.Fatal(err)
	}
	report := fx.pass(t)
	waited := false
	for _, wait := range report.Waits {
		waited = waited || wait.Reason == "authority_ineffective"
	}
	if !waited {
		t.Fatalf("changed authority submitted: %+v", report)
	}
}

func TestMergePassWithdrawsBeforeSubmit(t *testing.T) {
	fx, p := mergeSeed(t, 3)
	exec := happyMerger()
	exec.observe = func(w factory.MergeWork) (factory.MergeObservation, error) {
		return factory.MergeObservation{}, &factory.PublicationWait{Reason: "native_busy"}
	}
	fx.wire(exec)
	fx.pass(t)
	m := fx.merge(t, p)
	if m.Stage != factory.MergeOpen || m.Operation.Work != nil {
		t.Fatalf("seed observed despite busy native: %+v", m)
	}
	if _, err := fx.publish.db.WithdrawDispatch(context.Background(), p.Repository, "test withdrawal", "soda-tester"); err != nil {
		t.Fatal(err)
	}
	withdrawal := fx.publish.coord.cancelRepositoryMerges(context.Background(), p.Repository)
	m = fx.merge(t, p)
	if withdrawal.Pending || m.Stage != factory.MergeWithdrawn || len(exec.submits) != 0 {
		t.Fatalf("pre-submit withdrawal unconfirmed: %+v %+v", withdrawal, m)
	}
}

func TestMergePassWithdrawalAfterCommitConfirmsCompletion(t *testing.T) {
	fx, p := mergeSeed(t, 3)
	exec := happyMerger()
	submitted := false
	exec.submit = func(w factory.MergeWork) (factory.OperationOutcome, error) {
		submitted = true
		return factory.OperationOutcome{Effect: factory.OpEffectCommitted, Cancellation: factory.OpCancelNone, Completion: factory.OpCompletionPending, Receipt: []byte(`{"pr_number":9}`)}, nil
	}
	fx.wire(exec)
	fx.pass(t)
	if !submitted {
		t.Fatal("seed merge never submitted")
	}
	if _, err := fx.publish.db.WithdrawDispatch(context.Background(), p.Repository, "test withdrawal", "soda-tester"); err != nil {
		t.Fatal(err)
	}
	// Completion still pending: withdrawal cannot finish yet.
	withdrawal := fx.publish.coord.cancelRepositoryMerges(context.Background(), p.Repository)
	m := fx.merge(t, p)
	if !withdrawal.Pending || m.Stage != factory.MergeOpen {
		t.Fatalf("pending completion presented as withdrawn: %+v %+v", withdrawal, m)
	}
	for id, outcome := range exec.ledger {
		outcome.Completion = factory.OpCompletionComplete
		exec.ledger[id] = outcome
	}
	withdrawal = fx.publish.coord.cancelRepositoryMerges(context.Background(), p.Repository)
	m = fx.merge(t, p)
	// The effect stands: withdrawal finishes the bookkeeping honestly.
	if withdrawal.Pending || m.Stage != factory.MergeMerged {
		t.Fatalf("committed merge not confirmed under withdrawal: %+v %+v", withdrawal, m)
	}
}

func TestMergeCompletionReleasesCodeDependant(t *testing.T) {
	fx, p := mergeSeed(t, 3)
	endpoint := fx.publish.seed.decision[p.Issue]
	dependant := factory.Acceptance{
		ID: "d" + strings.Repeat("f", 24), Repository: p.Repository, IssueIndex: "4", Approver: 5, NativeRev: 41,
		TitleDigest: strings.Repeat("a", 64), ContentDigest: strings.Repeat("b", 64), ContentVersion: 2,
		Sources: []factory.SelectedSource{{ID: "11", ContentVersion: 0, Digest: strings.Repeat("c", 64)}},
		Prerequisites: []factory.AcceptedPrerequisite{{
			Occurrence: "21", DependsOn: "9", EndpointRepo: p.Repository, EndpointIssue: p.Issue,
			Outcome: factory.PrereqCode, PrereqAcceptance: endpoint.ID,
		}},
	}
	ctx := context.Background()
	if err := fx.publish.db.AdmitAcceptanceDecision(ctx, dependant); err != nil {
		t.Fatal(err)
	}
	fx.publish.reads.evidence["7/4"] = AcceptanceEvidence{
		Revision:     9,
		Issue:        AcceptanceIssueView{Index: "4", TitleDigest: dependant.TitleDigest, ContentDigest: dependant.ContentDigest, ContentVer: 2, Visible: true},
		Comments:     []AcceptanceComment{{ID: "11", Digest: dependant.Sources[0].Digest, ContentVer: 0, Visible: true}},
		Dependencies: []AcceptanceEdge{{Occurrence: "21", DependsOn: "9", Visible: true}},
	}
	before, err := fx.publish.coord.assessOne(ctx, p.Repository, 4)
	if err != nil {
		t.Fatal(err)
	}
	if before.control.Readiness != factory.ReadinessBlocked {
		t.Fatalf("dependant runnable before completion: %+v", before.control)
	}
	fx.wire(happyMerger())
	fx.pass(t)
	m := fx.merge(t, p)
	if m.Stage != factory.MergeMerged {
		t.Fatalf("seed merge unsettled: %+v", m)
	}
	after, err := fx.publish.db.IssueControl(ctx, p.Repository, 4)
	if err != nil {
		t.Fatal(err)
	}
	if after.Readiness != factory.ReadinessQueued || after.Reason != factory.ReasonEligible {
		t.Fatalf("confirmed completion did not release the dependant: %+v", after)
	}
	if _, err := fx.publish.db.IssueMergeCompletion(ctx, p.Repository, p.Issue); err != nil {
		t.Fatalf("completion record unreadable: %v", err)
	}
}
