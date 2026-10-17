package control

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"reflect"
	"slices"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
)

const publicationChildLimit = 256

// producePublishedChildren creates at most one reviewer or coder correction
// for each active publication head. First occurrences bind the publication,
// exact head and role (legacy stable IDs); same-head repeats use durable D07
// review-cycle rounds with round-derived fresh IDs, so a new decision on the
// same head creates a fresh reviewer run while replay reuses its run. A later
// pass adopts an existing attempt instead of launching a duplicate.
func (c *Coordinator) producePublishedChildren(ctx context.Context, report *PublishReport) {
	if c.Store == nil || c.Host == nil || c.Broker == nil || c.Publication == nil {
		return
	}
	publications, err := c.Store.ActivePublishedPublications(ctx, publicationChildLimit)
	if err != nil {
		publicationError(report, "", "child_producer_listing_unavailable")
		return
	}
	for _, publication := range publications {
		if err := ctx.Err(); err != nil {
			publicationWait(report, publication.ID, "child_producer_cancelled")
			return
		}
		c.producePublicationChild(ctx, publication, report)
	}
}

func (c *Coordinator) producePublicationChild(ctx context.Context, p factory.Publication, report *PublishReport) {
	if p.Validate() != nil || p.Stage != factory.PublicationPublished || p.WithdrawRequested ||
		p.PRNumber <= 0 || p.PRID <= 0 || p.PRCreate.Work == nil || p.PRCreate.BaseOID == "" {
		return
	}
	owner, err := c.Store.Assignment(ctx, p.AssignmentID)
	if err != nil || owner.Validate() != nil || owner.Role != project.RoleCoder ||
		owner.PublicationAssignment != owner.ID || owner.AttemptRoot == "" || owner.SourceCommit == "" {
		publicationError(report, p.ID, "publication_owner_unavailable")
		return
	}
	// Phased activation: children of a phased (explicit-phase) publication
	// owner enter the typed review/correct wire with their record phase,
	// subject and parent. Children of a pre-phase owner keep the legacy
	// wire, and follow-up children match the evidence they answer, so an
	// upgrade-era publication heals through the leg its current verdict
	// admits instead of stranding.
	phasedOwner := owner.Phase != ""
	latest, err := c.Store.LatestIssueAssignment(ctx, p.Repository, p.Issue)
	if err != nil {
		publicationError(report, p.ID, "issue_assignment_unavailable")
		return
	}
	if latest.AttemptRoot != owner.AttemptRoot || latest.Acceptance != p.Acceptance {
		return
	}
	allowance, err := c.Store.AttemptAllowance(ctx, p.Repository, p.Issue)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return
		}
		publicationError(report, p.ID, "attempt_allowance_unavailable")
		return
	}
	if allowance.RootAssignment != owner.AttemptRoot {
		return
	}
	if allowance.Closed {
		if allowance.Active {
			if stopErr := c.stopCandidatePreparationsForRoot(ctx, p.Repository, p.ProjectID, p.Issue, owner.AttemptRoot); stopErr != nil {
				publicationWait(report, p.ID, "attempt is closed; native preparation retirement remains pending")
				return
			}
		}
		publicationWait(report, p.ID, "attempt is closed; maintainer intervention is required")
		return
	}
	if !allowance.Active {
		return
	}
	if allowance.RemainingSeconds(time.Now()) <= 0 {
		closed, closeErr := c.Store.CloseExpiredAttempt(ctx, owner.ID)
		if closeErr != nil {
			publicationError(report, p.ID, "attempt_allowance_unavailable")
			return
		}
		if closed {
			if stopErr := c.stopCandidatePreparationsForRoot(ctx, p.Repository, p.ProjectID, p.Issue, owner.AttemptRoot); stopErr != nil {
				publicationWait(report, p.ID, "attempt is closed; native preparation retirement remains pending")
				return
			}
			publicationWait(report, p.ID, "attempt is closed; maintainer intervention is required")
		}
		return
	}
	head, err := c.Store.AcceptanceHead(ctx, p.Repository, p.Issue)
	if err != nil {
		publicationError(report, p.ID, "acceptance_unavailable")
		return
	}
	if head != p.Acceptance {
		return
	}

	review, hasReview, reviewPending, reviewRefused := c.currentReviewReport(ctx, p)
	if reviewRefused {
		c.fenceChildPublication(ctx, p, report)
		return
	}
	if reviewPending {
		return
	}
	role := project.RoleReviewer
	var checks *factory.CheckAssessment
	if hasReview {
		assessment, ok := c.currentChildCheckAssessment(ctx, p)
		if !ok {
			return
		}
		checks = assessment
		if (review.Verdict != "request-changes") && assessment.Verdict != factory.CheckFailed {
			return
		}
		role = project.RoleCoder
	} else if assessment, ok := c.currentChildCheckAssessment(ctx, p); ok {
		checks = assessment
	}
	if role == project.RoleReviewer && c.Reviews == nil {
		publicationWait(report, p.ID, "review_submission_unavailable")
		return
	}
	// D08 route before spending: a concrete unfixed defect takes the
	// bounded correction; optional-only, disputed or repetitive feedback
	// withholds the run pending the funded Jev decision, and optional
	// advice never forces a correction. The policy-required cause is
	// detected below for first inspections with passing checks.
	policyReview := false
	var policyActor int64
	if role == project.RoleCoder && hasReview && review.Verdict == "request-changes" {
		checksVerdict := ""
		if checks != nil {
			checksVerdict = checks.Verdict
		}
		// Correction-entry feedback routes only to a bounded correction
		// or to the withheld Jev decision; any other route is a
		// programming error, so it refuses loudly instead of spending.
		if route := routeDeliveryFeedback(deliveryFeedback{
			HasReview: true, ReviewVerdict: review.Verdict, ChecksVerdict: checksVerdict,
			NewestDispositions: newestCurrentHeadDispositions(p),
		}); route == routeJev {
			publicationWait(report, p.ID, "delivery_judgment_required")
			return
		} else if route != routeCorrection {
			publicationError(report, p.ID, "child_assignment_unrecorded")
			return
		}
	}
	if role == project.RoleReviewer && !hasReview && checks != nil && checks.Verdict == factory.CheckPass {
		if policy, err := c.Store.RepositoryPolicy(ctx, p.Repository); err == nil && policy.Validate() == nil {
			policyActor = policy.Review.ActorID
			policyReview = policyCauseOwed(p, checks.Verdict, policyActor)
		}
	}
	// Typed-wire leg: a phased child records its execution phase, issue
	// subject and parent. First reviews follow a phased owner; follow-up
	// children follow the evidence they answer, so a typed verdict earns
	// a phased child while legacy string verdicts keep the legacy child
	// they admit. Anything else stays pre-phase.
	childPhase := ""
	phasedChild := false
	if phasedOwner && (review.Legacy == nil) {
		childPhase = project.ExecutionCorrect
		if role == project.RoleReviewer {
			childPhase = project.ExecutionReview
		}
		phasedChild = true
	}
	lineage := attemptLineage{}
	parentAssignment := ""
	if phasedChild {
		if role == project.RoleReviewer {
			parentAssignment = owner.ID
		} else {
			reviewer, ok := c.currentReviewerForCorrection(ctx, p, owner)
			if !ok {
				publicationWait(report, p.ID, "child_assignment_unavailable")
				return
			}
			parentAssignment = reviewer.ID
		}
		lineage = attemptLineage{phase: childPhase, parent: parentAssignment}
	}
	// Durable D07 cycle: fresh independent reviews bind head/base only;
	// corrections and same-head re-reviews bind the full outstanding
	// review/check evidence. New digests open new rounds (fresh IDs);
	// identical digests replay their round (same IDs).
	var digest, triggerDecision, triggerReason string
	shortHead := p.Candidate
	if len(shortHead) > 8 {
		shortHead = shortHead[:8]
	}
	if role == project.RoleReviewer {
		if policyReview {
			// The bounded policy-required pass carries its separately
			// recorded cause; the digest stays head/base-only so a
			// replay reuses its round and never double-charges.
			digest = reviewCycleInputDigest(p.Candidate, p.PRCreate.BaseOID, nil, nil)
			triggerDecision, triggerReason = policyReviewDecision, policyReviewReason(policyActor, p.Candidate)
		} else if hasReview {
			digest = reviewCycleEvidenceDigest(p.Candidate, p.PRCreate.BaseOID, review, checks)
			triggerDecision, triggerReason = "re-review-same-head", "another inspection of head "+shortHead+" after "+review.Verdict+" with outstanding findings"
		} else if len(p.ReviewCycles) == 0 {
			digest = reviewCycleInputDigest(p.Candidate, p.PRCreate.BaseOID, nil, nil)
			triggerDecision, triggerReason = "initial-review", "first review of published head "+shortHead
		} else if reviewCycleHeadSeen(p, p.Candidate) {
			digest = reviewCycleInputDigest(p.Candidate, p.PRCreate.BaseOID, nil, nil)
			triggerDecision, triggerReason = "re-review-same-head", "another inspection of head "+shortHead+" without prior verdict"
		} else {
			digest = reviewCycleInputDigest(p.Candidate, p.PRCreate.BaseOID, nil, nil)
			triggerDecision, triggerReason = "re-review-after-correction", "fresh review of corrected head "+shortHead
		}
	} else {
		digest = reviewCycleEvidenceDigest(p.Candidate, p.PRCreate.BaseOID, review, checks)
		triggerDecision, triggerReason = "correction-after-review", "address "+review.Verdict+" with "+checks.Verdict+" checks on head "+shortHead
	}
	cycle, childID, expectedPrepID, cycleErr := c.ensureReviewCycle(ctx, &p, owner, role, digest, triggerDecision, triggerReason)
	if cycleErr != nil {
		publicationWait(report, p.ID, "child_cycle_unrecorded")
		return
	}
	existing, err := c.Store.Assignment(ctx, childID)
	if err == nil {
		if !existingMatchesPublicationChild(childID, existing, p, owner, role, childPhase, parentAssignment) {
			c.fenceChildPublication(ctx, p, report)
			return
		}
		if existing.Stage == factory.AssignmentFinished &&
			(existing.Outcome != factory.Succeeded || existing.Result == nil || !existing.Result.Reported || existing.Result.Status != "completed") {
			c.fenceChildPublication(ctx, p, report)
		}
		return
	}
	if !errors.Is(err, store.ErrNotFound) {
		publicationError(report, p.ID, "child_assignment_unavailable")
		return
	}
	if role == project.RoleCoder && !slices.Contains(allowance.Corrections, childID) &&
		len(allowance.Corrections) >= allowance.Limits.CorrectionCycles {
		c.fenceChildPublication(ctx, p, report)
		return
	}
	if role == project.RoleReviewer {
		// D08 positive evidence before buying a review: charge the
		// attempt-root allowance by the immutable reviewer assignment
		// identity before any native work or dispatch packet. Every
		// review consumes one ReviewRound; a discretionary re-review
		// of an inspected head additionally spends that head's single
		// extra pass, while the policy-required cause spends only the
		// round. Replays of the same assignment never recharge.
		// Exhaustion withholds the run instead of launching review
		// without cover.
		_, exhausted, chargeErr := c.Store.ChargeReviewDispatch(ctx, p.Repository, p.Issue, owner.AttemptRoot, childID, p.Candidate, policyReview)
		if chargeErr != nil {
			publicationError(report, p.ID, "review_allowance_unavailable")
			return
		}
		if exhausted {
			publicationWait(report, p.ID, "review_allowance_exhausted")
			return
		}
	}
	deps := c.dispatchDeps()
	occupancy, err := snapshotOccupancy(ctx, deps.Store)
	if err != nil {
		publicationError(report, p.ID, "capacity_accounting_unavailable")
		return
	}
	control, err := deps.Store.IssueControl(ctx, p.Repository, p.Issue)
	if err == nil {
		control, err = c.refreshReadiness(ctx, control)
	}
	if err != nil || control.Readiness != factory.ReadinessQueued || control.Acceptance != p.Acceptance {
		publicationWait(report, p.ID, "issue_readiness_not_current")
		return
	}
	exactPreparation := ""
	if role == project.RoleCoder {
		exactPreparation = owner.Preparation
	}
	plan, wait, failed := planAttemptForRole(ctx, deps, &occupancy, p.Repository, p.Issue, p.Acceptance, control, false, role, exactPreparation, true, "", lineage)
	if failed != nil {
		publicationWait(report, p.ID, failed.Reason)
		return
	}
	if wait != nil {
		publicationWait(report, p.ID, wait.Reason)
		return
	}
	if plan.attemptRoot != owner.AttemptRoot || plan.projectID != owner.ProjectID ||
		plan.policy.Roles[role].Model == "" ||
		plan.inputs.Revision != control.NativeRev {
		publicationWait(report, p.ID, "child_dispatch_inputs_changed")
		return
	}
	plan.inputs.Tip = p.Candidate
	preparationID := plan.prep.Preparation.ID
	contextPreparation := plan.prep.Preparation
	if role == project.RoleReviewer {
		prep, ready, prepErr := c.prepareReviewCandidate(ctx, p, owner, plan, report, expectedPrepID, childID)
		if prepErr != nil {
			publicationWait(report, p.ID, "review_preparation_unavailable")
			return
		}
		if !ready {
			publicationWait(report, p.ID, "review_preparation_pending")
			return
		}
		if prep.ID != expectedPrepID {
			publicationError(report, p.ID, "child_assignment_unrecorded")
			return
		}
		preparationID = prep.ID
		contextPreparation = prep
	}
	if !time.Now().Before(plan.deadline) {
		publicationWait(report, p.ID, "attempt_deadline_exhausted")
		return
	}
	// Full outstanding finding/correction evidence goes to corrections
	// always, and to same-head re-reviews; fresh independent reviews stay
	// evidence-free so distinct reviewer sessions cannot rubber-stamp.
	// Typed verdicts ride the typed evidence fields; legacy string reports
	// ride the legacy field — exactly one source per prompt, never both.
	var reviewInput *factory.ReviewReport
	var typedReview *factory.ReviewCycleReport
	var typedClear *factory.ClearReviewReport
	var checksPrompt *factory.CheckAssessment
	if role == project.RoleReviewer {
		if hasReview {
			reviewInput, typedReview, typedClear, checksPrompt = review.Legacy, review.Review, review.Clear, checks
		}
	} else {
		if hasReview {
			reviewInput, typedReview, typedClear = review.Legacy, review.Review, review.Clear
		}
		checksPrompt = checks
	}
	repositoryContext, contextErr := readRepositoryContext(ctx, c.Host, plan, contextPreparation, role,
		owner.SourceCommit, p.PRCreate.BaseOID, p.Candidate, false)
	if contextErr != nil {
		publicationWait(report, p.ID, "native prepared repository context is unavailable or changed")
		return
	}
	prompt, err := buildPublicationChildPrompt(plan, p, owner, preparationID, role, reviewInput, typedReview, typedClear, checksPrompt, repositoryContext)
	if err != nil {
		publicationWait(report, p.ID, "child_prompt_invalid")
		return
	}
	plan.prompt, plan.promptSHA = prompt, project.FactoryPromptDigest(prompt)
	// A phased corrector executes the controller-derived singleton plan,
	// never a rebuilt or agent-selected one. An accepted-policy gap is a
	// visible planning blocker; any other derivation failure refuses
	// loudly instead of launching without a manifest.
	var correctionPlan *factory.InvestigationPlan
	if phasedChild && role == project.RoleCoder {
		derived, derivationErr := c.correctionPlanFor(ctx, plan, p, owner, cycle, review)
		if derivationErr != nil {
			if reason, blocked := correctionPlanWaitReason(derivationErr); blocked {
				publicationWait(report, p.ID, reason)
				return
			}
			publicationError(report, p.ID, "child_assignment_unrecorded")
			return
		}
		correctionPlan = derived
	}
	// Recheck the source, preparation, current tuple and allowance after
	// the waits above: a moved head, retired preparation, replaced
	// acceptance or spent allowance waits instead of launching stale
	// work, and a phased correction additionally rebinds its current
	// reviewer before any host call.
	if wait, recheckErr := c.recheckPublicationChild(ctx, p, owner, plan, phasedChild, parentAssignment); recheckErr != nil {
		publicationError(report, p.ID, "child_assignment_unavailable")
		return
	} else if wait != nil {
		publicationWait(report, p.ID, wait.Reason)
		return
	}
	c.executePublicationChild(ctx, deps, &occupancy, plan, p, owner, childID, preparationID, role, correctionPlan, report)
}

