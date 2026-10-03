package control

import (
	"context"
	"encoding/base64"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
)

// PublicationExecutor is the coordinator's narrow publication surface:
// candidate validation with fresh native observations, the two
// conditional operations with lost-reply reconciliation, and receipt
// adoption. Implementations own all native effects; the coordinator owns
// stages, receipts and withdrawal. Every method reports its verdict as
// either an adopted outcome, a terminal PublicationRefusal, a retryable
// PublicationWait, or a transport error reconciled by a later pass.
type PublicationExecutor interface {
	ObservePublication(ctx context.Context, work factory.PublicationWork) (factory.PublicationObservation, error)
	SubmitPublish(ctx context.Context, work factory.PublicationWork) (factory.OperationOutcome, error)
	PushBranch(ctx context.Context, work factory.PublicationWork) (factory.OperationOutcome, error)
	SubmitPRCreate(ctx context.Context, work factory.PublicationWork) (factory.OperationOutcome, error)
	LookupOp(ctx context.Context, operationID string) (factory.OperationOutcome, error)
	CancelOp(ctx context.Context, operationID string) (factory.OperationOutcome, error)
	AdoptBranch(work factory.PublicationWork, outcome factory.OperationOutcome) (factory.BranchOutcome, error)
	AdoptPRCreation(work factory.PublicationWork, outcome factory.OperationOutcome) (factory.PRCreationOutcome, error)
}

// publishPassLimit bounds one publication pass: recovered publications
// plus newly publishable assignments.
const publishPassLimit = 256

// PublishLink is one exact published PR with its dispatch receipts.
type PublishLink struct {
	AssignmentID  string `json:"assignment_id"`
	PublicationID string `json:"publication_id"`
	Repository    int64  `json:"repository,string"`
	Issue         int64  `json:"issue,string"`
	PRNumber      int64  `json:"pr_number,string"`
	PRID          int64  `json:"pr_id,string"`
}

// PublishWait is one publication that did not advance this pass and why.
type PublishWait struct {
	ID     string `json:"id"`
	Reason string `json:"reason"`
	Detail string `json:"detail,omitempty"`
}

// PublishError is one unexpected publication infrastructure failure. The
// affected publication keeps its recorded state; a later pass retries.
type PublishError struct {
	ID     string `json:"id"`
	Reason string `json:"reason"`
	Detail string `json:"detail,omitempty"`
}

// PublishReport is the durable outcome of one publication pass: the PRs
// it linked, the publications it fenced or withdrew, and every
// publication that waited or errored.
type PublishReport struct {
	Published   []PublishLink  `json:"published"`
	Fenced      []string       `json:"fenced,omitempty"`
	Withdrawn   []string       `json:"withdrawn,omitempty"`
	Waits       []PublishWait  `json:"waits,omitempty"`
	Errors      []PublishError `json:"errors,omitempty"`
	Unavailable bool           `json:"unavailable,omitempty"`
}

// PublishPass reads recorded operations before considering any further write.
// Each phase has one immutable native authorization; a terminal refusal never
// changes identity or erases its receipt.
func (c *Coordinator) PublishPass(ctx context.Context) PublishReport {
	report := PublishReport{Published: []PublishLink{}}
	if c.Publication == nil {
		report.Unavailable = true
		return report
	}
	pending, err := c.Store.OpenPublications(ctx, publishPassLimit)
	if err != nil {
		publicationError(&report, "", "store_unavailable")
		return report
	}
	for _, p := range pending {
		c.reconcilePublication(ctx, p, &report)
	}
	assignments, err := c.Store.PublishableAssignments(ctx, publishPassLimit)
	if err != nil {
		publicationError(&report, "", "store_unavailable")
		return report
	}
	for _, a := range assignments {
		c.publishOne(ctx, a, &report)
	}
	return report
}

func publicationError(report *PublishReport, id, reason string) {
	report.Errors = append(report.Errors, PublishError{ID: id, Reason: reason})
}
func publicationWait(report *PublishReport, id, reason string) {
	report.Waits = append(report.Waits, PublishWait{ID: id, Reason: reason})
}

func (c *Coordinator) publishAfterSettle(ctx context.Context, a factory.Assignment) {
	if c.Publication == nil || a.Role != project.RoleCoder || a.Result == nil || !a.Result.Reported || a.Result.Status != "completed" {
		return
	}
	report := PublishReport{Published: []PublishLink{}}
	c.publishOne(ctx, a, &report)
	c.progressAfterPublish(ctx, a.ID)
}

