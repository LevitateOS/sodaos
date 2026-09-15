package host

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
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
	"unicode"
	"unicode/utf8"

	"github.com/coder/websocket"
	"github.com/levitateos/sodaos/internal/strictjson"
)

// The program is fixed product code, not a request-selected command or file.
//
//go:embed project_terminal.py
var projectTerminal string

const (
	terminalFrameLimit = 131072 // bounded 64-row metadata; IO payload bounds remain smaller
	terminalLimit      = 64
)

// TerminalRequest is private root:soda helper input. The web layer must resolve
// membership/login and authorize the real actor before using this operation.
type TerminalRequest struct {
	Action   string `json:"action"`
	ID       string `json:"id"`
	Project  string `json:"project"`
	Login    string `json:"login"`
	Identity int64  `json:"identity"`
	Cols     int    `json:"cols"`
	Rows     int    `json:"rows"`
	Expires  int64  `json:"expires"` // request/attachment deadline, never shell lifetime
	Name     string `json:"name"`
	Scope    string `json:"scope"` // opaque creation-context digest, never browser authority
}

type TerminalState struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatedAt int64  `json:"created_at"`
	Ready     bool   `json:"ready"`
	Attached  bool   `json:"attached"`
	State     string `json:"state"`
}

type TerminalFrame struct {
	Type      string           `json:"type"`
	Data      string           `json:"data,omitempty"`
	Cols      int              `json:"cols,omitempty"`
	Rows      int              `json:"rows,omitempty"`
	Reason    string           `json:"reason,omitempty"`
	Terminals *[]TerminalState `json:"terminals,omitempty"`
}

func ValidTerminalName(name string) bool {
	if !utf8.ValidString(name) || utf8.RuneCountInString(name) > 80 {
		return false
	}
	for _, c := range name {
		if unicode.IsControl(c) || unicode.Is(unicode.Cf, c) {
			return false
		}
	}
	return true
}

func terminalDimensions(cols, rows int) bool {
	return cols >= 2 && cols <= 500 && rows >= 2 && rows <= 300
}

func validTerminalActor(in TerminalRequest) bool {
	return projectID.MatchString(in.Project) && loginName.MatchString(in.Login) && in.Login != "root" && in.Identity > 0 && ValidTerminalName(in.Name)
}

func validTerminalWindow(in TerminalRequest, now time.Time) bool {
	return in.Expires > now.Unix() && in.Expires <= now.Add(12*time.Hour).Unix()
}

func validTerminalScope(action, scope string) bool {
	creating := action == "reserve" || action == "create"
	if creating {
		return len(scope) == 64 && containerID.MatchString(scope)
	}
	return scope == ""
}

func validListRequest(in TerminalRequest) bool {
	return in.ID == "" && in.Cols == 0 && in.Rows == 0 && in.Name == ""
}

func validSizedTerminalAction(in TerminalRequest) bool {
	return terminalDimensions(in.Cols, in.Rows) && (in.Action != "attach" || in.Name == "")
}

func validIdleTerminalAction(in TerminalRequest) bool {
	return in.Cols == 0 && in.Rows == 0 && (in.Action == "rename" || in.Name == "")
}

func validTerminalAction(in TerminalRequest) bool {
	if in.Action == "list" {
		return validListRequest(in)
	}
	if !terminalID.MatchString(in.ID) {
		return false
	}
	switch in.Action {
	case "reserve", "create", "attach":
		return validSizedTerminalAction(in)
	case "inspect", "end", "rename":
		return validIdleTerminalAction(in)
	}
	return false
}

func (in TerminalRequest) valid(now time.Time) bool {
	return validTerminalActor(in) && validTerminalWindow(in, now) && validTerminalScope(in.Action, in.Scope) && validTerminalAction(in)
}

func validTypedInput(f TerminalFrame) bool {
	data, err := base64.StdEncoding.Strict().DecodeString(f.Data)
	return err == nil && !strings.ContainsAny(f.Data, "\r\n") && len(data) > 0 && len(data) <= 16384 && f.Cols == 0 && f.Rows == 0
}

func validResizeInput(f TerminalFrame) bool {
	return f.Data == "" && terminalDimensions(f.Cols, f.Rows)
}

