package codex

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/levitateos/sodaos/internal/identity"
)

type Session struct {
	mu      sync.Mutex
	writeMu sync.Mutex
	cmd     *exec.Cmd
	in      io.WriteCloser
	root    string
	state   identity.Enrollment
	replies map[int64]chan message
	next    int64
	done    chan struct{}
	closed  bool
}
type message struct {
	ID     int64           `json:"id"`
	Method string          `json:"method"`
	Result json.RawMessage `json:"result"`
	Error  json.RawMessage `json:"error"`
	Params json.RawMessage `json:"params"`
}

func (p *Provider) Start(ctx context.Context, label string) (identity.EnrollmentSession, error) {
	s, err := p.createSession()
	if err != nil {
		return nil, err
	}
	if err = s.initialize(ctx); err != nil {
		_ = s.Close()
		return nil, err
	}
	if err = s.startDevice(ctx); err != nil {
		_ = s.Close()
		return nil, err
	}
	go s.expire()
	return s, nil
}

func (s *Session) initialize(ctx context.Context) error {
	if _, err := s.call(ctx, "initialize", map[string]any{"clientInfo": map[string]string{"name": "soda_identity_broker", "version": "1"}}); err != nil {
		return err
	}
	return s.send(map[string]any{"method": "initialized", "params": map[string]any{}})
}

func (s *Session) startDevice(ctx context.Context) error {
	raw, err := s.call(ctx, "account/login/start", map[string]string{"type": "chatgptDeviceCode"})
	if err != nil {
		return err
	}
	var wire struct {
		Type            string `json:"type"`
		LoginID         string `json:"loginId"`
		VerificationURL string `json:"verificationUrl"`
		UserCode        string `json:"userCode"`
	}
	if err = json.Unmarshal(raw, &wire); err != nil {
		return err
	}
	if wire.Type != "chatgptDeviceCode" || wire.LoginID == "" || wire.VerificationURL != "https://auth.openai.com/codex/device" || wire.UserCode == "" {
		return errors.New("invalid device enrollment response")
	}
	s.mu.Lock()
	s.state.ID = wire.LoginID
	s.state.VerificationURL = wire.VerificationURL
	s.state.UserCode = wire.UserCode
	s.mu.Unlock()
	return nil
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

func (p *Provider) createSession() (*Session, error) {
	root, err := os.MkdirTemp(p.config.Root, "enrollment-")
	if err != nil {
		return nil, err
	}
	s := &Session{root: root, replies: map[int64]chan message{}, done: make(chan struct{}), state: identity.Enrollment{State: "pending"}}
	s.cmd = exec.Command(p.config.Binary, "-c", `cli_auth_credentials_store="file"`, "-c", `sqlite_home="`+filepath.Join(root, "state")+`"`, "-c", `log_dir="`+filepath.Join(root, "logs")+`"`, "app-server")
	s.cmd.Env = p.environment(root)
	s.cmd.Dir = root
	s.in, err = s.cmd.StdinPipe()
	if err != nil {
		_ = os.RemoveAll(root)
		return nil, err
	}
	out, err := s.cmd.StdoutPipe()
	if err != nil {
		_ = os.RemoveAll(root)
		return nil, err
	}
	// Enrollment diagnostics and credentials remain confined to the tmpfs root.
	log, err := os.OpenFile(filepath.Join(root, "stderr"), os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		_ = os.RemoveAll(root)
		return nil, err
	}
	s.cmd.Stderr = log
	if err = s.cmd.Start(); err != nil {
		_ = log.Close()
		_ = os.RemoveAll(root)
		return nil, err
	}
	go func() { s.read(out); _ = s.cmd.Process.Kill(); _ = s.cmd.Wait(); _ = log.Close(); close(s.done) }()
	return s, nil
}

func (s *Session) send(msg any) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	return json.NewEncoder(s.in).Encode(msg)
}

func (s *Session) call(ctx context.Context, method string, params any) (json.RawMessage, error) {
	s.mu.Lock()
	s.next++
	id := s.next
	ch := make(chan message, 1)
	s.replies[id] = ch
	s.mu.Unlock()
	defer func() { s.mu.Lock(); delete(s.replies, id); s.mu.Unlock() }()
	if err := s.send(map[string]any{"id": id, "method": method, "params": params}); err != nil {
		return nil, err
	}
	timer := time.NewTimer(20 * time.Second)
	defer timer.Stop()
	select {
	case msg := <-ch:
		if len(msg.Error) > 0 && string(msg.Error) != "null" {
			return nil, protocolError(msg.Error)
		}
		return msg.Result, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-timer.C:
		return nil, errors.New("codex protocol timed out")
	case <-s.done:
		return nil, errors.New("codex process stopped")
	}
}

