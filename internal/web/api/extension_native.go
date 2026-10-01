package api

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	extensions "forgejo.org/extension-sdk"
	"github.com/levitateos/sodaos/internal/store"
	"github.com/levitateos/sodaos/internal/web/auth"
)

type extensionAuthorityKey struct{}

func requestExtensionAuthority(r *http.Request) (extensions.Authority, bool) {
	authority, ok := r.Context().Value(extensionAuthorityKey{}).(extensions.Authority)
	return authority, ok
}

func (s *API) extensionProtected(next func(http.ResponseWriter, *http.Request, store.Session), methods ...string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		if !auth.AllowAPIMethod(w, r, methods) {
			return
		}
		authority, ok := extensionProductAuthority(w, r)
		if !ok {
			return
		}
		if !extensionGenerationCurrent(w, r, authority) {
			return
		}
		actorID, ok := s.extensionProductActor(w, authority)
		if !ok {
			return
		}
		if !s.extensionProductMutation(w, r) {
			return
		}
		user, ok := s.extensionProductUser(w, r, actorID, authority.Actor.Username)
		if !ok {
			return
		}
		ctx := context.WithValue(r.Context(), extensionAuthorityKey{}, authority)
		session := store.Session{User: user, ContextID: authority.SessionGeneration}
		next(w, r.WithContext(ctx), session)
	}
}

func extensionProductAuthority(w http.ResponseWriter, r *http.Request) (extensions.Authority, bool) {
	authority, err := auth.ExtensionAuthority(r)
	if err != nil || !auth.ExtensionContribution(authority.Contribution) || !extensionProductContribution(r.URL.Path, authority.Contribution) {
		auth.JSONError(w, http.StatusForbidden, "native_authority_unavailable", "Current native authority is required.")
		return extensions.Authority{}, false
	}
	return authority, true
}

func extensionGenerationCurrent(w http.ResponseWriter, r *http.Request, authority extensions.Authority) bool {
	generations := r.Header.Values(extensions.SessionGenerationHeader)
	if len(generations) != 1 || generations[0] != authority.SessionGeneration {
		auth.JSONError(w, http.StatusConflict, "extension_session_changed", "The native extension page is no longer current.")
		return false
	}
	return true
}

func (s *API) extensionProductActor(w http.ResponseWriter, authority extensions.Authority) (int64, bool) {
	actorID, valid := auth.PositiveID(authority.Actor.ID)
	if !valid || s.Store == nil {
		auth.JSONError(w, http.StatusForbidden, "invalid_actor", "Native actor is invalid.")
		return 0, false
	}
	return actorID, true
}

func (s *API) extensionProductMutation(w http.ResponseWriter, r *http.Request) bool {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		if !s.extensionMutationOrigin(r) {
			auth.JSONError(w, http.StatusForbidden, "invalid_origin", "Same-origin Soda mutation required.")
			return false
		}
		r.Body = http.MaxBytesReader(w, r.Body, auth.APIBodyLimit)
	}
	return true
}

func (s *API) extensionProductUser(w http.ResponseWriter, r *http.Request, actorID int64, username string) (store.User, bool) {
	if err := s.Store.UpsertUser(r.Context(), store.User{ID: actorID, Login: username}); err != nil {
		auth.JSONError(w, http.StatusServiceUnavailable, "store_unavailable", "Soda profile storage is unavailable.")
		return store.User{}, false
	}
	user, err := s.Store.User(r.Context(), actorID)
	if err != nil {
		auth.JSONError(w, http.StatusServiceUnavailable, "store_unavailable", "Soda profile storage is unavailable.")
		return store.User{}, false
	}
	return user, true
}

func extensionProductContribution(path string, contribution extensions.Contribution) bool {
	if path == "/api/settings/tailnet" || strings.HasPrefix(path, "/api/settings/tailnet/") {
		return extensionAdminPageContribution(contribution, "tailnet")
	}
	return extensionPageContribution(contribution, "spaces") || extensionWorkspacePanelContribution(contribution)
}

