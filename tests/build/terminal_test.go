// test_terminal.py is NOT ported: its subject
// (internal/host/terminal/project_terminal.py) was retired from the tree
// (the PTY/tmux/systemd protocol it tested no longer exists), so the Python
// test itself fails at import. Terminal behavior moved to the
// internal/host/terminal package (service, native attach, identity), which
// carries its own Go tests. These guards pin that retirement so a
// resurrected subject or a moved successor fails loudly instead of silently
// dropping coverage.
//
// terminal_assets_test.go (the port of test_terminal_assets.py) is a
// different module and is unaffected by this file.
package build

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTerminalPythonSubjectRetired(t *testing.T) {
	_, err := os.Stat(filepath.Join(RepoRoot, "internal/host/terminal/project_terminal.py"))
	Check(t, os.IsNotExist(err), "project_terminal.py resurrected; port test_terminal.py against it")
}

func TestTerminalGoSuccessorOwnsTerminalBehavior(t *testing.T) {
	for _, name := range []string{"service.go", "native.go", "identity.go", "types.go"} {
		_, err := os.Stat(filepath.Join(RepoRoot, "internal/host/terminal", name))
		Check(t, err == nil, "terminal successor %s missing: %v", name, err)
	}
	Check(t, strings.Contains(ReadFile(t, "internal/host/terminal/native.go"), "func AttachNative"),
		"successor lost AttachNative")
	Check(t, strings.Contains(ReadFile(t, "internal/host/terminal/service.go"), "func Write("),
		"successor lost frame Write")
	for _, name := range []string{"service_test.go", "identity_test.go"} {
		_, err := os.Stat(filepath.Join(RepoRoot, "internal/host/terminal", name))
		Check(t, err == nil, "terminal successor %s missing: %v", name, err)
	}
}
