package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	sdk "forgejo.org/extension-sdk"
	"github.com/levitateos/factory/pkg/store"
	"github.com/levitateos/sodaos/internal/web/auth"
)

// Controller authority stays on Fountain. This adapter maps the selected
// canonical repository to its native locator and forwards live human admission;
// it creates no local placement, background PAT authority or inference funding.
func (s *API) apiFactoryController(w http.ResponseWriter, r *http.Request, session store.Session) {
	repository, ok := auth.PositiveID(r.PathValue("repositoryID"))
	if !ok || r.URL.RawQuery != "" || r.URL.ForceQuery {
		auth.JSONError(w, 400, "invalid_request", "Choose a repository without query parameters.")
		return
	}
	access, err := s.factoryVisibleRepository(r, session, repository)
	if err != nil {
		auth.ProviderError(w, err)
		return
	}
	authority, ok := requestExtensionAuthority(r)
	if !ok || !s.extensionSessionCurrent(r.Context(), r, session) {
		auth.JSONError(w, 403, "session_changed", "Refresh the current account before requesting runtime control.")
		return
	}
	native := strconv.FormatInt(access.repository.ID, 10)
	if r.Method == http.MethodGet {
		view, err := authority.RuntimeController(r.Context(), native)
		if err != nil {
			auth.JSONError(w, 503, "controller_unavailable", "Runtime control is unavailable for this repository.")
			return
		}
		if !s.extensionSessionCurrent(r.Context(), r, session) {
			auth.JSONError(w, 409, "session_changed", "Refresh the current account.")
			return
		}
		auth.JSONResponse(w, 200, view)
		return
	}
	if !s.factoryOwner(w, r, session, repository) {
		return
	}
	var request sdk.ApplianceRuntimeSelection
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096))
	decoder.DisallowUnknownFields()
	var trailing any
	if decoder.Decode(&request) != nil || decoder.Decode(&trailing) != io.EOF || !sdk.ValidApplianceRuntimeSelection(request) {
		auth.JSONError(w, 400, "invalid_controller_command", "Use the captured runtime, installation, revision and finite command.")
		return
	}
	receipt, err := authority.SelectRuntimeController(r.Context(), native, request)
	if err != nil {
		auth.JSONError(w, 503, "controller_unconfirmed", "Runtime selection was not confirmed. Keep its original command and check again.")
		return
	}
	if !s.extensionSessionCurrent(r.Context(), r, session) {
		auth.JSONError(w, 409, "session_changed", "The session changed. Keep the original command and check its receipt.")
		return
	}
	auth.JSONResponse(w, 200, receipt)
}
