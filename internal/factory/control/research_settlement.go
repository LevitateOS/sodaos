package control

import (
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
)

// settleResearch settles one researcher finish receipt into its typed
// research report. Research answers declared questions with findings
// and evidence, never a code candidate: the settled output carries the
// report and no candidate, publication or plan. A finish naming the
// wrong phase or an undecodable report fences for human attention
// instead of settling as fabricated success.
func settleResearch(assigned factory.Assignment, submission factory.MCPSubmission, now time.Time) (factory.Assignment, string, bool, error) {
	if assigned.Phase != project.ExecutionResearch {
		return fenceFW40(assigned, submission, now, "research finish does not match its assignment phase")
	}
	report, err := factory.DecodeResearchReport(submission.Tool, submission.Payload)
	if err != nil {
		return fenceFW40(assigned, submission, now, "invalid research report")
	}
	parsed := phaseResultFor(submission, project.ExecutionResearch, report.Conclusion, &factory.PhaseOutput{Research: &report})
	return finishFromMCPPhaseResult(assigned, submission, parsed, now), "research settled", true, nil
}
