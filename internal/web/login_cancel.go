package web

import (
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"errors"
	"net/http"

	"github.com/levitateos/sodaos/internal/store"
)

func validCancelLoginOrigin(forgejoURL string, r *http.Request) bool {
	origins := r.Header.Values("Origin")
	if r.Method == http.MethodPost && len(origins) != 1 {
		return false
	}
	if len(origins) > 1 || (len(origins) == 1 && origins[0] != forgejoURL) {
		return false
	}
	return true
}

func validateCancelLoginMetadata(r *http.Request) bool {
	if r.URL.RawQuery != "" || r.URL.ForceQuery {
		return false
	}
	if r.Header.Get("Sec-Fetch-Site") != "same-origin" {
		return false
	}
	return len(r.Header.Values("X-Soda-Logout")) == 1 && r.Header.Get("X-Soda-Logout") == "1"
}

func validateCancelLoginHeaders(s *Server, w http.ResponseWriter, r *http.Request) (int64, bool) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		w.Header().Set("Allow", "GET, POST")
		jsonError(w, 405, "method_not_allowed", "HTTP method not supported.")
		return 0, false
	}
	if !validateCancelLoginMetadata(r) || !validCancelLoginOrigin(s.Config.ForgejoURL, r) {
		jsonError(w, 403, "invalid_origin", "Use the same-origin sign-out action.")
		return 0, false
	}
	actor, ok := positiveID(r.Header.Get(expectedUserHeader))
	if !ok || len(r.Header.Values(expectedUserHeader)) != 1 {
		jsonError(w, 400, "invalid_actor_context", "Provide one expected Forgejo user ID.")
		return 0, false
	}
	return actor, true
}

func cancelLoginCSRF(cookieValue string) string {
	digest := sha256.Sum256([]byte("soda-cancel-login\x00" + cookieValue))
	return base64.RawURLEncoding.EncodeToString(digest[:])
}

func resolveCancelLoginCookie(w http.ResponseWriter, r *http.Request) (string, string, bool) {
	if _, err := requestCookie(r, sessionCookie); err != nil && !errors.Is(err, http.ErrNoCookie) {
		jsonError(w, 400, "invalid_cookie", "Ambiguous Soda session cookie.")
		return "", "", false
	}
	cookie, err := requestCookie(r, oauthCookie)
	if errors.Is(err, http.ErrNoCookie) {
		if r.Method != http.MethodGet {
			jsonError(w, 403, "invalid_cookie", "Sign-in cookie is unavailable.")
			return "", "", false
		}
		w.WriteHeader(http.StatusNoContent)
		return "", "", false
	}
	if err != nil {
		jsonError(w, 400, "invalid_cookie", "Ambiguous Soda sign-in cookie.")
		return "", "", false
	}
	return cookie.Value, cancelLoginCSRF(cookie.Value), true
}

func (s *Server) validateCancelLoginPOST(w http.ResponseWriter, r *http.Request, csrf string) bool {
	if !s.validAPIMutation(r, csrf) || r.Header.Get("Content-Type") != "application/json" {
		jsonError(w, 403, "invalid_csrf", "Request origin or CSRF token is invalid.")
		return false
	}
	r.Body = http.MaxBytesReader(w, r.Body, apiBodyLimit)
	if !decodeAPIObject(w, r, &struct{}{}) {
		return false
	}
	if s.Store == nil {
		jsonError(w, 503, "store_unavailable", "Soda session storage is unavailable.")
		return false
	}
	return true
}

func (s *Server) verifyCancelSessionContext(r *http.Request, contextID string) error {
	current, cookieErr := requestCookie(r, sessionCookie)
	if cookieErr != nil {
		if errors.Is(cookieErr, http.ErrNoCookie) {
			return nil
		}
		return cookieErr
	}
	session, err := s.Store.Session(r.Context(), current.Value)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err == nil && session.ContextID != contextID {
		return store.ErrLoginContext
	}
	return err
}

func (s *Server) executeCancelLogin(r *http.Request, cookieVal string, actor int64) error {
	s.terminalMu.Lock()
	defer s.terminalMu.Unlock()
	id, contextActor, err := s.Store.OAuthCancellationContext(r.Context(), cookieVal)
	if err != nil {
		return err
	}
	if contextActor != 0 && contextActor != actor {
		return store.ErrLoginContext
	}
	if err := s.verifyCancelSessionContext(r, id); err != nil {
		return err
	}
	if err := s.Store.EndLoginContext(r.Context(), id); err != nil {
		return err
	}
	s.cancelTerminals(id, "")
	return nil
}

// Anonymous cancellation proves possession of the unique pending OAuth cookie,
// not a caller-selected actor/context. The custom bootstrap header and same-origin
// fetch metadata prevent cross-origin disclosure without adding anonymous state.
func (s *Server) cancelLogin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	actor, ok := validateCancelLoginHeaders(s, w, r)
	if !ok {
		return
	}
	cookieVal, csrf, ok := resolveCancelLoginCookie(w, r)
	if !ok {
		return
	}
	if r.Method == http.MethodGet {
		jsonResponse(w, 200, struct {
			CSRF string `json:"csrf_token"`
		}{csrf})
		return
	}
	if !s.validateCancelLoginPOST(w, r, csrf) {
		return
	}
	if err := s.executeCancelLogin(r, cookieVal, actor); err != nil {
		jsonError(w, 409, "cancellation_unconfirmed", "Could not confirm sign-in cancellation. Retry sign-out.")
		return
	}
	s.cookie(w, sessionCookie, "", -1)
	s.cookie(w, oauthCookie, "", -1)
	w.WriteHeader(http.StatusNoContent)
}
