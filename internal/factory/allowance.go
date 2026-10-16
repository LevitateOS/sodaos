package factory

import (
	"errors"
	"math"
	"time"
)

// AttemptLimits is the owner's finite allowance for one automatic attempt.
// The attempt snapshots it; subsequent policy changes do not replenish it.
// ReviewRounds counts initial and policy-required reviews; MaxChildren caps
// concurrent native task children; DeliveryCalls funds Jev delivery
// judgments, where zero cannot authorize automatic merge.
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
)

// AttemptAllowance retains one automatic attempt across assignments,
// acceptance edits and pauses. An explicit maintainer Retry starts a new
// allowance and preserves this one in history. Active time includes review
// and native waits; callers may pause its clock only after confirming all
// affected runs stopped. Closed prevents automatic reactivation after a
// terminal attempt result. Corrections are charged before launch by the
// correction assignment identity. DeliveryCalls is the funded-Jev call
// subledger under the attempt root: reserved by immutable request identity,
// consumed together with the existing global budget in one store
// transaction, and never refilled by phase changes, reviews or input edits.
type AttemptAllowance struct {
	Limits         AttemptLimits `json:"limits"`
	Corrections    []string      `json:"corrections"`
	DeliveryCalls  []string      `json:"delivery_calls"`
	RootAssignment string        `json:"root_assignment"`
	Repository     int64         `json:"repository,string"`
	Issue          int64         `json:"issue,string"`
	Revision       int64         `json:"revision"`
	ActiveSeconds  int64         `json:"active_seconds"`
	CheckpointUnix int64         `json:"checkpoint_unix"`
	Active         bool          `json:"active"`
	Closed         bool          `json:"closed"`
}

func (a AttemptAllowance) Validate() error {
	if a.Repository <= 0 || a.Issue <= 0 || !ValidID(a.RootAssignment) || a.Revision < 0 || a.ActiveSeconds < 0 || a.CheckpointUnix <= 0 {
		return errors.New("invalid attempt allowance binding")
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
	return nil
}

// ReserveDeliveryCall charges one funded delivery judgment before its
// request sends. Replaying the same request never consumes a second call,
// even after exhaustion. Failure after reservation stays charged: there is
// no unreserve. A zero delivery limit authorizes no call, so automatic
// merge stays unreachable on an unfunded attempt.
func (a AttemptAllowance) ReserveDeliveryCall(id string, now time.Time) (AttemptAllowance, bool, error) {
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

// ConsumeCorrection charges one cycle before its coder can launch. Replaying
// that same assignment never consumes a second cycle, even after exhaustion.
// A replay does not itself authorize launch; the caller checks remaining time.
func (a AttemptAllowance) ConsumeCorrection(id string, now time.Time) (AttemptAllowance, bool, error) {
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
