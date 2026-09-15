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
	"sync/atomic"
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
func reserveTerminal(t *testing.T, s *Server, origin, project string) string {
	t.Helper()
	w := terminalAPI(t, s, origin, "POST", "/api/environments/"+project+"/terminal-sessions", map[string]any{"cols": 80, "rows": 24})
	var result struct {
		ID string `json:"id"`
	}
	if w.Code != 201 || json.Unmarshal(w.Body.Bytes(), &result) != nil || !BrowserTerminalID.MatchString(result.ID) {
		t.Fatalf("reservation %d %s", w.Code, w.Body.String())
	}
	return result.ID
}
func openManaged(t *testing.T, s *Server, srv *httptest.Server, project string) (*websocket.Conn, string) {
	t.Helper()
	id := reserveTerminal(t, s, srv.URL, project)
	c, _, err := websocket.Dial(t.Context(), "wss"+strings.TrimPrefix(srv.URL, "https")+"/-/soda/api/environments/"+project+"/terminal", &websocket.DialOptions{HTTPClient: srv.Client(), HTTPHeader: http.Header{"Origin": {srv.URL}, "Cookie": {"__Secure-sodaspaces-session=session-alice"}}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.CloseNow() })
	body := strings.Replace(terminalAuth, reservedTerminalID, id, 1)
	if project == secondTerminalProject {
		body = strings.Replace(body, `"repository_id":"7"`, `"repository_id":"8"`, 1)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()
	if err := c.Write(ctx, websocket.MessageText, []byte(body)); err != nil {
		t.Fatal(err)
	}
	_, b, err := c.Read(ctx)
	if err != nil || string(b) != `{"type":"ready"}` {
		t.Fatal("ready", err, string(b))
	}
	return c, id
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
func refusedTerminal(t *testing.T, srv *httptest.Server, body string) {
	t.Helper()
	c, _, err := terminalDial(t, srv, "")
	if err != nil {
		t.Fatal(err)
	}
	defer c.CloseNow()
	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()
	_ = c.Write(ctx, websocket.MessageText, []byte(body))
	if _, _, err := c.Read(ctx); err == nil {
		t.Fatal("invalid/repeated creation accepted")
	}
}

func TestTerminalReservationStartsNothingAndCannotBeReused(t *testing.T) {
	s, srv, native, _ := terminalWebFixture(t, 0)
	id := reserveTerminal(t, s, srv.URL, webTerminalProject)
	if native.Load() != 0 || exactMetadata(t, s, srv.URL, webTerminalProject, id).State != "opening" {
		t.Fatal("reservation started work")
	}
	// Client-supplied IDs, old correlation fields and expired reservations refuse.
	refusedTerminal(t, srv, strings.Replace(terminalAuth, reservedTerminalID, strings.Repeat("f", 32), 1))
	refusedTerminal(t, srv, strings.Replace(terminalAuth, `"id":`, `"request_id":`, 1))
	native.mu.Lock()
	key := webTerminalProject + "/" + id
	old := native.reservations[key]
	old.expires = time.Now().Add(-time.Second)
	native.reservations[key] = old
	native.mu.Unlock()
	refusedTerminal(t, srv, strings.Replace(terminalAuth, reservedTerminalID, id, 1))
	if native.Load() != 0 {
		t.Fatal("refused reservation started native work")
	}
	c, live := openManaged(t, s, srv, webTerminalProject)
	c.CloseNow()
	waitDetached(t, s, srv.URL, live)
	if exactAction(t, s, srv.URL, webTerminalProject, live, map[string]any{"action": "end"}) != 200 {
		t.Fatal("End")
	}
	refusedTerminal(t, srv, strings.Replace(terminalAuth, reservedTerminalID, live, 1))
	if native.count("create") != 1 {
		t.Fatal("old ID authorized another native Create")
	}
}

func TestTerminalIDsIndependentEndAndCollection(t *testing.T) {
	s, srv, native, _ := terminalWebFixture(t, 0)
	addSecondTerminalProject(t, s)
	_, one := openManaged(t, s, srv, webTerminalProject)
	_, two := openManaged(t, s, srv, webTerminalProject)
	_, three := openManaged(t, s, srv, secondTerminalProject)
	if one == two || native.count("create") != 3 {
		t.Fatal("not independent")
	}
	if exactMetadata(t, s, srv.URL, secondTerminalProject, one) != nil {
		t.Fatal("wrong project exposed terminal")
	}
	if exactAction(t, s, srv.URL, webTerminalProject, one, map[string]any{"action": "rename", "name": "Build Ω"}) != 200 {
		t.Fatal("rename")
	}
	if exactMetadata(t, s, srv.URL, webTerminalProject, one).Name != "Build Ω" {
		t.Fatal("name not observed natively")
	}
	w := terminalAPI(t, s, srv.URL, "GET", "/api/spaces", nil)
	var result spacesView
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &result) != nil {
		t.Fatal("collection", w.Code)
	}
	count := 0
	for _, row := range result.Items {
		count += len(row.Terminals)
	}
	if count != 3 {
		t.Fatal("native collection", count)
	}
	if exactAction(t, s, srv.URL, webTerminalProject, one, map[string]any{"action": "end"}) != 200 {
		t.Fatal("End")
	}
	if exactMetadata(t, s, srv.URL, webTerminalProject, one) != nil || exactMetadata(t, s, srv.URL, webTerminalProject, two) == nil || exactMetadata(t, s, srv.URL, secondTerminalProject, three) == nil {
		t.Fatal("End affected another terminal")
	}
}

func TestTerminalRetiredLifetimeActionsAndInvalidNamesRefuse(t *testing.T) {
	s, srv, native, _ := terminalWebFixture(t, 0)
	_, id := openManaged(t, s, srv, webTerminalProject)
	for _, body := range []any{
		map[string]any{"action": "hide", "attachment_id": id}, map[string]any{"action": "return"},
		map[string]any{"action": "retain", "seconds": 7200}, map[string]any{"action": "end", "seconds": 0},
		map[string]any{"action": "rename", "name": "bad\nname"}, map[string]any{"action": "rename", "name": strings.Repeat("x", 81)},
	} {
		if exactAction(t, s, srv.URL, webTerminalProject, id, body) != 400 {
			t.Fatal("invalid action admitted", body)
		}
	}
	if native.count("end") != 0 || native.count("rename") != 0 {
		t.Fatal("invalid action dispatched")
	}
	if terminalAPI(t, s, srv.URL, "GET", "/api/environments/"+webTerminalProject+"/terminal-attempts/"+id, nil).Code != 404 {
		t.Fatal("retired correlation route exists")
	}
}

func TestTerminalReservationsAreBoundedButExpiredOnAdmission(t *testing.T) {
	s, srv, native, _ := terminalWebFixture(t, 0)
	native.mu.Lock()
	original := native.reservations[webTerminalProject+"/"+reservedTerminalID]
	clear(native.reservations)
	clear(native.rows)
	for i := 0; i < 128; i++ {
		native.reservations[fmt.Sprintf("%s/%032x", webTerminalProject, i)] = original
	}
	native.mu.Unlock()
	path := "/api/environments/" + webTerminalProject + "/terminal-sessions"
	if terminalAPI(t, s, srv.URL, "POST", path, map[string]any{"cols": 80, "rows": 24}).Code != 503 {
		t.Fatal("reservation bound bypassed")
	}
	native.mu.Lock()
	for id, v := range native.reservations {
		v.expires = time.Now().Add(-time.Second)
		native.reservations[id] = v
	}
	native.mu.Unlock()
	reserveTerminal(t, s, srv.URL, webTerminalProject)
	native.mu.Lock()
	count := len(native.reservations)
	native.mu.Unlock()
	if count != 1 || native.Load() != 0 {
		t.Fatal("expired reservations retained capacity or started work")
	}
}

func TestTerminalSimultaneousCreateConsumesReservationOnce(t *testing.T) {
	s, srv, native, _ := terminalWebFixture(t, 0)
	id := reserveTerminal(t, s, srv.URL, webTerminalProject)
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c, _, err := terminalDial(t, srv, "")
			if err != nil {
				t.Error(err)
				return
			}
			defer c.CloseNow()
			ctx, cancel := context.WithTimeout(t.Context(), time.Second)
			defer cancel()
			_ = c.Write(ctx, websocket.MessageText, []byte(strings.Replace(terminalAuth, reservedTerminalID, id, 1)))
			_, _, _ = c.Read(ctx)
		}()
	}
	wg.Wait()
	if native.count("create") != 1 {
		t.Fatal("duplicate native creation")
	}
}

