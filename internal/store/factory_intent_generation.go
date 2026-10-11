package store

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
)

func (s *Store) IntentGeneration(ctx context.Context, revision string) (factory.IntentGeneration, error) {
	var g factory.IntentGeneration
	var raw []byte
	err := s.db.QueryRowContext(ctx, `SELECT data FROM factory_intent_generations WHERE revision=$1`, revision).Scan(&raw)
	if err == nil {
		err = json.Unmarshal(raw, &g)
	}
	return g, err
}
func (s *Store) BeginIntentGeneration(ctx context.Context, g factory.IntentGeneration) error {
	if !project.ValidDecisionID(g.RevisionID) || !project.ValidDecisionID(g.IntentID) || g.OperationID == "" || g.ActorID <= 0 || g.NativeRev < 1 || g.NotAfter <= 0 || !factory.ValidDigest(g.IntentDigest) || !json.Valid(g.Payload) || g.State != "pending" || len(g.Receipt) > 0 {
		return errors.New("invalid intent generation")
	}
	raw, err := json.Marshal(g)
	if err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	var head string
	var intentRaw []byte
	if err = tx.QueryRowContext(ctx, `SELECT revision FROM factory_intent_heads WHERE id=$1 FOR UPDATE`, g.IntentID).Scan(&head); err != nil {
		return err
	}
	if head != g.RevisionID {
		return ErrStaleRevision
	}
	if err = tx.QueryRowContext(ctx, `SELECT data FROM factory_intent_revisions WHERE id=$1`, head).Scan(&intentRaw); err != nil {
		return err
	}
	var r factory.IntentRevision
	if err = json.Unmarshal(intentRaw, &r); err != nil {
		return err
	}
	if r.State != "accepted" || r.Digest() != g.IntentDigest {
		return ErrStaleRevision
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO factory_intent_generations(revision,data) VALUES($1,$2) ON CONFLICT(revision) DO NOTHING`, g.RevisionID, string(raw))
	if err != nil {
		return err
	}
	var old []byte
	if err = tx.QueryRowContext(ctx, `SELECT data FROM factory_intent_generations WHERE revision=$1`, g.RevisionID).Scan(&old); err != nil {
		return err
	}
	same, err := sameJSONDocument(ctx, tx, old, raw)
	if err != nil {
		return err
	}
	if !same {
		return ErrCommandConflict
	}
	return tx.Commit()
}
func (s *Store) RecordIntentGeneration(ctx context.Context, g factory.IntentGeneration) error {
	old, err := s.IntentGeneration(ctx, g.RevisionID)
	if err != nil {
		return err
	}
	compare := g
	compare.State = old.State
	compare.Receipt = old.Receipt
	a, _ := json.Marshal(compare)
	b, _ := json.Marshal(old)
	if string(a) != string(b) {
		return ErrCommandConflict
	}
	if g.State != "pending" && g.State != "committed" && g.State != "refused" && g.State != "completed" {
		return errors.New("invalid generation outcome")
	}
	if old.State == "completed" && g.State != "completed" {
		return ErrCommandConflict
	}
	raw, err := json.Marshal(g)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `UPDATE factory_intent_generations SET data=$2 WHERE revision=$1`, g.RevisionID, string(raw))
	return err
}
func (s *Store) BindIntentTaskAcceptance(ctx context.Context, b factory.IntentTaskBinding) error {
	if !project.ValidDecisionID(b.RevisionID) || !project.ValidDecisionID(b.TaskKey) || !project.ValidDecisionID(b.AcceptanceID) || b.IssueID <= 0 || b.IssueIndex <= 0 || b.ContentVersion < 0 {
		return errors.New("invalid generated task binding")
	}
	raw, err := json.Marshal(b)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO factory_intent_tasks(revision,task,data) VALUES($1,$2,$3) ON CONFLICT(revision,task) DO NOTHING`, b.RevisionID, b.TaskKey, string(raw))
	return err
}
func (s *Store) IntentTaskBindings(ctx context.Context, revision string) ([]factory.IntentTaskBinding, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT data FROM factory_intent_tasks WHERE revision=$1 ORDER BY task`, revision)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []factory.IntentTaskBinding{}
	for rows.Next() {
		var raw []byte
		var b factory.IntentTaskBinding
		if err = rows.Scan(&raw); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(raw, &b); err != nil {
			return nil, err
		}
		result = append(result, b)
	}
	return result, rows.Err()
}

// GeneratedIntentCurrent revalidates the accepted product revision behind a
// derived acceptance. Ordinary human issue acceptances have no generated link.
func (s *Store) GeneratedIntentCurrent(ctx context.Context, acceptance string) (bool, bool, error) {
	var current bool
	err := s.db.QueryRowContext(ctx, `SELECT h.revision=t.revision AND r.data->>'state'='accepted' FROM factory_intent_tasks t JOIN factory_intent_revisions r ON r.id=t.revision JOIN factory_intent_heads h ON h.id=r.intent WHERE t.data->>'acceptance_id'=$1 LIMIT 1`, acceptance).Scan(&current)
	if errors.Is(err, ErrNotFound) {
		return true, false, nil
	}
	return current, true, err
}
