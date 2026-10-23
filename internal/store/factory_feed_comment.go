package store

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"unicode/utf8"

	"github.com/levitateos/sodaos/internal/factory"
)

// Durable feed communication records (R32.2/R32.14): comment identity,
// revision and scope plus the intake/judgment/draft/effect request row.
// Comment identity is server-generated; the caller's idempotency key
// replays the same submission to the same comment instead of
// duplicating it. Reads spend nothing: funding reserves through the
// existing priority judgment budget before a JEV call, and the native
// effect applies through the injected issue creator only.

// FeedCommentDetail carries one stored comment with its serving
// fields. Source is the persisted resolved snapshot of the feed
// entry the comment was submitted against, when it named one.
type FeedCommentDetail struct {
	Actor      string
	Assignment string
	Payload    *factory.FeedPayload
	Record     factory.CommentRecord
	Body       string
	AuthorName string
	At         int64
	Status     string
	Source     *factory.CommentSource
}

// FeedReplyIDs lists the actual conversation, including replies from other
// browsers. The caller authorizes the repository and source before this read.
func (s *Store) FeedReplyIDs(ctx context.Context, repository int64, kind, source string, before int64) ([]string, int64, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,seq FROM factory_feed_comments
		WHERE repository=$1 AND (($2='post' AND data->>'reply_to'=$3) OR (data->'source'->>'kind'=$2 AND data->'source'->>'id'=$3))
		AND NOT COALESCE((data->>'deleted')::boolean,false) AND NOT COALESCE((data->>'revoked')::boolean,false)
		AND ($4::bigint=0 OR seq<$4) ORDER BY seq DESC LIMIT 21`, repository, kind, source, before)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	ids := []string{}
	var next, last int64
	for rows.Next() {
		var id string
		var seq int64
		if err := rows.Scan(&id, &seq); err != nil {
			return nil, 0, err
		}
		if len(ids) == 20 {
			next = last
			break
		}
		ids = append(ids, id)
		last = seq
	}
	return ids, next, rows.Err()
}

// SaveFeedComment persists one comment submission and returns its
// durable identity. A repeated idempotency key from the same author and
// repository replays the original identity instead of inserting a
// duplicate; the body of the replayed call is ignored.
func (s *Store) SaveFeedComment(ctx context.Context, req factory.CommentRequest, author, authorName string, at int64) (string, int64, error) {
	return s.SaveFeedCommentWithSource(ctx, req, author, authorName, at, nil)
}

// SaveFeedCommentWithSource persists one comment submission with the
// server-resolved source snapshot its submission named (nil for a
// top-level comment). The snapshot is stored verbatim beside the
// comment so later judgment and authoring consume the authorized
// evidence as submitted; a replayed idempotency key recovers the
// original comment and its original snapshot.
func (s *Store) SaveFeedCommentWithSource(ctx context.Context, req factory.CommentRequest, author, authorName string, at int64, source *factory.CommentSource) (string, int64, error) {
	if err := req.Validate(); err != nil {
		return "", 0, err
	}
	if source != nil {
		if err := source.Validate(); err != nil {
			return "", 0, err
		}
		if req.SourceKind != source.Kind || req.SourceID != source.ID {
			return "", 0, errors.New("comment source snapshot does not match its submission")
		}
	}
	if author == "" || at <= 0 {
		return "", 0, errors.New("comment needs its author and time")
	}
	var id [16]byte
	if _, err := rand.Read(id[:]); err != nil {
		return "", 0, err
	}
	fields := map[string]any{
		"issue": req.Issue, "reply_to": req.ReplyTo, "edited": false,
		"deleted": false, "revoked": false, "at": at, "author_name": authorName,
	}
	if source != nil {
		fields["source"] = source
	}
	data, err := json.Marshal(fields)
	if err != nil {
		return "", 0, err
	}
	commentID := hex.EncodeToString(id[:])
	result, err := s.db.ExecContext(ctx, `INSERT INTO factory_feed_comments(id,repository,revision,author,body,idempotency,data)
		VALUES($1,$2,0,$3,$4,$5,$6) ON CONFLICT(repository,author,idempotency) DO NOTHING`,
		commentID, req.Repository, author, req.Body, req.IdempotencyKey, string(data))
	if err != nil {
		return "", 0, err
	}
	if n, err := result.RowsAffected(); err != nil {
		return "", 0, err
	} else if n == 1 {
		return commentID, 0, nil
	}
	var existing string
	if err := s.db.QueryRowContext(ctx, `SELECT id FROM factory_feed_comments WHERE repository=$1 AND author=$2 AND idempotency=$3`,
		req.Repository, author, req.IdempotencyKey).Scan(&existing); err != nil {
		return "", 0, err
	}
	var revision int64
	if err := s.db.QueryRowContext(ctx, `SELECT revision FROM factory_feed_comments WHERE id=$1`, existing).Scan(&revision); err != nil {
		return "", 0, err
	}
	return existing, revision, nil
}

// EditFeedComment records one new comment revision under CAS: the
// expected revision must match and the author must own the comment.
// Deleted or revoked sources refuse the edit.
func (s *Store) EditFeedComment(ctx context.Context, id, author, body string, expected int64, at int64) (int64, error) {
	if len(body) == 0 || utf8.RuneCountInString(body) > factory.CommentBodyLimit || at <= 0 {
		return 0, errors.New("comment edit needs its bounded body and time")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()
	var savedRevision int64
	var savedAuthor string
	var raw []byte
	if err := tx.QueryRowContext(ctx, `SELECT revision,author,data FROM factory_feed_comments WHERE id=$1 FOR UPDATE`, id).Scan(&savedRevision, &savedAuthor, &raw); err != nil {
		return 0, err
	}
	if savedAuthor != author || savedRevision != expected {
		return 0, ErrStaleRevision
	}
	var data map[string]any
	if err := json.Unmarshal(raw, &data); err != nil {
		return 0, err
	}
	if data["deleted"] == true || data["revoked"] == true {
		return 0, errors.New("comment source is closed to edits")
	}
	data["edited"] = true
	data["at"] = at
	encoded, err := json.Marshal(data)
	if err != nil {
		return 0, err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE factory_feed_comments SET revision=revision+1,body=$2,data=$3 WHERE id=$1`,
		id, body, string(encoded)); err != nil {
		return 0, err
	}
	// A new source revision reopens an unsettled draft and effect
	// against the edited text; the recorded judgment and route survive.
	// A settled native outcome (applied or verified) keeps its draft,
	// effect and retained bytes as audit: the linked issue already
	// exists and the driver reconciles rather than duplicating it.
	if _, err := tx.ExecContext(ctx, `UPDATE factory_feed_comment_requests
		SET revision=$2,
		draft=CASE WHEN effect IN ('applied','verified') THEN draft ELSE 'pending' END,
		effect=CASE WHEN effect IN ('applied','verified') THEN effect ELSE 'pending' END,
		data=CASE WHEN effect IN ('applied','verified') THEN data ELSE '{}' END
		WHERE comment=$1`, id, expected+1); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return expected + 1, nil
}

