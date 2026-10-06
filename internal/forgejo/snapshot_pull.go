package forgejo

import "strings"

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
