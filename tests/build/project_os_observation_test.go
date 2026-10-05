// test_project_os_observation.py is NOT ported: its subject
// (internal/host/project/project_os.py) was retired from the tree, so the
// Python test itself fails at import. The behavior moved first to Go
// parseOSRelease / Runtime.ObserveOS, then at executor cutover to the Rust
// `soda-host` daemon (rust/soda-host/src/project.rs, oracle-covered).
// These guards pin that retirement so a resurrected subject or a moved
// successor fails loudly instead of silently dropping coverage.
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

func TestOSRustSuccessorOwnsObservationBehavior(t *testing.T) {
	_, err := os.Stat(filepath.Join(RepoRoot, "internal/host/project/os.go"))
	Check(t, os.IsNotExist(err), "Go OS successor resurrected; behavior lives in the Rust daemon")
	successor := ReadFile(t, "rust/soda-host/src/project.rs")
	for _, want := range []string{
		"pub fn parse_os_release",
		"pub fn observe_os",
		`"4097"`,
		"raw.len() <= 4096",
		`"ambiguous OS release field"`,
		`"missing OS version"`,
		"PRETTY_NAME",
	} {
		Check(t, strings.Contains(successor, want), "successor lost %s", want)
	}
}
