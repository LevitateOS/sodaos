package webauth

import (
	"context"
	"crypto/subtle"
	"errors"
	"mime"
	"net/http"
	"strconv"
	"strings"

	"github.com/levitateos/sodaos/internal/store"
)

func (s *Service) sessionRoutes() {
	// Path-only registrations let us return JSON 405s rather than ServeMux's
	// HTML/text method errors. Unknown API routes must never become SPA HTML.
	s.mux.HandleFunc("/api/session", s.Protected(s.apiSession, http.MethodGet))
	s.mux.HandleFunc("/api/session/logout", s.Protected(s.apiLogout, http.MethodPost))
	s.mux.HandleFunc("/api/me/preferences", s.Protected(s.apiPreferences, http.MethodGet, http.MethodPatch))
	s.mux.HandleFunc("/api/me/development-keys", s.Protected(s.apiKeys, http.MethodGet, http.MethodPost))
	s.mux.HandleFunc("/api/me/development-keys/{key}", s.Protected(s.apiRemoveDevelopmentKey, http.MethodDelete))
	s.mux.HandleFunc("/api/me/forgejo-keys", s.Provider(s.apiForgejoKeys, "read:user", http.MethodGet))
	s.forgejoRoutes()
}

func (s *Service) loadAPISession(w http.ResponseWriter, r *http.Request) (store.Session, bool) {
	cookie, err := RequestCookie(r, SessionCookie)
	if err != nil || cookie.Value == "" {
		JSONError(w, http.StatusUnauthorized, "unauthenticated", "Sign in through Forgejo.")
		return store.Session{}, false
	}
	if s.Store == nil {
		JSONError(w, http.StatusServiceUnavailable, "store_unavailable", "Soda session storage is unavailable.")
		return store.Session{}, false
	}
	session, err := s.Store.Session(r.Context(), cookie.Value)
	if errors.Is(err, store.ErrNotFound) {
		JSONError(w, http.StatusUnauthorized, "unauthenticated", "Session expired; sign in again.")
		return store.Session{}, false
	}
	if err != nil {
		JSONError(w, http.StatusServiceUnavailable, "store_unavailable", "Soda session storage is unavailable.")
		return store.Session{}, false
	}
	return session, true
}

func checkExpectedActor(w http.ResponseWriter, r *http.Request, session store.Session) bool {
	expected := r.Header.Values(ExpectedUserHeader)
	if len(expected) == 0 && r.URL.Path == "/api/session" && r.Method == http.MethodGet {
		return true
	}
	if len(expected) != 1 {
		JSONError(w, 400, "invalid_actor_context", "Provide one expected Forgejo user ID.")
		return false
	}
	id, ok := PositiveID(expected[0])
	if !ok || id != session.User.ID {
		if !ok {
			JSONError(w, 400, "invalid_actor_context", "Provide one expected Forgejo user ID.")
		} else {
			JSONError(w, 403, "identity_mismatch", "Soda is signed in as a different user. Reload the repository and sign in again.")
		}
		return false
	}
	return true
}

func (s *Service) constrainAPIBody(w http.ResponseWriter, r *http.Request, session store.Session) bool {
	if r.Method == http.MethodGet || r.Method == http.MethodHead {
		return true
	}
	if !s.validAPIMutation(r, session.CSRF) {
		JSONError(w, http.StatusForbidden, "invalid_csrf", "Request origin or CSRF token is invalid.")
		return false
	}
	contentType, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || contentType != "application/json" || (params["charset"] != "" && !strings.EqualFold(params["charset"], "utf-8")) {
		JSONError(w, http.StatusUnsupportedMediaType, "unsupported_media_type", "Send a UTF-8 application/json object.")
		return false
	}
	r.Body = http.MaxBytesReader(w, r.Body, APIBodyLimit)
	return true
}

// Protected authenticates the Soda session, expected actor and mutation guards.
func (s *Service) Protected(next func(http.ResponseWriter, *http.Request, store.Session), methods ...string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		if !AllowAPIMethod(w, r, methods) {
			return
		}
		session, ok := s.loadAPISession(w, r)
		if !ok {
			return
		}
		if !checkExpectedActor(w, r, session) {
			return
		}
		if !s.constrainAPIBody(w, r, session) {
			return
		}
		next(w, r, session)
	}
}

// RequireCurrentSession performs a fresh store read at the caller's post-I/O
// boundary. It does not authorize a role or replace terminal admission locking.
// Cookie disambiguation and response/status policy remain with the caller.
func (s *Service) RequireCurrentSession(ctx context.Context, token string, original store.Session) error {
	current, err := s.Store.Session(ctx, token)
	if err != nil {
		return err
	}
	if current.User.ID != original.User.ID || current.ContextID != original.ContextID || current.CSRF != original.CSRF {
		return store.ErrGrantUnavailable
	}
	return nil
}

func (s *Service) validAPIMutation(r *http.Request, csrf string) bool {
	return s.ValidAPIMutation(r, csrf)
}

// ValidAPIMutation checks Origin/CSRF for browser mutations against the Forgejo origin.
func (s *Service) ValidAPIMutation(r *http.Request, csrf string) bool {
	origins, tokens := r.Header.Values("Origin"), r.Header.Values("X-CSRF-Token")
	if s.Config.ForgejoURL == "" || len(origins) != 1 || origins[0] != s.Config.ForgejoURL || len(tokens) != 1 || csrf == "" {
		return false
	}
	if site := r.Header.Get("Sec-Fetch-Site"); site != "" && site != "same-origin" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(tokens[0]), []byte(csrf)) == 1
}

type sessionUserView struct {
	ID              string `json:"id"`
	Login           string `json:"login"`
	SodaDisplayName string `json:"soda_display_name"`
}

type sessionView struct {
	User         sessionUserView `json:"user"`
	CSRFToken    string          `json:"csrf_token"`
	SodaOperator bool            `json:"soda_operator"`
	ForgejoURL   string          `json:"forgejo_url"`
}

func (s *Service) apiSession(w http.ResponseWriter, r *http.Request, v store.Session) {
	JSONResponse(w, http.StatusOK, sessionView{
		User:      sessionUserView{strconv.FormatInt(v.User.ID, 10), v.User.Login, v.User.Name},
		CSRFToken: v.CSRF, SodaOperator: v.User.ID == s.Config.OperatorID, ForgejoURL: s.Config.ForgejoURL,
	})
}

func (s *Service) apiLogout(w http.ResponseWriter, r *http.Request, v store.Session) {
	if !DecodeAPIObject(w, r, &struct{}{}) {
		return
	}
	var err error
	s.withSessionEndGate(func() {
		err = s.Store.EndLoginContext(r.Context(), v.ContextID)
		if err == nil {
			s.cancelTerminals(v.ContextID, "")
		}
	})
	if err != nil {
		JSONError(w, http.StatusServiceUnavailable, "store_unavailable", "Could not end this session.")
		return
	}
	s.cookie(w, SessionCookie, "", -1)
	s.cookie(w, OAuthCookie, "", -1)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Service) apiPreferences(w http.ResponseWriter, r *http.Request, v store.Session) {
	type preferences struct {
		DisplayName string `json:"display_name"`
	}
	if r.Method == http.MethodGet {
		JSONResponse(w, http.StatusOK, preferences{v.User.Name})
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
	if err := s.Store.RenameProfile(r.Context(), v.User.ID, name); err != nil {
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
	if !DecodeAPIObject(w, r, &struct{}{}) {
		return
	}
	removed, err := s.Store.RemoveKey(r.Context(), v.User.ID, id)
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
