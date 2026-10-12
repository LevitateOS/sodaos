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

func TestEffectiveDecompositionScopeProjectsEvidence(t *testing.T) {
	r := factory.IntentRevision{ID: "d111111111111111111111111", Intent: "d222222222222222222222222", Owner: 7, Objective: "Build a tracker", State: "accepted", MaxIssues: 10, CreatedUnix: 1, Ideas: []factory.ProductIdea{
		{ID: "d333333333333333333333333", Description: "Track requests", Consequence: "Clear ownership", Decision: "accept", Source: "requested", Provenance: "human", Actor: 7},
		{ID: "d666666666666666666666666", Description: "Keep the first version small", Consequence: "No extra integrations", Assumption: true, Decision: "accept", Source: "assumption", Provenance: "autonomous", Material: true},
		{ID: "d777777777777777777777777", Description: "Self-register accounts", Consequence: "Unverified access", Decision: "decline", Source: "autonomous", Provenance: "autonomous"},
		{ID: "d888888888888888888888888", Description: "Defer notifications", Consequence: "Later scope", Decision: "pending"},
		{ID: "d121212121212121212121212", Description: "Limit to current repository", Consequence: "No new enrollments", Assumption: true, Decision: "pending", Source: "assumption", Provenance: "autonomous", Material: true},
	}, Tasks: []factory.IntentTask{{Key: "d444444444444444444444444", Objective: "Request entry", Outcomes: []string{"Capture requests"}, IdeaIDs: []string{"d333333333333333333333333"}}}}
	scoped := effectiveDecompositionScope(r)
	if len(scoped.Tasks) != 1 || scoped.Tasks[0].Key != "d444444444444444444444444" {
		t.Fatalf("tasks not preserved=%+v", scoped.Tasks)
	}
	kept := map[string]bool{}
	for _, idea := range scoped.Ideas {
		kept[idea.ID] = true
	}
	for _, id := range []string{"d333333333333333333333333", "d666666666666666666666666", "d777777777777777777777777"} {
		if !kept[id] {
			t.Fatalf("effective idea dropped=%s kept=%v", id, kept)
		}
	}
	for _, id := range []string{"d888888888888888888888888", "d121212121212121212121212"} {
		if kept[id] {
			t.Fatalf("pending idea kept=%s", id)
		}
	}
	if scoped.Intent != r.Intent || scoped.ID != r.ID || scoped.State != r.State {
		t.Fatalf("revision identity changed=%+v", scoped)
	}
	previous := r
	previous.ID = "daaaaaaaaaaaaaaaaaaaaaaaa"
	previous.Tasks = []factory.IntentTask{{Key: "d444444444444444444444444", Objective: "Request entry", Outcomes: []string{"Capture requests"}, IdeaIDs: []string{"d333333333333333333333333"}}}
	scopedPrevious := effectiveDecompositionScope(previous)
	if len(scopedPrevious.Ideas) != len(scoped.Ideas) {
		t.Fatalf("previous scope diverges=%d vs %d", len(scopedPrevious.Ideas), len(scoped.Ideas))
	}
}

type capturingDecompositionChooser struct {
	states []decompositionEvidence
}

func (f *capturingDecompositionChooser) Select(_ context.Context, state any, _ string, choices map[string]any) (factory.PriorityAnswer, error) {
	if evidence, ok := state.(decompositionEvidence); ok {
		f.states = append(f.states, evidence)
	}
	a := factory.PriorityAnswer{Selected: "keep", Model: "synthetic-model", Confidence: 1, Probabilities: map[string]float64{}}
	for id := range choices {
		a.Probabilities[id] = 0
	}
	a.Probabilities["keep"] = 1
	return a, nil
}

func TestRecomposeDecompositionJudgesEffectiveScope(t *testing.T) {
	c, _, r := acceptedGenerationFixture(t)
	c.Store.PriorityCallBudget = 32
	r.Ideas = append(r.Ideas,
		factory.ProductIdea{ID: "d666666666666666666666666", Description: "Keep the first version small", Consequence: "No extra integrations", Assumption: true, Decision: "accept", Source: "assumption", Provenance: "autonomous", Material: true},
		factory.ProductIdea{ID: "d777777777777777777777777", Description: "Self-register accounts", Consequence: "Unverified access", Decision: "decline", Source: "autonomous", Provenance: "autonomous"},
		factory.ProductIdea{ID: "d121212121212121212121212", Description: "Limit to current repository", Consequence: "No new enrollments", Assumption: true, Decision: "pending", Source: "assumption", Provenance: "autonomous", Material: true},
	)
	chooser := &capturingDecompositionChooser{}
	c.Planning = &PlanningJudgments{Journal: c.Store, Funding: c.Store, Chooser: chooser}
	decomposition, err := c.RecomposeDecomposition(context.Background(), r, nil)
	if err != nil {
		t.Fatal(err)
	}
	if decomposition.Selected != "keep" {
		t.Fatalf("decomposition=%+v", decomposition)
	}
	if len(chooser.states) != 3 {
		t.Fatalf("captured states=%d", len(chooser.states))
	}
	for _, evidence := range chooser.states {
		for _, idea := range evidence.Current.Ideas {
			if idea.Decision == "pending" {
				t.Fatalf("pending idea in triplet evidence=%+v", idea)
			}
		}
		if len(evidence.Current.Tasks) != len(r.Tasks) {
			t.Fatalf("tasks narrowed=%d", len(evidence.Current.Tasks))
		}
		kept := map[string]bool{}
		for _, idea := range evidence.Current.Ideas {
			kept[idea.ID] = true
		}
		for _, id := range []string{"d333333333333333333333333", "d666666666666666666666666", "d777777777777777777777777"} {
			if !kept[id] {
				t.Fatalf("effective idea missing=%s", id)
			}
		}
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
