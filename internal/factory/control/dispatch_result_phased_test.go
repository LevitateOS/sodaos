package control

import (
	"crypto/sha256"
	"encoding/hex"
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
	a.PublicationAssignment = ""
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
	if finished.PublicationAssignment != "" {
		t.Fatalf("finished intent carries publication: %q", finished.PublicationAssignment)
	}
	if err := finished.Validate(); err != nil {
		t.Fatalf("finished intent assignment refused: %v", err)
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

// phaseTestRepairAssignment builds one valid stall, delivery or delivery
// repair cycle: a fresh repository root with no parent, publication or
// issue authority, mirroring the D15/D16 admission shape.
func phaseTestRepairAssignment(t *testing.T, phase, role string, subject factory.ExecutionSubject) factory.Assignment {
	t.Helper()
	id, run := factory.NewID(), factory.NewID()
	prompt := []byte("prompt bytes for " + phase)
	sum := sha256.Sum256(prompt)
	a := factory.Assignment{
		ID: id, AttemptRoot: id,
		Subject: subject, Phase: phase,
		ProjectID: "p" + strings.Repeat("1", 24), Role: role,
		Repository: 7, NativeRev: 1,
		Preparation: "f" + strings.Repeat("d", 24),
		Harness:     "codex-1.2.3", HarnessVers: "1.2.3",
		Model: "test-model", Connection: "conn",
		SourceCommit: phaseTestCommit(), Prompt: prompt, PromptSHA: hex.EncodeToString(sum[:]),
		Run: run, RunHistory: []string{run}, Stage: factory.AssignmentAssigned,
		Attempts: 1, CreatedUnix: 1700000000, ActorID: 7,
	}
	if err := a.Validate(); err != nil {
		t.Fatalf("repair assignment fixture invalid: %v", err)
	}
	return a
}

func phasedStallSubject() factory.ExecutionSubject {
	return factory.ExecutionSubject{
		Kind: factory.SubjectStall,
		Stall: &factory.StallSubject{
			StallID: strings.Repeat("7", 32), CycleID: strings.Repeat("8", 32),
			InventoryDigest: phaseTestSHA("inventory"),
		},
	}
}

func phasedDeliverySubject() factory.ExecutionSubject {
	return factory.ExecutionSubject{
		Kind: factory.SubjectDelivery,
		Delivery: &factory.DeliverySubject{
			DeliveryID: strings.Repeat("7", 32), IntentID: phasedDecisionID(20),
			AcceptedRevision: phasedDecisionID(21), ScopeDigest: phaseTestSHA("scope"),
			SourceCommit: phaseTestCommit(), PreparationDigest: phaseTestSHA("prep"),
			VerificationManifestDigest: phaseTestSHA("manifest"),
		},
	}
}

func phasedDeliveryRepairSubject() factory.ExecutionSubject {
	return factory.ExecutionSubject{
		Kind: factory.SubjectDeliveryRepair,
		DeliveryRepair: &factory.DeliveryRepairSubject{
			DeliveryID: strings.Repeat("7", 32), FailedVerificationRunID: strings.Repeat("8", 32),
			FailureDigest: phaseTestSHA("failure"), InventoryDigest: phaseTestSHA("inventory"),
			CycleID: strings.Repeat("9", 32),
		},
	}
}

func phasedStallRepair() *factory.PhaseOutput {
	return &factory.PhaseOutput{Repair: &factory.StallRepairReport{
		StallID: strings.Repeat("7", 32), CycleID: strings.Repeat("8", 32),
		InventoryDigest: phaseTestSHA("inventory"), Outcome: factory.RepairOutcomeRepair,
	}}
}

func phasedDeliveryRepair() *factory.PhaseOutput {
	return &factory.PhaseOutput{DeliveryRepair: &factory.DeliveryRepairReport{
		DeliveryID: strings.Repeat("7", 32), FailedVerificationRunID: strings.Repeat("8", 32),
		InputDigest: phaseTestSHA("input"), InventoryDigest: phaseTestSHA("inventory"),
		CycleID: strings.Repeat("9", 32), Outcome: factory.RepairOutcomeVerifiedWait,
	}}
}

func phasedAppVerification() *factory.PhaseOutput {
	return &factory.PhaseOutput{Verification: &factory.AppVerificationReport{
		DeliveryID: strings.Repeat("7", 32), InputDigest: phaseTestSHA("input"),
		Outcome: factory.AppVerificationVerified,
	}}
}

// A completed repair finishes with exactly its subject's report and no
// candidate: stall subjects carry the stall report, delivery repair
// subjects the delivery repair report.
func TestDerivePhasedRepairCompleted(t *testing.T) {
	for name, tc := range map[string]struct {
		subject factory.ExecutionSubject
		output  *factory.PhaseOutput
	}{
		"stall repair":    {phasedStallSubject(), phasedStallRepair()},
		"delivery repair": {phasedDeliveryRepairSubject(), phasedDeliveryRepair()},
	} {
		t.Run(name, func(t *testing.T) {
			a := phaseTestRepairAssignment(t, project.ExecutionRepair, project.RoleCoder, tc.subject)
			output := fencedResult(t, factory.PhaseResult{
				Version: 2, Phase: project.ExecutionRepair, Status: "completed",
				Summary: "repair proposed", Output: tc.output,
			})
			finished := finishPhasedFixture(t, a, phaseTestRun(a, factory.Succeeded), output)
			if finished.Outcome != factory.Succeeded || finished.Reason != factory.AssignReasonReported {
				t.Fatalf("repair outcome = %v/%v", finished.Outcome, finished.Reason)
			}
			result := finished.Result
			if !result.Reported || result.Phase != project.ExecutionRepair || result.Candidate != "" {
				t.Fatalf("repair result not exact: %+v", result)
			}
			if err := result.ValidatePhaseBinding(project.RoleCoder, project.ExecutionRepair); err != nil {
				t.Fatalf("completed repair does not bind: %v", err)
			}
			encoded, _ := json.Marshal(result.PhaseResult)
			if strings.Contains(string(encoded), `"candidate":`) {
				t.Fatal("completed repair exposes a candidate")
			}
			if err := finished.Validate(); err != nil {
				t.Fatalf("finished repair assignment refused: %v", err)
			}
		})
	}
}

// A completed verification finishes with exactly its app verification
// report and no candidate, legacy findings or reviewer report.
func TestDerivePhasedVerifyCompleted(t *testing.T) {
	a := phaseTestRepairAssignment(t, project.ExecutionVerify, project.RoleReviewer, phasedDeliverySubject())
	output := fencedResult(t, factory.PhaseResult{
		Version: 2, Phase: project.ExecutionVerify, Status: "completed",
		Summary: "delivery verified", Output: phasedAppVerification(),
	})
	finished := finishPhasedFixture(t, a, phaseTestRun(a, factory.Succeeded), output)
	if finished.Outcome != factory.Succeeded || finished.Reason != factory.AssignReasonReported {
		t.Fatalf("verify outcome = %v/%v", finished.Outcome, finished.Reason)
	}
	result := finished.Result
	if !result.Reported || result.Phase != project.ExecutionVerify || result.Review != nil {
		t.Fatalf("verify result not typed: %+v", result)
	}
	if result.Candidate != "" || len(result.Findings) != 0 || result.ReviewPassed {
		t.Fatalf("verify result authorizes publication: %+v", result)
	}
	if err := result.ValidatePhaseBinding(project.RoleReviewer, project.ExecutionVerify); err != nil {
		t.Fatalf("completed verification does not bind: %v", err)
	}
	if err := result.ValidateForAssignment(project.RoleReviewer, a.SourceCommit); err != nil {
		t.Fatalf("completed verification refused for reviewer: %v", err)
	}
	encoded, _ := json.Marshal(result.PhaseResult)
	if strings.Contains(string(encoded), `"candidate":`) {
		t.Fatal("completed verification exposes a candidate")
	}
	if err := finished.Validate(); err != nil {
		t.Fatalf("finished verify assignment refused: %v", err)
	}
}

// A valid report from another admitted subject or repair cycle must not
// settle this assignment, even when its phase and payload kind match.
func TestDerivePhasedRepairVerifyRejectsStaleIdentity(t *testing.T) {
	for _, tc := range []struct {
		name    string
		subject factory.ExecutionSubject
		output  func() *factory.PhaseOutput
		mutate  func(*factory.PhaseOutput)
	}{
		{"stall", phasedStallSubject(), phasedStallRepair, func(o *factory.PhaseOutput) { o.Repair.StallID = strings.Repeat("a", 32) }},
		{"stall cycle", phasedStallSubject(), phasedStallRepair, func(o *factory.PhaseOutput) { o.Repair.CycleID = strings.Repeat("a", 32) }},
		{"stall inventory", phasedStallSubject(), phasedStallRepair, func(o *factory.PhaseOutput) { o.Repair.InventoryDigest = phaseTestSHA("stale") }},
		{"repair delivery", phasedDeliveryRepairSubject(), phasedDeliveryRepair, func(o *factory.PhaseOutput) { o.DeliveryRepair.DeliveryID = strings.Repeat("a", 32) }},
		{"repair failed run", phasedDeliveryRepairSubject(), phasedDeliveryRepair, func(o *factory.PhaseOutput) { o.DeliveryRepair.FailedVerificationRunID = strings.Repeat("a", 32) }},
		{"repair cycle", phasedDeliveryRepairSubject(), phasedDeliveryRepair, func(o *factory.PhaseOutput) { o.DeliveryRepair.CycleID = strings.Repeat("a", 32) }},
		{"repair inventory", phasedDeliveryRepairSubject(), phasedDeliveryRepair, func(o *factory.PhaseOutput) { o.DeliveryRepair.InventoryDigest = phaseTestSHA("stale") }},
		{"verified delivery", phasedDeliverySubject(), phasedAppVerification, func(o *factory.PhaseOutput) { o.Verification.DeliveryID = strings.Repeat("a", 32) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			phase, role := project.ExecutionRepair, project.RoleCoder
			if tc.subject.Kind == factory.SubjectDelivery {
				phase, role = project.ExecutionVerify, project.RoleReviewer
			}
			a := phaseTestRepairAssignment(t, phase, role, tc.subject)
			report := factory.PhaseResult{Version: 2, Phase: phase, Status: "completed", Summary: "reported", Output: tc.output()}
			if err := report.BindAssignment(a); err != nil {
				t.Fatalf("matching report refused: %v", err)
			}
			tc.mutate(report.Output)
			if err := report.Validate(); err != nil {
				t.Fatalf("stale report must remain structurally valid: %v", err)
			}
			finished := finishPhasedFixture(t, a, phaseTestRun(a, factory.Succeeded), fencedResult(t, report))
			if finished.Outcome != factory.NeedsHuman || finished.Reason != factory.AssignReasonNoReport || finished.Result.Reported {
				t.Fatalf("stale report accepted: outcome=%s reason=%s reported=%v", finished.Outcome, finished.Reason, finished.Result.Reported)
			}
		})
	}
}

// Repair and verify refuse everything off-contract: cross-phase payloads,
// the other phase's report, wrong-subject reports and legacy harness
// shapes all synthesize to blocked without a recorded report.
func TestDerivePhasedRepairVerifyRefusals(t *testing.T) {
	repair := phaseTestRepairAssignment(t, project.ExecutionRepair, project.RoleCoder, phasedStallSubject())
	verify := phaseTestRepairAssignment(t, project.ExecutionVerify, project.RoleReviewer, phasedDeliverySubject())
	plan := factory.InvestigationPlan{
		ID: factory.NewID(), AttemptRoot: repair.AttemptRoot,
		ScopeDigest: phaseTestSHA("scope"), InputDigest: phaseTestSHA("input"),
		SourceCommit: phaseTestCommit(), IntegrationCommit: phaseTestCommit(),
		Preparation: repair.Preparation,
		Approach:    "inspect then plan", AcceptanceCoverage: []string{"accept-one"},
		Tasks: []factory.PlanTask{{
			ID: "task-one", Objective: "inspect", Outputs: []string{"notes"},
			AcceptanceRefs: []string{"accept-one"}, Lane: "lane-one",
			OwnedPaths: []string{"internal/x.go"},
		}},
		Lanes: []factory.PlanLane{{ID: "lane-one", TaskIDs: []string{"task-one"}}},
	}
	planDigest, err := factory.PlanDigest(plan)
	if err != nil {
		t.Fatal(err)
	}
	plan.Digest = planDigest
	legacy := `{"version":1,"phase":"implement","status":"completed","summary":"x","candidate":"` + strings.Repeat("e", 40) + `"}`
	for name, tc := range map[string]struct {
		a      factory.Assignment
		output string
	}{
		"plan on repair": {repair, fencedResult(t, factory.PhaseResult{
			Version: 2, Phase: project.ExecutionRepair, Status: "completed",
			Summary: "x", Output: &factory.PhaseOutput{Plan: &plan},
		})},
		"delivery report on stall repair": {repair, fencedResult(t, factory.PhaseResult{
			Version: 2, Phase: project.ExecutionRepair, Status: "completed",
			Summary: "x", Output: phasedDeliveryRepair(),
		})},
		"verification on repair": {repair, fencedResult(t, factory.PhaseResult{
			Version: 2, Phase: project.ExecutionRepair, Status: "completed",
			Summary: "x", Output: phasedAppVerification(),
		})},
		"stall report on verify": {verify, fencedResult(t, factory.PhaseResult{
			Version: 2, Phase: project.ExecutionVerify, Status: "completed",
			Summary: "x", Output: phasedStallRepair(),
		})},
		"legacy harness on verify": {verify, "notes\n```" + factory.ResultFence + "\n" + legacy + "\n```\n"},
		"unparseable repair":       {repair, "no fence here"},
	} {
		result, outcome, reason := deriveAttemptResult(tc.a, phaseTestRun(tc.a, factory.Succeeded), tc.output, "", 1700000100)
		if outcome != factory.NeedsHuman || reason != factory.AssignReasonNoReport {
			t.Fatalf("%s outcome = %v/%v", name, outcome, reason)
		}
		if result.Reported {
			t.Fatalf("%s reported: %+v", name, result)
		}
	}
}

// Non-completed repair and verify runs synthesize honestly with no
// payload, like every other phase.
func TestDerivePhasedRepairVerifyNonCompleted(t *testing.T) {
	repair := phaseTestRepairAssignment(t, project.ExecutionRepair, project.RoleCoder, phasedStallSubject())
	verify := phaseTestRepairAssignment(t, project.ExecutionVerify, project.RoleReviewer, phasedDeliverySubject())
	for name, a := range map[string]factory.Assignment{"repair": repair, "verify": verify} {
		run := phaseTestRun(a, factory.Failed)
		result, outcome, _ := deriveAttemptResult(a, run, "crashed without a report", "run failed", 1700000100)
		if outcome != factory.Failed || result.Reported {
			t.Fatalf("%s failed outcome = %v reported=%v", name, outcome, result.Reported)
		}
		if result.PhaseResult == nil || result.PhaseResult.Output != nil {
			t.Fatalf("%s failed result carries a payload: %+v", name, result.PhaseResult)
		}
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
