package web

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/levitateos/sodaos/internal/config"
	"github.com/levitateos/sodaos/internal/host"
	"github.com/levitateos/sodaos/internal/store"
)

// Project-local Linux names are a native provisioning constraint, not a
// restriction on Forgejo's own account names.
var projectLogin = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,30}$`)

func (s *Server) environmentRoutes() {
	s.mux.HandleFunc("/api/environments/{id}/lifecycle", s.apiProtected(s.apiLifecycle, "GET", "POST"))
	s.mux.HandleFunc("/api/environments/{id}/access-keys", s.apiProtected(s.apiAccessKeys, "GET", "POST"))
	s.mux.HandleFunc("/api/environments/{id}/terminal", s.apiTerminal)
	s.mux.HandleFunc("/api/environments/{id}/terminal-session", s.apiProtected(func(w http.ResponseWriter, r *http.Request, _ store.Session) {
		jsonError(w, 410, "terminal_client_obsolete", "Reload this page; terminal actions now require an exact ID.")
	}, http.MethodGet, http.MethodPost))
	s.mux.HandleFunc("/api/environments/{id}/terminal-sessions/{terminalID}", s.apiProtected(s.apiTerminalSession, http.MethodGet, http.MethodPost))
	s.mux.HandleFunc("/api/environments/{id}/terminal-attempts/{requestID}", s.apiProtected(s.apiTerminalAttempt, http.MethodGet))
	s.mux.HandleFunc("/api/spaces", s.apiProtected(s.apiSpaces, http.MethodGet))
	s.mux.HandleFunc("/api/environments", s.apiProtected(s.apiEnvironments, "GET", "POST"))
	s.mux.HandleFunc("/api/environments/{id}", s.apiProtected(s.apiEnvironment, "GET"))
	s.mux.HandleFunc("/api/environments/{id}/join", s.apiProtected(s.apiJoinEnvironment, "POST"))
	s.mux.HandleFunc("/api/environments/{id}/members", s.apiProtected(s.apiEnvironmentMembers, "GET"))
	s.mux.HandleFunc("/api/environments/{id}/connection", s.apiProtected(s.apiConnection, "GET"))
}

type environmentView struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	RepositoryID string `json:"repository_id"`
	Repository   string `json:"repository"`
	OwnerID      string `json:"owner_id"`
	Provisioned  bool   `json:"provisioned"`
}

type repositoryContextView struct {
	ID      string `json:"id"`
	OwnerID string `json:"owner_id"`
	Owner   string `json:"owner"`
	Name    string `json:"name"`
}

func environmentDTO(p store.Project) environmentView {
	return environmentView{p.ID, p.Name, strconv.FormatInt(p.RepositoryID, 10), p.Repository, strconv.FormatInt(p.OwnerID, 10), p.Ready}
}
func (s *Server) apiEnvironments(w http.ResponseWriter, r *http.Request, v store.Session) {
	if r.Method == "POST" {
		s.apiCreateEnvironment(w, r, v)
		return
	}
	if len(r.URL.RawQuery) > 8192 {
		jsonError(w, 400, "invalid_repository", "Provide one repository_id.")
		return
	}
	query, err := url.ParseQuery(r.URL.RawQuery)
	id, valid := positiveID(query.Get("repository_id"))
	if err != nil || len(query) != 1 || len(query["repository_id"]) != 1 || !valid {
		jsonError(w, 400, "invalid_repository", "Provide one repository_id.")
		return
	}
	access, err := s.visibleRepository(r, v, id)
	if err != nil {
		providerError(w, err)
		return
	}
	p, err := s.Store.ProjectByRepository(r.Context(), id)
	absent := errors.Is(err, sql.ErrNoRows)
	if err != nil && !absent {
		jsonError(w, 503, "store_unavailable", "Could not read environment.")
		return
	}
	items := []environmentView{}
	if !absent {
		items = append(items, environmentDTO(p))
	}
	jsonResponse(w, 200, struct {
		Items      []environmentView     `json:"items"`
		Repository repositoryContextView `json:"repository"`
		CanCreate  bool                  `json:"can_create"`
	}{Items: items, Repository: repositoryContextView{strconv.FormatInt(id, 10), strconv.FormatInt(access.repository.Owner.ID, 10), access.repository.Owner.Login, access.repository.Name}, CanCreate: absent && access.repository.Owner.ID == v.User.ID})
}
func validRepositoryPart(value string) bool {
	return value != "" && value != "." && value != ".." && len(value) <= 255 && !strings.ContainsAny(value, "/\\\x00\r\n")
}
func (s *Server) apiCreateEnvironment(w http.ResponseWriter, r *http.Request, v store.Session) {
	var input struct {
		RepositoryID string `json:"repository_id"`
	}
	if !decodeAPIObject(w, r, &input) {
		return
	}
	repositoryID, valid := positiveID(input.RepositoryID)
	if !valid {
		jsonError(w, 400, "invalid_repository", "Provide one repository_id.")
		return
	}
	access, err := s.visibleRepository(r, v, repositoryID)
	if err != nil {
		providerError(w, err)
		return
	}
	repo := access.repository
	if repo.Owner.ID != access.actor.ID {
		jsonError(w, 403, "owner_required", "Only the human repository owner can create its environment. Organization-owned environments are not supported.")
		return
	}
	bytes := make([]byte, 12)
	rand.Read(bytes)
	p := store.Project{ID: "p" + hex.EncodeToString(bytes), Name: repo.Name, RepositoryID: repo.ID, OwnerID: v.User.ID, Repository: repo.FullName}
	if err = s.Store.CreateProject(r.Context(), p); err != nil {
		jsonError(w, 409, "reservation_failed", "Repository may already have an environment reservation. Refresh its environment before retrying.")
		return
	}
	w.Header().Set("Location", config.SodaPath+"/api/environments/"+p.ID)
	env, err := s.Host.Create(r.Context(), host.Create{ID: p.ID, Owner: p.OwnerID})
	if err != nil {
		jsonResponse(w, 502, struct {
			Error       apiError        `json:"error"`
			Environment environmentView `json:"environment"`
		}{apiError{"provisioning_incomplete", "Reservation retained; native provisioning was not confirmed. Inspect this environment with the operator; do not recreate it."}, environmentDTO(p)})
		return
	}
	if err = s.Store.MarkReady(r.Context(), p.ID, env.IP); err != nil {
		jsonResponse(w, 503, struct {
			Error       apiError        `json:"error"`
			Environment environmentView `json:"environment"`
		}{apiError{"result_not_saved", "Native provisioning returned, but its result was not saved. Inspect the retained environment; do not recreate it."}, environmentDTO(p)})
		return
	}
	p.Ready = true
	jsonResponse(w, 201, environmentDTO(p))
}
func (s *Server) loadEnvironment(w http.ResponseWriter, r *http.Request) (store.Project, bool) {
	p, err := s.Store.Project(r.Context(), r.PathValue("id"))
	if errors.Is(err, sql.ErrNoRows) {
		jsonError(w, 404, "not_found", "Environment not found.")
		return p, false
	}
	if err != nil {
		jsonError(w, 503, "store_unavailable", "Could not read environment.")
		return p, false
	}
	return p, true
}
func (s *Server) apiEnvironment(w http.ResponseWriter, r *http.Request, v store.Session) {
	p, ok := s.loadEnvironment(w, r)
	if !ok {
		return
	}
	reader, allowed := s.authorizeEnvironmentRead(w, r, v, p)
	if !allowed {
		return
	}
	// Even an incomplete reservation has read-only inspection; ready is a
	// provisioning result, not evidence of a running or client-reachable service.
	env, nativeErr := s.Host.Inspect(r.Context(), p.ID)
	var observed *host.Environment
	if nativeErr == nil {
		observed = &env
	}
	jsonResponse(w, 200, struct {
		AuthorityUnavailable bool              `json:"authority_unavailable"`
		Environment          environmentView   `json:"environment"`
		Observed             *host.Environment `json:"observed"`
		NativeUnavailable    bool              `json:"native_unavailable"`
		Login                string            `json:"login"`
		Administrator        bool              `json:"environment_administrator"`
	}{reader.authorityUnavailable, environmentDTO(p), observed, nativeErr != nil, reader.login, reader.administrator})
}
func (s *Server) apiJoinEnvironment(w http.ResponseWriter, r *http.Request, v store.Session) {
	if !decodeAPIObject(w, r, &struct{}{}) {
		return
	}
	p, ok := s.loadEnvironment(w, r)
	if !ok {
		return
	}
	login, err := s.Store.MemberLogin(r.Context(), p.ID, v.User.ID)
	if err == nil {
		jsonResponse(w, 200, struct {
			Login string `json:"login"`
		}{login})
		return
	}
	if !errors.Is(err, sql.ErrNoRows) {
		jsonError(w, 503, "store_unavailable", "Could not inspect membership.")
		return
	}
	access, err := s.visibleRepository(r, v, p.RepositoryID)
	if err != nil {
		providerError(w, err)
		return
	}
	if !p.Ready {
		jsonError(w, 409, "not_provisioned", "Environment provisioning is incomplete.")
		return
	}
	login = access.actor.Login
	if !projectLogin.MatchString(login) || login == "root" {
		jsonError(w, 422, "unsupported_linux_login", "Your Forgejo username is not supported as a project Linux account. No automatic rename is performed.")
		return
	}
	keys, err := s.Store.Keys(r.Context(), v.User.ID)
	if err != nil {
		jsonError(w, 503, "store_unavailable", "Could not read development keys.")
		return
	}
	if len(keys) == 0 {
		jsonError(w, 422, "development_key_required", "Register a public development-access key before joining.")
		return
	}
	if len(keys) > 32 {
		jsonError(w, 422, "too_many_keys", "Native onboarding supports at most 32 development keys.")
		return
	}
	public := make([]string, 0, len(keys))
	for _, key := range keys {
		public = append(public, key.Public)
	}
	if err = s.Host.Join(r.Context(), host.Account{Project: p.ID, Login: login, Identity: v.User.ID, Keys: public}); err != nil {
		jsonError(w, 502, "account_incomplete", "Native account provisioning was not confirmed. Membership was not recorded; ask the operator to inspect the account.")
		return
	}
	if err = s.Store.Join(r.Context(), p.ID, v.User.ID, login); err != nil {
		jsonError(w, 503, "membership_not_saved", "Native account provisioning returned but membership could not be saved. Ask the operator to inspect the retained account.")
		return
	}
	jsonResponse(w, 200, struct {
		Login string `json:"login"`
	}{login})
}
func (s *Server) apiEnvironmentMembers(w http.ResponseWriter, r *http.Request, v store.Session) {
	p, ok := s.loadEnvironment(w, r)
	if !ok {
		return
	}
	type memberView struct {
		UserID string `json:"user_id"`
		Login  string `json:"login"`
	}
	items := []memberView{}
	reader, allowed := s.authorizeEnvironmentRead(w, r, v, p)
	if !allowed {
		return
	}
	if reader.administrator {
		members, err := s.Store.Members(r.Context(), p.ID)
		if err != nil {
			jsonError(w, 503, "store_unavailable", "Could not list members.")
			return
		}
		for _, member := range members {
			items = append(items, memberView{strconv.FormatInt(member.UserID, 10), member.Login})
		}
	} else if reader.login != "" {
		items = append(items, memberView{strconv.FormatInt(v.User.ID, 10), reader.login})
	}
	jsonResponse(w, 200, struct {
		Items                []memberView `json:"items"`
		AuthorityUnavailable bool         `json:"authority_unavailable"`
	}{items, reader.authorityUnavailable})
}
func (s *Server) apiConnection(w http.ResponseWriter, r *http.Request, v store.Session) {
	p, ok := s.loadEnvironment(w, r)
	if !ok {
		return
	}
	login, err := s.Store.MemberLogin(r.Context(), p.ID, v.User.ID)
	if errors.Is(err, sql.ErrNoRows) {
		jsonError(w, 403, "join_required", "Explicitly join this environment to obtain connection details.")
		return
	}
	if err != nil {
		jsonError(w, 503, "store_unavailable", "Could not read membership.")
		return
	}
	connection, err := s.Host.Connection(r.Context(), p.ID)
	if err != nil {
		jsonError(w, 503, "native_unavailable", "Current address and public host key are unavailable; do not use a cached address as proof of access.")
		return
	}
	jsonResponse(w, 200, struct {
		Login           string          `json:"login"`
		Connection      host.Connection `json:"connection"`
		RoutingVerified bool            `json:"routing_verified"`
	}{login, connection, false})
}
