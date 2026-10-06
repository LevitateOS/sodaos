// test_project_keys.py is NOT ported: its subjects
// (internal/host/project/project_keys.py, plus internal/host/terminal/
// project_terminal.py which the test imports for identity) were retired from
// the tree, so the Python test itself fails at import. The behavior moved
// first to Go Runtime.AccessKeys, then at executor cutover to the Rust
// `soda-host` daemon (lib/host/src/account/mod.rs, oracle-covered).
// These guards pin that retirement so a resurrected subject or a moved
// successor fails loudly instead of silently dropping coverage.
package build

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestKeysPythonSubjectRetired(t *testing.T) {
	for _, rel := range []string{
		"internal/host/project/project_keys.py",
		"internal/host/terminal/project_terminal.py",
	} {
		_, err := os.Stat(filepath.Join(RepoRoot, rel))
		Check(t, os.IsNotExist(err), "%s resurrected; port test_project_keys.py against it", rel)
	}
}

func TestKeysRustSuccessorOwnsKeyBehavior(t *testing.T) {
	_, err := os.Stat(filepath.Join(RepoRoot, "internal/host/project/access_keys.go"))
	Check(t, os.IsNotExist(err), "Go key successor resurrected; behavior lives in the Rust daemon")
	successor := ReadFile(t, "lib/host/src/account/mod.rs")
	for _, want := range []string{
		`"native key operation not confirmed"`,
		`"native keys changed or are not managed canonical keys"`,
		"AgentExec",
	} {
		Check(t, strings.Contains(successor, want), "successor lost %s", want)
	}
}
