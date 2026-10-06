package control

import (
	"context"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
)

// Reconciliation never depends on current actor credentials or open
// dispatch. Those are prerequisites for new writes, not for learning
// old effects.
func (c *Coordinator) reconcileMerge(ctx context.Context, m factory.Merge, report *MergeReport) {
	if m.Stage != factory.MergeOpen && m.Stage != factory.MergeFenced {
		return
	}
	p, err := c.Store.PublicationByAssignment(ctx, m.AssignmentID)
	if err != nil {
		mergeError(report, m.ID, "store_unavailable")
		return
	}
	open, _, _, err := c.Store.DispatchState(ctx, m.Repository)
	if err != nil {
		mergeError(report, m.ID, "store_unavailable")
		return
	}
	if m.WithdrawRequested || !open {
		c.withdrawMerge(ctx, &m, report)
		return
	}
	op := &m.Operation
	if op.Attempts != 0 {
		outcome, err := c.Merges.LookupOp(ctx, op.OperationID)
		if err != nil {
			c.mergeCallError(report, m.ID, err)
			return
		}
		if !outcome.NotObserved {
			if !c.adoptMergeObserved(op, outcome, time.Now()) {
				c.finishMerge(ctx, m, factory.MergeFenced, factory.NeedsHuman, factory.MergeReasonUnattributed, report)
				return
			}
			if !c.storeMerge(ctx, &m, report) {
				return
			}
		}
	}
	if !c.adoptMergeReceipt(ctx, &m, report) {
		return
	}
	open, _, _, err = c.Store.DispatchState(ctx, m.Repository)
	if err != nil {
		mergeError(report, m.ID, "store_unavailable")
		return
	}
	if m.WithdrawRequested || !open {
		c.withdrawMerge(ctx, &m, report)
		return
	}
	// A committed merge is retained even if native completion remains
	// unfinished. The ref effect never finishes the bookkeeping alone.
	if op.Effect == factory.OpEffectCommitted {
		c.completeMerge(ctx, &m, report)
		return
	}
	if op.Effect == factory.OpEffectNotCommitted {
		c.finishMerge(ctx, m, factory.MergeFailed, factory.Failed, factory.MergeReasonRefused, report)
		return
	}
	if op.Effect == factory.OpEffectIndeterminate {
		c.finishMerge(ctx, m, factory.MergeFenced, factory.NeedsHuman, factory.MergeReasonFenced, report)
		return
	}
	if m.Stage == factory.MergeFenced {
		m.Stage, m.Outcome, m.Reason, m.FinishedUnix = factory.MergeOpen, "", "", 0
		if !c.storeMerge(ctx, &m, report) {
			return
		}
	}
	if p.Stage != factory.PublicationPublished {
		c.finishMerge(ctx, m, factory.MergeFailed, factory.Failed, factory.MergeReasonInvalid, report)
		return
	}
	if factory.MergeTargetChanged(m, p) {
		// An undriven row follows the newest published head: a
		// correction supersedes the creation tip, so rebase the row
		// onto it instead of failing the merge. Any other target
		// change, or a row that already drove, keeps the invalid
		// failure.
		if op.Attempts == 0 && op.Work == nil && op.Effect == "" &&
			m.Repository == p.Repository && m.Issue == p.Issue &&
			m.PRNumber == p.PRNumber && m.PRID == p.PRID &&
			m.HeadRef == p.PRCreate.HeadRef && m.BaseRef == p.PRCreate.BaseRef &&
			m.BaseOID == p.PRCreate.BaseOID {
			m.HeadOID = p.Candidate
			if !c.storeMerge(ctx, &m, report) {
				return
			}
		} else {
			c.finishMerge(ctx, m, factory.MergeFailed, factory.Failed, factory.MergeReasonInvalid, report)
			return
		}
	}
	policy, revision, allowed := c.mergeAuthority(ctx, m, report)
	if !allowed {
		return
	}
	if op.Cancellation != "" && op.Cancellation != factory.OpCancelNone {
		mergeWait(report, m.ID, "cancellation_pending")
		return
	}
	if op.Effect == factory.OpEffectPending {
		mergeWait(report, m.ID, "merge_pending")
		return
	}
	work := c.mergeWork(m, policy)
	if op.Work == nil {
		if !c.observeMergeEvidence(ctx, &m, work, policy, revision, report) {
			return
		}
	}
	work = op.Work.Apply(work)
	// Recheck the local gate after recording. A concurrent withdrawal
	// cancels this same recorded identity, including a submit delayed
	// until after cancellation.
	open, _, _, err = c.Store.DispatchState(ctx, m.Repository)
	if err != nil {
		mergeError(report, m.ID, "store_unavailable")
		return
	}
	if !open {
		c.withdrawMerge(ctx, &m, report)
		return
	}
	if op.Effect == "" {
		outcome, err := c.Merges.SubmitMerge(ctx, work)
		if err != nil {
			c.mergeCallError(report, m.ID, err)
			return
		}
		if outcome.NotObserved {
			mergeWait(report, m.ID, "submit_unconfirmed")
			return
		}
		if !c.adoptMergeObserved(op, outcome, time.Now()) {
			c.finishMerge(ctx, m, factory.MergeFenced, factory.NeedsHuman, factory.MergeReasonUnattributed, report)
			return
		}
		if !c.storeMerge(ctx, &m, report) {
			return
		}
	}
	if !c.adoptMergeReceipt(ctx, &m, report) {
		return
	}
	switch op.Effect {
	case factory.OpEffectCommitted:
		c.completeMerge(ctx, &m, report)
	case factory.OpEffectNotCommitted:
		c.finishMerge(ctx, m, factory.MergeFailed, factory.Failed, factory.MergeReasonRefused, report)
	case factory.OpEffectIndeterminate:
		c.finishMerge(ctx, m, factory.MergeFenced, factory.NeedsHuman, factory.MergeReasonFenced, report)
	default:
		mergeWait(report, m.ID, "operation_pending")
	}
}

