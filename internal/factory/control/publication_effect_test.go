package control

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
)

func TestPublishPassReconcilesLostSubmitReply(t *testing.T) {
	fx := publishSeed(t)
	a := fx.finishReported(t, 3)
	exec := happyPublisher()
	exec.submit = func(w factory.PublicationWork) (factory.OperationOutcome, error) {
		exec.ledger[w.OperationID] = publicationTestOutcome(w, factory.OpRefPublish, pendingOutcome())
		return factory.OperationOutcome{}, errors.New("reply lost after registration")
	}
	fx.wire(exec)
	first := fx.coord.PublishPass(context.Background())
	if len(first.Errors) != 1 {
		t.Fatalf("lost reply: %+v", first)
	}
	if p := fx.publication(t, a); p.Publish.Effect != "" || p.Publish.Work == nil {
		t.Fatalf("lost intent: %+v", p.Publish)
	}
	fx.pass(t)
	fx.pass(t)
	if len(exec.submits) != 1 || len(exec.pushes) != 1 || fx.publication(t, a).Stage != factory.PublicationPublished {
		t.Fatal("lost submit duplicated or failed to reconcile")
	}
}

func TestPublishPassReplaysUnobservedSubmitWithByteIdenticalIntent(t *testing.T) {
	fx := publishSeed(t)
	a := fx.finishReported(t, 3)
	exec := happyPublisher()
	original := exec.submit
	attempt := 0
	exec.submit = func(w factory.PublicationWork) (factory.OperationOutcome, error) {
		attempt++
		if attempt == 1 {
			return factory.OperationOutcome{}, errors.New("request not received")
		}
		return original(w)
	}
	fx.wire(exec)
	first := fx.coord.PublishPass(context.Background())
	if len(first.Errors) != 1 {
		t.Fatalf("lost reply: %+v", first)
	}
	before, _ := json.Marshal(exec.submits[0].Intent())
	// An unrelated native change cannot rewrite a previously recorded intent.
	evidence := fx.reads.evidence["7/3"]
	evidence.Revision = 10
	fx.reads.evidence["7/3"] = evidence
	fx.pass(t)
	if len(exec.submits) != 2 {
		t.Fatalf("same intent not replayed: %d", len(exec.submits))
	}
	after, _ := json.Marshal(exec.submits[1].Intent())
	if string(before) != string(after) || fx.publication(t, a).Publish.Attempts != 1 {
		t.Fatalf("replay changed intent: %s -> %s", before, after)
	}
	branchObservations := 0
	for _, w := range exec.observes {
		if w.ExpectedOld == "absent" {
			branchObservations++
		}
	}
	if branchObservations != 1 {
		t.Fatal("immutable branch replay was reobserved")
	}
}

func TestPublishPassKeepsBranchCommittedPRFailedPartial(t *testing.T) {
	fx := publishSeed(t)
	a := fx.finishReported(t, 3)
	exec := happyPublisher()
	exec.submitPR = func(factory.PublicationWork) (factory.OperationOutcome, error) {
		return refusedOutcome("duplicate_pull_request"), nil
	}
	fx.wire(exec)
	fx.pass(t)
	fx.pass(t)
	fx.pass(t)
	p := fx.publication(t, a)
	if p.Stage != factory.PublicationFailed || p.Publish.Effect != factory.OpEffectCommitted || p.PRCreate.Effect != factory.OpEffectNotCommitted || p.PRCreate.Reason != "duplicate_pull_request" {
		t.Fatalf("partial outcome lost: %+v", p)
	}
	if len(exec.creates) != 1 || p.PRCreate.Attempts != 1 {
		t.Fatal("terminal refusal retried")
	}
}

func TestPublishPassDoesNotRetryTerminalBranchRefusal(t *testing.T) {
	fx := publishSeed(t)
	a := fx.finishReported(t, 3)
	exec := happyPublisher()
	exec.submit = func(factory.PublicationWork) (factory.OperationOutcome, error) {
		return refusedOutcome("stale_head"), nil
	}
	fx.wire(exec)
	fx.pass(t)
	fx.pass(t)
	p := fx.publication(t, a)
	if p.Stage != factory.PublicationFailed || p.Publish.Reason != "stale_head" || len(exec.submits) != 1 || len(exec.pushes) != 0 {
		t.Fatalf("terminal operation replaced: %+v", p)
	}
}

func TestPublishPassPendingPushIsBoundedPerPass(t *testing.T) {
	fx := publishSeed(t)
	fx.finishReported(t, 3)
	exec := happyPublisher()
	exec.push = func(factory.PublicationWork) (factory.OperationOutcome, error) { return pendingOutcome(), nil }
	fx.wire(exec)
	fx.pass(t)
	if len(exec.pushes) != 1 || len(exec.creates) != 0 {
		t.Fatal("pending push recursively repeated")
	}
	fx.pass(t)
	if len(exec.pushes) != 2 || len(exec.submits) != 1 {
		t.Fatal("pending push retry was unbounded or replaced identity")
	}
}

func TestPublishPassFencesIndeterminateEffect(t *testing.T) {
	fx := publishSeed(t)
	a := fx.finishReported(t, 3)
	exec := happyPublisher()
	exec.submit = func(factory.PublicationWork) (factory.OperationOutcome, error) {
		return factory.OperationOutcome{Effect: factory.OpEffectIndeterminate, Reason: "comparison_moved"}, nil
	}
	fx.wire(exec)
	fx.pass(t)
	fx.pass(t)
	if p := fx.publication(t, a); p.Stage != factory.PublicationFenced || p.Outcome != factory.NeedsHuman {
		t.Fatalf("uncertain effect not fenced: %+v", p)
	}
	if len(exec.submits) != 1 || len(exec.pushes) != 0 || len(exec.lookups) == 0 {
		t.Fatal("uncertain operation resubmitted")
	}
}
