package control

import (
	"context"
	"errors"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/identity"
)

func (c *Controller) acquireCredential(ctx context.Context, actor int64, r *factory.Run) error {
	request, err := c.credentialRequest(ctx, actor, r)
	if err != nil {
		return err
	}
	return c.reserveCredential(ctx, request, r)
}

func (c *Controller) credentialRequest(ctx context.Context, actor int64, r *factory.Run) (identity.AcquireRequest, error) {
	a, err := c.Store.FactoryAttempt(ctx, r.AttemptID)
	if err != nil {
		return identity.AcquireRequest{}, err
	}
	if a.Work.HumanID != actor || a.Work.RepositoryID <= 0 {
		return identity.AcquireRequest{}, identity.ErrDenied
	}
	return identity.AcquireRequest{ProviderID: identity.Codex, ActorID: actor, ConnectionID: c.Config.ConnectionID, ProjectID: c.Config.ProjectID, ExecutionID: r.ID, Kind: identity.Factory, Deadline: r.Deadline, Role: string(r.Role), RepositoryID: a.Work.RepositoryID}, nil
}

func (c *Controller) reserveCredential(ctx context.Context, request identity.AcquireRequest, r *factory.Run) error {
	if err := c.Workspace.CheckHarness(ctx); err != nil {
		return err
	}
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		lease, err := c.Identity.Acquire(ctx, request)
		if errors.Is(err, identity.ErrBusy) {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-ticker.C:
				continue
			}
		}
		if err != nil {
			return err
		}
		r.IdentityLeaseID, r.IdentityGeneration = lease.ID, lease.Generation
		return c.Store.SaveFactoryRun(ctx, *r)
	}
}

func (c *Controller) delegateCredential(ctx context.Context, r *factory.Run) ([]string, error) {
	var id string
	for _, resource := range r.Resources {
		if resource.Kind == "workspace" {
			id = resource.ID
		}
	}
	if id == "" || r.IdentityLeaseID == "" {
		return nil, errors.New("credential delivery requires a reserved native execution")
	}
	binding := identity.Binding{Kind: identity.Factory, ID: id, Project: c.Config.ProjectID, Generation: r.IdentityGeneration}
	// Record binding and delivery intent before the broker can release bytes.
	r.IdentityBinding, r.CredentialDelegated = &binding, true
	if err := c.Store.SaveFactoryRun(ctx, *r); err != nil {
		return nil, err
	}
	delivery, err := c.Identity.Register(ctx, r.IdentityLeaseID, binding)
	if err != nil {
		return nil, err
	}
	secrets, err := credentialStrings(delivery.Credential)
	if err != nil {
		return nil, err
	}
	return secrets, c.Workspace.SeedCredential(ctx, *r, delivery.Credential)
}

func (c *Controller) returnCredential(ctx context.Context, r *factory.Run) ([]string, error) {
	if r.IdentityBinding == nil {
		return nil, errors.New("credential return has no native binding")
	}
	state, err := c.Workspace.CaptureCredential(ctx, *r)
	if err != nil {
		return nil, err
	}
	if err = c.Identity.Return(ctx, r.IdentityLeaseID, *r.IdentityBinding, state); err != nil {
		return nil, err
	}
	r.CredentialReturned = true
	if err = c.Store.SaveFactoryRun(ctx, *r); err != nil {
		return nil, err
	}
	return credentialStrings(state)
}

func (c *Controller) returnDelegatedCredential(ctx context.Context, r *factory.Run) error {
	if r.IdentityLeaseID == "" || r.CredentialReturned {
		return nil
	}
	if r.CredentialDelegated {
		if _, err := c.returnCredential(ctx, r); err == nil {
			return nil
		}
	}
	// A failed capture or an interrupted delivery must never restore old state.
	return c.Identity.ReconcileLease(ctx, r.IdentityLeaseID)
}
