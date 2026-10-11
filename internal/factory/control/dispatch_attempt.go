package control

import (
	"context"
	"errors"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
)

// attemptPlan is everything one dispatch attempt verified, in check order:
// the bound authority and grants, the exact preparation, harness pin and
// source, the verified prompt bytes, and the deadline the reservation
// will hold.
type attemptPlan struct {
	effective            factory.EffectiveAuthority
	policy               factory.RepositoryPolicy
	sponsorship          factory.Sponsorship
	prep                 project.StoredPreparation
	pin                  project.FactoryHarnessPin
	acceptance           factory.Acceptance
	control              factory.IssueControl
	inputs               DispatchInputs
	projectID            string
	attemptRoot          string
	preparationOwner     string
	prompt               []byte
	promptSHA            string
	deadline             time.Time
	requirement          string
	approval             string
	planned              int
	gateRev              int64
	applianceConcurrent  int
	repositoryConcurrent int
	freshAttempt         bool
	attemptLimits        factory.AttemptLimits
}

// dispatchOne dispatches one queued issue or reports exactly why it
// waited. Checks run in fixed order so the same state reports the same reason.
// An explicit Retry can retain bounded private preparation custody while waiting.
func dispatchOne(ctx context.Context, deps DispatchDeps, occupancy *passOccupancy, control factory.IssueControl, report *DispatchReport) {
	repository, issue := control.Repository, control.Issue
	latest, err := deps.Store.LatestIssueAssignment(ctx, repository, issue)
	hasAssignment := err == nil
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		report.Errors = append(report.Errors, DispatchError{Repository: repository, Issue: issue, Reason: DispatchErrStore, Detail: "assignments unreadable"})
		return
	}
	attemptClosed := false
	if hasAssignment {
		attemptClosed, err = deps.Store.CloseExpiredAttempt(ctx, latest.ID)
		if err != nil {
			report.Errors = append(report.Errors, DispatchError{Repository: repository, Issue: issue, Reason: DispatchErrStore, Detail: "attempt allowance unreadable"})
			return
		}
		if attemptClosed && latest.Stage != factory.AssignmentFinished {
			report.Waits = append(report.Waits, DispatchWait{Repository: repository, Issue: issue, Reason: WaitAllowance, Detail: "attempt is closed; maintainer intervention is required"})
			return
		}
	}
	if hasAssignment && latest.Stage != factory.AssignmentFinished {
		report.Waits = append(report.Waits, DispatchWait{Repository: repository, Issue: issue, Reason: WaitAssigned})
		return
	}
	head, err := deps.Store.AcceptanceHead(ctx, repository, issue)
	if err != nil {
		report.Errors = append(report.Errors, DispatchError{Repository: repository, Issue: issue, Reason: DispatchErrStore, Detail: "acceptance head unreadable"})
		return
	}
	var retry *factory.RetryDecision
	if hasAssignment && latest.Acceptance == head {
		pending, retryErr := deps.Store.PendingExplicitRetry(ctx, repository, issue, head)
		if errors.Is(retryErr, store.ErrNotFound) {
			detail := "attempt result is recorded; explicit Retry is required"
			if attemptClosed {
				detail = "attempt is closed; maintainer intervention is required"
			}
			report.Waits = append(report.Waits, DispatchWait{Repository: repository, Issue: issue, Reason: WaitAttemptRecorded, Detail: detail})
			return
		}
		if retryErr != nil {
			report.Errors = append(report.Errors, DispatchError{Repository: repository, Issue: issue, Reason: DispatchErrStore, Detail: "explicit retry unreadable"})
			return
		}
		retry = &pending
	}
	if attemptClosed && retry == nil {
		report.Waits = append(report.Waits, DispatchWait{Repository: repository, Issue: issue, Reason: WaitAttemptRecorded, Detail: "attempt is closed; maintainer intervention is required"})
		return
	}
	if deps.Refresh != nil {
		control, err = deps.Refresh(ctx, control)
		if err != nil {
			report.Waits = append(report.Waits, DispatchWait{Repository: repository, Issue: issue, Reason: WaitInputsUnavailable, Detail: "readiness refresh is unavailable"})
			return
		}
	}
	plan, wait, failed := planAttempt(ctx, deps, occupancy, repository, issue, head, control, retry)
	if failed != nil {
		report.Errors = append(report.Errors, DispatchError{Repository: repository, Issue: issue, Reason: failed.Reason, Detail: failed.Detail})
		return
	}
	if wait != nil {
		report.Waits = append(report.Waits, DispatchWait{Repository: repository, Issue: issue, Reason: wait.Reason, Detail: wait.Detail})
		return
	}
	if deps.collectPlan != nil {
		deps.collectPlan(plan)
		return
	}
	if deps.checkPlan != nil && !deps.checkPlan(plan) {
		report.Waits = append(report.Waits, DispatchWait{Repository: repository, Issue: issue, Reason: WaitInputsStale, Detail: "priority input snapshot changed"})
		return
	}
	executeFreshAttempt(ctx, deps, occupancy, plan, repository, issue, report, retry)
}

