package factory

import (
	"encoding/json"
	"errors"
	"math"
	"reflect"
	"strings"
	"testing"
	"time"
)

func validAllowance() PlanningAllowance {
	return PlanningAllowance{ID: "d111111111111111111111111", Intent: "d222222222222222222222222", Owner: 1, Amount: 5, Unit: PlanningAllowanceUnit, Reserved: 1, Consumed: 2, Ceiling: 10, CreatedUnix: 1}
}

func TestPlanningAllowanceBoundsAndAvailability(t *testing.T) {
	a := validAllowance()
	if err := a.Validate(); err != nil {
		t.Fatal(err)
	}
	if a.Available() != 2 {
		t.Fatal("available miscomputed")
	}
	a.Reserved = 4
	if a.Validate() == nil {
		t.Fatal("overcommitted allowance accepted")
	}
	a = validAllowance()
	a.Amount = 11
	if a.Validate() == nil {
		t.Fatal("allowance above ceiling accepted")
	}
	a = validAllowance()
	a.Unit = "credits"
	if a.Validate() == nil {
		t.Fatal("billing unit accepted")
	}
}

func intentAllowanceFixture() (AttemptAllowance, time.Time) {
	now := time.Unix(1700000000, 0)
	return AttemptAllowance{
		Limits:         AttemptLimits{ActiveMinutes: 45},
		Intent:         &IntentAttemptScope{IntentID: NewID(), OwnerID: 7},
		Repository:     7,
		RootAssignment: NewID(), CheckpointUnix: now.Unix(), Active: true,
	}, now
}

func cloneIntentAllowance(a AttemptAllowance) AttemptAllowance {
	out := a
	if a.Intent != nil {
		scope := *a.Intent
		out.Intent = &scope
	}
	return out
}

func TestIntentAllowanceAdmitsOnlyStableTimeOnlyRecords(t *testing.T) {
	a, _ := intentAllowanceFixture()
	if err := a.Validate(); err != nil {
		t.Fatalf("valid intent time-only record refused: %v", err)
	}
	raw, err := json.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}
	var decoded AttemptAllowance
	if err = json.Unmarshal(raw, &decoded); err != nil || decoded.Intent == nil ||
		*decoded.Intent != *a.Intent || decoded.Issue != 0 || decoded.Completion != nil {
		t.Fatalf("intent scope lost its JSON round trip: %+v %v", decoded, err)
	}
	if decoded.Limits != (AttemptLimits{ActiveMinutes: 45}) || decoded.RootAssignment != a.RootAssignment {
		t.Fatalf("intent record lost limits or root in JSON round trip: %+v", decoded)
	}
	// No 1..120 reassessment cap on intent policy time: 180 admits, and the
	// upper bound matches AttemptLimits.Validate exactly.
	admitted := a
	admitted.Limits.ActiveMinutes = 180
	if err := admitted.Validate(); err != nil {
		t.Fatalf("admitted 180 intent minutes refused: %v", err)
	}
	bound := int(math.MaxInt64 / int64(time.Minute))
	for _, minutes := range []int{1, 45, 120, bound} {
		admitted := a
		admitted.Limits.ActiveMinutes = minutes
		if err := admitted.Validate(); err != nil {
			t.Fatalf("admitted %d intent minutes refused: %v", minutes, err)
		}
	}
	for _, minutes := range []int{0, -1, bound + 1, math.MaxInt} {
		denied := a
		denied.Limits.ActiveMinutes = minutes
		if denied.Validate() == nil {
			t.Fatalf("unadmitted %d intent minutes accepted", minutes)
		}
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
		func(x *AttemptAllowance) { x.UnchangedHeadExtra = map[string]string{strings.Repeat("c", 40): NewID()} },
	} {
		forged := cloneIntentAllowance(a)
		mutate(&forged)
		if forged.Validate() == nil {
			t.Fatalf("intent record with issue budget accepted: %+v", forged)
		}
	}
	for _, mutate := range []func(*AttemptAllowance){
		func(x *AttemptAllowance) { x.Intent = nil },
		func(x *AttemptAllowance) {
			x.Completion = &CompletionSubject{CompletionID: NewID(), CycleID: NewID(), InventoryDigest: strings.Repeat("d", 64)}
		},
		func(x *AttemptAllowance) { x.Intent.IntentID = "short" },
		func(x *AttemptAllowance) { x.Intent.OwnerID = 0 },
		func(x *AttemptAllowance) { x.Intent.OwnerID = -1 },
		func(x *AttemptAllowance) { x.Issue = -1 },
		func(x *AttemptAllowance) { x.Repository = 0 },
		func(x *AttemptAllowance) { x.RootAssignment = "short" },
		func(x *AttemptAllowance) { x.Revision = -1 },
		func(x *AttemptAllowance) { x.ActiveSeconds = -1 },
		func(x *AttemptAllowance) { x.CheckpointUnix = 0 },
	} {
		broken := cloneIntentAllowance(a)
		mutate(&broken)
		if broken.Validate() == nil {
			t.Fatalf("malformed intent record accepted: %+v", broken)
		}
	}
}

