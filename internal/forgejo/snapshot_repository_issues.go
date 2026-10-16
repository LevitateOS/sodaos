package forgejo

import (
	"strconv"

	extensions "forgejo.org/extension-sdk"
)

// RepositoryIssueItem is one enumerated open non-PR issue locator. The
// page carries no title, body or other content: exact bodies travel
// through the existing individual issue/dependency families under the same
// stable revision.
type RepositoryIssueItem = extensions.SnapshotRepositoryIssueItem

// RepositoryIssuesPage is one bounded open-issue enumeration page in
// ascending issue-index order. Read-only shape only: no acceptance,
// readiness or other factory policy is expressed here.
type RepositoryIssuesPage = extensions.SnapshotRepositoryIssuesPage

// validRepositoryIssuesPage checks that the page matches the request with
// every locator well-formed: strictly ascending canonical indexes, all
// above the exclusive cursor bound when one was supplied. A terminal page
// carries an empty cursor; a nonterminal page carries nonempty items with
// the cursor at the final returned index. An empty incomplete page is
// invalid. Hidden items are returned for the consumer's authorization
// decision, not refused here.
func validRepositoryIssuesPage(page *RepositoryIssuesPage, req SnapshotRequest) error {
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
		if !decimalID(item.IssueID) || !decimalID(item.IssueIndex) {
			return ErrInvalidSnapshot
		}
		index, err := strconv.ParseInt(item.IssueIndex, 10, 64)
		if err != nil || index <= previous {
			return ErrInvalidSnapshot
		}
		previous = index
	}
	if page.Complete {
		if page.NextCursor != "" {
			return ErrInvalidSnapshot
		}
		return nil
	}
	if len(page.Items) == 0 {
		return ErrInvalidSnapshot
	}
	if page.NextCursor != page.Items[len(page.Items)-1].IssueIndex {
		return ErrInvalidSnapshot
	}
	return nil
}
