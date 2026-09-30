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
	"github.com/levitateos/sodaos/internal/host"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
	"github.com/levitateos/sodaos/internal/web/api"
)

const (
	webTerminalProject    = "p0123456789abcdef01234567"
	secondTerminalProject = "p1123456789abcdef01234567"
	reservedTerminalID    = "0123456789abcdef0123456789abcdef"
)

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
	echo         bool
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

func terminalWebFixture(t *testing.T, cleanupReason ...string) (*Server, *httptest.Server, *terminalNativeFixture, <-chan struct{}) {
	t.Helper()
	s := apiTestServer(t)
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
	v := store.Session{User: store.User{ID: 1, Login: "alice"}, ContextID: nativeTerminalGeneration}
	// Transport tests start after a native reservation; endpoint coverage below
	// also exercises the actual server-issued allocation path.
	initial := webTerminalProject + "/" + reservedTerminalID
	calls.reservations[initial] = nativeCreationPermit{api.TerminalCreationScope(v), time.Now().Add(time.Minute)}
	calls.rows[initial] = host.TerminalState{ID: reservedTerminalID, CreatedAt: time.Now().Unix(), State: "opening"}
	closed := make(chan struct{}, 64)
	helper := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/inspect" {
			var in project.Create
			_ = json.NewDecoder(r.Body).Decode(&in)
			_ = json.NewEncoder(w).Encode(project.Environment{ID: in.ID, Running: true})
			return
		}
		if r.URL.Path == "/lifecycle" {
			var in project.Lifecycle
			_ = json.NewDecoder(r.Body).Decode(&in)
			_, _ = fmt.Fprintf(w, `{"environment":{"id":%q,"running":false},"boot_enabled":false}`, in.Project)
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
			if f.Type == "input" && calls.echo {
				output, _ := json.Marshal(host.TerminalFrame{Type: "output", Data: f.Data})
				if c.Write(r.Context(), websocket.MessageText, output) != nil {
					return
				}
			}
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