func (c *Coordinator) publishOne(ctx context.Context, a factory.Assignment, report *PublishReport) {
	if a.Role != project.RoleCoder || a.Result == nil || !a.Result.Reported || a.Result.Status != "completed" || !factory.ValidCommit(a.Result.Candidate) {
		return
	}
	if _, err := c.Store.PublicationByAssignment(ctx, a.ID); err == nil {
		return
	} else if !errors.Is(err, store.ErrNotFound) {
		publicationError(report, a.ID, "store_unavailable")
		return
	}
	policy, err := c.Store.RepositoryPolicy(ctx, a.Repository)
	if err != nil {
		publicationError(report, a.ID, "store_unavailable")
		return
	}
	p := factory.Publication{
		Publish: factory.PublicationOperation{Kind: factory.OpRefPublish}, PRCreate: factory.PublicationOperation{Kind: factory.OpPRCreate},
		Authority: a.Authority, ID: factory.NewID(), AssignmentID: a.ID, ProjectID: a.ProjectID, Role: a.Role,
		Acceptance: a.Acceptance, Preparation: a.Preparation, Run: a.Run, Candidate: a.Result.Candidate,
		BaseSHA: a.SourceCommit, TargetBranch: policy.TargetBranch, Stage: factory.PublicationOpen,
		Repository: a.Repository, Issue: a.Issue, CreatedUnix: time.Now().Unix(),
	}
	if err := c.Store.RecordPublication(ctx, p); err != nil {
		publicationError(report, a.ID, "store_unavailable")
		return
	}
	c.reconcilePublication(ctx, p, report)
}

func (c *Coordinator) assignmentForPublication(ctx context.Context, p factory.Publication) (factory.Assignment, error) {
	assignments, err := c.Store.IssueAssignments(ctx, p.Repository, p.Issue)
	if err != nil {
		return factory.Assignment{}, err
	}
	for _, a := range assignments {
		if a.ID == p.AssignmentID {
			return a, nil
		}
	}
	return factory.Assignment{}, store.ErrNotFound
}

