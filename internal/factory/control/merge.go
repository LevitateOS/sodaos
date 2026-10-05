package control

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/store"
)

// MergeExecutor is the coordinator's narrow merge surface: evidence
// observation with fresh native reads, the single conditional merge
// with lost-reply reconciliation, receipt adoption, and completion
// confirmation. Implementations own all native effects; the
// coordinator owns stages, receipts and withdrawal. Every method
// reports its verdict as either an adopted outcome, a terminal
// PublicationRefusal, a retryable PublicationWait, or a transport
// error reconciled by a later pass.
type MergeExecutor interface {
	ObserveMerge(ctx context.Context, work factory.MergeWork) (factory.MergeObservation, error)
	SubmitMerge(ctx context.Context, work factory.MergeWork) (factory.OperationOutcome, error)
	LookupOp(ctx context.Context, operationID string) (factory.OperationOutcome, error)
	CancelOp(ctx context.Context, operationID string) (factory.OperationOutcome, error)
	AdoptMerge(work factory.MergeWork, outcome factory.OperationOutcome) (factory.MergeOutcome, error)
	ObserveCompletion(ctx context.Context, work factory.MergeWork) (factory.MergeConfirmation, error)
}

// mergePassLimit bounds one merge pass: recovered merges plus newly
// mergeable publications.
const mergePassLimit = 256

// MergeLink is one exact merged PR with its dispatch receipts.
type MergeLink struct {
	AssignmentID  string `json:"assignment_id"`
	PublicationID string `json:"publication_id"`
	MergeID       string `json:"merge_id"`
	MergedCommit  string `json:"merged_commit"`
	Repository    int64  `json:"repository,string"`
	Issue         int64  `json:"issue,string"`
	PRNumber      int64  `json:"pr_number,string"`
	PRID          int64  `json:"pr_id,string"`
}

// MergeWait is one merge that did not advance this pass and why.
type MergeWait struct {
	ID     string `json:"id"`
	Reason string `json:"reason"`
	Detail string `json:"detail,omitempty"`
}

// MergeError is one unexpected merge infrastructure failure. The
// affected merge keeps its recorded state; a later pass retries.
type MergeError struct {
	ID     string `json:"id"`
	Reason string `json:"reason"`
	Detail string `json:"detail,omitempty"`
}

// MergeReport is the durable outcome of one merge pass: the PRs it
// merged, the merges it fenced or withdrew, and every merge that
// waited or errored.
type MergeReport struct {
	Merged      []MergeLink  `json:"merged"`
	Fenced      []string     `json:"fenced,omitempty"`
	Withdrawn   []string     `json:"withdrawn,omitempty"`
	Waits       []MergeWait  `json:"waits,omitempty"`
	Errors      []MergeError `json:"errors,omitempty"`
	Unavailable bool         `json:"unavailable,omitempty"`
}

// MergePass reads recorded operations before considering any further
// write. The single merge phase has one immutable native authorization;
// a terminal refusal never changes identity or erases its receipt.
func (c *Coordinator) MergePass(ctx context.Context) MergeReport {
	report := MergeReport{Merged: []MergeLink{}}
	if c.Merges == nil {
		report.Unavailable = true
		return report
	}
	pending, err := c.Store.OpenMerges(ctx, mergePassLimit)
	if err != nil {
		mergeError(&report, "", "store_unavailable")
		return report
	}
	for _, m := range pending {
		c.reconcileMerge(ctx, m, &report)
	}
	mergeable, err := c.Store.MergeablePublications(ctx, mergePassLimit)
	if err != nil {
		mergeError(&report, "", "store_unavailable")
		return report
	}
	for _, p := range mergeable {
		// Merge rows open only behind a passing verdict on the exact
		// current head and base (the review-cycle rule): opening one
		// early would strand the publication, since assessed checks
		// never revisit a publication that already holds a merge row
		// and a failed verdict on the row's head fails it terminally.
		if !c.checkCurrent(ctx, p) {
			continue
		}
		c.mergeOne(ctx, p, &report)
	}
	return report
}

