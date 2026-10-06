package control

import (
	"context"
	"errors"
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

type exportTerminal struct{ Reason string }
