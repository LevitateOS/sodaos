package factory

import (
	"errors"
	"math"
	"time"
)

// AttemptLimits is the owner's finite allowance for one automatic attempt.
// The attempt snapshots it; subsequent policy changes do not replenish it.
// ReviewRounds counts every dispatched independent review: first
// inspections and policy-required reviews alike; MaxChildren caps
// concurrent native task children; DeliveryCalls funds Jev delivery
// judgments, where zero cannot authorize automatic merge. Discretionary
// same-head re-reviews additionally spend the single unchanged-head extra
// pass per head (D08).
type AttemptLimits struct {
	ActiveMinutes    int `json:"active_minutes"`
	CorrectionCycles int `json:"correction_cycles"`
	ReviewRounds     int `json:"review_rounds"`
	MaxChildren      int `json:"max_children"`
	DeliveryCalls    int `json:"delivery_calls"`
}

// Frozen D04 allowance defaults.
const (
	DefaultReviewRounds  = 6
	DefaultMaxChildren   = 3
	DefaultDeliveryCalls = 8
)

func DefaultAttemptLimits() AttemptLimits {
	return AttemptLimits{
		ActiveMinutes: 120, CorrectionCycles: 3,
		ReviewRounds: DefaultReviewRounds, MaxChildren: DefaultMaxChildren, DeliveryCalls: DefaultDeliveryCalls,
	}
}

func (l AttemptLimits) Validate() error {
	if l.ActiveMinutes < 1 || int64(l.ActiveMinutes) > math.MaxInt64/int64(time.Minute) || l.CorrectionCycles < 0 {
		return errors.New("attempt limits must be finite and nonnegative with positive active time")
	}
	if l.ReviewRounds < 1 || l.ReviewRounds > 16 {
		return errors.New("attempt review rounds outside their bound")
	}
	if l.MaxChildren < 1 || l.MaxChildren > 4 {
		return errors.New("attempt child concurrency outside its bound")
	}
	if l.DeliveryCalls < 0 || l.DeliveryCalls > 16 {
		return errors.New("attempt delivery calls outside their bound")
	}
	return nil
}

// EffectiveAttemptLimits resolves omission only when recording a policy or the
// initial attempt. Review rounds and child concurrency accept no zero, so a
// zero there is an unambiguous omission filled with the frozen default.
// Active minutes keep their explicit-zero refusal, and correction and
// delivery calls keep an explicit zero: zero delivery calls is the valid
// unfunded posture that can never authorize automatic merge. Never call this
// on a running record: only explicit authorized Retry starts a new root.
func EffectiveAttemptLimits(limits *AttemptLimits) AttemptLimits {
	if limits == nil {
		return DefaultAttemptLimits()
	}
	out := *limits
	if out.ReviewRounds == 0 {
		out.ReviewRounds = DefaultReviewRounds
	}
	if out.MaxChildren == 0 {
		out.MaxChildren = DefaultMaxChildren
	}
	return out
}

var (
	ErrAttemptTimeExhausted = errors.New("attempt active-time allowance exhausted")
	ErrAttemptClosed        = errors.New("attempt is closed; explicit retry is required")
	ErrCorrectionExhausted  = errors.New("attempt correction allowance exhausted")
	ErrDeliveryExhausted    = errors.New("attempt delivery-call allowance exhausted")
	ErrAllowanceClock       = errors.New("attempt allowance clock moved backwards")
	// ErrReviewRoundsExhausted reports that no further independent review
	// fits the attempt's ReviewRounds: first inspections, policy-required
	// reviews and discretionary re-reviews all consume one round (D08).
	ErrReviewRoundsExhausted = errors.New("attempt review-round allowance exhausted")
	// ErrUnchangedHeadExtraExhausted reports that the single
	// discretionary extra pass on this exact head is spent: one
	// independent review has already inspected the head, and the one
	// extra inspection is gone. Round, model or base edits never reset
	// it; only a new head or a policy-required cause inspects again
	// without spending it (D08).
	ErrUnchangedHeadExtraExhausted = errors.New("unchanged-head discretionary extra pass exhausted")
	// ErrCompletionTimeOnly reports that a completion-root allowance carries
	// active time only: issue review, correction, delivery and child budget
	// can never be consumed on it, and AttemptLimits review/child defaults
	// never fill its zero sub-budgets.
	ErrCompletionTimeOnly = errors.New("completion root carries active time only")
)

