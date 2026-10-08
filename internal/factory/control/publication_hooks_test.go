package control

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/identity"
	"github.com/levitateos/sodaos/internal/project"
)

type publicationDispatchHost struct {
	*fakeDispatchHost
	exporter  *fakePublishHost
	candidate string
	stops     []project.FactoryStop
}

func (h *publicationDispatchHost) FactoryStop(_ context.Context, in project.FactoryStop) (project.FactoryState, error) {
	h.stops = append(h.stops, in)
	return project.FactoryState{
		ID: in.ID, Project: in.Project, Phase: project.FactoryCompleted, Retirement: "confirmed",
		Output: "```result-json\n" + `{"status":"completed","summary":"candidate ready","candidate":"` + h.candidate + `","review_passed":false,"findings":[]}` + "\n```",
	}, nil
}

func (h *publicationDispatchHost) FactoryExport(ctx context.Context, in project.FactoryExport) (project.FactoryExportState, error) {
	return h.exporter.FactoryExport(ctx, in)
}

func TestPublicationDispatchHandoffAccountsAndPublishesRecordedResult(t *testing.T) {
	ctx := context.Background()
	fx := publishSeed(t)
	decision := fx.seed.accept(t, 3, "d333333333333333333333333")
	fx.seed.queue(t, 3, decision.ID)
	fx.reads.evidence["7/3"] = AcceptanceEvidence{
		Revision: 9,
		Issue:    AcceptanceIssueView{Index: "3", TitleDigest: decision.TitleDigest, ContentDigest: decision.ContentDigest, ContentVer: decision.ContentVersion, Visible: true},
		Comments: []AcceptanceComment{{ID: "11", Digest: decision.Sources[0].Digest, ContentVer: decision.Sources[0].ContentVersion, Visible: true}},
	}
	h := &publicationDispatchHost{fakeDispatchHost: fx.seed.host, exporter: fx.host, candidate: strings.Repeat("2", 40)}
	closed := 0
	fx.coord.Host = h
	fx.coord.Broker = &stubBroker{
		get:   func(string, string) (identity.Execution, error) { return identity.Execution{}, identity.ErrNotFound },
		close: func(string, string) error { closed++; return nil },
	}
	fx.coord.DispatchReads = fx.seed.reads
	exec := happyPublisher()
	fx.wire(exec)

	report := fx.coord.Dispatch(ctx)
	if len(report.Errors) != 0 || len(report.Launched) != 1 {
		t.Fatalf("dispatch: %+v", report)
	}
	a, err := fx.db.Assignment(ctx, report.Launched[0].AssignmentID)
	if err != nil || a.Stage != factory.AssignmentFinished || a.Result == nil || !a.Result.Reported || a.Result.Candidate != h.candidate {
		t.Fatalf("recorded result: %+v %v", a, err)
	}
	run, err := fx.db.FactoryRun(ctx, a.Run)
	if err != nil || !run.Reconciled || run.Outcome != factory.Succeeded || closed != 1 || len(h.stops) != 1 {
		t.Fatalf("settlement: %+v %v close=%d stops=%d", run, err, closed, len(h.stops))
	}
	reservation, err := fx.db.Reservation(ctx, a.ID)
	if err != nil || reservation.State != factory.ReservationConsumed {
		t.Fatalf("accounting: %+v %v", reservation, err)
	}
	p := fx.publication(t, a)
	if p.Stage != factory.PublicationPublished || p.Run != a.Run || p.Candidate != a.Result.Candidate || p.PRNumber != 9 {
		t.Fatalf("automatic publication: %+v", p)
	}
	fx.coord.Dispatch(ctx)
	if len(h.launches) != 1 || len(exec.submits) != 1 || len(exec.creates) != 1 {
		t.Fatal("repeat dispatch duplicated execution or publication")
	}
}

