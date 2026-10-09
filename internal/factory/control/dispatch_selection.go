package control

import (
	"context"
	"errors"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/store"
)

// selectSponsorship picks the authority-bound active sponsorship: the
// first active grant in connection order, at its bound revision, with
// permission for the requested dispatch role. Any other connection is
// never a fallback.
func selectSponsorship(ctx context.Context, deps DispatchDeps, repository int64, effective factory.EffectiveAuthority, role string) (factory.Sponsorship, *planWait, *planWait) {
	sponsorships, err := deps.Store.Sponsorships(ctx, repository)
	if err != nil {
		return factory.Sponsorship{}, nil, waitFor(DispatchErrStore, "sponsorships unreadable")
	}
	for _, sponsorship := range sponsorships {
		if !sponsorship.Active {
			continue
		}
		if sponsorship.Validate() != nil {
			return factory.Sponsorship{}, waitFor(WaitAuthority, "sponsorship execution identity is unavailable"), nil
		}
		if sponsorship.Revision != effective.Authority.Sponsorship || sponsorship.Connection != effective.Authority.SponsorshipConnection {
			return factory.Sponsorship{}, waitFor(WaitAuthority, "sponsorship changed during dispatch"), nil
		}
		permitted := false
		for _, grantedRole := range sponsorship.Roles {
			permitted = permitted || grantedRole == role
		}
		if !permitted {
			return factory.Sponsorship{}, waitFor(WaitSponsorshipRole, "sponsorship does not permit this role"), nil
		}
		budget, err := deps.Store.ConnectionUsageBudget(ctx, sponsorship.Connection)
		if err != nil || budget.Revision != effective.Authority.ConnectionUsageBudget {
			return factory.Sponsorship{}, waitFor(WaitAuthority, "connection usage budget changed during dispatch"), nil
		}
		return sponsorship, nil, nil
	}
	return factory.Sponsorship{}, waitFor(WaitAuthority, "sponsorship changed during dispatch"), nil
}

// checkLimits enforces appliance, repository-session, sponsorship and
// allowance limits against unsettled runs, held reservations and confirmed
// usage, then sizes the plan's deadline from the remaining allowance. The
// deadline never exceeds the host's supervised bound.
func checkLimits(ctx context.Context, deps DispatchDeps, occupancy *passOccupancy, plan *attemptPlan) (*planWait, *planWait) {
	capacity, err := deps.Store.Capacity(ctx)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return waitFor(WaitAuthority, "capacity changed during dispatch"), nil
		}
		return nil, waitFor(DispatchErrStore, "capacity unreadable")
	}
	grant, err := deps.Store.OperatorGrant(ctx, plan.policy.Repository)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return waitFor(WaitAuthority, "operator grant changed during dispatch"), nil
		}
		return nil, waitFor(DispatchErrStore, "operator grant unreadable")
	}
	repository := plan.policy.Repository
	plan.applianceConcurrent = capacity.MaxConcurrentRuns
	if occupancy.heldTotal()+occupancy.unattributed >= capacity.MaxConcurrentRuns {
		return waitFor(WaitCapacity, "appliance runs at its limit"), nil
	}
	repoLimit := plan.policy.MaxConcurrent
	if grant.MaxConcurrent < repoLimit {
		repoLimit = grant.MaxConcurrent
	}
	if repoLimit > 1 {
		repoLimit = 1
	}
	plan.repositoryConcurrent = repoLimit
	unsettled, err := deps.Store.ProjectUnsettledRuns(ctx, plan.projectID, 1)
	if err != nil {
		return nil, waitFor(DispatchErrStore, "repository run state unreadable")
	}
	if len(unsettled) != 0 {
		return waitFor(WaitRepository, "repository has an unsettled factory session"), nil
	}
	if occupancy.heldRepo(repository)+occupancy.byProject[plan.projectID] >= repoLimit {
		return waitFor(WaitRepository, "repository runs at its limit"), nil
	}
	slots, planned := occupancy.heldConn(repository, plan.sponsorship.Connection)
	if slots >= plan.sponsorship.MaxConcurrent {
		return waitFor(WaitSponsorship, "sponsorship runs at its limit"), nil
	}
	used, err := deps.Store.UsageTotal(ctx, repository, plan.sponsorship.Connection)
	if err != nil {
		return nil, waitFor(DispatchErrStore, "confirmed usage unreadable")
	}
	remaining := plan.sponsorship.AllowanceMinutes - used - planned
	if remaining < 1 {
		return waitFor(WaitAllowance, "sponsorship allowance is exhausted"), nil
	}
	if remaining > MaxDispatchDeadline {
		remaining = MaxDispatchDeadline
	}
	budget, err := deps.Store.ConnectionUsageBudget(ctx, plan.sponsorship.Connection)
	if err != nil {
		return waitFor(WaitAuthority, "connection usage budget changed during dispatch"), nil
	}
	if budget.Revision != plan.effective.Authority.ConnectionUsageBudget {
		return waitFor(WaitAuthority, "connection usage budget changed during dispatch"), nil
	}
	charge, err := deps.Store.ConnectionUsageCharge(ctx, plan.sponsorship.Connection, time.Now())
	if err != nil {
		return nil, waitFor(DispatchErrStore, "connection usage unreadable")
	}
	left := budget.RollingMinutes*factory.UsageMicrosPerMinute - charge
	if left < factory.UsageMicrosPerMinute {
		return waitFor(WaitAllowance, "connection rolling usage budget is exhausted"), nil
	}
	connectionMinutes := left / factory.UsageMicrosPerMinute
	if connectionMinutes < int64(remaining) {
		remaining = int(connectionMinutes)
	}
	now := time.Now()
	deadline := now.Add(time.Duration(remaining) * time.Minute)
	plan.attemptLimits = plan.policy.AttemptLimits
	attemptSeconds := int64(plan.policy.AttemptLimits.ActiveMinutes) * 60
	if !plan.freshAttempt {
		allowance, allowanceErr := deps.Store.AttemptAllowance(ctx, repository, plan.control.Issue)
		switch {
		case allowanceErr == nil:
			if allowance.Closed {
				return waitFor(WaitAttemptRecorded, "attempt is terminal; explicit Retry is required"), nil
			}
			plan.attemptRoot = allowance.RootAssignment
			plan.attemptLimits = allowance.Limits
			attemptSeconds = allowance.RemainingSeconds(now)
		case errors.Is(allowanceErr, store.ErrNotFound):
		default:
			return nil, waitFor(DispatchErrStore, "attempt allowance unreadable")
		}
	}
	if attemptSeconds <= 0 {
		return waitFor(WaitAllowance, "attempt active-time allowance is exhausted"), nil
	}
	attemptDeadline := time.Unix(now.Unix()+attemptSeconds, 0)
	if attemptDeadline.Before(deadline) {
		deadline = attemptDeadline
	}
	if boundedMinutes := int((deadline.Sub(now) + time.Minute - 1) / time.Minute); boundedMinutes < remaining {
		remaining = boundedMinutes
	}
	plan.planned = remaining
	plan.deadline = deadline
	return nil, nil
}

