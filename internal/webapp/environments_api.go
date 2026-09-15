package webapp

import (
	"errors"
	"net/http"
	"net/url"
	"strconv"

	"github.com/levitateos/sodaos/internal/host"
	"github.com/levitateos/sodaos/internal/projectos"
	"github.com/levitateos/sodaos/internal/store"
	"github.com/levitateos/sodaos/internal/webauth"
)

func (s *API) environmentRoutes() {
	s.mux.HandleFunc("/api/repositories", s.Auth.Protected(s.apiRepositories, http.MethodGet))
	s.mux.HandleFunc("/api/environments/{id}/os", s.Auth.Protected(s.apiEnvironmentOS, "GET"))
	s.mux.HandleFunc("/api/repositories/{repositoryID}/profiles", s.Auth.Protected(s.apiProjectProfiles, "GET"))
	s.mux.HandleFunc("/api/environments/{id}/lifecycle", s.Auth.Protected(s.apiLifecycle, "GET", "POST"))
	s.mux.HandleFunc("/api/environments/{id}/access-keys", s.Auth.Protected(s.apiAccessKeys, "GET", "POST"))
	s.mux.HandleFunc("/api/environments/{id}/terminal", s.apiTerminal)
	s.mux.HandleFunc("/api/environments/{id}/terminal-session", s.Auth.Protected(func(w http.ResponseWriter, r *http.Request, _ store.Session) {
		webauth.JSONError(w, 410, "terminal_client_obsolete", "Reload this page; terminal actions now require an exact ID.")
	}, http.MethodGet, http.MethodPost))
	s.mux.HandleFunc("/api/environments/{id}/terminal-sessions/{terminalID}", s.Auth.Protected(s.apiTerminalSession, http.MethodGet, http.MethodPost))
	s.mux.HandleFunc("/api/environments/{id}/terminal-sessions", s.Auth.Protected(s.apiReserveTerminal, http.MethodPost))
	s.mux.HandleFunc("/api/spaces", s.Auth.Protected(s.apiSpaces, http.MethodGet))
	s.mux.HandleFunc("/api/environments", s.Auth.Protected(s.apiEnvironments, "GET", "POST"))
	s.mux.HandleFunc("/api/environments/{id}", s.Auth.Protected(s.apiEnvironment, "GET"))
	s.mux.HandleFunc("/api/environments/{id}/join", s.Auth.Protected(s.apiJoinEnvironment, "POST"))
	s.mux.HandleFunc("/api/environments/{id}/members", s.Auth.Protected(s.apiEnvironmentMembers, "GET"))
	s.mux.HandleFunc("/api/environments/{id}/connection", s.Auth.Protected(s.apiConnection, "GET"))
}

type EnvironmentView struct {
	Profile      *projectos.Profile `json:"profile"`
	ID           string             `json:"id"`
	Name         string             `json:"name"`
	RepositoryID string             `json:"repository_id"`
	Repository   string             `json:"repository"`
	OwnerID      string             `json:"owner_id"`
	Provisioned  bool               `json:"provisioned"`
}

type repositoryContextView struct {
	ID      string `json:"id"`
	OwnerID string `json:"owner_id"`
	Owner   string `json:"owner"`
	Name    string `json:"name"`
}

func EnvironmentDTO(p store.Project) EnvironmentView {
	return EnvironmentView{p.Profile, p.ID, p.Name, strconv.FormatInt(p.RepositoryID, 10), p.Repository, strconv.FormatInt(p.OwnerID, 10), p.Ready}
}

func environmentListQuery(query url.Values, err error) (int64, bool) {
	id, valid := webauth.PositiveID(query.Get("repository_id"))
	return id, err == nil && len(query) == 1 && len(query["repository_id"]) == 1 && valid
}

func parseEnvironmentsListQuery(w http.ResponseWriter, r *http.Request) (int64, bool) {
	if len(r.URL.RawQuery) > 8192 {
		webauth.JSONError(w, 400, "invalid_repository", "Provide one repository_id.")
		return 0, false
	}
	query, err := url.ParseQuery(r.URL.RawQuery)
	id, ok := environmentListQuery(query, err)
	if !ok {
		webauth.JSONError(w, 400, "invalid_repository", "Provide one repository_id.")
		return 0, false
	}
	return id, true
}

func listedEnvironments(p store.Project, absent bool) []EnvironmentView {
	items := []EnvironmentView{}
	if !absent {
		items = append(items, EnvironmentDTO(p))
	}
	return items
}

func (s *API) requireListedSession(w http.ResponseWriter, r *http.Request, v store.Session) bool {
	cookie, cookieErr := webauth.RequestCookie(r, webauth.SessionCookie)
	if cookieErr != nil || s.Auth.RequireCurrentSession(r.Context(), cookie.Value, v) != nil {
		webauth.JSONError(w, 401, "unauthenticated", "Soda context changed; reconnect.")
		return false
	}
	return true
}

