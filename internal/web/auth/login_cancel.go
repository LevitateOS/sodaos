package auth

import (
	"crypto/sha256"
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

func validateCancelLoginHeaders(s *Service, w http.ResponseWriter, r *http.Request) (int64, bool) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		w.Header().Set("Allow", "GET, POST")
		JSONError(w, 405, "method_not_allowed", "HTTP method not supported.")
		return 0, false
	}
	if !validateCancelLoginMetadata(r) || !validCancelLoginOrigin(s.Config.ForgejoURL, r) {
		JSONError(w, 403, "invalid_origin", "Use the same-origin sign-out action.")
		return 0, false
	}
	actor, ok := PositiveID(r.Header.Get(ExpectedUserHeader))
	if !ok || len(r.Header.Values(ExpectedUserHeader)) != 1 {
		JSONError(w, 400, "invalid_actor_context", "Provide one expected Forgejo user ID.")
		return 0, false
	}
	return actor, true
}

func cancelLoginCSRF(cookieValue string) string {
	digest := sha256.Sum256([]byte("soda-cancel-login\x00" + cookieValue))
	return base64.RawURLEncoding.EncodeToString(digest[:])
}

func resolveCancelLoginCookie(w http.ResponseWriter, r *http.Request) (string, string, bool) {
	if _, err := RequestCookie(r, SessionCookie); err != nil && !errors.Is(err, http.ErrNoCookie) {
		JSONError(w, 400, "invalid_cookie", "Ambiguous Soda session cookie.")
		return "", "", false
	}
	cookie, err := RequestCookie(r, OAuthCookie)
	if errors.Is(err, http.ErrNoCookie) {
		if r.Method != http.MethodGet {
			JSONError(w, 403, "invalid_cookie", "Sign-in cookie is unavailable.")
			return "", "", false
		}
		w.WriteHeader(http.StatusNoContent)
		return "", "", false
	}
	if err != nil {
		JSONError(w, 400, "invalid_cookie", "Ambiguous Soda sign-in cookie.")
		return "", "", false
	}
	return cookie.Value, cancelLoginCSRF(cookie.Value), true
}

func (s *Service) validateCancelLoginPOST(w http.ResponseWriter, r *http.Request, csrf string) bool {
	if !s.ValidAPIMutation(r, csrf) || r.Header.Get("Content-Type") != "application/json" {
		JSONError(w, 403, "invalid_csrf", "Request origin or CSRF token is invalid.")
		return false
	}
	r.Body = http.MaxBytesReader(w, r.Body, APIBodyLimit)
	if !DecodeAPIObject(w, r, &struct{}{}) {
		return false
	}
	if s.Store == nil {
		JSONError(w, 503, "store_unavailable", "Soda session storage is unavailable.")
		return false
	}
	return true
}

func (s *Service) verifyCancelSessionContext(r *http.Request, contextID string) error {
	current, cookieErr := RequestCookie(r, SessionCookie)
	if cookieErr != nil {
		if errors.Is(cookieErr, http.ErrNoCookie) {
			return nil
		}
		return cookieErr
	}
	session, err := s.Store.Session(r.Context(), current.Value)
	if errors.Is(err, store.ErrNotFound) {
		return nil
	}
	if err == nil && session.ContextID != contextID {
		return store.ErrLoginContext
	}
	return err
}

func (s *Service) executeCancelLogin(r *http.Request, cookieVal string, actor int64) error {
	var err error
	s.withSessionEndGate(func() {
		var id string
		var contextActor int64
		id, contextActor, err = s.Store.OAuthCancellationContext(r.Context(), cookieVal)
		if err != nil {
			return
		}
		if contextActor != 0 && contextActor != actor {
			err = store.ErrLoginContext
			return
		}
		if err = s.verifyCancelSessionContext(r, id); err != nil {
			return
		}
		if err = s.Store.EndLoginContext(r.Context(), id); err != nil {
			return
		}
		s.cancelTerminals(id, "")
	})
	return err
}

// Anonymous cancellation proves possession of the unique pending OAuth cookie,
// not a caller-selected actor/context. The custom bootstrap header and same-origin
// fetch metadata prevent cross-origin disclosure without adding anonymous state.
func (s *Service) cancelLogin(w http.ResponseWriter, r *http.Request) {
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
		JSONResponse(w, 200, struct {
			CSRF string `json:"csrf_token"`
		}{csrf})
		return
	}
	if !s.validateCancelLoginPOST(w, r, csrf) {
		return
	}
	if err := s.executeCancelLogin(r, cookieVal, actor); err != nil {
		JSONError(w, 409, "cancellation_unconfirmed", "Could not confirm sign-in cancellation. Retry sign-out.")
		return
	}
	s.cookie(w, SessionCookie, "", -1)
	s.cookie(w, OAuthCookie, "", -1)
	w.WriteHeader(http.StatusNoContent)
}
