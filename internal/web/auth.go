package web

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/levitateos/sodaos/internal/config"
	"github.com/levitateos/sodaos/internal/forgejo"
	"github.com/levitateos/sodaos/internal/store"
)

func token() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

const (
	sessionCookie      = "__Secure-sodaspaces-session"
	oauthCookie        = "__Secure-sodaspaces-oauth"
	expectedUserHeader = "X-Soda-Expected-User-ID"
)

// Ignore native Forgejo and legacy standalone Soda cookies. Duplicate names
// cannot select an actor or OAuth transaction by header order.
func requestCookie(r *http.Request, name string) (*http.Cookie, error) {
	cookies := r.CookiesNamed(name)
	if len(cookies) == 0 {
		return nil, http.ErrNoCookie
	}
	if len(cookies) != 1 || cookies[0].Value == "" || len(cookies[0].Value) > 128 {
		return nil, errors.New("invalid Soda cookie")
	}
	return cookies[0], nil
}

func (s *Server) cookie(w http.ResponseWriter, name, value string, seconds int) {
	http.SetCookie(w, &http.Cookie{Name: name, Value: value, Path: config.SodaPath + "/", MaxAge: seconds, HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode})
}
func (s *Server) authRoutes() {
	s.mux.HandleFunc("/api/login/cancel", s.cancelLogin)
	s.mux.HandleFunc("GET /login", s.login)
	s.mux.HandleFunc("GET /oauth/callback", s.callback)
}

// Native IDs have one bounded decimal representation; they are never usernames
// or caller-selected privileges. Zero represents absence only in stored state.
func positiveID(value string) (int64, bool) {
	if len(value) == 0 || len(value) > 19 {
		return 0, false
	}
	id, err := strconv.ParseInt(value, 10, 64)
	return id, err == nil && id > 0 && strconv.FormatInt(id, 10) == value
}

