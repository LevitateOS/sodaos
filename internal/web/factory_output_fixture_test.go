package web

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	extensions "forgejo.org/extension-sdk"
	"github.com/coder/websocket"
	"github.com/levitateos/sodaos/internal/store"
)

type factoryOutputScript struct {
	calls   atomic.Int32
	offsets chan float64
	serve   func(body map[string]any) (int, string)
}

func factoryOutputProxyFixture(t *testing.T, script *factoryOutputScript) (*httptest.Server, *atomic.Value, string, *Server) {
	t.Helper()
	s := apiTestServer(t)
	if err := s.Store.CreateProject(t.Context(), store.Project{ID: webTerminalProject, Name: "factory", RepositoryID: 7, OwnerID: 1, Repository: "alice/factory"}); err != nil {
		t.Fatal(err)
	}
	if err := s.Store.MarkReady(t.Context(), webTerminalProject, "10.89.0.2"); err != nil {
		t.Fatal(err)
	}
	s.Host.HTTP = &http.Client{Transport: roundTrip(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/factory-output" {
			return factoryViewsHostResponse(500, "unexpected helper call"), nil
		}
		script.calls.Add(1)
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if offset, ok := body["offset"].(float64); ok {
			select {
			case script.offsets <- offset:
			default:
			}
		}
		status, response := script.serve(body)
		return factoryViewsHostResponse(status, response), nil
	})}
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
	return proxy, permission, s.Config.ForgejoURL, s
}

func dialFactoryOutput(t *testing.T, proxy *httptest.Server, origin, runID string, handshake map[string]any) *websocket.Conn {
	t.Helper()
	wsURL := "wss" + strings.TrimPrefix(proxy.URL, "https") + "/factory/runs/" + runID + "/output"
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
	ctx, done := context.WithTimeout(t.Context(), 5*time.Second)
	defer done()
	data, _ := json.Marshal(handshake)
	if err := conn.Write(ctx, websocket.MessageText, data); err != nil {
		t.Fatal(err)
	}
	return conn
}

func readFactoryFrame(t *testing.T, conn *websocket.Conn) map[string]any {
	t.Helper()
	ctx, done := context.WithTimeout(t.Context(), 10*time.Second)
	defer done()
	_, body, err := conn.Read(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var frame map[string]any
	if err := json.Unmarshal(body, &frame); err != nil {
		t.Fatal(err)
	}
	return frame
}

func factoryOutputHandshakeFor(runID string) map[string]any {
	return map[string]any{"run_id": runID, "repository_id": "7", "session_generation": nativeTerminalGeneration, "cursor": 0}
}