func (c *Coordinator) adoptMergeReceipt(ctx context.Context, m *factory.Merge, report *MergeReport) bool {
	op := &m.Operation
	if op.Effect != factory.OpEffectCommitted {
		return true
	}
	if op.Work == nil {
		c.finishMerge(ctx, *m, factory.MergeFenced, factory.NeedsHuman, factory.MergeReasonUnattributed, report)
		return false
	}
	work := op.Work.Apply(c.mergeWork(*m, factory.RepositoryPolicy{}))
	result, err := c.Merges.AdoptMerge(work, mergeOperationOutcomeOf(*op))
	if err != nil {
		c.finishMerge(ctx, *m, factory.MergeFenced, factory.NeedsHuman, factory.MergeReasonUnattributed, report)
		return false
	}
	op.PRNumber, op.PRID, op.IssueID = result.PRNumber, result.PRID, result.IssueID
	op.HeadRef, op.BaseRef, op.HeadOID, op.BaseOID, op.MergedCommit = result.HeadRef, result.BaseRef, result.HeadOID, result.BaseOID, result.MergedCommit
	return c.storeMerge(ctx, m, report)
}

func (c *Coordinator) adoptMergeObserved(op *factory.MergeOperation, outcome factory.OperationOutcome, now time.Time) bool {
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

func mergeOperationOutcomeOf(op factory.MergeOperation) factory.OperationOutcome {
	return factory.OperationOutcome{Receipt: []byte(op.Receipt), Effect: op.Effect, Cancellation: op.Cancellation, Completion: op.Completion, Reason: op.Reason,
		OperationID: op.OperationID, InstallationID: op.InstallationID, Kind: op.Kind, ActorID: op.ActorID, RepositoryID: op.RepositoryID}
}
