package factory

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/project"
)

func implementReceipt() *PhaseOutput {
	return &PhaseOutput{Implementation: &ImplementationReceipt{Candidate: strings.Repeat("e", 40)}}
}

func investigateOutput() *PhaseOutput {
	plan := planFixture()
	return &PhaseOutput{Plan: &plan}
}

// scopedChoiceID derives a deterministic valid decision ID for fixtures.
func scopedChoiceID(n int) string { return fmt.Sprintf("d%024x", 0xd000+n) }

func decomposeOutput() *PhaseOutput {
	return &PhaseOutput{Decomposition: &ScopedDecomposition{
		Intent: scopedChoiceID(1), Revision: scopedChoiceID(2), Selected: "keep",
		Tasks: []IntentTask{{
			Key: scopedChoiceID(3), Objective: "Request entry",
			Outcomes: []string{"Capture requests"}, IdeaIDs: []string{scopedChoiceID(4)},
		}},
		Advice: []string{strings.Repeat("a", 64)},
	}}
}

func reviewVerdictOutput() *PhaseOutput {
	return &PhaseOutput{Review: &ReviewCycleReport{
		CycleID:     strings.Repeat("c", 32),
		NewFindings: []Finding{testFinding(0, true)},
		Assessments: []FindingAssessment{
			{Key: testFindingKey(0), Status: FindingStillOpen, Evidence: "line 42 still lacks the check"},
		},
	}}
}

func reviewClearOutput() *PhaseOutput {
	return &PhaseOutput{Clear: &ClearReviewReport{
		CycleID: strings.Repeat("c", 32), Conclusion: ClearReviewConclusion,
		Head: strings.Repeat("e", 40), BaseOID: strings.Repeat("f", 40),
		CoverageDigest: strings.Repeat("b", 64),
		Assessments: []FindingAssessment{
			{Key: testFindingKey(0), Status: FindingVerified, Evidence: "line 42 now carries the check"},
		},
	}}
}

func correctionOutput() *PhaseOutput {
	findings := []Finding{testFinding(0, true), testFinding(1, false)}
	keys := []string{findings[0].Key, findings[1].Key}
	digest, err := FindingSetDigest(keys)
	if err != nil {
		panic(err)
	}
	return &PhaseOutput{Correction: &CorrectionReport{
		SetDigest: digest, Findings: findings,
		Dispositions: []CorrectionDisposition{
			{Key: findings[0].Key, Disposition: DispositionFixed},
			{Key: findings[1].Key, Disposition: DispositionDeclinedOptional},
		},
	}}
}

// T03 (FW06A scope): each valid phase/role/result combination binds;
// cross-phase payload, extra JSON and trailing JSON refuse.
func TestPhaseResultValidCombinations(t *testing.T) {
	plan := planFixture()
	cases := []PhaseResult{
		{Version: 2, Phase: project.ExecutionInvestigate, Status: "completed", Summary: "plan ready", Output: &PhaseOutput{Plan: &plan}},
		{Version: 2, Phase: project.ExecutionInvestigate, Status: "blocked", Summary: "waiting on input", Output: &PhaseOutput{Plan: &plan}},
		{Version: 2, Phase: project.ExecutionImplement, Status: "completed", Summary: "candidate ready", Output: implementReceipt()},
		{Version: 2, Phase: project.ExecutionImplement, Status: "failed", Summary: "crashed", Output: &PhaseOutput{Implementation: &ImplementationReceipt{DiagnosticRefs: []string{"log tail"}}}},
		{Version: 2, Phase: project.ExecutionDecompose, Status: "completed", Summary: "single choice ready", Output: decomposeOutput()},
		{Version: 2, Phase: project.ExecutionDecompose, Status: "blocked", Summary: "waiting on input"},
		{Version: 2, Phase: project.ExecutionReview, Status: "completed", Summary: "verdict recorded", Output: reviewVerdictOutput()},
		{Version: 2, Phase: project.ExecutionReview, Status: "completed", Summary: "explicit clear", Output: reviewClearOutput()},
		{Version: 2, Phase: project.ExecutionReview, Status: "blocked", Summary: "waiting on checks"},
		{Version: 2, Phase: project.ExecutionCorrect, Status: "completed", Summary: "findings accounted", Output: correctionOutput()},
		{Version: 2, Phase: project.ExecutionCorrect, Status: "failed", Summary: "no fix", Output: nil},
		{Version: 2, Phase: project.ExecutionReassess, Status: "failed", Summary: "held for attention", Output: nil},
	}
	for _, result := range cases {
		if err := result.Validate(); err != nil {
			t.Fatalf("valid phase result refused: %+v %v", result, err)
		}
	}
}

