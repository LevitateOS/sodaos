package workspace

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHarnessRejectsChangedBytesBeforeExecution(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "bin"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "bin/codex"), []byte("changed"), 0o700); err != nil {
		t.Fatal(err)
	}
	w := Runtime{Config: Config{HarnessDirectory: root, HarnessSHA256: strings.Repeat("a", 64)}, Exec: executorFunc(func(context.Context, []byte, string, ...string) ([]byte, error) {
		t.Fatal("changed harness executed")
		return nil, nil
	})}
	if err := w.CheckHarness(t.Context()); err == nil {
		t.Fatal("changed executable accepted")
	}
}

func TestHarnessRejectsVersionMismatch(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "bin"), 0o700); err != nil {
		t.Fatal(err)
	}
	data := []byte("synthetic executable")
	if err := os.WriteFile(filepath.Join(root, "bin/codex"), data, 0o700); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(data)
	w := Runtime{Config: Config{HarnessDirectory: root, HarnessSHA256: hex.EncodeToString(digest[:]), HarnessVersion: "0.153.4"}, Exec: executorFunc(func(_ context.Context, _ []byte, executable string, args ...string) ([]byte, error) {
		if executable != filepath.Join(root, "bin/codex") || len(args) != 1 || args[0] != "--version" {
			t.Fatal("checked different executable")
		}
		return []byte("codex-cli 0.150.0"), nil
	})}
	if err := w.CheckHarness(t.Context()); err == nil {
		t.Fatal("wrong executable version accepted")
	}
}
