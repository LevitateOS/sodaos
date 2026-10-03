//go:build linux

package main

import (
	"bytes"
	"context"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory/control"
	"github.com/levitateos/sodaos/internal/store"
)

func operatorRoundTrip(t *testing.T, uid uint32) (*http.Response, error) {
	t.Helper()
	db, err := store.Open(filepath.Join(t.TempDir(), "dashboard.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	coordinator := control.NewCoordinator(db, nil, nil)
	socket := filepath.Join(t.TempDir(), "operator.sock")
	server, listener, err := operatorHTTPServer(socket, uid, coordinator)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close(); _ = server.Close() })
	go func() { _ = server.Serve(listener) }()
	// A bounded client turns a server-side deadlock into a fast failure
	// instead of hanging the whole package until the go test timeout.
	client := &http.Client{Timeout: 15 * time.Second, Transport: &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", socket)
	}}}
	req, err := http.NewRequest(http.MethodPost, "http://soda-operator/operator/factory", bytes.NewReader([]byte(`{"type":"status"}`)))
	if err != nil {
		t.Fatal(err)
	}
	return client.Do(req)
}

func TestOperatorSocketAdmitsConfiguredPeer(t *testing.T) {
	res, err := operatorRoundTrip(t, uint32(os.Geteuid()))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != http.StatusOK {
		t.Fatal("configured peer refused", res.StatusCode)
	}
}

func TestOperatorSocketRebindsStalePath(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "dashboard.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	coordinator := control.NewCoordinator(db, nil, nil)
	socket := filepath.Join(t.TempDir(), "operator.sock")
	first, firstListener, err := operatorHTTPServer(socket, uint32(os.Geteuid()), coordinator)
	if err != nil {
		t.Fatal(err)
	}
	_ = firstListener.Close()
	_ = first.Close()
	second, secondListener, err := operatorHTTPServer(socket, uint32(os.Geteuid()), coordinator)
	if err != nil {
		t.Fatal("restart did not rebind stale socket path:", err)
	}
	t.Cleanup(func() { _ = secondListener.Close(); _ = second.Close() })
}

func TestOperatorSocketRefusesForeignPeer(t *testing.T) {
	foreign := uint32(os.Geteuid()) + 1
	if _, err := operatorRoundTrip(t, foreign); err == nil {
		t.Fatal("foreign peer admitted")
	}
}
