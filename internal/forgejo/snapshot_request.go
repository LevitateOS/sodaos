package forgejo

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strconv"
	"strings"

	extensions "forgejo.org/extension-sdk"
)

// Snapshot families select bounded native evidence. Unknown families refuse.
type SnapshotFamily string

const (
	FamilyRepositoryIssues SnapshotFamily = "repository_issues"
	FamilyIssue            SnapshotFamily = "issue"
	FamilyComments         SnapshotFamily = "comments"
	FamilyDependencies     SnapshotFamily = "dependencies"
	FamilyPull             SnapshotFamily = "pull"
	FamilyReviews          SnapshotFamily = "reviews"
	FamilyReviewThreads    SnapshotFamily = "review_threads"
	FamilyMergeEligibility SnapshotFamily = "merge_eligibility"
	FamilyChecks           SnapshotFamily = "checks"
	FamilyRefs             SnapshotFamily = "refs"
	FamilyAncestry         SnapshotFamily = "ancestry"
)

func validFamily(family SnapshotFamily) bool {
	switch family {
	case FamilyRepositoryIssues, FamilyIssue, FamilyComments, FamilyDependencies, FamilyPull, FamilyReviews,
		FamilyReviewThreads, FamilyMergeEligibility,
		FamilyChecks, FamilyRefs, FamilyAncestry:
		return true
	default:
		return false
	}
}

// Snapshot bounds keep reads bounded and attributable.
const (
	SnapshotPageLimit   = 50
	SnapshotIDListLimit = 50
	SnapshotRefLimit    = 8
	SnapshotCursorLimit = 4096
)

var (
	// ErrInvalidSnapshot rejects malformed requests or snapshots rather than
	// silently broadening or defaulting them.
	ErrInvalidSnapshot = errors.New("invalid native snapshot")
	// ErrNativeBusy reports a non-idle revision observation around the read.
	ErrNativeBusy = errors.New("native mutation reservation is busy")
	// ErrStaleSnapshot reports an intervening native change between the two
	// revision observations.
	ErrStaleSnapshot = errors.New("native revision changed during snapshot read")
	// ErrIncompleteSnapshot reports a missing page or incomplete section.
	ErrIncompleteSnapshot = errors.New("native snapshot is incomplete")
)

// SnapshotRequest selects bounded native evidence. Native integer IDs use
// decimal strings; refs are full branch refs; SHAs are full object IDs.
type SnapshotRequest struct {
	RepositoryID string                              `json:"repository_id"`
	Families     []SnapshotFamily                    `json:"families"`
	IssueIndex   string                              `json:"issue_index,omitempty"`
	PullNumber   string                              `json:"pull_number,omitempty"`
	CommentIDs   []string                            `json:"comment_ids,omitempty"`
	SHA          string                              `json:"sha,omitempty"`
	Refs         []string                            `json:"refs,omitempty"`
	Limit        int                                 `json:"limit,omitempty"`
	Cursor       string                              `json:"cursor,omitempty"`
	Ancestry     *extensions.SnapshotAncestryRequest `json:"ancestry,omitempty"`
}

func decimalID(id string) bool {
	if id == "" || id[0] == '0' || len(id) > 20 {
		return false
	}
	for _, c := range id {
		if c < '0' || c > '9' {
			return false
		}
	}
	_, err := strconv.ParseInt(id, 10, 64)
	return err == nil
}

func fullOID(oid string) bool {
	if len(oid) != 40 && len(oid) != 64 {
		return false
	}
	for _, c := range oid {
		if c >= '0' && c <= '9' || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F' {
			continue
		}
		return false
	}
	return true
}

func fullBranchRef(ref string) bool {
	if !strings.HasPrefix(ref, "refs/heads/") || len(ref) > 512 {
		return false
	}
	name := strings.TrimSpace(ref)
	if name != ref || strings.ContainsAny(ref, " ~^:?*\\") || strings.Contains(ref, "..") {
		return false
	}
	return len(strings.TrimPrefix(ref, "refs/heads/")) > 0
}

func checkSnapshotFamilies(req SnapshotRequest, seen map[SnapshotFamily]bool) error {
	if len(req.Families) == 0 || len(req.Families) > 8 {
		return ErrInvalidSnapshot
	}
	for _, family := range req.Families {
		if !validFamily(family) || seen[family] {
			return ErrInvalidSnapshot
		}
		seen[family] = true
	}
	return nil
}

func checkSnapshotIssueIDs(req SnapshotRequest) error {
	if req.IssueIndex != "" && !decimalID(req.IssueIndex) {
		return ErrInvalidSnapshot
	}
	if req.PullNumber != "" && !decimalID(req.PullNumber) {
		return ErrInvalidSnapshot
	}
	if len(req.CommentIDs) > SnapshotIDListLimit {
		return ErrInvalidSnapshot
	}
	for _, id := range req.CommentIDs {
		if !decimalID(id) {
			return ErrInvalidSnapshot
		}
	}
	return nil
}

