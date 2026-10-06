// Merge records bind one published PR's exact candidate and verified base
// to its single native conditional merge. The merge advances only on
// attributable native effects behind current authority, exact
// head/base/review/check evidence and a confirmed issue outcome; a
// committed ref alone never finishes the bookkeeping. Confirmed
// completion is the attributable factory completion that releases
// eligible code dependants.
package factory

import (
	"errors"
	"fmt"

	"github.com/levitateos/sodaos/internal/project"
)

// Merge stages. Open work reconciles its recorded operation against
// native state and may still submit; every other stage is terminal for
// the automatic pass. Fenced work keeps an indeterminate native effect
// owned by offline recovery; later passes re-read it but submit nothing
// new.
const (
	MergeOpen      = "open"
	MergeMerged    = "merged"
	MergeFailed    = "failed"
	MergeWithdrawn = "withdrawn"
	MergeFenced    = "fenced"
)

// ValidMergeStage reports whether stage is a known merge stage.
func ValidMergeStage(stage string) bool {
	switch stage {
	case MergeOpen, MergeMerged, MergeFailed, MergeWithdrawn, MergeFenced:
		return true
	default:
		return false
	}
}

// MaxMergeAttempts permits one immutable merge operation. A terminal
// refusal never creates a replacement identity automatically.
const MaxMergeAttempts = 1

// Merge finish reasons. Bounded codes: views expose why a merge
// finished, never the evidence bytes behind it.
const (
	MergeReasonMerged       = "merge_completed"
	MergeReasonRefused      = "native_refused"
	MergeReasonInvalid      = "evidence_invalid"
	MergeReasonWithdrawn    = "merge_withdrawn"
	MergeReasonFenced       = "effect_indeterminate"
	MergeReasonSuperseded   = "acceptance_superseded"
	MergeReasonUnattributed = "effect_unattributed"
)

func validMergeReason(reason string) bool {
	switch reason {
	case MergeReasonMerged, MergeReasonRefused, MergeReasonInvalid,
		MergeReasonWithdrawn, MergeReasonFenced, MergeReasonSuperseded,
		MergeReasonUnattributed:
		return true
	default:
		return false
	}
}

// MergeResolution is the static maintainer-facing resolution for one
// merge reason: what would resolve it. No free text enters resolutions.
func MergeResolution(reason string) string {
	switch reason {
	case MergeReasonMerged:
		return "The exact candidate merged and its completion is confirmed."
	case MergeReasonRefused:
		return "Native refused the merge; inspect the recorded reason and correct the candidate."
	case MergeReasonInvalid:
		return "The merge evidence is stale or incomplete; reassess review and checks on the exact head."
	case MergeReasonWithdrawn:
		return "The merge was withdrawn before its native effect resolved."
	case MergeReasonFenced:
		return "The native merge effect is indeterminate; offline recovery owns it."
	case MergeReasonSuperseded:
		return "The acceptance changed; new work supersedes this merge."
	case MergeReasonUnattributed:
		return "The native merge effect cannot be attributed to its recorded intent."
	default:
		return ""
	}
}

// Merge is one publication's durable merge: the exact bound candidate
// and verified base, the separately persisted conditional operation,
// the fresh review/check observations the attempt bound, and the
// confirmed completion once native bookkeeping proves the outcome.
// Finished work is terminal and keeps its receipts for inspection.
type Merge struct {
	WithdrawRequested bool           `json:"withdraw_requested,omitempty"`
	Operation         MergeOperation `json:"operation"`
	Authority         AuthorityRef   `json:"authority"`
	ID                string         `json:"id"`
	PublicationID     string         `json:"publication_id"`
	AssignmentID      string         `json:"assignment_id"`
	ProjectID         string         `json:"project_id"`
	Role              string         `json:"role"`
	Acceptance        string         `json:"acceptance"`
	HeadRef           string         `json:"head_ref"`
	BaseRef           string         `json:"base_ref"`
	HeadOID           string         `json:"head_oid"`
	BaseOID           string         `json:"base_oid"`
	MergedCommit      string         `json:"merged_commit,omitempty"`
	Stage             string         `json:"stage"`
	Outcome           Outcome        `json:"outcome,omitempty"`
	Reason            string         `json:"reason,omitempty"`
	Repository        int64          `json:"repository,string"`
	Issue             int64          `json:"issue,string"`
	PRNumber          int64          `json:"pr_number"`
	PRID              int64          `json:"pr_id"`
	IssueID           int64          `json:"issue_id"`
	PRAuthorID        int64          `json:"pr_author_id"`
	ReviewerID        int64          `json:"reviewer_id"`
	Revision          int64          `json:"revision"`
	NativeRev         int64          `json:"native_revision"`
	CreatedUnix       int64          `json:"created_unix"`
	ObservedUnix      int64          `json:"observed_unix,omitempty"`
	FinishedUnix      int64          `json:"finished_unix,omitempty"`
	MergedUnix        int64          `json:"merged_unix,omitempty"`
	ClosedUnix        int64          `json:"closed_unix,omitempty"`
}

