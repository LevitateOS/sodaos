package web

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/coder/websocket"
	"github.com/levitateos/sodaos/internal/host"
	"github.com/levitateos/sodaos/internal/store"
	"github.com/levitateos/sodaos/internal/strictjson"
)

type terminalKey struct{ context, project string }
type terminalAttachment struct{ cancel context.CancelFunc }
type terminalPeer struct {
	token, contextID string
	cancel           context.CancelFunc
}
type browserTerminal struct {
	token       string
	cancel      context.CancelFunc
	ctx         context.Context
	id          string
	ready       chan struct{}
	started     bool
	unconfirmed bool
	attachment  *terminalAttachment
	// Zero means deliberately active. Attachment/output/pings never clear a deadline.
	retainUntil time.Time
}

var browserTerminalID = regexp.MustCompile(`^[0-9a-f]{32}$`)

// All registry fields are protected by terminalMu. Registry lifetime is the
// backend process, not a document/socket. Restart does not adopt native sessions;
// their independent safety leases expire if this owner disappears.
func (s *Server) cancelTerminals(contextID, token string) {
	for _, peer := range s.terminalPeers {
		if (contextID != "" && peer.contextID == contextID) || (token != "" && peer.token == token) {
			peer.cancel()
		}
	}
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

// This connection receives readiness/exit only, never terminal output. Browser
// cancellation stops lease renewal, but does not discard native cleanup evidence.
func (s *Server) ownBrowserTerminal(r *http.Request, v store.Session, p store.Project, login string, key terminalKey, entry *browserTerminal, native *host.Terminal) {
	defer s.terminalWG.Done()
	type result struct {
		frame host.TerminalFrame
		err   error
	}
	readCtx, stopRead := context.WithCancel(context.Background())
	defer stopRead()
	events := make(chan result, 2)
	go func() {
		for {
			frame, err := native.Receive(readCtx)
			select {
			case events <- result{frame, err}:
			case <-readCtx.Done():
				return
			}
			if err != nil || frame.Type == "closed" {
				return
			}
		}
	}()
	var last result
	defer func() {
		entry.cancel()
		if last.err == nil && last.frame.Type != "closed" {
			cleanup, done := context.WithTimeout(context.Background(), 20*time.Second)
			_ = native.Send(cleanup, host.TerminalFrame{Type: "close"})
			for last.err == nil && last.frame.Type != "closed" {
				select {
				case last = <-events:
				case <-cleanup.Done():
					last.err = cleanup.Err()
				}
			}
			done()
		}
		native.Close()
		confirmed := last.err == nil && last.frame.Type == "closed" && (last.frame.Reason == "disconnected" || last.frame.Reason == "expired" || last.frame.Reason == "exited" || last.frame.Reason == "launch_failed" || last.frame.Reason == "stream_failed")
		s.terminalMu.Lock()
		if confirmed && s.terminals[key] == entry {
			delete(s.terminals, key)
		} else {
			entry.unconfirmed = true
		}
		s.terminalMu.Unlock()
	}()
	start := time.NewTimer(30 * time.Second)
	select {
	case last = <-events:
	case <-entry.ctx.Done():
	case <-start.C:
	}
	start.Stop()
	s.terminalMu.Lock()
	entry.started = last.err == nil && last.frame.Type == "ready" && entry.ctx.Err() == nil
	close(entry.ready)
	started := entry.started
	s.terminalMu.Unlock()
	if !started {
		return
	}
	tick := time.NewTicker(15 * time.Second)
	defer tick.Stop()
	for {
		select {
		case <-entry.ctx.Done():
			return
		case last = <-events:
			return
		case <-tick.C:
			check, done := context.WithTimeout(entry.ctx, 10*time.Second)
			_, authorityErr := s.visibleRepository(r.WithContext(check), v, p.RepositoryID)
			s.terminalMu.Lock()
			live := authorityErr == nil && entry.ctx.Err() == nil && (entry.retainUntil.IsZero() || time.Now().Before(entry.retainUntil)) && s.terminalCurrent(check, entry.token, v, p, login)
			var sendErr error
			if live {
				sendErr = native.Send(check, host.TerminalFrame{Type: "heartbeat"})
			}
			s.terminalMu.Unlock()
			done()
			if !live || sendErr != nil {
				return
			}
		}
	}
}

// Protected metadata and explicit lifetime decisions. Neither a GET nor a
// successful automatic attachment renews abandonment or creates a terminal.
func (s *Server) apiTerminalSession(w http.ResponseWriter, r *http.Request, v store.Session) {
	if r.URL.RawQuery != "" || r.URL.ForceQuery {
		jsonError(w, 400, "invalid_request", "No query parameters are accepted.")
		return
	}
	p, ok := s.loadEnvironment(w, r)
	if !ok {
		return
	}
	login, err := s.Store.MemberLogin(r.Context(), p.ID, v.User.ID)
	if err != nil || !p.Ready || login == "root" || !projectLogin.MatchString(login) {
		jsonError(w, 403, "membership_required", "An existing account is required.")
		return
	}
	if _, err = s.visibleRepository(r, v, p.RepositoryID); err != nil {
		providerError(w, err)
		return
	}
	var in struct {
		Action  string `json:"action"`
		ID      string `json:"id"`
		Seconds int    `json:"seconds"`
	}
	if r.Method == "POST" {
		if !decodeAPIObject(w, r, &in) {
			return
		}
		if !browserTerminalID.MatchString(in.ID) || (in.Action != "end" && in.Action != "return" && in.Action != "retain") || (in.Action == "retain" && in.Seconds != 1800 && in.Seconds != 7200) || (in.Action != "retain" && in.Seconds != 0) {
			jsonError(w, 400, "invalid_action", "Choose an explicit terminal lifetime action.")
			return
		}
	}
	cookie, err := requestCookie(r, sessionCookie)
	if err != nil {
		jsonError(w, 401, "unauthenticated", "Sign in again.")
		return
	}
	s.terminalMu.Lock()
	defer s.terminalMu.Unlock()
	entry := s.terminals[terminalKey{v.ContextID, p.ID}]
	if s.terminalClosed || !s.terminalCurrent(r.Context(), cookie.Value, v, p, login) {
		jsonError(w, 401, "unauthenticated", "Session ended.")
		return
	}
	if entry == nil || entry.ctx.Err() != nil || (!entry.retainUntil.IsZero() && !time.Now().Before(entry.retainUntil)) {
		if entry != nil {
			entry.cancel()
		}
		if r.Method == "POST" {
			jsonError(w, 404, "terminal_absent", "Terminal ended or expired. Nothing was created.")
		} else if entry != nil {
			state := "ending"
			if entry.unconfirmed {
				state = "unconfirmed"
			}
			jsonResponse(w, 200, map[string]any{"terminal": map[string]any{"id": entry.id, "login": login, "repository_id": strconv.FormatInt(p.RepositoryID, 10), "state": state, "ready": false, "attached": false}})
		} else {
			jsonResponse(w, 200, map[string]any{"terminal": nil})
		}
		return
	}
	if r.Method == "POST" {
		if in.ID != entry.id {
			jsonError(w, 404, "terminal_absent", "Terminal not found in this login context.")
			return
		}
		switch in.Action {
		case "end":
			entry.cancel()
			jsonResponse(w, 200, map[string]any{"ending": true})
			return
		case "return":
			// A detached return is finite too; no unattended indefinite lease.
			entry.retainUntil = time.Now().Add(30 * time.Minute)
			if entry.attachment != nil {
				entry.retainUntil = time.Time{}
			}
		case "retain":
			entry.retainUntil = time.Now().Add(time.Duration(in.Seconds) * time.Second)
		}
	}
	until := int64(0)
	if !entry.retainUntil.IsZero() {
		until = entry.retainUntil.Unix()
	}
	jsonResponse(w, 200, map[string]any{"terminal": map[string]any{"id": entry.id, "login": login, "repository_id": strconv.FormatInt(p.RepositoryID, 10), "ready": entry.started, "attached": entry.attachment != nil, "retain_until": until}})
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
	ctx, cancel := context.WithDeadline(r.Context(), time.Unix(v.Expires, 0))
	defer cancel()
	s.terminalMu.Lock()
	existing := s.terminals[terminalKey{v.ContextID, p.ID}]
	if s.terminalClosed || len(s.terminalPeers) >= 128 || (existing != nil && existing.attachment != nil) {
		s.terminalMu.Unlock()
		jsonError(w, 409, "terminal_unavailable", "Terminal transport unavailable.")
		return
	}
	if s.terminalPeers == nil {
		s.terminalPeers = make(map[*http.Request]*terminalPeer)
	}
	s.terminalPeers[r] = &terminalPeer{token: cookie.Value, contextID: v.ContextID, cancel: cancel}
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
		_ = conn.Close(websocket.StatusPolicyViolation, "terminal unavailable or authorization failed")
	}
	first, done := context.WithTimeout(ctx, 5*time.Second)
	kind, body, err := conn.Read(first)
	done()
	var in struct {
		Action         string `json:"action"`
		ID             string `json:"id"`
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
	if !validActor || actor != v.User.ID || !validRepo || repo != p.RepositoryID || !s.validAPIMutation(authorized, v.CSRF) || in.Cols < 2 || in.Cols > 500 || in.Rows < 2 || in.Rows > 300 || (in.Action != "create" && in.Action != "attach") || (in.Action == "create" && in.ID != "") || (in.Action == "attach" && !browserTerminalID.MatchString(in.ID)) {
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
	key := terminalKey{v.ContextID, p.ID}
	attachment := &terminalAttachment{cancel: cancel}
	s.terminalMu.Lock()
	entry := s.terminals[key]
	if s.terminalClosed || s.terminalStopping[p.ID] || ctx.Err() != nil || !s.terminalCurrent(ctx, cookie.Value, v, p, login) {
		s.terminalMu.Unlock()
		refuse()
		return
	}
	if in.Action == "create" {
		if entry != nil || len(s.terminals) >= 64 {
			s.terminalMu.Unlock()
			refuse()
			return
		}
		var identifier [16]byte
		if _, err = rand.Read(identifier[:]); err != nil {
			s.terminalMu.Unlock()
			refuse()
			return
		}
		ownerCtx, ownerCancel := context.WithDeadline(context.Background(), time.Unix(v.Expires, 0))
		entry = &browserTerminal{token: cookie.Value, cancel: ownerCancel, ctx: ownerCtx, id: hex.EncodeToString(identifier[:]), ready: make(chan struct{}), attachment: attachment}
		if s.terminals == nil {
			s.terminals = make(map[terminalKey]*browserTerminal)
		}
		s.terminals[key] = entry
		owner, openErr := s.Host.OpenTerminal(ownerCtx, host.TerminalRequest{Action: "create", ID: entry.id, Project: p.ID, Login: login, Identity: v.User.ID, Cols: in.Cols, Rows: in.Rows, Expires: v.Expires})
		if openErr != nil {
			ownerCancel()
			entry.unconfirmed = true
			entry.attachment = nil
			close(entry.ready)
			s.terminalMu.Unlock()
			refuse()
			return
		}
		s.terminalWG.Add(1)
		go s.ownBrowserTerminal(authorized.Clone(ownerCtx), v, p, login, key, entry, owner)
	} else {
		if entry == nil || entry.id != in.ID || entry.ctx.Err() != nil || entry.attachment != nil || (!entry.retainUntil.IsZero() && !time.Now().Before(entry.retainUntil)) {
			s.terminalMu.Unlock()
			refuse()
			return
		}
		entry.attachment = attachment
	}
	s.terminalMu.Unlock()
	endAttachment := context.AfterFunc(entry.ctx, cancel)
	defer endAttachment()
	defer func() {
		s.terminalMu.Lock()
		if entry.attachment == attachment {
			entry.attachment = nil
			if entry.retainUntil.IsZero() {
				entry.retainUntil = time.Now().Add(30 * time.Minute)
			}
		}
		s.terminalMu.Unlock()
	}()
	// Publish the locator even while native startup is pending. Losing this reply
	// is recoverable by an authorized metadata GET, never by retrying creation.
	message, _ := json.Marshal(map[string]string{"type": "session", "id": entry.id})
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
	var native *host.Terminal
	if entry.started && entry.ctx.Err() == nil && ctx.Err() == nil && s.terminalCurrent(ctx, cookie.Value, v, p, login) {
		native, err = s.Host.OpenTerminal(ctx, host.TerminalRequest{Action: "attach", ID: entry.id, Project: p.ID, Login: login, Identity: v.User.ID, Cols: in.Cols, Rows: in.Rows, Expires: v.Expires})
	}
	s.terminalMu.Unlock()
	if native == nil || err != nil {
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
				live := entry.ctx.Err() == nil && ctx.Err() == nil && (entry.retainUntil.IsZero() || time.Now().Before(entry.retainUntil))
				var sendErr error
				if live {
					sendErr = native.Send(ctx, frame)
				}
				s.terminalMu.Unlock()
				if !live || sendErr != nil {
					return
				}
			case <-tick.C:
				check, done := context.WithTimeout(ctx, 5*time.Second)
				peer := conn.Ping(check) == nil
				s.terminalMu.Lock()
				live := peer && entry.ctx.Err() == nil && s.terminalCurrent(check, cookie.Value, v, p, login)
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
