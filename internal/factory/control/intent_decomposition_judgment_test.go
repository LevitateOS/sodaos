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
