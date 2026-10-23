package forgejo

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"

	extensions "forgejo.org/extension-sdk"
	"github.com/levitateos/sodaos/internal/config"
	"github.com/levitateos/sodaos/internal/factory"
)

// ReadDeliveryMaterial uses the existing merge actor and native snapshot
// owner. Every review summary and thread member must have its exact raw
// comment body under the same idle revision. Incomplete pages refuse rather
// than buying a judgment on a truncated conversation.
func (m *Merger) ReadDeliveryMaterial(ctx context.Context, w factory.MergeWork) (factory.DeliveryNativeMaterial, error) {
	var empty factory.DeliveryNativeMaterial
	if w.ValidateTarget() != nil {
		return empty, &factory.PublicationRefusal{Reason: "invalid_work"}
	}
	if err := m.checkActor(ctx, w.ActorID); err != nil {
		return empty, err
	}
	reader := &BackgroundSnapshotReader{Client: m.background, ActorID: strconv.FormatInt(w.ActorID, 10)}
	credential := extensions.CredentialFile(m.tokenFile)
	repo, number := strconv.FormatInt(w.Repository, 10), strconv.FormatInt(w.PRNumber, 10)
	issue, err := BracketedRead(ctx, reader, credential, SnapshotRequest{RepositoryID: repo, IssueIndex: number, Families: []SnapshotFamily{FamilyIssue, FamilyComments}, Limit: SnapshotPageLimit})
	if err != nil {
		return empty, mergeReadError(err)
	}
	pull, err := BracketedRead(ctx, reader, credential, SnapshotRequest{RepositoryID: repo, PullNumber: number, Families: []SnapshotFamily{FamilyPull, FamilyReviews, FamilyReviewThreads, FamilyMergeEligibility, FamilyRefs}, Refs: []string{w.HeadRef, w.BaseRef}, Limit: SnapshotPageLimit})
	if err != nil {
		return empty, mergeReadError(err)
	}
	bodies, err := m.readDeliveryBodies(ctx, reader, credential, w, &issue, pull)
	if err != nil {
		return empty, err
	}
	return deliveryMaterial(w, issue, pull, bodies)
}

// The native ordinary-comments list intentionally omits code/review rows.
// Explicit native member IDs recover each code body. Review summaries use
// the supported authenticated per-review read, checked against the complete
// native review digest and one unchanged idle revision around all reads.
func (m *Merger) readDeliveryBodies(ctx context.Context, reader SnapshotReader, credential extensions.CredentialFile, w factory.MergeWork, issue *NativeSnapshot, pull NativeSnapshot) (map[string]string, error) {
	wait := &factory.PublicationWait{Reason: "delivery_material_unavailable"}
	if issue.Revision != pull.Revision || issue.Comments == nil || pull.Threads == nil || pull.Reviews == nil {
		return nil, wait
	}
	seen := map[string]bool{}
	for _, comment := range issue.Comments.Items {
		seen[comment.ID] = true
	}
	var members []string
	for _, thread := range pull.Threads.Items {
		for _, member := range thread.Members {
			if !seen[member.ID] {
				seen[member.ID] = true
				members = append(members, member.ID)
			}
		}
	}
	if len(members) > 256 {
		return nil, wait
	}
	for len(members) > 0 {
		count := min(len(members), SnapshotIDListLimit)
		page, err := BracketedRead(ctx, reader, credential, SnapshotRequest{RepositoryID: strconv.FormatInt(w.Repository, 10), Families: []SnapshotFamily{FamilyComments}, CommentIDs: members[:count]})
		if err != nil || page.Revision != issue.Revision || page.Comments == nil || page.Comments.IssueID != issue.Issue.ID {
			return nil, wait
		}
		issue.Comments.Items = append(issue.Comments.Items, page.Comments.Items...)
		issue.Comments.Total = len(issue.Comments.Items)
		members = members[count:]
	}
	token, err := config.Secret(m.tokenFile)
	if err != nil {
		return nil, &factory.PublicationWait{Reason: "credential_invalid"}
	}
	repository, err := m.rest.RepositoryByID(ctx, token, w.Repository)
	if err != nil {
		return nil, wait
	}
	base := "/repos/" + url.PathEscape(repository.Owner.Login) + "/" + url.PathEscape(repository.Name) + "/pulls/" + strconv.FormatInt(w.PRNumber, 10) + "/reviews/"
	bodies := make(map[string]string, len(pull.Reviews.Items))
	total := 0
	for _, native := range pull.Reviews.Items {
		if !native.Visible || !native.Complete || native.Type == "PENDING" {
			return nil, wait
		}
		var review struct {
			ID   int64  `json:"id"`
			Body string `json:"body"`
			User struct {
				ID int64 `json:"id"`
			} `json:"user"`
		}
		if err := m.rest.request(ctx, "GET", base+native.ID, token, nil, &review); err != nil || strconv.FormatInt(review.ID, 10) != native.ID || strconv.FormatInt(review.User.ID, 10) != native.ReviewerID || ContentDigest(review.Body) != native.ContentDigest {
			return nil, &factory.PublicationWait{Reason: "delivery_review_body_unavailable"}
		}
		total += len(review.Body)
		if total > 128*1024 {
			return nil, wait
		}
		bodies[native.ID] = review.Body
	}
	after, err := reader.ReadNativeRevision(ctx)
	if err != nil || !after.Idle || after.Revision != issue.Revision {
		return nil, &factory.PublicationWait{Reason: "revision_moved"}
	}
	return bodies, nil
}

