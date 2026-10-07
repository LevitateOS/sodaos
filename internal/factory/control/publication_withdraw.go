package control

import (
	"context"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
)

func (c *Coordinator) cancelPublications(ctx context.Context, repository, issue int64, acceptance string) factory.PublicationWithdrawal {
	result := factory.PublicationWithdrawal{Publications: []string{}, Operations: []string{}}
	pending, err := c.Store.OutstandingPublications(ctx, repository, publishPassLimit+1)
	if err != nil {
		result.Pending = true
		return result
	}
	if len(pending) > publishPassLimit {
		result.Pending = true
		pending = pending[:publishPassLimit]
	}
	for _, p := range pending {
		if (issue != 0 && p.Issue != issue) || (acceptance != "" && p.Acceptance != acceptance) {
			continue
		}
		result.Publications = append(result.Publications, p.ID)
		for _, op := range []factory.PublicationOperation{p.Publish, p.PRCreate} {
			if op.OperationID != "" {
				result.Operations = append(result.Operations, op.OperationID)
			}
		}
		for _, op := range p.Corrections {
			if op.OperationID != "" {
				result.Operations = append(result.Operations, op.OperationID)
			}
		}
		a, err := c.assignmentForPublication(ctx, p)
		if err != nil {
			result.Pending = true
			continue
		}
		run, err := c.Store.FactoryRun(ctx, p.Run)
		if err != nil {
			result.Pending = true
			continue
		}
		report := PublishReport{Published: []PublishLink{}}
		if c.Publication == nil {
			result.Pending = true
			continue
		}
		c.withdrawPublication(ctx, a, &p, run, &report)
		current, err := c.Store.PublicationByAssignment(ctx, p.AssignmentID)
		if err != nil || current.Stage == factory.PublicationOpen || current.Stage == factory.PublicationFenced || len(report.Errors) > 0 || len(report.Waits) > 0 {
			result.Pending = true
		}
	}
	return result
}

// A successful host launch returns after the supervised CLI has retired. Only
// terminal replies enter settlement; a running duplicate remains supervised.
func (c *Coordinator) publishAfterDispatch(ctx context.Context, report DispatchReport) {
	if c.Publication == nil && c.Checks == nil && c.Merges == nil {
		return
	}
	bounded, stop := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Minute)
	defer stop()
	for _, launched := range report.Launched {
		if launched.Phase != project.FactoryCompleted && launched.Phase != project.FactoryFailed && launched.Phase != project.FactoryStopped {
			continue
		}
		run, err := c.Store.FactoryRun(bounded, launched.RunID)
		if err == nil && !run.Reconciled {
			c.settleRun(bounded, run)
		}
	}
	c.PublishPass(bounded)
	c.CheckPass(bounded)
	c.MergePass(bounded)
}

// Withdrawal stays open while any cancellation or effect is unknown. Native
// committed receipts are adopted even when cancellation arrived too late.
func (c *Coordinator) withdrawPublication(ctx context.Context, a factory.Assignment, p *factory.Publication, run factory.Run, report *PublishReport) {
	p.WithdrawRequested = true
	if !c.storePublication(ctx, p, report) {
		return
	}
	confirmed := true
	for _, op := range []*factory.PublicationOperation{&p.Publish, &p.PRCreate} {
		if op.Attempts == 0 {
			continue
		}
		outcome, err := c.Publication.CancelOp(ctx, op.OperationID)
		if err != nil || outcome.NotObserved {
			confirmed = false
			continue
		}
		if !c.adoptObserved(op, outcome, time.Now()) {
			confirmed = false
			continue
		}
		if outcome.Effect != factory.OpEffectCommitted && outcome.Effect != factory.OpEffectNotCommitted {
			confirmed = false
		}
	}
	for i := range p.Corrections {
		op := &p.Corrections[i]
		if op.Attempts == 0 {
			continue
		}
		outcome, err := c.Publication.CancelOp(ctx, op.OperationID)
		if err != nil || outcome.NotObserved {
			confirmed = false
			continue
		}
		if !c.adoptObserved(op, outcome, time.Now()) {
			confirmed = false
			continue
		}
		if outcome.Effect != factory.OpEffectCommitted && outcome.Effect != factory.OpEffectNotCommitted {
			confirmed = false
		}
		if outcome.Effect == factory.OpEffectCommitted {
			if op.Work == nil {
				confirmed = false
				continue
			}
			p.Candidate = op.Work.Candidate
			work := op.Work.Apply(factory.PublicationWork{AssignmentID: p.AssignmentID, Publication: p.ID})
			if _, err := c.Publication.AdoptBranch(work, operationOutcomeOf(*op)); err != nil {
				confirmed = false
			}
		}
	}
	if !c.storePublication(ctx, p, report) {
		return
	}
	if !c.adoptPublicationReceipts(ctx, a, p, run, report) {
		return
	}
	for _, op := range p.Corrections {
		if op.Effect == factory.OpEffectCommitted && op.Completion != factory.OpCompletionComplete {
			publicationWait(report, p.ID, "branch_completion_pending")
			return
		}
	}
	if !confirmed {
		publicationWait(report, p.ID, "cancellation_pending")
		return
	}
	if p.PRCreate.Effect == factory.OpEffectCommitted {
		c.completePublication(ctx, p, report)
		return
	}
	if p.Publish.Effect == factory.OpEffectCommitted && p.Publish.Completion != factory.OpCompletionComplete {
		publicationWait(report, p.ID, "branch_completion_pending")
		return
	}
	c.finishPublication(ctx, *p, factory.PublicationWithdrawn, factory.Cancelled, factory.PublishReasonWithdrawn, report)
}

