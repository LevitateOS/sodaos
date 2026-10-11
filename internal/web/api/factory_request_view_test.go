package api

import (
	"encoding/json"
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
)

func TestRequestWorkState(t *testing.T) {
	delivered := requestWorkInput{HasAssignment: true, AssignmentOutcome: factory.Succeeded, HasMerge: true, MergeStage: factory.MergeMerged, MergeClosureChecked: true}
	if got := requestWorkState(delivered); got != requestWorkDelivered {
		t.Fatalf("delivered: %s", got)
	}
	unclosed := delivered
	unclosed.MergeClosureChecked = false
	if got := requestWorkState(unclosed); got != requestWorkWorking {
		t.Fatalf("merge without closure: %s", got)
	}
	for _, stage := range []string{factory.MergeFailed, factory.MergeWithdrawn, factory.MergeFenced} {
		in := requestWorkInput{HasMerge: true, MergeStage: stage}
		if got := requestWorkState(in); got != requestWorkAttention {
			t.Fatalf("merge %s: %s", stage, got)
		}
	}
	for _, outcome := range []factory.Outcome{factory.Failed, factory.Cancelled, factory.NeedsHuman} {
		in := requestWorkInput{HasAssignment: true, AssignmentOutcome: outcome}
		if got := requestWorkState(in); got != requestWorkAttention {
			t.Fatalf("outcome %s: %s", outcome, got)
		}
	}
	running := requestWorkInput{HasAssignment: true}
	if got := requestWorkState(running); got != requestWorkWorking {
		t.Fatalf("running: %s", got)
	}
	pipelined := requestWorkInput{HasAssignment: true, AssignmentOutcome: factory.Succeeded}
	if got := requestWorkState(pipelined); got != requestWorkWorking {
		t.Fatalf("succeeded without merge: %s", got)
	}
	if got := requestWorkState(requestWorkInput{}); got != requestWorkWaiting {
		t.Fatalf("idle: %s", got)
	}
}

func TestRequestDirectionDTO(t *testing.T) {
	view := requestDirectionDTO(factory.IntentRevision{
		Objective: "Build a leave tracker", Approach: "incremental",
		EffectiveOutcomes: []string{"d1"},
		Ideas: []factory.ProductIdea{
			{ID: "d1", Decision: "accept", Source: "requested"},
			{ID: "d2", Decision: "pending"},
			{ID: "d3", Decision: "pending", Source: "assumption", Assumption: true, Material: true},
		},
		Tasks: []factory.IntentTask{{Outcomes: []string{"a"}}, {Outcomes: []string{"b", "c"}}},
	})
	if view.Objective != "Build a leave tracker" || view.Approach != "incremental" {
		t.Fatalf("direction: %+v", view)
	}
	if view.IdeasTotal != 3 || view.TasksTotal != 2 {
		t.Fatalf("totals: %+v", view)
	}
	if view.IdeasByDecision["accept"] != 1 || view.IdeasByDecision["pending"] != 2 {
		t.Fatalf("decisions: %+v", view.IdeasByDecision)
	}
	if view.IdeasBySource["requested"] != 1 || view.IdeasBySource["assumption"] != 1 || view.IdeasBySource["unclassified"] != 1 {
		t.Fatalf("sources: %+v", view.IdeasBySource)
	}
	if len(view.EffectiveOutcomes) != 1 || view.EffectiveOutcomes[0] != "d1" {
		t.Fatalf("effective: %+v", view.EffectiveOutcomes)
	}
	if view.Unresolved != 1 {
		t.Fatalf("unresolved: %+v", view)
	}
	legacy := requestDirectionDTO(factory.IntentRevision{})
	if legacy.EffectiveOutcomes == nil || legacy.Unresolved != 0 {
		t.Fatalf("legacy direction: %+v", legacy)
	}
}

func TestRequestUnresolvedAssumptions(t *testing.T) {
	ideas := []factory.ProductIdea{
		{ID: "d1", Decision: "pending", Assumption: true, Material: true},
		{ID: "d2", Decision: "pending", Assumption: true},
		{ID: "d3", Decision: "accept", Assumption: true, Material: true},
		{ID: "d4", Decision: "pending"},
	}
	unresolved := requestUnresolvedAssumptions(ideas)
	if len(unresolved) != 1 || unresolved[0] != "d1" {
		t.Fatalf("unresolved: %v", unresolved)
	}
}

func TestRequestMappingDTO(t *testing.T) {
	binding := factory.IntentTaskBinding{TaskKey: "task", IssueID: 103, IssueIndex: 3, AcceptanceID: "d1", CreatedUnix: 7, ContentVersion: 2}
	tasks := map[string]factory.IntentTask{"task": {Key: "task", IdeaIDs: []string{"d1"}}}
	view := requestMappingDTO(binding, tasks, map[string]bool{"d1": true})
	if view.TaskKey != "task" || view.IssueID != "103" || view.IssueIndex != "3" || view.AcceptanceID != "d1" || view.CreatedUnix != 7 || view.ContentVersion != 2 || !view.Effective {
		t.Fatalf("mapping: %+v", view)
	}
	scoped := requestMappingDTO(binding, tasks, map[string]bool{"d2": true})
	if scoped.Effective {
		t.Fatalf("out-of-scope mapping effective: %+v", scoped)
	}
	unscoped := requestMappingDTO(binding, tasks, map[string]bool{})
	if !unscoped.Effective {
		t.Fatalf("legacy mapping ineffective: %+v", unscoped)
	}
}

func TestRequestOverviewOmitsPercentages(t *testing.T) {
	view := factoryRequestOverviewView{
		WorkItems: requestCountsView{ByState: map[string]int{requestWorkDelivered: 2, requestWorkWorking: 1}, Total: 3},
		Mappings:  []requestMappingView{}, Attention: []requestAttentionView{},
	}
	body, err := json.Marshal(view)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatal(err)
	}
	work, ok := decoded["work_items"].(map[string]any)
	if !ok {
		t.Fatalf("work items: %v", decoded)
	}
	for key := range work {
		switch key {
		case "by_state", "outcomes_total", "outcomes_evidenced", "total":
		default:
			t.Fatalf("unexpected work fact %s: %v", key, work)
		}
	}
	if work["outcomes_evidenced"] != float64(0) {
		t.Fatalf("invented outcome evidence: %v", work)
	}
}
