package factory

import (
	"encoding/json"
	"errors"
	"math"
	"strings"
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
	if got := EffectiveAttemptLimits(nil); got != (AttemptLimits{ActiveMinutes: 120, CorrectionCycles: 3, ReviewRounds: 6, MaxChildren: 3, DeliveryCalls: 8}) {
		t.Fatalf("defaults: %+v", got)
	}
	for _, limits := range []AttemptLimits{
		{ActiveMinutes: 1, CorrectionCycles: 0, ReviewRounds: 1, MaxChildren: 1, DeliveryCalls: 0},
		{ActiveMinutes: 15, CorrectionCycles: 1, ReviewRounds: 6, MaxChildren: 3, DeliveryCalls: 8},
		{ActiveMinutes: 180, CorrectionCycles: 4, ReviewRounds: 16, MaxChildren: 4, DeliveryCalls: 16},
	} {
		if limits.Validate() != nil || EffectiveAttemptLimits(&limits) != limits {
			t.Fatalf("owner limits refused or replaced: %+v", limits)
		}
	}
	for _, limits := range []AttemptLimits{
		{ActiveMinutes: 0, CorrectionCycles: 3, ReviewRounds: 6, MaxChildren: 3, DeliveryCalls: 8},
		{ActiveMinutes: 120, CorrectionCycles: -1, ReviewRounds: 6, MaxChildren: 3, DeliveryCalls: 8},
		{ActiveMinutes: math.MaxInt, CorrectionCycles: 3, ReviewRounds: 6, MaxChildren: 3, DeliveryCalls: 8},
		{ActiveMinutes: 120, CorrectionCycles: 3, ReviewRounds: 0, MaxChildren: 3, DeliveryCalls: 8},
		{ActiveMinutes: 120, CorrectionCycles: 3, ReviewRounds: 17, MaxChildren: 3, DeliveryCalls: 8},
		{ActiveMinutes: 120, CorrectionCycles: 3, ReviewRounds: 6, MaxChildren: 0, DeliveryCalls: 8},
		{ActiveMinutes: 120, CorrectionCycles: 3, ReviewRounds: 6, MaxChildren: 5, DeliveryCalls: 8},
		{ActiveMinutes: 120, CorrectionCycles: 3, ReviewRounds: 6, MaxChildren: 3, DeliveryCalls: -1},
		{ActiveMinutes: 120, CorrectionCycles: 3, ReviewRounds: 6, MaxChildren: 3, DeliveryCalls: 17},
	} {
		if limits.Validate() == nil {
			t.Fatalf("invalid limits accepted: %+v", limits)
		}
	}
}

