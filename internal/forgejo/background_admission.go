package forgejo

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"

	extensions "forgejo.org/extension-sdk"

	"github.com/levitateos/sodaos/internal/config"
)

// maxBackgroundBodyBytes bounds one background response, mirroring the
// SDK transport. Anything larger refuses instead of growing the reader.
const maxBackgroundBodyBytes = 64 << 10

func validBackgroundAdmission(token string) bool {
	if len(token) != 43 {
		return false
	}
	for _, c := range token {
		if c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-' || c == '_' {
			continue
		}
		return false
	}
	return true
}

func validBackgroundOperationID(id string) bool {
	if id == "" || len(id) > 128 {
		return false
	}
	for _, c := range id {
		if c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-' || c == '_' || c == '.' || c == ':' {
			continue
		}
		return false
	}
	return true
}

func rejectBackgroundRedirect(*http.Request, []*http.Request) error {
	return http.ErrUseLastResponse
}

func backgroundSecret(credential extensions.CredentialFile) (string, error) {
	secret, err := config.Secret(string(credential))
	if err != nil {
		return "", err
	}
	if len(secret) > 4096 {
		return "", errors.New("background credential file is unavailable")
	}
	return secret, nil
}

func checkBackgroundRecord(b *ServiceBackground, operationID string, record extensions.OperationRecord) error {
	b.mu.Lock()
	pinned := b.pinned
	b.mu.Unlock()
	if pinned == "" || record.InstallationID != pinned || record.OperationID != operationID || record.Outcome == "" {
		return errors.New("invalid background response")
	}
	return nil
}

func (b *ServiceBackground) admissionForCall(ctx context.Context) (string, error) {
	b.mu.Lock()
	admission := b.admission
	b.mu.Unlock()
	if admission != "" {
		return admission, nil
	}
	return b.bootstrap(ctx, false, "")
}

// bootstrap replaces the service admission under the mutex: concurrent
// calls share one bootstrap instead of revoking each other. A rejected
// admission refreshes only while it remains current; explicit force
// refreshes pass no rejected admission and always bootstrap.
func (b *ServiceBackground) bootstrap(ctx context.Context, force bool, rejected string) (string, error) {
	b.mu.Lock()
	if b.socket == "" {
		b.mu.Unlock()
		return "", errors.New("service callback socket is not configured")
	}
	if call := b.bootstrapCall; call != nil && (force || rejected != "" || b.admission == "") {
		b.mu.Unlock()
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-call.done:
			if rejected != "" {
				if call.err != nil {
					return "", call.err
				}
				b.mu.Lock()
				admission := b.admission
				b.mu.Unlock()
				return admission, nil
			}
			if force && !call.force {
				return b.bootstrap(ctx, true, "")
			}
			return call.admission, call.err
		}
	}
	if rejected != "" && b.admission != "" && b.admission != rejected {
		admission := b.admission
		b.mu.Unlock()
		return admission, nil
	}
	if !force && b.admission != "" {
		admission := b.admission
		b.mu.Unlock()
		return admission, nil
	}
	call := &backgroundBootstrap{done: make(chan struct{}), force: force}
	b.bootstrapCall = call
	socket, installation, pinned := b.socket, b.installation, b.pinned
	b.mu.Unlock()

	admission, pinned, err := b.performBootstrap(ctx, socket, installation, pinned)
	b.mu.Lock()
	if err == nil {
		b.pinned = pinned
		b.admission = admission
	}
	call.admission, call.err = admission, err
	b.bootstrapCall = nil
	close(call.done)
	b.mu.Unlock()
	return admission, err
}

func (b *ServiceBackground) performBootstrap(ctx context.Context, socket, installation, pinned string) (string, string, error) {
	transport := &http.Transport{DialContext: func(dialCtx context.Context, _, _ string) (net.Conn, error) {
		conn, err := (&net.Dialer{}).DialContext(dialCtx, "unix", socket)
		if err != nil {
			return nil, err
		}
		return b.verifiedPeer(conn)
	}}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, CheckRedirect: rejectBackgroundRedirect}
	body, err := json.Marshal(struct {
		InstallationID string `json:"installation_id,omitempty"`
	}{InstallationID: installation})
	if err != nil {
		return "", "", err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://extensionHost"+extensions.BackgroundBootstrapPath, bytes.NewReader(body))
	if err != nil {
		return "", "", err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := client.Do(request)
	if err != nil {
		if ctx.Err() != nil {
			return "", "", ctx.Err()
		}
		return "", "", errors.New("background bootstrap request failed")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", "", errors.New("background bootstrap rejected")
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, maxBackgroundBodyBytes+1))
	if err != nil || len(data) > maxBackgroundBodyBytes {
		return "", "", errors.New("invalid background bootstrap response")
	}
	var bootstrap struct {
		Admission      string `json:"admission"`
		InstallationID string `json:"installation_id"`
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&bootstrap); err != nil {
		return "", "", errors.New("invalid background bootstrap response")
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		return "", "", errors.New("invalid background bootstrap response")
	}
	if !validBackgroundAdmission(bootstrap.Admission) || bootstrap.InstallationID == "" {
		return "", "", errors.New("invalid background bootstrap response")
	}
	if installation != "" && installation != bootstrap.InstallationID {
		return "", "", errors.New("background bootstrap installation mismatch")
	}
	if pinned != "" && pinned != bootstrap.InstallationID {
		return "", "", errors.New("background bootstrap installation mismatch")
	}
	return bootstrap.Admission, bootstrap.InstallationID, nil
}

// Every request opens a new connection. Bootstrap authenticates its own
// peer only; authenticate this connection before transmitting credentials.
func (b *ServiceBackground) verifiedPeer(conn net.Conn) (net.Conn, error) {
	peer, err := extensions.PeerCredential(conn)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	if peer.UID != b.hostUID {
		_ = conn.Close()
		return nil, errors.New("service callback host peer is not permitted")
	}
	return conn, nil
}
