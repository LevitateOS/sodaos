package web

import (
	"errors"
	"net/http"

	"github.com/levitateos/sodaos/internal/forgejo"
	"github.com/levitateos/sodaos/internal/store"
)

// environmentAdministrator checks current native authority for the two membership
// views. Failure withholds elevated visibility, not the member's own account or
// connection. It never changes Linux ownership or falls back to the creator ID.
func (s *Server) environmentAdministrator(r *http.Request, v store.Session, p store.Project) (bool, error) {
	if v.User.ID == s.Config.OperatorID {
		return true, nil
	}
	grant, err := s.userGrant(r, v)
	if err != nil {
		return false, err
	}
	if !forgejo.HasScope(grant.Scopes, "read:repository") {
		return false, store.ErrGrantUnavailable
	}
	repo, err := s.Forgejo.RepositoryByID(r.Context(), grant.Access, p.RepositoryID)
	if err != nil {
		return false, err
	}
	if repo.Owner.ID == v.User.ID {
		return true, nil
	}
	if !forgejo.HasScope(grant.Scopes, "read:user") || !forgejo.HasScope(grant.Scopes, "read:organization") {
		return false, store.ErrGrantUnavailable
	}
	// The stored display login may have changed upstream; do not use it in an
	// authority lookup. The grant must still identify the session's stable actor.
	actor, err := s.Forgejo.Current(r.Context(), grant.Access)
	if err != nil {
		return false, err
	}
	if actor.ID != v.User.ID || actor.Login == "" {
		return false, forgejo.ErrInvalidResponse
	}
	owner, err := s.Forgejo.OrganizationOwner(r.Context(), grant.Access, actor.Login, repo.Owner.Login)
	var native *forgejo.HTTPError
	if errors.As(err, &native) && (native.Status == 403 || native.Status == 404) {
		return false, nil
	}
	return owner, err
}
