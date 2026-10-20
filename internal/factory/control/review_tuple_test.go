package control

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
)

// Candidate-tuple seeding at review adoption (R20) against real
// PostgreSQL: the first honestly observed merge-base persists as the
// publication's candidate tuple at revision zero, a later review
// observing a new base advances it under CAS, and unobserved,
// moved-base receipts seed nothing. Every test below opens its own
// owned disposable database.

// tupleReviewSeed records the reviewer report and installs a reviewer
// double whose adopted receipt carries the given merge-base.
func tupleReviewSeed(t *testing.T, p factory.Publication, fx *publishFixture, runID, mergeBase string) *fakeReviewer {
	t.Helper()
	exec := &fakeReviewer{
		observed: factory.ReviewObservation{NativeRev: 9, ObservedUnix: time.Now().Unix()},
		outcome:  committedOutcome(`{"review_id":3}`),
		adopted: factory.ReviewOutcome{
			ReviewID: 3, CommentID: 4, ReviewerID: 6, PRID: p.PRID,
			PRNumber: p.PRNumber, IssueID: p.PRCreate.IssueID,
			HeadOID: p.Candidate, BaseOID: p.PRCreate.BaseOID,
			CommitID: p.Candidate, Event: "APPROVED", MergeBaseOID: mergeBase,
		},
	}
	exec.adopted.Operation = exec.outcome
	fx.coord.Reviews = exec
	ctx := context.Background()
	view, err := fx.db.FactoryRunView(ctx, runID)
	if err != nil {
		t.Fatal(err)
	}
	assignment, err := fx.db.Assignment(ctx, view.Attempt)
	if err != nil {
		t.Fatal(err)
	}
	pub, err := fx.db.PublicationByAssignment(ctx, assignment.PublicationAssignment)
	if err != nil {
		t.Fatal(err)
	}
	cycleID := reviewerCycleID(pub, assignment)
	if cycleID == "" {
		t.Fatalf("reviewer admitted no review cycle: reviewer=%s", assignment.ID)
	}
	output := fencedResult(t, factory.PhaseResult{
		Version: factory.PhaseResultVersion, Phase: project.ExecutionReview, Status: "completed",
		Summary: "solid",
		Output: &factory.PhaseOutput{Clear: &factory.ClearReviewReport{
			Conclusion: factory.ClearReviewConclusion, CycleID: cycleID,
			Head: assignment.SourceCommit, BaseOID: pub.PRCreate.BaseOID,
			CoverageDigest: strings.Repeat("c", 64),
		}},
	})
	run, err := fx.db.FactoryRun(ctx, runID)
	if err != nil {
		t.Fatal(err)
	}
	accountChildResult(t, fx, run, output)
	return exec
}

func readTuple(t *testing.T, fx *publishFixture, assignmentID string) *factory.CandidateTuple {
	t.Helper()
	p, err := fx.db.PublicationByAssignment(context.Background(), assignmentID)
	if err != nil {
		t.Fatal(err)
	}
	return p.Tuple
}

func TestSubmitReviewForRunSeedsCandidateTuple(t *testing.T) {
	ctx := context.Background()
	fx, a, p, run := reviewRunSeed(t, true)
	tupleReviewSeed(t, p, fx, run.ID, strings.Repeat("d", 40))
	if _, err := fx.coord.SubmitReviewForRun(ctx, run.ID); err != nil {
		t.Fatalf("submit refused: %v", err)
	}
	tuple := readTuple(t, fx, a.ID)
	if tuple == nil {
		t.Fatal("observed merge-base seeded nothing")
	}
	want := factory.CandidateTuple{Head: p.Candidate, Base: p.PRCreate.BaseOID, DiffBase: strings.Repeat("d", 40), Revision: 0}
	if *tuple != want {
		t.Fatalf("tuple = %+v, want %+v", tuple, want)
	}
}

func TestSubmitReviewForRunSkipsTupleWithoutMergeBase(t *testing.T) {
	ctx := context.Background()
	fx, a, p, run := reviewRunSeed(t, true)
	tupleReviewSeed(t, p, fx, run.ID, "")
	if _, err := fx.coord.SubmitReviewForRun(ctx, run.ID); err != nil {
		t.Fatalf("submit refused: %v", err)
	}
	if tuple := readTuple(t, fx, a.ID); tuple != nil {
		t.Fatalf("unobserved merge-base seeded %+v", tuple)
	}
}

func TestSubmitReviewForRunAdvancesTupleWithoutChurn(t *testing.T) {
	ctx := context.Background()
	fx, a, p, run := reviewRunSeed(t, true)
	exec := tupleReviewSeed(t, p, fx, run.ID, strings.Repeat("d", 40))
	if _, err := fx.coord.SubmitReviewForRun(ctx, run.ID); err != nil {
		t.Fatal(err)
	}
	exec.adopted.MergeBaseOID = strings.Repeat("e", 40)
	if _, err := fx.coord.SubmitReviewForRun(ctx, run.ID); err != nil {
		t.Fatal(err)
	}
	tuple := readTuple(t, fx, a.ID)
	want := factory.CandidateTuple{Head: p.Candidate, Base: p.PRCreate.BaseOID, DiffBase: strings.Repeat("e", 40), Revision: 1}
	if tuple == nil || *tuple != want {
		t.Fatalf("tuple = %+v, want %+v", tuple, want)
	}
	before, err := fx.db.PublicationByAssignment(ctx, a.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fx.coord.SubmitReviewForRun(ctx, run.ID); err != nil {
		t.Fatal(err)
	}
	after, err := fx.db.PublicationByAssignment(ctx, a.ID)
	if err != nil {
		t.Fatal(err)
	}
	if after.Revision != before.Revision || after.Tuple == nil || *after.Tuple != want {
		t.Fatalf("repeat submit churned: revision %d->%d tuple=%+v", before.Revision, after.Revision, after.Tuple)
	}
}

func TestSubmitReviewForRunSkipsTupleOnMovedBase(t *testing.T) {
	ctx := context.Background()
	fx, a, _, run := reviewRunSeed(t, true)
	p, err := fx.db.PublicationByAssignment(ctx, a.ID)
	if err != nil {
		t.Fatal(err)
	}
	exec := tupleReviewSeed(t, p, fx, run.ID, strings.Repeat("d", 40))
	exec.adopted.BaseOID = strings.Repeat("c", 40)
	if _, err := fx.coord.SubmitReviewForRun(ctx, run.ID); err != nil {
		t.Fatalf("submit refused: %v", err)
	}
	if tuple := readTuple(t, fx, a.ID); tuple != nil {
		t.Fatalf("moved base seeded %+v", tuple)
	}
}
