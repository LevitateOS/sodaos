// Package terminal attaches privileged project terminals. It does not own
// HTTP admission or project create/lifecycle.
package terminal

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/levitateos/sodaos/internal/strictjson"
)

type Executor interface {
	Run(context.Context, []byte, string, ...string) ([]byte, error)
}

type Service struct {
	Exec Executor

	mu      sync.Mutex
	streams map[*http.Request]context.CancelFunc
	closed  bool
	wg      sync.WaitGroup
}

const terminalInspect = `{"id":{{json .ID}},"running":{{json .State.Running}},"project":{{json (index .Config.Labels "org.soda.project")}},"owner":{{json (index .Config.Labels "org.soda.owner")}},"privileged":{{json .HostConfig.Privileged}},"userns":{{json .HostConfig.UsernsMode}},"mappings":{{json .HostConfig.IDMappings}}}`

type terminalInspection struct {
	ID         string `json:"id"`
	Running    bool   `json:"running"`
	Project    string `json:"project"`
	Owner      string `json:"owner"`
	Privileged bool   `json:"privileged"`
	Userns     string `json:"userns"`
	Mappings   struct {
		UIDMap []string `json:"UidMap"`
		GIDMap []string `json:"GidMap"`
	} `json:"mappings"`
}

func (s *Service) podman(ctx context.Context, in []byte, args ...string) ([]byte, error) {
	return s.Exec.Run(ctx, in, "/usr/bin/podman", args...)
}

// Native lifecycle may inspect stopped containers, never missing/replacement ones.
func terminalIsolation(v terminalInspection, id string) bool {
	if !containerID.MatchString(v.ID) || v.Project != id || v.Privileged || v.Userns != "private" {
		return false
	}
	return terminalIDMap(v.Mappings.UIDMap) && terminalIDMap(v.Mappings.GIDMap)
}

func terminalTargetReady(v terminalInspection, id string, requireRunning bool) bool {
	owner, err := strconv.ParseInt(v.Owner, 10, 64)
	if err != nil || owner <= 0 {
		return false
	}
	if requireRunning && !v.Running {
		return false
	}
	return terminalIsolation(v, id)
}

func (s *Service) projectContainer(ctx context.Context, id string, requireRunning bool) (string, error) {
	if !projectID.MatchString(id) {
		return "", errors.New("invalid project")
	}
	data, err := s.podman(ctx, nil, "--remote=false", "inspect", "--format", terminalInspect, "soda-"+id)
	if err != nil || len(data) > 4096 {
		return "", errors.New("terminal inspection unavailable")
	}
	var v terminalInspection
	if err = strictjson.Decode(bytes.NewReader(data), &v); err != nil {
		return "", errors.New("invalid terminal inspection")
	}
	if !terminalTargetReady(v, id, requireRunning) {
		return "", errors.New("terminal target not ready or isolated")
	}
	return v.ID, nil
}

// Podman reports the resulting namespace as "private", not its create-time
// "auto" selector. Require the production candidate's single 262144-ID mapping
// with container root shifted away from host root; do not infer isolation from
// the mode string alone or introduce identity-remapping policy.
func terminalIDMap(values []string) bool {
	if len(values) != 1 {
		return false
	}
	parts := strings.Split(values[0], ":")
	if len(parts) != 3 || parts[0] != "0" || parts[2] != "262144" {
		return false
	}
	base, err := strconv.ParseUint(parts[1], 10, 32)
	return err == nil && strconv.FormatUint(base, 10) == parts[1] && base > 0 && base+262144 <= 4294967295
}

// Close closes both pending and live streams; HTTP Shutdown alone does not
// close hijacked WebSockets. No native mutation/global user process lock is held.
func (s *Service) Close() {
	s.mu.Lock()
	s.closed = true
	for _, cancel := range s.streams {
		cancel()
	}
	s.mu.Unlock()
	s.wg.Wait()
}

func validPrivateTerminalRequest(r *http.Request) bool {
	return r.Method == "GET" && r.URL.RawQuery == "" && !r.URL.ForceQuery && r.URL.RawPath == "" && len(r.Header.Values("Origin")) == 0
}

func (s *Service) register(r *http.Request, cancel context.CancelFunc) (func(), bool) {
	s.mu.Lock()
	if s.closed || len(s.streams) >= 2*terminalLimit {
		s.mu.Unlock()
		return nil, false
	}
	if s.streams == nil {
		s.streams = make(map[*http.Request]context.CancelFunc)
	}
	s.streams[r] = cancel
	s.wg.Add(1)
	s.mu.Unlock()

	cleanup := func() {
		s.wg.Done()
		s.mu.Lock()
		delete(s.streams, r)
		s.mu.Unlock()
	}
	return cleanup, true
}

