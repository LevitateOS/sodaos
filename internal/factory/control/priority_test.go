package control

import (
	"context"
	"fmt"
	"github.com/levitateos/sodaos/internal/store"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
)

type priorityFixture struct {
	calls    []factory.PriorityRequest
	receipts []factory.PriorityReceipt
	reserved int
	choose   func(factory.PriorityRequest) factory.PriorityAnswer
}

func (f *priorityFixture) ReservePriorityJudgment(_ context.Context, _ string, calls int) (bool, error) {
	f.reserved = calls
	return true, nil
}
func (f *priorityFixture) RecordPriorityJudgment(_ context.Context, r factory.PriorityReceipt) error {
	f.receipts = append(f.receipts, r)
	return nil
}
func (f *priorityFixture) PriorityJudgment(_ context.Context, snapshot string) (factory.PriorityReceipt, error) {
	for _, r := range f.receipts {
		if r.Snapshot == snapshot {
			return r, nil
		}
	}
	return factory.PriorityReceipt{}, store.ErrNotFound
}
func (f *priorityFixture) Choose(_ context.Context, r factory.PriorityRequest) (factory.PriorityAnswer, error) {
	f.calls = append(f.calls, r)
	return f.choose(r), nil
}

func priorityLast(r factory.PriorityRequest) factory.PriorityAnswer {
	answer := factory.PriorityAnswer{Selected: r.Candidates[len(r.Candidates)-1].ID, Model: "synthetic-model", Confidence: 1, Probabilities: map[string]float64{}}
	for _, c := range r.Candidates {
		answer.Probabilities[c.ID] = 0
	}
	answer.Probabilities[answer.Selected] = 1
	return answer
}

func priorityCandidates(count int, now time.Time) []factory.PriorityCandidate {
	var result []factory.PriorityCandidate
	for i := 0; i < count; i++ {
		result = append(result, factory.PriorityCandidate{ID: fmt.Sprint(i), Control: factory.IssueControl{FirstSeenUnix: now.Unix() + int64(i)}})
	}
	return result
}

func TestPriorityCompleteTournamentAndFiniteCalls(t *testing.T) {
	now := time.Now()
	f := &priorityFixture{choose: priorityLast}
	deps := &PriorityJudgments{Journal: f, Chooser: f, MaxCalls: 2}
	selected, err := selectPriority(context.Background(), deps, priorityCandidates(256, now), now)
	if err != nil || selected != "255" || f.reserved != 2 || len(f.calls) != 2 || len(f.receipts) != 1 {
		t.Fatalf("selection=%s error=%v calls=%d reservation=%d", selected, err, len(f.calls), f.reserved)
	}
	seen := map[string]bool{}
	for _, call := range f.calls {
		if len(call.Candidates) > 255 {
			t.Fatal("Choice overflow")
		}
		for _, c := range call.Candidates {
			seen[c.ID] = true
		}
	}
	if len(seen) != 256 {
		t.Fatalf("covered %d candidates", len(seen))
	}
	f = &priorityFixture{choose: priorityLast}
	deps.Journal, deps.Chooser, deps.MaxCalls = f, f, 1
	if _, err := selectPriority(context.Background(), deps, priorityCandidates(256, now), now); err == nil || len(f.calls) != 0 || f.reserved != 0 {
		t.Fatal("overbudget selection spent calls")
	}
}

func TestPriorityUncertaintyAgeAndSingleCandidate(t *testing.T) {
	now := time.Now()
	f := &priorityFixture{choose: func(r factory.PriorityRequest) factory.PriorityAnswer {
		a := priorityLast(r)
		a.Confidence = 0.1
		return a
	}}
	deps := &PriorityJudgments{Journal: f, Chooser: f}
	candidates := priorityCandidates(2, now)
	if chosen, err := selectPriority(context.Background(), deps, candidates, now); err != nil || chosen != "0" {
		t.Fatalf("uncertainty=%s %v", chosen, err)
	}
	candidates[0].Control.FirstSeenUnix = now.Add(-25 * time.Hour).Unix()
	if chosen, err := selectPriority(context.Background(), deps, candidates, now); err != nil || chosen != "0" || len(f.calls) != 2 {
		t.Fatalf("age=%s %v", chosen, err)
	}
	if chosen, err := selectPriority(context.Background(), nil, candidates[:1], now); err != nil || chosen != "0" {
		t.Fatalf("single=%s %v", chosen, err)
	}
}

func TestPriorityDispatchSelectsAndRefusesStaleSet(t *testing.T) {
	for _, stale := range []bool{false, true} {
		t.Run(fmt.Sprint(stale), func(t *testing.T) {
			ctx := context.Background()
			db, _ := dispatchTestDB(t)
			fx := dispatchSeed(t, db)
			old := fx.accept(t, 3, "d333333333333333333333333")
			young := fx.accept(t, 5, "d555555555555555555555555")
			fx.queueAt(t, 3, old.ID, time.Now().Add(-time.Hour))
			fx.queueAt(t, 5, young.ID, time.Now())
			f := &priorityFixture{choose: func(r factory.PriorityRequest) factory.PriorityAnswer {
				if stale {
					if err := db.SaveCapacity(ctx, factory.Capacity{Revision: 1, UpdatedBy: 7, MaxConcurrentRuns: 1, MaxQueued: 10}); err != nil {
						t.Fatal(err)
					}
				}
				return priorityLast(r)
			}}
			deps := fx.deps()
			deps.Priority = &PriorityJudgments{Journal: f, Chooser: f}
			report := DispatchPass(ctx, deps)
			if stale {
				if len(report.Launched) != 0 || len(fx.host.launches) != 0 {
					t.Fatalf("stale launched: %+v", report)
				}
			} else if len(report.Launched) != 1 || report.Launched[0].Issue != 5 {
				t.Fatalf("priority launch: %+v", report)
			}
		})
	}
}