// Reconciliation never depends on an export, current actor credential, or open
// dispatch. Those are prerequisites for new writes, not for learning old effects.
func (c *Coordinator) reconcilePublication(ctx context.Context, p factory.Publication, report *PublishReport) {
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

// publicationAuthority validates Soda policy and accepted input at the native
// revision that the later ref observation must match.
func (c *Coordinator) publicationAuthority(ctx context.Context, p factory.Publication, report *PublishReport) (factory.RepositoryPolicy, int64, bool) {
	var empty factory.RepositoryPolicy
	effective, err := c.EffectiveAuthority(ctx, p.Repository)
	if err != nil {
		publicationError(report, p.ID, "store_unavailable")
		return empty, 0, false
	}
	if !effective.Effective {
		publicationWait(report, p.ID, "authority_ineffective")
		return empty, 0, false
	}
	policy, err := c.Store.RepositoryPolicy(ctx, p.Repository)
	if err != nil {
		publicationError(report, p.ID, "store_unavailable")
		return empty, 0, false
	}
	current := effective.Authority
	// Capacity changes only bound future reservations, per the existing grant contract.
	current.Capacity = p.Authority.Capacity
	current.RequirementsID, err = c.Store.RequirementHead(ctx, p.ProjectID)
	if err != nil {
		publicationError(report, p.ID, "store_unavailable")
		return empty, 0, false
	}
	current.ApprovalID, err = c.Store.ApprovalHead(ctx, p.ProjectID)
	if err != nil {
		publicationError(report, p.ID, "store_unavailable")
		return empty, 0, false
	}
	if current != p.Authority || policy.TargetBranch != p.TargetBranch {
		publicationWait(report, p.ID, "authority_changed")
		return empty, 0, false
	}
	status, err := c.AcceptanceStatus(ctx, p.Repository, strconv.FormatInt(p.Issue, 10))
	if err != nil {
		publicationWait(report, p.ID, "accepted_inputs_unavailable")
		return empty, 0, false
	}
	if !status.Valid || status.Acceptance == nil || status.Acceptance.ID != p.Acceptance {
		publicationWait(report, p.ID, "accepted_inputs_changed")
		return empty, 0, false
	}
	return policy, status.Revision, true
}

func (c *Coordinator) publicationWork(a factory.Assignment, p factory.Publication, run factory.Run, bundle []byte, policy factory.RepositoryPolicy, kind string) factory.PublicationWork {
	actor, old := policy.Publish.ActorID, "absent"
	if kind == factory.OpPRCreate {
		actor, old = policy.Create.ActorID, p.Candidate
	}
	return factory.PublicationWork{
		Bundle: bundle, AssignmentID: a.ID, Publication: p.ID, RunID: p.Run, Run: run,
		Candidate: p.Candidate, BaseSHA: p.BaseSHA, TargetBranch: p.TargetBranch,
		OperationID: factory.PublicationOperationID(p.ID, kind, 1), AuthRevision: factory.AuthRevisionFor(a.ID, p.ID, p.Revision),
		ExpectedOld: old, ComparisonRef: p.TargetBranch, PRTitle: factory.PRTitleFor(a.Issue),
		PRBody:     factory.PRBodyFor(a.ID, a.Acceptance, a.Run, p.Candidate, a.SourceCommit),
		Repository: p.Repository, Issue: p.Issue, ActorID: actor,
	}
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

func (c *Coordinator) completePublication(ctx context.Context, p *factory.Publication, report *PublishReport) {
	if p.Publish.Completion != factory.OpCompletionComplete || p.PRCreate.Completion != factory.OpCompletionComplete {
		publicationWait(report, p.ID, "native_completion_pending")
		return
	}
	c.finishPublication(ctx, *p, factory.PublicationPublished, factory.Succeeded, factory.PublishReasonLinked, report)
}

func (c *Coordinator) publicationCallError(report *PublishReport, id string, err error) {
	var wait *factory.PublicationWait
	var refused *factory.PublicationRefusal
	switch {
	case errors.As(err, &wait):
		publicationWait(report, id, wait.Reason)
	case errors.As(err, &refused):
		publicationWait(report, id, refused.Reason)
	default:
		publicationError(report, id, "native_unavailable")
	}
}

func (c *Coordinator) cancelRepositoryPublications(ctx context.Context, repository int64) factory.PublicationWithdrawal {
	return c.cancelPublications(ctx, repository, 0, "")
}

func (c *Coordinator) cancelAcceptancePublications(ctx context.Context, repository, issue int64, acceptance string) factory.PublicationWithdrawal {
	return c.cancelPublications(ctx, repository, issue, acceptance)
}

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
		if err != nil || current.Stage == factory.PublicationOpen || current.Stage == factory.PublicationFenced || len(report.Errors) > 0 {
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
	if !c.storePublication(ctx, p, report) {
		return
	}
	if !c.adoptPublicationReceipts(ctx, a, p, run, report) {
		return
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

type exportTerminal struct{ Reason string }

func (c *Coordinator) exportCandidate(ctx context.Context, p factory.Publication, run factory.Run) ([]byte, *exportTerminal, *planWait) {
	state, err := c.Host.FactoryExport(ctx, project.FactoryExport{Project: p.ProjectID, ID: p.Run, Role: p.Role, Preparation: p.Preparation, Candidate: p.Candidate})
	if err != nil {
		if errors.Is(err, project.ErrFactoryExportCandidate) || errors.Is(err, project.ErrFactoryExportBounds) || isHostNotFound(err) || isHostStale(err) {
			return nil, &exportTerminal{Reason: factory.PublishReasonExportFailed}, nil
		}
		return nil, nil, waitFor("host_unavailable", "candidate export unconfirmed")
	}
	if state.ID != p.Run || state.Project != p.ProjectID || state.Candidate != p.Candidate {
		return nil, nil, waitFor("host_unavailable", "candidate export identity differs")
	}
	bundle, err := decodeExportBundle(state.Bundle)
	if err != nil {
		return nil, &exportTerminal{Reason: factory.PublishReasonExportFailed}, nil
	}
	return bundle, nil, nil
}
func decodeExportBundle(encoded string) ([]byte, error) {
	if encoded == "" || len(encoded) > 8<<20 {
		return nil, errors.New("candidate export exceeds bounds")
	}
	bundle, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil || len(bundle) == 0 || len(bundle) > 4<<20 {
		return nil, errors.New("candidate export exceeds bounds")
	}
	return bundle, nil
}

const hostRunStale = "factory run incarnation changed"

func isHostStale(err error) bool { return err != nil && strings.Contains(err.Error(), hostRunStale) }