// AttemptAllowance retains one automatic attempt across assignments,
// acceptance edits and pauses. An explicit maintainer Retry starts a new
// allowance and preserves this one in history. Active time includes review
// and native waits; callers may pause its clock only after confirming all
// affected runs stopped. Closed prevents automatic reactivation after a
// terminal attempt result. Corrections are charged before launch by the
// correction assignment identity. Reviews are charged at dispatch by the
// immutable reviewer assignment identity: every dispatched independent
// review consumes one ReviewRound, and a discretionary re-review of an
// already inspected head additionally spends that head's single extra
// pass (ReviewedHeads/UnchangedHeadExtra, D08). DeliveryCalls is the
// funded-Jev call subledger under the attempt root: reserved by immutable
// request identity, consumed together with the existing global budget in
// one store transaction, and never refilled by phase changes, reviews or
// input edits.
type AttemptAllowance struct {
	Limits        AttemptLimits `json:"limits"`
	Corrections   []string      `json:"corrections"`
	DeliveryCalls []string      `json:"delivery_calls"`
	Reviews       []string      `json:"reviews,omitempty"`
	// ReviewedHeads binds each independently inspected head to the
	// reviewer assignment that first inspected it. Absent means the head
	// is new: its first inspection consumes ReviewRounds but never the
	// unchanged-head extra pass.
	ReviewedHeads map[string]string `json:"reviewed_heads,omitempty"`
	// UnchangedHeadExtra binds each head whose single discretionary extra
	// pass is spent to the reviewer assignment that spent it. At most one
	// entry per head, keyed by the exact head so round, model or base
	// edits never reset it. Policy-required reviews never write here:
	// they stay separately caused.
	UnchangedHeadExtra map[string]string `json:"unchanged_head_extra,omitempty"`
	// Completion binds a time-only reassessment record to its verified
	// completion (D13/FW32), reusing the canonical CompletionSubject. It is
	// required exactly when Issue is zero and forbidden on positive issues.
	Completion     *CompletionSubject `json:"completion,omitempty"`
	RootAssignment string             `json:"root_assignment"`
	Repository     int64              `json:"repository,string"`
	Issue          int64              `json:"issue,string"`
	Revision       int64              `json:"revision"`
	ActiveSeconds  int64              `json:"active_seconds"`
	CheckpointUnix int64              `json:"checkpoint_unix"`
	Active         bool               `json:"active"`
	Closed         bool               `json:"closed"`
}

func (a AttemptAllowance) Validate() error {
	if a.Repository <= 0 || a.Issue < 0 || !ValidID(a.RootAssignment) || a.Revision < 0 || a.ActiveSeconds < 0 || a.CheckpointUnix <= 0 {
		return errors.New("invalid attempt allowance binding")
	}
	if a.Issue == 0 {
		return a.validateCompletionTimeOnly()
	}
	if a.Completion != nil {
		return errors.New("issue attempt allowance carries no completion subject")
	}
	if err := a.Limits.Validate(); err != nil {
		return err
	}
	if len(a.Corrections) > a.Limits.CorrectionCycles {
		return errors.New("attempt correction consumption exceeds its allowance")
	}
	seen := make(map[string]bool, len(a.Corrections))
	for _, id := range a.Corrections {
		if !ValidID(id) || seen[id] {
			return errors.New("invalid correction assignment identity")
		}
		seen[id] = true
	}
	if len(a.DeliveryCalls) > a.Limits.DeliveryCalls {
		return errors.New("attempt delivery consumption exceeds its allowance")
	}
	seenDelivery := make(map[string]bool, len(a.DeliveryCalls))
	for _, id := range a.DeliveryCalls {
		if !ValidID(id) || seenDelivery[id] {
			return errors.New("invalid delivery request identity")
		}
		seenDelivery[id] = true
	}
	if len(a.Reviews) > a.Limits.ReviewRounds {
		return errors.New("attempt review consumption exceeds its allowance")
	}
	seenReview := make(map[string]bool, len(a.Reviews))
	for _, id := range a.Reviews {
		if !ValidID(id) || seenReview[id] {
			return errors.New("invalid reviewer assignment identity")
		}
		seenReview[id] = true
	}
	if len(a.ReviewedHeads) > a.Limits.ReviewRounds || len(a.UnchangedHeadExtra) > a.Limits.ReviewRounds {
		return errors.New("attempt review head ledger exceeds its allowance")
	}
	for head, id := range a.ReviewedHeads {
		if !ValidCommit(head) || !ValidID(id) || !seenReview[id] {
			return errors.New("invalid reviewed head binding")
		}
	}
	for head, id := range a.UnchangedHeadExtra {
		if !ValidCommit(head) || !ValidID(id) || !seenReview[id] {
			return errors.New("invalid unchanged-head extra pass binding")
		}
		if _, first := a.ReviewedHeads[head]; !first {
			return errors.New("extra pass spent on an uninspected head")
		}
	}
	return nil
}

