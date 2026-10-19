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
	// approveFirst skips the findings round: the first review approves
	// the initial candidate outright (clean-first-review variant).
	approveFirst bool
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
			Lane: "lane-one", OwnedPaths: []string{"README.md"}, CheckIDs: []string{"check-one"},
		}},
		Lanes: []factory.PlanLane{{ID: "lane-one", TaskIDs: []string{"task-one"}}},
		// The staged check is controller-selected verification: a
		// checkless plan cannot derive the singleton correction plan,
		// so the journey plan carries its check exactly like the
		// phased producer seed.
		Checks: []factory.CheckSpec{{
			ID: "check-one", Argv: []string{"make", "check"}, Cwd: ".",
			TimeoutSeconds: 60, Scope: factory.CheckScopeTask, Required: true,
		}},
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

// completionFindingKey is the journey's single stable actionable finding
// key: the first review reports it, the corrector answers it, and the
// re-review verifies it. One key across the round, never rewritten.
func completionFindingKey() string { return correctionTestKey(1) }

// completionFinding builds the journey's actionable code finding with its
// exact origin run. Every round must echo the identical struct: the
// canonical carried pipeline refuses rewritten content on a known key, so
// the corrector and re-reviewer resolve the round-one origin run from the
// recorded publication instead of stamping their own.
func completionFinding(originRun string) factory.Finding {
	return factory.Finding{
		Key: completionFindingKey(), OriginRun: originRun,
		Severity: factory.FindingSeverityMajor, Defect: "empty input needs a regression test",
		Evidence: "initial candidate lacks a regression test", RequiredOutcome: "add the regression test",
		FixKind: factory.FindingFixCode, Actionable: true,
	}
}

// completionRoundOneRun resolves the round-one reviewer run behind one
// recorded cycle: the earliest round's reviewer assignment names it. The
// corrector and re-reviewer need its identity to echo the finding
// byte-identically.
func (h *completionHost) completionRoundOneRun(ctx context.Context, p factory.Publication) (string, error) {
	earliest := -1
	reviewer := ""
	for _, recorded := range p.ReviewCycles {
		if recorded.PublicationID != p.ID {
			continue
		}
		if earliest < 0 || recorded.Round < earliest {
			earliest, reviewer = recorded.Round, recorded.ReviewerAssignment
		}
	}
	if reviewer == "" {
		return "", errors.New("journey names no round-one reviewer")
	}
	assignment, err := h.db.Assignment(ctx, reviewer)
	if err != nil {
		return "", err
	}
	if assignment.Run == "" {
		return "", errors.New("round-one reviewer names no run")
	}
	return assignment.Run, nil
}

// typedReviewOutput reports one phased reviewer run's typed verdict over
// the run's recorded cycle/head/base: a findings verdict with the
// journey's actionable finding when the first review meets the defective
// initial candidate, otherwise the explicit clear that verifies the
// carried set. Pre-phase reviewer runs keep the legacy string report;
// this entry only serves phased review children.
func (h *completionHost) typedReviewOutput(ctx context.Context, child factory.Assignment) (string, error) {
	p, err := h.db.PublicationByAssignment(ctx, child.PublicationAssignment)
	if err != nil {
		return "", err
	}
	var cycle factory.ReviewCycle
	found := false
	for _, recorded := range p.ReviewCycles {
		if recorded.ReviewerAssignment == child.ID {
			cycle, found = recorded, true
			break
		}
	}
	if !found {
		return "", errors.New("typed reviewer names no recorded cycle")
	}
	if !h.approveFirst && child.SourceCommit == strings.Repeat("2", 40) {
		return fencedPhaseOutput(factory.PhaseResult{
			Version: factory.PhaseResultVersion, Phase: project.ExecutionReview, Status: "completed",
			Summary: "reviewed",
			Output: &factory.PhaseOutput{Review: &factory.ReviewCycleReport{
				CycleID: cycle.ID, NewFindings: []factory.Finding{completionFinding(child.Run)},
			}},
		})
	}
	var carried []factory.Finding
	var assessments []factory.FindingAssessment
	if !h.approveFirst {
		origin, err := h.completionRoundOneRun(ctx, p)
		if err != nil {
			return "", err
		}
		carried = []factory.Finding{completionFinding(origin)}
		assessments = []factory.FindingAssessment{{
			Key: completionFindingKey(), Status: factory.FindingVerified,
			Evidence: "corrected candidate carries the regression test",
		}}
	}
	content, err := factory.FindingContentDigest(carried)
	if err != nil {
		return "", err
	}
	coverage, err := factory.ClearCoverageDigest(cycle.Head, cycle.BaseOID, content)
	if err != nil {
		return "", err
	}
	return fencedPhaseOutput(factory.PhaseResult{
		Version: factory.PhaseResultVersion, Phase: project.ExecutionReview, Status: "completed",
		Summary: "verdict recorded",
		Output: &factory.PhaseOutput{Clear: &factory.ClearReviewReport{
			CycleID: cycle.ID, Conclusion: factory.ClearReviewConclusion,
			Head: cycle.Head, BaseOID: cycle.BaseOID, CoverageDigest: coverage,
			Assessments: assessments,
		}},
	})
}

