// Installed client-probe support shared by the Go ports of
// tests/installed/*.py: private inputs, bounded local commands, and the exact
// failure-line shape the retired Python entrypoints printed. This file owns
// only the shared mechanics; each probe's validation and evidence stay in its
// own concern file. It never logs private contents.
package acceptance

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
	"time"
)

// probeFailure renders the retired Python entrypoints' exact stderr contract:
// a fixed prefix plus the causal failure type. The causal message is withheld
// unless detail carries a reportable operational message, so private request
// contents can never leak through the failure line.
type probeFailure struct {
	prefix string
	suffix string
	cause  error
	detail string
}

func (e *probeFailure) Error() string {
	line := e.prefix + fmt.Sprintf("%T", e.cause) + e.suffix
	if e.detail == "" {
		return line
	}
	return line + " " + e.detail
}

func (e *probeFailure) Unwrap() error { return e.cause }

// fail annotates a probe error with its Python-compatible failure prefix.
func fail(prefix string, err error) error {
	return &probeFailure{prefix: prefix, cause: err}
}

// failParen annotates a probe error with a parenthesized failure shape.
func failParen(prefix, suffix string, err error) error {
	return &probeFailure{prefix: prefix, suffix: suffix, cause: err}
}

// failDetail annotates a probe error whose operational message is safe to
// report alongside the failure type.
func failDetail(prefix, detail string, err error) error {
	return &probeFailure{prefix: prefix, cause: err, detail: detail}
}

// privateFile reads a restricted regular file with an explicit size bound.
// PrivateFile already enforces absolute path, regular non-symlink, 0600-style
// modes and a 1 MiB ceiling; the limit preserves each probe's tighter bound.
func privateFile(path string, limit int) ([]byte, error) {
	data, err := PrivateFile(path)
	if err != nil {
		return nil, err
	}
	if len(data) > limit {
		return nil, fmt.Errorf("restricted input exceeds bound: %s", path)
	}
	return data, nil
}

// privateDir validates a probe fixture directory: absolute, a real directory
// (never a symlink), and inaccessible to group/others. Probes that ran with an
// explicit owner check also require current-UID ownership.
func privateDir(path string, owner bool) error {
	if !filepath.IsAbs(path) {
		return errors.New("absolute fixture directory required")
	}
	st, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !st.IsDir() || st.Mode()&os.ModeSymlink != 0 {
		return errors.New("fixture directory must be a real directory")
	}
	if st.Mode().Perm()&0o077 != 0 {
		return errors.New("fixture directory must be inaccessible to group/others")
	}
	if owner && !ownedByCaller(st) {
		return errors.New("fixture directory must be owned by the caller")
	}
	return nil
}

// RestrictUmask applies the 077 creation mask the interactive probes set
// before touching probe output.
func RestrictUmask() { syscall.Umask(0o077) }

// ownedByCaller reports whether st is owned by the current UID.
func ownedByCaller(st os.FileInfo) bool {
	stat, ok := st.Sys().(*syscall.Stat_t)
	if !ok {
		return false
	}
	return stat.Uid == uint32(os.Getuid())
}

// runOutcome captures a bounded command's result. ExitCode is negative when
// the command never produced an exit status.
type runOutcome struct {
	stdout   []byte
	stderr   []byte
	exitCode int
}

// runBounded executes a local command with a hard timeout, capturing both
// streams. It never prints argv or stdin.
func runBounded(name string, args []string, stdin []byte, timeout time.Duration) (runOutcome, error) {
	return runBoundedEnv(name, args, stdin, nil, timeout)
}

// runBoundedEnv is runBounded with extra plain environment entries.
func runBoundedEnv(name string, args []string, stdin []byte, extraEnv []string, timeout time.Duration) (runOutcome, error) {
	return runBoundedDirEnv(name, args, stdin, extraEnv, "", timeout)
}

