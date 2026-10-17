package control

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/identity"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
)

type progressHost struct {
	*producerFlowHost
	states      map[string]project.FactoryState
	started     chan project.FactoryLaunch
	checksReady *atomic.Bool
	db          *store.Store
	mu          sync.Mutex
	errs        []string
}

// recordLaunchErr keeps one fixture-output failure for the test to assert.
// Hosts never panic and never fabricate output: a failed typed emission
// leaves the launch without output so production waits instead.
func (h *progressHost) recordLaunchErr(detail string) {
	h.mu.Lock()
	h.errs = append(h.errs, detail)
	h.mu.Unlock()
}

// launchErrs reports every recorded fixture-output failure.
func (h *progressHost) launchErrs() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]string(nil), h.errs...)
}

// The native job and its durable receipt outlive a disconnected launch RPC.
// Completion below changes only host truth; the production owner must perform
// all accounting and publication itself.
type lostReplyProgressHost struct {
	*producerFlowHost
	mu       sync.Mutex
	db       *store.Store
	states   map[string]project.FactoryState
	first    string
	entered  chan project.FactoryLaunch
	complete chan struct{}
	errs     []string
}

func (h *lostReplyProgressHost) recordLaunchErr(detail string) {
	h.mu.Lock()
	h.errs = append(h.errs, detail)
	h.mu.Unlock()
}

func (h *lostReplyProgressHost) launchErrs() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]string(nil), h.errs...)
}

func (h *lostReplyProgressHost) FactoryLaunch(ctx context.Context, in project.FactoryLaunch) (project.FactoryState, error) {
	h.mu.Lock()
	h.launches = append(h.launches, in)
	if settled, ok := h.states[in.Run.ID]; ok && !settled.Live {
		h.mu.Unlock()
		return settled, nil
	}
	if h.first == "" {
		h.first = in.Run.ID
	}
	h.states[in.Run.ID] = project.FactoryState{ID: in.Run.ID, Project: in.Run.Project, Role: in.Run.Role, Phase: project.FactoryRunning, Live: true, Delivered: true, LeaseID: "lease", Generation: 3}
	first := h.first == in.Run.ID && len(h.launches) == 1
	h.mu.Unlock()
	if !first && in.Run.ID == h.first {
		// A relaunch of the lost run before it settles repeats the lost
		// reply instead of duplicating execution.
		select {
		case <-ctx.Done():
			return project.FactoryState{}, ctx.Err()
		case <-h.complete:
			return h.FactoryInspect(ctx, project.FactoryInspect{ID: in.Run.ID})
		}
	}
	if !first {
		// Later phased runs (the advanced implement child) settle through
		// the same typed wire: report the admitted phase output promptly.
		// Only the first launch simulates the lost reply.
		output, outErr := wakeImplementOutput(ctx, h.db, in.Run.ID)
		if outErr != nil {
			h.recordLaunchErr(outErr.Error())
			return project.FactoryState{}, outErr
		}
		h.mu.Lock()
		h.states[in.Run.ID] = project.FactoryState{ID: in.Run.ID, Project: in.Run.Project, Role: in.Run.Role, Phase: project.FactoryCompleted, Retirement: "confirmed", Live: false, Delivered: true, LeaseID: "lease", Generation: 3, CredentialReturned: true, Output: output}
		completed := h.states[in.Run.ID]
		h.mu.Unlock()
		return completed, nil
	}
	h.entered <- in
	select {
	case <-ctx.Done():
		return project.FactoryState{}, ctx.Err()
	case <-h.complete:
		return h.FactoryInspect(ctx, project.FactoryInspect{ID: in.Run.ID})
	}
}

func (h *lostReplyProgressHost) FactoryInspect(_ context.Context, in project.FactoryInspect) (project.FactoryState, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if state, ok := h.states[in.ID]; ok {
		return state, nil
	}
	return project.FactoryState{}, errors.New("inspect names an unlaunched run")
}