func protocolError(raw json.RawMessage) error {
	var detail struct {
		Message string `json:"message"`
	}
	_ = json.Unmarshal(raw, &detail)
	message := strings.ToLower(detail.Message)
	if strings.Contains(message, "device") && (strings.Contains(message, "not enabled") || strings.Contains(message, "disabled")) {
		return errors.New("device-code login is disabled; enable it in ChatGPT security settings or workspace permissions")
	}
	return errors.New("codex protocol request failed")
}

func (s *Session) read(out io.Reader) {
	scanner := bufio.NewScanner(out)
	scanner.Buffer(make([]byte, 4096), 512<<10)
	for scanner.Scan() {
		var msg message
		if json.Unmarshal(scanner.Bytes(), &msg) != nil {
			break
		}
		s.mu.Lock()
		if msg.ID != 0 {
			if ch := s.replies[msg.ID]; ch != nil {
				ch <- msg
			}
		} else {
			s.notify(msg)
		}

		s.mu.Unlock()
	}
	s.mu.Lock()
	if s.state.State == "pending" {
		s.state.State = "failed"
		s.state.Error = "Provider enrollment ended"
	}
	s.mu.Unlock()
}

func (s *Session) notify(msg message) {
	if msg.Method != "account/login/completed" {
		return
	}
	var event struct {
		LoginID string `json:"loginId"`
		Success bool   `json:"success"`
	}
	if json.Unmarshal(msg.Params, &event) != nil {
		return
	}
	if s.state.ID != "" && event.LoginID != s.state.ID {
		return
	}
	if event.Success {
		s.state.State = "completed"
		return
	}
	s.state.State = "failed"
	s.state.Error = "Provider enrollment failed; retry or check device-code access"
}

func (s *Session) Snapshot() identity.Enrollment { s.mu.Lock(); defer s.mu.Unlock(); return s.state }

func (s *Session) Finish(ctx context.Context) (identity.Connection, []byte, error) {
	if s.Snapshot().State != "completed" {
		return identity.Connection{}, nil, errors.New("enrollment is incomplete")
	}
	conn, err := s.account(ctx)
	if err != nil {
		return conn, nil, err
	}
	if err = s.stop(); err != nil {
		return conn, nil, err
	}
	data, err := s.credentialFile()
	return conn, data, err
}

func (s *Session) account(ctx context.Context) (identity.Connection, error) {
	raw, err := s.call(ctx, "account/read", map[string]bool{"refreshToken": false})
	if err != nil {
		return identity.Connection{}, err
	}
	var response struct {
		Account *struct {
			Type  string  `json:"type"`
			Email *string `json:"email"`
			Plan  string  `json:"planType"`
		} `json:"account"`
	}
	if err = json.Unmarshal(raw, &response); err != nil || response.Account == nil || response.Account.Type != "chatgpt" {
		return identity.Connection{}, errors.New("managed ChatGPT enrollment required")
	}
	c := identity.Connection{Plan: response.Account.Plan}
	if response.Account.Email != nil {
		c.Email = *response.Account.Email
	}
	return c, nil
}

func (s *Session) credentialFile() ([]byte, error) {
	info, err := os.Lstat(filepath.Join(s.root, "auth.json"))
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0o077 != 0 || info.Size() > 256<<10 {
		return nil, errors.New("invalid credential file")
	}
	file, err := os.Open(filepath.Join(s.root, "auth.json"))
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()
	data, err := io.ReadAll(io.LimitReader(file, 256<<10+1))
	if err != nil || !identity.CredentialValid(data) {
		return nil, errors.New("invalid credential state")
	}
	return data, nil
}

func (s *Session) stop() error {
	s.mu.Lock()
	if !s.closed {
		s.closed = true
		_ = s.cmd.Process.Kill()
	}
	s.mu.Unlock()
	select {
	case <-s.done:
		return nil
	case <-time.After(5 * time.Second):
		return identity.ErrUncertain
	}
}

func (s *Session) Close() error {
	state := s.Snapshot()
	if state.State == "pending" && state.ID != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		_, _ = s.call(ctx, "account/login/cancel", map[string]string{"loginId": state.ID})
		cancel()
	}
	err := s.stop()
	if err == nil {
		err = os.RemoveAll(s.root)
	}
	return err
}
