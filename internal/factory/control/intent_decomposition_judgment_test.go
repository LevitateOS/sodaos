package control

import (
	"context"
	"errors"
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
)

type decompositionChooser struct {
	questions []string
	disagree  bool
}

func (f *decompositionChooser) Select(_ context.Context, _ any, question string, choices map[string]any) (factory.PriorityAnswer, error) {
	f.questions = append(f.questions, question)
	selected := "keep"
	if f.disagree && len(f.questions) == 2 {
		selected = "review"
	}
	a := factory.PriorityAnswer{Selected: selected, Model: "synthetic-model", Confidence: 1, Probabilities: map[string]float64{}}
	for id := range choices {
		a.Probabilities[id] = 0
	}
	a.Probabilities[selected] = 1
	return a, nil
}

type recomposeChooser struct {
	triplet      string
	tripletConf  float64
	selected     string
	optionsConf  float64
	calls        int
	optionsCalls int
}

func (f *recomposeChooser) Select(_ context.Context, _ any, _ string, choices map[string]any) (factory.PriorityAnswer, error) {
	f.calls++
	_, triplet := choices["revise"]
	selected, confidence := f.triplet, f.tripletConf
	if !triplet {
		f.optionsCalls++
		selected, confidence = f.selected, f.optionsConf
		if _, ok := choices[selected]; !ok {
			selected = "keep"
		}
	}
	share := (1 + confidence) / 2
	a := factory.PriorityAnswer{Selected: selected, Model: "synthetic-model", Confidence: confidence, Probabilities: map[string]float64{}}
	for id := range choices {
		a.Probabilities[id] = (1 - share) / float64(len(choices)-1)
	}
	a.Probabilities[selected] = share
	return a, nil
}

func multiOutcomeAccepted(t *testing.T) (*Coordinator, factory.IntentRevision) {
	t.Helper()
	c, _, r := acceptedGenerationFixture(t)
	r.Tasks[0].Outcomes = []string{"Capture requests", "Validate requests"}
	return c, r
}

func TestRecomposeDecompositionKeepsConfidentGraphs(t *testing.T) {
	c, _, r := acceptedGenerationFixture(t)
	c.Store.PriorityCallBudget = 32
	chooser := &recomposeChooser{triplet: "keep", tripletConf: 1}
	c.Planning = &PlanningJudgments{Journal: c.Store, Funding: c.Store, Chooser: chooser}
	decomposition, err := c.RecomposeDecomposition(context.Background(), r, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := decomposition.Validate(); err != nil {
		t.Fatal(err)
	}
	if decomposition.Selected != "keep" || len(decomposition.Tasks) != len(r.Tasks) || len(decomposition.Advice) != 3 || chooser.optionsCalls != 0 {
		t.Fatalf("decomposition=%+v optionsCalls=%d", decomposition, chooser.optionsCalls)
	}
}

func TestRecomposeDecompositionSelectsScopedAlternative(t *testing.T) {
	c, r := multiOutcomeAccepted(t)
	c.Store.PriorityCallBudget = 32
	candidates := factory.DecompositionCandidates(r)
	if len(candidates) < 2 {
		t.Fatal("no split candidate to select")
	}
	chooser := &recomposeChooser{triplet: "revise", tripletConf: 1, selected: candidates[1].Key, optionsConf: 0.8}
	c.Planning = &PlanningJudgments{Journal: c.Store, Funding: c.Store, Chooser: chooser}
	decomposition, err := c.RecomposeDecomposition(context.Background(), r, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := decomposition.Validate(); err != nil {
		t.Fatal(err)
	}
	if decomposition.Selected != candidates[1].Key || len(decomposition.Advice) != 4 || chooser.optionsCalls != 1 {
		t.Fatalf("decomposition=%+v optionsCalls=%d", decomposition, chooser.optionsCalls)
	}
	fresh, freshRevision := multiOutcomeAccepted(t)
	fresh.Store.PriorityCallBudget = 32
	fresh.Planning = &PlanningJudgments{Journal: fresh.Store, Funding: fresh.Store, Chooser: &recomposeChooser{triplet: "revise", tripletConf: 1, selected: candidates[1].Key, optionsConf: 0.2}}
	fallback, err := fresh.RecomposeDecomposition(context.Background(), freshRevision, nil)
	if err != nil {
		t.Fatal(err)
	}
	if fallback.Selected != "keep" {
		t.Fatalf("uncertain selection=%+v", fallback)
	}
}

func TestRecomposeDecompositionChallengesProductAmbiguity(t *testing.T) {
	c, r := multiOutcomeAccepted(t)
	c.Store.PriorityCallBudget = 32
	c.Planning = &PlanningJudgments{Journal: c.Store, Funding: c.Store, Chooser: &recomposeChooser{triplet: "review", tripletConf: 1}}
	if _, err := c.RecomposeDecomposition(context.Background(), r, nil); !errors.Is(err, ErrIntentDecompositionReview) {
		t.Fatalf("review error=%v", err)
	}
}

func TestIntentDecompositionFreshTripletCachesAndDisagreementRefuses(t *testing.T) {
	for _, disagree := range []bool{false, true} {
		t.Run(map[bool]string{false: "agree", true: "disagree"}[disagree], func(t *testing.T) {
			c, _, r := acceptedGenerationFixture(t)
			c.Store.PriorityCallBudget = 10
			chooser := &decompositionChooser{disagree: disagree}
			c.Planning = &PlanningJudgments{Chooser: chooser}
			for i := 0; i < 2; i++ {
				err := c.AdviseIntentDecomposition(context.Background(), r, nil)
				if disagree && !errors.Is(err, ErrIntentDecompositionReview) || !disagree && err != nil {
					t.Fatalf("advice error=%v", err)
				}
			}
			if len(chooser.questions) != 3 || chooser.questions[0] == chooser.questions[1] || chooser.questions[1] == chooser.questions[2] {
				t.Fatalf("fresh cached phrasings=%+v", chooser.questions)
			}
		})
	}
}
