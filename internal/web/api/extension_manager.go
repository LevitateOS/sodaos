package api

import (
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"strings"

	"github.com/levitateos/sodaos/internal/forgejo"
	"github.com/levitateos/sodaos/internal/web/auth"
)

// operatorExtensionManager is mounted only on the private SO_PEERCRED-gated
// operator listener. Root reaches Fountain as the configured Soda service UID;
// neither a browser Actor nor the Rust host RPC participates in this hop.
func (s *API) operatorExtensionManager(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	operation := strings.TrimPrefix(r.URL.Path, "/operator/extensions/")
	limit, _, ok := forgejo.ManagerControlBounds(operation)
	if !ok {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		auth.JSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "POST required.")
		return
	}
	media, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || media != "application/json" || r.URL.RawQuery != "" || r.URL.ForceQuery || r.URL.RawPath != "" {
		auth.JSONError(w, http.StatusBadRequest, "invalid_manager_request", "Manager operations require JSON and no query parameters.")
		return
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, limit))
	if err != nil || !json.Valid(body) {
		auth.JSONError(w, http.StatusBadRequest, "invalid_manager_request", "Manager request requires bounded JSON.")
		return
	}
	if s.ExtensionManager == nil {
		auth.JSONError(w, http.StatusServiceUnavailable, "manager_unavailable", "Extension manager is unavailable.")
		return
	}
	status, data, err := s.ExtensionManager.ExtensionManager(r.Context(), operation, body)
	if err != nil {
		auth.JSONError(w, http.StatusServiceUnavailable, "manager_unavailable", "Extension manager is unavailable.")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(data)
}
