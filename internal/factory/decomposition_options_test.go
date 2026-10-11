package factory

import (
	"strings"
	"testing"
)

func multiOutcomeRevision() IntentRevision {
	r := validIntent()
	r.Tasks = []IntentTask{
		{Key: "d444444444444444444444444", Objective: "Request entry", Outcomes: []string{"Capture requests", "Validate requests"}, IdeaIDs: []string{"d333333333333333333333333"}},
		{Key: "d555555555555555555555555", Objective: "Request list", Outcomes: []string{"Show requests"}, IdeaIDs: []string{"d333333333333333333333333"}, DependsOn: []string{"d444444444444444444444444"}},
	}
	return r
}

func TestDecompositionCandidatesPreserveScopeAndBounds(t *testing.T) {
	r := multiOutcomeRevision()
	if err := r.Validate(); err != nil {
		t.Fatal(err)
	}
	candidates := DecompositionCandidates(r)
	if len(candidates) < 2 || candidates[0].Key != "keep" {
		t.Fatalf("candidates=%+v", candidates)
	}
	seenSplit := false
	for _, candidate := range candidates {
		if !sameOutcomes(r.Tasks, candidate.Tasks) {
			t.Fatalf("candidate %s changed outcomes", candidate.Key)
		}
		revision := r
		revision.Tasks = candidate.Tasks
		if err := revision.Validate(); err != nil {
			t.Fatalf("candidate %s: %v", candidate.Key, err)
		}
		decomposition := ScopedDecomposition{Intent: r.Intent, Revision: r.ID, Selected: candidate.Key, Tasks: candidate.Tasks, Advice: []string{strings.Repeat("a", 64)}}
		if err := decomposition.Validate(); err != nil {
			t.Fatalf("candidate %s: %v", candidate.Key, err)
		}
		seenSplit = seenSplit || strings.HasPrefix(candidate.Key, "split:")
	}
	if !seenSplit {
		t.Fatal("multi-outcome task offered no split")
	}
}

func TestDecompositionCandidatesCombineDuplicatedWork(t *testing.T) {
	r := validIntent()
	r.Tasks = []IntentTask{
		{Key: "d444444444444444444444444", Objective: "Capture requests", Outcomes: []string{"Capture requests"}, IdeaIDs: []string{"d333333333333333333333333"}},
		{Key: "d555555555555555555555555", Objective: "Capture requests again", Outcomes: []string{"Capture requests"}, IdeaIDs: []string{"d333333333333333333333333"}},
	}
	if err := r.Validate(); err != nil {
		t.Fatal(err)
	}
	candidates := DecompositionCandidates(r)
	if len(candidates) != 2 || candidates[0].Key != "keep" || !strings.HasPrefix(candidates[1].Key, "combine:") || len(candidates[1].Tasks) != 1 {
		t.Fatalf("candidates=%+v", candidates)
	}
}

func TestDecompositionCandidatesKeepOnlyWithoutVariants(t *testing.T) {
	r := validIntent()
	r.Tasks[1].DependsOn = []string{"d444444444444444444444444"}
	if err := r.Validate(); err != nil {
		t.Fatal(err)
	}
	candidates := DecompositionCandidates(r)
	if len(candidates) != 1 || candidates[0].Key != "keep" {
		t.Fatalf("candidates=%+v", candidates)
	}
}

func TestScopedDecompositionRejectsBrokenGraphs(t *testing.T) {
	base := ScopedDecomposition{Intent: "d222222222222222222222222", Revision: "d111111111111111111111111", Selected: "keep", Advice: []string{strings.Repeat("a", 64)}}
	cyclic := base
	cyclic.Tasks = []IntentTask{
		{Key: "d444444444444444444444444", Objective: "First", Outcomes: []string{"Done"}, IdeaIDs: []string{"d333333333333333333333333"}, DependsOn: []string{"d555555555555555555555555"}},
		{Key: "d555555555555555555555555", Objective: "Second", Outcomes: []string{"Done"}, IdeaIDs: []string{"d333333333333333333333333"}, DependsOn: []string{"d444444444444444444444444"}},
	}
	if cyclic.Validate() == nil {
		t.Fatal("cyclic graph accepted")
	}
	dangling := base
	dangling.Tasks = []IntentTask{{Key: "d444444444444444444444444", Objective: "First", Outcomes: []string{"Done"}, IdeaIDs: []string{"d333333333333333333333333"}, DependsOn: []string{"d999999999999999999999999"}}}
	if dangling.Validate() == nil {
		t.Fatal("unknown dependency accepted")
	}
	unprovenanced := base
	unprovenanced.Tasks = []IntentTask{{Key: "d444444444444444444444444", Objective: "First", Outcomes: []string{"Done"}, IdeaIDs: []string{"d333333333333333333333333"}}}
	unprovenanced.Advice = []string{"not-a-digest"}
	if unprovenanced.Validate() == nil {
		t.Fatal("unprovenanced advice accepted")
	}
}
