package web

import (
	"net/http"
	"net/mail"
	"regexp"
	"strings"

	"github.com/levitateos/sodaos/internal/config"
	"github.com/levitateos/sodaos/internal/forgejo"
	"github.com/levitateos/sodaos/internal/host"
	"github.com/levitateos/sodaos/internal/store"
)

var projectLogin = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,30}$`)

type Page struct {
	Environment     host.Environment
	NativeError     bool
	Session         store.Session
	Operator        bool
	Keys            []store.Key
	Users           []store.User
	Projects        []store.Project
	Repositories    []forgejo.Repository
	RepositoryError bool
	Project         store.Project
	Member          string
	ForgejoURL      string
}

func (s *Server) page(v store.Session) Page {
	return Page{Session: v, Operator: v.User.ID == s.Config.OperatorID, ForgejoURL: s.Config.ForgejoURL}
}
func (s *Server) profile(w http.ResponseWriter, r *http.Request, v store.Session) {
	p := s.page(v)
	var err error
	p.Keys, err = s.Store.Keys(r.Context(), v.User.ID)
	if err != nil {
		s.fail(w, "Cannot load keys.", 500)
		return
	}
	s.render(w, "profile", p)
}
func (s *Server) updateProfile(w http.ResponseWriter, r *http.Request, v store.Session) {
	if err := s.Store.RenameProfile(r.Context(), v.User.ID, strings.TrimSpace(r.FormValue("name"))); err != nil {
		s.fail(w, "Cannot save profile; names must be at most 200 bytes.", 400)
		return
	}
	http.Redirect(w, r, "/profile", 303)
}
func (s *Server) addKey(w http.ResponseWriter, r *http.Request, v store.Session) {
	public, fingerprint, err := normalizeDevelopmentKey(r.FormValue("public_key"))
	if err != nil {
		s.fail(w, "Provide one public SSH key without authorized_keys options. Never submit a private key.", 400)
		return
	}
	if err = s.Store.AddKey(r.Context(), v.User.ID, public, fingerprint); err != nil {
		s.fail(w, "Could not register key.", 500)
		return
	}
	http.Redirect(w, r, "/profile", 303)
}
func (s *Server) people(w http.ResponseWriter, r *http.Request, v store.Session) {
	if !s.operator(w, v) {
		return
	}
	p := s.page(v)
	var err error
	p.Users, err = s.Store.Users(r.Context())
	if err != nil {
		s.fail(w, "Cannot list people.", 500)
		return
	}
	s.render(w, "people", p)
}
func (s *Server) createPerson(w http.ResponseWriter, r *http.Request, v store.Session) {
	if !s.operator(w, v) {
		return
	}
	login, email, password := strings.TrimSpace(r.FormValue("login")), strings.TrimSpace(r.FormValue("email")), r.FormValue("password")
	if _, err := mail.ParseAddress(email); err != nil || !projectLogin.MatchString(login) || login == "root" || len(password) < 12 {
		s.fail(w, "Provide a lowercase Linux-compatible username (1–31 characters, not root), email and initial password of at least 12 characters.", 400)
		return
	}
	admin, err := config.Secret(s.Config.AdminTokenFile)
	if err != nil {
		s.fail(w, "Operator provider credential is unavailable.", 503)
		return
	}
	u, err := s.Forgejo.CreateUser(r.Context(), admin, login, email, password)
	if err != nil {
		s.fail(w, "Forgejo could not create the user. Inspect native Forgejo administration before retrying.", 502)
		return
	}
	if err = s.Store.UpsertUser(r.Context(), store.User{ID: u.ID, Login: u.Login, Name: u.Name}); err != nil {
		s.fail(w, "Forgejo user created, but Soda profile could not be stored. Inspect native state before retrying.", 500)
		return
	}
	http.Redirect(w, r, "/people", 303)
}
