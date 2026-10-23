package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/url"

	"github.com/levitateos/sodaos/internal/application"
)

const ApplicationSetupsDDL = `CREATE TABLE application_setups(id TEXT PRIMARY KEY,application_id TEXT NOT NULL UNIQUE,administrator_id BIGINT NOT NULL CHECK(administrator_id>0),data_id TEXT NOT NULL UNIQUE,origin TEXT NOT NULL UNIQUE,hostname TEXT NOT NULL UNIQUE,port INTEGER NOT NULL UNIQUE,state TEXT NOT NULL CHECK(state IN ('pending','complete','aborted')),data JSONB NOT NULL CHECK(octet_length(data::text)<=8192))`

// Both normal registration and provisional setup serialize their namespace
// reservation so a foreign provisional address cannot be claimed in between.
func lockApplicationAdmissions(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(1936684132,1)`)
	return err
}
func reserveSetupNamespace(ctx context.Context, tx *sql.Tx, s application.SetupSpec) error {
	var conflict bool
	err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM applications WHERE id=$1 OR data_id=$2 OR origin=$3 OR hostname=$4 OR port=$5) OR EXISTS(SELECT 1 FROM application_setups WHERE application_id=$1 OR data_id=$2 OR origin=$3 OR hostname=$4 OR port=$5)`, s.ID, s.Data, s.Origin, mustHostname(s.Origin), s.Port).Scan(&conflict)
	if err != nil {
		return err
	}
	if conflict {
		return errors.New("application namespace is already reserved")
	}
	return nil
}
func mustHostname(origin string) string { u, _ := url.Parse(origin); return u.Hostname() }
func (s *Store) CreateApplicationSetup(ctx context.Context, record application.SetupRecord) error {
	if err := record.Validate(); err != nil {
		return err
	}
	if record.State != "pending" || record.Observation != nil {
		return errors.New("new setup must be pending and unobserved")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = lockApplicationAdmissions(ctx, tx); err != nil {
		return err
	}
	if err = reserveSetupNamespace(ctx, tx, record.Application); err != nil {
		return err
	}
	raw, err := json.Marshal(record)
	if err != nil {
		return err
	}
	a := record.Application
	_, err = tx.ExecContext(ctx, `INSERT INTO application_setups(id,application_id,administrator_id,data_id,origin,hostname,port,state,data) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`, record.ID, a.ID, record.AdministratorID, a.Data, a.Origin, mustHostname(a.Origin), a.Port, record.State, string(raw))
	if err != nil {
		return err
	}
	return tx.Commit()
}
func scanApplicationSetup(row interface{ Scan(...any) error }) (application.SetupRecord, error) {
	var r application.SetupRecord
	var raw []byte
	if err := row.Scan(&raw); err != nil {
		return r, err
	}
	if err := json.Unmarshal(raw, &r); err != nil {
		return r, err
	}
	return r, r.Validate()
}
func (s *Store) ApplicationSetup(ctx context.Context, id string) (application.SetupRecord, error) {
	return scanApplicationSetup(s.db.QueryRowContext(ctx, `SELECT data FROM application_setups WHERE id=$1`, id))
}
func (s *Store) ApplicationSetups(ctx context.Context) ([]application.SetupRecord, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT data FROM application_setups WHERE state='pending' ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []application.SetupRecord{}
	for rows.Next() {
		r, err := scanApplicationSetup(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, r)
	}
	return result, rows.Err()
}
func (s *Store) ObserveApplicationSetup(ctx context.Context, before application.SetupRecord, next *application.SetupObservation) error {
	if next != nil {
		if err := next.Validate(); err != nil {
			return err
		}
	}
	after := before
	after.Observation = next
	return s.updateApplicationSetup(ctx, before, after)
}
func (s *Store) updateApplicationSetup(ctx context.Context, before, after application.SetupRecord) error {
	old, _ := json.Marshal(before)
	raw, _ := json.Marshal(after)
	result, err := s.db.ExecContext(ctx, `UPDATE application_setups SET data=$1,state=$2 WHERE id=$3 AND data=$4::jsonb`, string(raw), after.State, before.ID, string(old))
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count != 1 {
		return errors.New("setup changed")
	}
	return nil
}
func (s *Store) SettleApplicationSetup(ctx context.Context, before application.SetupRecord, result application.SetupResult, abort bool) error {
	if err := before.Validate(); err != nil {
		return err
	}
	if before.State != "pending" || result.Operation != before.ID || result.Observation.Running || result.Observation.Ready {
		return errors.New("setup writer must stop before settlement")
	}
	if err := result.Observation.Validate(); err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = lockApplicationAdmissions(ctx, tx); err != nil {
		return err
	}
	current, err := scanApplicationSetup(tx.QueryRowContext(ctx, `SELECT data FROM application_setups WHERE id=$1 FOR UPDATE`, before.ID))
	if err != nil {
		return err
	}
	old, _ := json.Marshal(before)
	actual, _ := json.Marshal(current)
	if string(old) != string(actual) {
		return errors.New("setup changed")
	}
	after := before
	after.Observation = &result.Observation
	if abort {
		after.State = "aborted"
	} else {
		if result.Receipt == nil || result.Application == nil {
			return errors.New("confirmed setup receipt required")
		}
		if err = result.Receipt.Validate(before); err != nil {
			return err
		}
		spec := before.Application.Bind(result.Receipt.OwnerTeam)
		if err = result.Application.Validate(spec); err != nil {
			return err
		}
		if result.Application.Running || result.Application.Ready || result.Application.Container == "" {
			return errors.New("normal application must be installed stopped")
		}
		record := application.Record{Application: spec, AdministratorID: before.AdministratorID, Observation: result.Application}
		if err = insertApplicationRecord(ctx, tx, record); err != nil {
			return err
		}
		after.State = "complete"
	}
	raw, _ := json.Marshal(after)
	_, err = tx.ExecContext(ctx, `UPDATE application_setups SET state=$1,data=$2 WHERE id=$3`, after.State, string(raw), after.ID)
	if err != nil {
		return err
	}
	return tx.Commit()
}
