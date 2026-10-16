package factory

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func reviewAllowanceFixture(t *testing.T) (AttemptAllowance, time.Time, string) {
	t.Helper()
	allowance, now := allowanceFixture()
	return allowance, now, strings.Repeat("e", 40)
}

// TestReviewCounterFirstInspectionSpendsRoundsNotExtraPass proves the T44
// counter leg: the first inspection of a new head consumes one ReviewRound
// while the unchanged-head extra pass stays untouched, and replays of the
// same immutable reviewer assignment never charge twice.
func TestReviewCounterFirstInspectionSpendsRoundsNotExtraPass(t *testing.T) {
	allowance, now, head := reviewAllowanceFixture(t)
	id := NewID()
	next, consumed, err := allowance.ConsumeReview(id, head, false, now)
	if err != nil || !consumed {
		t.Fatalf("first inspection refused: consumed=%v err=%v", consumed, err)
	}
	if len(next.Reviews) != 1 || next.Reviews[0] != id {
		t.Fatalf("first inspection did not charge its round: %+v", next.Reviews)
	}
	if len(next.UnchangedHeadExtra) != 0 {
		t.Fatalf("first inspection spent the extra pass: %+v", next.UnchangedHeadExtra)
	}
	if got, want := next.ReviewedHeads[head], id; got != want {
		t.Fatalf("first inspection did not record its head: %q", got)
	}
	// Replay of the same assignment is identical: no second charge, even
	// after the attempt is otherwise exhausted.
	again, consumed, err := next.ConsumeReview(id, head, false, now)
	if err != nil || consumed {
		t.Fatalf("replay recharged: consumed=%v err=%v", consumed, err)
	}
	if len(again.Reviews) != 1 || len(again.UnchangedHeadExtra) != 0 {
		t.Fatalf("replay changed the ledger: %+v", again)
	}
	if err := again.Validate(); err != nil {
		t.Fatalf("charged allowance fails validation: %v", err)
	}
}

// TestReviewCounterExtraPassAtMostOneAcrossEdits proves the T44 finiteness
// leg: one discretionary extra pass per exact head, and round, model or
// base edits never reset it — a second discretionary inspection of the
// same head refuses while a new head still inspects.
func TestReviewCounterExtraPassAtMostOneAcrossEdits(t *testing.T) {
	allowance, now, head := reviewAllowanceFixture(t)
	first := NewID()
	next, _, err := allowance.ConsumeReview(first, head, false, now)
	if err != nil {
		t.Fatal(err)
	}
	// The one extra pass spends exactly once, however the round, model or
	// base behind the new reviewer assignment changed: the ledger keys by
	// the exact head, never by the assignment, round, model or base.
	extra := NewID()
	next, consumed, err := next.ConsumeReview(extra, head, false, now)
	if err != nil || !consumed {
		t.Fatalf("single extra pass refused: consumed=%v err=%v", consumed, err)
	}
	if got := next.UnchangedHeadExtra[head]; got != extra {
		t.Fatalf("extra pass recorded under the wrong assignment: %q", got)
	}
	for i, retry := range []string{NewID(), NewID()} {
		if _, _, err := next.ConsumeReview(retry, head, false, now); !errors.Is(err, ErrUnchangedHeadExtraExhausted) {
			t.Fatalf("second extra pass %d gave %v", i, err)
		}
	}
	// A new head still inspects: the bound is per head, not per attempt.
	other := strings.Repeat("f", 40)
	next, consumed, err = next.ConsumeReview(NewID(), other, false, now)
	if err != nil || !consumed {
		t.Fatalf("new head refused while another head exhausted its extra: consumed=%v err=%v", consumed, err)
	}
	if len(next.UnchangedHeadExtra) != 1 {
		t.Fatalf("new-head inspection touched the extra ledger: %+v", next.UnchangedHeadExtra)
	}
	if err := next.Validate(); err != nil {
		t.Fatalf("charged allowance fails validation: %v", err)
	}
}

