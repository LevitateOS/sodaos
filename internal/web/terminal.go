package web

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/coder/websocket"
	"github.com/levitateos/sodaos/internal/host"
	"github.com/levitateos/sodaos/internal/store"
	"github.com/levitateos/sodaos/internal/strictjson"
)

func validTerminalOrigin(r *http.Request, forgejoURL string) bool {
	origins := r.Header.Values("Origin")
	if r.URL.RawQuery != "" || r.URL.ForceQuery || len(origins) != 1 {
		return false
	}
	return strings.HasPrefix(forgejoURL, "https://") && origins[0] == forgejoURL && len(r.Header.Values("Sec-WebSocket-Protocol")) == 0
}

func validSecFetchHeaders(r *http.Request) bool {
	for name, expected := range map[string]string{
		"Sec-Fetch-Site": "same-origin",
		"Sec-Fetch-Mode": "websocket",
		"Sec-Fetch-Dest": "empty",
	} {
		values := r.Header.Values(name)
		if len(values) > 1 || (len(values) == 1 && values[0] != expected) {
			return false
		}
	}
	return true
}

func checkTerminalRequestHeaders(w http.ResponseWriter, r *http.Request, forgejoURL string) bool {
	w.Header().Set("Cache-Control", "no-store")
	if r.Method != "GET" {
		w.Header().Set("Allow", "GET")
		jsonError(w, 405, "method_not_allowed", "WebSocket GET required.")
		return false
	}
	if !validTerminalOrigin(r, forgejoURL) || !validSecFetchHeaders(r) {
		jsonError(w, 403, "invalid_origin", "Same-origin terminal required.")
		return false
	}
	return true
}

type terminalSessionAuth struct {
	cookie  string
	session store.Session
	project store.Project
	login   string
}

func validMemberLogin(p store.Project, login string, err error) bool {
	return err == nil && p.Ready && login != "root" && projectLogin.MatchString(login)
}

func (s *Server) authenticateTerminalSession(w http.ResponseWriter, r *http.Request) (terminalSessionAuth, bool) {
	cookie, err := requestCookie(r, sessionCookie)
	if err != nil || cookie.Value == "" || s.Store == nil {
		jsonError(w, 401, "unauthenticated", "Sign in through Forgejo.")
		return terminalSessionAuth{}, false
	}
	v, err := s.Store.Session(r.Context(), cookie.Value)
	if err != nil {
		jsonError(w, 401, "unauthenticated", "Sign in through Forgejo.")
		return terminalSessionAuth{}, false
	}
	p, ok := s.loadEnvironment(w, r)
	if !ok {
		return terminalSessionAuth{}, false
	}
	login, err := s.Store.MemberLogin(r.Context(), p.ID, v.User.ID)
	if !validMemberLogin(p, login, err) {
		jsonError(w, 403, "membership_required", "An existing provisioned account is required.")
		return terminalSessionAuth{}, false
	}
	return terminalSessionAuth{cookie: cookie.Value, session: v, project: p, login: login}, true
}

func (s *Server) registerTerminalPeer(r *http.Request, auth terminalSessionAuth, cancel context.CancelFunc) (*terminalPeer, bool) {
	s.terminalMu.Lock()
	defer s.terminalMu.Unlock()
	if s.terminalClosed || s.terminalStopping[auth.project.ID] || len(s.terminalPeers) >= 128 {
		return nil, false
	}
	if s.terminalPeers == nil {
		s.terminalPeers = make(map[*http.Request]*terminalPeer)
	}
	peer := &terminalPeer{
		token:     auth.cookie,
		contextID: auth.session.ContextID,
		project:   auth.project.ID,
		cancel:    cancel,
	}
	s.terminalPeers[r] = peer
	s.terminalWG.Add(1)
	return peer, true
}

func (s *Server) unregisterTerminalPeer(r *http.Request) {
	s.terminalMu.Lock()
	delete(s.terminalPeers, r)
	s.terminalMu.Unlock()
	s.terminalWG.Done()
}

type terminalHandshake struct {
	Action         string `json:"action"`
	ID             string `json:"id"`
	Name           string `json:"name"`
	ExpectedUserID string `json:"expected_user_id"`
	RepositoryID   string `json:"repository_id"`
	CSRF           string `json:"csrf_token"`
	Cols           int    `json:"cols"`
	Rows           int    `json:"rows"`
}

