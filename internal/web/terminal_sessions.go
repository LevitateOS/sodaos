package web

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/levitateos/sodaos/internal/host"
	"github.com/levitateos/sodaos/internal/store"
)

var browserTerminalID = regexp.MustCompile(`^[0-9a-f]{32}$`)

func newTerminalID() string { var b [16]byte; rand.Read(b[:]); return hex.EncodeToString(b[:]) }

type terminalPeer struct {
	token, contextID, project, id string
	cancel                        context.CancelFunc
}

// Native reservation consumption is serialized with End and Create in the
// project. A web restart or late helper dial cannot recreate an ended locator.
func terminalCreationScope(v store.Session) string {
	sum := sha256.Sum256([]byte(v.ContextID + "\x00" + v.CSRF))
	return hex.EncodeToString(sum[:])
}

type terminalView struct {
	host.TerminalState
	EnvironmentID string `json:"environment_id"`
	RepositoryID  string `json:"repository_id"`
	UserID        string `json:"user_id"`
	Login         string `json:"login"`
}

func terminalDTO(item host.TerminalState, v store.Session, p store.Project, login string) terminalView {
	return terminalView{item, p.ID, strconv.FormatInt(p.RepositoryID, 10), strconv.FormatInt(v.User.ID, 10), login}
}

func validTerminalName(name string) bool { return host.ValidTerminalName(name) }

// Every map access is under terminalMu. No provider/native IO runs under it.
func (s *Server) cancelTerminals(contextID, token string) {
	for _, peer := range s.terminalPeers {
		if (contextID != "" && peer.contextID == contextID) || (token != "" && peer.token == token) {
			peer.cancel()
		}
	}
}
func (s *Server) CloseTerminals() {
	s.terminalMu.Lock()
	s.terminalClosed = true
	for _, peer := range s.terminalPeers {
		peer.cancel()
	}
	s.terminalMu.Unlock()
	s.terminalWG.Wait()
}
func (s *Server) terminalCurrent(ctx context.Context, token string, original store.Session, project store.Project, login string) bool {
	check, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	current, err := s.Store.Session(check, token)
	if err != nil || current.User.ID != original.User.ID || current.ContextID != original.ContextID || current.CSRF != original.CSRF {
		return false
	}
	p, err := s.Store.Project(check, project.ID)
	if err != nil || !p.Ready || p.RepositoryID != project.RepositoryID {
		return false
	}
	member, err := s.Store.MemberLogin(check, project.ID, current.User.ID)
	return err == nil && member == login
}

func (s *Server) terminalAccount(w http.ResponseWriter, r *http.Request, v store.Session) (store.Project, string, string, bool) {
	if r.URL.RawQuery != "" || r.URL.ForceQuery {
		jsonError(w, 400, "invalid_request", "No query parameters are accepted.")
		return store.Project{}, "", "", false
	}
	p, ok := s.loadEnvironment(w, r)
	if !ok {
		return p, "", "", false
	}
	login, err := s.Store.MemberLogin(r.Context(), p.ID, v.User.ID)
	if err != nil || !p.Ready || login == "root" || !projectLogin.MatchString(login) {
		jsonError(w, 403, "membership_required", "An existing account is required.")
		return p, "", "", false
	}
	check, done := context.WithTimeout(r.Context(), 10*time.Second)
	defer done()
	if _, err = s.visibleRepository(r.WithContext(check), v, p.RepositoryID); err != nil {
		providerError(w, err)
		return p, "", "", false
	}
	cookie, err := requestCookie(r, sessionCookie)
	if err != nil {
		jsonError(w, 401, "unauthenticated", "Sign in again.")
		return p, "", "", false
	}
	return p, login, cookie.Value, true
}