func extensionPageContribution(contribution extensions.Contribution, id string) bool {
	return contribution.Kind == "page" && contribution.Scope == "global" && contribution.ID == id
}

func extensionAdminPageContribution(contribution extensions.Contribution, id string) bool {
	return contribution.Kind == "page" && contribution.Scope == "admin" && contribution.ID == id
}

func extensionWorkspacePanelContribution(contribution extensions.Contribution) bool {
	return contribution.Kind == "panel" && contribution.ID == "workspace" && contribution.Scope == "panel"
}

func (s *API) extensionMutationOrigin(r *http.Request) bool {
	origins := r.Header.Values("Origin")
	if s.Config.ForgejoURL == "" || len(origins) != 1 || origins[0] != s.Config.ForgejoURL {
		return false
	}
	sites := r.Header.Values("Sec-Fetch-Site")
	return len(sites) <= 1 && (len(sites) == 0 || sites[0] == "same-origin")
}

func (s *API) extensionSessionCurrent(ctx context.Context, r *http.Request, session store.Session) bool {
	initial, ok := requestExtensionAuthority(r)
	if !ok {
		return false
	}
	current, err := auth.ExtensionAuthority(r.WithContext(ctx))
	return err == nil && extensionAuthorityCurrent(current, initial) &&
		current.Actor.ID == strconv.FormatInt(session.User.ID, 10) && session.ContextID == initial.SessionGeneration
}

func extensionAuthorityCurrent(current, initial extensions.Authority) bool {
	return current.ExtensionID == initial.ExtensionID && current.InstanceID == initial.InstanceID &&
		current.SessionGeneration == initial.SessionGeneration && current.Contribution == initial.Contribution &&
		current.Actor == initial.Actor
}

