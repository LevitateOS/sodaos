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
	s.terminalPeers[r] = &terminalPeer{cookie.Value, v.ContextID, p.ID, cancel}
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
		RequestID      string `json:"request_id"`
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
	validAction := (in.Action == "create" && in.ID == "" && browserTerminalID.MatchString(in.RequestID) && validTerminalName(in.Name)) || (in.Action == "attach" && browserTerminalID.MatchString(in.ID) && in.RequestID == "" && in.Name == "")
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
	attachment := &terminalAttachment{id: newTerminalID()}
	s.terminalMu.Lock()
	entry := s.terminals[in.ID]
	if s.terminalClosed || s.terminalStopping[p.ID] || ctx.Err() != nil || !s.terminalCurrent(ctx, cookie.Value, v, p, login) {
		s.terminalMu.Unlock()
		refuse()
		return
	}
	if in.Action == "create" {
		s.expireTerminalReceipts()
		duplicate := false
		for _, e := range s.terminals {
			duplicate = duplicate || (e.session.ContextID == v.ContextID && e.requestID == in.RequestID)
		}
		for _, e := range s.terminalReceipts {
			duplicate = duplicate || (e.binding.session.ContextID == v.ContextID && e.view.RequestID == in.RequestID)
		}
		if duplicate || len(s.terminals) >= 64 {
			s.terminalMu.Unlock()
			refuse()
			return
		}
		id := newTerminalID()
		_, receiptExists := s.terminalReceipts[id]
		if s.terminals[id] != nil || receiptExists {
			s.terminalMu.Unlock()
			refuse()
			return
		}
		now := time.Now()
		hard := minTime(time.Unix(v.Expires, 0), now.Add(12*time.Hour))
		ownerCtx, ownerCancel := context.WithDeadline(context.Background(), hard)
		entry = &browserTerminal{terminalBinding: terminalBinding{cookie.Value, v, p, login, now, hard}, cancel: ownerCancel, ctx: ownerCtx, id: id, requestID: in.RequestID, name: in.Name, ready: make(chan struct{}), attachment: attachment, lastAttachment: attachment.id}
		if s.terminals == nil {
			s.terminals = make(map[string]*browserTerminal)
		}
		s.terminals[entry.id] = entry
		// Reservation/admission is atomic with Stop/logout. Native IO is cancellable
		// outside the global lock. Even an uncertain dispatch retains its exact slot.
		s.terminalWG.Add(1)
		go s.ownBrowserTerminal(authorized.Clone(ownerCtx), entry, in.Cols, in.Rows)
	} else {
		if entry == nil || !entry.matches(v, p, login, cookie.Value) || !entry.live() || entry.attachment != nil {
			s.terminalMu.Unlock()
			refuse()
			return
		}
		entry.attachment = attachment
		entry.lastAttachment = attachment.id
	}
	s.terminalMu.Unlock()
	endAttachment := context.AfterFunc(entry.ctx, cancel)
	defer endAttachment()
	defer func() {
		s.terminalMu.Lock()
		if entry.attachment == attachment {
			entry.attachment = nil
			if entry.retainUntil.IsZero() {
				entry.retain(1800)
			}
		}
		s.terminalMu.Unlock()
	}()
	message, _ := json.Marshal(map[string]string{"type": "session", "id": entry.id, "request_id": entry.requestID, "attachment_id": attachment.id})
	write, finish := context.WithTimeout(ctx, 5*time.Second)
	err = conn.Write(write, websocket.MessageText, message)
	finish()
	if err != nil {
		return
	}
	select {
	case <-entry.ready:
	case <-ctx.Done():
		return
	}
	s.terminalMu.Lock()
	live := s.terminals[entry.id] == entry && entry.started && entry.live() && entry.attachment == attachment && ctx.Err() == nil && s.terminalCurrent(ctx, cookie.Value, v, p, login)
	s.terminalMu.Unlock()
	if !live {
		refuse()
		return
	}
	native, err := s.Host.OpenTerminal(ctx, host.TerminalRequest{Action: "attach", ID: entry.id, Project: p.ID, Login: login, Identity: v.User.ID, Cols: in.Cols, Rows: in.Rows, Expires: entry.hardUntil.Unix()})
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
				s.terminalMu.Lock()
				live := entry.live() && entry.attachment == attachment && ctx.Err() == nil
				s.terminalMu.Unlock()
				if !live || native.Send(ctx, frame) != nil {
					return
				}
			case <-tick.C:
				check, done := context.WithTimeout(ctx, 5*time.Second)
				live := conn.Ping(check) == nil && s.terminalCurrent(check, cookie.Value, v, p, login)
				s.terminalMu.Lock()
				live = live && entry.live() && entry.attachment == attachment
				s.terminalMu.Unlock()
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
		if err != nil {
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
