// Process-group ownership adapted from soda-os bc1d3e0. No PID-file adoption.
package acceptance

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

type Process struct {
	cmd     *exec.Cmd
	done    chan struct{}
	err     error
	once    sync.Once
	stopErr error
}

func StartProcess(ctx context.Context, c Command, out, stderr io.Writer) (*Process, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	cmd := exec.Command(c.Name, c.Args...)
	cmd.Dir = c.Dir
	cmd.Env = append(os.Environ(), c.Env...)
	cmd.Stdin = c.Stdin
	cmd.Stdout = out
	cmd.Stderr = stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.WaitDelay = 2 * time.Second
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	p := &Process{cmd: cmd, done: make(chan struct{})}
	go func() { p.err = cmd.Wait(); close(p.done) }()
	return p, nil
}
func (p *Process) Wait(ctx context.Context) error {
	select {
	case <-p.done:
		return p.err
	case <-ctx.Done():
		return ctx.Err()
	}
}
func (p *Process) Done() <-chan struct{} { return p.done }
func (p *Process) Stop() error {
	p.once.Do(func() {
		select {
		case <-p.done:
			p.stopErr = p.err
			return
		default:
		}
		// Only this still-owned child group is signalled; cancellation cannot skip it.
		err := syscall.Kill(-p.cmd.Process.Pid, syscall.SIGTERM)
		if err != nil && !errors.Is(err, syscall.ESRCH) {
			p.stopErr = err
			return
		}
		select {
		case <-p.done:
			return
		case <-time.After(10 * time.Second):
		}
		err = syscall.Kill(-p.cmd.Process.Pid, syscall.SIGKILL)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		select {
		case <-p.done:
			p.stopErr = fmt.Errorf("guest required forced termination: %w", errOrTimeout(err))
		case <-ctx.Done():
			p.stopErr = errors.Join(err, ctx.Err())
		}
	})
	return p.stopErr
}
func errOrTimeout(err error) error {
	if err != nil {
		return err
	}
	return context.DeadlineExceeded
}
