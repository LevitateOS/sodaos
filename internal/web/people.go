package web

import (
	"net/http"
	"strings"

	"github.com/levitateos/sodaos/internal/forgejo"
	"github.com/levitateos/sodaos/internal/host"
	"github.com/levitateos/sodaos/internal/store"
)

type Page struct {
	Environment     host.Environment
	NativeError     bool
	Session         store.Session
	Keys            []store.Key
	Users           []store.User
	Projects        []store.Project
	Repositories    []forgejo.Repository
	RepositoryError bool
	NextPage        *int
	Project         store.Project
	Member          string
	ForgejoURL      string
}

func (s *Server) page(v store.Session) Page {
	return Page{Session: v, ForgejoURL: s.Config.ForgejoURL}
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
	grant, err := s.userGrant(r, v)
	if err != nil {
		providerError(w, err)
		return
	}
	if !forgejo.HasScope(grant.Scopes, "read:admin") {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusForbidden)
		s.render(w, "people-consent", s.page(v))
		return
	}
	page, ok := apiPage(w, r)
	if !ok {
		return
	}
	users, pagination, err := s.Forgejo.People(r.Context(), grant.Access, page)
	if err != nil {
		providerError(w, err)
		return
	}
	p := s.page(v)
	p.NextPage = pagination.NextPage
	for _, u := range users {
		p.Users = append(p.Users, store.User{ID: u.ID, Login: u.Login, Name: u.Name})
	}
	s.render(w, "people", p)
}
func (s *Server) createPerson(w http.ResponseWriter, r *http.Request, v store.Session) {
	grant, err := s.userGrant(r, v)
	if err != nil {
		providerError(w, err)
		return
	}
	if !forgejo.HasScope(grant.Scopes, "write:admin") {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusForbidden)
		s.render(w, "people-consent", s.page(v))
		return
	}
	login, email, password := r.FormValue("login"), r.FormValue("email"), r.FormValue("password")
	if login == "" || len(login) > 255 || email == "" || len(email) > 320 || password == "" || len(password) > 4096 {
		s.fail(w, "Provide a username, email and initial password within supported lengths.", 400)
		return
	}
	_, err = s.Forgejo.CreateUser(r.Context(), grant.Access, login, email, password)
	if err != nil {
		providerError(w, err)
		return
	}
	// Native onboarding/OAuth establishes the Soda profile, not an admin snapshot.
	http.Redirect(w, r, "/people", 303)
}
