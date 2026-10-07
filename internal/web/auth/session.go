package auth

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/levitateos/sodaos/internal/store"
)

func (s *Service) preferences(w http.ResponseWriter, r *http.Request, user store.User, beforeWrite func() bool) {
	type preferences struct {
		DisplayName string `json:"display_name"`
	}
	if r.Method == http.MethodGet {
		JSONResponse(w, http.StatusOK, preferences{user.Name})
		return
	}
	var input struct {
		DisplayName *string `json:"display_name"`
	}
	if !DecodeAPIObject(w, r, &input) {
		return
	}
	if input.DisplayName == nil || len(strings.TrimSpace(*input.DisplayName)) > 200 {
		JSONError(w, http.StatusBadRequest, "invalid_display_name", "Provide a Soda display name of at most 200 bytes.")
		return
	}
	name := strings.TrimSpace(*input.DisplayName)
	if beforeWrite != nil && !beforeWrite() {
		return
	}
	if err := s.Store.RenameProfile(r.Context(), user.ID, name); err != nil {
		JSONError(w, http.StatusServiceUnavailable, "store_unavailable", "Could not save Soda preferences.")
		return
	}
	JSONResponse(w, http.StatusOK, preferences{name})
}

type developmentKeyView struct {
	ID          string `json:"id"`
	PublicKey   string `json:"public_key"`
	Fingerprint string `json:"fingerprint"`
}

func (s *Service) apiKeys(w http.ResponseWriter, r *http.Request, v store.Session) {
	if r.Method == http.MethodPost {
		var input struct {
			PublicKey string `json:"public_key"`
		}
		if !DecodeAPIObject(w, r, &input) {
			return
		}
		public, fingerprint, err := NormalizeDevelopmentKey(input.PublicKey)
		if err != nil {
			JSONError(w, http.StatusBadRequest, "invalid_public_key", "Provide one public SSH key without authorized_keys options. Never submit a private key.")
			return
		}
		if err = s.Store.AddKey(r.Context(), v.User.ID, public, fingerprint); err != nil {
			JSONError(w, http.StatusServiceUnavailable, "store_unavailable", "Could not register development key.")
			return
		}
	}
	keys, err := s.Store.Keys(r.Context(), v.User.ID)
	if err != nil {
		JSONError(w, http.StatusServiceUnavailable, "store_unavailable", "Could not read development keys.")
		return
	}
	items := make([]developmentKeyView, 0, len(keys))
	for _, key := range keys {
		items = append(items, developmentKeyView{strconv.FormatInt(key.ID, 10), key.Public, key.Fingerprint})
	}
	JSONResponse(w, http.StatusOK, struct {
		Items []developmentKeyView `json:"items"`
	}{items})
}

func (s *Service) apiRemoveDevelopmentKey(w http.ResponseWriter, r *http.Request, v store.Session) {
	if r.URL.RawQuery != "" || r.URL.ForceQuery {
		JSONError(w, 400, "invalid_request", "No query parameters are accepted.")
		return
	}
	id, ok := PositiveID(r.PathValue("key"))
	if !ok {
		JSONError(w, 400, "invalid_key", "Provide a saved key ID.")
		return
	}
	var input struct {
		ConfirmLast bool `json:"confirm_last"`
	}
	if !DecodeAPIObject(w, r, &input) {
		return
	}
	removed, err := s.Store.RemoveKey(r.Context(), v.User.ID, id, input.ConfirmLast)
	if errors.Is(err, store.ErrLastKeyConfirmationRequired) {
		JSONError(w, http.StatusConflict, "final_key_confirmation_required", "Confirm removal of your final saved development key.")
		return
	}
	if err != nil {
		JSONError(w, 503, "store_unavailable", "Could not remove saved key.")
		return
	}
	if !removed {
		JSONError(w, 404, "not_found", "Saved key not found.")
		return
	}
	JSONResponse(w, 200, map[string]any{"removed": true, "existing_project_access_changed": false})
}
