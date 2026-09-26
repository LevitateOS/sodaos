package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/levitateos/sodaos/internal/factory"
)

// AdmitFactory is idempotent for an exact delivery. Another delivery cannot
// admit the same work while an attempt or its cleanup is still active.
func (s *Store) AdmitFactory(ctx context.Context, a factory.Attempt) (factory.Attempt, bool, error) {
	if err := validateFactoryAdmission(a); err != nil {
		return factory.Attempt{}, false, err
	}
	data, err := json.Marshal(a)
	if err != nil {
		return factory.Attempt{}, false, err
	}
	result, err := s.db.ExecContext(ctx, `INSERT INTO factory_attempts(id,repository_id,issue,delivery,phase,cleanup,revision,data) VALUES(?,?,?,?,?,?,0,?) ON CONFLICT(repository_id,issue,delivery) DO NOTHING`, a.ID, a.Work.RepositoryID, a.Work.Issue, a.Delivery, a.Phase, true, string(data))
	if err != nil {
		return factory.Attempt{}, false, fmt.Errorf("factory admission failed: %w", err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return factory.Attempt{}, false, err
	}
	if n == 1 {
		return a, true, nil
	}
	var existing []byte
	if err := s.db.QueryRowContext(ctx, `SELECT data FROM factory_attempts WHERE repository_id=? AND issue=? AND delivery=?`, a.Work.RepositoryID, a.Work.Issue, a.Delivery).Scan(&existing); err != nil {
		return factory.Attempt{}, false, err
	}
	if err := json.Unmarshal(existing, &a); err != nil {
		return factory.Attempt{}, false, err
	}
	return a, false, nil
}

func (s *Store) FactoryAttempt(ctx context.Context, id string) (factory.Attempt, error) {
	var a factory.Attempt
	var data []byte
	err := s.db.QueryRowContext(ctx, `SELECT data FROM factory_attempts WHERE id=?`, id).Scan(&data)
	if err == nil {
		err = json.Unmarshal(data, &a)
	}
	return a, err
}

// SaveFactoryAttempt rejects stale writers, including a controller trying to
// save progress after another process has withdrawn publication authority.
func (s *Store) SaveFactoryAttempt(ctx context.Context, a *factory.Attempt) (bool, error) {
	if err := a.Validate(); err != nil {
		return false, err
	}
	next := *a
	next.Revision++
	data, err := json.Marshal(next)
	if err != nil {
		return false, err
	}
	result, err := s.db.ExecContext(ctx, `UPDATE factory_attempts SET phase=?,cleanup=?,revision=?,data=? WHERE id=? AND revision=? AND (?=0 OR NOT EXISTS(SELECT 1 FROM factory_runs WHERE attempt_id=? AND (active=1 OR cleanup=0)))`, next.Phase, next.CleanupComplete, next.Revision, string(data), a.ID, a.Revision, next.CleanupComplete, a.ID)
	if err != nil {
		return false, err
	}
	n, err := result.RowsAffected()
	if err == nil && n == 1 {
		*a = next
	}
	return n == 1, err
}

// StartFactoryRun atomically consumes an execution and records resource intent.
// No runtime resource may be allocated before this transaction commits.
func (s *Store) StartFactoryRun(ctx context.Context, a *factory.Attempt, r factory.Run) error {
	attempt, err := admittedFactoryRun(*a, r)
	if err != nil {
		return err
	}
	adata, err := json.Marshal(attempt)
	if err != nil {
		return err
	}
	rdata, err := json.Marshal(r)
	if err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if err := insertFactoryRun(ctx, tx, *a, attempt, adata, r, rdata); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	*a = attempt
	return nil
}

func validateFactoryAdmission(a factory.Attempt) error {
	if err := a.Validate(); err != nil {
		return err
	}
	if a.Phase != factory.Implement || a.Executions != 0 || a.CIEvaluations != 0 || a.Revision != 0 || !a.CleanupComplete {
		return errors.New("admission must be a fresh attempt")
	}
	return nil
}

func admittedFactoryRun(a factory.Attempt, r factory.Run) (factory.Attempt, error) {
	if err := a.Validate(); err != nil {
		return a, err
	}
	if err := r.Validate(); err != nil {
		return a, err
	}
	if r.AttemptID != a.ID || r.Outcome != "" || r.CleanupComplete {
		return a, errors.New("new run must be active and owned by its attempt")
	}
	admitted, err := a.BeginRun(r.Role, r.Started)
	if err != nil {
		return a, err
	}
	if admitted.InputSHA != r.InputSHA || r.Deadline.After(admitted.Deadline) {
		return a, errors.New("run inputs differ from admitted role")
	}
	a.Revision++
	return a, nil
}

func insertFactoryRun(ctx context.Context, tx *sql.Tx, previous, next factory.Attempt, adata []byte, r factory.Run, rdata []byte) error {
	updated, err := tx.ExecContext(ctx, `UPDATE factory_attempts SET phase=?,cleanup=0,revision=?,data=? WHERE id=? AND revision=? AND phase=? AND json_extract(data,'$.executions')=?`, next.Phase, next.Revision, string(adata), previous.ID, previous.Revision, previous.Phase, previous.Executions)
	if err != nil {
		return err
	}
	n, err := updated.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return errors.New("attempt authority changed before launch")
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO factory_runs(id,attempt_id,active,cleanup,data) VALUES(?,?,1,0,?)`, r.ID, r.AttemptID, string(rdata))
	return err
}

func (s *Store) FactoryRun(ctx context.Context, id string) (factory.Run, error) {
	var r factory.Run
	var data []byte
	err := s.db.QueryRowContext(ctx, `SELECT data FROM factory_runs WHERE id=?`, id).Scan(&data)
	if err == nil {
		err = json.Unmarshal(data, &r)
	}
	return r, err
}

func (s *Store) SaveFactoryRun(ctx context.Context, r factory.Run) error {
	if err := r.Validate(); err != nil {
		return err
	}
	data, err := json.Marshal(r)
	if err != nil {
		return err
	}
	result, err := s.db.ExecContext(ctx, `UPDATE factory_runs SET active=?,cleanup=?,data=? WHERE id=? AND attempt_id=?`, r.Outcome == "", r.CleanupComplete, string(data), r.ID, r.AttemptID)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return ErrNotFound
	}
	return nil
}

// FactoryRuns returns only ledger-owned resources. Reconciliation never adopts
// containers or networks discovered by a prefix scan of the host runtime.
func (s *Store) FactoryRuns(ctx context.Context, attemptID string) ([]factory.Run, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT data FROM factory_runs WHERE attempt_id=? ORDER BY rowid`, attemptID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var runs []factory.Run
	for rows.Next() {
		var data []byte
		var r factory.Run
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(data, &r); err != nil {
			return nil, err
		}
		runs = append(runs, r)
	}
	return runs, rows.Err()
}

func (s *Store) FactoryAttempts(ctx context.Context) ([]factory.Attempt, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT data FROM factory_attempts WHERE phase!='finished' OR cleanup=0 ORDER BY rowid`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var attempts []factory.Attempt
	for rows.Next() {
		var data []byte
		var a factory.Attempt
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(data, &a); err != nil {
			return nil, err
		}
		attempts = append(attempts, a)
	}
	return attempts, rows.Err()
}
