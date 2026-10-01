package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/levitateos/sodaos/internal/factory"
)

// ErrAssignmentActive reports a dispatch packet for an issue that already
// carries an unfinished assignment. One issue runs at most one assignment
// at a time; a concurrent dispatcher won this issue.
var ErrAssignmentActive = errors.New("issue already carries an unfinished assignment")

// MaxQueuedDispatch bounds one global queued-issue listing for dispatch
// visits. The dispatch pass visits oldest first under its own tighter
// bound; the store only caps the read.
const MaxQueuedDispatch = 2048

// storeAssignedLimit bounds one unfinished-assignment listing.
const storeAssignedLimit = 1024

// RecordDispatchPacket stores one dispatch atomically: the assigned
// assignment, its held reservation, the recorded run and the display
// binding. Either the whole packet lands or nothing does, so a crash
// never leaves a reservation without its assignment or a run without
// its dispatch. A second unfinished assignment for the issue refuses
// with ErrAssignmentActive instead of dispatching twice.
func (s *Store) RecordDispatchPacket(ctx context.Context, a factory.Assignment, r factory.Reservation, run factory.Run, view factory.RunView) error {
	if err := a.Validate(); err != nil {
		return err
	}
	if err := r.Validate(); err != nil {
		return err
	}
	if err := run.Validate(); err != nil {
		return err
	}
	if err := view.Validate(); err != nil {
		return err
	}
	if a.Stage != factory.AssignmentAssigned || a.Outcome != "" || a.Result != nil {
		return errors.New("dispatch packet carries a fresh assignment")
	}
	if r.AssignmentID != a.ID || r.Repository != a.Repository || r.Connection != a.Connection || r.State != factory.ReservationHeld {
		return errors.New("dispatch packet reservation does not match its assignment")
	}
	if run.ID != a.Run || run.Outcome != "" || run.Reconciled {
		return errors.New("dispatch packet run does not match its assignment")
	}
	if view.RunID != run.ID || view.Repository != a.Repository || view.Issue != a.Issue || view.Attempt != a.ID {
		return errors.New("dispatch packet view does not match its assignment")
	}
	adata, err := json.Marshal(a)
	if err != nil {
		return err
	}
	rdata, err := json.Marshal(r)
	if err != nil {
		return err
	}
	rundata, err := json.Marshal(run)
	if err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err = tx.ExecContext(ctx, `INSERT INTO factory_assignments(id,repository,issue,run,stage,revision,data) VALUES(?,?,?,?,?,?,?)`,
		a.ID, a.Repository, a.Issue, a.Run, a.Stage, a.Revision, string(adata)); err != nil {
		return dispatchPacketError(err)
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO factory_reservations(assignment,repository,connection,state,data) VALUES(?,?,?,?,?)`,
		r.AssignmentID, r.Repository, r.Connection, r.State, string(rdata)); err != nil {
		return fmt.Errorf("dispatch packet failed: %w", err)
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO factory_runs(id,active,settled,data) VALUES(?,1,0,?)`, run.ID, string(rundata)); err != nil {
		return fmt.Errorf("dispatch packet failed: %w", err)
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO factory_run_views(run,repository,issue,attempt) VALUES(?,?,?,?)`,
		view.RunID, view.Repository, view.Issue, view.Attempt); err != nil {
		return fmt.Errorf("dispatch packet failed: %w", err)
	}
	return tx.Commit()
}

func dispatchPacketError(err error) error {
	if err == nil {
		return nil
	}
	if strings.Contains(err.Error(), "UNIQUE constraint failed") {
		return ErrAssignmentActive
	}
	return fmt.Errorf("dispatch packet failed: %w", err)
}

// Assignment returns one dispatch assignment by identity.
func (s *Store) Assignment(ctx context.Context, id string) (factory.Assignment, error) {
	var a factory.Assignment
	var data []byte
	err := s.db.QueryRowContext(ctx, `SELECT data FROM factory_assignments WHERE id=?`, id).Scan(&data)
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
	err := s.db.QueryRowContext(ctx, `SELECT data FROM factory_assignments WHERE run=?`, runID).Scan(&data)
	if err == nil {
		err = json.Unmarshal(data, &a)
	}
	return a, err
}

// IssueAssignments lists every recorded assignment for one native issue,
// oldest first.
func (s *Store) IssueAssignments(ctx context.Context, repository, issue int64) ([]factory.Assignment, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT data FROM factory_assignments WHERE repository=? AND issue=? ORDER BY rowid LIMIT ?`,
		repository, issue, factory.MaxDispatchAttempts+8)
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

// NoteDispatchAttempt advances one assigned assignment to its next run
// identity. Attempts are bounded; an exhausted assignment refuses with
// ErrAssignmentActive so the caller finishes it instead of launching.
func (s *Store) NoteDispatchAttempt(ctx context.Context, id, runID string) (factory.Assignment, error) {
	current, err := s.Assignment(ctx, id)
	if err != nil {
		return factory.Assignment{}, err
	}
	if current.Stage != factory.AssignmentAssigned || current.Attempts >= factory.MaxDispatchAttempts {
		return factory.Assignment{}, ErrAssignmentActive
	}
	if !factory.ValidID(runID) {
		return factory.Assignment{}, errors.New("invalid dispatch attempt run")
	}
	for _, seen := range current.RunHistory {
		if seen == runID {
			return factory.Assignment{}, errors.New("dispatch attempt reuses a run identity")
		}
	}
	next := current
	next.Attempts++
	next.Revision++
	next.Run = runID
	next.RunHistory = append(append([]string(nil), current.RunHistory...), runID)
	if err = next.Validate(); err != nil {
		return factory.Assignment{}, err
	}
	data, err := json.Marshal(next)
	if err != nil {
		return factory.Assignment{}, err
	}
	result, err := s.db.ExecContext(ctx, `UPDATE factory_assignments SET run=?,revision=?,data=? WHERE id=? AND revision=? AND stage='assigned'`,
		next.Run, next.Revision, string(data), id, current.Revision)
	if err != nil {
		return factory.Assignment{}, err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return factory.Assignment{}, err
	}
	if n != 1 {
		return factory.Assignment{}, ErrStaleRevision
	}
	return next, nil
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
	result, err := s.db.ExecContext(ctx, `UPDATE factory_assignments SET stage='finished',data=? WHERE id=? AND revision=? AND stage='assigned'`,
		string(data), a.ID, a.Revision)
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

// AssignedAssignments lists every unfinished dispatch assignment, oldest
// first, for crash recovery. Recovery resolves each one to consumed,
// released or finished; deferred work waits for a later pass.
func (s *Store) AssignedAssignments(ctx context.Context, limit int) ([]factory.Assignment, error) {
	if limit <= 0 || limit > storeAssignedLimit {
		return nil, errors.New("invalid assigned assignment listing limit")
	}
	rows, err := s.db.QueryContext(ctx, `SELECT data FROM factory_assignments WHERE stage='assigned' ORDER BY rowid LIMIT ?`, limit)
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

// QueuedControls lists queued readiness records across every repository,
// oldest first seen first with repository and issue breaking ties. The
// order is deterministic: the same records always visit in the same order.
func (s *Store) QueuedControls(ctx context.Context, limit int) ([]factory.IssueControl, error) {
	if limit <= 0 || limit > MaxQueuedDispatch {
		return nil, errors.New("invalid queued control listing limit")
	}
	rows, err := s.db.QueryContext(ctx, `SELECT data FROM issue_controls
		WHERE json_extract(data,'$.readiness')='queued'
		ORDER BY json_extract(data,'$.first_seen_unix') ASC, repository ASC, issue ASC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var controls []factory.IssueControl
	for rows.Next() {
		var data []byte
		var control factory.IssueControl
		if err = rows.Scan(&data); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(data, &control); err != nil {
			return nil, err
		}
		controls = append(controls, control)
	}
	return controls, rows.Err()
}

// ActiveRunCounts counts active recorded runs with no dispatch assignment:
// human and fixture runs that still occupy appliance and repository slots.
// Attributed runs count through their held reservations instead, never twice.
func (s *Store) ActiveRunCounts(ctx context.Context) (int, map[string]int, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT json_extract(data,'$.project_id') FROM factory_runs
		WHERE active=1 AND id NOT IN (SELECT run FROM factory_assignments WHERE run!='') LIMIT 1001`)
	if err != nil {
		return 0, nil, err
	}
	defer func() { _ = rows.Close() }()
	byProject := map[string]int{}
	total := 0
	for rows.Next() {
		var projectID string
		if err = rows.Scan(&projectID); err != nil {
			return 0, nil, err
		}
		total++
		byProject[projectID]++
	}
	return total, byProject, rows.Err()
}
