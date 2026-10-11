package control

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
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

// These boundaries supply native/provider facts. Only the serving Coordinator
// may turn those facts into settlements, publications, reviews or merges.
type completionHost struct {
	*producerFlowHost
	db      *store.Store
	states  map[string]project.FactoryState
	started chan project.FactoryLaunch
}

func (h *completionHost) FactoryLaunch(ctx context.Context, in project.FactoryLaunch) (project.FactoryState, error) {
	owner, err := h.db.AssignmentByRun(ctx, in.Run.ID)
	if err != nil {
		return project.FactoryState{}, err
	}
	state, err := h.producerFlowHost.FactoryLaunch(ctx, in)
	if err != nil {
		return state, err
	}
	state.LeaseID, state.Generation, state.Delivered = "lease-"+in.Run.ID, 1, true
	switch {
	case owner.Issue == 32:
		// Leave the dependent live so its admission is observable without
		// introducing another publication journey into this fixture.
		state.Phase, state.Live = project.FactoryRunning, true
	case in.Run.Role == project.RoleReviewer:
		verdict, body, findings := "approve", "corrected candidate is sound", "[]"
		if in.Run.SourceCommit == strings.Repeat("2", 40) {
			verdict, body, findings = "request-changes", "empty input needs a regression test", `["empty input"]`
		}
		state.Output = "```review-json\n" + fmt.Sprintf(`{"verdict":%q,"summary":"independent review","body":%q,"findings":%s}`, verdict, body, findings) + "\n```"
	default:
		candidate := strings.Repeat("2", 40)
		if in.Run.SourceCommit == candidate {
			candidate = strings.Repeat("3", 40)
		}
		state.Output = "```result-json\n" + fmt.Sprintf(`{"status":"completed","summary":"implemented","candidate":%q,"review_passed":false,"findings":[]}`, candidate) + "\n```"
	}
	if !state.Live {
		state.Retirement, state.CredentialReturned = "confirmed", true
	}
	h.states[in.Run.ID] = state
	h.started <- in
	return state, nil
}

func (h *completionHost) FactoryInspect(_ context.Context, in project.FactoryInspect) (project.FactoryState, error) {
	state, ok := h.states[in.ID]
	if !ok {
		return project.FactoryState{}, errors.New("unknown completion fixture run")
	}
	return state, nil
}

func (h *completionHost) FactoryStop(ctx context.Context, in project.FactoryStop) (project.FactoryState, error) {
	return h.FactoryInspect(ctx, project.FactoryInspect{ID: in.ID, Project: in.Project})
}

type completionReviewer struct {
	*fakeReviewer
	events map[string]string
}

type completionInputs struct {
	progressInputs
	mu     sync.Mutex
	closed atomic.Int64
}

func (r *completionInputs) ReadAcceptanceEvidence(ctx context.Context, repository, issue string, comments []string) (AcceptanceEvidence, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	evidence, err := r.progressInputs.ReadAcceptanceEvidence(ctx, repository, issue, comments)
	if issue == "31" && r.closed.Load() > 0 {
		evidence.Issue.Closed, evidence.Issue.ClosedUnix = true, r.closed.Load()
		evidence.Issue.ClosingOccurrence, evidence.Issue.LifecycleOccurrence = "501", "501"
	}
	return evidence, err
}

func (r *completionInputs) ReadDispatchInputs(ctx context.Context, repository, issue string, comments []string, target string) (DispatchInputs, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	inputs, err := r.progressInputs.ReadDispatchInputs(ctx, repository, issue, comments, target)
	if r.closed.Load() > 0 {
		inputs.Tip = strings.Repeat("3", 40)
	}
	if issue == "31" && r.closed.Load() > 0 {
		inputs.Issue.Closed, inputs.Issue.ClosedUnix = true, r.closed.Load()
		inputs.Issue.ClosingOccurrence, inputs.Issue.LifecycleOccurrence = "501", "501"
	}
	return inputs, err
}

func (r *completionReviewer) SubmitReview(ctx context.Context, work factory.ReviewWork) (factory.OperationOutcome, error) {
	id := int64(31)
	if work.Event == "REQUEST_CHANGES" {
		id = 30
	}
	r.outcome = committedOutcome(fmt.Sprintf(`{"review_id":%d}`, id))
	outcome, err := r.fakeReviewer.SubmitReview(ctx, work)
	if err == nil {
		r.events[work.HeadOID] = work.Event
	}
	return outcome, err
}

func (*completionReviewer) AdoptReview(work factory.ReviewWork, outcome factory.OperationOutcome) (factory.ReviewOutcome, error) {
	id := int64(31)
	if work.Event == "REQUEST_CHANGES" {
		id = 30
	}
	return factory.ReviewOutcome{
		Operation: outcome, ReviewID: id, CommentID: id + 10, ReviewerID: work.ActorID,
		PRID: work.PRID, PRNumber: work.PRNumber, IssueID: work.IssueID,
		HeadOID: work.HeadOID, BaseOID: work.BaseOID, CommitID: work.HeadOID, Event: work.Event,
	}, nil
}

