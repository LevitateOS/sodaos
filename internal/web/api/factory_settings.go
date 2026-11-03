package api

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/levitateos/factory/pkg/factory"

	"github.com/levitateos/factory/pkg/store"
	"github.com/levitateos/sodaos/internal/web/auth"
)

// factoryPrincipal names the admitted browser actor. Body fields can never
// select the authorizing human; the host-derived session owns attribution.
func factoryPrincipal(v store.Session) string {
	return "native:" + strconv.FormatInt(v.User.ID, 10)
}

// factoryRepository resolves the path repository under current native
// visibility. Absent or undisclosable repositories report 404.
func (s *API) factoryRepository(w http.ResponseWriter, r *http.Request, v store.Session) (int64, bool) {
	id, valid := auth.PositiveID(r.PathValue("repositoryID"))
	if !valid {
		auth.JSONError(w, 404, "not_found", "Repository not found.")
		return 0, false
	}
	if _, err := s.factoryVisibleRepository(r, v, id); err != nil {
		auth.ProviderError(w, err)
		return 0, false
	}
	return id, true
}

// factoryOwner admits the current native repository owner or administrator.
// Organization ownership counts; the appliance operator does not bypass.
func (s *API) factoryOwner(w http.ResponseWriter, r *http.Request, v store.Session, repository int64) bool {
	if err := s.requireFactoryOwner(r, v, repository); err != nil {
		factoryOwnerError(w, err)
		return false
	}
	return true
}

var errFactoryAdminRequired = errors.New("current repository administration is required")

func factoryOwnerError(w http.ResponseWriter, err error) {
	if errors.Is(err, errFactoryAdminRequired) {
		auth.JSONError(w, 403, "repository_admin_required", "Only the current repository owner or administrator can configure factory policy.")
		return
	}
	auth.ProviderError(w, err)
}

// requireFactoryOwner returns current native admission without writing an
// HTTP response, allowing shared services to use it as an authority port.
func (s *API) requireFactoryOwner(r *http.Request, v store.Session, repository int64) error {
	access, err := s.factoryVisibleRepository(r, v, repository)
	if err != nil {
		return err
	}
	if access.repository.Permissions != nil && access.repository.Permissions.Admin {
		return nil
	}
	allowed, err := s.environmentAdministrator(r, access)
	if err != nil {
		return err
	}
	if !allowed {
		return errFactoryAdminRequired
	}
	return nil
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
