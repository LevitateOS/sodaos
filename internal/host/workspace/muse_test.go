package workspace

import (
	"net"
	"os"
	"path/filepath"
	"testing"
)

func TestMuseInterfaceRefusesPrivateSiblingFiles(t *testing.T) {
	root, err := os.MkdirTemp(os.TempDir(), "mi-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(root) })
	socket := filepath.Join(root, "launch.sock")
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	if err := validateMuseInterface(socket); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "admin.sock"), []byte("synthetic private interface"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := validateMuseInterface(socket); err == nil {
		t.Fatal("private sibling would enter the workspace")
	}
}

func TestMuseCleanupPreservesUnretiredCredentialRoot(t *testing.T) {
	root := t.TempDir()
	run := "synthetic-run"
	credential := filepath.Join(root, run, "auth.json")
	if err := os.Mkdir(filepath.Dir(credential), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(credential, []byte("synthetic private input"), 0o600); err != nil {
		t.Fatal(err)
	}
	w := Runtime{Config: Config{MuseCredentialRoot: root}}
	if err := w.cleanupMuse(run); err == nil {
		t.Fatal("unretired credentials were removed")
	}
	if _, err := os.Stat(credential); err != nil {
		t.Fatal("uncertain custody was lost", err)
	}
	if err := os.Remove(credential); err != nil {
		t.Fatal(err)
	}
	if err := w.cleanupMuse(run); err != nil {
		t.Fatal(err)
	}
}
