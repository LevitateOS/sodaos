package web

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"github.com/levitateos/sodaos/internal/config"
	"github.com/levitateos/sodaos/internal/host"
	"github.com/levitateos/sodaos/internal/store"
	"net/http"
	"regexp"
	"strings"
)

var repoPart = regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9_.-]{0,99}$`)

func (s *Server) projectRoutes() {
	s.mux.HandleFunc("GET /projects", s.protected(s.projects))
	s.mux.HandleFunc("POST /projects", s.protected(s.createProject))
	s.mux.HandleFunc("GET /projects/{id}", s.protected(s.project))
	s.mux.HandleFunc("POST /projects/{id}/join", s.protected(s.joinProject))
}
func (s *Server) projects(w http.ResponseWriter, r *http.Request, v store.Session) {
	p := s.page(v)
	var err error
	p.Projects, err = s.Store.Projects(r.Context())
	if err != nil {
		s.fail(w, "Cannot list projects.", 500)
		return
	}
	s.render(w, "projects", p)
}
func (s *Server) createProject(w http.ResponseWriter, r *http.Request, v store.Session) {
	parts := strings.Split(strings.TrimSpace(r.FormValue("repository")), "/")
	if len(parts) != 2 || !repoPart.MatchString(parts[0]) || !repoPart.MatchString(parts[1]) {
		s.fail(w, "Choose an existing Forgejo repository as owner/name.", 400)
		return
	}
	admin, err := config.Secret(s.Config.AdminTokenFile)
	if err != nil {
		s.fail(w, "Provider credential unavailable.", 503)
		return
	}
	repo, err := s.Forgejo.Repository(r.Context(), admin, parts[0], parts[1])
	if err != nil {
		s.fail(w, "Cannot find that Forgejo repository.", 400)
		return
	}
	if repo.Owner.ID != v.User.ID || repo.ID <= 0 {
		s.fail(w, "Only the repository's human owner can create its project environment. Organization ownership is not supported yet.", 403)
		return
	}
	idBytes := make([]byte, 12)
	rand.Read(idBytes)
	p := store.Project{ID: "p" + hex.EncodeToString(idBytes), Name: repo.Name, RepositoryID: repo.ID, OwnerID: v.User.ID, Repository: repo.FullName}
	if err = s.Store.CreateProject(r.Context(), p); err != nil {
		s.fail(w, "Cannot reserve project. Check whether this repository already has an environment.", 409)
		return
	}
	env, err := s.Host.Create(r.Context(), host.Create{ID: p.ID, Owner: p.OwnerID})
	if err != nil {
		s.fail(w, "Project record retained, but native provisioning did not complete. Ask the operator to inspect its native state before retrying.", 502)
		return
	}
	if err = s.Store.MarkReady(r.Context(), p.ID, env.IP); err != nil {
		s.fail(w, "Environment created but its result could not be saved. Ask the operator to inspect native state.", 500)
		return
	}
	http.Redirect(w, r, "/projects/"+p.ID, 303)
}
func (s *Server) project(w http.ResponseWriter, r *http.Request, v store.Session) {
	p := s.page(v)
	var err error
	p.Project, err = s.Store.Project(r.Context(), r.PathValue("id"))
	if err != nil {
		s.fail(w, "Project not found.", 404)
		return
	}
	p.Member, err = s.Store.MemberLogin(r.Context(), p.Project.ID, v.User.ID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		s.fail(w, "Cannot load membership.", 500)
		return
	}
	if p.Project.Ready {
		p.Environment, err = s.Host.Inspect(r.Context(), p.Project.ID)
		if err != nil {
			p.NativeError = true
		}
	}
	s.render(w, "project", p)
}
func (s *Server) joinProject(w http.ResponseWriter, r *http.Request, v store.Session) {
	p, err := s.Store.Project(r.Context(), r.PathValue("id"))
	if err != nil || !p.Ready {
		s.fail(w, "Project is not provisioned.", 409)
		return
	}
	if _, err = s.Store.MemberLogin(r.Context(), p.ID, v.User.ID); err == nil {
		http.Redirect(w, r, "/projects/"+p.ID, 303)
		return
	} else if !errors.Is(err, sql.ErrNoRows) {
		s.fail(w, "Cannot inspect membership.", 500)
		return
	}
	keys, err := s.Store.Keys(r.Context(), v.User.ID)
	if err != nil {
		s.fail(w, "Cannot read public keys.", 500)
		return
	}
	if len(keys) == 0 {
		s.fail(w, "Register a public SSH key in your profile before joining.", 400)
		return
	}
	public := []string{}
	for _, k := range keys {
		public = append(public, k.Public)
	}
	if err = s.Host.Join(r.Context(), host.Account{Project: p.ID, Login: v.User.Login, Identity: v.User.ID, Keys: public}); err != nil {
		s.fail(w, "Native account provisioning failed; membership was not recorded. Ask the operator to inspect the native account.", 502)
		return
	}
	if err = s.Store.Join(r.Context(), p.ID, v.User.ID, v.User.Login); err != nil {
		s.fail(w, "Native account created, but membership could not be saved. Ask the operator to inspect the result.", 500)
		return
	}
	http.Redirect(w, r, "/projects/"+p.ID, 303)
}
