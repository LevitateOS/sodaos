package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/url"

	"github.com/levitateos/sodaos/internal/application"
)

// CreateApplication persists an operator-admitted initial reference and its
// explicit administrator grant. The caller admits the operator; this store
// never manufactures a native installation observation or app identity.
func (s *Store) CreateApplication(ctx context.Context, r application.Record) error {
	if err := r.Validate(); err != nil {
		return err
	}
	if r.Application.Revision != 1 || r.Observation != nil {
		return errors.New("initial application must be unobserved revision one")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	u, err := url.Parse(r.Application.Origin)
	if err != nil {
		return err
	}
	a := r.Application
	_, err = tx.ExecContext(ctx, `INSERT INTO applications(id,administrator_id,name,data_id,owner_team,origin,hostname,port,revision) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		a.ID, r.AdministratorID, a.Name, a.Data, a.OwnerTeam, a.Origin, u.Hostname(), a.Port, a.Revision)
	if err != nil {
		return err
	}
	if err := insertApplicationRevision(ctx, tx, a); err != nil {
		return err
	}
	return tx.Commit()
}

func insertApplicationRevision(ctx context.Context, tx *sql.Tx, a application.Spec) error {
	raw, err := json.Marshal(a)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO application_revisions(application_id,revision,spec) VALUES($1,$2,$3)`, a.ID, a.Revision, string(raw))
	return err
}

const applicationRecordSQL = `SELECT r.spec,a.administrator_id,a.observation FROM applications a JOIN application_revisions r ON r.application_id=a.id AND r.revision=a.revision WHERE a.id=$1 AND a.administrator_id=$2`

func scanApplication(row *sql.Row) (application.Record, error) {
	var raw, observed []byte
	var r application.Record
	if err := row.Scan(&raw, &r.AdministratorID, &observed); err != nil {
		return r, err
	}
	if err := json.Unmarshal(raw, &r.Application); err != nil {
		return r, err
	}
	if observed != nil {
		if err := json.Unmarshal(observed, &r.Observation); err != nil {
			return r, err
		}
	}
	return r, r.Validate()
}

// Application returns only records covered by the explicit administrator grant.
func (s *Store) Application(ctx context.Context, id string, actorID int64) (application.Record, error) {
	return scanApplication(s.db.QueryRowContext(ctx, applicationRecordSQL, id, actorID))
}

// ApplicationRevision retains exact prior bytes references for recovery. A
// desired replacement does not erase the image/configuration that preceded it.
func (s *Store) ApplicationRevision(ctx context.Context, id string, actorID, revision int64) (application.Spec, error) {
	var raw []byte
	var a application.Spec
	err := s.db.QueryRowContext(ctx, `SELECT r.spec FROM application_revisions r JOIN applications a ON a.id=r.application_id WHERE a.id=$1 AND a.administrator_id=$2 AND r.revision=$3`, id, actorID, revision).Scan(&raw)
	if err != nil {
		return a, err
	}
	if err := json.Unmarshal(raw, &a); err != nil {
		return a, err
	}
	return a, a.Validate()
}

// ReplaceApplication atomically preserves immutable history and advances the
// admitted references with a revision CAS. Native success remains unknown.
func (s *Store) ReplaceApplication(ctx context.Context, actorID, expectedRevision int64, next application.Spec) error {
	if err := next.Validate(); err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	current, err := scanApplication(tx.QueryRowContext(ctx, applicationRecordSQL+` FOR UPDATE OF a`, next.ID, actorID))
	if err != nil {
		return err
	}
	if current.Application.Revision != expectedRevision {
		return errors.New("application revision changed")
	}
	if err := current.Application.Replacement(next); err != nil {
		return err
	}
	if err := insertApplicationRevision(ctx, tx, next); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE applications SET revision=$1,observation=NULL WHERE id=$2`, next.Revision, next.ID); err != nil {
		return err
	}
	return tx.Commit()
}

// ObserveApplication publishes a matching native result with both reference
// revision and prior-observation CAS. Pass nil after an uncertain native call;
// nil remains unknown and cannot become readiness from desired references.
func (s *Store) ObserveApplication(ctx context.Context, id string, actorID, revision int64, expected, next *application.Observation) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	current, err := scanApplication(tx.QueryRowContext(ctx, applicationRecordSQL+` FOR UPDATE OF a`, id, actorID))
	if err != nil {
		return err
	}
	if current.Application.Revision != revision || !application.SameObservation(current.Observation, expected) {
		return errors.New("application observation or revision changed")
	}
	value, err := applicationObservationValue(next, current.Application)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE applications SET observation=$1 WHERE id=$2`, value, id); err != nil {
		return err
	}
	return tx.Commit()
}

func applicationObservationValue(next *application.Observation, spec application.Spec) (any, error) {
	if next == nil {
		return nil, nil
	}
	if err := next.Validate(spec); err != nil {
		return nil, err
	}
	raw, err := json.Marshal(next)
	if err != nil {
		return nil, err
	}
	return string(raw), nil
}

// OperatorApplication requires operator admission by its caller. Ordinary
// administrators use Application, which checks the stored stable grant.
func (s *Store) OperatorApplication(ctx context.Context, id string) (application.Record, error) {
	return scanApplication(s.db.QueryRowContext(ctx, `SELECT r.spec,a.administrator_id,a.observation FROM applications a JOIN application_revisions r ON r.application_id=a.id AND r.revision=a.revision WHERE a.id=$1`, id))
}

// Applications returns a bounded inventory for one admitted administrator.
func (s *Store) Applications(ctx context.Context, actorID int64) ([]application.Record, error) {
	return s.applicationInventory(ctx, `SELECT r.spec,a.administrator_id,a.observation FROM applications a JOIN application_revisions r ON r.application_id=a.id AND r.revision=a.revision WHERE a.administrator_id=$1 ORDER BY a.id LIMIT 33`, actorID)
}

// OperatorApplications requires operator admission by its caller. The 33rd row
// signals incomplete inventory rather than silently claiming complete coverage.
func (s *Store) OperatorApplications(ctx context.Context) ([]application.Record, error) {
	return s.applicationInventory(ctx, `SELECT r.spec,a.administrator_id,a.observation FROM applications a JOIN application_revisions r ON r.application_id=a.id AND r.revision=a.revision ORDER BY a.id LIMIT 33`)
}

func (s *Store) applicationInventory(ctx context.Context, query string, args ...any) ([]application.Record, error) {
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	records := []application.Record{}
	for rows.Next() {
		var raw, observed []byte
		var r application.Record
		if err := rows.Scan(&raw, &r.AdministratorID, &observed); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(raw, &r.Application); err != nil {
			return nil, err
		}
		if observed != nil {
			if err := json.Unmarshal(observed, &r.Observation); err != nil {
				return nil, err
			}
		}
		if err := r.Validate(); err != nil {
			return nil, err
		}
		records = append(records, r)
	}
	return records, rows.Err()
}