func oauthContextID(query url.Values, name string) (int64, bool) {
	values, present := query[name]
	if !present {
		return 0, true
	}
	if len(values) != 1 {
		return 0, false
	}
	return positiveID(values[0])
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
	if len(r.URL.RawQuery) > 8192 {
		http.Error(w, "Invalid sign-in request.", 400)
		return
	}
	query, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		http.Error(w, "Invalid sign-in request.", 400)
		return
	}
	if len(query["return_to"]) > 1 || query.Get("return_to") != "" {
		http.Error(w, "Unsupported sign-in destination.", http.StatusBadRequest)
		return
	}
	destination, hasDestination := query["destination"]
	if hasDestination && (len(destination) != 1 || (destination[0] != "spaces" && destination[0] != "runners" && destination[0] != "repository-spaces") || (destination[0] != "repository-spaces" && query.Has("repository_id"))) {
		http.Error(w, "Unsupported sign-in destination.", http.StatusBadRequest)
		return
	}
	repositoryID, repoOK := oauthContextID(query, "repository_id")
	expectedUserID, userOK := oauthContextID(query, "expected_user_id")
	if !repoOK || !userOK || (query.Get("destination") == "repository-spaces" && repositoryID == 0) {
		http.Error(w, "Invalid repository or expected user ID.", 400)
		return
	}
	settingsReturn := ""
	if query.Get("destination") == "runners" {
		settingsReturn = "runners"
	}
	state, verifier := token(), token()
	login := store.OAuthLogin{RepositorySettingsReturn: query.Get("destination") == "repository-spaces", Verifier: verifier, RepositoryID: repositoryID, ExpectedUserID: expectedUserID, SpacesReturn: query.Get("destination") == "spaces", SettingsReturn: settingsReturn}
	var session, previous string
	for name, target := range map[string]*string{sessionCookie: &session, oauthCookie: &previous} {
		c, err := requestCookie(r, name)
		if err == nil {
			*target = c.Value
		} else if !errors.Is(err, http.ErrNoCookie) {
			s.loginFailure(w, r, login, "Ambiguous Soda cookies; clear them and sign in again.", 400)
			return
		}
	}
	if err := s.Store.BeginOAuth(r.Context(), state, login, session, previous); err != nil {
		if errors.Is(err, store.ErrLoginContext) {
			s.cookie(w, sessionCookie, "", -1)
			s.cookie(w, oauthCookie, "", -1)
			s.loginFailure(w, r, login, "Previous sign-in expired or ended; start sign-in again.", 409)
		} else {
			s.loginFailure(w, r, login, "Cannot begin sign-in.", 500)
		}
		return
	}
	challenge := sha256.Sum256([]byte(verifier))
	s.cookie(w, oauthCookie, state, 600)
	scopes := "read:user read:repository read:organization"
	q := url.Values{"client_id": {s.Config.OAuthClientID}, "redirect_uri": {s.Config.OAuthCallbackURL()}, "response_type": {"code"}, "scope": {scopes}, "state": {state}, "code_challenge": {base64.RawURLEncoding.EncodeToString(challenge[:])}, "code_challenge_method": {"S256"}}
	http.Redirect(w, r, s.Config.ForgejoURL+"/login/oauth/authorize?"+q.Encode(), 302)
}
func (s *Server) callback(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
	if len(r.URL.RawQuery) > 8192 {
		s.callbackError(w, r, "Invalid sign-in response.", 400)
		return
	}
	query, queryErr := url.ParseQuery(r.URL.RawQuery)
	c, err := requestCookie(r, oauthCookie)
	state := query.Get("state")
	if queryErr != nil || len(query["state"]) != 1 || len(query["code"]) > 1 || len(query.Get("code")) > 4096 ||
		err != nil || state == "" || len(state) > 128 || subtle.ConstantTimeCompare([]byte(state), []byte(c.Value)) != 1 {
		s.callbackError(w, r, "Invalid sign-in state; sign in again.", 400)
		return
	}
	old, oldErr := requestCookie(r, sessionCookie)
	if oldErr != nil && !errors.Is(oldErr, http.ErrNoCookie) {
		s.callbackError(w, r, "Ambiguous Soda session; sign in again.", 400)
		return
	}
	var oldSession string
	if oldErr == nil {
		oldSession = old.Value
	}
	s.terminalMu.Lock()
	login, err := s.Store.ConsumeOAuth(r.Context(), state, oldSession)
	if err == nil {
		s.cancelTerminals("", oldSession)
	}
	s.terminalMu.Unlock()
	if err != nil {
		s.callbackError(w, r, "Sign-in expired or was already used.", 400)
		return
	}
	fail := func(message string, status int) {
		s.loginFailure(w, r, login.OAuthLogin, message, status)
	}
	code := query.Get("code")
	if code == "" {
		fail("Forgejo did not authorize sign-in.", 400)
		return
	}
	secret, err := config.Secret(s.Config.OAuthSecretFile)
	if err != nil {
		fail("Sign-in is not configured.", 503)
		return
	}
	grant, err := s.Forgejo.ExchangeGrant(r.Context(), s.Config.OAuthClientID, secret, code, s.Config.OAuthCallbackURL(), login.Verifier)
	if err != nil {
		fail("Forgejo sign-in failed.", 502)
		return
	}
	u, err := s.Forgejo.Current(r.Context(), grant.Access)
	if err != nil || u.ID <= 0 {
		fail("Could not identify this Forgejo user.", 502)
		return
	}
	if login.ExpectedUserID != 0 && u.ID != login.ExpectedUserID {
		fail("Forgejo account changed; reload the repository and sign in again.", http.StatusForbidden)
		return
	}
	scopes, err := s.Forgejo.GrantScopes(r.Context(), s.Config.OAuthClientID, secret, grant.Access, u.ID)
	if err != nil {
		fail("Could not verify Forgejo consent.", 502)
		return
	}
	if s.nativeOAuthReturn(login.OAuthLogin) != "" && (!forgejo.HasScope(scopes, "read:user") || !forgejo.HasScope(scopes, "read:repository") || !forgejo.HasScope(scopes, "read:organization")) {
		fail("Forgejo consent is missing required scopes.", http.StatusForbidden)
		return
	}
	// Only the consumed transaction selects the repository. Caller callback
	// parameters, cached names and provider-supplied URLs are not destinations.
	var repository *forgejo.Repository
	if !login.RepositorySettingsReturn && login.RepositoryID != 0 && forgejo.HasScope(scopes, "read:repository") {
		if repo, err := s.Forgejo.RepositoryByID(r.Context(), grant.Access, login.RepositoryID); err == nil {
			repository = &repo
		}
	}
	value, csrf := token(), token()
	if err = s.Store.FinishOAuth(r.Context(), login, store.User{ID: u.ID, Login: u.Login, Name: u.Name}, value, csrf, store.Grant{Access: grant.Access, Refresh: grant.Refresh, Scopes: scopes, Expires: grant.ExpiresAt}); err != nil {
		if errors.Is(err, store.ErrLoginContext) {
			fail("Sign-in cancelled, expired or superseded; reload and start again.", 409)
		} else {
			fail("Could not create session.", 500)
		}
		return
	}
	// Keep the short-lived OAuth cookie through callback completion so a logout
	// that began anonymously can still resolve this context after Set-Cookie.
	s.cookie(w, sessionCookie, value, int((12 * time.Hour).Seconds()))
	if destination := s.nativeOAuthReturn(login.OAuthLogin); destination != "" {
		http.Redirect(w, r, destination, http.StatusSeeOther)
		return
	}
	s.forgejoReturn(w, r, repository)
}