func TestPublicationAcceptanceWithdrawalReceiptKeepsPendingAndExactScope(t *testing.T) {
	ctx := context.Background()
	fx := publishSeed(t)
	a, other := fx.finishReported(t, 3), fx.finishReported(t, 4)
	exec := happyPublisher()
	exec.push = func(factory.PublicationWork) (factory.OperationOutcome, error) { return pendingOutcome(), nil }
	fx.wire(exec)
	fx.pass(t)
	p, otherP := fx.publication(t, a), fx.publication(t, other)
	exec.cancel = func(string) (factory.OperationOutcome, error) {
		return factory.OperationOutcome{}, errors.New("cancel reply lost")
	}
	command := factory.NewID()
	receipt, err := fx.coord.WithdrawAcceptance(ctx, command, "native:7", a.Repository, a.Issue, a.Acceptance, 7)
	if err != nil || !receipt.Withdrawn || !receipt.Publications.Pending || !reflect.DeepEqual(receipt.Publications.Publications, []string{p.ID}) || !reflect.DeepEqual(receipt.Publications.Operations, []string{p.Publish.OperationID}) {
		t.Fatalf("withdrawal receipt: %+v %v", receipt, err)
	}
	current := fx.publication(t, a)
	if current.Stage != factory.PublicationOpen || !current.WithdrawRequested || fx.publication(t, other).WithdrawRequested {
		t.Fatalf("withdrawal scope: %+v other=%+v", current, otherP)
	}
	again, err := fx.coord.WithdrawAcceptance(ctx, command, "native:7", a.Repository, a.Issue, a.Acceptance, 7)
	if err != nil || !reflect.DeepEqual(receipt, again) || len(exec.cancels) != 1 {
		t.Fatalf("withdrawal replay: %+v %v cancellations=%d", again, err, len(exec.cancels))
	}
	exec.cancel = nil
	fx.pass(t)
	if current = fx.publication(t, a); current.Stage != factory.PublicationWithdrawn || current.Publish.Cancellation != factory.OpCancelCancelled {
		t.Fatalf("eventual cancellation: %+v", current)
	}
	if fx.publication(t, other).WithdrawRequested {
		t.Fatal("issue withdrawal cancelled another issue")
	}
}

func TestPublicationReplacementAcceptanceCancelsPredecessorAndReplaysReceipt(t *testing.T) {
	ctx := context.Background()
	fx := publishSeed(t)
	a := fx.finishReported(t, 3)
	exec := happyPublisher()
	exec.push = func(factory.PublicationWork) (factory.OperationOutcome, error) { return pendingOutcome(), nil }
	fx.wire(exec)
	fx.pass(t)
	p := fx.publication(t, a)
	next := fx.seed.decision[3]
	next.ID, next.Predecessor, next.NativeRev = "d444444444444444444444444", next.ID, 9
	command := factory.NewID()
	receipt, err := fx.coord.AdmitAcceptance(ctx, command, "native:7", next)
	if err != nil || receipt.Head != next.ID || receipt.Publications.Pending || !reflect.DeepEqual(receipt.Publications.Operations, []string{p.Publish.OperationID}) {
		t.Fatalf("replacement receipt: %+v %v", receipt, err)
	}
	if current := fx.publication(t, a); current.Stage != factory.PublicationWithdrawn || !current.WithdrawRequested {
		t.Fatalf("predecessor publication remains active: %+v", current)
	}
	again, err := fx.coord.AdmitAcceptance(ctx, command, "native:7", next)
	if err != nil || !reflect.DeepEqual(receipt, again) || len(exec.cancels) != 1 {
		t.Fatalf("replacement replay: %+v %v cancellations=%d", again, err, len(exec.cancels))
	}
}

