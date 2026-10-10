package control

import (
	"context"
	"database/sql"
	"encoding/json"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/identity"
	"github.com/levitateos/sodaos/internal/project"
)

type progressHost struct {
	*producerFlowHost
	states      map[string]project.FactoryState
	started     chan project.FactoryLaunch
	checksReady *atomic.Bool
}

func (h *progressHost) FactoryLaunch(ctx context.Context, in project.FactoryLaunch) (project.FactoryState, error) {
	state, err := h.producerFlowHost.FactoryLaunch(ctx, in)
	if in.Run.Role == project.RoleCoder {
		state.Output = "```result-json\n" + `{"status":"completed","summary":"done","candidate":"` + strings.Repeat("2", 40) + `","review_passed":false,"findings":[]}` + "\n```"
	} else {
		state.Output = "```review-json\n" + `{"verdict":"approve","summary":"solid","body":"LGTM","findings":[]}` + "\n```"
	}
	h.states[in.Run.ID] = state
	h.started <- in
	return state, err
}

func (h *progressHost) FactoryInspect(ctx context.Context, in project.FactoryInspect) (project.FactoryState, error) {
	if state, ok := h.states[in.ID]; ok {
		return state, nil
	}
	return h.fakePublishHost.FactoryInspect(ctx, in)
}

func (h *progressHost) FactoryStop(_ context.Context, in project.FactoryStop) (project.FactoryState, error) {
	return h.states[in.ID], nil
}

func (h *progressHost) PrepareCandidate(ctx context.Context, in project.FactoryCandidate) (project.PrepareState, error) {
	h.prepPhase = project.PrepareRunning
	if h.checksReady.Load() {
		h.prepPhase = project.PrepareReady
	}
	return h.producerFlowHost.PrepareCandidate(ctx, in)
}

type progressRevision struct{ revision *atomic.Int64 }

func (r progressRevision) ObserveNativeRevision(context.Context) (int64, bool, error) {
	return r.revision.Load(), true, nil
}

type progressInputs struct {
	AcceptanceSource
	DispatchReads
	revision *atomic.Int64
}

func (r progressInputs) ReadAcceptanceEvidence(ctx context.Context, repository, issue string, comments []string) (AcceptanceEvidence, error) {
	evidence, err := r.AcceptanceSource.ReadAcceptanceEvidence(ctx, repository, issue, comments)
	evidence.Revision = r.revision.Load()
	return evidence, err
}

func (r progressInputs) ReadDispatchInputs(ctx context.Context, repository, issue string, comments []string, target string) (DispatchInputs, error) {
	inputs, err := r.DispatchReads.ReadDispatchInputs(ctx, repository, issue, comments, target)
	inputs.Revision = r.revision.Load()
	return inputs, err
}

type progressLiveHost struct {
	*stubHost
	block     atomic.Bool
	inspected atomic.Int32
	stopped   atomic.Int32
	launched  atomic.Int32
	entered   chan struct{}
}

func (h *progressLiveHost) FactoryInspect(ctx context.Context, in project.FactoryInspect) (project.FactoryState, error) {
	h.inspected.Add(1)
	if h.block.Load() {
		close(h.entered)
		<-ctx.Done()
		return project.FactoryState{}, ctx.Err()
	}
	return project.FactoryState{ID: in.ID, Project: in.Project, Phase: project.FactoryRunning, Live: true}, nil
}

func (h *progressLiveHost) FactoryStop(context.Context, project.FactoryStop) (project.FactoryState, error) {
	h.stopped.Add(1)
	return project.FactoryState{}, nil
}

func (h *progressLiveHost) FactoryLaunch(context.Context, project.FactoryLaunch) (project.FactoryState, error) {
	h.launched.Add(1)
	return project.FactoryState{}, nil
}

