// test_project_keys.py is NOT ported: its subjects
// (internal/host/project/project_keys.py, plus internal/host/terminal/
// project_terminal.py which the test imports for identity) were retired from
// the tree, so the Python test itself fails at import. The behavior moved to
// Runtime.AccessKeys in internal/host/project/access_keys.go, covered by
// internal/host/lifecycle_access_keys_test.go. These guards pin that
// retirement so a resurrected subject or a moved successor fails loudly
// instead of silently dropping coverage.
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

func TestKeysGoSuccessorOwnsKeyBehavior(t *testing.T) {
	successor := ReadFile(t, "internal/host/project/access_keys.go")
	for _, want := range []string{
		"func (r *Runtime) AccessKeys",
		`"native key operation not confirmed"`,
		`"native keys changed or are not managed canonical keys"`,
		"terminal.AgentExec",
	} {
		Check(t, strings.Contains(successor, want), "successor lost %s", want)
	}
	_, err := os.Stat(filepath.Join(RepoRoot, "internal/host/lifecycle_access_keys_test.go"))
	Check(t, err == nil, "successor key tests missing: %v", err)
}