func mergeError(report *MergeReport, id, reason string) {
	report.Errors = append(report.Errors, MergeError{ID: id, Reason: reason})
}
func mergeWait(report *MergeReport, id, reason string) {
	report.Waits = append(report.Waits, MergeWait{ID: id, Reason: reason})
}

func (c *Coordinator) mergeOne(ctx context.Context, p factory.Publication, report *MergeReport) {
	if p.Stage != factory.PublicationPublished || p.PRNumber <= 0 || p.PRID <= 0 || p.PRCreate.Work == nil {
		return
	}
	if _, err := c.Store.MergeByPublication(ctx, p.ID); err == nil {
		return
	} else if !errors.Is(err, store.ErrNotFound) {
		mergeError(report, p.ID, "store_unavailable")
		return
	}
	policy, err := c.Store.RepositoryPolicy(ctx, p.Repository)
	if err != nil {
		mergeError(report, p.ID, "store_unavailable")
		return
	}
	m := factory.Merge{
		Operation: factory.MergeOperation{Kind: factory.OpMerge},
		Authority: p.Authority, ID: factory.NewID(), PublicationID: p.ID,
		AssignmentID: p.AssignmentID, ProjectID: p.ProjectID, Role: p.Role,
		Acceptance: p.Acceptance, HeadRef: p.PRCreate.HeadRef, BaseRef: p.PRCreate.BaseRef,
		// The merge follows the latest published head: the creation tip
		// before any correction, the newest correction after one.
		HeadOID: p.Candidate, BaseOID: p.PRCreate.BaseOID,
		Stage: factory.MergeOpen, Repository: p.Repository, Issue: p.Issue,
		PRNumber: p.PRNumber, PRID: p.PRID, IssueID: p.PRCreate.IssueID,
		PRAuthorID: p.PRCreate.Work.ActorID, ReviewerID: policy.Review.ActorID,
		CreatedUnix: time.Now().Unix(),
	}
	if err := c.Store.RecordMerge(ctx, m); err != nil {
		mergeError(report, p.ID, "store_unavailable")
		return
	}
	c.reconcileMerge(ctx, m, report)
}

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

