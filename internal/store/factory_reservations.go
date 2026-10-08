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

// MaxHeldReservations bounds one held-reservation listing for dispatch
// occupancy and crash recovery.
const MaxHeldReservations = 1024

// Reservation returns one dispatch reservation by its assignment identity.
func (s *Store) Reservation(ctx context.Context, assignmentID string) (factory.Reservation, error) {
	var r factory.Reservation
	var data []byte
	err := s.db.QueryRowContext(ctx, `SELECT data FROM factory_reservations WHERE assignment=$1`, assignmentID).Scan(&data)
	if err == nil {
		err = json.Unmarshal(data, &r)
	}
	return r, err
}

// HeldReservations lists every held dispatch reservation, oldest first.
func (s *Store) HeldReservations(ctx context.Context, limit int) ([]factory.Reservation, error) {
	if limit <= 0 || limit > MaxHeldReservations {
		return nil, errors.New("invalid held reservation listing limit")
	}
	rows, err := s.db.QueryContext(ctx, `SELECT data FROM factory_reservations WHERE state='held' ORDER BY seq LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []factory.Reservation
	for rows.Next() {
		var data []byte
		var r factory.Reservation
		if err = rows.Scan(&data); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(data, &r); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// transitionReservation moves one held reservation to its terminal state.
// Only a held reservation transitions; anything else reports not found so
// a concurrent settle cannot consume capacity twice.
func (s *Store) transitionReservation(ctx context.Context, assignmentID, state string) error {
	if state != factory.ReservationConsumed && state != factory.ReservationReleased {
		return errors.New("invalid reservation transition")
	}
	result, err := s.db.ExecContext(ctx, `UPDATE factory_reservations
		SET state=$1, data=jsonb_set(jsonb_set(data,'{state}',to_jsonb($2::text)),'{revision}',to_jsonb((data->>'revision')::bigint+1))
		WHERE assignment=$3 AND state='held'`, state, state, assignmentID)
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

// ConsumeReservation marks one held reservation consumed by its recorded
// run. Consuming is the only transition a settled run takes; usage is
// recorded before this call so a crash replays through the held state.
func (s *Store) ConsumeReservation(ctx context.Context, assignmentID string) error {
	return s.transitionReservation(ctx, assignmentID, factory.ReservationConsumed)
}

// ReleaseReservation marks one held reservation released: its capacity was
// confirmed unused and counts against no limit anymore.
func (s *Store) ReleaseReservation(ctx context.Context, assignmentID string) error {
	return s.transitionReservation(ctx, assignmentID, factory.ReservationReleased)
}

// ReholdReservation returns one released reservation to held for the next
// attempt of the same assignment, carrying the new plan. Only a released
// reservation re-holds; held and consumed capacity never double counts.
func (s *Store) ReholdReservation(ctx context.Context, r factory.Reservation) error {
	if err := r.Validate(); err != nil {
		return err
	}
	if r.State != factory.ReservationHeld {
		return errors.New("only a held plan re-holds a reservation")
	}
	data, err := json.Marshal(r)
	if err != nil {
		return err
	}
	result, err := s.db.ExecContext(ctx, `UPDATE factory_reservations SET state='held',data=$1 WHERE assignment=$2 AND state='released'`,
		string(data), r.AssignmentID)
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

// RecordRunUsage appends one settled run's confirmed consumption. The first
// write wins: a replayed settle replays the recorded minutes instead of
// charging twice.
func (s *Store) RecordRunUsage(ctx context.Context, u factory.Usage) error {
	if err := u.Validate(); err != nil {
		return err
	}
	data, err := json.Marshal(u)
	if err != nil {
		return err
	}
	startedAt := floorUsageMicro(u.StartedAt.UTC())
	endedAt := ceilUsageMicro(u.EndedAt.UTC())
	_, err = s.db.ExecContext(ctx, `INSERT INTO factory_usage(run,repository,connection,minutes,data,started_at,ended_at) VALUES($1,$2,$3,$4,$5,$6,$7)
		ON CONFLICT(run) DO NOTHING`, u.RunID, u.Repository, u.Connection, u.Minutes, string(data), startedAt, endedAt)
	return err
}

// RunUsage returns one settled run's confirmed consumption.
func (s *Store) RunUsage(ctx context.Context, runID string) (factory.Usage, error) {
	var u factory.Usage
	var data []byte
	err := s.db.QueryRowContext(ctx, `SELECT data FROM factory_usage WHERE run=$1`, runID).Scan(&data)
	if err == nil {
		err = json.Unmarshal(data, &u)
	}
	return u, err
}

const usageMicrosPerMinute int64 = factory.UsageMicrosPerMinute

// ConnectionUsageCharge returns settled rolling-window usage in microseconds
// plus the conservative charge for every held reservation on this connection.
func (s *Store) ConnectionUsageCharge(ctx context.Context, connection string, now time.Time) (int64, error) {
	return connectionUsageCharge(ctx, s.db, connection, now)
}

func connectionUsageCharge(ctx context.Context, q grantCommandSQL, connection string, now time.Time) (int64, error) {
	windowEnd := ceilUsageMicro(now.UTC())
	windowStart := floorUsageMicro(now.UTC().Add(-24 * time.Hour))
	actualNow := now.UTC()
	actualWindowStart := actualNow.Add(-24 * time.Hour)
	var settled int64
	err := q.QueryRowContext(ctx, `SELECT LEAST(COALESCE(SUM(GREATEST(0,
		FLOOR(EXTRACT(EPOCH FROM (LEAST(ended_at,$3)-GREATEST(started_at,$2)))*1000000))),0),$4)::bigint
		FROM factory_usage WHERE connection=$1 AND ended_at>$2 AND started_at<$3`,
		connection, windowStart, windowEnd, int64(math.MaxInt64)).Scan(&settled)
	if err != nil {
		return 0, err
	}
	charge := settled
	var heldDuration time.Duration
	rows, err := q.QueryContext(ctx, `SELECT (r.data->>'planned_minutes')::integer, f.data->>'started'
		FROM factory_reservations r
		LEFT JOIN factory_assignments a ON a.id=r.assignment
		LEFT JOIN factory_runs f ON f.id=a.run
		WHERE r.connection=$1 AND r.state='held' ORDER BY r.seq LIMIT $2`, connection, MaxHeldReservations+1)
	if err != nil {
		return 0, err
	}
	count := 0
	for rows.Next() {
		count++
		if count > MaxHeldReservations {
			_ = rows.Close()
			return 0, errors.New("held connection usage exceeds accounting bound")
		}
		var planned int
		var rawStarted sql.NullString
		if err = rows.Scan(&planned, &rawStarted); err != nil {
			_ = rows.Close()
			return 0, err
		}
		if planned < 1 || planned > 180 || !rawStarted.Valid || rawStarted.String == "" {
			_ = rows.Close()
			return 0, errors.New("held connection usage has no valid run start")
		}
		started, parseErr := time.Parse(time.RFC3339Nano, rawStarted.String)
		if parseErr != nil {
			_ = rows.Close()
			return 0, errors.New("held connection usage has an invalid run start")
		}
		started = started.UTC()
		elapsed := actualNow.Sub(started)
		if elapsed < 0 {
			elapsed = 0
		}
		openStart := started
		if openStart.Before(actualWindowStart) {
			openStart = actualWindowStart
		}
		openDuration := actualNow.Sub(openStart)
		if openDuration < 0 {
			openDuration = 0
		}
		remaining := time.Duration(planned)*time.Minute - elapsed
		if remaining < 0 {
			remaining = 0
		}
		heldDuration += openDuration + remaining
	}
	if err = rows.Err(); err != nil {
		_ = rows.Close()
		return 0, err
	}
	if err = rows.Close(); err != nil {
		return 0, err
	}
	return saturatingUsageAdd(charge, usageDurationMicrosCeil(heldDuration)), nil
}

func floorUsageMicro(at time.Time) time.Time {
	return at.Truncate(time.Microsecond)
}

func ceilUsageMicro(at time.Time) time.Time {
	floor := floorUsageMicro(at)
	if floor.Equal(at) {
		return floor
	}
	return floor.Add(time.Microsecond)
}

func usageDurationMicrosCeil(duration time.Duration) int64 {
	if duration <= 0 {
		return 0
	}
	micros := int64(duration / time.Microsecond)
	if duration%time.Microsecond != 0 {
		micros++
	}
	return micros
}

func saturatingUsageAdd(left, right int64) int64 {
	if right > math.MaxInt64-left {
		return math.MaxInt64
	}
	return left + right
}

// UsageTotal sums confirmed consumption for one repository connection.
func (s *Store) UsageTotal(ctx context.Context, repository int64, connection string) (int, error) {
	var total int
	err := s.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(minutes),0) FROM factory_usage WHERE repository=$1 AND connection=$2`,
		repository, connection).Scan(&total)
	return total, err
}