func deliveryMaterial(w factory.MergeWork, issue, pull NativeSnapshot, reviewBodies map[string]string) (factory.DeliveryNativeMaterial, error) {
	var empty factory.DeliveryNativeMaterial
	wait := &factory.PublicationWait{Reason: "delivery_material_unavailable"}
	if issue.Revision < 1 || issue.Revision != pull.Revision ||
		HasHiddenEvidence(SnapshotRequest{Families: []SnapshotFamily{FamilyIssue, FamilyComments}}, issue) ||
		HasHiddenEvidence(SnapshotRequest{Families: []SnapshotFamily{FamilyPull, FamilyReviews, FamilyReviewThreads, FamilyMergeEligibility, FamilyRefs}}, pull) ||
		issue.Comments == nil || pull.Reviews == nil || pull.Threads == nil || pull.Eligibility == nil ||
		!issue.Comments.Complete || !pull.Reviews.Complete || !pull.Threads.Complete || !pull.Eligibility.Complete {
		return empty, wait
	}
	if err := checkMergeIdentity(w, issue.Issue, pull.Pull); err != nil {
		return empty, err
	}
	if err := checkMergeRefs(pull, w); err != nil {
		return empty, err
	}
	if pull.Pull.HeadTip != w.HeadOID || pull.Eligibility.HeadTip != w.HeadOID || pull.Eligibility.BaseTip != w.BaseOID || issue.Comments.IssueID != issue.Issue.ID || pull.Reviews.IssueID != issue.Issue.ID {
		return empty, &factory.PublicationWait{Reason: "delivery_native_tips_changed"}
	}
	out := factory.DeliveryNativeMaterial{Revision: issue.Revision, Repository: w.Repository, PRID: w.PRID, IssueID: w.IssueID, Head: w.HeadOID, Base: w.BaseOID, Title: issue.Issue.Title, Body: issue.Issue.Content, Eligible: pull.Eligibility.Eligible, Blocker: pull.Eligibility.Blocker, BlockingReviewIDs: append([]string(nil), pull.Eligibility.BlockingReviewIDs...)}
	comments := make(map[string]CommentEvidence, len(issue.Comments.Items))
	for _, comment := range issue.Comments.Items {
		if !comment.Visible || !comment.Complete || comment.IssueID != issue.Issue.ID || comment.ContentDigest != ContentDigest(comment.Content) {
			return empty, &factory.PublicationWait{Reason: "delivery_comment_material_unavailable"}
		}
		if _, duplicate := comments[comment.ID]; duplicate {
			return empty, wait
		}
		comments[comment.ID] = comment
		out.Comments = append(out.Comments, factory.DeliveryComment{ID: comment.ID, PosterID: comment.PosterID, ReviewID: comment.ReviewID, Type: comment.Type, Content: comment.Content, Digest: comment.ContentDigest, ContentVersion: comment.ContentVersion, Invalidated: comment.Invalidated})
	}
	for _, review := range pull.Reviews.Items {
		if !review.Visible || !review.Complete || review.IssueID != issue.Issue.ID || review.Type == "PENDING" {
			return empty, &factory.PublicationWait{Reason: "delivery_review_metadata_unavailable"}
		}
		body, found := reviewBodies[review.ID]
		if !found || ContentDigest(body) != review.ContentDigest {
			return empty, &factory.PublicationWait{Reason: "delivery_review_body_unavailable"}
		}
		out.Reviews = append(out.Reviews, factory.DeliveryNativeReview{ID: review.ID, ActorID: review.ReviewerID, Type: review.Type, Commit: review.CommitID, Content: body, Digest: review.ContentDigest, Official: review.Official, Stale: review.Stale, Dismissed: review.Dismissed})
	}
	for _, thread := range pull.Threads.Items {
		if !thread.Visible || !thread.Complete || thread.IssueID != issue.Issue.ID || len(thread.Members) != thread.MemberTotal {
			return empty, &factory.PublicationWait{Reason: "delivery_thread_metadata_unavailable"}
		}
		item := factory.DeliveryThread{RootID: thread.RootID, ReviewID: thread.ReviewID, Path: thread.Path, Side: thread.Side, Line: thread.Line, OriginalCommit: thread.OriginalCommit, HeadCommit: thread.HeadCommit, Digest: thread.Digest, Resolved: thread.Resolved, ResolverID: thread.ResolverID}
		for _, member := range thread.Members {
			body, ok := comments[member.ID]
			if !ok || !member.Visible || !member.Complete || body.ContentDigest != member.ContentDigest || body.ContentVersion != member.ContentVersion || member.IssueID != issue.Issue.ID {
				return empty, &factory.PublicationWait{Reason: "delivery_thread_body_unavailable"}
			}
			item.Members = append(item.Members, member.ID)
		}
		out.Threads = append(out.Threads, item)
	}
	raw, err := json.Marshal(out)
	if err != nil || len(raw) > 128*1024 {
		return empty, wait
	}
	return out, nil
}
