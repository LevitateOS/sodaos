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
	"github.com/levitateos/sodaos/internal/config"
	"github.com/levitateos/sodaos/internal/forgejo"
	"github.com/levitateos/sodaos/internal/host"
	"github.com/levitateos/sodaos/internal/store"
)

const webTerminalProject = "p0123456789abcdef01234567"
const secondTerminalProject = "p1123456789abcdef01234567"
const reservedTerminalID = "0123456789abcdef0123456789abcdef"
const terminalAuth = `{"action":"create","id":"0123456789abcdef0123456789abcdef","expected_user_id":"1","repository_id":"7","csrf_token":"csrf-alice","cols":80,"rows":24}`

type nativeCreationPermit struct {
	scope   string
	expires time.Time
}
type terminalNativeFixture struct {
	atomic.Int32 // Successful native creates/attachments, not metadata operations
	created      int
	reservations map[string]nativeCreationPermit
	mu           sync.Mutex
	rows         map[string]host.TerminalState
	writers      map[string]*websocket.Conn
	requests     []host.TerminalRequest
	fail         map[string]bool
}

func (f *terminalNativeFixture) count(action string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	if action == "create" {
		return f.created
	}
	n := 0
	for _, in := range f.requests {
		if in.Action == action {
			n++
		}
	}
	return n
}

