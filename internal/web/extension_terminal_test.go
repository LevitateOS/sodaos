package web

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	extensions "forgejo.org/extension-sdk"
	"github.com/coder/websocket"
)

const nativeTerminalGeneration = "test-session-generation"

func nativeTerminalContext() string {
	data, _ := json.Marshal(extensions.Authority{
		ExtensionID: "soda", InstanceID: "test-instance", SessionGeneration: nativeTerminalGeneration,
		Contribution: extensions.Contribution{Kind: "panel", ID: "workspace", Scope: "panel", Action: "get"},
		Actor:        extensions.Actor{ID: "1", Username: "soda-tester"},
	})
	return string(data)
}

func nativeTerminalHeaders(method, origin string) http.Header {
	context := nativeTerminalContext()
	if method == http.MethodPost {
		context = strings.Replace(context, `"action":"get"`, `"action":"post"`, 1)
	}
	return http.Header{
		extensions.ContextHeader:           {context},
		extensions.AdmissionHeader:         {"test-admission"},
		extensions.SessionGenerationHeader: {nativeTerminalGeneration},
		"Origin":                           {origin},
		"Sec-Fetch-Site":                   {"same-origin"},
	}
}

func nativeTerminalFixtureDir(t *testing.T) string {
	t.Helper()
	root := filepath.Join("..", "..", ".artifacts")
	if err := os.MkdirAll(root, 0o700); err != nil {
		t.Fatal(err)
	}
	dir, err := os.MkdirTemp(root, "native-terminal-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	return dir
}

func nativeTerminalCallback(t *testing.T, permission *atomic.Value) string {
	t.Helper()
	dir := nativeTerminalFixtureDir(t)
	path := filepath.Join(dir, "callback.sock")
	listener, err := net.Listen("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != extensions.NativeCallbackPath || r.Header.Get(extensions.AdmissionHeader) != "test-admission" {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		var in extensions.CallbackRequest
		if json.NewDecoder(r.Body).Decode(&in) != nil || in.Authority.SessionGeneration != nativeTerminalGeneration {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		switch in.Operation {
		case extensions.OperationCurrentActor:
			_ = json.NewEncoder(w).Encode(extensions.CallbackResponse{Actor: &extensions.Actor{ID: "1", Username: "soda-tester"}})
		case extensions.OperationRepository:
			_ = json.NewEncoder(w).Encode(extensions.CallbackResponse{Repository: &extensions.Repository{ID: in.RepositoryID, Owner: "soda-tester", Name: "repo", Permission: permission.Load().(string)}})
		default:
			w.WriteHeader(http.StatusForbidden)
		}
	}))
	server.Listener.Close()
	server.Listener = listener
	server.Start()
	t.Cleanup(server.Close)
	return path
}

func nativeTerminalProxyFixture(t *testing.T) (*httptest.Server, *terminalNativeFixture, *atomic.Value, string) {
	t.Helper()
	s, _, host, _ := terminalWebFixture(t, 0)
	permission := &atomic.Value{}
	permission.Store("write")
	t.Setenv(extensions.ServiceCallbackEnv, nativeTerminalCallback(t, permission))
	dir := nativeTerminalFixtureDir(t)
	socket := filepath.Join(dir, "service.sock")
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	private := httptest.NewUnstartedServer(s.ExtensionHandler())
	private.Listener.Close()
	private.Listener = listener
	private.Start()
	t.Cleanup(private.Close)
	application, closeTransport := Extension(socket)
	t.Cleanup(closeTransport)
	proxy := httptest.NewTLSServer(application.HTTP)
	t.Cleanup(proxy.Close)
	return proxy, host, permission, s.Config.ForgejoURL
}

func nativeTerminalRequest(t *testing.T, proxy *httptest.Server, origin, method, path, generation string, body []byte) *http.Response {
	t.Helper()
	r, err := http.NewRequestWithContext(t.Context(), method, proxy.URL+path, bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	r.Header = nativeTerminalHeaders(method, origin)
	r.Header.Set(extensions.SessionGenerationHeader, generation)
	if body != nil {
		r.Header.Set("Content-Type", "application/json")
	}
	response, err := proxy.Client().Do(r)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { response.Body.Close() })
	return response
}