func validIdleInput(f TerminalFrame) bool {
	return f.Data == "" && f.Cols == 0 && f.Rows == 0
}

func (f TerminalFrame) inputValid() bool {
	if f.Reason != "" || f.Terminals != nil {
		return false
	}
	switch f.Type {
	case "input":
		return validTypedInput(f)
	case "resize":
		return validResizeInput(f)
	case "heartbeat", "close":
		return validIdleInput(f)
	}
	return false
}

func validOutputShape(f TerminalFrame) bool {
	if f.Cols != 0 || f.Rows != 0 {
		return false
	}
	return f.Type == "metadata" || f.Terminals == nil
}

func validTerminalState(state string) bool {
	switch state {
	case "ready", "opening", "ending", "ended":
		return true
	default:
		return false
	}
}

func validTerminalItemFlags(ready, attached bool, state string) bool {
	if ready != (state == "ready") {
		return false
	}
	return !attached || ready
}

func validTerminalItem(item TerminalState, seen map[string]bool) bool {
	if !terminalID.MatchString(item.ID) || seen[item.ID] || !ValidTerminalName(item.Name) {
		return false
	}
	if item.CreatedAt <= 0 || item.CreatedAt > 9007199254740991 {
		return false
	}
	return validTerminalItemFlags(item.Ready, item.Attached, item.State) && validTerminalState(item.State)
}

func validMetadataOutput(f TerminalFrame) bool {
	if f.Data != "" || f.Reason != "" || f.Terminals == nil || len(*f.Terminals) > terminalLimit {
		return false
	}
	seen := make(map[string]bool)
	for _, item := range *f.Terminals {
		if !validTerminalItem(item, seen) {
			return false
		}
		seen[item.ID] = true
	}
	return true
}

func validOutputData(f TerminalFrame) bool {
	if f.Reason != "" {
		return false
	}
	data, err := base64.StdEncoding.Strict().DecodeString(f.Data)
	return err == nil && len(data) > 0 && len(data) <= 4096
}

func validClosedReason(reason string) bool {
	switch reason {
	case "disconnected", "expired", "exited", "launch_failed", "stream_failed", "cleanup_unconfirmed":
		return true
	default:
		return false
	}
}

func validClosedOutput(f TerminalFrame) bool {
	return f.Data == "" && validClosedReason(f.Reason)
}

func (f TerminalFrame) outputValid() bool {
	if !validOutputShape(f) {
		return false
	}
	switch f.Type {
	case "metadata":
		return validMetadataOutput(f)
	case "ready":
		return f.Data == "" && f.Reason == ""
	case "output":
		return validOutputData(f)
	case "closed":
		return validClosedOutput(f)
	default:
		return false
	}
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
	cmd := exec.Command("/usr/bin/podman", "--remote=false", "exec", "--interactive", container, "/usr/bin/python3", "-I", "-c", projectTerminal, in.Action, in.ID, in.Login, strconv.FormatInt(in.Identity, 10), strconv.Itoa(in.Cols), strconv.Itoa(in.Rows), strconv.FormatInt(seconds, 10), fmt.Sprintf("%x", sha256.Sum256([]byte(projectTerminal))), in.Name, in.Scope)
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
		// project-local attachment heartbeat still expires. Neither closing this CLI
		// nor detaching its PTY ends the independently supervised tmux server.
		_ = p.stdin.Close()
		select {
		case <-p.done:
		case <-time.After(3 * time.Second):
			_ = p.cmd.Process.Kill()
		}
		_ = p.stdout.Close()
	})
}

var (
	containerID = regexp.MustCompile(`^[0-9a-f]{64}$`)
	terminalID  = regexp.MustCompile(`^[0-9a-f]{32}$`)
)

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

