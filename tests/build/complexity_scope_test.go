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
			Check(t, strings.Contains(result.Stdout, "Complexity gate passed"), "stdout=%q", result.Stdout)
		})
	}
}

func TestComplexityScriptsStillOutOfScope(t *testing.T) {
	result := runGate(t, "scripts/pg_runtime_test.go")
	Require(t, result.Code == 0, "gate failed: %s", result.Stdout+result.Stderr)
	Check(t, strings.Contains(result.Stdout, "go: 0 functions"), "stdout=%q", result.Stdout)
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
		{name: "parse failure", stderr: "gocyclo: fixture.go:1:1: expected package, found EOF\n", code: "1", wantError: "expected package, found EOF"},
		{name: "missing path warning", stdout: "gocyclo: could not get file info for path fixture.go: stat fixture.go: no such file or directory\n", code: "0", wantError: "Complexity gate passed"},
		{name: "valid violation", stdout: `[{"path":"fixture.go","name":"fixture","line":1,"score":25}]`, code: "0", wantError: "fixture complexity 25"},
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
				Check(t, strings.Contains(result.Stderr, tc.wantError), "stderr=%q", result.Stderr)
			} else if tc.name == "missing path warning" {
				Require(t, result.Code != 0, "analyzer failure passed: %s", result.Stdout+result.Stderr)
				Check(t, !strings.Contains(result.Stdout, tc.wantError), "stdout=%q", result.Stdout)
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
		"printf 'go: downloading github.com/fzipp/gocyclo v0.6.0\\n' >&2\n" +
		"printf '[]'\n" +
		"exit 0\n"
	if err := os.WriteFile(fakeGo, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	env := SetEnv(os.Environ(), "PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	result := Run(t, RunOpt{Cwd: RepoRoot, Env: env, Timeout: 10 * time.Second}, "/bin/bash", filepath.Join(RepoRoot, "scripts/check-complexity.sh"), "internal/factory/control/operator.go")
	Require(t, result.Code == 0, "download diagnostic failed gate: %s", result.Stdout+result.Stderr)
	Check(t, strings.Contains(result.Stdout, "Complexity gate passed"), "stdout=%q", result.Stdout)
	Check(t, strings.Contains(result.Stderr, "go: downloading"), "download diagnostic lost: %q", result.Stderr)
}

func TestComplexityDefaultDiscoveryFailureFailsClosed(t *testing.T) {
	dir := t.TempDir()
	fakeBin := filepath.Join(dir, "bin")
	if err := os.MkdirAll(fakeBin, 0o755); err != nil {
		t.Fatal(err)
	}
	fakeGit := filepath.Join(fakeBin, "git")
	if err := os.WriteFile(fakeGit, []byte("#!/bin/sh\nif [ \"$1\" = rev-parse ]; then exec command -p git \"$@\"; fi\necho 'discovery unavailable' >&2\nexit 1\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	env := SetEnv(os.Environ(), "PATH", fakeBin+string(os.PathListSeparator)+os.Getenv("PATH"))
	env = SetEnv(env, "TMPDIR", dir)
	result := Run(t, RunOpt{Cwd: RepoRoot, Env: env, Timeout: 120 * time.Second}, "/bin/bash", filepath.Join(RepoRoot, "scripts/check-complexity.sh"))
	Require(t, result.Code != 0, "default discovery failure passed: %s", result.Stdout+result.Stderr)
	Check(t, strings.Contains(result.Stderr, "discovery unavailable"), "stderr=%q", result.Stderr)
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

func TestComplexityPrecommitRoutesStagedToolsFiles(t *testing.T) {
	hook := ReadFile(t, ".githooks/pre-commit")
	Check(t, strings.Contains(hook, "scripts/check-complexity.sh --staged"), "pre-commit complexity gate=%q", hook)
}

func TestTSComplexityGatePreservesAnalyzerFailureAndPaths(t *testing.T) {
	writeTSFixture := func(t *testing.T, dir, name, body string) string {
		t.Helper()
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		abs, err := filepath.Abs(path)
		if err != nil {
			t.Fatal(err)
		}
		return abs
	}
	dir := t.TempDir()
	// Absolute fixture paths: the gate resolves inputs against the
	// repository root, so fixtures must exist on disk.
	violation := writeTSFixture(t, dir, "violation file.ts",
		"export function big(a: number): number {\n\tlet x = 0;\n"+strings.Repeat("\tif (a) x++;\n", 21)+"\treturn x;\n}\n")
	broken := writeTSFixture(t, dir, "broken file.ts", "const x = ;\n")
	cleanFile := writeTSFixture(t, dir, "clean file.ts", "export function tiny(a: number): number {\n\treturn a;\n}\n")
	emptyBin := filepath.Join(dir, "emptybin")
	if err := os.MkdirAll(emptyBin, 0o755); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(dir, "node_modules/.bin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	gate := filepath.Join(RepoRoot, "scripts/check-ts-complexity.sh")
	for _, tc := range []struct {
		name       string
		args       []string
		wantPass   bool
		wantStderr string
	}{
		{name: "clean", args: []string{gate, cleanFile}, wantPass: true},
		{name: "complexity violation", args: []string{gate, violation}, wantStderr: "complexity 22"},
		{name: "parser failure", args: []string{gate, broken}, wantStderr: broken},
		{name: "operational failure", args: []string{gate, cleanFile}, wantStderr: "bun unavailable"},
		{name: "default spaced paths", args: []string{gate}, wantPass: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			env := os.Environ()
			if tc.name == "default spaced paths" {
				fakeGit := "#!/bin/sh\nif [ \"$1\" = rev-parse ]; then exec command -p git \"$@\"; fi\nif [ \"$1\" = show ]; then printf ''\nelse printf 'frontend/space name.ts\\0scripts/out-of-scope.ts\\0frontend/fixtures/ignored.ts\\0'\nfi\n"
				if err := os.WriteFile(filepath.Join(bin, "git"), []byte(fakeGit), 0o700); err != nil {
					t.Fatal(err)
				}
				env = SetEnv(env, "PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
			}
			if tc.name == "operational failure" {
				if err := os.WriteFile(filepath.Join(emptyBin, "bun"), []byte("#!/bin/sh\necho 'bun unavailable' >&2\nexit 2\n"), 0o700); err != nil {
					t.Fatal(err)
				}
				env = SetEnv(env, "PATH", emptyBin+string(os.PathListSeparator)+"/usr/bin:/bin")
			}
			result := Run(t, RunOpt{Cwd: dir, Env: env, Timeout: 60 * time.Second}, "/bin/bash", tc.args...)
			if (result.Code == 0) != tc.wantPass {
				t.Fatalf("gate status=%d stdout=%q stderr=%q", result.Code, result.Stdout, result.Stderr)
			}
			if tc.wantStderr != "" && !strings.Contains(result.Stderr, tc.wantStderr) {
				t.Fatalf("analyzer diagnostic lost: %q", result.Stderr)
			}
		})
	}
}
