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
	ErrAttemptRunsPending  = errors.New("attempt runs or preparations are not confirmed settled")
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

// CloseExpiredAttempt closes automatic continuation for the current exact
// assignment root once its cumulative active-time allowance is exhausted.
// It is safe to replay: a closed root can still be deactivated after its last
// descendant run is confirmed settled.
func (s *Store) CloseExpiredAttempt(ctx context.Context, assignmentID string) (bool, error) {
	if !factory.ValidID(assignmentID) {
		return false, errors.New("invalid attempt assignment")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback() }()
	var raw []byte
	if err = tx.QueryRowContext(ctx, `SELECT data FROM factory_assignments WHERE id=$1`, assignmentID).Scan(&raw); err != nil {
		return false, err
	}
	var assignment factory.Assignment
	if err = json.Unmarshal(raw, &assignment); err != nil {
		return false, err
	}
	if err = assignment.Validate(); err != nil {
		return false, err
	}
	allowance, err := lockAttemptAllowanceRootTx(ctx, tx, assignment.Repository, assignment.Issue, assignment.AttemptRoot)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	// Serialize the current publication-owner check with dispatch/retry writes
	// on this root. Historical children of that owner share the same allowance.
	owner, err := currentAttemptOwnerTx(ctx, tx, assignment.Repository, assignment.Issue, assignment.AttemptRoot)
	if errors.Is(err, sql.ErrNoRows) || err == nil && assignment.PublicationAssignment != owner {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	now := time.Now()
	if !allowance.Closed {
		if !allowance.Active {
			return false, tx.Commit()
		}
		if now.Unix() < allowance.CheckpointUnix {
			return false, factory.ErrAllowanceClock
		}
		if allowance.RemainingSeconds(now) > 0 {
			return false, tx.Commit()
		}
	}
	if err = transitionLockedAttemptRootTx(ctx, tx, allowance, true, now); err != nil {
		return false, err
	}
	if err = tx.Commit(); err != nil {
		return false, err
	}
	return true, nil
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

// attemptRootRunsSettledTx confirms every recorded run view is reconciled and
// every admitted native candidate preparation is quiescent. Retained history
// remains in scope.
func attemptRootRunsSettledTx(ctx context.Context, tx *sql.Tx, repository, issue int64, root string) (bool, error) {
	var settled bool
	err := tx.QueryRowContext(ctx, `SELECT NOT EXISTS (
		SELECT 1 FROM factory_runs r
		JOIN factory_run_views v ON v.run=r.id
		JOIN factory_assignments a ON a.id=v.attempt
		WHERE (r.active OR NOT r.settled OR COALESCE(r.data->>'reconciled','false') <> 'true')
		  AND a.repository=$1 AND a.issue=$2 AND a.data->>'attempt_root'=$3
		LIMIT 1
	) AND NOT EXISTS (
		SELECT 1 FROM project_preparations p
		WHERE p.project_id=(SELECT data->>'project_id' FROM factory_assignments WHERE id=$3)
		  AND p.data ? 'factory_admission'
		  AND (p.data->'factory_admission'->>'repository')::bigint=$1
		  AND (p.data->'factory_admission'->>'issue')::bigint=$2
		  AND p.data->'factory_admission'->>'attempt_root'=$3
		  AND NOT (COALESCE((p.data#>>'{state,ready}')::boolean,FALSE)
		    OR (COALESCE((p.data#>>'{factory_admission,stopped}')::boolean,FALSE)
		      AND COALESCE(p.data#>>'{factory_admission,retirement}'='confirmed',FALSE)))
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
	var preparationsPending bool
	if err = tx.QueryRowContext(ctx, `SELECT EXISTS (
		SELECT 1 FROM project_preparations p
		WHERE (p.data->'factory_admission'->>'repository')::bigint=$1
		  AND p.data ? 'factory_admission'
		  AND NOT (COALESCE((p.data#>>'{state,ready}')::boolean,FALSE)
		    OR (COALESCE((p.data#>>'{factory_admission,stopped}')::boolean,FALSE)
		      AND COALESCE(p.data#>>'{factory_admission,retirement}'='confirmed',FALSE)))
		LIMIT 1)`, repository).Scan(&preparationsPending); err != nil {
		return err
	}
	if preparationsPending {
		return ErrAttemptRunsPending
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

// ChargeReviewDispatch charges one independent reviewer dispatch against
// the attempt-root review allowance before its child packet commits, in
// one row-locked transaction. The charge keys on the immutable reviewer
// assignment identity: replays of the same assignment are idempotent and
// never consume a second round or pass, even after exhaustion. The first
// inspection of a new head consumes one ReviewRound but never the
// unchanged-head extra pass; a discretionary re-review of an inspected
// head additionally spends that head's single extra pass; a
// policy-required review consumes one ReviewRound and stays separately
// caused (D08).
//
// It reports (consumed=true) only when this call spent budget. A replay
// of an already charged assignment reports (consumed=false,
// exhausted=false) so the caller still dispatches. An exhausted, closed
// or time-spent root reports (consumed=false, exhausted=true) without an
// error so the caller waits with review_allowance_exhausted instead of
// launching a run it cannot cover.
func (s *Store) ChargeReviewDispatch(ctx context.Context, repository, issue int64, root, reviewerAssignment, head string, policy bool) (consumed, exhausted bool, err error) {
	if repository <= 0 || issue <= 0 || !factory.ValidID(root) || !factory.ValidID(reviewerAssignment) || !factory.ValidCommit(head) {
		return false, false, errors.New("invalid review dispatch charge")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return false, false, err
	}
	defer func() { _ = tx.Rollback() }()
	allowance, err := lockAttemptAllowanceRootTx(ctx, tx, repository, issue, root)
	if err != nil {
		return false, false, err
	}
	if allowance.Closed {
		return false, true, nil
	}
	now := time.Now()
	next, spent, err := allowance.ConsumeReview(reviewerAssignment, head, policy, now)
	if err != nil {
		if errors.Is(err, factory.ErrReviewRoundsExhausted) || errors.Is(err, factory.ErrUnchangedHeadExtraExhausted) ||
			errors.Is(err, factory.ErrAttemptTimeExhausted) {
			return false, true, nil
		}
		return false, false, err
	}
	if !spent {
		return false, false, tx.Commit()
	}
	next.Revision++
	if err = saveAttemptAllowanceTx(ctx, tx, next); err != nil {
		return false, false, err
	}
	return true, false, tx.Commit()
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

// ReserveDeliveryJudgment charges one funded delivery judgment against the
// attempt-root call subledger plus the operator's global judgment ceiling
// in one transaction, before any provider request. Replaying the same
// fingerprint never resends: an existing journal row refuses without new
// charges. Reusing a consumed request identity for different evidence
// refuses as a conflict instead of silently sharing funding. Failure
// after reservation stays charged; only explicit authorized Retry starts
// a new root. A zero delivery limit authorizes no call, so automatic
// merge stays unreachable on an unfunded attempt.
func (s *Store) ReserveDeliveryJudgment(ctx context.Context, repository, issue int64, root, request, fingerprint string) (bool, error) {
	if repository <= 0 || issue <= 0 || !factory.ValidID(root) || !factory.ValidID(request) || !factory.ValidDigest(fingerprint) {
		return false, errors.New("invalid delivery reservation")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback() }()
	allowance, err := lockAttemptAllowanceRootTx(ctx, tx, repository, issue, root)
	if err != nil {
		return false, err
	}
	if allowance.Closed {
		return false, nil
	}
	now := time.Now()
	next, consumed, err := allowance.ReserveDeliveryCall(request, now)
	if err != nil {
		if errors.Is(err, factory.ErrDeliveryExhausted) || errors.Is(err, factory.ErrAttemptTimeExhausted) {
			return false, nil
		}
		return false, err
	}
	var existing int
	if err = tx.QueryRowContext(ctx, `SELECT count(*) FROM factory_priority_judgments WHERE snapshot=$1`, fingerprint).Scan(&existing); err != nil {
		return false, err
	}
	if existing > 0 {
		return false, nil
	}
	if !consumed {
		return false, ErrCommandConflict
	}
	var spent int
	if err = tx.QueryRowContext(ctx, `SELECT spent FROM factory_judgment_budget WHERE id=1 FOR UPDATE`).Scan(&spent); err != nil {
		return false, err
	}
	if spent >= s.PriorityCallBudget {
		return false, nil
	}
	next.Revision++
	if err = saveAttemptAllowanceTx(ctx, tx, next); err != nil {
		return false, err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO factory_priority_judgments(snapshot,max_calls) VALUES($1,1)`, fingerprint); err != nil {
		return false, err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE factory_judgment_budget SET spent=spent+1 WHERE id=1`); err != nil {
		return false, err
	}
	return true, tx.Commit()
}

// RecordDeliveryJudgment persists one validated delivery receipt over its
// reserved fingerprint. Recording is append-only: a missing reservation
// or an already recorded receipt conflicts instead of duplicating spend
// or rewriting history.
func (s *Store) RecordDeliveryJudgment(ctx context.Context, r factory.DeliveryReceipt) error {
	if err := r.Validate(); err != nil {
		return err
	}
	raw, err := json.Marshal(r)
	if err != nil {
		return err
	}
	result, err := s.db.ExecContext(ctx, `UPDATE factory_priority_judgments SET data=$2 WHERE snapshot=$1 AND data IS NULL`, r.Fingerprint, string(raw))
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
	return ErrCommandConflict
}

// DeliveryJudgment reads one recorded delivery receipt by its evidence
// fingerprint. Unreserved fingerprints and non-delivery journal rows
// report ErrNotFound; a present but invalid receipt is a storage fault.
func (s *Store) DeliveryJudgment(ctx context.Context, fingerprint string) (factory.DeliveryReceipt, error) {
	var r factory.DeliveryReceipt
	if !factory.ValidDigest(fingerprint) {
		return r, errors.New("invalid delivery fingerprint")
	}
	var raw []byte
	err := s.db.QueryRowContext(ctx, `SELECT data FROM factory_priority_judgments WHERE snapshot=$1 AND data IS NOT NULL`, fingerprint).Scan(&raw)
	if err != nil {
		return r, err
	}
	if err = json.Unmarshal(raw, &r); err != nil {
		return r, err
	}
	if r.Kind != factory.DeliveryReceiptKind || r.Fingerprint != fingerprint {
		return factory.DeliveryReceipt{}, ErrNotFound
	}
	if err = r.Validate(); err != nil {
		return factory.DeliveryReceipt{}, err
	}
	return r, nil
}
