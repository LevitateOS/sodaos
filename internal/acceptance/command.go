// SSH shell quoting and separated command/evidence results adapted from
// soda-os bc1d3e0 internal/acceptance/{remote,command}.go.
package acceptance

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type Command struct {
	Name  string
	Args  []string
	Dir   string
	Stdin io.Reader
	Env   []string
}
type Result struct {
	Stdout, Stderr []byte
	Err            error
	Started        bool
	ExitCode       *int
}

// Execute does not print argv or stdin. Result.Err is execution, the returned
// error is evidence retention. Both must be checked, including expected denials.
func Execute(ctx context.Context, e *Evidence, label string, c Command) (Result, error) {
	if !regexp.MustCompile(`^[a-zA-Z0-9_-]+$`).MatchString(label) {
		return Result{}, errors.New("invalid command label")
	}
	out, err := e.Writer(label + ".stdout")
	if err != nil {
		return Result{}, err
	}
	stderr, err := e.Writer(label + ".stderr")
	if err != nil {
		return Result{}, errors.Join(err, out.Close())
	}
	var a, b bytes.Buffer
	// Captured bytes are sanitized, too; callers never receive secret-bearing logs.
	oa := &redactingWriter{out: nopCloser{io.MultiWriter(out, &a)}, secrets: e.secrets}
	ob := &redactingWriter{out: nopCloser{io.MultiWriter(stderr, &b)}, secrets: e.secrets}
	cmd := exec.CommandContext(ctx, c.Name, c.Args...)
	cmd.Dir = c.Dir
	cmd.Stdin = c.Stdin
	cmd.Env = append(os.Environ(), c.Env...)
	cmd.Stdout = oa
	cmd.Stderr = ob
	cmd.WaitDelay = 10 * time.Second
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		err := syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM)
		if errors.Is(err, syscall.ESRCH) {
			return os.ErrProcessDone
		}
		return err
	}
	runErr := cmd.Run()
	writeErr := errors.Join(oa.Close(), ob.Close(), out.Close(), stderr.Close())
	var exitCode *int
	if cmd.ProcessState != nil {
		code := cmd.ProcessState.ExitCode()
		exitCode = &code
		if cmd.ProcessState.Success() && runErr != nil && ctx.Err() == nil {
			// A successful child plus pipe/WaitDelay failure is not a native denial.
			writeErr = errors.Join(writeErr, runErr)
			runErr = nil
		}
	}
	runErr = errors.Join(runErr, ctx.Err())
	if runErr != nil {
		runErr = e.RedactError(fmt.Errorf("command execution: %w", runErr))
	}
	return Result{Stdout: a.Bytes(), Stderr: b.Bytes(), Err: runErr, Started: cmd.Process != nil, ExitCode: exitCode}, e.RedactError(writeErr)
}

type nopCloser struct{ io.Writer }

func (n nopCloser) Close() error { return nil }

func Quote(args []string) string {
	quoted := make([]string, len(args))
	for i, s := range args {
		quoted[i] = "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
	}
	return strings.Join(quoted, " ")
}

type Remote struct {
	User, Host, Key, KnownHosts string
	Port                        int
	Timeout                     time.Duration `json:"-"`
}

func (r Remote) Args() ([]string, error) {
	if !regexp.MustCompile(`^[a-z_][a-z0-9_-]*$`).MatchString(r.User) || r.Port < 1 || r.Port > 65535 {
		return nil, errors.New("invalid SSH user/port")
	}
	if net.ParseIP(r.Host) == nil && !regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9.-]*$`).MatchString(r.Host) {
		return nil, errors.New("invalid SSH host")
	}
	if _, err := PrivateFile(r.Key); err != nil {
		return nil, err
	}
	if !filepath.IsAbs(r.KnownHosts) {
		return nil, errors.New("absolute pinned known_hosts required")
	}
	st, err := os.Lstat(r.KnownHosts)
	if err != nil {
		return nil, err
	}
	if !st.Mode().IsRegular() || st.Mode().Perm()&0022 != 0 || st.Size() == 0 {
		return nil, errors.New("trusted regular known_hosts required")
	}
	return []string{"-F", "/dev/null", "-T", "-o", "BatchMode=yes", "-o", "IdentitiesOnly=yes", "-o", "StrictHostKeyChecking=yes", "-o", "GlobalKnownHostsFile=/dev/null", "-o", "UserKnownHostsFile=" + r.KnownHosts, "-o", "ConnectTimeout=10", "-o", "ServerAliveInterval=15", "-o", "ServerAliveCountMax=2", "-i", r.Key, "-p", strconv.Itoa(r.Port), r.User + "@" + r.Host}, nil
}
func (r Remote) Command(args []string, input io.Reader) (Command, error) {
	base, err := r.Args()
	if err != nil {
		return Command{}, err
	}
	duration := r.Timeout
	if duration == 0 {
		duration = 30 * time.Minute
	}
	if duration <= 0 || duration > 24*time.Hour {
		return Command{}, errors.New("bounded remote deadline required")
	}
	bounded := append([]string{"timeout", "--signal=TERM", "--kill-after=10s", strconv.FormatFloat(duration.Seconds(), 'f', 3, 64) + "s"}, args...)
	return Command{Name: "ssh", Args: append(base, Quote(bounded)), Stdin: input}, nil
}
func (r Remote) WaitReady(ctx context.Context) error {
	args, err := r.Args()
	if err != nil {
		return err
	}
	for {
		cctx, cancel := context.WithTimeout(ctx, 12*time.Second)
		cmd := exec.CommandContext(cctx, "ssh", append(args, "true")...)
		cmd.WaitDelay = time.Second
		err = cmd.Run()
		cancel()
		if err == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("pinned SSH readiness: %w", ctx.Err())
		case <-time.After(time.Second):
		}
	}
}
