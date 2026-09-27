package store

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/levitateos/sodaos/internal/identity"
)

// IdentityRefresh retains a verified native renewal only after durable admission
// withdrawal. A crash or concurrent revocation cannot restore the previous seed.
func (s *Store) IdentityRefresh(ctx context.Context, c identity.Connection, credential []byte) error {
	if s.grants == nil {
		return ErrGrantKey
	}
	if c.ProviderID != identity.Forgejo || c.State != identity.Ready || !identity.CredentialValid(credential) {
		return identity.ErrDenied
	}
	data, err := json.Marshal(c)
	if err != nil {
		return err
	}
	return s.identityAtomic(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, `UPDATE identity_connections SET state='ready',data=?,credential=? WHERE id=? AND owner_id=? AND generation=? AND state='reauth' AND json_extract(data,'$.provider_id')=?`, data, s.grants.seal(credential, identityBinding(c)), c.ID, c.OwnerID, c.Generation, identity.Forgejo)
		if err = identityChanged(res, err); err != nil {
			return err
		}
		return appendIdentityEvent(ctx, tx, identity.Event{Action: "refreshed", OwnerID: c.OwnerID, ActorID: c.OwnerID, ConnectionID: c.ID, Generation: c.Generation})
	})
}
