package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/levitateos/sodaos/internal/identity"
)

func identityBinding(c identity.Connection) string {
	return fmt.Sprintf("soda/identity/%s/%d", c.ID, c.Generation)
}

func (s *Store) IdentitySaveConnection(ctx context.Context, c identity.Connection, credential []byte) error {
	if s.grants == nil {
		return ErrGrantKey
	}
	if !identity.CredentialValid(credential) || c.OwnerID <= 0 || c.ID == "" || c.Generation <= 0 {
		return identity.ErrDenied
	}
	data, err := json.Marshal(c)
	if err != nil {
		return err
	}
	return s.identityAtomic(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `INSERT INTO identity_connections(id,owner_id,generation,state,data,credential) VALUES(?,?,?,?,?,?)`, c.ID, c.OwnerID, c.Generation, c.State, data, s.grants.seal(credential, identityBinding(c))); err != nil {
			return err
		}
		return appendIdentityEvent(ctx, tx, identity.Event{Action: "connected", OwnerID: c.OwnerID, ActorID: c.OwnerID, ConnectionID: c.ID, Generation: c.Generation})
	})
}

func (s *Store) IdentityConnection(ctx context.Context, id string) (identity.Connection, error) {
	var c identity.Connection
	var data []byte
	err := s.db.QueryRowContext(ctx, `SELECT data FROM identity_connections WHERE id=?`, id).Scan(&data)
	if err == nil {
		err = json.Unmarshal(data, &c)
	}
	return c, err
}

func (s *Store) IdentityCredential(ctx context.Context, c identity.Connection) ([]byte, error) {
	var encrypted []byte
	err := s.db.QueryRowContext(ctx, `SELECT credential FROM identity_connections WHERE id=? AND generation=? AND state='ready'`, c.ID, c.Generation).Scan(&encrypted)
	if err != nil {
		return nil, err
	}
	return s.grants.open(encrypted, identityBinding(c))
}

func (s *Store) IdentityConnections(ctx context.Context, owner int64) ([]identity.Connection, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT data FROM identity_connections WHERE owner_id=? ORDER BY id`, owner)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []identity.Connection{}
	for rows.Next() {
		var data []byte
		var c identity.Connection
		if err = rows.Scan(&data); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(data, &c); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Store) IdentityAvailable(ctx context.Context, actor int64, project string) ([]identity.Connection, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT c.data FROM identity_connections c WHERE c.state='ready' AND (c.owner_id=? OR EXISTS(SELECT 1 FROM identity_grants g WHERE g.connection_id=c.id AND g.user_id=? AND g.project_id=? AND g.revoked=0)) ORDER BY c.id`, actor, actor, project)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []identity.Connection{}
	for rows.Next() {
		var d []byte
		var c identity.Connection
		if err = rows.Scan(&d); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(d, &c); err != nil {
			return nil, err
		}
		if c.OwnerID != actor {
			c.Email = ""
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// IdentityState denies admission before callers attempt native termination.
func (s *Store) IdentityState(ctx context.Context, c identity.Connection, state string) error {
	c.State = state
	data, err := json.Marshal(c)
	if err != nil {
		return err
	}
	return s.identityAtomic(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, `UPDATE identity_connections SET state=?,data=?,credential=CASE WHEN ?='revoked' THEN X'' ELSE credential END WHERE id=? AND generation=?`, state, data, state, c.ID, c.Generation)
		if err = identityChanged(res, err); err != nil {
			return err
		}
		return appendIdentityEvent(ctx, tx, identity.Event{Action: state, OwnerID: c.OwnerID, ActorID: c.OwnerID, ConnectionID: c.ID, Generation: c.Generation})
	})
}

func identityChanged(res sql.Result, err error) error {
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return identity.ErrStale
	}
	return nil
}

func (s *Store) IdentitySaveGrant(ctx context.Context, g identity.Grant) error {
	data, err := json.Marshal(g)
	if err != nil {
		return err
	}
	return s.identityAtomic(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `INSERT INTO identity_grants(id,connection_id,user_id,project_id,revision,revoked,data) VALUES(?,?,?,?,?,?,?)`, g.ID, g.ConnectionID, g.UserID, g.ProjectID, g.Revision, g.Revoked, data); err != nil {
			return err
		}
		return appendIdentityEvent(ctx, tx, identity.Event{Action: "grant_created", ConnectionID: g.ConnectionID, GrantID: g.ID, ProjectID: g.ProjectID})
	})
}

func (s *Store) IdentityGrant(ctx context.Context, id string) (identity.Grant, error) {
	var g identity.Grant
	var data []byte
	err := s.db.QueryRowContext(ctx, `SELECT data FROM identity_grants WHERE id=?`, id).Scan(&data)
	if err == nil {
		err = json.Unmarshal(data, &g)
	}
	return g, err
}

