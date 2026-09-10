package host

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"
)

type terminalFake struct {
	mu         sync.Mutex
	calls      int
	starts     int
	observedID string
	observed   TerminalRequest
	inspect    []byte
	process    *terminalFakeProcess
}

func (f *terminalFake) Run(_ context.Context, _ []byte, command string, args ...string) ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	if command != "/usr/bin/podman" || !reflect.DeepEqual(args, []string{"--remote=false", "inspect", "--format", terminalInspect, "soda-p0123456789abcdef01234567"}) {
		return nil, errors.New("unexpected native call")
	}
	return f.inspect, nil
}
func (f *terminalFake) terminal(id string, in TerminalRequest) (terminalProcess, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.starts++
	f.observedID = id
	f.observed = in
	return f.process, nil
}

type terminalFakeProcess struct {
	once   sync.Once
	closed chan struct{}
	input  chan TerminalFrame
	output chan TerminalFrame
}

func (p *terminalFakeProcess) Close() { p.once.Do(func() { close(p.closed) }) }
func (p *terminalFakeProcess) Input(f TerminalFrame) error {
	select {
	case <-p.closed:
		return io.EOF
	case p.input <- f:
		return nil
	}
}
func (p *terminalFakeProcess) Output() (TerminalFrame, error) {
	select {
	case <-p.closed:
		return TerminalFrame{}, io.EOF
	case f := <-p.output:
		return f, nil
	}
}
func terminalFixture(t *testing.T) (*Daemon, *Client, *terminalFake) {
	t.Helper()
	p := &terminalFakeProcess{closed: make(chan struct{}), input: make(chan TerminalFrame, 4), output: make(chan TerminalFrame, 4)}
	f := &terminalFake{inspect: []byte(`{"id":"` + strings.Repeat("a", 64) + `","project":"p0123456789abcdef01234567","owner":"2","running":true,"privileged":false,"userns":"private","mappings":{"UidMap":["0:1000000:262144"],"GidMap":["0:1000000:262144"]}}`), process: p}
	d := &Daemon{Exec: f}
	// t.TempDir includes the full test/subtest name, which can overflow
	// sockaddr_un on builders with longer SSD-backed temporary paths.
	dir, err := os.MkdirTemp("", "soda-term-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(dir); err != nil {
			t.Error(err)
		}
	})
	socket := dir + "/host.sock"
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	server := &http.Server{Handler: d}
	go server.Serve(listener)
	t.Cleanup(func() { d.CloseTerminals(); server.Close() })
	return d, NewClient(socket), f
}
func terminalInput() TerminalRequest {
	return TerminalRequest{Action: "attach", ID: strings.Repeat("a", 32), Project: "p0123456789abcdef01234567", Login: "alice", Identity: 2, Cols: 80, Rows: 24, Expires: time.Now().Add(time.Minute).Unix()}
}
func TestTerminalIdentityAndIsolationRefusal(t *testing.T) {
	for _, change := range []func(*TerminalRequest){func(v *TerminalRequest) { v.Login = "root" }, func(v *TerminalRequest) { v.Login = "bob;id" }, func(v *TerminalRequest) { v.Project = "../../host" }, func(v *TerminalRequest) { v.Identity = 0 }, func(v *TerminalRequest) { v.Rows = 0 }, func(v *TerminalRequest) { v.Expires = time.Now().Add(13 * time.Hour).Unix() }} {
		in := terminalInput()
		change(&in)
		if in.valid(time.Now()) {
			t.Fatal("invalid terminal accepted")
		}
	}
	for _, field := range []string{"id", "project", "owner", "running", "privileged", "userns"} {
		t.Run(field, func(t *testing.T) {
			d, _, f := terminalFixture(t)
			var v map[string]any
			json.Unmarshal(f.inspect, &v)
			switch field {
			case "id":
				v[field] = "host"
			case "project":
				v[field] = "pffffffffffffffffffffffff"
			case "owner":
				v[field] = "0"
			case "running":
				v[field] = false
			case "privileged":
				v[field] = true
			case "userns":
				v[field] = "host"
			}
			f.inspect, _ = json.Marshal(v)
			if _, err := d.terminalContainer(context.Background(), terminalInput().Project); err == nil {
				t.Fatal("unsafe target accepted")
			}
			if f.starts != 0 {
				t.Fatal("refusal spawned terminal")
			}
		})
	}
}
func TestTerminalStreamUsesVerifiedIDAndDoesNotHoldMutationLock(t *testing.T) {
	d, c, f := terminalFixture(t)
	if err := d.acquireAdmission(t.Context()); err != nil {
		t.Fatal(err)
	}
	defer func() { <-d.admission }()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	stream, err := c.OpenTerminal(ctx, terminalInput())
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()
	f.process.output <- TerminalFrame{Type: "ready"}
	got, err := stream.Receive(ctx)
	if err != nil || got.Type != "ready" {
		t.Fatal("terminal not ready", err)
	}
	f.mu.Lock()
	if f.starts != 1 || f.observedID != strings.Repeat("a", 64) || f.observed.Login != "alice" {
		t.Error("terminal target not bound")
	}
	f.mu.Unlock()
	if err = stream.Send(ctx, TerminalFrame{Type: "resize", Cols: 100, Rows: 30}); err != nil {
		t.Fatal(err)
	}
	select {
	case got = <-f.process.input:
		if got.Type != "resize" || got.Cols != 100 {
			t.Fatal("control changed")
		}
	case <-ctx.Done():
		t.Fatal("control blocked")
	}
	stream.Close()
	select {
	case <-f.process.closed:
	case <-ctx.Done():
		t.Fatal("disconnect left native stream open")
	}
}
func TestTerminalBadFirstFrameNeverInspectsOrStarts(t *testing.T) {
	for _, body := range []string{`{"login":"root"}`, `{"project":"p0123456789abcdef01234567","command":"id"}`, `{"login":"alice","login":"bob"}`, `null`, strings.Repeat("x", 4097)} {
		t.Run(body[:min(len(body), 20)], func(t *testing.T) {
			_, c, f := terminalFixture(t)
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			conn, _, err := websocket.Dial(ctx, "ws://soda-host/terminal", &websocket.DialOptions{HTTPClient: c.HTTP})
			if err != nil {
				t.Fatal(err)
			}
			defer conn.CloseNow()
			_ = conn.Write(ctx, websocket.MessageText, []byte(body))
			_, _, err = conn.Read(ctx)
			if err == nil {
				t.Fatal("bad handshake accepted")
			}
			f.mu.Lock()
			defer f.mu.Unlock()
			if f.calls != 0 || f.starts != 0 {
				t.Fatal("bad handshake reached native state")
			}
		})
	}
}
func TestTerminalShutdownClosesHijackedStreams(t *testing.T) {
	d, c, f := terminalFixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	stream, err := c.OpenTerminal(ctx, terminalInput())
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()
	f.process.output <- TerminalFrame{Type: "ready"}
	if _, err = stream.Receive(ctx); err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() { d.CloseTerminals(); close(done) }()
	select {
	case <-done:
	case <-ctx.Done():
		t.Fatal("shutdown stuck")
	}
	select {
	case <-f.process.closed:
	default:
		t.Fatal("process not closed")
	}
	if _, err = c.OpenTerminal(ctx, terminalInput()); err == nil {
		t.Fatal("shutdown allowed a new stream")
	}
}
func TestTerminalCloseWaitsForNativeReceipt(t *testing.T) {
	_, c, f := terminalFixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	stream, err := c.OpenTerminal(ctx, terminalInput())
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()
	f.process.output <- TerminalFrame{Type: "ready"}
	if _, err = stream.Receive(ctx); err != nil {
		t.Fatal(err)
	}
	if err = stream.Send(ctx, TerminalFrame{Type: "close"}); err != nil {
		t.Fatal(err)
	}
	select {
	case <-f.process.input:
	case <-ctx.Done():
		t.Fatal("close was not delivered")
	}
	select {
	case <-f.process.closed:
		t.Fatal("transport closed before native teardown receipt")
	default:
	}
	f.process.output <- TerminalFrame{Type: "closed", Reason: "disconnected"}
	frame, err := stream.Receive(ctx)
	if err != nil || frame.Type != "closed" {
		t.Fatal("missing native receipt", err)
	}
}

