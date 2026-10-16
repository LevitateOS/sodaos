package control

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
)

// reviewCyclePassLimit bounds one review-cycle sweep: published
// publications reconsidered for checks and merge progress.
const reviewCyclePassLimit = 256

// CheckPass assesses native checks for every published publication that
// lacks a passing assessment on its current head: missing verdicts,
// verdicts on a superseded head, and non-passing verdicts that later CI
// may flip. Publications already passing on their head are left alone;
// the merge still re-verifies checks live before submitting, so a stale
// pass can never ride through. While no check assessor is wired, the
// pass reports unavailable instead of guessing.
func (c *Coordinator) CheckPass(ctx context.Context) CheckReport {
	ctx, unlock := c.progressPass(ctx)
	defer unlock()
	ctx, _, cancelPass := c.readinessPass(ctx)
	defer cancelPass()
	report := CheckReport{Assessed: []CheckLink{}}
	if c.Checks == nil {
		children := PublishReport{}
		c.producePublishedChildren(ctx, &children)
		report.Unavailable = true
		return report
	}
	pubs, err := c.Store.MergeablePublications(ctx, reviewCyclePassLimit)
	if err != nil {
		checkError(&report, "", "store_unavailable")
		return report
	}
	for _, p := range pubs {
		if c.checkCurrent(ctx, p) {
			continue
		}
		policy, err := c.Store.RepositoryPolicy(ctx, p.Repository)
		if err != nil {
			checkError(&report, p.ID, "store_unavailable")
			continue
		}
		one := c.AssessPublicationChecks(ctx, c.Checks, policy.Merge.ActorID, p.AssignmentID)
		report.Assessed = append(report.Assessed, one.Assessed...)
		report.Waits = append(report.Waits, one.Waits...)
		report.Errors = append(report.Errors, one.Errors...)
		for _, link := range one.Assessed {
			if link.Verdict != factory.CheckPass || c.Merges == nil {
				continue
			}
			if !c.currentPublicationApproved(ctx, p) {
				continue
			}
			// A fresh pass opens the merge row: the merge pass
			// itself still demands the independent review and
			// re-verifies checks live before submitting.
			merges := MergeReport{Merged: []MergeLink{}}
			c.mergeOne(ctx, p, &merges)
		}
	}
	children := PublishReport{}
	c.producePublishedChildren(ctx, &children)
	return report
}

// checkCurrent reports whether the latest stored assessment already passes
// on the publication's exact current head and base.
func (c *Coordinator) checkCurrent(ctx context.Context, p factory.Publication) bool {
	assessment, err := c.Store.CheckAssessment(ctx, p.Repository, p.PRNumber)
	if err != nil {
		return false
	}
	return assessment.Verdict == factory.CheckPass &&
		assessment.HeadOID == p.Candidate && assessment.BaseOID == p.PRCreate.BaseOID
}

// progressAfterPublish advances one published assignment toward merge: a
// fresh check assessment on its head, then its merge row once the head
// passes. Merge rows open only behind a passing verdict: opening one on a
// head that evidence already dooms would fail it terminally and strand
// the publication. Reviews gate the merge inside the merge pass itself.
func (c *Coordinator) progressAfterPublish(ctx context.Context, assignmentID string) {
	p, err := c.Store.PublicationByAssignment(ctx, assignmentID)
	if err != nil || p.Stage != factory.PublicationPublished {
		return
	}
	if c.Checks != nil {
		if policy, err := c.Store.RepositoryPolicy(ctx, p.Repository); err == nil && !c.checkCurrent(ctx, p) {
			c.AssessPublicationChecks(ctx, c.Checks, policy.Merge.ActorID, p.AssignmentID)
			if fresh, err := c.Store.PublicationByAssignment(ctx, assignmentID); err == nil {
				p = fresh
			}
		}
	}
	children := PublishReport{}
	c.producePublishedChildren(ctx, &children)
	if fresh, err := c.Store.PublicationByAssignment(ctx, assignmentID); err == nil {
		p = fresh
	}
	if c.Merges == nil || !c.checkCurrent(ctx, p) || !c.currentPublicationApproved(ctx, p) {
		return
	}
	report := MergeReport{Merged: []MergeLink{}}
	c.mergeOne(ctx, p, &report)
}

