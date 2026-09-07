package web

import (
	"bytes"
	"crypto/subtle"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"

	"github.com/levitateos/sodaos/internal/store"
)

const apiBodyLimit = 65536

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func jsonResponse(w http.ResponseWriter, status int, value any) {
	// Encode before committing headers so malformed server values cannot produce
	// an apparently successful, truncated JSON response.
	body, err := json.Marshal(value)
	if err != nil {
		status = http.StatusInternalServerError
		body = []byte(`{"error":{"code":"internal_error","message":"Cannot encode response."}}`)
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_, _ = w.Write(append(body, '\n'))
}

func jsonError(w http.ResponseWriter, status int, code, message string) {
	jsonResponse(w, status, struct {
		Error apiError `json:"error"`
	}{apiError{Code: code, Message: message}})
}

func (s *Server) apiRoutes() {
	// Path-only registrations let us return JSON 405s rather than ServeMux's
	// HTML/text method errors. Unknown API routes must never become SPA HTML.
	s.mux.HandleFunc("/api/session", s.apiProtected(s.apiSession, http.MethodGet))
	s.mux.HandleFunc("/api/session/logout", s.apiProtected(s.apiLogout, http.MethodPost))
	s.mux.HandleFunc("/api/me/preferences", s.apiProtected(s.apiPreferences, http.MethodGet, http.MethodPatch))
	s.mux.HandleFunc("/api/me/development-keys", s.apiProtected(s.apiKeys, http.MethodGet, http.MethodPost))
	s.forgejoRoutes()
	s.environmentRoutes()
	notFound := func(w http.ResponseWriter, r *http.Request) {
		jsonError(w, http.StatusNotFound, "not_found", "API route not found.")
	}
	s.mux.HandleFunc("/api", notFound)
	s.mux.HandleFunc("/api/", notFound)
}

func (s *Server) apiProtected(next func(http.ResponseWriter, *http.Request, store.Session), methods ...string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		allowed := false
		for _, method := range methods {
			allowed = allowed || r.Method == method
		}
		if !allowed {
			w.Header().Set("Allow", strings.Join(methods, ", "))
			jsonError(w, http.StatusMethodNotAllowed, "method_not_allowed", "HTTP method not supported.")
			return
		}
		cookie, err := requestCookie(r, sessionCookie)
		if err != nil || cookie.Value == "" {
			jsonError(w, http.StatusUnauthorized, "unauthenticated", "Sign in through Forgejo.")
			return
		}
		if s.Store == nil {
			jsonError(w, http.StatusServiceUnavailable, "store_unavailable", "Soda session storage is unavailable.")
			return
		}
		session, err := s.Store.Session(r.Context(), cookie.Value)
		if errors.Is(err, sql.ErrNoRows) {
			jsonError(w, http.StatusUnauthorized, "unauthenticated", "Session expired; sign in again.")
			return
		}
		if err != nil {
			jsonError(w, http.StatusServiceUnavailable, "store_unavailable", "Soda session storage is unavailable.")
			return
		}
		// A page must declare its expected actor, but this hint never selects a
		// session or grants permission. Only session bootstrap may omit it.
		expected := r.Header.Values(expectedUserHeader)
		if len(expected) != 0 || r.URL.Path != "/api/session" || r.Method != http.MethodGet {
			if len(expected) != 1 {
				jsonError(w, 400, "invalid_actor_context", "Provide one expected Forgejo user ID.")
				return
			}
			id, ok := positiveID(expected[0])
			if !ok {
				jsonError(w, 400, "invalid_actor_context", "Provide one expected Forgejo user ID.")
				return
			}
			if id != session.User.ID {
				jsonError(w, 403, "identity_mismatch", "Soda is signed in as a different user. Reload the repository and sign in again.")
				return
			}
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			if !s.validAPIMutation(r, session.CSRF) {
				jsonError(w, http.StatusForbidden, "invalid_csrf", "Request origin or CSRF token is invalid.")
				return
			}
			contentType, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
			if err != nil || contentType != "application/json" || (params["charset"] != "" && !strings.EqualFold(params["charset"], "utf-8")) {
				jsonError(w, http.StatusUnsupportedMediaType, "unsupported_media_type", "Send a UTF-8 application/json object.")
				return
			}
			r.Body = http.MaxBytesReader(w, r.Body, apiBodyLimit)
		}
		next(w, r, session)
	}
}

