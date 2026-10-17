package control

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
)

// F7 phased-result tests: decompose, review and correct runs report the
// strict v2 phase contract with exactly their typed payload. The decompose
// leg finishes end to end on the current wire; review and correct legs
// prove derive plus phase binding, with the assignment_result.go wire
// (completed-typed exemption and reviewer phased branch) as the recorded
// follow-up before their finish path closes.

func phasedDecisionID(n int) string { return fmt.Sprintf("d%024x", 0xe000+n) }

func phasedFinding(n int, actionable bool) factory.Finding {
	return factory.Finding{
		Key: fmt.Sprintf("%032x", n), OriginRun: strings.Repeat("1", 32),
		Severity: factory.FindingSeverityMajor, Defect: "missing bound",
		Evidence: "line 42 lacks a check", RequiredOutcome: "add the check",
		FixKind: factory.FindingFixCode, Actionable: actionable,
	}
}

func phasedDecomposition() *factory.PhaseOutput {
	return &factory.PhaseOutput{Decomposition: &factory.ScopedDecomposition{
		Intent: phasedDecisionID(1), Revision: phasedDecisionID(2), Selected: "keep",
		Tasks: []factory.IntentTask{{
			Key: phasedDecisionID(3), Objective: "Request entry",
			Outcomes: []string{"Capture requests"}, IdeaIDs: []string{phasedDecisionID(4)},
		}},
		Advice: []string{strings.Repeat("a", 64)},
	}}
}

func phasedReviewVerdict() *factory.PhaseOutput {
	return &factory.PhaseOutput{Review: &factory.ReviewCycleReport{
		CycleID:     strings.Repeat("c", 32),
		NewFindings: []factory.Finding{phasedFinding(0, true)},
		Assessments: []factory.FindingAssessment{
			{Key: phasedFinding(0, true).Key, Status: factory.FindingStillOpen, Evidence: "line 42 still lacks the check"},
		},
	}}
}

func phasedReviewClear(head, base string) *factory.PhaseOutput {
	return &factory.PhaseOutput{Clear: &factory.ClearReviewReport{
		CycleID: strings.Repeat("c", 32), Conclusion: factory.ClearReviewConclusion,
		Head: head, BaseOID: base, CoverageDigest: strings.Repeat("b", 64),
		Assessments: []factory.FindingAssessment{
			{Key: phasedFinding(0, true).Key, Status: factory.FindingVerified, Evidence: "line 42 now carries the check"},
		},
	}}
}

func phasedCorrection(t *testing.T) *factory.PhaseOutput {
	t.Helper()
	findings := []factory.Finding{phasedFinding(0, true), phasedFinding(1, false)}
	digest, err := factory.FindingSetDigest([]string{findings[0].Key, findings[1].Key})
	if err != nil {
		t.Fatal(err)
	}
	return &factory.PhaseOutput{Correction: &factory.CorrectionReport{
		SetDigest: digest, Findings: findings,
		Dispositions: []factory.CorrectionDisposition{
			{Key: findings[0].Key, Disposition: factory.DispositionFixed},
			{Key: findings[1].Key, Disposition: factory.DispositionDeclinedOptional},
		},
	}}
}

// phaseTestIntentAssignment builds one valid intent-subject decompose
// assignment: intent subjects enter only through decompose, carry no issue
// authority and keep their own attempt root.
func phaseTestIntentAssignment(t *testing.T) factory.Assignment {
	t.Helper()
	a := phaseTestAssignment(t, project.ExecutionDecompose, project.RoleCoder, false)
	a.Subject = factory.ExecutionSubject{
		Kind: factory.SubjectIntent,
		Intent: &factory.IntentSubject{
			IntentID: phasedDecisionID(10), RevisionID: phasedDecisionID(11),
			ScopeDigest: phaseTestSHA("intent-scope"), OwnerID: 7,
		},
	}
	a.Issue, a.Acceptance = 0, ""
	if err := a.Validate(); err != nil {
		t.Fatalf("intent assignment fixture invalid: %v", err)
	}
	return a
}

