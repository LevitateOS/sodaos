package main

import (
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
	if err != nil || info.Mode().Perm() != 0o660 {
		t.Fatalf("private socket mode: %v, %v", info, err)
	}
	if _, _, err := extensionListener(socket, http.NotFoundHandler()); err == nil {
		t.Fatal("existing socket replaced")
	}
	if _, _, err := extensionListener("relative.sock", http.NotFoundHandler()); err == nil {
		t.Fatal("relative socket accepted")
	}
}
