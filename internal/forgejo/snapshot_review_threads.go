package forgejo

import (
	"strconv"

	extensions "forgejo.org/extension-sdk"
)

// ThreadEvidence is the permission-checked view of one native code-review
// conversation: the anchored root, its location, the original and current
// commits, the resolver and the complete conversation digest.
type ThreadEvidence = extensions.SnapshotThread

// ThreadMemberEvidence is one permission-checked conversation member.
type ThreadMemberEvidence = extensions.SnapshotThreadMember

// ThreadPage carries one bounded conversation list with completeness
// evidence.
type ThreadPage = extensions.SnapshotThreadPage

// validThreadDigest checks the 64-hex conversation digest shape.
func validThreadDigest(digest string) bool {
	if len(digest) != 64 {
		return false
	}
	for _, c := range digest {
		if c >= '0' && c <= '9' || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F' {
			continue
		}
		return false
	}
	return true
}

// validThreadPage checks that the page matches the request: threads in
// strictly ascending root-ID order above the exclusive cursor bound, a
// nonterminal page carrying the cursor at the final returned root, and a
// terminal page carrying an empty cursor. Complete pages must be fully
// evidenced; hidden threads carry only their root locator with no digest,
// location, commits or members. An empty incomplete page is invalid.
func validThreadPage(page *ThreadPage, req SnapshotRequest) error {
	var bound int64
	if req.Cursor != "" {
		parsed, err := strconv.ParseInt(req.Cursor, 10, 64)
		if err != nil || parsed <= 0 {
			return ErrInvalidSnapshot
		}
		bound = parsed
	}
	previous := bound
	for _, item := range page.Items {
		if !decimalID(item.RootID) || !decimalID(item.IssueID) || !decimalID(item.ReviewID) {
			return ErrInvalidSnapshot
		}
		root, err := strconv.ParseInt(item.RootID, 10, 64)
		if err != nil || root <= previous {
			return ErrInvalidSnapshot
		}
		previous = root
		if err := validThread(item); err != nil {
			return err
		}
	}
	if page.Complete {
		if page.NextCursor != "" {
			return ErrInvalidSnapshot
		}
		for _, item := range page.Items {
			if !item.Complete {
				return ErrIncompleteSnapshot
			}
		}
		return nil
	}
	if len(page.Items) == 0 {
		return ErrInvalidSnapshot
	}
	if page.NextCursor != "" && page.NextCursor != page.Items[len(page.Items)-1].RootID {
		return ErrInvalidSnapshot
	}
	return nil
}

func checkHiddenThread(item ThreadEvidence) error {
	if item.Digest != "" || item.Path != "" || item.Side != "" || item.Line != 0 ||
		item.OriginalCommit != "" || item.HeadCommit != "" || len(item.Members) != 0 {
		return ErrInvalidSnapshot
	}
	return nil
}

func checkVisibleThreadShape(item ThreadEvidence) error {
	if item.Path == "" || (item.Side != "old" && item.Side != "new") || item.Line <= 0 {
		return ErrInvalidSnapshot
	}
	if item.OriginalCommit != "" && !fullOID(item.OriginalCommit) {
		return ErrInvalidSnapshot
	}
	if item.HeadCommit != "" && !fullOID(item.HeadCommit) {
		return ErrInvalidSnapshot
	}
	if !validThreadDigest(item.Digest) {
		return ErrInvalidSnapshot
	}
	if item.MemberTotal < 0 || len(item.Members) > item.MemberTotal {
		return ErrInvalidSnapshot
	}
	if item.Complete && len(item.Members) != item.MemberTotal {
		return ErrIncompleteSnapshot
	}
	if item.Resolved != (item.ResolverID != "") {
		return ErrInvalidSnapshot
	}
	if item.ResolverID != "" && !decimalID(item.ResolverID) {
		return ErrInvalidSnapshot
	}
	return nil
}

func checkThreadMembers(item ThreadEvidence) error {
	for _, member := range item.Members {
		if !decimalID(member.ID) || !decimalID(member.IssueID) {
			return ErrInvalidSnapshot
		}
		if !member.Complete {
			return ErrInvalidSnapshot
		}
		if !member.Visible {
			if member.ContentDigest != "" {
				return ErrInvalidSnapshot
			}
			continue
		}
		if member.ContentDigest == "" || member.ContentVersion < 0 {
			return ErrInvalidSnapshot
		}
	}
	return nil
}

// validThread checks one conversation: a visible thread carries its
// location, commits, resolver binding, digest and members; a hidden
// thread is redacted to its root locator.
func validThread(item ThreadEvidence) error {
	if !item.Visible {
		return checkHiddenThread(item)
	}
	if err := checkVisibleThreadShape(item); err != nil {
		return err
	}
	return checkThreadMembers(item)
}

// threadLocation renders one member location in the canonical digest
// encoding shared with the Fountain host: path, side and unsigned line
// joined by vertical bars. The single SDK implementation is the source of
// truth; this alias keeps Soda reconstruction byte-identical.
func threadLocation(path, side string, line int64) string {
	return extensions.RenderThreadLocation(path, side, line)
}
