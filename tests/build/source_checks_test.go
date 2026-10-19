// Port of test_source_checks.py: source-command sequencing without compilers.
package build

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// fakeTool is the PATH-tool double: POSIX shell that logs its invocation
// as one JSON object per line and fails
// the FAIL_COMMAND with 7. Argument JSON-escaping covers backslash,
// double-quote and newline; nothing else in the fixture needs escapes.
const fakeTool = `name=$(basename "$0")
json="\"$name\""
for a in "$@"; do
	e=$(printf '%s' "$a" | sed -e 's/\\/\\\\/g' -e 's/"/\\"/g' -e ':a;N;$!ba;s/\n/\\n/g')
	json="$json,\"$e\""
done
printf '{"command": [%s], "cwd": "%s", "env": {"GOWORK": "%s", "GOFLAGS": "%s", "CGO_ENABLED": "%s", "GOTOOLCHAIN": "%s"}}\n' "$json" "$(pwd -P)" "$GOWORK" "$GOFLAGS" "$CGO_ENABLED" "$GOTOOLCHAIN" >> "$COMMAND_LOG"
if [ "$name $*" = "$FAIL_COMMAND" ]; then exit 7; fi
`

func gateStub(gate string) string {
	return "#!/bin/bash\n" +
		`printf '{"command": ["bash", "scripts/` + gate + `"], "cwd": "%s", ` +
		`"env": {"GOWORK": "%s", "GOFLAGS": "%s", "CGO_ENABLED": "%s", "GOTOOLCHAIN": "%s"}}\n' ` +
		`"$PWD" "$GOWORK" "$GOFLAGS" "$CGO_ENABLED" "$GOTOOLCHAIN" >> "$COMMAND_LOG"` + "\n" +
		`if [ "bash scripts/` + gate + `" = "${FAIL_COMMAND:-}" ]; then exit 7; fi` + "\n"
}

var expectedSourceCommands = [][]string{
	{"go", "mod", "verify"},
	{"go", "test", "-mod=readonly", "./..."},
	{"bash", "scripts/check-sql-locality.sh"},
	{"bash", "scripts/check-no-npm.sh"},
	{"bash", "scripts/check-no-python.sh"},
	{"bash", "scripts/check-no-unit-tests.sh"},
	{"bash", "scripts/check-go-unitban.sh"},
	{"bun", "run", "typecheck"},
	{"bun", "run", "build:forgejo"},
	{"bun", "run", "test:frontend:prepared"},
	{"bun", "run", "test:forgejo:prepared"},
}

type sourceFixture struct {
	root  string
	tools string
	log   string
}

func newSourceFixture(t *testing.T) sourceFixture {
	t.Helper()
	tmp := TempDir(t)
	root, err := filepath.EvalSymlinks(tmp)
	Require(t, err == nil, "resolve tempdir: %v", err)
	scripts := filepath.Join(root, "scripts")
	Require(t, os.Mkdir(scripts, 0o755) == nil, "mkdir scripts")
	data, err := os.ReadFile(filepath.Join(RepoRoot, "scripts/check-source.sh"))
	Require(t, err == nil, "read check-source.sh: %v", err)
	WriteFile(t, filepath.Join(scripts, "check-source.sh"), data, 0o644)
	WriteFile(t, filepath.Join(root, "go.mod"), []byte("module fixture\n\ngo 1.26.7\n"), 0o644)
	for _, gate := range []string{"check-sql-locality.sh", "check-no-npm.sh", "check-no-python.sh", "check-no-unit-tests.sh", "check-go-unitban.sh"} {
		WriteFile(t, filepath.Join(scripts, gate), []byte(gateStub(gate)), 0o644)
	}
	tools := filepath.Join(root, "tools")
	Require(t, os.Mkdir(tools, 0o755) == nil, "mkdir tools")
	for _, name := range []string{"go", "bun"} {
		WriteFile(t, filepath.Join(tools, name), []byte("#!/bin/bash\n"+fakeTool), 0o700)
	}
	return sourceFixture{root: root, tools: tools, log: filepath.Join(root, "commands.jsonl")}
}

type loggedCommand struct {
	Command []string
	Cwd     string
	Env     map[string]any
}

