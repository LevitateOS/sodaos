package api

import (
	"errors"
	"net/http"

	"github.com/levitateos/factory/pkg/collaboration"
	factoryhttpapi "github.com/levitateos/factory/pkg/factory/httpapi"
	"github.com/levitateos/factory/pkg/store"
	"github.com/levitateos/sodaos/internal/web/auth"
)

// apiFactoryRepositoryHTTP supplies only this host's current ports/principal.
func (s *API) apiFactoryRepositoryHTTP(w http.ResponseWriter, r *http.Request, v store.Session) {
	actor, ok := s.factoryCurrentActor(w, r, v)
	if !ok {
		return
	}
	admitted := collaboration.ActorID(actor)
	transport := factoryhttpapi.Repositories{Store: s.Store,
		Visibility:     s.workJourneys(r, v, admitted).Visibility,
		Current:        func(w http.ResponseWriter, current *http.Request) bool { return s.checkLifecycleSession(w, current, v) },
		AuthorityError: factoryOwnerError, VisibilityError: auth.ProviderError, HostError: nativeRepositoryHTTPError,
	}
	switch r.Pattern {
	case "GET /api/repositories/{repositoryID}/factory":
		transport.Status = s.repositoryStatus(r, v, admitted)
	case "GET /api/repositories/{repositoryID}/factory/access":
		transport.Access = s.repositoryAccess(r, v, admitted)
	case "GET /api/repositories/{repositoryID}/factory/activity", "PUT /api/repositories/{repositoryID}/factory/activity":
		transport.Activity = s.repositoryActivity(r, v, admitted)
	case "PUT /api/repositories/{repositoryID}/factory/policy":
		transport.Policies = s.repositoryPolicies(r, v, admitted)
	case "GET /api/repositories/{repositoryID}/factory/repair", "PUT /api/repositories/{repositoryID}/factory/repair":
		transport.Repair = s.repairFunding(r, v, admitted)
	case "PUT /api/repositories/{repositoryID}/factory/sponsorships/{connection}":
		transport.Sponsorships = s.sponsorships(r, v, admitted)
	default:
		transport.Runtime = s.runtimeGrants(r, v, admitted)
	}
	product := s.factoryProduct(v, admitted, transport.Current)
	product.Repositories = transport
	product.Handler().ServeHTTP(w, r)
}

func nativeRepositoryHTTPError(w http.ResponseWriter, err error) bool {
	switch {
	case errors.Is(err, auth.ErrNativeSessionChanged):
		auth.ProviderError(w, err)
	case errors.Is(err, errFactoryOperatorRequired):
		auth.JSONError(w, 403, "operator_required", "Only the configured Soda operator can change appliance capacity.")
	case errors.Is(err, errFactoryEnvironmentOwnerRequired):
		auth.JSONError(w, 403, "repository_owner_required", "Only the current repository owner can permit factory environment setup.")
	case errors.Is(err, errSponsorshipProjectRequired):
		auth.JSONError(w, 409, "grant_unavailable", "Create this repository's Project before sponsoring Factory use.")
	default:
		return false
	}
	return true
}