// TestReviewCounterPolicyReviewsStaySeparatelyCaused proves the T44 policy
// leg: policy-required reviews consume ReviewRounds without touching the
// unchanged-head extra pass, and they never exhaust it — not even on an
// already inspected head, and not even when the discretionary extra is
// spent.
func TestReviewCounterPolicyReviewsStaySeparatelyCaused(t *testing.T) {
	allowance, now, head := reviewAllowanceFixture(t)
	next, _, err := allowance.ConsumeReview(NewID(), head, false, now)
	if err != nil {
		t.Fatal(err)
	}
	next, consumed, err := next.ConsumeReview(NewID(), head, false, now)
	if err != nil || !consumed {
		t.Fatalf("discretionary extra refused: consumed=%v err=%v", consumed, err)
	}
	// The discretionary extra is now spent; a policy-required review on
	// the same head still inspects, spending only a ReviewRound.
	next, consumed, err = next.ConsumeReview(NewID(), head, true, now)
	if err != nil || !consumed {
		t.Fatalf("policy review refused on an extra-exhausted head: consumed=%v err=%v", consumed, err)
	}
	if len(next.UnchangedHeadExtra) != 1 {
		t.Fatalf("policy review touched the extra ledger: %+v", next.UnchangedHeadExtra)
	}
	if len(next.Reviews) != 3 {
		t.Fatalf("policy review did not charge its round: %+v", next.Reviews)
	}
	// A policy first inspection records the head without spending extra.
	fresh := strings.Repeat("a", 40)
	next, consumed, err = next.ConsumeReview(NewID(), fresh, true, now)
	if err != nil || !consumed {
		t.Fatalf("policy first inspection refused: consumed=%v err=%v", consumed, err)
	}
	if _, ok := next.ReviewedHeads[fresh]; !ok {
		t.Fatalf("policy first inspection did not record its head: %+v", next.ReviewedHeads)
	}
	if len(next.UnchangedHeadExtra) != 1 {
		t.Fatalf("policy first inspection spent extra: %+v", next.UnchangedHeadExtra)
	}
	if err := next.Validate(); err != nil {
		t.Fatalf("charged allowance fails validation: %v", err)
	}
}

// TestReviewCounterRoundsBoundAndFits proves the idempotent-exhaustion leg:
// ReviewRounds bound every cause including policy, exhaustion replays stay
// free, and ReviewFits predicts ConsumeReview without spending.
func TestReviewCounterRoundsBoundAndFits(t *testing.T) {
	allowance, now, _ := reviewAllowanceFixture(t)
	allowance.Limits.ReviewRounds = 2
	next := allowance
	heads := []string{strings.Repeat("1", 40), strings.Repeat("2", 40)}
	for _, head := range heads {
		var consumed bool
		var err error
		next, consumed, err = next.ConsumeReview(NewID(), head, true, now)
		if err != nil || !consumed {
			t.Fatalf("policy round refused: consumed=%v err=%v", consumed, err)
		}
	}
	if next.ReviewFits(strings.Repeat("3", 40), true, now) {
		t.Fatal("fits reported room past the ReviewRounds bound")
	}
	if _, _, err := next.ConsumeReview(NewID(), strings.Repeat("3", 40), true, now); !errors.Is(err, ErrReviewRoundsExhausted) {
		t.Fatalf("over-round charge gave %v", err)
	}
	// Fits predicts the extra-pass leg too: spent extra does not fit a
	// discretionary re-review but still fits a policy cause.
	roomy, _ := allowanceFixture()
	head := strings.Repeat("4", 40)
	roomy, _, _ = roomy.ConsumeReview(NewID(), head, false, now)
	roomy, _, _ = roomy.ConsumeReview(NewID(), head, false, now)
	if roomy.ReviewFits(head, false, now) {
		t.Fatal("fits reported room for a spent discretionary extra pass")
	}
	if !roomy.ReviewFits(head, true, now) {
		t.Fatal("fits refused a separately-caused policy review on a spent head")
	}
	if !roomy.ReviewFits(strings.Repeat("5", 40), false, now) {
		t.Fatal("fits refused a new head while another head spent its extra")
	}
	// Malformed inputs never fit and never charge.
	if roomy.ReviewFits("not-a-commit", false, now) {
		t.Fatal("fits accepted a malformed head")
	}
	if _, _, err := roomy.ConsumeReview("bad-id", head, false, now); err == nil {
		t.Fatal("malformed reviewer assignment charged")
	}
	if _, _, err := roomy.ConsumeReview(NewID(), "bad-head", false, now); err == nil {
		t.Fatal("malformed head charged")
	}
}
