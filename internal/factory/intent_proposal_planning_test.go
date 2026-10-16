package factory

import (
	"fmt"
	"strings"
	"testing"
)

// FW08 pre-issue planning scope: a first draft with attributed effective
// choices plans without tasks, without an accepted graph and without
// history. IDs are deterministic fixtures, never native issue IDs.

func draftPlanningHead() IntentRevision {
	return IntentRevision{
		ID: "d111111111111111111111111", Intent: "d222222222222222222222222",
		Owner: 1, Repository: 42, Objective: "Build a leave approval workflow so managers can review time-off requests",
		State: "draft", MaxIssues: 10, PlanningCalls: 3, CreatedUnix: 1,
		Ideas: []ProductIdea{
			{ID: "d333333333333333333333333", Description: "Capture leave requests", Consequence: "Requests can be recorded", Decision: "accept", Source: "requested", Provenance: "human", Actor: 1},
			{ID: "d444444444444444444444444", Description: "Managers review requests", Consequence: "Decisions are visible", Decision: "accept", Source: "requested", Provenance: "human", Actor: 1},
		},
		EffectiveOutcomes: []string{"d333333333333333333333333", "d444444444444444444444444"},
	}
}

func candidateKey(n int) string { return fmt.Sprintf("d%024x", 0xa000+n) }

func TestDraftPlanningScopeNeedsAttributedEffectiveChoices(t *testing.T) {
	if err := ValidateDraftPlanningScope(draftPlanningHead()); err != nil {
		t.Fatalf("first draft refused: %v", err)
	}
	accepted := draftPlanningHead()
	accepted.State = "accepted"
	if err := ValidateDraftPlanningScope(accepted); err == nil {
		t.Fatal("accepted revision planned through the pre-issue path")
	}
	unattributed := draftPlanningHead()
	unattributed.Ideas = append([]ProductIdea(nil), unattributed.Ideas...)
	unattributed.Ideas[0].Source = ""
	if err := ValidateDraftPlanningScope(unattributed); err == nil {
		t.Fatal("unattributed effective choice admitted")
	}
	// An unrelated material pending branch stays pending while authorized
	// outcomes progress; it only blocks when a candidate references it.
	pending := draftPlanningHead()
	pending.Ideas = append(append([]ProductIdea(nil), pending.Ideas...), ProductIdea{ID: "d555555555555555555555555", Description: "Sync with payroll", Consequence: "Needs a product decision", Decision: "pending", Source: "autonomous", Provenance: "autonomous", Material: true})
	if err := ValidateDraftPlanningScope(pending); err != nil {
		t.Fatalf("unrelated pending branch blocked planning: %v", err)
	}
}

func broadPlanningTasks() []IntentTask {
	return []IntentTask{
		{Key: candidateKey(1), Objective: "Capture leave requests", Outcomes: []string{"Requests can be recorded"}, IdeaIDs: []string{"d333333333333333333333333"}},
		{Key: candidateKey(2), Objective: "Show captured requests", Outcomes: []string{"Requests can be found"}, IdeaIDs: []string{"d333333333333333333333333"}},
		{Key: candidateKey(3), Objective: "Managers review requests", Outcomes: []string{"Decisions are visible"}, IdeaIDs: []string{"d444444444444444444444444"}, DependsOn: []string{candidateKey(1)}},
	}
}

func TestAssessProposedTasksAcceptsBroadAndSmall(t *testing.T) {
	r := draftPlanningHead()
	if err := AssessProposedTasks(r, broadPlanningTasks(), r.MaxIssues); err != nil {
		t.Fatalf("broad output refused: %v", err)
	}
	small := []IntentTask{
		{Key: candidateKey(9), Objective: "Correct the approval confirmation wording", Outcomes: []string{"Requests can be recorded", "Decisions are visible"}, IdeaIDs: []string{"d333333333333333333333333", "d444444444444444444444444"}},
	}
	if err := AssessProposedTasks(r, small, r.MaxIssues); err != nil {
		t.Fatalf("small output refused: %v", err)
	}
	// Assessment is pure: replay reaches the same verdict and spends nothing.
	if err := AssessProposedTasks(r, broadPlanningTasks(), r.MaxIssues); err != nil {
		t.Fatalf("replay diverged: %v", err)
	}
}

func TestAssessProposedTasksRefusesUncoveredFalseAndOverBroad(t *testing.T) {
	r := draftPlanningHead()
	uncovered := []IntentTask{
		{Key: candidateKey(1), Objective: "Capture leave requests", Outcomes: []string{"Requests can be recorded"}, IdeaIDs: []string{"d333333333333333333333333"}},
	}
	if err := AssessProposedTasks(r, uncovered, r.MaxIssues); err == nil || !strings.Contains(err.Error(), "uncovered") {
		t.Fatalf("uncovered authorized outcome passed: %v", err)
	}
	falseDep := broadPlanningTasks()
	falseDep[2].DependsOn = []string{candidateKey(77)}
	if err := AssessProposedTasks(r, falseDep, r.MaxIssues); err == nil {
		t.Fatal("false dependency passed")
	}
	cyclic := broadPlanningTasks()
	cyclic[0].DependsOn = []string{candidateKey(3)}
	if err := AssessProposedTasks(r, cyclic, r.MaxIssues); err == nil {
		t.Fatal("cycle passed")
	}
	pending := broadPlanningTasks()
	pending[0].IdeaIDs = []string{"d555555555555555555555555"}
	if err := AssessProposedTasks(r, pending, r.MaxIssues); err == nil {
		t.Fatal("undecided choice reference passed")
	}
	native := broadPlanningTasks()
	native[0].Key = "42"
	if err := AssessProposedTasks(r, native, r.MaxIssues); err == nil {
		t.Fatal("native issue ID passed as a task key")
	}
	if err := AssessProposedTasks(r, broadPlanningTasks(), 2); err == nil {
		t.Fatal("over-bound candidate passed")
	}
	if err := AssessProposedTasks(r, nil, r.MaxIssues); err == nil {
		t.Fatal("empty candidate passed")
	}
}
