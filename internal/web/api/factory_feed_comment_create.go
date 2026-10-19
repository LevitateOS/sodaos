package api

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	sdk "forgejo.org/extension-sdk"
	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/factory/control"
	"github.com/levitateos/sodaos/internal/store"
)

// Comment native-issue binding (Lane A execution join for R32.14
// step 4): validate the recorded draft and current authority, then
// create through R13's native issue/graph operation with a
// deterministic comment-bound operation identity, so a lost effect
// reply reconciles through lookup instead of duplicating.
//
// Authority follows the existing intent-owner bar: the human comment
// author must hold native code-write at an idle revision
// (CheckIntentOwner); otherwise creation refuses and the pipeline
// waits honestly. Human comment submission authorizes ordinary issue
// creation without another per-comment approval, but it grants no
// implementation or deployment authority. AuthRevision binds the
// comment intake (comment ID plus source revision); the host does not
// interpret it.
//
// Product Lead LLM authoring (step 3) is bound separately in
// factory_feed_comment_draft.go through the existing
// intent-generation funding transport; this file owns only the
// native effect behind an already-recorded draft. Nothing here
// authors or fabricates a draft.

// CommentIssueStore resolves the comment author behind a draft for
// the native owner gate. *store.Store implements it; tests stub it.
type CommentIssueStore interface {
	FeedComment(ctx context.Context, id string) (store.FeedCommentDetail, error)
}

// GraphCommentIssueCreator creates native issues from recorded
// comment drafts through the coordinator's graph executor and looks
// up the issue already created from a comment.
type GraphCommentIssueCreator struct {
	Store CommentIssueStore
	Graph control.IssueGraphExecutor
	Actor int64
}

var _ CommentIssueCreator = (*GraphCommentIssueCreator)(nil)
var _ CommentIssueStore = (*store.Store)(nil)

func (c *GraphCommentIssueCreator) DeliverCommentFeedback(ctx context.Context, comment store.FeedCommentDetail, issue int64) error {
	if c == nil || c.Graph == nil || c.Actor <= 0 {
		return errors.New("Issue discussion delivery is unavailable. Retry after Factory is connected.")
	}
	writer, ok := c.Graph.(interface {
		DeliverIssueFeedback(context.Context, int64, int64, int64, string, string) error
	})
	if !ok {
		return errors.New("Issue discussion delivery is unavailable.")
	}
	author, err := strconv.ParseInt(comment.Record.Author, 10, 64)
	if err != nil || author <= 0 {
		return errors.New("Feedback author is unavailable.")
	}
	if _, err = c.Graph.CheckIntentOwner(ctx, comment.Record.Repository, author, c.Actor); err != nil {
		return errors.New("Feedback needs current repository write access. Restore access before retrying.")
	}
	marker := "<!-- soda-feed:" + comment.Record.ID + ":r" + strconv.FormatInt(comment.Record.Revision, 10) + " -->"
	body := "Feedback from @" + comment.AuthorName + " in the project feed:\n\n" + comment.Body
	return writer.DeliverIssueFeedback(ctx, comment.Record.Repository, issue, c.Actor, marker, body)
}

// commentIssueOperationID binds one native operation to one comment:
// repeated intake for the same source resolves the existing effect.
func commentIssueOperationID(comment string) string {
	return "commentissue-" + comment
}

// commentIssueAuthRevision binds the authorizing intake: the comment
// ID plus the exact source revision the draft was authored against.
func commentIssueAuthRevision(comment string, revision int64) string {
	return "comment/" + comment + "/r" + strconv.FormatInt(revision, 10)
}

// commentIssueBody renders the native issue body: the recorded draft
// plus a bounded Soda footer preserving the source-comment link and
// the work metadata native creation validates.
func commentIssueBody(draft factory.IssueDraft) string {
	var footer strings.Builder
	footer.WriteString("\n\n---\nSoda-Comment: " + draft.Comment + " r" + strconv.FormatInt(draft.Revision, 10) + "\n")
	footer.WriteString("Objective: " + draft.Objective + "\n")
	footer.WriteString("Acceptance: " + draft.Acceptance + "\n")
	footer.WriteString("Work-Metadata: " + draft.Metadata + "\n")
	body := draft.Body + footer.String()
	if len(body) > 16384+4096 {
		body = body[:16384+4096]
	}
	return body
}

