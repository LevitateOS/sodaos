// Bounded optional lifecycle-action configuration: per-event enablement
// gated by that event's own proved native capability. Disabled is the
// default; enabling without a recorded capability proof is refused by the
// store. Configuration never enrolls credentials and never replaces
// mandatory review.
package factory

import (
	"strings"
	"unicode/utf8"
)

// LifecycleConfig is one event's optional-effect configuration. Targets
// mirror the matching LifecycleAction bounds; an enabled config must
// carry effective targets while a disabled config may omit them.
type LifecycleConfig struct {
	Event        string  `json:"event"`
	Revision     int64   `json:"revision"`
	Enabled      bool    `json:"enabled"`
	ActorID      int64   `json:"actor_id"`
	RepositoryID int64   `json:"repository_id"`
	LabelID      int64   `json:"label_id,omitempty"`
	AssigneeID   int64   `json:"assignee_id,omitempty"`
	ReviewerIDs  []int64 `json:"reviewer_ids,omitempty"`
	Note         string  `json:"note,omitempty"`
}

// Validate checks event, revision, actor/repository and per-event target
// bounds. Capability gating is enforced by the store, not here.
func (c LifecycleConfig) Validate() error {
	if !ValidLifecycleEvent(c.Event) || c.Revision < 0 {
		return ErrLifecycleAction
	}
	if c.ActorID <= 0 || c.RepositoryID <= 0 {
		return ErrLifecycleAction
	}
	if c.LabelID < 0 || c.AssigneeID < 0 {
		return ErrLifecycleAction
	}
	for _, id := range c.ReviewerIDs {
		if id <= 0 {
			return ErrLifecycleAction
		}
	}
	switch c.Event {
	case LifecycleEventIssueCreated:
		return checkIssueCreatedConfig(c)
	case LifecycleEventPRCreated:
		return checkPRCreatedConfig(c)
	case LifecycleEventVerifiedMerge:
		return checkVerifiedMergeConfig(c)
	default:
		return ErrLifecycleAction
	}
}

func checkIssueCreatedConfig(c LifecycleConfig) error {
	if len(c.ReviewerIDs) > 0 || c.Note != "" {
		return ErrLifecycleAction
	}
	if c.Enabled && c.LabelID == 0 && c.AssigneeID == 0 {
		return ErrLifecycleAction
	}
	return nil
}

func checkPRCreatedConfig(c LifecycleConfig) error {
	if len(c.ReviewerIDs) > MaxLifecycleReviewers {
		return ErrLifecycleAction
	}
	seen := map[int64]bool{}
	for _, id := range c.ReviewerIDs {
		if seen[id] {
			return ErrLifecycleAction
		}
		seen[id] = true
	}
	if c.LabelID != 0 || c.AssigneeID != 0 || c.Note != "" {
		return ErrLifecycleAction
	}
	if c.Enabled && len(c.ReviewerIDs) == 0 {
		return ErrLifecycleAction
	}
	return nil
}

func checkVerifiedMergeConfig(c LifecycleConfig) error {
	if c.Note != "" && (strings.TrimSpace(c.Note) == "" || utf8.RuneCountInString(c.Note) > MaxCompletionNoteChars || strings.ContainsRune(c.Note, 0) || !utf8.ValidString(c.Note)) {
		return ErrLifecycleAction
	}
	if c.LabelID != 0 || c.AssigneeID != 0 || len(c.ReviewerIDs) > 0 {
		return ErrLifecycleAction
	}
	if c.Enabled && strings.TrimSpace(c.Note) == "" {
		return ErrLifecycleAction
	}
	return nil
}
