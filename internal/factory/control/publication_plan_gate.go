package control

import (
	"context"
	"errors"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
)

// FW15 phased publication join, Go side (T17).
//
// A finished phased implement assignment may publish only on completed plan
// task evidence: every planned task terminally completed, no failed or
// cancelled task, a valid candidate, and something new to publish. Missing
// evidence waits (the FW24 progress loop still owns persistence); conclusive
// evidence against publication refuses without recording anything, so no PR
// is ever fabricated and no duplicate write exists to replay.
//
// Two T17 clauses stay open joints and are NOT claimed here. Reported
// commits existing in candidate ancestry needs a Git reader the control
// package must not grow; the verdict collects the exact commit set that
// future check consumes. Final exact-tree checks need the host/native check
// owner; receipt check IDs are collected as evidence only. Review/correction
// findings join in FW22 and are untouched by this gate.

// Plan publication wait reasons. Both leave the assignment publishable for
// a later pass; neither records nor publishes anything.
const (
	// PlanWaitEvidenceUnavailable reports a missing plan head or revision:
	// the plan was never persisted or a replacement is in flight.
	PlanWaitEvidenceUnavailable = "plan_evidence_unavailable"
	// PlanWaitTasksIncomplete reports tasks that are missing, pending,
	// running or output-ready: descendants have not settled.
	PlanWaitTasksIncomplete = "plan_tasks_incomplete"
)

// Plan publication refusal reasons. The assignment stays listed so a later
// terminal owner (FW31) can dispose it; publication records nothing.
const (
	// PlanRefuseTasksFailed reports a failed or cancelled task: incomplete
	// work cannot publish.
	PlanRefuseTasksFailed = "plan_tasks_failed"
	// PlanRefuseEvidence reports malformed evidence: an invalid plan, a
	// receipt outside the admitted revision, an unknown task or check, or
	// a candidate no task accounts for.
	PlanRefuseEvidence = "plan_evidence_refused"
	// PlanRefuseUnchanged reports a wholly unchanged issue: no task
	// produced a commit and the candidate is the admitted base, so there
	// is no PR to make.
	PlanRefuseUnchanged = "no_changes_to_publish"
	// PlanRefuseCandidate reports a candidate that is not a commit.
	PlanRefuseCandidate = "candidate_invalid"
)

// PlanPublicationEvidence is the checkpoint evidence a ready verdict binds
// to the candidate: every task commit in plan order plus the check IDs the
// executor verified. Verification-only tasks contribute checks without a
// commit; no empty commit is manufactured for them.
type PlanPublicationEvidence struct {
	Commits          []string `json:"commits"`
	Trees            []string `json:"trees"`
	Checks           []string `json:"checks"`
	TaskCount        int      `json:"task_count"`
	VerificationOnly bool     `json:"verification_only"`
}

// PlanPublicationVerdict is the pure join decision. Exactly one of Ready,
// WaitReason, RefuseReason is set.
type PlanPublicationVerdict struct {
	Evidence     PlanPublicationEvidence
	WaitReason   string
	RefuseReason string
	Ready        bool
}

// judgePlanPublication decides the join over one admitted plan revision,
// its persisted receipts and the reported candidate. It is pure: loading
// stays with the caller, so restarts observe receipts instead of redoing
// completed writes and exact replays decide identically.
func judgePlanPublication(plan factory.InvestigationPlan, receipts []store.PlanTaskReceipt, candidate string) PlanPublicationVerdict {
	refuse := func(reason string) PlanPublicationVerdict {
		return PlanPublicationVerdict{RefuseReason: reason}
	}
	if err := plan.Validate(); err != nil {
		return refuse(PlanRefuseEvidence)
	}
	if !factory.ValidCommit(candidate) {
		return refuse(PlanRefuseCandidate)
	}
	byTask, knownChecks := indexPlanPublicationKeys(plan)
	stored, verdict, decided := admitPlanTaskReceipts(plan, receipts, byTask)
	if decided {
		return verdict
	}
	return collectPlanPublicationEvidence(plan, candidate, stored, knownChecks)
}

func indexPlanPublicationKeys(plan factory.InvestigationPlan) (map[string]int, map[string]bool) {
	byTask := make(map[string]int, len(plan.Tasks))
	for i, task := range plan.Tasks {
		byTask[task.ID] = i
	}
	knownChecks := make(map[string]bool, len(plan.Checks))
	for _, check := range plan.Checks {
		knownChecks[check.ID] = true
	}
	return byTask, knownChecks
}

