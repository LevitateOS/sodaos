package control

import (
	"context"
	"errors"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
)

// Reconciliation never depends on an export, current actor credential, or open
// dispatch. Those are prerequisites for new writes, not for learning old effects.
func (c *Coordinator) reconcilePublication(ctx context.Context, p factory.Publication, report *PublishReport) {
	if p.Stage == factory.PublicationPublished {
		a, err := c.assignmentForPublication(ctx, p)
		if err != nil {
			publicationError(report, p.ID, "store_unavailable")
			return
		}
		run, err := c.Store.FactoryRun(ctx, p.Run)
		if err != nil {
			publicationError(report, p.ID, "store_unavailable")
			return
		}
		open, _, _, err := c.Store.DispatchState(ctx, p.Repository)
		if err != nil {
			publicationError(report, p.ID, "store_unavailable")
			return
		}
		if p.WithdrawRequested || !open {
			c.withdrawPublication(ctx, a, &p, run, report)
			return
		}
		c.reconcileRecordedReviews(ctx, &p, report)
		corrections := CorrectionReport{}
		c.reconcileRecordedCorrection(ctx, &p, a, &corrections)
		for _, wait := range corrections.Waits {
			publicationWait(report, wait.ID, wait.Reason)
		}
		for _, failure := range corrections.Errors {
			publicationError(report, failure.ID, failure.Reason)
		}
		for _, id := range corrections.Fenced {
			report.Fenced = append(report.Fenced, id)
		}
		if len(p.Corrections) > 0 {
			last := p.Corrections[len(p.Corrections)-1]
			if last.Effect == factory.OpEffectCommitted && last.Completion != factory.OpCompletionComplete &&
				len(corrections.Waits) == 0 && len(corrections.Errors) == 0 {
				publicationWait(report, p.ID, "branch_completion_pending")
			}
		}
		return
	}
	if p.Stage != factory.PublicationOpen && p.Stage != factory.PublicationFenced {
		return
	}
	a, err := c.assignmentForPublication(ctx, p)
	if err != nil {
		publicationError(report, p.ID, "store_unavailable")
		return
	}
	run, err := c.Store.FactoryRun(ctx, p.Run)
	if err != nil {
		publicationError(report, p.ID, "store_unavailable")
		return
	}
	open, _, _, err := c.Store.DispatchState(ctx, p.Repository)
	if err != nil {
		publicationError(report, p.ID, "store_unavailable")
		return
	}
	if p.WithdrawRequested || !open {
		c.withdrawPublication(ctx, a, &p, run, report)
		return
	}
	for _, op := range []*factory.PublicationOperation{&p.Publish, &p.PRCreate} {
		if op.Attempts == 0 {
			continue
		}
		outcome, err := c.Publication.LookupOp(ctx, op.OperationID)
		if err != nil {
			c.publicationCallError(report, p.ID, err)
			return
		}
		if !outcome.NotObserved {
			if !c.adoptObserved(op, outcome, time.Now()) {
				c.finishPublication(ctx, p, factory.PublicationFenced, factory.NeedsHuman, factory.PublishReasonUnattributed, report)
				return
			}
			if !c.storePublication(ctx, &p, report) {
				return
			}
		}
	}
	if !c.adoptPublicationReceipts(ctx, a, &p, run, report) {
		return
	}
	open, _, _, err = c.Store.DispatchState(ctx, p.Repository)
	if err != nil {
		publicationError(report, p.ID, "store_unavailable")
		return
	}
	if p.WithdrawRequested || !open {
		c.withdrawPublication(ctx, a, &p, run, report)
		return
	}
	// A committed PR is retained even if native completion remains unfinished.
	if p.PRCreate.Effect == factory.OpEffectCommitted {
		c.completePublication(ctx, &p, report)
		return
	}
	if terminalPublicationEffect(p.Publish) || terminalPublicationEffect(p.PRCreate) {
		c.finishPublication(ctx, p, factory.PublicationFailed, factory.Failed, factory.PublishReasonRefused, report)
		return
	}
	if p.Publish.Effect == factory.OpEffectIndeterminate || p.PRCreate.Effect == factory.OpEffectIndeterminate {
		c.finishPublication(ctx, p, factory.PublicationFenced, factory.NeedsHuman, factory.PublishReasonFenced, report)
		return
	}
	if p.Stage == factory.PublicationFenced {
		p.Stage, p.Outcome, p.Reason, p.FinishedUnix = factory.PublicationOpen, "", "", 0
		if !c.storePublication(ctx, &p, report) {
			return
		}
	}
	policy, revision, allowed := c.publicationAuthority(ctx, p, report)
	if !allowed {
		return
	}
	op := &p.Publish
	if op.Effect == factory.OpEffectCommitted {
		if op.Completion != factory.OpCompletionComplete {
			publicationWait(report, p.ID, "branch_completion_pending")
			return
		}
		op = &p.PRCreate
	}
	if op.Cancellation != "" && op.Cancellation != factory.OpCancelNone {
		publicationWait(report, p.ID, "cancellation_pending")
		return
	}
	if op.Kind == factory.OpPRCreate && op.Effect == factory.OpEffectPending {
		publicationWait(report, p.ID, "creation_pending")
		return
	}
	bundle, terminal, failed := c.exportCandidate(ctx, p, run)
	if failed != nil {
		publicationWait(report, p.ID, failed.Reason)
		return
	}
	if terminal != nil {
		// A lost reply must be reconciled/cancelled before declaring failure.
		if op.Attempts > 0 {
			publicationWait(report, p.ID, terminal.Reason)
			return
		}
		c.finishPublication(ctx, p, factory.PublicationFailed, factory.Failed, terminal.Reason, report)
		return
	}
	work := c.publicationWork(a, p, run, bundle, policy, op.Kind)
	if op.Work == nil {
		observation, err := c.Publication.ObservePublication(ctx, work)
		if err != nil {
			var refused *factory.PublicationRefusal
			if errors.As(err, &refused) {
				c.finishPublication(ctx, p, factory.PublicationFailed, factory.Failed, factory.PublishReasonInvalid, report)
			} else {
				c.publicationCallError(report, p.ID, err)
			}
			return
		}
		if observation.NativeRev != revision {
			publicationWait(report, p.ID, "native_inputs_changed")
			return
		}
		if observation.TargetRef != factory.PublishBranchName(a.ID) || !factory.ValidCommit(observation.Comparison) {
			publicationWait(report, p.ID, "observation_invalid")
			return
		}
		expected := ""
		if op.Kind == factory.OpPRCreate {
			expected = p.Candidate
		}
		if observation.TargetTip != expected {
			c.finishPublication(ctx, p, factory.PublicationFailed, factory.Failed, factory.PublishReasonTargetOccupied, report)
			return
		}
		now := time.Now()
		work.NativeRev, work.ComparisonOID, work.NotAfter = observation.NativeRev, observation.Comparison, now.Add(10*time.Minute).Unix()
		intent := work.Intent()
		op.OperationID, op.Attempts, op.UpdatedUnix, op.Work = work.OperationID, 1, now.Unix(), &intent
		p.NativeRev, p.Comparison, p.TargetTip, p.ObservedUnix = observation.NativeRev, observation.Comparison, observation.TargetTip, now.Unix()
		if !c.storePublication(ctx, &p, report) {
			return
		}
	}
	work = op.Work.Apply(work)
	// Recheck the local gate after recording. A concurrent withdrawal cancels this
	// same recorded identity, including a submit delayed until after cancellation.
	open, _, _, err = c.Store.DispatchState(ctx, p.Repository)
	if err != nil {
		publicationError(report, p.ID, "store_unavailable")
		return
	}
	if !open {
		c.withdrawPublication(ctx, a, &p, run, report)
		return
	}
	var outcome factory.OperationOutcome
	if op.Effect == "" {
		if op.Kind == factory.OpRefPublish {
			outcome, err = c.Publication.SubmitPublish(ctx, work)
		} else {
			outcome, err = c.Publication.SubmitPRCreate(ctx, work)
		}
		if err != nil {
			c.publicationCallError(report, p.ID, err)
			return
		}
		if outcome.NotObserved {
			publicationWait(report, p.ID, "submit_unconfirmed")
			return
		}
		if !c.adoptObserved(op, outcome, time.Now()) {
			c.finishPublication(ctx, p, factory.PublicationFenced, factory.NeedsHuman, factory.PublishReasonUnattributed, report)
			return
		}
		if !c.storePublication(ctx, &p, report) {
			return
		}
	}
	if op.Kind == factory.OpRefPublish && op.Effect == factory.OpEffectPending && (op.Cancellation == "" || op.Cancellation == factory.OpCancelNone) {
		outcome, err = c.Publication.PushBranch(ctx, work)
		if err != nil {
			c.publicationCallError(report, p.ID, err)
			return
		}
		if outcome.NotObserved {
			publicationWait(report, p.ID, "push_unconfirmed")
			return
		}
		if !c.adoptObserved(op, outcome, time.Now()) {
			c.finishPublication(ctx, p, factory.PublicationFenced, factory.NeedsHuman, factory.PublishReasonUnattributed, report)
			return
		}
		if !c.storePublication(ctx, &p, report) {
			return
		}
	}
	if !c.adoptPublicationReceipts(ctx, a, &p, run, report) {
		return
	}
	switch op.Effect {
	case factory.OpEffectCommitted:
		if op.Kind == factory.OpPRCreate {
			c.completePublication(ctx, &p, report)
		} else if op.Completion == factory.OpCompletionComplete {
			// The distinct PR phase gets its own fresh acceptance/native bracket.
			c.reconcilePublication(ctx, p, report)
		} else {
			publicationWait(report, p.ID, "branch_completion_pending")
		}
	case factory.OpEffectNotCommitted:
		c.finishPublication(ctx, p, factory.PublicationFailed, factory.Failed, factory.PublishReasonRefused, report)
	case factory.OpEffectIndeterminate:
		c.finishPublication(ctx, p, factory.PublicationFenced, factory.NeedsHuman, factory.PublishReasonFenced, report)
	default:
		publicationWait(report, p.ID, "operation_pending")
	}
}

