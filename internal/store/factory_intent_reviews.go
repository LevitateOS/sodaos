package store

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/levitateos/sodaos/internal/factory"
)

// SaveIntentRevisionWithReview saves a settled proposal revision and its
// exact review presentation atomically under the existing head-serialized
// CAS rules. Identity replay preserves immutable data; conflicting, stale
// or interrupted saves refuse without partial success, so a result revision
// never exists with a lost or substituted presentation.
func (s *Store) SaveIntentRevisionWithReview(ctx context.Context, r factory.IntentRevision, review factory.SavedReview) error {
	if err := r.Validate(); err != nil {
		return err
	}
	if err := review.Validate(); err != nil {
		return err
	}
	if review.Intent != r.Intent || review.BasisRevision != r.Predecessor || review.ResultRevision != r.ID {
		return errors.New("review does not bind this revision transition")
	}
	raw, err := json.Marshal(r)
	if err != nil {
		return err
	}
	reviewRaw, err := json.Marshal(review)
	if err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if err := saveIntentRevisionTx(ctx, tx, r, raw); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO factory_intent_reviews(result_revision,data) VALUES($1,$2) ON CONFLICT(result_revision) DO NOTHING`, r.ID, string(reviewRaw)); err != nil {
		return err
	}
	var old []byte
	if err := tx.QueryRowContext(ctx, `SELECT data FROM factory_intent_reviews WHERE result_revision=$1`, r.ID).Scan(&old); err != nil {
		return err
	}
	same, err := sameJSONDocument(ctx, tx, old, reviewRaw)
	if err != nil {
		return err
	}
	if !same {
		return ErrCommandConflict
	}
	return tx.Commit()
}

// IntentReview returns the exact stored presentation for one result
// revision. Another owner's reviews and missing presentations both report
// ErrNotFound, so callers cannot distinguish them.
func (s *Store) IntentReview(ctx context.Context, resultRevision string, owner int64) (factory.SavedReview, error) {
	var review factory.SavedReview
	var raw []byte
	err := s.db.QueryRowContext(ctx, `SELECT v.data FROM factory_intent_reviews v JOIN factory_intent_revisions r ON r.id=v.result_revision JOIN factory_intent_heads h ON h.id=r.intent WHERE v.result_revision=$1 AND h.owner_id=$2`, resultRevision, owner).Scan(&raw)
	if err == nil {
		err = json.Unmarshal(raw, &review)
	}
	return review, err
}