func TestTerminalPrivateRouteRefusesOriginAndQueries(t *testing.T) {
	_, c, f := terminalFixture(t)
	for _, endpoint := range []string{"http://soda-host/terminal?token=synthetic", "http://soda-host/terminal?", "http://soda-host/terminal"} {
		req, _ := http.NewRequest("GET", endpoint, nil)
		if !strings.Contains(endpoint, "?") {
			req.Header.Set("Origin", "https://localhost")
		}
		response, err := c.HTTP.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		response.Body.Close()
		if response.StatusCode != 400 {
			t.Fatal("private route accepted browser/query input")
		}
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.calls != 0 || f.starts != 0 {
		t.Fatal("invalid request reached native state")
	}
}

func TestTerminalNativeMappingShape(t *testing.T) {
	if !terminalIDMap([]string{"0:1000000:262144"}) {
		t.Fatal("native auto mapping refused")
	}
	for _, value := range [][]string{nil, {"0:0:262144"}, {"0:1000000:1"}, {"1:1000000:262144"}, {"0:4294967295:262144"}, {"0:+1000000:262144"}, {"0:1000000:262144", "1:2:3"}} {
		if terminalIDMap(value) {
			t.Fatal("invalid native mapping accepted")
		}
	}
}

func TestTerminalControlsAreNotCommandParameters(t *testing.T) {
	for _, frame := range []TerminalFrame{{Type: "command", Data: "id"}, {Type: "input", Data: "not base64"}, {Type: "input", Data: "YQ==", Rows: 24}, {Type: "resize", Cols: 501, Rows: 24}, {Type: "heartbeat", Reason: "override"}} {
		if frame.inputValid() {
			t.Fatal("bad control accepted")
		}
	}
	for _, frame := range []TerminalFrame{{Type: "ready"}, {Type: "output", Data: "YQ=="}, {Type: "closed", Reason: "exited"}} {
		if !frame.outputValid() {
			t.Fatal("valid output refused")
		}
	}
}
