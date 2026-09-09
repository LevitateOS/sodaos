package web

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/levitateos/sodaos/internal/store"
)

func terminalAPI(t *testing.T, s *Server, origin, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	text := ""
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		text = string(b)
	}
	r := apiTestRequest(method, path, text, "alice")
	r.Header.Set("Origin", origin)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	return w
}
func exactMetadata(t *testing.T, s *Server, origin, project, id string) *terminalView {
	t.Helper()
	w := terminalAPI(t, s, origin, "GET", "/api/environments/"+project+"/terminal-sessions/"+id, nil)
	if w.Code != 200 {
		t.Fatalf("metadata %d %s", w.Code, w.Body.String())
	}
	var out struct {
		Terminal *terminalView `json:"terminal"`
	}
	if json.Unmarshal(w.Body.Bytes(), &out) != nil {
		t.Fatal("metadata")
	}
	return out.Terminal
}
func exactAction(t *testing.T, s *Server, origin, project, id string, body any) int {
	return terminalAPI(t, s, origin, "POST", "/api/environments/"+project+"/terminal-sessions/"+id, body).Code
}
func openManaged(t *testing.T, srv *httptest.Server, project, request string) (*websocket.Conn, string, string) {
	t.Helper()
	c, _, err := websocket.Dial(t.Context(), "wss"+strings.TrimPrefix(srv.URL, "https")+"/-/soda/api/environments/"+project+"/terminal", &websocket.DialOptions{HTTPClient: srv.Client(), HTTPHeader: http.Header{"Origin": {srv.URL}, "Cookie": {"__Secure-sodaspaces-session=session-alice"}}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.CloseNow() })
	repo := "7"
	if project == secondTerminalProject {
		repo = "8"
	}
	body := strings.Replace(terminalAuth, "0123456789abcdef0123456789abcdef", request, 1)
	body = strings.Replace(body, `"repository_id":"7"`, `"repository_id":"`+repo+`"`, 1)
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()
	if c.Write(ctx, websocket.MessageText, []byte(body)) != nil {
		t.Fatal("create write")
	}
	_, b, err := c.Read(ctx)
	var frame struct {
		Type, ID     string
		RequestID    string `json:"request_id"`
		AttachmentID string `json:"attachment_id"`
	}
	if err != nil || json.Unmarshal(b, &frame) != nil || frame.Type != "session" || frame.RequestID != request || !browserTerminalID.MatchString(frame.AttachmentID) {
		t.Fatal("locator", err, string(b))
	}
	_, b, err = c.Read(ctx)
	if err != nil || string(b) != `{"type":"ready"}` {
		t.Fatal("ready", err, string(b))
	}
	return c, frame.ID, frame.AttachmentID
}
func waitTerminalState(t *testing.T, s *Server, origin, project, id, state string) *terminalView {
	t.Helper()
	until := time.Now().Add(3 * time.Second)
	for time.Now().Before(until) {
		v := exactMetadata(t, s, origin, project, id)
		if v != nil && v.State == state {
			return v
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("missing terminal state", state)
	return nil
}
func addSecondTerminalProject(t *testing.T, s *Server) {
	t.Helper()
	if err := s.Store.CreateProject(t.Context(), store.Project{ID: secondTerminalProject, Name: "second", RepositoryID: 8, OwnerID: 1, Repository: "alice/second"}); err != nil {
		t.Fatal(err)
	}
	if err := s.Store.MarkReady(t.Context(), secondTerminalProject, ""); err != nil {
		t.Fatal(err)
	}
	if err := s.Store.Join(t.Context(), secondTerminalProject, 1, "original-alice"); err != nil {
		t.Fatal(err)
	}
}
func TestTerminalIDsIndependentStopCollectionAndReceipts(t *testing.T) {
	s, srv, calls, _ := terminalWebFixture(t, 0)
	addSecondTerminalProject(t, s)
	_, one, _ := openManaged(t, srv, webTerminalProject, strings.Repeat("1", 32))
	_, two, _ := openManaged(t, srv, webTerminalProject, strings.Repeat("2", 32))
	_, three, _ := openManaged(t, srv, secondTerminalProject, strings.Repeat("3", 32))
	if one == two || calls.Load() != 6 {
		t.Fatal("not independent")
	}
	if exactMetadata(t, s, srv.URL, secondTerminalProject, one) != nil {
		t.Fatal("wrong project exposed terminal")
	}
	if exactAction(t, s, srv.URL, webTerminalProject, one, map[string]any{"action": "rename", "name": "編集 · build"}) != 200 {
		t.Fatal("rename")
	}
	w := terminalAPI(t, s, srv.URL, "GET", "/api/spaces", nil)
	var collection spacesView
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &collection) != nil || !collection.Complete || len(collection.Items) != 2 || len(collection.Items[0].Terminals) != 2 || len(collection.Items[1].Terminals) != 1 {
		t.Fatal("collection", w.Code, w.Body.String())
	}
	if exactMetadata(t, s, srv.URL, webTerminalProject, one).Name != "編集 · build" || exactMetadata(t, s, srv.URL, webTerminalProject, two).Name != "" {
		t.Fatal("rename leaked")
	}
	if exactAction(t, s, srv.URL, webTerminalProject, one, map[string]any{"action": "end"}) != 200 {
		t.Fatal("end")
	}
	waitTerminalState(t, s, srv.URL, webTerminalProject, one, "ended")
	if !exactMetadata(t, s, srv.URL, webTerminalProject, two).Ready || !exactMetadata(t, s, srv.URL, secondTerminalProject, three).Ready {
		t.Fatal("End crossed ID")
	}
	w = terminalAPI(t, s, srv.URL, "POST", "/api/environments/"+webTerminalProject+"/lifecycle", map[string]any{"action": "stop", "confirm_stop": true})
	if w.Code != 200 {
		t.Fatal("Stop", w.Code, w.Body.String())
	}
	waitTerminalState(t, s, srv.URL, webTerminalProject, two, "ended")
	if !exactMetadata(t, s, srv.URL, secondTerminalProject, three).Ready {
		t.Fatal("Stop crossed project")
	}
	s.terminalMu.Lock()
	if len(s.terminals) != 1 {
		t.Error("confirmed cleanup did not release slots")
	}
	receipt := s.terminalReceipts[one]
	receipt.expires = time.Now().Add(-time.Second)
	s.terminalReceipts[one] = receipt
	s.terminalMu.Unlock()
	if exactMetadata(t, s, srv.URL, webTerminalProject, one) != nil {
		t.Fatal("expired receipt was cleanup proof")
	}
}
func TestTerminalAttachmentGenerationAndDeadlineCaps(t *testing.T) {
	s, srv, calls, _ := terminalWebFixture(t, 0)
	first, id, oldAttachment := openManaged(t, srv, webTerminalProject, strings.Repeat("1", 32))
	// Same-context competing writer reaches the bounded first-frame guard, not a
	// project-wide pre-upgrade refusal. Other IDs remain independent.
	rival, _, err := terminalDial(t, srv, "")
	if err != nil {
		t.Fatal(err)
	}
	defer rival.CloseNow()
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()
	_ = rival.Write(ctx, websocket.MessageText, attachAuth(id))
	if _, _, err = rival.Read(ctx); err == nil || calls.Load() != 2 {
		t.Fatal("two writers admitted")
	}
	if exactAction(t, s, srv.URL, webTerminalProject, id, map[string]any{"action": "retain", "seconds": 7200}) != 200 {
		t.Fatal("Keep")
	}
	keep := exactMetadata(t, s, srv.URL, webTerminalProject, id).RetainUntil
	if exactAction(t, s, srv.URL, webTerminalProject, id, map[string]any{"action": "hide", "attachment_id": oldAttachment}) != 200 || exactMetadata(t, s, srv.URL, webTerminalProject, id).RetainUntil != keep {
		t.Fatal("Hide overwrote Keep")
	}
	first.CloseNow()
	until := time.Now().Add(time.Second)
	for exactMetadata(t, s, srv.URL, webTerminalProject, id).Attached && time.Now().Before(until) {
		time.Sleep(time.Millisecond)
	}
	c, _, err := terminalDial(t, srv, "")
	if err != nil {
		t.Fatal(err)
	}
	defer c.CloseNow()
	ctx2, done := context.WithTimeout(t.Context(), 2*time.Second)
	defer done()
	_ = c.Write(ctx2, websocket.MessageText, attachAuth(id))
	_, body, err := c.Read(ctx2)
	var locator map[string]string
	if err != nil || json.Unmarshal(body, &locator) != nil {
		t.Fatal("attach")
	}
	if _, _, err = c.Read(ctx2); err != nil {
		t.Fatal(err)
	}
	for _, action := range []string{"hide", "return"} {
		if exactAction(t, s, srv.URL, webTerminalProject, id, map[string]any{"action": action, "attachment_id": oldAttachment}) != 409 {
			t.Fatal("stale window changed successor", action)
		}
	}
	if exactAction(t, s, srv.URL, webTerminalProject, id, map[string]any{"action": "return"}) != 409 {
		t.Fatal("active Return without generation")
	}
	if exactMetadata(t, s, srv.URL, webTerminalProject, id).RetainUntil != keep {
		t.Fatal("attach renewed retention")
	}
	if exactAction(t, s, srv.URL, webTerminalProject, id, map[string]any{"action": "return", "attachment_id": locator["attachment_id"]}) != 200 {
		t.Fatal("current Return")
	}
	s.terminalMu.Lock()
	s.terminals[id].hardUntil = time.Now().Add(time.Minute)
	s.terminalMu.Unlock()
	if exactAction(t, s, srv.URL, webTerminalProject, id, map[string]any{"action": "retain", "seconds": 7200}) != 200 {
		t.Fatal("Keep")
	}
	m := exactMetadata(t, s, srv.URL, webTerminalProject, id)
	if m.RetainUntil != m.HardUntil || m.EffectiveUntil != m.HardUntil {
		t.Fatal("not capped", m)
	}
}
func TestTerminalNamesActionsAndOldClientRefusal(t *testing.T) {
	s, srv, calls, _ := terminalWebFixture(t, 0)
	_, id, _ := openManaged(t, srv, webTerminalProject, strings.Repeat("1", 32))
	for _, name := range []string{strings.Repeat("界", 81), "newline\n", "tab\t", "\u202eoverride", "\x7f"} {
		if exactAction(t, s, srv.URL, webTerminalProject, id, map[string]any{"action": "rename", "name": name}) != 400 {
			t.Fatal("invalid label", name)
		}
	}
	for _, name := range []string{"", strings.Repeat("界", 80), "ordinary $shell 'text'"} {
		if exactAction(t, s, srv.URL, webTerminalProject, id, map[string]any{"action": "rename", "name": name}) != 200 {
			t.Fatal("valid label")
		}
	}
	for _, body := range []any{map[string]any{"action": "rename"}, map[string]any{"action": "end", "name": "bad"}, map[string]any{"action": "retain", "seconds": 3600}, map[string]any{"action": "end", "id": id}, map[string]any{"action": "hide"}} {
		if exactAction(t, s, srv.URL, webTerminalProject, id, body) != 400 {
			t.Fatal("malformed action")
		}
	}
	for _, id := range []string{"latest", "pending", strings.Repeat("A", 32), "01"} {
		if terminalAPI(t, s, srv.URL, "GET", "/api/environments/"+webTerminalProject+"/terminal-sessions/"+id, nil).Code != 400 {
			t.Fatal("ID alias")
		}
	}
	if terminalAPI(t, s, srv.URL, "GET", "/api/environments/"+webTerminalProject+"/terminal-session", nil).Code != 410 {
		t.Fatal("old client selected a session")
	}
	if calls.Load() != 2 {
		t.Fatal("metadata reached native")
	}
}
func TestTerminalCorrelationLostLocatorAndCapacityReservations(t *testing.T) {
	s, srv, calls, _ := terminalWebFixture(t, 0, "cleanup_unconfirmed")
	_, id, _ := openManaged(t, srv, webTerminalProject, strings.Repeat("1", 32))
	w := terminalAPI(t, s, srv.URL, "GET", "/api/environments/"+webTerminalProject+"/terminal-attempts/"+strings.Repeat("1", 32), nil)
	if !strings.Contains(w.Body.String(), id) {
		t.Fatal("exact attempt missing")
	}
	if strings.Contains(terminalAPI(t, s, srv.URL, "GET", "/api/environments/"+webTerminalProject+"/terminal-attempts/"+strings.Repeat("f", 32), nil).Body.String(), id) {
		t.Fatal("selected newest instead of attempt")
	}
	if exactAction(t, s, srv.URL, webTerminalProject, id, map[string]any{"action": "end"}) != 200 {
		t.Fatal("End")
	}
	waitTerminalState(t, s, srv.URL, webTerminalProject, id, "unconfirmed")
	// The uncertain ID remains reserved, but does not monopolize its whole project.
	_, other, _ := openManaged(t, srv, webTerminalProject, strings.Repeat("2", 32))
	if other == id {
		t.Fatal("replacement")
	}
	s.terminalMu.Lock()
	for len(s.terminals) < 64 {
		key := fmt.Sprintf("%032x", len(s.terminals)+100)
		s.terminals[key] = s.terminals[id]
	}
	s.terminalMu.Unlock()
	c, _, err := terminalDial(t, srv, "")
	if err != nil {
		t.Fatal(err)
	}
	defer c.CloseNow()
	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()
	_ = c.Write(ctx, websocket.MessageText, []byte(terminalAuth))
	if _, _, err = c.Read(ctx); err == nil || calls.Load() != 4 {
		t.Fatal("capacity evicted a reservation")
	}
}
func TestTerminalStopAcrossContextsAndLogoutRemainingProject(t *testing.T) {
	s, srv, _, _ := terminalWebFixture(t, 0)
	addSecondTerminalProject(t, s)
	_, one, _ := openManaged(t, srv, webTerminalProject, strings.Repeat("1", 32))
	_, otherProject, _ := openManaged(t, srv, secondTerminalProject, strings.Repeat("3", 32))
	grant, err := s.Store.Grant(t.Context(), "session-alice", 1)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Store.CreateGrantedSession(t.Context(), "second-context", 1, "second-csrf", grant); err != nil {
		t.Fatal(err)
	}
	c, _, err := websocket.Dial(t.Context(), "wss"+strings.TrimPrefix(srv.URL, "https")+"/-/soda/api/environments/"+webTerminalProject+"/terminal", &websocket.DialOptions{HTTPClient: srv.Client(), HTTPHeader: http.Header{"Origin": {srv.URL}, "Cookie": {"__Secure-sodaspaces-session=second-context"}}})
	if err != nil {
		t.Fatal(err)
	}
	defer c.CloseNow()
	ctx, done := context.WithTimeout(t.Context(), 2*time.Second)
	defer done()
	_ = c.Write(ctx, websocket.MessageText, []byte(strings.ReplaceAll(terminalAuth, "csrf-alice", "second-csrf")))
	_, body, err := c.Read(ctx)
	var locator map[string]string
	if err != nil || json.Unmarshal(body, &locator) != nil {
		t.Fatal("second context create")
	}
	if _, _, err = c.Read(ctx); err != nil {
		t.Fatal(err)
	}
	if exactMetadata(t, s, srv.URL, webTerminalProject, locator["id"]) != nil {
		t.Fatal("context disclosed another window's terminal")
	}
	if exactAction(t, s, srv.URL, webTerminalProject, locator["id"], map[string]any{"action": "rename", "name": "stolen"}) != 404 {
		t.Fatal("context rename crossed boundary")
	}
	w := terminalAPI(t, s, srv.URL, "POST", "/api/environments/"+webTerminalProject+"/lifecycle", map[string]any{"action": "stop", "confirm_stop": true})
	if w.Code != 200 {
		t.Fatal("stop")
	}
	waitTerminalState(t, s, srv.URL, webTerminalProject, one, "ended")
	if _, _, err = c.Read(ctx); err == nil {
		t.Fatal("Stop spared second context")
	}
	if !exactMetadata(t, s, srv.URL, secondTerminalProject, otherProject).Ready {
		t.Fatal("Stop crossed project")
	}
	w = terminalAPI(t, s, srv.URL, "POST", "/api/session/logout", map[string]any{})
	if w.Code != 204 {
		t.Fatal("logout")
	}
	s.terminalMu.Lock()
	entry := s.terminals[otherProject]
	if entry != nil && entry.ctx.Err() == nil {
		t.Error("logout retained context")
	}
	s.terminalMu.Unlock()
}

func TestTerminalReceiptCapacityDoesNotRetainNativeSlotOrFabricateProof(t *testing.T) {
	s, srv, _, _ := terminalWebFixture(t, 0)
	_, id, _ := openManaged(t, srv, webTerminalProject, strings.Repeat("1", 32))
	s.terminalMu.Lock()
	entry := s.terminals[id]
	s.terminalReceipts = make(map[string]terminalReceipt)
	for i := 0; i < 128; i++ {
		key := fmt.Sprintf("%032x", i+100)
		s.terminalReceipts[key] = terminalReceipt{entry.terminalBinding, entry.view(), time.Now().Add(time.Minute)}
	}
	s.terminalMu.Unlock()
	if exactAction(t, s, srv.URL, webTerminalProject, id, map[string]any{"action": "end"}) != 200 {
		t.Fatal("end")
	}
	until := time.Now().Add(2 * time.Second)
	for time.Now().Before(until) {
		s.terminalMu.Lock()
		count := len(s.terminals)
		s.terminalMu.Unlock()
		if count == 0 {
			break
		}
		time.Sleep(time.Millisecond)
	}
	if exactMetadata(t, s, srv.URL, webTerminalProject, id) != nil {
		t.Fatal("full receipt history invented confirmation")
	}
	s.terminalMu.Lock()
	defer s.terminalMu.Unlock()
	if len(s.terminals) != 0 || len(s.terminalReceipts) != 128 {
		t.Fatal("receipt capacity retained slot or evicted evidence")
	}
}

func TestTerminalPendingTransportLimitAndStopCancellation(t *testing.T) {
	s, srv, calls, _ := terminalWebFixture(t, 0)
	s.terminalMu.Lock()
	s.terminalPeers = make(map[*http.Request]*terminalPeer)
	for range 128 {
		s.terminalPeers[new(http.Request)] = &terminalPeer{cancel: func() {}}
	}
	s.terminalMu.Unlock()
	c, response, err := terminalDial(t, srv, "")
	if c != nil {
		c.CloseNow()
	}
	if err == nil || response == nil || response.StatusCode != 409 {
		t.Fatal("transport limit")
	}
	s.terminalMu.Lock()
	clear(s.terminalPeers)
	s.terminalMu.Unlock()
	c, _, err = terminalDial(t, srv, "")
	if err != nil {
		t.Fatal(err)
	}
	defer c.CloseNow()
	w := terminalAPI(t, s, srv.URL, "POST", "/api/environments/"+webTerminalProject+"/lifecycle", map[string]any{"action": "stop", "confirm_stop": true})
	if w.Code != 200 {
		t.Fatal("Stop")
	}
	ctx, done := context.WithTimeout(t.Context(), time.Second)
	defer done()
	_ = c.Write(ctx, websocket.MessageText, []byte(terminalAuth))
	if _, _, err = c.Read(ctx); err == nil || calls.Load() != 0 {
		t.Fatal("Stop allowed late pending creation")
	}
}

func TestTerminalPausedNativeDialCannotBlockLogout(t *testing.T) {
	s, srv, _, _ := terminalWebFixture(t, 0)
	entered := make(chan struct{})
	s.Host.HTTP = &http.Client{Transport: &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		close(entered)
		<-ctx.Done()
		return nil, ctx.Err()
	}}}
	c, _, err := terminalDial(t, srv, "")
	if err != nil {
		t.Fatal(err)
	}
	defer c.CloseNow()
	ctx, done := context.WithTimeout(t.Context(), 2*time.Second)
	defer done()
	_ = c.Write(ctx, websocket.MessageText, []byte(terminalAuth))
	select {
	case <-entered:
	case <-ctx.Done():
		t.Fatal("native dial not entered")
	}
	start := time.Now()
	w := terminalAPI(t, s, srv.URL, "POST", "/api/session/logout", map[string]any{})
	if w.Code != 204 || time.Since(start) > time.Second {
		t.Fatal("native IO held admission lock")
	}
	s.terminalMu.Lock()
	defer s.terminalMu.Unlock()
	if len(s.terminals) != 1 {
		t.Fatal("uncertain dispatch lost reservation")
	}
	for _, entry := range s.terminals {
		if entry.ctx.Err() == nil {
			t.Fatal("logout did not cancel owner")
		}
	}
}

func TestTerminalSimultaneousCorrelationAdmitsOneOwner(t *testing.T) {
	_, srv, calls, _ := terminalWebFixture(t, 0)
	peers := make([]*websocket.Conn, 2)
	for i := range peers {
		c, _, err := terminalDial(t, srv, "")
		if err != nil {
			t.Fatal(err)
		}
		peers[i] = c
		defer c.CloseNow()
	}
	var wg sync.WaitGroup
	start := make(chan struct{})
	for _, c := range peers {
		wg.Go(func() {
			<-start
			ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
			defer cancel()
			_ = c.Write(ctx, websocket.MessageText, []byte(terminalAuth))
			for i := 0; i < 2; i++ {
				_, _, _ = c.Read(ctx)
			}
		})
	}
	close(start)
	wg.Wait()
	if calls.Load() != 2 {
		t.Fatal("duplicate native ownership", calls.Load())
	}
}
