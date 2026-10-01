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
	digest := identity.AcquisitionDigest(in)
	existing, created, err := c.admitExecution(ctx, in, digest)
	if err != nil {
		return identity.Lease{}, err
	}
	if !created && existing.LeaseID != "" {
		return c.replayAcquisition(ctx, existing, digest)
	}
	// A fresh identity, or a pending identity whose earlier reservation
	// never landed, proceeds to reserve exactly one lease for this digest.
	l, err := c.reserveExecutionLease(ctx, in)
	if err != nil {
		return l, err
	}
	existing.State, existing.LeaseID = identity.ExecutionLive, l.ID
	if err = c.store.IdentityObserveExecution(ctx, existing); err != nil {
		return identity.Lease{}, err
	}
	return l, nil
}

// admitExecution records the (kind, execution_id) identity before any lease
// work. A terminal execution or a changed request for a live identity refuses;
// the same digest replays the recorded lease instead of reserving another.
func (c *Controller) admitExecution(ctx context.Context, in identity.AcquireRequest, digest string) (identity.Execution, bool, error) {
	e := identity.Execution{Kind: in.Kind, ExecutionID: in.ExecutionID, Digest: digest, State: identity.ExecutionPending}
	existing, created, err := c.store.IdentityAdmitExecution(ctx, e)
	if err != nil {
		return identity.Execution{}, false, err
	}
	if !created && (existing.State == identity.ExecutionTerminal || existing.Digest != digest) {
		return identity.Execution{}, false, identity.ErrDenied
	}
	return existing, created, nil
}

func (c *Controller) replayAcquisition(ctx context.Context, existing identity.Execution, digest string) (identity.Lease, error) {
	if existing.Digest != digest || existing.State == identity.ExecutionTerminal {
		return identity.Lease{}, identity.ErrDenied
	}
	if existing.LeaseID == "" {
		return identity.Lease{}, identity.ErrUncertain
	}
	l, err := c.store.IdentityLease(ctx, existing.LeaseID)
	if err != nil {
		return identity.Lease{}, identity.ErrUncertain
	}
	// Re-check current reservation authority: revocation or withdrawal
	// after the original acquire must not hand the lease out again.
	conn, err := c.store.IdentityConnection(ctx, l.ConnectionID)
	if err != nil {
		return identity.Lease{}, identity.ErrUncertain
	}
	if conn.ProviderID != l.ProviderID {
		return identity.Lease{}, identity.ErrDenied
	}
	if conn.State != identity.Ready || c.providers[conn.ProviderID] == nil {
		return identity.Lease{}, identity.ErrUncertain
	}
	if err = c.authorizeReservation(ctx, conn, &l); err != nil {
		return identity.Lease{}, err
	}
	return l, nil
}

func (c *Controller) reserveExecutionLease(ctx context.Context, in identity.AcquireRequest) (identity.Lease, error) {
	conn, err := c.store.IdentityConnection(ctx, in.ConnectionID)
	if err != nil {
		return identity.Lease{}, err
	}
	if conn.ProviderID != in.ProviderID {
		return identity.Lease{}, identity.ErrDenied
	}
	if conn.State != identity.Ready || c.providers[conn.ProviderID] == nil {
		return identity.Lease{}, identity.ErrUncertain
	}
	l := identity.Lease{ID: newID(), ProviderID: conn.ProviderID, ConnectionID: conn.ID, Generation: conn.Generation, ActorID: in.ActorID, ProjectID: in.ProjectID, ExecutionID: in.ExecutionID, Kind: in.Kind, Role: in.Role, Deadline: in.Deadline, RepositoryID: in.RepositoryID}
	if err = c.authorizeReservation(ctx, conn, &l); err != nil {
		return l, err
	}
	return l, c.store.IdentityReserve(ctx, l)
}

// GetExecution returns the immutable acquisition digest, lease/binding
// metadata and pending/live/terminal state without credentials.
func (c *Controller) GetExecution(ctx context.Context, kind, executionID string) (identity.Execution, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if (kind != identity.Factory && kind != identity.Terminal) || executionID == "" {
		return identity.Execution{}, identity.ErrDenied
	}
	e, err := c.store.IdentityExecution(ctx, kind, executionID)
	if errors.Is(err, store.ErrNotFound) {
		return identity.Execution{}, identity.ErrNotFound
	}
	return e, err
}

