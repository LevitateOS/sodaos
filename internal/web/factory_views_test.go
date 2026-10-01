package web

import (
	"context"
	"encoding/json"
	"errors"
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
	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/store"
)

func factoryViewsFixture(t *testing.T, host roundTrip) *Server {
	t.Helper()
	s := apiTestServer(t)
	if err := s.Store.CreateProject(t.Context(), store.Project{ID: webTerminalProject, Name: "factory", RepositoryID: 7, OwnerID: 1, Repository: "alice/factory"}); err != nil {
		t.Fatal(err)
	}
	if err := s.Store.MarkReady(t.Context(), webTerminalProject, "10.89.0.2"); err != nil {
		t.Fatal(err)
	}
	s.Host.HTTP = &http.Client{Transport: host}
	return s
}

func factoryViewsRun(t *testing.T, s *Server) factory.Run {
	t.Helper()
	now := time.Now()
	r := factory.Run{ID: factory.NewID(), ProjectID: webTerminalProject, Role: "coder", InputSHA: strings.Repeat("a", 40), Started: now, Deadline: now.Add(time.Hour), Image: "sha256:" + strings.Repeat("b", 64), Harness: "codex-0.157.1", Model: "test"}
	if err := s.Store.RecordFactoryRun(t.Context(), r); err != nil {
		t.Fatal(err)
	}
	return r
}

func factoryViewsHostResponse(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}
}

func readFactoryRun(t *testing.T, s *Server, runID string) (int, map[string]any) {
	t.Helper()
	w := httptest.NewRecorder()
	nativeAPIServe(t, s, w, apiTestRequest(http.MethodGet, "/api/factory/runs/"+runID, "", "alice"))
	var decoded map[string]any
	if w.Code == 200 {
		if err := json.Unmarshal(w.Body.Bytes(), &decoded); err != nil {
			t.Fatal(err)
		}
	}
	return w.Code, decoded
}

func TestFactoryRunStatusPendingLiveAndExcerpt(t *testing.T) {
	inspect := &atomic.Value{}
	inspect.Store(factoryViewsHostResponse(404, "not found"))
	s := factoryViewsFixture(t, roundTrip(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/factory-inspect" {
			return factoryViewsHostResponse(500, "unexpected helper call"), nil
		}
		return inspect.Load().(*http.Response), nil
	}))
	r := factoryViewsRun(t, s)
	code, decoded := readFactoryRun(t, s, r.ID)
	if code != 200 {
		t.Fatalf("pending status: %d", code)
	}
	if _, present := decoded["state"]; present {
		t.Fatalf("pending run carries host state: %v", decoded["state"])
	}
	if _, present := decoded["view"]; present {
		t.Fatalf("unbound run carries a binding: %v", decoded["view"])
	}
	run, _ := decoded["run"].(map[string]any)
	if run["id"] != r.ID || run["role"] != "coder" || run["harness"] != "codex-0.157.1" {
		t.Fatalf("record: %v", decoded["run"])
	}
	if _, present := run["identity_binding"]; present {
		t.Fatal("broker binding leaked into the display record")
	}
	live := `{"id":"` + r.ID + `","project":"` + webTerminalProject + `","phase":"running","live":true,` +
		`"container":"` + strings.Repeat("c", 64) + `","unit":"soda-factory-` + r.ID + `.service","invocation":"` + strings.Repeat("d", 32) + `"}`
	inspect.Store(factoryViewsHostResponse(200, live))
	code, decoded = readFactoryRun(t, s, r.ID)
	if code != 200 {
		t.Fatalf("live status: %d", code)
	}
	state, _ := decoded["state"].(map[string]any)
	if state["phase"] != "running" || state["live"] != true || state["terminal"] != false {
		t.Fatalf("live state: %v", state)
	}
	if state["container"] != strings.Repeat("c", 64) || state["unit"] != "soda-factory-"+r.ID+".service" || state["invocation"] != strings.Repeat("d", 32) {
		t.Fatalf("process binding: %v", state)
	}
	big := `{"id":"` + r.ID + `","project":"` + webTerminalProject + `","phase":"completed","exit_code":0,"output":"` + strings.Repeat("x", 20000) + `"}`
	inspect.Store(factoryViewsHostResponse(200, big))
	code, decoded = readFactoryRun(t, s, r.ID)
	if code != 200 {
		t.Fatalf("terminal status: %d", code)
	}
	state, _ = decoded["state"].(map[string]any)
	output, _ := state["output"].(string)
	if state["terminal"] != true || state["output_truncated"] != true || len(output) != 16*1024 {
		t.Fatalf("excerpt: truncated=%v len=%d", state["output_truncated"], len(output))
	}
}