// planWait is one non-launch outcome: either a retryable wait or an
// unexpected failure. Exactly one of the dispatchOne results is set.
type planWait struct {
	Reason string
	Detail string
}

func waitFor(reason, detail string) *planWait { return &planWait{Reason: reason, Detail: detail} }

// admissionWait maps an atomic packet refusal to its wait. A refusal
// proves a concurrent admission consumed the room first; the issue waits
// for a later pass instead of launching.
func admissionWait(err error) *planWait {
	switch {
	case errors.Is(err, store.ErrCapacityFull):
		return waitFor(WaitCapacity, "appliance runs at its limit")
	case errors.Is(err, store.ErrRepositoryFull):
		return waitFor(WaitRepository, "repository runs at its limit")
	case errors.Is(err, store.ErrSponsorshipFull):
		return waitFor(WaitSponsorship, "sponsorship runs at its limit")
	case errors.Is(err, store.ErrAllowanceExhausted):
		return waitFor(WaitAllowance, "sponsorship allowance is exhausted")
	case errors.Is(err, factory.ErrAttemptTimeExhausted):
		return waitFor(WaitAllowance, "attempt active-time allowance is exhausted")
	case errors.Is(err, factory.ErrAttemptClosed):
		return waitFor(WaitAttemptRecorded, "attempt is terminal; explicit Retry is required")
	case errors.Is(err, store.ErrConnectionUsageBudget):
		return waitFor(WaitAllowance, "connection rolling usage budget is exhausted")
	case errors.Is(err, store.ErrAdmissionChanged):
		return waitFor(WaitAuthority, "grants changed during dispatch")
	default:
		return nil
	}
}

// refreshOccupancy reloads the pass snapshot after a lost admission race
// so later issues plan against the winner's reservation instead of stale
// emptiness. The packet gate stays authoritative: if the reload itself
// fails, the pass continues and the gate refuses any over-admission.
func refreshOccupancy(ctx context.Context, deps DispatchDeps, occupancy *passOccupancy, report *DispatchReport) {
	fresh, err := snapshotOccupancy(ctx, deps.Store)
	if err != nil {
		report.Errors = append(report.Errors, DispatchError{Reason: DispatchErrStore, Detail: "capacity accounting unavailable"})
		return
	}
	*occupancy = fresh
}

// planAttempt verifies one dispatch end to end. Explicit Retry first records
// custody for its fresh preparation; run admission waits for confirmed readiness.
// Every failed check reports its wait or failure.
func planAttempt(ctx context.Context, deps DispatchDeps, occupancy *passOccupancy, repository, issue int64, head string, control factory.IssueControl, retry *factory.RetryDecision) (*attemptPlan, *planWait, *planWait) {
	id := ""
	if retry != nil {
		id = retry.CommandID
	}
	return planAttemptForRole(ctx, deps, occupancy, repository, issue, head, control, retry != nil, project.RoleCoder, "", false, id)
}