// CloseExecution creates a terminal acquisition tombstone even before
// acquisition arrives, prevents subsequent acquisition/registration and
// reconciles any existing lease. It reports uncertain until custody and
// process retirement are established; a consumed start marker followed by an
// absent unit stays fenced rather than treated as completion.
func (c *Controller) CloseExecution(ctx context.Context, kind, executionID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if (kind != identity.Factory && kind != identity.Terminal) || executionID == "" {
		return identity.ErrDenied
	}
	existing, created, err := c.store.IdentityAdmitExecution(ctx, identity.Execution{Kind: kind, ExecutionID: executionID, State: identity.ExecutionTerminal})
	if err != nil {
		return err
	}
	if created {
		return nil
	}
	// A terminal execution with a surviving lease still needs retirement;
	// closing twice must not report success while native work continues.
	// The last observed binding stays retained for attribution.
	if existing.LeaseID != "" {
		if l, err := c.store.IdentityLease(ctx, existing.LeaseID); err == nil {
			if l.Binding != nil {
				existing.Binding = l.Binding
			}
			if err = c.end(ctx, l); err != nil {
				existing.State = identity.ExecutionTerminal
				_ = c.store.IdentityObserveExecution(ctx, existing)
				return identity.ErrUncertain
			}
		}
		existing.LeaseID = ""
	}
	existing.State = identity.ExecutionTerminal
	return c.store.IdentityObserveExecution(ctx, existing)
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
	if err = c.registrationExecution(ctx, l); err != nil {
		return identity.Delivery{}, err
	}
	l.Binding = &b
	if err = c.store.IdentityRegister(ctx, l); err != nil {
		return identity.Delivery{}, err
	}
	if err = c.observeExecutionBinding(ctx, l); err != nil {
		_ = c.uncertain(ctx, l)
		return identity.Delivery{}, identity.ErrUncertain
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

// registrationExecution refuses a lease whose execution was closed after
// acquisition. A closed execution whose lease survived reconciliation stays
// fenced; it can never deliver credentials again.
func (c *Controller) registrationExecution(ctx context.Context, l identity.Lease) error {
	e, err := c.store.IdentityExecution(ctx, l.Kind, l.ExecutionID)
	if err != nil {
		return identity.ErrDenied
	}
	if e.State == identity.ExecutionTerminal || e.LeaseID != l.ID {
		return identity.ErrDenied
	}
	return nil
}

func (c *Controller) observeExecutionBinding(ctx context.Context, l identity.Lease) error {
	e, err := c.store.IdentityExecution(ctx, l.Kind, l.ExecutionID)
	if err != nil {
		return err
	}
	e.Binding = l.Binding
	return c.store.IdentityObserveExecution(ctx, e)
}

// observeTerminal retains the nonsecret acquisition identity after its lease
// is gone. A missing execution is ignored; only admitted leases reach it.
func (c *Controller) observeTerminal(ctx context.Context, l identity.Lease) error {
	e, err := c.store.IdentityExecution(ctx, l.Kind, l.ExecutionID)
	if errors.Is(err, store.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	e.State = identity.ExecutionTerminal
	return c.store.IdentityObserveExecution(ctx, e)
}

// releaseExecutionLease detaches a reconciled lease from its execution
// without tombstoning it: recovery may retry the same identity, gated by
// current connection authority and native re-attestation. Only Return and
// CloseExecution make an execution terminal. A tombstone keeps its state.
func (c *Controller) releaseExecutionLease(ctx context.Context, l identity.Lease) error {
	e, err := c.store.IdentityExecution(ctx, l.Kind, l.ExecutionID)
	if errors.Is(err, store.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if e.LeaseID != "" && e.LeaseID != l.ID {
		return nil
	}
	e.LeaseID, e.Binding = "", nil
	if e.State == identity.ExecutionLive {
		e.State = identity.ExecutionPending
	}
	return c.store.IdentityObserveExecution(ctx, e)
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
	if conn.State != identity.Ready || conn.Generation != l.Generation || conn.ProviderID != l.ProviderID {
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
	if l.ProviderID == identity.Muse {
		if err := c.store.IdentityForgetLease(ctx, l.ID); err != nil {
			return err
		}
		return c.observeTerminal(ctx, l)
	}
	if !identity.CredentialValid(data) {
		_ = c.uncertain(ctx, l)
		return identity.ErrUncertain
	}
	if err := c.store.IdentityReturn(ctx, l, data); err != nil {
		return err
	}
	return c.observeTerminal(ctx, l)
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
		if err := c.store.IdentityForgetLease(ctx, l.ID); err != nil {
			return err
		}
		return c.releaseExecutionLease(ctx, l)
	}
	if l.ProviderID != identity.Muse {
		if err := c.uncertain(ctx, l); err != nil {
			return err
		}
	}
	if l.Binding != nil {
		if err := c.runtime.Stop(ctx, l); err != nil {
			_ = c.uncertain(ctx, l)
			return identity.ErrUncertain
		}
	}
	if err := c.store.IdentityForgetLease(ctx, l.ID); err != nil {
		return err
	}
	return c.releaseExecutionLease(ctx, l)
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
		if err := c.store.IdentityForgetLease(ctx, l.ID); err != nil {
			return err
		}
		return c.releaseExecutionLease(ctx, l)
	}
	if l.ProviderID == identity.Muse {
		return c.end(ctx, l)
	}
	data, captureErr := c.runtime.Finish(ctx, l)
	defer func() {
		for i := range data {
			data[i] = 0
		}
	}()
	// Capture failure does not establish retirement; always attempt native stop.
	stopErr := c.runtime.Stop(ctx, l)
	if captureErr != nil || !identity.CredentialValid(data) || stopErr != nil {
		_ = c.uncertain(ctx, l)
		return identity.ErrUncertain
	}
	if err := c.store.IdentityReturn(ctx, l, data); err != nil {
		_ = c.uncertain(ctx, l)
		return err
	}
	return c.releaseExecutionLease(ctx, l)
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
	var failures []error
	for _, l := range all {
		if l.ConnectionID == id {
			if err = c.end(ctx, l); err != nil {
				failures = append(failures, err)
			}
		}
	}
	return errors.Join(failures...)
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
	var failures []error
	for _, l := range all {
		if l.GrantID == id {
			if err = c.finish(ctx, l); err != nil {
				failures = append(failures, err)
			}
		}
	}
	return errors.Join(failures...)
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
