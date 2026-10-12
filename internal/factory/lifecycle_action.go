// Bounded optional native lifecycle actions: pure domain policy for the
// three UX-L effects. Every optional effect is disabled by default; the
// complete Factory default workflow never depends on these records.
package factory

import (
	"errors"
	"strings"
	"unicode/utf8"
)

// Optional lifecycle events admitted as effect triggers.
const (
	LifecycleEventIssueCreated  = "issue_created"
	LifecycleEventPRCreated     = "pr_created"
	LifecycleEventVerifiedMerge = "verified_merge"
)

// Optional lifecycle actions, one per event.
const (
	LifecycleActionIssueLabelAssignee = "issue_label_assignee"
	LifecycleActionPRReviewerRequest  = "pr_reviewer_request"
	LifecycleActionCompletionNote     = "original_completion_note"
)

// Bounds for the three optional effects.
const (
	// MaxLifecycleReviewers caps human reviewer requests per PR/event.
	MaxLifecycleReviewers = 3
	// MaxCompletionNoteChars caps one completion note per event.
	MaxCompletionNoteChars = 1000
	// MaxLifecycleAttempts is one initial attempt plus one retry. The
	// retry is admitted only after a definitive no-effect outcome.
	MaxLifecycleAttempts = 2
)

// Per-target result states for one optional action.
const (
	LifecycleResultPending   = "pending"
	LifecycleResultDelivered = "delivered"
	LifecycleResultFailed    = "failed"
	LifecycleResultUncertain = "uncertain"
	LifecycleResultDenied    = "denied"
)

// LifecycleActionError describes a rejected action decision.
var ErrLifecycleAction = errors.New("invalid lifecycle action")

// LifecycleAction is one durable optional-effect record: the event that
// triggered it, its exact actor/targets, finite attempt budget and one
// result per target. Native operation identities live with their target
// result so replay and reconciliation stay per target.
type LifecycleAction struct {
	ID           string            `json:"id"`
	Event        string            `json:"event"`
	Action       string            `json:"action"`
	ActorID      int64             `json:"actor_id"`
	RepositoryID int64             `json:"repository_id"`
	IssueID      int64             `json:"issue_id"`
	LabelID      int64             `json:"label_id,omitempty"`
	AssigneeID   int64             `json:"assignee_id,omitempty"`
	ReviewerIDs  []int64           `json:"reviewer_ids,omitempty"`
	Note         string            `json:"note,omitempty"`
	AuthRevision string            `json:"auth_revision"`
	MergeRef     string            `json:"merge_ref,omitempty"`
	Closed       bool              `json:"closed,omitempty"`
	Attempts     int               `json:"attempts"`
	Revoked      bool              `json:"revoked"`
	Results      map[string]string `json:"results,omitempty"`
	Operations   map[string]string `json:"operations,omitempty"`
}

// ValidLifecycleEvent reports whether event admits optional effects.
func ValidLifecycleEvent(event string) bool {
	return event == LifecycleEventIssueCreated || event == LifecycleEventPRCreated || event == LifecycleEventVerifiedMerge
}

// EventAction maps each admitted event to its single optional action.
func EventAction(event string) string {
	switch event {
	case LifecycleEventIssueCreated:
		return LifecycleActionIssueLabelAssignee
	case LifecycleEventPRCreated:
		return LifecycleActionPRReviewerRequest
	case LifecycleEventVerifiedMerge:
		return LifecycleActionCompletionNote
	default:
		return ""
	}
}

// Lifecycle native operation kinds, one per admitted event. These strings
// must equal the extension SDK's OperationKindLifecycleIssue/Review/Note;
// the SDK is the canonical owner.
const (
	LifecycleKindIssue  = "issue.lifecycle.apply"
	LifecycleKindReview = "pull_request.review.request"
	LifecycleKindNote   = "issue.comment.publish"
)

// LifecycleEventKind maps each admitted event to its native operation
// kind. Unknown events map to "".
func LifecycleEventKind(event string) string {
	switch event {
	case LifecycleEventIssueCreated:
		return LifecycleKindIssue
	case LifecycleEventPRCreated:
		return LifecycleKindReview
	case LifecycleEventVerifiedMerge:
		return LifecycleKindNote
	default:
		return ""
	}
}

