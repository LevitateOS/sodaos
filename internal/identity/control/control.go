// Package control owns subscription admission and credential custody.
package control

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sync"

	"github.com/levitateos/sodaos/internal/identity"
	"github.com/levitateos/sodaos/internal/store"
)

type Controller struct {
	mu          sync.Mutex
	store       *store.Store
	providers   map[string]identity.Provider
	runtime     identity.Runtime
	enrollments map[string]*enrollment
}
type enrollment struct {
	owner      int64
	label      string
	providerID string
	session    identity.EnrollmentSession
	result     identity.Enrollment
}

func New(s *store.Store, p map[string]identity.Provider, r identity.Runtime) (*Controller, error) {
	if s == nil || len(p) == 0 || r == nil {
		return nil, errors.New("identity custody dependencies required")
	}
	providers := make(map[string]identity.Provider, len(p))
	for id, provider := range p {
		if !identity.ProviderValid(id) || provider == nil {
			return nil, errors.New("invalid subscription provider")
		}
		providers[id] = provider
	}
	return &Controller{store: s, providers: providers, runtime: r, enrollments: map[string]*enrollment{}}, nil
}
func newID() string { var b [16]byte; _, _ = rand.Read(b[:]); return hex.EncodeToString(b[:]) }

func (c *Controller) owned(ctx context.Context, owner int64, id string) (identity.Connection, error) {
	conn, err := c.store.IdentityConnection(ctx, id)
	if err != nil {
		return conn, err
	}
	if owner <= 0 || conn.OwnerID != owner {
		return conn, identity.ErrDenied
	}
	return conn, nil
}

func (c *Controller) Connections(ctx context.Context, owner int64) ([]identity.Connection, error) {
	if owner <= 0 {
		return nil, identity.ErrDenied
	}
	return c.store.IdentityConnections(ctx, owner)
}

func (c *Controller) Available(ctx context.Context, actor int64, project string) ([]identity.Connection, error) {
	if actor <= 0 || project == "" {
		return nil, identity.ErrDenied
	}
	return c.store.IdentityAvailable(ctx, actor, project)
}

func (c *Controller) StartEnrollment(ctx context.Context, owner int64, providerID, label string) (identity.Enrollment, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.providers[providerID] == nil || owner <= 0 || len(label) > 100 || label == "" {
		return identity.Enrollment{}, identity.ErrDenied
	}
	if c.pendingEnrollment(owner, providerID) {
		return identity.Enrollment{}, identity.ErrBusy
	}
	s, err := c.providers[providerID].Start(ctx, owner)
	if err != nil {
		return identity.Enrollment{}, err
	}
	result := s.Snapshot()
	result.ProviderID = providerID
	if result.ID == "" {
		_ = s.Close()
		return identity.Enrollment{}, identity.ErrUncertain
	}
	c.enrollments[result.ID] = &enrollment{owner: owner, label: label, providerID: providerID, session: s, result: result}
	return result, nil
}

func (c *Controller) pendingEnrollment(owner int64, providerID string) bool {
	for _, e := range c.enrollments {
		if e.owner == owner && e.providerID == providerID && e.session != nil && e.result.Connection == nil && enrollmentUnretained(e.session) {
			return true
		}
	}
	return false
}

func enrollmentUnretained(session identity.EnrollmentSession) bool {
	state := session.Snapshot().State
	return state == "pending" || state == "completed"
}

func (c *Controller) Enrollment(ctx context.Context, owner int64, id string) (identity.Enrollment, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e := c.enrollments[id]
	if e == nil || e.owner != owner {
		return identity.Enrollment{}, identity.ErrDenied
	}
	if e.session == nil {
		return e.result, nil
	}
	e.result = e.session.Snapshot()
	e.result.ProviderID = e.providerID
	if e.result.State != "completed" {
		return e.result, nil
	}
	conn, data, err := e.session.Finish(ctx)
	if err != nil {
		e.result.State = "failed"
		e.result.Error = "Provider enrollment could not be retained"
		_ = e.session.Close()
		e.session = nil
		return e.result, err
	}
	conn.ProviderID = e.providerID
	conn.ID = newID()
	conn.OwnerID = owner
	conn.Label = e.label
	conn.Generation = 1
	conn.State = identity.Ready
	err = c.store.IdentitySaveConnection(ctx, conn, data)
	for i := range data {
		data[i] = 0
	}
	_ = e.session.Close()
	e.session = nil
	if err != nil {
		e.result.State = "failed"
		e.result.Error = "Provider enrollment could not be retained"
		return e.result, err
	}
	e.result.Connection = &conn
	e.result.UserCode = ""
	e.result.VerificationURL = ""
	return e.result, nil
}

func (c *Controller) CancelEnrollment(ctx context.Context, owner int64, id string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	e := c.enrollments[id]
	if e == nil || e.owner != owner {
		return identity.ErrDenied
	}
	if e.session != nil {
		if err := e.session.Close(); err != nil {
			return err
		}
		e.session = nil
	}
	e.result.State = "canceled"
	e.result.UserCode = ""
	return nil
}

func (c *Controller) Grants(ctx context.Context, owner int64, id string) ([]identity.Grant, error) {
	if _, err := c.owned(ctx, owner, id); err != nil {
		return nil, err
	}
	return c.store.IdentityGrants(ctx, id)
}

func (c *Controller) CreateGrant(ctx context.Context, owner int64, in identity.GrantRequest) (identity.Grant, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := in.Validate(); err != nil {
		return identity.Grant{}, err
	}
	conn, err := c.owned(ctx, owner, in.ConnectionID)
	if err != nil {
		return identity.Grant{}, err
	}
	if conn.State != identity.Ready {
		return identity.Grant{}, identity.ErrUncertain
	}
	g := identity.Grant{ID: newID(), ConnectionID: in.ConnectionID, UserID: in.UserID, ProjectID: in.ProjectID, Revision: 1}
	return g, c.store.IdentitySaveGrant(ctx, g)
}

func (c *Controller) Leases(ctx context.Context, owner int64, id string) ([]identity.Lease, error) {
	conn, err := c.store.IdentityConnection(ctx, id)
	if err != nil {
		return nil, err
	}
	if owner <= 0 {
		return nil, identity.ErrDenied
	}
	all, err := c.store.IdentityLeases(ctx)
	if err != nil {
		return nil, err
	}
	out := []identity.Lease{}
	for _, l := range all {
		if l.ConnectionID == id && (conn.OwnerID == owner || l.ActorID == owner) {
			l.Binding = nil
			out = append(out, l)
		}
	}
	return out, nil
}