func terminalWebFixture(t *testing.T, providerStatus int, cleanupReason ...string) (*Server, *httptest.Server, *terminalNativeFixture, <-chan struct{}) {
	t.Helper()
	s := grantedTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if providerStatus != 0 {
			w.WriteHeader(providerStatus)
			return
		}
		switch r.URL.Path {
		case "/api/v1/user":
			fmt.Fprint(w, `{"id":1,"login":"renamed-alice"}`)
		case "/api/v1/repositories/8":
			fmt.Fprint(w, `{"id":8,"name":"second","full_name":"alice/second","owner":{"id":1,"login":"alice"}}`)
		case "/api/v1/repositories/7":
			fmt.Fprint(w, `{"id":7,"name":"repo","full_name":"alice/repo","owner":{"id":1,"login":"alice"}}`)
		default:
			t.Error("unexpected authority request")
			w.WriteHeader(500)
		}
	})
	if err := s.Store.CreateProject(t.Context(), store.Project{ID: webTerminalProject, RepositoryID: 7, OwnerID: 1}); err != nil {
		t.Fatal(err)
	}
	if err := s.Store.MarkReady(t.Context(), webTerminalProject, "10.0.0.2"); err != nil {
		t.Fatal(err)
	}
	if err := s.Store.Join(t.Context(), webTerminalProject, 1, "original-alice"); err != nil {
		t.Fatal(err)
	}
	calls := &terminalNativeFixture{rows: make(map[string]host.TerminalState), reservations: make(map[string]nativeCreationPermit), writers: make(map[string]*websocket.Conn), fail: make(map[string]bool)}
	if len(cleanupReason) != 0 {
		calls.fail["end"] = true
	}
	v, _ := s.Store.Session(t.Context(), "session-alice")
	// Transport tests start after a native reservation; endpoint coverage below
	// also exercises the actual server-issued allocation path.
	initial := webTerminalProject + "/" + reservedTerminalID
	calls.reservations[initial] = nativeCreationPermit{terminalCreationScope(v), time.Now().Add(time.Minute)}
	calls.rows[initial] = host.TerminalState{ID: reservedTerminalID, CreatedAt: time.Now().Unix(), State: "opening"}
	closed := make(chan struct{}, 64)
	helper := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/inspect" {
			var in host.Create
			_ = json.NewDecoder(r.Body).Decode(&in)
			_ = json.NewEncoder(w).Encode(host.Environment{ID: in.ID, Running: true})
			return
		}
		if r.URL.Path == "/lifecycle" {
			var in host.Lifecycle
			_ = json.NewDecoder(r.Body).Decode(&in)
			fmt.Fprintf(w, `{"environment":{"id":%q,"running":false},"boot_enabled":false}`, in.Project)
			return
		}
		if r.URL.Path != "/terminal" {
			http.NotFound(w, r)
			return
		}
		c, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer c.CloseNow()
		_, body, err := c.Read(r.Context())
		if err != nil {
			return
		}
		var in host.TerminalRequest
		if json.Unmarshal(body, &in) != nil || (in.Project != webTerminalProject && in.Project != secondTerminalProject) || in.Login != "original-alice" || in.Identity != 1 {
			t.Error("untrusted native dispatch")
		}
		if in.Expires > time.Now().Add(12*time.Hour).Unix() {
			t.Error("unbounded native request")
		}
		key := in.Project + "/" + in.ID
		calls.mu.Lock()
		calls.requests = append(calls.requests, in)
		items := []host.TerminalState{}
		var end *websocket.Conn
		valid := true
		switch in.Action {
		case "reserve":
			for target, permit := range calls.reservations {
				if !time.Now().Before(permit.expires) {
					delete(calls.reservations, target)
					delete(calls.rows, target)
				}
			}
			if _, exists := calls.rows[key]; exists || len(calls.reservations) >= 128 {
				valid = false
				break
			}
			calls.reservations[key] = nativeCreationPermit{in.Scope, time.Now().Add(2 * time.Minute)}
			calls.rows[key] = host.TerminalState{ID: in.ID, Name: in.Name, CreatedAt: time.Now().Unix(), State: "opening"}
		case "create":
			permit, exists := calls.reservations[key]
			if !exists || !time.Now().Before(permit.expires) || permit.scope != in.Scope {
				valid = false
				break
			}
			delete(calls.reservations, key)
			calls.Add(1)
			calls.created++
			calls.rows[key] = host.TerminalState{ID: in.ID, Name: in.Name, CreatedAt: time.Now().Unix(), Ready: true, State: "ready"}
		case "attach":
			if row, exists := calls.rows[key]; !exists || !row.Ready || calls.writers[key] != nil {
				valid = false
				break
			}
			calls.Add(1)
			calls.writers[key] = c
		case "end":
			if !calls.fail["end"] {
				delete(calls.rows, key)
				delete(calls.reservations, key)
				end = calls.writers[key]
			}
		case "rename":
			row, exists := calls.rows[key]
			valid = exists
			if exists {
				row.Name = in.Name
				calls.rows[key] = row
			}
		case "inspect", "list":
		default:
			valid = false
		}
		for target, row := range calls.rows {
			permit, pending := calls.reservations[target]
			if pending && !time.Now().Before(permit.expires) {
				row.State = "ended"
			}
			if target == key || in.Action == "list" && !pending && strings.HasPrefix(target, in.Project+"/") {
				row.Attached = calls.writers[target] != nil
				items = append(items, row)
			}
		}
		failed := calls.fail[in.Action]
		calls.mu.Unlock()
		if end != nil {
			_ = end.CloseNow()
		}
		if !valid || failed {
			return
		}
		if in.Action != "attach" {
			body, _ := json.Marshal(host.TerminalFrame{Type: "metadata", Terminals: &items})
			_ = c.Write(r.Context(), websocket.MessageText, body)
			return
		}
		defer func() {
			calls.mu.Lock()
			if calls.writers[key] == c {
				delete(calls.writers, key)
			}
			calls.mu.Unlock()
			closed <- struct{}{}
		}()
		_ = c.Write(r.Context(), websocket.MessageText, []byte(`{"type":"ready"}`))
		for {
			_, body, err := c.Read(r.Context())
			if err != nil {
				return
			}
			var f host.TerminalFrame
			_ = json.Unmarshal(body, &f)
			if f.Type == "close" {
				body, _ := json.Marshal(host.TerminalFrame{Type: "closed", Reason: "disconnected"})
				_ = c.Write(r.Context(), websocket.MessageText, body)
				return
			}
		}
	}))
	t.Cleanup(helper.Close)
	s.Host.HTTP = &http.Client{Transport: &http.Transport{DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, network, helper.Listener.Addr().String())
	}}}
	server := httptest.NewTLSServer(s)
	s.Config.ForgejoURL = server.URL
	t.Cleanup(server.Close)
	t.Cleanup(s.CloseTerminals)
	return s, server, calls, closed
}
func terminalDial(t *testing.T, server *httptest.Server, query string) (*websocket.Conn, *http.Response, error) {
	t.Helper()
	return websocket.Dial(t.Context(), "wss"+strings.TrimPrefix(server.URL, "https")+config.SodaPath+"/api/environments/"+webTerminalProject+"/terminal"+query, &websocket.DialOptions{HTTPClient: server.Client(), HTTPHeader: http.Header{"Origin": {server.URL}, "Cookie": {"__Secure-sodaspaces-session=session-alice"}, "Sec-Fetch-Site": {"same-origin"}}})
}
func terminalReady(t *testing.T, c *websocket.Conn) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()
	if c.Write(ctx, websocket.MessageText, []byte(terminalAuth)) != nil {
		t.Fatal("auth write failed")
	}
	_, b, err := c.Read(ctx)
	if err != nil || string(b) != `{"type":"ready"}` {
		t.Fatal("terminal not ready", err)
	}
}
func TestBrowserTerminalDenialsNeverReachNative(t *testing.T) {
	for _, body := range []string{`{}`, strings.Replace(terminalAuth, `"1"`, `"2"`, 1), strings.Replace(terminalAuth, `"7"`, `"8"`, 1), strings.Replace(terminalAuth, "csrf-alice", "wrong", 1), strings.Replace(terminalAuth, `"cols":80`, `"cols":1`, 1), strings.Replace(terminalAuth, `"cols":80`, `"cols":80,"cols":81`, 1), strings.Replace(terminalAuth, `"cols":80`, `"command":"id","cols":80`, 1)} {
		t.Run(body, func(t *testing.T) {
			_, srv, calls, _ := terminalWebFixture(t, 0)
			c, _, err := terminalDial(t, srv, "")
			if err != nil {
				t.Fatal(err)
			}
			defer c.CloseNow()
			ctx, done := context.WithTimeout(t.Context(), 2*time.Second)
			defer done()
			_ = c.Write(ctx, websocket.MessageText, []byte(body))
			if _, _, err = c.Read(ctx); err == nil {
				t.Fatal("invalid auth accepted")
			}
			if calls.Load() != 0 {
				t.Fatal("denial reached helper")
			}
		})
	}
	for _, status := range []int{403, 404, 503} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			_, srv, calls, _ := terminalWebFixture(t, status)
			c, _, err := terminalDial(t, srv, "")
			if err != nil {
				t.Fatal(err)
			}
			defer c.CloseNow()
			ctx, done := context.WithTimeout(t.Context(), 2*time.Second)
			defer done()
			_ = c.Write(ctx, websocket.MessageText, []byte(terminalAuth))
			if _, _, err = c.Read(ctx); err == nil {
				t.Fatal("degraded launch accepted")
			}
			if calls.Load() != 0 {
				t.Fatal("provider denial reached helper")
			}
		})
	}
}
func TestBrowserTerminalLogoutDuplicateAndOriginalLogin(t *testing.T) {
	s, srv, calls, closed := terminalWebFixture(t, 0)
	c, _, err := terminalDial(t, srv, "")
	if err != nil {
		t.Fatal(err)
	}
	defer c.CloseNow()
	terminalReady(t, c)
	duplicate, _, err := terminalDial(t, srv, "")
	if err != nil {
		t.Fatal(err)
	}
	defer duplicate.CloseNow()
	ctx, done := context.WithTimeout(t.Context(), time.Second)
	defer done()
	_ = duplicate.Write(ctx, websocket.MessageText, []byte(terminalAuth))
	if _, _, err = duplicate.Read(ctx); err == nil {
		t.Fatal("duplicate correlation accepted")
	}
	w := httptest.NewRecorder()
	req := apiTestRequest("POST", "/api/session/logout", "{}", "alice")
	req.Header.Set("Origin", srv.URL)
	s.ServeHTTP(w, req)
	if w.Code != 204 {
		t.Fatal("logout failed", w.Code)
	}
	select {
	case <-closed:
	case <-time.After(2 * time.Second):
		t.Fatal("logout did not close native stream")
	}
	if calls.Load() != 2 {
		t.Fatal("duplicate replayed native creation or attachment")
	}
}
func TestBrowserTerminalPendingLogoutAndShutdown(t *testing.T) {
	for _, shutdown := range []bool{false, true} {
		t.Run(fmt.Sprint(shutdown), func(t *testing.T) {
			s, srv, calls, _ := terminalWebFixture(t, 0)
			c, _, err := terminalDial(t, srv, "")
			if err != nil {
				t.Fatal(err)
			}
			defer c.CloseNow()
			if shutdown {
				s.CloseTerminals()
			} else {
				w := httptest.NewRecorder()
				req := apiTestRequest("POST", "/api/session/logout", "{}", "alice")
				req.Header.Set("Origin", srv.URL)
				s.ServeHTTP(w, req)
				if w.Code != 204 {
					t.Fatal("logout failed")
				}
			}
			ctx, done := context.WithTimeout(t.Context(), time.Second)
			defer done()
			_ = c.Write(ctx, websocket.MessageText, []byte(terminalAuth))
			if _, _, err = c.Read(ctx); err == nil {
				t.Fatal("pending terminal survived invalidation")
			}
			if calls.Load() != 0 {
				t.Fatal("late auth launched")
			}
		})
	}
}
func TestBrowserTerminalRotationAndActiveShutdown(t *testing.T) {
	for _, rotation := range []bool{false, true} {
		t.Run(fmt.Sprint(rotation), func(t *testing.T) {
			s, srv, _, closed := terminalWebFixture(t, 0)
			c, _, err := terminalDial(t, srv, "")
			if err != nil {
				t.Fatal(err)
			}
			defer c.CloseNow()
			terminalReady(t, c)
			if rotation {
				if err := s.Store.BeginOAuth(t.Context(), "terminal-state", store.OAuthLogin{Verifier: "synthetic-verifier"}, "session-alice", ""); err != nil {
					t.Fatal(err)
				}
				r := httptest.NewRequest("GET", config.SodaPath+"/oauth/callback?state=terminal-state", nil)
				r.AddCookie(&http.Cookie{Name: sessionCookie, Value: "session-alice"})
				r.AddCookie(&http.Cookie{Name: oauthCookie, Value: "terminal-state"})
				w := httptest.NewRecorder()
				s.ServeHTTP(w, r)
				if w.Code != 400 {
					t.Fatal("expected consumed but non-authorized OAuth response")
				}
			} else {
				s.CloseTerminals()
			}
			select {
			case <-closed:
			case <-time.After(2 * time.Second):
				t.Fatal("rotation/shutdown retained native stream")
			}
		})
	}
}

