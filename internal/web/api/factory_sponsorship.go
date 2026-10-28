package api

import (
	"errors"
	"net/http"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/identity"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
	"github.com/levitateos/sodaos/internal/web/auth"
)

// sponsorshipView redacts the private allowance for viewers other than the
// sponsoring connection owner. References stay; spending details do not.
func sponsorshipView(sp factory.Sponsorship, owner bool) factory.Sponsorship {
	if owner {
		return sp
	}
	sp.AllowanceMinutes = 0
	return sp
}

// factorySponsorshipRequest carries only what the connection owner decides:
// which roles may use the connection, the allowance and whether it is on.
// The server picks the broker grant and reads the credential generation.
type factorySponsorshipRequest struct {
	CommandID        string   `json:"command_id"`
	ExpectedRevision int64    `json:"expected_revision"`
	Roles            []string `json:"roles"`
	AllowanceMinutes int      `json:"allowance_minutes"`
	Active           bool     `json:"active"`
}

func (s *API) apiFactorySponsorship(w http.ResponseWriter, r *http.Request, v store.Session) {
	repository, ok := s.factoryRepository(w, r, v)
	if !ok || !s.checkLifecycleSession(w, r, v) {
		return
	}
	connection := r.PathValue("connection")
	var in factorySponsorshipRequest
	if !auth.DecodeAPIObject(w, r, &in) || !settingsCommandID(w, in.CommandID) {
		return
	}
	sponsorship := factory.Sponsorship{
		Repository: repository, Revision: in.ExpectedRevision, GrantedBy: v.User.ID,
		Connection: connection, Roles: in.Roles, AllowanceMinutes: in.AllowanceMinutes, Active: in.Active,
	}
	if sponsorship.Revision < 0 {
		auth.JSONError(w, 400, "invalid_sponsorship", "Sponsorship is incomplete or invalid.")
		return
	}
	if !s.resolveSponsorshipBroker(w, r, v, &sponsorship) {
		return
	}
	if sponsorship.Validate() != nil {
		auth.JSONError(w, 400, "invalid_sponsorship", "Sponsorship is incomplete or invalid.")
		return
	}
	receipt, err := s.Coordinator.ApplySponsorship(r.Context(), in.CommandID, factoryPrincipal(v), sponsorship)
	if err != nil {
		factoryCommandError(w, err)
		return
	}
	auth.JSONResponse(w, 200, receipt)
}

// resolveSponsorshipBroker fills the sponsorship's broker references from
// the identity broker: the caller must own the connection, its current
// credential generation is recorded, and the grant is the live grant that
// gives the repository's Project this connection. A grant the sponsorship
// already records is kept while it stays live; otherwise the newest live
// grant is used. Credential custody stays with the broker.
func (s *API) resolveSponsorshipBroker(w http.ResponseWriter, r *http.Request, v store.Session, sponsorship *factory.Sponsorship) bool {
	if s.Identity == nil {
		auth.JSONError(w, 503, "identity_unavailable", "Identity service is unavailable.")
		return false
	}
	connections, err := s.Identity.Connections(r.Context(), v.User.ID)
	if err != nil {
		if errors.Is(err, identity.ErrDenied) {
			auth.JSONError(w, 403, "connection_owner_required", "Only the connection owner can sponsor factory use.")
			return false
		}
		auth.JSONError(w, 503, "store_unavailable", "Could not read provider connection.")
		return false
	}
	var connection identity.Connection
	found := false
	for _, c := range connections {
		if c.ID == sponsorship.Connection {
			connection, found = c, true
			break
		}
	}
	if !found {
		auth.JSONError(w, 404, "not_found", "Provider connection not found.")
		return false
	}
	if connection.OwnerID != v.User.ID {
		auth.JSONError(w, 403, "connection_owner_required", "Only the connection owner can sponsor factory use.")
		return false
	}
	projectRecord, err := s.Store.ProjectByRepository(r.Context(), sponsorship.Repository)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			auth.JSONError(w, 409, "grant_unavailable", "Create this repository's Project before sponsoring Factory use.")
			return false
		}
		auth.JSONError(w, 503, "store_unavailable", "Could not read the repository's Project.")
		return false
	}
	grants, err := s.Identity.Grants(r.Context(), v.User.ID, sponsorship.Connection)
	if err != nil {
		if errors.Is(err, identity.ErrDenied) {
			auth.JSONError(w, 403, "connection_owner_required", "Only the connection owner can sponsor factory use.")
			return false
		}
		if !errors.Is(err, identity.ErrNotFound) {
			auth.JSONError(w, 503, "store_unavailable", "Could not read broker grant.")
			return false
		}
		grants = nil
	}
	previous := ""
	if stored, err := s.Store.Sponsorship(r.Context(), sponsorship.Repository, sponsorship.Connection); err == nil {
		previous = stored.GrantID
	}
	var grant identity.Grant
	for _, g := range grants {
		if g.ConnectionID != sponsorship.Connection || g.Revoked || g.ProjectID != projectRecord.ID || g.UserID <= 0 {
			continue
		}
		if g.ID == previous {
			grant = g
			break
		}
		if grant.ID == "" || g.Revision > grant.Revision {
			grant = g
		}
	}
	if grant.ID == "" || !project.ValidID(grant.ProjectID) {
		auth.JSONError(w, 409, "grant_unavailable", "Give this repository's Project access to the connection first.")
		return false
	}
	sponsorship.GrantID, sponsorship.Generation = grant.ID, connection.Generation
	sponsorship.ActorID, sponsorship.ProjectID = grant.UserID, grant.ProjectID
	return true
}

type factoryEnvironmentGrantRequest struct {
	Profile          *project.Profile `json:"profile"`
	CommandID        string           `json:"command_id"`
	Project          string           `json:"project,omitempty"`
	ExpectedRevision int64            `json:"expected_revision"`
	Active           bool             `json:"active"`
}

func (s *API) apiFactoryEnvironmentGrant(w http.ResponseWriter, r *http.Request, v store.Session) {
	repository, ok := s.factoryRepository(w, r, v)
	if !ok {
		return
	}
	access, err := s.visibleRepository(r, v, repository)
	if err != nil {
		auth.ProviderError(w, err)
		return
	}
	owner, err := s.environmentAdministrator(r, access)
	if err != nil {
		auth.ProviderError(w, err)
		return
	}
	if !owner || !s.checkLifecycleSession(w, r, v) {
		if !owner {
			auth.JSONError(w, 403, "repository_owner_required", "Only the current repository owner can permit factory environment setup.")
		}
		return
	}
	var in factoryEnvironmentGrantRequest
	if !auth.DecodeAPIObject(w, r, &in) || !settingsCommandID(w, in.CommandID) {
		return
	}
	grant := project.EnvironmentGrant{
		Repository: repository, Revision: in.ExpectedRevision,
		Owner: v.User.ID, Profile: in.Profile, Project: in.Project, Active: in.Active,
	}
	if grant.Revision < 0 || grant.Validate() != nil {
		auth.JSONError(w, 400, "invalid_environment_grant", "Environment grant is incomplete or invalid.")
		return
	}
	receipt, err := s.Coordinator.ApplyEnvironmentGrant(r.Context(), in.CommandID, factoryPrincipal(v), grant)
	if err != nil {
		factoryCommandError(w, err)
		return
	}
	auth.JSONResponse(w, 200, receipt)
}
