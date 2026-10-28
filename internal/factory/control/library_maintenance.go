package control

import (
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
)

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