func (h *lostReplyProgressHost) FactoryStop(_ context.Context, in project.FactoryStop) (project.FactoryState, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	state, ok := h.states[in.ID]
	if !ok {
		return project.FactoryState{}, nil
	}
	if state.Live {
		return project.FactoryState{}, errors.New("live run must not be stopped by observation timeout")
	}
	return state, nil
}

func (h *lostReplyProgressHost) finish() {
	h.mu.Lock()
	if h.first != "" {
		if state, ok := h.states[h.first]; ok && state.Live {
			output, outErr := wakeInvestigateOutput(context.Background(), h.db, h.first)
			if outErr != nil {
				h.errs = append(h.errs, outErr.Error())
			} else {
				state.Live = false
				state.Phase, state.Retirement = project.FactoryCompleted, "confirmed"
				state.CredentialReturned = true
				state.Output = output
				h.states[h.first] = state
			}
		}
	}
	h.mu.Unlock()
	close(h.complete)
}

func TestCoordinatorSettlesLostLaunchReplyWithoutManualProgress(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	fx := publishSeed(t)
	decision := fx.seed.accept(t, 31, "d333333333333333333333333")
	fx.seed.queue(t, 31, decision.ID)
	fx.reads.evidence[strconv.FormatInt(fx.seed.repo, 10)+"/31"] = AcceptanceEvidence{
		Revision: 41,
		Issue:    AcceptanceIssueView{Index: decision.IssueIndex, TitleOccurrence: factory.OriginalOccurrence, LifecycleOccurrence: factory.OriginalOccurrence, TitleDigest: decision.TitleDigest, ContentDigest: decision.ContentDigest, ContentVer: decision.ContentVersion, Visible: true},
		Comments: []AcceptanceComment{{ID: decision.Sources[0].ID, Digest: decision.Sources[0].Digest, ContentVer: decision.Sources[0].ContentVersion, Visible: true}},
	}
	host := &lostReplyProgressHost{producerFlowHost: &producerFlowHost{fakePublishHost: fx.host}, db: fx.db, states: map[string]project.FactoryState{}, entered: make(chan project.FactoryLaunch, 1), complete: make(chan struct{})}
	fx.coord.Host, fx.coord.DispatchReads = host, fx.seed.reads
	fx.coord.Broker = &stubBroker{get: func(kind, id string) (identity.Execution, error) {
		return identity.Execution{Kind: kind, ExecutionID: id, State: identity.ExecutionTerminal, Digest: "digest", LeaseID: "lease", Binding: &identity.Binding{Kind: kind, ID: id, Generation: 3, CredentialRoot: "/home/coder/runs/" + id}}, nil
	}, close: func(string, string) error { return nil }}
	executor := happyPublisher()
	observe := executor.observe
	executor.observe = func(work factory.PublicationWork) (factory.PublicationObservation, error) {
		observation, err := observe(work)
		observation.NativeRev = 41
		return observation, err
	}
	fx.wire(executor)
	if err := fx.coord.Start(ctx, filepath.Join(t.TempDir(), "coordinator.lock")); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = fx.coord.Close() })
	request, cancel := context.WithTimeout(ctx, 300*time.Millisecond)
	defer cancel()
	done := make(chan struct{})
	go func() {
		_, _, _ = fx.coord.ObserveIssueEvent(request, readinessHint("lost-launch-reply", fx.seed.repo, 31))
		close(done)
	}()
	var launch project.FactoryLaunch
	select {
	case launch = <-host.entered:
	case <-time.After(time.Second):
		t.Fatal("normal intake did not launch the recorded run")
	}
	if launch.Run.Role != project.RoleCoder || launch.Run.ExecutionPhase != project.ExecutionInvestigate {
		t.Fatalf("intake opened the wrong phase: role=%q phase=%q", launch.Run.Role, launch.Run.ExecutionPhase)
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		host.finish()
		<-done
		t.Fatal("request cancellation retained the scheduling lease")
	}
	run, err := fx.db.FactoryRun(ctx, launch.Run.ID)
	if err != nil || run.Reconciled || !run.Deadline.Equal(launch.Run.Deadline) {
		t.Fatalf("lost reply changed admitted run custody/deadline: %+v %v", run, err)
	}
	assignment, err := fx.db.AssignmentByRun(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	reservation, err := fx.db.Reservation(ctx, assignment.ID)
	if err != nil || reservation.State != factory.ReservationHeld {
		t.Fatalf("live run lost its held reservation: %+v %v", reservation, err)
	}
	host.finish()
	if errs := host.launchErrs(); len(errs) != 0 {
		t.Fatalf("lost-run typed report failed: %v", errs)
	}
	deadline := time.Now().Add(15 * time.Second)
	for {
		stored, readErr := fx.db.FactoryRun(ctx, run.ID)
		publication, publicationErr := fx.db.PublicationByAssignment(ctx, assignment.ID)
		if readErr == nil && publicationErr == nil && stored.Reconciled && publication.Stage == factory.PublicationPublished {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("production owner did not settle/publish retained result: run=%+v publication=%+v errors=%v/%v", stored, publication, readErr, publicationErr)
		}
		time.Sleep(20 * time.Millisecond)
	}
	if err := fx.coord.Close(); err != nil {
		t.Fatal(err)
	}
	if errs := host.launchErrs(); len(errs) != 0 {
		t.Fatalf("typed phase reports failed: %v", errs)
	}
	var investigate, implement int
	for _, launched := range host.launches {
		switch launched.Run.ExecutionPhase {
		case project.ExecutionInvestigate:
			investigate++
		case project.ExecutionImplement:
			implement++
		default:
			t.Fatalf("lost reply admitted an unadmitted phase: %q", launched.Run.ExecutionPhase)
		}
		if launched.Run.Role != project.RoleCoder {
			t.Fatalf("lost reply admitted a non-coder role: %q", launched.Run.Role)
		}
	}
	if investigate != 1 || implement != 1 || len(executor.pushes) != 1 || len(executor.creates) != 1 {
		t.Fatalf("lost response duplicated a provider run, branch or PR: investigate=%d implement=%d pushes=%d creates=%d", investigate, implement, len(executor.pushes), len(executor.creates))
	}
}

