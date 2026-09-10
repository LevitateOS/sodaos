package web

import (
	"context"
	"net/http"
	"time"

	"github.com/levitateos/sodaos/internal/projectos"
	"github.com/levitateos/sodaos/internal/store"
)

func (s *Server) apiProjectProfiles(w http.ResponseWriter, r *http.Request, v store.Session) {
	id, ok := positiveID(r.PathValue("repositoryID"))
	if !ok || r.URL.RawQuery != "" || r.URL.ForceQuery {
		jsonError(w, 400, "invalid_repository", "Provide one repository ID without query parameters.")
		return
	}
	access, err := s.visibleRepository(r, v, id)
	if err != nil {
		providerError(w, err)
		return
	}
	if access.repository.Owner.ID != v.User.ID {
		jsonError(w, 403, "owner_required", "Only the human repository owner can select a creation profile.")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	profile, err := s.Host.ResolveProfile(ctx)
	if err != nil {
		jsonError(w, 503, "profile_unavailable", "Installed Project OS could not be confirmed. No image was pulled or started.")
		return
	}
	cookie, err := requestCookie(r, sessionCookie)
	if err != nil {
		jsonError(w, 401, "unauthorized", "Reconnect to Soda.")
		return
	}
	current, err := s.Store.Session(ctx, cookie.Value)
	if err != nil || current.ContextID != v.ContextID || current.CSRF != v.CSRF || current.User.ID != v.User.ID {
		jsonError(w, 401, "unauthorized", "Soda context changed.")
		return
	}
	jsonResponse(w, 200, struct {
		Items []projectos.Profile `json:"items"`
	}{[]projectos.Profile{profile}})
}
