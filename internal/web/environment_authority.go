package web

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/levitateos/sodaos/internal/forgejo"
	"github.com/levitateos/sodaos/internal/store"
)

var errRepositoryConsent = errors.New("repository and user consent required")
var errProviderIdentity = errors.New("provider identity differs from Soda session")

// Request-local verified facts, never serialized or persisted as copied roles.
type repositoryAccess struct {
	actor      forgejo.User
	repository forgejo.Repository
	grant      store.Grant
}

func (s *Server) visibleRepository(r *http.Request, v store.Session, id int64) (repositoryAccess, error) {
	var access repositoryAccess
	grant, err := s.userGrant(r, v)
	if err != nil {
		return access, err
	}
	if !forgejo.HasScope(grant.Scopes, "read:user") || !forgejo.HasScope(grant.Scopes, "read:repository") {
		return access, errRepositoryConsent
	}
	actor, err := s.Forgejo.Current(r.Context(), grant.Access)
	if err != nil {
		return access, err
	}
	if actor.ID != v.User.ID {
		return access, errProviderIdentity
	}
	if actor.Login == "" {
		return access, forgejo.ErrInvalidResponse
	}
	repo, err := s.Forgejo.RepositoryByID(r.Context(), grant.Access, id)
	if err != nil {
		return access, err
	}
	return repositoryAccess{actor, repo, grant}, nil
}

// Ownership consumes the already-verified repository/actor, avoiding a second
// lookup or stale display login. Visibility alone never confers administration.
func (s *Server) environmentAdministrator(r *http.Request, a repositoryAccess) (bool, error) {
	if a.repository.Owner.ID == a.actor.ID {
		return true, nil
	}
	if !forgejo.HasScope(a.grant.Scopes, "read:organization") {
		return false, store.ErrGrantUnavailable
	}
	owner, err := s.Forgejo.OrganizationOwner(r.Context(), a.grant.Access, a.actor.Login, a.repository.Owner.Login)
	var native *forgejo.HTTPError
	if errors.As(err, &native) && (native.Status == 403 || native.Status == 404) {
		return false, nil
	}
	return owner, err
}

type environmentReader struct {
	login                               string
	administrator, authorityUnavailable bool
}

// Only existing membership or the explicit Soda operator permits degraded reads.
// New/nonmember readers fail before any metadata or native inspection is exposed.
func (s *Server) authorizeEnvironmentRead(w http.ResponseWriter, r *http.Request, v store.Session, p store.Project) (environmentReader, bool) {
	var reader environmentReader
	login, err := s.Store.MemberLogin(r.Context(), p.ID, v.User.ID)
	member := err == nil
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		jsonError(w, 503, "store_unavailable", "Could not read membership.")
		return reader, false
	}
	reader.login = login
	if v.User.ID == s.Config.OperatorID {
		reader.administrator = true
		return reader, true
	}
	access, err := s.visibleRepository(r, v, p.RepositoryID)
	if err != nil {
		if !member {
			providerError(w, err)
			return reader, false
		}
		reader.authorityUnavailable = true
		return reader, true
	}
	reader.administrator, err = s.environmentAdministrator(r, access)
	reader.authorityUnavailable = err != nil
	return reader, true
}
