package factory

import "testing"

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
