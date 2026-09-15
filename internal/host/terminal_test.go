package host

import (
	"context"
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

	"github.com/levitateos/sodaos/internal/hostterminal"
)

// Host keeps a Client-facing integration check that terminal streams do not
// hold the mutation admission gate (Service unit coverage lives in hostterminal).
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
	inspectFormat := `{"id":{{json .ID}},"running":{{json .State.Running}},"project":{{json (index .Config.Labels "org.soda.project")}},"owner":{{json (index .Config.Labels "org.soda.owner")}},"privileged":{{json .HostConfig.Privileged}},"userns":{{json .HostConfig.UsernsMode}},"mappings":{{json .HostConfig.IDMappings}}}`
	if command != "/usr/bin/podman" || !reflect.DeepEqual(args, []string{"--remote=false", "inspect", "--format", inspectFormat, "soda-p0123456789abcdef01234567"}) {
		return nil, errors.New("unexpected native call")
	}
	return f.inspect, nil
}
func (f *terminalFake) Terminal(id string, in TerminalRequest) (hostterminal.Process, error) {
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
	d := testDaemonPtr(f, Config{})
	d.Terminal = &hostterminal.Service{Exec: f}
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

func TestTerminalStreamDoesNotHoldMutationLock(t *testing.T) {
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

func TestTerminalShutdownWhileAdmissionHeld(t *testing.T) {
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
	if err := d.acquireAdmission(ctx); err != nil {
		t.Fatal(err)
	}
	defer func() { <-d.admission }()
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
}
