package control

import (
	"context"
	"errors"
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
)

type assumptionChooser struct {
	selected   string
	confidence float64
	calls      int
}

func (f *assumptionChooser) Select(_ context.Context, _ any, _ string, choices map[string]any) (factory.PriorityAnswer, error) {
	f.calls++
	selected := f.selected
	if _, ok := choices[selected]; !ok {
		selected = "apply_default"
	}
	share := (1 + f.confidence) / 2
	a := factory.PriorityAnswer{Selected: selected, Model: "synthetic-model", Confidence: f.confidence, Probabilities: map[string]float64{}}
	for id := range choices {
		a.Probabilities[id] = (1 - share) / float64(len(choices)-1)
	}
	a.Probabilities[selected] = share
	return a, nil
}

func draftAssumptionFixture(t *testing.T, ideas []factory.ProductIdea, calls int) (*Coordinator, factory.IntentRevision) {
	t.Helper()
	c := coordinatorFixture(t, nil, nil)
	r := factory.IntentRevision{ID: "d111111111111111111111111", Intent: "d222222222222222222222222", Owner: 7, Objective: "Time-off requests my manager can approve", State: "draft", MaxIssues: 10, PlanningCalls: calls, CreatedUnix: 1, Ideas: ideas}
	if err := c.Store.SaveIntentRevision(context.Background(), r); err != nil {
		t.Fatal(err)
	}
	if err := c.Store.SavePlanningAllowance(context.Background(), factory.PlanningAllowance{ID: "d999999999999999999999999", Intent: r.Intent, Owner: r.Owner, Amount: calls, Unit: factory.PlanningAllowanceUnit, Ceiling: calls, CreatedUnix: 1}); err != nil {
		t.Fatal(err)
	}
	c.Store.PriorityCallBudget = 10
	return c, r
}

func TestJudgeAssumptionsChallengesOnlyPendingAssumptions(t *testing.T) {
	ideas := []factory.ProductIdea{
		{ID: "d333333333333333333333333", Description: "Everyone can see each request", Consequence: "Individual dates are exposed", Assumption: true, Decision: "pending"},
		{ID: "d444444444444444444444444", Description: "The direct manager approves", Consequence: "Clear routing", Assumption: true, Decision: "accept"},
		{ID: "d555555555555555555555555", Description: "Email the whole team", Consequence: "Noise", Assumption: true, Decision: "decline"},
		{ID: "d666666666666666666666666", Description: "Track leave requests", Consequence: "Visibility", Decision: "pending"},
	}
	c, r := draftAssumptionFixture(t, ideas, 5)
	chooser := &assumptionChooser{selected: "challenge", confidence: 1}
	c.Planning = &PlanningJudgments{Journal: c.Store, Funding: c.Store, Chooser: chooser}
	content, err := factory.ComposeProposalContent(r)
	if err != nil {
		t.Fatal(err)
	}
	challenges, err := c.JudgeAssumptions(context.Background(), r, content)
	if err != nil {
		t.Fatal(err)
	}
	if len(challenges) != 1 || challenges[0].Idea != "d333333333333333333333333" || challenges[0].Assumption != ideas[0].Description || challenges[0].Consequence != ideas[0].Consequence || challenges[0].Recommendation == "" {
		t.Fatalf("challenges=%+v", challenges)
	}
	if _, err := c.JudgeAssumptions(context.Background(), r, content); err != nil {
		t.Fatal(err)
	}
	if chooser.calls != 1 {
		t.Fatalf("judgment repeated, calls=%d", chooser.calls)
	}
	for i, idea := range r.Ideas {
		if idea.Decision != ideas[i].Decision {
			t.Fatal("challenge flipped a recorded decision")
		}
	}
}

func TestJudgeAssumptionsUncertainOrDefaultVerdictsProceed(t *testing.T) {
	ideas := []factory.ProductIdea{{ID: "d333333333333333333333333", Description: "Everyone can see each request", Consequence: "Individual dates are exposed", Assumption: true, Decision: "pending"}}
	for _, verdict := range []assumptionChooser{{selected: "challenge", confidence: 0.3}, {selected: "apply_default", confidence: 0.9}} {
		c, r := draftAssumptionFixture(t, ideas, 5)
		chooser := verdict
		c.Planning = &PlanningJudgments{Journal: c.Store, Funding: c.Store, Chooser: &chooser}
		content, err := factory.ComposeProposalContent(r)
		if err != nil {
			t.Fatal(err)
		}
		challenges, err := c.JudgeAssumptions(context.Background(), r, content)
		if err != nil || len(challenges) != 0 {
			t.Fatalf("verdict %+v: challenges=%+v err=%v", verdict, challenges, err)
		}
	}
}

func TestJudgeAssumptionsExhaustionAndMismatchFail(t *testing.T) {
	ideas := []factory.ProductIdea{
		{ID: "d333333333333333333333333", Description: "Everyone can see each request", Consequence: "Individual dates are exposed", Assumption: true, Decision: "pending"},
		{ID: "d444444444444444444444444", Description: "Weekends never count", Consequence: "Wrong balances", Assumption: true, Decision: "pending"},
	}
	c, r := draftAssumptionFixture(t, ideas, 1)
	c.Planning = &PlanningJudgments{Journal: c.Store, Funding: c.Store, Chooser: &assumptionChooser{selected: "challenge", confidence: 1}}
	content, err := factory.ComposeProposalContent(r)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.JudgeAssumptions(context.Background(), r, content); !errors.Is(err, ErrPriorityUnavailable) {
		t.Fatalf("exhaustion error=%v", err)
	}
	stale := r
	stale.ID = "d999999999999999999999999"
	if _, err := c.JudgeAssumptions(context.Background(), stale, content); err == nil {
		t.Fatal("mismatched content accepted")
	}
}
