// Lifecycle intervention records: sticky pause/cancel, remaining-allowance
// resume/retry decisions and fixed takeover. These are pure domain records;
// the coordinator orders withdrawal before stopping, and later operation
// tasks connect native cancellation to the same withdrawal path.
package factory

import (
	"errors"
	"strings"
)

// Repository control actions admitted on one repository's dispatch gate.
const (
	// ActionPause closes dispatch and stops outstanding runs. It stays
	// latched until an explicit resume reopens the gate.
	ActionPause = "pause"
	// ActionResume reopens a paused gate after every recorded run settles
	// and every grant revalidates. It launches nothing by itself.
	ActionResume = "resume"
)

// Run control actions admitted on one recorded run.
const (
	// ActionStop retires one run through the same stop path as the
	// operator command. A stopped run identity never runs again.
	ActionStop = "stop"
	// ActionRetry records a queued explicit retry after the prior run is
	// reconciled. No launcher consumes it yet; it stays queued.
	ActionRetry = "retry"
	// ActionTakeover copies a reconciled run's retained work into the
	// admitted member's own checkout. It never transfers credentials.
	ActionTakeover = "takeover"
)

// ValidRepositoryAction reports whether action is a repository control.
func ValidRepositoryAction(action string) bool {
	return action == ActionPause || action == ActionResume
}

// ValidRunAction reports whether action is a run control.
func ValidRunAction(action string) bool {
	return action == ActionStop || action == ActionRetry || action == ActionTakeover
}

// Withdrawal causes owned by lifecycle controls. Grant withdrawals keep
// their own causes; these name the control that closed dispatch.
const (
	CauseControlPaused = "control_paused"
	CauseProjectStop   = "project_stop"
)

// RunStopOutcome is one run's effect status inside a pause or project-stop
// receipt. The action (dispatch closed) and each native effect (run
// confirmed or still uncertain) stay separate facts.
type RunStopOutcome struct {
	ID        string `json:"id"`
	Outcome   string `json:"outcome,omitempty"`
	Reason    string `json:"reason,omitempty"`
	Confirmed bool   `json:"confirmed"`
	Uncertain bool   `json:"uncertain"`
}

// PauseReceipt is the durable outcome of one pause command: the withdrawal
// that closed dispatch first, then every outstanding run's stop outcome.
// Uncertain runs stay fenced; they never appear cancelled.
type PauseReceipt struct {
	Withdrawal Withdrawal       `json:"withdrawal"`
	Runs       []RunStopOutcome `json:"runs"`
	CommandID  string           `json:"command_id"`
	Paused     bool             `json:"paused"`
}

func (r PauseReceipt) Validate() error {
	if !ValidID(r.CommandID) || !r.Paused {
		return errors.New("invalid pause receipt")
	}
	if err := r.Withdrawal.Validate(); err != nil {
		return err
	}
	if r.Runs == nil {
		return errors.New("pause receipt must list stopped runs")
	}
	for _, outcome := range r.Runs {
		if !ValidID(outcome.ID) || len(outcome.Reason) > 256 || len(outcome.Outcome) > 32 {
			return errors.New("invalid pause run outcome")
		}
		if outcome.Confirmed == outcome.Uncertain {
			return errors.New("pause run outcome must confirm exactly one effect state")
		}
	}
	return nil
}

// ResumeReceipt is the durable outcome of one resume command: the reopened
// gate revision and the visible effective authority afterwards. Queued work
// stays queued; resume launches nothing.
type ResumeReceipt struct {
	Effective EffectiveAuthority `json:"effective"`
	CommandID string             `json:"command_id"`
	Revision  int64              `json:"revision"`
	Reopened  bool               `json:"reopened"`
}

func (r ResumeReceipt) Validate() error {
	if !ValidID(r.CommandID) || !r.Reopened || r.Revision < 0 {
		return errors.New("invalid resume receipt")
	}
	return nil
}

// RetryDecision is the durable outcome of one retry command. The prior run
// is reconciled and current authority revalidated, but remaining allowances
// cannot be established without usage records, so the retry stays queued
// instead of launching. A later scheduler consumes it; nothing else may.
type RetryDecision struct {
	Prior     string `json:"prior_run"`
	CommandID string `json:"command_id"`
	Reason    string `json:"reason"`
	Queued    bool   `json:"queued"`
}

func (d RetryDecision) Validate() error {
	if !ValidID(d.CommandID) || !ValidID(d.Prior) || !d.Queued {
		return errors.New("invalid retry decision")
	}
	if d.Reason == "" || len(d.Reason) > 256 {
		return errors.New("invalid retry queue reason")
	}
	return nil
}

// TakeoverRecord binds one reconciled run to the admitted member's
// independent checkout destination. The destination is derived by the host,
// never supplied by the caller; provider homes and role Git configuration
// are excluded from the copy.
type TakeoverRecord struct {
	Run       string `json:"run"`
	Member    string `json:"member"`
	Project   string `json:"project"`
	Dest      string `json:"destination"`
	CommandID string `json:"command_id"`
	Copied    string `json:"copied"`
}

func (r TakeoverRecord) Validate() error {
	if !ValidID(r.Run) || !ValidID(r.CommandID) || !ValidProjectID(r.Project) {
		return errors.New("invalid takeover identity")
	}
	if r.Member == "" || len(r.Member) > 32 || r.Member != strings.TrimSpace(r.Member) {
		return errors.New("invalid takeover member")
	}
	if r.Dest == "" || len(r.Dest) > 256 || !strings.HasPrefix(r.Dest, "/home/"+r.Member+"/") {
		return errors.New("invalid takeover destination")
	}
	return nil
}

// StartVerification reports one coordinated Project start: the host
// lifecycle result plus proof that no old run revived. Revived names runs
// with a live lease or live native boundary; unverified names runs whose
// state could not be established, including runs the host or broker never
// recorded. Leases stay closed, grants and readiness are untouched, and
// the maintenance hold keeps whatever state it had.
type StartVerification struct {
	Revived    []string `json:"revived"`
	Unverified []string `json:"unverified"`
	Hold       bool     `json:"hold"`
	Started    bool     `json:"started"`
}

func (v StartVerification) Validate() error {
	if !v.Started {
		return errors.New("invalid start verification")
	}
	for _, id := range v.Revived {
		if !ValidID(id) {
			return errors.New("invalid revived run identity")
		}
	}
	for _, id := range v.Unverified {
		if !ValidID(id) {
			return errors.New("invalid unverified run identity")
		}
	}
	return nil
}