func TestCoordinatorWakePreservesLiveCLIAndCloseJoinsInspection(t *testing.T) {
	ctx := context.Background()
	host := &progressLiveHost{stubHost: &stubHost{}, entered: make(chan struct{})}
	c := coordinatorFixture(t, host, nil)
	if err := c.Start(ctx, filepath.Join(t.TempDir(), "coordinator.lock")); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = c.Close() })
	run := recordRun(t, c, nil)
	bounded, _, cancel := c.readinessPass(ctx)
	defer cancel()
	readinessBudgetFrom(bounded).reserveNativeRPC(readinessMaxNativeRPC)
	c.advanceProgress(bounded)
	if host.inspected.Load() != 0 {
		t.Fatal("inspection exceeded the shared native-call budget")
	}
	c.advanceProgress(ctx)
	if host.inspected.Load() != 1 || host.stopped.Load() != 0 {
		t.Fatal("periodic inspection stopped a live CLI")
	}
	host.block.Store(true)
	select {
	case <-host.entered:
	case <-time.After(15 * time.Second):
		t.Fatal("owner did not inspect the live run")
	}
	closed := make(chan error, 1)
	go func() { closed <- c.Close() }()
	select {
	case err := <-closed:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Close did not cancel and join inspection")
	}
	if host.stopped.Load() != 0 || host.launched.Load() != 0 {
		t.Fatal("closing the wake worker changed native execution")
	}
	current, err := c.Store.FactoryRun(ctx, run.ID)
	if err != nil || current.Reconciled || current.Outcome != "" {
		t.Fatalf("live execution was settled during Close: reconciled=%t outcome=%q error=%v", current.Reconciled, current.Outcome, err)
	}
}

func TestCoordinatorWakeResumesReadinessWithoutRedelivery(t *testing.T) {
	ctx := context.Background()
	source := &fakeEvidenceSource{evidence: map[string]AcceptanceEvidence{
		"42/3": readinessEvidence(12, readinessView("3"), nil, nil),
	}}
	c := readinessCoordinator(t, source, &fakeObserver{revision: 12, idle: true})
	grantFullAuthority(t, c)
	if err := c.Start(ctx, filepath.Join(t.TempDir(), "coordinator.lock")); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = c.Close() })
	hint := readinessHint("delivery-background-resume", 42, 3)
	if err := c.Store.EnqueueReadinessWork(ctx, readinessDeliverySourceID(hint.Delivery), hint.Delivery, factory.DependenceRef{Repository: 42, Issue: 3}); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(15 * time.Second)
	for {
		seen, err := c.Store.IntakeDeliverySeen(ctx, hint.Delivery)
		if err != nil {
			t.Fatal(err)
		}
		if seen {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("durable readiness stayed pending without another delivery")
		}
		time.Sleep(20 * time.Millisecond)
	}
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
	control, err := c.Store.IssueControl(ctx, 42, 3)
	if err != nil || control.NativeRev != 12 {
		t.Fatalf("owner did not assess the durable source: revision=%d error=%v", control.NativeRev, err)
	}
}