// selectPreparationFor binds a ready role preparation built from the current
// requirement and approval decisions. An exact ID preserves child work.
func selectPreparationFor(ctx context.Context, deps DispatchDeps, plan *attemptPlan, role, exactID string) (*planWait, *planWait) {
	requirement, err := deps.Store.RequirementHead(ctx, plan.projectID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return waitFor(WaitPreparation, "project has no accepted requirements"), nil
		}
		return nil, waitFor(DispatchErrStore, "requirement head unreadable")
	}
	approval, err := deps.Store.ApprovalHead(ctx, plan.projectID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return waitFor(WaitPreparation, "project has no approved setup"), nil
		}
		return nil, waitFor(DispatchErrStore, "approval head unreadable")
	}
	preparations, err := deps.Store.ProjectPreparations(ctx, plan.projectID)
	if err != nil {
		return nil, waitFor(DispatchErrStore, "preparations unreadable")
	}
	for _, prep := range preparations {
		if prep.Preparation.Role != role || !prep.State.Ready || (exactID != "" && prep.Preparation.ID != exactID) {
			continue
		}
		if prep.Preparation.Requirements.ID != requirement || prep.Preparation.Approval.ID != approval {
			continue
		}
		plan.prep = prep
		plan.requirement, plan.approval = requirement, approval
		return nil, nil
	}
	return waitFor(WaitPreparation, "no ready role preparation matches the current decisions"), nil
}

// checkHarnessFor requires the staged pin to match the selected role's
// policy family and version, with valid content and a pinned image.
func checkHarnessFor(ctx context.Context, deps DispatchDeps, plan *attemptPlan, role string) *planWait {
	selection := plan.policy.Roles[role]
	if selection.Harness == "" || selection.HarnessVers == "" || selection.Model == "" {
		return waitFor(WaitHarness, "policy has no selection for this role")
	}
	pin, err := deps.Host.FactoryHarness(ctx, selection.Harness)
	if err != nil {
		return waitFor(WaitHarness, "staged harness pin unreadable")
	}
	if err := pin.Validate(); err != nil {
		return waitFor(WaitHarness, "staged harness is not pinned")
	}
	if pin.Harness != selection.Harness || pin.Version != selection.HarnessVers {
		return waitFor(WaitHarness, "policy selects harness "+selection.Harness+" version "+selection.HarnessVers)
	}
	plan.pin = pin
	return nil
}
