package store

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
)

// Post dispositions persist beside the immutable posts they resolve:
// one row per suggestion or answer, carrying its applied, refused or
// incorporated state with the actor and reason behind it. Reads
// attach the recorded disposition; the pending state is the absence
// of a row.
func initializePostResolutionSchema(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS factory_post_resolutions(post TEXT PRIMARY KEY,repository BIGINT NOT NULL,resolution TEXT NOT NULL,reason TEXT NOT NULL DEFAULT '',actor TEXT NOT NULL,assignment TEXT NOT NULL,submission TEXT NOT NULL,at BIGINT NOT NULL);`)
	return err
}

// recordMCPPostResolutionTx persists one resolve_post disposition.
// The target must be a live suggestion or answer in this repository
// and match its resolution; a second disposition for the same post
// refuses so recorded states survive exactly as written.
func recordMCPPostResolutionTx(ctx context.Context, tx *sql.Tx, cap factory.MCPCapability, submission factory.MCPSubmission, receipt *factory.MCPReceipt) error {
	if submission.Tool != factory.MCPToolResolvePost {
		return nil
	}
	if !factory.MCPToolAdmitted(cap.Profile, submission.Tool) {
		return errors.New("post resolution is not admitted for this worker")
	}
	resolution, err := factory.DecodePostResolution(submission.Tool, submission.Payload)
	if err != nil {
		return err
	}
	var kind string
	if err = tx.QueryRowContext(ctx, `SELECT COALESCE(data->'communication'->>'kind','') FROM factory_feed_comments WHERE id=$1 AND repository=$2 AND NOT COALESCE((data->>'deleted')::boolean,false) AND NOT COALESCE((data->>'revoked')::boolean,false)`, resolution.Post, cap.Repository).Scan(&kind); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("resolution target is not available in this project")
		}
		return err
	}
	if err = factory.ResolutionForPost(kind, resolution.Resolution); err != nil {
		return err
	}
	var exists bool
	if err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM factory_post_resolutions WHERE post=$1)`, resolution.Post).Scan(&exists); err != nil {
		return err
	}
	if exists {
		return errors.New("post resolution already recorded")
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO factory_post_resolutions(post,repository,resolution,reason,actor,assignment,submission,at) VALUES($1,$2,$3,$4,$5,$6,$7,$8)`,
		resolution.Post, cap.Repository, resolution.Resolution, resolution.Reason, string(cap.Profile), submission.AssignmentID, submission.SubmissionID, time.Now().Unix())
	return err
}

// feedPostResolutions attaches recorded dispositions without mutating
// the original posts or claiming an unrecorded state: resolved posts
// carry their state and reason, everything else stays pending.
func (s *Store) feedPostResolutions(ctx context.Context, details map[string]FeedCommentDetail) error {
	ids := make([]string, 0, len(details))
	for id := range details {
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return nil
	}
	// Post IDs are globally unique content digests and the details
	// only exist when their posts passed the repository-scoped read,
	// so matching IDs is exact without another repository filter.
	rows, err := s.db.QueryContext(ctx, `SELECT post,resolution,reason FROM factory_post_resolutions WHERE post=ANY($1)`, ids)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id, resolution, reason string
		if err := rows.Scan(&id, &resolution, &reason); err != nil {
			return err
		}
		if detail, ok := details[id]; ok && detail.Payload != nil && detail.Payload.Communication != nil {
			detail.Payload.Communication.Disposition = resolution
			detail.Payload.Communication.DispositionReason = reason
		}
	}
	return rows.Err()
}