// observeMergeEvidence binds the exact fresh evidence behind one merge
// submit: the independent approval on the exact head, the persisted
// ST11 pass under the current adopted definitions, and a fresh
// check verdict on the merge bracket so regressed checks cannot ride
// an older pass. Missing evidence waits for a later pass; rejected or
// misidentified candidates fail. The bound intent is persisted before
// any submit.
func (c *Coordinator) observeMergeEvidence(ctx context.Context, m *factory.Merge, work factory.MergeWork, policy factory.RepositoryPolicy, revision int64, report *MergeReport) bool {
	observation, err := c.Merges.ObserveMerge(ctx, work)
	if err != nil {
		var refused *factory.PublicationRefusal
		if errors.As(err, &refused) {
			c.finishMerge(ctx, *m, factory.MergeFailed, factory.Failed, factory.MergeReasonInvalid, report)
		} else {
			c.mergeCallError(report, m.ID, err)
		}
		return false
	}
	if observation.NativeRev != revision {
		mergeWait(report, m.ID, "native_inputs_changed")
		return false
	}
	if observation.PRID != m.PRID || observation.PRNumber != m.PRNumber || observation.IssueID != m.IssueID ||
		observation.PRAuthorID != m.PRAuthorID || observation.HeadRef != m.HeadRef || observation.BaseRef != m.BaseRef ||
		observation.HeadOID != m.HeadOID || observation.BaseOID != m.BaseOID || observation.ReviewerID != m.ReviewerID || observation.ReviewID <= 0 {
		mergeWait(report, m.ID, "observation_invalid")
		return false
	}
	assessment, err := c.Store.CheckAssessment(ctx, m.Repository, m.PRNumber)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			mergeWait(report, m.ID, "check_evidence_missing")
		} else {
			mergeError(report, m.ID, "store_unavailable")
		}
		return false
	}
	work.AssessmentRevision, work.ReviewID = assessment.Revision, observation.ReviewID
	if err := factory.VerifyMergeCheckEvidence(work, assessment, policy); err != nil {
		// A failed verdict on the exact head is terminal: new code
		// means a new publication cycle. Anything else waits for a
		// fresh assessment on this head.
		if assessment.Verdict == factory.CheckFailed && assessment.HeadOID == m.HeadOID && assessment.BaseOID == m.BaseOID {
			c.finishMerge(ctx, *m, factory.MergeFailed, factory.Failed, factory.MergeReasonInvalid, report)
		} else {
			mergeWait(report, m.ID, "check_evidence_stale")
		}
		return false
	}
	target := factory.CheckTarget{Repository: m.Repository, PRNumber: m.PRNumber, PRID: m.PRID, IssueID: m.IssueID, HeadRef: m.HeadRef, BaseRef: m.BaseRef, HeadOID: m.HeadOID, BaseOID: m.BaseOID}
	adopted := factory.AdoptedChecks{Checks: append([]string(nil), policy.Checks...), PolicyRevision: policy.Revision, Digest: factory.ChecksDigest(policy.Checks)}
	fresh, err := factory.VerifyChecks(target, adopted, policy, observation.Checks, time.Now().Unix())
	if err != nil {
		mergeError(report, m.ID, "check_evidence_invalid")
		return false
	}
	switch fresh.Verdict {
	case factory.CheckPass:
	case factory.CheckPending:
		mergeWait(report, m.ID, "check_evidence_pending")
		return false
	case factory.CheckRefused:
		mergeWait(report, m.ID, "check_evidence_hidden")
		return false
	default:
		c.finishMerge(ctx, *m, factory.MergeFailed, factory.Failed, factory.MergeReasonInvalid, report)
		return false
	}
	now := time.Now()
	work.NativeRev, work.NotAfter = observation.NativeRev, now.Add(10*time.Minute).Unix()
	intent := work.Intent()
	op := &m.Operation
	op.OperationID, op.Attempts, op.UpdatedUnix, op.Work = work.OperationID, 1, now.Unix(), &intent
	m.NativeRev, m.ObservedUnix = observation.NativeRev, now.Unix()
	return c.storeMerge(ctx, m, report)
}

// mergeAuthority validates Soda policy and accepted input at the native
// revision that the later merge observation must match.
func (c *Coordinator) mergeAuthority(ctx context.Context, m factory.Merge, report *MergeReport) (factory.RepositoryPolicy, int64, bool) {
	var empty factory.RepositoryPolicy
	effective, err := c.EffectiveAuthority(ctx, m.Repository)
	if err != nil {
		mergeError(report, m.ID, "store_unavailable")
		return empty, 0, false
	}
	if !effective.Effective {
		mergeWait(report, m.ID, "authority_ineffective")
		return empty, 0, false
	}
	policy, err := c.Store.RepositoryPolicy(ctx, m.Repository)
	if err != nil {
		mergeError(report, m.ID, "store_unavailable")
		return empty, 0, false
	}
	current := effective.Authority
	// Capacity changes only bound future reservations, per the existing grant contract.
	current.Capacity = m.Authority.Capacity
	current.RequirementsID, err = c.Store.RequirementHead(ctx, m.ProjectID)
	if err != nil {
		mergeError(report, m.ID, "store_unavailable")
		return empty, 0, false
	}
	current.ApprovalID, err = c.Store.ApprovalHead(ctx, m.ProjectID)
	if err != nil {
		mergeError(report, m.ID, "store_unavailable")
		return empty, 0, false
	}
	if current != m.Authority || policy.TargetBranch != m.BaseRef || policy.Review.ActorID != m.ReviewerID {
		mergeWait(report, m.ID, "authority_changed")
		return empty, 0, false
	}
	status, err := c.AcceptanceStatus(ctx, m.Repository, strconv.FormatInt(m.Issue, 10))
	if err != nil {
		mergeWait(report, m.ID, "accepted_inputs_unavailable")
		return empty, 0, false
	}
	if !status.Valid || status.Acceptance == nil || status.Acceptance.ID != m.Acceptance {
		mergeWait(report, m.ID, "accepted_inputs_changed")
		return empty, 0, false
	}
	return policy, status.Revision, true
}

