package host

import (
	"bufio"
	"bytes"
	"context"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/levitateos/sodaos/internal/strictjson"
)

// The program is fixed product code, not a request-selected command or file.
//
//go:embed project_terminal.py
var projectTerminal string

const terminalFrameLimit = 32768
const terminalLimit = 64

// TerminalRequest is private root:soda helper input. The web layer must resolve
// membership/login and authorize the real actor before using this operation.
type TerminalRequest struct {
	Project  string `json:"project"`
	Login    string `json:"login"`
	Identity int64  `json:"identity"`
	Cols     int    `json:"cols"`
	Rows     int    `json:"rows"`
	Expires  int64  `json:"expires"`
}

type TerminalFrame struct {
	Type   string `json:"type"`
	Data   string `json:"data,omitempty"`
	Cols   int    `json:"cols,omitempty"`
	Rows   int    `json:"rows,omitempty"`
	Reason string `json:"reason,omitempty"`
}

func terminalDimensions(cols, rows int) bool {
	return cols >= 2 && cols <= 500 && rows >= 2 && rows <= 300
}
func (in TerminalRequest) valid(now time.Time) bool {
	return projectID.MatchString(in.Project) && loginName.MatchString(in.Login) && in.Login != "root" && in.Identity > 0 && terminalDimensions(in.Cols, in.Rows) && in.Expires > now.Unix() && in.Expires <= now.Add(2*time.Hour).Unix()
}
func (f TerminalFrame) inputValid() bool {
	if f.Reason != "" {
		return false
	}
	switch f.Type {
	case "input":
		data, err := base64.StdEncoding.Strict().DecodeString(f.Data)
		return err == nil && !strings.ContainsAny(f.Data, "\r\n") && len(data) > 0 && len(data) <= 16384 && f.Cols == 0 && f.Rows == 0
	case "resize":
		return f.Data == "" && terminalDimensions(f.Cols, f.Rows)
	case "heartbeat", "close":
		return f.Data == "" && f.Cols == 0 && f.Rows == 0
	}
	return false
}
func (f TerminalFrame) outputValid() bool {
	if f.Cols != 0 || f.Rows != 0 {
		return false
	}
	switch f.Type {
	case "ready":
		return f.Data == "" && f.Reason == ""
	case "output":
		data, err := base64.StdEncoding.Strict().DecodeString(f.Data)
		return err == nil && len(data) > 0 && len(data) <= 4096 && f.Reason == ""
	case "closed":
		return f.Data == "" && (f.Reason == "disconnected" || f.Reason == "expired" || f.Reason == "exited" || f.Reason == "launch_failed" || f.Reason == "stream_failed" || f.Reason == "cleanup_unconfirmed")
	}
	return false
}

// Streaming commands have a separate concrete boundary from buffered mutations.
// No caller can supply an executable, shell arguments, environment or host flags.
type terminalProcess interface {
	Input(TerminalFrame) error
	Output() (TerminalFrame, error)
	Close()
}
type terminalExecutor interface {
	terminal(string, TerminalRequest) (terminalProcess, error)
}

type nativeTerminal struct {
	cmd     *exec.Cmd
	stdin   *os.File
	stdout  io.ReadCloser
	scanner *bufio.Scanner
	done    chan struct{}
	once    sync.Once
}

func (Native) terminal(container string, in TerminalRequest) (terminalProcess, error) {
	seconds := in.Expires - time.Now().Unix()
	if !containerID.MatchString(container) || !in.valid(time.Now()) || seconds < 1 {
		return nil, errors.New("invalid terminal target")
	}
	cmd := exec.Command("/usr/bin/podman", "--remote=false", "exec", "--interactive", container, "/usr/bin/python3", "-I", "-c", projectTerminal, in.Login, strconv.FormatInt(in.Identity, 10), strconv.Itoa(in.Cols), strconv.Itoa(in.Rows), strconv.FormatInt(seconds, 10))
	// Terminal bytes never enter stderr diagnostics, journal or command-error text.
	cmd.Stderr = io.Discard
	input, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	output, childOutput, err := os.Pipe()
	if err != nil {
		input.Close()
		return nil, err
	}
	cmd.Stdout = childOutput
	if err = cmd.Start(); err != nil {
		input.Close()
		output.Close()
		childOutput.Close()
		return nil, err
	}
	_ = childOutput.Close()
	p := &nativeTerminal{cmd: cmd, stdin: input.(*os.File), stdout: output, scanner: bufio.NewScanner(output), done: make(chan struct{})}
	p.scanner.Buffer(make([]byte, 4096), terminalFrameLimit)
	go func() { _ = cmd.Wait(); close(p.done) }()
	return p, nil
}
func (p *nativeTerminal) Input(f TerminalFrame) error {
	if !f.inputValid() {
		return errors.New("invalid terminal control")
	}
	body, err := json.Marshal(f)
	if err != nil {
		return err
	}
	if err = p.stdin.SetWriteDeadline(time.Now().Add(2 * time.Second)); err != nil {
		return err
	}
	_, err = p.stdin.Write(append(body, '\n'))
	return err
}
func (p *nativeTerminal) Output() (TerminalFrame, error) {
	var f TerminalFrame
	if !p.scanner.Scan() {
		return f, io.EOF
	}
	if err := strictjson.Decode(bytes.NewReader(p.scanner.Bytes()), &f); err != nil || !f.outputValid() {
		return f, errors.New("invalid terminal response")
	}
	return f, nil
}
func (p *nativeTerminal) Close() {
	p.once.Do(func() {
		// Closing stdin requests launcher EOF. If conmon does not propagate it, the
		// independent project-local heartbeat lease still expires. Killing this CLI is
		// NOT reported as confirmation that its container login process exited.
		_ = p.stdin.Close()
		select {
		case <-p.done:
		case <-time.After(3 * time.Second):
			_ = p.cmd.Process.Kill()
		}
		_ = p.stdout.Close()
	})
}

