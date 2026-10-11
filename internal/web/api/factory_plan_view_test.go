package api

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/store"
)

func TestPlanFreshnessStates(t *testing.T) {
	if got := planFreshnessDTO(planFreshnessInput{}); got.State != planFreshMissing || got.Provisional {
		t.Fatalf("missing: %+v", got)
	}
	current := planFreshnessInput{HasPlan: true, HasControl: true, PlanAcceptance: "a1", ControlAcceptance: "a1", PlanFingerprint: "f1", ControlFingerprint: "f1"}
	if got := planFreshnessDTO(current); got.State != planFreshCurrent || got.Provisional || len(got.Reasons) != 0 {
		t.Fatalf("current: %+v", got)
	}
	provisional := current
	provisional.Provisional = true
	if got := planFreshnessDTO(provisional); got.State != planFreshProvisional || !got.Provisional {
		t.Fatalf("provisional: %+v", got)
	}
	changed := provisional
	changed.ControlAcceptance = "a2"
	if got := planFreshnessDTO(changed); got.State != planFreshStale || !got.Provisional || len(got.Reasons) != 1 || got.Reasons[0] != planReasonAcceptance {
		t.Fatalf("acceptance changed: %+v", got)
	}
	drifted := current
	drifted.ControlFingerprint = "f2"
	if got := planFreshnessDTO(drifted); got.State != planFreshStale || got.Reasons[0] != planReasonInputs {
		t.Fatalf("inputs changed: %+v", got)
	}
	uncontrolled := current
	uncontrolled.HasControl = false
	if got := planFreshnessDTO(uncontrolled); got.State != planFreshStale || got.Reasons[0] != planReasonNoReadiness {
		t.Fatalf("readiness missing: %+v", got)
	}
}

func TestPlanJudgmentDTO(t *testing.T) {
	if got := planJudgmentDTO(factory.PriorityAnswer{}); got != nil {
		t.Fatalf("empty answer rendered: %+v", got)
	}
	judgment := planJudgmentDTO(factory.PriorityAnswer{Selected: "7/42", Model: "jev-1.13.0", Confidence: 0.8, Probabilities: map[string]float64{"7/42": 0.8, "7/43": 0.2}})
	if judgment == nil || len(judgment.Probabilities) != 2 || judgment.ProbabilityN != 2 {
		t.Fatalf("bounded judgment: %+v", judgment)
	}
	big := map[string]float64{}
	for i := 0; i < maxRenderedProbabilities+1; i++ {
		big[fmt.Sprintf("7/%d", i)] = 0
	}
	capped := planJudgmentDTO(factory.PriorityAnswer{Selected: "7/0", Probabilities: big})
	if capped == nil || capped.Probabilities != nil || capped.ProbabilityN != maxRenderedProbabilities+1 {
		t.Fatalf("oversized judgment: %+v", capped)
	}
}

func TestPlanDTORendersEmptyArrays(t *testing.T) {
	body, err := json.Marshal(planDTO(factory.PlanningRecord{Snapshot: "s", Outcome: "planned"}))
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"steps", "missing"} {
		items, ok := decoded[key].([]any)
		if !ok || len(items) != 0 {
			t.Fatalf("%s: %v", key, decoded[key])
		}
	}
}

func TestPriorityComparisonMasksText(t *testing.T) {
	secret := "secret-native-text"
	receipt := factory.PriorityReceipt{
		Snapshot: strings.Repeat("a", 64), Outcome: "selected", Selected: "7/42", CreatedUnix: 100,
		Candidates: []factory.PriorityCandidate{
			{ID: "7/42", Control: factory.IssueControl{Repository: 7, Issue: 42, Readiness: "queued", FirstSeenUnix: 10, AssessedUnix: 90, Blockers: []factory.Blocker{{Code: "pending", Detail: secret}}}, Objective: secret, Impact: "known-impact"},
			{ID: "7/43", Control: factory.IssueControl{Repository: 7, Issue: 43, Readiness: "queued", FirstSeenUnix: 20}, Objective: secret},
			{ID: "8/9", Control: factory.IssueControl{Repository: 8, Issue: 9, Readiness: "queued", FirstSeenUnix: 30}, Objective: secret},
		},
		Calls: []factory.PriorityCall{{Answer: factory.PriorityAnswer{Selected: "7/42", Model: "jev-1.13.0", Confidence: 0.7, Probabilities: map[string]float64{"7/42": 0.7, "7/43": 0.2, "8/9": 0.1}}}},
	}
	view := priorityComparisonDTO(receipt, 7, 200)
	if len(view.Candidates) != 2 || view.CandidatesTotal != 3 || view.OtherCount != 1 {
		t.Fatalf("candidates: %+v", view)
	}
	if !view.SelectedVisible || view.Selected != "7/42" {
		t.Fatalf("selected: %+v", view)
	}
	if len(view.Probabilities) != 2 || view.ProbabilitiesFull || view.ProbabilityN != 3 {
		t.Fatalf("distribution: %+v", view)
	}
	if view.Candidates[0].AgeSeconds != 190 || !view.Candidates[0].ImpactKnown || view.Candidates[1].ImpactKnown {
		t.Fatalf("ages/evidence: %+v", view.Candidates)
	}
	body, err := json.Marshal(view)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), secret) || strings.Contains(string(body), "8/9") {
		t.Fatalf("text or foreign identity leaked: %s", body)
	}
}

