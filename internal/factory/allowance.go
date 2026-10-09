package factory

import (
	"errors"
	"math"
	"time"
)

// AttemptLimits is the owner's finite allowance for one automatic attempt.
// The attempt snapshots it; subsequent policy changes do not replenish it.
type AttemptLimits struct {
	ActiveMinutes    int `json:"active_minutes"`
	CorrectionCycles int `json:"correction_cycles"`
}

func DefaultAttemptLimits() AttemptLimits {
	return AttemptLimits{ActiveMinutes: 120, CorrectionCycles: 3}
}

func (l AttemptLimits) Validate() error {
	if l.ActiveMinutes < 1 || int64(l.ActiveMinutes) > math.MaxInt64/int64(time.Minute) || l.CorrectionCycles < 0 {
		return errors.New("attempt limits must be finite and nonnegative with positive active time")
	}
	return nil
}

// EffectiveAttemptLimits resolves omission only when recording a policy or the
// initial attempt. A recorded allowance always contains concrete limits.
func EffectiveAttemptLimits(limits *AttemptLimits) AttemptLimits {
	if limits == nil {
		return DefaultAttemptLimits()
	}
	return *limits
}

var (
	ErrAttemptTimeExhausted = errors.New("attempt active-time allowance exhausted")
	ErrAttemptClosed        = errors.New("attempt is closed; explicit retry is required")
	ErrCorrectionExhausted  = errors.New("attempt correction allowance exhausted")
	ErrAllowanceClock       = errors.New("attempt allowance clock moved backwards")
)

// AttemptAllowance retains one automatic attempt across assignments,
// acceptance edits and pauses. An explicit maintainer Retry starts a new
// allowance and preserves this one in history. Active time includes review
// and native waits; callers may pause its clock only after confirming all
// affected runs stopped. Closed prevents automatic reactivation after a
// terminal attempt result. Corrections are charged before launch by the
// correction assignment identity.
type AttemptAllowance struct {
	Limits         AttemptLimits `json:"limits"`
	Corrections    []string      `json:"corrections"`
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
	return nil
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
