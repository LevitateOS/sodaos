package control

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/host"
	"github.com/levitateos/sodaos/internal/project"
)

type publicationRecoveryHost struct {
	*producerFlowHost
	output string
}

func (h *publicationRecoveryHost) FactoryInspect(_ context.Context, in project.FactoryInspect) (project.FactoryState, error) {
	return project.FactoryState{ID: in.ID, Project: in.Project, Phase: project.FactoryCompleted, Output: h.output}, nil
}

func TestPublishPassWithdrawalFailureStaysOpen(t *testing.T) {
	fx := publishSeed(t)
	a := fx.finishReported(t, 3)
	exec := happyPublisher()
	exec.push = func(factory.PublicationWork) (factory.OperationOutcome, error) { return pendingOutcome(), nil }
	fx.wire(exec)
	fx.pass(t)
	if _, err := fx.db.WithdrawDispatch(context.Background(), a.Repository, "pause", "operator"); err != nil {
		t.Fatal(err)
	}
	exec.cancel = func(string) (factory.OperationOutcome, error) {
		return factory.OperationOutcome{}, errors.New("cancel reply lost")
	}
	fx.coord.PublishPass(context.Background())
	p := fx.publication(t, a)
	if p.Stage != factory.PublicationOpen || !p.WithdrawRequested || p.Publish.Effect != factory.OpEffectPending {
		t.Fatalf("unknown cancellation marked finished: %+v", p)
	}
	exec.cancel = nil
	fx.pass(t)
	p = fx.publication(t, a)
	if p.Stage != factory.PublicationWithdrawn || p.Publish.Cancellation != factory.OpCancelCancelled {
		t.Fatalf("confirmed cancellation not retained: %+v", p)
	}
	if len(exec.pushes) != 1 || len(exec.creates) != 0 {
		t.Fatal("withdrawal performed new mutation")
	}
}

func TestPublishPassLostPRReplyRetainsLateCommittedLinkAndCompletion(t *testing.T) {
	fx := publishSeed(t)
	a := fx.finishReported(t, 3)
	exec := happyPublisher()
	exec.submitPR = func(w factory.PublicationWork) (factory.OperationOutcome, error) {
		outcome := committedOutcome(`{"pr_number":9}`)
		outcome.Completion = factory.OpCompletionPending
		exec.ledger[w.OperationID] = publicationTestOutcome(w, factory.OpPRCreate, outcome)
		return factory.OperationOutcome{}, errors.New("reply lost after PR commit")
	}
	fx.wire(exec)
	first := fx.coord.PublishPass(context.Background())
	if len(first.Errors) != 1 {
		t.Fatalf("lost PR reply: %+v", first)
	}
	if _, err := fx.db.WithdrawDispatch(context.Background(), a.Repository, "pause", "operator"); err != nil {
		t.Fatal(err)
	}
	fx.pass(t)
	p := fx.publication(t, a)
	if p.PRNumber != 9 || p.PRID != 8 || p.PRCreate.Effect != factory.OpEffectCommitted || p.PRCreate.Completion != factory.OpCompletionPending || p.PRCreate.Cancellation != factory.OpCancelTooLate {
		t.Fatalf("committed late PR lost: %+v", p)
	}
	if p.Stage == factory.PublicationPublished || p.Stage == factory.PublicationWithdrawn {
		t.Fatalf("pending completion marked finished: %+v", p)
	}
	outcome := exec.ledger[p.PRCreate.OperationID]
	outcome.Completion = factory.OpCompletionComplete
	exec.ledger[p.PRCreate.OperationID] = outcome
	fx.pass(t)
	p = fx.publication(t, a)
	if p.PRNumber != 9 || p.PRCreate.Completion != factory.OpCompletionComplete || len(exec.creates) != 1 {
		t.Fatalf("late completion duplicated/hidden: %+v", p)
	}
}

func TestPublishPassRecoversCommittedPRWithoutExport(t *testing.T) {
	fx := publishSeed(t)
	a := fx.finishReported(t, 3)
	exec := happyPublisher()
	exec.submitPR = func(w factory.PublicationWork) (factory.OperationOutcome, error) {
		exec.ledger[w.OperationID] = publicationTestOutcome(w, factory.OpPRCreate, committedOutcome(`{"pr_number":9}`))
		return factory.OperationOutcome{}, errors.New("reply lost")
	}
	fx.wire(exec)
	first := fx.coord.PublishPass(context.Background())
	if len(first.Errors) != 1 {
		t.Fatalf("lost PR reply: %+v", first)
	}
	fx.host.exportErr = host.ErrRunNotFound
	fx.pass(t)
	if p := fx.publication(t, a); p.Stage != factory.PublicationPublished || p.PRNumber != 9 || len(exec.creates) != 1 {
		t.Fatalf("receipt recovery depended on export: %+v", p)
	}
}

