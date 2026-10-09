package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
)

// storeAssignedLimit bounds one unfinished-assignment listing.
const storeAssignedLimit = 1024

// Assignment returns one dispatch assignment by identity.
func (s *Store) Assignment(ctx context.Context, id string) (factory.Assignment, error) {
	var a factory.Assignment
	var data []byte
	err := s.db.QueryRowContext(ctx, `SELECT data FROM factory_assignments WHERE id=$1`, id).Scan(&data)
	if err == nil {
		err = json.Unmarshal(data, &a)
	}
	return a, err
}

// AssignmentByRun returns the assignment whose latest attempt runs under
// the given run identity.
func (s *Store) AssignmentByRun(ctx context.Context, runID string) (factory.Assignment, error) {
	var a factory.Assignment
	var data []byte
	err := s.db.QueryRowContext(ctx, `SELECT data FROM factory_assignments WHERE run=$1`, runID).Scan(&data)
	if err == nil {
		err = json.Unmarshal(data, &a)
	}
	return a, err
}

// FinishAssignment stores one assignment's terminal outcome exactly once.
// Only an assigned assignment at its recorded revision finishes; anything
// else reports stale instead of overwriting a concurrent finish.
func (s *Store) FinishAssignment(ctx context.Context, a factory.Assignment) error {
	if err := a.Validate(); err != nil {
		return err
	}
	if a.Stage != factory.AssignmentFinished {
		return errors.New("only a finished assignment finishes")
	}
	data, err := json.Marshal(a)
	if err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	paused, err := attemptPauseActiveTx(ctx, tx, a.Repository)
	if err != nil {
		return err
	}
	var currentData []byte
	if err = tx.QueryRowContext(ctx, `SELECT data FROM factory_assignments WHERE id=$1 FOR UPDATE`, a.ID).Scan(&currentData); err != nil {
		return err
	}
	var current factory.Assignment
	if err = json.Unmarshal(currentData, &current); err != nil {
		return err
	}
	if current.Revision != a.Revision || current.Stage != factory.AssignmentAssigned || current.AttemptRoot != a.AttemptRoot || current.PublicationAssignment != a.PublicationAssignment {
		return ErrNotFound
	}
	result, err := tx.ExecContext(ctx, `UPDATE factory_assignments SET stage='finished',data=$1
		WHERE id=$2 AND revision=$3 AND stage='assigned'
		AND data->>'attempt_root'=$4 AND data->>'publication_assignment'=$5`,
		string(data), a.ID, a.Revision, a.AttemptRoot, a.PublicationAssignment)
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
	if !(a.Outcome == factory.Succeeded && a.Reason == factory.AssignReasonReported) {
		closeRoot := a.Reason != factory.AssignReasonSuperseded && a.Reason != factory.AssignReasonWithdrawn && a.Reason != factory.AssignReasonCancelled
		if paused {
			closeRoot = false
		}
		if err = transitionAttemptAssignmentTx(ctx, tx, a.ID, closeRoot, time.Now()); err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
	}
	return tx.Commit()
}

// AssignedAssignments lists every unfinished dispatch assignment, oldest
// first, for crash recovery. Recovery resolves each one to consumed,
// released or finished; deferred work waits for a later pass.
func (s *Store) AssignedAssignments(ctx context.Context, limit int) ([]factory.Assignment, error) {
	if limit <= 0 || limit > storeAssignedLimit {
		return nil, errors.New("invalid assigned assignment listing limit")
	}
	rows, err := s.db.QueryContext(ctx, `SELECT data FROM factory_assignments WHERE stage='assigned' ORDER BY seq LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []factory.Assignment
	for rows.Next() {
		var data []byte
		var a factory.Assignment
		if err = rows.Scan(&data); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(data, &a); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}