// LifecycleHandshake is one authenticated native capability answer: the
// current installation, a stable native operations generation, and the
// supported operation kinds. Unknown kinds are ignored by the registrar;
// only the three lifecycle kinds can register markers.
type LifecycleHandshake struct {
	Installation string   `json:"installation"`
	Generation   string   `json:"generation"`
	Kinds        []string `json:"kinds"`
}

// ValidHandshake reports whether h carries an attributable answer. Empty
// kinds are valid and mean no lifecycle operation is available.
func ValidHandshake(h LifecycleHandshake) bool {
	return h.Installation != "" && h.Generation != ""
}

// Validate checks identity, event/action pairing, bounds and targets.
func (a LifecycleAction) Validate() error {
	if a.ID == "" || !ValidLifecycleEvent(a.Event) || EventAction(a.Event) != a.Action {
		return ErrLifecycleAction
	}
	if a.ActorID <= 0 || a.RepositoryID <= 0 || a.IssueID <= 0 || a.AuthRevision == "" {
		return ErrLifecycleAction
	}
	if a.Attempts < 0 || a.Attempts > MaxLifecycleAttempts {
		return ErrLifecycleAction
	}
	for _, state := range a.Results {
		switch state {
		case LifecycleResultPending, LifecycleResultDelivered, LifecycleResultFailed, LifecycleResultUncertain, LifecycleResultDenied:
		default:
			return ErrLifecycleAction
		}
	}
	switch a.Action {
	case LifecycleActionIssueLabelAssignee:
		if a.LabelID < 0 || a.AssigneeID < 0 || (a.LabelID == 0 && a.AssigneeID == 0) {
			return ErrLifecycleAction
		}
		if len(a.ReviewerIDs) > 0 || a.Note != "" {
			return ErrLifecycleAction
		}
	case LifecycleActionPRReviewerRequest:
		if len(a.ReviewerIDs) == 0 || len(a.ReviewerIDs) > MaxLifecycleReviewers {
			return ErrLifecycleAction
		}
		seen := map[int64]bool{}
		for _, id := range a.ReviewerIDs {
			if id <= 0 || seen[id] {
				return ErrLifecycleAction
			}
			seen[id] = true
		}
		if a.LabelID != 0 || a.AssigneeID != 0 || a.Note != "" {
			return ErrLifecycleAction
		}
	case LifecycleActionCompletionNote:
		if strings.TrimSpace(a.Note) == "" || utf8.RuneCountInString(a.Note) > MaxCompletionNoteChars || strings.ContainsRune(a.Note, 0) || !utf8.ValidString(a.Note) {
			return ErrLifecycleAction
		}
		if a.MergeRef == "" || !a.Closed {
			return ErrLifecycleAction
		}
		if a.LabelID != 0 || a.AssigneeID != 0 || len(a.ReviewerIDs) > 0 {
			return ErrLifecycleAction
		}
	default:
		return ErrLifecycleAction
	}
	return nil
}

// Targets lists the per-target result keys for this action.
func (a LifecycleAction) Targets() []string {
	switch a.Action {
	case LifecycleActionIssueLabelAssignee:
		var out []string
		if a.LabelID > 0 {
			out = append(out, "label")
		}
		if a.AssigneeID > 0 {
			out = append(out, "assignee")
		}
		return out
	case LifecycleActionPRReviewerRequest:
		out := make([]string, 0, len(a.ReviewerIDs))
		for _, id := range a.ReviewerIDs {
			out = append(out, "reviewer:"+itoa(id))
		}
		return out
	case LifecycleActionCompletionNote:
		return []string{"note"}
	default:
		return nil
	}
}

// Applicable reports whether the action may attempt its effect now.
// Completion notes additionally require verified merge AND original
// closure; revocation stops every further effect.
func (a LifecycleAction) Applicable() bool {
	if a.Revoked || a.Validate() != nil {
		return false
	}
	return true
}

// CanAttempt reports whether target may start attempt number n (1-based).
// The first attempt needs a pending target; the single retry needs a
// definitive no-effect (failed) result, never an uncertain one.
func (a LifecycleAction) CanAttempt(target string, n int) bool {
	if !a.Applicable() || n < 1 || n > MaxLifecycleAttempts || n != a.Attempts+1 {
		return false
	}
	state := a.Results[target]
	if n == 1 {
		return state == "" || state == LifecycleResultPending
	}
	return state == LifecycleResultFailed
}

