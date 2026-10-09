package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"math"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
)

var (
	ErrAttemptRunsPending  = errors.New("attempt runs are not confirmed settled")
	ErrAttemptRootMismatch = errors.New("assignment does not match the recorded attempt root")
)

// AttemptAllowance returns the newest attempt allowance for the issue.
// Explicit maintainer retries keep earlier allowance rows in history.
func (s *Store) AttemptAllowance(ctx context.Context, repository, issue int64) (factory.AttemptAllowance, error) {
	var allowance factory.AttemptAllowance
	var root string
	var revision int64
	var data []byte
	err := s.db.QueryRowContext(ctx, `SELECT l.root_assignment,l.revision,l.data FROM factory_attempt_allowances l JOIN factory_assignments a ON a.id=l.root_assignment WHERE l.repository=$1 AND l.issue=$2 ORDER BY a.seq DESC LIMIT 1`, repository, issue).Scan(&root, &revision, &data)
	if err != nil {
		return allowance, err
	}
	if err = json.Unmarshal(data, &allowance); err != nil {
		return allowance, err
	}
	if err = validateAttemptAllowanceRow(allowance, repository, issue, root, revision); err != nil {
		return factory.AttemptAllowance{}, err
	}
	return allowance, nil
}

// CapAttemptDeadline returns the earlier of the requested absolute expiry and
// the exact assignment root's active-time expiry. Native operations are
// admitted against this immutable wall-clock bound; existing intents are
// always replayed from their recorded NotAfter instead of calling this again.
func (s *Store) CapAttemptDeadline(ctx context.Context, assignmentID string, requested int64) (int64, error) {
	if !factory.ValidID(assignmentID) || requested <= 0 {
		return 0, factory.ErrAttemptTimeExhausted
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()
	var raw []byte
	if err = tx.QueryRowContext(ctx, `SELECT data FROM factory_assignments WHERE id=$1`, assignmentID).Scan(&raw); err != nil {
		return 0, err
	}
	var assignment factory.Assignment
	if err = json.Unmarshal(raw, &assignment); err != nil {
		return 0, err
	}
	if err = assignment.Validate(); err != nil {
		return 0, err
	}
	allowance, err := lockAttemptAllowanceRootTx(ctx, tx, assignment.Repository, assignment.Issue, assignment.AttemptRoot)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, factory.ErrAttemptTimeExhausted
	}
	if err != nil {
		return 0, err
	}
	now := time.Now()
	deadline, err := attemptDeadlineUnix(allowance, now)
	if err != nil {
		return 0, err
	}
	if requested < deadline {
		deadline = requested
	}
	if deadline <= now.Unix() {
		return 0, factory.ErrAttemptTimeExhausted
	}
	if err = tx.Commit(); err != nil {
		return 0, err
	}
	return deadline, nil
}

func attemptDeadlineUnix(allowance factory.AttemptAllowance, now time.Time) (int64, error) {
	if err := allowance.Validate(); err != nil {
		return 0, factory.ErrAttemptTimeExhausted
	}
	if now.Unix() < allowance.CheckpointUnix {
		return 0, factory.ErrAllowanceClock
	}
	if allowance.Closed {
		return 0, factory.ErrAttemptClosed
	}
	if !allowance.Active {
		return 0, factory.ErrAttemptTimeExhausted
	}
	remaining := int64(allowance.Limits.ActiveMinutes)*60 - allowance.ActiveSeconds
	if remaining <= 0 || allowance.CheckpointUnix > math.MaxInt64-remaining {
		return 0, factory.ErrAttemptTimeExhausted
	}
	deadline := allowance.CheckpointUnix + remaining
	if deadline <= now.Unix() {
		return 0, factory.ErrAttemptTimeExhausted
	}
	return deadline, nil
}

func validateAttemptAllowanceRow(a factory.AttemptAllowance, repository, issue int64, root string, revision int64) error {
	if err := a.Validate(); err != nil {
		return err
	}
	if a.Repository != repository || a.Issue != issue || a.RootAssignment != root || a.Revision != revision {
		return errors.New("attempt allowance row binding does not match its data")
	}
	return nil
}