// runBoundedDirEnv is runBoundedEnv with a fixed working directory for
// tools that resolve paths from the cwd. Empty dir inherits the caller.
func runBoundedDirEnv(name string, args []string, stdin []byte, extraEnv []string, dir string, timeout time.Duration) (runOutcome, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	runCtx, cancelRun := context.WithCancel(ctx)
	defer cancelRun()
	cmd := exec.Command(name, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	if stdin != nil {
		cmd.Stdin = bytes.NewReader(stdin)
	}
	if extraEnv != nil {
		cmd.Env = append(os.Environ(), extraEnv...)
	}
	stdout := boundedCapture{limit: probeStdoutLimit, onLimit: cancelRun}
	stderr := boundedCapture{limit: probeStderrLimit, onLimit: cancelRun}
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	process, err := StartCommand(runCtx, cmd)
	if err != nil {
		return runOutcome{exitCode: -1}, err
	}
	waitErr := process.Wait(runCtx)
	if waitErr != nil && (errors.Is(waitErr, context.DeadlineExceeded) || errors.Is(waitErr, context.Canceled)) {
		cleanupErr := process.Stop()
		select {
		case <-process.Done():
		default:
			return runOutcome{exitCode: -1}, errors.Join(waitErr, cleanupErr, errors.New("owned process cleanup remains unresolved"))
		}
		outcome := boundedOutcome(process, &stdout, &stderr)
		return outcome, errors.Join(waitErr, cleanupErr, stdout.err, stderr.err)
	}
	outcome := boundedOutcome(process, &stdout, &stderr)
	if stdout.err != nil || stderr.err != nil {
		return outcome, errors.Join(stdout.err, stderr.err, process.cleanupErr, process.waitErr)
	}
	if process.cleanupErr != nil {
		return outcome, errors.Join(process.cleanupErr, process.waitErr)
	}
	if process.waitErr == nil {
		return outcome, nil
	}
	var exitErr *exec.ExitError
	if errors.As(process.waitErr, &exitErr) {
		return outcome, nil
	}
	return outcome, process.waitErr
}

const (
	probeStdoutLimit = 8 * 1024 * 1024
	probeStderrLimit = 1 * 1024 * 1024
)

var errProbeOutputLimit = errors.New("command output exceeded capture bound")

type boundedCapture struct {
	buffer  bytes.Buffer
	limit   int
	err     error
	onLimit context.CancelFunc
}

func (capture *boundedCapture) Write(data []byte) (int, error) {
	remaining := capture.limit - capture.buffer.Len()
	if remaining <= 0 {
		capture.err = errProbeOutputLimit
		capture.onLimit()
		return 0, capture.err
	}
	if len(data) > remaining {
		_, _ = capture.buffer.Write(data[:remaining])
		capture.err = errProbeOutputLimit
		capture.onLimit()
		return remaining, capture.err
	}
	return capture.buffer.Write(data)
}

func boundedOutcome(process *Process, stdout, stderr *boundedCapture) runOutcome {
	outcome := runOutcome{
		stdout:   stdout.buffer.Bytes(),
		stderr:   stderr.buffer.Bytes(),
		exitCode: -1,
	}
	if process.cmd.ProcessState != nil {
		outcome.exitCode = process.cmd.ProcessState.ExitCode()
	}
	return outcome
}

// uuidHex returns 32 lowercase hex digits with RFC 4122 version-4 bits, the
// Go equivalent of Python's uuid.uuid4().hex run identifiers.
func uuidHex() (string, error) {
	var id [16]byte
	if _, err := io.ReadFull(rand.Reader, id[:]); err != nil {
		return "", err
	}
	id[6] = id[6]&0x0f | 0x40
	id[8] = id[8]&0x3f | 0x80
	return hex.EncodeToString(id[:]), nil
}

// machineArch reports the uname-style machine name the Python probes recorded
// (platform.machine), keeping evidence stable across the language port.
func machineArch() string {
	switch runtime.GOARCH {
	case "amd64":
		return "x86_64"
	case "386":
		return "i686"
	case "arm64":
		return "aarch64"
	default:
		return runtime.GOARCH
	}
}