func (s *API) apiEnvironments(w http.ResponseWriter, r *http.Request, v store.Session) {
	if r.Method == "POST" {
		s.apiCreateEnvironment(w, r, v)
		return
	}
	id, ok := parseEnvironmentsListQuery(w, r)
	if !ok {
		return
	}
	access, err := s.visibleRepository(r, v, id)
	if err != nil {
		webauth.ProviderError(w, err)
		return
	}
	p, err := s.Store.ProjectByRepository(r.Context(), id)
	absent := errors.Is(err, store.ErrNotFound)
	if err != nil && !absent {
		webauth.JSONError(w, 503, "store_unavailable", "Could not read environment.")
		return
	}
	if !s.requireListedSession(w, r, v) {
		return
	}
	webauth.JSONResponse(w, 200, struct {
		Items      []EnvironmentView     `json:"items"`
		Repository repositoryContextView `json:"repository"`
		CanCreate  bool                  `json:"can_create"`
	}{Items: listedEnvironments(p, absent), Repository: repositoryContextView{strconv.FormatInt(id, 10), strconv.FormatInt(access.repository.Owner.ID, 10), access.repository.Owner.Login, access.repository.Name}, CanCreate: absent && access.repository.Owner.ID == v.User.ID})
}

func (s *API) loadEnvironment(w http.ResponseWriter, r *http.Request) (store.Project, bool) {
	p, err := s.Store.Project(r.Context(), r.PathValue("id"))
	if errors.Is(err, store.ErrNotFound) {
		webauth.JSONError(w, 404, "not_found", "Environment not found.")
		return p, false
	}
	if err != nil {
		webauth.JSONError(w, 503, "store_unavailable", "Could not read environment.")
		return p, false
	}
	return p, true
}

func environmentProfileMismatch(p store.Project, env host.Environment) bool {
	return p.Profile != nil && (env.Profile == nil || *p.Profile != *env.Profile)
}

func observedEnvironment(p store.Project, env host.Environment, nativeErr error) (*host.Environment, error) {
	if nativeErr == nil && environmentProfileMismatch(p, env) {
		nativeErr = errors.New("creation profile mismatch")
	}
	if nativeErr != nil {
		return nil, nativeErr
	}
	return &env, nil
}

func (s *API) apiEnvironment(w http.ResponseWriter, r *http.Request, v store.Session) {
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
	observed, nativeErr := observedEnvironment(p, env, nativeErr)
	if !s.requireListedSession(w, r, v) {
		return
	}
	webauth.JSONResponse(w, 200, struct {
		AuthorityUnavailable bool              `json:"authority_unavailable"`
		Environment          EnvironmentView   `json:"environment"`
		Observed             *host.Environment `json:"observed"`
		NativeUnavailable    bool              `json:"native_unavailable"`
		Login                string            `json:"login"`
		Administrator        bool              `json:"environment_administrator"`
	}{reader.authorityUnavailable, EnvironmentDTO(p), observed, nativeErr != nil, reader.login, reader.administrator})
}

func (s *API) apiEnvironmentMembers(w http.ResponseWriter, r *http.Request, v store.Session) {
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
			webauth.JSONError(w, 503, "store_unavailable", "Could not list members.")
			return
		}
		for _, member := range members {
			items = append(items, memberView{strconv.FormatInt(member.UserID, 10), member.Login})
		}
	} else if reader.login != "" {
		items = append(items, memberView{strconv.FormatInt(v.User.ID, 10), reader.login})
	}
	cookie, cookieErr := webauth.RequestCookie(r, webauth.SessionCookie)
	if cookieErr != nil || s.Auth.RequireCurrentSession(r.Context(), cookie.Value, v) != nil {
		webauth.JSONError(w, 401, "unauthenticated", "Soda context changed; reconnect.")
		return
	}
	webauth.JSONResponse(w, 200, struct {
		Items                []memberView `json:"items"`
		AuthorityUnavailable bool         `json:"authority_unavailable"`
	}{items, reader.authorityUnavailable})
}

func (s *API) apiConnection(w http.ResponseWriter, r *http.Request, v store.Session) {
	p, ok := s.loadEnvironment(w, r)
	if !ok {
		return
	}
	login, err := s.Store.MemberLogin(r.Context(), p.ID, v.User.ID)
	if errors.Is(err, store.ErrNotFound) {
		webauth.JSONError(w, 403, "join_required", "Explicitly join this environment to obtain connection details.")
		return
	}
	if err != nil {
		webauth.JSONError(w, 503, "store_unavailable", "Could not read membership.")
		return
	}
	connection, err := s.Host.Connection(r.Context(), p.ID)
	if err != nil {
		webauth.JSONError(w, 503, "native_unavailable", "Current address and public host key are unavailable; do not use a cached address as proof of access.")
		return
	}
	cookie, cookieErr := webauth.RequestCookie(r, webauth.SessionCookie)
	if cookieErr != nil || s.Auth.RequireCurrentSession(r.Context(), cookie.Value, v) != nil {
		webauth.JSONError(w, 401, "unauthenticated", "Soda context changed; reconnect.")
		return
	}
	webauth.JSONResponse(w, 200, struct {
		Login           string          `json:"login"`
		Connection      host.Connection `json:"connection"`
		RoutingVerified bool            `json:"routing_verified"`
	}{login, connection, false})
}
