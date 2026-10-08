package control

import (
	"bytes"
	"context"
	"errors"
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
	report := CheckReport{Assessed: []CheckLink{}}
	if c.Checks == nil {
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
			// A fresh pass opens the merge row: the merge pass
			// itself still demands the independent review and
			// re-verifies checks live before submitting.
			merges := MergeReport{Merged: []MergeLink{}}
			c.mergeOne(ctx, p, &merges)
		}
	}
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
	if c.Merges == nil || !c.checkCurrent(ctx, p) {
		return
	}
	report := MergeReport{Merged: []MergeLink{}}
	c.mergeOne(ctx, p, &report)
}

// SubmitReviewForRun submits one settled reviewer run's genuine verdict
// through the separate reviewer actor: it re-reads the recorded
// publication, refuses a superseded head, observes the exact native
// target and adopts the committed receipt. Each reviewer run reviews the
// head it started from, carried in its recorded input; a correction that
// landed first makes the verdict stale, never submitted. While no
// reviewer is wired, submission reports unavailable instead of guessing.
func (c *Coordinator) SubmitReviewForRun(ctx context.Context, runID, output string) (factory.ReviewOutcome, error) {
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
	view, err := c.Store.FactoryRunView(ctx, runID)
	if err != nil {
		return empty, err
	}
	p, err := c.Store.PublicationByAssignment(ctx, view.Attempt)
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
	report, ok := factory.ParseReviewReport(output)
	if !ok {
		return empty, errors.New("review report unparseable")
	}
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
		OperationID: "review-" + run.ID, AuthRevision: factory.ReviewAuthRevision(view.Attempt, run.ID),
		Repository: p.Repository, ActorID: policy.Review.ActorID,
		PRNumber: p.PRNumber, PRID: p.PRID, IssueID: p.PRCreate.IssueID, PRAuthorID: p.PRCreate.Work.ActorID,
		HeadRef: p.PRCreate.HeadRef, BaseRef: p.PRCreate.BaseRef, HeadOID: head, BaseOID: p.PRCreate.BaseOID,
		Event: event, Body: body,
	}
	observed, err := c.Reviews.ObserveReview(ctx, w)
	if err != nil {
		return empty, err
	}
	w.NativeRev, w.NotAfter = observed.NativeRev, time.Now().Unix()+600
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

// correctAfterSettle advances one completed coder run beyond its
// assignment's finishing one as a correction to the same PR. The
// finishing run published above; later runs never reach accounting
// because the assignment already finished, so this path finds them
// through their recorded run view. Best effort: waits and errors stay
// in the correction report for the next trigger; the settled run above
// already recorded.
func (c *Coordinator) correctAfterSettle(ctx context.Context, run factory.Run, output string) {
	if c.Publication == nil || run.Role != project.RoleCoder || run.Outcome != factory.Succeeded {
		return
	}
	view, err := c.Store.FactoryRunView(ctx, run.ID)
	if err != nil {
		return
	}
	a, err := c.Store.Assignment(ctx, view.Attempt)
	if err != nil || a.Stage != factory.AssignmentFinished || a.Run == run.ID {
		return
	}
	if _, err := c.Store.PublicationByAssignment(ctx, a.ID); err != nil {
		return
	}
	c.PublishCorrection(ctx, a.ID, run.ID, output)
	c.progressAfterPublish(ctx, a.ID)
}

// reviewAfterSettle submits a settled reviewer run's verdict and advances
// its publication toward merge. Best effort like the coder publish path:
// failures wait for an explicit submission; the settled run above already
// recorded.
func (c *Coordinator) reviewAfterSettle(ctx context.Context, run factory.Run, output string) {
	if c.Reviews == nil || run.Role != project.RoleReviewer {
		return
	}
	if _, err := c.SubmitReviewForRun(ctx, run.ID, output); err != nil {
		return
	}
	if view, err := c.Store.FactoryRunView(ctx, run.ID); err == nil {
		c.progressAfterPublish(ctx, view.Attempt)
	}
}
