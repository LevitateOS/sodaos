package webapp

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"github.com/levitateos/sodaos/internal/webauth"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"time"

	"github.com/levitateos/sodaos/internal/config"
	"github.com/levitateos/sodaos/internal/host"
	"github.com/levitateos/sodaos/internal/projectos"
	"github.com/levitateos/sodaos/internal/store"
	"github.com/levitateos/sodaos/internal/tailnet"
)

// Project-local Linux names are a native provisioning constraint, not a
// restriction on Forgejo's own account names.
var projectLogin = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,30}$`)

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

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type createEnvironmentInput struct {
	RepositoryID string                    `json:"repository_id"`
	ProfileID    string                    `json:"profile_id"`
	Tailnet      *tailnet.ProjectSelection `json:"tailnet,omitempty"`
}

var (
	errStoreUnavailable     = errors.New("could not inspect reservation")
	errReservationFailed    = errors.New("repository already has a reservation")
	errProfileUnavailable   = errors.New("profile unavailable")
	errSessionCookieMissing = errors.New("session cookie missing")
	errSessionChanged       = errors.New("session changed")
)

func parseCreateEnvironmentInput(w http.ResponseWriter, r *http.Request) (*createEnvironmentInput, int64, bool) {
	var input createEnvironmentInput
	if !webauth.DecodeAPIObject(w, r, &input) {
		return nil, 0, false
	}
	repoID, valid := webauth.PositiveID(input.RepositoryID)
	if !valid || (input.ProfileID != "" && input.ProfileID != projectos.RockyHeadless) || (input.Tailnet != nil && input.Tailnet.Validate() != nil) {
		webauth.JSONError(w, 400, "invalid_repository", "Provide a repository_id and a supported profile_id.")
		return nil, 0, false
	}
	return &input, repoID, true
}

func (s *API) checkRepositoryReservation(ctx context.Context, repoID int64) error {
	_, err := s.Store.ProjectByRepository(ctx, repoID)
	if errors.Is(err, store.ErrNotFound) {
		return nil
	}
	if err != nil {
		return errStoreUnavailable
	}
	return errReservationFailed
}

func (s *API) precheckRepositoryAndReservation(w http.ResponseWriter, r *http.Request, v store.Session, repoID int64) bool {
	access, err := s.visibleRepository(r, v, repoID)
	if err != nil {
		webauth.ProviderError(w, err)
		return false
	}
	if access.repository.Owner.ID != access.actor.ID {
		webauth.JSONError(w, 403, "owner_required", "Only the human repository owner can create its environment. Organization-owned environments are not supported.")
		return false
	}
	if err := s.checkRepositoryReservation(r.Context(), repoID); err != nil {
		if errors.Is(err, errReservationFailed) {
			webauth.JSONError(w, 409, "reservation_failed", "Repository already has a reservation. Refresh; do not recreate it.")
		} else {
			webauth.JSONError(w, 503, "store_unavailable", "Could not inspect reservation.")
		}
		return false
	}
	return true
}

func (s *API) checkTailnetPreflight(ctx context.Context, sel *tailnet.ProjectSelection) error {
	if sel == nil || !sel.Enabled {
		return nil
	}
	options, err := s.Host.TailnetOptions(ctx)
	if err != nil {
		return err
	}
	if !options.Available {
		return tailnet.ErrUnsupported
	}
	if options.Revision != sel.Revision || options.Binding != sel.Binding {
		return tailnet.ErrConflict
	}
	return nil
}

func (s *API) resolveNativeProfile(ctx context.Context) (projectos.Profile, error) {
	tctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	profile, err := s.Host.ResolveProfile(tctx)
	if err != nil {
		return projectos.Profile{}, errProfileUnavailable
	}
	return profile, nil
}

func (s *API) precheckProfileAndTailnet(w http.ResponseWriter, ctx context.Context, sel *tailnet.ProjectSelection) (projectos.Profile, bool) {
	profile, err := s.resolveNativeProfile(ctx)
	if err != nil {
		webauth.JSONError(w, 422, "profile_unavailable", "The native installed Project OS is unavailable or incompatible. No reservation was created.")
		return projectos.Profile{}, false
	}
	if err := s.checkTailnetPreflight(ctx, sel); err != nil {
		tailnetError(w, err)
		return projectos.Profile{}, false
	}
	return profile, true
}

func (s *API) verifyCurrentSessionMatch(r *http.Request, v store.Session) error {
	cookie, err := webauth.RequestCookie(r, webauth.SessionCookie)
	if err != nil {
		return errSessionCookieMissing
	}
	current, err := s.Store.Session(r.Context(), cookie.Value)
	if err != nil || current.ContextID != v.ContextID || current.CSRF != v.CSRF || current.User.ID != v.User.ID {
		return errSessionChanged
	}
	return nil
}

func (s *API) reconfirmRepositoryAndSession(w http.ResponseWriter, r *http.Request, v store.Session, repoID int64) (repositoryAccess, bool) {
	access, err := s.visibleRepository(r, v, repoID)
	if err != nil {
		webauth.ProviderError(w, err)
		return repositoryAccess{}, false
	}
	if err := s.verifyCurrentSessionMatch(r, v); err != nil {
		if errors.Is(err, errSessionCookieMissing) {
			webauth.JSONError(w, 401, "unauthorized", "Reconnect to Soda.")
		} else {
			webauth.JSONError(w, 401, "unauthorized", "Soda context changed.")
		}
		return repositoryAccess{}, false
	}
	if access.repository.Owner.ID != v.User.ID {
		webauth.JSONError(w, 403, "owner_required", "Repository ownership changed.")
		return repositoryAccess{}, false
	}
	return access, true
}

func (s *API) provisionAndSaveProject(w http.ResponseWriter, ctx context.Context, p store.Project) (store.Project, bool) {
	w.Header().Set("Location", config.SodaPath+"/api/environments/"+p.ID)
	env, err := s.Host.Create(ctx, host.Create{ID: p.ID, Owner: p.OwnerID, Profile: p.Profile})
	if err != nil {
		webauth.JSONResponse(w, 502, struct {
			Error       apiError        `json:"error"`
			Environment EnvironmentView `json:"environment"`
		}{apiError{"provisioning_incomplete", "Reservation retained; native provisioning was not confirmed. Inspect this environment with the operator; do not recreate it."}, EnvironmentDTO(p)})
		return p, false
	}
	if err = s.Store.MarkReady(ctx, p.ID, env.IP); err != nil {
		webauth.JSONResponse(w, 503, struct {
			Error       apiError        `json:"error"`
			Environment EnvironmentView `json:"environment"`
		}{apiError{"result_not_saved", "Native provisioning returned, but its result was not saved. Inspect the retained environment; do not recreate it."}, EnvironmentDTO(p)})
		return p, false
	}
	p.Ready = true
	return p, true
}

func (s *API) applyCreatedEnvironmentTailnet(w http.ResponseWriter, r *http.Request, v store.Session, p store.Project, sel *tailnet.ProjectSelection) (string, bool) {
	if sel == nil || !sel.Enabled {
		return "", true
	}
	if !s.authorizeProjectTailnet(w, r, v, p) || !s.tailnetSession(w, r, v) {
		return "", false
	}
	network, err := s.Host.TailnetProject(r.Context(), tailnet.ProjectRequest{
		Project: p.ID, Action: "enable", Revision: "0", Binding: sel.Binding, ConfirmID: p.ID,
	})
	if !s.tailnetSession(w, r, v) || !s.authorizeProjectTailnet(w, r, v, p) {
		return "", false
	}
	if err == nil && network.Outcome == "queued" {
		return "queued", true
	}
	return "unconfirmed", true
}

func (s *API) apiCreateEnvironment(w http.ResponseWriter, r *http.Request, v store.Session) {
	input, repositoryID, ok := parseCreateEnvironmentInput(w, r)
	if !ok {
		return
	}
	if !s.precheckRepositoryAndReservation(w, r, v, repositoryID) {
		return
	}
	profile, ok := s.precheckProfileAndTailnet(w, r.Context(), input.Tailnet)
	if !ok {
		return
	}
	access, ok := s.reconfirmRepositoryAndSession(w, r, v, repositoryID)
	if !ok {
		return
	}
	repo := access.repository
	bytes := make([]byte, 12)
	rand.Read(bytes)
	p := store.Project{Profile: &profile, ID: "p" + hex.EncodeToString(bytes), Name: repo.Name, RepositoryID: repo.ID, OwnerID: v.User.ID, Repository: repo.FullName}
	if err := s.Store.CreateProject(r.Context(), p); err != nil {
		webauth.JSONError(w, 409, "reservation_failed", "Repository may already have an environment reservation. Refresh its environment before retrying.")
		return
	}
	p, ok = s.provisionAndSaveProject(w, r.Context(), p)
	if !ok {
		return
	}
	tailnetOutcome, ok := s.applyCreatedEnvironmentTailnet(w, r, v, p, input.Tailnet)
	if !ok {
		return
	}
	result := struct {
		EnvironmentView
		TailnetOutcome string `json:"tailnet_outcome,omitempty"`
	}{EnvironmentView: EnvironmentDTO(p), TailnetOutcome: tailnetOutcome}
	webauth.JSONResponse(w, 201, result)
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

func validJoinSSHSelection(selection string) bool {
	return selection == "" || selection == "saved" || selection == "none"
}

func (s *API) joinPublicKeys(ctx context.Context, userID int64, selection string) ([]string, error) {
	if selection == "none" {
		return []string{}, nil
	}
	keys, err := s.Store.Keys(ctx, userID)
	if err != nil {
		return nil, err
	}
	if len(keys) > 32 {
		return nil, errTooManyJoinKeys
	}
	public := make([]string, 0, len(keys))
	for _, key := range keys {
		public = append(public, key.Public)
	}
	return public, nil
}

var errTooManyJoinKeys = errors.New("too many development keys")

func (s *API) persistEnvironmentJoin(ctx context.Context, r *http.Request, v store.Session, p store.Project, login string, public []string) error {
	cookie, cookieErr := webauth.RequestCookie(r, webauth.SessionCookie)
	if cookieErr != nil || s.Auth.RequireCurrentSession(ctx, cookie.Value, v) != nil {
		return errJoinUnauthorized
	}
	if err := s.Host.Join(ctx, host.Account{Project: p.ID, Login: login, Identity: v.User.ID, Keys: public}); err != nil {
		return errJoinAccountIncomplete
	}
	if err := s.Store.Join(ctx, p.ID, v.User.ID, login); err != nil {
		return errJoinMembershipNotSaved
	}
	return nil
}

var (
	errJoinUnauthorized       = errors.New("join unauthorized")
	errJoinAccountIncomplete  = errors.New("join account incomplete")
	errJoinMembershipNotSaved = errors.New("join membership not saved")
)

func (s *API) writeJoinLogin(w http.ResponseWriter, login string) {
	webauth.JSONResponse(w, 200, struct {
		Login string `json:"login"`
	}{login})
}

func (s *API) reportJoinPersist(w http.ResponseWriter, login string, err error) {
	switch err {
	case errJoinUnauthorized:
		webauth.JSONError(w, 401, "unauthorized", "Soda context changed. Reconnect before acting.")
	case errJoinAccountIncomplete:
		webauth.JSONError(w, 502, "account_incomplete", "Native account provisioning was not confirmed. Membership was not recorded; ask the operator to inspect the account.")
	case errJoinMembershipNotSaved:
		webauth.JSONError(w, 503, "membership_not_saved", "Native account provisioning returned but membership could not be saved. Ask the operator to inspect the retained account.")
	case nil:
		s.writeJoinLogin(w, login)
	}
}

func (s *API) admitNewJoin(w http.ResponseWriter, r *http.Request, v store.Session, p store.Project, sshKeys string) (login string, public []string, ok bool) {
	access, err := s.visibleRepository(r, v, p.RepositoryID)
	if err != nil {
		webauth.ProviderError(w, err)
		return "", nil, false
	}
	if !p.Ready {
		webauth.JSONError(w, 409, "not_provisioned", "Environment provisioning is incomplete.")
		return "", nil, false
	}
	login = access.actor.Login
	if !projectLogin.MatchString(login) || login == "root" {
		webauth.JSONError(w, 422, "unsupported_linux_login", "Your Forgejo username is not supported as a project Linux account. No automatic rename is performed.")
		return "", nil, false
	}
	// Empty legacy requests retain saved-key behavior. New browser callers can
	// explicitly choose account-only provisioning even when saved SSH keys exist.
	public, err = s.joinPublicKeys(r.Context(), v.User.ID, sshKeys)
	if errors.Is(err, errTooManyJoinKeys) {
		webauth.JSONError(w, 422, "too_many_keys", "Native onboarding supports at most 32 development keys.")
		return "", nil, false
	}
	if err != nil {
		webauth.JSONError(w, 503, "store_unavailable", "Could not read development keys.")
		return "", nil, false
	}
	return login, public, true
}

func (s *API) apiJoinEnvironment(w http.ResponseWriter, r *http.Request, v store.Session) {
	var input struct {
		SSHKeys string `json:"ssh_keys"`
	}
	if !webauth.DecodeAPIObject(w, r, &input) {
		return
	}
	p, ok := s.loadEnvironment(w, r)
	if !ok {
		return
	}
	if !validJoinSSHSelection(input.SSHKeys) {
		webauth.JSONError(w, 400, "invalid_ssh_selection", "Select saved or none for optional external SSH keys.")
		return
	}
	login, err := s.Store.MemberLogin(r.Context(), p.ID, v.User.ID)
	if err == nil {
		s.writeJoinLogin(w, login)
		return
	}
	if !errors.Is(err, store.ErrNotFound) {
		webauth.JSONError(w, 503, "store_unavailable", "Could not inspect membership.")
		return
	}
	login, public, ok := s.admitNewJoin(w, r, v, p, input.SSHKeys)
	if !ok {
		return
	}
	// Fresh admission after provider/state I/O, not rollback after dispatch.
	s.reportJoinPersist(w, login, s.persistEnvironmentJoin(r.Context(), r, v, p, login, public))
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
