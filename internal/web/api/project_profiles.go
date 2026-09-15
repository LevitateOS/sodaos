package api

import (
	"context"
	"github.com/levitateos/sodaos/internal/web/auth"
	"net/http"
	"time"

	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
)

func (s *API) apiProjectProfiles(w http.ResponseWriter, r *http.Request, v store.Session) {
	id, ok := auth.PositiveID(r.PathValue("repositoryID"))
	if !ok || r.URL.RawQuery != "" || r.URL.ForceQuery {
		auth.JSONError(w, 400, "invalid_repository", "Provide one repository ID without query parameters.")
		return
	}
	access, err := s.visibleRepository(r, v, id)
	if err != nil {
		auth.ProviderError(w, err)
		return
	}
	if access.repository.Owner.ID != v.User.ID {
		auth.JSONError(w, 403, "owner_required", "Only the human repository owner can select a creation profile.")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	profile, err := s.Host.ResolveProfile(ctx)
	if err != nil {
		auth.JSONError(w, 503, "profile_unavailable", "Installed Project OS could not be confirmed. No image was pulled or started.")
		return
	}
	cookie, err := auth.RequestCookie(r, auth.SessionCookie)
	if err != nil {
		auth.JSONError(w, 401, "unauthorized", "Reconnect to Soda.")
		return
	}
	if err := s.Auth.RequireCurrentSession(ctx, cookie.Value, v); err != nil {
		auth.JSONError(w, 401, "unauthorized", "Soda context changed.")
		return
	}
	auth.JSONResponse(w, 200, struct {
		Items []project.Profile `json:"items"`
	}{[]project.Profile{profile}})
}
