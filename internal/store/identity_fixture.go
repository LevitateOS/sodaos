package store

import (
	"context"
	"encoding/json"

	"github.com/levitateos/sodaos/internal/identity"
)

// Test-only identity seed helpers. Production code must never call these:
// they exist so fixture setup across test packages shares one purpose-scoped
// owner instead of calling the production save path directly. They mirror
// the IdentitySaveConnection/IdentitySaveGrant logic exactly (validation,
// seal binding, transaction, credential-free event append); any semantic
// change must land in identity.go/identity_events.go first and be mirrored
// here. They live here because raw SQL is allowed only in internal/store.

// seedIdentityInsert is the single owner of fixture insert semantics: one
// transactional insert plus one credential-free audit event. Both Seed
// helpers funnel through it; the seal binding stays owned by identityBinding
// in identity.go and is reused, never redefined.
func (s *Store) seedIdentityInsert(ctx context.Context, insert func(*tx) error, event identity.Event) error {
	return s.identityAtomic(ctx, func(tx *tx) error {
		if err := insert(tx); err != nil {
			return err
		}
		return appendIdentityEvent(ctx, tx, event)
	})
}

// SeedIdentityConnection inserts one identity connection with its sealed
// credential and "connected" audit event, mirroring IdentitySaveConnection.
func (s *Store) SeedIdentityConnection(ctx context.Context, c identity.Connection, credential []byte) error {
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
	return s.seedIdentityInsert(ctx, func(tx *tx) error {
		if _, err := tx.exec(ctx, `INSERT INTO identity_connections(id,owner_id,generation,state,data,credential) VALUES(?,?,?,?,?,?)`, c.ID, c.OwnerID, c.Generation, c.State, data, s.grants.seal(credential, identityBinding(c))); err != nil {
			return err
		}
		return nil
	}, identity.Event{Action: "connected", OwnerID: c.OwnerID, ActorID: c.OwnerID, ConnectionID: c.ID, Generation: c.Generation})
}

// SeedIdentityGrant inserts one identity grant with its "grant_created"
// audit event, mirroring IdentitySaveGrant.
func (s *Store) SeedIdentityGrant(ctx context.Context, g identity.Grant) error {
	data, err := json.Marshal(g)
	if err != nil {
		return err
	}
	return s.seedIdentityInsert(ctx, func(tx *tx) error {
		if _, err := tx.exec(ctx, `INSERT INTO identity_grants(id,connection_id,user_id,project_id,revision,revoked,data) VALUES(?,?,?,?,?,?,?)`, g.ID, g.ConnectionID, g.UserID, g.ProjectID, g.Revision, g.Revoked, data); err != nil {
			return err
		}
		return nil
	}, identity.Event{Action: "grant_created", ConnectionID: g.ConnectionID, GrantID: g.ID, ProjectID: g.ProjectID})
}
