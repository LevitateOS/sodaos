package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/levitateos/sodaos/internal/factory"
)

// ErrAssignmentActive reports a dispatch packet for an issue that already
// carries an unfinished assignment. One issue runs at most one assignment
// at a time; a concurrent dispatcher won this issue.
var ErrAssignmentActive = errors.New("issue already carries an unfinished assignment")

// Admission refusals from the atomic dispatch gate. The packet records
// first and the limits are rechecked inside the same transaction, so a
// refusal means a concurrent admission consumed the room this attempt
// planned against; the dispatcher waits instead of launching.
var (
	ErrCapacityFull       = errors.New("appliance runs at its limit")
	ErrRepositoryFull     = errors.New("repository runs at its limit")
	ErrSponsorshipFull    = errors.New("sponsorship runs at its limit")
	ErrAllowanceExhausted = errors.New("sponsorship allowance is exhausted")
	ErrAdmissionChanged   = errors.New("admission grants changed during dispatch")
)

// RecordDispatchPacket stores one dispatch atomically: the gate
// registration, the assigned assignment, its held reservation, the
// recorded run and the display binding. Either the whole packet lands or
// nothing does, so a crash never leaves a reservation without its
// assignment or a run without its dispatch. A second unfinished
// assignment for the issue refuses with ErrAssignmentActive instead of
// dispatching twice.
//
// The packet is also the atomic admission gate: the limit rows are
// locked, the packet records, and then appliance, repository,
// sponsorship and allowance limits are rechecked inside the same
// transaction. Concurrent passes serialize here, so only room that
// actually exists is admitted; the losers wait instead of launching.
func (s *Store) RecordDispatchPacket(ctx context.Context, d factory.DispatchRegistration, a factory.Assignment, r factory.Reservation, run factory.Run, view factory.RunView) error {
	if err := d.Validate(); err != nil {
		return err
	}
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
	if d.ID != a.ID || d.Repository != a.Repository {
		return errors.New("dispatch packet registration does not match its assignment")
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
	tx, err := s.begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if err = registerDispatchTx(ctx, tx, d); err != nil {
		return err
	}
	if _, err = tx.exec(ctx, `INSERT INTO factory_assignments(id,repository,issue,run,stage,revision,data) VALUES(?,?,?,?,?,?,?)`,
		a.ID, a.Repository, a.Issue, a.Run, a.Stage, a.Revision, string(adata)); err != nil {
		return dispatchPacketError(err)
	}
	if _, err = tx.exec(ctx, `INSERT INTO factory_reservations(assignment,repository,connection,state,data) VALUES(?,?,?,?,?)`,
		r.AssignmentID, r.Repository, r.Connection, r.State, string(rdata)); err != nil {
		return fmt.Errorf("dispatch packet failed: %w", err)
	}
	if _, err = tx.exec(ctx, `INSERT INTO factory_runs(id,active,settled,data) VALUES(?,TRUE,FALSE,?)`, run.ID, string(rundata)); err != nil {
		return fmt.Errorf("dispatch packet failed: %w", err)
	}
	if _, err = tx.exec(ctx, `INSERT INTO factory_run_views(run,repository,issue,attempt) VALUES(?,?,?,?)`,
		view.RunID, view.Repository, view.Issue, view.Attempt); err != nil {
		return fmt.Errorf("dispatch packet failed: %w", err)
	}
	if err = checkAdmissionTx(ctx, tx, a.Repository, a.Connection, a.ProjectID); err != nil {
		return err
	}
	return tx.Commit()
}

// checkAdmissionTx enforces appliance, repository, sponsorship and
// allowance limits inside the admission transaction. The limit rows are
// locked first so concurrent admissions serialize here; the packet's own
// held reservation is already recorded, so every comparison accounts for
// it and refuses exactly when the pre-packet state plus this admission
// would exceed a limit. Count reads are capped just past each limit,
// which decides exact admission without scanning settled history.
func checkAdmissionTx(ctx context.Context, t *tx, repository int64, connection, projectID string) error {
	var cdata, pdata, gdata, sdata []byte
	if err := t.queryRow(ctx, `SELECT data FROM factory_capacity WHERE id=1 FOR UPDATE`).Scan(&cdata); err != nil {
		return admissionChanged(err)
	}
	var capacity factory.Capacity
	if err := json.Unmarshal(cdata, &capacity); err != nil {
		return err
	}
	if err := t.queryRow(ctx, `SELECT data FROM factory_policies WHERE repository=? FOR UPDATE`, repository).Scan(&pdata); err != nil {
		return admissionChanged(err)
	}
	var policy factory.RepositoryPolicy
	if err := json.Unmarshal(pdata, &policy); err != nil {
		return err
	}
	if err := t.queryRow(ctx, `SELECT data FROM factory_operator_grants WHERE repository=? FOR UPDATE`, repository).Scan(&gdata); err != nil {
		return admissionChanged(err)
	}
	var grant factory.OperatorGrant
	if err := json.Unmarshal(gdata, &grant); err != nil {
		return err
	}
	if err := t.queryRow(ctx, `SELECT data FROM factory_sponsorships WHERE repository=? AND connection=? FOR UPDATE`, repository, connection).Scan(&sdata); err != nil {
		return admissionChanged(err)
	}
	var sponsorship factory.Sponsorship
	if err := json.Unmarshal(sdata, &sponsorship); err != nil {
		return err
	}
	heldTotal, err := cappedCountTx(ctx, t, `SELECT 1 FROM factory_reservations WHERE state='held'`, capacity.MaxConcurrentRuns+1)
	if err != nil {
		return err
	}
	unattributed, err := cappedCountTx(ctx, t, `SELECT 1 FROM factory_runs WHERE active AND id NOT IN (SELECT run FROM factory_assignments WHERE run!='')`, capacity.MaxConcurrentRuns+1)
	if err != nil {
		return err
	}
	if heldTotal+unattributed > capacity.MaxConcurrentRuns {
		return ErrCapacityFull
	}
	repoLimit := policy.MaxConcurrent
	if grant.MaxConcurrent < repoLimit {
		repoLimit = grant.MaxConcurrent
	}
	heldRepo, err := cappedCountTx(ctx, t, `SELECT 1 FROM factory_reservations WHERE state='held' AND repository=?`, repoLimit+1, repository)
	if err != nil {
		return err
	}
	unattributedProject, err := cappedCountTx(ctx, t, `SELECT 1 FROM factory_runs WHERE active AND id NOT IN (SELECT run FROM factory_assignments WHERE run!='') AND data->>'project_id'=?`, repoLimit+1, projectID)
	if err != nil {
		return err
	}
	if heldRepo+unattributedProject > repoLimit {
		return ErrRepositoryFull
	}
	slots, err := cappedCountTx(ctx, t, `SELECT 1 FROM factory_reservations WHERE state='held' AND repository=? AND connection=?`, sponsorship.MaxConcurrent+1, repository, connection)
	if err != nil {
		return err
	}
	if slots > sponsorship.MaxConcurrent {
		return ErrSponsorshipFull
	}
	var planned int
	err = t.queryRow(ctx, `SELECT coalesce(sum((data->>'planned_minutes')::bigint),0)::bigint FROM factory_reservations WHERE state='held' AND repository=? AND connection=?`,
		repository, connection).Scan(&planned)
	if err != nil {
		return err
	}
	var used int
	err = t.queryRow(ctx, `SELECT COALESCE(SUM(minutes),0) FROM factory_usage WHERE repository=? AND connection=?`,
		repository, connection).Scan(&used)
	if err != nil {
		return err
	}
	if sponsorship.AllowanceMinutes-used-planned < 0 {
		return ErrAllowanceExhausted
	}
	return nil
}

func admissionChanged(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrNotFound) {
		return ErrAdmissionChanged
	}
	return err
}

// cappedCountTx counts a query's rows up to and including limit: it
// returns the exact count below the limit and the limit itself above
// it. Admission compares against limits far below any table size, so
// the capped count decides exact admission with bounded work.
func cappedCountTx(ctx context.Context, t *tx, query string, limit int, args ...any) (int, error) {
	var n int
	params := append(append([]any{}, args...), limit)
	err := t.queryRow(ctx, `SELECT count(*) FROM (`+query+` LIMIT ?) t`, params...).Scan(&n)
	return n, err
}

func dispatchPacketError(err error) error {
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return ErrAssignmentActive
	}
	return fmt.Errorf("dispatch packet failed: %w", err)
}