func (c *Coordinator) currentPublicationApproved(ctx context.Context, p factory.Publication) bool {
	review, ok, pending, refused := c.currentReviewReport(ctx, p)
	return ok && !pending && !refused && review.Verdict == "approve"
}

// SubmitReviewForRun submits one settled reviewer run's genuine verdict
// through the separate reviewer actor: it re-reads the recorded
// publication, refuses a superseded head, observes the exact native
// target and adopts the committed receipt. Each reviewer run reviews the
// head it started from, carried in its recorded input; a correction that
// landed first makes the verdict stale, never submitted. While no
// reviewer is wired, submission reports unavailable instead of guessing.
func (c *Coordinator) SubmitReviewForRun(ctx context.Context, runID string) (factory.ReviewOutcome, error) {
	var empty factory.ReviewOutcome
	if c.Reviews == nil {
		return empty, errors.New("review submission unavailable")
	}
	run, err := c.Store.FactoryRun(ctx, runID)
	if err != nil {
		return empty, ErrNotFound
	}
	if run.Role != project.RoleReviewer {
		return empty, errors.New("run is not a review")
	}
	if !run.Reconciled {
		return empty, errors.New("review run is not settled")
	}
	assignment, owner, err := c.publicationAssignmentsForRun(ctx, run)
	if err != nil {
		return empty, err
	}
	p, err := c.Store.PublicationByAssignment(ctx, owner.ID)
	if err != nil {
		return empty, err
	}
	for i := range p.ReviewOperations {
		op := p.ReviewOperations[i]
		if op.RunID != run.ID {
			continue
		}
		if err := op.ValidateForPublication(p); err != nil {
			return empty, err
		}
		// A retry always looks up the saved immutable tuple before reading the
		// caller's output, current policy, head or native observation. Absence
		// cannot prove an earlier request will not arrive, so it never submits
		// again or rebuilds the operation identity.
		outcome, err := c.Reviews.LookupOp(ctx, op.Work.OperationID)
		if err != nil {
			return empty, err
		}
		if outcome.NotObserved {
			return empty, errors.New("recorded review outcome remains unresolved")
		}
		updated, err := op.WithOutcome(outcome)
		if err != nil {
			return empty, err
		}
		if !sameReviewOperationOutcome(op.Outcome, updated.Outcome) {
			p.ReviewOperations[i] = updated
			p.Revision++
			if err := c.Store.UpdatePublication(ctx, p); err != nil {
				return empty, err
			}
		}
		if outcome.Effect != factory.OpEffectCommitted || outcome.Completion != factory.OpCompletionComplete {
			return empty, errors.New("recorded review submission is not complete")
		}
		adopted, err := c.Reviews.AdoptReview(op.Work, outcome)
		if err != nil {
			return empty, err
		}
		if adopted.ReviewerID != op.Work.ActorID || adopted.Event != op.Work.Event || adopted.HeadOID != op.Work.HeadOID {
			return empty, errors.New("review receipt differs from its recorded work")
		}
		return adopted, nil
	}
	if p.Stage != factory.PublicationPublished || p.PRNumber <= 0 || p.PRID <= 0 || p.PRCreate.Work == nil {
		return empty, errors.New("review target is not published")
	}
	head := run.InputSHA
	if !factory.ValidCommit(head) {
		return empty, errors.New("review run lacks its exact head")
	}
	if head != p.Candidate {
		return empty, errors.New("review head superseded by a correction")
	}
	if assignment.Stage != factory.AssignmentFinished || assignment.Outcome != factory.Succeeded ||
		assignment.Result == nil || !assignment.Result.Reported || assignment.Result.Review == nil {
		return empty, errors.New("review lacks its recorded report")
	}
	report := *assignment.Result.Review
	policy, err := c.Store.RepositoryPolicy(ctx, p.Repository)
	if err != nil {
		return empty, err
	}
	event := "APPROVED"
	if report.Verdict == "request-changes" {
		event = "REQUEST_CHANGES"
	}
	body := report.Body
	if strings.TrimSpace(body) == "" {
		body = report.Summary
	}
	w := factory.ReviewWork{
		OperationID: "review-" + run.ID, AuthRevision: factory.ReviewAuthRevision(owner.ID, run.ID),
		Repository: p.Repository, ActorID: policy.Review.ActorID,
		PRNumber: p.PRNumber, PRID: p.PRID, IssueID: p.PRCreate.IssueID, PRAuthorID: p.PRCreate.Work.ActorID,
		HeadRef: p.PRCreate.HeadRef, BaseRef: p.PRCreate.BaseRef, HeadOID: head, BaseOID: p.PRCreate.BaseOID,
		Event: event, Body: body,
	}
	observed, err := c.Reviews.ObserveReview(ctx, w)
	if err != nil {
		return empty, err
	}
	notAfter, err := c.Store.CapAttemptDeadline(ctx, owner.ID, time.Now().Add(10*time.Minute).Unix())
	if err != nil {
		return empty, err
	}
	w.NativeRev, w.NotAfter = observed.NativeRev, notAfter
	if err := w.Validate(); err != nil {
		return empty, err
	}
	reviewOperation := factory.ReviewOperation{RunID: run.ID, Work: w}
	if err := p.CanAppendReviewOperation(reviewOperation); err != nil {
		return empty, err
	}
	// Persist the exact authorization before the only submission call. The
	// store transaction shares the withdrawal gate, so a refused registration
	// has no native effect.
	p.ReviewOperations = append(p.ReviewOperations, reviewOperation)
	p.Revision++
	if err := c.Store.UpdatePublication(ctx, p); err != nil {
		return empty, err
	}
	open, _, _, err := c.Store.DispatchState(ctx, p.Repository)
	if err != nil {
		return empty, errors.New("review dispatch state is unconfirmed after registration")
	}
	if !open {
		return empty, errors.New("review dispatch closed after registration")
	}
	outcome, err := c.Reviews.SubmitReview(ctx, w)
	if err != nil {
		return empty, err
	}
	updated, err := p.ReviewOperations[len(p.ReviewOperations)-1].WithOutcome(outcome)
	if err != nil {
		return empty, err
	}
	p.ReviewOperations[len(p.ReviewOperations)-1] = updated
	p.Revision++
	if err := c.Store.UpdatePublication(ctx, p); err != nil {
		return empty, err
	}
	adopted, err := c.Reviews.AdoptReview(w, outcome)
	if err != nil {
		return empty, err
	}
	if outcome.Completion != factory.OpCompletionComplete || adopted.ReviewerID != w.ActorID || adopted.Event != w.Event || adopted.HeadOID != w.HeadOID {
		return empty, errors.New("review submission incomplete")
	}
	return adopted, nil
}

