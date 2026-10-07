package forgejo

import extensions "forgejo.org/extension-sdk"

// LifecycleEvent is one title/lifecycle transition.
type LifecycleEvent = extensions.SnapshotLifecycleEvent

// CreationProvenance binds verified creation evidence. Verified requires
// visibility plus the native first-created flag; cached text alone never
// verifies provenance.
type CreationProvenance = extensions.SnapshotCreationProvenance

// IssueEvidence is the permission-checked view of one native issue. Hidden
// issues preserve only their locator IDs with Visible=false.
type IssueEvidence = extensions.SnapshotIssue

// CommentEvidence is the permission-checked view of one native comment.
type CommentEvidence = extensions.SnapshotComment

// CommentPage carries one bounded comment list with completeness evidence.
type CommentPage = extensions.SnapshotCommentPage

// DependencyEvidence is one edge occurrence. OccurrenceID distinguishes
// removal plus readdition of the same edge; hidden occurrences redact the
// target so an inaccessible prerequisite blocks without leaking it.
type DependencyEvidence = extensions.SnapshotDependency

// DependencyPage carries one bounded edge list with completeness evidence.
type DependencyPage = extensions.SnapshotDependencyPage

func validIssue(issue *IssueEvidence, req SnapshotRequest) error {
	if !decimalID(issue.ID) || !decimalID(issue.Index) {
		return ErrInvalidSnapshot
	}
	if req.IssueIndex != "" && issue.Index != req.IssueIndex {
		return ErrInvalidSnapshot
	}
	if !issue.Visible {
		if issue.Title != "" || issue.Content != "" || issue.TitleDigest != "" || issue.ContentDigest != "" {
			return ErrInvalidSnapshot
		}
		return nil
	}
	if issue.TitleDigest == "" || issue.ContentDigest == "" {
		return ErrInvalidSnapshot
	}
	if issue.TitleDigest != ContentDigest(issue.Title) || issue.ContentDigest != ContentDigest(issue.Content) {
		return ErrInvalidSnapshot
	}
	return nil
}

func validCommentPage(page *CommentPage) error {
	if page.Total < 0 || len(page.Items) != page.Total {
		return ErrIncompleteSnapshot
	}
	for _, item := range page.Items {
		if !decimalID(item.ID) || !item.Complete {
			return ErrInvalidSnapshot
		}
		if !item.Visible {
			if item.Content != "" || item.ContentDigest != "" {
				return ErrInvalidSnapshot
			}
			continue
		}
		if item.ContentDigest == "" || item.ContentDigest != ContentDigest(item.Content) {
			return ErrInvalidSnapshot
		}
	}
	return nil
}

func validDependencyPage(page *DependencyPage) error {
	if page.Total < 0 || len(page.Items) != page.Total {
		return ErrIncompleteSnapshot
	}
	for _, item := range page.Items {
		if !decimalID(item.OccurrenceID) || !item.Complete {
			return ErrInvalidSnapshot
		}
		if !item.Visible && item.DependencyID != "" {
			return ErrInvalidSnapshot
		}
		if item.Visible && !decimalID(item.DependencyID) {
			return ErrInvalidSnapshot
		}
	}
	return nil
}