// typedCorrectionOutput reports one phased corrector run's typed correction
// over its recorded cycle: the journey finding echoed byte-identically
// with a fixed disposition, in both envelopes the wire consumes (the typed
// phase result for settlement and the fenced correction report for the
// canonical report gate). The report's set digest is the finding-set
// digest its own Validate demands; whether that answers the recorded
// cycle's digest domain belongs to admission, never the fixture.
func (h *completionHost) typedCorrectionOutput(ctx context.Context, child factory.Assignment) (string, error) {
	p, err := h.db.PublicationByAssignment(ctx, child.PublicationAssignment)
	if err != nil {
		return "", err
	}
	known := false
	for _, recorded := range p.ReviewCycles {
		if recorded.CorrectionAssignment == child.ID {
			known = true
			break
		}
	}
	if !known {
		return "", errors.New("typed corrector names no recorded cycle")
	}
	origin, err := h.completionRoundOneRun(ctx, p)
	if err != nil {
		return "", err
	}
	finding := completionFinding(origin)
	keys := []string{finding.Key}
	setDigest, err := factory.FindingSetDigest(keys)
	if err != nil {
		return "", err
	}
	corrected := strings.Repeat("3", 40)
	report := factory.CorrectionReport{
		SetDigest: setDigest, NewCandidate: corrected, Findings: []factory.Finding{finding},
		Dispositions: []factory.CorrectionDisposition{{Key: finding.Key, Disposition: factory.DispositionFixed}},
		Evidence: []factory.CorrectionFixEvidence{{
			Key: finding.Key, ChangedPaths: []string{"README.md"}, Commits: []string{corrected},
		}},
	}
	output, err := fencedPhaseOutput(factory.PhaseResult{
		Version: factory.PhaseResultVersion, Phase: project.ExecutionCorrect, Status: "completed",
		Summary: "corrected",
		Output:  &factory.PhaseOutput{Correction: &report},
	})
	if err != nil {
		return "", err
	}
	payload, err := json.Marshal(report)
	if err != nil {
		return "", err
	}
	return output + "```" + factory.CorrectionReportFence + "\n" + string(payload) + "\n```\n", nil
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
		if owner.Phase == project.ExecutionReview {
			// Phased review children report the typed verdict the
			// phased wire consumes; legacy string reports refuse
			// on phased assignments.
			output, typedErr := h.typedReviewOutput(ctx, owner)
			if typedErr != nil {
				return project.FactoryState{}, typedErr
			}
			state.Output = output
			break
		}
		verdict, body, findings := "approve", "corrected candidate is sound", "[]"
		if !h.approveFirst && in.Run.SourceCommit == strings.Repeat("2", 40) {
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
	case owner.Phase == project.ExecutionCorrect:
		// Phased corrector children report the typed correction the
		// phased wire consumes; legacy harness shapes refuse on
		// phased assignments.
		output, typedErr := h.typedCorrectionOutput(ctx, owner)
		if typedErr != nil {
			return project.FactoryState{}, typedErr
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

// servingJourneyChooser is the deterministic adapter behind the composed
// serving journey: planning inputs take the serving planner's answers so
// dispatch keeps its recorded-plan shape, while the D13 reassessment
// decision judges sealed substance and continues. Both halves are
// deterministic; no live Jev is consulted.
type servingJourneyChooser struct {
	planner *servingPlanChooser
	decider continueChooser
}

func (c *servingJourneyChooser) Select(ctx context.Context, state any, question string, choices map[string]any) (factory.PriorityAnswer, error) {
	if _, ok := state.(factory.PlanningInput); ok {
		planner := c.planner
		if planner == nil {
			planner = &servingPlanChooser{}
		}
		return planner.Select(ctx, state, question, choices)
	}
	return c.decider.Select(ctx, state, question, choices)
}

type completionInputs struct {
	progressInputs
	mu     sync.Mutex
	merged atomic.Bool
	// mergedTip is the actual merged commit once the native merge lands;
	// post-merge reads report it as the branch tip, exactly like native
	// truth advancing past the merge. It must never be a hardcoded
	// candidate: each journey variant merges its own head.
	mergedTip string
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
		inputs.Tip = r.mergedTip
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

// TestServingCleanFirstReviewMergesWithoutCorrectionAndPicksUpDependent
// (T32 clean-first-review variant): the first independent review approves
// a finding-free candidate, so no correction runs; the explicit clear
// journals clean_review delivery acceptance with zero delivery calls, the
// union-bound merge authorization carries it, and the dependent still
// picks up only after the funded D13 continue releases the fence.
func TestServingCleanFirstReviewMergesWithoutCorrectionAndPicksUpDependent(t *testing.T) {
	servingJourney(t, servingVariant{approveFirst: true, delivery: factory.AcceptanceCleanReview})
}

// TestServingJevAcceptanceAfterFixMergesAndPicksUpDependent (T32
// direct-after-fix variant): review finds a defect, correction fixes it,
// re-review approves, and the funded ready_to_merge decision journals
// jev_acceptance; the union-bound merge authorization carries the funded
// decision, and the dependent picks up after the D13 continue release.
func TestServingJevAcceptanceAfterFixMergesAndPicksUpDependent(t *testing.T) {
	servingJourney(t, servingVariant{delivery: factory.AcceptanceJevAcceptance})
}

// admitServingDeliveryAcceptance journals delivery acceptance on the
// journey publication behind one open merge row and returns the
// union-bound authorization for it. kind selects clean_review (zero
// delivery calls over the approving round's explicit clear) or
// jev_acceptance (the deterministic funded ready_to_merge decision over
// the same evidence). The round's reported findings are empty in every
// journey variant — request-changes findings are fixed by correction and
// verified by the approving re-review — so the acceptance binds the
// exact empty set with its coverage. Scope binds the admitted plan,
// checks bind the recorded assessment, and policy binds the repository
// revision: every digest names persisted evidence, never fixture prose.
// A row whose publication already journals the round's acceptance reuses
// it, so repeated polls never duplicate the journal.
func admitServingDeliveryAcceptance(t *testing.T, ctx context.Context, coord *Coordinator, db *store.Store, m factory.Merge, kind string) (factory.MergeAuthorization, error) {
	t.Helper()
	var empty factory.MergeAuthorization
	refuse := func(format string, args ...any) (factory.MergeAuthorization, error) {
		return empty, fmt.Errorf(format, args...)
	}
	p, err := db.PublicationByAssignment(ctx, m.AssignmentID)
	if err != nil {
		return refuse("journey publication unreadable: %v", err)
	}
	round := 0
	var cycle factory.ReviewCycle
	for _, recorded := range p.ReviewCycles {
		if recorded.Head == m.HeadOID && recorded.Round >= round {
			round, cycle = recorded.Round, recorded
		}
	}
	if round == 0 {
		return refuse("no recorded review cycle answers merge head %.8s", m.HeadOID)
	}
	reviewer, err := db.Assignment(ctx, cycle.ReviewerAssignment)
	if err != nil {
		return refuse("approving reviewer assignment unreadable: %v", err)
	}
	if reviewer.Result == nil {
		return refuse("approving reviewer recorded no result")
	}
	emptyContent, err := factory.FindingContentDigest(nil)
	if err != nil {
		return refuse("%v", err)
	}
	headID, _, err := db.PlanHead(ctx, cycle.AttemptRoot)
	if err != nil {
		return refuse("admitted plan head unreadable: %v", err)
	}
	plan, err := db.InvestigationPlan(ctx, cycle.AttemptRoot, headID)
	if err != nil {
		return refuse("admitted plan unreadable: %v", err)
	}
	assessment, err := db.CheckAssessment(ctx, p.Repository, p.PRNumber)
	if err != nil {
		return refuse("recorded check assessment unreadable: %v", err)
	}
	if !factory.ValidDigest(assessment.ChecksDigest) {
		return refuse("recorded checks lack their digest: %+v", assessment)
	}
	policy, err := db.RepositoryPolicy(ctx, p.Repository)
	if err != nil {
		return refuse("repository policy unreadable: %v", err)
	}
	policyRevision := strconv.FormatInt(policy.Revision, 10)
	// The funded fingerprint rebuilds deterministically from persisted
	// evidence, so a journaled jev acceptance rebinds its exact decision
	// on every poll without spending another delivery call.
	fpIn := factory.DeliveryFingerprintInput{
		Reviews:     []string{cycle.ReviewerAssignment},
		AttemptRoot: cycle.AttemptRoot, PublicationID: p.ID,
		ScopeDigest: plan.ScopeDigest, FixDigest: emptyContent,
		DiffDigest:  dispatchDigest("serving diff"),
		CheckDigest: assessment.ChecksDigest, TreeDigest: dispatchDigest("serving tree"),
		Head: cycle.Head, Base: cycle.BaseOID, Model: "jev-test",
		Rubric:         "ready_to_merge only with full evidence cover",
		PolicyRevision: policy.Revision, Round: round,
	}
	servingDecision := func() (factory.DeliveryDecision, string, error) {
		if kind != factory.AcceptanceJevAcceptance {
			return factory.DeliveryDecision{}, "", nil
		}
		fingerprint, err := factory.DeliveryFingerprint(fpIn)
		if err != nil {
			return factory.DeliveryDecision{}, "", err
		}
		answer, err := readyDeliveryAnswer(fingerprint)
		if err != nil {
			return factory.DeliveryDecision{}, "", err
		}
		return factory.DeliveryDecision{Answer: answer, Effective: factory.DeliveryEffectiveReady, ReviewsFit: true}, fingerprint, nil
	}
	for _, recorded := range p.DeliveryAcceptances {
		if recorded.Round == round && recorded.Kind == kind {
			decision, _, err := servingDecision()
			if err != nil {
				return refuse("delivery decision unrebuilt: %v", err)
			}
			return servingUnionBinding(t, m, recorded, decision), nil
		}
	}
	var acceptance factory.DeliveryAcceptance
	var journaled factory.Publication
	// Both acceptance kinds bind the controller's canonical carried set:
	// the clear path proves coverage over it, the funded path digests it.
	carried, carriedErr := coord.reviewCarriedFindings(ctx, p, cycle)
	if carriedErr != nil {
		return refuse("carried findings unready: %v", carriedErr)
	}
	switch kind {
	case factory.AcceptanceCleanReview:
		var admitErr error
		journaled, acceptance, admitErr = AdmitClearReview(ClearAdmissionInput{
			Publication: p, Cycle: cycle,
			Reviewer:       reviewer,
			Carried:        carried,
			ScopeDigest:    plan.ScopeDigest,
			PolicyRevision: policyRevision,
			CheckDigest:    assessment.ChecksDigest,
			DiffBase:       p.PRCreate.BaseOID,
		})
		if admitErr != nil {
			return refuse("explicit clear refused: %v", admitErr)
		}
	case factory.AcceptanceJevAcceptance:
		decision, fingerprint, decisionErr := servingDecision()
		if decisionErr != nil {
			return refuse("delivery decision unbuilt: %v", decisionErr)
		}
		var admitErr error
		journaled, acceptance, admitErr = AdmitJevAcceptance(JevAdmissionInput{
			Publication: p, Cycle: cycle, Carried: carried,
			FingerprintInput: fpIn, Fingerprint: fingerprint,
			Decision: decision, DecisionID: factory.NewID(),
			ScopeDigest: plan.ScopeDigest, PolicyRevision: policyRevision,
			CheckDigest: assessment.ChecksDigest, DiffBase: p.PRCreate.BaseOID,
		})
		if admitErr != nil {
			return refuse("funded acceptance refused: %v", admitErr)
		}
	default:
		t.Fatalf("unknown delivery kind %q", kind)
	}
	journaled.Revision = p.Revision + 1
	if err := db.UpdatePublication(ctx, journaled); err != nil {
		return refuse("delivery acceptance unjournaled: %v", err)
	}
	decision, _, err := servingDecision()
	if err != nil {
		return refuse("delivery decision unrebuilt: %v", err)
	}
	return servingUnionBinding(t, m, acceptance, decision), nil
}

// servingUnionBinding carries one journaled delivery acceptance to the
// open merge row: the generation binds the acceptance identity, kind,
// attempt root, origin receipt and review round over the row's exact
// head, base and authority, with the funded decision behind jev kinds
// and none behind clean ones.
func servingUnionBinding(t *testing.T, m factory.Merge, acceptance factory.DeliveryAcceptance, decision factory.DeliveryDecision) factory.MergeAuthorization {
	t.Helper()
	fingerprint := strings.Repeat("e", 64)
	decisionID := ""
	if acceptance.Kind == factory.AcceptanceJevAcceptance {
		fingerprint, decisionID = decision.Answer.Fingerprint, acceptance.DecisionID
	}
	return factory.MergeAuthorization{
		Generation: factory.MergeGeneration{
			Authority: m.Authority, PublicationID: m.PublicationID,
			DecisionID:        decisionID,
			NativeOperationID: factory.MergeGenerationOperationID(m.PublicationID, 1),
			Fingerprint:       fingerprint, Head: m.HeadOID, Base: m.BaseOID, DiffBase: m.BaseOID,
			ChecksDigest:   strings.Repeat("c", 64),
			AcceptanceKind: acceptance.Kind, AcceptanceID: acceptance.ID,
			AttemptRoot: acceptance.AttemptRoot, OriginReceipt: acceptance.OriginReceipt,
			Generation:         1,
			ReviewRound:        acceptance.Round,
			AssessmentRevision: 2, ObservedNativeRev: 12, NotAfter: 1300,
		},
		Decision:             decision,
		ResolutionsConfirmed: true,
	}
}

// servingVariant selects one prompt-entry serving journey shape: the
// later native lifecycle ("" for the base dependent pickup,
// "reopened"/"reclosed" for fence+freshness blocking), whether the first
// review approves outright, and which delivery acceptance the merge
// carries to its authorization.
type servingVariant struct {
	laterLifecycle string
	approveFirst   bool
	delivery       string
}

func servingCorrectionAndDependent(t *testing.T, laterLifecycle string, configure ...func(*Coordinator)) {
	servingJourney(t, servingVariant{laterLifecycle: laterLifecycle}, configure...)
}

func servingJourney(t *testing.T, variant servingVariant, configure ...func(*Coordinator)) {
	laterLifecycle := variant.laterLifecycle
	// The journey's final candidate: the initial implementation without
	// correction, the corrected candidate after it.
	finalHead := strings.Repeat("2", 40)
	delivery := variant.delivery
	if delivery == "" {
		delivery = factory.AcceptanceCleanReview
	}
	if !variant.approveFirst {
		finalHead = strings.Repeat("3", 40)
	}
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
	host := &completionHost{producerFlowHost: &producerFlowHost{fakePublishHost: fx.host}, db: fx.db, states: map[string]project.FactoryState{}, started: make(chan project.FactoryLaunch, 8), approveFirst: variant.approveFirst}
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
		if target.HeadOID == finalHead && !correctedChecksReady.Load() {
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
			// the PR, before any coordinator completion is recorded. The
			// reported tip is the work's exact merged head.
			inputs.mu.Lock()
			inputs.merged.Store(true)
			inputs.mergedTip = work.HeadOID
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
	var authTries, authAdmitted, authBound int
	var lastAuthErr string
	diagnose := func() {
		_ = fx.coord.Close()
		owner, _ := fx.db.LatestIssueAssignment(ctx, 7, 31)
		publication, _ := fx.db.PublicationByAssignment(ctx, owner.ID)
		t.Logf("serving owner: phase=%q stage=%s reason=%s", owner.Phase, owner.Stage, owner.Reason)
		rev, hasReview, reviewPending, reviewRefused := fx.coord.currentReviewReport(ctx, publication)
		t.Logf("serving current review: has=%v pending=%v refused=%v verdict=%q legacy=%v", hasReview, reviewPending, reviewRefused, rev.Verdict, rev.Legacy != nil)
		if rc, ok := fx.coord.currentReviewerForCorrection(ctx, publication, owner); ok {
			t.Logf("serving correction reviewer: %s phase=%q", rc.ID, rc.Phase)
		} else {
			t.Logf("serving correction reviewer: unavailable")
		}
		if allowance, err := fx.db.AttemptAllowance(ctx, 7, 31); err != nil {
			t.Logf("serving allowance: err=%v", err)
		} else {
			t.Logf("serving allowance: corrections=%d reviews=%d active=%v closed=%v", len(allowance.Corrections), len(allowance.Reviews), allowance.Active, allowance.Closed)
		}
		t.Logf("serving owner result: phased=%v", owner.Result != nil && owner.Result.PhaseResult != nil)
		if owner.Result != nil && owner.Result.PhaseResult != nil && owner.Result.PhaseResult.Output != nil {
			out := owner.Result.PhaseResult.Output
			t.Logf("serving owner output: plan=%v implementation=%v review=%v clear=%v correction=%v candidate=%q status=%q",
				out.Plan != nil, out.Implementation != nil, out.Review != nil, out.Clear != nil, out.Correction != nil,
				owner.Result.Candidate, owner.Result.Status)
		}
		freshPub, _ := fx.db.PublicationByAssignment(ctx, owner.ID)
		for _, recorded := range freshPub.ReviewCycles {
			if recorded.CorrectionAssignment == "" {
				continue
			}
			t.Logf("serving correction cycle: round=%d head=%.8s digest=%.16s corrector=%.8s",
				recorded.Round, recorded.Head, recorded.FindingDigest, recorded.CorrectionAssignment)
			if child, err := fx.db.Assignment(ctx, recorded.CorrectionAssignment); err != nil {
				t.Logf("serving corrector: err=%v", err)
			} else if child.Result == nil {
				t.Logf("serving corrector: stage=%s outcome=%s reported=false", child.Stage, child.Outcome)
			} else {
				t.Logf("serving corrector: stage=%s outcome=%s status=%q candidate=%.8s phased=%v",
					child.Stage, child.Outcome, child.Result.Status, child.Result.Candidate, child.Result.PhaseResult != nil)
			}
		}

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
		t.Logf("serving auth: tries=%d admitted=%d bound=%d lastErr=%s", authTries, authAdmitted, authBound, lastAuthErr)
		t.Logf("serving confirms=%d", len(merger.confirms))
		if completion, err := fx.db.IssueMergeCompletion(ctx, 7, 31); err != nil {
			t.Logf("serving completion: err=%v", err)
		} else {
			t.Logf("serving completion: merged=%.8s occurrence=%s", completion.MergedCommit, completion.OriginalClosingOccurrence)
		}
		if cycles, err := fx.db.PendingReassessmentCycles(ctx, 7); err != nil {
			t.Logf("serving cycles: err=%v", err)
		} else {
			for _, c := range cycles {
				t.Logf("serving cycle: id=%.8s state=%s decision=%s wait=%s", c.Cycle.CycleID, c.Cycle.State, c.Cycle.Decision, c.WaitReason)
			}
		}
		if pending, err := fx.db.CompletionPending(ctx, 7); err != nil {
			t.Logf("serving fence: err=%v", err)
		} else {
			t.Logf("serving fence pending=%v", pending)
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
	// The serving journey funds that decision deterministically from the
	// start: planning inputs take the serving planner's answers while the
	// sealed completion judges continue exactly like the completion
	// supplement, so the base variant picks up its dependent only after
	// the release. Scenarios that wire their own Planning (prompt-entry
	// journey, provisional planning) keep it; the fence enumerator
	// defaults to the scripted pages either way.
	if fx.coord.Planning == nil {
		fx.coord.Planning = &PlanningJudgments{Journal: fx.db, Funding: fx.db, Chooser: &servingJourneyChooser{}}
	}
	if fx.coord.CompletionEnumerator == nil {
		fx.coord.CompletionEnumerator = twoPageEnumerator(t)
	}
	// The D13 fence release collects the merged completion through the
	// same drive production uses: native inventory pages plus the store's
	// local capture. Without the local half the cycle holds honestly
	// instead of releasing the dependent.
	if fx.coord.CompletionLocalCapture == nil {
		fx.coord.CompletionLocalCapture = fx.db.CaptureCompletionLocal
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
	// authorizeServingMerge binds the enforced merge gate to the journey's
	// delivery acceptance: once the open row carries the final candidate,
	// the round's acceptance journals on the publication and the
	// union-bound authorization answers the row through the same decision
	// gate production enforces. Earlier candidates stay unauthorized so
	// no superseded head merges.
	authorizeServingMerge := func() {
		merges, err := fx.db.OpenMerges(ctx, 16)
		if err != nil {
			return
		}
		for _, m := range merges {
			if m.Stage != factory.MergeOpen || m.Authorization.Bound() {
				continue
			}
			if m.HeadOID != finalHead {
				continue
			}
			authTries++
			auth, err := admitServingDeliveryAcceptance(t, ctx, fx.coord, fx.db, m, delivery)
			if err != nil {
				lastAuthErr = err.Error()
				continue
			}
			authAdmitted++
			if err := fx.coord.AuthorizeMerge(ctx, m.PublicationID, auth); err != nil {
				lastAuthErr = err.Error()
				continue
			}
			authBound++
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
	// Prompt-entry journeys carry priority tournaments, graph recovery
	// and planning records through every pass, so correction and its
	// exact-head CI land later than in the base variant.
	waitDeadline := time.Now().Add(110 * time.Second)
	waited := false
	for !waited {
		authorizeServingMerge()
		recordServingCheckpoints()
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
	// Merge, the funded D13 release, dependent preparation and pickup
	// span many serving passes; exact-head CI arrives early but the tail
	// needs room past slow preparation confirmations.
	deadline := time.Now().Add(100 * time.Second)
	for {
		authorizeServingMerge()
		recordServingCheckpoints()
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
				if readErr != nil || completion.MergedCommit != finalHead || completion.OriginalClosingOccurrence != "501" || completion.OriginalClosedUnix <= 0 {
					t.Fatalf("dependent ran without exact confirmed merge: %+v %v", completion, readErr)
				}
				// Phased runs: investigate and implement ride separate runs,
				// then review, the approved correction cycle when the first
				// review finds the defect, and the picked-up dependent.
				wantRuns, wantReviews, wantPushes := 6, 2, 2
				if variant.approveFirst {
					// No correction runs and nothing repushes: the single
					// publish carries the approved candidate straight to
					// review and merge.
					wantRuns, wantReviews, wantPushes = 4, 1, 1
				}
				if len(host.launches) != wantRuns || len(publisher.pushes) != wantPushes || len(publisher.creates) != 1 || len(reviewer.submitted) != wantReviews || len(merger.submits) != 1 || len(merger.confirms) != 1 {
					t.Fatalf("unexpected serving effects: runs=%d pushes=%d PRs=%d reviews=%d merges=%d confirmations=%d", len(host.launches), len(publisher.pushes), len(publisher.creates), len(reviewer.submitted), len(merger.submits), len(merger.confirms))
				}
				last := reviewer.submitted[len(reviewer.submitted)-1]
				if last.Event != "APPROVED" || last.HeadOID != completion.MergedCommit || last.ActorID == last.PRAuthorID {
					t.Fatal("merge lacked independent exact-head approval")
				}
				if !variant.approveFirst && reviewer.submitted[0].Event != "REQUEST_CHANGES" {
					t.Fatal("correction did not start from request-changes")
				}
				// The merge authorization carries the variant's delivery
				// acceptance: prove the journaled kind on the publication.
				root, readErr := fx.db.LatestIssueAssignment(ctx, 7, 31)
				if readErr != nil {
					t.Fatal(readErr)
				}
				journaledPub, readErr := fx.db.PublicationByAssignment(ctx, root.ID)
				if readErr != nil {
					t.Fatal(readErr)
				}
				seen := false
				for _, recorded := range journaledPub.DeliveryAcceptances {
					if recorded.Tuple.Head == completion.MergedCommit && recorded.Kind == delivery {
						seen = true
					}
				}
				if !seen {
					t.Fatalf("publication journals no %s acceptance for the merged head: %+v", delivery, journaledPub.DeliveryAcceptances)
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
				// Five runs without the dependent: investigate, implement,
				// review, correction and re-review. The reopened/reclosed
				// original stays blocked on prereq truth behind the fence.
				if len(host.launches) != 5 || len(merger.submits) != 1 || len(merger.confirms) != 1 {
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
