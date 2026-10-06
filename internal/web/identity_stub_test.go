package web

import (
	"context"
	"errors"

	"github.com/levitateos/sodaos/internal/identity"
)

// errStubIdentity injects a generic broker call failure.
var errStubIdentity = errors.New("stub identity unavailable")

// stubIdentityClient is an in-memory api.IdentityClient for tests. It models
// the broker's owner-scoped metadata semantics: Connections lists only the
// caller's connections, and Grants enforces connection ownership before
// listing that connection's grants. connErr/grantsErr inject call failures.
type stubIdentityClient struct {
	connections []identity.Connection
	grants      []identity.Grant
	connErr     error
	grantsErr   error
}

func (s *stubIdentityClient) Connections(_ context.Context, owner int64) ([]identity.Connection, error) {
	if s.connErr != nil {
		return nil, s.connErr
	}
	if owner <= 0 {
		return nil, identity.ErrDenied
	}
	var out []identity.Connection
	for _, c := range s.connections {
		if c.OwnerID == owner {
			out = append(out, c)
		}
	}
	return out, nil
}

func (s *stubIdentityClient) Grants(_ context.Context, owner int64, id string) ([]identity.Grant, error) {
	if s.grantsErr != nil {
		return nil, s.grantsErr
	}
	var connection *identity.Connection
	for i := range s.connections {
		if s.connections[i].ID == id {
			connection = &s.connections[i]
			break
		}
	}
	if connection == nil {
		return nil, identity.ErrNotFound
	}
	if owner <= 0 || connection.OwnerID != owner {
		return nil, identity.ErrDenied
	}
	var out []identity.Grant
	for _, g := range s.grants {
		if g.ConnectionID == id {
			out = append(out, g)
		}
	}
	return out, nil
}

func (s *stubIdentityClient) Available(_ context.Context, _ int64, _ string) ([]identity.Connection, error) {
	return nil, identity.ErrDenied
}

func (s *stubIdentityClient) StartEnrollment(_ context.Context, _ int64, _, _ string) (identity.Enrollment, error) {
	return identity.Enrollment{}, identity.ErrDenied
}

func (s *stubIdentityClient) Enrollment(_ context.Context, _ int64, _ string) (identity.Enrollment, error) {
	return identity.Enrollment{}, identity.ErrDenied
}

func (s *stubIdentityClient) CancelEnrollment(_ context.Context, _ int64, _ string) error {
	return identity.ErrDenied
}

func (s *stubIdentityClient) CreateGrant(_ context.Context, _ int64, _ identity.GrantRequest) (identity.Grant, error) {
	return identity.Grant{}, identity.ErrDenied
}

func (s *stubIdentityClient) RevokeGrant(_ context.Context, _ int64, _ string) error {
	return identity.ErrDenied
}

func (s *stubIdentityClient) Leases(_ context.Context, _ int64, _ string) ([]identity.Lease, error) {
	return nil, identity.ErrDenied
}

func (s *stubIdentityClient) EndLease(_ context.Context, _ int64, _ string) error {
	return identity.ErrDenied
}

func (s *stubIdentityClient) Revoke(_ context.Context, _ int64, _ string) error {
	return identity.ErrDenied
}