// validateCompletionTimeOnly validates the explicitly completion-bound
// time-only branch: Issue zero with the canonical completion subject,
// ActiveMinutes as the admitted EffectiveReassessmentMinutes (1..120),
// and every issue sub-budget (correction, review, child, delivery) zero
// with no recorded consumption. AttemptLimits.Validate stays unchanged
// for ordinary issue policy and never runs here: a zero review or child
// budget is the valid time-only posture, not an omission to default.
func (a AttemptAllowance) validateCompletionTimeOnly() error {
	if a.Completion == nil {
		return errors.New("completion attempt allowance lacks its completion subject")
	}
	if !ValidID(a.Completion.CompletionID) || !ValidID(a.Completion.CycleID) {
		return errors.New("invalid completion subject identity")
	}
	if !ValidDigest(a.Completion.InventoryDigest) {
		return errors.New("invalid completion subject inventory digest")
	}
	if a.Limits.ActiveMinutes != EffectiveReassessmentMinutes(a.Limits.ActiveMinutes) ||
		a.Limits.ActiveMinutes < MinReassessmentMinutes || a.Limits.ActiveMinutes > MaxReassessmentMinutes {
		return errors.New("completion active time is not its admitted reassessment minutes")
	}
	if a.Limits.CorrectionCycles != 0 || a.Limits.ReviewRounds != 0 ||
		a.Limits.MaxChildren != 0 || a.Limits.DeliveryCalls != 0 {
		return errors.New("completion allowance carries only active time")
	}
	if len(a.Corrections) != 0 || len(a.DeliveryCalls) != 0 || len(a.Reviews) != 0 ||
		len(a.ReviewedHeads) != 0 || len(a.UnchangedHeadExtra) != 0 {
		return errors.New("completion allowance carries only active time")
	}
	return nil
}

// ReserveDeliveryCall charges one funded delivery judgment before its
// request sends. Replaying the same request never consumes a second call,
// even after exhaustion. Failure after reservation stays charged: there is
// no unreserve. A zero delivery limit authorizes no call, so automatic
// merge stays unreachable on an unfunded attempt.
func (a AttemptAllowance) ReserveDeliveryCall(id string, now time.Time) (AttemptAllowance, bool, error) {
	if a.Issue == 0 || a.Completion != nil {
		return AttemptAllowance{}, false, ErrCompletionTimeOnly
	}
	if !ValidID(id) {
		return AttemptAllowance{}, false, errors.New("invalid delivery request identity")
	}
	next, err := a.Checkpoint(now, a.Active)
	if err != nil {
		return AttemptAllowance{}, false, err
	}
	for _, seen := range a.DeliveryCalls {
		if seen == id {
			return next, false, nil
		}
	}
	if next.RemainingSeconds(now) == 0 {
		return AttemptAllowance{}, false, ErrAttemptTimeExhausted
	}
	if len(next.DeliveryCalls) >= next.Limits.DeliveryCalls {
		return AttemptAllowance{}, false, ErrDeliveryExhausted
	}
	next.DeliveryCalls = append(append([]string(nil), next.DeliveryCalls...), id)
	return next, true, nil
}

// RemainingSeconds includes the current active interval without changing the
// record. A clock regression grants no remaining time until it is resolved.
func (a AttemptAllowance) RemainingSeconds(now time.Time) int64 {
	if a.Validate() != nil || now.Unix() < a.CheckpointUnix {
		return 0
	}
	remaining := int64(a.Limits.ActiveMinutes)*60 - a.ActiveSeconds
	if a.Active {
		remaining -= now.Unix() - a.CheckpointUnix
	}
	if remaining < 0 {
		return 0
	}
	return remaining
}

// Checkpoint closes the previous interval and records the next accounting mode.
// An identical replay has no effect; time is never subtracted or replenished.
func (a AttemptAllowance) Checkpoint(now time.Time, active bool) (AttemptAllowance, error) {
	if err := a.Validate(); err != nil {
		return AttemptAllowance{}, err
	}
	if now.Unix() < a.CheckpointUnix {
		return AttemptAllowance{}, ErrAllowanceClock
	}
	if a.Closed && active {
		return AttemptAllowance{}, ErrAttemptClosed
	}
	if a.Active {
		elapsed := now.Unix() - a.CheckpointUnix
		if elapsed > math.MaxInt64-a.ActiveSeconds {
			return AttemptAllowance{}, errors.New("attempt active time overflow")
		}
		a.ActiveSeconds += elapsed
	}
	a.CheckpointUnix, a.Active = now.Unix(), active
	return a, nil
}

