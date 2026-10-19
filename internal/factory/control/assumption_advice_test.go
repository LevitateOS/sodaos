package control

import (
	"context"
	"errors"
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/store"
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
	return c, r
}

func TestJudgeAssumptionsChallengesOnlyPendingAssumptions(t *testing.T) {
	ideas := []factory.ProductIdea{
		{ID: "d333333333333333333333333", Description: "Everyone can see each request", Consequence: "Individual dates are exposed", Assumption: true, Decision: "pending", Source: "assumption", Provenance: "autonomous", Material: true},
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
	review, err := c.JudgeAssumptions(context.Background(), r, content, r.Ideas)
	if err != nil {
		t.Fatal(err)
	}
	if len(review.Challenges) != 1 || review.Challenges[0].Idea != "d333333333333333333333333" || review.Challenges[0].Recommendation == "" {
		t.Fatalf("challenges=%+v", review.Challenges)
	}
	if len(review.Settled) != 0 {
		t.Fatalf("settled=%+v", review.Settled)
	}
	if _, err := c.JudgeAssumptions(context.Background(), r, content, r.Ideas); err != nil {
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

func TestJudgeAssumptionsDefaultVerdictSettlesNonMaterial(t *testing.T) {
	ideas := []factory.ProductIdea{{ID: "d333333333333333333333333", Description: "List newest first", Consequence: "Easy to change sort", Assumption: true, Decision: "pending"}}
	for _, verdict := range []assumptionChooser{{selected: "challenge", confidence: 0.3}, {selected: "apply_default", confidence: 0.9}} {
		c, r := draftAssumptionFixture(t, ideas, 5)
		chooser := verdict
		c.Planning = &PlanningJudgments{Journal: c.Store, Funding: c.Store, Chooser: &chooser}
		content, err := factory.ComposeProposalContent(r)
		if err != nil {
			t.Fatal(err)
		}
		review, err := c.JudgeAssumptions(context.Background(), r, content, r.Ideas)
		if err != nil || len(review.Challenges) != 0 || len(review.Settled) != 1 || review.Settled[0].Provenance != "autonomous" {
			t.Fatalf("verdict %+v: review=%+v err=%v", verdict, review, err)
		}
	}
	material := []factory.ProductIdea{{ID: "d333333333333333333333333", Description: "Everyone can see each request", Consequence: "Individual dates are exposed", Assumption: true, Decision: "pending", Source: "assumption", Provenance: "autonomous", Material: true}}
	c, r := draftAssumptionFixture(t, material, 5)
	c.Planning = &PlanningJudgments{Journal: c.Store, Funding: c.Store, Chooser: &assumptionChooser{selected: "apply_default", confidence: 0.9}}
	content, err := factory.ComposeProposalContent(r)
	if err != nil {
		t.Fatal(err)
	}
	review, err := c.JudgeAssumptions(context.Background(), r, content, r.Ideas)
	if err != nil || len(review.Challenges) != 0 || len(review.Settled) != 0 {
		t.Fatalf("material default: review=%+v err=%v", review, err)
	}
}

func TestJudgeAssumptionsSettlesNoQuestionDefault(t *testing.T) {
	ideas := []factory.ProductIdea{
		{ID: "d333333333333333333333333", Description: "Build the smallest complete workflow for this objective", Consequence: "Additional capabilities need a new product decision", Decision: "pending", Source: "autonomous", Provenance: "autonomous"},
		{ID: "d444444444444444444444444", Description: "Keep within existing access", Consequence: "No new enrollment", Decision: "pending", Source: "supported", Provenance: "autonomous", Evidence: "prior approved request"},
		{ID: "d555555555555555555555555", Description: "Ask full address", Consequence: "Collect data", Decision: "edit", Correction: "Ask only work email", Source: "requested", Provenance: "human", Actor: 7},
	}
	c, r := draftAssumptionFixture(t, ideas, 5)
	chooser := &assumptionChooser{selected: "challenge", confidence: 1}
	c.Planning = &PlanningJudgments{Journal: c.Store, Funding: c.Store, Chooser: chooser}
	content, err := factory.ComposeProposalContent(r)
	if err != nil {
		t.Fatal(err)
	}
	review, err := c.JudgeAssumptions(context.Background(), r, content, r.Ideas)
	if err != nil {
		t.Fatal(err)
	}
	if len(review.Challenges) != 0 || len(review.Settled) != 2 || chooser.calls != 0 {
		t.Fatalf("review=%+v calls=%d", review, chooser.calls)
	}
	proposal := factory.SettleAssumptions(r, review)
	if err := proposal.Validate(); err != nil {
		t.Fatal(err)
	}
	byID := map[string]factory.ProductIdea{}
	for _, idea := range proposal.Ideas {
		byID[idea.ID] = idea
	}
	if byID["d333333333333333333333333"].Decision != "accept" || byID["d333333333333333333333333"].Provenance != "autonomous" {
		t.Fatalf("autonomous settlement=%+v", byID["d333333333333333333333333"])
	}
	if byID["d444444444444444444444444"].Decision != "accept" || byID["d444444444444444444444444"].Evidence != "prior approved request" {
		t.Fatalf("supported settlement=%+v", byID["d444444444444444444444444"])
	}
	if byID["d555555555555555555555555"].Decision != "edit" || byID["d555555555555555555555555"].Correction != "Ask only work email" || byID["d555555555555555555555555"].Provenance != "human" {
		t.Fatalf("correction=%+v", byID["d555555555555555555555555"])
	}
}

func TestJudgeAssumptionsTargetedCorrectionHoldsOneBranch(t *testing.T) {
	ideas := []factory.ProductIdea{
		{ID: "d333333333333333333333333", Description: "Track leave requests", Consequence: "Managers review requests", Decision: "edit", Correction: "Track leave requests with half-day granularity", Source: "requested", Provenance: "human", Actor: 7},
		{ID: "d444444444444444444444444", Description: "Everyone can see each request", Consequence: "Individual dates are exposed", Assumption: true, Decision: "pending", Source: "assumption", Provenance: "autonomous", Material: true},
		{ID: "d555555555555555555555555", Description: "List newest first", Consequence: "Easy to change sort", Decision: "pending", Source: "autonomous", Provenance: "autonomous"},
	}
	c, r := draftAssumptionFixture(t, ideas, 5)
	c.Planning = &PlanningJudgments{Journal: c.Store, Funding: c.Store, Chooser: &assumptionChooser{selected: "challenge", confidence: 1}}
	content, err := factory.ComposeProposalContent(r)
	if err != nil {
		t.Fatal(err)
	}
	review, err := c.JudgeAssumptions(context.Background(), r, content, r.Ideas)
	if err != nil {
		t.Fatal(err)
	}
	if len(review.Challenges) != 1 || review.Challenges[0].Idea != "d444444444444444444444444" {
		t.Fatalf("challenges=%+v", review.Challenges)
	}
	if len(review.Settled) != 1 || review.Settled[0].Idea != "d555555555555555555555555" {
		t.Fatalf("settled=%+v", review.Settled)
	}
	proposal := factory.SettleAssumptions(r, review)
	if proposal.Ideas[0].Correction != "Track leave requests with half-day granularity" || proposal.Ideas[1].Decision != "pending" || proposal.Ideas[2].Decision != "accept" {
		t.Fatalf("proposal=%+v", proposal.Ideas)
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
	if _, err := c.JudgeAssumptions(context.Background(), r, content, r.Ideas); !errors.Is(err, store.ErrPlanningAllowanceExhausted) {
		t.Fatalf("exhaustion error=%v", err)
	}
	stale := r
	stale.ID = "d999999999999999999999999"
	if _, err := c.JudgeAssumptions(context.Background(), stale, content, stale.Ideas); err == nil {
		t.Fatal("mismatched content accepted")
	}
}
