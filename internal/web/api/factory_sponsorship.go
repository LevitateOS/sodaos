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
	sp.AllowanceMinutes, sp.MaxConcurrent = 0, 0
	return sp
}

type factorySponsorshipRequest struct {
	CommandID        string   `json:"command_id"`
	GrantID          string   `json:"grant_id"`
	ExpectedRevision int64    `json:"expected_revision"`
	Generation       int64    `json:"generation"`
	Roles            []string `json:"roles"`
	AllowanceMinutes int      `json:"allowance_minutes"`
	MaxConcurrent    int      `json:"max_concurrent"`
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
		Connection: connection, GrantID: in.GrantID, Generation: in.Generation, Roles: in.Roles,
		AllowanceMinutes: in.AllowanceMinutes, MaxConcurrent: in.MaxConcurrent, Active: in.Active,
	}
	if sponsorship.Revision < 0 || sponsorship.Validate() != nil {
		auth.JSONError(w, 400, "invalid_sponsorship", "Sponsorship is incomplete or invalid.")
		return
	}
	if !s.checkSponsorshipBroker(w, r, v, sponsorship) {
		return
	}
	receipt, err := s.Coordinator.ApplySponsorship(r.Context(), in.CommandID, factoryPrincipal(v), in.ExpectedRevision, sponsorship)
	if err != nil {
		factoryCommandError(w, err)
		return
	}
	auth.JSONResponse(w, 200, receipt)
}

// checkSponsorshipBroker verifies the sponsorship references the caller's
// own current connection and a live canonical broker grant. It reads grant
// metadata only from the identity broker; credential custody stays with the
// broker. Refusals preserve the established sponsorship admission contract.
func (s *API) checkSponsorshipBroker(w http.ResponseWriter, r *http.Request, v store.Session, sponsorship factory.Sponsorship) bool {
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
	if connection.Generation != sponsorship.Generation {
		auth.JSONError(w, 409, "stale_credential_generation", "Connection credential changed; refresh and retry.")
		return false
	}
	grants, err := s.Identity.Grants(r.Context(), v.User.ID, sponsorship.Connection)
	if err != nil {
		if errors.Is(err, identity.ErrNotFound) {
			auth.JSONError(w, 404, "not_found", "Broker grant not found.")
			return false
		}
		if errors.Is(err, identity.ErrDenied) {
			auth.JSONError(w, 403, "connection_owner_required", "Only the connection owner can sponsor factory use.")
			return false
		}
		auth.JSONError(w, 503, "store_unavailable", "Could not read broker grant.")
		return false
	}
	var grant identity.Grant
	found = false
	for _, g := range grants {
		if g.ID == sponsorship.GrantID {
			grant, found = g, true
			break
		}
	}
	if !found {
		auth.JSONError(w, 404, "not_found", "Broker grant not found.")
		return false
	}
	if grant.ConnectionID != sponsorship.Connection || grant.Revoked {
		auth.JSONError(w, 409, "grant_unavailable", "Broker grant is revoked or belongs elsewhere.")
		return false
	}
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
	receipt, err := s.Coordinator.ApplyEnvironmentGrant(r.Context(), in.CommandID, factoryPrincipal(v), in.ExpectedRevision, grant)
	if err != nil {
		factoryCommandError(w, err)
		return
	}
	auth.JSONResponse(w, 200, receipt)
}
