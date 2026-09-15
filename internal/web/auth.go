package web

import (
	"context"
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

func loginQuery(r *http.Request) (url.Values, int, string) {
	if len(r.URL.RawQuery) > 8192 {
		return nil, 400, "Invalid sign-in request."
	}
	query, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		return nil, 400, "Invalid sign-in request."
	}
	return query, 0, ""
}

func validLoginDestination(query url.Values) bool {
	if len(query["return_to"]) > 1 || query.Get("return_to") != "" {
		return false
	}
	destination, hasDestination := query["destination"]
	if !hasDestination {
		return true
	}
	if len(destination) != 1 {
		return false
	}
	switch destination[0] {
	case "spaces", "runners", "tailnet":
		return !query.Has("repository_id")
	case "repository-spaces":
		return true
	}
	return false
}

func loginOAuthContext(query url.Values) (int64, int64, bool) {
	repositoryID, repoOK := oauthContextID(query, "repository_id")
	expectedUserID, userOK := oauthContextID(query, "expected_user_id")
	if !repoOK || !userOK || (query.Get("destination") == "repository-spaces" && repositoryID == 0) {
		return 0, 0, false
	}
	return repositoryID, expectedUserID, true
}

func loginSettingsReturn(destination string) string {
	if destination == "runners" || destination == "tailnet" {
		return destination
	}
	return ""
}

func readLoginCookies(r *http.Request) (session, previous string, status int, message string) {
	for name, target := range map[string]*string{sessionCookie: &session, oauthCookie: &previous} {
		c, err := requestCookie(r, name)
		if err == nil {
			*target = c.Value
		} else if !errors.Is(err, http.ErrNoCookie) {
			return "", "", 400, "Ambiguous Soda cookies; clear them and sign in again."
		}
	}
	return session, previous, 0, ""
}

func (s *Server) beginLoginAttempt(w http.ResponseWriter, r *http.Request, state string, login store.OAuthLogin, session, previous string) bool {
	if err := s.Store.BeginOAuth(r.Context(), state, login, session, previous); err != nil {
		if errors.Is(err, store.ErrLoginContext) {
			s.cookie(w, sessionCookie, "", -1)
			s.cookie(w, oauthCookie, "", -1)
			s.loginFailure(w, r, login, "Previous sign-in expired or ended; start sign-in again.", 409)
		} else {
			s.loginFailure(w, r, login, "Cannot begin sign-in.", 500)
		}
		return false
	}
	return true
}

func (s *Server) redirectLogin(w http.ResponseWriter, r *http.Request, state, verifier string) {
	challenge := sha256.Sum256([]byte(verifier))
	s.cookie(w, oauthCookie, state, 600)
	scopes := "read:user read:repository read:organization"
	q := url.Values{"client_id": {s.Config.OAuthClientID}, "redirect_uri": {s.Config.OAuthCallbackURL()}, "response_type": {"code"}, "scope": {scopes}, "state": {state}, "code_challenge": {base64.RawURLEncoding.EncodeToString(challenge[:])}, "code_challenge_method": {"S256"}}
	http.Redirect(w, r, s.Config.ForgejoURL+"/login/oauth/authorize?"+q.Encode(), http.StatusFound)
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
	query, status, message := loginQuery(r)
	if status != 0 {
		http.Error(w, message, status)
		return
	}
	if !validLoginDestination(query) {
		http.Error(w, "Unsupported sign-in destination.", http.StatusBadRequest)
		return
	}
	repositoryID, expectedUserID, ok := loginOAuthContext(query)
	if !ok {
		http.Error(w, "Invalid repository or expected user ID.", 400)
		return
	}
	destination := query.Get("destination")
	state, verifier := token(), token()
	login := store.OAuthLogin{RepositorySettingsReturn: destination == "repository-spaces", Verifier: verifier, RepositoryID: repositoryID, ExpectedUserID: expectedUserID, SpacesReturn: destination == "spaces", SettingsReturn: loginSettingsReturn(destination)}
	session, previous, status, message := readLoginCookies(r)
	if status != 0 {
		s.loginFailure(w, r, login, message, status)
		return
	}
	if !s.beginLoginAttempt(w, r, state, login, session, previous) {
		return
	}
	s.redirectLogin(w, r, state, verifier)
}

