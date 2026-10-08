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
	effective   factory.EffectiveAuthority
	policy      factory.RepositoryPolicy
	sponsorship factory.Sponsorship
	prep        project.StoredPreparation
	pin         project.FactoryHarnessPin
	acceptance  factory.Acceptance
	inputs      DispatchInputs
	projectID   string
	prompt      []byte
	promptSHA   string
	deadline    time.Time
	requirement string
	approval    string
	planned     int
	gateRev     int64
}

// dispatchOne dispatches one queued issue or reports exactly why it
// waited. Checks run in fixed order so the same state always reports the
// same reason; waits hold nothing and release nothing.
func dispatchOne(ctx context.Context, deps DispatchDeps, occupancy *passOccupancy, control factory.IssueControl, report *DispatchReport) {
	repository, issue := control.Repository, control.Issue
	assignments, err := deps.Store.IssueAssignments(ctx, repository, issue)
	if err != nil {
		report.Errors = append(report.Errors, DispatchError{Repository: repository, Issue: issue, Reason: DispatchErrStore, Detail: "assignments unreadable"})
		return
	}
	for _, a := range assignments {
		if a.Stage != factory.AssignmentFinished {
			report.Waits = append(report.Waits, DispatchWait{Repository: repository, Issue: issue, Reason: WaitAssigned})
			return
		}
	}
	head, err := deps.Store.AcceptanceHead(ctx, repository, issue)
	if err != nil {
		report.Errors = append(report.Errors, DispatchError{Repository: repository, Issue: issue, Reason: DispatchErrStore, Detail: "acceptance head unreadable"})
		return
	}
	for _, a := range assignments {
		if a.Acceptance == head {
			report.Waits = append(report.Waits, DispatchWait{Repository: repository, Issue: issue, Reason: WaitAttemptRecorded})
			return
		}
	}
	plan, wait, failed := planAttempt(ctx, deps, occupancy, repository, issue, head)
	if failed != nil {
		report.Errors = append(report.Errors, DispatchError{Repository: repository, Issue: issue, Reason: failed.Reason, Detail: failed.Detail})
		return
	}
	if wait != nil {
		report.Waits = append(report.Waits, DispatchWait{Repository: repository, Issue: issue, Reason: wait.Reason, Detail: wait.Detail})
		return
	}
	executeFreshAttempt(ctx, deps, occupancy, plan, repository, issue, report)
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

// planAttempt verifies one dispatch end to end without recording
// anything. Every check that fails reports its wait or failure; a full
// plan hands its exact bound inputs to the executor.
func planAttempt(ctx context.Context, deps DispatchDeps, occupancy *passOccupancy, repository, issue int64, head string) (*attemptPlan, *planWait, *planWait) {
	plan := &attemptPlan{}
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
	sponsorship, wait, failed := selectSponsorship(ctx, deps, repository, effective)
	if failed != nil || wait != nil {
		return nil, wait, failed
	}
	plan.sponsorship = sponsorship
	decision, err := deps.Store.AcceptanceDecision(ctx, head)
	if err != nil {
		return nil, nil, waitFor(DispatchErrStore, "acceptance decision unreadable")
	}
	plan.acceptance = decision
	if wait, failed := checkLimits(ctx, deps, occupancy, plan); failed != nil || wait != nil {
		return nil, wait, failed
	}
	if wait, failed := selectPreparation(ctx, deps, plan); failed != nil || wait != nil {
		return nil, wait, failed
	}
	if wait := checkHarness(ctx, deps, plan); wait != nil {
		return nil, wait, nil
	}
	if wait, failed := readAttemptInputs(ctx, deps, plan, repository, issue); failed != nil || wait != nil {
		return nil, wait, failed
	}
	prompt, err := factory.BuildDispatchPrompt(factory.PromptInputs{
		Repository: repository, Issue: issue, NativeRev: plan.inputs.Revision,
		AcceptanceID: head, TargetBranch: policy.TargetBranch, SourceCommit: plan.inputs.Tip,
		Preparation: plan.prep.Preparation.ID,
		Harness:     plan.pin.Harness, Model: policy.Roles[project.RoleCoder].Model, Role: project.RoleCoder,
		Title: plan.inputs.Issue.Title, Body: plan.inputs.Issue.Body,
		Sources:     promptSections(plan.acceptance.Sources, plan.inputs.Comments),
		Resolutions: promptSections(plan.acceptance.Resolutions, plan.inputs.Comments),
	})
	if err != nil {
		return nil, waitFor(WaitPrompt, "accepted inputs exceed the prompt bound"), nil
	}
	plan.prompt = prompt
	plan.promptSHA = project.FactoryPromptDigest(prompt)
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