func TestPriorityComparisonMasksForeignChoice(t *testing.T) {
	receipt := factory.PriorityReceipt{
		Snapshot: strings.Repeat("b", 64), Outcome: "selected", Selected: "8/9", CreatedUnix: 100,
		Candidates: []factory.PriorityCandidate{
			{ID: "7/42", Control: factory.IssueControl{Repository: 7, Issue: 42, Readiness: "queued", FirstSeenUnix: 10}},
			{ID: "8/9", Control: factory.IssueControl{Repository: 8, Issue: 9, Readiness: "queued", FirstSeenUnix: 300}},
		},
		Calls: []factory.PriorityCall{{Answer: factory.PriorityAnswer{Selected: "8/9", Probabilities: map[string]float64{"7/42": 0.4, "8/9": 0.6}}}},
	}
	view := priorityComparisonDTO(receipt, 7, 200)
	if view.SelectedVisible || view.Selected != "" {
		t.Fatalf("foreign choice exposed: %+v", view)
	}
	if len(view.Probabilities) != 1 || view.ProbabilitiesFull {
		t.Fatalf("distribution: %+v", view)
	}
	if view.Candidates[0].AgeSeconds != 190 {
		t.Fatalf("age: %+v", view.Candidates[0])
	}
	future := priorityCandidateDTO(factory.PriorityCandidate{Control: factory.IssueControl{FirstSeenUnix: 500}}, 200)
	if future.AgeSeconds != 0 {
		t.Fatalf("future age: %+v", future)
	}
}

func TestPriorityComparisonWithoutCalls(t *testing.T) {
	view := priorityComparisonDTO(factory.PriorityReceipt{Snapshot: strings.Repeat("c", 64), Outcome: "unavailable", CreatedUnix: 50}, 7, 200)
	if len(view.Candidates) != 0 || len(view.Calls) != 0 || view.Probabilities != nil || !view.SelectedVisible {
		t.Fatalf("empty comparison: %+v", view)
	}
	body, err := json.Marshal(view)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"candidates", "calls"} {
		if items, ok := decoded[key].([]any); !ok || len(items) != 0 {
			t.Fatalf("%s: %v", key, decoded[key])
		}
	}
}

func TestParsePlanComparisonQuery(t *testing.T) {
	cursor, limit, ok := parsePlanComparisonQuery("")
	if !ok || limit != store.DefaultPriorityComparisons || cursor.BeforeSnapshot != "" {
		t.Fatalf("defaults: %+v %d %v", cursor, limit, ok)
	}
	cursor, limit, ok = parsePlanComparisonQuery("limit=20&before_unix=100&before_snapshot=" + strings.Repeat("d", 64))
	if !ok || limit != 20 || cursor.BeforeUnix != 100 || cursor.BeforeSnapshot != strings.Repeat("d", 64) {
		t.Fatalf("cursor: %+v %d %v", cursor, limit, ok)
	}
	for _, raw := range []string{
		"limit=0", "limit=21", "limit=many",
		"before_unix=100", "before_snapshot=" + strings.Repeat("d", 64),
		"before_unix=0&before_snapshot=" + strings.Repeat("d", 64),
		"before_unix=100&before_snapshot=short",
		"limit=5&unknown=1", strings.Repeat("x", 2049),
	} {
		if _, _, ok := parsePlanComparisonQuery(raw); ok {
			t.Fatalf("accepted %q", raw)
		}
	}
}