func (s *API) registerExtensionProductRoutes(mux *http.ServeMux) {
	protect := s.extensionProtected
	mux.HandleFunc("GET /api/repositories", protect(s.apiRepositories, http.MethodGet))
	mux.HandleFunc("GET /api/repositories/{repositoryID}/profiles", protect(s.apiProjectProfiles, http.MethodGet))
	mux.HandleFunc("GET /api/repositories/{repositoryID}/tailnet-options", protect(s.apiTailnetOptions, http.MethodGet))
	mux.HandleFunc("GET /api/repositories/{repositoryID}/factory", protect(s.apiFactoryStatus, http.MethodGet))
	mux.HandleFunc("PUT /api/repositories/{repositoryID}/factory/policy", protect(s.apiFactoryPolicy, http.MethodPut))
	mux.HandleFunc("PUT /api/repositories/{repositoryID}/factory/operator-grant", protect(s.apiFactoryOperatorGrant, http.MethodPut))
	mux.HandleFunc("PUT /api/repositories/{repositoryID}/factory/environment-grant", protect(s.apiFactoryEnvironmentGrant, http.MethodPut))
	mux.HandleFunc("PUT /api/repositories/{repositoryID}/factory/sponsorships/{connection}", protect(s.apiFactorySponsorship, http.MethodPut))
	mux.HandleFunc("PUT /api/factory/capacity", protect(s.apiFactoryCapacity, http.MethodPut))
	mux.HandleFunc("POST /api/repositories/{repositoryID}/factory/actions", protect(s.apiFactoryActions, http.MethodPost))
	mux.HandleFunc("POST /api/factory/runs/{runID}/actions", protect(s.apiFactoryRunActions, http.MethodPost))
	mux.HandleFunc("GET /api/factory/commands/{commandID}", protect(s.apiFactoryCommand, http.MethodGet))
	mux.HandleFunc("GET /api/spaces", protect(s.apiSpaces, http.MethodGet))
	mux.HandleFunc("GET /api/environments", protect(s.apiEnvironments, http.MethodGet))
	mux.HandleFunc("POST /api/environments", protect(s.apiEnvironments, http.MethodPost))
	mux.HandleFunc("GET /api/environments/{id}", protect(s.apiEnvironment, http.MethodGet))
	mux.HandleFunc("POST /api/environments/{id}/join", protect(s.apiJoinEnvironment, http.MethodPost))
	mux.HandleFunc("GET /api/environments/{id}/members", protect(s.apiEnvironmentMembers, http.MethodGet))
	mux.HandleFunc("GET /api/environments/{id}/connection", protect(s.apiConnection, http.MethodGet))
	mux.HandleFunc("GET /api/environments/{id}/os", protect(s.apiEnvironmentOS, http.MethodGet))
	mux.HandleFunc("GET /api/environments/{id}/lifecycle", protect(s.apiLifecycle, http.MethodGet))
	mux.HandleFunc("POST /api/environments/{id}/lifecycle", protect(s.apiLifecycle, http.MethodPost))
	mux.HandleFunc("GET /api/environments/{id}/access-keys", protect(s.apiAccessKeys, http.MethodGet))
	mux.HandleFunc("POST /api/environments/{id}/access-keys", protect(s.apiAccessKeys, http.MethodPost))
	mux.HandleFunc("GET /api/environments/{id}/preparation", protect(s.apiPreparation, http.MethodGet))
	mux.HandleFunc("POST /api/environments/{id}/preparation", protect(s.apiPreparation, http.MethodPost))
	mux.HandleFunc("POST /api/environments/{id}/preparation/acceptances", protect(s.apiPreparationAcceptances, http.MethodPost))
	mux.HandleFunc("POST /api/environments/{id}/preparation/actions", protect(s.apiPreparationActions, http.MethodPost))
	mux.HandleFunc("POST /api/environments/{id}/identity/launch", protect(s.apiIdentityLaunch, http.MethodPost))
	mux.HandleFunc("GET /api/environments/{id}/identity/connections", protect(s.apiIdentityAvailable, http.MethodGet))
	mux.HandleFunc("POST /api/environments/{id}/identity/grants", protect(s.apiIdentityCreateGrant, http.MethodPost))
	mux.HandleFunc("GET /api/identity/connections", protect(s.apiIdentityConnections, http.MethodGet))
	mux.HandleFunc("POST /api/identity/enrollments", protect(s.apiIdentityStartEnrollment, http.MethodPost))
	mux.HandleFunc("GET /api/identity/enrollments/{identityID}", protect(s.apiIdentityEnrollment, http.MethodGet))
	mux.HandleFunc("POST /api/identity/enrollments/{identityID}/cancel", protect(s.apiIdentityCancelEnrollment, http.MethodPost))
	mux.HandleFunc("GET /api/identity/connections/{identityID}/grants", protect(s.apiIdentityGrants, http.MethodGet))
	mux.HandleFunc("GET /api/identity/connections/{identityID}/leases", protect(s.apiIdentityLeases, http.MethodGet))
	mux.HandleFunc("POST /api/identity/connections/{identityID}/revoke", protect(s.apiIdentityRevoke, http.MethodPost))
	mux.HandleFunc("POST /api/identity/grants/{identityID}/revoke", protect(s.apiIdentityRevokeGrant, http.MethodPost))
	mux.HandleFunc("POST /api/identity/leases/{identityID}/end", protect(s.apiIdentityEndLease, http.MethodPost))
	mux.HandleFunc("GET /api/settings/tailnet", protect(s.apiTailnetSettings, http.MethodGet))
	mux.HandleFunc("POST /api/settings/tailnet/host", protect(s.apiTailnetHost, http.MethodPost))
	mux.HandleFunc("POST /api/settings/tailnet/enrollment", protect(s.apiTailnetEnrollment, http.MethodPost))
	mux.HandleFunc("GET /api/environments/{id}/tailnet", protect(s.apiProjectTailnet, http.MethodGet))
	mux.HandleFunc("POST /api/environments/{id}/tailnet", protect(s.apiProjectTailnet, http.MethodPost))
}

func (s *API) extensionActor(r *http.Request, session store.Session) (extensions.Authority, bool) {
	authority, ok := requestExtensionAuthority(r)
	return authority, ok && authority.Actor.ID == strconv.FormatInt(session.User.ID, 10) && strings.EqualFold(authority.Actor.Username, session.User.Login)
}
