// test_project_os_observation.py is NOT ported: its subject
// (internal/host/project/project_os.py) was retired from the tree, so the
// Python test itself fails at import. The behavior moved to parseOSRelease /
// Runtime.ObserveOS in internal/host/project/os.go, covered by
// internal/host/os_test.go. These guards pin that retirement so a
// resurrected subject or a moved successor fails loudly instead of silently
// dropping coverage.
package build

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOSPythonSubjectRetired(t *testing.T) {
	_, err := os.Stat(filepath.Join(RepoRoot, "internal/host/project/project_os.py"))
	Check(t, os.IsNotExist(err), "project_os.py resurrected; port test_project_os_observation.py against it")
}

func TestOSGoSuccessorOwnsObservationBehavior(t *testing.T) {
	successor := ReadFile(t, "internal/host/project/os.go")
	for _, want := range []string{
		"func parseOSRelease",
		"func (r *Runtime) ObserveOS",
		`"oversized OS release file"`,
		`"ambiguous OS release field"`,
		`"missing OS version"`,
		"PRETTY_NAME",
	} {
		Check(t, strings.Contains(successor, want), "successor lost %s", want)
	}
	_, err := os.Stat(filepath.Join(RepoRoot, "internal/host/os_test.go"))
	Check(t, err == nil, "successor os tests missing: %v", err)
}
