package control

import (
	"context"
	"errors"
	"time"

	"github.com/levitateos/sodaos/internal/identity"
	"github.com/levitateos/sodaos/internal/store"
)

func (c *Controller) Acquire(ctx context.Context, in identity.AcquireRequest) (identity.Lease, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := in.Validate(time.Now()); err != nil {
		return identity.Lease{}, err
	}
	conn, err := c.store.IdentityConnection(ctx, in.ConnectionID)
	if err != nil {
		return identity.Lease{}, err
	}
	if conn.State != identity.Ready {
		return identity.Lease{}, identity.ErrUncertain
	}
	l := identity.Lease{ID: newID(), ConnectionID: conn.ID, Generation: conn.Generation, ActorID: in.ActorID, ProjectID: in.ProjectID, ExecutionID: in.ExecutionID, Kind: in.Kind, Role: in.Role, Deadline: in.Deadline}
	if err = c.authorizeReservation(ctx, conn, &l); err != nil {
		return l, err
	}
	return l, c.store.IdentityReserve(ctx, l)
}

func (c *Controller) authorizeReservation(ctx context.Context, conn identity.Connection, l *identity.Lease) error {
	if conn.OwnerID == l.ActorID {
		return nil
	}
	if l.ProjectID == "" {
		return identity.ErrDenied
	}
	grants, err := c.store.IdentityGrants(ctx, conn.ID)
	if err != nil {
		return err
	}
	for _, g := range grants {
		if !g.Revoked && g.UserID == l.ActorID && g.ProjectID == l.ProjectID {
			l.GrantID = g.ID
			l.GrantRevision = g.Revision
			return nil
		}
	}
	return identity.ErrDenied
}

func (c *Controller) Register(ctx context.Context, id string, b identity.Binding) (identity.Delivery, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	l, conn, err := c.registration(ctx, id, b)
	if err != nil {
		return identity.Delivery{}, err
	}
	l.Binding = &b
	if err = c.store.IdentityRegister(ctx, l); err != nil {
		return identity.Delivery{}, err
	}
	if err = c.runtime.Validate(ctx, l); err != nil {
		_ = c.uncertain(ctx, l)
		return identity.Delivery{}, identity.ErrDenied
	}
	data, err := c.store.IdentityCredential(ctx, conn)
	if err != nil {
		return identity.Delivery{}, err
	}
	return identity.Delivery{Lease: l, Credential: data}, nil
}

func (c *Controller) registration(ctx context.Context, id string, b identity.Binding) (identity.Lease, identity.Connection, error) {
	l, err := c.store.IdentityLease(ctx, id)
	if err != nil {
		return l, identity.Connection{}, err
	}
	if err = b.Validate(); err != nil {
		return l, identity.Connection{}, err
	}
	if l.Binding != nil || b.Kind != l.Kind || b.Generation != l.Generation || !l.Deadline.After(time.Now()) {
		return l, identity.Connection{}, identity.ErrDenied
	}
	conn, err := c.registrationAuthority(ctx, l)
	return l, conn, err
}

func (c *Controller) registrationAuthority(ctx context.Context, l identity.Lease) (identity.Connection, error) {
	conn, err := c.store.IdentityConnection(ctx, l.ConnectionID)
	if err != nil {
		return conn, err
	}
	if conn.State != identity.Ready || conn.Generation != l.Generation {
		return conn, identity.ErrStale
	}
	if l.GrantID != "" {
		g, err := c.store.IdentityGrant(ctx, l.GrantID)
		if err != nil {
			return conn, err
		}
		if g.Revoked || g.Revision != l.GrantRevision {
			return conn, identity.ErrDenied
		}
	}
	return conn, nil
}

func (c *Controller) Return(ctx context.Context, id string, b identity.Binding, data []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	l, err := c.store.IdentityLease(ctx, id)
	if err != nil {
		return err
	}
	if l.Binding == nil || *l.Binding != b {
		return identity.ErrDenied
	}
	// Trusted native termination is mandatory even when the caller reports exit.
	if err = c.runtime.Stop(ctx, l); err != nil {
		_ = c.uncertain(ctx, l)
		return identity.ErrUncertain
	}
	if !identity.CredentialValid(data) {
		_ = c.uncertain(ctx, l)
		return identity.ErrUncertain
	}
	return c.store.IdentityReturn(ctx, l, data)
}

