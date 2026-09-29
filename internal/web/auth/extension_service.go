package auth

import (
	"mime"
	"net/http"
	"path"
	"strings"

	extensions "forgejo.org/extension-sdk"
	"github.com/levitateos/sodaos/internal/config"
	"github.com/levitateos/sodaos/internal/store"
)

type extensionRequest struct {
	authority extensions.Authority
	user      store.User
	userID    int64
}

// ExtensionHandler serves the bounded read and preference mutation available
// to the private native extension listener.
func (s *Service) ExtensionHandler() http.Handler {
	return http.HandlerFunc(s.serveExtension)
}

func (s *Service) serveExtension(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if !validExtensionPath(r) {
		http.NotFound(w, r)
		return
	}
	if !allowExtensionMethod(w, r) {
		return
	}
	request, ok := s.extensionRequest(w, r)
	if !ok {
		return
	}
	if r.URL.Path == "/api/session" {
		writeExtensionSession(w, request, s.Config)
		return
	}
	s.preferences(w, r, request.user, func() bool {
		return revalidateExtensionAuthority(w, r)
	})
}

func validExtensionPath(r *http.Request) bool {
	return r.URL.RawPath == "" && r.URL.RawQuery == "" && !r.URL.ForceQuery &&
		!strings.Contains(r.URL.Path, "\\") && path.Clean(r.URL.Path) == r.URL.Path
}

func allowExtensionMethod(w http.ResponseWriter, r *http.Request) bool {
	var methods []string
	switch r.URL.Path {
	case "/api/session":
		methods = []string{http.MethodGet}
	case "/api/me/preferences":
		methods = []string{http.MethodGet, http.MethodPatch}
	default:
		http.NotFound(w, r)
		return false
	}
	return AllowAPIMethod(w, r, methods)
}

func (s *Service) extensionRequest(w http.ResponseWriter, r *http.Request) (extensionRequest, bool) {
	authority, id, ok := extensionIdentity(w, r)
	if !ok {
		return extensionRequest{}, false
	}
	user, ok := s.extensionUser(w, r, id, authority.Actor.Username)
	if !ok {
		return extensionRequest{}, false
	}
	return extensionRequest{authority: authority, user: user, userID: id}, true
}

func extensionIdentity(w http.ResponseWriter, r *http.Request) (extensions.Authority, int64, bool) {
	authority, err := ExtensionAuthority(r)
	if err != nil || !ExtensionContribution(authority.Contribution) {
		JSONError(w, http.StatusForbidden, "native_authority_unavailable", "Current native authority is required.")
		return extensions.Authority{}, 0, false
	}
	id, valid := PositiveID(authority.Actor.ID)
	if !valid {
		JSONError(w, http.StatusForbidden, "invalid_actor", "Native actor is invalid.")
		return extensions.Authority{}, 0, false
	}
	return authority, id, true
}

func (s *Service) extensionUser(w http.ResponseWriter, r *http.Request, id int64, login string) (store.User, bool) {
	if r.Method == http.MethodPatch && !extensionMutation(w, r) {
		return store.User{}, false
	}
	if s.Store == nil || s.Store.UpsertUser(r.Context(), store.User{ID: id, Login: login}) != nil {
		JSONError(w, http.StatusServiceUnavailable, "store_unavailable", "Soda profile storage is unavailable.")
		return store.User{}, false
	}
	user, err := s.Store.User(r.Context(), id)
	if err != nil {
		JSONError(w, http.StatusServiceUnavailable, "store_unavailable", "Soda profile storage is unavailable.")
		return store.User{}, false
	}
	if r.Method == http.MethodGet && !revalidateExtensionAuthority(w, r) {
		return store.User{}, false
	}
	return user, true
}

func writeExtensionSession(w http.ResponseWriter, request extensionRequest, config *config.Config) {
	JSONResponse(w, http.StatusOK, struct {
		User              sessionUserView `json:"user"`
		SessionGeneration string          `json:"session_generation"`
		SodaOperator      bool            `json:"soda_operator"`
		ForgejoURL        string          `json:"forgejo_url"`
	}{
		sessionUserView{request.authority.Actor.ID, request.authority.Actor.Username, request.user.Name},
		request.authority.SessionGeneration,
		request.userID == config.OperatorID,
		config.ForgejoURL,
	})
}

func revalidateExtensionAuthority(w http.ResponseWriter, r *http.Request) bool {
	if _, err := ExtensionAuthority(r); err != nil {
		JSONError(w, http.StatusForbidden, "native_authority_unavailable", "Current native authority is required.")
		return false
	}
	return true
}

func extensionMutation(w http.ResponseWriter, r *http.Request) bool {
	contentType, parameters, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || contentType != "application/json" || (parameters["charset"] != "" && !strings.EqualFold(parameters["charset"], "utf-8")) {
		JSONError(w, http.StatusUnsupportedMediaType, "unsupported_media_type", "Send a UTF-8 application/json object.")
		return false
	}
	r.Body = http.MaxBytesReader(w, r.Body, APIBodyLimit)
	return true
}