func readTerminalHandshake(ctx context.Context, conn *websocket.Conn) (terminalHandshake, error) {
	var in terminalHandshake
	first, done := context.WithTimeout(ctx, 5*time.Second)
	kind, body, err := conn.Read(first)
	done()
	if err != nil {
		return in, err
	}
	if kind != websocket.MessageText || len(body) > 4096 || strictjson.Decode(bytes.NewReader(body), &in) != nil {
		return in, errors.New("invalid handshake")
	}
	return in, nil
}

func validHandshakeDimensions(cols, rows int) bool {
	return cols >= 2 && cols <= 500 && rows >= 2 && rows <= 300
}

func validHandshakeAction(id, action, name string) bool {
	if !browserTerminalID.MatchString(id) {
		return false
	}
	return (action == "create" && validTerminalName(name)) || (action == "attach" && name == "")
}

func validHandshakeActorAndRepo(in terminalHandshake, userID, repoID int64) bool {
	actor, validActor := positiveID(in.ExpectedUserID)
	repo, validRepo := positiveID(in.RepositoryID)
	return validActor && actor == userID && validRepo && repo == repoID
}

func (s *Server) validateTerminalHandshake(ctx context.Context, r *http.Request, auth terminalSessionAuth, in terminalHandshake) (*http.Request, bool) {
	if !validHandshakeDimensions(in.Cols, in.Rows) || !validHandshakeAction(in.ID, in.Action, in.Name) || !validHandshakeActorAndRepo(in, auth.session.User.ID, auth.project.RepositoryID) {
		return nil, false
	}
	authorized := r.Clone(ctx)
	authorized.Header.Set("X-CSRF-Token", in.CSRF)
	if !s.validAPIMutation(authorized, auth.session.CSRF) {
		return nil, false
	}
	check, checked := context.WithTimeout(ctx, 15*time.Second)
	defer checked()
	if _, err := s.visibleRepository(authorized.WithContext(check), auth.session, auth.project.RepositoryID); err != nil {
		return nil, false
	}
	return authorized, true
}

func peerIDInUse(peers map[*http.Request]*terminalPeer, current *terminalPeer, projectID, id string) bool {
	for _, other := range peers {
		if other != current && other.project == projectID && other.id == id {
			return true
		}
	}
	return false
}

func (s *Server) claimTerminalPeer(ctx context.Context, peer *terminalPeer, auth terminalSessionAuth, terminalID string) bool {
	s.terminalMu.Lock()
	defer s.terminalMu.Unlock()
	if s.terminalClosed || s.terminalStopping[auth.project.ID] || ctx.Err() != nil || !s.terminalCurrent(ctx, auth.cookie, auth.session, auth.project, auth.login) {
		return false
	}
	if peerIDInUse(s.terminalPeers, peer, auth.project.ID, terminalID) {
		return false
	}
	peer.id = terminalID
	return true
}

func (s *Server) prepareTerminalPeer(ctx context.Context, r *http.Request, auth terminalSessionAuth, in terminalHandshake, peer *terminalPeer) (*http.Request, bool) {
	authorized, ok := s.validateTerminalHandshake(ctx, r, auth, in)
	if !ok || !s.claimTerminalPeer(ctx, peer, auth, in.ID) {
		return nil, false
	}
	return authorized, true
}

func (s *Server) createHostTerminal(ctx context.Context, auth terminalSessionAuth, in terminalHandshake) bool {
	items, err := s.Host.TerminalStates(ctx, host.TerminalRequest{
		Action:   "create",
		ID:       in.ID,
		Project:  auth.project.ID,
		Login:    auth.login,
		Identity: auth.session.User.ID,
		Cols:     in.Cols,
		Rows:     in.Rows,
		Name:     in.Name,
		Scope:    terminalCreationScope(auth.session),
	})
	return err == nil && len(items) == 1 && items[0].Ready
}

func (s *Server) isTerminalLive(ctx context.Context, auth terminalSessionAuth) bool {
	s.terminalMu.Lock()
	defer s.terminalMu.Unlock()
	return ctx.Err() == nil && !s.terminalClosed && !s.terminalStopping[auth.project.ID] && s.terminalCurrent(ctx, auth.cookie, auth.session, auth.project, auth.login)
}