func readTerminalRequest(ctx context.Context, conn *websocket.Conn) (TerminalRequest, bool) {
	first, firstCancel := context.WithTimeout(ctx, 5*time.Second)
	kind, body, err := conn.Read(first)
	firstCancel()
	var in TerminalRequest
	if err != nil || kind != websocket.MessageText || len(body) > 4096 || strictjson.Decode(bytes.NewReader(body), &in) != nil || !in.Valid(time.Now()) {
		return TerminalRequest{}, false
	}
	return in, true
}

func (s *Service) launch(ctx context.Context, launcher attachLauncher, conn *websocket.Conn, in TerminalRequest) (Process, bool) {
	inspectCtx, inspectCancel := context.WithTimeout(ctx, 10*time.Second)
	id, err := s.projectContainer(inspectCtx, in.Project, true)
	inspectCancel()
	if err != nil {
		Write(ctx, conn, TerminalFrame{Type: "closed", Reason: "launch_failed"})
		return nil, false
	}
	if ctx.Err() != nil {
		return nil, false
	}
	p, err := launcher.Terminal(id, in)
	if err != nil {
		Write(ctx, conn, TerminalFrame{Type: "closed", Reason: "launch_failed"})
		return nil, false
	}
	return p, true
}

func pumpIncoming(ctx context.Context, conn *websocket.Conn, p Process, incoming chan<- error) {
	for {
		kind, body, err := conn.Read(ctx)
		if err != nil {
			incoming <- err
			return
		}
		var f TerminalFrame
		if kind != websocket.MessageText || strictjson.Decode(bytes.NewReader(body), &f) != nil || !f.InputValid() {
			incoming <- errors.New("invalid control")
			p.Close()
			return
		}
		if err = p.Input(f); err != nil {
			incoming <- err
			p.Close()
			return
		}
		if f.Type == "close" {
			// Let the launcher report teardown before closing its stdout.
			// No further input/heartbeats are accepted; its lease stays bounded.
			return
		}
	}
}

func pumpOutgoing(ctx context.Context, conn *websocket.Conn, p Process) {
	for {
		f, err := p.Output()
		if err != nil {
			return
		}
		if err = Write(ctx, conn, f); err != nil {
			return
		}
		if f.Type == "closed" || f.Type == "metadata" {
			return
		}
	}
}

func pumpIO(ctx context.Context, conn *websocket.Conn, p Process) {
	stopped := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			p.Close()
		case <-stopped:
		}
	}()
	defer close(stopped)

	// Only the authenticated backend supplies attached-access heartbeats. This
	// bounds the PTY bridge after logout, not the systemd-owned shell's lifetime.
	incoming := make(chan error, 1)
	go pumpIncoming(ctx, conn, p, incoming)

	// A disconnect must also stop a blocked native output read.
	go func() {
		select {
		case <-incoming:
			p.Close()
		case <-stopped:
		}
	}()

	pumpOutgoing(ctx, conn, p)
}

// Handler serves the private Unix-socket terminal attach route.
func (s *Service) Handler(w http.ResponseWriter, r *http.Request) {
	// This handler is private to the existing filesystem-authorized Unix socket.
	// Browser Origin/cookie/CSRF authorization belongs to the web route.
	if !validPrivateTerminalRequest(r) {
		http.Error(w, "invalid private terminal request", 400)
		return
	}
	launcher, ok := s.Exec.(attachLauncher)
	if !ok {
		http.Error(w, "terminal unavailable", http.StatusServiceUnavailable)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Hour)
	defer cancel()

	cleanup, ok := s.register(r, cancel)
	if !ok {
		http.Error(w, "terminal unavailable", http.StatusServiceUnavailable)
		return
	}
	defer cleanup()

	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		return
	}
	defer conn.CloseNow()
	conn.SetReadLimit(FrameLimit)

	in, ok := readTerminalRequest(ctx, conn)
	if !ok {
		return
	}
	ctx, expiryCancel := context.WithDeadline(ctx, time.Unix(in.Expires, 0))
	defer expiryCancel()

	p, ok := s.launch(ctx, launcher, conn, in)
	if !ok {
		return
	}
	defer p.Close()

	pumpIO(ctx, conn, p)
}

func Write(ctx context.Context, c *websocket.Conn, f TerminalFrame) error {
	body, err := json.Marshal(f)
	if err != nil {
		return err
	}
	writeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err = c.Write(writeCtx, websocket.MessageText, body); err != nil {
		return fmt.Errorf("terminal transport ended")
	}
	return nil
}
