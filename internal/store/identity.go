package store

import (
	"context"
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
	if !identity.ProviderValid(c.ProviderID) || !identity.CredentialValid(credential) || c.OwnerID <= 0 || c.ID == "" || c.Generation <= 0 {
		return identity.ErrDenied
	}
	data, err := json.Marshal(c)
	if err != nil {
		return err
	}
	return s.identityAtomic(ctx, func(tx *tx) error {
		if _, err := tx.exec(ctx, `INSERT INTO identity_connections(id,owner_id,generation,state,data,credential) VALUES(?,?,?,?,?,?)`, c.ID, c.OwnerID, c.Generation, c.State, data, s.grants.seal(credential, identityBinding(c))); err != nil {
			return err
		}
		return appendIdentityEvent(ctx, tx, identity.Event{Action: "connected", OwnerID: c.OwnerID, ActorID: c.OwnerID, ConnectionID: c.ID, Generation: c.Generation})
	})
}

func (s *Store) IdentityConnection(ctx context.Context, id string) (identity.Connection, error) {
	var c identity.Connection
	var data []byte
	err := s.queryRow(ctx, `SELECT data FROM identity_connections WHERE id=?`, id).Scan(&data)
	if err == nil {
		err = json.Unmarshal(data, &c)
	}
	return c, err
}

func (s *Store) IdentitySaveGrant(ctx context.Context, g identity.Grant) error {
	data, err := json.Marshal(g)
	if err != nil {
		return err
	}
	return s.identityAtomic(ctx, func(tx *tx) error {
		if _, err := tx.exec(ctx, `INSERT INTO identity_grants(id,connection_id,user_id,project_id,revision,revoked,data) VALUES(?,?,?,?,?,?,?)`, g.ID, g.ConnectionID, g.UserID, g.ProjectID, g.Revision, g.Revoked, data); err != nil {
			return err
		}
		return appendIdentityEvent(ctx, tx, identity.Event{Action: "grant_created", ConnectionID: g.ConnectionID, GrantID: g.ID, ProjectID: g.ProjectID})
	})
}

func (s *Store) IdentityGrant(ctx context.Context, id string) (identity.Grant, error) {
	var g identity.Grant
	var data []byte
	err := s.queryRow(ctx, `SELECT data FROM identity_grants WHERE id=?`, id).Scan(&data)
	if err == nil {
		err = json.Unmarshal(data, &g)
	}
	return g, err
}