func TestNativeTerminalReserveCreateAttachAndAuthority(t *testing.T) {
	proxy, host, permission, origin := nativeTerminalProxyFixture(t)
	host.echo = true
	base := "/environments/" + webTerminalProject
	body := []byte(`{"cols":80,"rows":24,"name":"Shell"}`)
	if response := nativeTerminalRequest(t, proxy, origin, http.MethodPost, base+"/terminal-sessions", "stale-generation", body); response.StatusCode != http.StatusForbidden {
		t.Fatalf("stale reservation returned %d", response.StatusCode)
	}
	if host.count("reserve") != 0 {
		t.Fatal("stale generation reached native reservation")
	}
	if response := nativeTerminalRequest(t, proxy, "https://other.invalid", http.MethodPost, base+"/terminal-sessions", nativeTerminalGeneration, body); response.StatusCode != http.StatusForbidden {
		t.Fatalf("cross-origin reservation returned %d", response.StatusCode)
	}
	if host.count("reserve") != 0 {
		t.Fatal("cross-origin request reached native reservation")
	}
	response := nativeTerminalRequest(t, proxy, origin, http.MethodPost, base+"/terminal-sessions", nativeTerminalGeneration, body)
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("reservation returned %d", response.StatusCode)
	}
	var reserved struct {
		ID string `json:"id"`
	}
	if json.NewDecoder(response.Body).Decode(&reserved) != nil || reserved.ID == "" {
		t.Fatal("reservation omitted ID")
	}
	wsURL := "wss" + strings.TrimPrefix(proxy.URL, "https") + base + "/terminal"
	connect := func(generation string) *websocket.Conn {
		t.Helper()
		headers := nativeTerminalHeaders(http.MethodGet, origin)
		headers.Del(extensions.SessionGenerationHeader)
		conn, response, err := websocket.Dial(t.Context(), wsURL, &websocket.DialOptions{HTTPClient: proxy.Client(), HTTPHeader: headers})
		if err != nil {
			if response != nil {
				body, _ := io.ReadAll(response.Body)
				t.Fatalf("%v: %s", err, body)
			}
			t.Fatal(err)
		}
		t.Cleanup(func() { conn.CloseNow() })
		ctx, done := context.WithTimeout(t.Context(), 3*time.Second)
		defer done()
		in := map[string]any{"action": "create", "id": reserved.ID, "name": "Shell", "expected_user_id": "1", "repository_id": "7", "session_generation": generation, "cols": 80, "rows": 24}
		data, _ := json.Marshal(in)
		if err := conn.Write(ctx, websocket.MessageText, data); err != nil {
			t.Fatal(err)
		}
		return conn
	}
	stale := connect("stale-generation")
	ctx, done := context.WithTimeout(t.Context(), 3*time.Second)
	if _, _, err := stale.Read(ctx); err == nil || host.count("create") != 0 {
		t.Fatal("stale WebSocket handshake created a shell")
	}
	done()
	conn := connect(nativeTerminalGeneration)
	ctx, done = context.WithTimeout(t.Context(), 3*time.Second)
	_, ready, err := conn.Read(ctx)
	done()
	if err != nil || string(ready) != `{"type":"ready"}` || host.count("create") != 1 {
		t.Fatalf("native terminal did not attach: %v %s", err, ready)
	}
	ctx, done = context.WithTimeout(t.Context(), 3*time.Second)
	if err := conn.Write(ctx, websocket.MessageText, []byte(`{"type":"input","data":"YQ=="}`)); err != nil {
		t.Fatal(err)
	}
	_, output, err := conn.Read(ctx)
	done()
	if err != nil || string(output) != `{"type":"output","data":"YQ=="}` {
		t.Fatalf("terminal input/output did not cross the bridge: %v %s", err, output)
	}
	permission.Store("read")
	denied := nativeTerminalRequest(t, proxy, origin, http.MethodGet, base+"/terminal-sessions/"+reserved.ID, nativeTerminalGeneration, nil)
	if denied.StatusCode != http.StatusForbidden {
		t.Fatalf("lost repository write still inspected terminal: %d", denied.StatusCode)
	}
	permission.Store("write")
	ended := nativeTerminalRequest(t, proxy, origin, http.MethodPost, base+"/terminal-sessions/"+reserved.ID, nativeTerminalGeneration, []byte(`{"action":"end"}`))
	if ended.StatusCode != http.StatusOK || host.count("end") != 1 {
		t.Fatalf("native terminal End returned %d", ended.StatusCode)
	}
}
