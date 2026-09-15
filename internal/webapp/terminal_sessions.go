package webapp

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"github.com/levitateos/sodaos/internal/webauth"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/levitateos/sodaos/internal/host"
	"github.com/levitateos/sodaos/internal/store"
)

var BrowserTerminalID = regexp.MustCompile(`^[0-9a-f]{32}$`)

func newTerminalID() string { var b [16]byte; rand.Read(b[:]); return hex.EncodeToString(b[:]) }

// Native reservation consumption is serialized with End and Create in the
// project. A web restart or late helper dial cannot recreate an ended locator.
func TerminalCreationScope(v store.Session) string {
	sum := sha256.Sum256([]byte(v.ContextID + "\x00" + v.CSRF))
	return hex.EncodeToString(sum[:])
}

type TerminalView struct {
	host.TerminalState
	EnvironmentID string `json:"environment_id"`
	RepositoryID  string `json:"repository_id"`
	UserID        string `json:"user_id"`
	Login         string `json:"login"`
}

func terminalDTO(item host.TerminalState, v store.Session, p store.Project, login string) TerminalView {
	return TerminalView{item, p.ID, strconv.FormatInt(p.RepositoryID, 10), strconv.FormatInt(v.User.ID, 10), login}
}

func validTerminalName(name string) bool { return host.ValidTerminalName(name) }

// Every map access is under terminalMu. No provider/native IO runs under it.
func (s *API) cancelTerminals(contextID, token string) {
	for _, peer := range s.TerminalPeers {
		if (contextID != "" && peer.ContextID == contextID) || (token != "" && peer.Token == token) {
			peer.Cancel()
		}
	}
}

func (s *API) CloseTerminals() {
	s.terminalMu.Lock()
	s.terminalClosed = true
	for _, peer := range s.TerminalPeers {
		peer.Cancel()
	}
	s.terminalMu.Unlock()
	s.terminalWG.Wait()
}