var containerID = regexp.MustCompile(`^[0-9a-f]{64}$`)

const terminalInspect = `{"id":{{json .ID}},"running":{{json .State.Running}},"project":{{json (index .Config.Labels "org.soda.project")}},"owner":{{json (index .Config.Labels "org.soda.owner")}},"privileged":{{json .HostConfig.Privileged}},"userns":{{json .HostConfig.UsernsMode}},"mappings":{{json .HostConfig.IDMappings}}}`

func (d *Daemon) terminalContainer(ctx context.Context, id string) (string, error) {
	return d.projectContainer(ctx, id, true)
}

// Native lifecycle may inspect stopped containers, never missing/replacement ones.
func (d *Daemon) projectContainer(ctx context.Context, id string, requireRunning bool) (string, error) {
	if !projectID.MatchString(id) {
		return "", errors.New("invalid project")
	}
	data, err := d.podman(ctx, nil, "--remote=false", "inspect", "--format", terminalInspect, "soda-"+id)
	if err != nil || len(data) > 4096 {
		return "", errors.New("terminal inspection unavailable")
	}
	var v struct {
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
	if err = strictjson.Decode(bytes.NewReader(data), &v); err != nil {
		return "", errors.New("invalid terminal inspection")
	}
	owner, e := strconv.ParseInt(v.Owner, 10, 64)
	if !containerID.MatchString(v.ID) || (requireRunning && !v.Running) || v.Project != id || e != nil || owner <= 0 || v.Privileged || v.Userns != "private" || !terminalIDMap(v.Mappings.UIDMap) || !terminalIDMap(v.Mappings.GIDMap) {
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

// Shutdown closes both pending and live streams; HTTP Shutdown alone does not
// close hijacked WebSockets. No native mutation/global user process lock is held.
func (d *Daemon) CloseTerminals() {
	d.terminalMu.Lock()
	d.terminalClosed = true
	for _, cancel := range d.terminals {
		cancel()
	}
	d.terminalMu.Unlock()
	d.terminalWG.Wait()
}
func (d *Daemon) terminalHandler(w http.ResponseWriter, r *http.Request) {
	// This handler is private to the existing filesystem-authorized Unix socket.
	// Browser Origin/cookie/CSRF authorization belongs to a separate future web route.
	if r.Method != "GET" || r.URL.RawQuery != "" || r.URL.ForceQuery || r.URL.RawPath != "" || len(r.Header.Values("Origin")) != 0 {
		http.Error(w, "invalid private terminal request", 400)
		return
	}
	executor, ok := d.Exec.(terminalExecutor)
	if !ok {
		http.Error(w, "terminal unavailable", 503)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Hour)
	defer cancel()
	d.terminalMu.Lock()
	if d.terminalClosed || len(d.terminals) >= terminalLimit {
		d.terminalMu.Unlock()
		http.Error(w, "terminal unavailable", 503)
		return
	}
	if d.terminals == nil {
		d.terminals = make(map[*http.Request]context.CancelFunc)
	}
	d.terminals[r] = cancel
	d.terminalWG.Add(1)
	d.terminalMu.Unlock()
	defer d.terminalWG.Done()
	defer func() { d.terminalMu.Lock(); delete(d.terminals, r); d.terminalMu.Unlock() }()
	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		return
	}
	defer conn.CloseNow()
	conn.SetReadLimit(terminalFrameLimit)
	first, firstCancel := context.WithTimeout(ctx, 5*time.Second)
	kind, body, err := conn.Read(first)
	firstCancel()
	var in TerminalRequest
	if err != nil || kind != websocket.MessageText || len(body) > 4096 || strictjson.Decode(bytes.NewReader(body), &in) != nil || !in.valid(time.Now()) {
		return
	}
	ctx, expiryCancel := context.WithDeadline(ctx, time.Unix(in.Expires, 0))
	defer expiryCancel()
	inspectCtx, inspectCancel := context.WithTimeout(ctx, 10*time.Second)
	id, err := d.terminalContainer(inspectCtx, in.Project)
	inspectCancel()
	if err != nil {
		writeTerminal(ctx, conn, TerminalFrame{Type: "closed", Reason: "launch_failed"})
		return
	}
	if ctx.Err() != nil {
		return
	}
	p, err := executor.terminal(id, in)
	if err != nil {
		writeTerminal(ctx, conn, TerminalFrame{Type: "closed", Reason: "launch_failed"})
		return
	}
	defer p.Close()
	stopped := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			p.Close()
		case <-stopped:
		}
	}()
	defer close(stopped)
	// Only the authenticated backend supplies heartbeat controls. Ordinary data
	// does not renew the project-local lease after its Soda session is invalidated.
	incoming := make(chan error, 1)
	go func() {
		for {
			kind, body, err := conn.Read(ctx)
			if err != nil {
				incoming <- err
				return
			}
			var f TerminalFrame
			if kind != websocket.MessageText || strictjson.Decode(bytes.NewReader(body), &f) != nil || !f.inputValid() {
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
	}()
	// A disconnect must also stop a blocked native output read.
	go func() {
		select {
		case <-incoming:
			p.Close()
		case <-stopped:
		}
	}()
	for {
		f, err := p.Output()
		if err != nil {
			return
		}
		if err = writeTerminal(ctx, conn, f); err != nil {
			return
		}
		if f.Type == "closed" {
			return
		}
	}
}
func writeTerminal(ctx context.Context, c *websocket.Conn, f TerminalFrame) error {
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
