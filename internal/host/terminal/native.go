package terminal

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"time"

	"github.com/levitateos/sodaos/internal/strictjson"
)

// The program is fixed product code, not a request-selected command or file.
//
//go:embed project_terminal.py
var ProjectTerminal string

// Process is a streaming native attachment. No caller can supply an executable,
// shell arguments, environment or host flags.
type Process interface {
	Input(TerminalFrame) error
	Output() (TerminalFrame, error)
	Close()
}

type attachLauncher interface {
	Terminal(string, TerminalRequest) (Process, error)
}

type nativeTerminal struct {
	cmd     *exec.Cmd
	stdin   *os.File
	stdout  io.ReadCloser
	scanner *bufio.Scanner
	done    chan struct{}
	once    sync.Once
}

// AttachNative starts the fixed podman/python attachment bridge.
func AttachNative(container string, in TerminalRequest) (Process, error) {
	seconds := in.Expires - time.Now().Unix()
	if !containerID.MatchString(container) || !in.Valid(time.Now()) || seconds < 1 {
		return nil, errors.New("invalid terminal target")
	}
	cmd := exec.Command("/usr/bin/podman", "--remote=false", "exec", "--interactive", container, "/usr/bin/python3", "-I", "-c", ProjectTerminal, in.Action, in.ID, in.Login, strconv.FormatInt(in.Identity, 10), strconv.Itoa(in.Cols), strconv.Itoa(in.Rows), strconv.FormatInt(seconds, 10), fmt.Sprintf("%x", sha256.Sum256([]byte(ProjectTerminal))), in.Name, in.Scope)
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
	p.scanner.Buffer(make([]byte, 4096), FrameLimit)
	go func() { _ = cmd.Wait(); close(p.done) }()
	return p, nil
}

func (p *nativeTerminal) Input(f TerminalFrame) error {
	if !f.InputValid() {
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
	if err := strictjson.Decode(bytes.NewReader(p.scanner.Bytes()), &f); err != nil || !f.OutputValid() {
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
