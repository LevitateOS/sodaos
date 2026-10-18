package store

import (
	"context"
	"database/sql"
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

func checkSubmittedGenerationGraph(g factory.IntentGeneration) error {
	if err := checkGenerationIdentity(g); err != nil {
		return err
	}
	seen := map[string]bool{}
	for _, task := range g.SubmittedTasks {
		if !project.ValidDecisionID(task.Key) || seen[task.Key] {
			return errors.New("invalid submitted task graph")
		}
		seen[task.Key] = true
	}
	for key, ref := range g.Delivered {
		if !seen[key] || g.TaskIssueIDs[key] != 0 || ref.IssueID <= 0 || ref.IssueIndex <= 0 || ref.ContentVersion < 0 {
			return errors.New("invalid delivered reuse")
		}
	}
	if len(g.Delivered) > 0 {
		var payload struct {
			Edits []struct {
				IssueID int64 `json:"issue_id"`
			} `json:"edits"`
		}
		if err := json.Unmarshal(g.Payload, &payload); err != nil {
			return errors.New("invalid intent generation")
		}
		for _, edit := range payload.Edits {
			for _, ref := range g.Delivered {
				if edit.IssueID == ref.IssueID {
					return errors.New("delivered history unexpectedly mutated")
				}
			}
		}
	}
	return nil
}

func checkGenerationIdentity(g factory.IntentGeneration) error {
	if !project.ValidDecisionID(g.RevisionID) || !project.ValidDecisionID(g.IntentID) || g.OperationID == "" || g.ActorID <= 0 || g.NativeRev < 1 || g.NotAfter <= 0 || !factory.ValidDigest(g.IntentDigest) || !json.Valid(g.Payload) || g.State != "pending" || len(g.Receipt) > 0 || len(g.SubmittedTasks) == 0 {
		return errors.New("invalid intent generation")
	}
	return nil
}

func checkGenerationHeadTx(ctx context.Context, tx *sql.Tx, g factory.IntentGeneration) error {
	var head string
	var intentRaw []byte
	if err := tx.QueryRowContext(ctx, `SELECT revision FROM factory_intent_heads WHERE id=$1 FOR UPDATE`, g.IntentID).Scan(&head); err != nil {
		return err
	}
	if head != g.RevisionID {
		return ErrStaleRevision
	}
	if err := tx.QueryRowContext(ctx, `SELECT data FROM factory_intent_revisions WHERE id=$1`, head).Scan(&intentRaw); err != nil {
		return err
	}
	var r factory.IntentRevision
	if err := json.Unmarshal(intentRaw, &r); err != nil {
		return err
	}
	if r.State != "accepted" || r.Digest() != g.IntentDigest {
		return ErrStaleRevision
	}
	return nil
}

func checkDeliveredBindingTx(ctx context.Context, tx *sql.Tx, g factory.IntentGeneration, key string, ref factory.DeliveredIssue) error {
	rows, err := tx.QueryContext(ctx, `SELECT data FROM factory_intent_tasks WHERE task=$1 AND revision IN (SELECT id FROM factory_intent_revisions WHERE intent=$2 AND id<>$3)`, key, g.IntentID, g.RevisionID)
	if err != nil {
		return err
	}
	chained := false
	for rows.Next() {
		var rawBinding []byte
		var b factory.IntentTaskBinding
		if err = rows.Scan(&rawBinding); err != nil {
			rows.Close()
			return err
		}
		if err = json.Unmarshal(rawBinding, &b); err != nil {
			rows.Close()
			return err
		}
		if b.IssueID == ref.IssueID && b.IssueIndex == ref.IssueIndex && b.ContentVersion == ref.ContentVersion {
			chained = true
		}
	}
	rows.Close()
	if rows.Err() != nil {
		return rows.Err()
	}
	if !chained {
		return errors.New("invalid delivered reuse")
	}
	return nil
}

func insertGenerationRecordTx(ctx context.Context, tx *sql.Tx, g factory.IntentGeneration, raw []byte) error {
	if _, err := tx.ExecContext(ctx, `INSERT INTO factory_intent_generations(revision,data) VALUES($1,$2) ON CONFLICT(revision) DO NOTHING`, g.RevisionID, string(raw)); err != nil {
		return err
	}
	var old []byte
	if err := tx.QueryRowContext(ctx, `SELECT data FROM factory_intent_generations WHERE revision=$1`, g.RevisionID).Scan(&old); err != nil {
		return err
	}
	same, err := sameJSONDocument(ctx, tx, old, raw)
	if err != nil {
		return err
	}
	if !same {
		return ErrCommandConflict
	}
	return nil
}

func (s *Store) BeginIntentGeneration(ctx context.Context, g factory.IntentGeneration) error {
	if err := checkSubmittedGenerationGraph(g); err != nil {
		return err
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
	if err = checkGenerationHeadTx(ctx, tx, g); err != nil {
		return err
	}
	for key, ref := range g.Delivered {
		if err = checkDeliveredBindingTx(ctx, tx, g, key, ref); err != nil {
			return err
		}
	}
	if err = insertGenerationRecordTx(ctx, tx, g, raw); err != nil {
		return err
	}
	return tx.Commit()
}

func compareGenerationRecord(old, g factory.IntentGeneration) (a, b []byte, err error) {
	compare := g
	compare.State = old.State
	compare.Receipt = old.Receipt
	if a, err = json.Marshal(compare); err != nil {
		return nil, nil, err
	}
	if b, err = json.Marshal(old); err != nil {
		return nil, nil, err
	}
	return a, b, nil
}

func checkGenerationReceipt(g factory.IntentGeneration) error {
	if g.State != "pending" && g.State != "committed" && g.State != "refused" && g.State != "completed" {
		return errors.New("invalid generation outcome")
	}
	if len(g.Receipt) > 0 && len(g.Delivered) > 0 {
		var receipt struct {
			Issues []struct {
				IssueID int64 `json:"issue_id"`
			} `json:"issues"`
		}
		if err := json.Unmarshal(g.Receipt, &receipt); err != nil {
			return errors.New("invalid generation receipt")
		}
		for _, entry := range receipt.Issues {
			for _, ref := range g.Delivered {
				if entry.IssueID == ref.IssueID {
					return errors.New("delivered history unexpectedly mutated")
				}
			}
		}
	}
	return nil
}

func checkGenerationRollback(old, g factory.IntentGeneration) error {
	if old.State == "completed" && g.State != "completed" || old.State == "refused" && g.State != "refused" || old.State == "committed" && g.State == "pending" {
		return ErrCommandConflict
	}
	return nil
}

func writeGenerationRecordTx(ctx context.Context, tx *sql.Tx, g factory.IntentGeneration, b []byte) error {
	raw, err := json.Marshal(g)
	if err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `UPDATE factory_intent_generations SET data=$2 WHERE revision=$1 AND data=$3::jsonb`, g.RevisionID, string(raw), string(b))
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
	return nil
}

func (s *Store) RecordIntentGeneration(ctx context.Context, g factory.IntentGeneration) error {
	old, err := s.IntentGeneration(ctx, g.RevisionID)
	if err != nil {
		return err
	}
	a, b, err := compareGenerationRecord(old, g)
	if err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	same, err := sameJSONDocument(ctx, tx, a, b)
	if err != nil {
		return err
	}
	if !same {
		return ErrCommandConflict
	}
	if err = checkGenerationReceipt(g); err != nil {
		return err
	}
	if err = checkGenerationRollback(old, g); err != nil {
		return err
	}
	if err = writeGenerationRecordTx(ctx, tx, g, b); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) BindIntentTaskAcceptance(ctx context.Context, b factory.IntentTaskBinding) error {
	if !project.ValidDecisionID(b.RevisionID) || !project.ValidDecisionID(b.TaskKey) || !project.ValidDecisionID(b.AcceptanceID) || b.IssueID <= 0 || b.IssueIndex <= 0 || b.ContentVersion < 0 || b.CreatedUnix < 0 {
		return errors.New("invalid generated task binding")
	}
	raw, err := json.Marshal(b)
	if err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	var head string
	var intent string
	if err = tx.QueryRowContext(ctx, `SELECT intent FROM factory_intent_revisions WHERE id=$1`, b.RevisionID).Scan(&intent); err != nil {
		return err
	}
	if err = tx.QueryRowContext(ctx, `SELECT revision FROM factory_intent_heads WHERE id=$1 FOR UPDATE`, intent).Scan(&head); err != nil {
		return err
	}
	if head != b.RevisionID {
		return ErrStaleRevision
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO factory_intent_tasks(revision,task,data) VALUES($1,$2,$3) ON CONFLICT(revision,task) DO NOTHING`, b.RevisionID, b.TaskKey, string(raw))
	if err != nil {
		return err
	}
	var old []byte
	if err = tx.QueryRowContext(ctx, `SELECT data FROM factory_intent_tasks WHERE revision=$1 AND task=$2`, b.RevisionID, b.TaskKey).Scan(&old); err != nil {
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
// FW32 (T37): this tests for existence of a current accepted-head binding,
// never an arbitrary LIMIT 1 row over historical bindings. A copied frozen
// acceptance appears in several revisions at once; only a binding whose
// revision is still the accepted head counts as current.
func (s *Store) GeneratedIntentCurrent(ctx context.Context, acceptance string) (bool, bool, error) {
	var generated, current bool
	err := s.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM factory_intent_tasks WHERE data->>'acceptance_id'=$1), EXISTS(SELECT 1 FROM factory_intent_tasks t JOIN factory_intent_revisions r ON r.id=t.revision JOIN factory_intent_heads h ON h.id=r.intent AND h.revision=r.id WHERE t.data->>'acceptance_id'=$1 AND r.data->>'state'='accepted')`, acceptance).Scan(&generated, &current)
	if err != nil {
		return false, false, err
	}
	if !generated {
		return true, false, nil
	}
	return current, true, nil
}

// PendingIntentGenerations recovers accepted heads that lack confirmed graph
// adoption. Refused terminal operations await a newly reviewed revision.
func (s *Store) PendingIntentGenerations(ctx context.Context, after string, limit int) ([]factory.IntentRevision, error) {
	if limit < 1 || limit > 64 {
		return nil, errors.New("invalid pending intent limit")
	}
	rows, err := s.db.QueryContext(ctx, `SELECT r.data FROM factory_intent_heads h JOIN factory_intent_revisions r ON r.id=h.revision LEFT JOIN factory_intent_generations g ON g.revision=r.id WHERE r.id>$2 AND r.data->>'state'='accepted' AND (g.revision IS NULL OR g.data->>'state' IN ('pending','committed')) ORDER BY r.id LIMIT $1`, limit, after)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []factory.IntentRevision{}
	for rows.Next() {
		var raw []byte
		var r factory.IntentRevision
		if err = rows.Scan(&raw); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(raw, &r); err != nil {
			return nil, err
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

// PreviousGeneratedIntent follows recorded predecessors rather than assuming
// the previous review draft already owns a native graph.
func (s *Store) PreviousGeneratedIntent(ctx context.Context, intent, revision string) (factory.IntentRevision, error) {
	cursor := revision
	seen := map[string]bool{}
	for cursor != "" {
		if seen[cursor] || len(seen) > 256 {
			return factory.IntentRevision{}, errors.New("invalid intent revision history")
		}
		seen[cursor] = true
		var raw []byte
		if err := s.db.QueryRowContext(ctx, `SELECT data FROM factory_intent_revisions WHERE id=$1 AND intent=$2`, cursor, intent).Scan(&raw); err != nil {
			return factory.IntentRevision{}, err
		}
		var r factory.IntentRevision
		if err := json.Unmarshal(raw, &r); err != nil {
			return factory.IntentRevision{}, err
		}
		if cursor != revision && r.State == "accepted" {
			g, err := s.IntentGeneration(ctx, cursor)
			if err == nil && (g.State == "committed" || g.State == "completed") {
				return r, nil
			}
			if err != nil && !errors.Is(err, ErrNotFound) {
				return factory.IntentRevision{}, err
			}
		}
		cursor = r.Predecessor
	}
	return factory.IntentRevision{}, ErrNotFound
}