// correctionPlanFor derives the singleton correction plan for one phased
// correction round from the accepted issue inputs and the owner's
// immutable recorded investigation plan.
func (c *Coordinator) correctionPlanFor(ctx context.Context, plan *attemptPlan, p factory.Publication, owner factory.Assignment, cycle factory.ReviewCycle, review ResolvedReviewVerdict) (*factory.InvestigationPlan, error) {
	_ = ctx
	if owner.Phase != project.ExecutionInvestigate || owner.Result == nil || owner.Result.PhaseResult == nil ||
		owner.Result.PhaseResult.Output == nil || owner.Result.PhaseResult.Output.Plan == nil {
		return nil, &correctionPlanBlocked{reason: "correction_plan_unavailable", detail: "publication owner carries no recorded investigation plan"}
	}
	if cycle.CorrectionAssignment == "" || cycle.Head != p.Candidate || cycle.BaseOID != p.PRCreate.BaseOID {
		return nil, &correctionPlanBlocked{reason: "correction_plan_unavailable", detail: "correction answers another round"}
	}
	return deriveCorrectionPlan(plan.acceptance, *owner.Result.PhaseResult.Output.Plan,
		owner.AttemptRoot, p.Candidate, p.PRCreate.BaseOID, owner.Preparation,
		cycle.ID, cycle.FindingDigest, correctionFindingKeys(review), plan.policy.Checks)
}

