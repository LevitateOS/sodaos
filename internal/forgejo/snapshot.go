// Package forgejo snapshot reads are the thin Soda caller for permission-checked
// native evidence (FT10, F-read).
//
// A SnapshotRequest selects bounded native families; BracketedRead observes
// the atomic native revision before and after the read and accepts the
// snapshot only when both observations are equal and idle. A changed or busy
// observation, a missing page, or a malformed snapshot refuses; hidden
// records are returned with Visible=false so the consumer blocks work
// without leaking inaccessible content.
//
// The revision is caller bracket evidence, never host content: BracketedRead
// binds the agreed idle revision after the bracket holds. BackgroundSnapshotReader
// is the thin transport over the Fountain SDK. There is no native SQL
// access, no second graph, and no factory approval, acceptance or readiness
// field here.
package forgejo

import (
	"context"
	"strings"

	extensions "forgejo.org/extension-sdk"
)

// PullEvidence is the permission-checked view of one native pull request.
type PullEvidence struct {
	ID           string `json:"id"`
	IssueID      string `json:"issue_id"`
	Number       string `json:"number"`
	HeadRepoID   string `json:"head_repo_id,omitempty"`
	HeadBranch   string `json:"head_branch,omitempty"`
	HeadTip      string `json:"head_tip,omitempty"`
	BaseBranch   string `json:"base_branch,omitempty"`
	MergeBase    string `json:"merge_base,omitempty"`
	HasMerged    bool   `json:"has_merged,omitempty"`
	MergedCommit string `json:"merged_commit,omitempty"`
	MergerID     string `json:"merger_id,omitempty"`
	MergedUnix   int64  `json:"merged_unix,omitempty"`
	MaintainerEd bool   `json:"allow_maintainer_edit,omitempty"`
	Flow         string `json:"flow,omitempty"`
	Status       string `json:"status,omitempty"`
	Visible      bool   `json:"visible"`
	HiddenReason string `json:"hidden_reason,omitempty"`
	Complete     bool   `json:"complete"`
}

// ReviewEvidence is the permission-checked view of one native review.
type ReviewEvidence struct {
	ID            string `json:"id"`
	IssueID       string `json:"issue_id"`
	Type          string `json:"type,omitempty"`
	ReviewerID    string `json:"reviewer_id,omitempty"`
	CommitID      string `json:"commit_id,omitempty"`
	Official      bool   `json:"official,omitempty"`
	Stale         bool   `json:"stale,omitempty"`
	Dismissed     bool   `json:"dismissed,omitempty"`
	ContentDigest string `json:"content_digest,omitempty"`
	CreatedUnix   int64  `json:"created_unix,omitempty"`
	UpdatedUnix   int64  `json:"updated_unix,omitempty"`
	Visible       bool   `json:"visible"`
	HiddenReason  string `json:"hidden_reason,omitempty"`
	Complete      bool   `json:"complete"`
}

// ReviewPage carries one bounded review list with completeness evidence.
type ReviewPage struct {
	IssueID  string           `json:"issue_id"`
	Items    []ReviewEvidence `json:"items"`
	Total    int              `json:"total"`
	Complete bool             `json:"complete"`
}

// CheckEvidence is the permission-checked view of one native commit status.
type CheckEvidence struct {
	ID           string `json:"id"`
	Index        int64  `json:"index,omitempty"`
	SHA          string `json:"sha"`
	Context      string `json:"context,omitempty"`
	State        string `json:"state,omitempty"`
	CreatorID    string `json:"creator_id,omitempty"`
	CreatedUnix  int64  `json:"created_unix,omitempty"`
	UpdatedUnix  int64  `json:"updated_unix,omitempty"`
	Visible      bool   `json:"visible"`
	HiddenReason string `json:"hidden_reason,omitempty"`
	Complete     bool   `json:"complete"`
}

// CheckSet carries one bounded check list for an exact commit.
type CheckSet struct {
	SHA      string          `json:"sha"`
	Items    []CheckEvidence `json:"items"`
	Total    int             `json:"total"`
	Complete bool            `json:"complete"`
}

// RefEvidence is the permission-checked view of one native branch tip.
type RefEvidence struct {
	Ref          string `json:"ref"`
	OID          string `json:"oid,omitempty"`
	Exists       bool   `json:"exists,omitempty"`
	Visible      bool   `json:"visible"`
	HiddenReason string `json:"hidden_reason,omitempty"`
	Complete     bool   `json:"complete"`
}