func (c *Coordinator) mergeWork(m factory.Merge, policy factory.RepositoryPolicy) factory.MergeWork {
	return factory.MergeWork{
		MergeID: m.ID, PublicationID: m.PublicationID,
		OperationID: factory.MergeOperationID(m.PublicationID, 1), AuthRevision: factory.MergeAuthRevision(m.AssignmentID, m.ID, m.Revision),
		HeadRef: m.HeadRef, BaseRef: m.BaseRef, HeadOID: m.HeadOID, BaseOID: m.BaseOID,
		Repository: m.Repository, Issue: m.Issue, PRNumber: m.PRNumber, PRID: m.PRID, IssueID: m.IssueID,
		PRAuthorID: m.PRAuthorID, ReviewerID: m.ReviewerID, ActorID: policy.Merge.ActorID,
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

// completeMerge confirms the native bookkeeping behind a committed,
// completed merge before finishing: the PR merged to the exact head,
// the base tip carries it, and the issue closed. Confirmed completion
// then reassesses dependants so an eligible one becomes runnable.
func (c *Coordinator) completeMerge(ctx context.Context, m *factory.Merge, report *MergeReport) {
	op := &m.Operation
	if op.Completion != factory.OpCompletionComplete {
		mergeWait(report, m.ID, "native_completion_pending")
		return
	}
	if op.Work == nil {
		c.finishMerge(ctx, *m, factory.MergeFenced, factory.NeedsHuman, factory.MergeReasonUnattributed, report)
		return
	}
	work := op.Work.Apply(c.mergeWork(*m, factory.RepositoryPolicy{}))
	confirmation, err := c.Merges.ObserveCompletion(ctx, work)
	if err != nil {
		var refused *factory.PublicationRefusal
		if errors.As(err, &refused) {
			c.finishMerge(ctx, *m, factory.MergeFenced, factory.NeedsHuman, factory.MergeReasonUnattributed, report)
		} else {
			c.mergeCallError(report, m.ID, err)
		}
		return
	}
	if confirmation.MergedCommit != m.HeadOID || confirmation.BaseTip != m.HeadOID || confirmation.MergerID != work.ActorID || !confirmation.IssueClosed || confirmation.MergedUnix <= 0 || confirmation.ClosedUnix <= 0 {
		c.finishMerge(ctx, *m, factory.MergeFenced, factory.NeedsHuman, factory.MergeReasonUnattributed, report)
		return
	}
	m.MergedCommit, m.MergedUnix, m.ClosedUnix = confirmation.MergedCommit, confirmation.MergedUnix, confirmation.ClosedUnix
	c.finishMerge(ctx, *m, factory.MergeMerged, factory.Succeeded, factory.MergeReasonMerged, report)
	// Best-effort dependant release: confirmed completion reassesses
	// dependants, and queued work launches within current limits.
	// Failures wait for the next trigger; the merge above already
	// recorded.
	_, _ = c.assessCascade(ctx, m.Repository, m.Issue, make(map[factory.DependenceRef]bool))
	// Completion itself is new dependant input (the code-prereq
	// completion record): release dependants even when the merged
	// issue's own verdict is unchanged, mirroring run settlement.
	c.assessDispatchDependants(ctx, m.Repository, m.Issue)
	c.dispatchAfterIntake(ctx)
}

func (c *Coordinator) mergeCallError(report *MergeReport, id string, err error) {
	var wait *factory.PublicationWait
	var refused *factory.PublicationRefusal
	switch {
	case errors.As(err, &wait):
		mergeWait(report, id, wait.Reason)
	case errors.As(err, &refused):
		mergeWait(report, id, refused.Reason)
	default:
		mergeError(report, id, "native_unavailable")
	}
}

func (c *Coordinator) cancelRepositoryMerges(ctx context.Context, repository int64) factory.MergeWithdrawal {
	return c.cancelMerges(ctx, repository, 0, "")
}

func (c *Coordinator) cancelAcceptanceMerges(ctx context.Context, repository, issue int64, acceptance string) factory.MergeWithdrawal {
	return c.cancelMerges(ctx, repository, issue, acceptance)
}

func (c *Coordinator) cancelMerges(ctx context.Context, repository, issue int64, acceptance string) factory.MergeWithdrawal {
	result := factory.MergeWithdrawal{Merges: []string{}, Operations: []string{}}
	pending, err := c.Store.OutstandingMerges(ctx, repository, mergePassLimit+1)
	if err != nil {
		result.Pending = true
		return result
	}
	if len(pending) > mergePassLimit {
		result.Pending = true
		pending = pending[:mergePassLimit]
	}
	for _, m := range pending {
		if (issue != 0 && m.Issue != issue) || (acceptance != "" && m.Acceptance != acceptance) {
			continue
		}
		result.Merges = append(result.Merges, m.ID)
		if m.Operation.OperationID != "" {
			result.Operations = append(result.Operations, m.Operation.OperationID)
		}
		if c.Merges == nil {
			result.Pending = true
			continue
		}
		report := MergeReport{Merged: []MergeLink{}}
		c.withdrawMerge(ctx, &m, &report)
		current, err := c.Store.MergeByPublication(ctx, m.PublicationID)
		if err != nil || current.Stage == factory.MergeOpen || current.Stage == factory.MergeFenced || len(report.Errors) > 0 {
			result.Pending = true
		}
	}
	return result
}

// Withdrawal stays open while any cancellation or effect is unknown. A
// natively committed merge still confirms its completion: withdrawal
// cannot erase the effect, only finish its bookkeeping honestly.
func (c *Coordinator) withdrawMerge(ctx context.Context, m *factory.Merge, report *MergeReport) {
	m.WithdrawRequested = true
	if !c.storeMerge(ctx, m, report) {
		return
	}
	op := &m.Operation
	confirmed := true
	if op.Attempts != 0 {
		outcome, err := c.Merges.CancelOp(ctx, op.OperationID)
		if err != nil || outcome.NotObserved {
			confirmed = false
		} else {
			if !c.adoptMergeObserved(op, outcome, time.Now()) {
				confirmed = false
			}
			if outcome.Effect != factory.OpEffectCommitted && outcome.Effect != factory.OpEffectNotCommitted {
				confirmed = false
			}
		}
	}
	if !c.storeMerge(ctx, m, report) {
		return
	}
	if !c.adoptMergeReceipt(ctx, m, report) {
		return
	}
	if !confirmed {
		mergeWait(report, m.ID, "cancellation_pending")
		return
	}
	if op.Effect == factory.OpEffectCommitted {
		c.completeMerge(ctx, m, report)
		return
	}
	c.finishMerge(ctx, *m, factory.MergeWithdrawn, factory.Cancelled, factory.MergeReasonWithdrawn, report)
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

func (c *Coordinator) storeMerge(ctx context.Context, m *factory.Merge, report *MergeReport) bool {
	m.Revision++
	if err := c.Store.UpdateMerge(ctx, *m); err != nil {
		mergeError(report, m.ID, "store_conflict")
		return false
	}
	return true
}

func (c *Coordinator) finishMerge(ctx context.Context, m factory.Merge, stage string, outcome factory.Outcome, reason string, report *MergeReport) {
	m.Stage, m.Outcome, m.Reason, m.FinishedUnix = stage, outcome, reason, time.Now().Unix()
	if !c.storeMerge(ctx, &m, report) {
		return
	}
	switch stage {
	case factory.MergeMerged:
		report.Merged = append(report.Merged, MergeLink{AssignmentID: m.AssignmentID, PublicationID: m.PublicationID, MergeID: m.ID, MergedCommit: m.MergedCommit, Repository: m.Repository, Issue: m.Issue, PRNumber: m.PRNumber, PRID: m.PRID})
	case factory.MergeWithdrawn:
		report.Withdrawn = append(report.Withdrawn, m.ID)
	case factory.MergeFenced:
		report.Fenced = append(report.Fenced, m.ID)
	}
}