// correctionFindingKeys returns the deterministic finding keys one phased
// correction answers: the typed verdict's new findings plus its carried
// assessments. Legacy string verdicts carry no stable keys.
func correctionFindingKeys(review ResolvedReviewVerdict) []string {
	seen := map[string]bool{}
	var keys []string
	if review.Review != nil {
		for _, finding := range review.Review.NewFindings {
			if !seen[finding.Key] {
				seen[finding.Key] = true
				keys = append(keys, finding.Key)
			}
		}
		for _, assessment := range review.Review.Assessments {
			if !seen[assessment.Key] {
				seen[assessment.Key] = true
				keys = append(keys, assessment.Key)
			}
		}
	}
	if review.Clear != nil {
		for _, assessment := range review.Clear.Assessments {
			if !seen[assessment.Key] {
				seen[assessment.Key] = true
				keys = append(keys, assessment.Key)
			}
		}
	}
	return keys
}

// recheckPublicationChild re-reads the publication, current tuple and
// allowance after the wait-inducing preparation and planning above. A
// changed candidate, fenced publication, replaced acceptance, retired
// allowance or rebound reviewer waits instead of launching.
func (c *Coordinator) recheckPublicationChild(ctx context.Context, p factory.Publication, owner factory.Assignment, plan *attemptPlan, phasedChild bool, parentAssignment string) (*planWait, error) {
	fresh, err := c.Store.PublicationByAssignment(ctx, owner.ID)
	if err != nil {
		return nil, err
	}
	if fresh.Stage != factory.PublicationPublished || fresh.WithdrawRequested ||
		fresh.Candidate != p.Candidate || fresh.PRNumber != p.PRNumber || fresh.PRID != p.PRID ||
		fresh.PRCreate.BaseOID != p.PRCreate.BaseOID {
		return waitFor("child_dispatch_inputs_changed", "published candidate changed before launch"), nil
	}
	head, err := c.Store.AcceptanceHead(ctx, p.Repository, p.Issue)
	if err != nil {
		return nil, err
	}
	if head != p.Acceptance {
		return waitFor("child_dispatch_inputs_changed", "acceptance head changed before launch"), nil
	}
	allowance, err := c.Store.AttemptAllowance(ctx, p.Repository, p.Issue)
	if err != nil {
		return nil, err
	}
	if allowance.RootAssignment != owner.AttemptRoot || allowance.Closed || !allowance.Active ||
		allowance.RemainingSeconds(time.Now()) <= 0 {
		return waitFor("child_dispatch_inputs_changed", "attempt allowance changed before launch"), nil
	}
	if plan.inputs.Tip != fresh.Candidate {
		return waitFor("child_dispatch_inputs_changed", "published candidate changed before launch"), nil
	}
	if phasedChild && plan.phase == project.ExecutionCorrect {
		reviewer, ok := c.currentReviewerForCorrection(ctx, fresh, owner)
		if !ok || reviewer.ID != parentAssignment {
			return waitFor("child_dispatch_inputs_changed", "current reviewer changed before launch"), nil
		}
	}
	return nil, nil
}