func (s *Server) apiReserveTerminal(w http.ResponseWriter, r *http.Request, v store.Session) {
	var in struct {
		Cols int    `json:"cols"`
		Rows int    `json:"rows"`
		Name string `json:"name"`
	}
	if !decodeAPIObject(w, r, &in) {
		return
	}
	if in.Cols < 2 || in.Cols > 500 || in.Rows < 2 || in.Rows > 300 || !validTerminalName(in.Name) {
		jsonError(w, 400, "invalid_request", "Choose bounded terminal dimensions and name.")
		return
	}
	p, login, token, ok := s.terminalAccount(w, r, v)
	if !ok {
		return
	}
	id := newTerminalID()
	items, err := s.terminalOperation(r, v, p, login, token, host.TerminalRequest{Action: "reserve", ID: id, Cols: in.Cols, Rows: in.Rows, Name: in.Name, Scope: terminalCreationScope(v)})
	if err != nil || len(items) != 1 || items[0].ID != id {
		jsonError(w, 503, "terminal_unavailable", "Terminal reservation unavailable; no shell creation was requested.")
		return
	}
	jsonResponse(w, 201, map[string]string{"id": id})
}

// Short native operations share logout/shutdown cancellation with attachments,
// not their lifetime. Lookups remain possible after a browser/web restart.
func (s *Server) terminalOperation(r *http.Request, v store.Session, p store.Project, login, token string, in host.TerminalRequest) ([]host.TerminalState, error) {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	s.terminalMu.Lock()
	read := in.Action == "list" || in.Action == "inspect"
	if s.terminalClosed || (!read && s.terminalStopping[p.ID]) || len(s.terminalPeers) >= 128 || !s.terminalCurrent(ctx, token, v, p, login) {
		s.terminalMu.Unlock()
		return nil, errors.New("terminal operation unavailable")
	}
	if s.terminalPeers == nil {
		s.terminalPeers = make(map[*http.Request]*terminalPeer)
	}
	s.terminalPeers[r] = &terminalPeer{token: token, contextID: v.ContextID, project: p.ID, cancel: cancel}
	s.terminalWG.Add(1)
	s.terminalMu.Unlock()
	defer s.terminalWG.Done()
	defer func() { s.terminalMu.Lock(); delete(s.terminalPeers, r); s.terminalMu.Unlock() }()
	in.Project, in.Login, in.Identity = p.ID, login, v.User.ID
	items, err := s.Host.TerminalStates(ctx, in)
	s.terminalMu.Lock()
	current := ctx.Err() == nil && !s.terminalClosed && s.terminalCurrent(ctx, token, v, p, login)
	s.terminalMu.Unlock()
	if !current {
		return nil, errors.New("terminal authorization ended")
	}
	return items, err
}

func terminalMetadata(w http.ResponseWriter, view *terminalView) {
	jsonResponse(w, 200, struct {
		Terminal *terminalView `json:"terminal"`
	}{view})
}
func (s *Server) apiTerminalSession(w http.ResponseWriter, r *http.Request, v store.Session) {
	id := r.PathValue("terminalID")
	if !browserTerminalID.MatchString(id) {
		jsonError(w, 400, "invalid_request", "Provide an exact terminal ID.")
		return
	}
	in := host.TerminalRequest{Action: "inspect", ID: id}
	if r.Method == "POST" {
		var action struct {
			Action string  `json:"action"`
			Name   *string `json:"name"`
		}
		if !decodeAPIObject(w, r, &action) {
			return
		}
		if action.Action != "end" && action.Action != "rename" || action.Action == "end" && action.Name != nil || action.Action == "rename" && (action.Name == nil || !validTerminalName(*action.Name)) {
			jsonError(w, 400, "invalid_action", "Choose End or Rename for this exact terminal.")
			return
		}
		in.Action = action.Action
		if action.Name != nil {
			in.Name = *action.Name
		}
	}
	p, login, token, ok := s.terminalAccount(w, r, v)
	if !ok {
		return
	}
	items, err := s.terminalOperation(r, v, p, login, token, in)
	if err != nil {
		jsonError(w, 503, "terminal_unavailable", "The exact native outcome is unavailable. Nothing was retried or replaced.")
		return
	}
	if len(items) == 0 {
		terminalMetadata(w, nil)
		return
	}
	view := terminalDTO(items[0], v, p, login)
	terminalMetadata(w, &view)
}
