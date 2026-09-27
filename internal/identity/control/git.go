package control

import (
	"context"
	"net/http"
	"time"

	"github.com/levitateos/sodaos/internal/forgejo"
	"github.com/levitateos/sodaos/internal/identity"
)

type gitProvider interface {
	ResolveGitRepository(context.Context, int64, []byte, string, string) (forgejo.Repository, error)
	GitAuthority(context.Context, int64, []byte, int64, bool) (forgejo.Repository, forgejo.User, string, error)
	ProxyGit(http.ResponseWriter, *http.Request, int64, []byte, forgejo.Repository, forgejo.User) bool
}

func (c *Controller) AcquireGit(ctx context.Context, in identity.GitAcquireRequest) (identity.Lease, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	a := in.Acquire
	if !validGitAcquire(in) {
		return identity.Lease{}, identity.ErrDenied
	}
	p, ok := c.providers[identity.Forgejo].(gitProvider)
	if !ok {
		return identity.Lease{}, identity.ErrDenied
	}
	conn, data, err := c.forgejoCredentialLocked(ctx, a.ActorID, a.ConnectionID)
	defer clear(data)
	if err != nil {
		return identity.Lease{}, err
	}
	repo, err := p.ResolveGitRepository(ctx, a.ActorID, data, in.Owner, in.Repository)
	if err != nil || (in.ExpectedRepositoryID > 0 && repo.ID != in.ExpectedRepositoryID) {
		return identity.Lease{}, identity.ErrDenied
	}
	if _, _, _, err = p.GitAuthority(ctx, a.ActorID, data, repo.ID, false); err != nil {
		return identity.Lease{}, identity.ErrDenied
	}
	l := identity.Lease{ID: newID(), ProviderID: identity.Forgejo, ConnectionID: conn.ID, Generation: conn.Generation, ActorID: a.ActorID, ProjectID: a.ProjectID, ExecutionID: a.ExecutionID, Kind: a.Kind, Role: a.Role, Deadline: a.Deadline, RepositoryID: repo.ID}
	return l, c.store.IdentityReserve(ctx, l)
}

func (c *Controller) RegisterGit(ctx context.Context, id string, b identity.Binding) (identity.GitSession, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	l, conn, err := c.registration(ctx, id, b)
	if err != nil {
		return identity.GitSession{}, err
	}
	if !validGitRegistration(l, conn, b) {
		return identity.GitSession{}, identity.ErrDenied
	}
	p, ok := c.providers[identity.Forgejo].(gitProvider)
	if !ok {
		return identity.GitSession{}, identity.ErrDenied
	}
	_, data, err := c.forgejoCredentialLocked(ctx, l.ActorID, l.ConnectionID)
	defer clear(data)
	if err != nil {
		return identity.GitSession{}, err
	}
	_, user, email, err := p.GitAuthority(ctx, l.ActorID, data, l.RepositoryID, false)
	if err != nil {
		return identity.GitSession{}, identity.ErrDenied
	}
	l.Binding = &b
	if err := c.registerGitNative(ctx, l); err != nil {
		return identity.GitSession{}, err
	}
	name := user.Name
	if name == "" {
		name = user.Login
	}
	return identity.GitSession{Lease: l, Name: name, Email: email}, nil
}

func (c *Controller) registerGitNative(ctx context.Context, l identity.Lease) error {
	if err := c.store.IdentityRegister(ctx, l); err != nil {
		return err
	}
	if err := c.runtime.Validate(ctx, l); err != nil {
		_ = c.uncertain(ctx, l)
		return identity.ErrDenied
	}
	return nil
}

func validGitRegistration(l identity.Lease, conn identity.Connection, b identity.Binding) bool {
	return l.ProviderID == identity.Forgejo && l.RepositoryID > 0 && l.ActorID == conn.OwnerID && b.Project == l.ProjectID && b.Scope == "git"
}

func validGitAcquire(in identity.GitAcquireRequest) bool {
	a := in.Acquire
	return a.Validate(time.Now()) == nil && a.ProviderID == identity.Forgejo && a.ProjectID != "" && (a.Kind != identity.Factory || in.ExpectedRepositoryID > 0)
}
