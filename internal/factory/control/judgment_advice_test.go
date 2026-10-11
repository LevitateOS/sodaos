package control

import (
	"context"
	"errors"
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
)

type purposeChooser struct {
	calls int
	fail  bool
}

func (f *purposeChooser) Select(_ context.Context, _ any, _ string, choices map[string]any) (factory.PriorityAnswer, error) {
	f.calls++
	if f.fail {
		return factory.PriorityAnswer{}, errors.New("consultation down")
	}
	a := factory.PriorityAnswer{Selected: "a", Model: "synthetic-model", Confidence: 1, Probabilities: map[string]float64{}}
	for id := range choices {
		a.Probabilities[id] = 0
	}
	a.Probabilities["a"] = 1
	return a, nil
}

func purposeAdvice(c *Coordinator, r factory.IntentRevision) PurposeAdvice {
	return PurposeAdvice{
		Purpose:  factory.JudgmentConsolidation,
		State:    map[string]string{"objective": r.Objective},
		Question: "Which option preserves the requested outcomes?",
		Options:  map[string]any{"a": "First option", "b": "Second option"},
		Outcome:  "consolidation",
		Reserve: func(ctx context.Context, snapshot string) (bool, error) {
			return c.Store.ReserveIntentJudgment(ctx, r, snapshot)
		},
	}
}

func TestAdvisePurposeCachesCompletedResult(t *testing.T) {
	c, _, r := acceptedGenerationFixture(t)
	chooser := &purposeChooser{}
	c.Planning = &PlanningJudgments{Journal: c.Store, Funding: c.Store, Chooser: chooser}
	advice := purposeAdvice(c, r)
	first, cached, err := c.AdvisePurpose(context.Background(), advice)
	if err != nil || cached || first.Selected != "a" {
		t.Fatalf("first advice=%+v cached=%v err=%v", first, cached, err)
	}
	second, cached, err := c.AdvisePurpose(context.Background(), advice)
	if err != nil || !cached || second.Selected != "a" || chooser.calls != 1 {
		t.Fatalf("reuse advice=%+v cached=%v calls=%d err=%v", second, cached, chooser.calls, err)
	}
}

func TestAdvisePurposeFailureStaysChargedWithoutRepeat(t *testing.T) {
	c, _, r := acceptedGenerationFixture(t)
	failing := &purposeChooser{fail: true}
	c.Planning = &PlanningJudgments{Journal: c.Store, Funding: c.Store, Chooser: failing}
	advice := purposeAdvice(c, r)
	if _, _, err := c.AdvisePurpose(context.Background(), advice); !errors.Is(err, ErrPriorityUnavailable) {
		t.Fatalf("failure error=%v", err)
	}
	healthy := &purposeChooser{}
	c.Planning.Chooser = healthy
	if _, _, err := c.AdvisePurpose(context.Background(), advice); !errors.Is(err, ErrPriorityUnavailable) {
		t.Fatalf("charged retry error=%v", err)
	}
	if healthy.calls != 0 {
		t.Fatal("charged fingerprint silently repeated the consultation")
	}
}

func TestAdvisePurposeReserveRefusalStaysUnavailable(t *testing.T) {
	c, _, r := acceptedGenerationFixture(t)
	chooser := &purposeChooser{}
	c.Planning = &PlanningJudgments{Journal: c.Store, Funding: c.Store, Chooser: chooser}
	advice := purposeAdvice(c, r)
	advice.Reserve = func(context.Context, string) (bool, error) { return false, nil }
	if _, _, err := c.AdvisePurpose(context.Background(), advice); !errors.Is(err, ErrPriorityUnavailable) {
		t.Fatalf("refusal error=%v", err)
	}
	if chooser.calls != 0 {
		t.Fatal("unreserved consultation was attempted")
	}
	advice = purposeAdvice(c, r)
	advice.Purpose = "invented"
	if _, _, err := c.AdvisePurpose(context.Background(), advice); err == nil {
		t.Fatal("unknown purpose accepted")
	}
}