func TestPhaseResultRefusals(t *testing.T) {
	plan := planFixture()
	goodPlan := &PhaseOutput{Plan: &plan}
	cases := map[string]PhaseResult{
		"version":        {Version: 1, Phase: project.ExecutionImplement, Status: "completed", Summary: "x", Output: implementReceipt()},
		"phase":          {Version: 2, Phase: "publish", Status: "completed", Summary: "x", Output: implementReceipt()},
		"empty phase":    {Version: 2, Status: "completed", Summary: "x", Output: implementReceipt()},
		"status":         {Version: 2, Phase: project.ExecutionImplement, Status: "done", Summary: "x", Output: implementReceipt()},
		"summary":        {Version: 2, Phase: project.ExecutionImplement, Status: "completed", Output: implementReceipt()},
		"cross-phase":    {Version: 2, Phase: project.ExecutionInvestigate, Status: "completed", Summary: "x", Output: implementReceipt()},
		"plan-implement": {Version: 2, Phase: project.ExecutionImplement, Status: "completed", Summary: "x", Output: goodPlan},
		"no output":      {Version: 2, Phase: project.ExecutionImplement, Status: "completed", Summary: "x"},
		"candidate kept": {Version: 2, Phase: project.ExecutionImplement, Status: "failed", Summary: "x", Output: &PhaseOutput{Implementation: &ImplementationReceipt{Candidate: strings.Repeat("e", 40)}}},
		"no candidate":   {Version: 2, Phase: project.ExecutionImplement, Status: "completed", Summary: "x", Output: &PhaseOutput{Implementation: &ImplementationReceipt{}}},
		"diagnostics kept": {
			Version: 2, Phase: project.ExecutionImplement, Status: "completed", Summary: "x",
			Output: &PhaseOutput{Implementation: &ImplementationReceipt{Candidate: strings.Repeat("e", 40), DiagnosticRefs: []string{"log"}}},
		},
		"deferred review": {Version: 2, Phase: project.ExecutionReview, Status: "completed", Summary: "x", Output: goodPlan},
		"review both verdicts": {
			Version: 2, Phase: project.ExecutionReview, Status: "completed", Summary: "x",
			Output: &PhaseOutput{Review: reviewVerdictOutput().Review, Clear: reviewClearOutput().Clear},
		},
		"review no verdict":         {Version: 2, Phase: project.ExecutionReview, Status: "completed", Summary: "x", Output: &PhaseOutput{}},
		"review verdict on blocked": {Version: 2, Phase: project.ExecutionReview, Status: "blocked", Summary: "x", Output: reviewVerdictOutput()},
		"review verdict without anything blocking": {
			Version: 2, Phase: project.ExecutionReview, Status: "completed", Summary: "x",
			Output: &PhaseOutput{Review: &ReviewCycleReport{
				CycleID:     strings.Repeat("c", 32),
				Assessments: []FindingAssessment{{Key: testFindingKey(0), Status: FindingVerified, Evidence: "all clear"}},
			}},
		},
		"decompose cross-phase":       {Version: 2, Phase: project.ExecutionDecompose, Status: "completed", Summary: "x", Output: goodPlan},
		"decompose no payload":        {Version: 2, Phase: project.ExecutionDecompose, Status: "completed", Summary: "x", Output: &PhaseOutput{}},
		"decompose payload on failed": {Version: 2, Phase: project.ExecutionDecompose, Status: "failed", Summary: "x", Output: decomposeOutput()},
		"correct payload on failed":   {Version: 2, Phase: project.ExecutionCorrect, Status: "failed", Summary: "x", Output: correctionOutput()},
		"correct cross-phase":         {Version: 2, Phase: project.ExecutionCorrect, Status: "completed", Summary: "x", Output: implementReceipt()},
		"reassess plan":               {Version: 2, Phase: project.ExecutionReassess, Status: "failed", Summary: "x", Output: goodPlan},
		"reassess receipt":            {Version: 2, Phase: project.ExecutionReassess, Status: "failed", Summary: "x", Output: implementReceipt()},
		"reassess completed":          {Version: 2, Phase: project.ExecutionReassess, Status: "completed", Summary: "x"},
	}
	for name, result := range cases {
		if err := result.Validate(); err == nil {
			t.Fatalf("phase result %s admitted", name)
		}
	}
	// A completed planning result carries no candidate anywhere: the plan
	// shape has no candidate field, so it can never authorize publication.
	encoded, _ := json.Marshal(PhaseResult{Version: 2, Phase: project.ExecutionInvestigate, Status: "completed", Summary: "plan", Output: goodPlan})
	if strings.Contains(string(encoded), `"candidate":`) {
		t.Fatal("planning result exposes a candidate")
	}
	// Completed decompose, review and correct results carry no candidate
	// either: choosing boundaries, assessing findings and accounting
	// findings never authorize publication.
	for name, output := range map[string]*PhaseOutput{
		"decompose": decomposeOutput(), "review": reviewVerdictOutput(),
		"clear": reviewClearOutput(), "correct": correctionOutput(),
	} {
		phase := project.ExecutionDecompose
		switch name {
		case "review", "clear":
			phase = project.ExecutionReview
		case "correct":
			phase = project.ExecutionCorrect
		}
		encoded, _ := json.Marshal(PhaseResult{Version: 2, Phase: phase, Status: "completed", Summary: name, Output: output})
		if strings.Contains(string(encoded), `"candidate":`) {
			t.Fatalf("%s result exposes a candidate", name)
		}
	}
}