func (f sourceFixture) run(t *testing.T, fail string, suites ...string) ProcResult {
	t.Helper()
	env := SetEnv(os.Environ(), "PATH", f.tools+string(os.PathListSeparator)+os.Getenv("PATH"))
	env = SetEnv(env, "COMMAND_LOG", f.log)
	env = SetEnv(env, "FAIL_COMMAND", fail)
	env = SetEnv(env, "GOTOOLCHAIN", "auto")
	args := append([]string{filepath.Join(f.root, "scripts/check-source.sh")}, suites...)
	return Run(t, RunOpt{Cwd: f.tools, Env: env, Timeout: 30 * time.Second}, "/bin/bash", args...)
}

func (f sourceFixture) commands(t *testing.T) []loggedCommand {
	t.Helper()
	data, err := os.ReadFile(f.log)
	if os.IsNotExist(err) {
		return nil
	}
	Require(t, err == nil, "read command log: %v", err)
	var out []loggedCommand
	for _, line := range strings.Split(string(data), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var raw struct {
			Command []string
			Cwd     string
			Env     map[string]any
		}
		Require(t, json.Unmarshal([]byte(line), &raw) == nil, "parse log line")
		out = append(out, loggedCommand(raw))
	}
	return out
}

func commandNames(commands []loggedCommand) [][]string {
	var out [][]string
	for _, entry := range commands {
		out = append(out, entry.Command)
	}
	return out
}

func equalCommands(a, b [][]string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if len(a[i]) != len(b[i]) {
			return false
		}
		for j := range a[i] {
			if a[i][j] != b[i][j] {
				return false
			}
		}
	}
	return true
}

func TestSourceCommandsWorkWithoutAStageOrGitCheckout(t *testing.T) {
	for _, tc := range []struct {
		name     string
		args     []string
		commands [][]string
	}{
		{"default", nil, expectedSourceCommands},
		{"all", []string{"all"}, expectedSourceCommands},
		{"go", []string{"go"}, expectedSourceCommands[:7]},
		{"browser", []string{"browser"}, expectedSourceCommands[7:]},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fixture := newSourceFixture(t)
			result := fixture.run(t, "", tc.args...)
			Require(t, result.Code == 0, "check-source failed: %s", result.Stderr)
			commands := fixture.commands(t)
			Check(t, equalCommands(commandNames(commands), tc.commands), "commands = %v", commandNames(commands))
			for _, entry := range commands {
				Check(t, entry.Cwd == fixture.root, "cwd = %q", entry.Cwd)
				want := map[string]any{"GOWORK": "off", "GOFLAGS": "-mod=readonly", "CGO_ENABLED": "0", "GOTOOLCHAIN": "local"}
				Check(t, len(entry.Env) == len(want), "env = %v", entry.Env)
				for key, value := range want {
					Check(t, entry.Env[key] == value, "env[%s] = %v", key, entry.Env[key])
				}
			}
			Check(t, strings.Contains(result.Stdout, "Local source checks executed"), "stdout=%q", result.Stdout)
		})
	}
}

func TestSourceFailuresContinueIndependentSuites(t *testing.T) {
	fixture := newSourceFixture(t)
	for _, suite := range []string{"all", "go", "browser"} {
		commands := expectedSourceCommands
		if suite == "go" {
			commands = commands[:7]
		} else if suite == "browser" {
			commands = commands[7:]
		}
		for index, command := range commands {
			fail := strings.Join(command, " ")
			t.Run(suite+"/"+fail, func(t *testing.T) {
				WriteFile(t, fixture.log, []byte{}, 0o644)
				result := fixture.run(t, fail, suite)
				Check(t, result.Code == 7, "exit = %d", result.Code)
				want := commands
				// A failed browser build prevents only its dependent prepared suites.
				if fail == "bun run build:forgejo" {
					want = commands[:index+1]
				}
				Check(t, equalCommands(commandNames(fixture.commands(t)), want),
					"commands = %v", commandNames(fixture.commands(t)))
				Check(t, strings.Contains(result.Stdout, "Local source checks executed"), "stdout=%q", result.Stdout)
			})
		}
	}
}

func TestSourceRejectsUnknownSuiteBeforeRunningTools(t *testing.T) {
	for _, args := range [][]string{{"browsre"}, {"go", "browser"}} {
		fixture := newSourceFixture(t)
		result := fixture.run(t, "", args...)
		Check(t, result.Code == 2, "exit = %d", result.Code)
		Check(t, len(fixture.commands(t)) == 0, "unexpected command execution")
	}
}