func terminalPublicationEffect(op factory.PublicationOperation) bool {
	return op.Effect == factory.OpEffectNotCommitted
}

func (c *Coordinator) adoptPublicationReceipts(ctx context.Context, a factory.Assignment, p *factory.Publication, run factory.Run, report *PublishReport) bool {
	for _, op := range []*factory.PublicationOperation{&p.Publish, &p.PRCreate} {
		if op.Effect != factory.OpEffectCommitted {
			continue
		}
		if op.Work == nil {
			c.finishPublication(ctx, *p, factory.PublicationFenced, factory.NeedsHuman, factory.PublishReasonUnattributed, report)
			return false
		}
		work := op.Work.Apply(c.publicationWork(a, *p, run, nil, factory.RepositoryPolicy{}, op.Kind))
		var err error
		if op.Kind == factory.OpRefPublish {
			_, err = c.Publication.AdoptBranch(work, operationOutcomeOf(*op))
		} else {
			var result factory.PRCreationOutcome
			result, err = c.Publication.AdoptPRCreation(work, operationOutcomeOf(*op))
			if err == nil {
				op.PRNumber, op.PRID, op.IssueID = result.PRNumber, result.PRID, result.IssueID
				op.HeadRef, op.BaseRef, op.HeadOID, op.BaseOID = result.HeadRef, result.BaseRef, result.HeadOID, result.BaseOID
				p.PRNumber, p.PRID = result.PRNumber, result.PRID
			}
		}
		if err != nil {
			c.finishPublication(ctx, *p, factory.PublicationFenced, factory.NeedsHuman, factory.PublishReasonUnattributed, report)
			return false
		}
	}
	return c.storePublication(ctx, p, report)
}
