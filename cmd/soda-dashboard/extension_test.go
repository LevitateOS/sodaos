package main

import (
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

func TestExtensionListenerUsesPrivateSocketWithoutReplacingFiles(t *testing.T) {
	root, err := filepath.Abs("../../.artifacts/tmp")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(root, 0o700); err != nil {
		t.Fatal(err)
	}
	dir, err := os.MkdirTemp(root, "ipc-")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(dir) }()
	socket := filepath.Join(dir, "s.sock")
	_, listener, err := extensionListener(socket, http.NotFoundHandler())
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	info, err := os.Stat(socket)
	// 662: group rw for the IPC group plus other-write so the bare
	// Forgejo extension uid (supplemental groups dropped by su-exec)
	// can connect; request auth stays at the protocol layer.
	if err != nil || info.Mode().Perm() != 0o662 {
		t.Fatalf("private socket mode: %v, %v", info, err)
	}
	if _, _, err := extensionListener(socket, http.NotFoundHandler()); err == nil {
		t.Fatal("existing socket replaced")
	}
	if _, _, err := extensionListener("relative.sock", http.NotFoundHandler()); err == nil {
		t.Fatal("relative socket accepted")
	}
}

func TestExtensionListenerReclaimsStaleSocket(t *testing.T) {
	root, err := filepath.Abs("../../.artifacts/tmp")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(root, 0o700); err != nil {
		t.Fatal(err)
	}
	dir, err := os.MkdirTemp(root, "ipc-")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(dir) }()
	socket := filepath.Join(dir, "s.sock")
	stale, err := net.ListenUnix("unix", &net.UnixAddr{Name: socket, Net: "unix"})
	if err != nil {
		t.Fatal(err)
	}
	stale.SetUnlinkOnClose(false)
	if err := stale.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(socket); err != nil {
		t.Fatalf("stale socket fixture missing: %v", err)
	}
	_, listener, err := extensionListener(socket, http.NotFoundHandler())
	if err != nil {
		t.Fatalf("stale socket not reclaimed: %v", err)
	}
	defer listener.Close()
}