func (s *Server) validAPIMutation(r *http.Request, csrf string) bool {
	origins, tokens := r.Header.Values("Origin"), r.Header.Values("X-CSRF-Token")
	if s.Config.ForgejoURL == "" || len(origins) != 1 || origins[0] != s.Config.ForgejoURL || len(tokens) != 1 || csrf == "" {
		return false
	}
	if site := r.Header.Get("Sec-Fetch-Site"); site != "" && site != "same-origin" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(tokens[0]), []byte(csrf)) == 1
}

func decodeAPIObject(w http.ResponseWriter, r *http.Request, out any) bool {
	body, err := io.ReadAll(r.Body)
	var oversized *http.MaxBytesError
	if errors.As(err, &oversized) {
		jsonError(w, http.StatusRequestEntityTooLarge, "body_too_large", "Request exceeds 64 KiB.")
		return false
	}
	body = bytes.TrimSpace(body)
	if err != nil || len(body) == 0 || body[0] != '{' {
		jsonError(w, http.StatusBadRequest, "invalid_json", "Send exactly one JSON object.")
		return false
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(out); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid_json", "JSON fields or values are invalid.")
		return false
	}
	if err = decoder.Decode(new(any)); err != io.EOF {
		jsonError(w, http.StatusBadRequest, "invalid_json", "Send exactly one JSON object.")
		return false
	}
	return true
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

func (s *Server) apiSession(w http.ResponseWriter, r *http.Request, v store.Session) {
	jsonResponse(w, http.StatusOK, sessionView{
		User:      sessionUserView{strconv.FormatInt(v.User.ID, 10), v.User.Login, v.User.Name},
		CSRFToken: v.CSRF, SodaOperator: v.User.ID == s.Config.OperatorID, ForgejoURL: s.Config.ForgejoURL,
	})
}

func (s *Server) apiLogout(w http.ResponseWriter, r *http.Request, v store.Session) {
	if !decodeAPIObject(w, r, &struct{}{}) {
		return
	}
	if err := s.Store.EndLoginContext(r.Context(), v.ContextID); err != nil {
		jsonError(w, http.StatusServiceUnavailable, "store_unavailable", "Could not end this session.")
		return
	}
	s.cookie(w, sessionCookie, "", -1)
	s.cookie(w, oauthCookie, "", -1)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) apiPreferences(w http.ResponseWriter, r *http.Request, v store.Session) {
	type preferences struct {
		DisplayName string `json:"display_name"`
	}
	if r.Method == http.MethodGet {
		jsonResponse(w, http.StatusOK, preferences{v.User.Name})
		return
	}
	var input struct {
		DisplayName *string `json:"display_name"`
	}
	if !decodeAPIObject(w, r, &input) {
		return
	}
	if input.DisplayName == nil || len(strings.TrimSpace(*input.DisplayName)) > 200 {
		jsonError(w, http.StatusBadRequest, "invalid_display_name", "Provide a Soda display name of at most 200 bytes.")
		return
	}
	name := strings.TrimSpace(*input.DisplayName)
	if err := s.Store.RenameProfile(r.Context(), v.User.ID, name); err != nil {
		jsonError(w, http.StatusServiceUnavailable, "store_unavailable", "Could not save Soda preferences.")
		return
	}
	jsonResponse(w, http.StatusOK, preferences{name})
}

type developmentKeyView struct {
	ID          string `json:"id"`
	PublicKey   string `json:"public_key"`
	Fingerprint string `json:"fingerprint"`
}

func (s *Server) apiKeys(w http.ResponseWriter, r *http.Request, v store.Session) {
	if r.Method == http.MethodPost {
		var input struct {
			PublicKey string `json:"public_key"`
		}
		if !decodeAPIObject(w, r, &input) {
			return
		}
		public, fingerprint, err := normalizeDevelopmentKey(input.PublicKey)
		if err != nil {
			jsonError(w, http.StatusBadRequest, "invalid_public_key", "Provide one public SSH key without authorized_keys options. Never submit a private key.")
			return
		}
		if err = s.Store.AddKey(r.Context(), v.User.ID, public, fingerprint); err != nil {
			jsonError(w, http.StatusServiceUnavailable, "store_unavailable", "Could not register development key.")
			return
		}
	}
	keys, err := s.Store.Keys(r.Context(), v.User.ID)
	if err != nil {
		jsonError(w, http.StatusServiceUnavailable, "store_unavailable", "Could not read development keys.")
		return
	}
	items := make([]developmentKeyView, 0, len(keys))
	for _, key := range keys {
		items = append(items, developmentKeyView{strconv.FormatInt(key.ID, 10), key.Public, key.Fingerprint})
	}
	jsonResponse(w, http.StatusOK, struct {
		Items []developmentKeyView `json:"items"`
	}{items})
}
