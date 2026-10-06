package api

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/coder/websocket"
	"github.com/levitateos/sodaos/internal/host"
	"github.com/levitateos/sodaos/internal/web/auth"
)

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