func (s *API) terminalCurrent(ctx context.Context, token string, original store.Session, project store.Project, login string) bool {
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

func rejectTerminalQuery(w http.ResponseWriter, r *http.Request) bool {
	if r.URL.RawQuery != "" || r.URL.ForceQuery {
		webauth.JSONError(w, 400, "invalid_request", "No query parameters are accepted.")
		return true
	}
	return false
}

func terminalMemberReady(p store.Project, login string, err error) bool {
	return err == nil && p.Ready && login != "root" && projectLogin.MatchString(login)
}

func (s *API) terminalAccount(w http.ResponseWriter, r *http.Request, v store.Session) (store.Project, string, string, bool) {
	if rejectTerminalQuery(w, r) {
		return store.Project{}, "", "", false
	}
	p, ok := s.loadEnvironment(w, r)
	if !ok {
		return p, "", "", false
	}
	login, err := s.Store.MemberLogin(r.Context(), p.ID, v.User.ID)
	if !terminalMemberReady(p, login, err) {
		webauth.JSONError(w, 403, "membership_required", "An existing account is required.")
		return p, "", "", false
	}
	check, done := context.WithTimeout(r.Context(), 10*time.Second)
	defer done()
	if _, err = s.visibleRepository(r.WithContext(check), v, p.RepositoryID); err != nil {
		webauth.ProviderError(w, err)
		return p, "", "", false
	}
	cookie, err := webauth.RequestCookie(r, webauth.SessionCookie)
	if err != nil {
		webauth.JSONError(w, 401, "unauthenticated", "Sign in again.")
		return p, "", "", false
	}
	return p, login, cookie.Value, true
}

func validTerminalGeometry(cols, rows int, name string) bool {
	return cols >= 2 && cols <= 500 && rows >= 2 && rows <= 300 && validTerminalName(name)
}

func reservedTerminal(items []host.TerminalState, id string, err error) bool {
	return err == nil && len(items) == 1 && items[0].ID == id
}

func (s *API) apiReserveTerminal(w http.ResponseWriter, r *http.Request, v store.Session) {
	var in struct {
		Cols int    `json:"cols"`
		Rows int    `json:"rows"`
		Name string `json:"name"`
	}
	if !webauth.DecodeAPIObject(w, r, &in) {
		return
	}
	if !validTerminalGeometry(in.Cols, in.Rows, in.Name) {
		webauth.JSONError(w, 400, "invalid_request", "Choose bounded terminal dimensions and name.")
		return
	}
	p, login, token, ok := s.terminalAccount(w, r, v)
	if !ok {
		return
	}
	id := newTerminalID()
	items, err := s.terminalOperation(r, v, p, login, token, host.TerminalRequest{Action: "reserve", ID: id, Cols: in.Cols, Rows: in.Rows, Name: in.Name, Scope: TerminalCreationScope(v)})
	if !reservedTerminal(items, id, err) {
		webauth.JSONError(w, 503, "terminal_unavailable", "Terminal reservation unavailable; no shell creation was requested.")
		return
	}
	webauth.JSONResponse(w, 201, map[string]string{"id": id})
}

func terminalReadAction(action string) bool {
	return action == "list" || action == "inspect"
}

func (s *API) terminalOperationAdmitted(ctx context.Context, token string, v store.Session, p store.Project, login string, read bool) bool {
	return !s.terminalClosed && (read || !s.TerminalStopping[p.ID]) && len(s.TerminalPeers) < 128 && s.terminalCurrent(ctx, token, v, p, login)
}

func (s *API) terminalStillAuthorized(ctx context.Context, token string, v store.Session, p store.Project, login string) bool {
	return ctx.Err() == nil && !s.terminalClosed && s.terminalCurrent(ctx, token, v, p, login)
}

func (s *API) dropTerminalPeer(r *http.Request) {
	s.terminalMu.Lock()
	delete(s.TerminalPeers, r)
	s.terminalMu.Unlock()
}

// Short native operations share logout/shutdown cancellation with attachments,
// not their lifetime. Lookups remain possible after a browser/web restart.
func (s *API) terminalOperation(r *http.Request, v store.Session, p store.Project, login, token string, in host.TerminalRequest) ([]host.TerminalState, error) {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	s.terminalMu.Lock()
	if !s.terminalOperationAdmitted(ctx, token, v, p, login, terminalReadAction(in.Action)) {
		s.terminalMu.Unlock()
		return nil, errors.New("terminal operation unavailable")
	}
	if s.TerminalPeers == nil {
		s.TerminalPeers = make(map[*http.Request]*TerminalPeer)
	}
	s.TerminalPeers[r] = &TerminalPeer{Token: token, ContextID: v.ContextID, Project: p.ID, Cancel: cancel}
	s.terminalWG.Add(1)
	s.terminalMu.Unlock()
	defer s.terminalWG.Done()
	defer s.dropTerminalPeer(r)
	in.Project, in.Login, in.Identity = p.ID, login, v.User.ID
	items, err := s.Host.TerminalStates(ctx, in)
	s.terminalMu.Lock()
	current := s.terminalStillAuthorized(ctx, token, v, p, login)
	s.terminalMu.Unlock()
	if !current {
		return nil, errors.New("terminal authorization ended")
	}
	return items, err
}

func terminalMetadata(w http.ResponseWriter, view *TerminalView) {
	webauth.JSONResponse(w, 200, struct {
		Terminal *TerminalView `json:"terminal"`
	}{view})
}

type terminalSessionMutation struct {
	Action string  `json:"action"`
	Name   *string `json:"name"`
}

func validTerminalSessionMutation(action terminalSessionMutation) bool {
	switch action.Action {
	case "end":
		return action.Name == nil
	case "rename":
		return action.Name != nil && validTerminalName(*action.Name)
	default:
		return false
	}
}

func parseTerminalSessionAction(w http.ResponseWriter, r *http.Request, in *host.TerminalRequest) bool {
	if r.Method != "POST" {
		return true
	}
	var action terminalSessionMutation
	if !webauth.DecodeAPIObject(w, r, &action) {
		return false
	}
	if !validTerminalSessionMutation(action) {
		webauth.JSONError(w, 400, "invalid_action", "Choose End or Rename for this exact terminal.")
		return false
	}
	in.Action = action.Action
	if action.Name != nil {
		in.Name = *action.Name
	}
	return true
}

func (s *API) apiTerminalSession(w http.ResponseWriter, r *http.Request, v store.Session) {
	id := r.PathValue("terminalID")
	if !BrowserTerminalID.MatchString(id) {
		webauth.JSONError(w, 400, "invalid_request", "Provide an exact terminal ID.")
		return
	}
	in := host.TerminalRequest{Action: "inspect", ID: id}
	if !parseTerminalSessionAction(w, r, &in) {
		return
	}
	p, login, token, ok := s.terminalAccount(w, r, v)
	if !ok {
		return
	}
	items, err := s.terminalOperation(r, v, p, login, token, in)
	if err != nil {
		webauth.JSONError(w, 503, "terminal_unavailable", "The exact native outcome is unavailable. Nothing was retried or replaced.")
		return
	}
	if len(items) == 0 {
		terminalMetadata(w, nil)
		return
	}
	view := terminalDTO(items[0], v, p, login)
	terminalMetadata(w, &view)
}