// currentReviewerForCorrection resolves the reviewer assignment behind the
// newest committed verdict on the publication's exact current head and
// base. A pending or refused current-head operation, a missing or stale
// reviewer, or any other head all report false with zero downstream
// effect: no native call happens before every check passes.
func (c *Coordinator) currentReviewerForCorrection(ctx context.Context, p factory.Publication, owner factory.Assignment) (factory.Assignment, bool) {
	var empty factory.Assignment
	for i := len(p.ReviewOperations) - 1; i >= 0; i-- {
		op := p.ReviewOperations[i]
		if op.Work.HeadOID != p.Candidate {
			continue
		}
		if op.Work.BaseOID != p.PRCreate.BaseOID {
			return empty, false
		}
		if op.Outcome.Effect == factory.OpEffectPending || op.Outcome.Effect == factory.OpEffectIndeterminate ||
			op.Outcome.Completion == factory.OpCompletionPending {
			return empty, false
		}
		if op.Outcome.Effect != factory.OpEffectCommitted || op.Outcome.Completion != factory.OpCompletionComplete {
			return empty, false
		}
		reviewer, err := c.Store.AssignmentByRun(ctx, op.RunID)
		if err != nil || reviewer.Validate() != nil || reviewer.Role != project.RoleReviewer ||
			reviewer.PublicationAssignment != owner.ID || reviewer.AttemptRoot != owner.AttemptRoot ||
			reviewer.SourceCommit != p.Candidate || reviewer.Repository != p.Repository ||
			reviewer.Issue != p.Issue || reviewer.Acceptance != p.Acceptance ||
			reviewer.Stage != factory.AssignmentFinished || reviewer.Outcome != factory.Succeeded ||
			reviewer.Result == nil || !reviewer.Result.Reported || reviewer.Result.Status != "completed" {
			return empty, false
		}
		return reviewer, true
	}
	return empty, false
}

