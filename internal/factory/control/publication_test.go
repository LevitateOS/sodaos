package control

import (
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
)

func TestPublishPassLinksOneExactPR(t *testing.T) {
	fx := publishSeed(t)
	a := fx.finishReported(t, 3)
	exec := happyPublisher()
	fx.wire(exec)
	first := fx.pass(t)
	p := fx.publication(t, a)
	if len(first.Published) != 1 || p.Stage != factory.PublicationPublished || p.PRNumber != 9 || p.PRID != 8 || p.Publish.Effect != factory.OpEffectCommitted {
		t.Fatalf("PR linkage: %+v %+v", first, p)
	}
	if p.Publish.OperationID == p.PRCreate.OperationID || p.Publish.Work == nil || p.PRCreate.Work == nil || len(exec.observes) != 2 {
		t.Fatalf("separate observed intents: %+v", p)
	}
	fx.pass(t)
	if len(exec.submits) != 1 || len(exec.pushes) != 1 || len(exec.creates) != 1 {
		t.Fatalf("replayed mutations: %+v", exec)
	}
}

func TestPublishPassRefusesOccupiedTarget(t *testing.T) {
	fx := publishSeed(t)
	a := fx.finishReported(t, 3)
	exec := happyPublisher()
	exec.observe = func(w factory.PublicationWork) (factory.PublicationObservation, error) {
		return factory.PublicationObservation{TargetRef: factory.PublishBranchName(w.AssignmentID), TargetTip: strings.Repeat("f", 40), Comparison: strings.Repeat("1", 40), NativeRev: 9}, nil
	}
	fx.wire(exec)
	fx.pass(t)
	p := fx.publication(t, a)
	if p.Stage != factory.PublicationFailed || p.Reason != factory.PublishReasonTargetOccupied || len(exec.submits) != 0 {
		t.Fatalf("occupied branch adopted: %+v", p)
	}
}

func TestPublishPassRejectsChangedAcceptedSource(t *testing.T) {
	fx := publishSeed(t)
	a := fx.finishReported(t, 3)
	exec := happyPublisher()
	fx.wire(exec)
	evidence := fx.reads.evidence["7/3"]
	evidence.Comments[0].ContentVer++
	fx.reads.evidence["7/3"] = evidence
	fx.pass(t)
	p := fx.publication(t, a)
	if p.Publish.Attempts != 0 || len(exec.submits) != 0 || p.Stage == factory.PublicationPublished {
		t.Fatalf("changed source authorized publication: %+v", p)
	}
}

func TestPublishPassRequiresAcceptanceAndRefsAtSameRevision(t *testing.T) {
	fx := publishSeed(t)
	a := fx.finishReported(t, 3)
	exec := happyPublisher()
	fx.wire(exec)
	evidence := fx.reads.evidence["7/3"]
	evidence.Revision++
	fx.reads.evidence["7/3"] = evidence
	fx.pass(t)
	if p := fx.publication(t, a); p.Publish.Attempts != 0 || len(exec.submits) != 0 {
		t.Fatalf("cross-revision snapshot submitted: %+v", p)
	}
}