func (c *Coordinator) adoptObserved(op *factory.PublicationOperation, outcome factory.OperationOutcome, now time.Time) bool {
	if outcome.NotObserved || outcome.OperationID != op.OperationID || op.Work == nil {
		return false
	}
	if op.InstallationID != "" && op.InstallationID != outcome.InstallationID {
		return false
	}
	tombstone := outcome.Kind == "" && outcome.Effect == factory.OpEffectNotCommitted && outcome.Cancellation == factory.OpCancelCancelled
	if !tombstone && (outcome.Kind != op.Kind || outcome.ActorID != op.Work.ActorID || outcome.RepositoryID != op.Work.Repository || outcome.InstallationID == "") {
		return false
	}
	if (op.Effect == factory.OpEffectCommitted || op.Effect == factory.OpEffectNotCommitted) && op.Effect != outcome.Effect {
		return false
	}
	if op.Receipt != "" && op.Receipt != string(outcome.Receipt) {
		return false
	}
	op.Effect, op.Cancellation, op.Completion, op.Reason = outcome.Effect, outcome.Cancellation, outcome.Completion, outcome.Reason
	op.Receipt, op.UpdatedUnix = string(outcome.Receipt), now.Unix()
	op.InstallationID, op.ActorID, op.RepositoryID = outcome.InstallationID, outcome.ActorID, outcome.RepositoryID
	return true
}
func operationOutcomeOf(op factory.PublicationOperation) factory.OperationOutcome {
	return factory.OperationOutcome{Receipt: []byte(op.Receipt), Effect: op.Effect, Cancellation: op.Cancellation, Completion: op.Completion, Reason: op.Reason,
		OperationID: op.OperationID, InstallationID: op.InstallationID, Kind: op.Kind, ActorID: op.ActorID, RepositoryID: op.RepositoryID}
}
func (c *Coordinator) storePublication(ctx context.Context, p *factory.Publication, report *PublishReport) bool {
	p.Revision++
	if err := c.Store.UpdatePublication(ctx, *p); err != nil {
		publicationError(report, p.ID, "store_conflict")
		return false
	}
	return true
}
func (c *Coordinator) finishPublication(ctx context.Context, p factory.Publication, stage string, outcome factory.Outcome, reason string, report *PublishReport) {
	p.Stage, p.Outcome, p.Reason, p.FinishedUnix = stage, outcome, reason, time.Now().Unix()
	if !c.storePublication(ctx, &p, report) {
		return
	}
	switch stage {
	case factory.PublicationPublished:
		report.Published = append(report.Published, PublishLink{AssignmentID: p.AssignmentID, PublicationID: p.ID, Repository: p.Repository, Issue: p.Issue, PRNumber: p.PRNumber, PRID: p.PRID})
	case factory.PublicationWithdrawn:
		report.Withdrawn = append(report.Withdrawn, p.ID)
	case factory.PublicationFenced:
		report.Fenced = append(report.Fenced, p.ID)
	}
}