// currentReviewReport returns only evidence whose native operation completed
// for the publication's current exact head and base. A pending or refused
// current-head operation blocks replacement review/correction work. The
// verdict resolves through the single canonical consumer: legacy string
// reports for pre-phase runs, typed verdicts for phased runs, with the
// full head/base/cycle/reviewer/carried binding enforced for the latter
// before it routes any further work.
func (c *Coordinator) currentReviewReport(ctx context.Context, p factory.Publication) (ResolvedReviewVerdict, bool, bool, bool) {
	var empty ResolvedReviewVerdict
	for i := len(p.ReviewOperations) - 1; i >= 0; i-- {
		op := p.ReviewOperations[i]
		if op.Work.HeadOID != p.Candidate {
			continue
		}
		if op.Work.BaseOID != p.PRCreate.BaseOID {
			return empty, false, false, true
		}
		if op.Outcome.Effect == factory.OpEffectPending || op.Outcome.Effect == factory.OpEffectIndeterminate ||
			op.Outcome.Completion == factory.OpCompletionPending {
			return empty, false, true, false
		}
		if op.Outcome.Effect != factory.OpEffectCommitted || op.Outcome.Completion != factory.OpCompletionComplete {
			return empty, false, false, true
		}
		assignment, err := c.Store.AssignmentByRun(ctx, op.RunID)
		if err != nil || assignment.Validate() != nil || assignment.Role != project.RoleReviewer ||
			assignment.PublicationAssignment != p.AssignmentID || assignment.SourceCommit != p.Candidate ||
			assignment.Stage != factory.AssignmentFinished || assignment.Outcome != factory.Succeeded ||
			assignment.Result == nil || !assignment.Result.Reported {
			return empty, false, false, true
		}
		resolved, err := resolveAssignmentVerdict(assignment)
		if err != nil {
			return empty, false, false, true
		}
		wantEvent := "APPROVED"
		if resolved.Verdict == "request-changes" {
			wantEvent = "REQUEST_CHANGES"
		}
		if op.Work.Event != wantEvent {
			return empty, false, false, true
		}
		if resolved.Legacy != nil {
			if resolved.Legacy.Validate() != nil {
				return empty, false, false, true
			}
			return resolved, true, false, false
		}
		if _, _, err := c.bindPhasedReviewVerdict(ctx, p, assignment, p.Candidate, resolved); err != nil {
			return empty, false, false, true
		}
		return resolved, true, false, false
	}
	return empty, false, false, false
}

// currentChildCheckAssessment accepts only the current policy's complete
// canonical verdict for this publication's exact current head/base.
func (c *Coordinator) currentChildCheckAssessment(ctx context.Context, p factory.Publication) (*factory.CheckAssessment, bool) {
	assessment, err := c.Store.CheckAssessment(ctx, p.Repository, p.PRNumber)
	if err != nil || assessment.Validate() != nil {
		return nil, false
	}
	policy, err := c.Store.RepositoryPolicy(ctx, p.Repository)
	if err != nil || policy.Validate() != nil {
		return nil, false
	}
	if assessment.Repository != p.Repository || assessment.PRNumber != p.PRNumber || assessment.PRID != p.PRID ||
		assessment.IssueID != p.PRCreate.IssueID || assessment.HeadOID != p.Candidate ||
		assessment.BaseOID != p.PRCreate.BaseOID || assessment.HeadRef != p.PRCreate.HeadRef ||
		assessment.BaseRef != p.PRCreate.BaseRef || assessment.PolicyRevision != policy.Revision ||
		!slices.Equal(assessment.Checks, policy.Checks) || assessment.ChecksDigest != factory.ChecksDigest(policy.Checks) ||
		(assessment.Verdict != factory.CheckPass && assessment.Verdict != factory.CheckFailed) {
		return nil, false
	}
	return &assessment, true
}

