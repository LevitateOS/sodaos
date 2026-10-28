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

// FactoryPromptRecord is the admitted investigation bound to one intent:
// the revision that opened it and the prompt digest it recorded. Cancelled
// marks a fenced record that admits no further revision.
type FactoryPromptRecord struct {
	Intent     string `json:"intent"`
	Revision   uint64 `json:"revision"`
	Assignment string `json:"assignment"`
	Cancelled  bool   `json:"cancelled"`
}
