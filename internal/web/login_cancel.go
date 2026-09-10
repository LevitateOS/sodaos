package web

import (
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"errors"
	"net/http"

	"github.com/levitateos/sodaos/internal/store"
)

// Anonymous cancellation proves possession of the unique pending OAuth cookie,
// not a caller-selected actor/context. The custom bootstrap header and same-origin
// fetch metadata prevent cross-origin disclosure without adding anonymous state.
func (s *Server) cancelLogin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		w.Header().Set("Allow", "GET, POST")
		jsonError(w, 405, "method_not_allowed", "HTTP method not supported.")
		return
	}
	if r.URL.RawQuery != "" || r.URL.ForceQuery || r.Header.Get("Sec-Fetch-Site") != "same-origin" || len(r.Header.Values("X-Soda-Logout")) != 1 || r.Header.Get("X-Soda-Logout") != "1" {
		jsonError(w, 403, "invalid_origin", "Use the same-origin sign-out action.")
		return
	}
	actor, ok := positiveID(r.Header.Get(expectedUserHeader))
	if !ok || len(r.Header.Values(expectedUserHeader)) != 1 {
		jsonError(w, 400, "invalid_actor_context", "Provide one expected Forgejo user ID.")
		return
	}
	if origins := r.Header.Values("Origin"); (r.Method == http.MethodPost && len(origins) != 1) || len(origins) > 1 || (len(origins) == 1 && origins[0] != s.Config.ForgejoURL) {
		jsonError(w, 403, "invalid_origin", "Use the same-origin sign-out action.")
		return
	}
	if _, err := requestCookie(r, sessionCookie); err != nil && !errors.Is(err, http.ErrNoCookie) {
		jsonError(w, 400, "invalid_cookie", "Ambiguous Soda session cookie.")
		return
	}
	cookie, err := requestCookie(r, oauthCookie)
	if errors.Is(err, http.ErrNoCookie) {
		if r.Method != http.MethodGet {
			jsonError(w, 403, "invalid_cookie", "Sign-in cookie is unavailable.")
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if err != nil {
		jsonError(w, 400, "invalid_cookie", "Ambiguous Soda sign-in cookie.")
		return
	}
	digest := sha256.Sum256([]byte("soda-cancel-login\x00" + cookie.Value))
	csrf := base64.RawURLEncoding.EncodeToString(digest[:])
	if r.Method == http.MethodGet {
		jsonResponse(w, 200, struct {
			CSRF string `json:"csrf_token"`
		}{csrf})
		return
	}
	if !s.validAPIMutation(r, csrf) || r.Header.Get("Content-Type") != "application/json" {
		jsonError(w, 403, "invalid_csrf", "Request origin or CSRF token is invalid.")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, apiBodyLimit)
	if !decodeAPIObject(w, r, &struct{}{}) {
		return
	}
	if s.Store == nil {
		jsonError(w, 503, "store_unavailable", "Soda session storage is unavailable.")
		return
	}
	s.terminalMu.Lock()
	defer s.terminalMu.Unlock()
	id, contextActor, err := s.Store.OAuthCancellationContext(r.Context(), cookie.Value)
	if err == nil && contextActor != 0 && contextActor != actor {
		err = store.ErrLoginContext
	}
	if err == nil {
		// Never use a pending cookie to cancel a different current browser actor.
		current, cookieErr := requestCookie(r, sessionCookie)
		if cookieErr == nil {
			var session store.Session
			session, err = s.Store.Session(r.Context(), current.Value)
			if errors.Is(err, sql.ErrNoRows) {
				err = nil
			} else if err == nil && session.ContextID != id {
				err = store.ErrLoginContext
			}
		} else if !errors.Is(cookieErr, http.ErrNoCookie) {
			err = cookieErr
		}
	}
	if err == nil {
		err = s.Store.EndLoginContext(r.Context(), id)
	}
	if err != nil {
		jsonError(w, 409, "cancellation_unconfirmed", "Could not confirm sign-in cancellation. Retry sign-out.")
		return
	}
	s.cancelTerminals(id, "")
	s.cookie(w, sessionCookie, "", -1)
	s.cookie(w, oauthCookie, "", -1)
	w.WriteHeader(http.StatusNoContent)
}