func TestPublicationGrantChangeReportsNativePendingSeparatelyFromDispatch(t *testing.T) {
	ctx := context.Background()
	fx := publishSeed(t)
	a := fx.finishReported(t, 3)
	exec := happyPublisher()
	exec.push = func(factory.PublicationWork) (factory.OperationOutcome, error) { return pendingOutcome(), nil }
	fx.wire(exec)
	fx.pass(t)
	p := fx.publication(t, a)
	exec.cancel = func(string) (factory.OperationOutcome, error) {
		return factory.OperationOutcome{}, errors.New("cancel unavailable")
	}
	policy, err := fx.db.RepositoryPolicy(ctx, a.Repository)
	if err != nil {
		t.Fatal(err)
	}
	policy.MaxConcurrent++
	command := factory.NewID()
	receipt, err := fx.coord.ApplyPolicy(ctx, command, "native:7", policy.Revision, policy)
	if err != nil || receipt.Withdrawn || !receipt.Effective.Effective || !receipt.PublicationsPending || receipt.MergesPending {
		t.Fatalf("grant receipt: %+v %v", receipt, err)
	}
	open, _, _, err := fx.db.DispatchState(ctx, a.Repository)
	current := fx.publication(t, a)
	if err != nil || !open || !current.WithdrawRequested || current.ID != p.ID || current.Publish.OperationID != p.Publish.OperationID {
		t.Fatalf("grant change mixed dispatch with native cancellation: open=%v err=%v", open, err)
	}
	again, err := fx.coord.ApplyPolicy(ctx, command, "native:7", policy.Revision, policy)
	if err != nil || !reflect.DeepEqual(receipt, again) || len(exec.cancels) != 1 {
		t.Fatalf("grant replay: %+v %v cancellations=%d", again, err, len(exec.cancels))
	}
}

func TestPublicationPauseReceiptPreservesPendingNativeCancellation(t *testing.T) {
	ctx := context.Background()
	fx := publishSeed(t)
	a := fx.finishReported(t, 3)
	exec := happyPublisher()
	exec.push = func(factory.PublicationWork) (factory.OperationOutcome, error) { return pendingOutcome(), nil }
	fx.wire(exec)
	fx.pass(t)
	p := fx.publication(t, a)
	exec.cancel = func(string) (factory.OperationOutcome, error) {
		return factory.OperationOutcome{}, errors.New("cancel unavailable")
	}
	command := factory.NewID()
	receipt, err := fx.coord.PauseRepository(ctx, command, "native:7", a.Repository)
	if err != nil || !receipt.Paused || !receipt.Withdrawal.Publications.Pending || !reflect.DeepEqual(receipt.Withdrawal.Publications.Operations, []string{p.Publish.OperationID}) {
		t.Fatalf("pause receipt: %+v %v", receipt, err)
	}
	open, _, _, err := fx.db.DispatchState(ctx, a.Repository)
	if err != nil || open || fx.publication(t, a).Stage != factory.PublicationOpen {
		t.Fatalf("pause guessed native cancellation: open=%v err=%v", open, err)
	}
	again, err := fx.coord.PauseRepository(ctx, command, "native:7", a.Repository)
	if err != nil || !reflect.DeepEqual(receipt, again) || len(exec.cancels) != 1 {
		t.Fatalf("pause replay: %+v %v cancellations=%d", again, err, len(exec.cancels))
	}
}

func TestPublicationReconcileReceiptPersistsExactLink(t *testing.T) {
	ctx := context.Background()
	fx := publishSeed(t)
	a := fx.finishReported(t, 3)
	exec := happyPublisher()
	fx.wire(exec)
	command := factory.Command{ID: factory.NewID(), Type: factory.CommandReconcile, Principal: "os-uid:0", Digest: factory.CommandDigest(factory.CommandReconcile, "")}
	receipt, err := fx.coord.Reconcile(ctx, command)
	if err != nil || receipt.Publication == nil || len(receipt.Publication.Published) != 1 || receipt.Publication.Published[0].AssignmentID != a.ID || receipt.Publication.Published[0].PRNumber != 9 {
		t.Fatalf("reconcile receipt: %+v %v", receipt, err)
	}
	again, err := fx.coord.Reconcile(ctx, command)
	if err != nil || !reflect.DeepEqual(receipt, again) || len(exec.submits) != 1 || len(exec.creates) != 1 {
		t.Fatalf("reconcile replay: %+v %v", again, err)
	}
}
