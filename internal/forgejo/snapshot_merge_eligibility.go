package forgejo

import (
	extensions "forgejo.org/extension-sdk"
)

// MergeEligibility is the permission-checked actor-bound merge readiness
// of one pull request: the bound tips, the eligibility verdict with its
// first typed native blocker, and the official rejecting review IDs
// behind a requested-changes blocker. The read performs no mutation and
// grants no mutation authority.
type MergeEligibility = extensions.SnapshotMergeEligibility

// validMergeBlocker checks the bounded blocker vocabulary. Empty means
// eligible.
func validMergeBlocker(blocker string) bool {
	switch blocker {
	case extensions.MergeBlockerMerged, extensions.MergeBlockerClosed,
		extensions.MergeBlockerPermission, extensions.MergeBlockerWorkInProgress,
		extensions.MergeBlockerNotMergeable, extensions.MergeBlockerChecking,
		extensions.MergeBlockerStatusChecks, extensions.MergeBlockerApprovals,
		extensions.MergeBlockerRequestedChanges, extensions.MergeBlockerReviewRequests,
		extensions.MergeBlockerOutdatedBranch, extensions.MergeBlockerProtectedFiles,
		extensions.MergeBlockerSignature, extensions.MergeBlockerDependency,
		extensions.MergeBlockerBranchProtection:
		return true
	default:
		return false
	}
}

// validEligibility checks that the verdict matches the request: the bound
// pull number, well-formed tips, an eligible verdict with no blocker or a
// blocked verdict with exactly one valid blocker, and decimal blocking
// review IDs. Hidden verdicts redact to the pull locator.
func validEligibility(got *MergeEligibility, req SnapshotRequest) error {
	if !decimalID(got.PullNumber) {
		return ErrInvalidSnapshot
	}
	if req.PullNumber != "" && got.PullNumber != req.PullNumber {
		return ErrInvalidSnapshot
	}
	if !got.Visible {
		if got.Eligible || got.Blocker != "" || len(got.BlockingReviewIDs) != 0 {
			return ErrInvalidSnapshot
		}
		return nil
	}
	if got.HeadTip != "" && !fullOID(got.HeadTip) {
		return ErrInvalidSnapshot
	}
	if got.BaseTip != "" && !fullOID(got.BaseTip) {
		return ErrInvalidSnapshot
	}
	if got.Eligible {
		if got.Blocker != "" || len(got.BlockingReviewIDs) != 0 {
			return ErrInvalidSnapshot
		}
		return nil
	}
	if !validMergeBlocker(got.Blocker) {
		return ErrInvalidSnapshot
	}
	seen := make(map[string]bool, len(got.BlockingReviewIDs))
	for _, id := range got.BlockingReviewIDs {
		if !decimalID(id) || seen[id] {
			return ErrInvalidSnapshot
		}
		seen[id] = true
	}
	return nil
}
