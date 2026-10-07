// Port of test_complexity_scope.py: tools/ is shipping code.
package build

import (
	"os"
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
	for _, path := range []string{"tools/soda-rootfs-server/main.go", "tools/soda-avatars/main.go"} {
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

func TestComplexityMissingSelectedInputFails(t *testing.T) {
	result := runGate(t, "internal/does-not-exist.go")
	Require(t, result.Code != 0, "missing selected input passed: %s", result.Stdout+result.Stderr)
	Check(t, strings.Contains(result.Stderr, "selected input is missing"), "stderr=%q", result.Stderr)
}

func TestComplexityAnalyzerFailuresCannotPass(t *testing.T) {
	for _, tc := range []struct {
		name, stdout, stderr string
		code                 string
		wantError            string
	}{
		{name: "parse failure", stderr: "gocyclo: fixture.go:1:1: expected package, found EOF\n", code: "1", wantError: "analysis did not complete"},
		{name: "missing path warning", stderr: "gocyclo: could not get file info for path fixture.go: stat fixture.go: no such file or directory\n", code: "0", wantError: "unexpected output"},
		{name: "valid violation", stdout: "10 package function fixture.go:1:1\n", code: "1"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			fakeGo := filepath.Join(dir, "go")
			script := "#!/bin/sh\nprintf '%s' " + shellQuote(tc.stdout) + "\nprintf '%s' " + shellQuote(tc.stderr) + " >&2\nexit " + tc.code + "\n"
			if err := os.WriteFile(fakeGo, []byte(script), 0o700); err != nil {
				t.Fatal(err)
			}
			env := SetEnv(os.Environ(), "PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
			result := Run(t, RunOpt{Cwd: RepoRoot, Env: env, Timeout: 120 * time.Second}, "/bin/bash", filepath.Join(RepoRoot, "scripts/check-complexity.sh"), "internal/factory/control/operator.go")
			if tc.name == "valid violation" {
				Require(t, result.Code != 0, "valid violation passed")
				Check(t, strings.Contains(result.Stdout, "10 package function"), "stdout=%q", result.Stdout)
			} else {
				Require(t, result.Code != 0, "analyzer failure passed: %s", result.Stdout+result.Stderr)
				Check(t, strings.Contains(result.Stderr, tc.wantError), "stderr=%q", result.Stderr)
			}
		})
	}
}

func TestComplexityDefaultDiscoveryFailureFailsClosed(t *testing.T) {
	dir := t.TempDir()
	fakeBin := filepath.Join(dir, "bin")
	if err := os.MkdirAll(fakeBin, 0o755); err != nil {
		t.Fatal(err)
	}
	fakeFind := filepath.Join(fakeBin, "find")
	if err := os.WriteFile(fakeFind, []byte("#!/bin/sh\nexit 23\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	env := SetEnv(os.Environ(), "PATH", fakeBin+string(os.PathListSeparator)+os.Getenv("PATH"))
	env = SetEnv(env, "TMPDIR", dir)
	result := Run(t, RunOpt{Cwd: RepoRoot, Env: env, Timeout: 120 * time.Second}, "/bin/bash", filepath.Join(RepoRoot, "scripts/check-complexity.sh"))
	Require(t, result.Code != 0, "default discovery failure passed: %s", result.Stdout+result.Stderr)
	Check(t, strings.Contains(result.Stderr, "could not discover Go files"), "stderr=%q", result.Stderr)
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
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