// RecordAttempt returns the action with attempt n started for target.
func (a LifecycleAction) RecordAttempt(target, operationID string) LifecycleAction {
	if a.Results == nil {
		a.Results = map[string]string{}
	}
	if a.Operations == nil {
		a.Operations = map[string]string{}
	}
	a.Attempts++
	a.Results[target] = LifecycleResultUncertain
	a.Operations[target] = operationID
	return a
}

// RecordResult returns the action with target settled to a terminal state.
// Uncertain outcomes stay uncertain until authoritative lookup reconciles
// them; only delivered, failed or denied settle a target.
func (a LifecycleAction) RecordResult(target, state string) LifecycleAction {
	if a.Results == nil {
		a.Results = map[string]string{}
	}
	switch state {
	case LifecycleResultDelivered, LifecycleResultFailed, LifecycleResultDenied:
		a.Results[target] = state
	default:
		a.Results[target] = LifecycleResultUncertain
	}
	return a
}

// Settled reports whether every target reached a terminal state.
func (a LifecycleAction) Settled() bool {
	targets := a.Targets()
	if len(targets) == 0 {
		return false
	}
	for _, t := range targets {
		switch a.Results[t] {
		case LifecycleResultDelivered, LifecycleResultFailed, LifecycleResultDenied:
		default:
			return false
		}
	}
	return true
}

func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

// Native denial reasons: outcomes that stop further attempts rather
// than admitting the single retry. Authority loss covers withdrawn
// permission and actor/credential mismatch; ordered cancellation
// covers revocation races admitted before submit.
const (
	LifecycleReasonAuthorityLost         = "authority_lost"
	LifecycleReasonCancelledBeforeSubmit = "cancelled_before_submit"
	LifecycleReasonCancelledBeforeAdmit  = "cancelled_before_admission"
)

// LifecycleDenyReason reports whether a native reason code denies the
// action instead of failing it.
func LifecycleDenyReason(reason string) bool {
	switch reason {
	case LifecycleReasonAuthorityLost, LifecycleReasonCancelledBeforeSubmit, LifecycleReasonCancelledBeforeAdmit:
		return true
	default:
		return false
	}
}

// RecordActionAttempt returns the action with one attempt started across
// every target under a single operation identity. One native operation
// covers all targets of an attempt (label plus assignee, all reviewers,
// one note), so the finite attempt budget counts submits, not targets.
func (a LifecycleAction) RecordActionAttempt(operationID string) LifecycleAction {
	if a.Results == nil {
		a.Results = map[string]string{}
	}
	if a.Operations == nil {
		a.Operations = map[string]string{}
	}
	a.Attempts++
	for _, target := range a.Targets() {
		a.Results[target] = LifecycleResultUncertain
		a.Operations[target] = operationID
	}
	return a
}

// RecordActionResult returns the action with every target settled to a
// terminal state. Uncertain outcomes stay uncertain until authoritative
// lookup reconciles them; only delivered, failed or denied settle.
func (a LifecycleAction) RecordActionResult(state string) LifecycleAction {
	if a.Results == nil {
		a.Results = map[string]string{}
	}
	for _, target := range a.Targets() {
		switch state {
		case LifecycleResultDelivered, LifecycleResultFailed, LifecycleResultDenied:
			a.Results[target] = state
		default:
			a.Results[target] = LifecycleResultUncertain
		}
	}
	return a
}

// LifecycleWork is one native lifecycle submit: the durable operation
// identity, the enrolled actor, the exact target and the current
// authorization facts. IssueIndex carries the original issue index for
// completion notes only; the executor resolves and revalidates the
// closed target before submit and every other event leaves it zero.
type LifecycleWork struct {
	OperationID  string
	Event        string
	ActorID      int64
	RepositoryID int64
	IssueID      int64
	IssueIndex   int64
	LabelID      int64
	AssigneeID   int64
	ReviewerIDs  []int64
	Note         string
	AuthRevision string
	MergeRef     string
	Closed       bool
	NativeRev    int64
	NotAfter     int64
}
