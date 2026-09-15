package api

import (
	"context"
	"github.com/levitateos/sodaos/internal/web/auth"
	"net/http"
	"time"

	"github.com/levitateos/sodaos/internal/store"
)

func rejectOSQuery(w http.ResponseWriter, r *http.Request) bool {
	if r.URL.RawQuery != "" || r.URL.ForceQuery {
		auth.JSONError(w, 400, "invalid_request", "OS observations accept no query parameters.")
		return true
	}
	return false
}

func (s *API) confirmOSSession(w http.ResponseWriter, r *http.Request, ctx context.Context, v store.Session) bool {
	cookie, err := auth.RequestCookie(r, auth.SessionCookie)
	if err != nil {
		auth.JSONError(w, 401, "unauthorized", "Reconnect to Soda.")
		return false
	}
	if err := s.Auth.RequireCurrentSession(ctx, cookie.Value, v); err != nil {
		auth.JSONError(w, 401, "unauthorized", "Soda context changed.")
		return false
	}
	return true
}

func (s *API) apiEnvironmentOS(w http.ResponseWriter, r *http.Request, v store.Session) {
	if rejectOSQuery(w, r) {
		return
	}
	p, ok := s.loadEnvironment(w, r)
	if !ok {
		return
	}
	if _, allowed := s.authorizeEnvironmentRead(w, r, v, p); !allowed {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	observed, err := s.Host.ObserveOS(ctx, p.ID)
	if err != nil || environmentProfileMismatch(p, observed.Environment) {
		auth.JSONError(w, 503, "os_unavailable", "Native OS observation unavailable. Nothing was started or repaired.")
		return
	}
	if !s.confirmOSSession(w, r, ctx, v) {
		return
	}
	auth.JSONResponse(w, 200, observed)
}
