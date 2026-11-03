package api

import (
	"errors"
	"net/http"
	"strconv"

	extensions "forgejo.org/extension-sdk"
	"github.com/levitateos/factory/pkg/collaboration"
	"github.com/levitateos/factory/pkg/store"
	"github.com/levitateos/sodaos/internal/web/auth"
)

// Factory records and request scopes use local IDs. Native authority and the
// broker keep the native session and repository keys at their own boundaries.
func (s *API) factoryVisibleRepository(r *http.Request, v store.Session, local int64) (repositoryAccess, error) {
	native, err := s.FountainIdentities.NativeRepository(r.Context(), collaboration.RepositoryID(local))
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return repositoryAccess{}, extensions.ErrRepositoryNotVisible
		}
		return repositoryAccess{}, err
	}
	id, err := strconv.ParseInt(native, 10, 64)
	if err != nil {
		return repositoryAccess{}, err
	}
	return s.visibleRepository(r, v, id)
}

func (s *API) factoryExecutionRepository(r *http.Request, v store.Session, local int64) (repositoryAccess, error) {
	access, err := s.factoryVisibleRepository(r, v, local)
	if err == nil && !repositoryExecutionAllowed(access) {
		err = errRepositoryWriteRequired
	}
	return access, err
}

func (s *API) factoryCurrentActor(w http.ResponseWriter, r *http.Request, v store.Session) (int64, bool) {
	actor, err := s.FountainIdentities.LocalActor(r.Context(), strconv.FormatInt(v.User.ID, 10))
	if err != nil {
		auth.JSONError(w, http.StatusServiceUnavailable, "identity_unavailable", "Factory account identity is unavailable.")
		return 0, false
	}
	return int64(actor), true
}

func (s *API) apiFactoryIdentity(w http.ResponseWriter, r *http.Request, v store.Session) {
	if r.URL.RawQuery != "" || r.URL.ForceQuery {
		auth.JSONError(w, http.StatusBadRequest, "invalid_request", "No query parameters are accepted.")
		return
	}
	actor, ok := s.factoryCurrentActor(w, r, v)
	if !ok {
		return
	}
	auth.JSONResponse(w, http.StatusOK, struct {
		Host   collaboration.HostID `json:"host"`
		Actor  int64                `json:"actor,string"`
		Native string               `json:"native_actor"`
	}{s.FountainIdentities.HostID(), actor, strconv.FormatInt(v.User.ID, 10)})
}

type factoryRepositoryIdentity struct {
	Host          collaboration.HostID       `json:"host"`
	Repository    collaboration.RepositoryID `json:"repository,string"`
	Actor         int64                      `json:"actor,string"`
	Native        string                     `json:"native_repository"`
	Name          string                     `json:"name"`
	DefaultBranch string                     `json:"default_branch"`
}

// Resolve a canonical repository back to native navigation metadata. Lookup is
// scoped to the adopted host and checks current native visibility before reply.
func (s *API) apiFactoryRepositoryIdentity(w http.ResponseWriter, r *http.Request, v store.Session) {
	local, ok := auth.PositiveID(r.PathValue("repositoryID"))
	if !ok || r.URL.RawQuery != "" || r.URL.ForceQuery {
		auth.JSONError(w, http.StatusBadRequest, "invalid_request", "A Factory repository identity is required.")
		return
	}
	access, err := s.factoryVisibleRepository(r, v, local)
	if err != nil {
		auth.ProviderError(w, err)
		return
	}
	actor, ok := s.factoryCurrentActor(w, r, v)
	if !ok {
		return
	}
	auth.JSONResponse(w, http.StatusOK, factoryRepositoryIdentity{
		s.FountainIdentities.HostID(), collaboration.RepositoryID(local), actor,
		strconv.FormatInt(access.repository.ID, 10), access.repository.FullName, access.repository.DefaultBranch,
	})
}

// Resolve a native Fountain locator only after checking its current authority.
// This is identity discovery for the host adapter, never a permission grant.
func (s *API) apiFactoryResolveNativeRepository(w http.ResponseWriter, r *http.Request, v store.Session) {
	var request struct {
		Repository string `json:"repository"`
	}
	if !auth.DecodeAPIObject(w, r, &request) {
		return
	}
	native, ok := auth.PositiveID(request.Repository)
	if !ok || r.URL.RawQuery != "" {
		auth.JSONError(w, http.StatusBadRequest, "invalid_request", "A native repository identity is required.")
		return
	}
	access, err := s.visibleRepository(r, v, native)
	if err != nil {
		auth.ProviderError(w, err)
		return
	}
	local, err := s.FountainIdentities.LocalRepository(r.Context(), native)
	if err != nil {
		auth.JSONError(w, http.StatusServiceUnavailable, "identity_unavailable", "Factory repository identity is unavailable.")
		return
	}
	actor, ok := s.factoryCurrentActor(w, r, v)
	if !ok {
		return
	}
	auth.JSONResponse(w, http.StatusOK, factoryRepositoryIdentity{s.FountainIdentities.HostID(), local, actor, request.Repository, access.repository.FullName, access.repository.DefaultBranch})
}