// Only persisted transaction fields select these fixed native rendering views.
func (s *Server) nativeOAuthReturn(login store.OAuthLogin) string {
	view := ""
	if login.RepositorySettingsReturn {
		view = "repository-spaces"
	} else if login.SettingsReturn == "runners" {
		view = "runners"
	} else if login.SpacesReturn {
		view = "spaces"
	}
	if view == "" {
		return ""
	}
	destination := s.Config.ForgejoURL + "/?soda-view=" + view
	if login.RepositorySettingsReturn {
		destination += "&repository_id=" + strconv.FormatInt(login.RepositoryID, 10)
	}
	return destination
}

func (s *Server) callbackError(w http.ResponseWriter, r *http.Request, message string, status int) {
	if strings.Contains(r.Header.Get("Accept"), "text/html") {
		http.Redirect(w, r, s.Config.ForgejoURL+"/user/login?redirect_to="+url.QueryEscape("/?soda-view=spaces&soda-connect=failed"), http.StatusSeeOther)
		return
	}
	http.Error(w, message, status)
}

func (s *Server) loginFailure(w http.ResponseWriter, r *http.Request, login store.OAuthLogin, message string, status int) {
	if destination := s.nativeOAuthReturn(login); destination != "" {
		http.Redirect(w, r, destination+"&soda-connect=failed", http.StatusSeeOther)
		return
	}
	http.Error(w, message, status)
}

// Fixed bookmark entry establishes the native actor before automatic connection.
func (s *Server) nativePageEntry(w http.ResponseWriter, r *http.Request, login store.OAuthLogin) {
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
	if r.URL.RawQuery != "" || r.URL.ForceQuery {
		http.Error(w, "Page entries do not accept navigation parameters.", 400)
		return
	}
	if config.BaseURL(s.Config.ForgejoURL) != nil || !strings.HasPrefix(s.Config.ForgejoURL, "https://") {
		http.Error(w, "Native origin unavailable.", 503)
		return
	}
	if _, err := requestCookie(r, sessionCookie); err != nil && !errors.Is(err, http.ErrNoCookie) {
		http.Error(w, "Ambiguous Soda cookies.", 400)
		return
	}
	destination := strings.TrimPrefix(s.nativeOAuthReturn(login), s.Config.ForgejoURL)
	http.Redirect(w, r, s.Config.ForgejoURL+"/user/login?redirect_to="+url.QueryEscape(destination), http.StatusSeeOther)
}