func (s *Store) IdentityGrants(ctx context.Context, id string) ([]identity.Grant, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT data FROM identity_grants WHERE connection_id=? ORDER BY id`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []identity.Grant{}
	for rows.Next() {
		var d []byte
		var g identity.Grant
		if err = rows.Scan(&d); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(d, &g); err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

func (s *Store) IdentityRevokeGrant(ctx context.Context, g identity.Grant) error {
	g.Revoked = true
	g.Revision++
	data, err := json.Marshal(g)
	if err != nil {
		return err
	}
	return s.identityAtomic(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, `UPDATE identity_grants SET revoked=1,revision=?,data=? WHERE id=? AND revision=?`, g.Revision, data, g.ID, g.Revision-1)
		if err = identityChanged(res, err); err != nil {
			return err
		}
		return appendIdentityEvent(ctx, tx, identity.Event{Action: "grant_revoked", ConnectionID: g.ConnectionID, GrantID: g.ID, ProjectID: g.ProjectID})
	})
}

func (s *Store) IdentityLeases(ctx context.Context) ([]identity.Lease, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT data FROM identity_leases ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []identity.Lease{}
	for rows.Next() {
		var d []byte
		var l identity.Lease
		if err = rows.Scan(&d); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(d, &l); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

func (s *Store) IdentityLease(ctx context.Context, id string) (identity.Lease, error) {
	var l identity.Lease
	var d []byte
	err := s.db.QueryRowContext(ctx, `SELECT data FROM identity_leases WHERE id=?`, id).Scan(&d)
	if err == nil {
		err = json.Unmarshal(d, &l)
	}
	return l, err
}

func (s *Store) IdentityReserve(ctx context.Context, l identity.Lease) error {
	d, err := json.Marshal(l)
	if err != nil {
		return err
	}
	return s.identityAtomic(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, `INSERT INTO identity_leases(id,connection_id,data) SELECT ?,id,? FROM identity_connections WHERE id=? AND generation=? AND state='ready' AND NOT EXISTS(SELECT 1 FROM identity_leases WHERE connection_id=?) AND (?='' OR EXISTS(SELECT 1 FROM identity_grants WHERE id=? AND connection_id=? AND user_id=? AND project_id=? AND revision=? AND revoked=0))`, l.ID, d, l.ConnectionID, l.Generation, l.ConnectionID, l.GrantID, l.GrantID, l.ConnectionID, l.ActorID, l.ProjectID, l.GrantRevision)
		if err != nil {
			return err
		}
		n, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if n != 1 {
			return identity.ErrBusy
		}
		return appendIdentityEvent(ctx, tx, leaseEvent(l, "reserved"))
	})
}

func (s *Store) IdentityRegister(ctx context.Context, l identity.Lease) error {
	d, err := json.Marshal(l)
	if err != nil {
		return err
	}
	return s.identityAtomic(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, `UPDATE identity_leases SET data=? WHERE id=? AND json_extract(data,'$.binding') IS NULL`, d, l.ID)
		if err = identityChanged(res, err); err != nil {
			return err
		}
		return appendIdentityEvent(ctx, tx, leaseEvent(l, "registered"))
	})
}

// IdentityReturn atomically accepts maintained bytes and releases one generation.
func (s *Store) IdentityReturn(ctx context.Context, l identity.Lease, credential []byte) error {
	if !identity.CredentialValid(credential) {
		return identity.ErrUncertain
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	var data []byte
	var c identity.Connection
	if err = tx.QueryRowContext(ctx, `SELECT data FROM identity_connections WHERE id=? AND generation=? AND state='ready'`, l.ConnectionID, l.Generation).Scan(&data); err != nil {
		return identity.ErrStale
	}
	if err = json.Unmarshal(data, &c); err != nil {
		return err
	}
	c.Generation++
	data, err = json.Marshal(c)
	if err != nil {
		return err
	}
	res, err := tx.ExecContext(ctx, `UPDATE identity_connections SET generation=?,data=?,credential=? WHERE id=? AND generation=?`, c.Generation, data, s.grants.seal(credential, identityBinding(c)), c.ID, l.Generation)
	if err = identityChanged(res, err); err != nil {
		return err
	}
	res, err = tx.ExecContext(ctx, `DELETE FROM identity_leases WHERE id=? AND connection_id=? AND json_extract(data,'$.generation')=?`, l.ID, l.ConnectionID, l.Generation)
	if err = identityChanged(res, err); err != nil {
		return err
	}
	if err = appendIdentityEvent(ctx, tx, leaseEvent(l, "returned")); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) IdentityForgetLease(ctx context.Context, id string) error {
	return s.identityAtomic(ctx, func(tx *sql.Tx) error {
		var data []byte
		var l identity.Lease
		if err := tx.QueryRowContext(ctx, `SELECT data FROM identity_leases WHERE id=?`, id).Scan(&data); err != nil {
			return err
		}
		if err := json.Unmarshal(data, &l); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM identity_leases WHERE id=?`, id); err != nil {
			return err
		}
		return appendIdentityEvent(ctx, tx, leaseEvent(l, "reconciled"))
	})
}
