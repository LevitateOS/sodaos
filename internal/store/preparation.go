package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/levitateos/sodaos/internal/project"
)

// SaveLifecycleGrant records the owner's standing create/start grant. The
// revision CAS rejects stale writers; a withdrawn grant is an inactive row,
// never a deleted one.
func (s *Store) SaveLifecycleGrant(ctx context.Context, g project.LifecycleGrant) error {
	if err := g.Validate(); err != nil {
		return err
	}
	next := g
	next.Revision++
	nextData, err := json.Marshal(next)
	if err != nil {
		return err
	}
	result, err := s.db.ExecContext(ctx, `INSERT INTO project_lifecycle_grants(project_id,revision,data) VALUES(?,?,?)
		ON CONFLICT(project_id) DO UPDATE SET revision=?,data=? WHERE project_lifecycle_grants.revision=?`,
		g.Project, next.Revision, string(nextData), next.Revision, string(nextData), g.Revision)
	if err != nil {
		return fmt.Errorf("lifecycle grant save failed: %w", err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return errors.New("stale lifecycle grant revision")
	}
	return nil
}

// LifecycleGrant returns the standing create/start grant for one project.
func (s *Store) LifecycleGrant(ctx context.Context, projectID string) (project.LifecycleGrant, error) {
	var g project.LifecycleGrant
	var data []byte
	err := s.db.QueryRowContext(ctx, `SELECT data FROM project_lifecycle_grants WHERE project_id=?`, projectID).Scan(&data)
	if err == nil {
		err = json.Unmarshal(data, &g)
	}
	return g, err
}

// SaveMaintenanceHold records the maintenance hold with CAS revision. Callers
// sync the native marker through the host; this row stays the source of truth.
func (s *Store) SaveMaintenanceHold(ctx context.Context, m project.MaintenanceHold) error {
	if err := m.Validate(); err != nil {
		return err
	}
	next := m
	next.Revision++
	nextData, err := json.Marshal(next)
	if err != nil {
		return err
	}
	hold := 0
	if next.Hold {
		hold = 1
	}
	result, err := s.db.ExecContext(ctx, `INSERT INTO project_maintenance(project_id,revision,hold,data) VALUES(?,?,?,?)
		ON CONFLICT(project_id) DO UPDATE SET revision=?,hold=?,data=? WHERE project_maintenance.revision=?`,
		m.Project, next.Revision, hold, string(nextData), next.Revision, hold, string(nextData), m.Revision)
	if err != nil {
		return fmt.Errorf("maintenance hold save failed: %w", err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return errors.New("stale maintenance hold revision")
	}
	return nil
}

// MaintenanceHold returns the current maintenance hold for one project.
// A missing row means no hold was ever recorded.
func (s *Store) MaintenanceHold(ctx context.Context, projectID string) (project.MaintenanceHold, error) {
	var m project.MaintenanceHold
	var data []byte
	err := s.db.QueryRowContext(ctx, `SELECT data FROM project_maintenance WHERE project_id=?`, projectID).Scan(&data)
	if err == nil {
		err = json.Unmarshal(data, &m)
	}
	return m, err
}

// AdmitPreparation records one preparation identity with its immutable
// requirement/admin references. Repeating the same identity returns the
// recorded row; changed references for that identity conflict.
func (s *Store) AdmitPreparation(ctx context.Context, p project.StoredPreparation) (project.StoredPreparation, bool, error) {
	if err := p.Validate(); err != nil {
		return project.StoredPreparation{}, false, err
	}
	data, err := json.Marshal(p)
	if err != nil {
		return project.StoredPreparation{}, false, err
	}
	prep := p.Preparation
	result, err := s.db.ExecContext(ctx, `INSERT INTO project_preparations(id,project_id,role,revision,requirements,approval,data) VALUES(?,?,?,?,?,?,?)
		ON CONFLICT(id) DO NOTHING`, prep.ID, prep.Project, prep.Role, prep.Revision, prep.Requirements.ID, prep.Approval.ID, string(data))
	if err != nil {
		return project.StoredPreparation{}, false, fmt.Errorf("preparation admission failed: %w", err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return project.StoredPreparation{}, false, err
	}
	if n == 1 {
		return p, true, nil
	}
	existing, err := s.Preparation(ctx, prep.ID)
	if err != nil {
		return project.StoredPreparation{}, false, err
	}
	if existing.Preparation.Requirements != prep.Requirements || existing.Preparation.Approval != prep.Approval ||
		existing.Preparation.SourceCommit != prep.SourceCommit || existing.Preparation.SetupDigest != prep.SetupDigest {
		return project.StoredPreparation{}, false, errors.New("preparation identity already carries different approved inputs")
	}
	return existing, false, nil
}

// ObservePreparation advances only the observed state under CAS revision. The
// approved references are additionally guarded by a schema trigger.
func (s *Store) ObservePreparation(ctx context.Context, p project.StoredPreparation) error {
	if err := p.Validate(); err != nil {
		return err
	}
	next := p
	next.Preparation.Revision++
	data, err := json.Marshal(next)
	if err != nil {
		return err
	}
	result, err := s.db.ExecContext(ctx, `UPDATE project_preparations SET revision=?,data=? WHERE id=? AND revision=?`,
		next.Preparation.Revision, string(data), p.Preparation.ID, p.Preparation.Revision)
	if err != nil {
		return fmt.Errorf("preparation observation failed: %w", err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return errors.New("stale preparation revision")
	}
	return nil
}

// Preparation returns one durable preparation record by identity.
func (s *Store) Preparation(ctx context.Context, id string) (project.StoredPreparation, error) {
	var p project.StoredPreparation
	var data []byte
	err := s.db.QueryRowContext(ctx, `SELECT data FROM project_preparations WHERE id=?`, id).Scan(&data)
	if err == nil {
		err = json.Unmarshal(data, &p)
	}
	return p, err
}

// ProjectPreparations lists the durable preparation records for one project.
func (s *Store) ProjectPreparations(ctx context.Context, projectID string) ([]project.StoredPreparation, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT data FROM project_preparations WHERE project_id=? ORDER BY id LIMIT 129`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []project.StoredPreparation{}
	for rows.Next() {
		var data []byte
		var p project.StoredPreparation
		if err = rows.Scan(&data); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(data, &p); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}
