package control

import (
	"context"
	"errors"
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/host"
)

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
