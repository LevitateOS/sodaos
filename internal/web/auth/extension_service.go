package auth

import (
	"context"
	"mime"
	"net/http"
	"path"
	"strings"
	"time"

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
	authority, id, ok := extensionIdentity(w, r)
	if !ok {
		return
	}
	if r.URL.Path == "/api/session" {
		s.serveExtensionSession(w, r, authority, id)
		return
	}
	s.serveExtensionResource(w, r, authority, id)
}

func (s *Service) serveExtensionSession(w http.ResponseWriter, r *http.Request, authority extensions.Authority, id int64) {
	if !allowExtensionNoQuery(w, r) {
		return
	}
	request, ok := s.extensionRequest(w, r, authority, id)
	if !ok {
		return
	}
	writeExtensionSession(w, request, s.Config)
}

func (s *Service) serveExtensionResource(w http.ResponseWriter, r *http.Request, authority extensions.Authority, id int64) {
	if !extensionSessionGenerationMatches(r, authority) {
		JSONError(w, http.StatusConflict, "extension_session_changed", "The native extension page is no longer current.")
		return
	}
	if r.Method != http.MethodGet && !s.extensionMutation(w, r) {
		return
	}
	request, ok := s.extensionRequest(w, r, authority, id)
	if !ok {
		return
	}
	session := store.Session{User: request.user, ContextID: authority.SessionGeneration}
	s.serveExtensionAccount(w, r, request, session)
}

func (s *Service) serveExtensionAccount(w http.ResponseWriter, r *http.Request, request extensionRequest, session store.Session) {
	switch {
	case r.URL.Path == "/api/me/preferences":
		s.serveExtensionPreferences(w, r, request.user)
	case r.URL.Path == "/api/me/development-keys":
		s.serveExtensionDevelopmentKeys(w, r, session)
	case strings.HasPrefix(r.URL.Path, "/api/me/development-keys/"):
		s.serveExtensionDevelopmentKeyRemoval(w, r, session)
	case r.URL.Path == "/api/me/forgejo-keys":
		s.apiExtensionForgejoKeys(w, r, request)
	default:
		http.NotFound(w, r)
	}
}

func (s *Service) serveExtensionPreferences(w http.ResponseWriter, r *http.Request, user store.User) {
	if !allowExtensionNoQuery(w, r) {
		return
	}
	s.preferences(w, r, user, func() bool {
		return revalidateExtensionAuthority(w, r)
	})
}

func (s *Service) serveExtensionDevelopmentKeys(w http.ResponseWriter, r *http.Request, session store.Session) {
	if !allowExtensionNoQuery(w, r) {
		return
	}
	if r.Method != http.MethodGet && !revalidateExtensionAuthority(w, r) {
		return
	}
	s.apiKeys(w, r, session)
}

func (s *Service) serveExtensionDevelopmentKeyRemoval(w http.ResponseWriter, r *http.Request, session store.Session) {
	if !allowExtensionNoQuery(w, r) || !revalidateExtensionAuthority(w, r) {
		return
	}
	s.apiRemoveDevelopmentKey(w, r, session)
}

func allowExtensionNoQuery(w http.ResponseWriter, r *http.Request) bool {
	if r.URL.RawQuery != "" || r.URL.ForceQuery {
		JSONError(w, http.StatusBadRequest, "invalid_request", "No query parameters are accepted.")
		return false
	}
	return true
}

func validExtensionPath(r *http.Request) bool {
	return r.URL.RawPath == "" && !r.URL.ForceQuery &&
		!strings.Contains(r.URL.Path, "\\") && path.Clean(r.URL.Path) == r.URL.Path
}

func allowExtensionMethod(w http.ResponseWriter, r *http.Request) bool {
	var methods []string
	switch r.URL.Path {
	case "/api/session":
		methods = []string{http.MethodGet}
	case "/api/me/preferences":
		methods = []string{http.MethodGet, http.MethodPatch}
	case "/api/me/development-keys":
		methods = []string{http.MethodGet, http.MethodPost}
	case "/api/me/forgejo-keys":
		methods = []string{http.MethodGet}
	default:
		if !extensionDevelopmentKeyPath(r.URL.Path) {
			http.NotFound(w, r)
			return false
		}
		methods = []string{http.MethodDelete}
	}
	return AllowAPIMethod(w, r, methods)
}

func extensionDevelopmentKeyPath(path string) bool {
	parts := strings.Split(path, "/")
	return len(parts) == 5 && parts[1] == "api" && parts[2] == "me" && parts[3] == "development-keys" && parts[4] != ""
}

func (s *Service) extensionRequest(w http.ResponseWriter, r *http.Request, authority extensions.Authority, id int64) (extensionRequest, bool) {
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

func extensionSessionGenerationMatches(r *http.Request, authority extensions.Authority) bool {
	values := r.Header.Values(extensions.SessionGenerationHeader)
	return len(values) == 1 && values[0] == authority.SessionGeneration
}

func (s *Service) apiExtensionForgejoKeys(w http.ResponseWriter, r *http.Request, request extensionRequest) {
	page, ok := parseForgejoKeysPage(r.URL.RawQuery)
	if !ok {
		JSONError(w, http.StatusBadRequest, "invalid_page", "Select page 1 through 8 of your own public keys.")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	keys, err := request.authority.Native().PublicSSHKeys(ctx)
	if err != nil || len(keys) > 100 {
		JSONError(w, http.StatusServiceUnavailable, "forgejo_keys_unavailable", "Native Forgejo public keys are unavailable.")
		return
	}
	start := min(int(page-1)*10, len(keys))
	end := min(start+10, len(keys))
	items := make([]profileKeyView, 0, end-start)
	for _, key := range keys[start:end] {
		public, fingerprint, err := NormalizeDevelopmentKey(key.Key)
		if err != nil {
			JSONError(w, http.StatusServiceUnavailable, "invalid_profile_key", "Forgejo returned an unsupported development public key. No keys were imported.")
			return
		}
		items = append(items, profileKeyView{developmentKeyView{key.ID, public, fingerprint}, "Forgejo public key"})
	}
	if ctx.Err() != nil || !revalidateExtensionAuthority(w, r) {
		return
	}
	JSONResponse(w, http.StatusOK, struct {
		Items []profileKeyView `json:"items"`
		Page  int64            `json:"page"`
		More  bool             `json:"more"`
	}{items, page, end < len(keys)})
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

func (s *Service) extensionMutation(w http.ResponseWriter, r *http.Request) bool {
	if !validExtensionMutationOrigin(r, s.Config.ForgejoURL) {
		JSONError(w, http.StatusForbidden, "invalid_origin", "Same-origin native extension mutation required.")
		return false
	}
	contentType, parameters, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || contentType != "application/json" || (parameters["charset"] != "" && !strings.EqualFold(parameters["charset"], "utf-8")) {
		JSONError(w, http.StatusUnsupportedMediaType, "unsupported_media_type", "Send a UTF-8 application/json object.")
		return false
	}
	r.Body = http.MaxBytesReader(w, r.Body, APIBodyLimit)
	return true
}

func validExtensionMutationOrigin(r *http.Request, forgejoURL string) bool {
	origins := r.Header.Values("Origin")
	sites := r.Header.Values("Sec-Fetch-Site")
	return forgejoURL != "" && len(origins) == 1 && origins[0] == forgejoURL &&
		len(sites) <= 1 && (len(sites) == 0 || sites[0] == "same-origin")
}
