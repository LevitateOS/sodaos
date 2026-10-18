package forgejo

import (
	"strconv"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
)

func checkMergeIdentity(w factory.MergeWork, i *IssueEvidence, p *PullEvidence) error {
	if i == nil || p == nil || !i.Visible || !p.Visible || !i.IsPull || i.IsClosed || p.HasMerged ||
		i.ID != strconv.FormatInt(w.IssueID, 10) || i.Provenance.PosterID != strconv.FormatInt(w.PRAuthorID, 10) ||
		p.ID != strconv.FormatInt(w.PRID, 10) || p.IssueID != i.ID || p.HeadRepoID != strconv.FormatInt(w.Repository, 10) ||
		p.HeadBranch != factory.RefHead(w.HeadRef) || p.BaseBranch != factory.RefHead(w.BaseRef) {
		return &factory.PublicationRefusal{Reason: "pr_mismatch"}
	}
	return nil
}

func checkMergeRefs(snapshot NativeSnapshot, w factory.MergeWork) error {
	for _, ref := range snapshot.Refs {
		if !ref.Visible || !ref.Exists {
			return &factory.PublicationRefusal{Reason: "pr_mismatch"}
		}
		if ref.Ref == w.HeadRef && ref.OID != w.HeadOID {
			return &factory.PublicationRefusal{Reason: "stale_head"}
		}
		if ref.Ref == w.BaseRef && ref.OID != w.BaseOID {
			return &factory.PublicationRefusal{Reason: "stale_base_or_result"}
		}
	}
	return nil
}

func findMergeApproval(snapshot NativeSnapshot, issueID string, w factory.MergeWork) (int64, error) {
	if snapshot.Reviews == nil || snapshot.Reviews.IssueID != issueID {
		return 0, &factory.PublicationWait{Reason: "merge_evidence_unavailable"}
	}
	var approval int64
	for _, review := range snapshot.Reviews.Items {
		if !review.Visible || review.IssueID != issueID {
			return 0, &factory.PublicationWait{Reason: "merge_evidence_unavailable"}
		}
		reviewer, err := strconv.ParseInt(review.ReviewerID, 10, 64)
		if err != nil {
			return 0, &factory.PublicationWait{Reason: "merge_evidence_unavailable"}
		}
		if reviewer == w.PRAuthorID {
			return 0, &factory.PublicationRefusal{Reason: "author_reviewed"}
		}
		switch {
		case review.Dismissed || review.Stale:
			continue
		case review.Type == "REQUEST_CHANGES" && review.CommitID == w.HeadOID:
			return 0, &factory.PublicationRefusal{Reason: "changes_requested"}
		case review.Type == "APPROVED" && review.Official && review.CommitID == w.HeadOID && reviewer == w.ReviewerID:
			id, err := strconv.ParseInt(review.ID, 10, 64)
			if err != nil || id <= 0 {
				return 0, &factory.PublicationWait{Reason: "merge_evidence_unavailable"}
			}
			approval = id
		}
	}
	return approval, nil
}

func matchMergeTarget(w factory.MergeWork, issue, snapshot NativeSnapshot) (factory.MergeObservation, error) {
	var empty factory.MergeObservation
	i, p := issue.Issue, snapshot.Pull
	if err := checkMergeIdentity(w, i, p); err != nil {
		return empty, err
	}
	observed := factory.MergeObservation{
		NativeRev: snapshot.Revision, ObservedUnix: time.Now().Unix(),
		PRID: w.PRID, PRNumber: w.PRNumber, IssueID: w.IssueID, PRAuthorID: w.PRAuthorID,
		HeadRef: w.HeadRef, BaseRef: w.BaseRef, HeadOID: w.HeadOID, BaseOID: w.BaseOID,
		ReviewerID: w.ReviewerID,
	}
	if p.HeadTip != w.HeadOID {
		return empty, &factory.PublicationRefusal{Reason: "stale_head"}
	}
	if err := checkMergeRefs(snapshot, w); err != nil {
		return empty, err
	}
	approval, err := findMergeApproval(snapshot, i.ID, w)
	if err != nil {
		return empty, err
	}
	if approval == 0 {
		// The approval may still arrive: a missing approval waits for
		// evidence, while a rejected or misidentified candidate refuses.
		return empty, &factory.PublicationWait{Reason: "approval_missing"}
	}
	observed.ReviewID = approval
	checks, err := mapObservedChecks(snapshot, w.HeadRef, w.BaseRef, "merge_evidence_unavailable")
	if err != nil {
		return empty, err
	}
	observed.Checks = checks
	return observed, nil
}
