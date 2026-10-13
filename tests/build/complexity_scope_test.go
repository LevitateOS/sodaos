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
			script := "#!/bin/sh\n[ \"$2\" != -n ] || exit 0\nprintf '%s' " + shellQuote(tc.stdout) + "\nprintf '%s' " + shellQuote(tc.stderr) + " >&2\nexit " + tc.code + "\n"
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

func TestComplexityToolDownloadDiagnosticsAreNotFindings(t *testing.T) {
	dir := t.TempDir()
	fakeGo := filepath.Join(dir, "go")
	script := "#!/bin/sh\n" +
		"if [ \"$2\" = -n ]; then\n" +
		"  printf 'go: downloading github.com/fzipp/gocyclo v0.6.0\\n' >&2\n" +
		"  touch " + shellQuote(filepath.Join(dir, "prepared")) + "\n" +
		"else\n" +
		"  test -f " + shellQuote(filepath.Join(dir, "prepared")) + " || exit 1\n" +
		"fi\n"
	if err := os.WriteFile(fakeGo, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	env := SetEnv(os.Environ(), "PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	result := Run(t, RunOpt{Cwd: RepoRoot, Env: env, Timeout: 10 * time.Second}, "/bin/bash", filepath.Join(RepoRoot, "scripts/check-complexity.sh"), "internal/factory/control/operator.go")
	Require(t, result.Code == 0, "download diagnostic failed gate: %s", result.Stdout+result.Stderr)
	Check(t, strings.Contains(result.Stdout, "below 10"), "stdout=%q", result.Stdout)
	Check(t, strings.Contains(result.Stderr, "go: downloading"), "download diagnostic lost: %q", result.Stderr)
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

func TestTSComplexityGatePreservesAnalyzerFailureAndPaths(t *testing.T) {
	for _, tc := range []struct {
		name, output, diagnostic, status string
		defaultFiles, wantPass           bool
	}{
		{name: "clean", status: "0", wantPass: true},
		{name: "complexity violation", output: "frontend/space name.ts:1:1: error eslint(complexity): Maximum allowed is 9.\n", status: "1"},
		{name: "parser failure", output: "frontend/space name.ts:1:1: error: Expected expression\n", status: "1"},
		{name: "operational failure", diagnostic: "oxlint unavailable\n", status: "2"},
		{name: "default spaced paths", status: "0", wantPass: true, defaultFiles: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			for _, path := range []string{"scripts", "node_modules/.bin"} {
				if err := os.MkdirAll(filepath.Join(dir, path), 0o755); err != nil {
					t.Fatal(err)
				}
			}
			gate := filepath.Join(dir, "scripts/check-ts-complexity.sh")
			if err := os.WriteFile(gate, []byte(ReadFile(t, "scripts/check-ts-complexity.sh")), 0o700); err != nil {
				t.Fatal(err)
			}
			tool := "#!/bin/sh\n" +
				"[ \"$#\" -eq 4 ] && [ \"$1\" = -A ] && [ \"$2\" = correctness ] && [ \"$3\" = --format=agent ] && [ \"$4\" = 'frontend/space name.ts' ] || { echo 'wrong profile or paths' >&2; exit 2; }\n" +
				"printf '%s' " + shellQuote(tc.output) + "\nprintf '%s' " + shellQuote(tc.diagnostic) + " >&2\nexit " + tc.status + "\n"
			bin := filepath.Join(dir, "node_modules/.bin")
			if err := os.WriteFile(filepath.Join(bin, "oxlint"), []byte(tool), 0o700); err != nil {
				t.Fatal(err)
			}
			args := []string{gate, "frontend/space name.ts", "scripts/out-of-scope.ts", "frontend/fixtures/ignored.ts"}
			env := os.Environ()
			if tc.defaultFiles {
				fakeGit := "#!/bin/sh\n[ \"$1\" = ls-files ] && [ \"$2\" = -z ] || exit 2\nprintf 'frontend/space name.ts\\0scripts/out-of-scope.ts\\0frontend/fixtures/ignored.ts\\0'\n"
				if err := os.WriteFile(filepath.Join(bin, "git"), []byte(fakeGit), 0o700); err != nil {
					t.Fatal(err)
				}
				env = SetEnv(env, "PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
				args = []string{gate}
			}
			result := Run(t, RunOpt{Cwd: dir, Env: env, Timeout: 10 * time.Second}, "/bin/bash", args...)
			if (result.Code == 0) != tc.wantPass || strings.Contains(result.Stderr, "wrong profile or paths") {
				t.Fatalf("gate status=%d stdout=%q stderr=%q", result.Code, result.Stdout, result.Stderr)
			}
			if tc.diagnostic != "" && !strings.Contains(result.Stderr, strings.TrimSpace(tc.diagnostic)) {
				t.Fatalf("analyzer diagnostic lost: %q", result.Stderr)
			}
			if tc.output != "" && !strings.Contains(result.Stdout, strings.TrimSpace(tc.output)) {
				t.Fatalf("analyzer output lost: %q", result.Stdout)
			}
		})
	}
}