func (c *Coordinator) prepareReviewCandidate(ctx context.Context, p factory.Publication, owner factory.Assignment, plan *attemptPlan, report *PublishReport, expectedPrepID, expectedChildID string) (project.Preparation, bool, error) {
	source := plan.prep.Preparation
	prep := project.Preparation{
		ID: expectedPrepID, Project: p.ProjectID, Role: project.RoleReviewer,
		Requirements: source.Requirements, Approval: source.Approval, SourceCommit: p.Candidate,
		SetupDigest: source.SetupDigest, Tools: append([]string(nil), source.Tools...), Credential: source.Credential,
	}
	if err := prep.Validate(); err != nil {
		return project.Preparation{}, false, err
	}
	requested := time.Now().Add(10 * time.Minute).Unix()
	if plan.deadline.Unix() < requested {
		requested = plan.deadline.Unix()
	}
	registration := store.CandidatePreparationRegistration{
		Repository: p.Repository, Issue: p.Issue, ActorID: owner.ActorID, Project: p.ProjectID,
		OwnerAssignment: owner.ID, ChildAssignment: expectedChildID,
		AttemptRoot: owner.AttemptRoot, PublicationID: p.ID, Connection: plan.sponsorship.Connection,
		Authority: plan.effective.Authority, GateRevision: plan.gateRev, Control: plan.control,
		RequestedNotAfter: requested,
	}
	storedRecord, created, err := c.Store.AdmitCandidatePreparation(ctx, project.StoredPreparation{Preparation: prep}, registration)
	if err != nil {
		// A changed control, lost capacity race or unavailable Store admits
		// nothing. The same owner retries; only a proved identity conflict
		// makes this publication terminal.
		if errors.Is(err, store.ErrCandidatePreparationConflict) || errors.Is(err, store.ErrAttemptRootMismatch) {
			c.fenceChildPublication(ctx, p, report)
		}
		return project.Preparation{}, false, err
	}
	stored := storedRecord.Preparation
	if stored.Preparation.ID != prep.ID || stored.Preparation.Role != prep.Role ||
		stored.Preparation.Project != prep.Project || stored.Preparation.SourceCommit != prep.SourceCommit ||
		!stored.Preparation.Requirements.Equal(prep.Requirements) || stored.Preparation.Approval != prep.Approval ||
		stored.Preparation.SetupDigest != prep.SetupDigest || stored.Preparation.Credential != prep.Credential ||
		!slices.Equal(stored.Preparation.Tools, prep.Tools) {
		c.fenceChildPublication(ctx, p, report)
		return project.Preparation{}, false, errors.New("review preparation identity differs")
	}
	if stored.State.Ready {
		if !candidatePreparationStateMatches(stored.State, prep) {
			c.fenceChildPublication(ctx, p, report)
			return project.Preparation{}, false, errors.New("review preparation state differs")
		}
		return prep, true, nil
	}
	if !created {
		state, inspectErr := c.Host.InspectPreparation(ctx, project.PrepareInspect{Project: p.ProjectID, ID: prep.ID})
		if inspectErr == nil {
			if state.ID != "" && !candidatePreparationIdentityMatches(state, prep) {
				c.fenceChildPublication(ctx, p, report)
				return project.Preparation{}, false, errors.New("native review preparation differs")
			}
			if err := c.observeCandidatePreparation(ctx, stored, state); err != nil {
				return project.Preparation{}, false, err
			}
			if state.Ready {
				return prep, true, nil
			}
			if state.Phase == project.PrepareRunning {
				return project.Preparation{}, false, nil
			}
			if state.Phase == project.PrepareFailed || state.Phase == project.PrepareStopped || state.Phase == project.PrepareInterrupted {
				c.fenceChildPublication(ctx, p, report)
				return project.Preparation{}, false, errors.New("review preparation requires intervention")
			}
		} else if !errors.Is(inspectErr, project.ErrPreparationNotFound) {
			return project.Preparation{}, false, inspectErr
		}
	}
	parentRun, err := c.Store.FactoryRun(ctx, p.Run)
	if err != nil || parentRun.Role != project.RoleCoder || !parentRun.Reconciled || parentRun.Outcome != factory.Succeeded {
		return project.Preparation{}, false, errors.New("published candidate run is not settled")
	}
	candidateAssignment, err := c.Store.AssignmentByRun(ctx, p.Run)
	if err != nil || candidateAssignment.Validate() != nil || candidateAssignment.Role != project.RoleCoder ||
		candidateAssignment.PublicationAssignment != owner.ID || candidateAssignment.AttemptRoot != owner.AttemptRoot ||
		candidateAssignment.Stage != factory.AssignmentFinished || candidateAssignment.Outcome != factory.Succeeded ||
		candidateAssignment.Result == nil || !candidateAssignment.Result.Reported || candidateAssignment.Result.Status != "completed" ||
		candidateAssignment.Result.Candidate != p.Candidate {
		return project.Preparation{}, false, errors.New("published candidate lacks its exact recorded report")
	}
	bundle, terminal, wait := c.exportCandidate(ctx, p, parentRun)
	if wait != nil {
		return project.Preparation{}, false, errors.New(wait.Detail)
	}
	if terminal != nil {
		c.fenceChildPublication(ctx, p, report)
		return project.Preparation{}, false, errors.New("published candidate export is invalid")
	}
	notAfter, err := c.Store.AuthorizeCandidatePreparation(ctx, prep.ID)
	if err != nil {
		return project.Preparation{}, false, err
	}
	deadline := time.Unix(notAfter, 0)
	if !time.Now().Before(deadline) {
		return project.Preparation{}, false, errors.New("review preparation deadline elapsed")
	}
	input := project.FactoryCandidate{Preparation: prep, SourcePreparation: source.ID, Bundle: bundle, Deadline: deadline}
	if err := input.Validate(); err != nil {
		return project.Preparation{}, false, err
	}
	state, err := c.Host.PrepareCandidate(ctx, input)
	if err != nil {
		return project.Preparation{}, false, err
	}
	if state.ID != prep.ID || state.Project != p.ProjectID || state.Role != project.RoleReviewer ||
		state.SourceCommit != p.Candidate || state.SetupDigest != prep.SetupDigest {
		c.fenceChildPublication(ctx, p, report)
		return project.Preparation{}, false, errors.New("native review preparation receipt differs")
	}
	if err := c.observeCandidatePreparation(ctx, stored, state); err != nil {
		return project.Preparation{}, false, err
	}
	if !state.Ready {
		if state.Phase == project.PrepareFailed || state.Phase == project.PrepareStopped || state.Phase == project.PrepareInterrupted {
			c.fenceChildPublication(ctx, p, report)
			return project.Preparation{}, false, errors.New("review preparation requires intervention")
		}
		return project.Preparation{}, false, nil
	}
	return prep, true, nil
}