func TestBrowserTerminalLogoutDuringFreshAuthorityNeverSpawns(t *testing.T) {
	s, srv, calls, _ := terminalWebFixture(t, 0)
	entered, release := make(chan struct{}), make(chan struct{})
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(entered)
		<-release
		fmt.Fprint(w, `{"id":1,"login":"alice"}`)
	}))
	defer provider.Close()
	defer close(release)
	s.Forgejo = forgejo.New(provider.URL)
	c, _, err := terminalDial(t, srv, "")
	if err != nil {
		t.Fatal(err)
	}
	defer c.CloseNow()
	ctx, done := context.WithTimeout(t.Context(), 2*time.Second)
	defer done()
	if c.Write(ctx, websocket.MessageText, []byte(terminalAuth)) != nil {
		t.Fatal("auth write failed")
	}
	select {
	case <-entered:
	case <-ctx.Done():
		t.Fatal("authority was not consulted")
	}
	w := httptest.NewRecorder()
	req := apiTestRequest("POST", "/api/session/logout", "{}", "alice")
	req.Header.Set("Origin", srv.URL)
	s.ServeHTTP(w, req)
	if w.Code != 204 {
		t.Fatal("logout blocked or failed")
	}
	if _, _, err = c.Read(ctx); err == nil {
		t.Fatal("logout did not cancel pending authority")
	}
	if calls.Load() != 0 {
		t.Fatal("logout race launched native session")
	}
}