// DeleteFeedComment tombstones one comment: the row and its bound
// pipeline still resolve, but the feed projection hides the body.
func (s *Store) DeleteFeedComment(ctx context.Context, id, author string, expected int64) (int64, error) {
	return s.markFeedComment(ctx, id, author, expected, "deleted")
}

// MarkFeedCommentRevoked records access loss on one comment: bound
// effects block instead of applying against a source the caller can no
// longer see.
func (s *Store) MarkFeedCommentRevoked(ctx context.Context, id string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	var raw []byte
	if err := tx.QueryRowContext(ctx, `SELECT data FROM factory_feed_comments WHERE id=$1 FOR UPDATE`, id).Scan(&raw); err != nil {
		return err
	}
	var data map[string]any
	if err := json.Unmarshal(raw, &data); err != nil {
		return err
	}
	data["revoked"] = true
	encoded, err := json.Marshal(data)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE factory_feed_comments SET revision=revision+1,data=$2 WHERE id=$1`, id, string(encoded)); err != nil {
		return err
	}
	if err := blockCommentEffectTx(ctx, tx, id); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) markFeedComment(ctx context.Context, id, author string, expected int64, flag string) (int64, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()
	var savedRevision int64
	var savedAuthor string
	var raw []byte
	if err := tx.QueryRowContext(ctx, `SELECT revision,author,data FROM factory_feed_comments WHERE id=$1 FOR UPDATE`, id).Scan(&savedRevision, &savedAuthor, &raw); err != nil {
		return 0, err
	}
	if savedAuthor != author || savedRevision != expected {
		return 0, ErrStaleRevision
	}
	var data map[string]any
	if err := json.Unmarshal(raw, &data); err != nil {
		return 0, err
	}
	data[flag] = true
	encoded, err := json.Marshal(data)
	if err != nil {
		return 0, err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE factory_feed_comments SET revision=revision+1,data=$2 WHERE id=$1`, id, string(encoded)); err != nil {
		return 0, err
	}
	if err := blockCommentEffectTx(ctx, tx, id); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return expected + 1, nil
}