func TestPublishPassFailsUnrecoverableExport(t *testing.T) {
	fx := publishSeed(t)
	a := fx.finishReported(t, 3)
	fx.host.exportErr = host.ErrRunNotFound
	exec := happyPublisher()
	fx.wire(exec)
	fx.pass(t)
	if p := fx.publication(t, a); p.Stage != factory.PublicationFailed || p.Reason != factory.PublishReasonExportFailed || len(exec.observes) != 0 {
		t.Fatalf("missing bundle advanced: %+v", p)
	}
}

func TestPublishPassWaitsWithoutExecutor(t *testing.T) {
	fx := publishSeed(t)
	fx.finishReported(t, 3)
	if report := fx.coord.PublishPass(context.Background()); !report.Unavailable {
		t.Fatalf("report: %+v", report)
	}
}

func TestPublishPassRecoversFinishedReviewerAndCorrectionChildren(t *testing.T) {
	t.Run("review child", func(t *testing.T) {
		fx, _, publication, run := reviewRunSeed(t, true)
		output := "```review-json\n{\"verdict\":\"approve\",\"summary\":\"solid\",\"body\":\"LGTM\",\"findings\":[]}\n```"
		assignment := accountChildResult(t, fx, run, output)
		reviewer := &fakeReviewer{
			observed: factory.ReviewObservation{NativeRev: 9, ObservedUnix: time.Now().Unix()},
			outcome:  committedOutcome(`{"review_id":3}`),
			adopted: factory.ReviewOutcome{
				ReviewID: 3, CommentID: 4, ReviewerID: 6, PRID: publication.PRID,
				PRNumber: publication.PRNumber, IssueID: publication.PRCreate.IssueID,
				HeadOID: publication.Candidate, BaseOID: publication.PRCreate.BaseOID,
				CommitID: publication.Candidate, Event: "APPROVED",
			},
		}
		fx.coord.Reviews = reviewer
		first := fx.coord.PublishPass(context.Background())
		if len(first.Errors) != 0 || len(reviewer.submitted) != 1 || reviewer.submitted[0].OperationID != "review-"+run.ID {
			t.Fatalf("finished reviewer result was not submitted from storage: report=%+v calls=%+v", first, reviewer.submitted)
		}
		second := fx.coord.PublishPass(context.Background())
		stored, err := fx.db.PublicationByAssignment(context.Background(), assignment.PublicationAssignment)
		if len(second.Errors) != 0 || err != nil || len(stored.ReviewOperations) != 1 || len(reviewer.submitted) != 1 {
			t.Fatalf("review child replay repeated its effect: report=%+v pub=%+v calls=%+v err=%v", second, stored, reviewer.submitted, err)
		}
	})

	t.Run("correction child", func(t *testing.T) {
		fx := publishSeed(t)
		owner := fx.finishReported(t, 17)
		fx.wire(happyPublisher())
		fx.pass(t)
		run := correctionRun(t, fx, owner, strings.Repeat("d", 40))
		first := fx.coord.PublishPass(context.Background())
		if len(first.Errors) != 0 {
			t.Fatalf("finished correction result was not published: %+v", first)
		}
		publishing := fx.exec
		if len(publishing.submits) != 2 || len(publishing.pushes) != 2 {
			t.Fatalf("correction effect counts = submits %d pushes %d", len(publishing.submits), len(publishing.pushes))
		}
		second := fx.coord.PublishPass(context.Background())
		stored, err := fx.db.PublicationByAssignment(context.Background(), owner.ID)
		if len(second.Errors) != 0 || err != nil || stored.Run != run.ID || len(stored.Corrections) != 1 || len(publishing.submits) != 2 || len(publishing.pushes) != 2 {
			t.Fatalf("correction child replay repeated its effect: report=%+v pub=%+v submits=%d pushes=%d err=%v", second, stored, len(publishing.submits), len(publishing.pushes), err)
		}
	})

	t.Run("lost correction reply", func(t *testing.T) {
		fx := publishSeed(t)
		owner := fx.finishReported(t, 18)
		fx.wire(happyPublisher())
		fx.pass(t)
		run := correctionRun(t, fx, owner, strings.Repeat("d", 40))
		exec := fx.exec
		submitsBefore := len(exec.submits)
		exec.submit = func(work factory.PublicationWork) (factory.OperationOutcome, error) {
			outcome := committedOutcome(`{"ref":"` + factory.PublishBranchName(work.AssignmentID) + `"}`)
			exec.ledger[work.OperationID] = publicationTestOutcome(work, factory.OpRefPublish, outcome)
			return factory.OperationOutcome{}, errors.New("reply lost after correction commit")
		}
		first := fx.coord.PublishPass(context.Background())
		if len(first.Errors) != 1 {
			t.Fatalf("lost correction reply was not retained as uncertain: %+v", first)
		}
		before, err := fx.db.PublicationByAssignment(context.Background(), owner.ID)
		if err != nil || len(before.Corrections) != 1 || before.Corrections[0].RunID != run.ID || before.Run == run.ID {
			t.Fatalf("correction intent lost its child run before lookup: %+v %v", before, err)
		}
		second := fx.coord.PublishPass(context.Background())
		after, err := fx.db.PublicationByAssignment(context.Background(), owner.ID)
		if len(second.Errors) != 0 || err != nil || after.Run != run.ID || after.Candidate != strings.Repeat("d", 40) ||
			len(after.Corrections) != 1 || len(exec.submits) != submitsBefore+1 || countOperation(exec.lookups, after.Corrections[0].OperationID) != 1 {
			t.Fatalf("correction recovery lost run attribution or resubmitted: report=%+v pub=%+v submits=%d lookups=%d err=%v", second, after, len(exec.submits), len(exec.lookups), err)
		}
	})
}