// NativeSnapshot is one revision-bound evidence set. Revision is the idle
// native revision both bracket observations agreed on.
type NativeSnapshot struct {
	Revision     int64           `json:"revision"`
	RepositoryID string          `json:"repository_id"`
	Issue        *IssueEvidence  `json:"issue,omitempty"`
	Comments     *CommentPage    `json:"comments,omitempty"`
	Dependencies *DependencyPage `json:"dependencies,omitempty"`
	Pull         *PullEvidence   `json:"pull,omitempty"`
	Reviews      *ReviewPage     `json:"reviews,omitempty"`
	Checks       *CheckSet       `json:"checks,omitempty"`
	Refs         []RefEvidence   `json:"refs,omitempty"`
}

// SnapshotReader performs revision observations and snapshot reads over the
// background channel. Snapshot reads present the bound native actor
// credential privately; revision observations use owning-installation
// admission. BackgroundSnapshotReader is the thin SDK transport; tests
// supply fakes.
type SnapshotReader interface {
	ReadNativeRevision(ctx context.Context) (extensions.NativeRevisionObservation, error)
	ReadSnapshot(ctx context.Context, credential extensions.CredentialFile, req SnapshotRequest) (NativeSnapshot, error)
}

// BracketedRead validates the request, brackets one snapshot read between
// two revision observations, and accepts the result only when both
// observations are equal and idle and every requested family is present and
// complete. Hidden records are returned with Visible=false for the
// consumer's authorization decision; they are not silently dropped. The
// agreed idle revision is bound by the caller after the bracket holds; any
// revision the transport returned is replaced, never trusted.
func BracketedRead(ctx context.Context, reader SnapshotReader, credential extensions.CredentialFile, req SnapshotRequest) (NativeSnapshot, error) {
	if reader == nil {
		return NativeSnapshot{}, ErrInvalidSnapshot
	}
	if err := ValidateRequest(req); err != nil {
		return NativeSnapshot{}, err
	}
	before, err := reader.ReadNativeRevision(ctx)
	if err != nil {
		return NativeSnapshot{}, err
	}
	if before.Revision < 1 {
		return NativeSnapshot{}, ErrInvalidSnapshot
	}
	if !before.Idle {
		return NativeSnapshot{}, ErrNativeBusy
	}
	snapshot, err := reader.ReadSnapshot(ctx, credential, req)
	if err != nil {
		return NativeSnapshot{}, err
	}
	after, err := reader.ReadNativeRevision(ctx)
	if err != nil {
		return NativeSnapshot{}, err
	}
	if !after.Idle {
		return NativeSnapshot{}, ErrNativeBusy
	}
	if after.Revision != before.Revision {
		return NativeSnapshot{}, ErrStaleSnapshot
	}
	snapshot.Revision = before.Revision
	if err := ValidateSnapshot(req, snapshot, before.Revision); err != nil {
		return NativeSnapshot{}, err
	}
	return snapshot, nil
}

// ValidateSnapshot checks that the snapshot matches the request and the
// bracketed revision with every requested family present and complete.
// Visible records must carry their digests; hidden records must be redacted
// to locators.
func ValidateSnapshot(req SnapshotRequest, snapshot NativeSnapshot, revision int64) error {
	if revision < 1 || snapshot.Revision != revision {
		return ErrStaleSnapshot
	}
	if snapshot.RepositoryID != req.RepositoryID {
		return ErrInvalidSnapshot
	}
	for _, family := range req.Families {
		switch family {
		case FamilyIssue:
			if snapshot.Issue == nil || !snapshot.Issue.Complete {
				return ErrIncompleteSnapshot
			}
			if err := validIssue(snapshot.Issue, req); err != nil {
				return err
			}
		case FamilyComments:
			if snapshot.Comments == nil || !snapshot.Comments.Complete {
				return ErrIncompleteSnapshot
			}
			if err := validCommentPage(snapshot.Comments); err != nil {
				return err
			}
		case FamilyDependencies:
			if snapshot.Dependencies == nil || !snapshot.Dependencies.Complete {
				return ErrIncompleteSnapshot
			}
			if err := validDependencyPage(snapshot.Dependencies); err != nil {
				return err
			}
		case FamilyPull:
			if snapshot.Pull == nil || !snapshot.Pull.Complete {
				return ErrIncompleteSnapshot
			}
			if err := validPull(snapshot.Pull, req); err != nil {
				return err
			}
		case FamilyReviews:
			if snapshot.Reviews == nil || !snapshot.Reviews.Complete {
				return ErrIncompleteSnapshot
			}
			if err := validReviewPage(snapshot.Reviews); err != nil {
				return err
			}
		case FamilyChecks:
			if snapshot.Checks == nil || !snapshot.Checks.Complete {
				return ErrIncompleteSnapshot
			}
			if err := validCheckSet(snapshot.Checks, req); err != nil {
				return err
			}
		case FamilyRefs:
			if err := validRefs(snapshot.Refs, req); err != nil {
				return err
			}
		default:
			return ErrInvalidSnapshot
		}
	}
	return nil
}

