package api

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/levitateos/sodaos/internal/web/auth"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/factory/control"
	"github.com/levitateos/sodaos/internal/store"
)

// factoryPrincipal names the admitted browser actor. Body fields can never
// select the authorizing human; the host-derived session owns attribution.
func factoryPrincipal(v store.Session) string {
	return "native:" + strconv.FormatInt(v.User.ID, 10)
}

func factoryCommandError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, store.ErrCommandConflict):
		auth.JSONError(w, 409, "command_conflict", "Command identity reused for different content.")
	case errors.Is(err, store.ErrStaleRevision):
		auth.JSONError(w, 412, "stale_revision", "Grant changed while applying; refresh and retry.")
	case errors.Is(err, control.ErrCommandRunning):
		auth.JSONResponse(w, 202, map[string]any{"running": true})
	case errors.Is(err, control.ErrIneffectiveAuthority):
		auth.JSONError(w, 409, "authority_ineffective", "Current grants do not authorize this action.")
	default:
		auth.JSONError(w, 503, "store_unavailable", "Factory authority storage is unavailable.")
	}
}

// factoryRepository resolves the path repository under current native
// visibility. Absent or undisclosable repositories report 404.
func (s *API) factoryRepository(w http.ResponseWriter, r *http.Request, v store.Session) (int64, bool) {
	id, valid := auth.PositiveID(r.PathValue("repositoryID"))
	if !valid {
		auth.JSONError(w, 404, "not_found", "Repository not found.")
		return 0, false
	}
	if _, err := s.visibleRepository(r, v, id); err != nil {
		auth.ProviderError(w, err)
		return 0, false
	}
	return id, true
}

// factoryOwner admits the current native repository owner or administrator.
// Organization ownership counts; the appliance operator does not bypass.
func (s *API) factoryOwner(w http.ResponseWriter, r *http.Request, v store.Session, repository int64) bool {
	access, err := s.visibleRepository(r, v, repository)
	if err != nil {
		auth.ProviderError(w, err)
		return false
	}
	if access.repository.Permissions != nil && access.repository.Permissions.Admin {
		return true
	}
	allowed, err := s.environmentAdministrator(r, access)
	if err != nil {
		auth.ProviderError(w, err)
		return false
	}
	if !allowed {
		auth.JSONError(w, 403, "repository_admin_required", "Only the current repository owner or administrator can configure factory policy.")
		return false
	}
	return true
}

// factoryOperator admits only the configured Soda operator.
func (s *API) factoryOperator(w http.ResponseWriter, v store.Session) bool {
	if v.User.ID != s.Config.OperatorID {
		auth.JSONError(w, 403, "operator_required", "Only the configured Soda operator can change appliance capacity.")
		return false
	}
	return true
}

// settingsCommandID validates the client-generated command identity every
// settings mutation carries for idempotent replay.
func settingsCommandID(w http.ResponseWriter, commandID string) bool {
	if !factory.ValidID(commandID) {
		auth.JSONError(w, 400, "invalid_command", "Settings changes require a client-generated command identity.")
		return false
	}
	return true
}

type actorRefRequest struct {
	TokenID string `json:"token_id"`
	ActorID string `json:"actor_id"`
}

func (a actorRefRequest) resolve(kind string) (factory.ActorBindingRef, bool) {
	token, tokenOK := auth.PositiveID(a.TokenID)
	actor, actorOK := auth.PositiveID(a.ActorID)
	ref := factory.ActorBindingRef{TokenID: token, ActorID: actor, Kind: kind}
	return ref, tokenOK && actorOK && ref.Validate() == nil
}