func TestTerminalEndRevokesReservationBeforeLateNativeCreate(t *testing.T) {
	s, srv, native, _ := terminalWebFixture(t, 0)
	entered, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	defer once.Do(func() { close(release) })
	var dials atomic.Int32
	dial := s.Host.HTTP.Transport.(*http.Transport).DialContext
	s.Host.HTTP = &http.Client{Transport: &http.Transport{DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
		if dials.Add(1) == 1 {
			close(entered)
			select {
			case <-release:
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
		return dial(ctx, network, address)
	}}}
	c, _, err := terminalDial(t, srv, "")
	if err != nil {
		t.Fatal(err)
	}
	defer c.CloseNow()
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()
	_ = c.Write(ctx, websocket.MessageText, []byte(terminalAuth))
	select {
	case <-entered:
	case <-ctx.Done():
		t.Fatal("Create dial not entered")
	}
	if v := exactMetadata(t, s, srv.URL, webTerminalProject, reservedTerminalID); v == nil || v.State != "opening" {
		t.Fatal("in-flight permission became false absence")
	}
	if exactAction(t, s, srv.URL, webTerminalProject, reservedTerminalID, map[string]any{"action": "end"}) != 200 {
		t.Fatal("End")
	}
	once.Do(func() { close(release) })
	if _, _, err := c.Read(ctx); err == nil {
		t.Fatal("late Create survived End")
	}
	if native.count("create") != 0 || exactMetadata(t, s, srv.URL, webTerminalProject, reservedTerminalID) != nil {
		t.Fatal("late native work resurrected an ended locator")
	}
}

func TestTerminalPausedNativeDialCannotBlockLogout(t *testing.T) {
	s, srv, native, _ := terminalWebFixture(t, 0)
	entered, release := make(chan struct{}), make(chan struct{})
	defer close(release)
	s.Host.HTTP = &http.Client{Transport: &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		close(entered)
		select {
		case <-release:
		case <-ctx.Done():
		}
		return nil, context.Canceled
	}}}
	c, _, err := terminalDial(t, srv, "")
	if err != nil {
		t.Fatal(err)
	}
	defer c.CloseNow()
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()
	_ = c.Write(ctx, websocket.MessageText, []byte(terminalAuth))
	select {
	case <-entered:
	case <-ctx.Done():
		t.Fatal("native dial not entered")
	}
	if terminalAPI(t, s, srv.URL, "POST", "/api/session/logout", map[string]any{}).Code != 204 {
		t.Fatal("logout blocked")
	}
	if _, _, err := c.Read(ctx); err == nil || native.Load() != 0 {
		t.Fatal("stale native dispatch")
	}
	// Native unused allocations expire; logout does not perform native cleanup.
	// The old context cannot consume them through the protected browser endpoint.
}