func sameReviewOperationOutcome(a, b factory.OperationOutcome) bool {
	return a.OperationID == b.OperationID && a.InstallationID == b.InstallationID && a.Kind == b.Kind &&
		a.ActorID == b.ActorID && a.RepositoryID == b.RepositoryID && bytes.Equal(a.Receipt, b.Receipt) &&
		a.Effect == b.Effect && a.Cancellation == b.Cancellation && a.Completion == b.Completion &&
		a.Reason == b.Reason && a.NotObserved == b.NotObserved
}

func (c *Coordinator) reconcileRecordedReviews(ctx context.Context, p *factory.Publication, report *PublishReport) {
	for i := range p.ReviewOperations {
		op := p.ReviewOperations[i]
		prior := op.Outcome
		if prior.Effect == factory.OpEffectCommitted && prior.Completion == factory.OpCompletionComplete ||
			prior.Effect == factory.OpEffectNotCommitted && prior.Cancellation == factory.OpCancelCancelled {
			continue
		}
		if c.Reviews == nil {
			publicationWait(report, op.Work.OperationID, "review_reconciliation_unavailable")
			continue
		}
		outcome, err := c.Reviews.LookupOp(ctx, op.Work.OperationID)
		if err != nil {
			c.publicationCallError(report, op.Work.OperationID, err)
			continue
		}
		if outcome.NotObserved {
			publicationWait(report, op.Work.OperationID, "review_outcome_unresolved")
			continue
		}
		updated, err := op.WithOutcome(outcome)
		if err != nil {
			publicationError(report, op.Work.OperationID, "review_outcome_unattributed")
			continue
		}
		if !sameReviewOperationOutcome(prior, updated.Outcome) {
			p.ReviewOperations[i] = updated
			p.Revision++
			if err := c.Store.UpdatePublication(ctx, *p); err != nil {
				publicationError(report, op.Work.OperationID, "store_unavailable")
				return
			}
		}
		if outcome.Effect == factory.OpEffectCommitted {
			adopted, err := c.Reviews.AdoptReview(op.Work, outcome)
			if err != nil || adopted.ReviewerID != op.Work.ActorID || adopted.Event != op.Work.Event || adopted.HeadOID != op.Work.HeadOID {
				publicationError(report, op.Work.OperationID, "review_receipt_unattributed")
				continue
			}
			if outcome.Completion != factory.OpCompletionComplete {
				publicationWait(report, op.Work.OperationID, "review_completion_pending")
			}
		} else if outcome.Effect != factory.OpEffectNotCommitted {
			publicationWait(report, op.Work.OperationID, "review_effect_pending")
		}
	}
}