func TestParsePhaseResultIsStrict(t *testing.T) {
	plan := planFixture()
	payload, _ := json.Marshal(PhaseResult{Version: 2, Phase: project.ExecutionInvestigate, Status: "completed", Summary: "plan", Output: &PhaseOutput{Plan: &plan}})
	fenced := "noise\n```" + ResultFence + "\n" + string(payload) + "\n```\ntail"
	result, ok := ParsePhaseResult(fenced)
	if !ok || result.Phase != project.ExecutionInvestigate {
		t.Fatalf("valid fenced result refused: %v", ok)
	}
	// Unknown fields refuse, including every legacy harness field.
	legacy := `{"version":2,"phase":"implement","status":"completed","summary":"x","candidate":"` + strings.Repeat("e", 40) + `","review_passed":false,"findings":[]}`
	if _, ok := ParsePhaseResult("```" + ResultFence + "\n" + legacy + "\n```"); ok {
		t.Fatal("legacy harness payload admitted as phase result")
	}
	extra := `{"version":2,"phase":"implement","status":"completed","summary":"x","output":{"implementation":{"candidate":"` + strings.Repeat("e", 40) + `"}},"extra":1}`
	if _, ok := ParsePhaseResult("```" + ResultFence + "\n" + extra + "\n```"); ok {
		t.Fatal("extra JSON field admitted")
	}
	// Trailing JSON after the value refuses.
	trailing, _ := json.Marshal(PhaseResult{Version: 2, Phase: project.ExecutionCorrect, Status: "failed", Summary: "x"})
	if _, ok := ParsePhaseResult("```" + ResultFence + "\n" + string(trailing) + " {}\n```"); ok {
		t.Fatal("trailing JSON admitted")
	}
	if _, ok := ParsePhaseResult("no fence here"); ok {
		t.Fatal("unfenced output admitted")
	}
	// The last fenced block wins.
	first, _ := json.Marshal(PhaseResult{Version: 2, Phase: project.ExecutionCorrect, Status: "failed", Summary: "first"})
	if _, ok := ParsePhaseResult("```" + ResultFence + "\n" + string(first) + "\n```\n```" + ResultFence + "\n" + string(payload) + "\n```"); !ok {
		t.Fatal("last fenced block refused")
	}
}

func TestPhaseResultBindsItsAssignment(t *testing.T) {
	plan := planFixture()
	result := PhaseResult{Version: 2, Phase: project.ExecutionInvestigate, Status: "completed", Summary: "plan", Output: &PhaseOutput{Plan: &plan}}
	phased := Assignment{Phase: project.ExecutionInvestigate}
	if err := result.BindAssignment(phased); err != nil {
		t.Fatalf("bound result refused: %v", err)
	}
	if err := result.BindAssignment(Assignment{Phase: project.ExecutionImplement}); err == nil {
		t.Fatal("cross-phase binding admitted")
	}
	if err := result.BindAssignment(Assignment{}); err == nil {
		t.Fatal("pre-phase binding admitted")
	}
	bound := AssignmentResult{
		AssignmentID: NewID(), RunID: NewID(), RecordedUnix: 1700000000,
		Phase: project.ExecutionInvestigate, PhaseResult: &result,
	}
	if err := bound.ValidatePhaseBinding(project.RoleCoder, project.ExecutionInvestigate); err != nil {
		t.Fatalf("valid phase binding refused: %v", err)
	}
	for name, mutate := range map[string]func(*AssignmentResult){
		"role":     func(r *AssignmentResult) { r.Phase = project.ExecutionReview },
		"phase":    func(r *AssignmentResult) { r.Phase = project.ExecutionImplement },
		"payload":  func(r *AssignmentResult) { r.PhaseResult.Phase = project.ExecutionImplement },
		"missing":  func(r *AssignmentResult) { r.PhaseResult = nil },
		"unphased": func(r *AssignmentResult) { r.Phase = "" },
	} {
		next := bound
		out := *next.PhaseResult
		next.PhaseResult = &out
		mutate(&next)
		if err := next.ValidatePhaseBinding(project.RoleCoder, project.ExecutionInvestigate); err == nil {
			t.Fatalf("phase binding %s admitted", name)
		}
	}
}
