package factory

import "testing"

func TestProposalUsesReviewedCorrectionsAndRealDependencies(t *testing.T) {
	r := validIntent()
	r.State = "draft"
	r.Ideas[0].Decision = "edit"
	r.Ideas[0].Correction = "Build accessible request entry"
	ids := make([]string, 104)
	for i := range ids {
		ids[i] = "d111111111111111111111111"
	}
	ids[2] = "d222222222222222222222222"
	ids[3] = "d333333333333333333333333"
	proposal := ProposeIntent(r, ids)
	if proposal.Tasks[0].Objective != r.Ideas[0].Correction || proposal.Tasks[1].DependsOn[0] != proposal.Tasks[0].Key {
		t.Fatal("correction or prerequisite lost")
	}
	r.Ideas[0].Decision = "decline"
	if len(ProposeIntent(r, ids).Tasks) != 0 {
		t.Fatal("declined product generated work")
	}
}
