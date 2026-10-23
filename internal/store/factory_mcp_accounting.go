package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
)

var ErrMCPAccountingTimeUnavailable = errors.New("confirmed retired run accounting time unavailable")

// FinishedMCPAccountingRuns visits only held accounting behind a retired run
// whose immutable MCP terminal already finished its exact assignment.
func (s *Store) FinishedMCPAccountingRuns(ctx context.Context, limit int) ([]factory.Run, error) {
	if limit < 1 || limit > MaxHeldReservations {
		return nil, errors.New("invalid finished accounting bound")
	}
	rows, err := s.db.QueryContext(ctx, `SELECT r.data FROM factory_runs r
JOIN factory_run_views v ON v.run=r.id JOIN factory_assignments a ON a.id=v.attempt
JOIN factory_reservations p ON p.assignment=a.id
WHERE r.settled AND NOT r.active AND r.data->>'reconciled'='true'
AND a.stage='finished' AND a.run=r.id AND p.state='held'
AND EXISTS(SELECT 1 FROM factory_mcp_submissions m WHERE m.assignment=a.id
AND starts_with(m.tool,'factory.finish_') AND m.state IN ('recorded','accepted'))
ORDER BY r.seq LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var runs []factory.Run
	for rows.Next() {
		var raw []byte
		var run factory.Run
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(raw, &run); err != nil {
			return nil, err
		}
		runs = append(runs, run)
	}
	return runs, rows.Err()
}

// AccountFinishedMCPRun records usage and consumes its reservation in one
// transaction. It never writes the finished assignment or its MCP result.
func (s *Store) AccountFinishedMCPRun(ctx context.Context, id string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var raw []byte
	var ended sql.NullString
	var settled, active bool
	if err := tx.QueryRowContext(ctx, `SELECT data,settled,active,data->>'settled_at' FROM factory_runs WHERE id=$1 FOR UPDATE`, id).Scan(&raw, &settled, &active, &ended); err != nil {
		return err
	}
	var run factory.Run
	if json.Unmarshal(raw, &run) != nil || run.Validate() != nil || run.ID != id || !settled || active || !run.Reconciled {
		return errors.New("run accounting requires its confirmed retired record")
	}
	if err := tx.QueryRowContext(ctx, `SELECT a.data FROM factory_assignments a JOIN factory_run_views v ON v.attempt=a.id WHERE v.run=$1`, id).Scan(&raw); err != nil {
		return err
	}
	var a factory.Assignment
	if json.Unmarshal(raw, &a) != nil || a.Validate() != nil || a.Stage != factory.AssignmentFinished || a.Run != id || a.ProjectID != run.ProjectID || a.Role != run.Role || a.SourceCommit != run.InputSHA {
		return errors.New("run accounting differs from its finished assignment")
	}
	if err := tx.QueryRowContext(ctx, `SELECT data FROM factory_reservations WHERE assignment=$1 FOR UPDATE`, a.ID).Scan(&raw); err != nil {
		return err
	}
	var reservation factory.Reservation
	if json.Unmarshal(raw, &reservation) != nil || reservation.Validate() != nil || reservation.AssignmentID != a.ID || reservation.Repository != a.Repository || reservation.Connection != a.Connection || (reservation.State != factory.ReservationHeld && reservation.State != factory.ReservationConsumed) {
		return errors.New("run accounting reservation differs from its assignment")
	}
	var usage factory.Usage
	err = tx.QueryRowContext(ctx, `SELECT data FROM factory_usage WHERE run=$1`, id).Scan(&raw)
	if err == nil {
		if json.Unmarshal(raw, &usage) != nil || usage.Validate() != nil || usage.RunID != id || usage.Repository != a.Repository || usage.Connection != a.Connection || !usage.StartedAt.Equal(run.Started) {
			return errors.New("retained usage differs from its run")
		}
	} else if errors.Is(err, sql.ErrNoRows) {
		var end time.Time
		if ended.Valid {
			end, err = time.Parse(time.RFC3339Nano, ended.String)
		} else {
			end, err = legacyFinishedRunEnd(ctx, tx, a, run)
		}
		if err != nil || end.Before(run.Started) {
			return ErrMCPAccountingTimeUnavailable
		}
		elapsed := end.Sub(run.Started)
		usage = factory.Usage{RunID: id, Repository: a.Repository, Connection: a.Connection, Minutes: int((elapsed + time.Minute - time.Nanosecond) / time.Minute), StartedAt: run.Started.UTC(), EndedAt: end.UTC()}
		if err := usage.Validate(); err != nil {
			return err
		}
		body, _ := json.Marshal(usage)
		if _, err := tx.ExecContext(ctx, `INSERT INTO factory_usage(run,repository,connection,minutes,data,started_at,ended_at) VALUES($1,$2,$3,$4,$5,$6,$7) ON CONFLICT(run) DO NOTHING`, id, a.Repository, a.Connection, usage.Minutes, string(body), floorUsageMicro(usage.StartedAt), ceilUsageMicro(usage.EndedAt)); err != nil {
			return err
		}
	} else {
		return err
	}
	if reservation.State == factory.ReservationHeld {
		if _, err := tx.ExecContext(ctx, `UPDATE factory_reservations SET state='consumed',data=jsonb_set(jsonb_set(data,'{state}','"consumed"'::jsonb),'{revision}',to_jsonb((data->>'revision')::bigint+1)) WHERE assignment=$1 AND state='held'`, a.ID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// Retained runs predating settled_at may reuse only an already closed,
// inactive singleton root's recorded retirement clock. A broader history
// cannot identify this run's end and remains held for known evidence.
func legacyFinishedRunEnd(ctx context.Context, tx *sql.Tx, a factory.Assignment, run factory.Run) (time.Time, error) {
	if a.AttemptRoot != a.ID || len(a.RunHistory) != 1 || a.RunHistory[0] != run.ID {
		return time.Time{}, errors.New("run retirement clock is not uniquely bound")
	}
	var raw []byte
	var count int
	if err := tx.QueryRowContext(ctx, `SELECT data,(SELECT count(*) FROM factory_run_views v JOIN factory_assignments x ON x.id=v.attempt WHERE x.data->>'attempt_root'=$1) FROM factory_attempt_allowances WHERE root_assignment=$1 AND repository=$2 AND issue=$3`, a.ID, a.Repository, a.Issue).Scan(&raw, &count); err != nil {
		return time.Time{}, err
	}
	var allowance factory.AttemptAllowance
	if json.Unmarshal(raw, &allowance) != nil || allowance.Validate() != nil || allowance.RootAssignment != a.ID || allowance.Active || !allowance.Closed || count != 1 || allowance.CheckpointUnix <= 0 {
		return time.Time{}, errors.New("run retirement clock unavailable")
	}
	return time.Unix(allowance.CheckpointUnix, 0).UTC(), nil
}