func candidatePreparationStateMatches(state project.PrepareState, prep project.Preparation) bool {
	return candidatePreparationIdentityMatches(state, prep) && state.Ready && !state.Stopped
}

func candidatePreparationIdentityMatches(state project.PrepareState, prep project.Preparation) bool {
	return state.ID == prep.ID && state.Project == prep.Project && state.Role == prep.Role &&
		state.SourceCommit == prep.SourceCommit && state.SetupDigest == prep.SetupDigest
}

func (c *Coordinator) observeCandidatePreparation(ctx context.Context, prior project.StoredPreparation, state project.PrepareState) error {
	current, err := c.Store.Preparation(ctx, prior.Preparation.ID)
	if err != nil {
		return err
	}
	if reflect.DeepEqual(current.State, state) {
		return nil
	}
	current.State = state
	if err := c.Store.ObservePreparation(ctx, current); err != nil {
		latest, readErr := c.Store.Preparation(ctx, prior.Preparation.ID)
		if readErr == nil && reflect.DeepEqual(latest.State, state) {
			return nil
		}
		return err
	}
	return nil
}

func (c *Coordinator) executePublicationChild(ctx context.Context, deps DispatchDeps, occupancy *passOccupancy, plan *attemptPlan, p factory.Publication, owner factory.Assignment, assignmentID, preparationID, role string, correctionPlan *factory.InvestigationPlan, report *PublishReport) {
	now := time.Now()
	runID := factory.NewID()
	// A phased corrector stages its controller-derived singleton plan as
	// the actual correct-phase manifest bytes (C15 wire) with no child
	// concurrency. Reviewers stage no manifest; legacy children stage
	// neither phase nor manifest.
	var manifest []byte
	var manifestDigest string
	if plan.phase == project.ExecutionCorrect {
		if correctionPlan == nil {
			publicationError(report, p.ID, "child_assignment_unrecorded")
			return
		}
		selection := plan.policy.Roles[role]
		staged, err := stageExecutionManifest(correctionPlan, runID, plan.deadline, selection.Profile, project.ExecutionCorrect, 0)
		if err != nil {
			publicationError(report, p.ID, "child_assignment_unrecorded")
			return
		}
		manifest, manifestDigest = staged, project.ExecutionManifestDigest(staged)
	}
	authority := plan.effective.Authority
	authority.RequirementsID, authority.ApprovalID = plan.requirement, plan.approval
	assignment := factory.Assignment{
		Authority: authority,
		ID:        assignmentID, AttemptRoot: owner.AttemptRoot, PublicationAssignment: owner.ID,
		Subject: plan.subject, Phase: plan.phase, ParentAssignment: plan.parent,
		ProjectID: owner.ProjectID, Role: role,
		Repository: p.Repository, Issue: p.Issue, NativeRev: plan.inputs.Revision,
		Acceptance: owner.Acceptance, Preparation: preparationID,
		Harness: plan.pin.Harness + "-" + plan.pin.Version, HarnessVers: plan.pin.Version,
		Model: plan.policy.Roles[role].Model, Connection: plan.sponsorship.Connection,
		SourceCommit: p.Candidate, Prompt: plan.prompt, PromptSHA: plan.promptSHA,
		Run: runID, RunHistory: []string{runID}, Stage: factory.AssignmentAssigned,
		Attempts: 1, CreatedUnix: now.Unix(), ActorID: owner.ActorID,
	}
	if err := assignment.Validate(); err != nil {
		publicationError(report, p.ID, "child_assignment_invalid")
		return
	}
	registration := factory.DispatchRegistration{ID: assignmentID, Repository: p.Repository, Revision: plan.gateRev, Authority: authority}
	reservation := factory.Reservation{
		AssignmentID: assignmentID, Repository: p.Repository, Connection: plan.sponsorship.Connection,
		State: factory.ReservationHeld, PlannedMinutes: plan.planned,
	}
	run := factory.Run{
		ID: runID, ProjectID: owner.ProjectID, Role: role, InputSHA: p.Candidate,
		Started: now, Deadline: plan.deadline,
		Image: plan.pin.Image, Harness: assignment.Harness, Model: assignment.Model,
	}
	view := factory.RunView{RunID: runID, Repository: p.Repository, Issue: p.Issue, Attempt: assignmentID}
	if err := deps.Store.RecordDispatchPacket(ctx, registration, plan.control, assignment, reservation, run, view); err != nil {
		if errors.Is(err, store.ErrAssignmentActive) || errors.Is(err, store.ErrAdmissionChanged) || errors.Is(err, store.ErrDispatchControlStale) {
			publicationWait(report, p.ID, "child_admission_changed")
			return
		}
		if wait := admissionWait(err); wait != nil {
			publicationWait(report, p.ID, wait.Reason)
			return
		}
		publicationError(report, p.ID, "child_assignment_unrecorded")
		return
	}
	occupancy.hold(reservation)
	launch := project.FactoryLaunch{
		Run: project.FactoryRun{
			ID: runID, Project: owner.ProjectID, Role: role,
			Preparation: preparationID, Harness: plan.pin.Harness, HarnessVers: plan.pin.Version,
			Model: assignment.Model, Assignment: assignment.PromptSHA, SourceCommit: p.Candidate,
			Connection: plan.sponsorship.Connection, Actor: assignment.ActorID, Deadline: plan.deadline,
			ExecutionPhase: plan.phase, ManifestDigest: manifestDigest,
		},
		Prompt: plan.prompt, HarnessSHA256: plan.pin.SHA256,
		ExecutionManifest: manifest,
	}
	if err := launch.Validate(); err != nil {
		publicationError(report, p.ID, "child_launch_invalid")
		return
	}
	if plan.phase == project.ExecutionCorrect {
		if err := verifyStagedManifest(launch); err != nil {
			publicationError(report, p.ID, "child_launch_invalid")
			return
		}
	}
	tail := newDispatchReport()
	launchTail(ctx, deps, assignment, run, launch, &tail)
	for _, wait := range tail.Waits {
		publicationWait(report, p.ID, wait.Reason)
	}
	for _, failure := range tail.Errors {
		publicationError(report, p.ID, failure.Reason)
	}
}