func TestCoordinatorWakeRetiresExpiredRecordedRun(t *testing.T) {
	t.Parallel()
	for _, phase := range []string{project.FactoryApproved, project.FactoryRunning} {
		t.Run(phase, func(t *testing.T) {
			t.Parallel()
			var stops atomic.Int32
			host := &stubHost{
				inspect: func(in project.FactoryInspect) (project.FactoryState, error) {
					return project.FactoryState{ID: in.ID, Project: in.Project, Phase: phase, Live: phase == project.FactoryRunning}, nil
				},
				stop: func(in project.FactoryStop) (project.FactoryState, error) {
					stops.Add(1)
					return project.FactoryState{ID: in.ID, Project: in.Project, Phase: project.FactoryStopped, Retirement: "confirmed", Reason: "expired"}, nil
				},
			}
			broker := &stubBroker{get: func(kind, id string) (identity.Execution, error) {
				return identity.Execution{Kind: kind, ExecutionID: id, State: identity.ExecutionTerminal}, nil
			}, close: func(string, string) error { return nil }}
			c := coordinatorFixture(t, host, broker)
			if err := c.Start(context.Background(), filepath.Join(t.TempDir(), "coordinator.lock")); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = c.Close() })
			// Recorded after startup reconciliation, so only the serving owner
			// can retire it. No request or test-provided advancement follows.
			run := recordRun(t, c, func(run *factory.Run) {
				run.Started = time.Now().Add(-2 * time.Minute)
				run.Deadline = time.Now().Add(-time.Second)
			})
			deadline := time.Now().Add(12 * time.Second)
			for {
				current, err := c.Store.FactoryRun(context.Background(), run.ID)
				if err != nil {
					t.Fatal(err)
				}
				if current.Reconciled {
					if current.Outcome != factory.Cancelled || !current.Deadline.Equal(run.Deadline) || stops.Load() != 1 {
						t.Fatalf("expiry changed lifetime or repeated retirement: %+v stops=%d", current, stops.Load())
					}
					break
				}
				if time.Now().After(deadline) {
					t.Fatalf("expired %s receipt retained active custody", phase)
				}
				time.Sleep(20 * time.Millisecond)
			}
		})
	}
}