func TestFactoryRunStatusBindingAndRefusals(t *testing.T) {
	s := factoryViewsFixture(t, roundTrip(func(r *http.Request) (*http.Response, error) {
		return factoryViewsHostResponse(404, "not found"), nil
	}))
	r := factoryViewsRun(t, s)
	if _, _, err := s.Store.RecordFactoryRunView(t.Context(), factory.RunView{RunID: r.ID, Repository: 7, Issue: 42, Attempt: "attempt-1"}); err != nil {
		t.Fatal(err)
	}
	code, decoded := readFactoryRun(t, s, r.ID)
	if code != 200 {
		t.Fatalf("bound status: %d", code)
	}
	view, _ := decoded["view"].(map[string]any)
	if view["repository"] != "7" || view["issue"] != "42" || view["attempt"] != "attempt-1" {
		t.Fatalf("binding: %v", decoded["view"])
	}
	other := factoryViewsRun(t, s)
	if _, _, err := s.Store.RecordFactoryRunView(t.Context(), factory.RunView{RunID: other.ID, Repository: 8, Issue: 9}); err != nil {
		t.Fatal(err)
	}
	if code, _ := readFactoryRun(t, s, other.ID); code != 409 {
		t.Fatalf("mismatched binding: %d", code)
	}
	if code, _ := readFactoryRun(t, s, factory.NewID()); code != 404 {
		t.Fatalf("unknown run: %d", code)
	}
	if code, _ := readFactoryRun(t, s, "short"); code != 404 {
		t.Fatalf("bad run identity: %d", code)
	}
	w := httptest.NewRecorder()
	nativeAPIServe(t, s, w, apiTestRequest(http.MethodGet, "/api/factory/runs/"+r.ID+"?cursor=1", "", "alice"))
	if w.Code != 400 {
		t.Fatalf("query accepted: %d", w.Code)
	}
	readOnly := func(in extensions.CallbackRequest) extensions.CallbackResponse {
		return extensions.CallbackResponse{Repository: &extensions.Repository{ID: in.RepositoryID, Owner: "alice", Name: "repo", Permission: "read"}}
	}
	denied := httptest.NewRecorder()
	nativeAPIServeWithCallback(t, s, denied, apiTestRequest(http.MethodGet, "/api/factory/runs/"+r.ID, "", "bob"), readOnly)
	if denied.Code != 403 {
		t.Fatalf("read-only viewer: %d", denied.Code)
	}
	broken := factoryViewsFixture(t, roundTrip(func(r *http.Request) (*http.Response, error) {
		return factoryViewsHostResponse(500, "helper failed"), nil
	}))
	id := factoryViewsRun(t, broken).ID
	if code, _ := readFactoryRun(t, broken, id); code != 503 {
		t.Fatalf("broken host: %d", code)
	}
}

