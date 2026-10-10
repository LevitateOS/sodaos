package control

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/identity"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
)

type publicationDispatchHost struct {
	*fakeDispatchHost
	exporter  *fakePublishHost
	candidate string
	stops     []project.FactoryStop
}

type cancelingPublicationCheckObserver struct {
	cancel      context.CancelFunc
	caller      context.Context
	ctxErr      error
	ctxDeadline time.Time
}

func (o *cancelingPublicationCheckObserver) ObserveChecks(ctx context.Context, _ factory.CheckTarget, _ int64) (factory.ObservedChecks, error) {
	o.ctxDeadline, _ = ctx.Deadline()
	if o.cancel != nil {
		o.cancel()
	}
	select {
	case <-ctx.Done():
		o.ctxErr = ctx.Err()
	case <-o.caller.Done():
		o.ctxErr = ctx.Err()
	case <-time.After(3 * time.Second):
		o.ctxErr = ctx.Err()
	}
	return factory.ObservedChecks{}, errors.New("caller canceled during check observation")
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

func TestDispatchPublicationPassRetainsCallerCancellation(t *testing.T) {
	// This fixture uses synthetic completed records only to reach the check
	// boundary. It proves context propagation and custody handling, not native
	// provider execution or effects.
	for _, mode := range []string{"cancellation", "deadline"} {
		t.Run(mode, func(t *testing.T) {
			fx := publishSeed(t)
			a := fx.finishReported(t, 3)
			exec := happyPublisher()
			fx.wire(exec)
			fx.pass(t)
			p := fx.publication(t, a)

			var ctx context.Context
			var cancel context.CancelFunc
			if mode == "deadline" {
				ctx, cancel = context.WithDeadline(context.Background(), time.Now().Add(2*time.Second))
			} else {
				ctx, cancel = context.WithCancel(context.Background())
			}
			defer cancel()
			observer := &cancelingPublicationCheckObserver{caller: ctx}
			if mode == "cancellation" {
				observer.cancel = cancel
			}
			fx.coord.Checks = observer
			merger := happyMerger()
			fx.coord.Merges = merger

			started := time.Now()
			fx.coord.Dispatch(ctx)
			if elapsed := time.Since(started); elapsed > 3*time.Second {
				t.Fatalf("dispatch did not return promptly after caller expiry: %v", elapsed)
			}
			if mode == "cancellation" && observer.ctxErr != context.Canceled {
				t.Fatalf("publication child context lost caller cancellation: %v", observer.ctxErr)
			}
			if mode == "deadline" {
				callerDeadline, ok := ctx.Deadline()
				if !ok || observer.ctxDeadline.IsZero() || observer.ctxDeadline.After(callerDeadline) {
					t.Fatalf("publication child deadline %v exceeds caller deadline %v", observer.ctxDeadline, callerDeadline)
				}
				if observer.ctxErr != context.DeadlineExceeded {
					t.Fatalf("publication child did not expire with caller deadline: %v", observer.ctxErr)
				}
			}
			current := fx.publication(t, a)
			if current.Stage != factory.PublicationPublished {
				t.Fatalf("expired assessment changed publication custody: %+v", current)
			}
			if _, err := fx.db.MergeByPublication(context.Background(), p.ID); !errors.Is(err, store.ErrNotFound) {
				t.Fatalf("expired assessment created merge custody: %v", err)
			}
			if len(merger.submits) != 0 {
				t.Fatalf("expired assessment submitted merge work: %+v", merger.submits)
			}
		})
	}
}

func TestPublicationAcceptanceWithdrawalReceiptKeepsPendingAndExactScope(t *testing.T) {
	ctx := context.Background()
	fx := publishSeed(t)
	exec := happyPublisher()
	fx.wire(exec)
	other := fx.finishReported(t, 4)
	fx.pass(t)
	otherP := fx.publication(t, other)
	if otherP.Stage != factory.PublicationPublished || otherP.Publish.Effect != factory.OpEffectCommitted || otherP.PRCreate.Effect != factory.OpEffectCommitted {
		t.Fatalf("other issue did not complete native publication: %+v", otherP)
	}
	// Preserve the known completed native receipts while making this historical
	// publication terminal so the repository can admit a second active attempt.
	otherP.Stage, otherP.Outcome, otherP.Reason, otherP.FinishedUnix = factory.PublicationFenced, factory.NeedsHuman, factory.PublishReasonFenced, time.Now().Unix()
	otherP.Revision++
	if err := fx.db.UpdatePublication(ctx, otherP); err != nil {
		t.Fatalf("terminalize completed historical publication: %v", err)
	}
	a := fx.finishReported(t, 3)
	exec.push = func(factory.PublicationWork) (factory.OperationOutcome, error) { return pendingOutcome(), nil }
	fx.pass(t)
	p := fx.publication(t, a)
	exec.cancel = func(string) (factory.OperationOutcome, error) {
		return factory.OperationOutcome{}, errors.New("cancel reply lost")
	}
	command := factory.NewID()
	receipt, err := fx.coord.WithdrawAcceptance(ctx, command, "native:7", a.Repository, a.Issue, a.Acceptance, 7)
	if err != nil || !receipt.Withdrawn || !receipt.Publications.Pending || !reflect.DeepEqual(receipt.Publications.Publications, []string{p.ID}) || !reflect.DeepEqual(receipt.Publications.Operations, []string{p.Publish.OperationID}) {
		t.Fatalf("withdrawal receipt: %+v %v", receipt, err)
	}
	current := fx.publication(t, a)
	otherCurrent := fx.publication(t, other)
	if current.Stage != factory.PublicationOpen || !current.WithdrawRequested || otherCurrent.WithdrawRequested ||
		otherCurrent.Publish.Effect != factory.OpEffectCommitted || otherCurrent.PRCreate.Effect != factory.OpEffectCommitted ||
		otherCurrent.Publish.Cancellation != factory.OpCancelNone || otherCurrent.PRCreate.Cancellation != factory.OpCancelNone {
		t.Fatalf("withdrawal scope: %+v other=%+v", current, otherCurrent)
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
	otherCurrent = fx.publication(t, other)
	if otherCurrent.WithdrawRequested || otherCurrent.Publish.Cancellation != factory.OpCancelNone || otherCurrent.PRCreate.Cancellation != factory.OpCancelNone {
		t.Fatalf("issue withdrawal changed another issue's completed native effects: %+v", otherCurrent)
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

func TestReconcileSettlesOnlyAndReplaysWithoutAdvancement(t *testing.T) {
	ctx := context.Background()
	fx, _, p, childRun := reviewRunSeed(t, true)
	accountChildResult(t, fx, childRun, "```review-json\n{\"verdict\":\"approve\",\"summary\":\"solid\",\"body\":\"LGTM\",\"findings\":[]}\n```")
	childExec := &fakeReviewer{
		observed: factory.ReviewObservation{NativeRev: 9, ObservedUnix: time.Now().Unix()},
		outcome:  committedOutcome(`{"review_id":5}`),
		adopted:  factory.ReviewOutcome{ReviewID: 5, CommentID: 6, ReviewerID: 6, PRID: p.PRID, PRNumber: p.PRNumber, IssueID: p.PRCreate.IssueID, HeadOID: p.Candidate, BaseOID: p.PRCreate.BaseOID, CommitID: p.Candidate, Event: "APPROVED"},
	}
	childExec.adopted.Operation = childExec.outcome
	fx.coord.Reviews = childExec

	// A queued issue would be dispatchable if operator Reconcile advanced
	// the workflow while settling the recorded reviewer child.
	queued := fx.seed.accept(t, 5, "d555555555555555555555555")
	fx.seed.queue(t, 5, queued.ID)
	fx.coord.DispatchReads = fx.seed.reads
	fx.coord.Checks = happyCheckObserver()
	fx.coord.Merges = happyMerger()
	readsBefore := fx.seed.reads.calls
	publicationExec := fx.exec
	publicationSubmits, publicationCreates, publicationPushes := len(publicationExec.submits), len(publicationExec.creates), len(publicationExec.pushes)

	// Reconcile still retires and records a real outstanding run.
	run := recordRun(t, fx.coord, nil)
	host := &publicationDispatchHost{fakeDispatchHost: fx.seed.host, exporter: fx.host, candidate: strings.Repeat("2", 40)}
	fx.coord.Host = host
	closed := 0
	fx.coord.Broker = &stubBroker{
		get:   func(string, string) (identity.Execution, error) { return identity.Execution{}, identity.ErrNotFound },
		close: func(string, string) error { closed++; return nil },
	}
	command := factory.Command{ID: factory.NewID(), Type: factory.CommandReconcile, Principal: "os-uid:0", Digest: factory.CommandDigest(factory.CommandReconcile, "")}
	receipt, err := fx.coord.Reconcile(ctx, command)
	if err != nil || !reflect.DeepEqual(receipt.Settled, []string{run.ID}) || len(receipt.Fenced) != 0 {
		t.Fatalf("reconcile settlement: %+v %v", receipt, err)
	}
	settled, err := fx.db.FactoryRun(ctx, run.ID)
	if err != nil || !settled.Reconciled || settled.Outcome != factory.Succeeded || closed != 1 {
		t.Fatalf("settled accounting state: %+v %v closes=%d", settled, err, closed)
	}
	checks := fx.coord.Checks.(*fakeCheckObserver)
	merger := fx.coord.Merges.(*fakeMerger)
	if len(host.launches) != 0 || fx.seed.reads.calls != readsBefore || len(childExec.works) != 0 || len(checks.targets) != 0 || len(merger.observes) != 0 || len(merger.submits) != 0 ||
		len(publicationExec.submits) != publicationSubmits || len(publicationExec.creates) != publicationCreates || len(publicationExec.pushes) != publicationPushes {
		t.Fatalf("reconcile advanced work: launches=%d dispatch_reads=%d reviews=%+v checks=%+v merges=%+v publication=(%d,%d,%d)", len(host.launches), fx.seed.reads.calls-readsBefore, childExec.works, checks.targets, merger.submits,
			len(publicationExec.submits), len(publicationExec.creates), len(publicationExec.pushes))
	}
	again, err := fx.coord.Reconcile(ctx, command)
	if err != nil || !reflect.DeepEqual(receipt, again) || closed != 1 || len(host.launches) != 0 || len(childExec.works) != 0 {
		t.Fatalf("reconcile replay: %+v %v closes=%d launches=%d reviews=%d", again, err, closed, len(host.launches), len(childExec.works))
	}
}

func TestReconcileDoesNotPublishSettledCandidate(t *testing.T) {
	ctx := context.Background()
	fx := publishSeed(t)
	candidate := fx.finishReported(t, 3)
	exec := happyPublisher()
	fx.wire(exec)
	command := factory.Command{ID: factory.NewID(), Type: factory.CommandReconcile, Principal: "os-uid:0", Digest: factory.CommandDigest(factory.CommandReconcile, "")}
	receipt, err := fx.coord.Reconcile(ctx, command)
	if err != nil || len(receipt.Settled) != 0 || len(receipt.Fenced) != 0 {
		t.Fatalf("reconcile receipt: %+v %v", receipt, err)
	}
	if _, err := fx.db.PublicationByAssignment(ctx, candidate.ID); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("reconcile published a settled candidate: %v", err)
	}
	if len(exec.submits) != 0 || len(exec.creates) != 0 || len(exec.pushes) != 0 {
		t.Fatalf("reconcile performed native publication: submits=%d creates=%d pushes=%d", len(exec.submits), len(exec.creates), len(exec.pushes))
	}
	again, err := fx.coord.Reconcile(ctx, command)
	if err != nil || !reflect.DeepEqual(receipt, again) || len(exec.submits) != 0 || len(exec.creates) != 0 || len(exec.pushes) != 0 {
		t.Fatalf("reconcile replay advanced publication: %+v %v submits=%d creates=%d pushes=%d", again, err, len(exec.submits), len(exec.creates), len(exec.pushes))
	}
}