func validCallbackState(state string, c *http.Cookie) bool {
	if state == "" || len(state) > 128 || c == nil {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(state), []byte(c.Value)) == 1
}

func validateCallbackRequest(r *http.Request) (url.Values, string, error) {
	if len(r.URL.RawQuery) > 8192 {
		return nil, "", errors.New("Invalid sign-in response.")
	}
	query, queryErr := url.ParseQuery(r.URL.RawQuery)
	c, err := requestCookie(r, oauthCookie)
	state := query.Get("state")
	if queryErr != nil || len(query["state"]) != 1 || len(query["code"]) > 1 || len(query.Get("code")) > 4096 {
		return nil, "", errors.New("Invalid sign-in state; sign in again.")
	}
	if err != nil || !validCallbackState(state, c) {
		return nil, "", errors.New("Invalid sign-in state; sign in again.")
	}
	return query, state, nil
}

func (s *Server) consumeOAuthState(r *http.Request, state string) (store.OAuthAttempt, error) {
	old, oldErr := requestCookie(r, sessionCookie)
	if oldErr != nil && !errors.Is(oldErr, http.ErrNoCookie) {
		return store.OAuthAttempt{}, errors.New("Ambiguous Soda session; sign in again.")
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
		return store.OAuthAttempt{}, errors.New("Sign-in expired or was already used.")
	}
	return login, nil
}

type verifiedOAuthUser struct {
	grant  forgejo.TokenResponse
	user   forgejo.User
	scopes string
}

func (s *Server) exchangeOAuthGrant(ctx context.Context, code, verifier string) (forgejo.TokenResponse, string, int, string) {
	if code == "" {
		return forgejo.TokenResponse{}, "", 400, "Forgejo did not authorize sign-in."
	}
	secret, err := config.Secret(s.Config.OAuthSecretFile)
	if err != nil {
		return forgejo.TokenResponse{}, "", 503, "Sign-in is not configured."
	}
	grant, err := s.Forgejo.ExchangeGrant(ctx, s.Config.OAuthClientID, secret, code, s.Config.OAuthCallbackURL(), verifier)
	if err != nil {
		return forgejo.TokenResponse{}, "", 502, "Forgejo sign-in failed."
	}
	return grant, secret, 0, ""
}

func (s *Server) identifyOAuthUser(ctx context.Context, token string, expectedUserID int64) (forgejo.User, int, string) {
	u, err := s.Forgejo.Current(ctx, token)
	if err != nil || u.ID <= 0 {
		return forgejo.User{}, 502, "Could not identify this Forgejo user."
	}
	if expectedUserID != 0 && u.ID != expectedUserID {
		return forgejo.User{}, http.StatusForbidden, "Forgejo account changed; reload the repository and sign in again."
	}
	return u, 0, ""
}

func (s *Server) verifyNativeOAuthScopes(login store.OAuthLogin, scopes string) bool {
	if s.nativeOAuthReturn(login) == "" {
		return true
	}
	return forgejo.HasScope(scopes, "read:user") &&
		forgejo.HasScope(scopes, "read:repository") &&
		forgejo.HasScope(scopes, "read:organization")
}

func (s *Server) exchangeOAuthUserAndScopes(
	ctx context.Context,
	code string,
	login store.OAuthAttempt,
) (verifiedOAuthUser, int, string) {
	grant, secret, status, msg := s.exchangeOAuthGrant(ctx, code, login.Verifier)
	if status != 0 {
		return verifiedOAuthUser{}, status, msg
	}
	u, status, msg := s.identifyOAuthUser(ctx, grant.Access, login.ExpectedUserID)
	if status != 0 {
		return verifiedOAuthUser{}, status, msg
	}
	scopes, err := s.Forgejo.GrantScopes(ctx, s.Config.OAuthClientID, secret, grant.Access, u.ID)
	if err != nil {
		return verifiedOAuthUser{}, 502, "Could not verify Forgejo consent."
	}
	if !s.verifyNativeOAuthScopes(login.OAuthLogin, scopes) {
		return verifiedOAuthUser{}, http.StatusForbidden, "Forgejo consent is missing required scopes."
	}
	return verifiedOAuthUser{grant: grant, user: u, scopes: scopes}, 0, ""
}