func validPull(pull *PullEvidence, req SnapshotRequest) error {
	if !decimalID(pull.ID) || !decimalID(pull.Number) {
		return ErrInvalidSnapshot
	}
	if req.PullNumber != "" && pull.Number != req.PullNumber {
		return ErrInvalidSnapshot
	}
	if !pull.Visible {
		return nil
	}
	if pull.HeadTip != "" && !fullOID(pull.HeadTip) {
		return ErrInvalidSnapshot
	}
	if !pull.HasMerged && pull.HeadTip == "" {
		return ErrIncompleteSnapshot
	}
	return nil
}

func validReviewPage(page *ReviewPage) error {
	if page.Total < 0 || len(page.Items) != page.Total {
		return ErrIncompleteSnapshot
	}
	for _, item := range page.Items {
		if !decimalID(item.ID) || !item.Complete {
			return ErrInvalidSnapshot
		}
		if !item.Visible {
			continue
		}
		if item.Type == "" || !fullOID(item.CommitID) || item.ContentDigest == "" {
			return ErrInvalidSnapshot
		}
	}
	return nil
}

func validCheckSet(set *CheckSet, req SnapshotRequest) error {
	if !fullOID(set.SHA) || req.SHA != "" && !strings.EqualFold(set.SHA, req.SHA) {
		return ErrInvalidSnapshot
	}
	if set.Total < 0 || len(set.Items) != set.Total {
		return ErrIncompleteSnapshot
	}
	for _, item := range set.Items {
		if !decimalID(item.ID) || !item.Complete {
			return ErrInvalidSnapshot
		}
		if !item.Visible {
			continue
		}
		if !strings.EqualFold(item.SHA, set.SHA) || item.State == "" {
			return ErrInvalidSnapshot
		}
	}
	return nil
}

func validRefs(refs []RefEvidence, req SnapshotRequest) error {
	if len(refs) < len(req.Refs) {
		return ErrIncompleteSnapshot
	}
	seen := make(map[string]bool, len(refs))
	for _, ref := range refs {
		if !fullBranchRef(ref.Ref) || !ref.Complete {
			return ErrInvalidSnapshot
		}
		if !ref.Visible {
			if ref.OID != "" || ref.Exists {
				return ErrInvalidSnapshot
			}
		} else if ref.Exists != (ref.OID != "") || ref.OID != "" && !fullOID(ref.OID) {
			return ErrInvalidSnapshot
		}
		seen[ref.Ref] = true
	}
	for _, want := range req.Refs {
		if !seen[want] {
			return ErrIncompleteSnapshot
		}
	}
	return nil
}

// HasHiddenEvidence reports whether any requested evidence is hidden. The
// consumer refuses authorization while this is true; the read itself stays
// informative rather than failing.
func HasHiddenEvidence(req SnapshotRequest, snapshot NativeSnapshot) bool {
	for _, family := range req.Families {
		switch family {
		case FamilyIssue:
			if snapshot.Issue != nil && !snapshot.Issue.Visible {
				return true
			}
		case FamilyComments:
			if snapshot.Comments != nil {
				for _, item := range snapshot.Comments.Items {
					if !item.Visible {
						return true
					}
				}
			}
		case FamilyDependencies:
			if snapshot.Dependencies != nil {
				for _, item := range snapshot.Dependencies.Items {
					if !item.Visible {
						return true
					}
				}
			}
		case FamilyPull:
			if snapshot.Pull != nil && !snapshot.Pull.Visible {
				return true
			}
		case FamilyReviews:
			if snapshot.Reviews != nil {
				for _, item := range snapshot.Reviews.Items {
					if !item.Visible {
						return true
					}
				}
			}
		case FamilyChecks:
			if snapshot.Checks != nil {
				for _, item := range snapshot.Checks.Items {
					if !item.Visible {
						return true
					}
				}
			}
		case FamilyRefs:
			for _, ref := range snapshot.Refs {
				if !ref.Visible {
					return true
				}
			}
		}
	}
	return false
}
