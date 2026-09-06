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
	cmd        *exec.Cmd
	done       chan struct{}
	err        error
	waitErr    error
	cleanupErr error
	mu         sync.Mutex // closes signalling before reaping the pinned group leader
	sealed     bool
	once       sync.Once
	stopErr    error
}

func StartProcess(ctx context.Context, c Command, out, stderr io.Writer) (*Process, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := ownedGroupsSupported(); err != nil {
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
	go func() {
		// WNOWAIT leaves the exited leader unreaped. Its PID/PGID cannot be reused
		// while we terminate any remaining group members, even on normal parent exit.
		observed := waitOwnedExit(cmd.Process.Pid)
		p.mu.Lock()
		var cleanup error
		if !errors.Is(observed, syscall.ECHILD) {
			// On an observation error, terminate our still-unreaped child too.
			cleanup = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		}
		if errors.Is(cleanup, syscall.ESRCH) {
			cleanup = nil
		}
		p.sealed = true
		p.mu.Unlock()
		// Never hold the signalling lock across a potentially uninterruptible
		// kernel wait. Stop can still time out and report incomplete cleanup.
		p.cleanupErr = errors.Join(observed, cleanup)
		p.waitErr = cmd.Wait()
		p.err = errors.Join(p.cleanupErr, p.waitErr)
		close(p.done)
	}()
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
func (p *Process) signal(sig syscall.Signal) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.sealed {
		return nil
	}
	err := syscall.Kill(-p.cmd.Process.Pid, sig)
	if errors.Is(err, syscall.ESRCH) {
		return nil
	}
	return err
}
func (p *Process) Stop() error {
	p.once.Do(func() {
		if err := p.signal(syscall.SIGTERM); err != nil {
			p.stopErr = err
			return
		}
		select {
		case <-p.done:
			p.stopErr = p.err
			return
		case <-time.After(10 * time.Second):
		}
		err := p.signal(syscall.SIGKILL)
		select {
		case <-p.done:
			p.stopErr = errors.Join(fmt.Errorf("owned group required forced termination: %w", context.DeadlineExceeded), err, p.err)
		case <-time.After(5 * time.Second):
			p.stopErr = errors.Join(err, errors.New("owned group cleanup did not complete"))
		}
	})
	return p.stopErr
}
