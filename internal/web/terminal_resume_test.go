package web

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/levitateos/sodaos/internal/web/api"
)

func waitDetached(t *testing.T, s *Server, origin, id string) *api.TerminalView {
	t.Helper()
	until := time.Now().Add(2 * time.Second)
	for time.Now().Before(until) {
		v := exactMetadata(t, s, origin, webTerminalProject, id)
		if v != nil && !v.Attached {
			return v
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("attachment did not detach")
	return nil
}
func attachAuth(id string) []byte {
	return []byte(strings.Replace(strings.Replace(terminalAuth, `"create"`, `"attach"`, 1), reservedTerminalID, id, 1))
}
func attachManaged(t *testing.T, srv *httptest.Server, id string) *websocket.Conn {
	t.Helper()
	c, _, err := terminalDial(t, srv, "")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.CloseNow() })
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()
	_ = c.Write(ctx, websocket.MessageText, attachAuth(id))
	_, b, err := c.Read(ctx)
	if err != nil || string(b) != `{"type":"ready"}` {
		t.Fatal("attach", err, string(b))
	}
	return c
}

func TestTerminalDetachDoesNotOwnNativeLifetime(t *testing.T) {
	s, srv, native, _ := terminalWebFixture(t, 0)
	c, id := openManaged(t, s, srv, webTerminalProject)
	created := exactMetadata(t, s, srv.URL, webTerminalProject, id).CreatedAt
	c.CloseNow()
	detached := waitDetached(t, s, srv.URL, id)
	if detached.State != "ready" || !detached.Ready || native.count("end") != 0 {
		t.Fatal("browser departure ended native work")
	}
	c = attachManaged(t, srv, id)
	if v := exactMetadata(t, s, srv.URL, webTerminalProject, id); v.CreatedAt != created || !v.Attached || native.count("create") != 1 {
		t.Fatal("attachment replaced native session")
	}
	if exactAction(t, s, srv.URL, webTerminalProject, id, map[string]any{"action": "end"}) != 200 {
		t.Fatal("End")
	}
	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()
	if _, _, err := c.Read(ctx); err == nil {
		t.Fatal("End retained attachment")
	}
}

func TestTerminalNativeLookupSurvivesWebRestart(t *testing.T) {
	s, srv, native, _ := terminalWebFixture(t, 0)
	_, id := openManaged(t, s, srv, webTerminalProject)
	s.CloseTerminals()
	next := New(s.Config, s.Store)
	next.SetHost(s.Host)
	next.SetForgejo(s.Forgejo)
	nextServer := httptest.NewTLSServer(next)
	next.Config.ForgejoURL = nextServer.URL
	t.Cleanup(nextServer.Close)
	t.Cleanup(next.CloseTerminals)
	waitDetached(t, next, nextServer.URL, id)
	attachManaged(t, nextServer, id)
	if native.count("create") != 1 || native.count("end") != 0 {
		t.Fatal("web restart destroyed or recreated work")
	}
	refusedTerminal(t, nextServer, strings.Replace(terminalAuth, reservedTerminalID, id, 1))
}

