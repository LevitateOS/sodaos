package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strconv"

	"github.com/levitateos/sodaos/internal/factory"
)

func (s *Store) CreateDepartment(ctx context.Context, d factory.Department) error {
	if err := d.Validate(); err != nil {
		return err
	}
	raw, err := json.Marshal(d)
	if err != nil {
		return err
	}
	result, err := s.db.ExecContext(ctx, `INSERT INTO factory_departments(id,owner_id,data) VALUES($1,$2,$3) ON CONFLICT(id) DO NOTHING`, d.ID, d.Owner, string(raw))
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrCommandConflict
	}
	return nil
}

func (s *Store) Departments(ctx context.Context, owner int64) ([]factory.Department, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT data FROM factory_departments WHERE owner_id=$1 ORDER BY id LIMIT 101`, owner)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []factory.Department{}
	for rows.Next() {
		var raw []byte
		var d factory.Department
		if err = rows.Scan(&raw); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(raw, &d); err != nil {
			return nil, err
		}
		result = append(result, d)
	}
	return result, rows.Err()
}

// SaveIntentRevision serializes revisions on a durable head and preserves all
// review history. Identity replay is exact; changed content cannot overwrite it.
func (s *Store) SaveIntentRevision(ctx context.Context, r factory.IntentRevision) error {
	if err := r.Validate(); err != nil {
		return err
	}
	raw, err := json.Marshal(r)
	if err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if r.Department != "" {
		var owner int64
		if err = tx.QueryRowContext(ctx, `SELECT owner_id FROM factory_departments WHERE id=$1`, r.Department).Scan(&owner); err != nil {
			return err
		}
		if owner != r.Owner {
			return errors.New("department is private to another owner")
		}
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO factory_intent_heads(id,owner_id,revision) VALUES($1,$2,'') ON CONFLICT(id) DO NOTHING`, r.Intent, r.Owner)
	if err != nil {
		return err
	}
	var head string
	var owner int64
	if err = tx.QueryRowContext(ctx, `SELECT revision,owner_id FROM factory_intent_heads WHERE id=$1 FOR UPDATE`, r.Intent).Scan(&head, &owner); err != nil {
		return err
	}
	if owner != r.Owner {
		return ErrCommandConflict
	}
	if head == r.ID {
		var old []byte
		if err = tx.QueryRowContext(ctx, `SELECT data FROM factory_intent_revisions WHERE id=$1`, r.ID).Scan(&old); err != nil {
			return err
		}
		same, e := sameJSONDocument(ctx, tx, old, raw)
		if e != nil {
			return e
		}
		if !same {
			return ErrCommandConflict
		}
		return tx.Commit()
	}
	if head != r.Predecessor {
		return ErrStaleRevision
	}
	if head != "" {
		var headRaw []byte
		var headRev factory.IntentRevision
		if err = tx.QueryRowContext(ctx, `SELECT data FROM factory_intent_revisions WHERE id=$1`, head).Scan(&headRaw); err != nil {
			return err
		}
		if err = json.Unmarshal(headRaw, &headRev); err != nil {
			return err
		}
		if headRev.State == "accepted" && r.Repository != headRev.Repository {
			return errors.New("repository change needs a separate build request")
		}
		// The first commitment pins the repository for the whole intent:
		// any later draft, rejected, declined, or abandoned head keeps it.
		// Revision rows are immutable and the head lock serializes saves,
		// so one committed row read here binds this save exactly.
		var committed string
		err = tx.QueryRowContext(ctx, `SELECT data->>'repository' FROM factory_intent_revisions WHERE intent=$1 AND data->>'state'='accepted' LIMIT 1`, r.Intent).Scan(&committed)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		if err == nil && committed != strconv.FormatInt(r.Repository, 10) {
			return errors.New("repository change needs a separate build request")
		}
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO factory_intent_revisions(id,intent,data) VALUES($1,$2,$3)`, r.ID, r.Intent, string(raw)); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE factory_intent_heads SET revision=$2 WHERE id=$1`, r.Intent, r.ID); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) IntentRevision(ctx context.Context, intent string, owner int64) (factory.IntentRevision, error) {
	var r factory.IntentRevision
	var raw []byte
	err := s.db.QueryRowContext(ctx, `SELECT r.data FROM factory_intent_heads h JOIN factory_intent_revisions r ON r.id=h.revision WHERE h.id=$1 AND h.owner_id=$2`, intent, owner).Scan(&raw)
	if err == nil {
		err = json.Unmarshal(raw, &r)
	}
	return r, err
}