func (s *Server) resolveOAuthReturnRepository(ctx context.Context, login store.OAuthAttempt, token string, scopes string) *forgejo.Repository {
	if login.RepositorySettingsReturn || login.RepositoryID == 0 || !forgejo.HasScope(scopes, "read:repository") {
		return nil
	}
	repo, err := s.Forgejo.RepositoryByID(ctx, token, login.RepositoryID)
	if err != nil {
		return nil
	}
	return &repo
}

func (s *Server) completeOAuthSession(
	w http.ResponseWriter,
	r *http.Request,
	login store.OAuthAttempt,
	auth verifiedOAuthUser,
	repo *forgejo.Repository,
) {
	value, csrf := token(), token()
	err := s.Store.FinishOAuth(r.Context(), login, store.User{ID: auth.user.ID, Login: auth.user.Login, Name: auth.user.Name}, value, csrf, store.Grant{Access: auth.grant.Access, Refresh: auth.grant.Refresh, Scopes: auth.scopes, Expires: auth.grant.ExpiresAt})
	if err != nil {
		if errors.Is(err, store.ErrLoginContext) {
			s.loginFailure(w, r, login.OAuthLogin, "Sign-in cancelled, expired or superseded; reload and start again.", 409)
		} else {
			s.loginFailure(w, r, login.OAuthLogin, "Could not create session.", 500)
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
	s.forgejoReturn(w, r, repo)
}

func (s *Server) callback(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
	query, state, err := validateCallbackRequest(r)
	if err != nil {
		s.callbackError(w, r, err.Error(), 400)
		return
	}
	login, err := s.consumeOAuthState(r, state)
	if err != nil {
		s.callbackError(w, r, err.Error(), 400)
		return
	}
	auth, status, msg := s.exchangeOAuthUserAndScopes(r.Context(), query.Get("code"), login)
	if status != 0 {
		s.loginFailure(w, r, login.OAuthLogin, msg, status)
		return
	}
	repo := s.resolveOAuthReturnRepository(r.Context(), login, auth.grant.Access, auth.scopes)
	s.completeOAuthSession(w, r, login, auth, repo)
}

// Only persisted transaction fields select these fixed native rendering views.
// Spaces views stay on the dashboard host. Runners and Tailnet render inside
// Forgejo's real administration layout, so their fixed destinations address the
// admin host and require native admin-page eligibility; protected Soda APIs
// still require the configured Soda operator independently.
func (s *Server) nativeOAuthReturn(login store.OAuthLogin) string {
	view := ""
	if login.RepositorySettingsReturn {
		view = "repository-spaces"
	} else if login.SettingsReturn == "runners" || login.SettingsReturn == "tailnet" {
		view = login.SettingsReturn
	} else if login.SpacesReturn {
		view = "spaces"
	}
	if view == "" {
		return ""
	}
	destination := s.Config.ForgejoURL + "/?soda-view=" + view
	if view == "runners" || view == "tailnet" {
		destination = s.Config.ForgejoURL + "/admin?soda-view=" + view
	}
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
		http.Error(w, "Native origin unavailable.", http.StatusServiceUnavailable)
		return
	}
	if _, err := requestCookie(r, sessionCookie); err != nil && !errors.Is(err, http.ErrNoCookie) {
		http.Error(w, "Ambiguous Soda cookies.", 400)
		return
	}
	destination := strings.TrimPrefix(s.nativeOAuthReturn(login), s.Config.ForgejoURL)
	http.Redirect(w, r, s.Config.ForgejoURL+"/user/login?redirect_to="+url.QueryEscape(destination), http.StatusSeeOther)
}