func existingMatchesPublicationChild(expectedID string, a factory.Assignment, p factory.Publication, owner factory.Assignment, role, phase, parent string) bool {
	if !(a.Validate() == nil && a.ID == expectedID &&
		a.PublicationAssignment == owner.ID && a.AttemptRoot == owner.AttemptRoot &&
		a.Repository == p.Repository && a.Issue == p.Issue && a.Acceptance == p.Acceptance &&
		a.Role == role && a.SourceCommit == p.Candidate) {
		return false
	}
	if phase == "" {
		return a.Phase == "" && a.ParentAssignment == "" && a.Subject.Kind == ""
	}
	return a.Phase == phase && a.ParentAssignment == parent &&
		a.Subject.Kind == factory.SubjectIssue && a.Subject.Issue != nil &&
		a.Subject.Issue.Issue == p.Issue && a.Subject.Issue.Acceptance == p.Acceptance
}

func publicationChildID(publication, head, role string) string {
	hash := sha256.Sum256([]byte("soda-factory-child-v1\x00" + publication + "\x00" + head + "\x00" + role))
	return hex.EncodeToString(hash[:16])
}

func publicationPreparationID(publication, head string) string {
	hash := sha256.Sum256([]byte("soda-factory-review-preparation-v1\x00" + publication + "\x00" + head))
	return "f" + hex.EncodeToString(hash[:12])
}

func buildPublicationChildPrompt(plan *attemptPlan, p factory.Publication, owner factory.Assignment, preparationID, role string, review *factory.ReviewReport, typedReview *factory.ReviewCycleReport, typedClear *factory.ClearReviewReport, checks *factory.CheckAssessment, repositoryContext *project.FactoryPreparationContext) ([]byte, error) {
	selection := plan.policy.Roles[role]
	// Publication children always serve a real execution phase, never the
	// legacy empty phase: reviewer children inspect the published head and
	// coder children correct it. The prompt carries the admitted phase so
	// the template owns phased rendering and the recorded result binds it.
	phase := project.ExecutionCorrect
	if role == project.RoleReviewer {
		phase = project.ExecutionReview
	}
	return factory.BuildDispatchPrompt(factory.PromptInputs{
		Project: plan.projectID, Repository: p.Repository, Issue: p.Issue, NativeRev: plan.inputs.Revision,
		Phase:        phase,
		AcceptanceID: p.Acceptance, TargetBranch: plan.policy.TargetBranch,
		SourceCommit: p.Candidate, ApprovedBase: owner.SourceCommit, BaseCommit: p.PRCreate.BaseOID,
		PublicationAssignment: p.AssignmentID, Preparation: preparationID,
		RequirementsID: plan.requirement, ApprovalID: plan.approval,
		Harness: plan.pin.Harness + "-" + plan.pin.Version, Model: selection.Model, Role: role,
		ProviderConnection: plan.sponsorship.Connection, RequiredChecks: plan.policy.Checks,
		ApplianceConcurrent: plan.applianceConcurrent, RepositoryConcurrent: plan.repositoryConcurrent,
		SponsorshipConcurrent: plan.sponsorship.MaxConcurrent, AttemptLimits: plan.attemptLimits,
		Title: plan.inputs.Issue.Title, Body: plan.inputs.Issue.Body,
		Sources:       promptSections(plan.acceptance.Sources, plan.inputs.Comments),
		Resolutions:   promptSections(plan.acceptance.Resolutions, plan.inputs.Comments),
		Prerequisites: plan.acceptance.Prerequisites, Control: plan.control,
		Review: review, TypedReview: typedReview, TypedClear: typedClear, CheckAssessment: checks,
		RepositoryContext: repositoryContext,
	})
}

func (c *Coordinator) fenceChildPublication(ctx context.Context, p factory.Publication, report *PublishReport) {
	p.Stage, p.Outcome, p.Reason, p.FinishedUnix = factory.PublicationFenced, factory.NeedsHuman, factory.PublishReasonFenced, time.Now().Unix()
	p.Revision++
	if err := c.Store.UpdatePublication(ctx, p); err != nil {
		publicationError(report, p.ID, "child_result_store_conflict")
		return
	}
	report.Fenced = append(report.Fenced, p.ID)
	owner, err := c.Store.Assignment(ctx, p.AssignmentID)
	if err != nil || owner.Validate() != nil {
		publicationWait(report, p.ID, "candidate_preparation_owner_unavailable")
		return
	}
	if err := c.stopCandidatePreparationsForRoot(ctx, p.Repository, p.ProjectID, p.Issue, owner.AttemptRoot); err != nil {
		publicationWait(report, p.ID, "candidate_preparation_retirement_pending")
	}
}