func (c *Controller) uncertain(ctx context.Context, l identity.Lease) error {
	conn, err := c.store.IdentityConnection(ctx, l.ConnectionID)
	if err != nil {
		return err
	}
	if conn.State == identity.Revoked {
		return nil
	}
	return c.store.IdentityState(ctx, conn, identity.Reauth)
}

func (c *Controller) end(ctx context.Context, l identity.Lease) error {
	if l.Binding == nil {
		return c.store.IdentityForgetLease(ctx, l.ID)
	}
	if err := c.uncertain(ctx, l); err != nil {
		return err
	}
	if l.Binding != nil {
		if err := c.runtime.Stop(ctx, l); err != nil {
			return identity.ErrUncertain
		}
	}
	return c.store.IdentityForgetLease(ctx, l.ID)
}

func (c *Controller) EndLease(ctx context.Context, owner int64, id string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	l, err := c.store.IdentityLease(ctx, id)
	if err != nil {
		return err
	}
	if l.ActorID != owner {
		if _, err = c.owned(ctx, owner, l.ConnectionID); err != nil {
			return err
		}
	}
	return c.finish(ctx, l)
}

func (c *Controller) finish(ctx context.Context, l identity.Lease) error {
	if l.Binding == nil {
		return c.store.IdentityForgetLease(ctx, l.ID)
	}
	data, err := c.runtime.Finish(ctx, l)
	if err != nil || !identity.CredentialValid(data) {
		_ = c.uncertain(ctx, l)
		return identity.ErrUncertain
	}
	defer func() {
		for i := range data {
			data[i] = 0
		}
	}()
	if err = c.store.IdentityReturn(ctx, l, data); err != nil {
		_ = c.uncertain(ctx, l)
	}
	return err
}

// ReconcileLease is scoped recovery, never cancellation of another execution.
func (c *Controller) ReconcileLease(ctx context.Context, id string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	l, err := c.store.IdentityLease(ctx, id)
	if errors.Is(err, store.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	return c.end(ctx, l)
}

func (c *Controller) Revoke(ctx context.Context, owner int64, id string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	conn, err := c.owned(ctx, owner, id)
	if err != nil {
		return err
	}
	if err = c.store.IdentityState(ctx, conn, identity.Revoked); err != nil {
		return err
	}
	all, err := c.store.IdentityLeases(ctx)
	if err != nil {
		return err
	}
	for _, l := range all {
		if l.ConnectionID == id {
			if err = c.end(ctx, l); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *Controller) RevokeGrant(ctx context.Context, owner int64, id string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	g, err := c.store.IdentityGrant(ctx, id)
	if err != nil {
		return err
	}
	if _, err = c.owned(ctx, owner, g.ConnectionID); err != nil {
		return err
	}
	if !g.Revoked {
		if err = c.store.IdentityRevokeGrant(ctx, g); err != nil {
			return err
		}
	}
	all, err := c.store.IdentityLeases(ctx)
	if err != nil {
		return err
	}
	for _, l := range all {
		if l.GrantID == id {
			if err = c.end(ctx, l); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *Controller) Reconcile(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	all, err := c.store.IdentityLeases(ctx)
	if err != nil {
		return err
	}
	var failures []error
	for _, l := range all {
		if err = c.end(ctx, l); err != nil {
			failures = append(failures, err)
		}
	}
	return errors.Join(failures...)
}

func (c *Controller) Sweep(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	all, err := c.store.IdentityLeases(ctx)
	if err != nil {
		return err
	}
	var failures []error
	for _, l := range all {
		if err = c.sweepLease(ctx, l); err != nil {
			failures = append(failures, err)
		}
	}
	return errors.Join(failures...)
}

func (c *Controller) sweepLease(ctx context.Context, l identity.Lease) error {
	conn, err := c.store.IdentityConnection(ctx, l.ConnectionID)
	if err != nil {
		return err
	}
	if conn.State != identity.Ready {
		return c.end(ctx, l)
	}
	if !l.Deadline.After(time.Now()) {
		return c.finish(ctx, l)
	}
	return nil
}

func (c *Controller) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	var failures []error
	for _, e := range c.enrollments {
		if e.session != nil {
			if err := e.session.Close(); err != nil {
				failures = append(failures, err)
			}
		}
	}
	return errors.Join(failures...)
}
