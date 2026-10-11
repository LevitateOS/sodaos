package factory

import (
	"fmt"
	"strings"
	"testing"
)

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

func TestProposalPreservesLongObjectiveAndBoundsReviewLabels(t *testing.T) {
	for _, length := range []int{3990, 4000, 16000} {
		r := validIntent()
		r.State = "draft"
		r.Objective = strings.Repeat("x", length)
		r.Ideas = nil
		ids := make([]string, 104)
		for i := range ids {
			ids[i] = fmt.Sprintf("d%024x", 1000+i)
		}
		proposal := ProposeIntent(r, ids)
		if err := proposal.Validate(); err != nil {
			t.Fatalf("objective length %d: %v", length, err)
		}
		if proposal.Objective != r.Objective {
			t.Fatal("review shortened original objective")
		}
		for _, task := range proposal.Tasks {
			if len(task.Objective) > 4000 {
				t.Fatal("review task exceeds bounded objective")
			}
		}
	}
}