// A completed decomposition finishes end to end: single chosen candidate,
// no commit or PR, and no publishable candidate anywhere.
func TestDerivePhasedDecomposeCompleted(t *testing.T) {
	a := phaseTestIntentAssignment(t)
	output := fencedResult(t, factory.PhaseResult{
		Version: 2, Phase: project.ExecutionDecompose, Status: "completed",
		Summary: "single choice ready", Output: phasedDecomposition(),
	})
	finished := finishPhasedFixture(t, a, phaseTestRun(a, factory.Succeeded), output)
	if finished.Outcome != factory.Succeeded || finished.Reason != factory.AssignReasonReported {
		t.Fatalf("decomposition outcome = %v/%v", finished.Outcome, finished.Reason)
	}
	result := finished.Result
	if result == nil || !result.Reported || result.Phase != project.ExecutionDecompose {
		t.Fatalf("decomposition result not bound: %+v", result)
	}
	if result.Candidate != "" || result.PhaseResult.Output.Decomposition == nil {
		t.Fatalf("decomposition result not exact: %+v", result)
	}
	encoded, _ := json.Marshal(result.PhaseResult)
	if strings.Contains(string(encoded), `"candidate":`) {
		t.Fatal("completed decomposition exposes a candidate")
	}
	if err := result.ValidatePhaseBinding(project.RoleCoder, project.ExecutionDecompose); err != nil {
		t.Fatalf("completed decomposition does not bind: %v", err)
	}
}

// A completed review derives with its typed verdict and no candidate,
// legacy string findings or approve verdict: both the findings verdict
// and the explicit clear.
func TestDerivePhasedReviewCompleted(t *testing.T) {
	for name, output := range map[string]func(head string) *factory.PhaseOutput{
		"findings verdict": func(string) *factory.PhaseOutput { return phasedReviewVerdict() },
		// The wire binds the explicit clear to the assignment's source:
		// the reviewer finished on that exact candidate.
		"explicit clear": func(head string) *factory.PhaseOutput {
			return phasedReviewClear(head, strings.Repeat("f", 40))
		},
	} {
		t.Run(name, func(t *testing.T) {
			a := phaseTestAssignment(t, project.ExecutionReview, project.RoleReviewer, true)
			fenced := fencedResult(t, factory.PhaseResult{
				Version: 2, Phase: project.ExecutionReview, Status: "completed",
				Summary: "verdict recorded", Output: output(a.SourceCommit),
			})
			finished := finishPhasedFixture(t, a, phaseTestRun(a, factory.Succeeded), fenced)
			if finished.Outcome != factory.Succeeded || finished.Reason != factory.AssignReasonReported {
				t.Fatalf("review outcome = %v/%v", finished.Outcome, finished.Reason)
			}
			result := finished.Result
			if !result.Reported || result.Phase != project.ExecutionReview || result.Review != nil {
				t.Fatalf("review result not typed: %+v", result)
			}
			if result.Candidate != "" || len(result.Findings) != 0 || result.ReviewPassed {
				t.Fatalf("review result authorizes publication: %+v", result)
			}
			if err := result.ValidatePhaseBinding(project.RoleReviewer, project.ExecutionReview); err != nil {
				t.Fatalf("completed review does not bind: %v", err)
			}
			encoded, _ := json.Marshal(result.PhaseResult)
			if strings.Contains(string(encoded), `"candidate":`) {
				t.Fatal("completed review exposes a candidate")
			}
		})
	}
}