func TestSourceNativeGateStillSurroundsSharedSourceChecks(t *testing.T) {
	scripts := packageScripts(t)
	Check(t, scripts["check:source"] == "bash scripts/check-source.sh", "check:source=%q", scripts["check:source"])
	parts := strings.Split(ReadFile(t, "scripts/check-native.sh"), "bun run check:source\n")
	Require(t, len(parts) == 2, "check-native.sh has %d markers", len(parts))
	before, after := parts[0], parts[1]
	for _, want := range []string{
		`$(uname -s) == Linux && $(uname -m) == "$arch"`,
		"Check requires a clean exact-revision checkout",
		"soda-build candidate artifacts directory required (payload.json, candidate.json, host.oci)",
		"cargo run --locked -p soda-release-tools",
		`--candidate "$artifacts"`,
		"--bin soda-candidate-check",
		`--soda-revision "$revision"`,
		`--forgejo-revision "$forgejo_revision"`,
	} {
		Check(t, strings.Contains(before, want), "missing before marker %q", want)
	}
	for _, want := range []string{
		`$(git rev-parse HEAD) == "$revision"`,
		"git status --porcelain --untracked-files=normal",
	} {
		Check(t, strings.Contains(after, want), "missing after marker %q", want)
	}
}

func TestOxfmtGatePreservesFormatterFailureAndPaths(t *testing.T) {
	for _, tc := range []struct {
		name, output, diagnostic string
		path                     string
		status                   string
		wantPass                 bool
		defaultFiles             bool
	}{
		{name: "formatted", status: "0", wantPass: true},
		{name: "formatting needed", output: "frontend/file.ts\n", status: "1"},
		{name: "formatter failure", diagnostic: "formatter unavailable\n", status: "2"},
		{name: "explicit paths", path: "frontend/space name.ts", status: "0", wantPass: true},
		{name: "default paths", path: "frontend/space name.ts", status: "0", wantPass: true, defaultFiles: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := tc.path
			if path == "" {
				path = "frontend/file.ts"
			}
			dir := t.TempDir()
			for _, path := range []string{"scripts", "node_modules/.bin"} {
				if err := os.MkdirAll(filepath.Join(dir, path), 0o755); err != nil {
					t.Fatal(err)
				}
			}
			gate := filepath.Join(dir, "scripts/check-oxfmt.sh")
			if err := os.WriteFile(gate, []byte(ReadFile(t, "scripts/check-oxfmt.sh")), 0o700); err != nil {
				t.Fatal(err)
			}
			tool := "#!/bin/sh\n" +
				"[ \"$#\" -eq 2 ] && [ \"$1\" = --list-different ] && [ \"$2\" = " + shellQuote(path) + " ] || { echo 'path split' >&2; exit 2; }\n" +
				"printf '%s' " + shellQuote(tc.output) + "\nprintf '%s' " + shellQuote(tc.diagnostic) + " >&2\nexit " + tc.status + "\n"
			if err := os.WriteFile(filepath.Join(dir, "node_modules/.bin/oxfmt"), []byte(tool), 0o700); err != nil {
				t.Fatal(err)
			}
			args := []string{gate, path}
			env := os.Environ()
			if tc.defaultFiles {
				fakeGit := "#!/bin/sh\n[ \"$1\" = ls-files ] && [ \"$2\" = -z ] || exit 2\nprintf 'frontend/space name.ts\\0'\n"
				bin := filepath.Join(dir, "node_modules/.bin")
				if err := os.WriteFile(filepath.Join(bin, "git"), []byte(fakeGit), 0o700); err != nil {
					t.Fatal(err)
				}
				env = SetEnv(env, "PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
				args = []string{gate}
			}
			result := Run(t, RunOpt{Cwd: dir, Env: env, Timeout: 10 * time.Second}, "/bin/bash", args...)
			if (result.Code == 0) != tc.wantPass || strings.Contains(result.Stderr, "path split") {
				t.Fatalf("gate status=%d stdout=%q stderr=%q", result.Code, result.Stdout, result.Stderr)
			}
			if tc.diagnostic != "" && !strings.Contains(result.Stderr, strings.TrimSpace(tc.diagnostic)) {
				t.Fatalf("formatter diagnostic lost: %q", result.Stderr)
			}
			if tc.output != "" && !strings.Contains(result.Stdout, strings.TrimSpace(tc.output)) {
				t.Fatalf("unformatted file output lost: %q", result.Stdout)
			}
		})
	}
}
