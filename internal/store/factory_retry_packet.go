package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/levitateos/sodaos/internal/factory"
)

// RecordRetryPacket advances one assigned assignment to its next run
// identity and records the attempt's run, display binding and held
// capacity atomically. Attempts are bounded; an exhausted assignment
// refuses with ErrAssignmentActive so the caller finishes it instead of
// launching. The expected revision guards concurrent retries: only the
// first packet at this revision records, and the admission limits are
// rechecked inside the same transaction, so a retry never spends room a
// concurrent admission consumed.
func (s *Store) RecordRetryPacket(ctx context.Context, a factory.Assignment, run factory.Run, view factory.RunView, planned int) (factory.Assignment, error) {
	if err := run.Validate(); err != nil {
		return factory.Assignment{}, err
	}
	if err := view.Validate(); err != nil {
		return factory.Assignment{}, err
	}
	if run.ID == "" || view.RunID != run.ID || view.Repository != a.Repository || view.Issue != a.Issue || view.Attempt != a.ID {
		return factory.Assignment{}, errors.New("retry packet view does not match its attempt")
	}
	if run.Outcome != "" || run.Reconciled {
		return factory.Assignment{}, errors.New("retry packet carries a fresh run")
	}
	tx, err := s.begin(ctx)
	if err != nil {
		return factory.Assignment{}, err
	}
	defer func() { _ = tx.Rollback() }()
	var raw []byte
	if err = tx.queryRow(ctx, `SELECT data FROM factory_assignments WHERE id=? FOR UPDATE`, a.ID).Scan(&raw); err != nil {
		return factory.Assignment{}, err
	}
	var current factory.Assignment
	if err = json.Unmarshal(raw, &current); err != nil {
		return factory.Assignment{}, err
	}
	if current.Revision != a.Revision {
		return factory.Assignment{}, ErrStaleRevision
	}
	if current.Stage != factory.AssignmentAssigned || current.Attempts >= factory.MaxDispatchAttempts {
		return factory.Assignment{}, ErrAssignmentActive
	}
	for _, seen := range current.RunHistory {
		if seen == run.ID {
			return factory.Assignment{}, errors.New("dispatch attempt reuses a run identity")
		}
	}
	next := current
	next.Attempts++
	next.Revision++
	next.Run = run.ID
	next.RunHistory = append(append([]string(nil), current.RunHistory...), run.ID)
	if err = next.Validate(); err != nil {
		return factory.Assignment{}, err
	}
	data, err := json.Marshal(next)
	if err != nil {
		return factory.Assignment{}, err
	}
	if _, err = tx.exec(ctx, `UPDATE factory_assignments SET run=?,revision=?,data=? WHERE id=? AND stage='assigned'`,
		next.Run, next.Revision, string(data), a.ID); err != nil {
		return factory.Assignment{}, err
	}
	rundata, err := json.Marshal(run)
	if err != nil {
		return factory.Assignment{}, err
	}
	if _, err = tx.exec(ctx, `INSERT INTO factory_runs(id,active,settled,data) VALUES(?,TRUE,FALSE,?)`, run.ID, string(rundata)); err != nil {
		return factory.Assignment{}, fmt.Errorf("retry packet failed: %w", err)
	}
	if _, err = tx.exec(ctx, `INSERT INTO factory_run_views(run,repository,issue,attempt) VALUES(?,?,?,?)`,
		view.RunID, view.Repository, view.Issue, view.Attempt); err != nil {
		return factory.Assignment{}, fmt.Errorf("retry packet failed: %w", err)
	}
	var reservation factory.Reservation
	var rdata []byte
	if err = tx.queryRow(ctx, `SELECT data FROM factory_reservations WHERE assignment=? FOR UPDATE`, a.ID).Scan(&rdata); err != nil {
		return factory.Assignment{}, err
	}
	if err = json.Unmarshal(rdata, &reservation); err != nil {
		return factory.Assignment{}, err
	}
	switch reservation.State {
	case factory.ReservationReleased:
		reservation.State, reservation.PlannedMinutes = factory.ReservationHeld, planned
		reservation.Revision++
		if err = reservation.Validate(); err != nil {
			return factory.Assignment{}, err
		}
		rehold, err := json.Marshal(reservation)
		if err != nil {
			return factory.Assignment{}, err
		}
		if _, err = tx.exec(ctx, `UPDATE factory_reservations SET state='held',data=? WHERE assignment=? AND state='released'`,
			string(rehold), a.ID); err != nil {
			return factory.Assignment{}, fmt.Errorf("retry packet failed: %w", err)
		}
	case factory.ReservationHeld:
	default:
		return factory.Assignment{}, errors.New("reservation behind a retry is not held")
	}
	if err = checkAdmissionTx(ctx, tx, a.Repository, a.Connection, a.ProjectID); err != nil {
		return factory.Assignment{}, err
	}
	if err = tx.Commit(); err != nil {
		return factory.Assignment{}, err
	}
	return next, nil
}
