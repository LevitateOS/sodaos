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
	s.mux.HandleFunc("POST /logout", s.protected(func(w http.ResponseWriter, r *http.Request, v store.Session) {
		c, _ := r.Cookie("soda_session")
		if err := s.Store.DeleteSession(r.Context(), c.Value); err != nil {
			s.fail(w, "Could not end this session.", 500)
			return
		}
		s.cookie(w, "soda_session", "", -1)
		http.Redirect(w, r, "/", 303)
	}))
	s.mux.HandleFunc("GET /profile", s.protected(s.profile))
	s.mux.HandleFunc("POST /profile", s.protected(s.updateProfile))
	s.mux.HandleFunc("POST /keys", s.protected(s.addKey))
	s.mux.HandleFunc("GET /people", s.protected(s.people))
	s.mux.HandleFunc("POST /people", s.protected(s.createPerson))
}
func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	returnPath := r.URL.Query().Get("return_to")
	if returnPath == "" {
		returnPath = "/projects"
	}
	if returnPath != "/projects" {
		s.fail(w, "Unsupported sign-in destination.", http.StatusBadRequest)
		return
	}
	state, verifier := token(), token()
	if err := s.Store.BeginOAuth(r.Context(), state, verifier, returnPath); err != nil {
		s.fail(w, "Cannot begin sign-in.", 500)
		return
	}
	challenge := sha256.Sum256([]byte(verifier))
	s.cookie(w, "soda_oauth", state, 600)
	scopes := "read:user read:repository read:organization"
	if r.URL.Query().Get("administration") == "1" {
		scopes += " write:admin"
	}
	q := url.Values{"client_id": {s.Config.OAuthClientID}, "redirect_uri": {s.Config.PublicURL + "/oauth/callback"}, "response_type": {"code"}, "scope": {scopes}, "state": {state}, "code_challenge": {base64.RawURLEncoding.EncodeToString(challenge[:])}, "code_challenge_method": {"S256"}}
	http.Redirect(w, r, s.Config.ForgejoURL+"/login/oauth/authorize?"+q.Encode(), 302)
}
func (s *Server) callback(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	c, err := r.Cookie("soda_oauth")
	state := r.URL.Query().Get("state")
	if err != nil || state == "" || subtle.ConstantTimeCompare([]byte(state), []byte(c.Value)) != 1 {
		s.fail(w, "Invalid sign-in state; sign in again.", 400)
		return
	}
	s.cookie(w, "soda_oauth", "", -1)
	oauth, err := s.Store.ConsumeOAuth(r.Context(), state)
	if err != nil {
		s.fail(w, "Sign-in expired or was already used.", 400)
		return
	}
	code := r.URL.Query().Get("code")
	if code == "" {
		s.fail(w, "Forgejo did not authorize sign-in.", 400)
		return
	}
	secret, err := config.Secret(s.Config.OAuthSecretFile)
	if err != nil {
		s.fail(w, "Sign-in is not configured.", 503)
		return
	}
	grant, err := s.Forgejo.ExchangeGrant(r.Context(), s.Config.OAuthClientID, secret, code, s.Config.PublicURL+"/oauth/callback", oauth.Verifier)
	if err != nil {
		s.fail(w, "Forgejo sign-in failed.", 502)
		return
	}
	u, err := s.Forgejo.Current(r.Context(), grant.Access)
	if err != nil || u.ID <= 0 {
		s.fail(w, "Could not identify this Forgejo user.", 502)
		return
	}
	scopes, err := s.Forgejo.GrantScopes(r.Context(), s.Config.OAuthClientID, secret, grant.Access, u.ID)
	if err != nil {
		s.fail(w, "Could not verify Forgejo consent.", 502)
		return
	}
	if err = s.Store.UpsertUser(r.Context(), store.User{ID: u.ID, Login: u.Login, Name: u.Name}); err != nil {
		s.fail(w, "Could not save Soda profile.", 500)
		return
	}
	if old, e := r.Cookie("soda_session"); e == nil {
		if err = s.Store.DeleteSession(r.Context(), old.Value); err != nil {
			s.fail(w, "Could not rotate session.", 500)
			return
		}
	}
	value, csrf := token(), token()
	if err = s.Store.CreateGrantedSession(r.Context(), value, u.ID, csrf, store.Grant{Access: grant.Access, Refresh: grant.Refresh, Scopes: scopes, Expires: grant.ExpiresAt}); err != nil {
		s.fail(w, "Could not create session.", 500)
		return
	}
	s.cookie(w, "soda_session", value, int((12 * time.Hour).Seconds()))
	// Finish a pre-removal sign-in without returning to the retired SPA.
	if oauth.ReturnPath == "/app/" {
		oauth.ReturnPath = "/projects"
	}
	http.Redirect(w, r, oauth.ReturnPath, 303)
}
func (s *Server) protected(next func(http.ResponseWriter, *http.Request, store.Session)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		c, err := r.Cookie("soda_session")
		if err != nil {
			s.redirectLogin(w, r)
			return
		}
		v, err := s.Store.Session(r.Context(), c.Value)
		if err != nil {
			s.redirectLogin(w, r)
			return
		}
		if r.Method == "POST" {
			r.Body = http.MaxBytesReader(w, r.Body, 65536)
			if err = r.ParseForm(); err != nil {
				s.fail(w, "Invalid form.", 400)
				return
			}
			if !s.validMutation(r, v.CSRF) {
				s.fail(w, "Invalid request origin or CSRF token.", 403)
				return
			}
		}
		next(w, r, v)
	}
}
func (s *Server) validMutation(r *http.Request, csrf string) bool {
	origin := r.Header.Get("Origin")
	if origin != "" && origin != s.Config.PublicURL {
		return false
	}
	return csrf != "" && subtle.ConstantTimeCompare([]byte(r.FormValue("csrf")), []byte(csrf)) == 1
}
func (s *Server) redirectLogin(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", "/login")
		w.WriteHeader(200)
		return
	}
	http.Redirect(w, r, "/login", 303)
}
func (s *Server) fail(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	s.render(w, "error", struct{ Message string }{message})
}