func admitPlanTaskReceipts(plan factory.InvestigationPlan, receipts []store.PlanTaskReceipt, byTask map[string]int) (map[string]store.PlanTaskReceipt, PlanPublicationVerdict, bool) {
	refused := PlanPublicationVerdict{RefuseReason: PlanRefuseEvidence}
	stored := make(map[string]store.PlanTaskReceipt, len(receipts))
	for _, receipt := range receipts {
		if receipt.AttemptRoot != plan.AttemptRoot || receipt.PlanID != plan.ID {
			return nil, refused, true
		}
		if _, ok := byTask[receipt.TaskID]; !ok {
			return nil, refused, true
		}
		if _, dup := stored[receipt.TaskID]; dup {
			return nil, refused, true
		}
		stored[receipt.TaskID] = receipt
	}
	return stored, PlanPublicationVerdict{}, false
}

func collectPlanPublicationEvidence(plan factory.InvestigationPlan, candidate string, stored map[string]store.PlanTaskReceipt, knownChecks map[string]bool) PlanPublicationVerdict {
	refuse := func(reason string) PlanPublicationVerdict {
		return PlanPublicationVerdict{RefuseReason: reason}
	}
	wait := func(reason string) PlanPublicationVerdict {
		return PlanPublicationVerdict{WaitReason: reason}
	}
	evidence := PlanPublicationEvidence{TaskCount: len(plan.Tasks)}
	for _, task := range plan.Tasks {
		receipt, ok := stored[task.ID]
		if !ok {
			return wait(PlanWaitTasksIncomplete)
		}
		switch receipt.State {
		case store.PlanTaskFailed, store.PlanTaskCancelled:
			return refuse(PlanRefuseTasksFailed)
		case store.PlanTaskCompleted:
		case store.PlanTaskPending, store.PlanTaskRunning, store.PlanTaskOutputReady:
			return wait(PlanWaitTasksIncomplete)
		default:
			return refuse(PlanRefuseEvidence)
		}
		for _, id := range receipt.Checks {
			if !knownChecks[id] {
				return refuse(PlanRefuseEvidence)
			}
			evidence.Checks = unionKeys(evidence.Checks, []string{id})
		}
		if receipt.Commit == "" && receipt.Tree == "" {
			continue
		}
		if !factory.ValidCommit(receipt.Commit) || !factory.ValidCommit(receipt.Tree) {
			return refuse(PlanRefuseEvidence)
		}
		evidence.Commits = append(evidence.Commits, receipt.Commit)
		evidence.Trees = append(evidence.Trees, receipt.Tree)
	}
	if len(evidence.Commits) == 0 {
		evidence.VerificationOnly = true
		if candidate == plan.SourceCommit {
			return refuse(PlanRefuseUnchanged)
		}
		return refuse(PlanRefuseEvidence)
	}
	if candidate == plan.SourceCommit {
		return refuse(PlanRefuseEvidence)
	}
	return PlanPublicationVerdict{Ready: true, Evidence: evidence}
}

// gatePhasedPublication requires completed plan task evidence before a
// phased implement assignment may publish. Pre-phase work keeps the
// existing path untouched. It reports false after recording the wait, so
// the caller publishes nothing and retries on a later pass.
func (c *Coordinator) gatePhasedPublication(ctx context.Context, a factory.Assignment, report *PublishReport) bool {
	if a.Phase != project.ExecutionImplement {
		return true
	}
	headID, _, err := c.Store.PlanHead(ctx, a.AttemptRoot)
	switch {
	case err == nil:
	case errors.Is(err, store.ErrNotFound):
		publicationWait(report, a.ID, PlanWaitEvidenceUnavailable)
		return false
	default:
		publicationError(report, a.ID, "store_unavailable")
		return false
	}
	plan, err := c.Store.InvestigationPlan(ctx, a.AttemptRoot, headID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			publicationWait(report, a.ID, PlanWaitEvidenceUnavailable)
			return false
		}
		publicationError(report, a.ID, "store_unavailable")
		return false
	}
	receipts, err := c.Store.PlanTaskReceipts(ctx, a.AttemptRoot, headID)
	if err != nil {
		publicationError(report, a.ID, "store_unavailable")
		return false
	}
	verdict := judgePlanPublication(plan, receipts, a.Result.Candidate)
	if verdict.Ready {
		return true
	}
	if verdict.RefuseReason != "" {
		publicationWait(report, a.ID, verdict.RefuseReason)
		return false
	}
	publicationWait(report, a.ID, verdict.WaitReason)
	return false
}