func TestIntentScopeForbiddenOnPositiveIssues(t *testing.T) {
	a, _ := allowanceFixture()
	a.Intent = &IntentAttemptScope{IntentID: NewID(), OwnerID: 7}
	if a.Validate() == nil {
		t.Fatal("positive-issue allowance accepted an intent scope")
	}
	plain, _ := allowanceFixture()
	if plain.Intent != nil || plain.Validate() != nil {
		t.Fatal("positive-issue allowance changed shape")
	}
}

func TestIntentRootRefusesEveryIssueBudgetButKeepsTime(t *testing.T) {
	a, start := intentAllowanceFixture()
	head := strings.Repeat("c", 40)
	before := cloneIntentAllowance(a)
	if out, charged, err := a.ConsumeReview(NewID(), head, false, start); !errors.Is(err, ErrIntentTimeOnly) || charged || !reflect.DeepEqual(out, AttemptAllowance{}) {
		t.Fatalf("intent root charged a review round: %+v %v %v", out, charged, err)
	}
	if out, charged, err := a.ConsumeCorrection(NewID(), start); !errors.Is(err, ErrIntentTimeOnly) || charged || !reflect.DeepEqual(out, AttemptAllowance{}) {
		t.Fatalf("intent root charged a correction cycle: %+v %v %v", out, charged, err)
	}
	if out, charged, err := a.ReserveDeliveryCall(NewID(), start); !errors.Is(err, ErrIntentTimeOnly) || charged || !reflect.DeepEqual(out, AttemptAllowance{}) {
		t.Fatalf("intent root reserved a delivery call: %+v %v %v", out, charged, err)
	}
	if a.ReviewFits(head, true, start) || a.ReviewFits(head, false, start) {
		t.Fatal("intent root fits a review")
	}
	if !reflect.DeepEqual(a, before) {
		t.Fatal("intent refusal mutated the record")
	}
	// Forged positive-issue mixed shape refuses the same way, with no
	// mutation or charge.
	forged, _ := allowanceFixture()
	forged.Intent = &IntentAttemptScope{IntentID: NewID(), OwnerID: 7}
	if forged.Validate() == nil {
		t.Fatal("forged positive-issue intent scope validated")
	}
	if _, _, err := forged.ConsumeReview(NewID(), head, false, start); !errors.Is(err, ErrIntentTimeOnly) {
		t.Fatalf("forged positive-issue intent consumed a review round: %v", err)
	}
	if _, _, err := forged.ConsumeCorrection(NewID(), start); !errors.Is(err, ErrIntentTimeOnly) {
		t.Fatalf("forged positive-issue intent consumed a correction cycle: %v", err)
	}
	if _, _, err := forged.ReserveDeliveryCall(NewID(), start); !errors.Is(err, ErrIntentTimeOnly) {
		t.Fatalf("forged positive-issue intent reserved a delivery call: %v", err)
	}
	if forged.ReviewFits(head, true, start) {
		t.Fatal("forged positive-issue intent fits a review")
	}
	// Active time accrues across pause/resume/checkpoint with the same
	// root and scope; the zero issue sub-budgets never default.
	stopped, err := a.Checkpoint(start.Add(10*time.Minute), false)
	if err != nil || stopped.ActiveSeconds != 600 || stopped.RemainingSeconds(start.Add(time.Hour)) != 45*60-600 {
		t.Fatalf("intent active time did not accrue: %+v %v", stopped, err)
	}
	if stopped.RootAssignment != a.RootAssignment || stopped.Intent == nil || *stopped.Intent != *a.Intent {
		t.Fatalf("checkpoint changed intent root or scope: %+v", stopped)
	}
	if stopped.Limits != (AttemptLimits{ActiveMinutes: 45}) {
		t.Fatalf("checkpoint defaulted intent sub-budgets: %+v", stopped.Limits)
	}
	if stopped.Limits.Validate() == nil {
		t.Fatal("intent zero sub-budgets validate as issue policy")
	}
	if err := stopped.Validate(); err != nil {
		t.Fatalf("checkpointed intent record refused: %v", err)
	}
	resumed, err := stopped.Checkpoint(start.Add(2*time.Hour), true)
	if err != nil || resumed.ActiveSeconds != 600 {
		t.Fatalf("queued intent interval charged: %+v %v", resumed, err)
	}
	replayed, err := resumed.Checkpoint(start.Add(2*time.Hour), true)
	if err != nil || replayed.ActiveSeconds != 600 {
		t.Fatalf("intent checkpoint replay granted time: %+v %v", replayed, err)
	}
	if _, err := resumed.Checkpoint(start, true); !errors.Is(err, ErrAllowanceClock) || resumed.RemainingSeconds(start) != 0 {
		t.Fatal("intent clock regression granted time", err)
	}
	closed := stopped
	closed.Closed = true
	if _, err := closed.Checkpoint(start.Add(11*time.Minute), true); !errors.Is(err, ErrAttemptClosed) {
		t.Fatalf("closed intent root resumed: %v", err)
	}
	short, start := intentAllowanceFixture()
	short.Limits.ActiveMinutes = 1
	if short.RemainingSeconds(start.Add(time.Minute)) != 0 {
		t.Fatal("intent time exhaustion did not zero remaining time")
	}
}
