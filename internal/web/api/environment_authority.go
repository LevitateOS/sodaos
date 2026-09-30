package api

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	extensions "forgejo.org/extension-sdk"
	"github.com/levitateos/sodaos/internal/web/auth"

	"github.com/levitateos/sodaos/internal/forgejo"
	"github.com/levitateos/sodaos/internal/store"
)

var (
	errRepositoryDenied        = errors.New("repository is not visible")
	errRepositoryWriteRequired = errors.New("repository write permission required")
)

// Request-local verified facts, never serialized or persisted as copied roles.
type repositoryAccess struct {
	actor      forgejo.User
	repository forgejo.Repository
	native     bool
}

func (s *API) visibleRepository(r *http.Request, v store.Session, id int64) (repositoryAccess, error) {
	authority, ok := requestExtensionAuthority(r)
	if !ok {
		return repositoryAccess{}, errors.New("native authority is unavailable")
	}
	return s.nativeVisibleRepository(r, v, id, authority)
}

func (s *API) nativeVisibleRepository(r *http.Request, v store.Session, id int64, authority extensions.Authority) (repositoryAccess, error) {
	repository, err := authority.Native().Repository(r.Context(), strconv.FormatInt(id, 10))
	if err != nil {
		if errors.Is(err, extensions.ErrRepositoryNotVisible) {
			return repositoryAccess{}, errors.Join(errRepositoryDenied, err)
		}
		return repositoryAccess{}, err
	}
	actor := forgejo.User{ID: v.User.ID, Login: authority.Actor.Username, Admin: authority.Actor.SiteAdmin}
	owner := forgejo.User{Login: repository.Owner}
	if strings.EqualFold(owner.Login, actor.Login) {
		owner.ID = actor.ID
	}
	permissions := &forgejo.RepositoryPermissions{Pull: true}
	permissions.Push = repository.Permission == "write" || repository.Permission == "admin"
	permissions.Admin = repository.Permission == "admin" || owner.ID == actor.ID
	return repositoryAccess{actor: actor, repository: forgejo.Repository{
		ID: id, Name: repository.Name, FullName: repository.Owner + "/" + repository.Name,
		Owner: owner, Permissions: permissions,
	}, native: true}, nil
}

// Current native code-write authority admits project execution; visibility and
// retained membership alone do not. Missing capabilities fail closed.
func repositoryExecutionAllowed(a repositoryAccess) bool {
	return a.repository.Permissions != nil && a.repository.Permissions.Push
}

func (s *API) executionRepository(r *http.Request, v store.Session, id int64) (repositoryAccess, error) {
	access, err := s.visibleRepository(r, v, id)
	if err == nil && !repositoryExecutionAllowed(access) {
		err = errRepositoryWriteRequired
	}
	return access, err
}

func reportExecutionAuthorityError(w http.ResponseWriter, err error) {
	if errors.Is(err, errRepositoryWriteRequired) {
		auth.JSONError(w, 403, "repository_write_required", "Repository write permission is required for project execution.")
		return
	}
	auth.ProviderError(w, err)
}

// Ownership consumes the already-verified repository/actor, avoiding a second
// lookup or stale display login. Visibility alone never confers administration.
func (s *API) environmentAdministrator(r *http.Request, a repositoryAccess) (bool, error) {
	if a.repository.Owner.ID == a.actor.ID {
		return true, nil
	}
	authority, ok := requestExtensionAuthority(r)
	if !ok {
		return false, errors.New("native authority is unavailable")
	}
	return authority.Native().OrganizationOwner(r.Context(), a.repository.Owner.Login)
}

type environmentReader struct {
	login                               string
	administrator, authorityUnavailable bool
	repositoryVisible, executionAllowed bool
}

// Only existing membership or the explicit Soda operator permits degraded reads.
// New/nonmember readers fail before any metadata or native inspection is exposed.
var errEnvironmentReadStore = errors.New("could not read membership")

func (s *API) readEnvironmentAuthority(r *http.Request, v store.Session, p store.Project) (environmentReader, error) {
	var reader environmentReader
	login, err := s.Store.MemberLogin(r.Context(), p.ID, v.User.ID)
	member := err == nil
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		return reader, errEnvironmentReadStore
	}
	reader.login = login
	if v.User.ID == s.Config.OperatorID {
		reader.administrator = true
		access, nativeErr := s.visibleRepository(r, v, p.RepositoryID)
		reader.repositoryVisible = nativeErr == nil
		reader.executionAllowed = nativeErr == nil && repositoryExecutionAllowed(access)
		return reader, nil
	}
	access, err := s.visibleRepository(r, v, p.RepositoryID)
	if err != nil {
		if !member {
			return reader, err
		}
		reader.authorityUnavailable = true
		return reader, nil
	}
	reader.repositoryVisible = true
	reader.executionAllowed = repositoryExecutionAllowed(access)
	reader.administrator, err = s.environmentAdministrator(r, access)
	reader.authorityUnavailable = err != nil
	return reader, nil
}

func (s *API) authorizeEnvironmentRead(w http.ResponseWriter, r *http.Request, v store.Session, p store.Project) (environmentReader, bool) {
	reader, err := s.readEnvironmentAuthority(r, v, p)
	if errors.Is(err, errEnvironmentReadStore) {
		auth.JSONError(w, 503, "store_unavailable", "Could not read membership.")
	} else if err != nil {
		auth.ProviderError(w, err)
	}
	return reader, err == nil
}
