package control

import (
	"context"
	"encoding/json"
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

// completionPlan builds the phased investigate report for one serving
// assignment: the executable plan over the assignment's exact attempt
// root, source and preparation. The phased admission world refuses
// legacy candidate payloads on investigate runs, so the host reports
// the plan the implement child advances from.
func completionPlan(owner factory.Assignment) (factory.InvestigationPlan, error) {
	plan := factory.InvestigationPlan{
		ID: factory.NewID(), AttemptRoot: owner.AttemptRoot,
		ScopeDigest: dispatchDigest("serving scope"), InputDigest: dispatchDigest("serving input"),
		SourceCommit: owner.SourceCommit, IntegrationCommit: owner.SourceCommit,
		Preparation:        owner.Preparation,
		Approach:           "inspect the selected source, then implement behind checks",
		AcceptanceCoverage: []string{"accept-one"},
		Tasks: []factory.PlanTask{{
			ID: "task-one", Objective: "objective 31; inspect README.md",
			Outputs: []string{"notes"}, AcceptanceRefs: []string{"accept-one"},
			Lane: "lane-one", OwnedPaths: []string{"README.md"},
		}},
		Lanes: []factory.PlanLane{{ID: "lane-one", TaskIDs: []string{"task-one"}}},
	}
	digest, err := factory.PlanDigest(plan)
	if err != nil {
		return factory.InvestigationPlan{}, err
	}
	plan.Digest = digest
	if err := plan.Executable(); err != nil {
		return factory.InvestigationPlan{}, err
	}
	return plan, nil
}

// fencedPhaseOutput marshals one phased report inside the result fence
// the settlement boundary parses.
func fencedPhaseOutput(result factory.PhaseResult) (string, error) {
	payload, err := json.Marshal(result)
	if err != nil {
		return "", err
	}
	return "work\n```" + factory.ResultFence + "\n" + string(payload) + "\n```\n", nil
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
	case owner.Phase == project.ExecutionInvestigate:
		plan, planErr := completionPlan(owner)
		if planErr != nil {
			return project.FactoryState{}, planErr
		}
		output, marshalErr := fencedPhaseOutput(factory.PhaseResult{
			Version: factory.PhaseResultVersion, Phase: project.ExecutionInvestigate, Status: "completed",
			Summary: "plan ready", Output: &factory.PhaseOutput{Plan: &plan},
		})
		if marshalErr != nil {
			return project.FactoryState{}, marshalErr
		}
		state.Output = output
	case owner.Phase == project.ExecutionImplement:
		candidate := strings.Repeat("2", 40)
		if in.Run.SourceCommit == candidate {
			candidate = strings.Repeat("3", 40)
		}
		output, marshalErr := fencedPhaseOutput(factory.PhaseResult{
			Version: factory.PhaseResultVersion, Phase: project.ExecutionImplement, Status: "completed",
			Summary: "implemented", Output: &factory.PhaseOutput{Implementation: &factory.ImplementationReceipt{Candidate: candidate}},
		})
		if marshalErr != nil {
			return project.FactoryState{}, marshalErr
		}
		state.Output = output
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
	mu        sync.Mutex
	merged    atomic.Bool
	closed    atomic.Int64
	closing   atomic.Int64
	lifecycle atomic.Int64
}

func (r *completionInputs) ReadAcceptanceEvidence(ctx context.Context, repository, issue string, comments []string) (AcceptanceEvidence, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	evidence, err := r.progressInputs.ReadAcceptanceEvidence(ctx, repository, issue, comments)
	if issue == "31" && r.lifecycle.Load() > 0 {
		evidence.Issue.Closed, evidence.Issue.ClosedUnix = r.closed.Load() > 0, r.closed.Load()
		evidence.Issue.ClosingOccurrence = ""
		if r.closing.Load() > 0 {
			evidence.Issue.ClosingOccurrence = strconv.FormatInt(r.closing.Load(), 10)
		}
		evidence.Issue.LifecycleOccurrence = strconv.FormatInt(r.lifecycle.Load(), 10)
	}
	return evidence, err
}

func (r *completionInputs) ReadDispatchInputs(ctx context.Context, repository, issue string, comments []string, target string) (DispatchInputs, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	inputs, err := r.progressInputs.ReadDispatchInputs(ctx, repository, issue, comments, target)
	if r.merged.Load() {
		inputs.Tip = strings.Repeat("3", 40)
	}
	if issue == "31" && r.lifecycle.Load() > 0 {
		inputs.Issue.Closed, inputs.Issue.ClosedUnix = r.closed.Load() > 0, r.closed.Load()
		inputs.Issue.ClosingOccurrence = ""
		if r.closing.Load() > 0 {
			inputs.Issue.ClosingOccurrence = strconv.FormatInt(r.closing.Load(), 10)
		}
		inputs.Issue.LifecycleOccurrence = strconv.FormatInt(r.lifecycle.Load(), 10)
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
	servingCorrectionAndDependent(t, "")
}

func TestServingCoordinatorDoesNotReuseOriginalClosureAfterReopen(t *testing.T) {
	for _, lifecycle := range []string{"reopened", "reclosed"} {
		t.Run(lifecycle, func(t *testing.T) {
			t.Parallel()
			servingCorrectionAndDependent(t, lifecycle)
		})
	}
}

func servingCorrectionAndDependent(t *testing.T, laterLifecycle string, configure ...func(*Coordinator)) {
	t.Helper()
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
	mergeObserve, mergeSubmit, mergeConfirm := merger.observe, merger.submit, merger.confirm
	merger.observe = func(work factory.MergeWork) (factory.MergeObservation, error) {
		if reviewer.events[work.HeadOID] != "APPROVED" {
			return factory.MergeObservation{}, errors.New("native fixture has no current independent approval")
		}
		observation, err := mergeObserve(work)
		observation.NativeRev, observation.Checks.NativeRev = revision.Load(), revision.Load()
		return observation, err
	}
	merger.submit = func(work factory.MergeWork) (factory.OperationOutcome, error) {
		if work.Issue != 31 || len(publisher.creates) != 1 || !strings.Contains(publisher.creates[0].PRBody, "\nCloses #31\n") {
			return factory.OperationOutcome{}, errors.New("native fixture has no exact original issue closing reference")
		}
		outcome, err := mergeSubmit(work)
		if err == nil {
			// Native merge bookkeeping closes the original issue named by
			// the PR, before any coordinator completion is recorded.
			inputs.mu.Lock()
			inputs.merged.Store(true)
			inputs.closed.Store(time.Now().Unix())
			inputs.closing.Store(501)
			inputs.lifecycle.Store(501)
			inputs.mu.Unlock()
		}
		return outcome, err
	}
	merger.confirm = func(work factory.MergeWork) (factory.MergeConfirmation, error) {
		confirmation, err := mergeConfirm(work)
		confirmation.NativeRev = revision.Load()
		if err == nil {
			inputs.mu.Lock()
			confirmation.OriginalClosed = inputs.closed.Load() > 0
			confirmation.OriginalClosedUnix = inputs.closed.Load()
			confirmation.OriginalClosingOccurrence = strconv.FormatInt(inputs.closing.Load(), 10)
			// Supply a later native lifecycle after this exact completion
			// snapshot, before the serving owner reassesses dependants.
			if laterLifecycle == "reopened" {
				inputs.closed.Store(0)
				inputs.closing.Store(0)
				inputs.lifecycle.Store(502)
				revision.Add(1)
			} else if laterLifecycle == "reclosed" {
				// Equal timestamps cannot substitute for occurrence identity.
				inputs.closing.Store(503)
				inputs.lifecycle.Store(503)
				revision.Add(1)
			}
			inputs.mu.Unlock()
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
		t.Logf("DEBUG owner: phase=%q result=%+v", owner.Phase, owner.Result)
		if headID, rev, err := fx.db.PlanHead(ctx, owner.AttemptRoot); err != nil {
			t.Logf("DEBUG planhead err=%v", err)
		} else {
			t.Logf("DEBUG planhead=%s rev=%d", headID, rev)
			if receipts, err := fx.db.PlanTaskReceipts(ctx, owner.AttemptRoot, headID); err != nil {
				t.Logf("DEBUG receipts err=%v", err)
			} else {
				for _, r := range receipts {
					t.Logf("DEBUG receipt task=%s state=%s commit=%s rev=%d", r.TaskID, r.State, r.Commit, r.Revision)
				}
			}
		}
		if dbgHead, err := fx.db.AcceptanceHead(ctx, 7, 31); err == nil {
			if childID, _, ok := advanceChildID(owner, dbgHead); ok {
				if child, err := fx.db.Assignment(ctx, childID); err != nil {
					t.Logf("DEBUG child err=%v", err)
				} else {
					t.Logf("DEBUG child: phase=%q stage=%s outcome=%s reason=%s result=%+v", child.Phase, child.Stage, child.Outcome, child.Reason, child.Result)
				}
			}
		}
		if queued, err := fx.db.PublishableAssignments(ctx, 10); err == nil {
			for _, q := range queued {
				t.Logf("DEBUG publishable issue=%d phase=%q stage=%s", q.Issue, q.Phase, q.Stage)
			}
		}
		head, _ := fx.db.AcceptanceHead(ctx, 7, 31)
		childID, plan, ok := advanceChildID(owner, head)
		t.Logf("DEBUG advance ok=%v child=%s hasplan=%v subject=%+v outcome=%s reason=%s head=%s acpt=%s", ok, childID, plan != nil, owner.Subject, owner.Outcome, owner.Reason, head, owner.Acceptance)
		if plan != nil {
			t.Logf("DEBUG plan root=%s source=%s prep=%s ownerRoot=%s ownerSource=%s ownerPrep=%s", plan.AttemptRoot, plan.SourceCommit, plan.Preparation, owner.AttemptRoot, owner.SourceCommit, owner.Preparation)
		}
		report := fx.coord.Dispatch(ctx)
		for _, w := range report.Waits {
			t.Logf("DEBUG dispatch wait repo=%d issue=%d reason=%s detail=%s", w.Repository, w.Issue, w.Reason, w.Detail)
		}
		for _, e := range report.Errors {
			t.Logf("DEBUG dispatch error repo=%d issue=%d reason=%s detail=%s", e.Repository, e.Issue, e.Reason, e.Detail)
		}
		for _, l := range report.Launched {
			t.Logf("DEBUG dispatch launched issue=%d phase=%s assignment=%s", l.Issue, l.Phase, l.AssignmentID)
		}
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
	for _, apply := range configure {
		apply(fx.coord)
	}
	// The D13 fence (FW31) holds same-repository admissions behind the
	// merged completion until its funded reassessment decision releases.
	// The serving journey scripts that decision deterministically once
	// the merge completion lands (see wireD13Continue below): sealed
	// substance judges continue exactly like the completion supplement,
	// so the base variant picks up its dependent only after the release.
	// Planning stays unwired until then: an early Planning without a
	// recorded plan would hold every dispatch behind plan_not_current.
	// Scenarios that wire their own Planning (prompt-entry journey) keep
	// it; the fence enumerator defaults to the scripted pages either way.
	if fx.coord.CompletionEnumerator == nil {
		fx.coord.CompletionEnumerator = twoPageEnumerator(t)
	}
	// wireD13Continue funds the reassessment decision after the merge
	// completion lands. It runs once per journey: the continue chooser
	// judges the sealed substance and continues, opening the fence for
	// the dependent. Reopened/reclosed variants wire it too — their
	// dependents stay blocked on prereq truth regardless.
	wireD13Continue := func() {
		if fx.coord.Planning != nil {
			return
		}
		if _, err := fx.db.IssueMergeCompletion(ctx, 7, 31); err != nil {
			return
		}
		if fx.db.PriorityCallBudget < 100 {
			fx.db.PriorityCallBudget = 100
		}
		fx.coord.Planning = &PlanningJudgments{Journal: fx.db, Funding: fx.db, Chooser: &continueChooser{}}
	}
	// recordServingCheckpoints supplies the FW15 checkpoint evidence the
	// phased publication gate requires: the native executor finished the
	// planned task and reported its checkpoint commit, so the fixture
	// walks the receipt from pending to completed under the reported
	// plan head. The gate itself still enforces completeness — without
	// this evidence the journey waits at plan_tasks_incomplete instead
	// of publishing.
	recordServingCheckpoints := func() {
		// LatestIssueAssignment only names root attempts, so resolve
		// the advanced implement child through its deterministic
		// identity instead.
		root, err := fx.db.LatestIssueAssignment(ctx, 7, 31)
		if err != nil {
			return
		}
		head, err := fx.db.AcceptanceHead(ctx, 7, 31)
		if err != nil {
			return
		}
		childID, _, ok := advanceChildID(root, head)
		if !ok {
			return
		}
		latest, err := fx.db.Assignment(ctx, childID)
		if err != nil || latest.Phase != project.ExecutionImplement {
			return
		}
		if latest.Result == nil || !latest.Result.Reported || latest.Result.Candidate == "" {
			return
		}
		headID, _, err := fx.db.PlanHead(ctx, latest.AttemptRoot)
		if err != nil {
			return
		}
		if receipt, err := fx.db.PlanTaskReceipt(ctx, latest.AttemptRoot, headID, "task-one"); err == nil && receipt.State == store.PlanTaskCompleted {
			return
		}
		candidate := latest.Result.Candidate
		pending := store.PlanTaskReceipt{AttemptRoot: latest.AttemptRoot, PlanID: headID, TaskID: "task-one", State: store.PlanTaskPending}
		running := pending
		running.Revision, running.State, running.Child = 0, store.PlanTaskRunning, "child-task-one"
		ready := running
		ready.Revision, ready.State, ready.OutputDigest = 0, store.PlanTaskOutputReady, dispatchDigest("serving task output")
		// The completed checkpoint binds the reported candidate as the
		// task's commit: the receipt accounts for exactly what published.
		completed := ready
		completed.Revision, completed.State, completed.Commit, completed.Tree = 1, store.PlanTaskCompleted, candidate, candidate
		completed.Reason = "planned task finished with its checkpoint commit"
		_ = fx.db.SavePlanTaskReceipt(ctx, pending)
		_ = fx.db.SavePlanTaskReceipt(ctx, running)
		ready.Revision = 1
		_ = fx.db.SavePlanTaskReceipt(ctx, ready)
		completed.Revision = 2
		_ = fx.db.SavePlanTaskReceipt(ctx, completed)
	}
	// authorizeServingMerge binds the enforced merge gate (FW22) to the
	// journey's admitted correction: once the open row carries the
	// corrected candidate, the ready first-generation fixture auth
	// answers it through the same decision gate production enforces.
	// Earlier candidates stay unauthorized so no superseded head merges.
	authorizeServingMerge := func() {
		merges, err := fx.db.OpenMerges(ctx, 16)
		if err != nil {
			return
		}
		for _, m := range merges {
			if m.Stage != factory.MergeOpen || m.Authorization.Bound() {
				continue
			}
			if m.HeadOID != strings.Repeat("3", 40) {
				continue
			}
			_ = fx.coord.AuthorizeMerge(ctx, m.PublicationID, gateTestBinding(t, m))
		}
	}
	if err = fx.coord.Start(ctx, filepath.Join(t.TempDir(), "coordinator.lock")); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = fx.coord.Close() })
	if fx.coord.Graph == nil {
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
	}
	// Supply only later native CI truth. No further intake or stage call
	// wakes the coordinator; its serving owner must complete the journey.
	// The fixture pumps run here too: checkpoint evidence unblocks
	// publication, which unblocks review, correction and the pending
	// exact-head CI this wait watches for.
	waitDeadline := time.Now().Add(55 * time.Second)
	waited := false
	for !waited {
		authorizeServingMerge()
		recordServingCheckpoints()
		wireD13Continue()
		select {
		case <-checksPending:
			waited = true
		case <-time.After(20 * time.Millisecond):
		}
		if !waited && time.Now().After(waitDeadline) {
			diagnose()
			t.Fatal("serving owner did not produce correction and observe pending exact-head CI")
		}
	}
	if _, err = fx.db.IssueMergeCompletion(ctx, 7, 31); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("pending corrected-head CI acquired completion: %v", err)
	}
	correctedChecksReady.Store(true)
	deadline := time.Now().Add(55 * time.Second)
	for {
		authorizeServingMerge()
		recordServingCheckpoints()
		wireD13Continue()
		select {
		case launch := <-host.started:
			owner, readErr := fx.db.AssignmentByRun(ctx, launch.Run.ID)
			if readErr != nil {
				t.Fatal(readErr)
			}
			if owner.Issue == 32 {
				if laterLifecycle != "" {
					diagnose()
					t.Fatalf("dependent launched after original issue was %s", laterLifecycle)
				}
				if err = fx.coord.Close(); err != nil {
					t.Fatal(err)
				}
				completion, readErr := fx.db.IssueMergeCompletion(ctx, 7, 31)
				if readErr != nil || completion.MergedCommit != strings.Repeat("3", 40) || completion.OriginalClosingOccurrence != "501" || completion.OriginalClosedUnix <= 0 {
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
		if laterLifecycle != "" {
			// Join an ordinary owner pass that recorded completion and
			// assessed the newer native endpoint. No manual pass wakes it.
			c := fx.coord
			c.progressMu.Lock()
			completion, completionErr := fx.db.IssueMergeCompletion(ctx, 7, 31)
			control, controlErr := fx.db.IssueControl(ctx, 7, 32)
			latest, assignmentErr := fx.db.LatestIssueAssignment(ctx, 7, 32)
			blocked := completionErr == nil && controlErr == nil && control.NativeRev == revision.Load() && control.Readiness == factory.ReadinessBlocked && control.Reason == factory.BlockerCodePending
			c.progressMu.Unlock()
			if blocked {
				if completion.OriginalClosingOccurrence != "501" || completion.OriginalClosedUnix <= 0 {
					t.Fatalf("recorded completion lost its native closing occurrence: %+v", completion)
				}
				if !errors.Is(assignmentErr, store.ErrNotFound) {
					t.Fatalf("blocked dependent acquired assignment: %+v %v", latest, assignmentErr)
				}
				if err = fx.coord.Close(); err != nil {
					t.Fatal(err)
				}
				if len(host.launches) != 4 || len(merger.submits) != 1 || len(merger.confirms) != 1 {
					t.Fatalf("original lifecycle change caused duplicate serving effects: runs=%d merges=%d confirmations=%d", len(host.launches), len(merger.submits), len(merger.confirms))
				}
				return
			}
		}
		if time.Now().After(deadline) {
			diagnose()
			t.Fatal("serving owner did not confirm merge and pick up the accepted dependent")
		}
		time.Sleep(20 * time.Millisecond)
	}
}
