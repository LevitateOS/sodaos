package forgejo

import (
	"sort"
	"strconv"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
)

func matchMergeTarget(w factory.MergeWork, issue, snapshot NativeSnapshot) (factory.MergeObservation, error) {
	var empty factory.MergeObservation
	i, p := issue.Issue, snapshot.Pull
	if i == nil || p == nil || !i.Visible || !p.Visible || !i.IsPull || i.IsClosed || p.HasMerged ||
		i.ID != strconv.FormatInt(w.IssueID, 10) || i.Provenance.PosterID != strconv.FormatInt(w.PRAuthorID, 10) ||
		p.ID != strconv.FormatInt(w.PRID, 10) || p.IssueID != i.ID || p.HeadRepoID != strconv.FormatInt(w.Repository, 10) ||
		p.HeadBranch != factory.RefHead(w.HeadRef) || p.BaseBranch != factory.RefHead(w.BaseRef) {
		return empty, &factory.PublicationRefusal{Reason: "pr_mismatch"}
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
	for _, ref := range snapshot.Refs {
		if !ref.Visible || !ref.Exists {
			return empty, &factory.PublicationRefusal{Reason: "pr_mismatch"}
		}
		if ref.Ref == w.HeadRef && ref.OID != w.HeadOID {
			return empty, &factory.PublicationRefusal{Reason: "stale_head"}
		}
		if ref.Ref == w.BaseRef && ref.OID != w.BaseOID {
			return empty, &factory.PublicationRefusal{Reason: "stale_base_or_result"}
		}
	}
	if snapshot.Reviews == nil || snapshot.Reviews.IssueID != i.ID {
		return empty, &factory.PublicationWait{Reason: "merge_evidence_unavailable"}
	}
	var approval int64
	for _, review := range snapshot.Reviews.Items {
		if !review.Visible || review.IssueID != i.ID {
			return empty, &factory.PublicationWait{Reason: "merge_evidence_unavailable"}
		}
		reviewer, err := strconv.ParseInt(review.ReviewerID, 10, 64)
		if err != nil {
			return empty, &factory.PublicationWait{Reason: "merge_evidence_unavailable"}
		}
		if reviewer == w.PRAuthorID {
			return empty, &factory.PublicationRefusal{Reason: "author_reviewed"}
		}
		switch {
		case review.Dismissed || review.Stale:
			continue
		case review.Type == "REQUEST_CHANGES" && review.CommitID == w.HeadOID:
			return empty, &factory.PublicationRefusal{Reason: "changes_requested"}
		case review.Type == "APPROVED" && review.Official && review.CommitID == w.HeadOID && reviewer == w.ReviewerID:
			id, err := strconv.ParseInt(review.ID, 10, 64)
			if err != nil || id <= 0 {
				return empty, &factory.PublicationWait{Reason: "merge_evidence_unavailable"}
			}
			approval = id
		}
	}
	if approval == 0 {
		// The approval may still arrive: a missing approval waits for
		// evidence, while a rejected or misidentified candidate refuses.
		return empty, &factory.PublicationWait{Reason: "approval_missing"}
	}
	observed.ReviewID = approval
	checks, err := matchMergeChecks(w, snapshot)
	if err != nil {
		return empty, err
	}
	observed.Checks = checks
	return observed, nil
}

// matchMergeChecks maps the merge bracket's check evidence for a fresh
// factory verdict. The latest status per context wins, ordered by
// native update stamp then native ID, matching combined-status
// semantics; hidden check items mark the observation hidden. Stale
// tips stay in the observation for the verdict.
func matchMergeChecks(w factory.MergeWork, snapshot NativeSnapshot) (factory.ObservedChecks, error) {
	var empty factory.ObservedChecks
	if snapshot.Checks == nil {
		return empty, &factory.PublicationWait{Reason: "merge_evidence_unavailable"}
	}
	observed := factory.ObservedChecks{NativeRev: snapshot.Revision, Complete: true}
	for _, ref := range snapshot.Refs {
		switch ref.Ref {
		case w.HeadRef:
			if !ref.Visible {
				observed.Hidden = true
				continue
			}
			if ref.Exists {
				observed.HeadTip = ref.OID
			}
		case w.BaseRef:
			if !ref.Visible {
				observed.Hidden = true
				continue
			}
			if ref.Exists {
				observed.BaseTip = ref.OID
			}
		}
	}
	type latest struct {
		state   string
		updated int64
		id      int64
	}
	winners := make(map[string]latest)
	contexts := make(map[string]bool)
	for _, item := range snapshot.Checks.Items {
		if !item.Visible {
			observed.Hidden = true
			continue
		}
		contexts[item.Context] = true
		id, err := strconv.ParseInt(item.ID, 10, 64)
		if err != nil || id <= 0 {
			return empty, &factory.PublicationWait{Reason: "merge_evidence_unavailable"}
		}
		prior, seen := winners[item.Context]
		if !seen || item.UpdatedUnix > prior.updated || item.UpdatedUnix == prior.updated && id > prior.id {
			winners[item.Context] = latest{state: item.State, updated: item.UpdatedUnix, id: id}
		}
	}
	observed.ObservedContexts = len(contexts)
	for context, winner := range winners {
		observed.Checks = append(observed.Checks, factory.ObservedCheck{Context: context, State: winner.state})
	}
	sort.Slice(observed.Checks, func(i, j int) bool { return observed.Checks[i].Context < observed.Checks[j].Context })
	if err := observed.Validate(); err != nil {
		return empty, &factory.PublicationWait{Reason: "merge_evidence_unavailable"}
	}
	return observed, nil
}
