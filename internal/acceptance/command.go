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
	Artifacts      map[string]string
}

func validCommandLabel(label string) bool {
	return regexp.MustCompile(`^[a-zA-Z0-9_-]+$`).MatchString(label)
}

func commandEvidenceWriters(e *Evidence, label string) (io.WriteCloser, io.WriteCloser, error) {
	out, err := e.Writer(label + ".stdout")
	if err != nil {
		return nil, nil, err
	}
	stderr, err := e.Writer(label + ".stderr")
	if err != nil {
		return nil, nil, errors.Join(err, out.Close())
	}
	return out, stderr, nil
}

func closeCommandWriters(writers ...io.Closer) error {
	var err error
	for _, w := range writers {
		err = errors.Join(err, w.Close())
	}
	return err
}

func waitCommandProcess(ctx context.Context, p *Process) error {
	runErr := p.Wait(ctx)
	if ctx.Err() != nil {
		runErr = errors.Join(runErr, ctx.Err(), p.Stop())
	}
	return runErr
}

func processStillOwned(p *Process) bool {
	select {
	case <-p.Done():
		return false
	default:
		return true
	}
}

type commandCaptures struct {
	oa, ob, out, stderr io.Closer
}

func (c commandCaptures) Close() error {
	return closeCommandWriters(c.oa, c.ob, c.out, c.stderr)
}

func closeWritersWhenDone(p *Process, captures commandCaptures) {
	go func() { <-p.Done(); _ = captures.Close() }()
}

func pipeFailureAfterSuccess(ctx context.Context, p *Process) bool {
	return p.cmd.ProcessState.Success() && p.waitErr != nil && ctx.Err() == nil
}

func commandExitOutcome(ctx context.Context, p *Process, runErr, writeErr error) (*int, error, error) {
	if p.cmd.ProcessState == nil {
		return nil, runErr, writeErr
	}
	code := p.cmd.ProcessState.ExitCode()
	if pipeFailureAfterSuccess(ctx, p) {
		// Copy/pipe failures after exit zero belong to retention, not a
		// native denial. Group cleanup failures remain execution failures.
		return &code, p.cleanupErr, errors.Join(writeErr, p.waitErr)
	}
	return &code, runErr, writeErr
}

func redactCommandResult(e *Evidence, stdout, stderr []byte, runErr, writeErr error, exitCode *int) (Result, error) {
	if runErr != nil {
		runErr = e.RedactError(fmt.Errorf("command execution: %w", runErr))
	}
	return Result{Stdout: stdout, Stderr: stderr, Err: runErr, Started: true, ExitCode: exitCode}, e.RedactError(writeErr)
}

// Execute does not print argv or stdin. Result.Err is execution, the returned
// error is evidence retention. Both must be checked, including expected denials.
func Execute(ctx context.Context, e *Evidence, label string, c Command) (Result, error) {
	if !validCommandLabel(label) {
		return Result{}, errors.New("invalid command label")
	}
	out, stderr, err := commandEvidenceWriters(e, label)
	if err != nil {
		return Result{}, err
	}
	var a, b bytes.Buffer
	// Captured bytes are sanitized, too; callers never receive secret-bearing logs.
	oa := &redactingWriter{out: nopCloser{io.MultiWriter(out, &a)}, secrets: e.secrets}
	ob := &redactingWriter{out: nopCloser{io.MultiWriter(stderr, &b)}, secrets: e.secrets}
	p, runErr := StartProcess(ctx, c, oa, ob)
	captures := commandCaptures{oa, ob, out, stderr}
	if p == nil {
		return Result{Err: e.RedactError(runErr)}, e.RedactError(captures.Close())
	}
	runErr = waitCommandProcess(ctx, p)
	if processStillOwned(p) {
		closeWritersWhenDone(p, captures)
		return Result{Err: e.RedactError(runErr), Started: true}, errors.New("evidence still owned by incomplete process cleanup")
	}
	exitCode, runErr, writeErr := commandExitOutcome(ctx, p, runErr, captures.Close())
	return redactCommandResult(e, a.Bytes(), b.Bytes(), runErr, writeErr, exitCode)
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

func validSSHPort(port int) bool {
	return port >= 1 && port <= 65535
}

func validSSHUser(user string) bool {
	return regexp.MustCompile(`^[a-z_][a-z0-9_-]*$`).MatchString(user)
}

func validSSHHost(host string) bool {
	return net.ParseIP(host) != nil || regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9.-]*$`).MatchString(host)
}

func trustedKnownHosts(path string) error {
	if !filepath.IsAbs(path) {
		return errors.New("absolute pinned known_hosts required")
	}
	st, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !regularUnwritableFile(st) || st.Size() == 0 {
		return errors.New("trusted regular known_hosts required")
	}
	return nil
}

func (r Remote) Args() ([]string, error) {
	if !validSSHUser(r.User) || !validSSHPort(r.Port) {
		return nil, errors.New("invalid SSH user/port")
	}
	if !validSSHHost(r.Host) {
		return nil, errors.New("invalid SSH host")
	}
	if _, err := PrivateFile(r.Key); err != nil {
		return nil, err
	}
	if err := trustedKnownHosts(r.KnownHosts); err != nil {
		return nil, err
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
	if _, err := exec.LookPath("ssh"); err != nil {
		return err
	}
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
