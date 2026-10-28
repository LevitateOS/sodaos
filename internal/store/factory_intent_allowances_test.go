package store

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
)

func TestPlanningAllowanceCASAndEffect(t *testing.T) {
	s, _ := postgresFixture(t)
	ctx := context.Background()
	intent := factory.IntentRevision{ID: "d111111111111111111111111", Intent: "d222222222222222222222222", Owner: 1, Objective: "Tracker", State: "draft", MaxIssues: 10, CreatedUnix: 1}
	if err := s.SaveIntentRevision(ctx, intent); err != nil {
		t.Fatal(err)
	}
	a := factory.PlanningAllowance{ID: "d333333333333333333333333", Intent: intent.Intent, Owner: 1, Amount: 5, Unit: factory.PlanningAllowanceUnit, Ceiling: 10, CreatedUnix: 1}
	if err := s.SavePlanningAllowance(ctx, a); err != nil {
		t.Fatal(err)
	}
	if err := s.SavePlanningAllowance(ctx, a); err != nil {
		t.Fatal(err)
	}
	b := a
	b.ID = "d444444444444444444444444"
	b.Predecessor = a.ID
	b.Amount = 8
	if s.SavePlanningAllowance(ctx, b) == nil {
		t.Fatal("allowance increase without effect accepted")
	}
	b.Effect = "Add 3 calls to compare two more alternatives"
	if err := s.SavePlanningAllowance(ctx, b); err != nil {
		t.Fatal(err)
	}
	got, err := s.PlanningAllowance(ctx, intent.Intent, 1)
	if err != nil || got.ID != b.ID || got.Reserved != 0 || got.Consumed != 0 || got.Available() != 8 {
		t.Fatalf("allowance read %v %v", got, err)
	}
	if _, err = s.PlanningAllowance(ctx, intent.Intent, 2); err == nil {
		t.Fatal("allowance visible to another owner")
	}
}

func TestPlanningUsageDerivesFromRuntimeRecords(t *testing.T) {
	s, _ := postgresFixture(t)
	ctx := context.Background()
	r := factory.IntentRevision{ID: "d111111111111111111111111", Intent: "d222222222222222222222222", Owner: 1, Objective: "Tracker", State: "draft", MaxIssues: 10, CreatedUnix: 1}
	if err := s.SaveIntentRevision(ctx, r); err != nil {
		t.Fatal(err)
	}
	if err := s.SavePlanningAllowance(ctx, factory.PlanningAllowance{ID: "d333333333333333333333333", Intent: r.Intent, Owner: 1, Amount: 4, Unit: factory.PlanningAllowanceUnit, Ceiling: 4, CreatedUnix: 1}); err != nil {
		t.Fatal(err)
	}
	answered := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	failed := "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	pending := "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
	for _, snapshot := range []string{answered, failed, pending} {
		if ok, err := s.ReserveIntentJudgment(ctx, r, snapshot); err != nil || !ok {
			t.Fatalf("reserve %v %v", ok, err)
		}
	}
	receipt := func(snapshot, failure string) factory.PriorityReceipt {
		call := factory.PriorityCall{Answer: factory.PriorityAnswer{Selected: "keep", Model: "synthetic-model", Confidence: 1, Probabilities: map[string]float64{"keep": 1}}}
		if failure != "" {
			call.Failure = failure
		}
		return factory.PriorityReceipt{Snapshot: snapshot, Calls: []factory.PriorityCall{call}, Selected: call.Answer.Selected, Outcome: "decomposition", CreatedUnix: 1}
	}
	if err := s.RecordPriorityJudgment(ctx, receipt(answered, "")); err != nil {
		t.Fatal(err)
	}
	failedReceipt := receipt(failed, "consultation_failed")
	failedReceipt.Outcome = "unavailable"
	if err := s.RecordPriorityJudgment(ctx, failedReceipt); err != nil {
		t.Fatal(err)
	}
	got, err := s.PlanningAllowance(ctx, r.Intent, r.Owner)
	if err != nil {
		t.Fatal(err)
	}
	if got.Reserved != 1 || got.Consumed != 2 || got.Available() != 1 {
		t.Fatalf("live usage=%+v", got)
	}
	if ok, err := s.ReserveIntentJudgment(ctx, r, answered); err != nil || ok {
		t.Fatal("recorded snapshot reserved twice")
	}
}

func TestConcurrentPlanningReservationsCannotOverspend(t *testing.T) {
	s, _ := postgresFixture(t)
	ctx := context.Background()
	r := factory.IntentRevision{ID: "d111111111111111111111111", Intent: "d222222222222222222222222", Owner: 1, Objective: "Tracker", State: "draft", MaxIssues: 10, CreatedUnix: 1}
	if err := s.SaveIntentRevision(ctx, r); err != nil {
		t.Fatal(err)
	}
	if err := s.SavePlanningAllowance(ctx, factory.PlanningAllowance{ID: "d333333333333333333333333", Intent: r.Intent, Owner: 1, Amount: 1, Unit: factory.PlanningAllowanceUnit, Ceiling: 1, CreatedUnix: 1}); err != nil {
		t.Fatal(err)
	}
	var admitted atomic.Int32
	var group sync.WaitGroup
	for i := 0; i < 8; i++ {
		group.Add(1)
		go func(i int) {
			defer group.Done()
			snapshot := string(make([]byte, 0, 64))
			for j := 0; j < 64; j++ {
				snapshot += string(rune('a' + (i+j)%6))
			}
			ok, err := s.ReserveIntentJudgment(ctx, r, snapshot)
			if ok {
				admitted.Add(1)
			} else if !errors.Is(err, ErrPlanningAllowanceExhausted) {
				t.Error(err)
				return
			}
		}(i)
	}
	group.Wait()
	if admitted.Load() != 1 {
		t.Fatalf("admitted=%d", admitted.Load())
	}
}
