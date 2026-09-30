package api

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	extensions "forgejo.org/extension-sdk"
	"github.com/coder/websocket"
	"github.com/levitateos/sodaos/internal/host"
	"github.com/levitateos/sodaos/internal/project"
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
	s.registerExtensionProductRoutes(mux)
	mux.Handle("/", s.Auth.ExtensionHandler())
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawPath != "" || r.URL.ForceQuery ||
			strings.Contains(r.URL.Path, "\\") || path.Clean(r.URL.Path) != r.URL.Path {
			http.NotFound(w, r)
			return
		}
		if !privateTerminalStreamRoute(r) {
			r.Body = http.MaxBytesReader(w, r.Body, auth.APIBodyLimit)
		}
		mux.ServeHTTP(w, r)
	})
}

func privateTerminalStreamRoute(r *http.Request) bool {
	parts := strings.Split(r.URL.Path, "/")
	return r.Method == http.MethodGet && len(parts) == 5 && parts[1] == "api" && parts[2] == "environments" && parts[3] != "" && parts[4] == "terminal"
}

type extensionTerminalIdentity struct {
	authority extensions.Authority
	project   store.Project
	login     string
	actorID   int64
}

func nativeTerminalScope(generation string) string {
	sum := sha256.Sum256([]byte("native-terminal\x00" + generation))
	return hex.EncodeToString(sum[:])
}

func extensionTerminalGeneration(r *http.Request, authority extensions.Authority) bool {
	values := r.Header.Values(extensions.SessionGenerationHeader)
	return len(values) == 1 && values[0] != "" && values[0] == authority.SessionGeneration
}

func extensionTerminalOrigin(r *http.Request, origin string) bool {
	values := r.Header.Values("Origin")
	fetch := r.Header.Values("Sec-Fetch-Site")
	return len(values) == 1 && values[0] == origin && strings.HasPrefix(origin, "https://") &&
		len(fetch) <= 1 && (len(fetch) == 0 || fetch[0] == "same-origin")
}

func nativeTerminalContribution(authority extensions.Authority) bool {
	return extensionPageContribution(authority.Contribution, "spaces") || extensionWorkspacePanelContribution(authority.Contribution)
}

func nativeTerminalMember(p store.Project, login string, err error) bool {
	return err == nil && p.Ready && login != "root" && project.ValidLogin(login)
}

func (s *API) nativeTerminalActor(authority extensions.Authority) (int64, bool) {
	id, ok := auth.PositiveID(authority.Actor.ID)
	return id, ok && s.Store != nil
}

func (s *API) extensionTerminalAccount(w http.ResponseWriter, r *http.Request, generationHeader bool) (extensionTerminalIdentity, bool) {
	var identity extensionTerminalIdentity
	authority, err := auth.ExtensionAuthority(r)
	if err != nil || !nativeTerminalContribution(authority) {
		auth.JSONError(w, 403, "native_authority_unavailable", "Current native terminal authority is required.")
		return identity, false
	}
	if generationHeader && !extensionTerminalGeneration(r, authority) {
		auth.JSONError(w, 403, "session_changed", "Reload the current native session.")
		return identity, false
	}
	actorID, ok := s.nativeTerminalActor(authority)
	if !ok {
		auth.JSONError(w, 403, "invalid_actor", "Native actor is invalid.")
		return identity, false
	}
	p, err := s.Store.Project(r.Context(), r.PathValue("id"))
	if err != nil {
		auth.JSONError(w, 404, "not_found", "Environment not found.")
		return identity, false
	}
	login, err := s.Store.MemberLogin(r.Context(), p.ID, actorID)
	if !nativeTerminalMember(p, login, err) {
		auth.JSONError(w, 403, "membership_required", "An existing provisioned account is required.")
		return identity, false
	}
	identity = extensionTerminalIdentity{authority: authority, project: p, login: login, actorID: actorID}
	if !s.extensionTerminalRepository(r.Context(), identity) {
		auth.JSONError(w, 403, "repository_write_required", "Current repository write permission is required.")
		return extensionTerminalIdentity{}, false
	}
	return identity, true
}

