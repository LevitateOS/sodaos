package control

import (
	"context"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
)

// issueLibraryMaintenance reads explicit accepted issue metadata through
// the canonical Soda block parser. Ordinary prose never changes the
// assignment role, and ambiguous metadata refuses.
func issueLibraryMaintenance(body string) (bool, error) {
	block, err := factory.ParseSodaIssueBlock(body)
	if err != nil {
		return false, err
	}
	return block.Role == factory.SodaRoleLibraryAdmin, nil
}

// Initial work selects maintenance only from the current, accepted native body.
// The ordinary admission then rechecks that same acceptance with the right
// reviewer preparation and sponsorship before anything launches.
func selectInitialIssuePhase(ctx context.Context, deps DispatchDeps, repository, issue int64, head string) (string, string, *planWait, *planWait) {
	policy, err := deps.Store.RepositoryPolicy(ctx, repository)
	if err != nil {
		return "", "", nil, waitFor(DispatchErrStore, "policy unreadable")
	}
	decision, err := deps.Store.AcceptanceDecision(ctx, head)
	if err != nil {
		return "", "", nil, waitFor(DispatchErrStore, "acceptance decision unreadable")
	}
	plan := &attemptPlan{policy: policy, acceptance: decision}
	if wait, failed := readAttemptInputs(ctx, deps, plan, repository, issue); wait != nil || failed != nil {
		return "", "", wait, failed
	}
	block, err := factory.ParseSodaIssueBlock(plan.inputs.Issue.Body)
	if err != nil {
		return "", "", waitFor(WaitInputsChanged, err.Error()), nil
	}
	switch block.Role {
	case factory.SodaRoleLibraryAdmin:
		return project.ExecutionMaintainLibrary, project.RoleReviewer, nil, nil
	case factory.SodaRoleResearch:
		return project.ExecutionResearch, project.RoleReviewer, nil, nil
	}
	return project.ExecutionInvestigate, project.RoleCoder, nil, nil
}

func settleLibraryMaintenance(assigned factory.Assignment, submission factory.MCPSubmission, now time.Time) (factory.Assignment, string, bool, error) {
	if assigned.Phase != project.ExecutionMaintainLibrary {
		return fenceFW40(assigned, submission, now, "library finish does not match its assignment phase")
	}
	change, err := factory.DecodeLibraryChange(submission.Tool, submission.Payload)
	if err != nil {
		return fenceFW40(assigned, submission, now, "invalid library maintenance result")
	}
	parsed := phaseResultFor(submission, project.ExecutionMaintainLibrary, change.Summary, &factory.PhaseOutput{Maintenance: &factory.LibraryMaintenanceResult{SubmissionID: submission.SubmissionID}})
	return finishFromMCPPhaseResult(assigned, submission, parsed, now), "library maintenance settled", true, nil
}
