package factory

import (
	"encoding/json"
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

// T03 (FW06A scope): each valid phase/role/result combination binds;
// cross-phase payload, extra JSON and trailing JSON refuse.
func TestPhaseResultValidCombinations(t *testing.T) {
	plan := planFixture()
	cases := []PhaseResult{
		{Version: 2, Phase: project.ExecutionInvestigate, Status: "completed", Summary: "plan ready", Output: &PhaseOutput{Plan: &plan}},
		{Version: 2, Phase: project.ExecutionInvestigate, Status: "blocked", Summary: "waiting on input", Output: &PhaseOutput{Plan: &plan}},
		{Version: 2, Phase: project.ExecutionImplement, Status: "completed", Summary: "candidate ready", Output: implementReceipt()},
		{Version: 2, Phase: project.ExecutionImplement, Status: "failed", Summary: "crashed", Output: &PhaseOutput{Implementation: &ImplementationReceipt{DiagnosticRefs: []string{"log tail"}}}},
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
		"diagnostics kept": {Version: 2, Phase: project.ExecutionImplement, Status: "completed", Summary: "x",
			Output: &PhaseOutput{Implementation: &ImplementationReceipt{Candidate: strings.Repeat("e", 40), DiagnosticRefs: []string{"log"}}}},
		"deferred review":    {Version: 2, Phase: project.ExecutionReview, Status: "completed", Summary: "x", Output: goodPlan},
		"reassess plan":      {Version: 2, Phase: project.ExecutionReassess, Status: "failed", Summary: "x", Output: goodPlan},
		"reassess receipt":   {Version: 2, Phase: project.ExecutionReassess, Status: "failed", Summary: "x", Output: implementReceipt()},
		"reassess completed": {Version: 2, Phase: project.ExecutionReassess, Status: "completed", Summary: "x"},
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
