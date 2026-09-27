package control

import (
	"context"
	"errors"

	"github.com/levitateos/sodaos/internal/identity"
)

// Reject denies reuse after native subscription rejection, then retires every copy.
func (c *Controller) Reject(ctx context.Context, id string, b identity.Binding) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	l, err := c.store.IdentityLease(ctx, id)
	if err != nil {
		return err
	}
	if l.ProviderID != identity.Muse || l.Binding == nil || *l.Binding != b {
		return identity.ErrDenied
	}
	if err = c.uncertain(ctx, l); err != nil {
		return err
	}
	return c.retireConnection(ctx, l.ConnectionID)
}

func (c *Controller) retireConnection(ctx context.Context, id string) error {
	all, err := c.store.IdentityLeases(ctx)
	if err != nil {
		return err
	}
	var failures []error
	for _, other := range all {
		if other.ConnectionID == id {
			if err := c.end(ctx, other); err != nil {
				failures = append(failures, err)
			}
		}
	}
	return errors.Join(failures...)
}