func lockAttemptAllowanceTx(ctx context.Context, tx *sql.Tx, repository, issue int64) (factory.AttemptAllowance, error) {
	var allowance factory.AttemptAllowance
	var root string
	var revision int64
	var data []byte
	err := tx.QueryRowContext(ctx, `SELECT l.root_assignment,l.revision,l.data FROM factory_attempt_allowances l JOIN factory_assignments a ON a.id=l.root_assignment WHERE l.repository=$1 AND l.issue=$2 ORDER BY a.seq DESC LIMIT 1 FOR UPDATE OF l`, repository, issue).Scan(&root, &revision, &data)
	if err != nil {
		return allowance, err
	}
	if err = json.Unmarshal(data, &allowance); err != nil {
		return allowance, err
	}
	if err = validateAttemptAllowanceRow(allowance, repository, issue, root, revision); err != nil {
		return factory.AttemptAllowance{}, err
	}
	return allowance, nil
}

func lockAttemptAllowanceRootTx(ctx context.Context, tx *sql.Tx, repository, issue int64, root string) (factory.AttemptAllowance, error) {
	var allowance factory.AttemptAllowance
	var storedRoot string
	var revision int64
	var data []byte
	err := tx.QueryRowContext(ctx, `SELECT root_assignment,revision,data FROM factory_attempt_allowances WHERE repository=$1 AND issue=$2 AND root_assignment=$3 FOR UPDATE`, repository, issue, root).Scan(&storedRoot, &revision, &data)
	if err != nil {
		return allowance, err
	}
	if err = json.Unmarshal(data, &allowance); err != nil {
		return allowance, err
	}
	if err = validateAttemptAllowanceRow(allowance, repository, issue, storedRoot, revision); err != nil {
		return factory.AttemptAllowance{}, err
	}
	return allowance, nil
}

// attemptRootRunsSettledTx confirms every recorded run view for this root has
// stopped and been reconciled. Retained historical attempts remain in scope.
func attemptRootRunsSettledTx(ctx context.Context, tx *sql.Tx, repository, issue int64, root string) (bool, error) {
	var settled bool
	err := tx.QueryRowContext(ctx, `SELECT NOT EXISTS (
		SELECT 1 FROM factory_runs r
		JOIN factory_run_views v ON v.run=r.id
		JOIN factory_assignments a ON a.id=v.attempt
		WHERE (r.active OR NOT r.settled OR COALESCE(r.data->>'reconciled','false') <> 'true')
		  AND a.repository=$1 AND a.issue=$2 AND a.data->>'attempt_root'=$3
		LIMIT 1
	)`, repository, issue, root).Scan(&settled)
	return settled, err
}

// transitionAttemptRootTx applies durable lifecycle state only after every
// descendant process has stopped. A terminal decision may close admission
// immediately while retaining its active clock/slot until settlement.
func transitionAttemptRootTx(ctx context.Context, tx *sql.Tx, repository, issue int64, root string, closeRoot bool, now time.Time) error {
	allowance, err := lockAttemptAllowanceRootTx(ctx, tx, repository, issue, root)
	if err != nil {
		return err
	}
	return transitionLockedAttemptRootTx(ctx, tx, allowance, closeRoot, now)
}

func transitionLockedAttemptRootTx(ctx context.Context, tx *sql.Tx, allowance factory.AttemptAllowance, closeRoot bool, now time.Time) error {
	var err error
	repository, issue, root := allowance.Repository, allowance.Issue, allowance.RootAssignment
	settled, err := attemptRootRunsSettledTx(ctx, tx, repository, issue, root)
	if err != nil {
		return err
	}
	prior := allowance
	if closeRoot {
		allowance.Closed = true
	}
	if settled && allowance.Active {
		allowance, err = allowance.Checkpoint(now, false)
		if err != nil {
			return err
		}
	}
	if allowance.Active == prior.Active && allowance.Closed == prior.Closed && allowance.ActiveSeconds == prior.ActiveSeconds && allowance.CheckpointUnix == prior.CheckpointUnix {
		return nil
	}
	allowance.Revision++
	return saveAttemptAllowanceTx(ctx, tx, allowance)
}