func TestCoordinatorWakesForCodingDelayedChecksAndReviewer(t *testing.T) {
	ctx := context.Background()
	fx := publishSeed(t)
	policy, err := fx.db.RepositoryPolicy(ctx, fx.seed.repo)
	if err != nil {
		t.Fatal(err)
	}
	policy.Review = factory.ActorBindingRef{TokenID: 2, ActorID: 6, Kind: factory.OpReviewSubmit}
	if err = fx.db.SaveRepositoryPolicy(ctx, policy); err != nil {
		t.Fatal(err)
	}
	sponsor, err := fx.db.Sponsorship(ctx, fx.seed.repo, "conn")
	if err != nil {
		t.Fatal(err)
	}
	sponsor.Roles = append(sponsor.Roles, project.RoleReviewer)
	if err = fx.db.SaveSponsorship(ctx, sponsor); err != nil {
		t.Fatal(err)
	}
	decision := fx.seed.accept(t, 31, "d333333333333333333333333")
	fx.seed.queue(t, 31, decision.ID)
	key := strconv.FormatInt(fx.seed.repo, 10) + "/31"
	fx.reads.evidence[key] = AcceptanceEvidence{
		Revision: 41,
		Issue:    AcceptanceIssueView{Index: decision.IssueIndex, TitleDigest: decision.TitleDigest, ContentDigest: decision.ContentDigest, ContentVer: decision.ContentVersion, Visible: true},
		Comments: []AcceptanceComment{{ID: decision.Sources[0].ID, Digest: decision.Sources[0].Digest, ContentVer: decision.Sources[0].ContentVersion, Visible: true}},
	}
	var ready atomic.Bool
	var revision atomic.Int64
	revision.Store(41)
	host := &progressHost{producerFlowHost: &producerFlowHost{fakePublishHost: fx.host}, states: map[string]project.FactoryState{}, started: make(chan project.FactoryLaunch, 4), checksReady: &ready}
	inputs := progressInputs{AcceptanceSource: fx.reads, DispatchReads: fx.seed.reads, revision: &revision}
	fx.coord.Host, fx.coord.DispatchReads, fx.coord.Readiness = host, inputs, progressRevision{&revision}
	fx.coord.AcceptanceReads = inputs
	fx.coord.Broker = &stubBroker{get: func(string, string) (identity.Execution, error) { return identity.Execution{}, identity.ErrNotFound }, close: func(string, string) error { return nil }}
	reviewExecutor := &fakeReviewer{
		observed: factory.ReviewObservation{NativeRev: 42, ObservedUnix: time.Now().Unix()},
		outcome:  committedOutcome(`{"review_id":3}`),
		adopted: factory.ReviewOutcome{ReviewID: 3, CommentID: 4, ReviewerID: 6, PRID: 8, PRNumber: 9, IssueID: 10,
			HeadOID: strings.Repeat("2", 40), BaseOID: strings.Repeat("1", 40), CommitID: strings.Repeat("2", 40), Event: "APPROVED"},
	}
	fx.coord.Reviews = reviewExecutor
	executor := happyPublisher()
	observe := executor.observe
	executor.observe = func(work factory.PublicationWork) (factory.PublicationObservation, error) {
		refs, err := observe(work)
		refs.NativeRev = revision.Load()
		return refs, err
	}
	fx.wire(executor)
	pending := make(chan struct{}, 4)
	fx.coord.Checks = &fakeCheckObserver{observe: func(target factory.CheckTarget) (factory.ObservedChecks, error) {
		observed, err := happyCheckObserver().observe(target)
		observed.NativeRev = revision.Load()
		if !ready.Load() {
			observed.Checks[0].State = "pending"
			select {
			case pending <- struct{}{}:
			default:
			}
		}
		return observed, err
	}}
	if err = fx.coord.Start(ctx, filepath.Join(t.TempDir(), "coordinator.lock")); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = fx.coord.Close() })
	await := func() project.FactoryLaunch {
		t.Helper()
		select {
		case launched := <-host.started:
			return launched
		case <-time.After(15 * time.Second):
			t.Fatal("owner did not wake without another HTTP request")
			return project.FactoryLaunch{}
		}
	}
	coder := await()
	if coder.Run.Role != project.RoleCoder {
		t.Fatalf("first role = %q", coder.Run.Role)
	}
	select {
	case <-pending:
	case <-time.After(5 * time.Second):
		if err = fx.coord.Close(); err != nil {
			t.Fatal(err)
		}
		run, _ := fx.db.FactoryRun(ctx, coder.Run.ID)
		owner, _ := fx.db.AssignmentByRun(ctx, run.ID)
		publication, _ := fx.db.PublicationByAssignment(ctx, owner.ID)
		t.Fatalf("published candidate did not observe pending native checks: reconciled=%t assignment=%s publication=%s reason=%s", run.Reconciled, owner.Stage, publication.Stage, publication.Reason)
	}
	// Actions completed after issue intake ended, with no new HTTP request.
	revision.Store(42)
	ready.Store(true)
	reviewer := await()
	if reviewer.Run.Role != project.RoleReviewer || reviewer.Run.SourceCommit != strings.Repeat("2", 40) {
		t.Fatalf("reviewer lost exact candidate: %+v", reviewer.Run)
	}
	deadline := time.Now().Add(15 * time.Second)
	for {
		run, err := fx.db.FactoryRun(ctx, reviewer.Run.ID)
		if err != nil {
			t.Fatal(err)
		}
		owner, _ := fx.db.AssignmentByRun(ctx, coder.Run.ID)
		publication, _ := fx.db.PublicationByAssignment(ctx, owner.ID)
		checks, _ := fx.db.CheckAssessment(ctx, publication.Repository, publication.PRNumber)
		if run.Reconciled && len(publication.ReviewOperations) == 1 && checks.Verdict == factory.CheckPass {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("completed reviewer was not settled by the owner wake")
		}
		time.Sleep(20 * time.Millisecond)
	}
	if err = fx.coord.Close(); err != nil {
		t.Fatal(err)
	}
	if len(host.launches) != 2 || len(executor.pushes) != 1 || len(executor.creates) != 1 || len(reviewExecutor.submitted) != 1 {
		t.Fatal("wake duplicated a provider role, branch or PR")
	}
	control, err := fx.db.IssueControl(ctx, fx.seed.repo, 31)
	if err != nil || control.NativeRev != 42 {
		t.Fatalf("native readiness did not refresh: %+v %v", control, err)
	}
}