// The phased reviewer refuses everything the legacy fence once accepted:
// string findings, approve verdicts, cross-phase payloads and silence.
// No legacy review reader is added on this path.
func TestDerivePhasedReviewRefusals(t *testing.T) {
	a := phaseTestAssignment(t, project.ExecutionReview, project.RoleReviewer, true)
	run := phaseTestRun(a, factory.Succeeded)
	legacy := "notes\n```" + factory.ReviewReportFence + "\n" +
		`{"verdict":"approve","summary":"looks good","body":"checked","findings":[]}` + "\n```\n"
	crossPhase := fencedResult(t, factory.PhaseResult{
		Version: 2, Phase: project.ExecutionImplement, Status: "completed",
		Summary: "candidate ready",
		Output:  &factory.PhaseOutput{Implementation: &factory.ImplementationReceipt{Candidate: strings.Repeat("e", 40)}},
	})
	for name, output := range map[string]string{
		"legacy string-findings report": legacy,
		"cross-phase payload":           crossPhase,
		"unparseable output":            "no fence here",
	} {
		result, outcome, reason := deriveAttemptResult(a, run, output, "", 1700000100)
		if outcome != factory.NeedsHuman || reason != factory.AssignReasonNoReport {
			t.Fatalf("%s outcome = %v/%v", name, outcome, reason)
		}
		if result.Reported {
			t.Fatalf("%s reported: %+v", name, result)
		}
	}
	// A clear for another head derives (shape only) but never finishes:
	// the wire binds the explicit clear to the assignment's source, so
	// the finished assignment refuses without a recorded result.
	wrongHead := fencedResult(t, factory.PhaseResult{
		Version: 2, Phase: project.ExecutionReview, Status: "completed",
		Summary: "verdict recorded",
		Output: &factory.PhaseOutput{Clear: &factory.ClearReviewReport{
			CycleID: strings.Repeat("c", 32), Conclusion: factory.ClearReviewConclusion,
			Head: strings.Repeat("3", 40), BaseOID: strings.Repeat("f", 40),
			CoverageDigest: strings.Repeat("b", 64),
		}},
	})
	derived, outcome, _ := deriveAttemptResult(a, run, wrongHead, "", 1700000100)
	if outcome != factory.Succeeded {
		t.Fatalf("clear shape refused at derive: %v", outcome)
	}
	if err := derived.ValidateForAssignment(project.RoleReviewer, a.SourceCommit); err == nil {
		t.Fatal("clear for another head finished")
	}
}

// A completed correction finishes with exactly its typed dispositions.
func TestDerivePhasedCorrectCompleted(t *testing.T) {
	a := phaseTestAssignment(t, project.ExecutionCorrect, project.RoleCoder, true)
	output := fencedResult(t, factory.PhaseResult{
		Version: 2, Phase: project.ExecutionCorrect, Status: "completed",
		Summary: "findings accounted", Output: phasedCorrection(t),
	})
	finished := finishPhasedFixture(t, a, phaseTestRun(a, factory.Succeeded), output)
	if finished.Outcome != factory.Succeeded || finished.Reason != factory.AssignReasonReported {
		t.Fatalf("correction outcome = %v/%v", finished.Outcome, finished.Reason)
	}
	result := finished.Result
	if !result.Reported || result.Phase != project.ExecutionCorrect || result.Candidate != "" {
		t.Fatalf("correction result not exact: %+v", result)
	}
	if result.PhaseResult.Output.Correction == nil {
		t.Fatalf("correction result lacks its dispositions: %+v", result)
	}
	if err := result.ValidatePhaseBinding(project.RoleCoder, project.ExecutionCorrect); err != nil {
		t.Fatalf("completed correction does not bind: %v", err)
	}
}

// Non-completed decompose, review and correct runs synthesize honestly
// with no payload, like every other phase.
func TestDerivePhasedDecomposeReviewCorrectNonCompleted(t *testing.T) {
	for _, tc := range []struct {
		phase, role string
		intent      bool
	}{
		{project.ExecutionDecompose, project.RoleCoder, true},
		{project.ExecutionReview, project.RoleReviewer, false},
		{project.ExecutionCorrect, project.RoleCoder, false},
	} {
		var a factory.Assignment
		if tc.intent {
			a = phaseTestIntentAssignment(t)
		} else {
			a = phaseTestAssignment(t, tc.phase, tc.role, true)
		}
		run := phaseTestRun(a, factory.Failed)
		result, outcome, _ := deriveAttemptResult(a, run, "crashed without a report", "run failed", 1700000100)
		if outcome != factory.Failed || result.Reported {
			t.Fatalf("%s failed outcome = %v reported=%v", tc.phase, outcome, result.Reported)
		}
		if result.PhaseResult == nil || result.PhaseResult.Output != nil {
			t.Fatalf("%s failed result carries a payload: %+v", tc.phase, result.PhaseResult)
		}
	}
}
