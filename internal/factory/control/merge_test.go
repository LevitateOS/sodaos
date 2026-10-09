package control

import (
	"context"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
)

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

func TestAttemptSlotRemainsActiveUntilConfirmedMerge(t *testing.T) {
	ctx := context.Background()
	fx, p := mergeSeed(t, 3)
	allowance, err := fx.publish.db.AttemptAllowance(ctx, p.Repository, p.Issue)
	if err != nil || !allowance.Active || allowance.Closed {
		t.Fatalf("published coder attempt allowance = %+v, %v", allowance, err)
	}
	if p.Stage != factory.PublicationPublished || p.PRNumber <= 0 {
		t.Fatalf("seed publication is not awaiting review: %+v", p)
	}

	secondHead := fx.publish.seed.accept(t, 4, "d444444444444444444444444")
	fx.publish.seed.queue(t, 4, secondHead.ID)
	waiting := DispatchPass(ctx, fx.publish.seed.deps())
	if len(waiting.Launched) != 0 || waitReason(waiting, 4) != WaitRepository {
		t.Fatalf("second issue bypassed active attempt slot: %+v", waiting)
	}

	fx.wire(happyMerger())
	if report := fx.pass(t); len(report.Merged) != 1 {
		t.Fatalf("confirmed merge report = %+v", report)
	}
	allowance, err = fx.publish.db.AttemptAllowance(ctx, p.Repository, p.Issue)
	if err != nil || !allowance.Closed || allowance.Active {
		t.Fatalf("merged coder attempt allowance = %+v, %v", allowance, err)
	}

	ready := DispatchPass(ctx, fx.publish.seed.deps())
	if len(ready.Launched) != 1 || ready.Launched[0].Issue != 4 {
		t.Fatalf("confirmed merge did not release the next attempt slot: %+v", ready)
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
