package forgejo

import (
	"strings"

	extensions "forgejo.org/extension-sdk"
)

// PullEvidence is the permission-checked view of one native pull request.
type PullEvidence = extensions.SnapshotPull

// ReviewEvidence is the permission-checked view of one native review.
type ReviewEvidence = extensions.SnapshotReview

// ReviewPage carries one bounded review list with completeness evidence.
type ReviewPage = extensions.SnapshotReviewPage

// CheckEvidence is the permission-checked view of one native commit status.
type CheckEvidence = extensions.SnapshotCheck

// CheckSet carries one bounded check list for an exact commit.
type CheckSet = extensions.SnapshotCheckSet

// RefEvidence is the permission-checked view of one native branch tip.
type RefEvidence = extensions.SnapshotRef

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
