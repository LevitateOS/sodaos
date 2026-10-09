package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
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