// commentIssueGeneration builds the exact durable graph operation
// for one validated draft: a single keyed create carrying the
// comment-bound identity, the service actor, the idle native
// revision behind the owner gate and a five-minute horizon.
func commentIssueGeneration(draft factory.IssueDraft, actor int64, nativeRev int64, now time.Time) (factory.IntentGeneration, error) {
	payload, err := json.Marshal(sdk.IssueGraphPayload{
		Creates: []sdk.IssueGraphCreate{{
			Key:   draft.Comment,
			Title: draft.Title,
			Body:  commentIssueBody(draft),
		}},
	})
	if err != nil {
		return factory.IntentGeneration{}, err
	}
	return factory.IntentGeneration{
		OperationID:  commentIssueOperationID(draft.Comment),
		ActorID:      actor,
		NativeRev:    nativeRev,
		AuthRevision: commentIssueAuthRevision(draft.Comment, draft.Revision),
		NotAfter:     now.Add(5 * time.Minute).Unix(),
		Payload:      payload,
	}, nil
}

// commentIssueAuthor resolves the human comment author behind a
// draft: the owner the native gate checks. Nothing substitutes for
// the recorded author.
func (c *GraphCommentIssueCreator) commentIssueAuthor(ctx context.Context, draft factory.IssueDraft) (int64, error) {
	if c == nil || c.Store == nil {
		return 0, errors.New("comment issue store is unavailable")
	}
	detail, err := c.Store.FeedComment(ctx, draft.Comment)
	if err != nil {
		return 0, err
	}
	author, err := strconv.ParseInt(detail.Record.Author, 10, 64)
	if err != nil || author <= 0 {
		return 0, errors.New("comment author is not a native owner")
	}
	return author, nil
}

// commentReceiptIndex extracts the native issue index for one
// comment key from a graph receipt. Entries without a committed
// index report not-found, never zero as success.
func commentReceiptIndex(record sdk.OperationRecord, key string) (int64, error) {
	if len(record.Receipt) == 0 {
		return 0, nil
	}
	var receipt sdk.IssueGraphReceipt
	if err := json.Unmarshal(record.Receipt, &receipt); err != nil {
		return 0, errors.New("native graph receipt refused")
	}
	for _, entry := range receipt.Issues {
		if entry.Key == key && entry.IssueIndex > 0 {
			return entry.IssueIndex, nil
		}
	}
	return 0, nil
}

// CreateCommentIssue validates one recorded draft and its current
// authority, then creates the native issue through the graph
// operation. An uncertain transport outcome reconciles through
// LookupCommentIssue instead of a second identity.
func (c *GraphCommentIssueCreator) CreateCommentIssue(ctx context.Context, draft factory.IssueDraft, repository int64) (int64, error) {
	if err := draft.Validate(); err != nil {
		return 0, err
	}
	if repository <= 0 {
		return 0, errors.New("comment issue needs its repository")
	}
	if c == nil || c.Graph == nil || c.Actor <= 0 {
		return 0, errors.New("comment native creation is unavailable")
	}
	author, err := c.commentIssueAuthor(ctx, draft)
	if err != nil {
		return 0, err
	}
	revision, err := c.Graph.CheckIntentOwner(ctx, repository, author, c.Actor)
	if err != nil {
		return 0, err
	}
	generation, err := commentIssueGeneration(draft, c.Actor, revision, time.Now())
	if err != nil {
		return 0, err
	}
	record, err := c.Graph.SubmitGraph(ctx, generation, repository)
	if err != nil {
		return 0, err
	}
	index, err := commentReceiptIndex(record, draft.Comment)
	if err != nil {
		return 0, err
	}
	if index <= 0 {
		return 0, errors.New("native creation left no committed issue")
	}
	return index, nil
}

// LookupCommentIssue returns the native issue index already created
// from a comment, or zero when none is recorded. Lookup errors
// propagate so the driver falls through to the self-reconciling
// create path.
func (c *GraphCommentIssueCreator) LookupCommentIssue(ctx context.Context, repository int64, commentID string) (int64, error) {
	if repository <= 0 || commentID == "" {
		return 0, errors.New("comment lookup needs its repository and comment")
	}
	if c == nil || c.Graph == nil || c.Actor <= 0 {
		return 0, errors.New("comment native lookup is unavailable")
	}
	lookup, err := c.Graph.LookupGraph(ctx, factory.IntentGeneration{
		OperationID: commentIssueOperationID(commentID), ActorID: c.Actor,
	}, repository)
	if err != nil {
		return 0, err
	}
	if lookup.Record == nil {
		return 0, nil
	}
	return commentReceiptIndex(*lookup.Record, commentID)
}