func TestTerminalFreshAuthorizedContextMayReattachButLocatorIsNotAuthority(t *testing.T) {
	s, srv, native, _ := terminalWebFixture(t, 0)
	c, id := openManaged(t, s, srv, webTerminalProject)
	c.CloseNow()
	waitDetached(t, s, srv.URL, id)
	grant, err := s.Store.Grant(t.Context(), "session-alice", 1)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Store.CreateGrantedSession(t.Context(), "other-context", 1, "other-csrf", grant); err != nil {
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
	if _, _, err := c.Read(ctx); err != nil {
		t.Fatal("fresh authorized attachment refused", err)
	}
	if native.count("create") != 1 {
		t.Fatal("fresh context recreated shell")
	}
	w := terminalAPI(t, s, srv.URL, "GET", "/api/environments/"+webTerminalProject+"/terminal-sessions/"+id, nil)
	if w.Code != 200 {
		t.Fatal("original authorized actor lost lookup")
	}
	// A stale CSRF/actor or missing native membership is still refused upstream.
	refusedTerminal(t, srv, strings.Replace(string(attachAuth(id)), "csrf-alice", "wrong", 1))
}

func TestTerminalUnavailableObservationDoesNotBecomeAbsenceOrPermanentCustody(t *testing.T) {
	s, srv, native, _ := terminalWebFixture(t, 0)
	c, id := openManaged(t, s, srv, webTerminalProject)
	c.CloseNow()
	waitDetached(t, s, srv.URL, id)
	native.mu.Lock()
	native.fail["inspect"] = true
	native.mu.Unlock()
	path := "/api/environments/" + webTerminalProject + "/terminal-sessions/" + id
	if terminalAPI(t, s, srv.URL, "GET", path, nil).Code != 503 {
		t.Fatal("unavailable became absence")
	}
	native.mu.Lock()
	delete(native.fail, "inspect")
	native.fail["end"] = true
	native.mu.Unlock()
	if exactAction(t, s, srv.URL, webTerminalProject, id, map[string]any{"action": "end"}) != 503 {
		t.Fatal("unconfirmed End reported success")
	}
	if exactMetadata(t, s, srv.URL, webTerminalProject, id).State != "ready" {
		t.Fatal("outcome notice overrode native observation")
	}
	native.mu.Lock()
	delete(native.fail, "end")
	native.mu.Unlock()
	if exactAction(t, s, srv.URL, webTerminalProject, id, map[string]any{"action": "end"}) != 200 || exactMetadata(t, s, srv.URL, webTerminalProject, id) != nil {
		t.Fatal("explicit exact retry could not recover")
	}
	_, other := openManaged(t, s, srv, webTerminalProject)
	if other == id || native.count("create") != 2 {
		t.Fatal("new explicit work reused old ID")
	}
}

func TestTerminalLostCreateReplyRecoversThroughIssuedNativeID(t *testing.T) {
	s, srv, native, _ := terminalWebFixture(t, 0)
	native.mu.Lock()
	native.fail["create"] = true
	native.mu.Unlock()
	refusedTerminal(t, srv, terminalAuth)
	if exactMetadata(t, s, srv.URL, webTerminalProject, reservedTerminalID) == nil {
		t.Fatal("lost native creation not discoverable")
	}
	refusedTerminal(t, srv, terminalAuth)
	attachManaged(t, srv, reservedTerminalID)
	if native.count("create") != 1 {
		t.Fatal("lost acknowledgement replayed Create")
	}
}

func TestTerminalStopGateAndPendingTransportBound(t *testing.T) {
	s, srv, native, _ := terminalWebFixture(t, 0)
	s.App.TerminalLock().Lock()
	s.App.TerminalStopping = map[string]bool{webTerminalProject: true}
	s.App.TerminalLock().Unlock()
	if terminalAPI(t, s, srv.URL, "POST", "/api/environments/"+webTerminalProject+"/terminal-sessions", map[string]any{"cols": 80, "rows": 24}).Code != 503 {
		t.Fatal("Stop admitted reservation")
	}
	c, response, err := terminalDial(t, srv, "")
	if c != nil {
		c.CloseNow()
	}
	if err == nil || response.StatusCode != 409 || native.Load() != 0 {
		t.Fatal("Stop admitted stream")
	}
	s.App.TerminalLock().Lock()
	clear(s.App.TerminalStopping)
	s.App.TerminalPeers = make(map[*http.Request]*api.TerminalPeer)
	for i := 0; i < 128; i++ {
		s.App.TerminalPeers[new(http.Request)] = &api.TerminalPeer{Cancel: func() {}}
	}
	s.App.TerminalLock().Unlock()
	c, response, err = terminalDial(t, srv, "")
	if c != nil {
		c.CloseNow()
	}
	if err == nil || response.StatusCode != 409 || native.Load() != 0 {
		t.Fatal("pending transport bound bypassed")
	}
	s.App.TerminalLock().Lock()
	clear(s.App.TerminalPeers)
	s.App.TerminalLock().Unlock()
}