func checkSnapshotRevisions(req SnapshotRequest) error {
	if req.SHA != "" && !fullOID(req.SHA) {
		return ErrInvalidSnapshot
	}
	if len(req.Refs) > SnapshotRefLimit {
		return ErrInvalidSnapshot
	}
	for _, ref := range req.Refs {
		if !fullBranchRef(ref) {
			return ErrInvalidSnapshot
		}
	}
	return nil
}

func checkSnapshotPaging(req SnapshotRequest, seen map[SnapshotFamily]bool) error {
	if req.Limit < 0 || req.Limit > SnapshotPageLimit || len(req.Cursor) > SnapshotCursorLimit {
		return ErrInvalidSnapshot
	}
	if req.Cursor != "" && !decimalID(req.Cursor) {
		return ErrInvalidSnapshot
	}
	if seen[FamilyRefs] && len(req.Refs) == 0 {
		return ErrInvalidSnapshot
	}
	return nil
}

func checkSnapshotLocators(req SnapshotRequest, seen map[SnapshotFamily]bool) error {
	if !decimalID(req.RepositoryID) {
		return ErrInvalidSnapshot
	}
	if err := checkSnapshotFamilies(req, seen); err != nil {
		return err
	}
	if err := checkSnapshotIssueIDs(req); err != nil {
		return err
	}
	if err := checkSnapshotRevisions(req); err != nil {
		return err
	}
	return checkSnapshotPaging(req, seen)
}

// ValidateRequest bounds the requested families and locators.
func ValidateRequest(req SnapshotRequest) error {
	seen := make(map[SnapshotFamily]bool, len(req.Families))
	if err := checkSnapshotLocators(req, seen); err != nil {
		return err
	}
	// repository_issues enumerates open non-PR issues of exactly
	// RepositoryID. It is requested alone with only Limit/Cursor
	// selectors: the cursor is an exclusive issue-index lower bound
	// (empty starts enumeration). No issue/pull/ref selector may
	// accompany it. Read-only shape only; no acceptance, readiness or
	// other factory policy is expressed here.
	if seen[FamilyRepositoryIssues] {
		if len(req.Families) != 1 || req.IssueIndex != "" || req.PullNumber != "" ||
			len(req.CommentIDs) > 0 || req.SHA != "" || len(req.Refs) > 0 || req.Ancestry != nil {
			return ErrInvalidSnapshot
		}
	}
	return checkSnapshotFamilyShape(req, seen)
}

func checkSnapshotAncestry(req SnapshotRequest, seen map[SnapshotFamily]bool) error {
	if seen[FamilyAncestry] != (req.Ancestry != nil) {
		return ErrInvalidSnapshot
	}
	if req.Ancestry != nil && (!seen[FamilyRefs] || !fullOID(req.Ancestry.AncestorOID) || !fullOID(req.Ancestry.DescendantOID)) {
		return ErrInvalidSnapshot
	}
	return nil
}

func checkSnapshotSelectors(req SnapshotRequest, seen map[SnapshotFamily]bool) error {
	if seen[FamilyChecks] && req.SHA == "" {
		return ErrInvalidSnapshot
	}
	if seen[FamilyIssue] && req.IssueIndex == "" {
		return ErrInvalidSnapshot
	}
	if seen[FamilyDependencies] && req.IssueIndex == "" {
		return ErrInvalidSnapshot
	}
	if seen[FamilyPull] && req.PullNumber == "" {
		return ErrInvalidSnapshot
	}
	if seen[FamilyComments] {
		byIssue := req.IssueIndex != ""
		byIDs := len(req.CommentIDs) > 0
		if byIssue == byIDs {
			return ErrInvalidSnapshot
		}
	}
	if seen[FamilyReviews] {
		byIssue := req.IssueIndex != ""
		byPull := req.PullNumber != ""
		if byIssue == byPull {
			return ErrInvalidSnapshot
		}
	}
	// review_threads enumerates native code-review conversations on one
	// pull request, selected either by its issue or by its pull number,
	// never both and never neither. merge_eligibility reads the
	// actor-bound merge readiness of one pull request, so it requires
	// the pull selector; it combines with other families under the same
	// revision bracket. Read-only shape only; no acceptance, readiness or
	// other factory policy is expressed here.
	if seen[FamilyReviewThreads] {
		byIssue := req.IssueIndex != ""
		byPull := req.PullNumber != ""
		if byIssue == byPull {
			return ErrInvalidSnapshot
		}
	}
	if seen[FamilyMergeEligibility] && req.PullNumber == "" {
		return ErrInvalidSnapshot
	}
	return nil
}

func checkSnapshotFamilyShape(req SnapshotRequest, seen map[SnapshotFamily]bool) error {
	if err := checkSnapshotAncestry(req, seen); err != nil {
		return err
	}
	return checkSnapshotSelectors(req, seen)
}

// ContentDigest binds exact native text bytes with the same SHA-256 hex the
// native conversion uses, so Soda can compare digests without trusting
// cached text.
func ContentDigest(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:])
}