// publicationAssignmentsForRun resolves the run's own assignment and the
// coder assignment that owns its publication. Both scopes remain immutable;
// the native operation uses the publication owner while usage belongs to the run.
// reviewCyclePhase is the single D07 phase for publication review cycles.
// Reviewer and correction assignments for one round share the same cycle
// identity derivation; their assignment IDs stay distinct via role.
const reviewCyclePhase = "review"

// reviewCycleInputDigest binds one cycle's inputs: exact head/base plus the
// outstanding review/check evidence the child must answer. Fresh independent
// reviews bind head/base only; corrections and same-head re-reviews bind the
// prior review verdict/findings and check verdict. Sorted findings keep the
// digest stable; the hex encoding satisfies ValidDigest. New decisions change
// the digest (new round, fresh IDs); replay reuses the same digest (same
// round, same IDs).
func reviewCycleInputDigest(head, base string, review *factory.ReviewReport, checks *factory.CheckAssessment) string {
	h := sha256.New()
	h.Write([]byte("review-cycle-input-v1\x00" + head + "\x00" + base + "\x00"))
	if review != nil {
		h.Write([]byte(review.Verdict + "\x00" + review.Summary + "\x00" + review.Body + "\x00"))
		sorted := append([]string(nil), review.Findings...)
		sort.Strings(sorted)
		for _, f := range sorted {
			h.Write([]byte(f + "\x00"))
		}
	} else {
		h.Write([]byte("\x00"))
	}
	if checks != nil {
		h.Write([]byte(checks.Verdict + "\x00" + checks.Reason + "\x00"))
	} else {
		h.Write([]byte("\x00"))
	}
	return hex.EncodeToString(h.Sum(nil))
}

func maxReviewCycleRound(p factory.Publication) int {
	max := 0
	for _, c := range p.ReviewCycles {
		if c.Round > max {
			max = c.Round
		}
	}
	return max
}

func reviewCycleHeadSeen(p factory.Publication, head string) bool {
	for _, c := range p.ReviewCycles {
		if c.Head == head {
			return true
		}
	}
	return false
}

// reviewCycleAssignmentID derives fresh reviewer/correction assignment IDs
// from the cycle identity and role: publication, attempt root, round and
// phase via the cycle ID, plus role for reviewer/corrector distinctness. New
// rounds are distinct; replay of a round is identical.
func reviewCycleAssignmentID(cycleID, role string) string {
	sum := sha256.Sum256([]byte("soda-factory-cycle-child-v1\x00" + cycleID + "\x00" + role))
	return hex.EncodeToString(sum[:16])
}

