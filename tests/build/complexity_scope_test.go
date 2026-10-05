// Port of test_complexity_scope.py: tools/ is shipping code.
package build

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func runGate(t *testing.T, paths ...string) ProcResult {
	t.Helper()
	args := append([]string{filepath.Join(RepoRoot, "scripts/check-complexity.sh")}, paths...)
	return Run(t, RunOpt{Cwd: RepoRoot, Timeout: 120 * time.Second}, "/bin/bash", args...)
}

func TestComplexityToolsFilesPassAsShipping(t *testing.T) {
	for _, path := range []string{"tools/soda-candidate/main.go", "tools/soda-avatars/main.go"} {
		t.Run(path, func(t *testing.T) {
			result := runGate(t, path)
			Require(t, result.Code == 0, "gate failed: %s", result.Stdout+result.Stderr)
			Check(t, strings.Contains(result.Stdout, "below 10"), "stdout=%q", result.Stdout)
		})
	}
}

func TestComplexityScriptsStillOutOfScope(t *testing.T) {
	result := runGate(t, "scripts/fake.go")
	Require(t, result.Code == 0, "gate failed: %s", result.Stdout+result.Stderr)
	Check(t, strings.Contains(result.Stdout, "no shipping Go files"), "stdout=%q", result.Stdout)
}

func TestComplexityPrecommitRoutesStagedToolsFiles(t *testing.T) {
	var prodline string
	for _, line := range strings.Split(ReadFile(t, ".githooks/pre-commit"), "\n") {
		if strings.HasPrefix(line, "prodgofiles=") {
			prodline = line
			break
		}
	}
	Require(t, prodline != "", "prodgofiles line not found")
	Check(t, strings.Contains(prodline, "tools"), "prodgofiles=%q", prodline)
}
