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
	s, _ := postgresFixture(t)
	parent := saveTestPost(t, s, 7, "42", "parent", "parent")
	for i := 0; i < 23; i++ {
		_, _, err := s.SaveFeedPost(ctx, factory.PostRequest{Repository: 7, Body: "reply", ParentID: parent, IdempotencyKey: fmt.Sprint(i)}, "43", "Another person", 1729000001)
		if err != nil {
			t.Fatal(err)
		}
	}
	_, _, err := s.SaveFeedPost(ctx, factory.PostRequest{Repository: 8, Body: "other project", ParentID: parent, IdempotencyKey: "foreign"}, "44", "Other", 1729000002)
	if err == nil {
		t.Fatal("cross-repository parent admitted")
	}
	first, next, err := s.FeedReplyIDs(ctx, 7, "post", parent, 0)
	if err != nil || len(first) != 20 || next == 0 {
		t.Fatalf("first page: %d %d %v", len(first), next, err)
	}
	second, next, err := s.FeedReplyIDs(ctx, 7, "post", parent, next)
	if err != nil || len(second) != 3 || next != 0 {
		t.Fatalf("second page: %d %d %v", len(second), next, err)
	}
	grandchild, _, err := s.SaveFeedPost(ctx, factory.PostRequest{Repository: 7, Body: "nested reply", ParentID: first[0], IdempotencyKey: "nested"}, "43", "Another person", 1729000003)
	if err != nil {
		t.Fatal(err)
	}
	children, _, err := s.FeedReplyIDs(ctx, 7, "post", first[0], 0)
	if err != nil || len(children) != 1 || children[0] != grandchild {
		t.Fatalf("direct children: %v %v", children, err)
	}
	rootChildren, _, err := s.FeedReplyIDs(ctx, 7, "post", parent, 0)
	if err != nil || len(rootChildren) != 20 {
		t.Fatalf("root page: %v %v", rootChildren, err)
	}
	for _, child := range rootChildren {
		if child == grandchild {
			t.Fatal("grandchild flattened under root")
		}
	}
	seen := map[string]bool{}
	for _, id := range append(first, second...) {
		if seen[id] {
			t.Fatal("reply duplicated across pages")
		}
		seen[id] = true
	}
	if _, err := s.DeleteFeedPost(ctx, second[0], "43", 0); err != nil {
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

func saveTestPost(t *testing.T, s *Store, repository int64, author, body, key string) string {
	t.Helper()
	id, revision, err := s.SaveFeedPost(context.Background(), factory.PostRequest{
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
	s, _ := postgresFixture(t)
	ctx := context.Background()
	id := saveTestPost(t, s, 7, "42", "existing work", "old-feedback")
	if err := s.RecordPostIntake(ctx, id, 0); err != nil {
		t.Fatal(err)
	}
	pipe, err := s.PostIntake(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	next := pipe
	next.Judgment = factory.PostJudgmentApplied
	next.Route = factory.PostRouteExistingWork
	next.Issue = 55
	if err = s.AdvancePostIntake(ctx, pipe, next); err != nil {
		t.Fatal(err)
	}
	queued, err := s.PendingPostIntakes(ctx, 4)
	if err != nil || len(queued) != 1 {
		t.Fatalf("newly assessed feedback absent: %+v %v", queued, err)
	}
	if _, err = s.db.ExecContext(ctx, `UPDATE factory_post_intakes SET data=data-'feedback_revision' WHERE post=$1`, id); err != nil {
		t.Fatal(err)
	}
	queued, err = s.PendingPostIntakes(ctx, 4)
	if err != nil || len(queued) != 0 {
		t.Fatalf("historical mapping replayed: %+v %v", queued, err)
	}
}

func TestFeedPostSaveReplaysIdempotencyKey(t *testing.T) {
	ctx := context.Background()
	s, _ := postgresFixture(t)
	first := saveTestPost(t, s, 7, "42", "please fix the login redirect", "key-1")
	again, revision, err := s.SaveFeedPost(ctx, factory.PostRequest{
		Repository: 7, Body: "a different body", IdempotencyKey: "key-1",
	}, "42", "ada", 1729000060)
	if err != nil {
		t.Fatal(err)
	}
	if again != first || revision != 0 {
		t.Fatalf("replay = %q rev %d, want %q rev 0", again, revision, first)
	}
	detail, err := s.FeedPost(ctx, first)
	if err != nil {
		t.Fatal(err)
	}
	if detail.Body != "please fix the login redirect" || detail.AuthorName != "42-name" || detail.At != 1729000000 {
		t.Fatalf("replay rewrote the original: %+v", detail)
	}
	// A different key from the same author is a new comment.
	other := saveTestPost(t, s, 7, "42", "another thought", "key-2")
	if other == first {
		t.Fatal("distinct key collided")
	}
}

func TestFeedPostEditDeleteRevoke(t *testing.T) {
	ctx := context.Background()
	s, _ := postgresFixture(t)
	id := saveTestPost(t, s, 7, "42", "original", "key-1")
	if err := s.RecordPostIntake(ctx, id, 0); err != nil {
		t.Fatal(err)
	}
	revision, err := s.EditFeedPost(ctx, id, "42", "edited text", 0, 1729000100)
	if err != nil || revision != 1 {
		t.Fatalf("edit = %d %v", revision, err)
	}
	detail, err := s.FeedPost(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if detail.Body != "edited text" || !detail.Record.Edited || detail.Record.Revision != 1 {
		t.Fatalf("edited detail: %+v", detail)
	}
	pipe, err := s.PostIntake(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if pipe.Revision != 1 {
		t.Fatalf("pipeline not rebound: %+v", pipe)
	}
	// Stale revision refuses.
	if _, err := s.EditFeedPost(ctx, id, "42", "stale", 0, 1729000200); err == nil {
		t.Fatal("stale edit admitted")
	}
	// Foreign author refuses.
	if _, err := s.EditFeedPost(ctx, id, "43", "hijack", 1, 1729000200); err == nil {
		t.Fatal("foreign edit admitted")
	}
	// Delete tombstones and blocks the unsettled effect.
	revision, err = s.DeleteFeedPost(ctx, id, "42", 1)
	if err != nil || revision != 2 {
		t.Fatalf("delete = %d %v", revision, err)
	}
	pipe, err = s.PostIntake(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if pipe.Effect != factory.PostEffectBlocked {
		t.Fatalf("deleted effect = %q", pipe.Effect)
	}
	// Deleted sources refuse edits but still resolve.
	if _, err := s.EditFeedPost(ctx, id, "42", "resurrect", 2, 1729000300); err == nil {
		t.Fatal("edit after delete admitted")
	}
	if _, err := s.FeedPost(ctx, id); err != nil {
		t.Fatal(err)
	}
	// Revoke blocks the unsettled effect on a second comment.
	second := saveTestPost(t, s, 7, "42", "second", "key-2")
	if err := s.RecordPostIntake(ctx, second, 0); err != nil {
		t.Fatal(err)
	}
	if err := s.MarkFeedPostRevoked(ctx, second); err != nil {
		t.Fatal(err)
	}
	pipe, err = s.PostIntake(ctx, second)
	if err != nil {
		t.Fatal(err)
	}
	if pipe.Effect != factory.PostEffectBlocked {
		t.Fatalf("revoked effect = %q", pipe.Effect)
	}
}

func TestPostPipelineAdvanceIsCAS(t *testing.T) {
	ctx := context.Background()
	s, _ := postgresFixture(t)
	id := saveTestPost(t, s, 7, "42", "route me", "key-1")
	if err := s.RecordPostIntake(ctx, id, 0); err != nil {
		t.Fatal(err)
	}
	// Replaying the intake recovers the row.
	if err := s.RecordPostIntake(ctx, id, 0); err != nil {
		t.Fatal(err)
	}
	pipe, err := s.PostIntake(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	next := pipe
	next.Judgment = factory.PostJudgmentApplied
	next.Route = factory.PostRouteDiscussion
	if err := s.AdvancePostIntake(ctx, pipe, next); err != nil {
		t.Fatal(err)
	}
	// The pre-advance snapshot is now stale and refuses.
	stale := next
	stale.Route = factory.PostRouteClarify
	if err := s.AdvancePostIntake(ctx, pipe, stale); err == nil {
		t.Fatal("stale advance admitted")
	}
	// Attempts exhaust at the bound.
	for i := 0; i < factory.PostAttemptLimit; i++ {
		if err := s.NotePostAttempt(ctx, id); err != nil {
			t.Fatalf("attempt %d: %v", i, err)
		}
	}
	if err := s.NotePostAttempt(ctx, id); err == nil {
		t.Fatal("over-limit attempt admitted")
	}
}

func TestPostDraftBindsSourceRevision(t *testing.T) {
	ctx := context.Background()
	s, _ := postgresFixture(t)
	id := saveTestPost(t, s, 7, "42", "needs work", "key-1")
	if err := s.RecordPostIntake(ctx, id, 0); err != nil {
		t.Fatal(err)
	}
	draft := factory.IssueDraft{SourcePost: id, Title: "Fix it", Body: "Change the code.", Objective: "Works.", Acceptance: "Tests pass.", Metadata: `{"work":"soda"}`}
	if err := s.RecordPostDraft(ctx, draft); err != nil {
		t.Fatal(err)
	}
	if _, err := s.PostDraft(ctx, id, 0); err != nil {
		t.Fatal(err)
	}
	// After an edit the retained draft no longer binds.
	if _, err := s.EditFeedPost(ctx, id, "42", "needs work, updated", 0, 1729000100); err != nil {
		t.Fatal(err)
	}
	if _, err := s.PostDraft(ctx, id, 1); err == nil {
		t.Fatal("stale-revision draft served")
	}
	// Malformed drafts refuse before any native effect.
	broken := draft
	broken.Acceptance = ""
	if err := s.RecordPostDraft(ctx, broken); err == nil {
		t.Fatal("malformed draft retained")
	}
}

func TestSubscribedFeedMergesPostsNewestFirst(t *testing.T) {
	ctx := context.Background()
	s, dsn := postgresFixture(t)
	raw := journeySQL(t, dsn)
	acceptance := "d" + strings.Repeat("a", 24)
	insertJourneyAssignment(t, raw, journeyAssignment(strings.Repeat("1", 32), strings.Repeat("1", 32), acceptance, 7, 3, true), "assigned", 1)
	first := saveTestPost(t, s, 7, "42", "first comment", "key-1")
	insertJourneyAssignment(t, raw, journeyAssignment(strings.Repeat("2", 32), strings.Repeat("2", 32), acceptance, 7, 4, true), "assigned", 1)
	second := saveTestPost(t, s, 9, "43", "second comment", "key-9")

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
	if _, err := s.DeleteFeedPost(ctx, first, "42", 0); err != nil {
		t.Fatal(err)
	}
	if err := s.MarkFeedPostRevoked(ctx, second); err != nil {
		t.Fatal(err)
	}
	remaining, _, _, err := s.SubscribedFeed(ctx, factory.FeedQuery{Repositories: []int64{7, 9}, Kinds: []string{factory.FeedKindPost}, Limit: 10})
	if err != nil || len(remaining) != 0 {
		t.Fatalf("closed posts surface: %+v %v", remaining, err)
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

// TestPostSourceResolutionAndRoundTrip proves the authorized
// source snapshot: records and posts resolve inside their repository,
// cross-repository sources refuse as not found, and the persisted
// snapshot reads back on the saved comment.
func TestPostSourceResolutionAndRoundTrip(t *testing.T) {
	ctx := context.Background()
	s, _ := postgresFixture(t)
	recordID := insertTestAssignmentRow(t, s, ctx, 7, 55)

	source, err := s.ResolvePostSource(ctx, 7, factory.PostSourceRecord, recordID)
	if err != nil {
		t.Fatal(err)
	}
	if source.Kind != factory.PostSourceRecord || source.ID != recordID || source.Issue != 55 {
		t.Fatalf("record source: %+v", source)
	}
	if _, err := s.ResolvePostSource(ctx, 8, factory.PostSourceRecord, recordID); err != ErrNotFound {
		t.Fatalf("cross-repository record err = %v, want not found", err)
	}
	if _, err := s.ResolvePostSource(ctx, 7, factory.PostSourceRecord, factory.NewID()); err != ErrNotFound {
		t.Fatalf("unknown record err = %v, want not found", err)
	}

	id, _, err := s.SaveFeedPostWithSource(ctx, factory.PostRequest{
		Repository: 7, Body: "more on the filter", SourceKind: factory.PostSourceRecord,
		SourceID: recordID, IdempotencyKey: "key-source",
	}, "42", "ada", 1729000000, source)
	if err != nil {
		t.Fatal(err)
	}
	detail, err := s.FeedPost(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if detail.Source == nil || detail.Source.ID != recordID || detail.Source.Issue != 55 {
		t.Fatalf("source round trip lost: %+v", detail.Source)
	}
	// A mismatched snapshot refuses before any write.
	mismatch := *source
	mismatch.ID = factory.NewID()
	if _, _, err := s.SaveFeedPostWithSource(ctx, factory.PostRequest{
		Repository: 7, Body: "x", SourceKind: factory.PostSourceRecord,
		SourceID: recordID, IdempotencyKey: "key-mismatch",
	}, "42", "ada", 1729000000, &mismatch); err == nil {
		t.Fatal("mismatched source snapshot admitted")
	}
	// Post sources resolve to the stored post in the same repository.
	post, err := s.ResolvePostSource(ctx, 7, factory.PostSourcePost, id)
	if err != nil {
		t.Fatal(err)
	}
	if post.Summary != "more on the filter" {
		t.Fatalf("post source summary: %+v", post)
	}
}

// TestPendingPostPipelinesAndCandidates proves the service work
// list and the existing-work candidate read: pending requests list
// until they settle, and recent worked issues list newest-first per
// issue inside their repository only.
func TestPendingPostPipelinesAndCandidates(t *testing.T) {
	ctx := context.Background()
	s, _ := postgresFixture(t)
	id := saveTestPost(t, s, 7, "42", "route me", "key-1")
	if err := s.RecordPostIntake(ctx, id, 0); err != nil {
		t.Fatal(err)
	}
	pending, err := s.PendingPostIntakes(ctx, 8)
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 1 || pending[0].SourcePost != id {
		t.Fatalf("pending list: %+v", pending)
	}
	pipe, err := s.PostIntake(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	next := pipe
	next.Judgment = factory.PostJudgmentApplied
	next.Route = factory.PostRouteDiscussion
	if err := s.AdvancePostIntake(ctx, pipe, next); err != nil {
		t.Fatal(err)
	}
	pending, err = s.PendingPostIntakes(ctx, 8)
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 0 {
		t.Fatalf("settled discussion still listed: %+v", pending)
	}
	if _, err := s.PendingPostIntakes(ctx, 0); err == nil {
		t.Fatal("unbounded pending listing admitted")
	}
	exhausted := saveTestPost(t, s, 7, "42", "failed authoring", "exhausted")
	if err := s.RecordPostIntake(ctx, exhausted, 0); err != nil {
		t.Fatal(err)
	}
	exhaustedPipe, err := s.PostIntake(ctx, exhausted)
	if err != nil {
		t.Fatal(err)
	}
	authoring := exhaustedPipe
	authoring.Judgment, authoring.Route = factory.PostJudgmentApplied, factory.PostRouteNewIssue
	if err := s.AdvancePostIntake(ctx, exhaustedPipe, authoring); err != nil {
		t.Fatal(err)
	}
	for range factory.PostAttemptLimit {
		if err := s.NotePostAttempt(ctx, exhausted); err != nil {
			t.Fatal(err)
		}
	}
	details, err := s.FeedPostDetails(ctx, []string{exhausted})
	if err != nil || details[exhausted].Status != "draft:failed" {
		t.Fatalf("feed header hid exhausted authoring: %+v, %v", details, err)
	}
	ready := saveTestPost(t, s, 7, "42", "next request", "next")
	if err := s.RecordPostIntake(ctx, ready, 0); err != nil {
		t.Fatal(err)
	}
	pending, err = s.PendingPostIntakes(ctx, 1)
	if err != nil || len(pending) != 1 || pending[0].SourcePost != ready {
		t.Fatalf("exhausted request consumed bounded service slot: %+v, %v", pending, err)
	}

	insertTestAssignmentRow(t, s, ctx, 7, 55)
	insertTestAssignmentRow(t, s, ctx, 7, 9)
	insertTestAssignmentRow(t, s, ctx, 7, 55)
	insertTestAssignmentRow(t, s, ctx, 8, 77)
	candidates, err := s.RecentPostWorkCandidates(ctx, 7, 8)
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 2 || candidates[0].Issue != 55 || candidates[1].Issue != 9 {
		t.Fatalf("candidates: %+v", candidates)
	}
	for _, candidate := range candidates {
		if candidate.Origin != factory.PostCandidateRecent {
			t.Fatalf("candidate origin: %+v", candidate)
		}
	}
}

func TestPostStatusShowsExhaustionWithoutDiscardingConfirmedIssue(t *testing.T) {
	p := factory.PostIntake{Judgment: factory.PostJudgmentApplied, Route: factory.PostRouteNewIssue, Draft: factory.PostDraftPending, Effect: factory.PostEffectPending, Attempts: factory.PostAttemptLimit}
	if got := PostStatus(&p); got != "draft:failed" {
		t.Fatalf("exhausted authoring shown as %s", got)
	}
	p.Draft = factory.PostDraftApplied
	if got := PostStatus(&p); got != "effect:unconfirmed" {
		t.Fatalf("unknown native outcome shown as %s", got)
	}
	p.Effect, p.Issue = factory.PostEffectVerified, 9
	if got := PostStatus(&p); got != "issue:9" {
		t.Fatalf("confirmed issue lost: %s", got)
	}
	p.Judgment = factory.PostJudgmentPending
	if got := PostStatus(&p); got != "judgment:failed" {
		t.Fatalf("exhausted judgment shown as %s", got)
	}
}

func TestRetryFeedPostPreservesPostAndReconcilesUncertainEffect(t *testing.T) {
	ctx := context.Background()
	s, _ := postgresFixture(t)
	id := saveTestPost(t, s, 7, "42", "Original request", "retry-1")
	if err := s.RecordPostIntake(ctx, id, 0); err != nil {
		t.Fatal(err)
	}
	for range factory.PostAttemptLimit {
		if err := s.NotePostAttempt(ctx, id); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.RetryFeedPost(ctx, id, "43", 0, "Other account", 1729000020); err == nil {
		t.Fatal("another author changed request")
	}
	if err := s.RetryFeedPost(ctx, id, "42", 0, "Clarified request", 1729000020); err != nil {
		t.Fatal(err)
	}
	detail, err := s.FeedPost(ctx, id)
	if err != nil || detail.Body != "Clarified request" || detail.Record.Revision != 1 {
		t.Fatalf("post lost: %+v %v", detail, err)
	}
	pipe, err := s.PostIntake(ctx, id)
	if err != nil || pipe.Attempts != 0 || pipe.Judgment != "pending" || pipe.Revision != 1 {
		t.Fatalf("retry state: %+v %v", pipe, err)
	}
	next := pipe
	next.Judgment = factory.PostJudgmentApplied
	next.Route = factory.PostRouteNewIssue
	next.Draft = factory.PostDraftApplied
	next.Effect = factory.PostEffectFailed
	if err := s.AdvancePostIntake(ctx, pipe, next); err != nil {
		t.Fatal(err)
	}
	if err := s.RetryFeedPost(ctx, id, "42", 1, "Clarified request", 1729000030); err != nil {
		t.Fatal(err)
	}
	resumed, _ := s.PostIntake(ctx, id)
	if resumed.Revision != 1 || resumed.Draft != factory.PostDraftApplied || resumed.Effect != factory.PostEffectFailed {
		t.Fatalf("uncertain native effect was replaced: %+v", resumed)
	}
	done := resumed
	done.Effect = factory.PostEffectApplied
	done.Issue = 5
	if err := s.AdvancePostIntake(ctx, resumed, done); err != nil {
		t.Fatal(err)
	}
	if err := s.RetryFeedPost(ctx, id, "42", 1, "Duplicate", 1729000040); err == nil {
		t.Fatal("confirmed issue was duplicated")
	}
}

func TestPendingPostRequestsRotatePastWaitingWork(t *testing.T) {
	ctx := context.Background()
	s, _ := postgresFixture(t)
	first := saveTestPost(t, s, 7, "42", "Waiting", "fair-1")
	second := saveTestPost(t, s, 7, "42", "Ready", "fair-2")
	for _, id := range []string{first, second} {
		if err := s.RecordPostIntake(ctx, id, 0); err != nil {
			t.Fatal(err)
		}
	}
	page, err := s.PendingPostIntakes(ctx, 1)
	if err != nil || len(page) != 1 || page[0].SourcePost != first {
		t.Fatalf("first: %+v %v", page, err)
	}
	if err := s.NotePostProgress(ctx, first, "Waiting for setup"); err != nil {
		t.Fatal(err)
	}
	page, err = s.PendingPostIntakes(ctx, 1)
	if err != nil || len(page) != 1 || page[0].SourcePost != second {
		t.Fatalf("waiting work starved next request: %+v %v", page, err)
	}
}

func TestPostProductReviewKeepsCanonicalScopeAndOwner(t *testing.T) {
	s, _ := postgresFixture(t)
	ctx := context.Background()
	idea := factory.ProductIdea{ID: "d888888888888888888888888", Description: "Track requests", Consequence: "Requests persist", Decision: "accept", Source: "requested", Provenance: "human", Actor: 42}
	accepted := factory.IntentRevision{ID: "d111111111111111111111111", Intent: "d000000000000000000000000", Owner: 42, Repository: 7, Objective: "Track equipment requests", State: "accepted", MaxIssues: 10, CreatedUnix: 1, Ideas: []factory.ProductIdea{idea}, Tasks: []factory.IntentTask{{Key: "d999999999999999999999999", Objective: "Record requests", Outcomes: []string{"Requests persist"}, IdeaIDs: []string{idea.ID}}}}
	if err := s.SaveIntentRevision(ctx, accepted); err != nil {
		t.Fatal(err)
	}
	var post string
	if err := s.db.QueryRowContext(ctx, `SELECT id FROM factory_feed_posts WHERE data->>'event_identity'=$1`, "intent-revision:"+accepted.ID).Scan(&post); err != nil {
		t.Fatal(err)
	}
	source, err := s.ResolvePostSource(ctx, 7, factory.PostSourcePost, post)
	if err != nil || source.Product == nil || source.Product.Intent != accepted.Intent || source.Product.AcceptedRevision != accepted.ID {
		t.Fatalf("canonical source missing: %+v %v", source, err)
	}
	if _, err := s.ResolvePostSource(ctx, 8, factory.PostSourcePost, post); err == nil {
		t.Fatal("foreign repository acquired product source")
	}
	save := func(key string) string {
		t.Helper()
		id, _, err := s.SaveFeedPostWithSource(ctx, factory.PostRequest{Repository: 7, Body: "Require Finance approval above EUR 1000; retain existing requests.", SourceKind: source.Kind, SourceID: source.ID, IdempotencyKey: key}, "43", "Robin", 2, source)
		if err != nil {
			t.Fatal(err)
		}
		if err = s.RecordPostIntake(ctx, id, 0); err != nil {
			t.Fatal(err)
		}
		p, err := s.PostIntake(ctx, id)
		if err != nil {
			t.Fatal(err)
		}
		next := p
		next.Judgment = factory.PostJudgmentApplied
		next.Route = factory.PostRouteNewIssue
		if err = s.AdvancePostIntake(ctx, p, next); err != nil {
			t.Fatal(err)
		}
		return id
	}
	id, stale := save("finance"), save("another-change")
	detail, err := s.FeedPost(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	pipe, err := s.PostIntake(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	view, err := s.PostProductChange(ctx, detail, pipe, 42)
	if err != nil || view == nil || view.State != "review_required" || !view.CanReview {
		t.Fatalf("review not available: %+v %v", view, err)
	}
	if other, err := s.PostProductChange(ctx, detail, pipe, 43); err != nil || other.CanReview {
		t.Fatalf("nonowner review authority: %+v %v", other, err)
	}
	if _, err = s.ReviewPostProductChange(ctx, id, 0, 43, 3); err == nil {
		t.Fatal("comment author became product owner")
	}
	if _, err = s.ReviewPostProductChange(ctx, id, 1, 42, 3); err == nil {
		t.Fatal("stale comment reviewed")
	}
	draft, err := s.ReviewPostProductChange(ctx, id, 0, 42, 3)
	if err != nil {
		t.Fatal(err)
	}
	if draft.Intent != accepted.Intent || draft.Predecessor != accepted.ID || draft.State != "draft" || len(draft.Ideas) != 2 || len(draft.EffectiveOutcomes) != 2 || draft.EffectiveOutcomes[0] != idea.ID || draft.Tasks[0].Key != accepted.Tasks[0].Key {
		t.Fatalf("successor changed baseline: %+v", draft)
	}
	if !strings.Contains(draft.Ideas[1].Evidence, "43") || draft.Ideas[1].Description != detail.Body {
		t.Fatal("original request or attribution lost")
	}
	view, err = s.PostProductChange(ctx, detail, pipe, 42)
	if err != nil || view.State != "draft" || !view.CanReview || view.DraftRevision != draft.ID {
		t.Fatalf("owner cannot continue: %+v %v", view, err)
	}
	if other, err := s.PostProductChange(ctx, detail, pipe, 43); err != nil || other.CanReview {
		t.Fatalf("nonowner continuation authority: %+v %v", other, err)
	}
	replay, err := s.ReviewPostProductChange(ctx, id, 0, 42, 4)
	if err != nil || replay.ID != draft.ID || replay.CreatedUnix != draft.CreatedUnix {
		t.Fatalf("handoff duplicated: %+v %v", replay, err)
	}
	if _, err = s.ReviewPostProductChange(ctx, stale, 0, 42, 4); err == nil {
		t.Fatal("another comment replaced current review")
	}
	if _, err = s.EditFeedPost(ctx, id, "43", "Different request", 0, 5); err == nil {
		t.Fatal("admitted source rewritten")
	}
	if err = s.RetryFeedPost(ctx, id, "43", 0, "Different request", 5); err == nil {
		t.Fatal("admitted source retried")
	}
	queued, err := s.PendingPostIntakes(ctx, 20)
	if err != nil || len(queued) != 0 {
		t.Fatalf("review waits in automatic issue queue: %+v %v", queued, err)
	}
	inherited, err := s.ResolvePostSource(ctx, 7, factory.PostSourcePost, id)
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
	view, err = s.PostProductChange(ctx, detail, pipe, 42)
	if err != nil || view.State != "accepted" || !view.CanReview {
		t.Fatalf("owner cannot open accepted change: %+v %v", view, err)
	}
	if other, err := s.PostProductChange(ctx, detail, pipe, 43); err != nil || other.State != "accepted" || other.CanReview {
		t.Fatalf("collaborator gained accepted owner control: %+v %v", other, err)
	}
	replay, err = s.ReviewPostProductChange(ctx, id, 0, 42, 6)
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
	view, err = s.PostProductChange(ctx, detail, pipe, 42)
	if err != nil || view.State != "superseded" {
		t.Fatalf("declined Finance proposal still presented as accepted: %+v %v", view, err)
	}
	// A corrupted same-product binding is a projection error, never a
	// silent absence: the store surfaces the error so the API projects the
	// lane explicitly instead of falling back to detached generic progress.
	if _, err = s.db.ExecContext(ctx, `UPDATE factory_feed_posts SET data = data || '{"product_change": "bogus"}' WHERE id=$1`, id); err != nil {
		t.Fatal(err)
	}
	if _, err = s.PostProductChange(ctx, detail, pipe, 42); err == nil {
		t.Fatal("corrupted product binding reads as a successful projection")
	}
}
