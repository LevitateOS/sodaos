package store

import (
	"context"
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
	result, err := s.db.ExecContext(ctx, `INSERT INTO `+table+`(`+key+`,revision,data) VALUES(?,?,?)
		ON CONFLICT(`+key+`) DO UPDATE SET revision=?,data=? WHERE `+table+`.revision=?`,
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
	err := s.db.QueryRowContext(ctx, `SELECT data FROM `+table+` WHERE `+key+`=?`, id).Scan(&data)
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
	result, err := s.db.ExecContext(ctx, `INSERT INTO factory_sponsorships(repository,connection,revision,data) VALUES(?,?,?,?)
		ON CONFLICT(repository,connection) DO UPDATE SET revision=?,data=? WHERE factory_sponsorships.revision=?`,
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
	err := s.db.QueryRowContext(ctx, `SELECT data FROM factory_sponsorships WHERE repository=? AND connection=?`, repository, connection).Scan(&data)
	if err == nil {
		err = json.Unmarshal(data, &sp)
	}
	return sp, err
}

// Sponsorships lists the recorded connection sponsorships for a repository.
func (s *Store) Sponsorships(ctx context.Context, repository int64) ([]factory.Sponsorship, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT data FROM factory_sponsorships WHERE repository=? ORDER BY connection LIMIT 33`, repository)
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
	var open int
	var revision int64
	var data []byte
	var withdrawal factory.Withdrawal
	err := s.db.QueryRowContext(ctx, `SELECT open,revision,data FROM factory_dispatch WHERE repository=?`, repository).Scan(&open, &revision, &data)
	if errors.Is(err, ErrNotFound) {
		return true, 0, factory.Withdrawal{}, nil
	}
	if err == nil {
		err = json.Unmarshal(data, &withdrawal)
	}
	return open == 1, revision, withdrawal, err
}

// RegisterDispatch records one outstanding dispatch under the open gate. A
// delayed registration after withdrawal is refused; it cannot escape the
// captured set. The same ID with identical content replays.
func (s *Store) RegisterDispatch(ctx context.Context, d factory.DispatchRegistration) error {
	if err := d.Validate(); err != nil {
		return err
	}
	open, _, _, err := s.DispatchState(ctx, d.Repository)
	if err != nil {
		return err
	}
	if !open {
		return ErrDispatchClosed
	}
	data, err := json.Marshal(d)
	if err != nil {
		return err
	}
	result, err := s.db.ExecContext(ctx, `INSERT INTO factory_dispatch_regs(id,repository,revision,data) VALUES(?,?,?,?) ON CONFLICT(id) DO NOTHING`,
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
	if err = s.db.QueryRowContext(ctx, `SELECT data FROM factory_dispatch_regs WHERE id=?`, d.ID).Scan(&raw); err != nil {
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
	open, revision, recorded, err := s.DispatchState(ctx, repository)
	if err != nil {
		return factory.Withdrawal{}, err
	}
	if !open {
		return recorded, nil
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id FROM factory_dispatch_regs WHERE repository=? ORDER BY rowid LIMIT ?`, repository, factory.MaxCapturedDispatch+1)
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
	withdrawal := factory.Withdrawal{Repository: repository, Revision: revision + 1, Cause: cause, ClosedBy: closedBy, Captured: captured}
	if err = withdrawal.Validate(); err != nil {
		return factory.Withdrawal{}, err
	}
	data, err := json.Marshal(withdrawal)
	if err != nil {
		return factory.Withdrawal{}, err
	}
	result, err := s.db.ExecContext(ctx, `INSERT INTO factory_dispatch(repository,revision,open,data) VALUES(?,?,0,?)
		ON CONFLICT(repository) DO UPDATE SET revision=?,open=0,data=? WHERE factory_dispatch.revision=? AND factory_dispatch.open=1`,
		repository, withdrawal.Revision, string(data), withdrawal.Revision, string(data), revision)
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
	return withdrawal, nil
}

// ReopenDispatch reopens a withdrawn gate under CAS revision after an
// authorized control revalidated every grant. Outstanding registrations
// stay recorded; new dispatches bind fresh authority.
func (s *Store) ReopenDispatch(ctx context.Context, repository, expectedRevision int64) error {
	if repository <= 0 || expectedRevision < 0 {
		return errors.New("invalid dispatch reopen")
	}
	result, err := s.db.ExecContext(ctx, `UPDATE factory_dispatch SET revision=revision+1,open=1 WHERE repository=? AND revision=? AND open=0`,
		repository, expectedRevision)
	if err != nil {
		return fmt.Errorf("dispatch reopen failed: %w", err)
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
