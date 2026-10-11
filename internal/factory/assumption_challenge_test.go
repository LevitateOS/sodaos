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