func TestRecoverDispatchDerivesReviewerResultBeforeChildSubmission(t *testing.T) {
	fx, producer, owner, publication := publishedChildProducerSeed(t, 18)
	run, err := fx.db.FactoryRun(context.Background(), producer.launches[0].Run.ID)
	if err != nil {
		t.Fatal(err)
	}
	output := "```review-json\n{\"verdict\":\"request-changes\",\"summary\":\"fix\",\"body\":\"empty input panics\",\"findings\":[\"empty input\"]}\n```"
	run.Outcome, run.Summary, run.Reconciled = factory.Succeeded, "review settled", true
	if err := fx.db.SaveFactoryRun(context.Background(), run); err != nil {
		t.Fatal(err)
	}
	fx.coord.Host = &publicationRecoveryHost{producerFlowHost: producer, output: output}
	recovered := RecoverDispatch(context.Background(), fx.coord.dispatchDeps())
	if len(recovered.Errors) != 0 || len(recovered.Recovered) != 1 {
		t.Fatalf("reviewer assignment did not recover: %+v", recovered)
	}
	view, err := fx.db.FactoryRunView(context.Background(), run.ID)
	if err != nil {
		t.Fatal(err)
	}
	assignment, err := fx.db.Assignment(context.Background(), view.Attempt)
	if err != nil || assignment.Stage != factory.AssignmentFinished || assignment.Result == nil || !assignment.Result.Reported || assignment.Result.Review == nil {
		t.Fatalf("canonical recovered reviewer result missing: %+v %v", assignment, err)
	}
	reviewer := &fakeReviewer{
		observed: factory.ReviewObservation{NativeRev: 9, ObservedUnix: time.Now().Unix()},
		outcome:  committedOutcome(`{"review_id":3}`),
		adopted: factory.ReviewOutcome{
			ReviewID: 3, CommentID: 4, ReviewerID: 6, PRID: publication.PRID,
			PRNumber: publication.PRNumber, IssueID: publication.PRCreate.IssueID,
			HeadOID: publication.Candidate, BaseOID: publication.PRCreate.BaseOID,
			CommitID: publication.Candidate, Event: "REQUEST_CHANGES",
		},
	}
	fx.coord.Reviews = reviewer
	report := fx.coord.PublishPass(context.Background())
	stored, err := fx.db.PublicationByAssignment(context.Background(), owner.ID)
	if len(report.Waits) != 0 || len(report.Errors) != 0 || len(reviewer.submitted) != 1 || err != nil || len(stored.ReviewOperations) != 1 {
		t.Fatalf("recovered reviewer result not consumed: %+v", report)
	}
}

func TestPendingPublicationChildrenPageCurrentHead(t *testing.T) {
	ctx := context.Background()
	fx, owner, _, reviewRun := reviewRunSeed(t, true)
	reviewer := accountChildResult(t, fx, reviewRun, "```review-json\n{\"verdict\":\"approve\",\"summary\":\"solid\",\"body\":\"LGTM\",\"findings\":[]}\n```")
	correction := correctionRun(t, fx, owner, strings.Repeat("d", 40))
	first, err := fx.db.PendingPublicationChildren(ctx, "", 1)
	if err != nil || len(first) != 1 || first[0].ID != reviewer.ID {
		t.Fatalf("first child page = %+v, %v", first, err)
	}
	second, err := fx.db.PendingPublicationChildren(ctx, first[0].ID, 1)
	if err != nil || len(second) != 1 || second[0].Run != correction.ID {
		t.Fatalf("waiting reviewer displaced later coder result: %+v, %v", second, err)
	}
	tail, err := fx.db.PendingPublicationChildren(ctx, second[0].ID, 1)
	if err != nil || len(tail) != 0 {
		t.Fatalf("child cursor did not reach the tail: %+v, %v", tail, err)
	}
	changed := fx.coord.PublishCorrection(ctx, correction.ID)
	if len(changed.Errors) != 0 || len(changed.Corrected) != 1 {
		t.Fatalf("current child correction failed: %+v", changed)
	}
	pending, err := fx.db.PendingPublicationChildren(ctx, "", 2)
	if err != nil || len(pending) != 0 {
		t.Fatalf("superseded child head remained eligible: %+v, %v", pending, err)
	}
}
