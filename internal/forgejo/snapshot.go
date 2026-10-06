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

	extensions "forgejo.org/extension-sdk"
)

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