func (s *Server) openHostTerminal(ctx context.Context, auth terminalSessionAuth, in terminalHandshake) (*host.Terminal, error) {
	if in.Action == "create" && !s.createHostTerminal(ctx, auth, in) {
		return nil, errors.New("terminal create failed")
	}
	if !s.isTerminalLive(ctx, auth) {
		return nil, errors.New("terminal no longer live")
	}
	expires := min(auth.session.Expires, time.Now().Add(12*time.Hour).Unix())
	return s.Host.OpenTerminal(ctx, host.TerminalRequest{
		Action:   "attach",
		ID:       in.ID,
		Project:  auth.project.ID,
		Login:    auth.login,
		Identity: auth.session.User.ID,
		Cols:     in.Cols,
		Rows:     in.Rows,
		Expires:  expires,
	})
}

func refuseTerminal(conn *websocket.Conn) {
	_ = conn.Close(websocket.StatusPolicyViolation, "terminal unavailable or authorization failed; use an exact ID or reload this page")
}

func pumpBrowserControls(ctx context.Context, cancel context.CancelFunc, conn *websocket.Conn, controls chan<- host.TerminalFrame) {
	defer cancel()
	for {
		kind, body, err := conn.Read(ctx)
		if err != nil {
			return
		}
		var frame host.TerminalFrame
		if kind != websocket.MessageText || strictjson.Decode(bytes.NewReader(body), &frame) != nil || (frame.Type != "input" && frame.Type != "resize") {
			return
		}
		select {
		case controls <- frame:
		case <-ctx.Done():
			return
		}
	}
}

func (s *Server) checkTerminalHeartbeat(ctx context.Context, conn *websocket.Conn, native *host.Terminal, authorized *http.Request, auth terminalSessionAuth) bool {
	check, done := context.WithTimeout(ctx, 5*time.Second)
	defer done()
	_, authorityErr := s.visibleRepository(authorized.WithContext(check), auth.session, auth.project.RepositoryID)
	live := authorityErr == nil && conn.Ping(check) == nil && s.terminalCurrent(check, auth.cookie, auth.session, auth.project, auth.login)
	if !live {
		return false
	}
	return native.Send(check, host.TerminalFrame{Type: "heartbeat"}) == nil
}

func (s *Server) pumpNativeControls(ctx context.Context, cancel context.CancelFunc, conn *websocket.Conn, native *host.Terminal, controls <-chan host.TerminalFrame, authorized *http.Request, auth terminalSessionAuth) {
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
			if !s.checkTerminalHeartbeat(ctx, conn, native, authorized, auth) {
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

func pumpNativeToBrowser(ctx context.Context, conn *websocket.Conn, native *host.Terminal) {
	for {
		frame, err := native.Receive(ctx)
		if err != nil || frame.Type == "metadata" {
			return
		}
		body, _ := json.Marshal(frame)
		write, done := context.WithTimeout(ctx, 5*time.Second)
		err = conn.Write(write, websocket.MessageText, body)
		done()
		if err != nil || frame.Type == "closed" {
			return
		}
	}
}

func (s *Server) apiTerminal(w http.ResponseWriter, r *http.Request) {
	if !checkTerminalRequestHeaders(w, r, s.Config.ForgejoURL) {
		return
	}
	auth, ok := s.authenticateTerminalSession(w, r)
	if !ok {
		return
	}
	ctx, cancel := context.WithDeadline(r.Context(), time.Unix(auth.session.Expires, 0))
	defer cancel()

	peer, ok := s.registerTerminalPeer(r, auth, cancel)
	if !ok {
		jsonError(w, 409, "terminal_unavailable", "Terminal transport unavailable.")
		return
	}
	defer s.unregisterTerminalPeer(r)

	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		return
	}
	defer conn.CloseNow()
	conn.SetReadLimit(32768)
	closePeer := context.AfterFunc(ctx, func() { conn.CloseNow() })
	defer closePeer()

	in, err := readTerminalHandshake(ctx, conn)
	if err != nil {
		refuseTerminal(conn)
		return
	}
	authorized, ok := s.prepareTerminalPeer(ctx, r, auth, in, peer)
	if !ok {
		refuseTerminal(conn)
		return
	}
	native, err := s.openHostTerminal(ctx, auth, in)
	if err != nil {
		refuseTerminal(conn)
		return
	}
	defer native.Close()
	stopNative := context.AfterFunc(ctx, native.Close)
	defer stopNative()

	controls := make(chan host.TerminalFrame, 1)
	go pumpBrowserControls(ctx, cancel, conn, controls)
	go s.pumpNativeControls(ctx, cancel, conn, native, controls, authorized, auth)
	pumpNativeToBrowser(ctx, conn, native)
}
