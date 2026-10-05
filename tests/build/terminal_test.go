// test_terminal.py is NOT ported: its subject
// (internal/host/terminal/project_terminal.py) was retired from the tree
// (the PTY/tmux/systemd protocol it tested no longer exists), so the Python
// test itself fails at import. Terminal behavior moved first to the Go
// internal/host/terminal package, then at executor cutover to the Rust
// `soda-host` daemon (rust/soda-host/src/texec.rs, oracle-covered); the
// frame wire surface stayed in Go as internal/host/terminal.go. These
// guards pin that retirement so a resurrected subject or a moved successor
// fails loudly instead of silently dropping coverage.
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

func TestTerminalRustSuccessorOwnsTerminalBehavior(t *testing.T) {
	_, err := os.Stat(filepath.Join(RepoRoot, "internal/host/terminal"))
	Check(t, os.IsNotExist(err), "Go terminal executor resurrected; behavior lives in the Rust daemon")
	executor := ReadFile(t, "rust/soda-host/src/texec.rs")
	for _, want := range []string{
		"`AttachNative`",
		"pub fn attach(container: &str",
	} {
		Check(t, strings.Contains(executor, want), "successor lost %s", want)
	}
	wire := ReadFile(t, "internal/host/terminal.go")
	Check(t, strings.Contains(wire, "func Write("), "successor lost frame Write")
}