// planAttemptForRole shares the mutable dispatch checks between initial coder
// work and retries. A recorded child supplies its own role and exact approved
// preparation; callers keep its prompt bytes instead of rebuilding a new plan.
func planAttemptForRole(ctx context.Context, deps DispatchDeps, occupancy *passOccupancy, repository, issue int64, head string, control factory.IssueControl, freshAttempt bool, role, preparation string, recordedPrompt bool, retryID string) (*attemptPlan, *planWait, *planWait) {
	if control.Validate() != nil || control.Readiness != factory.ReadinessQueued || control.Repository != repository || control.Issue != issue || control.Acceptance != head {
		return nil, waitFor(WaitInputsChanged, "queued readiness changed during dispatch"), nil
	}
	plan := &attemptPlan{control: control, freshAttempt: freshAttempt}
	effective, err := deps.Authority(ctx, repository)
	if err != nil {
		return nil, nil, waitFor(DispatchErrStore, "authority unreadable")
	}
	plan.effective = effective
	if !effective.Effective {
		if onlyDispatchClosed(effective) {
			return nil, waitFor(WaitDispatchClosed, "dispatch gate is closed"), nil
		}
		return nil, waitFor(WaitAuthority, "grants do not authorize dispatch"), nil
	}
	p, err := deps.Store.ProjectByRepository(ctx, repository)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, waitFor(WaitPreparation, "repository has no project"), nil
		}
		return nil, nil, waitFor(DispatchErrStore, "project unreadable")
	}
	plan.projectID = p.ID
	policy, err := deps.Store.RepositoryPolicy(ctx, repository)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, waitFor(WaitAuthority, "policy changed during dispatch"), nil
		}
		return nil, nil, waitFor(DispatchErrStore, "policy unreadable")
	}
	plan.policy = policy
	sponsorship, wait, failed := selectSponsorship(ctx, deps, repository, effective, role)
	if failed != nil || wait != nil {
		return nil, wait, failed
	}
	plan.sponsorship = sponsorship
	if sponsorship.ProjectID != plan.projectID {
		return nil, waitFor(WaitAuthority, "sponsorship grant belongs to a different Project"), nil
	}
	decision, err := deps.Store.AcceptanceDecision(ctx, head)
	if err != nil {
		return nil, nil, waitFor(DispatchErrStore, "acceptance decision unreadable")
	}
	plan.acceptance = decision
	if wait, failed := checkLimits(ctx, deps, occupancy, plan); failed != nil || wait != nil {
		return nil, wait, failed
	}
	if wait, failed := selectPreparationFor(ctx, deps, plan, role, preparation); failed != nil || wait != nil {
		return nil, wait, failed
	}
	plan.effective.Authority.RequirementsID = plan.requirement
	plan.effective.Authority.ApprovalID = plan.approval
	if wait := checkHarnessFor(ctx, deps, plan, role); wait != nil {
		return nil, wait, nil
	}
	if wait, failed := readAttemptInputs(ctx, deps, plan, repository, issue); failed != nil || wait != nil {
		return nil, wait, failed
	}
	if plan.inputs.Revision != control.NativeRev {
		return nil, waitFor(WaitInputsChanged, "readiness observation changed before dispatch"), nil
	}
	if retryID != "" {
		if deps.PrepareRetry == nil {
			return nil, waitFor(WaitPreparation, "fresh retry preparation is unavailable"), nil
		}
		ready, err := deps.PrepareRetry(ctx, plan, retryID)
		if err != nil {
			return nil, waitFor(WaitPreparation, "fresh retry preparation is not confirmed"), nil
		}
		if !ready {
			return nil, waitFor(WaitPreparation, "fresh retry preparation is running"), nil
		}
	} else if role == project.RoleCoder && !recordedPrompt && plan.prep.Preparation.SourceCommit != plan.inputs.Tip {
		if deps.PrepareSource == nil {
			return nil, waitFor(WaitPreparation, "current source preparation is unavailable"), nil
		}
		ready, err := deps.PrepareSource(ctx, plan)
		if err != nil || !ready {
			return nil, waitFor(WaitPreparation, "current source preparation is not confirmed"), nil
		}
	}
	selection := policy.Roles[role]
	if recordedPrompt {
		plan.prompt = nil
		plan.promptSHA = ""
	} else {
		repositoryContext, contextErr := readRepositoryContext(ctx, deps.Host, plan, plan.prep.Preparation, role,
			plan.prep.Preparation.SourceCommit, plan.prep.Preparation.SourceCommit, plan.inputs.Tip, true)
		if contextErr != nil {
			return nil, waitFor(WaitSource, "native prepared repository context is unavailable or changed"), nil
		}
		prompt, err := factory.BuildDispatchPrompt(factory.PromptInputs{
			Project: plan.projectID, Repository: repository, Issue: issue, NativeRev: plan.inputs.Revision,
			AcceptanceID: head, TargetBranch: policy.TargetBranch, SourceCommit: plan.inputs.Tip,
			ApprovedBase:   plan.prep.Preparation.SourceCommit,
			Preparation:    plan.prep.Preparation.ID,
			RequirementsID: plan.requirement, ApprovalID: plan.approval,
			Harness: plan.pin.Harness, Model: selection.Model, Role: role,
			ProviderConnection:    plan.sponsorship.Connection,
			RequiredChecks:        policy.Checks,
			ApplianceConcurrent:   plan.applianceConcurrent,
			RepositoryConcurrent:  plan.repositoryConcurrent,
			SponsorshipConcurrent: plan.sponsorship.MaxConcurrent,
			AttemptLimits:         plan.attemptLimits,
			Title:                 plan.inputs.Issue.Title, Body: plan.inputs.Issue.Body,
			Sources:       promptSections(plan.acceptance.Sources, plan.inputs.Comments),
			Resolutions:   promptSections(plan.acceptance.Resolutions, plan.inputs.Comments),
			Prerequisites: plan.acceptance.Prerequisites, Control: plan.control,
			RepositoryContext: repositoryContext,
		})
		if err != nil {
			if errors.Is(err, factory.ErrPromptPrerequisiteEvidence) {
				return nil, waitFor(WaitInputsChanged, "readiness does not bind accepted prerequisite outcomes"), nil
			}
			return nil, waitFor(WaitPrompt, "accepted inputs exceed the prompt bound"), nil
		}
		plan.prompt = prompt
		plan.promptSHA = project.FactoryPromptDigest(prompt)
	}
	open, gateRev, _, err := deps.Store.DispatchState(ctx, repository)
	if err != nil {
		return nil, nil, waitFor(DispatchErrStore, "dispatch gate unreadable")
	}
	if !open {
		return nil, waitFor(WaitDispatchClosed, "dispatch gate closed during dispatch"), nil
	}
	plan.gateRev = gateRev
	return plan, nil, nil
}