// Validate rejects malformed merges. Open work carries no outcome;
// finished work carries its outcome, reason and finish time. A merged
// merge links exactly the PR its own merge receipt proved and carries
// the confirmed completion stamps.
func (m Merge) Validate() error {
	if !ValidID(m.ID) || !ValidID(m.PublicationID) || !ValidID(m.AssignmentID) {
		return errors.New("invalid merge identity")
	}
	if !ValidProjectID(m.ProjectID) || m.Role != project.RoleCoder {
		return errors.New("invalid merge address")
	}
	if m.Repository <= 0 || m.Issue <= 0 || m.PRNumber <= 0 || m.PRID <= 0 || m.IssueID <= 0 || m.Revision < 0 {
		return errors.New("invalid merge scope")
	}
	if m.PRAuthorID <= 0 || m.ReviewerID <= 0 || m.ReviewerID == m.PRAuthorID {
		return errors.New("merge review evidence needs its independent reviewer")
	}
	if !project.ValidDecisionID(m.Acceptance) {
		return errors.New("invalid merge acceptance")
	}
	if !ValidTargetBranch(m.HeadRef) || !ValidTargetBranch(m.BaseRef) || m.HeadRef == m.BaseRef {
		return errors.New("invalid merge refs")
	}
	if !ValidCommit(m.HeadOID) || !ValidCommit(m.BaseOID) || m.HeadOID == m.BaseOID {
		return errors.New("merge candidate is not a fresh exact commit")
	}
	if m.MergedCommit != "" && (!ValidCommit(m.MergedCommit) || m.MergedCommit != m.HeadOID) {
		return errors.New("merge realized commit differs from its exact head")
	}
	if m.NativeRev < 0 || (m.NativeRev == 0) != (m.ObservedUnix == 0) {
		return errors.New("invalid merge observation binding")
	}
	if m.ObservedUnix < 0 || m.CreatedUnix <= 0 || m.MergedUnix < 0 || m.ClosedUnix < 0 {
		return errors.New("invalid merge time")
	}
	if err := m.Operation.Validate(); err != nil {
		return err
	}
	if m.Operation.Kind != OpMerge {
		return errors.New("merge operation has the wrong kind")
	}
	if op := m.Operation; op.Work != nil && (op.Work.HeadOID != m.HeadOID || op.Work.BaseOID != m.BaseOID || op.Work.Repository != m.Repository || op.Work.PRNumber != m.PRNumber) {
		return errors.New("merge operation intent differs from its merge")
	}
	if op := m.Operation; op.PRNumber != 0 && (op.Effect != OpEffectCommitted || op.PRNumber != m.PRNumber || op.PRID != m.PRID || op.IssueID != m.IssueID) {
		return errors.New("merge PR is not the recorded merge effect")
	}
	switch m.Stage {
	case MergeOpen:
		if m.Outcome != "" || m.Reason != "" || m.FinishedUnix != 0 || m.MergedCommit != "" || m.MergedUnix != 0 || m.ClosedUnix != 0 {
			return errors.New("open merge carries no outcome")
		}
	case MergeMerged:
		if m.Outcome != Succeeded || m.Reason != MergeReasonMerged || m.FinishedUnix <= 0 {
			return errors.New("merged merge lacks its outcome")
		}
		if m.Operation.Effect != OpEffectCommitted || m.Operation.Completion != OpCompletionComplete {
			return errors.New("merged merge lacks native completion")
		}
		if m.Operation.MergedCommit != m.HeadOID || m.MergedCommit != m.HeadOID || m.MergedUnix <= 0 || m.ClosedUnix <= 0 {
			return errors.New("merged merge lacks its confirmed completion")
		}
		if m.Operation.PRNumber != m.PRNumber || m.Operation.PRID != m.PRID || m.Operation.IssueID != m.IssueID {
			return errors.New("merged PR is not the recorded merge effect")
		}
	case MergeFailed, MergeWithdrawn, MergeFenced:
		if !validMergeReason(m.Reason) || m.FinishedUnix <= 0 {
			return errors.New("finished merge lacks its outcome")
		}
		if m.Reason == MergeReasonMerged {
			return errors.New("unfinished merge cannot claim completed linkage")
		}
		switch m.Stage {
		case MergeFailed:
			if m.Outcome != Failed {
				return errors.New("failed merge lacks its outcome")
			}
		case MergeWithdrawn:
			if m.Outcome != Cancelled {
				return errors.New("withdrawn merge lacks its outcome")
			}
			if op := m.Operation; op.Attempts != 0 && op.Effect != OpEffectCommitted && op.Effect != OpEffectNotCommitted {
				return errors.New("withdrawn merge still has an unresolved native effect")
			}
		case MergeFenced:
			if m.Outcome != NeedsHuman {
				return errors.New("fenced merge lacks its outcome")
			}
		}
	default:
		return errors.New("invalid merge stage")
	}
	return nil
}

// MergeWithdrawal reports ordered local withdrawal separately from
// outstanding native effects. Operation identities remain available for
// recovery.
type MergeWithdrawal struct {
	Merges     []string `json:"merges"`
	Operations []string `json:"operations"`
	Pending    bool     `json:"pending"`
}

// MergeAuthRevision binds one merge's Soda authority reference into the
// opaque native authorization revision: assignment, merge and Soda
// revision, never a credential.
func MergeAuthRevision(assignmentID, mergeID string, revision int64) string {
	return fmt.Sprintf("soda-assignment:%s:merge:%s:revision:%d", assignmentID, mergeID, revision)
}

// MergeTargetChanged reports whether the recorded merge target differs
// from the publication it was created from. Any difference refuses the
// merge instead of merging a moved candidate.
func MergeTargetChanged(m Merge, p Publication) bool {
	return m.Repository != p.Repository || m.Issue != p.Issue || m.PRNumber != p.PRNumber || m.PRID != p.PRID ||
		m.HeadRef == "" || m.BaseRef == "" || m.HeadOID == "" || m.BaseOID == "" ||
		p.PRCreate.HeadRef != m.HeadRef || p.PRCreate.BaseRef != m.BaseRef ||
		p.Candidate != m.HeadOID || p.PRCreate.BaseOID != m.BaseOID
}
