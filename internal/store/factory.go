package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
)

// RecordFactoryRun stores a fresh supervised run before any host call. A
// consumed run ID is never reused; only the supervisor mutates the record.
func (s *Store) RecordFactoryRun(ctx context.Context, r factory.Run) error {
	if err := r.Validate(); err != nil {
		return err
	}
	if r.Outcome != "" || r.Reconciled {
		return errors.New("new run must be active and unsettled")
	}
	data, err := json.Marshal(r)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO factory_runs(id,active,settled,data) VALUES(?,1,0,?)`, r.ID, string(data))
	if err != nil {
		return fmt.Errorf("factory run record failed: %w", err)
	}
	return nil
}

func (s *Store) FactoryRun(ctx context.Context, id string) (factory.Run, error) {
	var r factory.Run
	var data []byte
	err := s.db.QueryRowContext(ctx, `SELECT data FROM factory_runs WHERE id=?`, id).Scan(&data)
	if err == nil {
		err = json.Unmarshal(data, &r)
	}
	return r, err
}

// SaveFactoryRun persists supervisor observations. Credential facts and the
// settled marker only advance; a terminal outcome never changes.
func (s *Store) SaveFactoryRun(ctx context.Context, r factory.Run) error {
	if err := r.Validate(); err != nil {
		return err
	}
	data, err := json.Marshal(r)
	if err != nil {
		return err
	}
	result, err := s.db.ExecContext(ctx, `UPDATE factory_runs SET active=?,settled=?,data=? WHERE id=?
AND (coalesce(json_extract(data,'$.identity_lease_id'),'')='' OR (json_extract(data,'$.identity_lease_id')=? AND json_extract(data,'$.identity_generation')=?))
AND (json_extract(data,'$.identity_binding') IS NULL OR json_extract(data,'$.identity_binding')=json_extract(?,'$.identity_binding'))
AND (coalesce(json_extract(data,'$.credential_delegated'),0)=0 OR ?=1)
AND (coalesce(json_extract(data,'$.credential_returned'),0)=0 OR ?=1)
AND (coalesce(json_extract(data,'$.outcome'),'')='' OR json_extract(data,'$.outcome')=?)
AND (json_extract(data,'$.reconciled')=0 OR ?=1)`, r.Outcome == "", r.Reconciled, string(data), r.ID, r.IdentityLeaseID, r.IdentityGeneration, string(data), r.CredentialDelegated, r.CredentialReturned, r.Outcome, r.Reconciled)
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

// FactoryRuns returns recorded runs, newest first, bounded for the private
// operator status read. Reconciliation never adopts host state it did not
// record; unknown runs stay invisible until recorded.
func (s *Store) FactoryRuns(ctx context.Context, limit int) ([]factory.Run, error) {
	if limit < 1 || limit > 1000 {
		return nil, errors.New("invalid run list bound")
	}
	rows, err := s.db.QueryContext(ctx, `SELECT data FROM factory_runs ORDER BY rowid DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var runs []factory.Run
	for rows.Next() {
		var data []byte
		var r factory.Run
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(data, &r); err != nil {
			return nil, err
		}
		runs = append(runs, r)
	}
	return runs, rows.Err()
}

// RecordFactoryCommand stores an operator command exactly once. The same ID
// with the same digest returns the stored outcome; a changed payload
// conflicts instead of executing twice.
func (s *Store) RecordFactoryCommand(ctx context.Context, c factory.Command, now time.Time) (factory.Command, bool, error) {
	if err := c.Validate(); err != nil {
		return factory.Command{}, false, err
	}
	if c.Outcome != "" {
		return factory.Command{}, false, errors.New("new command must not carry an outcome")
	}
	created := now.UTC().Format(time.RFC3339Nano)
	result, err := s.db.ExecContext(ctx, `INSERT INTO factory_commands(id,type,target,principal,digest,payload,created) VALUES(?,?,?,?,?,?,?) ON CONFLICT(id) DO NOTHING`, c.ID, c.Type, c.Target, c.Principal, c.Digest, c.Payload, created)
	if err != nil {
		return factory.Command{}, false, fmt.Errorf("factory command record failed: %w", err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return factory.Command{}, false, err
	}
	stored, err := s.FactoryCommand(ctx, c.ID)
	if err != nil {
		return factory.Command{}, false, err
	}
	if stored.Digest != c.Digest {
		return factory.Command{}, false, ErrCommandConflict
	}
	return stored, n == 1, nil
}

func (s *Store) FactoryCommand(ctx context.Context, id string) (factory.Command, error) {
	var c factory.Command
	var created, finished string
	err := s.db.QueryRowContext(ctx, `SELECT id,type,target,principal,digest,payload,outcome,created,finished FROM factory_commands WHERE id=?`, id).Scan(&c.ID, &c.Type, &c.Target, &c.Principal, &c.Digest, &c.Payload, &c.Outcome, &created, &finished)
	if err != nil {
		return c, err
	}
	c.Created, c.Finished = created, finished
	return c, nil
}

// FinishFactoryCommand records the durable outcome exactly once. A lost reply
// replays the stored outcome; it never re-executes the command.
func (s *Store) FinishFactoryCommand(ctx context.Context, id, outcome string, now time.Time) error {
	if len(outcome) > 64<<10 {
		return errors.New("command outcome exceeds retained output limit")
	}
	finished := now.UTC().Format(time.RFC3339Nano)
	result, err := s.db.ExecContext(ctx, `UPDATE factory_commands SET outcome=?,finished=? WHERE id=? AND finished=''`, outcome, finished, id)
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