// blockCommentEffectTx settles an unsettled native effect as blocked
// when its source closes: a deleted or revoked comment never applies a
// native issue against text its readers can no longer see.
func blockCommentEffectTx(ctx context.Context, tx *sql.Tx, id string) error {
	_, err := tx.ExecContext(ctx, `UPDATE factory_feed_comment_requests SET effect='blocked'
		WHERE comment=$1 AND effect IN ('pending','failed')`, id)
	return err
}

// FeedComment reads one durable comment with its serving fields.
func (s *Store) FeedComment(ctx context.Context, id string) (FeedCommentDetail, error) {
	var out FeedCommentDetail
	var repository, revision, issue int64
	var author, body, replyTo, authorName string
	var edited, deleted, revoked bool
	var at int64
	var sourceRaw, dataRaw []byte
	err := s.db.QueryRowContext(ctx, `SELECT id,repository,revision,author,body,
		COALESCE(data->>'reply_to',''),COALESCE((data->>'issue')::bigint,0),
		COALESCE((data->>'edited')::boolean,false),COALESCE((data->>'deleted')::boolean,false),
		COALESCE((data->>'revoked')::boolean,false),COALESCE(data->>'author_name',''),
		COALESCE((data->>'at')::bigint,0),COALESCE(data->'source','null'::jsonb),data FROM factory_feed_comments WHERE id=$1`, id).
		Scan(&out.Record.ID, &repository, &revision, &author, &body, &replyTo, &issue, &edited, &deleted, &revoked, &authorName, &at, &sourceRaw, &dataRaw)
	if err != nil {
		return FeedCommentDetail{}, err
	}
	out.Record = factory.CommentRecord{ID: out.Record.ID, Revision: revision, Repository: repository, Author: author, Issue: issue, ReplyTo: replyTo, Edited: edited, Deleted: deleted, Revoked: revoked}
	out.Body, out.AuthorName, out.At = body, authorName, at
	if err := applyFeedCommunicationDetail(&out, dataRaw); err != nil {
		return FeedCommentDetail{}, err
	}
	if len(sourceRaw) != 0 && string(sourceRaw) != "null" {
		var source factory.CommentSource
		if err := json.Unmarshal(sourceRaw, &source); err != nil {
			return FeedCommentDetail{}, err
		}
		if err := source.Validate(); err != nil {
			return FeedCommentDetail{}, err
		}
		out.Source = &source
	}
	if out.Payload != nil && out.Payload.Communication != nil && out.Payload.Communication.Kind == "ask" {
		details := map[string]FeedCommentDetail{out.Record.ID: out}
		if err := s.feedQuestionStates(ctx, details); err != nil {
			return FeedCommentDetail{}, err
		}
		out = details[out.Record.ID]
	}
	if out.Payload != nil && out.Payload.Communication != nil {
		details := map[string]FeedCommentDetail{out.Record.ID: out}
		if err := s.feedPostResolutions(ctx, details); err != nil {
			return FeedCommentDetail{}, err
		}
		out = details[out.Record.ID]
	}
	return out, nil
}

