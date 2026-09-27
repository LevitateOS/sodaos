package muse

import (
	"bufio"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sync"
	"time"

	"github.com/levitateos/sodaos/internal/identity"
)

var deviceURL = regexp.MustCompile(`https://auth\.meta\.com/oauth/device/\?code=([A-Z0-9]{4}-[A-Z0-9]{4})`)

type Session struct {
	mu    sync.Mutex
	cmd   *exec.Cmd
	root  string
	state identity.Enrollment
	done  chan struct{}
}

func (p *Provider) Start(ctx context.Context, label string) (identity.EnrollmentSession, error) {
	root, err := os.MkdirTemp(p.config.Root, "enrollment-")
	if err != nil {
		return nil, err
	}
	s := &Session{root: root, done: make(chan struct{}), state: identity.Enrollment{ID: filepath.Base(root), ProviderID: identity.Muse, State: "pending"}}
	// Cancellation of the HTTP request must not cancel an admitted enrollment.
	s.cmd = exec.Command(p.config.Binary, "login")
	s.cmd.Env = environment(root)
	s.cmd.Dir = root
	out, err := s.cmd.StdoutPipe()
	if err != nil {
		_ = os.RemoveAll(root)
		return nil, err
	}
	// Never relay native diagnostics: they may carry credentials or presentation.
	s.cmd.Stderr = nil
	if err = s.cmd.Start(); err != nil {
		_ = os.RemoveAll(root)
		return nil, errors.New("muse enrollment could not start")
	}
	go s.read(out)
	go s.expire()

	return s, nil
}

func (s *Session) Snapshot() identity.Enrollment { s.mu.Lock(); defer s.mu.Unlock(); return s.state }

func (s *Session) Finish(context.Context) (identity.Connection, []byte, error) {
	if s.Snapshot().State != "completed" {
		return identity.Connection{}, nil, errors.New("muse enrollment is incomplete")
	}
	data, err := credentialFile(filepath.Join(s.root, "config", "muse", "auth.json"))
	return identity.Connection{ProviderID: identity.Muse}, data, err
}

func (s *Session) Close() error {
	s.mu.Lock()
	if s.state.State == "pending" {
		s.state.State = "canceled"
	}
	s.mu.Unlock()
	_ = s.cmd.Process.Kill()
	select {
	case <-s.done:
		return os.RemoveAll(s.root)
	case <-time.After(5 * time.Second):
		return identity.ErrUncertain
	}
}

func (s *Session) read(out io.Reader) {
	reader := bufio.NewReader(out)
	// Native prompts need not end with a newline. Keep only a bounded window
	// long enough to recognize the fixed device URL.
	var window []byte
	for {
		ch, err := reader.ReadByte()
		if err != nil {
			break
		}
		window = append(window, ch)
		if len(window) > 128 {
			window = window[len(window)-128:]
		}
		if match := deviceURL.FindSubmatch(window); match != nil {
			s.mu.Lock()
			s.state.VerificationURL = string(match[0])
			s.state.UserCode = string(match[1])
			s.mu.Unlock()
		}
	}
	s.complete(s.cmd.Wait())
	close(s.done)
}

func (s *Session) complete(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state.State != "pending" {
		return
	}
	if err == nil {
		s.state.State = "completed"
		return
	}
	s.state.State = "failed"
	s.state.Error = "Provider enrollment failed"
}

func (s *Session) expire() {
	timer := time.NewTimer(10 * time.Minute)
	defer timer.Stop()
	select {
	case <-timer.C:
		_ = s.Close()
	case <-s.done:
	}
}