// reviewCyclePreparationID derives a fresh reviewer preparation identity
// from the cycle identity. Corrector preparations reuse the owner coder
// preparation (distinct from every reviewer session); reviewer sessions are
// fresh per same-head round and distinct from the corrector.
func reviewCyclePreparationID(cycleID string) string {
	sum := sha256.Sum256([]byte("soda-factory-review-preparation-v1\x00" + cycleID))
	return "f" + hex.EncodeToString(sum[:12])
}

// ensureReviewCycle persists one durable D07 review cycle for the child being
// dispatched and returns the cycle plus the exact child/preparation IDs to
// use. First occurrence of a head reuses the legacy head-based IDs so
// pre-cycle publications keep their reviewer binding; repeats on the same
// head use round-derived fresh IDs (new decision distinct, replay identical).
// Reviewer rounds leave CorrectionAssignment empty; correction rounds record
// a fresh (unused) reviewer ID alongside the used correction ID so the cycle
// always validates while reviewer/corrector sessions stay distinct.
func (c *Coordinator) ensureReviewCycle(ctx context.Context, p *factory.Publication, owner factory.Assignment, role, digest, triggerDecision, triggerReason string) (factory.ReviewCycle, string, string, error) {
	var empty factory.ReviewCycle
	if !factory.ValidDigest(digest) || strings.TrimSpace(triggerDecision) == "" || strings.TrimSpace(triggerReason) == "" {
		return empty, "", "", errors.New("review cycle needs its digest and triggering decision")
	}
	if role != project.RoleReviewer && role != project.RoleCoder {
		return empty, "", "", errors.New("review cycle needs its reviewer or corrector role")
	}
	// Replay: the latest round already binds these exact inputs.
	if n := len(p.ReviewCycles); n > 0 {
		last := p.ReviewCycles[n-1]
		if last.Head == p.Candidate && last.BaseOID == p.PRCreate.BaseOID && last.FindingDigest == digest {
			var childID, prepID string
			if role == project.RoleReviewer {
				childID = last.ReviewerAssignment
				// Replay keeps the recorded IDs: first occurrence of a head
				// used legacy head-based IDs, repeats used round-derived IDs.
				// Infer from the stored child ID so new-head rounds (legacy
				// even when round > 1) replay their legacy preparation.
				if childID == publicationChildID(p.ID, p.Candidate, project.RoleReviewer) {
					prepID = publicationPreparationID(p.ID, p.Candidate)
				} else {
					prepID = reviewCyclePreparationID(last.ID)
				}
			} else {
				childID = last.CorrectionAssignment
				if childID == "" {
					return empty, "", "", errors.New("recorded cycle lacks its correction assignment")
				}
				prepID = owner.Preparation
			}
			if childID == "" {
				return empty, "", "", errors.New("recorded cycle lacks its child assignment")
			}
			return last, childID, prepID, nil
		}
	}
	round := maxReviewCycleRound(*p) + 1
	if round < 1 {
		round = 1
	}
	cycleID := factory.ReviewCycleID(p.ID, owner.AttemptRoot, round, reviewCyclePhase)
	firstHead := !reviewCycleHeadSeen(*p, p.Candidate)
	var reviewerAssignment, correctionAssignment, childID, prepID string
	if role == project.RoleReviewer {
		if firstHead && round == 1 {
			childID = publicationChildID(p.ID, p.Candidate, project.RoleReviewer)
			prepID = publicationPreparationID(p.ID, p.Candidate)
		} else if firstHead {
			// New head after prior rounds: legacy head-based IDs keep the
			// reviewer binding stable per head (fresh via new head).
			childID = publicationChildID(p.ID, p.Candidate, project.RoleReviewer)
			prepID = publicationPreparationID(p.ID, p.Candidate)
		} else {
			childID = reviewCycleAssignmentID(cycleID, project.RoleReviewer)
			prepID = reviewCyclePreparationID(cycleID)
		}
		reviewerAssignment, correctionAssignment = childID, ""
	} else {
		if firstHead {
			// First correction on a head that only has reviewer cycles so
			// far still gets a head-stable ID when the head is new;
			// same-head repeats use round-derived fresh IDs below.
			// Corrections share the head with their reviewer round, so a
			// head seen only by reviewer rounds still counts as seen.
			// Use derived IDs whenever any cycle exists for this head to
			// keep reviewer/corrector distinct per round.
			if len(p.ReviewCycles) > 0 {
				reviewerAssignment = reviewCycleAssignmentID(cycleID, project.RoleReviewer)
				correctionAssignment = reviewCycleAssignmentID(cycleID, project.RoleCoder)
				childID = correctionAssignment
			} else {
				reviewerAssignment = publicationChildID(p.ID, p.Candidate, project.RoleReviewer)
				correctionAssignment = publicationChildID(p.ID, p.Candidate, project.RoleCoder)
				childID = correctionAssignment
			}
		} else {
			reviewerAssignment = reviewCycleAssignmentID(cycleID, project.RoleReviewer)
			correctionAssignment = reviewCycleAssignmentID(cycleID, project.RoleCoder)
			childID = correctionAssignment
		}
		prepID = owner.Preparation
	}
	cycle := factory.ReviewCycle{
		PublicationID: p.ID, AttemptRoot: owner.AttemptRoot, ID: cycleID, Phase: reviewCyclePhase,
		TriggerDecision: triggerDecision, TriggerReason: triggerReason,
		Head: p.Candidate, BaseOID: p.PRCreate.BaseOID, FindingDigest: digest,
		ReviewerAssignment: reviewerAssignment, CorrectionAssignment: correctionAssignment, Round: round,
	}
	if err := cycle.Validate(); err != nil {
		return empty, "", "", err
	}
	p.ReviewCycles = append(p.ReviewCycles, cycle)
	p.Revision++
	if err := c.Store.UpdatePublication(ctx, *p); err != nil {
		// Drop the optimistic append; the next pass re-reads and replays.
		p.ReviewCycles = p.ReviewCycles[:len(p.ReviewCycles)-1]
		p.Revision--
		return empty, "", "", err
	}
	return cycle, childID, prepID, nil
}

