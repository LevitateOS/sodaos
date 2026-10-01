package factory

import (
	"errors"
	"math"
	"testing"
	"time"
)

func allowanceFixture() (AttemptAllowance, time.Time) {
	now := time.Unix(1700000000, 0)
	return AttemptAllowance{
		Limits: DefaultAttemptLimits(), Corrections: []string{},
		Repository: 7, Issue: 3, RootAssignment: NewID(),
		CheckpointUnix: now.Unix(), Active: true,
	}, now
}

func TestAttemptLimitsDefaultsAndOwnerValues(t *testing.T) {
	if got := EffectiveAttemptLimits(nil); got.ActiveMinutes != 120 || got.CorrectionCycles != 3 {
		t.Fatalf("defaults: %+v", got)
	}
	for _, limits := range []AttemptLimits{{1, 0}, {15, 1}, {180, 4}} {
		if limits.Validate() != nil || EffectiveAttemptLimits(&limits) != limits {
			t.Fatalf("owner limits refused or replaced: %+v", limits)
		}
	}
	for _, limits := range []AttemptLimits{{0, 3}, {120, -1}, {math.MaxInt, 3}} {
		if limits.Validate() == nil {
			t.Fatalf("invalid limits accepted: %+v", limits)
		}
	}
}

func TestAttemptAllowanceExcludesOnlyConfirmedQueuedIntervals(t *testing.T) {
	a, start := allowanceFixture()
	paused, err := a.Checkpoint(start.Add(10*time.Minute), false)
	if err != nil || paused.ActiveSeconds != 600 {
		t.Fatalf("first active interval: %+v %v", paused, err)
	}
	resumed, err := paused.Checkpoint(start.Add(2*time.Hour), true)
	if err != nil || resumed.ActiveSeconds != 600 || resumed.RemainingSeconds(start.Add(2*time.Hour)) != 6600 {
		t.Fatalf("queued interval charged: %+v %v", resumed, err)
	}
	ended, err := resumed.Checkpoint(start.Add(2*time.Hour+5*time.Minute), false)
	if err != nil || ended.ActiveSeconds != 900 || ended.RemainingSeconds(start.Add(24*time.Hour)) != 6300 {
		t.Fatalf("resumed interval: %+v %v", ended, err)
	}
	if a.ActiveSeconds != 0 {
		t.Fatal("checkpoint mutated prior record")
	}
	if _, err := ended.Checkpoint(start, true); !errors.Is(err, ErrAllowanceClock) || ended.RemainingSeconds(start) != 0 {
		t.Fatal("clock regression granted time", err)
	}
}

func TestCorrectionConsumptionChargesFailuresAndReplays(t *testing.T) {
	a, start := allowanceFixture()
	first := NewID()
	next, consumed, err := a.ConsumeCorrection(first, start)
	if err != nil || !consumed || len(next.Corrections) != 1 || len(a.Corrections) != 0 {
		t.Fatalf("first charge: %+v %v %v", next, consumed, err)
	}
	replayed, consumed, err := next.ConsumeCorrection(first, start.Add(time.Minute))
	if err != nil || consumed || len(replayed.Corrections) != 1 || replayed.ActiveSeconds != 60 {
		t.Fatalf("lost-reply replay: %+v %v %v", replayed, consumed, err)
	}
	// No successful correction is reported: each failed launch still keeps its
	// prior charge, and a different assignment consumes the next cycle.
	for range 2 {
		replayed, consumed, err = replayed.ConsumeCorrection(NewID(), start.Add(time.Minute))
		if err != nil || !consumed {
			t.Fatalf("remaining cycle refused: %v %v", consumed, err)
		}
	}
	if _, _, err := replayed.ConsumeCorrection(NewID(), start.Add(time.Minute)); !errors.Is(err, ErrCorrectionExhausted) {
		t.Fatal("exhaustion did not stop non-progress", err)
	}
	if replay, consumed, err := replayed.ConsumeCorrection(first, start.Add(time.Minute)); err != nil || consumed || len(replay.Corrections) != 3 {
		t.Fatalf("exhausted replay charged again: %+v %v %v", replay, consumed, err)
	}
}

func TestAttemptTimeExhaustionStopsCorrections(t *testing.T) {
	a, start := allowanceFixture()
	a.Limits = AttemptLimits{ActiveMinutes: 1, CorrectionCycles: 3}
	if a.RemainingSeconds(start.Add(59*time.Second)) != 1 || a.RemainingSeconds(start.Add(time.Minute)) != 0 {
		t.Fatal("exact deadline not enforced")
	}
	if _, _, err := a.ConsumeCorrection(NewID(), start.Add(time.Minute)); !errors.Is(err, ErrAttemptTimeExhausted) {
		t.Fatal("correction launched after time exhaustion", err)
	}
}