func (d *Daemon) terminalContainer(ctx context.Context, id string) (string, error) {
	return d.projectContainer(ctx, id, true)
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

func (d *Daemon) projectContainer(ctx context.Context, id string, requireRunning bool) (string, error) {
	if !projectID.MatchString(id) {
		return "", errors.New("invalid project")
	}
	data, err := d.podman(ctx, nil, "--remote=false", "inspect", "--format", terminalInspect, "soda-"+id)
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

func validPrivateTerminalRequest(r *http.Request) bool {
	return r.Method == "GET" && r.URL.RawQuery == "" && !r.URL.ForceQuery && r.URL.RawPath == "" && len(r.Header.Values("Origin")) == 0
}

func (d *Daemon) registerTerminal(r *http.Request, cancel context.CancelFunc) (func(), bool) {
	d.terminalMu.Lock()
	if d.terminalClosed || len(d.terminals) >= 2*terminalLimit {
		d.terminalMu.Unlock()
		return nil, false
	}
	if d.terminals == nil {
		d.terminals = make(map[*http.Request]context.CancelFunc)
	}
	d.terminals[r] = cancel
	d.terminalWG.Add(1)
	d.terminalMu.Unlock()

	cleanup := func() {
		d.terminalWG.Done()
		d.terminalMu.Lock()
		delete(d.terminals, r)
		d.terminalMu.Unlock()
	}
	return cleanup, true
}

func readTerminalRequest(ctx context.Context, conn *websocket.Conn) (TerminalRequest, bool) {
	first, firstCancel := context.WithTimeout(ctx, 5*time.Second)
	kind, body, err := conn.Read(first)
	firstCancel()
	var in TerminalRequest
	if err != nil || kind != websocket.MessageText || len(body) > 4096 || strictjson.Decode(bytes.NewReader(body), &in) != nil || !in.valid(time.Now()) {
		return TerminalRequest{}, false
	}
	return in, true
}

func (d *Daemon) launchTerminal(ctx context.Context, executor terminalExecutor, conn *websocket.Conn, in TerminalRequest) (terminalProcess, bool) {
	inspectCtx, inspectCancel := context.WithTimeout(ctx, 10*time.Second)
	id, err := d.terminalContainer(inspectCtx, in.Project)
	inspectCancel()
	if err != nil {
		writeTerminal(ctx, conn, TerminalFrame{Type: "closed", Reason: "launch_failed"})
		return nil, false
	}
	if ctx.Err() != nil {
		return nil, false
	}
	p, err := executor.terminal(id, in)
	if err != nil {
		writeTerminal(ctx, conn, TerminalFrame{Type: "closed", Reason: "launch_failed"})
		return nil, false
	}
	return p, true
}

func pumpIncomingTerminal(ctx context.Context, conn *websocket.Conn, p terminalProcess, incoming chan<- error) {
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
}

func pumpOutgoingTerminal(ctx context.Context, conn *websocket.Conn, p terminalProcess) {
	for {
		f, err := p.Output()
		if err != nil {
			return
		}
		if err = writeTerminal(ctx, conn, f); err != nil {
			return
		}
		if f.Type == "closed" || f.Type == "metadata" {
			return
		}
	}
}

func pumpTerminalIO(ctx context.Context, conn *websocket.Conn, p terminalProcess) {
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
	go pumpIncomingTerminal(ctx, conn, p, incoming)

	// A disconnect must also stop a blocked native output read.
	go func() {
		select {
		case <-incoming:
			p.Close()
		case <-stopped:
		}
	}()

	pumpOutgoingTerminal(ctx, conn, p)
}

func (d *Daemon) terminalHandler(w http.ResponseWriter, r *http.Request) {
	// This handler is private to the existing filesystem-authorized Unix socket.
	// Browser Origin/cookie/CSRF authorization belongs to the web route.
	if !validPrivateTerminalRequest(r) {
		http.Error(w, "invalid private terminal request", 400)
		return
	}
	executor, ok := d.Exec.(terminalExecutor)
	if !ok {
		http.Error(w, "terminal unavailable", http.StatusServiceUnavailable)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Hour)
	defer cancel()

	cleanup, ok := d.registerTerminal(r, cancel)
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
	conn.SetReadLimit(terminalFrameLimit)

	in, ok := readTerminalRequest(ctx, conn)
	if !ok {
		return
	}
	ctx, expiryCancel := context.WithDeadline(ctx, time.Unix(in.Expires, 0))
	defer expiryCancel()

	p, ok := d.launchTerminal(ctx, executor, conn, in)
	if !ok {
		return
	}
	defer p.Close()

	pumpTerminalIO(ctx, conn, p)
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
