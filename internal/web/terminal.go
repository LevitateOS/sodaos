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
	"github.com/levitateos/sodaos/internal/store"
	"github.com/levitateos/sodaos/internal/strictjson"
)

type terminalKey struct{ context, project string }
type browserTerminal struct {
	token  string
	cancel context.CancelFunc
}

// Called with terminalMu held. Logout/rotation and final launch authorization
// share this lock; no durable registry, copied provider permissions or replay.
func (s *Server) cancelTerminals(contextID, token string) {
	for key, entry := range s.terminals {
		if (contextID != "" && key.context == contextID) || (token != "" && entry.token == token) {
			entry.cancel()
		}
	}
}
func (s *Server) CloseTerminals() {
	s.terminalMu.Lock()
	s.terminalClosed = true
	for _, entry := range s.terminals {
		entry.cancel()
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
	until := time.Now().Add(2 * time.Hour)
	if expiry := time.Unix(v.Expires, 0); expiry.Before(until) {
		until = expiry
	}
	ctx, cancel := context.WithDeadline(r.Context(), until)
	defer cancel()
	key := terminalKey{v.ContextID, p.ID}
	entry := &browserTerminal{cookie.Value, cancel}
	s.terminalMu.Lock()
	if s.terminalClosed || len(s.terminals) >= 64 || s.terminals[key] != nil || !s.terminalCurrent(ctx, cookie.Value, v, p, login) {
		s.terminalMu.Unlock()
		jsonError(w, 409, "terminal_unavailable", "Terminal unavailable or already open for this login and project.")
		return
	}
	if s.terminals == nil {
		s.terminals = make(map[terminalKey]*browserTerminal)
	}
	s.terminals[key] = entry
	s.terminalWG.Add(1)
	s.terminalMu.Unlock()
	defer s.terminalWG.Done()
	defer func() { s.terminalMu.Lock(); delete(s.terminals, key); s.terminalMu.Unlock() }()
	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		return
	}
	defer conn.CloseNow()
	conn.SetReadLimit(32768)
	// Cancellation also closes hijacked connections while no Read/Write is active.
	stopClose := context.AfterFunc(ctx, func() { conn.CloseNow() })
	defer stopClose()
	refuse := func() { _ = conn.Close(websocket.StatusPolicyViolation, "terminal authorization failed") }
	first, done := context.WithTimeout(ctx, 5*time.Second)
	kind, body, err := conn.Read(first)
	done()
	var in struct {
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
	if !validActor || actor != v.User.ID || !validRepo || repo != p.RepositoryID || !s.validAPIMutation(authorized, v.CSRF) || in.Cols < 2 || in.Cols > 500 || in.Rows < 2 || in.Rows > 300 {
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
	// Linearize native dispatch against successful local logout/rotation. Opening
	// the private transport is bounded (5s); no provider request runs under this lock.
	s.terminalMu.Lock()
	var native *host.Terminal
	if !s.terminalClosed && ctx.Err() == nil && s.terminalCurrent(ctx, cookie.Value, v, p, login) {
		native, err = s.Host.OpenTerminal(ctx, host.TerminalRequest{Project: p.ID, Login: login, Identity: v.User.ID, Cols: in.Cols, Rows: in.Rows, Expires: until.Unix()})
	}
	s.terminalMu.Unlock()
	if native == nil || err != nil {
		refuse()
		return
	}
	defer native.Close()
	stopNative := context.AfterFunc(ctx, native.Close)
	defer stopNative()
	// One serialized native writer, one bounded browser reader. A full queue
	// applies pressure rather than accumulating input. Heartbeats are server-only.
	controls := make(chan host.TerminalFrame, 1)
	go func() {
		defer cancel()
		for {
			kind, body, err := conn.Read(ctx)
			if err != nil {
				return
			}
			var frame host.TerminalFrame
			if kind != websocket.MessageText || strictjson.Decode(bytes.NewReader(body), &frame) != nil || (frame.Type != "input" && frame.Type != "resize" && frame.Type != "close") {
				return
			}
			select {
			case controls <- frame:
			case <-ctx.Done():
				return
			}
			if frame.Type == "close" {
				<-ctx.Done()
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
				if native.Send(ctx, frame) != nil {
					return
				}
				if frame.Type == "close" {
					<-ctx.Done()
					return
				}
			case <-tick.C:
				check, done := context.WithTimeout(ctx, 5*time.Second)
				peer := conn.Ping(check) == nil
				s.terminalMu.Lock()
				live := peer && ctx.Err() == nil && s.terminalCurrent(check, cookie.Value, v, p, login)
				var sendErr error
				if live {
					sendErr = native.Send(check, host.TerminalFrame{Type: "heartbeat"})
				}
				s.terminalMu.Unlock()
				done()
				if !live || sendErr != nil {
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