func transitionAttemptAssignmentTx(ctx context.Context, tx *sql.Tx, assignmentID string, closeRoot bool, now time.Time) error {
	var data []byte
	if err := tx.QueryRowContext(ctx, `SELECT data FROM factory_assignments WHERE id=$1`, assignmentID).Scan(&data); err != nil {
		return err
	}
	var assignment factory.Assignment
	if err := json.Unmarshal(data, &assignment); err != nil {
		return err
	}
	if err := assignment.Validate(); err != nil {
		return err
	}
	allowance, err := lockAttemptAllowanceRootTx(ctx, tx, assignment.Repository, assignment.Issue, assignment.AttemptRoot)
	if errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if err != nil {
		return err
	}
	currentOwner, err := currentAttemptOwnerTx(ctx, tx, assignment.Repository, assignment.Issue, assignment.AttemptRoot)
	if errors.Is(err, sql.ErrNoRows) || err == nil && assignment.PublicationAssignment != currentOwner {
		return nil
	}
	if err != nil {
		return err
	}
	return transitionLockedAttemptRootTx(ctx, tx, allowance, closeRoot, now)
}

func currentAttemptOwnerTx(ctx context.Context, tx *sql.Tx, repository, issue int64, root string) (string, error) {
	var data []byte
	err := tx.QueryRowContext(ctx, `SELECT data FROM factory_assignments
		WHERE repository=$1 AND issue=$2 ORDER BY seq DESC LIMIT 1`, repository, issue).Scan(&data)
	if err != nil {
		return "", err
	}
	var latest factory.Assignment
	if err = json.Unmarshal(data, &latest); err != nil {
		return "", err
	}
	if err = latest.Validate(); err != nil {
		return "", err
	}
	if latest.AttemptRoot != root {
		return "", sql.ErrNoRows
	}
	return latest.PublicationAssignment, nil
}

func attemptPauseActiveTx(ctx context.Context, tx *sql.Tx, repository int64) (bool, error) {
	open, revision, raw, err := ensureDispatchGateTx(ctx, tx, repository)
	if err != nil {
		return false, err
	}
	var withdrawal factory.Withdrawal
	if err = json.Unmarshal(raw, &withdrawal); err != nil {
		return false, err
	}
	if open && revision == 0 && withdrawal.Repository == 0 {
		return false, nil
	}
	if err = withdrawal.Validate(); err != nil {
		return false, err
	}
	for _, cause := range withdrawal.ActiveCauses {
		if cause == factory.CauseControlPaused || cause == factory.CauseProjectStop {
			return true, nil
		}
	}
	return false, nil
}

func ensureAttemptDeadlineForRegistrationTx(ctx context.Context, tx *sql.Tx, assignmentID string, deadlines ...int64) error {
	if len(deadlines) == 0 {
		return factory.ErrAttemptTimeExhausted
	}
	var data []byte
	if err := tx.QueryRowContext(ctx, `SELECT data FROM factory_assignments WHERE id=$1`, assignmentID).Scan(&data); err != nil {
		return err
	}
	var assignment factory.Assignment
	if err := json.Unmarshal(data, &assignment); err != nil {
		return err
	}
	if err := assignment.Validate(); err != nil {
		return err
	}
	allowance, err := lockAttemptAllowanceRootTx(ctx, tx, assignment.Repository, assignment.Issue, assignment.AttemptRoot)
	if errors.Is(err, sql.ErrNoRows) {
		return factory.ErrAttemptTimeExhausted
	}
	if err != nil {
		return err
	}
	deadline, err := attemptDeadlineUnix(allowance, time.Now())
	if err != nil {
		return err
	}
	for _, registered := range deadlines {
		if registered <= time.Now().Unix() || registered > deadline {
			return factory.ErrAttemptTimeExhausted
		}
	}
	return nil
}

func deactivateReleasedAttemptTx(ctx context.Context, tx *sql.Tx, assignmentID string, now time.Time) error {
	var data []byte
	if err := tx.QueryRowContext(ctx, `SELECT data FROM factory_assignments WHERE id=$1`, assignmentID).Scan(&data); err != nil {
		return err
	}
	var assignment factory.Assignment
	if err := json.Unmarshal(data, &assignment); err != nil {
		return err
	}
	if err := assignment.Validate(); err != nil {
		return err
	}
	allowance, err := lockAttemptAllowanceRootTx(ctx, tx, assignment.Repository, assignment.Issue, assignment.AttemptRoot)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	var latestID string
	err = tx.QueryRowContext(ctx, `SELECT id FROM factory_assignments WHERE repository=$1 AND issue=$2 ORDER BY seq DESC LIMIT 1`, assignment.Repository, assignment.Issue).Scan(&latestID)
	if errors.Is(err, sql.ErrNoRows) || err == nil && latestID != assignment.ID {
		return nil
	}
	if err != nil {
		return err
	}
	owner, err := currentAttemptOwnerTx(ctx, tx, assignment.Repository, assignment.Issue, assignment.AttemptRoot)
	if errors.Is(err, sql.ErrNoRows) || err == nil && assignment.PublicationAssignment != owner {
		return nil
	}
	if err != nil {
		return err
	}
	var state string
	err = tx.QueryRowContext(ctx, `SELECT state FROM factory_reservations WHERE assignment=$1`, assignmentID).Scan(&state)
	if errors.Is(err, sql.ErrNoRows) || err == nil && state != factory.ReservationReleased {
		return nil
	}
	if err != nil {
		return err
	}
	return transitionLockedAttemptRootTx(ctx, tx, allowance, false, now)
}