func (h *progressHost) FactoryLaunch(ctx context.Context, in project.FactoryLaunch) (project.FactoryState, error) {
	state, err := h.producerFlowHost.FactoryLaunch(ctx, in)
	// Phased runs report the CURRENT typed v2 contract for their admitted
	// phase; pre-phase children (CaseA's reviewer under its pre-phase
	// owner) keep the legacy fences. Never force Phase empty, never inject
	// a result past the parser: the output must settle through production.
	switch in.Run.ExecutionPhase {
	case project.ExecutionInvestigate:
		output, outErr := wakeInvestigateOutput(ctx, h.db, in.Run.ID)
		if outErr != nil {
			h.recordLaunchErr(outErr.Error())
		} else {
			state.Output = output
		}
	case project.ExecutionImplement:
		output, outErr := wakeImplementOutput(ctx, h.db, in.Run.ID)
		if outErr != nil {
			h.recordLaunchErr(outErr.Error())
		} else {
			state.Output = output
		}
	case project.ExecutionReview:
		output, outErr := wakeReviewClearOutput(ctx, h.db, in.Run.ID)
		if outErr != nil {
			h.recordLaunchErr(outErr.Error())
		} else {
			state.Output = output
		}
	default:
		if in.Run.Role == project.RoleCoder {
			state.Output = "```result-json\n" + `{"status":"completed","summary":"done","candidate":"` + strings.Repeat("2", 40) + `","review_passed":false,"findings":[]}` + "\n```"
		} else {
			state.Output = "```review-json\n" + `{"verdict":"approve","summary":"solid","body":"LGTM","findings":[]}` + "\n```"
		}
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

type preparationAdmissionRaceInputs struct {
	DispatchReads
	reassess func(context.Context) error
}

func (r *preparationAdmissionRaceInputs) ReadDispatchInputs(ctx context.Context, repository, issue string, comments []string, target string) (DispatchInputs, error) {
	inputs, err := r.DispatchReads.ReadDispatchInputs(ctx, repository, issue, comments, target)
	if err == nil && r.reassess != nil {
		err = r.reassess(ctx)
	}
	return inputs, err
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
		Issue:    AcceptanceIssueView{Index: decision.IssueIndex, TitleOccurrence: factory.OriginalOccurrence, LifecycleOccurrence: factory.OriginalOccurrence, TitleDigest: decision.TitleDigest, ContentDigest: decision.ContentDigest, ContentVer: decision.ContentVersion, Visible: true},
		Comments: []AcceptanceComment{{ID: decision.Sources[0].ID, Digest: decision.Sources[0].Digest, ContentVer: decision.Sources[0].ContentVersion, Visible: true}},
	}
	var ready atomic.Bool
	var revision atomic.Int64
	revision.Store(41)
	host := &progressHost{producerFlowHost: &producerFlowHost{fakePublishHost: fx.host}, db: fx.db, states: map[string]project.FactoryState{}, started: make(chan project.FactoryLaunch, 4), checksReady: &ready}
	inputs := progressInputs{AcceptanceSource: fx.reads, DispatchReads: fx.seed.reads, revision: &revision}
	fx.coord.Host, fx.coord.DispatchReads, fx.coord.Readiness = host, inputs, progressRevision{&revision}
	fx.coord.AcceptanceReads = inputs
	fx.coord.Broker = &stubBroker{get: func(string, string) (identity.Execution, error) { return identity.Execution{}, identity.ErrNotFound }, close: func(string, string) error { return nil }}
	reviewExecutor := &fakeReviewer{
		observed: factory.ReviewObservation{NativeRev: 42, ObservedUnix: time.Now().Unix()},
		outcome:  committedOutcome(`{"review_id":3}`),
		adopted: factory.ReviewOutcome{
			ReviewID: 3, CommentID: 4, ReviewerID: 6, PRID: 8, PRNumber: 9, IssueID: 10,
			HeadOID: strings.Repeat("2", 40), BaseOID: strings.Repeat("1", 40), CommitID: strings.Repeat("2", 40), Event: "APPROVED",
		},
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
	if coder.Run.Role != project.RoleCoder || coder.Run.ExecutionPhase != project.ExecutionInvestigate {
		t.Fatalf("first role = %q phase = %q", coder.Run.Role, coder.Run.ExecutionPhase)
	}
	if errs := host.launchErrs(); len(errs) != 0 {
		t.Fatalf("investigate typed report failed: %v", errs)
	}
	// The phased journey advances on the owner wake: the planned
	// implementation child launches on a later tick than its investigate
	// root, still with no new HTTP request.
	implement := await()
	if implement.Run.Role != project.RoleCoder || implement.Run.ExecutionPhase != project.ExecutionImplement {
		t.Fatalf("implement lost its phase: role=%q phase=%q", implement.Run.Role, implement.Run.ExecutionPhase)
	}
	if errs := host.launchErrs(); len(errs) != 0 {
		t.Fatalf("implement typed report failed: %v", errs)
	}
	// Explicit publication-readiness stage for the newly introduced
	// implement child: the checks below observe the investigate root's
	// publication, which must reach published with the fixed candidate
	// and the implement run linked before the original pending-check
	// deadline starts. This separates new-phase publication from check
	// observation; the checks deadline itself stays at 5s.
	investigateRoot, err := fx.db.AssignmentByRun(ctx, coder.Run.ID)
	if err != nil {
		t.Fatal(err)
	}
	publishDeadline := time.Now().Add(15 * time.Second)
	for {
		readyPub, readyErr := fx.db.PublicationByAssignment(ctx, investigateRoot.ID)
		if readyErr == nil && readyPub.Stage == factory.PublicationPublished &&
			readyPub.AssignmentID == investigateRoot.ID &&
			readyPub.Candidate == strings.Repeat("2", 40) &&
			readyPub.Run == implement.Run.ID {
			break
		}
		if time.Now().After(publishDeadline) {
			t.Fatalf("implement child never reached published publication: pub=%+v err=%v", readyPub, readyErr)
		}
		time.Sleep(20 * time.Millisecond)
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
	if reviewer.Run.Role != project.RoleReviewer || reviewer.Run.ExecutionPhase != project.ExecutionReview || reviewer.Run.SourceCommit != strings.Repeat("2", 40) {
		t.Fatalf("reviewer lost exact candidate: %+v", reviewer.Run)
	}
	if errs := host.launchErrs(); len(errs) != 0 {
		t.Fatalf("reviewer typed report failed: %v", errs)
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
	if errs := host.launchErrs(); len(errs) != 0 {
		t.Fatalf("typed phase reports failed: %v", errs)
	}
	var investigate, implementCount, review int
	for _, launched := range host.launches {
		switch launched.Run.ExecutionPhase {
		case project.ExecutionInvestigate:
			investigate++
		case project.ExecutionImplement:
			implementCount++
		case project.ExecutionReview:
			review++
		default:
			t.Fatalf("wake admitted an unadmitted phase: %q", launched.Run.ExecutionPhase)
		}
	}
	if investigate != 1 || implementCount != 1 || review != 1 || len(executor.pushes) != 1 || len(executor.creates) != 1 || len(reviewExecutor.submitted) != 1 {
		t.Fatalf("wake duplicated a provider role, branch or PR: investigate=%d implement=%d review=%d pushes=%d creates=%d submitted=%d", investigate, implementCount, review, len(executor.pushes), len(executor.creates), len(reviewExecutor.submitted))
	}
	control, err := fx.db.IssueControl(ctx, fx.seed.repo, 31)
	if err != nil || control.NativeRev != 42 {
		t.Fatalf("native readiness did not refresh: %+v %v", control, err)
	}
}

func TestCoordinatorWakeClosesExpiredRootWithoutMoreRuns(t *testing.T) {
	t.Parallel()
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

func TestCoordinatorWakeRetriesStaleReviewPreparationAdmission(t *testing.T) {
	t.Parallel()
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
	owner := fx.finishReported(t, 33)
	key := strconv.FormatInt(owner.Repository, 10) + "/33"
	evidence := fx.reads.evidence[key]
	evidence.Revision = 41
	fx.reads.evidence[key] = evidence
	prior, err := fx.db.AttemptAllowance(ctx, owner.Repository, owner.Issue)
	if err != nil {
		t.Fatal(err)
	}
	var ready atomic.Bool
	ready.Store(true)
	var reassessing atomic.Bool
	reassessing.Store(true)
	// raced counts successful stale-input mutations inside the race
	// hook: the witness below requires the race to have actually run,
	// not just the earlier reservation to have landed.
	var raced atomic.Int64
	host := &progressHost{producerFlowHost: &producerFlowHost{fakePublishHost: fx.host}, db: fx.db, states: map[string]project.FactoryState{}, started: make(chan project.FactoryLaunch, 2), checksReady: &ready}
	fx.coord.Host = host
	fx.coord.DispatchReads = &preparationAdmissionRaceInputs{DispatchReads: fx.seed.reads, reassess: func(ctx context.Context) error {
		if !reassessing.Load() {
			return nil
		}
		control, err := fx.db.IssueControl(ctx, owner.Repository, owner.Issue)
		if err != nil {
			return err
		}
		// A concurrent normal assessment changes semantic readiness after
		// planning, before the Store's authoritative preparation admission.
		if control.Fingerprint == strings.Repeat("f", 64) {
			control.Fingerprint = strings.Repeat("e", 64)
		} else {
			control.Fingerprint = strings.Repeat("f", 64)
		}
		_, _, err = fx.db.RecordIssueAssessment(ctx, control, time.Now())
		if err == nil {
			raced.Add(1)
		}
		return err
	}}
	fx.coord.Reviews = &fakeReviewer{}
	fx.coord.Broker = &stubBroker{get: func(string, string) (identity.Execution, error) { return identity.Execution{}, identity.ErrNotFound }, close: func(string, string) error { return nil }}
	executor := happyPublisher()
	observe := executor.observe
	executor.observe = func(work factory.PublicationWork) (factory.PublicationObservation, error) {
		observation, err := observe(work)
		observation.NativeRev = 41
		return observation, err
	}
	fx.wire(executor)
	if err = fx.coord.Start(ctx, filepath.Join(t.TempDir(), "coordinator.lock")); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = fx.coord.Close() })
	publication, err := fx.db.PublicationByAssignment(ctx, owner.ID)
	if err != nil || publication.Stage != factory.PublicationPublished {
		t.Fatalf("stale admission changed publication: stage=%s err=%v", publication.Stage, err)
	}
	// The D08 review reservation is the legitimate charge the stale cycle
	// must carry: exactly one reviewer assignment bound to the publication
	// child identity, revision bumped once, every unrelated budget and the
	// clock forward-only. Replays of the same charge commit nothing, so the
	// bump stays exactly one across wake ticks.
	expectedChild := publicationChildID(publication.ID, publication.Candidate, project.RoleReviewer)
	var current factory.AttemptAllowance
	chargeDeadline := time.Now().Add(10 * time.Second)
	for {
		var readErr error
		current, readErr = fx.db.AttemptAllowance(ctx, owner.Repository, owner.Issue)
		if readErr != nil {
			t.Fatal(readErr)
		}
		if current.Revision == prior.Revision+1 && raced.Load() >= 1 {
			break
		}
		if current.Revision != prior.Revision && current.Revision != prior.Revision+1 {
			t.Fatalf("stale admission overcharged the allowance: before=%+v after=%+v", prior, current)
		}
		if time.Now().After(chargeDeadline) {
			t.Fatalf("stale cycle recorded no review reservation or race never ran: before=%+v after=%+v raced=%d", prior, current, raced.Load())
		}
		time.Sleep(20 * time.Millisecond)
	}
	nowUnix := time.Now().Unix()
	if current.RootAssignment != prior.RootAssignment || current.Repository != prior.Repository || current.Issue != prior.Issue ||
		current.Limits != prior.Limits || current.Closed || !current.Active ||
		current.ActiveSeconds < prior.ActiveSeconds || current.CheckpointUnix < prior.CheckpointUnix ||
		current.CheckpointUnix > nowUnix ||
		(prior.Active && current.ActiveSeconds-prior.ActiveSeconds > current.CheckpointUnix-prior.CheckpointUnix) ||
		len(current.Corrections) != 0 || !slices.Equal(current.DeliveryCalls, prior.DeliveryCalls) ||
		len(current.UnchangedHeadExtra) != 0 || current.Completion != nil {
		t.Fatalf("stale admission changed more than the review reservation: before=%+v after=%+v now=%d", prior, current, nowUnix)
	}
	if len(current.Reviews) != len(prior.Reviews)+1 {
		t.Fatalf("review reservation is not exactly one charge: before=%+v after=%+v child=%s", prior, current, expectedChild)
	}
	if !slices.Equal(current.Reviews[:len(current.Reviews)-1], prior.Reviews) || current.Reviews[len(current.Reviews)-1] != expectedChild {
		t.Fatalf("review reservation lost its exact identity: before=%+v after=%+v child=%s", prior, current, expectedChild)
	}
	if len(current.ReviewedHeads) != len(prior.ReviewedHeads)+1 || current.ReviewedHeads[publication.Candidate] != expectedChild {
		t.Fatalf("review reservation lost its exact head binding: before=%+v after=%+v child=%s", prior, current, expectedChild)
	}
	for head, id := range prior.ReviewedHeads {
		if current.ReviewedHeads[head] != id {
			t.Fatalf("review reservation rewrote a prior head binding: before=%+v after=%+v", prior, current)
		}
	}
	if len(host.preparations) != 0 || len(host.launches) != 0 {
		t.Fatal("stale admission reached native preparation or execution")
	}
	reassessing.Store(false)
	select {
	case launch := <-host.started:
		child, err := fx.db.AssignmentByRun(ctx, launch.Run.ID)
		if launch.Run.Role != project.RoleReviewer || launch.Run.SourceCommit != publication.Candidate ||
			err != nil || child.ID != publicationChildID(publication.ID, publication.Candidate, project.RoleReviewer) ||
			child.AttemptRoot != owner.AttemptRoot || child.PublicationAssignment != owner.ID {
			t.Fatalf("owner wake lost exact reviewer binding: %+v", launch.Run)
		}
	case <-time.After(15 * time.Second):
		if err = fx.coord.Close(); err != nil {
			t.Fatal(err)
		}
		publication, _ = fx.db.PublicationByAssignment(ctx, owner.ID)
		current, _ = fx.db.AttemptAllowance(ctx, owner.Repository, owner.Issue)
		admissions, _ := fx.db.CandidatePreparations(ctx, owner.Repository, owner.ProjectID, owner.Issue, owner.AttemptRoot, "", 64, 0)
		t.Fatalf("owner wake did not retry stale preparation admission: stage=%s reason=%s active=%t closed=%t admissions=%d preparations=%d", publication.Stage, publication.Reason, current.Active, current.Closed, len(admissions), len(host.preparations))
	}
	if err = fx.coord.Close(); err != nil {
		t.Fatal(err)
	}
	if len(host.preparations) != 1 || len(host.launches) != 1 || len(fx.exec.pushes) != 1 || len(fx.exec.creates) != 1 {
		t.Fatal("admission recovery duplicated preparation, execution or publication")
	}
}
