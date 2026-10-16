package factory

import (
	"strings"
	"testing"
)

func checkFixture() CheckSpec {
	return CheckSpec{ID: "unit", Argv: []string{"go", "test", "./internal/factory/"}, Cwd: ".", TimeoutSeconds: 300, Scope: CheckScopeTask, Required: true}
}

func graphFixture() ([]PlanTask, []PlanLane, []CheckSpec) {
	tasks := []PlanTask{
		{
			ID: "design", Objective: "Settle the approach", Outputs: []string{"approach note"},
			AcceptanceRefs: []string{"accept-1"}, Lane: "main",
			OwnedPaths: []string{"docs/plan.md"}, CheckIDs: []string{"unit"},
		},
		{
			ID: "build", Objective: "Implement the change", Outputs: []string{"candidate diff"},
			AcceptanceRefs: []string{"accept-1"}, DependsOn: []string{"design"}, Lane: "main",
			OwnedPaths: []string{"internal/factory/planning.go"}, InterfaceWrite: []string{"plan-bytes"},
			CheckIDs: []string{"unit"},
		},
		{
			ID: "docs", Objective: "Record the contract", Outputs: []string{"guide delta"},
			AcceptanceRefs: []string{"accept-2"}, Lane: "side",
			OwnedPaths: []string{"docs/reference/factory.md"},
		},
	}
	lanes := []PlanLane{{ID: "main", TaskIDs: []string{"design", "build"}}, {ID: "side", TaskIDs: []string{"docs"}}}
	return tasks, lanes, []CheckSpec{checkFixture()}
}

// planFixture returns a valid executable investigation plan. Callers mutate
// the copy to prove each refusal.
func planFixture() InvestigationPlan {
	tasks, lanes, checks := graphFixture()
	commit := strings.Repeat("c", 40)
	plan := InvestigationPlan{
		ID: NewID(), AttemptRoot: NewID(),
		ScopeDigest: strings.Repeat("a", 64), InputDigest: strings.Repeat("b", 64),
		SourceCommit: commit, IntegrationCommit: commit,
		Preparation: "f" + strings.Repeat("d", 24),
		Approach:    "Investigate, plan, then implement behind checks.",
		Inspected: []InspectedSource{
			{Path: "internal/factory/planning.go", Commit: commit, Evidence: "placeholder steps lack evidence"},
		},
		Findings:           []string{"planning path carries no task graph"},
		AcceptanceCoverage: []string{"accept-1", "accept-2"},
		Rejected:           []RejectedAlternative{{Approach: "hand-written steps", Reason: "no validation or receipts"}},
		Tasks:              tasks, Lanes: lanes, Checks: checks,
	}
	digest, err := PlanDigest(plan)
	if err != nil {
		panic(err)
	}
	plan.Digest = digest
	return plan
}

func TestPlanKeys(t *testing.T) {
	for _, key := range []string{"a", "task-1", "lane_2", "check-9_x", "a" + strings.Repeat("b", 31)} {
		if !ValidPlanKey(key) {
			t.Fatalf("valid plan key refused: %s", key)
		}
	}
	for _, key := range []string{"", "A", "1a", "a$b", "a" + strings.Repeat("b", 32), "a b"} {
		if ValidPlanKey(key) {
			t.Fatalf("invalid plan key admitted: %q", key)
		}
	}
}

func TestCheckSpecBounds(t *testing.T) {
	if err := checkFixture().Validate(); err != nil {
		t.Fatalf("valid check refused: %v", err)
	}
	cases := map[string]func(*CheckSpec){
		"key":     func(c *CheckSpec) { c.ID = "Bad" },
		"scope":   func(c *CheckSpec) { c.Scope = "suite" },
		"timeout": func(c *CheckSpec) { c.TimeoutSeconds = 601 },
		"argv":    func(c *CheckSpec) { c.Argv = []string{} },
		"cwd":     func(c *CheckSpec) { c.Cwd = "/tmp" },
	}
	for name, mutate := range cases {
		next := checkFixture()
		mutate(&next)
		if err := next.Validate(); err == nil {
			t.Fatalf("check %s admitted", name)
		}
	}
	big := checkFixture()
	big.Argv = make([]string, MaxCheckArgv+1)
	for i := range big.Argv {
		big.Argv[i] = "x"
	}
	if big.Validate() == nil {
		t.Fatal("oversized argv admitted")
	}
}

func TestOwnershipEntries(t *testing.T) {
	for _, entry := range []string{"internal/factory/planning.go", "docs/", "frontend/src/work.ts"} {
		if !validOwnershipEntry(entry) {
			t.Fatalf("valid ownership refused: %s", entry)
		}
	}
	for _, entry := range []string{"", ".", "internal/*.go", "/tmp/x", "../x", "a/../b", ".git/config", "x/.soda-home/y", "a\\b", "x]"} {
		if validOwnershipEntry(entry) {
			t.Fatalf("invalid ownership admitted: %q", entry)
		}
	}
}

func TestValidatePlanGraphValid(t *testing.T) {
	tasks, lanes, checks := graphFixture()
	if err := ValidatePlanGraph(tasks, lanes, checks); err != nil {
		t.Fatalf("valid graph refused: %v", err)
	}
}