func TestEffectiveAttemptLimitsFillsOnlyUnambiguousOmissions(t *testing.T) {
	partial := AttemptLimits{ActiveMinutes: 45, CorrectionCycles: 1}
	if got := EffectiveAttemptLimits(&partial); got != (AttemptLimits{ActiveMinutes: 45, CorrectionCycles: 1, ReviewRounds: 6, MaxChildren: 3}) {
		t.Fatalf("creation omitted review/child defaults: %+v", got)
	}
	explicit := AttemptLimits{ActiveMinutes: 45, CorrectionCycles: 1, ReviewRounds: 2, MaxChildren: 2, DeliveryCalls: 0}
	if got := EffectiveAttemptLimits(&explicit); got != explicit {
		t.Fatalf("creation replaced explicit values: %+v", got)
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
	a.Limits = AttemptLimits{ActiveMinutes: 1, CorrectionCycles: 3, ReviewRounds: 6, MaxChildren: 3, DeliveryCalls: 8}
	if a.RemainingSeconds(start.Add(59*time.Second)) != 1 || a.RemainingSeconds(start.Add(time.Minute)) != 0 {
		t.Fatal("exact deadline not enforced")
	}
	if _, _, err := a.ConsumeCorrection(NewID(), start.Add(time.Minute)); !errors.Is(err, ErrAttemptTimeExhausted) {
		t.Fatal("correction launched after time exhaustion", err)
	}
}

func TestDeliveryReservationIsIdempotentBoundedAndStaysCharged(t *testing.T) {
	a, start := allowanceFixture()
	first := NewID()
	next, consumed, err := a.ReserveDeliveryCall(first, start)
	if err != nil || !consumed || len(next.DeliveryCalls) != 1 || len(a.DeliveryCalls) != 0 {
		t.Fatalf("first reservation: %+v %v %v", next, consumed, err)
	}
	replayed, consumed, err := next.ReserveDeliveryCall(first, start.Add(time.Minute))
	if err != nil || consumed || len(replayed.DeliveryCalls) != 1 || replayed.ActiveSeconds != 60 {
		t.Fatalf("lost-reply replay: %+v %v %v", replayed, consumed, err)
	}
	for i := 1; i < 8; i++ {
		replayed, consumed, err = replayed.ReserveDeliveryCall(NewID(), start.Add(time.Minute))
		if err != nil || !consumed {
			t.Fatalf("funded call %d refused: %v %v", i, consumed, err)
		}
	}
	if _, _, err := replayed.ReserveDeliveryCall(NewID(), start.Add(time.Minute)); !errors.Is(err, ErrDeliveryExhausted) {
		t.Fatal("exhausted delivery subledger admitted a call", err)
	}
	if replay, consumed, err := replayed.ReserveDeliveryCall(first, start.Add(time.Minute)); err != nil || consumed || len(replay.DeliveryCalls) != 8 {
		t.Fatalf("exhausted replay charged again: %+v %v %v", replay, consumed, err)
	}
	if _, _, err := replayed.ReserveDeliveryCall("short", start); err == nil {
		t.Fatal("invalid delivery identity admitted")
	}
	// A zero delivery limit authorizes no call, so automatic merge stays
	// unreachable on an unfunded attempt.
	unfunded := a
	unfunded.Limits.DeliveryCalls = 0
	if _, _, err := unfunded.ReserveDeliveryCall(NewID(), start); !errors.Is(err, ErrDeliveryExhausted) {
		t.Fatal("unfunded attempt reserved a delivery call", err)
	}
	// Failure after reservation stays charged: replay observes the same
	// operation instead of freeing it, and closure bars new spending.
	if _, _, err := replayed.ReserveDeliveryCall(NewID(), start.Add(2*time.Hour)); !errors.Is(err, ErrAttemptTimeExhausted) {
		t.Fatal("expired attempt reserved a delivery call", err)
	}
	closed := replayed
	closed.Closed = true
	if _, _, err := closed.ReserveDeliveryCall(NewID(), start.Add(time.Minute)); !errors.Is(err, ErrAttemptClosed) {
		t.Fatal("closed attempt reserved a delivery call", err)
	}
}

func TestSpendingNeverResetsAttemptBudgets(t *testing.T) {
	a, start := allowanceFixture()
	charged, _, err := a.ConsumeCorrection(NewID(), start.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	reserved, _, err := charged.ReserveDeliveryCall(NewID(), start.Add(2*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if reserved.ActiveSeconds != 120 || len(reserved.Corrections) != 1 || len(reserved.DeliveryCalls) != 1 {
		t.Fatalf("spending reset budgets: %+v", reserved)
	}
	rested, err := reserved.Checkpoint(start.Add(3*time.Minute), false)
	if err != nil || len(rested.Corrections) != 1 || len(rested.DeliveryCalls) != 1 {
		t.Fatalf("checkpoint dropped consumption: %+v %v", rested, err)
	}
	if err := rested.Validate(); err != nil {
		t.Fatal(err)
	}
	over := rested
	over.DeliveryCalls = append(append([]string(nil), rested.DeliveryCalls...), NewID(), NewID(), NewID(), NewID(), NewID(), NewID(), NewID(), NewID())
	if over.Validate() == nil {
		t.Fatal("over-limit delivery subledger validated")
	}
	dup := rested
	dup.DeliveryCalls = []string{rested.DeliveryCalls[0], rested.DeliveryCalls[0]}
	if dup.Validate() == nil {
		t.Fatal("duplicated delivery identity validated")
	}
}

func completionAllowanceFixture() (AttemptAllowance, time.Time) {
	now := time.Unix(1700000000, 0)
	return AttemptAllowance{
		Limits:         AttemptLimits{ActiveMinutes: 30},
		Completion:     &CompletionSubject{CompletionID: NewID(), CycleID: NewID(), InventoryDigest: strings.Repeat("d", 64)},
		Repository:     7,
		RootAssignment: NewID(), CheckpointUnix: now.Unix(), Active: true,
	}, now
}

func TestCompletionAllowanceAdmitsOnlyAdmittedTimeOnlyRecords(t *testing.T) {
	a, _ := completionAllowanceFixture()
	if err := a.Validate(); err != nil {
		t.Fatalf("valid completion time-only record refused: %v", err)
	}
	raw, err := json.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}
	var decoded AttemptAllowance
	if err = json.Unmarshal(raw, &decoded); err != nil || decoded.Completion == nil ||
		*decoded.Completion != *a.Completion || decoded.Issue != 0 {
		t.Fatalf("completion subject lost its JSON round trip: %+v %v", decoded, err)
	}
	for _, minutes := range []int{1, 30, 120} {
		admitted := a
		admitted.Limits.ActiveMinutes = minutes
		if err := admitted.Validate(); err != nil {
			t.Fatalf("admitted %d minutes refused: %v", minutes, err)
		}
	}
	for _, minutes := range []int{0, -1, 121, 200} {
		denied := a
		denied.Limits.ActiveMinutes = minutes
		if denied.Validate() == nil {
			t.Fatalf("unadmitted %d minutes accepted", minutes)
		}
	}
	// AttemptLimits.Validate stays unchanged for ordinary issue policy: the
	// time-only zero sub-budgets never validate as issue limits.
	if (AttemptLimits{ActiveMinutes: 30}).Validate() == nil {
		t.Fatal("issue limits accepted zero review and child budgets")
	}
	for _, mutate := range []func(*AttemptAllowance){
		func(x *AttemptAllowance) { x.Limits.CorrectionCycles = 1 },
		func(x *AttemptAllowance) { x.Limits.ReviewRounds = 1 },
		func(x *AttemptAllowance) { x.Limits.MaxChildren = 1 },
		func(x *AttemptAllowance) { x.Limits.DeliveryCalls = 1 },
		func(x *AttemptAllowance) { x.Corrections = []string{NewID()} },
		func(x *AttemptAllowance) { x.DeliveryCalls = []string{NewID()} },
		func(x *AttemptAllowance) { x.Reviews = []string{NewID()} },
		func(x *AttemptAllowance) { x.ReviewedHeads = map[string]string{strings.Repeat("c", 40): NewID()} },
	} {
		forged := a
		mutate(&forged)
		if forged.Validate() == nil {
			t.Fatalf("completion record with issue budget accepted: %+v", forged)
		}
	}
	for _, mutate := range []func(*AttemptAllowance){
		func(x *AttemptAllowance) { x.Completion = nil },
		func(x *AttemptAllowance) { x.Completion.CompletionID = "short" },
		func(x *AttemptAllowance) { x.Completion.CycleID = "short" },
		func(x *AttemptAllowance) { x.Completion.InventoryDigest = "short" },
		func(x *AttemptAllowance) { x.Issue = -1 },
		func(x *AttemptAllowance) { x.Repository = 0 },
	} {
		broken := a
		subject := *a.Completion
		broken.Completion = &subject
		mutate(&broken)
		if broken.Validate() == nil {
			t.Fatalf("malformed completion record accepted: %+v", broken)
		}
	}
}

func TestCompletionSubjectForbiddenOnPositiveIssues(t *testing.T) {
	a, _ := allowanceFixture()
	a.Completion = &CompletionSubject{CompletionID: NewID(), CycleID: NewID(), InventoryDigest: strings.Repeat("d", 64)}
	if a.Validate() == nil {
		t.Fatal("positive-issue allowance accepted a completion subject")
	}
	plain, _ := allowanceFixture()
	if plain.Completion != nil || plain.Validate() != nil {
		t.Fatal("positive-issue allowance changed shape")
	}
}

func TestCompletionRootRefusesEveryIssueBudgetButKeepsTime(t *testing.T) {
	a, start := completionAllowanceFixture()
	head := strings.Repeat("c", 40)
	if _, _, err := a.ConsumeReview(NewID(), head, false, start); !errors.Is(err, ErrCompletionTimeOnly) {
		t.Fatalf("completion root consumed a review round: %v", err)
	}
	if _, _, err := a.ConsumeCorrection(NewID(), start); !errors.Is(err, ErrCompletionTimeOnly) {
		t.Fatalf("completion root consumed a correction cycle: %v", err)
	}
	if _, _, err := a.ReserveDeliveryCall(NewID(), start); !errors.Is(err, ErrCompletionTimeOnly) {
		t.Fatalf("completion root reserved a delivery call: %v", err)
	}
	if a.ReviewFits(head, true, start) {
		t.Fatal("completion root fits a review")
	}
	stopped, err := a.Checkpoint(start.Add(10*time.Minute), false)
	if err != nil || stopped.ActiveSeconds != 600 || stopped.RemainingSeconds(start.Add(time.Hour)) != 30*60-600 {
		t.Fatalf("completion active time did not accrue: %+v %v", stopped, err)
	}
	if err := stopped.Validate(); err != nil {
		t.Fatalf("checkpointed completion record refused: %v", err)
	}
	closed := stopped
	closed.Closed = true
	if _, err := closed.Checkpoint(start.Add(11*time.Minute), true); !errors.Is(err, ErrAttemptClosed) {
		t.Fatalf("closed completion root resumed: %v", err)
	}
}

func TestClosedAttemptAllowanceCanStopButCannotResumeOrConsumeCorrection(t *testing.T) {
	a, start := allowanceFixture()
	a.Closed = true

	stopped, err := a.Checkpoint(start.Add(time.Minute), false)
	if err != nil || stopped.Active || !stopped.Closed || stopped.ActiveSeconds != 60 {
		t.Fatalf("closed clock did not preserve elapsed charge while stopping: %+v %v", stopped, err)
	}
	if _, err := stopped.Checkpoint(start.Add(2*time.Minute), true); !errors.Is(err, ErrAttemptClosed) {
		t.Fatalf("closed attempt resumed: %v", err)
	}
	if _, _, err := a.ConsumeCorrection(NewID(), start.Add(time.Minute)); !errors.Is(err, ErrAttemptClosed) {
		t.Fatalf("closed attempt consumed a correction: %v", err)
	}
}
