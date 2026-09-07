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
	repositoryID, repoOK := oauthContextID(query, "repository_id")
	expectedUserID, userOK := oauthContextID(query, "expected_user_id")
	if !repoOK || !userOK {
		http.Error(w, "Invalid repository or expected user ID.", 400)
		return
	}
	state, verifier := token(), token()
	if err := s.Store.BeginOAuth(r.Context(), state, store.OAuthLogin{Verifier: verifier, RepositoryID: repositoryID, ExpectedUserID: expectedUserID}); err != nil {
		http.Error(w, "Cannot begin sign-in.", 500)
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
		http.Error(w, "Invalid sign-in response.", 400)
		return
	}
	query, queryErr := url.ParseQuery(r.URL.RawQuery)
	c, err := requestCookie(r, oauthCookie)
	state := query.Get("state")
	if queryErr != nil || len(query["state"]) != 1 || len(query["code"]) > 1 || len(query.Get("code")) > 4096 ||
		err != nil || state == "" || len(state) > 128 || subtle.ConstantTimeCompare([]byte(state), []byte(c.Value)) != 1 {
		http.Error(w, "Invalid sign-in state; sign in again.", 400)
		return
	}
	old, oldErr := requestCookie(r, sessionCookie)
	if oldErr != nil && !errors.Is(oldErr, http.ErrNoCookie) {
		http.Error(w, "Ambiguous Soda session; sign in again.", 400)
		return
	}
	s.cookie(w, oauthCookie, "", -1)
	login, err := s.Store.ConsumeOAuth(r.Context(), state)
	if err != nil {
		http.Error(w, "Sign-in expired or was already used.", 400)
		return
	}
	code := query.Get("code")
	if code == "" {
		http.Error(w, "Forgejo did not authorize sign-in.", 400)
		return
	}
	secret, err := config.Secret(s.Config.OAuthSecretFile)
	if err != nil {
		http.Error(w, "Sign-in is not configured.", 503)
		return
	}
	grant, err := s.Forgejo.ExchangeGrant(r.Context(), s.Config.OAuthClientID, secret, code, s.Config.OAuthCallbackURL(), login.Verifier)
	if err != nil {
		http.Error(w, "Forgejo sign-in failed.", 502)
		return
	}
	u, err := s.Forgejo.Current(r.Context(), grant.Access)
	if err != nil || u.ID <= 0 {
		http.Error(w, "Could not identify this Forgejo user.", 502)
		return
	}
	if login.ExpectedUserID != 0 && u.ID != login.ExpectedUserID {
		http.Error(w, "Forgejo account changed; reload the repository and sign in again.", http.StatusForbidden)
		return
	}
	scopes, err := s.Forgejo.GrantScopes(r.Context(), s.Config.OAuthClientID, secret, grant.Access, u.ID)
	if err != nil {
		http.Error(w, "Could not verify Forgejo consent.", 502)
		return
	}
	// Only the consumed transaction selects the repository. Caller callback
	// parameters, cached names and provider-supplied URLs are not destinations.
	var repository *forgejo.Repository
	if login.RepositoryID != 0 && forgejo.HasScope(scopes, "read:repository") {
		if repo, err := s.Forgejo.RepositoryByID(r.Context(), grant.Access, login.RepositoryID); err == nil {
			repository = &repo
		}
	}
	if err = s.Store.UpsertUser(r.Context(), store.User{ID: u.ID, Login: u.Login, Name: u.Name}); err != nil {
		http.Error(w, "Could not save Soda profile.", 500)
		return
	}
	if oldErr == nil {
		if err = s.Store.DeleteSession(r.Context(), old.Value); err != nil {
			http.Error(w, "Could not rotate session.", 500)
			return
		}
	}
	value, csrf := token(), token()
	if err = s.Store.CreateGrantedSession(r.Context(), value, u.ID, csrf, store.Grant{Access: grant.Access, Refresh: grant.Refresh, Scopes: scopes, Expires: grant.ExpiresAt}); err != nil {
		http.Error(w, "Could not create session.", 500)
		return
	}
	s.cookie(w, sessionCookie, value, int((12 * time.Hour).Seconds()))
	s.forgejoReturn(w, r, repository)
}