func sameNativeTerminalAuthority(current, original extensions.Authority) bool {
	return current.Actor == original.Actor && current.SessionGeneration == original.SessionGeneration &&
		current.InstanceID == original.InstanceID && current.Contribution == original.Contribution
}

func nativeTerminalMembershipCurrent(ctx context.Context, login, expected string, err error) bool {
	return ctx.Err() == nil && err == nil && login == expected
}

func (s *API) extensionTerminalRepository(ctx context.Context, identity extensionTerminalIdentity) bool {
	check, done := context.WithTimeout(ctx, 5*time.Second)
	defer done()
	repo, err := identity.authority.Native().Repository(check, strconv.FormatInt(identity.project.RepositoryID, 10))
	return check.Err() == nil && err == nil && repo.ID == strconv.FormatInt(identity.project.RepositoryID, 10) &&
		(repo.Permission == "write" || repo.Permission == "admin")
}

func (s *API) extensionTerminalCurrent(ctx context.Context, r *http.Request, identity extensionTerminalIdentity) bool {
	if ctx.Err() != nil {
		return false
	}
	check, done := context.WithTimeout(ctx, 5*time.Second)
	defer done()
	current, err := auth.ExtensionAuthority(r.WithContext(check))
	if err != nil || !sameNativeTerminalAuthority(current, identity.authority) {
		return false
	}
	p, err := s.Store.Project(check, identity.project.ID)
	if err != nil || !p.Ready || p.RepositoryID != identity.project.RepositoryID {
		return false
	}
	login, err := s.Store.MemberLogin(check, p.ID, identity.actorID)
	return nativeTerminalMembershipCurrent(check, login, identity.login, err) && s.extensionTerminalRepository(check, identity)
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

func nativeTerminalSessionActor(authority extensions.Authority, present, valid bool, v store.Session) bool {
	return present && valid && authority.Actor.ID == strconv.FormatInt(v.User.ID, 10) && authority.Actor.Username == v.User.Login
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

func (s *API) extensionTerminalAttach(w http.ResponseWriter, r *http.Request, identity extensionTerminalIdentity) {
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	peer, ok := s.registerExtensionTerminalPeer(r, identity, cancel)
	if !ok {
		auth.JSONError(w, 409, "terminal_unavailable", "Terminal transport unavailable.")
		return
	}
	defer s.unregisterTerminalPeer(r)
	// The private proxy uses an internal Host; exact public Origin was checked
	// before this upgrade, so the library's Host comparison does not apply here.
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
	if err != nil {
		return
	}
	defer conn.CloseNow()
	conn.SetReadLimit(32768)
	closePeer := context.AfterFunc(ctx, func() { conn.CloseNow() })
	defer closePeer()
	in, err := readTerminalHandshake(ctx, conn)
	if err != nil || !validNativeTerminalHandshake(in, identity) {
		refuseTerminal(conn)
		return
	}
	if !s.extensionClaimTerminal(ctx, r, identity, peer, in.ID) {
		refuseTerminal(conn)
		return
	}
	native, err := s.extensionOpenHostTerminal(ctx, r, identity, in)
	if err != nil {
		refuseTerminal(conn)
		return
	}
	defer native.Close()
	stopNative := context.AfterFunc(ctx, native.Close)
	defer stopNative()
	controls := make(chan host.TerminalFrame, 1)
	go pumpTerminalInput(ctx, cancel, conn, controls)
	go s.pumpExtensionControls(ctx, cancel, conn, native, controls, r, identity)
	pumpNativeToExtension(ctx, conn, native)
}

func (s *API) registerExtensionTerminalPeer(r *http.Request, identity extensionTerminalIdentity, cancel context.CancelFunc) (*TerminalPeer, bool) {
	s.terminalMu.Lock()
	defer s.terminalMu.Unlock()
	if s.terminalClosed || s.TerminalStopping[identity.project.ID] || len(s.TerminalPeers) >= 128 {
		return nil, false
	}
	if s.TerminalPeers == nil {
		s.TerminalPeers = make(map[*http.Request]*TerminalPeer)
	}
	peer := &TerminalPeer{ContextID: identity.authority.SessionGeneration, Project: identity.project.ID, Cancel: cancel}
	s.TerminalPeers[r] = peer
	s.terminalWG.Add(1)
	return peer, true
}

func validNativeTerminalHandshake(in terminalHandshake, identity extensionTerminalIdentity) bool {
	return validHandshakeDimensions(in.Cols, in.Rows) && validHandshakeAction(in.ID, in.Action, in.Name) &&
		validHandshakeRepository(in, identity.project.RepositoryID) && in.SessionGeneration == identity.authority.SessionGeneration
}

func (s *API) extensionClaimTerminal(ctx context.Context, r *http.Request, identity extensionTerminalIdentity, peer *TerminalPeer, id string) bool {
	if !s.extensionTerminalCurrent(ctx, r, identity) {
		return false
	}
	s.terminalMu.Lock()
	defer s.terminalMu.Unlock()
	if s.terminalClosed || s.TerminalStopping[identity.project.ID] || ctx.Err() != nil || peerIDInUse(s.TerminalPeers, peer, identity.project.ID, id) {
		return false
	}
	peer.ID = id
	return true
}

func (s *API) extensionOpenHostTerminal(ctx context.Context, r *http.Request, identity extensionTerminalIdentity, in terminalHandshake) (*host.Terminal, error) {
	if !s.extensionTerminalCurrent(ctx, r, identity) {
		return nil, errors.New("terminal authority ended")
	}
	if in.Action == "create" {
		items, err := s.Host.TerminalStates(ctx, host.TerminalRequest{Action: "create", ID: in.ID, Project: identity.project.ID, Login: identity.login, Identity: identity.actorID, Cols: in.Cols, Rows: in.Rows, Name: in.Name, Scope: nativeTerminalScope(identity.authority.SessionGeneration)})
		if err != nil || len(items) != 1 || !items[0].Ready {
			return nil, errors.New("terminal create failed")
		}
	}
	if !s.extensionTerminalCurrent(ctx, r, identity) {
		return nil, errors.New("terminal authority ended")
	}
	return s.Host.OpenTerminal(ctx, host.TerminalRequest{Action: "attach", ID: in.ID, Project: identity.project.ID, Login: identity.login, Identity: identity.actorID, Cols: in.Cols, Rows: in.Rows, Expires: time.Now().Add(12 * time.Hour).Unix()})
}

func (s *API) pumpExtensionControls(ctx context.Context, cancel context.CancelFunc, conn *websocket.Conn, native *host.Terminal, controls <-chan host.TerminalFrame, r *http.Request, identity extensionTerminalIdentity) {
	defer cancel()
	tick := time.NewTicker(15 * time.Second)
	defer tick.Stop()
	for {
		select {
		case frame := <-controls:
			if ctx.Err() != nil || native.Send(ctx, frame) != nil {
				return
			}
		case <-tick.C:
			if !s.extensionTerminalHeartbeat(ctx, conn, native, r, identity) {
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

func (s *API) extensionTerminalHeartbeat(ctx context.Context, conn *websocket.Conn, native *host.Terminal, r *http.Request, identity extensionTerminalIdentity) bool {
	check, done := context.WithTimeout(ctx, 5*time.Second)
	defer done()
	return s.extensionTerminalCurrent(check, r, identity) && conn.Ping(check) == nil && native.Send(check, host.TerminalFrame{Type: "heartbeat"}) == nil
}