func TestBrowserTerminalRejectsBrowserHeartbeatsAndBadControls(t *testing.T) {
	for _, body := range []string{`{"type":"heartbeat"}`, `{"type":"input","data":"!"}`, `{"type":"resize","rows":999,"cols":80}`, `{"type":"input","data":"YQ==","command":"id"}`} {
		t.Run(body, func(t *testing.T) {
			_, srv, _, closed := terminalWebFixture(t, 0)
			c, _, err := terminalDial(t, srv, "")
			if err != nil {
				t.Fatal(err)
			}
			defer c.CloseNow()
			terminalReady(t, c)
			ctx, done := context.WithTimeout(t.Context(), 2*time.Second)
			defer done()
			_ = c.Write(ctx, websocket.MessageText, []byte(body))
			select {
			case <-closed:
			case <-ctx.Done():
				t.Fatal("bad control retained native stream")
			}
		})
	}
}

func TestBrowserTerminalPreUpgradeBoundaries(t *testing.T) {
	s, srv, calls, _ := terminalWebFixture(t, 0)
	for _, query := range []string{"?", "?ticket=synthetic"} {
		c, r, err := terminalDial(t, srv, query)
		if c != nil {
			c.CloseNow()
		}
		if err == nil || r.StatusCode != 403 {
			t.Fatal("query accepted")
		}
	}
	for _, change := range []func(*http.Request){func(r *http.Request) { r.Header.Set("Origin", "https://elsewhere.test") }, func(r *http.Request) { r.Header.Set("Sec-Fetch-Site", "cross-site") }, func(r *http.Request) { r.Header.Set("Cookie", "__Secure-sodaspaces-session=session-bob") }, func(r *http.Request) { r.Header.Set("Sec-WebSocket-Protocol", "secret") }} {
		w := httptest.NewRecorder()
		r := apiTestRequest("GET", "/api/environments/"+webTerminalProject+"/terminal", "", "alice")
		r.Header.Set("Origin", srv.URL)
		change(r)
		s.ServeHTTP(w, r)
		if w.Code != 403 {
			t.Fatal("preupgrade boundary", w.Code)
		}
	}
	if calls.Load() != 0 {
		t.Fatal("preupgrade denial reached native")
	}
}
