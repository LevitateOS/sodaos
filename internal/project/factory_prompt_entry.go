// Prompt-first investigation entry: pure caller-side validation for admitting
// exactly one investigation per user intent from an existing terminal
// session. This file is pure validation: no I/O, no SQL, no privilege.
//
// A prompt-first entry reuses the existing session's authN/account/project
// selection (Project/Actor/Connection triple); it mints no new authority.
// Admission itself belongs strictly to FW11 (dispatch attempt admission):
// this record is FW11's input shape, and FW11 remains the sole writer of
// admitted attempts. The launcher (host executor call path) rejects every
// entry this validation refuses, plus stale/cancelled intents at launch.
package project

import "errors"

// Prompt-first investigation entry route: the investigate phase only. Any
// other phase routed through this entry is a misuse and refuses.
const PromptEntryPhase = ExecutionInvestigate

// Intent states observed by prompt-first entry. Only an accepted intent can
// open an investigation; draft is not yet authorized, and the terminal
// states are closed (cancelled) for new entries.
const (
	PromptIntentDraft     = "draft"
	PromptIntentAccepted  = "accepted"
	PromptIntentDeclined  = "declined"
	PromptIntentRejected  = "rejected"
	PromptIntentAbandoned = "abandoned"
)

// FactoryPromptEntry is one prompt-first investigation entry request: the
// user's prompt bytes plus the existing-session bindings FW11 admission
// checks before opening the investigate run. Assignment is the canonical
// digest over the exact Prompt bytes (the prompt/record round-trip binds
// the staged prompt to the recorded attempt). Revision is the attempt
// revision: the first investigation opens at revision 1, and a Retry
// advances it; replays repeat the recorded revision with identical bytes.
type FactoryPromptEntry struct {
	Intent      string `json:"intent"`
	Revision    uint64 `json:"revision"`
	IntentState string `json:"intent_state"`
	Project     string `json:"project"`
	Actor       int64  `json:"actor"`
	Connection  string `json:"connection"`
	Role        string `json:"role"`
	Preparation string `json:"preparation"`
	Harness     string `json:"harness"`
	HarnessVers string `json:"harness_version"`
	Phase       string `json:"phase"`
	Prompt      []byte `json:"prompt"`
	Assignment  string `json:"assignment"`
}

// Validate checks the entry shape: intent identity and accepted state,
// existing-session sponsorship, investigation route (phase/role),
// preparation directory reference and harness flags, and the prompt/record
// round-trip (bytes within the investigate bound with a matching digest).
func (e FactoryPromptEntry) Validate() error {
	if !ValidDecisionID(e.Intent) || e.Revision == 0 {
		return errors.New("invalid prompt entry intent or revision")
	}
	if e.IntentState != PromptIntentAccepted {
		return errors.New("prompt entry needs an accepted intent")
	}
	// Existing session sponsorship: the entry reuses the authenticated
	// session's account/project triple; empty or unshaped values select
	// nothing and refuse.
	if !ValidID(e.Project) || e.Actor <= 0 || e.Connection == "" || len(e.Connection) > 128 {
		return errors.New("invalid prompt entry session")
	}
	// Route: prompt-first entry opens investigations only. A wrong phase,
	// role, preparation reference or harness pin is launcher misuse.
	if e.Phase != PromptEntryPhase || !ValidFactoryRole(e.Role) {
		return errors.New("invalid prompt entry route")
	}
	if !ValidPreparationID(e.Preparation) {
		return errors.New("invalid prompt entry directory")
	}
	if !ValidHarnessFamily(e.Harness) || !ValidHarnessVersion(e.HarnessVers) {
		return errors.New("unsupported prompt entry harness")
	}
	if len(e.Prompt) == 0 || len(e.Prompt) > MaxPromptForPhase(e.Phase) {
		return errors.New("invalid prompt entry size")
	}
	if FactoryPromptDigest(e.Prompt) != e.Assignment {
		return errors.New("prompt entry bytes do not match their digest")
	}
	return nil
}

// FactoryPromptRecord is the admitted investigation bound to one intent:
// the revision that opened it and the prompt digest it recorded. Cancelled
// marks a fenced record that admits no further revision.
type FactoryPromptRecord struct {
	Intent     string `json:"intent"`
	Revision   uint64 `json:"revision"`
	Assignment string `json:"assignment"`
	Cancelled  bool   `json:"cancelled"`
}

// AdmitPromptEntry enforces exactly one investigation per intent against the
// recorded admission. A nil record admits the first revision; repeating the
// recorded revision with identical bytes is a replay (no duplicate
// investigation, replay=true); a higher revision supersedes (the old attempt
// stays fenced until FW11 Retry advancement); a lower revision is stale; a
// cancelled record, or an entry for another intent, refuses.
func AdmitPromptEntry(recorded *FactoryPromptRecord, incoming FactoryPromptEntry) (FactoryPromptRecord, bool, error) {
	if err := incoming.Validate(); err != nil {
		return FactoryPromptRecord{}, false, err
	}
	if recorded == nil {
		return FactoryPromptRecord{Intent: incoming.Intent, Revision: incoming.Revision, Assignment: incoming.Assignment}, false, nil
	}
	if recorded.Intent != incoming.Intent {
		return FactoryPromptRecord{}, false, errors.New("prompt entry intent mismatch")
	}
	if recorded.Cancelled {
		return FactoryPromptRecord{}, false, errors.New("prompt entry intent is cancelled")
	}
	switch {
	case incoming.Revision == recorded.Revision:
		if incoming.Assignment != recorded.Assignment {
			return FactoryPromptRecord{}, false, errors.New("prompt entry revision conflict")
		}
		return *recorded, true, nil
	case incoming.Revision > recorded.Revision:
		return FactoryPromptRecord{Intent: incoming.Intent, Revision: incoming.Revision, Assignment: incoming.Assignment}, false, nil
	default:
		return FactoryPromptRecord{}, false, errors.New("stale prompt entry revision")
	}
}
