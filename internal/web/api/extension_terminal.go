package api

import (
	"context"
	"errors"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/levitateos/sodaos/internal/host"
	"github.com/levitateos/sodaos/internal/store"
	"github.com/levitateos/sodaos/internal/web/auth"
)

// ExtensionHandler is mounted only on the private Soda service socket. It
// reuses the existing auth service for nonterminal routes.
func (s *API) ExtensionHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/environments/{id}/terminal", s.extensionTerminalStream)
	mux.HandleFunc("/api/environments/{id}/terminal-sessions", s.extensionReserveTerminal)
	mux.HandleFunc("/api/environments/{id}/terminal-sessions/{terminalID}", s.extensionTerminalSession)
	mux.HandleFunc("/api/factory/runs/{runID}/output", s.factoryOutputStream)
	s.registerExtensionProductRoutes(mux)
	mux.Handle("/", s.Auth.ExtensionHandler())
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawPath != "" || r.URL.ForceQuery ||
			strings.Contains(r.URL.Path, "\\") || path.Clean(r.URL.Path) != r.URL.Path {
			http.NotFound(w, r)
			return
		}
		if !privateTerminalStreamRoute(r) && !privateFactoryOutputRoute(r) {
			r.Body = http.MaxBytesReader(w, r.Body, auth.APIBodyLimit)
		}
		mux.ServeHTTP(w, r)
	})
}

func privateTerminalStreamRoute(r *http.Request) bool {
	parts := strings.Split(r.URL.Path, "/")
	return r.Method == http.MethodGet && len(parts) == 5 && parts[1] == "api" && parts[2] == "environments" && parts[3] != "" && parts[4] == "terminal"
}

func (s *API) extensionTerminalOperation(r *http.Request, identity extensionTerminalIdentity, in host.TerminalRequest) ([]host.TerminalState, error) {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	s.terminalMu.Lock()
	if s.terminalClosed || (in.Action != "inspect" && s.TerminalStopping[identity.project.ID]) || len(s.TerminalPeers) >= 128 {
		s.terminalMu.Unlock()
		return nil, errors.New("terminal unavailable")
	}
	if s.TerminalPeers == nil {
		s.TerminalPeers = make(map[*http.Request]*TerminalPeer)
	}
	s.TerminalPeers[r] = &TerminalPeer{ContextID: identity.authority.SessionGeneration, Project: identity.project.ID, Cancel: cancel}
	s.terminalWG.Add(1)
	s.terminalMu.Unlock()
	defer s.terminalWG.Done()
	defer s.dropTerminalPeer(r)
	if !s.extensionTerminalCurrent(ctx, r, identity) {
		return nil, errors.New("terminal authorization ended")
	}
	in.Project, in.Login, in.Identity = identity.project.ID, identity.login, identity.actorID
	items, err := s.Host.TerminalStates(ctx, in)
	if !s.extensionTerminalCurrent(ctx, r, identity) {
		return nil, errors.New("terminal authorization ended")
	}
	return items, err
}

func (s *API) extensionTerminalStates(r *http.Request, ctx context.Context, v store.Session, p store.Project, login, action string) ([]host.TerminalState, error) {
	authority, ok := requestExtensionAuthority(r)
	actorID, valid := s.nativeTerminalActor(authority)
	if !nativeTerminalSessionActor(authority, ok, valid, v) {
		return nil, errors.New("native terminal authority unavailable")
	}
	currentProject, err := s.Store.Project(ctx, p.ID)
	if err != nil || !currentProject.Ready || currentProject.RepositoryID != p.RepositoryID {
		return nil, errors.New("terminal project changed")
	}
	member, err := s.Store.MemberLogin(ctx, p.ID, actorID)
	if !nativeTerminalMember(currentProject, member, err) || member != login {
		return nil, errors.New("terminal member changed")
	}
	identity := extensionTerminalIdentity{authority: authority, project: currentProject, login: login, actorID: actorID}
	return s.extensionTerminalOperation(r.WithContext(ctx), identity, host.TerminalRequest{Action: action})
}

func extensionTerminalView(item host.TerminalState, identity extensionTerminalIdentity) TerminalView {
	return TerminalView{item, identity.project.ID, strconv.FormatInt(identity.project.RepositoryID, 10), strconv.FormatInt(identity.actorID, 10), identity.login}
}

func (s *API) extensionReserveTerminal(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		auth.JSONError(w, 405, "method_not_allowed", "POST required.")
		return
	}
	if !extensionTerminalOrigin(r, s.Config.ForgejoURL) {
		auth.JSONError(w, 403, "invalid_origin", "Same-origin terminal required.")
		return
	}
	identity, ok := s.extensionTerminalAccount(w, r, true)
	if !ok {
		return
	}
	var in struct {
		Cols int    `json:"cols"`
		Rows int    `json:"rows"`
		Name string `json:"name"`
	}
	if !auth.DecodeAPIObject(w, r, &in) {
		return
	}
	if !validTerminalGeometry(in.Cols, in.Rows, in.Name) {
		auth.JSONError(w, 400, "invalid_request", "Choose bounded terminal dimensions and name.")
		return
	}
	id := newTerminalID()
	items, err := s.extensionTerminalOperation(r, identity, host.TerminalRequest{Action: "reserve", ID: id, Cols: in.Cols, Rows: in.Rows, Name: in.Name, Scope: nativeTerminalScope(identity.authority.SessionGeneration)})
	if !reservedTerminal(items, id, err) {
		auth.JSONError(w, 503, "terminal_unavailable", "Terminal reservation unavailable; no shell creation was requested.")
		return
	}
	auth.JSONResponse(w, 201, map[string]string{"id": id})
}

func (s *API) extensionTerminalSession(w http.ResponseWriter, r *http.Request) {
	if !extensionTerminalSessionMethod(w, r, s.Config.ForgejoURL) {
		return
	}
	identity, ok := s.extensionTerminalAccount(w, r, true)
	if !ok {
		return
	}
	id := r.PathValue("terminalID")
	if !TerminalID.MatchString(id) {
		auth.JSONError(w, 400, "invalid_request", "Provide an exact terminal ID.")
		return
	}
	in := host.TerminalRequest{Action: "inspect", ID: id}
	if !parseTerminalSessionAction(w, r, &in) {
		return
	}
	items, err := s.extensionTerminalOperation(r, identity, in)
	if err != nil {
		auth.JSONError(w, 503, "terminal_unavailable", "The exact native outcome is unavailable. Nothing was retried or replaced.")
		return
	}
	if len(items) == 0 {
		terminalMetadata(w, nil)
		return
	}
	view := extensionTerminalView(items[0], identity)
	terminalMetadata(w, &view)
}

func extensionTerminalSessionMethod(w http.ResponseWriter, r *http.Request, origin string) bool {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		auth.JSONError(w, 405, "method_not_allowed", "GET or POST required.")
		return false
	}
	if r.Method == http.MethodPost && !extensionTerminalOrigin(r, origin) {
		auth.JSONError(w, 403, "invalid_origin", "Same-origin terminal required.")
		return false
	}
	return true
}

func (s *API) extensionTerminalStream(w http.ResponseWriter, r *http.Request) {
	if !checkTerminalRequestHeaders(w, r, s.Config.ForgejoURL) {
		return
	}
	identity, ok := s.extensionTerminalAccount(w, r, false)
	if !ok {
		return
	}
	s.extensionTerminalAttach(w, r, identity)
}
