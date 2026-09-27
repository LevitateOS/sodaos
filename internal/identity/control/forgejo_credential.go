package control

import (
	"context"
	"time"

	"github.com/levitateos/sodaos/internal/identity"
)

const forgejoRenewalMargin = time.Minute

// forgejoCredential is broker-internal custody for mediated Git. The caller must
// clear returned bytes after use; no administration or runtime route exposes it.
func (c *Controller) forgejoCredential(ctx context.Context, owner int64, id string) (identity.Connection, []byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.forgejoCredentialLocked(ctx, owner, id)
}

func (c *Controller) forgejoCredentialLocked(ctx context.Context, owner int64, id string) (identity.Connection, []byte, error) {
	conn, err := c.owned(ctx, owner, id)
	if err != nil {
		return identity.Connection{}, nil, err
	}
	if conn.ProviderID != identity.Forgejo || conn.State != identity.Ready {
		return identity.Connection{}, nil, identity.ErrDenied
	}
	refresher, ok := c.providers[identity.Forgejo].(identity.CredentialRefresher)
	if !ok {
		return identity.Connection{}, nil, identity.ErrDenied
	}
	data, err := c.store.IdentityCredential(ctx, conn)
	if err != nil {
		return identity.Connection{}, nil, err
	}
	expiry, err := refresher.CredentialExpiry(owner, data)
	if err != nil {
		clear(data)
		return identity.Connection{}, nil, identity.ErrDenied
	}
	if expiry.After(time.Now().Add(forgejoRenewalMargin)) {
		return conn, data, nil
	}
	defer clear(data)
	if err := c.store.IdentityState(ctx, conn, identity.Reauth); err != nil {
		return identity.Connection{}, nil, err
	}
	return c.renewForgejoCredential(ctx, conn, refresher, data)
}

func validForgejoRenewal(conn identity.Connection, owner int64) bool {
	return conn.OwnerID == owner && conn.ProviderID == identity.Forgejo && conn.State == identity.Ready && conn.Email != "" && conn.Plan == ""
}

func (c *Controller) renewForgejoCredential(ctx context.Context, conn identity.Connection, refresher identity.CredentialRefresher, old []byte) (identity.Connection, []byte, error) {
	renewed, data, err := refresher.Refresh(ctx, conn.OwnerID, old)
	defer clear(data)
	if err != nil || !validForgejoRenewal(renewed, conn.OwnerID) {
		return identity.Connection{}, nil, identity.ErrUncertain
	}
	expiry, err := refresher.CredentialExpiry(conn.OwnerID, data)
	if err != nil || !expiry.After(time.Now().Add(forgejoRenewalMargin)) {
		return identity.Connection{}, nil, identity.ErrUncertain
	}
	conn.Email = renewed.Email
	if err := c.store.IdentityRefresh(ctx, conn, data); err != nil {
		return identity.Connection{}, nil, identity.ErrUncertain
	}
	retained, err := c.store.IdentityCredential(ctx, conn)
	if err != nil {
		return identity.Connection{}, nil, identity.ErrUncertain
	}
	return conn, retained, nil
}
