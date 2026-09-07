package web

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/levitateos/sodaos/internal/config"
	"github.com/levitateos/sodaos/internal/store"
)

func token() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}
func (s *Server) cookie(w http.ResponseWriter, name, value string, seconds int) {
	http.SetCookie(w, &http.Cookie{Name: name, Value: value, Path: "/", MaxAge: seconds, HttpOnly: true, Secure: strings.HasPrefix(s.Config.PublicURL, "https://"), SameSite: http.SameSiteLaxMode})
}
func (s *Server) authRoutes() {
	s.mux.HandleFunc("GET /login", s.login)
	s.mux.HandleFunc("GET /oauth/callback", s.callback)
}
func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if r.URL.Query().Get("return_to") != "" {
		http.Error(w, "Unsupported sign-in destination.", http.StatusBadRequest)
		return
	}
	state, verifier := token(), token()
	if err := s.Store.BeginOAuth(r.Context(), state, verifier); err != nil {
		http.Error(w, "Cannot begin sign-in.", 500)
		return
	}
	challenge := sha256.Sum256([]byte(verifier))
	s.cookie(w, "soda_oauth", state, 600)
	scopes := "read:user read:repository read:organization"
	q := url.Values{"client_id": {s.Config.OAuthClientID}, "redirect_uri": {s.Config.PublicURL + "/oauth/callback"}, "response_type": {"code"}, "scope": {scopes}, "state": {state}, "code_challenge": {base64.RawURLEncoding.EncodeToString(challenge[:])}, "code_challenge_method": {"S256"}}
	http.Redirect(w, r, s.Config.ForgejoURL+"/login/oauth/authorize?"+q.Encode(), 302)
}
func (s *Server) callback(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	c, err := r.Cookie("soda_oauth")
	state := r.URL.Query().Get("state")
	if err != nil || state == "" || subtle.ConstantTimeCompare([]byte(state), []byte(c.Value)) != 1 {
		http.Error(w, "Invalid sign-in state; sign in again.", 400)
		return
	}
	s.cookie(w, "soda_oauth", "", -1)
	verifier, err := s.Store.ConsumeOAuth(r.Context(), state)
	if err != nil {
		http.Error(w, "Sign-in expired or was already used.", 400)
		return
	}
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "Forgejo did not authorize sign-in.", 400)
		return
	}
	secret, err := config.Secret(s.Config.OAuthSecretFile)
	if err != nil {
		http.Error(w, "Sign-in is not configured.", 503)
		return
	}
	grant, err := s.Forgejo.ExchangeGrant(r.Context(), s.Config.OAuthClientID, secret, code, s.Config.PublicURL+"/oauth/callback", verifier)
	if err != nil {
		http.Error(w, "Forgejo sign-in failed.", 502)
		return
	}
	u, err := s.Forgejo.Current(r.Context(), grant.Access)
	if err != nil || u.ID <= 0 {
		http.Error(w, "Could not identify this Forgejo user.", 502)
		return
	}
	scopes, err := s.Forgejo.GrantScopes(r.Context(), s.Config.OAuthClientID, secret, grant.Access, u.ID)
	if err != nil {
		http.Error(w, "Could not verify Forgejo consent.", 502)
		return
	}
	if err = s.Store.UpsertUser(r.Context(), store.User{ID: u.ID, Login: u.Login, Name: u.Name}); err != nil {
		http.Error(w, "Could not save Soda profile.", 500)
		return
	}
	if old, e := r.Cookie("soda_session"); e == nil {
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
	s.cookie(w, "soda_session", value, int((12 * time.Hour).Seconds()))
	s.forgejoHome(w, r)
}
