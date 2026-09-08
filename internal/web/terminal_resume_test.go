package web

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
)

func resumeMetadata(t *testing.T, s *Server, origin string) map[string]any {
	t.Helper()
	w := httptest.NewRecorder()
	r := apiTestRequest("GET", "/api/environments/"+webTerminalProject+"/terminal-session", "", "alice")
	r.Header.Set("Origin", origin)
	s.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatalf("metadata status %d", w.Code)
	}
	var body struct {
		Terminal map[string]any `json:"terminal"`
	}
	if json.Unmarshal(w.Body.Bytes(), &body) != nil {
		t.Fatal("invalid metadata")
	}
	return body.Terminal
}
func resumeAction(t *testing.T, s *Server, origin, id, action string, seconds int) int {
	t.Helper()
	body, _ := json.Marshal(map[string]any{"id": id, "action": action, "seconds": seconds})
	w := httptest.NewRecorder()
	r := apiTestRequest("POST", "/api/environments/"+webTerminalProject+"/terminal-session", string(body), "alice")
	r.Header.Set("Origin", origin)
	s.ServeHTTP(w, r)
	return w.Code
}
func waitDetached(t *testing.T, s *Server, origin string) map[string]any {
	t.Helper()
	until := time.Now().Add(2 * time.Second)
	for time.Now().Before(until) {
		m := resumeMetadata(t, s, origin)
		if m != nil && m["attached"] == false {
			return m
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("attachment did not detach")
	return nil
}
func attachAuth(id string) []byte {
	var in map[string]any
	_ = json.Unmarshal([]byte(terminalAuth), &in)
	in["action"] = "attach"
	in["id"] = id
	body, _ := json.Marshal(in)
	return body
}
func TestTerminalDetachReattachesWithoutCreationOrDeadlineRenewal(t *testing.T) {
	s, srv, calls, closed := terminalWebFixture(t, 0)
	c, _, err := terminalDial(t, srv, "")
	if err != nil {
		t.Fatal(err)
	}
	terminalReady(t, c)
	m := resumeMetadata(t, s, srv.URL)
	id := m["id"].(string)
	c.CloseNow()
	select {
	case <-closed:
	case <-time.After(2 * time.Second):
		t.Fatal("attachment not closed")
	}
	m = waitDetached(t, s, srv.URL)
	deadline := m["retain_until"]
	if deadline.(float64) <= float64(time.Now().Unix()) {
		t.Fatal("no detached retention")
	}
	select {
	case <-closed:
		t.Fatal("browser departure ended native owner")
	case <-time.After(20 * time.Millisecond):
	}
	c, _, err = terminalDial(t, srv, "")
	if err != nil {
		t.Fatal(err)
	}
	defer c.CloseNow()
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()
	if c.Write(ctx, websocket.MessageText, attachAuth(id)) != nil {
		t.Fatal("attach failed")
	}
	for i := 0; i < 2; i++ {
		if _, _, err = c.Read(ctx); err != nil {
			t.Fatal("reattach failed", err)
		}
	}
	m = resumeMetadata(t, s, srv.URL)
	if m["id"] != id || m["retain_until"] != deadline || calls.Load() != 3 {
		t.Fatal("reattach replaced session or renewed abandonment")
	}
	if resumeAction(t, s, srv.URL, id, "return", 0) != 200 || resumeMetadata(t, s, srv.URL)["retain_until"] != float64(0) {
		t.Fatal("deliberate return did not activate")
	}
	if resumeAction(t, s, srv.URL, id, "retain", 7200) != 200 {
		t.Fatal("finite away failed")
	}
	if got := resumeMetadata(t, s, srv.URL)["retain_until"].(float64); got < float64(time.Now().Add(119*time.Minute).Unix()) || got > float64(time.Now().Add(121*time.Minute).Unix()) {
		t.Fatal("not finite away")
	}
	if resumeAction(t, s, srv.URL, id, "end", 0) != 200 {
		t.Fatal("End failed")
	}
	if _, _, err = c.Read(ctx); err == nil {
		t.Fatal("End retained attachment")
	}
	if calls.Load() != 3 {
		t.Fatal("End created a native session")
	}
}
func TestTerminalMissingAttachAndLifetimeActionsCreateNothing(t *testing.T) {
	s, srv, calls, _ := terminalWebFixture(t, 0)
	id := strings.Repeat("f", 32)
	if resumeMetadata(t, s, srv.URL) != nil {
		t.Fatal("unexpected terminal")
	}
	for _, action := range []string{"end", "return"} {
		if resumeAction(t, s, srv.URL, id, action, 0) != 404 {
			t.Fatal("absent action not refused")
		}
	}
	if resumeAction(t, s, srv.URL, id, "retain", 1800) != 404 {
		t.Fatal("absent retention accepted")
	}
	c, _, err := terminalDial(t, srv, "")
	if err != nil {
		t.Fatal(err)
	}
	defer c.CloseNow()
	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()
	_ = c.Write(ctx, websocket.MessageText, attachAuth(id))
	if _, _, err = c.Read(ctx); err == nil {
		t.Fatal("missing attach accepted")
	}
	if calls.Load() != 0 {
		t.Fatal("missing target reached native")
	}
}
func TestTerminalIdentifierDoesNotGrantAnotherSodaContextAccess(t *testing.T) {
	s, srv, calls, _ := terminalWebFixture(t, 0)
	c, _, err := terminalDial(t, srv, "")
	if err != nil {
		t.Fatal(err)
	}
	terminalReady(t, c)
	id := resumeMetadata(t, s, srv.URL)["id"].(string)
	c.CloseNow()
	waitDetached(t, s, srv.URL)
	grant, err := s.Store.Grant(t.Context(), "session-alice", 1)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Store.CreateGrantedSession(t.Context(), "other-context", 1, "other-csrf", grant); err != nil {
		t.Fatal(err)
	}
	c, _, err = websocket.Dial(t.Context(), "wss"+strings.TrimPrefix(srv.URL, "https")+"/-/soda/api/environments/"+webTerminalProject+"/terminal", &websocket.DialOptions{HTTPClient: srv.Client(), HTTPHeader: http.Header{"Origin": {srv.URL}, "Cookie": {"__Secure-sodaspaces-session=other-context"}}})
	if err != nil {
		t.Fatal(err)
	}
	defer c.CloseNow()
	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()
	_ = c.Write(ctx, websocket.MessageText, []byte(strings.ReplaceAll(string(attachAuth(id)), "csrf-alice", "other-csrf")))
	if _, _, err = c.Read(ctx); err == nil {
		t.Fatal("cross-context adoption")
	}
	if calls.Load() != 2 {
		t.Fatal("cross-context request reached helper")
	}
}
func TestTerminalExpiredRetentionCannotBeExtendedOrReattached(t *testing.T) {
	s, srv, calls, _ := terminalWebFixture(t, 0)
	c, _, err := terminalDial(t, srv, "")
	if err != nil {
		t.Fatal(err)
	}
	terminalReady(t, c)
	id := resumeMetadata(t, s, srv.URL)["id"].(string)
	c.CloseNow()
	waitDetached(t, s, srv.URL)
	s.terminalMu.Lock()
	for _, entry := range s.terminals {
		entry.retainUntil = time.Now().Add(-time.Second)
	}
	s.terminalMu.Unlock()
	if resumeAction(t, s, srv.URL, id, "return", 0) != 404 {
		t.Fatal("expired session revived")
	}
	until := time.Now().Add(2 * time.Second)
	for resumeMetadata(t, s, srv.URL) != nil && time.Now().Before(until) {
		time.Sleep(5 * time.Millisecond)
	}
	if resumeMetadata(t, s, srv.URL) != nil || calls.Load() != 2 {
		t.Fatal("expiry created or exposed terminal")
	}
}
func TestTerminalUnconfirmedCleanupKeepsSlotAndRefusesReplacement(t *testing.T) {
	s, srv, calls, _ := terminalWebFixture(t, 0, "cleanup_unconfirmed")
	c, _, err := terminalDial(t, srv, "")
	if err != nil {
		t.Fatal(err)
	}
	terminalReady(t, c)
	id := resumeMetadata(t, s, srv.URL)["id"].(string)
	if resumeAction(t, s, srv.URL, id, "end", 0) != 200 {
		t.Fatal("End failed")
	}
	c.CloseNow()
	until := time.Now().Add(2 * time.Second)
	for time.Now().Before(until) {
		m := resumeMetadata(t, s, srv.URL)
		if m != nil && m["state"] == "unconfirmed" {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	m := resumeMetadata(t, s, srv.URL)
	if m == nil || m["state"] != "unconfirmed" || m["id"] != id {
		t.Fatal("uncertain cleanup freed slot")
	}
	c, _, err = terminalDial(t, srv, "")
	if err != nil {
		t.Fatal(err)
	}
	defer c.CloseNow()
	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()
	_ = c.Write(ctx, websocket.MessageText, []byte(terminalAuth))
	if _, _, err = c.Read(ctx); err == nil || calls.Load() != 2 {
		t.Fatal("unconfirmed terminal replaced")
	}
}

func TestTerminalStopGateAndExplicitCreateContract(t *testing.T) {
	for _, body := range []string{strings.Replace(terminalAuth, `"action":"create",`, "", 1), strings.Replace(terminalAuth, `"action":"create"`, `"action":"create","id":"`+strings.Repeat("a", 32)+`"`, 1), terminalAuth} {
		t.Run(body, func(t *testing.T) {
			s, srv, calls, _ := terminalWebFixture(t, 0)
			s.terminalMu.Lock()
			s.terminalStopping = map[string]bool{webTerminalProject: true}
			s.terminalMu.Unlock()
			c, _, err := terminalDial(t, srv, "")
			if err != nil {
				t.Fatal(err)
			}
			defer c.CloseNow()
			ctx, cancel := context.WithTimeout(t.Context(), time.Second)
			defer cancel()
			_ = c.Write(ctx, websocket.MessageText, []byte(body))
			if _, _, err = c.Read(ctx); err == nil || calls.Load() != 0 {
				t.Fatal("invalid or Stop-racing creation reached helper")
			}
		})
	}
}
