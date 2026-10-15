package forgejo

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"time"
)

// ManagerControlBounds admits only the frozen manager operations. These
// requests use OS peer mapping, independently of background/browser admission.
func ManagerControlBounds(operation string) (int64, time.Duration, bool) {
	switch operation {
	case "state":
		return 16 << 10, 5 * time.Second, true
	case "snapshot":
		return 16 << 10, 10 * time.Second, true
	case "reconcile":
		return 16 << 10, 300 * time.Second, true
	case "cache":
		return 2 << 20, 600 * time.Second, true
	default:
		return 0, 0, false
	}
}

// ExtensionManager forwards one fixed operation to Fountain's authoritative
// manager. Fountain owns the wire schema and validates the body. Preserve its
// rejection/status and bounded JSON verbatim; never bootstrap an Actor, send a
// PAT, follow redirects, or retransmit an uncertain lifecycle operation.
func (b *ServiceBackground) ExtensionManager(ctx context.Context, operation string, body []byte) (int, []byte, error) {
	limit, budget, ok := ManagerControlBounds(operation)
	if !ok || int64(len(body)) > limit || !json.Valid(body) {
		return 0, nil, errors.New("invalid extension manager request")
	}
	ctx, cancel := context.WithTimeout(ctx, budget)
	defer cancel()
	transport := &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		conn, err := (&net.Dialer{}).DialContext(ctx, "unix", b.socket)
		if err != nil {
			return nil, err
		}
		return b.verifiedPeer(conn)
	}}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, CheckRedirect: rejectBackgroundRedirect}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://extensionHost/v1/extensions/manager/"+operation, bytes.NewReader(body))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	response, err := client.Do(req)
	if err != nil {
		return 0, nil, errors.New("extension manager is unavailable")
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, (128<<10)+1))
	if err != nil || len(data) > 128<<10 || !json.Valid(data) || response.StatusCode != http.StatusOK && (response.StatusCode < 400 || response.StatusCode > 599) {
		return 0, nil, errors.New("invalid extension manager response")
	}
	return response.StatusCode, data, nil
}
