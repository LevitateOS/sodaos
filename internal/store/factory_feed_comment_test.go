package store

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
)

func TestFeedRepliesIncludeOtherAuthorsAndPageWithoutCrossingSources(t *testing.T) {
	ctx := context.Background()
	s, _ := postgresFixture(t, nil)
	parent := saveTestComment(t, s, 7, "42", "parent", "parent")
	for i := 0; i < 23; i++ {
		_, _, err := s.SaveFeedComment(ctx, factory.CommentRequest{Repository: 7, Body: "reply", ReplyTo: parent, IdempotencyKey: fmt.Sprint(i)}, "43", "Another person", 1729000001)
		if err != nil {
			t.Fatal(err)
		}
	}
	_, _, err := s.SaveFeedComment(ctx, factory.CommentRequest{Repository: 8, Body: "other project", ReplyTo: parent, IdempotencyKey: "foreign"}, "44", "Other", 1729000002)
	if err != nil {
		t.Fatal(err)
	}
	first, next, err := s.FeedReplyIDs(ctx, 7, "post", parent, 0)
	if err != nil || len(first) != 20 || next == 0 {
		t.Fatalf("first page: %d %d %v", len(first), next, err)
	}
	second, next, err := s.FeedReplyIDs(ctx, 7, "post", parent, next)
	if err != nil || len(second) != 3 || next != 0 {
		t.Fatalf("second page: %d %d %v", len(second), next, err)
	}
	seen := map[string]bool{}
	for _, id := range append(first, second...) {
		if seen[id] {
			t.Fatal("reply duplicated across pages")
		}
		seen[id] = true
	}
	if _, err := s.DeleteFeedComment(ctx, second[0], "43", 0); err != nil {
		t.Fatal(err)
	}
	remaining, _, err := s.FeedReplyIDs(ctx, 7, "post", parent, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range remaining {
		if id == second[0] {
			t.Fatal("deleted reply exposed")
		}
	}
}

func saveTestComment(t *testing.T, s *Store, repository int64, author, body, key string) string {
	t.Helper()
	id, revision, err := s.SaveFeedComment(context.Background(), factory.CommentRequest{
		Repository: repository, Body: body, IdempotencyKey: key,
	}, author, author+"-name", 1729000000)
	if err != nil {
		t.Fatal(err)
	}
	if revision != 0 {
		t.Fatalf("fresh comment revision = %d", revision)
	}
	return id
}

func TestFeedbackQueueDoesNotReplayHistoricalIssueMappings(t *testing.T) {
	s, _ := postgresFixture(t, nil)
	ctx := context.Background()
	id := saveTestComment(t, s, 7, "42", "existing work", "old-feedback")
	if err := s.RecordCommentIntake(ctx, id, 0); err != nil {
		t.Fatal(err)
	}
	pipe, err := s.CommentPipeline(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	next := pipe
	next.Judgment = factory.CommentJudgmentApplied
	next.Route = factory.CommentRouteExistingWork
	next.Issue = 55
	if err = s.AdvanceCommentPipeline(ctx, pipe, next); err != nil {
		t.Fatal(err)
	}
	queued, err := s.PendingCommentPipelines(ctx, 4)
	if err != nil || len(queued) != 1 {
		t.Fatalf("newly assessed feedback absent: %+v %v", queued, err)
	}
	if _, err = s.db.ExecContext(ctx, `UPDATE factory_feed_comment_requests SET data=data-'feedback_revision' WHERE comment=$1`, id); err != nil {
		t.Fatal(err)
	}
	queued, err = s.PendingCommentPipelines(ctx, 4)
	if err != nil || len(queued) != 0 {
		t.Fatalf("historical mapping replayed: %+v %v", queued, err)
	}
}

func TestFeedCommentSaveReplaysIdempotencyKey(t *testing.T) {
	ctx := context.Background()
	s, _ := postgresFixture(t, nil)
	first := saveTestComment(t, s, 7, "42", "please fix the login redirect", "key-1")
	again, revision, err := s.SaveFeedComment(ctx, factory.CommentRequest{
		Repository: 7, Body: "a different body", IdempotencyKey: "key-1",
	}, "42", "ada", 1729000060)
	if err != nil {
		t.Fatal(err)
	}
	if again != first || revision != 0 {
		t.Fatalf("replay = %q rev %d, want %q rev 0", again, revision, first)
	}
	detail, err := s.FeedComment(ctx, first)
	if err != nil {
		t.Fatal(err)
	}
	if detail.Body != "please fix the login redirect" || detail.AuthorName != "42-name" || detail.At != 1729000000 {
		t.Fatalf("replay rewrote the original: %+v", detail)
	}
	// A different key from the same author is a new comment.
	other := saveTestComment(t, s, 7, "42", "another thought", "key-2")
	if other == first {
		t.Fatal("distinct key collided")
	}
}

func TestFeedCommentEditDeleteRevoke(t *testing.T) {
	ctx := context.Background()
	s, _ := postgresFixture(t, nil)
	id := saveTestComment(t, s, 7, "42", "original", "key-1")
	if err := s.RecordCommentIntake(ctx, id, 0); err != nil {
		t.Fatal(err)
	}
	revision, err := s.EditFeedComment(ctx, id, "42", "edited text", 0, 1729000100)
	if err != nil || revision != 1 {
		t.Fatalf("edit = %d %v", revision, err)
	}
	detail, err := s.FeedComment(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if detail.Body != "edited text" || !detail.Record.Edited || detail.Record.Revision != 1 {
		t.Fatalf("edited detail: %+v", detail)
	}
	pipe, err := s.CommentPipeline(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if pipe.Revision != 1 {
		t.Fatalf("pipeline not rebound: %+v", pipe)
	}
	// Stale revision refuses.
	if _, err := s.EditFeedComment(ctx, id, "42", "stale", 0, 1729000200); err == nil {
		t.Fatal("stale edit admitted")
	}
	// Foreign author refuses.
	if _, err := s.EditFeedComment(ctx, id, "43", "hijack", 1, 1729000200); err == nil {
		t.Fatal("foreign edit admitted")
	}
	// Delete tombstones and blocks the unsettled effect.
	revision, err = s.DeleteFeedComment(ctx, id, "42", 1)
	if err != nil || revision != 2 {
		t.Fatalf("delete = %d %v", revision, err)
	}
	pipe, err = s.CommentPipeline(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if pipe.Effect != factory.CommentEffectBlocked {
		t.Fatalf("deleted effect = %q", pipe.Effect)
	}
	// Deleted sources refuse edits but still resolve.
	if _, err := s.EditFeedComment(ctx, id, "42", "resurrect", 2, 1729000300); err == nil {
		t.Fatal("edit after delete admitted")
	}
	if _, err := s.FeedComment(ctx, id); err != nil {
		t.Fatal(err)
	}
	// Revoke blocks the unsettled effect on a second comment.
	second := saveTestComment(t, s, 7, "42", "second", "key-2")
	if err := s.RecordCommentIntake(ctx, second, 0); err != nil {
		t.Fatal(err)
	}
	if err := s.MarkFeedCommentRevoked(ctx, second); err != nil {
		t.Fatal(err)
	}
	pipe, err = s.CommentPipeline(ctx, second)
	if err != nil {
		t.Fatal(err)
	}
	if pipe.Effect != factory.CommentEffectBlocked {
		t.Fatalf("revoked effect = %q", pipe.Effect)
	}
}

func TestCommentPipelineAdvanceIsCAS(t *testing.T) {
	ctx := context.Background()
	s, _ := postgresFixture(t, nil)
	id := saveTestComment(t, s, 7, "42", "route me", "key-1")
	if err := s.RecordCommentIntake(ctx, id, 0); err != nil {
		t.Fatal(err)
	}
	// Replaying the intake recovers the row.
	if err := s.RecordCommentIntake(ctx, id, 0); err != nil {
		t.Fatal(err)
	}
	pipe, err := s.CommentPipeline(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	next := pipe
	next.Judgment = factory.CommentJudgmentApplied
	next.Route = factory.CommentRouteDiscussion
	if err := s.AdvanceCommentPipeline(ctx, pipe, next); err != nil {
		t.Fatal(err)
	}
	// The pre-advance snapshot is now stale and refuses.
	stale := next
	stale.Route = factory.CommentRouteClarify
	if err := s.AdvanceCommentPipeline(ctx, pipe, stale); err == nil {
		t.Fatal("stale advance admitted")
	}
	// Attempts exhaust at the bound.
	for i := 0; i < factory.CommentAttemptLimit; i++ {
		if err := s.NoteCommentAttempt(ctx, id); err != nil {
			t.Fatalf("attempt %d: %v", i, err)
		}
	}
	if err := s.NoteCommentAttempt(ctx, id); err == nil {
		t.Fatal("over-limit attempt admitted")
	}
}

func TestCommentDraftBindsSourceRevision(t *testing.T) {
	ctx := context.Background()
	s, _ := postgresFixture(t, nil)
	id := saveTestComment(t, s, 7, "42", "needs work", "key-1")
	if err := s.RecordCommentIntake(ctx, id, 0); err != nil {
		t.Fatal(err)
	}
	draft := factory.IssueDraft{Comment: id, Title: "Fix it", Body: "Change the code.", Objective: "Works.", Acceptance: "Tests pass.", Metadata: `{"work":"soda"}`}
	if err := s.RecordCommentDraft(ctx, draft); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CommentDraft(ctx, id, 0); err != nil {
		t.Fatal(err)
	}
	// After an edit the retained draft no longer binds.
	if _, err := s.EditFeedComment(ctx, id, "42", "needs work, updated", 0, 1729000100); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CommentDraft(ctx, id, 1); err == nil {
		t.Fatal("stale-revision draft served")
	}
	// Malformed drafts refuse before any native effect.
	broken := draft
	broken.Acceptance = ""
	if err := s.RecordCommentDraft(ctx, broken); err == nil {
		t.Fatal("malformed draft retained")
	}
}

func TestSubscribedFeedMergesPostsNewestFirst(t *testing.T) {
	ctx := context.Background()
	s, dsn := postgresFixture(t, nil)
	raw := journeySQL(t, dsn)
	acceptance := "d" + strings.Repeat("a", 24)
	insertJourneyAssignment(t, raw, journeyAssignment(strings.Repeat("1", 32), strings.Repeat("1", 32), acceptance, 7, 3, true), "assigned", 1)
	first := saveTestComment(t, s, 7, "42", "first comment", "key-1")
	insertJourneyAssignment(t, raw, journeyAssignment(strings.Repeat("2", 32), strings.Repeat("2", 32), acceptance, 7, 4, true), "assigned", 1)
	second := saveTestComment(t, s, 9, "43", "second comment", "key-9")

	page, _, truncated, err := s.SubscribedFeed(ctx, factory.FeedQuery{Repositories: []int64{7, 9}, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if truncated || len(page) != 4 {
		t.Fatalf("combined window = %+v truncated=%v", page, truncated)
	}
	// Insertion order was record, post, record, post: newest-first must
	// interleave them exactly in reverse.
	if page[0].ID != second || page[0].Kind != factory.FeedKindPost ||
		page[1].ID != strings.Repeat("2", 32) || page[1].Kind != factory.FeedKindRecord ||
		page[2].ID != first || page[2].Kind != factory.FeedKindPost ||
		page[3].ID != strings.Repeat("1", 32) || page[3].Kind != factory.FeedKindRecord {
		t.Fatalf("unordered combined window: %+v", page)
	}
	// Kind filters still scope.
	posts, _, _, err := s.SubscribedFeed(ctx, factory.FeedQuery{Repositories: []int64{7, 9}, Kinds: []string{factory.FeedKindPost}, Limit: 10})
	if err != nil || len(posts) != 2 {
		t.Fatalf("posts-only window = %+v %v", posts, err)
	}
	records, _, _, err := s.SubscribedFeed(ctx, factory.FeedQuery{Repositories: []int64{7, 9}, Kinds: []string{factory.FeedKindRecord}, Limit: 10})
	if err != nil || len(records) != 2 {
		t.Fatalf("records-only window = %+v %v", records, err)
	}
	// Cursor paging crosses the kind boundary.
	head, resume, truncated, err := s.SubscribedFeed(ctx, factory.FeedQuery{Repositories: []int64{7, 9}, Limit: 3})
	if err != nil || len(head) != 3 || !truncated {
		t.Fatalf("cut window = %+v truncated=%v %v", head, truncated, err)
	}
	tail, _, truncated, err := s.SubscribedFeed(ctx, factory.FeedQuery{Repositories: []int64{7, 9}, Limit: 3, Cursor: resume})
	if err != nil || len(tail) != 1 || truncated || tail[0].ID != strings.Repeat("1", 32) {
		t.Fatalf("resumed window = %+v truncated=%v %v", tail, truncated, err)
	}
	// Deleted and revoked sources leave the feed but still resolve.
	if _, err := s.DeleteFeedComment(ctx, first, "42", 0); err != nil {
		t.Fatal(err)
	}
	if err := s.MarkFeedCommentRevoked(ctx, second); err != nil {
		t.Fatal(err)
	}
	remaining, _, _, err := s.SubscribedFeed(ctx, factory.FeedQuery{Repositories: []int64{7, 9}, Kinds: []string{factory.FeedKindPost}, Limit: 10})
	if err != nil || len(remaining) != 0 {
		t.Fatalf("closed posts surface: %+v %v", remaining, err)
	}
}

func TestReserveCommentJudgmentIsIdempotent(t *testing.T) {
	ctx := context.Background()
	s, _ := postgresFixture(t, nil)
	id := saveTestComment(t, s, 7, "42", "route me", "key-1")
	funded, err := s.ReserveCommentJudgment(ctx, id, 0)
	if err != nil || !funded {
		t.Fatalf("reservation = %v %v", funded, err)
	}
	// The same source revision replays without new spend.
	again, err := s.ReserveCommentJudgment(ctx, id, 0)
	if err != nil || again {
		t.Fatalf("replay reservation = %v %v", again, err)
	}
	var spent int
	if err := s.db.QueryRowContext(ctx, `SELECT spent FROM factory_judgment_budget WHERE id=1`).Scan(&spent); err != nil {
		t.Fatal(err)
	}
	if spent != 1 {
		t.Fatalf("judgment spend = %d, want 1", spent)
	}
}

// insertTestAssignmentRow seeds one minimal finished assignment row
// behind a feed record / work-candidate read.
func insertTestAssignmentRow(t *testing.T, s *Store, ctx context.Context, repository, issue int64) string {
	t.Helper()
	id := factory.NewID()
	data, err := json.Marshal(factory.Assignment{
		ID: id, Repository: repository, Issue: issue,
		Run: factory.NewID(), Stage: factory.AssignmentFinished,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.ExecContext(ctx, `INSERT INTO factory_assignments(id,repository,issue,run,stage,revision,data) VALUES($1,$2,$3,$4,'finished',0,$5)`,
		id, repository, issue, factory.NewID(), string(data)); err != nil {
		t.Fatalf("seed assignment: %v", err)
	}
	return id
}

// TestCommentSourceResolutionAndRoundTrip proves the authorized
// source snapshot: records and posts resolve inside their repository,
// cross-repository sources refuse as not found, and the persisted
// snapshot reads back on the saved comment.
func TestCommentSourceResolutionAndRoundTrip(t *testing.T) {
	ctx := context.Background()
	s, _ := postgresFixture(t, nil)
	recordID := insertTestAssignmentRow(t, s, ctx, 7, 55)

	source, err := s.ResolveCommentSource(ctx, 7, factory.CommentSourceRecord, recordID)
	if err != nil {
		t.Fatal(err)
	}
	if source.Kind != factory.CommentSourceRecord || source.ID != recordID || source.Issue != 55 {
		t.Fatalf("record source: %+v", source)
	}
	if _, err := s.ResolveCommentSource(ctx, 8, factory.CommentSourceRecord, recordID); err != ErrNotFound {
		t.Fatalf("cross-repository record err = %v, want not found", err)
	}
	if _, err := s.ResolveCommentSource(ctx, 7, factory.CommentSourceRecord, factory.NewID()); err != ErrNotFound {
		t.Fatalf("unknown record err = %v, want not found", err)
	}

	id, _, err := s.SaveFeedCommentWithSource(ctx, factory.CommentRequest{
		Repository: 7, Body: "more on the filter", SourceKind: factory.CommentSourceRecord,
		SourceID: recordID, IdempotencyKey: "key-source",
	}, "42", "ada", 1729000000, source)
	if err != nil {
		t.Fatal(err)
	}
	detail, err := s.FeedComment(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if detail.Source == nil || detail.Source.ID != recordID || detail.Source.Issue != 55 {
		t.Fatalf("source round trip lost: %+v", detail.Source)
	}
	// A mismatched snapshot refuses before any write.
	mismatch := *source
	mismatch.ID = factory.NewID()
	if _, _, err := s.SaveFeedCommentWithSource(ctx, factory.CommentRequest{
		Repository: 7, Body: "x", SourceKind: factory.CommentSourceRecord,
		SourceID: recordID, IdempotencyKey: "key-mismatch",
	}, "42", "ada", 1729000000, &mismatch); err == nil {
		t.Fatal("mismatched source snapshot admitted")
	}
	// Post sources resolve to the stored post in the same repository.
	post, err := s.ResolveCommentSource(ctx, 7, factory.CommentSourcePost, id)
	if err != nil {
		t.Fatal(err)
	}
	if post.Summary != "more on the filter" {
		t.Fatalf("post source summary: %+v", post)
	}
}

// TestPendingCommentPipelinesAndCandidates proves the service work
// list and the existing-work candidate read: pending requests list
// until they settle, and recent worked issues list newest-first per
// issue inside their repository only.
func TestPendingCommentPipelinesAndCandidates(t *testing.T) {
	ctx := context.Background()
	s, _ := postgresFixture(t, nil)
	id := saveTestComment(t, s, 7, "42", "route me", "key-1")
	if err := s.RecordCommentIntake(ctx, id, 0); err != nil {
		t.Fatal(err)
	}
	pending, err := s.PendingCommentPipelines(ctx, 8)
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 1 || pending[0].Comment != id {
		t.Fatalf("pending list: %+v", pending)
	}
	pipe, err := s.CommentPipeline(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	next := pipe
	next.Judgment = factory.CommentJudgmentApplied
	next.Route = factory.CommentRouteDiscussion
	if err := s.AdvanceCommentPipeline(ctx, pipe, next); err != nil {
		t.Fatal(err)
	}
	pending, err = s.PendingCommentPipelines(ctx, 8)
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 0 {
		t.Fatalf("settled discussion still listed: %+v", pending)
	}
	if _, err := s.PendingCommentPipelines(ctx, 0); err == nil {
		t.Fatal("unbounded pending listing admitted")
	}
	exhausted := saveTestComment(t, s, 7, "42", "failed authoring", "exhausted")
	if err := s.RecordCommentIntake(ctx, exhausted, 0); err != nil {
		t.Fatal(err)
	}
	exhaustedPipe, err := s.CommentPipeline(ctx, exhausted)
	if err != nil {
		t.Fatal(err)
	}
	authoring := exhaustedPipe
	authoring.Judgment, authoring.Route = factory.CommentJudgmentApplied, factory.CommentRouteNewIssue
	if err := s.AdvanceCommentPipeline(ctx, exhaustedPipe, authoring); err != nil {
		t.Fatal(err)
	}
	for range factory.CommentAttemptLimit {
		if err := s.NoteCommentAttempt(ctx, exhausted); err != nil {
			t.Fatal(err)
		}
	}
	details, err := s.FeedPostDetails(ctx, []string{exhausted})
	if err != nil || details[exhausted].Status != "draft:failed" {
		t.Fatalf("feed header hid exhausted authoring: %+v, %v", details, err)
	}
	ready := saveTestComment(t, s, 7, "42", "next request", "next")
	if err := s.RecordCommentIntake(ctx, ready, 0); err != nil {
		t.Fatal(err)
	}
	pending, err = s.PendingCommentPipelines(ctx, 1)
	if err != nil || len(pending) != 1 || pending[0].Comment != ready {
		t.Fatalf("exhausted request consumed bounded service slot: %+v, %v", pending, err)
	}

	insertTestAssignmentRow(t, s, ctx, 7, 55)
	insertTestAssignmentRow(t, s, ctx, 7, 9)
	insertTestAssignmentRow(t, s, ctx, 7, 55)
	insertTestAssignmentRow(t, s, ctx, 8, 77)
	candidates, err := s.RecentCommentWorkCandidates(ctx, 7, 8)
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 2 || candidates[0].Issue != 55 || candidates[1].Issue != 9 {
		t.Fatalf("candidates: %+v", candidates)
	}
	for _, candidate := range candidates {
		if candidate.Origin != factory.CommentCandidateRecent {
			t.Fatalf("candidate origin: %+v", candidate)
		}
	}
}

func TestPostStatusShowsExhaustionWithoutDiscardingConfirmedIssue(t *testing.T) {
	p := factory.CommentPipeline{Judgment: factory.CommentJudgmentApplied, Route: factory.CommentRouteNewIssue, Draft: factory.CommentDraftPending, Effect: factory.CommentEffectPending, Attempts: factory.CommentAttemptLimit}
	if got := PostStatus(&p); got != "draft:failed" {
		t.Fatalf("exhausted authoring shown as %s", got)
	}
	p.Draft = factory.CommentDraftApplied
	if got := PostStatus(&p); got != "effect:unconfirmed" {
		t.Fatalf("unknown native outcome shown as %s", got)
	}
	p.Effect, p.Issue = factory.CommentEffectVerified, 9
	if got := PostStatus(&p); got != "issue:9" {
		t.Fatalf("confirmed issue lost: %s", got)
	}
	p.Judgment = factory.CommentJudgmentPending
	if got := PostStatus(&p); got != "judgment:failed" {
		t.Fatalf("exhausted judgment shown as %s", got)
	}
}

func TestRetryFeedCommentPreservesPostAndReconcilesUncertainEffect(t *testing.T) {
	ctx := context.Background()
	s, _ := postgresFixture(t, nil)
	id := saveTestComment(t, s, 7, "42", "Original request", "retry-1")
	if err := s.RecordCommentIntake(ctx, id, 0); err != nil {
		t.Fatal(err)
	}
	for range factory.CommentAttemptLimit {
		if err := s.NoteCommentAttempt(ctx, id); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.RetryFeedComment(ctx, id, "43", 0, "Other account", 1729000020); err == nil {
		t.Fatal("another author changed request")
	}
	if err := s.RetryFeedComment(ctx, id, "42", 0, "Clarified request", 1729000020); err != nil {
		t.Fatal(err)
	}
	detail, err := s.FeedComment(ctx, id)
	if err != nil || detail.Body != "Clarified request" || detail.Record.Revision != 1 {
		t.Fatalf("post lost: %+v %v", detail, err)
	}
	pipe, err := s.CommentPipeline(ctx, id)
	if err != nil || pipe.Attempts != 0 || pipe.Judgment != "pending" || pipe.Revision != 1 {
		t.Fatalf("retry state: %+v %v", pipe, err)
	}
	next := pipe
	next.Judgment = factory.CommentJudgmentApplied
	next.Route = factory.CommentRouteNewIssue
	next.Draft = factory.CommentDraftApplied
	next.Effect = factory.CommentEffectFailed
	if err := s.AdvanceCommentPipeline(ctx, pipe, next); err != nil {
		t.Fatal(err)
	}
	if err := s.RetryFeedComment(ctx, id, "42", 1, "Clarified request", 1729000030); err != nil {
		t.Fatal(err)
	}
	resumed, _ := s.CommentPipeline(ctx, id)
	if resumed.Revision != 1 || resumed.Draft != factory.CommentDraftApplied || resumed.Effect != factory.CommentEffectFailed {
		t.Fatalf("uncertain native effect was replaced: %+v", resumed)
	}
	done := resumed
	done.Effect = factory.CommentEffectApplied
	done.Issue = 5
	if err := s.AdvanceCommentPipeline(ctx, resumed, done); err != nil {
		t.Fatal(err)
	}
	if err := s.RetryFeedComment(ctx, id, "42", 1, "Duplicate", 1729000040); err == nil {
		t.Fatal("confirmed issue was duplicated")
	}
	// Same-content recovery after a prelaunch failure retains the original
	// source, route and attempts; one further failure consumes the permission.
	prelaunch := saveTestComment(t, s, 7, "42", "Original prelaunch request", "retry-prelaunch")
	if err := s.RecordCommentIntake(ctx, prelaunch, 0); err != nil {
		t.Fatal(err)
	}
	original, _ := s.CommentPipeline(ctx, prelaunch)
	routed := original
	routed.Judgment, routed.Route = factory.CommentJudgmentApplied, factory.CommentRouteNewIssue
	if err := s.AdvanceCommentPipeline(ctx, original, routed); err != nil {
		t.Fatal(err)
	}
	for range factory.CommentAttemptLimit {
		if err := s.NoteCommentAttempt(ctx, prelaunch); err != nil {
			t.Fatal(err)
		}
	}
	stopped, _ := s.CommentPipeline(ctx, prelaunch)
	if err := s.RetryFeedComment(ctx, prelaunch, "42", 0, "", 1729000050); err != nil {
		t.Fatal(err)
	}
	continued, err := s.CommentPipeline(ctx, prelaunch)
	if err != nil || !continued.Resume || continued.Attempts != factory.CommentAttemptLimit || continued.Revision != 0 || continued.Judgment != routed.Judgment || continued.Route != routed.Route || !continued.AdmitCommentDraft() || PostStatus(&continued) != "draft:pending" {
		t.Fatalf("prelaunch recovery replaced original state: %+v %v", continued, err)
	}
	if err := s.AdvanceCommentPipeline(ctx, stopped, stopped); err != ErrStaleRevision {
		t.Fatalf("old pipeline write replaced resumption: %v", err)
	}
	if err := s.RetryFeedComment(ctx, prelaunch, "42", 0, "", 1729000051); err == nil {
		t.Fatal("duplicate recovery request admitted")
	}
	if err := s.NoteCommentAttempt(ctx, prelaunch); err != nil {
		t.Fatal(err)
	}
	failed, _ := s.CommentPipeline(ctx, prelaunch)
	if failed.Resume || failed.Attempts != factory.CommentAttemptLimit || failed.Revision != 0 || failed.AdmitCommentDraft() {
		t.Fatalf("recovery failure lost original attempts or kept running: %+v", failed)
	}
	if err := s.DiscardCommentDraftData(ctx, prelaunch); err != nil {
		t.Fatal(err)
	}
	var recoveryAudit []byte
	if err := s.db.QueryRowContext(ctx, `SELECT data FROM factory_feed_comment_requests WHERE comment=$1`, prelaunch).Scan(&recoveryAudit); err != nil {
		t.Fatal(err)
	}
	var audit map[string]any
	if err := json.Unmarshal(recoveryAudit, &audit); err != nil || audit["resumed"] != true || audit["resumed_at"] != float64(1729000050) {
		t.Fatalf("discarding draft erased recovery history: %s %v", recoveryAudit, err)
	}
	if err := s.RetryFeedComment(ctx, prelaunch, "42", 0, "", 1729000052); err == nil {
		t.Fatal("second prelaunch recovery permission admitted")
	}
	failedEffect := failed
	failedEffect.Draft, failedEffect.Effect = factory.CommentDraftApplied, factory.CommentEffectFailed
	if err := s.AdvanceCommentPipeline(ctx, failed, failedEffect); err != nil {
		t.Fatal(err)
	}
	if err := s.RetryFeedComment(ctx, prelaunch, "42", 0, "", 1729000053); err == nil {
		t.Fatal("native-effect retry reset a resumed request's historical failures")
	}
	retainedEffect, _ := s.CommentPipeline(ctx, prelaunch)
	if retainedEffect != failedEffect {
		t.Fatalf("refused native-effect retry changed history: %+v", retainedEffect)
	}
	requests, err := s.PendingCommentPipelines(ctx, 64)
	if err != nil {
		t.Fatal(err)
	}
	for _, request := range requests {
		if request.Comment == prelaunch {
			t.Fatal("failed recovery kept consuming service slots")
		}
	}
	admittedID := saveTestComment(t, s, 7, "42", "Admitted request", "retry-admitted")
	if err := s.RecordCommentIntake(ctx, admittedID, 0); err != nil {
		t.Fatal(err)
	}
	original, _ = s.CommentPipeline(ctx, admittedID)
	routed = original
	routed.Judgment, routed.Route = factory.CommentJudgmentApplied, factory.CommentRouteNewIssue
	if err := s.AdvanceCommentPipeline(ctx, original, routed); err != nil {
		t.Fatal(err)
	}
	for range factory.CommentAttemptLimit {
		if err := s.NoteCommentAttempt(ctx, admittedID); err != nil {
			t.Fatal(err)
		}
	}
	assignment := insertTestAssignmentRow(t, s, ctx, 7, 1)
	if _, err := s.db.ExecContext(ctx, `UPDATE factory_assignments SET data=data || jsonb_build_object('subject',jsonb_build_object('kind','intent','intent',jsonb_build_object('intent_id',$2::text))) WHERE id=$1`, assignment, factory.CommentDecisionID("intent", admittedID, 0)); err != nil {
		t.Fatal(err)
	}
	if err := s.RetryFeedComment(ctx, admittedID, "42", 0, "", 1729000060); err == nil {
		t.Fatal("admitted planning attempt received prelaunch recovery permission")
	}
	admittedPipe, _ := s.CommentPipeline(ctx, admittedID)
	if admittedPipe.Resume || admittedPipe.Revision != 0 || admittedPipe.Attempts != factory.CommentAttemptLimit || admittedPipe.Judgment != routed.Judgment {
		t.Fatalf("admitted request history changed: %+v", admittedPipe)
	}
}

func TestPendingCommentRequestsRotatePastWaitingWork(t *testing.T) {
	ctx := context.Background()
	s, _ := postgresFixture(t, nil)
	first := saveTestComment(t, s, 7, "42", "Waiting", "fair-1")
	second := saveTestComment(t, s, 7, "42", "Ready", "fair-2")
	for _, id := range []string{first, second} {
		if err := s.RecordCommentIntake(ctx, id, 0); err != nil {
			t.Fatal(err)
		}
	}
	page, err := s.PendingCommentPipelines(ctx, 1)
	if err != nil || len(page) != 1 || page[0].Comment != first {
		t.Fatalf("first: %+v %v", page, err)
	}
	if err := s.NoteCommentProgress(ctx, first, "Waiting for setup"); err != nil {
		t.Fatal(err)
	}
	page, err = s.PendingCommentPipelines(ctx, 1)
	if err != nil || len(page) != 1 || page[0].Comment != second {
		t.Fatalf("waiting work starved next request: %+v %v", page, err)
	}
}

func TestCommentProductReviewKeepsCanonicalScopeAndOwner(t *testing.T) {
	s, _ := postgresFixture(t, nil)
	ctx := context.Background()
	idea := factory.ProductIdea{ID: "d888888888888888888888888", Description: "Track requests", Consequence: "Requests persist", Decision: "accept", Source: "requested", Provenance: "human", Actor: 42}
	accepted := factory.IntentRevision{ID: "d111111111111111111111111", Intent: "d000000000000000000000000", Owner: 42, Repository: 7, Objective: "Track equipment requests", State: "accepted", MaxIssues: 10, CreatedUnix: 1, Ideas: []factory.ProductIdea{idea}, Tasks: []factory.IntentTask{{Key: "d999999999999999999999999", Objective: "Record requests", Outcomes: []string{"Requests persist"}, IdeaIDs: []string{idea.ID}}}}
	if err := s.SaveIntentRevision(ctx, accepted); err != nil {
		t.Fatal(err)
	}
	var post string
	if err := s.db.QueryRowContext(ctx, `SELECT id FROM factory_feed_comments WHERE data->>'event_identity'=$1`, "intent-revision:"+accepted.ID).Scan(&post); err != nil {
		t.Fatal(err)
	}
	source, err := s.ResolveCommentSource(ctx, 7, factory.CommentSourcePost, post)
	if err != nil || source.Product == nil || source.Product.Intent != accepted.Intent || source.Product.AcceptedRevision != accepted.ID {
		t.Fatalf("canonical source missing: %+v %v", source, err)
	}
	if _, err := s.ResolveCommentSource(ctx, 8, factory.CommentSourcePost, post); err == nil {
		t.Fatal("foreign repository acquired product source")
	}
	save := func(key string) string {
		t.Helper()
		id, _, err := s.SaveFeedCommentWithSource(ctx, factory.CommentRequest{Repository: 7, Body: "Require Finance approval above EUR 1000; retain existing requests.", SourceKind: source.Kind, SourceID: source.ID, IdempotencyKey: key}, "43", "Robin", 2, source)
		if err != nil {
			t.Fatal(err)
		}
		if err = s.RecordCommentIntake(ctx, id, 0); err != nil {
			t.Fatal(err)
		}
		p, err := s.CommentPipeline(ctx, id)
		if err != nil {
			t.Fatal(err)
		}
		next := p
		next.Judgment = factory.CommentJudgmentApplied
		next.Route = factory.CommentRouteNewIssue
		if err = s.AdvanceCommentPipeline(ctx, p, next); err != nil {
			t.Fatal(err)
		}
		return id
	}
	id, stale := save("finance"), save("another-change")
	detail, err := s.FeedComment(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	pipe, err := s.CommentPipeline(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	view, err := s.CommentProductChange(ctx, detail, pipe, 42)
	if err != nil || view == nil || view.State != "review_required" || !view.CanReview {
		t.Fatalf("review not available: %+v %v", view, err)
	}
	if other, err := s.CommentProductChange(ctx, detail, pipe, 43); err != nil || other.CanReview {
		t.Fatalf("nonowner review authority: %+v %v", other, err)
	}
	if _, err = s.ReviewCommentProductChange(ctx, id, 0, 43, 3); err == nil {
		t.Fatal("comment author became product owner")
	}
	if _, err = s.ReviewCommentProductChange(ctx, id, 1, 42, 3); err == nil {
		t.Fatal("stale comment reviewed")
	}
	draft, err := s.ReviewCommentProductChange(ctx, id, 0, 42, 3)
	if err != nil {
		t.Fatal(err)
	}
	if draft.Intent != accepted.Intent || draft.Predecessor != accepted.ID || draft.State != "draft" || len(draft.Ideas) != 2 || len(draft.EffectiveOutcomes) != 2 || draft.EffectiveOutcomes[0] != idea.ID || draft.Tasks[0].Key != accepted.Tasks[0].Key {
		t.Fatalf("successor changed baseline: %+v", draft)
	}
	if !strings.Contains(draft.Ideas[1].Evidence, "43") || draft.Ideas[1].Description != detail.Body {
		t.Fatal("original request or attribution lost")
	}
	view, err = s.CommentProductChange(ctx, detail, pipe, 42)
	if err != nil || view.State != "draft" || !view.CanReview || view.DraftRevision != draft.ID {
		t.Fatalf("owner cannot continue: %+v %v", view, err)
	}
	if other, err := s.CommentProductChange(ctx, detail, pipe, 43); err != nil || other.CanReview {
		t.Fatalf("nonowner continuation authority: %+v %v", other, err)
	}
	replay, err := s.ReviewCommentProductChange(ctx, id, 0, 42, 4)
	if err != nil || replay.ID != draft.ID || replay.CreatedUnix != draft.CreatedUnix {
		t.Fatalf("handoff duplicated: %+v %v", replay, err)
	}
	if _, err = s.ReviewCommentProductChange(ctx, stale, 0, 42, 4); err == nil {
		t.Fatal("another comment replaced current review")
	}
	if _, err = s.EditFeedComment(ctx, id, "43", "Different request", 0, 5); err == nil {
		t.Fatal("admitted source rewritten")
	}
	if err = s.RetryFeedComment(ctx, id, "43", 0, "Different request", 5); err == nil {
		t.Fatal("admitted source retried")
	}
	queued, err := s.PendingCommentPipelines(ctx, 20)
	if err != nil || len(queued) != 0 {
		t.Fatalf("review waits in automatic issue queue: %+v %v", queued, err)
	}
	inherited, err := s.ResolveCommentSource(ctx, 7, factory.CommentSourcePost, id)
	if err != nil || inherited.Product == nil || *inherited.Product != *source.Product {
		t.Fatalf("thread lost source lineage: %+v %v", inherited, err)
	}
	adopted := draft
	adopted.ID = "d222222222222222222222222"
	adopted.Predecessor = draft.ID
	adopted.State = "accepted"
	adopted.Tasks = append(append([]factory.IntentTask(nil), draft.Tasks...), factory.IntentTask{Key: "d777777777777777777777777", Objective: "Finance approval", Outcomes: []string{"Threshold is applied"}, IdeaIDs: []string{draft.Ideas[1].ID}})
	if err = s.SaveIntentRevision(ctx, adopted); err != nil {
		t.Fatal(err)
	}
	view, err = s.CommentProductChange(ctx, detail, pipe, 42)
	if err != nil || view.State != "accepted" || !view.CanReview {
		t.Fatalf("owner cannot open accepted change: %+v %v", view, err)
	}
	if other, err := s.CommentProductChange(ctx, detail, pipe, 43); err != nil || other.State != "accepted" || other.CanReview {
		t.Fatalf("collaborator gained accepted owner control: %+v %v", other, err)
	}
	if view.Verified {
		t.Fatalf("accepted change claims verification without the F10 fact: %+v", view)
	}
	replay, err = s.ReviewCommentProductChange(ctx, id, 0, 42, 6)
	head, headErr := s.IntentRevision(ctx, accepted.Intent, 42)
	if err != nil || replay.ID != draft.ID || headErr != nil || head.ID != adopted.ID {
		t.Fatal("retry rolled back accepted head")
	}
	var revisions, graphs int
	if err = s.db.QueryRowContext(ctx, `SELECT count(*) FROM factory_intent_revisions WHERE intent=$1`, accepted.Intent).Scan(&revisions); err != nil {
		t.Fatal(err)
	}
	if err = s.db.QueryRowContext(ctx, `SELECT count(*) FROM factory_intent_generations`).Scan(&graphs); err != nil {
		t.Fatal(err)
	}
	if revisions != 3 || graphs != 0 {
		t.Fatalf("review silently generated scope: revisions=%d graphs=%d", revisions, graphs)
	}
	removed := adopted
	removed.ID, removed.Predecessor = "d333333333333333333333333", adopted.ID
	removed.Ideas = append([]factory.ProductIdea(nil), adopted.Ideas...)
	removed.Ideas[1].Decision = "decline"
	removed.EffectiveOutcomes = []string{idea.ID}
	removed.Tasks = accepted.Tasks
	if err = s.SaveIntentRevision(ctx, removed); err != nil {
		t.Fatal(err)
	}
	view, err = s.CommentProductChange(ctx, detail, pipe, 42)
	if err != nil || view.State != "superseded" {
		t.Fatalf("declined Finance proposal still presented as accepted: %+v %v", view, err)
	}
	// A corrupted same-product binding is a projection error, never a
	// silent absence: the store surfaces the error so the API projects the
	// lane explicitly instead of falling back to detached generic progress.
	if _, err = s.db.ExecContext(ctx, `UPDATE factory_feed_comments SET data = data || '{"product_change": "bogus"}' WHERE id=$1`, id); err != nil {
		t.Fatal(err)
	}
	if _, err = s.CommentProductChange(ctx, detail, pipe, 42); err == nil {
		t.Fatal("corrupted product binding reads as a successful projection")
	}
}

func TestCommentProductLineageAdoptedWorkAndVerification(t *testing.T) {
	s, _ := postgresFixture(t, nil)
	ctx := context.Background()
	idea := factory.ProductIdea{ID: "d888888888888888888888888", Description: "Track requests", Consequence: "Requests persist", Decision: "accept", Source: "requested", Provenance: "human", Actor: 42}
	accepted := factory.IntentRevision{ID: "d111111111111111111111111", Intent: "d000000000000000000000000", Owner: 42, Repository: 7, Objective: "Track equipment requests", State: "accepted", MaxIssues: 10, CreatedUnix: 1, Ideas: []factory.ProductIdea{idea}, Tasks: []factory.IntentTask{{Key: "d999999999999999999999999", Objective: "Record requests", Outcomes: []string{"Requests persist"}, IdeaIDs: []string{idea.ID}}}}
	if err := s.SaveIntentRevision(ctx, accepted); err != nil {
		t.Fatal(err)
	}
	var post string
	if err := s.db.QueryRowContext(ctx, `SELECT id FROM factory_feed_comments WHERE data->>'event_identity'=$1`, "intent-revision:"+accepted.ID).Scan(&post); err != nil {
		t.Fatal(err)
	}
	source, err := s.ResolveCommentSource(ctx, 7, factory.CommentSourcePost, post)
	if err != nil {
		t.Fatal(err)
	}
	save := func(key string, via *factory.CommentSource) (string, factory.CommentPipeline, FeedCommentDetail) {
		t.Helper()
		id, _, err := s.SaveFeedCommentWithSource(ctx, factory.CommentRequest{Repository: 7, Body: "Require Finance approval above EUR 1000; retain existing requests.", SourceKind: via.Kind, SourceID: via.ID, IdempotencyKey: key}, "43", "Robin", 2, via)
		if err != nil {
			t.Fatal(err)
		}
		if err = s.RecordCommentIntake(ctx, id, 0); err != nil {
			t.Fatal(err)
		}
		pipe, err := s.CommentPipeline(ctx, id)
		if err != nil {
			t.Fatal(err)
		}
		next := pipe
		next.Judgment, next.Route = factory.CommentJudgmentApplied, factory.CommentRouteNewIssue
		if err = s.AdvanceCommentPipeline(ctx, pipe, next); err != nil {
			t.Fatal(err)
		}
		pipe, err = s.CommentPipeline(ctx, id)
		if err != nil {
			t.Fatal(err)
		}
		detail, err := s.FeedComment(ctx, id)
		if err != nil {
			t.Fatal(err)
		}
		return id, pipe, detail
	}
	// A pipeline that leaves the same-product lane no longer speaks for
	// the accepted direction, and its review lane refuses with it. This
	// detour runs on its own intent so the main review head stays put.
	detourAccepted := accepted
	detourAccepted.ID, detourAccepted.Intent = "d101010101010101010101010", "d000000000000000000000001"
	if err := s.SaveIntentRevision(ctx, detourAccepted); err != nil {
		t.Fatal(err)
	}
	var detourPost string
	if err := s.db.QueryRowContext(ctx, `SELECT id FROM factory_feed_comments WHERE data->>'event_identity'=$1`, "intent-revision:"+detourAccepted.ID).Scan(&detourPost); err != nil {
		t.Fatal(err)
	}
	detourSource, err := s.ResolveCommentSource(ctx, 7, factory.CommentSourcePost, detourPost)
	if err != nil {
		t.Fatal(err)
	}
	second, secondPipe, _ := save("finance-detour", detourSource)
	if _, err = s.ReviewCommentProductChange(ctx, second, 0, 42, 3); err != nil {
		t.Fatal(err)
	}
	detour := secondPipe
	detour.Issue = 7
	if err = s.AdvanceCommentPipeline(ctx, secondPipe, detour); err != nil {
		t.Fatal(err)
	}
	detourPipe, err := s.CommentPipeline(ctx, second)
	if err != nil || detourPipe.Issue != 7 {
		t.Fatalf("detour issue missing: %+v %v", detourPipe, err)
	}
	detourDetail, err := s.FeedComment(ctx, second)
	if err != nil {
		t.Fatal(err)
	}
	detourView, err := s.CommentProductChange(ctx, detourDetail, detourPipe, 42)
	if err != nil || detourView.State != "superseded" || detourView.CanReview || detourView.EvidenceStatus != CommentProductEvidenceUnavailable {
		t.Fatalf("detoured pipeline keeps product authority: %+v %v", detourView, err)
	}
	if _, err = s.ReviewCommentProductChange(ctx, second, 0, 42, 4); err == nil {
		t.Fatal("detoured pipeline admitted a same-product review")
	}
	id, pipe, detail := save("finance-lineage", source)
	draft, err := s.ReviewCommentProductChange(ctx, id, 0, 42, 3)
	if err != nil {
		t.Fatal(err)
	}
	commentIdea := draft.Ideas[1].ID
	if commentIdea != factory.CommentDecisionID("product-idea", id, 0) {
		t.Fatalf("comment idea is not deterministic: %s", commentIdea)
	}
	view, err := s.CommentProductChange(ctx, detail, pipe, 42)
	if err != nil || view.State != "draft" || view.EvidenceStatus != CommentProductEvidenceUnavailable || view.AdoptedRevision != "" || view.Verified || len(view.Evidence) != 0 {
		t.Fatalf("draft claims work evidence: %+v %v", view, err)
	}
	unrelated := accepted
	unrelated.ID, unrelated.Predecessor = "d444444444444444444444444", draft.ID
	if err = s.SaveIntentRevision(ctx, unrelated); err != nil {
		t.Fatal(err)
	}
	view, err = s.CommentProductChange(ctx, detail, pipe, 42)
	if err != nil || view.State != "superseded" || view.AdoptedRevision != "" || view.EvidenceStatus != CommentProductEvidenceUnavailable {
		t.Fatalf("unrelated accepted revision claims adoption: %+v %v", view, err)
	}
	adopted := draft
	adopted.ID, adopted.Predecessor, adopted.State = "d222222222222222222222222", unrelated.ID, "accepted"
	adopted.EffectiveOutcomes = []string{idea.ID, commentIdea}
	financeTask := factory.IntentTask{Key: "d777777777777777777777777", Objective: "Finance approval", Outcomes: []string{"Threshold is applied"}, IdeaIDs: []string{commentIdea}}
	adopted.Tasks = append(append([]factory.IntentTask(nil), draft.Tasks...), financeTask)
	if err = s.SaveIntentRevision(ctx, adopted); err != nil {
		t.Fatal(err)
	}
	head, err := s.IntentRevision(ctx, accepted.Intent, 42)
	if err != nil || head.ID != adopted.ID {
		t.Fatalf("adoption missed the head: %+v %v", head, err)
	}
	view, err = s.CommentProductChange(ctx, detail, pipe, 42)
	if err != nil || view.State != "accepted" || view.AdoptedRevision != adopted.ID || view.ScopeDigest != adopted.Digest() {
		t.Fatalf("adoption without lineage: %+v %v", view, err)
	}
	if view.EvidenceStatus != CommentProductEvidenceUnavailable || view.Verified || len(view.Evidence) != 0 {
		t.Fatalf("undelivered adoption claims work: %+v", view)
	}
	if pipe.Issue != 0 {
		t.Fatalf("same-product pipeline left its zero issue: %+v", pipe)
	}
	// The canonical graph behind the adopted revision carries both tasks;
	// only the comment idea's key is lineage.
	generation := factory.IntentGeneration{RevisionID: adopted.ID, IntentID: adopted.Intent, IntentDigest: adopted.Digest(), OperationID: "lineage-op", ActorID: 42, NativeRev: 1, NotAfter: 3, Payload: []byte(`{}`), State: "pending", SubmittedTasks: adopted.Tasks}
	if err = s.BeginIntentGeneration(ctx, generation); err != nil {
		t.Fatal(err)
	}
	generation.State = "completed"
	if err = s.RecordIntentGeneration(ctx, generation); err != nil {
		t.Fatal(err)
	}
	originalBinding := factory.IntentTaskBinding{RevisionID: adopted.ID, TaskKey: adopted.Tasks[0].Key, IssueID: 100, IssueIndex: 11, AcceptanceID: "d555555555555555555555555", CreatedUnix: 4}
	matchedBinding := factory.IntentTaskBinding{RevisionID: adopted.ID, TaskKey: financeTask.Key, IssueID: 101, IssueIndex: 12, AcceptanceID: "d666666666666666666666666", CreatedUnix: 4}
	if err = s.BindIntentTaskAcceptance(ctx, originalBinding); err != nil {
		t.Fatal(err)
	}
	if err = s.BindIntentTaskAcceptance(ctx, matchedBinding); err != nil {
		t.Fatal(err)
	}
	// One completion row exists per repository generation; each task's
	// canonical completion is the latest generation for its issue and
	// acceptance.
	complete := func(binding factory.IntentTaskBinding, generation int64) {
		t.Helper()
		event := factory.CompletionEvent{ID: factory.NewID(), Repository: 7, Issue: binding.IssueIndex, Generation: generation, PublicationID: factory.NewID(), AcceptanceID: binding.AcceptanceID, MergedCommit: strings.Repeat("a", 40), ClosingOccurrence: "1"}
		raw, err := json.Marshal(event)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = s.db.ExecContext(ctx, `INSERT INTO factory_completions(repository,generation,event,publication,occurrence,data) VALUES(7,$1,$2,$3,'1',$4)`, generation, event.ID, event.PublicationID, string(raw)); err != nil {
			t.Fatal(err)
		}
	}
	complete(originalBinding, 1)
	complete(matchedBinding, 2)
	view, err = s.CommentProductChange(ctx, detail, pipe, 42)
	if err != nil || view.State != "accepted" || view.EvidenceStatus != CommentProductEvidenceRecorded || view.Verified {
		t.Fatalf("delivered adoption unread: %+v %v", view, err)
	}
	if len(view.Evidence) != 1 || view.Evidence[0].TaskKey != financeTask.Key || view.Evidence[0].Issue != matchedBinding.IssueIndex || view.Evidence[0].Acceptance != matchedBinding.AcceptanceID || view.Evidence[0].Completion == "" {
		t.Fatalf("lineage misses the comment task: %+v", view.Evidence)
	}
	// Only the controller-owned F10 fact verifies the adopted scope: a
	// closed native issue or a worker exit never flips this bit.
	delivery := factory.NewID()
	assignmentID := factory.NewID()
	assignment, err := json.Marshal(map[string]any{
		"id": assignmentID, "attempt_root": assignmentID, "repository": "7", "issue": 0, "run": factory.NewID(), "stage": "finished",
		"phase": "verify", "role": "soda-reviewer", "outcome": "succeeded", "source_commit": strings.Repeat("b", 40),
		"subject": map[string]any{"kind": "delivery", "delivery": map[string]any{"delivery_id": delivery, "intent_id": adopted.Intent, "accepted_revision": adopted.ID, "scope_digest": adopted.Digest()}},
		"result":  map[string]any{"phase_result": map[string]any{"output": map[string]any{"verification": map[string]any{"outcome": "verified"}}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.db.ExecContext(ctx, `INSERT INTO factory_assignments(id,repository,issue,run,stage,revision,data) VALUES($1,7,0,$2,'finished',0,$3)`, assignmentID, factory.NewID(), string(assignment)); err != nil {
		t.Fatal(err)
	}
	postRaw, err := json.Marshal(map[string]any{"actor": "factory", "event_identity": delivery, "assignment": assignmentID, "communication": map[string]any{"component_id": "F10"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.db.ExecContext(ctx, `INSERT INTO factory_feed_comments(id,repository,revision,author,body,idempotency,data) VALUES($1,7,0,'factory','verified',$2,$3)`, factory.NewID(), "f10-"+delivery, string(postRaw)); err != nil {
		t.Fatal(err)
	}
	view, err = s.CommentProductChange(ctx, detail, pipe, 42)
	if err != nil || !view.Verified || view.EvidenceStatus != CommentProductEvidenceRecorded || len(view.Evidence) != 1 {
		t.Fatalf("controller verification missing: %+v %v", view, err)
	}
	// A later superseding head keeps the adopted lineage as history.
	removed := adopted
	removed.ID, removed.Predecessor = "d333333333333333333333333", adopted.ID
	removed.Ideas = append([]factory.ProductIdea(nil), adopted.Ideas...)
	removed.Ideas[1].Decision = "decline"
	removed.EffectiveOutcomes = []string{idea.ID}
	removed.Tasks = accepted.Tasks
	if err = s.SaveIntentRevision(ctx, removed); err != nil {
		t.Fatal(err)
	}
	view, err = s.CommentProductChange(ctx, detail, pipe, 42)
	if err != nil || view.State != "superseded" || view.CanReview {
		t.Fatalf("superseded adoption still authoritative: %+v %v", view, err)
	}
	if view.AdoptedRevision != adopted.ID || view.EvidenceStatus != CommentProductEvidenceRecorded || !view.Verified || len(view.Evidence) != 1 || view.Evidence[0].TaskKey != financeTask.Key {
		t.Fatalf("historical adoption lost: %+v", view)
	}
}

func TestCommentProductLineageKeepsCompletedWorkWhenScopePartial(t *testing.T) {
	s, _ := postgresFixture(t, nil)
	ctx := context.Background()
	idea := factory.ProductIdea{ID: "d888888888888888888888888", Description: "Track requests", Consequence: "Requests persist", Decision: "accept", Source: "requested", Provenance: "human", Actor: 42}
	accepted := factory.IntentRevision{ID: "d111111111111111111111111", Intent: "d000000000000000000000000", Owner: 42, Repository: 7, Objective: "Track equipment requests", State: "accepted", MaxIssues: 10, CreatedUnix: 1, Ideas: []factory.ProductIdea{idea}, Tasks: []factory.IntentTask{{Key: "d999999999999999999999999", Objective: "Record requests", Outcomes: []string{"Requests persist"}, IdeaIDs: []string{idea.ID}}}}
	if err := s.SaveIntentRevision(ctx, accepted); err != nil {
		t.Fatal(err)
	}
	var post string
	if err := s.db.QueryRowContext(ctx, `SELECT id FROM factory_feed_comments WHERE data->>'event_identity'=$1`, "intent-revision:"+accepted.ID).Scan(&post); err != nil {
		t.Fatal(err)
	}
	source, err := s.ResolveCommentSource(ctx, 7, factory.CommentSourcePost, post)
	if err != nil {
		t.Fatal(err)
	}
	id, _, err := s.SaveFeedCommentWithSource(ctx, factory.CommentRequest{Repository: 7, Body: "Require Finance approval above EUR 1000; retain existing requests.", SourceKind: source.Kind, SourceID: source.ID, IdempotencyKey: "finance-partial"}, "43", "Robin", 2, source)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.RecordCommentIntake(ctx, id, 0); err != nil {
		t.Fatal(err)
	}
	pipe, err := s.CommentPipeline(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	next := pipe
	next.Judgment, next.Route = factory.CommentJudgmentApplied, factory.CommentRouteNewIssue
	if err = s.AdvanceCommentPipeline(ctx, pipe, next); err != nil {
		t.Fatal(err)
	}
	pipe, err = s.CommentPipeline(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	detail, err := s.FeedComment(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	draft, err := s.ReviewCommentProductChange(ctx, id, 0, 42, 3)
	if err != nil {
		t.Fatal(err)
	}
	commentIdea := draft.Ideas[1].ID
	adopted := draft
	adopted.ID, adopted.Predecessor, adopted.State = "d222222222222222222222222", draft.ID, "accepted"
	adopted.EffectiveOutcomes = []string{idea.ID, commentIdea}
	financeTask := factory.IntentTask{Key: "d777777777777777777777777", Objective: "Finance approval", Outcomes: []string{"Threshold is applied"}, IdeaIDs: []string{commentIdea}}
	adopted.Tasks = append(append([]factory.IntentTask(nil), draft.Tasks...), financeTask)
	if err = s.SaveIntentRevision(ctx, adopted); err != nil {
		t.Fatal(err)
	}
	// The canonical graph behind the adopted revision carries both tasks;
	// only the comment idea's key is lineage.
	generation := factory.IntentGeneration{RevisionID: adopted.ID, IntentID: adopted.Intent, IntentDigest: adopted.Digest(), OperationID: "partial-op", ActorID: 42, NativeRev: 1, NotAfter: 3, Payload: []byte(`{}`), State: "pending", SubmittedTasks: adopted.Tasks}
	if err = s.BeginIntentGeneration(ctx, generation); err != nil {
		t.Fatal(err)
	}
	generation.State = "completed"
	if err = s.RecordIntentGeneration(ctx, generation); err != nil {
		t.Fatal(err)
	}
	unrelatedBinding := factory.IntentTaskBinding{RevisionID: adopted.ID, TaskKey: adopted.Tasks[0].Key, IssueID: 100, IssueIndex: 11, AcceptanceID: "d555555555555555555555555", CreatedUnix: 4}
	matchedBinding := factory.IntentTaskBinding{RevisionID: adopted.ID, TaskKey: financeTask.Key, IssueID: 101, IssueIndex: 12, AcceptanceID: "d666666666666666666666666", CreatedUnix: 4}
	if err = s.BindIntentTaskAcceptance(ctx, unrelatedBinding); err != nil {
		t.Fatal(err)
	}
	if err = s.BindIntentTaskAcceptance(ctx, matchedBinding); err != nil {
		t.Fatal(err)
	}
	// Mixed scope: the comment idea's task completes while the unrelated
	// task in the same scope stays pending with no completion row.
	event := factory.CompletionEvent{ID: factory.NewID(), Repository: 7, Issue: matchedBinding.IssueIndex, Generation: 2, PublicationID: factory.NewID(), AcceptanceID: matchedBinding.AcceptanceID, MergedCommit: strings.Repeat("a", 40), ClosingOccurrence: "1"}
	raw, err := json.Marshal(event)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.db.ExecContext(ctx, `INSERT INTO factory_completions(repository,generation,event,publication,occurrence,data) VALUES(7,$1,$2,$3,'1',$4)`, 2, event.ID, event.PublicationID, string(raw)); err != nil {
		t.Fatal(err)
	}
	view, err := s.CommentProductChange(ctx, detail, pipe, 42)
	if err != nil {
		t.Fatal(err)
	}
	if view.State != "accepted" || view.EvidenceStatus != CommentProductEvidenceRecorded {
		t.Fatalf("partial scope hides completed comment work: %+v", view)
	}
	if len(view.Evidence) != 1 || view.Evidence[0].TaskKey != financeTask.Key || view.Evidence[0].Issue != matchedBinding.IssueIndex || view.Evidence[0].Acceptance != matchedBinding.AcceptanceID || view.Evidence[0].Completion == "" {
		t.Fatalf("partial lineage misses the completed comment task: %+v", view.Evidence)
	}
}

func TestCommentProductLineageShowsPendingMatchingTask(t *testing.T) {
	s, _ := postgresFixture(t, nil)
	ctx := context.Background()
	idea := factory.ProductIdea{ID: "d888888888888888888888888", Description: "Track requests", Consequence: "Requests persist", Decision: "accept", Source: "requested", Provenance: "human", Actor: 42}
	accepted := factory.IntentRevision{ID: "d111111111111111111111111", Intent: "d000000000000000000000000", Owner: 42, Repository: 7, Objective: "Track equipment requests", State: "accepted", MaxIssues: 10, CreatedUnix: 1, Ideas: []factory.ProductIdea{idea}, Tasks: []factory.IntentTask{{Key: "d999999999999999999999999", Objective: "Record requests", Outcomes: []string{"Requests persist"}, IdeaIDs: []string{idea.ID}}}}
	if err := s.SaveIntentRevision(ctx, accepted); err != nil {
		t.Fatal(err)
	}
	var post string
	if err := s.db.QueryRowContext(ctx, `SELECT id FROM factory_feed_comments WHERE data->>'event_identity'=$1`, "intent-revision:"+accepted.ID).Scan(&post); err != nil {
		t.Fatal(err)
	}
	source, err := s.ResolveCommentSource(ctx, 7, factory.CommentSourcePost, post)
	if err != nil {
		t.Fatal(err)
	}
	id, _, err := s.SaveFeedCommentWithSource(ctx, factory.CommentRequest{Repository: 7, Body: "Require Finance approval above EUR 1000; retain existing requests.", SourceKind: source.Kind, SourceID: source.ID, IdempotencyKey: "finance-pending"}, "43", "Robin", 2, source)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.RecordCommentIntake(ctx, id, 0); err != nil {
		t.Fatal(err)
	}
	pipe, err := s.CommentPipeline(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	next := pipe
	next.Judgment, next.Route = factory.CommentJudgmentApplied, factory.CommentRouteNewIssue
	if err = s.AdvanceCommentPipeline(ctx, pipe, next); err != nil {
		t.Fatal(err)
	}
	pipe, err = s.CommentPipeline(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	detail, err := s.FeedComment(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	draft, err := s.ReviewCommentProductChange(ctx, id, 0, 42, 3)
	if err != nil {
		t.Fatal(err)
	}
	commentIdea := draft.Ideas[1].ID
	adopted := draft
	adopted.ID, adopted.Predecessor, adopted.State = "d222222222222222222222222", draft.ID, "accepted"
	adopted.EffectiveOutcomes = []string{idea.ID, commentIdea}
	financeTask := factory.IntentTask{Key: "d777777777777777777777777", Objective: "Finance approval", Outcomes: []string{"Threshold is applied"}, IdeaIDs: []string{commentIdea}}
	adopted.Tasks = append(append([]factory.IntentTask(nil), draft.Tasks...), financeTask)
	if err = s.SaveIntentRevision(ctx, adopted); err != nil {
		t.Fatal(err)
	}
	generation := factory.IntentGeneration{RevisionID: adopted.ID, IntentID: adopted.Intent, IntentDigest: adopted.Digest(), OperationID: "pending-op", ActorID: 42, NativeRev: 1, NotAfter: 3, Payload: []byte(`{}`), State: "pending", SubmittedTasks: adopted.Tasks}
	if err = s.BeginIntentGeneration(ctx, generation); err != nil {
		t.Fatal(err)
	}
	generation.State = "completed"
	if err = s.RecordIntentGeneration(ctx, generation); err != nil {
		t.Fatal(err)
	}
	// The matching task has its exact native binding but no completion
	// row: actual progress stays visible as pending without inferring
	// delivery or completion.
	matchedBinding := factory.IntentTaskBinding{RevisionID: adopted.ID, TaskKey: financeTask.Key, IssueID: 101, IssueIndex: 12, AcceptanceID: "d666666666666666666666666", CreatedUnix: 4}
	if err = s.BindIntentTaskAcceptance(ctx, matchedBinding); err != nil {
		t.Fatal(err)
	}
	view, err := s.CommentProductChange(ctx, detail, pipe, 42)
	if err != nil {
		t.Fatal(err)
	}
	if view.State != "accepted" || view.EvidenceStatus != CommentProductEvidenceRecorded {
		t.Fatalf("pending matching task hides comment evidence: %+v", view)
	}
	if view.AdoptedRevision != adopted.ID || view.ScopeDigest != adopted.Digest() {
		t.Fatalf("pending adoption without lineage: %+v", view)
	}
	// No F10 verification event exists here, so whole-product
	// verification stays strict.
	if view.Verified {
		t.Fatalf("pending work verifies the product without an F10 fact: %+v", view)
	}
	if len(view.Evidence) != 1 {
		t.Fatalf("pending lineage misses the bound comment task: %+v", view.Evidence)
	}
	got := view.Evidence[0]
	if got.TaskKey != financeTask.Key || got.Issue != matchedBinding.IssueIndex || got.Acceptance != matchedBinding.AcceptanceID {
		t.Fatalf("pending lineage loses the native binding: %+v", view.Evidence)
	}
	if got.Completion != "" || !got.Pending {
		t.Fatalf("bound task without completion is not marked pending: %+v", view.Evidence)
	}
}

func TestCommentProductLineageShowsMixedMatchingTasks(t *testing.T) {
	s, _ := postgresFixture(t, nil)
	ctx := context.Background()
	idea := factory.ProductIdea{ID: "d888888888888888888888888", Description: "Track requests", Consequence: "Requests persist", Decision: "accept", Source: "requested", Provenance: "human", Actor: 42}
	accepted := factory.IntentRevision{ID: "d111111111111111111111111", Intent: "d000000000000000000000000", Owner: 42, Repository: 7, Objective: "Track equipment requests", State: "accepted", MaxIssues: 10, CreatedUnix: 1, Ideas: []factory.ProductIdea{idea}, Tasks: []factory.IntentTask{{Key: "d999999999999999999999999", Objective: "Record requests", Outcomes: []string{"Requests persist"}, IdeaIDs: []string{idea.ID}}}}
	if err := s.SaveIntentRevision(ctx, accepted); err != nil {
		t.Fatal(err)
	}
	var post string
	if err := s.db.QueryRowContext(ctx, `SELECT id FROM factory_feed_comments WHERE data->>'event_identity'=$1`, "intent-revision:"+accepted.ID).Scan(&post); err != nil {
		t.Fatal(err)
	}
	source, err := s.ResolveCommentSource(ctx, 7, factory.CommentSourcePost, post)
	if err != nil {
		t.Fatal(err)
	}
	id, _, err := s.SaveFeedCommentWithSource(ctx, factory.CommentRequest{Repository: 7, Body: "Require Finance approval above EUR 1000; retain existing requests.", SourceKind: source.Kind, SourceID: source.ID, IdempotencyKey: "finance-mixed"}, "43", "Robin", 2, source)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.RecordCommentIntake(ctx, id, 0); err != nil {
		t.Fatal(err)
	}
	pipe, err := s.CommentPipeline(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	next := pipe
	next.Judgment, next.Route = factory.CommentJudgmentApplied, factory.CommentRouteNewIssue
	if err = s.AdvanceCommentPipeline(ctx, pipe, next); err != nil {
		t.Fatal(err)
	}
	pipe, err = s.CommentPipeline(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	detail, err := s.FeedComment(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	draft, err := s.ReviewCommentProductChange(ctx, id, 0, 42, 3)
	if err != nil {
		t.Fatal(err)
	}
	commentIdea := draft.Ideas[1].ID
	adopted := draft
	adopted.ID, adopted.Predecessor, adopted.State = "d222222222222222222222222", draft.ID, "accepted"
	adopted.EffectiveOutcomes = []string{idea.ID, commentIdea}
	firstTask := factory.IntentTask{Key: "d777777777777777777777777", Objective: "Finance approval", Outcomes: []string{"Threshold is applied"}, IdeaIDs: []string{commentIdea}}
	secondTask := factory.IntentTask{Key: "d707070707070707070707070", Objective: "Finance audit trail", Outcomes: []string{"Approvals are listed"}, IdeaIDs: []string{commentIdea}}
	adopted.Tasks = append(append(append([]factory.IntentTask(nil), draft.Tasks...), firstTask), secondTask)
	if err = s.SaveIntentRevision(ctx, adopted); err != nil {
		t.Fatal(err)
	}
	generation := factory.IntentGeneration{RevisionID: adopted.ID, IntentID: adopted.Intent, IntentDigest: adopted.Digest(), OperationID: "mixed-op", ActorID: 42, NativeRev: 1, NotAfter: 3, Payload: []byte(`{}`), State: "pending", SubmittedTasks: adopted.Tasks}
	if err = s.BeginIntentGeneration(ctx, generation); err != nil {
		t.Fatal(err)
	}
	generation.State = "completed"
	if err = s.RecordIntentGeneration(ctx, generation); err != nil {
		t.Fatal(err)
	}
	firstBinding := factory.IntentTaskBinding{RevisionID: adopted.ID, TaskKey: firstTask.Key, IssueID: 101, IssueIndex: 12, AcceptanceID: "d666666666666666666666666", CreatedUnix: 4}
	secondBinding := factory.IntentTaskBinding{RevisionID: adopted.ID, TaskKey: secondTask.Key, IssueID: 102, IssueIndex: 13, AcceptanceID: "d616161616161616161616161", CreatedUnix: 4}
	if err = s.BindIntentTaskAcceptance(ctx, firstBinding); err != nil {
		t.Fatal(err)
	}
	if err = s.BindIntentTaskAcceptance(ctx, secondBinding); err != nil {
		t.Fatal(err)
	}
	// One matching task completes while the other bound matching task
	// stays pending with no completion row: neither delivery nor
	// completion is inferred for the pending task.
	event := factory.CompletionEvent{ID: factory.NewID(), Repository: 7, Issue: firstBinding.IssueIndex, Generation: 2, PublicationID: factory.NewID(), AcceptanceID: firstBinding.AcceptanceID, MergedCommit: strings.Repeat("a", 40), ClosingOccurrence: "1"}
	raw, err := json.Marshal(event)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.db.ExecContext(ctx, `INSERT INTO factory_completions(repository,generation,event,publication,occurrence,data) VALUES(7,$1,$2,$3,'1',$4)`, 2, event.ID, event.PublicationID, string(raw)); err != nil {
		t.Fatal(err)
	}
	view, err := s.CommentProductChange(ctx, detail, pipe, 42)
	if err != nil {
		t.Fatal(err)
	}
	if view.State != "accepted" || view.EvidenceStatus != CommentProductEvidenceRecorded {
		t.Fatalf("mixed matching tasks hide comment evidence: %+v", view)
	}
	// No F10 verification event exists here, so whole-product
	// verification stays strict.
	if view.Verified {
		t.Fatalf("mixed work verifies the product without an F10 fact: %+v", view)
	}
	if len(view.Evidence) != 2 {
		t.Fatalf("mixed lineage misses a matching task: %+v", view.Evidence)
	}
	byKey := map[string]CommentProductTaskEvidence{}
	for _, task := range view.Evidence {
		byKey[task.TaskKey] = task
	}
	completed, ok := byKey[firstTask.Key]
	if !ok || completed.Issue != firstBinding.IssueIndex || completed.Acceptance != firstBinding.AcceptanceID || completed.Completion == "" || completed.Pending {
		t.Fatalf("completed matching task is not exact: %+v", view.Evidence)
	}
	pending, ok := byKey[secondTask.Key]
	if !ok || pending.Issue != secondBinding.IssueIndex || pending.Acceptance != secondBinding.AcceptanceID || pending.Completion != "" || !pending.Pending {
		t.Fatalf("pending matching task is not marked pending: %+v", view.Evidence)
	}
}