func TestSpacesShowsFactoryRuns(t *testing.T) {
	s, _, _ := spacesFixture(t, true)
	first := lifecycleWebRun(t, s)
	second := lifecycleWebRun(t, s)
	if _, _, err := s.Store.RecordFactoryRunView(t.Context(), factory.RunView{RunID: first.ID, Repository: 7, Issue: 42, Attempt: "attempt-1"}); err != nil {
		t.Fatal(err)
	}
	settled := second
	settled.Outcome, settled.Summary, settled.Reconciled = factory.Succeeded, "done", true
	if err := s.Store.SaveFactoryRun(t.Context(), settled); err != nil {
		t.Fatal(err)
	}
	view := readSpaces(t, s, spacesCallback(200, new(atomic.Int32)))
	if len(view.Items) != 1 {
		t.Fatalf("items: %d", len(view.Items))
	}
	rows := view.Items[0].FactoryRuns
	if len(rows) != 2 {
		t.Fatalf("runs: %+v", rows)
	}
	// Newest first; the unbound run carries no binding, never a guess.
	if rows[0].ID != second.ID || rows[0].Outcome != factory.Succeeded || !rows[0].Reconciled || rows[0].Issue != "" {
		t.Fatalf("settled row: %+v", rows[0])
	}
	if rows[1].ID != first.ID || rows[1].Role != "coder" || rows[1].Issue != "42" || rows[1].Attempt != "attempt-1" || rows[1].Reconciled {
		t.Fatalf("bound row: %+v", rows[1])
	}
	var raw strings.Builder
	_ = json.NewEncoder(&raw).Encode(view)
	assertNoSecrets(t, raw.String())
}

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

func TestFactoryOutputStreamDeliversStatusOutputAndEOF(t *testing.T) {
	script := &factoryOutputScript{offsets: make(chan float64, 16)}
	proxy, _, origin, s := factoryOutputProxyFixture(t, script)
	run := factoryViewsRun(t, s)
	script.serve = func(body map[string]any) (int, string) {
		offset, _ := body["offset"].(float64)
		if offset == 0 {
			return 200, `{"id":"` + run.ID + `","project":"` + webTerminalProject + `","phase":"running","live":true,` +
				`"container":"` + strings.Repeat("c", 64) + `","unit":"soda-factory-` + run.ID + `.service","invocation":"` + strings.Repeat("d", 32) + `",` +
				`"total":5,"offset":0,"next":5,"data":"aGVsbG8="}`
		}
		return 200, `{"id":"` + run.ID + `","project":"` + webTerminalProject + `","phase":"completed","terminal":true,"exit_code":0,` +
			`"container":"` + strings.Repeat("c", 64) + `","unit":"soda-factory-` + run.ID + `.service","invocation":"` + strings.Repeat("d", 32) + `",` +
			`"total":5,"offset":5,"next":5,"data":""}`
	}
	conn := dialFactoryOutput(t, proxy, origin, run.ID, factoryOutputHandshakeFor(run.ID))
	status := readFactoryFrame(t, conn)
	if status["type"] != "status" || status["run_id"] != run.ID || status["phase"] != "running" || status["live"] != true {
		t.Fatalf("status: %v", status)
	}
	if status["container"] != strings.Repeat("c", 64) || status["unit"] != "soda-factory-"+run.ID+".service" {
		t.Fatalf("process binding: %v", status)
	}
	output := readFactoryFrame(t, conn)
	if output["type"] != "output" || output["data"] != "aGVsbG8=" || output["cursor"] != float64(0) || output["next"] != float64(5) {
		t.Fatalf("output: %v", output)
	}
	if done := readFactoryFrame(t, conn); done["type"] != "status" || done["phase"] != "completed" || done["terminal"] != true {
		t.Fatalf("terminal status: %v", done)
	}
	if closed := readFactoryFrame(t, conn); closed["type"] != "closed" || closed["reason"] != "eof" {
		t.Fatalf("eof: %v", closed)
	}
	// Reattachment resumes by cursor through reads only; the host sees the
	// resumed offset and no launch-shaped call exists on this route.
	resume := factoryOutputHandshakeFor(run.ID)
	resume["cursor"] = 5
	reattached := dialFactoryOutput(t, proxy, origin, run.ID, resume)
	if status := readFactoryFrame(t, reattached); status["phase"] != "completed" {
		t.Fatalf("resumed status: %v", status)
	}
	if closed := readFactoryFrame(t, reattached); closed["reason"] != "eof" {
		t.Fatalf("resumed eof: %v", closed)
	}
	sawResume := false
	for {
		select {
		case offset := <-script.offsets:
			if offset == 5 {
				sawResume = true
			}
		default:
			if !sawResume {
				t.Fatal("resumed cursor never reached the host")
			}
			return
		}
	}
}