func saveAttemptAllowanceTx(ctx context.Context, tx *sql.Tx, allowance factory.AttemptAllowance) error {
	if err := allowance.Validate(); err != nil {
		return err
	}
	data, err := json.Marshal(allowance)
	if err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `UPDATE factory_attempt_allowances SET revision=$1,data=$2 WHERE repository=$3 AND issue=$4 AND root_assignment=$5`,
		allowance.Revision, string(data), allowance.Repository, allowance.Issue, allowance.RootAssignment)
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

// FreezeAttemptAllowances closes active issue clocks only under the exact
// closed pause revision and after every repository run view is settled.
func (s *Store) FreezeAttemptAllowances(ctx context.Context, repository, expectedGateRevision int64, now time.Time) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	open, revision, raw, err := ensureDispatchGateTx(ctx, tx, repository)
	if err != nil {
		return err
	}
	if open || revision != expectedGateRevision {
		return ErrStaleRevision
	}
	var withdrawal factory.Withdrawal
	if err = json.Unmarshal(raw, &withdrawal); err != nil {
		return err
	}
	if err = withdrawal.Validate(); err != nil {
		return err
	}
	paused := false
	for _, cause := range withdrawal.ActiveCauses {
		if cause == factory.CauseControlPaused {
			paused = true
			break
		}
	}
	if !paused {
		return errors.New("attempt clocks can freeze only for an explicit repository pause")
	}
	rows, err := tx.QueryContext(ctx, `SELECT r.settled,r.data FROM factory_run_views v JOIN factory_runs r ON r.id=v.run WHERE v.repository=$1 ORDER BY v.seq FOR UPDATE OF r`, repository)
	if err != nil {
		return err
	}
	for rows.Next() {
		var settled bool
		var data []byte
		if err = rows.Scan(&settled, &data); err != nil {
			_ = rows.Close()
			return err
		}
		var run factory.Run
		if err = json.Unmarshal(data, &run); err != nil {
			_ = rows.Close()
			return err
		}
		if !settled || !run.Reconciled {
			_ = rows.Close()
			return ErrAttemptRunsPending
		}
	}
	if err = rows.Err(); err != nil {
		_ = rows.Close()
		return err
	}
	if err = rows.Close(); err != nil {
		return err
	}
	const pageLimit = 128
	var afterRoot string
	for {
		rows, err := tx.QueryContext(ctx, `SELECT issue,root_assignment,revision,data FROM factory_attempt_allowances WHERE repository=$1 AND root_assignment>$2 ORDER BY root_assignment LIMIT $3 FOR UPDATE`, repository, afterRoot, pageLimit)
		if err != nil {
			return err
		}
		locked := make([]factory.AttemptAllowance, 0, pageLimit)
		for rows.Next() {
			var issue, revision int64
			var root string
			var data []byte
			if err = rows.Scan(&issue, &root, &revision, &data); err != nil {
				_ = rows.Close()
				return err
			}
			var allowance factory.AttemptAllowance
			if err = json.Unmarshal(data, &allowance); err != nil {
				_ = rows.Close()
				return err
			}
			if err = validateAttemptAllowanceRow(allowance, repository, issue, root, revision); err != nil {
				_ = rows.Close()
				return err
			}
			locked = append(locked, allowance)
		}
		if err = rows.Err(); err != nil {
			_ = rows.Close()
			return err
		}
		if err = rows.Close(); err != nil {
			return err
		}
		if len(locked) == 0 {
			break
		}
		for _, allowance := range locked {
			afterRoot = allowance.RootAssignment
			if !allowance.Active {
				continue
			}
			next, err := allowance.Checkpoint(now, false)
			if err != nil {
				return err
			}
			next.Revision++
			if err = saveAttemptAllowanceTx(ctx, tx, next); err != nil {
				return err
			}
		}
	}
	return tx.Commit()
}