func TestCoordinatorWakeClosesExpiredRootWithoutMoreRuns(t *testing.T) {
	ctx := context.Background()
	fx := publishSeed(t)
	owner := fx.finishReported(t, 32)
	fx.wire(happyPublisher())
	if err := fx.coord.Start(ctx, filepath.Join(t.TempDir(), "coordinator.lock")); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = fx.coord.Close() })
	prior, err := fx.db.AttemptAllowance(ctx, owner.Repository, owner.Issue)
	if err != nil {
		t.Fatal(err)
	}
	prior.ActiveSeconds, prior.CheckpointUnix = int64(prior.Limits.ActiveMinutes*60), time.Now().Unix()
	prior.Revision++
	raw, err := json.Marshal(prior)
	if err != nil {
		t.Fatal(err)
	}
	fixture, err := sql.Open("pgx", fx.dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = fixture.Close() }()
	if _, err = fixture.ExecContext(ctx, `UPDATE factory_attempt_allowances SET revision=$1,data=$2 WHERE root_assignment=$3`, prior.Revision, string(raw), owner.ID); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(15 * time.Second)
	for {
		closed, err := fx.db.AttemptAllowance(ctx, owner.Repository, owner.Issue)
		if err != nil {
			t.Fatal(err)
		}
		if closed.Closed && !closed.Active {
			if closed.RootAssignment != prior.RootAssignment || closed.Limits != prior.Limits || closed.ActiveSeconds < prior.ActiveSeconds {
				t.Fatal("owner wake replenished the attempt allowance")
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("expired root stayed active without another request")
		}
		time.Sleep(20 * time.Millisecond)
	}
	fx.coord.advanceProgress(ctx)
	if err = fx.coord.Close(); err != nil {
		t.Fatal(err)
	}
	runs, err := fx.db.FactoryRuns(ctx, 10)
	if err != nil || len(runs) != 1 || runs[0].ID != owner.Run {
		t.Fatalf("expiry launched or replaced a run: %+v %v", runs, err)
	}
	if len(fx.exec.pushes) != 1 || len(fx.exec.creates) != 1 {
		t.Fatal("expiry rewrote the recorded publication")
	}
}
