package control

import (
	"encoding/json"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
)

// settleCorrectionFinish keeps the typed correction report all the way from
// the authenticated proposal to the existing correction admission. Findings
// come from the granted canonical context, never a worker replacement set.
// Dispositions and fix evidence remain nominations; native final/check/tree
// evidence is independently required before settlement and publication.
func settleCorrectionFinish(assigned factory.Assignment, submission factory.MCPSubmission, now time.Time) (factory.Assignment, string, bool, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(submission.Payload, &fields); err != nil {
		return fenceFW40(assigned, submission, now, "correction payload refused: "+err.Error())
	}
	raw, present := fields["dispositions"]
	if !isArrayRaw(raw, present) {
		return parkFW40()
	}
	if assigned.Phase != project.ExecutionCorrect || assigned.ReviewContext == nil || assigned.ReviewContext.ValidateFor(assigned) != nil {
		return fenceFW40(assigned, submission, now, "correction finish lacks its granted canonical review context")
	}
	context := assigned.ReviewContext
	digest, _ := stringField(fields, "finding_digest")
	candidate, _ := stringField(fields, "candidate")
	if digest != context.Cycle.FindingDigest || !factory.ValidCommit(candidate) {
		return fenceFW40(assigned, submission, now, "correction finish differs from its exact finding set or candidate")
	}
	report := factory.CorrectionReport{SetDigest: digest, NewCandidate: candidate, Findings: append([]factory.Finding(nil), context.CarriedFindings...)}
	if err := strictDecodeField(raw, &report.Dispositions); err != nil {
		return fenceFW40(assigned, submission, now, "correction dispositions refused: "+err.Error())
	}
	if evidence, present := fields["fix_evidence"]; present {
		if err := strictDecodeField(evidence, &report.Evidence); err != nil {
			return fenceFW40(assigned, submission, now, "correction fix evidence refused: "+err.Error())
		}
	}
	if err := report.Validate(); err != nil {
		return fenceFW40(assigned, submission, now, "correction report refused: "+err.Error())
	}
	parsed := phaseResultFor(submission, project.ExecutionCorrect, "worker proposes correction dispositions", &factory.PhaseOutput{Correction: &report})
	return finishFromMCPPhaseResult(assigned, submission, parsed, now), "typed correction finish settled", true, nil
}