// ListIntentRevisions returns bounded owner-private resume summaries with a
// stable intent-ID cursor. It lists only the caller's current heads; another
// actor's drafts and inaccessible private details never appear. Empty returns
// an empty slice, never a null; database failures return an error for 503.
func (s *Store) ListIntentRevisions(ctx context.Context, owner int64, f factory.IntentListFilter) ([]factory.IntentSummary, string, error) {
	if owner <= 0 {
		return nil, "", errors.New("invalid intent owner")
	}
	if f.Limit == 0 {
		f.Limit = 20
	}
	if err := f.Validate(); err != nil {
		return nil, "", err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT h.id, r.data FROM factory_intent_heads h JOIN factory_intent_revisions r ON r.id=h.revision WHERE h.owner_id=$1 AND h.id>$2 AND ($3='' OR r.data->>'state'=$3) AND ($4<=0 OR (r.data->>'repository')::bigint=$4) AND ($5='' OR r.data->>'department'=$5) ORDER BY h.id LIMIT $6`, owner, f.Cursor, f.State, f.Repository, f.Department, f.Limit+1)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()
	result := []factory.IntentSummary{}
	for rows.Next() {
		var id string
		var raw []byte
		var r factory.IntentRevision
		if err = rows.Scan(&id, &raw); err != nil {
			return nil, "", err
		}
		if err = json.Unmarshal(raw, &r); err != nil {
			return nil, "", err
		}
		if err = r.Validate(); err != nil {
			return nil, "", err
		}
		result = append(result, r.Summary())
	}
	if err = rows.Err(); err != nil {
		return nil, "", err
	}
	next := ""
	if len(result) > f.Limit {
		next = result[f.Limit-1].Intent
		result = result[:f.Limit]
	}
	return result, next, nil
}

// IntentHistory returns immutable revisions for one owner-private request,
// newest first, bounded to 50 entries. It follows the head backwards through
// recorded predecessors; a repository change after commitment creates a
// separate intent identity and never appears here.
func (s *Store) IntentHistory(ctx context.Context, intent string, owner int64, limit int) ([]factory.IntentRevision, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit < 1 || limit > 50 {
		return nil, errors.New("invalid history limit")
	}
	head, err := s.IntentRevision(ctx, intent, owner)
	if err != nil {
		return nil, err
	}
	result := []factory.IntentRevision{}
	cursor := head.ID
	seen := map[string]bool{}
	for cursor != "" && len(result) < limit {
		if seen[cursor] || len(seen) > 256 {
			return nil, errors.New("invalid intent revision history")
		}
		seen[cursor] = true
		var raw []byte
		var r factory.IntentRevision
		if err = s.db.QueryRowContext(ctx, `SELECT data FROM factory_intent_revisions WHERE id=$1 AND intent=$2`, cursor, intent).Scan(&raw); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(raw, &r); err != nil {
			return nil, err
		}
		if r.Owner != owner {
			return nil, ErrNotFound
		}
		result = append(result, r)
		cursor = r.Predecessor
	}
	return result, nil
}

// ReservePriorityJudgment consumes the maximum possible call count before any
// request. Failed or interrupted consultations retain the reservation.
func (s *Store) ReservePriorityJudgment(ctx context.Context, snapshot string, maxCalls int) (bool, error) {
	if !factory.ValidDigest(snapshot) || maxCalls < 1 || maxCalls > 16 {
		return false, errors.New("invalid priority reservation")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback() }()
	var spent int
	if err = tx.QueryRowContext(ctx, `SELECT spent FROM factory_judgment_budget WHERE id=1 FOR UPDATE`).Scan(&spent); err != nil {
		return false, err
	}
	var existing int
	if err = tx.QueryRowContext(ctx, `SELECT count(*) FROM factory_priority_judgments WHERE snapshot=$1`, snapshot).Scan(&existing); err != nil {
		return false, err
	}
	if existing > 0 || s.PriorityCallBudget-spent < maxCalls {
		return false, nil
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO factory_priority_judgments(snapshot,max_calls) VALUES($1,$2)`, snapshot, maxCalls); err != nil {
		return false, err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE factory_judgment_budget SET spent=spent+$1 WHERE id=1`, maxCalls); err != nil {
		return false, err
	}
	return true, tx.Commit()
}

func (s *Store) RecordPriorityJudgment(ctx context.Context, r factory.PriorityReceipt) error {
	if !factory.ValidDigest(r.Snapshot) || r.CreatedUnix <= 0 {
		return errors.New("invalid priority receipt")
	}
	raw, err := json.Marshal(r)
	if err != nil {
		return err
	}
	result, err := s.db.ExecContext(ctx, `UPDATE factory_priority_judgments SET data=$2 WHERE snapshot=$1 AND data IS NULL`, r.Snapshot, string(raw))
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 1 {
		return nil
	}
	if len(r.Calls) > 0 {
		return ErrCommandConflict
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO factory_priority_judgments(snapshot,max_calls,data) VALUES($1,0,$2) ON CONFLICT(snapshot) DO NOTHING`, r.Snapshot, string(raw))
	return err
}

