package acceptance

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func writePrivateFixture(t *testing.T, path string, data []byte, mode os.FileMode) {
	t.Helper()
	if err := os.WriteFile(path, data, mode); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, mode); err != nil {
		t.Fatal(err)
	}
}

func TestPrivateFileCapsAndReadsTheOpenedDescriptor(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "fixture")
	writePrivateFixture(t, path, []byte("original"), 0o600)

	data, err := privateFile(path, len("original"))
	if err != nil || string(data) != "original" {
		t.Fatalf("exact-bound input = %q, %v", data, err)
	}
	if _, err := privateFile(path, len("original")-1); err == nil {
		t.Fatal("cap-plus-one input accepted")
	}

	opened, limit, err := openPrivateInput(path, len("original"))
	if err != nil {
		t.Fatal(err)
	}
	moved := filepath.Join(dir, "opened")
	if err := os.Rename(path, moved); err != nil {
		_ = opened.Close()
		t.Fatal(err)
	}
	writePrivateFixture(t, path, []byte("replacement"), 0o600)
	data, readErr := readPrivateInput(opened, limit, path)
	closeErr := opened.Close()
	if readErr != nil || closeErr != nil || string(data) != "original" {
		t.Fatalf("opened descriptor input = %q, read %v, close %v", data, readErr, closeErr)
	}

	// Growth after the opened-file size check is detected with one extra byte.
	opened, limit, err = openPrivateInput(path, len("replacement"))
	if err != nil {
		t.Fatal(err)
	}
	appendFile, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		_ = opened.Close()
		t.Fatal(err)
	}
	if _, err := appendFile.Write([]byte("x")); err != nil {
		_ = appendFile.Close()
		_ = opened.Close()
		t.Fatal(err)
	}
	if err := appendFile.Close(); err != nil {
		_ = opened.Close()
		t.Fatal(err)
	}
	_, readErr = readPrivateInput(opened, limit, path)
	closeErr = opened.Close()
	if readErr == nil || closeErr != nil {
		t.Fatalf("growth read = %v, close %v", readErr, closeErr)
	}
}

func TestPrivateFileRefusesUntrustedAndBlockingInput(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "fixture")
	writePrivateFixture(t, path, []byte("fixture"), 0o600)
	if _, err := privateFile("relative-fixture", 64); err == nil {
		t.Fatal("relative input accepted")
	}

	if err := os.Chmod(path, 0o640); err != nil {
		t.Fatal(err)
	}
	if _, err := privateFile(path, 64); err == nil {
		t.Fatal("group-readable input accepted")
	}
	if err := os.Chmod(path, 0o600); err != nil {
		t.Fatal(err)
	}

	link := filepath.Join(dir, "link")
	if err := os.Symlink(path, link); err != nil {
		t.Fatal(err)
	}
	if _, err := privateFile(link, 64); err == nil {
		t.Fatal("symlink input accepted")
	}

	fifo := filepath.Join(dir, "fifo")
	if err := syscall.Mkfifo(fifo, 0o600); err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	if _, err := privateFile(fifo, 64); err == nil {
		t.Fatal("FIFO input accepted")
	}
	if time.Since(start) > time.Second {
		t.Fatal("FIFO admission blocked")
	}
}

func TestPrivateFileEnforcesGlobalCeilingBeforeRead(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sparse")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Truncate(maxPrivateInputBytes + 1); err != nil {
		_ = file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := privateFile(path, maxPrivateInputBytes+10); err == nil {
		t.Fatal("input above the shared ceiling accepted")
	}
}
