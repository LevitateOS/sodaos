package api

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/levitateos/sodaos/internal/host"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
)

// scriptedFactoryOutput serves one recorded log in bounded slices with a
// fixed terminal phase, like the native host output route.
type scriptedFactoryOutput struct {
	log      []byte
	terminal bool
	phase    string
	limit    int
}

func (s *scriptedFactoryOutput) RoundTrip(r *http.Request) (*http.Response, error) {
	var in project.FactoryOutput
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(body, &in); err != nil {
		return nil, err
	}
	if r.URL.Path != "/factory-output" {
		return &http.Response{StatusCode: http.StatusNotFound, Body: io.NopCloser(strings.NewReader("")), Header: make(http.Header), Request: r}, nil
	}
	s.limit = in.Limit
	start := in.Offset
	if start < 0 {
		start = 0
	}
	if start > int64(len(s.log)) {
		start = int64(len(s.log))
	}
	end := start + int64(in.Limit)
	if end > int64(len(s.log)) {
		end = int64(len(s.log))
	}
	chunk := s.log[start:end]
	out := project.FactoryOutputState{
		ID: in.ID, Project: in.Project, Phase: s.phase, Terminal: s.terminal,
		Total: int64(len(s.log)), Offset: start, Next: start + int64(len(chunk)),
	}
	if len(chunk) > 0 {
		out.Data = base64.StdEncoding.EncodeToString(chunk)
	}
	buf, err := json.Marshal(out)
	if err != nil {
		return nil, err
	}
	return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewReader(buf)), Header: make(http.Header), Request: r}, nil
}

func factoryOutputTestPair(t *testing.T) (serverConn, clientConn *websocket.Conn, cleanup func()) {
	t.Helper()
	connCh := make(chan *websocket.Conn, 1)
	errCh := make(chan error, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
		if err != nil {
			errCh <- err
			return
		}
		connCh <- conn
	}))
	dial, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	client, _, err := websocket.Dial(dial, "ws"+strings.TrimPrefix(server.URL, "http"), nil)
	cancel()
	if err != nil {
		server.Close()
		t.Fatalf("dial test peer: %v", err)
	}
	client.SetReadLimit(1 << 20)
	select {
	case conn := <-connCh:
		return conn, client, func() {
			client.CloseNow()
			conn.CloseNow()
			server.Close()
		}
	case err := <-errCh:
		server.Close()
		t.Fatalf("accept test peer: %v", err)
	case <-time.After(10 * time.Second):
		server.Close()
		t.Fatal("accept test peer: timeout")
	}
	return nil, nil, nil
}

func readFactoryTestFrame(t *testing.T, conn *websocket.Conn) map[string]any {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, body, err := conn.Read(ctx)
	if err != nil {
		t.Fatalf("read frame: %v", err)
	}
	var frame map[string]any
	if err := json.Unmarshal(body, &frame); err != nil {
		t.Fatalf("decode frame: %v", err)
	}
	return frame
}

func factoryOutputTestBytes(t *testing.T, frame map[string]any) []byte {
	t.Helper()
	if frame["type"] != "output" {
		t.Fatalf("expected output frame, got %v", frame["type"])
	}
	raw, err := base64.StdEncoding.DecodeString(frame["data"].(string))
	if err != nil {
		t.Fatalf("decode output data: %v", err)
	}
	return raw
}

func TestFactoryOutputFrameFitsProxyWireLimit(t *testing.T) {
	raw := make([]byte, project.MaxFactoryOutputRead)
	for i := range raw {
		raw[i] = byte(i)
	}
	frame := factoryOutputFrame{
		Type: "output", Data: base64.StdEncoding.EncodeToString(raw),
		Cursor: project.MaxFactoryOutputOffset, Next: project.MaxFactoryOutputOffset, Truncated: true,
	}
	body, err := json.Marshal(frame)
	if err != nil {
		t.Fatal(err)
	}
	if len(body) > 32768 {
		t.Fatalf("max output frame is %d wire bytes, above the 32768 proxy limit", len(body))
	}
}

func TestPumpFactorySliceDrainsTerminalOutputBeforeEOF(t *testing.T) {
	log := make([]byte, 2*project.MaxFactoryOutputRead)
	for i := range log {
		log[i] = byte(i % 251)
	}
	script := &scriptedFactoryOutput{log: log, terminal: true, phase: project.FactoryCompleted}
	s := &API{Host: &host.Client{HTTP: &http.Client{Transport: script}}}
	identity := factoryViewerIdentity{project: store.Project{ID: "p0123456789abcdef01234567"}, runID: strings.Repeat("a", 32)}
	serverConn, clientConn, cleanup := factoryOutputTestPair(t)
	defer cleanup()
	ctx := context.Background()
	var last factoryStatusFrame
	sent := false

	next, done, ok := s.pumpFactorySlice(ctx, serverConn, identity, 0, &last, &sent)
	if !ok {
		t.Fatal("first terminal slice not served")
	}
	if done {
		t.Fatalf("terminal slice ended at cursor %d with %d of %d bytes unread", next, int64(len(log))-next, len(log))
	}
	if next != int64(project.MaxFactoryOutputRead) {
		t.Fatalf("first slice advanced to %d", next)
	}
	if status := readFactoryTestFrame(t, clientConn); status["type"] != "status" {
		t.Fatalf("first frame: %v", status["type"])
	}
	var got []byte
	got = append(got, factoryOutputTestBytes(t, readFactoryTestFrame(t, clientConn))...)

	final, done, ok := s.pumpFactorySlice(ctx, serverConn, identity, next, &last, &sent)
	if !ok || !done {
		t.Fatalf("final slice ended=%v served=%v", done, ok)
	}
	if final != int64(len(log)) {
		t.Fatalf("final cursor %d, want %d", final, len(log))
	}
	got = append(got, factoryOutputTestBytes(t, readFactoryTestFrame(t, clientConn))...)
	if !bytes.Equal(got, log) {
		t.Fatalf("delivered %d bytes, want %d exact log bytes", len(got), len(log))
	}
	closed := readFactoryTestFrame(t, clientConn)
	if closed["type"] != "closed" || closed["reason"] != "eof" {
		t.Fatalf("terminal close: %+v", closed)
	}
}

func TestPumpFactorySliceEndsEmptyTerminalRun(t *testing.T) {
	script := &scriptedFactoryOutput{terminal: true, phase: project.FactoryCompleted}
	s := &API{Host: &host.Client{HTTP: &http.Client{Transport: script}}}
	identity := factoryViewerIdentity{project: store.Project{ID: "p0123456789abcdef01234567"}, runID: strings.Repeat("a", 32)}
	serverConn, clientConn, cleanup := factoryOutputTestPair(t)
	defer cleanup()
	var last factoryStatusFrame
	sent := false
	next, done, ok := s.pumpFactorySlice(context.Background(), serverConn, identity, 0, &last, &sent)
	if !ok || !done || next != 0 {
		t.Fatalf("empty terminal slice served=%v ended=%v cursor=%d", ok, done, next)
	}
	if status := readFactoryTestFrame(t, clientConn); status["type"] != "status" {
		t.Fatalf("first frame: %v", status["type"])
	}
	closed := readFactoryTestFrame(t, clientConn)
	if closed["type"] != "closed" || closed["reason"] != "eof" {
		t.Fatalf("terminal close: %+v", closed)
	}
}
