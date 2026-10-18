package control

import (
	"context"
	"encoding/json"
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
	db        *store.Store
	candidate string
	stops     []project.FactoryStop
}

// phasedDispatchOutput renders the strict v2 report the settling phase
// requires: an executable plan bound to the investigating assignment's
// exact root, source and preparation, or the implemented candidate
// bound to the implementation child. Planning never names a
// publishable candidate; only the implement report carries one.
func phasedDispatchOutput(a factory.Assignment, candidate string) (string, error) {
	var result factory.PhaseResult
	switch a.Phase {
	case project.ExecutionInvestigate:
		plan := factory.InvestigationPlan{
			ID: factory.NewID(), AttemptRoot: a.AttemptRoot,
			ScopeDigest: dispatchDigest("dispatch handoff scope"), InputDigest: dispatchDigest("dispatch handoff input"),
			SourceCommit: a.SourceCommit, IntegrationCommit: a.SourceCommit,
			Preparation:        a.Preparation,
			Approach:           "implement the accepted objective behind checks",
			AcceptanceCoverage: []string{"accept-one"},
			Tasks: []factory.PlanTask{{
				ID: "task-one", Objective: "implement the accepted objective",
				Outputs: []string{"candidate"}, AcceptanceRefs: []string{"accept-one"},
				Lane: "lane-one", OwnedPaths: []string{"internal/x.go"},
			}},
			Lanes: []factory.PlanLane{{ID: "lane-one", TaskIDs: []string{"task-one"}}},
		}
		digest, err := factory.PlanDigest(plan)
		if err != nil {
			return "", err
		}
		plan.Digest = digest
		if err := plan.Validate(); err != nil {
			return "", err
		}
		if err := plan.Executable(); err != nil {
			return "", err
		}
		result = factory.PhaseResult{
			Version: factory.PhaseResultVersion, Phase: project.ExecutionInvestigate, Status: "completed",
			Summary: "plan ready", Output: &factory.PhaseOutput{Plan: &plan},
		}
	case project.ExecutionImplement:
		if !factory.ValidCommit(candidate) {
			return "", errors.New("implement report needs its candidate")
		}
		result = factory.PhaseResult{
			Version: factory.PhaseResultVersion, Phase: project.ExecutionImplement, Status: "completed",
			Summary: "implemented", Output: &factory.PhaseOutput{Implementation: &factory.ImplementationReceipt{Candidate: candidate}},
		}
	default:
		return "", errors.New("dispatch host covers only investigate and implement phases")
	}
	data, err := json.Marshal(result)
	if err != nil {
		return "", err
	}
	return "work log\n```" + factory.ResultFence + "\n" + string(data) + "\n```\ntrailer\n", nil
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

func (h *publicationDispatchHost) FactoryStop(ctx context.Context, in project.FactoryStop) (project.FactoryState, error) {
	h.stops = append(h.stops, in)
	assignment, err := h.db.AssignmentByRun(ctx, in.ID)
	if err != nil {
		return project.FactoryState{}, err
	}
	output, err := phasedDispatchOutput(assignment, h.candidate)
	if err != nil {
		return project.FactoryState{}, err
	}
	return project.FactoryState{
		ID: in.ID, Project: in.Project, Phase: project.FactoryCompleted, Retirement: "confirmed",
		Output: output,
	}, nil
}

func (h *publicationDispatchHost) FactoryExport(ctx context.Context, in project.FactoryExport) (project.FactoryExportState, error) {
	return h.exporter.FactoryExport(ctx, in)
}

// standaloneStopHost confirms the stop of one recorded standalone run
// without manufacturing phased output: production settles the confirmed
// retirement and ignores the missing-assignment accounting result. All
// other stops delegate to the publication host, preserving the
// unconfirmed-stop behavior for runs the host cannot observe.
type standaloneStopHost struct {
	*publicationDispatchHost
	runID string
}

func (h *standaloneStopHost) FactoryStop(ctx context.Context, in project.FactoryStop) (project.FactoryState, error) {
	if in.ID == h.runID {
		return project.FactoryState{ID: in.ID, Project: in.Project, Phase: project.FactoryCompleted, Retirement: "confirmed"}, nil
	}
	return h.publicationDispatchHost.FactoryStop(ctx, in)
}

func TestPublicationDispatchHandoffAccountsAndPublishesRecordedResult(t *testing.T) {
	ctx := context.Background()
	fx := publishSeed(t)
	decision := fx.seed.accept(t, 3, "d333333333333333333333333")
	fx.seed.queue(t, 3, decision.ID)
	fx.reads.evidence["7/3"] = AcceptanceEvidence{
		Revision: 41,
		Issue:    AcceptanceIssueView{Index: "3", TitleDigest: decision.TitleDigest, TitleOccurrence: decision.TitleOccurrence, ContentDigest: decision.ContentDigest, ContentVer: decision.ContentVersion, Visible: true},
		Comments: []AcceptanceComment{{ID: "11", Digest: decision.Sources[0].Digest, ContentVer: decision.Sources[0].ContentVersion, Visible: true}},
	}
	h := &publicationDispatchHost{fakeDispatchHost: fx.seed.host, exporter: fx.host, db: fx.db, candidate: strings.Repeat("2", 40)}
	closed := 0
	fx.coord.Host = h
	fx.coord.Broker = &stubBroker{
		get:   func(string, string) (identity.Execution, error) { return identity.Execution{}, identity.ErrNotFound },
		close: func(string, string) error { closed++; return nil },
	}
	fx.coord.DispatchReads = fx.seed.reads
	exec := happyPublisher()
	// This fixture's native revision is the seed's 41 on every fake;
	// the publisher observes the same native state as acceptance.
	nativeObserve := exec.observe
	exec.observe = func(w factory.PublicationWork) (factory.PublicationObservation, error) {
		observation, err := nativeObserve(w)
		if err != nil {
			return observation, err
		}
		observation.NativeRev = 41
		return observation, nil
	}
	fx.wire(exec)

	report := fx.coord.Dispatch(ctx)
	if len(report.Errors) != 0 || len(report.Launched) != 1 {
		t.Fatalf("dispatch: %+v", report)
	}
	a, err := fx.db.Assignment(ctx, report.Launched[0].AssignmentID)
	if err != nil || a.Stage != factory.AssignmentFinished || a.Result == nil || !a.Result.Reported || a.Result.Candidate != "" {
		t.Fatalf("recorded investigate result: %+v %v", a, err)
	}
	// The reported plan advances the implement child through the same
	// dispatch; only the implement report carries the candidate.
	advanced := fx.coord.Dispatch(ctx)
	if len(advanced.Errors) != 0 || len(advanced.Launched) != 1 {
		t.Fatalf("implement dispatch: %+v", advanced)
	}
	child, err := fx.db.Assignment(ctx, advanced.Launched[0].AssignmentID)
	if err != nil || child.Stage != factory.AssignmentFinished || child.Result == nil || !child.Result.Reported || child.Result.Candidate != h.candidate {
		t.Fatalf("recorded implement result: %+v %v", child, err)
	}
	run, err := fx.db.FactoryRun(ctx, child.Run)
	if err != nil || !run.Reconciled || run.Outcome != factory.Succeeded || closed != 2 || len(h.stops) != 2 {
		t.Fatalf("settlement: %+v %v close=%d stops=%d", run, err, closed, len(h.stops))
	}
	reservation, err := fx.db.Reservation(ctx, child.ID)
	if err != nil || reservation.State != factory.ReservationConsumed {
		t.Fatalf("accounting: %+v %v", reservation, err)
	}
	// The FW15 plan-task evidence walks pending to completed under the
	// admitted head, binding the reported candidate as the task commit.
	headID, _, err := fx.db.PlanHead(ctx, a.AttemptRoot)
	if err != nil {
		t.Fatalf("plan head: %v", err)
	}
	pending := store.PlanTaskReceipt{AttemptRoot: a.AttemptRoot, PlanID: headID, TaskID: "task-one", State: store.PlanTaskPending}
	if err := fx.db.SavePlanTaskReceipt(ctx, pending); err != nil {
		t.Fatalf("pending receipt unrecorded: %v", err)
	}
	running := pending
	running.Revision, running.State, running.Child = 0, store.PlanTaskRunning, "child-task-one"
	if err := fx.db.SavePlanTaskReceipt(ctx, running); err != nil {
		t.Fatalf("running receipt unrecorded: %v", err)
	}
	ready := running
	ready.Revision, ready.State, ready.OutputDigest = 1, store.PlanTaskOutputReady, dispatchDigest("task output")
	if err := fx.db.SavePlanTaskReceipt(ctx, ready); err != nil {
		t.Fatalf("ready receipt unrecorded: %v", err)
	}
	completed := ready
	completed.Revision, completed.State, completed.Commit, completed.Tree = 2, store.PlanTaskCompleted, h.candidate, h.candidate
	completed.Reason = "planned task finished with its checkpoint commit"
	if err := fx.db.SavePlanTaskReceipt(ctx, completed); err != nil {
		t.Fatalf("completed receipt unrecorded: %v", err)
	}
	published := fx.coord.Dispatch(ctx)
	if len(published.Errors) != 0 {
		t.Fatalf("publication dispatch: %+v", published)
	}
	// The recording pass opens the publication; the next pass drives
	// its native publish/PR effects to Published through the same path.
	driven := fx.coord.Dispatch(ctx)
	if len(driven.Errors) != 0 {
		t.Fatalf("publication drive: %+v", driven)
	}
	p := fx.publication(t, a)
	if p.Stage != factory.PublicationPublished || p.Run != child.Run || p.Candidate != h.candidate || p.PRNumber != 9 {
		t.Fatalf("automatic publication: %+v", p)
	}
	fx.coord.Dispatch(ctx)
	if len(h.launches) != 2 || len(exec.submits) != 1 || len(exec.creates) != 1 {
		t.Fatal("repeat dispatch duplicated execution or publication")
	}
}

// TestInvestigateJourneySelectsOneIssueWithoutDoubleSpend proves the R15
// selected investigation-to-plan journey at the dispatch layer: with two
// queued issues a single pass investigates exactly one, the unselected
// issue receives no generative run and no funding, the settled plan
// advances to its implement child instead of relaunching investigation,
// and a further replay launches nothing new.
func TestInvestigateJourneySelectsOneIssueWithoutDoubleSpend(t *testing.T) {
	ctx := context.Background()
	fx := publishSeed(t)
	decision := fx.seed.accept(t, 3, "d333333333333333333333333")
	fx.seed.queue(t, 3, decision.ID)
	fourth := fx.seed.accept(t, 4, "d444444444444444444444444")
	fx.seed.queue(t, 4, fourth.ID)
	fx.reads.evidence["7/3"] = AcceptanceEvidence{
		Revision: 41,
		Issue:    AcceptanceIssueView{Index: "3", TitleDigest: decision.TitleDigest, TitleOccurrence: decision.TitleOccurrence, ContentDigest: decision.ContentDigest, ContentVer: decision.ContentVersion, Visible: true},
		Comments: []AcceptanceComment{{ID: "11", Digest: decision.Sources[0].Digest, ContentVer: decision.Sources[0].ContentVersion, Visible: true}},
	}
	fx.reads.evidence["7/4"] = AcceptanceEvidence{
		Revision: 41,
		Issue:    AcceptanceIssueView{Index: "4", TitleDigest: fourth.TitleDigest, TitleOccurrence: fourth.TitleOccurrence, ContentDigest: fourth.ContentDigest, ContentVer: fourth.ContentVersion, Visible: true},
		Comments: []AcceptanceComment{{ID: "11", Digest: fourth.Sources[0].Digest, ContentVer: fourth.Sources[0].ContentVersion, Visible: true}},
	}
	h := &publicationDispatchHost{fakeDispatchHost: fx.seed.host, exporter: fx.host, db: fx.db, candidate: strings.Repeat("2", 40)}
	fx.coord.Host = h
	fx.coord.Broker = &stubBroker{
		get:   func(string, string) (identity.Execution, error) { return identity.Execution{}, identity.ErrNotFound },
		close: func(string, string) error { return nil },
	}
	fx.coord.DispatchReads = fx.seed.reads
	fx.wire(happyPublisher())

	first := fx.coord.Dispatch(ctx)
	if len(first.Errors) != 0 || len(first.Launched) != 1 {
		t.Fatalf("select dispatch: %+v", first)
	}
	root, err := fx.db.Assignment(ctx, first.Launched[0].AssignmentID)
	if err != nil || root.Issue != 3 || root.Phase != project.ExecutionInvestigate || root.Stage != factory.AssignmentFinished {
		t.Fatalf("selected investigate root: %+v %v", root, err)
	}
	if _, err := fx.db.LatestIssueAssignment(ctx, 7, 4); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("unselected issue received a run: %v", err)
	}

	second := fx.coord.Dispatch(ctx)
	if len(second.Errors) != 0 || len(second.Launched) != 1 {
		t.Fatalf("advance dispatch: %+v", second)
	}
	child, err := fx.db.Assignment(ctx, second.Launched[0].AssignmentID)
	if err != nil || child.Phase != project.ExecutionImplement || child.ParentAssignment != root.ID || child.Stage != factory.AssignmentFinished {
		t.Fatalf("advanced implement child: %+v %v", child, err)
	}
	if _, err := fx.db.LatestIssueAssignment(ctx, 7, 4); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("unselected issue received a run during advance: %v", err)
	}

	third := fx.coord.Dispatch(ctx)
	if len(third.Errors) != 0 || len(third.Launched) != 0 {
		t.Fatalf("replay dispatch relaunched: %+v", third)
	}
	if len(h.launches) != 2 {
		t.Fatalf("launches = %d, want exactly the investigate and implement runs", len(h.launches))
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

	// Reconcile still retires and records a real outstanding run. The
	// standalone run carries no assignment: production settles the
	// confirmed retirement and ignores the missing-assignment
	// accounting result, so the stopping host confirms this exact run
	// without manufacturing phased output.
	run := recordRun(t, fx.coord, nil)
	host := &publicationDispatchHost{fakeDispatchHost: fx.seed.host, exporter: fx.host, db: fx.db, candidate: strings.Repeat("2", 40)}
	fx.coord.Host = &standaloneStopHost{publicationDispatchHost: host, runID: run.ID}
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
