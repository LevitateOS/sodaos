package control

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
)

func correctionRun(t *testing.T, fx *publishFixture, a factory.Assignment, candidate string) factory.Run {
	t.Helper()
	publication := fx.publication(t, a)
	_, run := fx.childRun(t, a, publication, project.RoleCoder, publication.Candidate, a.Preparation, true)
	accountChildResult(t, fx, run, correctionOutput(candidate))
	return run
}

func correctionOutput(candidate string) string {
	return "fixed the finding\n```" + factory.ResultFence + "\n{\"status\":\"completed\",\"summary\":\"fixed\",\"candidate\":\"" + candidate + "\",\"review_passed\":false,\"findings\":[]}\n```\n"
}

func TestPublishCorrectionAdvancesHead(t *testing.T) {
	fx, p := checkSeed(t, 6)
	ctx := context.Background()
	a, err := fx.db.Assignment(ctx, p.AssignmentID)
	if err != nil {
		t.Fatal(err)
	}
	candidate := strings.Repeat("d", 40)
	run := correctionRun(t, fx, a, candidate)
	child, err := fx.db.AssignmentByRun(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	queued, err := fx.db.PublishableAssignments(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	for _, assignment := range queued {
		if assignment.ID == child.ID {
			t.Fatalf("correction child was nominated as a new publication owner: %+v", child)
		}
	}
	pass := fx.coord.PublishPass(ctx)
	if len(pass.Errors) != 0 {
		t.Fatalf("publication pass after correction result: %+v", pass)
	}
	if _, err := fx.db.PublicationByAssignment(ctx, child.ID); err == nil {
		t.Fatal("publication pass created a separate child publication")
	}
	linked, err := fx.db.PublicationByAssignment(ctx, a.ID)
	if err != nil || linked.ID != p.ID || linked.PRID != p.PRID || linked.PRNumber != p.PRNumber {
		t.Fatalf("child pass changed the parent's linked PR: %+v %v", linked, err)
	}
	report := fx.coord.PublishCorrection(ctx, run.ID)
	if len(report.Errors) != 0 || len(report.Waits) != 0 || len(report.Corrected) != 1 {
		t.Fatalf("correction report: %+v", report)
	}
	link := report.Corrected[0]
	if link.HeadOID != candidate || link.PublicationID != p.ID {
		t.Fatalf("correction link: %+v", link)
	}
	after, err := fx.db.PublicationByAssignment(ctx, a.ID)
	if err != nil {
		t.Fatal(err)
	}
	if after.Candidate != candidate || after.Run != run.ID || after.Stage != factory.PublicationPublished {
		t.Fatalf("publication head did not advance: %+v", after)
	}
	if after.Publish.Effect != factory.OpEffectCommitted || after.PRCreate.Effect != factory.OpEffectCommitted ||
		after.PRNumber != p.PRNumber || after.PRID != p.PRID {
		t.Fatalf("initial receipts changed: %+v", after)
	}
	if len(after.Corrections) != 1 {
		t.Fatalf("correction unrecorded: %+v", after)
	}
	op := after.Corrections[0]
	if op.OperationID != factory.PublicationOperationID(p.ID, factory.OpRefPublish, 2) ||
		op.Effect != factory.OpEffectCommitted || op.Work == nil ||
		op.Work.Candidate != candidate || op.Work.ExpectedOld != p.Candidate ||
		op.Work.CorrectionNumber != p.PRNumber {
		t.Fatalf("correction op: %+v", op)
	}
	work := fx.exec.pushes[len(fx.exec.pushes)-1]
	if work.AssignmentID != a.ID || work.Publication != p.ID || work.RunID != run.ID {
		t.Fatalf("correction work lost its parent authority or child run: %+v", work)
	}
	if err := after.Validate(); err != nil {
		t.Fatalf("corrected publication invalid: %v", err)
	}
	// The same run never opens a second correction identity.
	replay := fx.coord.PublishCorrection(ctx, run.ID)
	if len(replay.Corrected) != 0 || len(replay.Waits) != 1 || replay.Waits[0].Reason != "correction_recorded" {
		t.Fatalf("correction replay: %+v", replay)
	}
}

func TestPublishCorrectionRefusesWithoutChange(t *testing.T) {
	fx, p := checkSeed(t, 7)
	ctx := context.Background()
	a, err := fx.db.Assignment(ctx, p.AssignmentID)
	if err != nil {
		t.Fatal(err)
	}
	run := correctionRun(t, fx, a, p.Candidate)
	// The unchanged head is not a correction.
	same := fx.coord.PublishCorrection(ctx, run.ID)
	if len(same.Corrected) != 0 || len(same.Waits) != 1 || same.Waits[0].Reason != "candidate_invalid" {
		t.Fatalf("unchanged head: %+v", same)
	}
	native := fx.exec
	observes, submits, pushes := len(native.observes), len(native.submits), len(native.pushes)
	creates, exports := len(native.creates), len(fx.host.exports)
	// An unlinked run cannot correct the publication: a fresh run
	// without a view carries no assignment linkage.
	bare := factory.Run{
		ID: factory.NewID(), ProjectID: a.ProjectID, Role: project.RoleCoder, InputSHA: a.SourceCommit,
		Started: time.Now().Add(-time.Minute), Deadline: time.Now().Add(time.Hour),
		Image: "sha256:" + strings.Repeat("c", 64), Harness: a.Harness, Model: a.Model,
	}
	if err := fx.db.RecordFactoryRun(ctx, bare); err != nil {
		t.Fatal(err)
	}
	bare.Outcome, bare.Reconciled = factory.Succeeded, true
	if err := fx.db.SaveFactoryRun(ctx, bare); err != nil {
		t.Fatal(err)
	}
	unlinked := fx.coord.PublishCorrection(ctx, bare.ID)
	if len(unlinked.Corrected) != 0 || len(unlinked.Waits) != 1 || unlinked.Waits[0].Reason != "run_unlinked" {
		t.Fatalf("unlinked run: %+v", unlinked)
	}
	if len(native.observes) != observes || len(native.submits) != submits || len(native.pushes) != pushes ||
		len(native.creates) != creates || len(fx.host.exports) != exports {
		t.Fatalf("unlinked correction reached native operations: publisher=%+v exports=%+v", native, fx.host.exports)
	}
	if len(fx.dbMustPublication(t, a.ID).Corrections) != 0 {
		t.Fatal("refused corrections polluted the record")
	}
}

func TestWithdrawalCancelsRecordedCorrectionAndKeepsPublishedHead(t *testing.T) {
	fx, p := checkSeed(t, 8)
	ctx := context.Background()
	exec := happyPublisher()
	fx.wire(exec)
	seedPublishedNativeOperations(exec, p)
	intent := *p.Publish.Work
	intent.OperationID = factory.PublicationOperationID(p.ID, factory.OpRefPublish, 2)
	intent.Candidate, intent.ExpectedOld = strings.Repeat("d", 40), p.Candidate
	intent.CorrectionNumber, intent.CorrectionAuthor = p.PRNumber, p.PRCreate.Work.ActorID
	op := factory.PublicationOperation{Work: &intent, OperationID: intent.OperationID, Kind: factory.OpRefPublish, Attempts: 1, UpdatedUnix: time.Now().Unix()}
	p.Corrections = append(p.Corrections, op)
	p.Revision++
	if err := fx.db.UpdatePublication(ctx, p); err != nil {
		t.Fatalf("record correction intent: %v", err)
	}
	exec.ledger[intent.OperationID] = publicationTestOutcome(factory.PublicationWork{
		OperationID: intent.OperationID, Repository: intent.Repository, ActorID: intent.ActorID,
	}, factory.OpRefPublish, pendingOutcome())
	if _, err := fx.db.WithdrawDispatch(ctx, p.Repository, "pause", "operator"); err != nil {
		t.Fatal(err)
	}
	withdrawal := fx.coord.cancelRepositoryPublications(ctx, p.Repository)
	if withdrawal.Pending {
		t.Fatalf("confirmed cancellation remained pending: %+v", withdrawal)
	}
	found := false
	for _, id := range withdrawal.Operations {
		if id == intent.OperationID {
			found = true
		}
	}
	if !found {
		t.Fatalf("correction operation omitted from cancellation set: %+v", withdrawal)
	}
	if countOperation(exec.cancels, intent.OperationID) != 1 {
		t.Fatalf("correction identity not cancelled exactly: %+v", exec.cancels)
	}
	stored, err := fx.db.PublicationByAssignment(ctx, p.AssignmentID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Stage != factory.PublicationPublished || stored.Candidate != p.Candidate ||
		stored.Corrections[0].Effect != factory.OpEffectNotCommitted || stored.Corrections[0].Cancellation != factory.OpCancelCancelled {
		t.Fatalf("withdrawal changed linked publication or lost correction verdict: %+v", stored)
	}
}

func TestWithdrawalWaitsForCommittedCorrectionCompletion(t *testing.T) {
	fx, p := checkSeed(t, 9)
	ctx := context.Background()
	exec := happyPublisher()
	fx.wire(exec)
	seedPublishedNativeOperations(exec, p)
	intent := *p.Publish.Work
	intent.OperationID = factory.PublicationOperationID(p.ID, factory.OpRefPublish, 2)
	intent.Candidate, intent.ExpectedOld = strings.Repeat("d", 40), p.Candidate
	intent.CorrectionNumber, intent.CorrectionAuthor = p.PRNumber, p.PRCreate.Work.ActorID
	outcome := committedOutcome("correction-receipt")
	outcome.Completion = factory.OpCompletionPending
	op := factory.PublicationOperation{
		Work: &intent, OperationID: intent.OperationID, Kind: factory.OpRefPublish,
		Effect: factory.OpEffectCommitted, Cancellation: factory.OpCancelTooLate,
		Completion: factory.OpCompletionPending, Receipt: string(outcome.Receipt), Attempts: 1, UpdatedUnix: time.Now().Unix(),
	}
	p.Corrections = append(p.Corrections, op)
	p.Candidate = intent.Candidate
	p.Revision++
	if err := fx.db.UpdatePublication(ctx, p); err != nil {
		t.Fatalf("record committed correction: %v", err)
	}
	exec.ledger[intent.OperationID] = publicationTestOutcome(factory.PublicationWork{
		OperationID: intent.OperationID, Repository: intent.Repository, ActorID: intent.ActorID,
	}, factory.OpRefPublish, outcome)
	if _, err := fx.db.WithdrawDispatch(ctx, p.Repository, "pause", "operator"); err != nil {
		t.Fatal(err)
	}
	withdrawal := fx.coord.cancelRepositoryPublications(ctx, p.Repository)
	if !withdrawal.Pending {
		t.Fatalf("withdrawal claimed completion while correction remained incomplete: %+v", withdrawal)
	}
	if countOperation(exec.cancels, intent.OperationID) != 1 {
		t.Fatalf("correction cancellation was not observed: %+v", exec.cancels)
	}
	stored, err := fx.db.PublicationByAssignment(ctx, p.AssignmentID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Stage != factory.PublicationPublished || stored.Candidate != intent.Candidate ||
		stored.Corrections[0].Effect != factory.OpEffectCommitted || stored.Corrections[0].Completion != factory.OpCompletionPending {
		t.Fatalf("withdrawal lost committed correction custody: %+v", stored)
	}
	recovery, err := fx.db.OpenPublications(ctx, 10)
	if err != nil || len(recovery) != 1 || recovery[0].AssignmentID != p.AssignmentID {
		t.Fatalf("incomplete correction no longer selected for recovery: %+v %v", recovery, err)
	}
}

func seedPublishedNativeOperations(exec *fakePublisher, p factory.Publication) {
	for _, op := range []factory.PublicationOperation{p.Publish, p.PRCreate} {
		work := op.Work.Apply(factory.PublicationWork{OperationID: op.OperationID, Repository: p.Repository})
		outcome := factory.OperationOutcome{
			Receipt: []byte(op.Receipt), Effect: factory.OpEffectCommitted,
			Cancellation: factory.OpCancelNone, Completion: factory.OpCompletionComplete,
		}
		exec.ledger[op.OperationID] = publicationTestOutcome(work, op.Kind, outcome)
	}
}

func countOperation(ids []string, want string) int {
	n := 0
	for _, id := range ids {
		if id == want {
			n++
		}
	}
	return n
}

func (fx *publishFixture) dbMustPublication(t *testing.T, assignmentID string) factory.Publication {
	t.Helper()
	p, err := fx.db.PublicationByAssignment(context.Background(), assignmentID)
	if err != nil {
		t.Fatal(err)
	}
	return p
}