// ReviewFits reports whether one more independent review of head fits the
// remaining allowance without charging it. A first inspection of a new
// head and a policy-required review need only a remaining ReviewRound; a
// discretionary re-review of an already inspected head additionally needs
// that head's single extra pass. Reads spend nothing.
func (a AttemptAllowance) ReviewFits(head string, policy bool, now time.Time) bool {
	if a.Issue == 0 || a.Completion != nil {
		return false
	}
	if !ValidID(a.RootAssignment) || !ValidCommit(head) {
		return false
	}
	if err := a.Limits.Validate(); err != nil {
		return false
	}
	if a.Validate() != nil || now.Unix() < a.CheckpointUnix || a.RemainingSeconds(now) == 0 {
		return false
	}
	if len(a.Reviews) >= a.Limits.ReviewRounds {
		return false
	}
	if _, seen := a.ReviewedHeads[head]; seen && !policy {
		if _, spent := a.UnchangedHeadExtra[head]; spent {
			return false
		}
	}
	return true
}

// ConsumeReview charges one independent reviewer dispatch by its immutable
// reviewer assignment identity, at dispatch and never on result or retry.
// Replaying the same assignment never consumes a second round or pass,
// even after exhaustion. The first inspection of a new head consumes one
// ReviewRound but never the unchanged-head extra pass; a discretionary
// re-review of an inspected head consumes one ReviewRound plus that
// head's single extra pass (at most one across round, model or base
// edits, keyed by the exact head); a policy-required review consumes one
// ReviewRound and stays separately caused, never touching the extra pass.
// A replay does not itself authorize dispatch; the caller checks remaining
// time. Only explicit authorized Retry starts a new root.
func (a AttemptAllowance) ConsumeReview(id, head string, policy bool, now time.Time) (AttemptAllowance, bool, error) {
	if a.Issue == 0 || a.Completion != nil {
		return AttemptAllowance{}, false, ErrCompletionTimeOnly
	}
	if !ValidID(id) {
		return AttemptAllowance{}, false, errors.New("invalid reviewer assignment identity")
	}
	if !ValidCommit(head) {
		return AttemptAllowance{}, false, errors.New("invalid review head")
	}
	next, err := a.Checkpoint(now, a.Active)
	if err != nil {
		return AttemptAllowance{}, false, err
	}
	for _, seen := range a.Reviews {
		if seen == id {
			return next, false, nil
		}
	}
	if next.RemainingSeconds(now) == 0 {
		return AttemptAllowance{}, false, ErrAttemptTimeExhausted
	}
	if len(next.Reviews) >= next.Limits.ReviewRounds {
		return AttemptAllowance{}, false, ErrReviewRoundsExhausted
	}
	_, seen := next.ReviewedHeads[head]
	if !seen {
		if next.ReviewedHeads == nil {
			next.ReviewedHeads = make(map[string]string)
		}
		next.ReviewedHeads[head] = id
	} else if !policy {
		if _, spent := next.UnchangedHeadExtra[head]; spent {
			return AttemptAllowance{}, false, ErrUnchangedHeadExtraExhausted
		}
		if next.UnchangedHeadExtra == nil {
			next.UnchangedHeadExtra = make(map[string]string)
		}
		next.UnchangedHeadExtra[head] = id
	}
	next.Reviews = append(append([]string(nil), next.Reviews...), id)
	return next, true, nil
}

// ConsumeCorrection charges one cycle before its coder can launch. Replaying
// that same assignment never consumes a second cycle, even after exhaustion.
// A replay does not itself authorize launch; the caller checks remaining time.
func (a AttemptAllowance) ConsumeCorrection(id string, now time.Time) (AttemptAllowance, bool, error) {
	if a.Issue == 0 || a.Completion != nil {
		return AttemptAllowance{}, false, ErrCompletionTimeOnly
	}
	if !ValidID(id) {
		return AttemptAllowance{}, false, errors.New("invalid correction assignment identity")
	}
	next, err := a.Checkpoint(now, a.Active)
	if err != nil {
		return AttemptAllowance{}, false, err
	}
	for _, seen := range a.Corrections {
		if seen == id {
			return next, false, nil
		}
	}
	if next.RemainingSeconds(now) == 0 {
		return AttemptAllowance{}, false, ErrAttemptTimeExhausted
	}
	if len(next.Corrections) >= next.Limits.CorrectionCycles {
		return AttemptAllowance{}, false, ErrCorrectionExhausted
	}
	next.Corrections = append(append([]string(nil), next.Corrections...), id)
	return next, true, nil
}
