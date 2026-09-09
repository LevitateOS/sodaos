package web

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"regexp"
	"strconv"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/levitateos/sodaos/internal/host"
	"github.com/levitateos/sodaos/internal/store"
)

var browserTerminalID = regexp.MustCompile(`^[0-9a-f]{32}$`)

func newTerminalID() string { var b [16]byte; rand.Read(b[:]); return hex.EncodeToString(b[:]) }

type terminalAttachment struct{ id string }
type terminalPeer struct {
	token, contextID, project string
	cancel                    context.CancelFunc
}

// Immutable authorization facts, never serialized as credentials or updated from
// a later page's repository/account. Shared by the live owner and its short receipt.
type terminalBinding struct {
	token              string
	session            store.Session
	project            store.Project
	login              string
	created, hardUntil time.Time
}

func (b terminalBinding) matches(v store.Session, p store.Project, login, token string) bool {
	return b.token == token && b.session.ContextID == v.ContextID && b.session.User.ID == v.User.ID && b.session.CSRF == v.CSRF && b.project.ID == p.ID && b.project.RepositoryID == p.RepositoryID && b.login == login
}

type browserTerminal struct {
	terminalBinding
	cancel               context.CancelFunc
	ctx                  context.Context
	id, requestID, name  string
	ready                chan struct{}
	started, unconfirmed bool
	attachment           *terminalAttachment
	lastAttachment       string
	// Zero means deliberately active. Attachment/output/pings never clear this.
	retainUntil time.Time
}
type terminalView struct {
	ID             string `json:"id"`
	RequestID      string `json:"request_id"`
	EnvironmentID  string `json:"environment_id"`
	RepositoryID   string `json:"repository_id"`
	UserID         string `json:"user_id"`
	Login          string `json:"login"`
	Name           string `json:"name"`
	CreatedAt      int64  `json:"created_at"`
	HardUntil      int64  `json:"hard_until"`
	RetainUntil    int64  `json:"retain_until"`
	EffectiveUntil int64  `json:"effective_until"`
	Ready          bool   `json:"ready"`
	Attached       bool   `json:"attached"`
	State          string `json:"state"`
}
type terminalReceipt struct {
	binding terminalBinding
	view    terminalView
	expires time.Time
}

func (e *browserTerminal) view() terminalView {
	state := "opening"
	if e.started {
		state = "ready"
	}
	if e.ctx.Err() != nil {
		state = "ending"
	}
	if e.unconfirmed {
		state = "unconfirmed"
	}
	until := int64(0)
	if !e.retainUntil.IsZero() {
		until = e.retainUntil.Unix()
	}
	effective := e.hardUntil.Unix()
	if until != 0 && until < effective {
		effective = until
	}
	return terminalView{e.id, e.requestID, e.project.ID, strconv.FormatInt(e.project.RepositoryID, 10), strconv.FormatInt(e.session.User.ID, 10), e.login, e.name, e.created.Unix(), e.hardUntil.Unix(), until, effective, e.started && e.ctx.Err() == nil, e.attachment != nil && e.ctx.Err() == nil, state}
}
func (e *browserTerminal) live() bool {
	return e.ctx.Err() == nil && (e.retainUntil.IsZero() || time.Now().Before(e.retainUntil))
}
func (e *browserTerminal) retain(seconds int) {
	e.retainUntil = minTime(time.Now().Add(time.Duration(seconds)*time.Second), e.hardUntil)
}
func minTime(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}
func validTerminalName(name string) bool {
	if !utf8.ValidString(name) || utf8.RuneCountInString(name) > 80 {
		return false
	}
	for _, c := range name {
		if unicode.IsControl(c) || unicode.Is(unicode.Cf, c) {
			return false
		}
	}
	return true
}

