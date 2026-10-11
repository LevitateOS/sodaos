package factory

import "testing"

func TestAssumptionChallengeRecommendsSourcedResolution(t *testing.T) {
	idea := ProductIdea{ID: "d333333333333333333333333", Description: "Everyone can see each request", Consequence: "Individual dates are exposed", Assumption: true, Decision: "pending"}
	challenge := AssumptionChallenge{Idea: idea.ID, Assumption: idea.Description, Consequence: idea.Consequence, Recommendation: RecommendAssumptionResolution(idea)}
	if err := challenge.Validate(); err != nil {
		t.Fatal(err)
	}
	if challenge.Recommendation == "" {
		t.Fatal("empty recommendation")
	}
	challenge.Idea = "bad"
	if challenge.Validate() == nil {
		t.Fatal("unbound challenge accepted")
	}
}

func TestSettleAssumptionsAcceptsOnlyPendingMatches(t *testing.T) {
	r := validIntent()
	r.State = "draft"
	r.Repository = 0
	r.Tasks = nil
	r.Ideas = []ProductIdea{
		{ID: "d333333333333333333333333", Description: "List newest first", Consequence: "Easy to change sort", Decision: "pending", Source: "autonomous", Provenance: "autonomous"},
		{ID: "d444444444444444444444444", Description: "Ask full address", Consequence: "Collect data", Decision: "edit", Correction: "Ask only work email", Source: "requested", Provenance: "human", Actor: 1},
		{ID: "d555555555555555555555555", Description: "Everyone can see each request", Consequence: "Dates are exposed", Assumption: true, Decision: "pending", Source: "assumption", Provenance: "autonomous", Material: true},
	}
	review := AssumptionReview{
		Challenges: []AssumptionChallenge{{Idea: "d555555555555555555555555", Assumption: "Everyone can see each request", Consequence: "Dates are exposed", Recommendation: "Confirm it"}},
		Settled:    []AssumptionSettlement{{Idea: "d333333333333333333333333", Source: "autonomous", Provenance: "autonomous"}},
	}
	if err := review.Validate(); err != nil {
		t.Fatal(err)
	}
	settled := SettleAssumptions(r, review)
	if err := settled.Validate(); err != nil {
		t.Fatal(err)
	}
	if settled.Ideas[0].Decision != "accept" || settled.Ideas[1].Decision != "edit" || settled.Ideas[1].Correction != "Ask only work email" || settled.Ideas[2].Decision != "pending" {
		t.Fatalf("ideas=%+v", settled.Ideas)
	}
	if r.Ideas[0].Decision != "pending" {
		t.Fatal("settlement mutated its input")
	}
	conflict := review
	conflict.Settled = append(conflict.Settled, AssumptionSettlement{Idea: "d555555555555555555555555", Source: "autonomous", Provenance: "autonomous"})
	if conflict.Validate() == nil {
		t.Fatal("challenge/settlement conflict accepted")
	}
	forged := AssumptionReview{Settled: []AssumptionSettlement{{Idea: "d333333333333333333333333", Source: "requested", Provenance: "human"}}}
	if forged.Validate() == nil {
		t.Fatal("human settlement accepted")
	}
}