// ReserveIntentJudgment admits one planning consultation against the current
// independent allowance for this owner-private intent, plus the operator's
// global consultation ceiling, under one transaction. The embedded revision
// PlanningCalls field no longer funds anything: a missing allowance or an
// exhausted amount refuses with ErrPlanningAllowanceExhausted, without
// fallback and without inventing funding.
func (s *Store) ReserveIntentJudgment(ctx context.Context, r factory.IntentRevision, snapshot string) (bool, error) {
	if !factory.ValidDigest(snapshot) {
		return false, errors.New("invalid planning snapshot")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback() }()
	var head string
	if err = tx.QueryRowContext(ctx, `SELECT revision FROM factory_intent_heads WHERE id=$1 AND owner_id=$2 FOR UPDATE`, r.Intent, r.Owner).Scan(&head); err != nil {
		return false, err
	}
	if head != r.ID {
		return false, ErrStaleRevision
	}
	var allowanceRevision string
	if err = tx.QueryRowContext(ctx, `SELECT revision FROM factory_intent_allowance_heads WHERE intent=$1 AND owner_id=$2 FOR UPDATE`, r.Intent, r.Owner).Scan(&allowanceRevision); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, ErrPlanningAllowanceExhausted
		}
		return false, err
	}
	var allowanceRaw []byte
	var allowance factory.PlanningAllowance
	if err = tx.QueryRowContext(ctx, `SELECT data FROM factory_intent_allowances WHERE id=$1`, allowanceRevision).Scan(&allowanceRaw); err != nil {
		return false, err
	}
	if err = json.Unmarshal(allowanceRaw, &allowance); err != nil {
		return false, err
	}
	if allowance.Intent != r.Intent || allowance.Owner != r.Owner || allowance.Unit != factory.PlanningAllowanceUnit {
		return false, errors.New("planning allowance identity mismatch")
	}
	var spent, used, existing int
	if err = tx.QueryRowContext(ctx, `SELECT spent FROM factory_judgment_budget WHERE id=1 FOR UPDATE`).Scan(&spent); err != nil {
		return false, err
	}
	if err = tx.QueryRowContext(ctx, `SELECT count(*) FROM factory_intent_planning_calls WHERE intent=$1`, r.Intent).Scan(&used); err != nil {
		return false, err
	}
	if err = tx.QueryRowContext(ctx, `SELECT count(*) FROM factory_priority_judgments WHERE snapshot=$1`, snapshot).Scan(&existing); err != nil {
		return false, err
	}
	if used >= allowance.Amount {
		return false, ErrPlanningAllowanceExhausted
	}
	if existing > 0 || spent >= s.PriorityCallBudget {
		return false, nil
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO factory_priority_judgments(snapshot,max_calls) VALUES($1,1)`, snapshot); err != nil {
		return false, err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO factory_intent_planning_calls(intent,snapshot) VALUES($1,$2)`, r.Intent, snapshot); err != nil {
		return false, err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE factory_judgment_budget SET spent=spent+1 WHERE id=1`); err != nil {
		return false, err
	}
	return true, tx.Commit()
}

func (s *Store) PriorityJudgment(ctx context.Context, snapshot string) (factory.PriorityReceipt, error) {
	var r factory.PriorityReceipt
	var raw []byte
	err := s.db.QueryRowContext(ctx, `SELECT data FROM factory_priority_judgments WHERE snapshot=$1 AND data IS NOT NULL`, snapshot).Scan(&raw)
	if err == nil {
		err = json.Unmarshal(raw, &r)
	}
	return r, err
}

func (s *Store) ReserveIssuePlanning(ctx context.Context, acceptance, snapshot string) (bool, error) {
	var raw []byte
	err := s.db.QueryRowContext(ctx, `SELECT r.data FROM factory_intent_tasks t JOIN factory_intent_revisions r ON r.id=t.revision JOIN factory_intent_heads h ON h.id=r.intent AND h.revision=r.id WHERE t.data->>'acceptance_id'=$1 AND r.data->>'state'='accepted' LIMIT 1`, acceptance).Scan(&raw)
	if errors.Is(err, ErrNotFound) {
		if current, generated, e := s.GeneratedIntentCurrent(ctx, acceptance); e != nil {
			return false, e
		} else if generated && !current {
			return false, ErrStaleRevision
		}
		return s.ReservePriorityJudgment(ctx, snapshot, 1)
	}
	if err != nil {
		return false, err
	}
	var r factory.IntentRevision
	if err = json.Unmarshal(raw, &r); err != nil {
		return false, err
	}
	return s.ReserveIntentJudgment(ctx, r, snapshot)
}
