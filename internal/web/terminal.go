package web

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/coder/websocket"
	"github.com/levitateos/sodaos/internal/host"
	"github.com/levitateos/sodaos/internal/strictjson"
)

func (s *Server) apiTerminal(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if r.Method != "GET" {
		w.Header().Set("Allow", "GET")
		jsonError(w, 405, "method_not_allowed", "WebSocket GET required.")
		return
	}
	origins := r.Header.Values("Origin")
	if r.URL.RawQuery != "" || r.URL.ForceQuery || len(origins) != 1 || !strings.HasPrefix(s.Config.ForgejoURL, "https://") || origins[0] != s.Config.ForgejoURL || len(r.Header.Values("Sec-WebSocket-Protocol")) != 0 {
		jsonError(w, 403, "invalid_origin", "Same-origin terminal required.")
		return
	}
	for name, expected := range map[string]string{"Sec-Fetch-Site": "same-origin", "Sec-Fetch-Mode": "websocket", "Sec-Fetch-Dest": "empty"} {
		values := r.Header.Values(name)
		if len(values) > 1 || (len(values) == 1 && values[0] != expected) {
			jsonError(w, 403, "invalid_origin", "Same-origin terminal required.")
			return
		}
	}
	cookie, err := requestCookie(r, sessionCookie)
	if err != nil || cookie.Value == "" || s.Store == nil {
		jsonError(w, 401, "unauthenticated", "Sign in through Forgejo.")
		return
	}
	v, err := s.Store.Session(r.Context(), cookie.Value)
	if err != nil {
		jsonError(w, 401, "unauthenticated", "Sign in through Forgejo.")
		return
	}
	p, ok := s.loadEnvironment(w, r)
	if !ok {
		return
	}
	login, err := s.Store.MemberLogin(r.Context(), p.ID, v.User.ID)
	if err != nil || !p.Ready || login == "root" || !projectLogin.MatchString(login) {
		jsonError(w, 403, "membership_required", "An existing provisioned account is required.")
		return
	}
	ctx, cancel := context.WithDeadline(r.Context(), time.Unix(v.Expires, 0))
	defer cancel()
	s.terminalMu.Lock()
	if s.terminalClosed || s.terminalStopping[p.ID] || len(s.terminalPeers) >= 128 {
		s.terminalMu.Unlock()
		jsonError(w, 409, "terminal_unavailable", "Terminal transport unavailable.")
		return
	}
	if s.terminalPeers == nil {
		s.terminalPeers = make(map[*http.Request]*terminalPeer)
	}
	peer := &terminalPeer{token: cookie.Value, contextID: v.ContextID, project: p.ID, cancel: cancel}
	s.terminalPeers[r] = peer
	s.terminalWG.Add(1)
	s.terminalMu.Unlock()
	defer s.terminalWG.Done()
	defer func() { s.terminalMu.Lock(); delete(s.terminalPeers, r); s.terminalMu.Unlock() }()
	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		return
	}
	defer conn.CloseNow()
	conn.SetReadLimit(32768)
	closePeer := context.AfterFunc(ctx, func() { conn.CloseNow() })
	defer closePeer()
	refuse := func() {
		_ = conn.Close(websocket.StatusPolicyViolation, "terminal unavailable or authorization failed; use an exact ID or reload this page")
	}
	first, done := context.WithTimeout(ctx, 5*time.Second)
	kind, body, err := conn.Read(first)
	done()
	var in struct {
		Action         string `json:"action"`
		ID             string `json:"id"`
		Name           string `json:"name"`
		ExpectedUserID string `json:"expected_user_id"`
		RepositoryID   string `json:"repository_id"`
		CSRF           string `json:"csrf_token"`
		Cols           int    `json:"cols"`
		Rows           int    `json:"rows"`
	}
	if err != nil || kind != websocket.MessageText || len(body) > 4096 || strictjson.Decode(bytes.NewReader(body), &in) != nil {
		refuse()
		return
	}
	actor, validActor := positiveID(in.ExpectedUserID)
	repo, validRepo := positiveID(in.RepositoryID)
	authorized := r.Clone(ctx)
	authorized.Header.Set("X-CSRF-Token", in.CSRF)
	validAction := browserTerminalID.MatchString(in.ID) && ((in.Action == "create" && validTerminalName(in.Name)) || (in.Action == "attach" && in.Name == ""))
	if !validActor || actor != v.User.ID || !validRepo || repo != p.RepositoryID || !s.validAPIMutation(authorized, v.CSRF) || in.Cols < 2 || in.Cols > 500 || in.Rows < 2 || in.Rows > 300 || !validAction {
		refuse()
		return
	}
	check, checked := context.WithTimeout(ctx, 15*time.Second)
	_, err = s.visibleRepository(authorized.WithContext(check), v, p.RepositoryID)
	checked()
	if err != nil {
		refuse()
		return
	}
	s.terminalMu.Lock()
	if s.terminalClosed || s.terminalStopping[p.ID] || ctx.Err() != nil || !s.terminalCurrent(ctx, cookie.Value, v, p, login) {
		s.terminalMu.Unlock()
		refuse()
		return
	}
	for _, other := range s.terminalPeers {
		if other != peer && other.project == p.ID && other.id == in.ID {
			s.terminalMu.Unlock()
			refuse()
			return
		}
	}
	peer.id = in.ID // bounded in-flight/writer exclusion, not retained shell custody
	s.terminalMu.Unlock()
	if in.Action == "create" {
		items, err := s.Host.TerminalStates(ctx, host.TerminalRequest{Action: "create", ID: in.ID, Project: p.ID, Login: login, Identity: v.User.ID, Cols: in.Cols, Rows: in.Rows, Name: in.Name, Scope: terminalCreationScope(v)})
		if err != nil || len(items) != 1 || !items[0].Ready {
			refuse()
			return
		}
	}
	s.terminalMu.Lock()
	live := ctx.Err() == nil && !s.terminalClosed && !s.terminalStopping[p.ID] && s.terminalCurrent(ctx, cookie.Value, v, p, login)
	s.terminalMu.Unlock()
	if !live {
		refuse()
		return
	}
	native, err := s.Host.OpenTerminal(ctx, host.TerminalRequest{Action: "attach", ID: in.ID, Project: p.ID, Login: login, Identity: v.User.ID, Cols: in.Cols, Rows: in.Rows, Expires: min(v.Expires, time.Now().Add(12*time.Hour).Unix())})
	if err != nil {
		refuse()
		return
	}
	defer native.Close()
	stopNative := context.AfterFunc(ctx, native.Close)
	defer stopNative()
	controls := make(chan host.TerminalFrame, 1)
	go func() {
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
	}()
	go func() {
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
				check, done := context.WithTimeout(ctx, 5*time.Second)
				_, authorityErr := s.visibleRepository(authorized.WithContext(check), v, p.RepositoryID)
				live := authorityErr == nil && conn.Ping(check) == nil && s.terminalCurrent(check, cookie.Value, v, p, login)
				var err error
				if live {
					err = native.Send(check, host.TerminalFrame{Type: "heartbeat"})
				}
				done()
				if !live || err != nil {
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()
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
