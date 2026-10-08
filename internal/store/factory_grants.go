package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/levitateos/sodaos/internal/factory"
)

// ErrStaleRevision reports a CAS write against a superseded grant, dispatch
// or decision revision. Callers refresh and retry with a new command.
var ErrStaleRevision = errors.New("stale record revision")

// ErrDispatchClosed reports a registration against withdrawn dispatch. The
// caller waits for an authorized resume instead of dispatching around it.
var ErrDispatchClosed = errors.New("factory dispatch is closed")

// ErrDispatchConflict reports a reused dispatch identity with changed content.
var ErrDispatchConflict = errors.New("dispatch identity reused for different content")

// saveRevisionedGrant CAS-saves one revisioned JSON grant row. Withdrawn
// grants stay as inactive rows, never deleted ones.
func (s *Store) saveRevisionedGrant(ctx context.Context, table, key string, id any, revision int64, next any, what string) error {
	nextData, err := json.Marshal(next)
	if err != nil {
		return err
	}
	result, err := s.db.ExecContext(ctx, `INSERT INTO `+table+`(`+key+`,revision,data) VALUES($1,$2,$3)
		ON CONFLICT(`+key+`) DO UPDATE SET revision=$4,data=$5 WHERE `+table+`.revision=$6`,
		id, revision+1, string(nextData), revision+1, string(nextData), revision)
	if err != nil {
		return fmt.Errorf("%s save failed: %w", what, err)
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

func (s *Store) loadGrant(ctx context.Context, table, key string, id any, out any) error {
	var data []byte
	err := s.db.QueryRowContext(ctx, `SELECT data FROM `+table+` WHERE `+key+`=$1`, id).Scan(&data)
	if err == nil {
		err = json.Unmarshal(data, out)
	}
	return err
}

// SaveRepositoryPolicy records the owner's policy under CAS revision.
func (s *Store) SaveRepositoryPolicy(ctx context.Context, p factory.RepositoryPolicy) error {
	if err := p.Validate(); err != nil {
		return err
	}
	next := p
	next.Revision++
	return s.saveRevisionedGrant(ctx, "factory_policies", "repository", p.Repository, p.Revision, next, "repository policy")
}

// RepositoryPolicy returns the standing policy for one repository.
func (s *Store) RepositoryPolicy(ctx context.Context, repository int64) (factory.RepositoryPolicy, error) {
	var p factory.RepositoryPolicy
	return p, s.loadGrant(ctx, "factory_policies", "repository", repository, &p)
}

// SaveCapacity records the appliance capacity under CAS revision.
func (s *Store) SaveCapacity(ctx context.Context, c factory.Capacity) error {
	if err := c.Validate(); err != nil {
		return err
	}
	next := c
	next.Revision++
	return s.saveRevisionedGrant(ctx, "factory_capacity", "id", 1, c.Revision, next, "appliance capacity")
}

// Capacity returns the appliance capacity record.
func (s *Store) Capacity(ctx context.Context) (factory.Capacity, error) {
	var c factory.Capacity
	return c, s.loadGrant(ctx, "factory_capacity", "id", 1, &c)
}

// SaveOperatorGrant records the operator's repository permission under CAS.
func (s *Store) SaveOperatorGrant(ctx context.Context, g factory.OperatorGrant) error {
	if err := g.Validate(); err != nil {
		return err
	}
	next := g
	next.Revision++
	return s.saveRevisionedGrant(ctx, "factory_operator_grants", "repository", g.Repository, g.Revision, next, "operator grant")
}

// OperatorGrant returns the operator permission for one repository.
func (s *Store) OperatorGrant(ctx context.Context, repository int64) (factory.OperatorGrant, error) {
	var g factory.OperatorGrant
	return g, s.loadGrant(ctx, "factory_operator_grants", "repository", repository, &g)
}

// SaveSponsorship records one connection sponsorship under CAS revision.
func (s *Store) SaveSponsorship(ctx context.Context, sp factory.Sponsorship) error {
	if err := sp.Validate(); err != nil {
		return err
	}
	next := sp
	next.Revision++
	nextData, err := json.Marshal(next)
	if err != nil {
		return err
	}
	result, err := s.db.ExecContext(ctx, `INSERT INTO factory_sponsorships(repository,connection,revision,data) VALUES($1,$2,$3,$4)
		ON CONFLICT(repository,connection) DO UPDATE SET revision=$5,data=$6 WHERE factory_sponsorships.revision=$7`,
		sp.Repository, sp.Connection, next.Revision, string(nextData), next.Revision, string(nextData), sp.Revision)
	if err != nil {
		return fmt.Errorf("sponsorship save failed: %w", err)
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

// Sponsorship returns one connection sponsorship for a repository.
func (s *Store) Sponsorship(ctx context.Context, repository int64, connection string) (factory.Sponsorship, error) {
	var sp factory.Sponsorship
	var data []byte
	err := s.db.QueryRowContext(ctx, `SELECT data FROM factory_sponsorships WHERE repository=$1 AND connection=$2`, repository, connection).Scan(&data)
	if err == nil {
		err = json.Unmarshal(data, &sp)
	}
	return sp, err
}

// Sponsorships lists the recorded connection sponsorships for a repository.
func (s *Store) Sponsorships(ctx context.Context, repository int64) ([]factory.Sponsorship, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT data FROM factory_sponsorships WHERE repository=$1 ORDER BY connection LIMIT 33`, repository)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []factory.Sponsorship{}
	for rows.Next() {
		var data []byte
		var sp factory.Sponsorship
		if err = rows.Scan(&data); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(data, &sp); err != nil {
			return nil, err
		}
		out = append(out, sp)
	}
	return out, rows.Err()
}

// DispatchState returns the dispatch gate for one repository. A missing row
// means dispatch was never withdrawn: open at revision zero.
func (s *Store) DispatchState(ctx context.Context, repository int64) (bool, int64, factory.Withdrawal, error) {
	var open bool
	var revision int64
	var data []byte
	var withdrawal factory.Withdrawal
	err := s.db.QueryRowContext(ctx, `SELECT open,revision,data FROM factory_dispatch WHERE repository=$1`, repository).Scan(&open, &revision, &data)
	if errors.Is(err, ErrNotFound) {
		return true, 0, factory.Withdrawal{}, nil
	}
	if err == nil {
		err = json.Unmarshal(data, &withdrawal)
	}
	if err == nil && !open {
		if err = withdrawal.Validate(); err == nil && len(withdrawal.ActiveCauses) == 0 {
			err = errors.New("closed dispatch has no active cause")
		}
	} else if err == nil && len(withdrawal.ActiveCauses) != 0 {
		err = errors.New("open dispatch has active closure causes")
	}
	return open, revision, withdrawal, err
}

// RegisterDispatch records one outstanding dispatch under the open gate. A
// delayed registration after withdrawal is refused; it cannot escape the
// captured set. The same ID with identical content replays.
func (s *Store) RegisterDispatch(ctx context.Context, d factory.DispatchRegistration) error {
	if err := d.Validate(); err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if err = registerDispatchTx(ctx, tx, d); err != nil {
		return err
	}
	return tx.Commit()
}

// registerDispatchTx records one outstanding dispatch under the open gate
// inside the caller's transaction. The dispatch packet shares it so a
// refused packet leaves no orphan registration behind.
func registerDispatchTx(ctx context.Context, t *sql.Tx, d factory.DispatchRegistration) error {
	if _, err := t.ExecContext(ctx, `INSERT INTO factory_dispatch(repository,revision,open,data)
		VALUES($1,0,TRUE,'{}'::jsonb) ON CONFLICT(repository) DO NOTHING`, d.Repository); err != nil {
		return err
	}
	var open bool
	err := t.QueryRowContext(ctx, `SELECT open FROM factory_dispatch WHERE repository=$1 FOR UPDATE`, d.Repository).Scan(&open)
	if err != nil {
		if !errors.Is(err, ErrNotFound) {
			return err
		}
		// Never withdrawn: open at revision zero.
		open = true
	}
	if !open {
		return ErrDispatchClosed
	}
	data, err := json.Marshal(d)
	if err != nil {
		return err
	}
	result, err := t.ExecContext(ctx, `INSERT INTO factory_dispatch_regs(id,repository,revision,data) VALUES($1,$2,$3,$4) ON CONFLICT(id) DO NOTHING`,
		d.ID, d.Repository, d.Revision, string(data))
	if err != nil {
		return fmt.Errorf("dispatch registration failed: %w", err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 1 {
		return nil
	}
	var existing factory.DispatchRegistration
	var raw []byte
	if err = t.QueryRowContext(ctx, `SELECT data FROM factory_dispatch_regs WHERE id=$1`, d.ID).Scan(&raw); err != nil {
		return err
	}
	if err = json.Unmarshal(raw, &existing); err != nil {
		return err
	}
	if existing != d {
		return ErrDispatchConflict
	}
	return nil
}

// WithdrawDispatch atomically closes dispatch for one repository and captures
// every outstanding ID in order. Closing an already closed gate replays its
// recorded withdrawal instead of capturing twice.
func (s *Store) WithdrawDispatch(ctx context.Context, repository int64, cause, closedBy string) (factory.Withdrawal, error) {
	if repository <= 0 || cause == "" || len(cause) > 256 || closedBy == "" || len(closedBy) > 128 {
		return factory.Withdrawal{}, errors.New("invalid dispatch withdrawal")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return factory.Withdrawal{}, err
	}
	defer func() { _ = tx.Rollback() }()
	// Create the logical default-open row before locking it. Dispatch
	// registration and publication correction registration lock this same row,
	// so the captured set cannot miss work that committed just before closure.
	if _, err = tx.ExecContext(ctx, `INSERT INTO factory_dispatch(repository,revision,open,data)
		VALUES($1,0,TRUE,'{}'::jsonb) ON CONFLICT(repository) DO NOTHING`, repository); err != nil {
		return factory.Withdrawal{}, err
	}
	var open bool
	var revision int64
	var raw []byte
	err = tx.QueryRowContext(ctx, `SELECT open,revision,data FROM factory_dispatch WHERE repository=$1 FOR UPDATE`, repository).Scan(&open, &revision, &raw)
	if err != nil {
		return factory.Withdrawal{}, err
	}
	var recorded factory.Withdrawal
	if err = json.Unmarshal(raw, &recorded); err != nil {
		return factory.Withdrawal{}, err
	}
	if !open {
		if err = recorded.Validate(); err != nil || len(recorded.ActiveCauses) == 0 {
			if err == nil {
				err = errors.New("closed dispatch has no active cause")
			}
			return factory.Withdrawal{}, err
		}
		for _, active := range recorded.ActiveCauses {
			if active == cause {
				if err = tx.Commit(); err != nil {
					return factory.Withdrawal{}, err
				}
				return recorded, nil
			}
		}
		if len(recorded.ActiveCauses) >= factory.MaxActiveDispatchCauses {
			return factory.Withdrawal{}, errors.New("too many active dispatch causes")
		}
		recorded.ActiveCauses = append(recorded.ActiveCauses, cause)
		if err = recorded.Validate(); err != nil {
			return factory.Withdrawal{}, err
		}
		nextRevision := revision + 1
		data, marshalErr := json.Marshal(recorded)
		if marshalErr != nil {
			return factory.Withdrawal{}, marshalErr
		}
		result, updateErr := tx.ExecContext(ctx, `UPDATE factory_dispatch SET revision=$1,data=$2
			WHERE repository=$3 AND revision=$4 AND NOT open`, nextRevision, string(data), repository, revision)
		if updateErr != nil {
			return factory.Withdrawal{}, fmt.Errorf("dispatch cause update failed: %w", updateErr)
		}
		n, affectedErr := result.RowsAffected()
		if affectedErr != nil {
			return factory.Withdrawal{}, affectedErr
		}
		if n != 1 {
			return factory.Withdrawal{}, ErrStaleRevision
		}
		if err = tx.Commit(); err != nil {
			return factory.Withdrawal{}, err
		}
		return recorded, nil
	}
	rows, err := tx.QueryContext(ctx, `SELECT id FROM factory_dispatch_regs WHERE repository=$1 ORDER BY seq LIMIT $2`, repository, factory.MaxCapturedDispatch+1)
	if err != nil {
		return factory.Withdrawal{}, err
	}
	captured := []string{}
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			_ = rows.Close()
			return factory.Withdrawal{}, err
		}
		captured = append(captured, id)
	}
	if err = rows.Close(); err != nil {
		return factory.Withdrawal{}, err
	}
	if len(captured) > factory.MaxCapturedDispatch {
		return factory.Withdrawal{}, errors.New("too many outstanding dispatches to withdraw")
	}
	withdrawal := factory.Withdrawal{Repository: repository, Revision: revision + 1, Cause: cause, ClosedBy: closedBy, Captured: captured, ActiveCauses: []string{cause}}
	if err = withdrawal.Validate(); err != nil {
		return factory.Withdrawal{}, err
	}
	data, err := json.Marshal(withdrawal)
	if err != nil {
		return factory.Withdrawal{}, err
	}
	result, err := tx.ExecContext(ctx, `UPDATE factory_dispatch SET revision=$1,open=FALSE,data=$2
		WHERE repository=$3 AND revision=$4 AND open`, withdrawal.Revision, string(data), repository, revision)
	if err != nil {
		return factory.Withdrawal{}, fmt.Errorf("dispatch withdrawal failed: %w", err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return factory.Withdrawal{}, err
	}
	if n != 1 {
		return factory.Withdrawal{}, ErrStaleRevision
	}
	if err = tx.Commit(); err != nil {
		return factory.Withdrawal{}, err
	}
	return withdrawal, nil
}

// ReopenDispatch clears non-Project-stop causes under the locked dispatch
// row. A Project stop remains in force until a verified Project start clears it.
func (s *Store) ReopenDispatch(ctx context.Context, repository, expectedRevision int64) (bool, int64, error) {
	return s.clearDispatchCauses(ctx, repository, expectedRevision, true)
}

// ClearProjectStop removes only the Project lifecycle cause after verified
// start and run quiescence. First-closure attribution remains a receipt.
func (s *Store) ClearProjectStop(ctx context.Context, repository, expectedRevision int64) (bool, int64, error) {
	return s.clearDispatchCauses(ctx, repository, expectedRevision, false)
}

// clearDispatchCauses is the single row-lock/CAS transaction for the two
// authorized cause-clear operations. Resume retains project_stop; verified
// Start removes only project_stop.
func (s *Store) clearDispatchCauses(ctx context.Context, repository, expectedRevision int64, retainProjectStop bool) (bool, int64, error) {
	if repository <= 0 || expectedRevision < 0 {
		return false, 0, errors.New("invalid dispatch cause clear")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return false, 0, err
	}
	defer func() { _ = tx.Rollback() }()
	var open bool
	var revision int64
	var raw []byte
	if err = tx.QueryRowContext(ctx, `SELECT open,revision,data FROM factory_dispatch WHERE repository=$1 FOR UPDATE`, repository).Scan(&open, &revision, &raw); err != nil {
		return false, 0, err
	}
	if open || revision != expectedRevision {
		return false, revision, ErrStaleRevision
	}
	var withdrawal factory.Withdrawal
	if err = json.Unmarshal(raw, &withdrawal); err != nil {
		return false, revision, err
	}
	if err = withdrawal.Validate(); err != nil || len(withdrawal.ActiveCauses) == 0 {
		if err == nil {
			err = errors.New("closed dispatch has no active cause")
		}
		return false, revision, err
	}
	remaining := make([]string, 0, len(withdrawal.ActiveCauses))
	for _, active := range withdrawal.ActiveCauses {
		keep := active == factory.CauseProjectStop
		if !retainProjectStop {
			keep = active != factory.CauseProjectStop
		}
		if keep {
			remaining = append(remaining, active)
		}
	}
	if len(remaining) == len(withdrawal.ActiveCauses) {
		if err = tx.Commit(); err != nil {
			return false, revision, err
		}
		return false, revision, nil
	}
	withdrawal.ActiveCauses = remaining
	nextRevision := revision + 1
	data, err := json.Marshal(withdrawal)
	if err != nil {
		return false, revision, fmt.Errorf("dispatch cause clear failed: %w", err)
	}
	nextOpen := len(remaining) == 0
	result, err := tx.ExecContext(ctx, `UPDATE factory_dispatch SET revision=$1,open=$2,data=$3 WHERE repository=$4 AND revision=$5 AND NOT open`,
		nextRevision, nextOpen, string(data), repository, revision)
	if err != nil {
		return false, revision, err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return false, revision, err
	}
	if n != 1 {
		return false, revision, ErrStaleRevision
	}
	if err = tx.Commit(); err != nil {
		return false, revision, err
	}
	return nextOpen, nextRevision, nil
}