func (c *Coordinator) publicationAssignmentsForRun(ctx context.Context, run factory.Run) (factory.Assignment, factory.Assignment, error) {
	view, err := c.Store.FactoryRunView(ctx, run.ID)
	if err != nil {
		return factory.Assignment{}, factory.Assignment{}, err
	}
	assignment, err := c.Store.Assignment(ctx, view.Attempt)
	if err != nil {
		return factory.Assignment{}, factory.Assignment{}, err
	}
	if assignment.Validate() != nil || assignment.Run != run.ID || assignment.ProjectID != run.ProjectID ||
		assignment.Role != run.Role || assignment.SourceCommit != run.InputSHA ||
		assignment.Repository != view.Repository || assignment.Issue != view.Issue {
		return factory.Assignment{}, factory.Assignment{}, errors.New("run differs from its recorded assignment")
	}
	owner := assignment
	if assignment.PublicationAssignment != assignment.ID {
		owner, err = c.Store.Assignment(ctx, assignment.PublicationAssignment)
		if err != nil {
			return factory.Assignment{}, factory.Assignment{}, err
		}
	}
	if owner.Validate() != nil || owner.PublicationAssignment != owner.ID || owner.Role != project.RoleCoder ||
		owner.Repository != assignment.Repository || owner.Issue != assignment.Issue ||
		owner.ProjectID != assignment.ProjectID || owner.Acceptance != assignment.Acceptance ||
		owner.AttemptRoot != assignment.AttemptRoot ||
		(assignment.Role == project.RoleCoder && assignment.Preparation != owner.Preparation) {
		return factory.Assignment{}, factory.Assignment{}, errors.New("assignment differs from its publication owner")
	}
	return assignment, owner, nil
}