// admitAttemptAllowanceTx checkpoints active time and ensures the exact run
// deadline fits the remaining recorded allowance. limits is supplied only
// when the packet may create the issue's initial allowance.
func admitAttemptAllowanceTx(ctx context.Context, tx *sql.Tx, repository, issue int64, assignment string, limits *factory.AttemptLimits, deadline, now time.Time) error {
	if !deadline.After(now) {
		return factory.ErrAttemptTimeExhausted
	}
	allowance, err := lockAttemptAllowanceTx(ctx, tx, repository, issue)
	if errors.Is(err, sql.ErrNoRows) {
		if limits == nil {
			return factory.ErrAttemptTimeExhausted
		}
		return insertAttemptAllowanceTx(ctx, tx, repository, issue, assignment, *limits, deadline, now)
	}
	if err != nil {
		return err
	}
	if allowance.RootAssignment != assignment {
		return ErrAttemptRootMismatch
	}
	if allowance.Closed {
		return factory.ErrAttemptClosed
	}
	next, err := allowance.Checkpoint(now, true)
	if err != nil {
		return err
	}
	remaining := next.RemainingSeconds(now)
	if remaining == 0 || deadline.After(time.Unix(now.Unix()+remaining, 0)) {
		return factory.ErrAttemptTimeExhausted
	}
	if next.ActiveSeconds != allowance.ActiveSeconds || next.CheckpointUnix != allowance.CheckpointUnix || next.Active != allowance.Active {
		next.Revision++
		if err = saveAttemptAllowanceTx(ctx, tx, next); err != nil {
			return err
		}
	}
	return nil
}

// consumeCorrectionAttemptTx charges a correction child before its dispatch
// packet commits. The root row is locked and matched to the immutable child
// binding; replays of the same assignment are idempotent.
func consumeCorrectionAttemptTx(ctx context.Context, tx *sql.Tx, assignment factory.Assignment, now time.Time) error {
	allowance, err := lockAttemptAllowanceTx(ctx, tx, assignment.Repository, assignment.Issue)
	if err != nil {
		return err
	}
	if allowance.RootAssignment != assignment.AttemptRoot {
		return ErrAttemptRootMismatch
	}
	if allowance.Closed {
		return factory.ErrAttemptClosed
	}
	next, consumed, err := allowance.ConsumeCorrection(assignment.ID, now)
	if err != nil {
		return err
	}
	if consumed || next.ActiveSeconds != allowance.ActiveSeconds || next.CheckpointUnix != allowance.CheckpointUnix || next.Active != allowance.Active {
		next.Revision++
		return saveAttemptAllowanceTx(ctx, tx, next)
	}
	return nil
}

// startFreshAttemptAllowanceTx is called only after the explicit retry
// command and prior run accounting have been rechecked under the gate.
func startFreshAttemptAllowanceTx(ctx context.Context, tx *sql.Tx, repository, issue int64, assignment string, limits factory.AttemptLimits, deadline, now time.Time) error {
	prior, err := lockAttemptAllowanceTx(ctx, tx, repository, issue)
	if err != nil {
		return err
	}
	next, err := prior.Checkpoint(now, false)
	if err != nil {
		return err
	}
	next.Revision++
	next.Closed = true
	if err = saveAttemptAllowanceTx(ctx, tx, next); err != nil {
		return err
	}
	return insertAttemptAllowanceTx(ctx, tx, repository, issue, assignment, limits, deadline, now)
}

func insertAttemptAllowanceTx(ctx context.Context, tx *sql.Tx, repository, issue int64, assignment string, limits factory.AttemptLimits, deadline, now time.Time) error {
	allowance := factory.AttemptAllowance{
		Limits: limits, RootAssignment: assignment, Repository: repository,
		Issue: issue, CheckpointUnix: now.Unix(), Active: true,
	}
	if err := allowance.Validate(); err != nil {
		return err
	}
	remaining := allowance.RemainingSeconds(now)
	if !deadline.After(now) || remaining == 0 || deadline.After(time.Unix(now.Unix()+remaining, 0)) {
		return factory.ErrAttemptTimeExhausted
	}
	data, err := json.Marshal(allowance)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO factory_attempt_allowances(repository,issue,root_assignment,revision,data) VALUES($1,$2,$3,$4,$5)`,
		repository, issue, assignment, allowance.Revision, string(data))
	return err
}