// Every registry access is under terminalMu. Only bounded in-process facts and
// local store checks run there, never provider or native network IO.
func (s *Server) expireTerminalReceipts() {
	for id, receipt := range s.terminalReceipts {
		if !time.Now().Before(receipt.expires) {
			delete(s.terminalReceipts, id)
		}
	}
}
func (s *Server) cancelTerminals(contextID, token string) {
	for _, peer := range s.terminalPeers {
		if (contextID != "" && peer.contextID == contextID) || (token != "" && peer.token == token) {
			peer.cancel()
		}
	}
	for _, entry := range s.terminals {
		if (contextID != "" && entry.session.ContextID == contextID) || (token != "" && entry.token == token) {
			entry.cancel()
		}
	}
	for id, receipt := range s.terminalReceipts {
		if (contextID != "" && receipt.binding.session.ContextID == contextID) || (token != "" && receipt.binding.token == token) {
			delete(s.terminalReceipts, id)
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

// Receives readiness/exit only, never terminal output. Browser cancellation stops
// renewal but does not discard cleanup evidence. The independent native lease is
// still required if this owner/process loses its stream.
func (s *Server) ownBrowserTerminal(r *http.Request, entry *browserTerminal, cols, rows int) {
	defer s.terminalWG.Done()
	native, err := s.Host.OpenTerminal(entry.ctx, host.TerminalRequest{Action: "create", ID: entry.id, Project: entry.project.ID, Login: entry.login, Identity: entry.session.User.ID, Cols: cols, Rows: rows, Expires: entry.hardUntil.Unix()})
	if err != nil {
		entry.cancel()
		s.terminalMu.Lock()
		entry.unconfirmed = true
		close(entry.ready)
		s.terminalMu.Unlock()
		return
	}
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
		defer s.terminalMu.Unlock()
		if confirmed && s.terminals[entry.id] == entry {
			delete(s.terminals, entry.id)
			s.expireTerminalReceipts()
			expires := minTime(time.Now().Add(5*time.Minute), time.Unix(entry.session.Expires, 0))
			// Never evict another receipt to imply a complete history. No receipt means
			// unknown, including a full receipt bound or an expired authentication.
			if len(s.terminalReceipts) < 128 && time.Now().Before(expires) {
				if s.terminalReceipts == nil {
					s.terminalReceipts = make(map[string]terminalReceipt)
				}
				view := entry.view()
				view.State = "ended"
				view.Ready = false
				view.Attached = false
				s.terminalReceipts[entry.id] = terminalReceipt{entry.terminalBinding, view, expires}
			}
		} else {
			entry.unconfirmed = true
		}
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
			_, err := s.visibleRepository(r.WithContext(check), entry.session, entry.project.RepositoryID)
			current := err == nil && s.terminalCurrent(check, entry.token, entry.session, entry.project, entry.login)
			s.terminalMu.Lock()
			live := current && s.terminals[entry.id] == entry && entry.live()
			s.terminalMu.Unlock()
			if live {
				err = native.Send(check, host.TerminalFrame{Type: "heartbeat"})
			}
			done()
			if !live || err != nil {
				return
			}
		}
	}
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
func terminalMetadata(w http.ResponseWriter, view *terminalView) {
	jsonResponse(w, 200, struct {
		Terminal *terminalView `json:"terminal"`
	}{view})
}
func (s *Server) apiTerminalAttempt(w http.ResponseWriter, r *http.Request, v store.Session) {
	requestID := r.PathValue("requestID")
	if !browserTerminalID.MatchString(requestID) {
		jsonError(w, 400, "invalid_request", "Provide an exact request ID.")
		return
	}
	p, login, token, ok := s.terminalAccount(w, r, v)
	if !ok {
		return
	}
	s.terminalMu.Lock()
	if s.terminalClosed || !s.terminalCurrent(r.Context(), token, v, p, login) {
		s.terminalMu.Unlock()
		jsonError(w, 401, "unauthenticated", "Session ended.")
		return
	}
	s.expireTerminalReceipts()
	for _, entry := range s.terminals {
		if entry.requestID == requestID && entry.matches(v, p, login, token) {
			if !entry.live() {
				entry.cancel()
			}
			view := entry.view()
			s.terminalMu.Unlock()
			terminalMetadata(w, &view)
			return
		}
	}
	for _, receipt := range s.terminalReceipts {
		if receipt.view.RequestID == requestID && receipt.binding.matches(v, p, login, token) {
			s.terminalMu.Unlock()
			terminalMetadata(w, &receipt.view)
			return
		}
	}
	s.terminalMu.Unlock()
	terminalMetadata(w, nil) // Absence is unknown, never proof of no native effect.
}
func (s *Server) apiTerminalSession(w http.ResponseWriter, r *http.Request, v store.Session) {
	id := r.PathValue("terminalID")
	if !browserTerminalID.MatchString(id) {
		jsonError(w, 400, "invalid_request", "Provide an exact terminal ID.")
		return
	}
	var in struct {
		Action     string  `json:"action"`
		Seconds    int     `json:"seconds"`
		Attachment string  `json:"attachment_id"`
		Name       *string `json:"name"`
	}
	if r.Method == "POST" {
		if !decodeAPIObject(w, r, &in) {
			return
		}
		valid := false
		switch in.Action {
		case "end":
			valid = in.Seconds == 0 && in.Attachment == "" && in.Name == nil
		case "rename":
			valid = in.Seconds == 0 && in.Attachment == "" && in.Name != nil && validTerminalName(*in.Name)
		case "retain":
			valid = (in.Seconds == 1800 || in.Seconds == 7200) && in.Attachment == "" && in.Name == nil
		case "hide":
			valid = in.Seconds == 0 && browserTerminalID.MatchString(in.Attachment) && in.Name == nil
		case "return":
			valid = in.Seconds == 0 && (in.Attachment == "" || browserTerminalID.MatchString(in.Attachment)) && in.Name == nil
		}
		if !valid {
			jsonError(w, 400, "invalid_action", "Choose a valid exact-terminal action.")
			return
		}
	}
	p, login, token, ok := s.terminalAccount(w, r, v)
	if !ok {
		return
	}
	s.terminalMu.Lock()
	if s.terminalClosed || !s.terminalCurrent(r.Context(), token, v, p, login) {
		s.terminalMu.Unlock()
		jsonError(w, 401, "unauthenticated", "Session ended.")
		return
	}
	s.expireTerminalReceipts()
	entry := s.terminals[id]
	if entry == nil || !entry.matches(v, p, login, token) {
		receipt, found := s.terminalReceipts[id]
		allowed := found && receipt.binding.matches(v, p, login, token)
		s.terminalMu.Unlock()
		if r.Method == "GET" {
			if allowed {
				terminalMetadata(w, &receipt.view)
			} else {
				terminalMetadata(w, nil)
			}
		} else {
			jsonError(w, 404, "terminal_unknown", "No live terminal is authorized at that ID.")
		}
		return
	}
	if !entry.live() {
		entry.cancel()
	}
	if r.Method == "POST" {
		if !entry.live() {
			s.terminalMu.Unlock()
			jsonError(w, 409, "terminal_ending", "Terminal is ending or unconfirmed; inspect its exact result.")
			return
		}
		switch in.Action {
		case "end":
			entry.cancel()
			s.terminalMu.Unlock()
			jsonResponse(w, 200, map[string]any{"ending": true})
			return
		case "rename":
			entry.name = *in.Name
		case "hide":
			if entry.lastAttachment != in.Attachment {
				s.terminalMu.Unlock()
				jsonError(w, 409, "attachment_changed", "The attachment changed; its lifetime was not modified.")
				return
			}
			if entry.retainUntil.IsZero() {
				entry.retain(1800)
			}
		case "return":
			if (entry.attachment != nil && in.Attachment != entry.attachment.id) || (in.Attachment != "" && in.Attachment != entry.lastAttachment) {
				s.terminalMu.Unlock()
				jsonError(w, 409, "attachment_changed", "The attachment changed; its lifetime was not modified.")
				return
			}
			if entry.attachment != nil {
				entry.retainUntil = time.Time{}
			} else {
				entry.retain(1800)
			}
		case "retain":
			entry.retain(in.Seconds)
		}
	}
	view := entry.view()
	s.terminalMu.Unlock()
	terminalMetadata(w, &view)
}
