package factory

import (
	"strings"
	"testing"
)

func TestPlanningPendingOutcomesAreProvisionalAndRelevantChangesInvalidate(t *testing.T) {
	control := IssueControl{Repository: 7, Issue: 3, Acceptance: "accepted", Readiness: ReadinessBlocked, NativeRev: 40, Blockers: []Blocker{{Code: BlockerCodePending}}}
	if !ReadyToPlan(control) {
		t.Fatal("pending visible outcome prevents early planning")
	}
	input := PlanningInput{Candidate: PriorityCandidate{ID: "7/3", Control: control, Source: "original"}, Missing: control.Blockers}
	old, _ := PlanningSnapshot(input)
	input.Candidate.Control.NativeRev++
	input.Candidate.Control.AssessedUnix++
	unchanged, _ := PlanningSnapshot(input)
	if old != unchanged {
		t.Fatal("unrelated native observation invalidated plan")
	}
	input.Candidate.Source = "changed"
	changed, _ := PlanningSnapshot(input)
	if old == changed {
		t.Fatal("source change kept stale plan")
	}
	control.Blockers = []Blocker{{Code: BlockerCycle}}
	if ReadyToPlan(control) {
		t.Fatal("cycle was treated as missing future outcome")
	}
}

// T11 (FW06A scope): unique keys, DAG, coverage, ownership and limits
// validate on the canonical plan revision; the semantic digest is stable
// and generation-scoped, so a replaced head never rewrites history.
func TestInvestigationPlanValidatesAndExecutes(t *testing.T) {
	plan := planFixture()
	if err := plan.Validate(); err != nil {
		t.Fatalf("valid plan refused: %v", err)
	}
	if err := plan.Executable(); err != nil {
		t.Fatalf("complete plan refused execution: %v", err)
	}
	again, err := PlanDigest(plan)
	if err != nil || again != plan.Digest {
		t.Fatal("plan digest unstable")
	}
	next := plan
	next.Generation = 1
	nextDigest, err := PlanDigest(next)
	if err != nil || nextDigest == plan.Digest {
		t.Fatal("plan generation shares its digest")
	}
}

func TestInvestigationPlanRefusals(t *testing.T) {
	cases := map[string]func(*InvestigationPlan){
		"identity":    func(p *InvestigationPlan) { p.ID = "short" },
		"generation":  func(p *InvestigationPlan) { p.Generation = -1 },
		"scope":       func(p *InvestigationPlan) { p.ScopeDigest = "short" },
		"source":      func(p *InvestigationPlan) { p.SourceCommit = strings.Repeat("z", 40) },
		"preparation": func(p *InvestigationPlan) { p.Preparation = "bad" },
		"inspected path": func(p *InvestigationPlan) {
			p.Inspected[0].Path = "internal/*.go"
		},
		"inspected commit": func(p *InvestigationPlan) {
			p.Inspected[0].Commit = "short"
		},
		"approach": func(p *InvestigationPlan) { p.Approach = "" },
		"alternative": func(p *InvestigationPlan) {
			p.Rejected[0].Reason = ""
		},
		"coverage": func(p *InvestigationPlan) { p.AcceptanceCoverage = nil },
		"digest":   func(p *InvestigationPlan) { p.Digest = strings.Repeat("0", 64) },
		"graph": func(p *InvestigationPlan) {
			p.Tasks[1].DependsOn = []string{"docs"}
		},
	}
	for name, mutate := range cases {
		plan := planFixture()
		mutate(&plan)
		if err := plan.Validate(); err == nil {
			t.Fatalf("plan %s admitted", name)
		}
	}
	// Unresolved mandatory facts validate as a record but never execute.
	plan := planFixture()
	plan.Unresolved = []string{"auth boundary undecided"}
	plan.Digest, _ = PlanDigest(plan)
	if err := plan.Validate(); err != nil {
		t.Fatalf("blocked plan refused as record: %v", err)
	}
	if err := plan.Executable(); err == nil {
		t.Fatal("unresolved plan admitted for execution")
	}
	// Too many inspected sources refuse.
	plan = planFixture()
	plan.Inspected = make([]InspectedSource, MaxInspectedRefs+1)
	for i := range plan.Inspected {
		plan.Inspected[i] = InspectedSource{Path: "docs/p.md", Commit: strings.Repeat("c", 40), Evidence: "e"}
	}
	plan.Digest, _ = PlanDigest(plan)
	if err := plan.Validate(); err == nil {
		t.Fatal("oversized inspection admitted")
	}
}