func TestServingCoordinatorCompletesCorrectionAndPicksUpDependent(t *testing.T) {
	ctx := context.Background()
	fx := publishSeed(t, map[string]string{"README.md": dispatchDigest("synthetic selected source: README.md")}) // substrate and standing operator/project authority only
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
	var revision atomic.Int64
	revision.Store(41)
	var correctedChecksReady atomic.Bool
	checksPending := make(chan struct{}, 1)
	host := &completionHost{producerFlowHost: &producerFlowHost{fakePublishHost: fx.host}, db: fx.db, states: map[string]project.FactoryState{}, started: make(chan project.FactoryLaunch, 8)}
	inputs := &completionInputs{progressInputs: progressInputs{AcceptanceSource: fx.reads, DispatchReads: fx.seed.reads, revision: &revision}}
	fx.coord.Host, fx.coord.AcceptanceReads, fx.coord.DispatchReads = host, inputs, inputs
	fx.coord.Readiness = progressRevision{&revision}
	fx.coord.Broker = &stubBroker{get: func(kind, id string) (identity.Execution, error) {
		return identity.Execution{Kind: kind, ExecutionID: id, State: identity.ExecutionTerminal, Digest: "digest", LeaseID: "lease-" + id, Binding: &identity.Binding{Kind: kind, ID: id, Generation: 1, CredentialRoot: "/home/coder/runs/" + id}}, nil
	}, close: func(string, string) error { return nil }}
	publisher := happyPublisher()
	observe := publisher.observe
	publisher.observe = func(work factory.PublicationWork) (factory.PublicationObservation, error) {
		observation, err := observe(work)
		observation.NativeRev = revision.Load()
		return observation, err
	}
	fx.wire(publisher)
	reviewer := &completionReviewer{fakeReviewer: &fakeReviewer{observed: factory.ReviewObservation{NativeRev: 41, ObservedUnix: time.Now().Unix()}}, events: map[string]string{}}
	fx.coord.Reviews = reviewer
	fx.coord.Checks = &fakeCheckObserver{observe: func(target factory.CheckTarget) (factory.ObservedChecks, error) {
		observed, err := happyCheckObserver().observe(target)
		observed.NativeRev = revision.Load()
		if target.HeadOID == strings.Repeat("3", 40) && !correctedChecksReady.Load() {
			observed.Checks[0].State = "pending"
			select {
			case checksPending <- struct{}{}:
			default:
			}
		}
		return observed, err
	}}
	merger := happyMerger()
	mergeObserve, mergeConfirm := merger.observe, merger.confirm
	merger.observe = func(work factory.MergeWork) (factory.MergeObservation, error) {
		if reviewer.events[work.HeadOID] != "APPROVED" {
			return factory.MergeObservation{}, errors.New("native fixture has no current independent approval")
		}
		observation, err := mergeObserve(work)
		observation.NativeRev, observation.Checks.NativeRev = revision.Load(), revision.Load()
		return observation, err
	}
	merger.confirm = func(work factory.MergeWork) (factory.MergeConfirmation, error) {
		confirmation, err := mergeConfirm(work)
		confirmation.NativeRev = revision.Load()
		if err == nil {
			inputs.closed.Store(confirmation.ClosedUnix)
		}
		return confirmation, err
	}
	fx.coord.Merges = merger
	diagnose := func() {
		_ = fx.coord.Close()
		owner, _ := fx.db.LatestIssueAssignment(ctx, 7, 31)
		publication, _ := fx.db.PublicationByAssignment(ctx, owner.ID)
		merge, _ := fx.db.MergeByPublication(ctx, publication.ID)
		checks, _ := fx.db.CheckAssessment(ctx, 7, publication.PRNumber)
		dependent, _ := fx.db.IssueControl(ctx, 7, 32)
		t.Logf("serving state: assignment=%s/%s publication=%s/%s corrections=%d checks=%s/%s merge=%s/%s dependent=%s/%s", owner.Stage, owner.Reason, publication.Stage, publication.Reason, len(publication.Corrections), checks.Verdict, checks.Reason, merge.Stage, merge.Reason, dependent.Readiness, dependent.Reason)
		for _, launch := range host.launches {
			t.Logf("observed launch: role=%s source=%s", launch.Run.Role, launch.Run.SourceCommit)
		}
		for _, review := range reviewer.submitted {
			t.Logf("observed review: event=%s head=%s", review.Event, review.HeadOID)
		}
	}
	created := time.Now().Add(-time.Minute).Unix()
	for _, issue := range []int64{31, 32} {
		index := strconv.FormatInt(issue, 10)
		title, body := "objective "+index+"; inspect README.md", "body "+index
		view := AcceptanceIssueView{ID: strconv.FormatInt(issue+100, 10), Index: index, CreatedUnix: created, PosterID: "5", TitleOccurrence: factory.OriginalOccurrence, LifecycleOccurrence: factory.OriginalOccurrence, TitleDigest: dispatchDigest(title), ContentDigest: dispatchDigest(body), Visible: true}
		fx.reads.evidence["7/"+index] = AcceptanceEvidence{Revision: 41, Issue: view}
		fx.seed.reads.inputs["7/"+index] = DispatchInputs{Issue: DispatchIssue{Title: title, Body: body, TitleOccurrence: factory.OriginalOccurrence, LifecycleOccurrence: factory.OriginalOccurrence, TitleDigest: view.TitleDigest, ContentDigest: view.ContentDigest, Visible: true}, Tip: fx.seed.tip, TipRef: "refs/heads/main", Revision: 41}
	}
	main := fx.reads.evidence["7/31"].Issue
	dependentView := fx.reads.evidence["7/32"]
	dependentView.Dependencies = []AcceptanceEdge{{Occurrence: "21", DependsOn: "131", Visible: true}}
	fx.reads.evidence["7/32"] = dependentView
	if err = fx.coord.Start(ctx, filepath.Join(t.TempDir(), "coordinator.lock")); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = fx.coord.Close() })
	_, _, err = fx.coord.ObserveIssueEvent(ctx, IntakeHint{Delivery: "serving-original-creation", Repository: 7, Issue: 31, Created: true, Creator: 5, CreatorWrite: true, CreationVerified: true, NativeIssueID: 131, CreatedUnix: created, TitleDigest: main.TitleDigest, ContentDigest: main.ContentDigest})
	if err != nil && !errors.Is(err, ErrReadinessPassPending) {
		t.Fatal(err)
	}
	head, err := fx.db.AcceptanceHead(ctx, 7, 31)
	if err != nil || head != factory.InitialAcceptanceID(7, "131") {
		t.Fatalf("normal creation did not admit exact initial inputs: head=%q error=%v", head, err)
	}
	dependent := factory.Acceptance{ID: "d" + strings.Repeat("f", 24), Repository: 7, IssueIndex: "32", Approver: 5, NativeRev: revision.Load(), TitleOccurrence: factory.OriginalOccurrence, TitleDigest: dependentView.Issue.TitleDigest, ContentDigest: dependentView.Issue.ContentDigest, Prerequisites: []factory.AcceptedPrerequisite{{Occurrence: "21", DependsOn: "131", EndpointRepo: 7, EndpointIssue: 31, Outcome: factory.PrereqCode, PrereqAcceptance: head}}}
	if _, err = fx.coord.AdmitAcceptance(ctx, factory.NewID(), "native:5", dependent); err != nil {
		t.Fatal(err)
	}
	// Supply only later native CI truth. No further intake or stage call
	// wakes the coordinator; its serving owner must complete the journey.
	select {
	case <-checksPending:
	case <-time.After(45 * time.Second):
		diagnose()
		t.Fatal("serving owner did not produce correction and observe pending exact-head CI")
	}
	if _, err = fx.db.IssueMergeCompletion(ctx, 7, 31); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("pending corrected-head CI acquired completion: %v", err)
	}
	correctedChecksReady.Store(true)
	deadline := time.Now().Add(35 * time.Second)
	for {
		select {
		case launch := <-host.started:
			owner, readErr := fx.db.AssignmentByRun(ctx, launch.Run.ID)
			if readErr != nil {
				t.Fatal(readErr)
			}
			if owner.Issue == 32 {
				if err = fx.coord.Close(); err != nil {
					t.Fatal(err)
				}
				completion, readErr := fx.db.IssueMergeCompletion(ctx, 7, 31)
				if readErr != nil || completion.MergedCommit != strings.Repeat("3", 40) {
					t.Fatalf("dependent ran without exact confirmed corrected merge: %+v %v", completion, readErr)
				}
				if len(host.launches) != 5 || len(publisher.pushes) != 2 || len(publisher.creates) != 1 || len(reviewer.submitted) != 2 || len(merger.submits) != 1 || len(merger.confirms) != 1 {
					t.Fatalf("unexpected serving effects: runs=%d pushes=%d PRs=%d reviews=%d merges=%d confirmations=%d", len(host.launches), len(publisher.pushes), len(publisher.creates), len(reviewer.submitted), len(merger.submits), len(merger.confirms))
				}
				if reviewer.submitted[0].Event != "REQUEST_CHANGES" || reviewer.submitted[1].Event != "APPROVED" || reviewer.submitted[1].HeadOID != completion.MergedCommit || reviewer.submitted[1].ActorID == reviewer.submitted[1].PRAuthorID {
					t.Fatal("correction did not require independent exact-head approval")
				}
				return
			}
		default:
		}
		if time.Now().After(deadline) {
			diagnose()
			t.Fatal("serving owner did not confirm merge and pick up the accepted dependent")
		}
		time.Sleep(20 * time.Millisecond)
	}
}