func TestValidatePlanGraphRefusals(t *testing.T) {
	cases := map[string]func(*[]PlanTask, *[]PlanLane, *[]CheckSpec){
		"duplicate task": func(tasks *[]PlanTask, _ *[]PlanLane, _ *[]CheckSpec) {
			(*tasks)[1].ID = (*tasks)[0].ID
		},
		"unknown lane": func(tasks *[]PlanTask, _ *[]PlanLane, _ *[]CheckSpec) {
			(*tasks)[0].Lane = "ghost"
		},
		"lane mismatch": func(tasks *[]PlanTask, _ *[]PlanLane, _ *[]CheckSpec) {
			(*tasks)[0].Lane = "side"
		},
		"uncovered task": func(_ *[]PlanTask, lanes *[]PlanLane, _ *[]CheckSpec) {
			(*lanes)[0].TaskIDs = (*lanes)[0].TaskIDs[:1]
		},
		"unknown dep": func(tasks *[]PlanTask, _ *[]PlanLane, _ *[]CheckSpec) {
			(*tasks)[0].DependsOn = []string{"ghost"}
		},
		"unordered dep": func(tasks *[]PlanTask, _ *[]PlanLane, _ *[]CheckSpec) {
			(*tasks)[0].DependsOn = []string{"build"}
		},
		"unknown check": func(tasks *[]PlanTask, _ *[]PlanLane, _ *[]CheckSpec) {
			(*tasks)[0].CheckIDs = []string{"ghost"}
		},
		"no outputs": func(tasks *[]PlanTask, _ *[]PlanLane, _ *[]CheckSpec) {
			(*tasks)[0].Outputs = nil
		},
		"no acceptance": func(tasks *[]PlanTask, _ *[]PlanLane, _ *[]CheckSpec) {
			(*tasks)[0].AcceptanceRefs = nil
		},
		"glob ownership": func(tasks *[]PlanTask, _ *[]PlanLane, _ *[]CheckSpec) {
			(*tasks)[0].OwnedPaths = []string{"internal/*"}
		},
	}
	for name, mutate := range cases {
		tasks, lanes, checks := graphFixture()
		mutate(&tasks, &lanes, &checks)
		if err := ValidatePlanGraph(tasks, lanes, checks); err == nil {
			t.Fatalf("graph %s admitted", name)
		}
	}
	// Concurrent cross-lane overlap without a dependency refuses; the same
	// pair with an ordering dependency passes.
	tasks, lanes, checks := graphFixture()
	tasks[2].OwnedPaths = []string{"internal/factory/planning.go"}
	if err := ValidatePlanGraph(tasks, lanes, checks); err == nil {
		t.Fatal("concurrent ownership overlap admitted")
	}
	tasks[2].DependsOn = []string{"build"}
	if err := ValidatePlanGraph(tasks, lanes, checks); err != nil {
		t.Fatalf("serialized overlap refused: %v", err)
	}
	tasks, lanes, checks = graphFixture()
	tasks[2].InterfaceWrite = []string{"plan-bytes"}
	if err := ValidatePlanGraph(tasks, lanes, checks); err == nil {
		t.Fatal("concurrent interface write admitted")
	}
	// Same-lane tasks serialize, so shared ownership inside one lane passes.
	tasks, lanes, checks = graphFixture()
	tasks[1].OwnedPaths = []string{"docs/plan.md"}
	if err := ValidatePlanGraph(tasks, lanes, checks); err != nil {
		t.Fatalf("same-lane overlap refused: %v", err)
	}
}

func TestValidatePlanGraphBounds(t *testing.T) {
	tasks, _, checks := graphFixture()
	many := make([]PlanTask, 0, MaxPlanTasks+1)
	ids := make([]string, 0, MaxPlanTasks+1)
	for i := 0; i <= MaxPlanTasks; i++ {
		next := tasks[0]
		next.ID = "t" + strings.Repeat("k", 3) + string(rune('a'+i/26)) + string(rune('a'+i%26))
		many = append(many, next)
		ids = append(ids, next.ID)
	}
	if err := ValidatePlanGraph(many, []PlanLane{{ID: "main", TaskIDs: ids}}, checks); err == nil {
		t.Fatal("oversized task set admitted")
	}
	lanes := []PlanLane{
		{ID: "l1", TaskIDs: []string{"design"}}, {ID: "l2", TaskIDs: []string{"build"}},
		{ID: "l3", TaskIDs: []string{"docs"}}, {ID: "l4", TaskIDs: []string{}},
		{ID: "l5", TaskIDs: []string{}},
	}
	if err := ValidatePlanGraph(tasks, lanes, checks); err == nil {
		t.Fatal("oversized lane set admitted")
	}
	over := make([]CheckSpec, 0, MaxPlanChecks+1)
	for i := 0; i <= MaxPlanChecks; i++ {
		next := checkFixture()
		next.ID = "c" + strings.Repeat("k", 3) + string(rune('a'+i/26)) + string(rune('a'+i%26))
		over = append(over, next)
	}
	if err := ValidatePlanGraph(tasks, []PlanLane{{ID: "main", TaskIDs: []string{"design", "build", "docs"}}}, over); err == nil {
		t.Fatal("oversized check set admitted")
	}
	owned := tasks
	owned[0].OwnedPaths = make([]string, MaxTaskOwnership+1)
	for i := range owned[0].OwnedPaths {
		owned[0].OwnedPaths[i] = "docs/p" + string(rune('a'+i%26)) + ".md"
	}
	if err := ValidatePlanGraph(owned, []PlanLane{{ID: "main", TaskIDs: []string{"design", "build", "docs"}}}, checks); err == nil {
		t.Fatal("oversized ownership admitted")
	}
	big, lanes, checks := graphFixture()
	big[0].Objective = strings.Repeat("x", MaxPlanTextBytes+1)
	if err := ValidatePlanGraph(big, lanes, checks); err == nil {
		t.Fatal("oversized objective admitted")
	}
}
