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

func TestClarifyIdeaCarriesAssumptionBasis(t *testing.T) {
	idea := clarifyIdea("d111111111111111111111111")
	if !idea.Assumption || idea.Decision != "pending" || idea.Source != "assumption" || idea.Provenance != "autonomous" || !idea.Material || idea.Actor != 0 {
		t.Fatalf("clarify basis=%+v", idea)
	}
	r := factory.IntentRevision{ID: "d222222222222222222222222", Intent: "d333333333333333333333333", Owner: 7, Objective: "Build a tracker", State: "draft", MaxIssues: 10, PlanningCalls: 3, CreatedUnix: 1, Ideas: []factory.ProductIdea{idea}}
	if err := r.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestAppendClarifyOnceReusesStableIdentity(t *testing.T) {
	base := factory.IntentRevision{Ideas: []factory.ProductIdea{{ID: "d111111111111111111111111", Description: "Track leave requests", Decision: "accept"}}}
	first := appendClarifyOnce(base, "d222222222222222222222222")
	clarified := 0
	for _, idea := range first.Ideas {
		if idea.Description == "Describe the first user action and the result that should count as success" {
			clarified++
		}
	}
	if clarified != 1 {
		t.Fatalf("round one clarifications=%d", clarified)
	}
	second := appendClarifyOnce(first, "d333333333333333333333333")
	clarified = 0
	for _, idea := range second.Ideas {
		if idea.Description == "Describe the first user action and the result that should count as success" {
			clarified++
			if idea.ID != "d222222222222222222222222" || idea.Decision != "pending" {
				t.Fatalf("stable clarification lost=%+v", idea)
			}
		}
	}
	if clarified != 1 {
		t.Fatalf("round two clarifications=%d, want exactly one stable idea", clarified)
	}
}

func TestAppendClarifyOncePreservesAnsweredClarification(t *testing.T) {
	answered := clarifyIdea("d222222222222222222222222")
	answered.Decision = "edit"
	answered.Correction = "The first action is submitting the request form"
	answered.Provenance = "human"
	answered.Actor = 7
	r := factory.IntentRevision{Ideas: []factory.ProductIdea{
		{ID: "d111111111111111111111111", Description: "Track leave requests", Decision: "accept"},
		answered,
	}}
	next := appendClarifyOnce(r, "d333333333333333333333333")
	if len(next.Ideas) != 2 {
		t.Fatalf("answered clarification duplicated: %+v", next.Ideas)
	}
	if next.Ideas[1].Correction != "The first action is submitting the request form" || next.Ideas[1].Provenance != "human" {
		t.Fatalf("answer not preserved=%+v", next.Ideas[1])
	}
}

func TestAppendClarifyOnceIgnoresOrdinaryIdeas(t *testing.T) {
	r := factory.IntentRevision{Ideas: []factory.ProductIdea{
		{ID: "d111111111111111111111111", Description: "Track leave requests", Consequence: "Managers review requests", Decision: "accept", Source: "requested", Provenance: "human", Actor: 7},
	}}
	next := appendClarifyOnce(r, "d222222222222222222222222")
	if len(next.Ideas) != 2 || next.Ideas[0].ID != "d111111111111111111111111" {
		t.Fatalf("ordinary idea touched=%+v", next.Ideas)
	}
	if next.Ideas[1].ID != "d222222222222222222222222" || next.Ideas[1].Decision != "pending" {
		t.Fatalf("clarification missing=%+v", next.Ideas)
	}
}

func TestExploreResponseKeepsRevisionFlatWithExploration(t *testing.T) {
	proposal := factory.IntentRevision{ID: "d111111111111111111111111", Intent: "d222222222222222222222222", State: "draft"}
	content := factory.DescribeAlternatives("Time-off requests my manager can approve")
	challenges := []factory.AssumptionChallenge{{Idea: "d333333333333333333333333", Assumption: "Everyone can see each request", Consequence: "Dates are exposed", Recommendation: "Confirm it"}}
	raw, err := json.Marshal(attachExploration(proposal, factory.ProposalContent{Producer: content.Producer, Supported: content.Supported, Unknowns: content.Unknowns}, challenges, []string{"d333333333333333333333333"}, nil, nil))
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
	if list, ok := exploration["settled"].([]any); !ok || len(list) != 1 || list[0] != "d333333333333333333333333" {
		t.Fatalf("settled=%v", exploration["settled"])
	}
}