// RecordCommentIntake opens the durable request behind one comment
// revision: intake recorded, judgment/draft/effect pending. Replaying
// the same comment recovers the row instead of duplicating it.
func (s *Store) RecordCommentIntake(ctx context.Context, commentID string, revision int64) error {
	if _, err := s.FeedComment(ctx, commentID); err != nil {
		return err
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO factory_feed_comment_requests(comment,revision,intake,judgment,route,draft,effect,issue,attempts,data)
		VALUES($1,$2,'recorded','pending','', 'pending','pending',0,0,'{}') ON CONFLICT(comment) DO NOTHING`, commentID, revision)
	return err
}

// CommentPipeline reads the durable request behind one comment.
func (s *Store) CommentPipeline(ctx context.Context, commentID string) (factory.CommentPipeline, error) {
	var pipe factory.CommentPipeline
	err := s.db.QueryRowContext(ctx, `SELECT comment,revision,intake,judgment,route,draft,effect,issue,attempts
		FROM factory_feed_comment_requests WHERE comment=$1`, commentID).
		Scan(&pipe.Comment, &pipe.Revision, &pipe.Intake, &pipe.Judgment, &pipe.Route, &pipe.Draft, &pipe.Effect, &pipe.Issue, &pipe.Attempts)
	if err != nil {
		return factory.CommentPipeline{}, err
	}
	return pipe, pipe.Validate()
}

// AdvanceCommentPipeline moves one request row through CAS: every state
// column must still hold its expected value or the write refuses with
// ErrStaleRevision instead of dropping a concurrent advance.
func (s *Store) AdvanceCommentPipeline(ctx context.Context, expected, next factory.CommentPipeline) error {
	if err := next.Validate(); err != nil {
		return err
	}
	if expected.Comment == "" || expected.Comment != next.Comment {
		return errors.New("comment pipeline advance needs its comment identity")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `UPDATE factory_feed_comment_requests
		SET revision=$2,intake=$3,judgment=$4,route=$5,draft=$6,effect=$7,issue=$8,attempts=$9,
		data=CASE WHEN $12='pending' AND $4='applied' AND $5='existing_work' THEN data || jsonb_build_object('feedback_revision',$2::bigint) ELSE data END
		WHERE comment=$1 AND revision=$10 AND intake=$11 AND judgment=$12 AND route=$13 AND draft=$14 AND effect=$15 AND issue=$16 AND attempts=$17`,
		next.Comment, next.Revision, next.Intake, next.Judgment, next.Route, next.Draft, next.Effect, next.Issue, next.Attempts,
		expected.Revision, expected.Intake, expected.Judgment, expected.Route, expected.Draft, expected.Effect, expected.Issue, expected.Attempts)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return ErrStaleRevision
	}
	if next.Issue > 0 && next.Issue != expected.Issue && (next.Effect == factory.CommentEffectApplied || next.Effect == factory.CommentEffectVerified) {
		var repository int64
		if err = tx.QueryRowContext(ctx, `SELECT repository FROM factory_feed_comments WHERE id=$1`, next.Comment).Scan(&repository); err != nil {
			return err
		}
		if err = recordFactoryFeedEventTx(ctx, tx, repository, next.Issue, "", "issue:"+strconv.FormatInt(next.Issue, 10), "native_issue:created", factory.FeedCommunication{
			ComponentID: "F05", Title: fmt.Sprintf("Issue #%d created from a comment", next.Issue), Summary: "The native repository confirmed the new issue.",
			ReplyTo: next.Comment, Items: []factory.FeedCommunicationItem{{Label: fmt.Sprintf("Issue #%d", next.Issue), State: "created"}},
		}, 0); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// NoteCommentAttempt records one funded judgment, draft or native
// retry against the bounded attempt count. Exhaustion refuses instead
// of spending blindly.
func (s *Store) NoteCommentAttempt(ctx context.Context, commentID string) error {
	result, err := s.db.ExecContext(ctx, `UPDATE factory_feed_comment_requests SET attempts=attempts+1
		WHERE comment=$1 AND attempts<$2`, commentID, factory.CommentAttemptLimit)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return errors.New("comment attempts are exhausted")
	}
	return nil
}

// CommentJudgmentSnapshot binds one comment source revision to its
// funded JEV Choice reservation: the same revision replays the same
// snapshot, so a retried submission recovers its judgment funding
// instead of reserving twice.
func CommentJudgmentSnapshot(commentID string, revision int64) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("feed-comment/v1:%s:%d", commentID, revision)))
	return hex.EncodeToString(sum[:])
}

// ReserveCommentJudgment funds at most one JEV Choice call behind one
// comment source revision through the existing priority judgment
// budget. A recovered reservation reports false with no new spend.
func (s *Store) ReserveCommentJudgment(ctx context.Context, commentID string, revision int64) (bool, error) {
	return s.ReservePriorityJudgment(ctx, CommentJudgmentSnapshot(commentID, revision), 1)
}

// CommentJudgmentReservationOpen reports whether one comment revision
// holds an unfulfilled JEV judgment call: reserved but never answered,
// so a bounded retry may proceed under the existing reservation. JEV
// calls are free; the reservation is duplicate-effect protection and a
// provider-limit bound, not a monetary gate. A recorded answer or no
// reservation at all leaves no retry permission: the former already
// settled, the latter waits on the provider limit.
func (s *Store) CommentJudgmentReservationOpen(ctx context.Context, commentID string, revision int64) (bool, error) {
	var open int
	err := s.db.QueryRowContext(ctx, `SELECT count(*) FROM factory_priority_judgments WHERE snapshot=$1 AND data IS NULL`,
		CommentJudgmentSnapshot(commentID, revision)).Scan(&open)
	if err != nil {
		return false, err
	}
	return open > 0, nil
}

// RecordCommentJudgmentReceipt retains the funded JEV answer behind one
// comment source revision for audit and replay.
func (s *Store) RecordCommentJudgmentReceipt(ctx context.Context, commentID string, revision int64, receipt factory.PriorityReceipt) error {
	want := CommentJudgmentSnapshot(commentID, revision)
	if receipt.Snapshot != want {
		return errors.New("comment judgment receipt needs its bound snapshot")
	}
	return s.RecordPriorityJudgment(ctx, receipt)
}

// RecordCommentDraft retains one LLM-authored issue draft bound to its
// exact comment source revision. A malformed draft refuses before any
// native effect.
func (s *Store) RecordCommentDraft(ctx context.Context, draft factory.IssueDraft) error {
	if err := draft.Validate(); err != nil {
		return err
	}
	raw, err := json.Marshal(draft)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `UPDATE factory_feed_comment_requests SET data=$2
		WHERE comment=$1 AND revision=$3`, draft.Comment, string(raw), draft.Revision)
	return err
}

// DiscardCommentDraftData wipes retained draft bytes when the draft
// is refused as malformed: a refused draft must never stay servable
// behind its pipeline row.
func (s *Store) DiscardCommentDraftData(ctx context.Context, commentID string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE factory_feed_comment_requests SET data='{}' WHERE comment=$1`, commentID)
	return err
}

