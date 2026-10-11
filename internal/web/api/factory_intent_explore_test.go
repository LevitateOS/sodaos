package api

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
)

func TestIntentDesignRespectsExplicitApproachAndChangesGraph(t *testing.T) {
	r := factory.IntentRevision{Objective: "Build accessible request entry", Ideas: []factory.ProductIdea{{ID: "d111111111111111111111111", Description: "Accessible request entry", Decision: "accept"}}}
	ids := make([]string, 104)
	for i := range ids {
		ids[i] = fmt.Sprintf("d%024x", 1000+i)
	}
	proposal := factory.ProposeIntent(r, ids)
	complete := applyIntentDesign(r, ids, proposal, factory.PriorityAnswer{Selected: "complete_first", Confidence: 1})
	if complete.Approach != "complete_first" || len(complete.Tasks) != 1 {
		t.Fatal("design advice did not produce real combined work")
	}
	r.Approach = "incremental"
	candidates := intentDesignCandidates(r)
	if len(candidates) != 2 || candidates[0].ID != "incremental" || candidates[1].ID != "clarify" {
		t.Fatal("explicit human choice was overridden or scope review removed")
	}
	explicit := applyIntentDesign(r, ids, proposal, factory.PriorityAnswer{Selected: "complete_first", Confidence: 1})
	if explicit.Approach != "incremental" || len(explicit.Tasks) != 2 {
		t.Fatal("human choice lost")
	}
}

func TestExploreResponseKeepsRevisionFlatWithExploration(t *testing.T) {
	proposal := factory.IntentRevision{ID: "d111111111111111111111111", Intent: "d222222222222222222222222", State: "draft"}
	content := factory.DescribeAlternatives("Time-off requests my manager can approve")
	challenges := []factory.AssumptionChallenge{{Idea: "d333333333333333333333333", Assumption: "Everyone can see each request", Consequence: "Dates are exposed", Recommendation: "Confirm it"}}
	raw, err := json.Marshal(attachExploration(proposal, factory.ProposalContent{Producer: content.Producer, Supported: content.Supported, Unknowns: content.Unknowns}, challenges, nil, nil))
	if err != nil {
		t.Fatal(err)
	}
	var merged map[string]any
	if err := json.Unmarshal(raw, &merged); err != nil {
		t.Fatal(err)
	}
	if merged["id"] != proposal.ID || merged["state"] != "draft" {
		t.Fatalf("revision fields not flat: %v", merged)
	}
	exploration, ok := merged["exploration"].(map[string]any)
	if !ok || exploration["recommendation"] != nil {
		t.Fatalf("exploration missing: %v", merged)
	}
	if _, ok := exploration["content"].(map[string]any); !ok {
		t.Fatal("content missing from exploration")
	}
	if list, ok := exploration["challenges"].([]any); !ok || len(list) != 1 {
		t.Fatalf("challenges=%v", exploration["challenges"])
	}
}