func TestFactoryOutputStreamRejectsInput(t *testing.T) {
	script := &factoryOutputScript{offsets: make(chan float64, 16)}
	proxy, _, origin, s := factoryOutputProxyFixture(t, script)
	run := factoryViewsRun(t, s)
	script.serve = func(body map[string]any) (int, string) {
		return 200, `{"id":"` + run.ID + `","project":"` + webTerminalProject + `","phase":"running","live":true,"total":0,"offset":0,"next":0,"data":""}`
	}
	conn := dialFactoryOutput(t, proxy, origin, run.ID, factoryOutputHandshakeFor(run.ID))
	if status := readFactoryFrame(t, conn); status["type"] != "status" {
		t.Fatalf("status: %v", status)
	}
	ctx, done := context.WithTimeout(t.Context(), 5*time.Second)
	defer done()
	if err := conn.Write(ctx, websocket.MessageText, []byte(`{"type":"input","data":"ZWNobyBoaQ=="}`)); err != nil {
		t.Fatal(err)
	}
	_, _, err := conn.Read(ctx)
	var closed websocket.CloseError
	if !errors.As(err, &closed) || closed.Code != websocket.StatusPolicyViolation {
		t.Fatalf("input accepted: %v", err)
	}
}

func TestFactoryOutputStreamRefusesStaleHandshake(t *testing.T) {
	script := &factoryOutputScript{offsets: make(chan float64, 16)}
	script.serve = func(body map[string]any) (int, string) {
		return 200, `{"phase":"running"}`
	}
	proxy, _, origin, s := factoryOutputProxyFixture(t, script)
	run := factoryViewsRun(t, s)
	stale := factoryOutputHandshakeFor(run.ID)
	stale["session_generation"] = "stale-generation"
	conn := dialFactoryOutput(t, proxy, origin, run.ID, stale)
	ctx, done := context.WithTimeout(t.Context(), 5*time.Second)
	defer done()
	if _, _, err := conn.Read(ctx); err == nil {
		t.Fatal("stale handshake attached")
	}
	if calls := script.calls.Load(); calls != 0 {
		t.Fatalf("stale handshake reached the host %d times", calls)
	}
}

func TestFactoryOutputStreamClosesOnUnknownAndStale(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		reason string
	}{
		{"unknown", 404, "run_unknown"},
		{"stale", 409, "incarnation_changed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			script := &factoryOutputScript{offsets: make(chan float64, 16)}
			script.serve = func(body map[string]any) (int, string) { return tc.status, "helper refused" }
			proxy, _, origin, s := factoryOutputProxyFixture(t, script)
			run := factoryViewsRun(t, s)
			conn := dialFactoryOutput(t, proxy, origin, run.ID, factoryOutputHandshakeFor(run.ID))
			if closed := readFactoryFrame(t, conn); closed["type"] != "closed" || closed["reason"] != tc.reason {
				t.Fatalf("close: %v", closed)
			}
		})
	}
}

func TestFactoryOutputStreamRequiresWrite(t *testing.T) {
	script := &factoryOutputScript{offsets: make(chan float64, 16)}
	script.serve = func(body map[string]any) (int, string) { return 200, `{}` }
	proxy, permission, origin, s := factoryOutputProxyFixture(t, script)
	run := factoryViewsRun(t, s)
	permission.Store("read")
	wsURL := "wss" + strings.TrimPrefix(proxy.URL, "https") + "/factory/runs/" + run.ID + "/output"
	headers := nativeTerminalHeaders(http.MethodGet, origin)
	headers.Del(extensions.SessionGenerationHeader)
	_, response, err := websocket.Dial(t.Context(), wsURL, &websocket.DialOptions{HTTPClient: proxy.Client(), HTTPHeader: headers})
	if err == nil {
		t.Fatal("read-only viewer attached")
	}
	if response == nil || response.StatusCode != http.StatusForbidden {
		t.Fatalf("read-only dial: %v", response)
	}
	_ = response.Body.Close()
	if calls := script.calls.Load(); calls != 0 {
		t.Fatalf("refused viewer reached the host %d times", calls)
	}
}