// CommentDraft reads the retained draft behind one comment, refusing
// when the bound revision moved on (edited source) or no draft exists.
func (s *Store) CommentDraft(ctx context.Context, commentID string, revision int64) (factory.IssueDraft, error) {
	var raw []byte
	var saved int64
	if err := s.db.QueryRowContext(ctx, `SELECT data,revision FROM factory_feed_comment_requests WHERE comment=$1`, commentID).Scan(&raw, &saved); err != nil {
		return factory.IssueDraft{}, err
	}
	if saved != revision || len(raw) == 0 || string(raw) == "{}" {
		return factory.IssueDraft{}, sql.ErrNoRows
	}
	var draft factory.IssueDraft
	if err := json.Unmarshal(raw, &draft); err != nil {
		return factory.IssueDraft{}, err
	}
	if draft.Comment != commentID || draft.Revision != revision {
		return factory.IssueDraft{}, sql.ErrNoRows
	}
	return draft, draft.Validate()
}

// FeedPostDetails reads the serving fields behind a set of post
// entries: author, time and the pipeline-derived status. Comments
// without a pipeline row report status open; deleted or revoked
// sources never reach the feed.
func (s *Store) FeedPostDetails(ctx context.Context, ids []string) (map[string]FeedCommentDetail, error) {
	out := make(map[string]FeedCommentDetail, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := s.db.QueryContext(ctx, `SELECT c.id,c.repository,c.revision,c.author,c.body,
		COALESCE(c.data->>'reply_to',''),COALESCE((c.data->>'issue')::bigint,0),
		COALESCE((c.data->>'edited')::boolean,false),COALESCE((c.data->>'deleted')::boolean,false),
		COALESCE((c.data->>'revoked')::boolean,false),COALESCE(c.data->>'author_name',''),
		COALESCE((c.data->>'at')::bigint,0),
		COALESCE(r.judgment,''),COALESCE(r.route,''),COALESCE(r.draft,''),COALESCE(r.effect,''),COALESCE(r.issue,0),COALESCE(r.attempts,0),c.data
		FROM factory_feed_comments c LEFT JOIN factory_feed_comment_requests r ON r.comment=c.id
		WHERE c.id = ANY($1)`, ids)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var detail FeedCommentDetail
		var judgment, route, draft, effect string
		var issue int64
		var attempts int
		var dataRaw []byte
		if err := rows.Scan(&detail.Record.ID, &detail.Record.Repository, &detail.Record.Revision,
			&detail.Record.Author, &detail.Body, &detail.Record.ReplyTo, &detail.Record.Issue,
			&detail.Record.Edited, &detail.Record.Deleted, &detail.Record.Revoked,
			&detail.AuthorName, &detail.At, &judgment, &route, &draft, &effect, &issue, &attempts, &dataRaw); err != nil {
			return nil, err
		}
		if detail.Record.Deleted || detail.Record.Revoked {
			continue
		}
		if judgment == "" {
			detail.Status = PostStatus(nil)
		} else {
			detail.Status = PostStatus(&factory.CommentPipeline{
				Comment: detail.Record.ID, Intake: factory.CommentIntakeRecorded,
				Judgment: judgment, Route: route, Draft: draft, Effect: effect, Issue: issue, Attempts: attempts,
			})
		}
		if err := applyFeedCommunicationDetail(&detail, dataRaw); err != nil {
			return nil, err
		}
		out[detail.Record.ID] = detail
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := s.feedQuestionStates(ctx, out); err != nil {
		return nil, err
	}
	if err := s.feedPostResolutions(ctx, out); err != nil {
		return nil, err
	}
	return out, nil
}

// ResolveCommentSource resolves one caller-named feed entry inside
// the given repository into its persisted source snapshot. Unknown,
// closed or cross-repository entries report ErrNotFound without
// disclosing which: the caller already proved repository visibility,
// and the source must live inside that same repository. Evidence
// comes from the canonical owners only — the assignment row and the
// feed record read for records, the stored comment and its pipeline
// status for posts.
func (s *Store) ResolveCommentSource(ctx context.Context, repository int64, kind, id string) (*factory.CommentSource, error) {
	if repository <= 0 {
		return nil, errors.New("comment source needs its repository scope")
	}
	switch kind {
	case factory.CommentSourceRecord:
		assignment, err := s.Assignment(ctx, id)
		if err != nil || assignment.Repository != repository {
			if err != nil && !errors.Is(err, ErrNotFound) {
				return nil, err
			}
			return nil, ErrNotFound
		}
		source := &factory.CommentSource{Kind: kind, ID: id, Issue: assignment.Issue}
		details, err := s.FeedRecordDetails(ctx, []string{id})
		if err != nil {
			return nil, err
		}
		if detail, ok := details[id]; ok {
			source.PR, source.Ref, source.Status, source.Summary = detail.PR, detail.Ref, detail.Status, detail.Summary
			if source.Issue <= 0 {
				source.Issue = detail.Issue
			}
		}
		return source, source.Validate()
	case factory.CommentSourcePost:
		detail, err := s.FeedComment(ctx, id)
		if err != nil || detail.Record.Repository != repository || detail.Record.Deleted || detail.Record.Revoked {
			if err != nil && !errors.Is(err, ErrNotFound) {
				return nil, err
			}
			return nil, ErrNotFound
		}
		source := &factory.CommentSource{
			Kind: kind, ID: id, Issue: detail.Record.Issue,
			Summary: TruncateFeedSummary(detail.Body), Status: PostStatus(nil),
		}
		if detail.Payload != nil && detail.Payload.Communication != nil {
			source.Status = detail.Status
			return source, source.Validate()
		}
		if pipe, err := s.CommentPipeline(ctx, id); err == nil {
			source.Status = PostStatus(&pipe)
			if source.Issue <= 0 {
				source.Issue = pipe.Issue
			}
		} else if !errors.Is(err, ErrNotFound) {
			return nil, err
		}
		return source, source.Validate()
	default:
		return nil, errors.New("comment source needs its known kind")
	}
}

// PendingCommentPipelines lists durable comment requests the
// service-side driver may still advance: judgment pending, or a
// new_issue route whose draft or native effect is unsettled. Closed
// (deleted/revoked) sources never list. Settled routes
// (discussion, existing work, clarify), failed judgments, invalid
// drafts and settled effects are final and stay off the list, so a
// restart resumes each stored request instead of re-deriving it.
func (s *Store) PendingCommentPipelines(ctx context.Context, limit int) ([]factory.CommentPipeline, error) {
	if limit <= 0 || limit > 64 {
		return nil, errors.New("invalid pending comment listing limit")
	}
	rows, err := s.db.QueryContext(ctx, `SELECT r.comment,r.revision,r.intake,r.judgment,r.route,r.draft,r.effect,r.issue,r.attempts
		FROM factory_feed_comment_requests r
		JOIN factory_feed_comments c ON c.id=r.comment
		WHERE COALESCE((c.data->>'deleted')::boolean,false)=false
		AND COALESCE((c.data->>'revoked')::boolean,false)=false
		AND r.attempts<$2
		AND (r.judgment='pending'
			OR (r.judgment='applied' AND r.route='new_issue'
				AND (r.draft='pending' OR (r.draft='applied' AND r.effect IN ('pending','failed'))))
			OR (r.judgment='applied' AND r.route='existing_work' AND r.effect IN ('pending','failed') AND (r.data->>'feedback_revision')::bigint=r.revision))
		ORDER BY COALESCE(r.data->>'last_checked_at',''),c.seq LIMIT $1`, limit, factory.CommentAttemptLimit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []factory.CommentPipeline
	for rows.Next() {
		var pipe factory.CommentPipeline
		if err := rows.Scan(&pipe.Comment, &pipe.Revision, &pipe.Intake, &pipe.Judgment, &pipe.Route, &pipe.Draft, &pipe.Effect, &pipe.Issue, &pipe.Attempts); err != nil {
			return nil, err
		}
		if err := pipe.Validate(); err != nil {
			return nil, err
		}
		out = append(out, pipe)
	}
	return out, rows.Err()
}

// NoteCommentProgress retains a safe user-facing reason and rotates waiting
// requests through the continuation queue without consuming model attempts.
func (s *Store) NoteCommentProgress(ctx context.Context, id, detail string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE factory_feed_comment_requests
		SET data=data || jsonb_build_object('last_checked_at',clock_timestamp()::text,'detail',$2::text) WHERE comment=$1`, id, TruncateFeedSummary(detail))
	return err
}

func (s *Store) CommentProgress(ctx context.Context, id string) (string, error) {
	var detail string
	err := s.db.QueryRowContext(ctx, `SELECT COALESCE(data->>'detail','') FROM factory_feed_comment_requests WHERE comment=$1`, id).Scan(&detail)
	return detail, err
}

func (s *Store) CommentFeedbackRequested(ctx context.Context, id string, revision int64) (bool, error) {
	var pending bool
	err := s.db.QueryRowContext(ctx, `SELECT COALESCE((data->>'feedback_revision')::bigint=$2,false) FROM factory_feed_comment_requests WHERE comment=$1`, id, revision).Scan(&pending)
	return pending, err
}

// RetryFeedComment preserves the original post and prior processing history.
// Uncertain issue writes keep their draft/identity for reconciliation; a new
// judgment/planning attempt gets a new source revision instead of another post.
func (s *Store) RetryFeedComment(ctx context.Context, id, author string, expected int64, body string, at int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	var savedAuthor, oldBody string
	var revision int64
	var closed bool
	if err := tx.QueryRowContext(ctx, `SELECT author,body,revision,COALESCE((data->>'deleted')::boolean,false) OR COALESCE((data->>'revoked')::boolean,false)
		FROM factory_feed_comments WHERE id=$1 FOR UPDATE`, id).Scan(&savedAuthor, &oldBody, &revision, &closed); err != nil {
		return err
	}
	if savedAuthor != author || revision != expected || closed {
		return ErrStaleRevision
	}
	var draft, effect, route, judgment string
	var attempts int
	if err := tx.QueryRowContext(ctx, `SELECT draft,effect,route,judgment,attempts FROM factory_feed_comment_requests WHERE comment=$1 FOR UPDATE`, id).Scan(&draft, &effect, &route, &judgment, &attempts); err != nil {
		return err
	}
	if !(factory.CommentPipeline{Draft: draft, Effect: effect, Route: route, Judgment: judgment, Attempts: attempts}).RetryAllowed() {
		return errors.New("this request is still processing or has already completed")
	}
	if effect == factory.CommentEffectApplied || effect == factory.CommentEffectVerified {
		return errors.New("this request already created an issue")
	}
	if body == "" {
		body = oldBody
	}
	if err := (factory.CommentRequest{Repository: 1, Body: body, IdempotencyKey: "retry"}).Validate(); err != nil {
		return err
	}
	if draft == factory.CommentDraftApplied {
		if body != oldBody {
			return errors.New("check the previous issue creation before changing this request")
		}
		_, err = tx.ExecContext(ctx, `UPDATE factory_feed_comment_requests SET attempts=0,data=data || '{"detail":""}'::jsonb WHERE comment=$1`, id)
	} else {
		_, err = tx.ExecContext(ctx, `UPDATE factory_feed_comments SET revision=revision+1,body=$2,
		 data=data || jsonb_build_object('edited',true,'at',$3::bigint,'history',COALESCE(data->'history','[]'::jsonb) || jsonb_build_array(jsonb_build_object('revision',revision,'body',body))) WHERE id=$1`, id, body, at)
		if err == nil {
			_, err = tx.ExecContext(ctx, `UPDATE factory_feed_comment_requests SET revision=$2,judgment='pending',route='',draft='pending',effect='pending',issue=0,attempts=0,
		 data=jsonb_build_object('history',COALESCE(data->'history','[]'::jsonb) || jsonb_build_array(jsonb_build_object('revision',revision,'judgment',judgment,'route',route,'issue',issue))) WHERE comment=$1`, id, expected+1)
		}
	}
	if err != nil {
		return err
	}
	return tx.Commit()
}

// RecentCommentWorkCandidates lists the most recently worked native
// issues in one repository as existing-work candidates for comment
// routing: the latest assignment per issue supplies the recorded
// summary and status through the canonical feed record read. Issues
// with no recorded factory work never appear; nothing is inferred
// beyond the recorded rows.
func (s *Store) RecentCommentWorkCandidates(ctx context.Context, repository int64, limit int) ([]factory.CommentWorkCandidate, error) {
	if repository <= 0 || limit <= 0 || limit > 32 {
		return nil, errors.New("invalid comment work candidate listing")
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id,issue FROM (
			SELECT DISTINCT ON (issue) id,issue,seq FROM factory_assignments
			WHERE repository=$1 AND issue>0 ORDER BY issue,seq DESC
		) latest ORDER BY seq DESC LIMIT $2`, repository, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var ids []string
	issueByID := map[string]int64{}
	for rows.Next() {
		var id string
		var issue int64
		if err := rows.Scan(&id, &issue); err != nil {
			return nil, err
		}
		ids = append(ids, id)
		issueByID[id] = issue
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return nil, nil
	}
	details, err := s.FeedRecordDetails(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make([]factory.CommentWorkCandidate, 0, len(ids))
	for _, id := range ids {
		candidate := factory.CommentWorkCandidate{Issue: issueByID[id], Origin: factory.CommentCandidateRecent}
		if detail, ok := details[id]; ok {
			candidate.Summary, candidate.Status = detail.Summary, detail.Status
		}
		if err := candidate.Validate(); err != nil {
			return nil, err
		}
		out = append(out, candidate)
	}
	return out, nil
}

// PostStatus derives one post's feed status from its pipeline row: the
// separate states stay distinguishable instead of collapsing into one
// generic marker.
func PostStatus(pipe *factory.CommentPipeline) string {
	if pipe == nil {
		return "open"
	}
	if pipe.Judgment == factory.CommentJudgmentPending {
		if pipe.Attempts >= factory.CommentAttemptLimit {
			return "judgment:failed"
		}
		return "judgment:pending"
	}
	if pipe.Judgment == factory.CommentJudgmentFailed {
		return "judgment:failed"
	}
	switch pipe.Route {
	case factory.CommentRouteExistingWork:
		if pipe.Effect == factory.CommentEffectApplied || pipe.Effect == factory.CommentEffectVerified {
			return "feedback:delivered"
		}
		return "route:" + pipe.Route
	case factory.CommentRouteDiscussion, factory.CommentRouteClarify:
		return "route:" + pipe.Route
	}
	if pipe.Route != factory.CommentRouteNewIssue {
		return "open"
	}
	if pipe.Draft == factory.CommentDraftPending {
		if pipe.Attempts >= factory.CommentAttemptLimit {
			return "draft:failed"
		}
		return "draft:pending"
	}
	if pipe.Draft == factory.CommentDraftInvalid {
		return "draft:invalid"
	}
	switch pipe.Effect {
	case factory.CommentEffectVerified:
		return fmt.Sprintf("issue:%d", pipe.Issue)
	case factory.CommentEffectApplied:
		return "effect:applied"
	case factory.CommentEffectBlocked:
		return "effect:blocked"
	case factory.CommentEffectFailed:
		return "effect:failed"
	default:
		if pipe.Attempts >= factory.CommentAttemptLimit {
			return "effect:unconfirmed"
		}
		return "effect:pending"
	}
}
