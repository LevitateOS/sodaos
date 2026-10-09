package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
)

// RecordRetryPacket advances one assigned assignment to its next run
// identity and records the attempt's run, display binding and held
// capacity atomically. Attempts are bounded; an exhausted assignment
// refuses with ErrAssignmentActive so the caller finishes it instead of
// launching. The expected revision guards concurrent retries: only the
// first packet at this revision records. Admission limits and fresh authority
// are rechecked inside the same transaction, so a retry never spends room a
// concurrent admission consumed.
func (s *Store) RecordRetryPacket(ctx context.Context, a factory.Assignment, expected factory.IssueControl, authority factory.AuthorityRef, run factory.Run, view factory.RunView, planned int) (factory.Assignment, error) {
	if err := run.Validate(); err != nil {
		return factory.Assignment{}, err
	}
	if err := view.Validate(); err != nil {
		return factory.Assignment{}, err
	}
	if run.ID == "" || run.ProjectID != a.ProjectID || run.Role != a.Role || run.InputSHA != a.SourceCommit ||
		view.RunID != run.ID || view.Repository != a.Repository || view.Issue != a.Issue || view.Attempt != a.ID {
		return factory.Assignment{}, errors.New("retry packet view does not match its attempt")
	}
	if run.Outcome != "" || run.Reconciled {
		return factory.Assignment{}, errors.New("retry packet carries a fresh run")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return factory.Assignment{}, err
	}
	defer func() { _ = tx.Rollback() }()
	// Match dispatch and grant-command lock order before reading the current
	// selected connection sponsorship and its budget.
	if _, _, _, err = ensureDispatchGateTx(ctx, tx, a.Repository); err != nil {
		return factory.Assignment{}, err
	}
	if err = checkQueuedControlTx(ctx, tx, expected, a.Repository, a.Issue, a.Acceptance); err != nil {
		return factory.Assignment{}, err
	}
	var raw []byte
	if err = tx.QueryRowContext(ctx, `SELECT data FROM factory_assignments WHERE id=$1 FOR UPDATE`, a.ID).Scan(&raw); err != nil {
		return factory.Assignment{}, err
	}
	var current factory.Assignment
	if err = json.Unmarshal(raw, &current); err != nil {
		return factory.Assignment{}, err
	}
	if current.Revision != a.Revision {
		return factory.Assignment{}, ErrStaleRevision
	}
	if current.Repository != a.Repository || current.Issue != a.Issue || current.Run != a.Run || current.Connection != a.Connection ||
		run.ProjectID != current.ProjectID || run.Role != current.Role || run.InputSHA != current.SourceCommit ||
		run.Harness != current.Harness || run.Model != current.Model {
		return factory.Assignment{}, ErrStaleRevision
	}
	if current.AttemptRoot != a.AttemptRoot || current.PublicationAssignment != a.PublicationAssignment {
		return factory.Assignment{}, ErrStaleRevision
	}
	if current.Stage != factory.AssignmentAssigned || current.Attempts >= factory.MaxDispatchAttempts {
		return factory.Assignment{}, ErrAssignmentActive
	}
	if err = validateAttemptPacketTx(ctx, tx, current, false); err != nil {
		return factory.Assignment{}, err
	}
	grants, err := loadAdmissionGrantsTx(ctx, tx, current.Repository, current.Connection)
	if err != nil {
		return factory.Assignment{}, err
	}
	if err = checkAssignmentAuthorityTx(ctx, tx, authority, current, grants); err != nil {
		return factory.Assignment{}, err
	}
	var previousData []byte
	if err = tx.QueryRowContext(ctx, `SELECT data FROM factory_runs WHERE id=$1 FOR UPDATE`, current.Run).Scan(&previousData); err != nil {
		return factory.Assignment{}, err
	}
	var previous factory.Run
	if err = json.Unmarshal(previousData, &previous); err != nil {
		return factory.Assignment{}, err
	}
	if previous.Admission == nil || previous.Admission.Policy.TargetBranch != grants.policy.TargetBranch ||
		!slices.Equal(previous.Admission.Policy.Checks, grants.policy.Checks) ||
		previous.Admission.Policy.Roles[current.Role] != grants.policy.Roles[current.Role] ||
		previous.Admission.Profile != *grants.environment.Profile {
		return factory.Assignment{}, ErrAdmissionChanged
	}
	run.Admission = &factory.RunAdmission{Authority: authority, Policy: grants.policy, Profile: *grants.environment.Profile}
	if err = run.Validate(); err != nil {
		return factory.Assignment{}, err
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
	if _, err = tx.ExecContext(ctx, `UPDATE factory_assignments SET run=$1,revision=$2,data=$3 WHERE id=$4 AND stage='assigned'`,
		next.Run, next.Revision, string(data), a.ID); err != nil {
		return factory.Assignment{}, err
	}
	rundata, err := json.Marshal(run)
	if err != nil {
		return factory.Assignment{}, err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO factory_runs(id,active,settled,data) VALUES($1,TRUE,FALSE,$2)`, run.ID, string(rundata)); err != nil {
		return factory.Assignment{}, fmt.Errorf("retry packet failed: %w", err)
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO factory_run_views(run,repository,issue,attempt) VALUES($1,$2,$3,$4)`,
		view.RunID, view.Repository, view.Issue, view.Attempt); err != nil {
		return factory.Assignment{}, fmt.Errorf("retry packet failed: %w", err)
	}
	var reservation factory.Reservation
	var rdata []byte
	if err = tx.QueryRowContext(ctx, `SELECT data FROM factory_reservations WHERE assignment=$1 FOR UPDATE`, a.ID).Scan(&rdata); err != nil {
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
		if _, err = tx.ExecContext(ctx, `UPDATE factory_reservations SET state='held',data=$1 WHERE assignment=$2 AND state='released'`,
			string(rehold), a.ID); err != nil {
			return factory.Assignment{}, fmt.Errorf("retry packet failed: %w", err)
		}
	case factory.ReservationHeld:
	default:
		return factory.Assignment{}, errors.New("reservation behind a retry is not held")
	}
	if err = checkAdmissionLimitsTx(ctx, tx, current.Repository, current.Connection, current.ProjectID, grants); err != nil {
		return factory.Assignment{}, err
	}
	if err = admitAttemptAllowanceTx(ctx, tx, a.Repository, a.Issue, current.AttemptRoot, nil, run.Deadline, time.Now()); err != nil {
		return factory.Assignment{}, err
	}
	if err = tx.Commit(); err != nil {
		return factory.Assignment{}, err
	}
	return next, nil
}
