package main

import (
	"context"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// FW38/T47 proves the bounded listener lifetime: the fixed socket
// binds with group-restricted permissions, a live socket is never
// replaced, a stale socket from an unclean shutdown is reclaimed,
// and unlink/rebind reconnects without new run authority.
func unixClient(socket string) *http.Client {
	return &http.Client{Transport: &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "unix", socket)
		},
	}}
}

func TestFactoryMCPListenerLifecycle(t *testing.T) {
	root := t.TempDir()
	socket := filepath.Join(root, "factory-ipc", "mcp.sock")
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/mcp" || r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		_, _ = w.Write([]byte(`{}`))
	})

	server, listener, err := factoryMCPListener(socket, handler)
	if err != nil {
		t.Fatalf("first bind refused: %v", err)
	}
	defer listener.Close()
	go func() { _ = server.Serve(listener) }()
	defer server.Close()

	info, err := os.Lstat(socket)
	if err != nil {
		t.Fatalf("socket missing: %v", err)
	}
	if info.Mode().Type() != os.ModeSocket {
		t.Fatal("listener path is not a socket")
	}
	if perm := info.Mode().Perm(); perm != 0o660 {
		t.Fatalf("socket mode = %o, want 660", perm)
	}

	roundtrip := func() {
		t.Helper()
		res, err := unixClient(socket).Post("http://factory/mcp", "application/json", strings.NewReader(`{}`))
		if err != nil {
			t.Fatalf("unix roundtrip: %v", err)
		}
		defer res.Body.Close()
		if _, err := io.Copy(io.Discard, res.Body); err != nil {
			t.Fatal(err)
		}
		if res.StatusCode != http.StatusOK {
			t.Fatalf("unix roundtrip = %d, want 200", res.StatusCode)
		}
	}
	roundtrip()

	// A live socket is never replaced.
	if _, _, err := factoryMCPListener(socket, handler); err == nil {
		t.Fatal("live socket replaced")
	}
	// A non-socket file is never replaced either.
	blocker := filepath.Join(root, "factory-ipc", "blocker.sock")
	if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := factoryMCPListener(blocker, handler); err == nil {
		t.Fatal("non-socket file replaced")
	}

	// Unclean shutdown leaves a stale socket file with no listener:
	// disable unlink-on-close (a clean Close would remove the file)
	// and close the listener. Reclaim binds a fresh listener on the
	// same path and reconnects with no new authority issued.
	unixListener, ok := listener.(*net.UnixListener)
	if !ok {
		t.Fatal("factory listener is not a unix listener")
	}
	unixListener.SetUnlinkOnClose(false)
	_ = listener.Close()
	_ = server.Close()
	if info, err := os.Lstat(socket); err != nil || info.Mode().Type() != os.ModeSocket {
		t.Fatal("unclean shutdown left no stale socket file to reclaim")
	}
	if factoryMCPSocketLive(socket) {
		t.Fatal("closed listener still answers")
	}
	rebound, second, err := factoryMCPListener(socket, handler)
	if err != nil {
		t.Fatalf("stale socket not reclaimed: %v", err)
	}
	defer second.Close()
	go func() { _ = rebound.Serve(second) }()
	defer rebound.Close()
	roundtrip()
}

func TestFactoryMCPListenerRequiresAbsolutePath(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {})
	if _, _, err := factoryMCPListener("", handler); err == nil {
		t.Fatal("empty socket admitted")
	}
	if _, _, err := factoryMCPListener("relative/mcp.sock", handler); err == nil {
		t.Fatal("relative socket admitted")
	}
}
