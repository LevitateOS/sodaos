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
// as one JSON object per line, fakes `go version` from go.mod, and fails
// the FAIL_COMMAND with 7. Argument JSON-escaping covers backslash,
// double-quote and newline; nothing else in the fixture needs escapes.
const fakeTool = `name=$(basename "$0")
json="\"$name\""
for a in "$@"; do
	e=$(printf '%s' "$a" | sed -e 's/\\/\\\\/g' -e 's/"/\\"/g' -e ':a;N;$!ba;s/\n/\\n/g')
	json="$json,\"$e\""
done
printf '{"command": [%s], "cwd": "%s", "env": {"GOWORK": "%s", "GOFLAGS": "%s", "CGO_ENABLED": "%s", "GOTOOLCHAIN": "%s"}}\n' "$json" "$(pwd -P)" "$GOWORK" "$GOFLAGS" "$CGO_ENABLED" "$GOTOOLCHAIN" >> "$COMMAND_LOG"
if [ "$name $*" = "go version" ]; then
	pin=$(sed -n 's/^go //p' go.mod | head -1)
	echo "go version go$pin fixture/fixture"
fi
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
	{"go", "version"},
	{"go", "mod", "verify"},
	{"go", "test", "-mod=readonly", "./..."},
	{"bash", "scripts/check-sql-locality.sh"},
	{"bash", "scripts/check-no-npm.sh"},
	{"bash", "scripts/check-no-python.sh"},
	{"bun", "run", "typecheck"},
	{"bun", "run", "test"},
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
	for _, gate := range []string{"check-sql-locality.sh", "check-no-npm.sh", "check-no-python.sh"} {
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

func (f sourceFixture) run(t *testing.T, fail string) ProcResult {
	t.Helper()
	env := SetEnv(os.Environ(), "PATH", f.tools+string(os.PathListSeparator)+os.Getenv("PATH"))
	env = SetEnv(env, "COMMAND_LOG", f.log)
	env = SetEnv(env, "FAIL_COMMAND", fail)
	env = SetEnv(env, "GOTOOLCHAIN", "auto")
	return Run(t, RunOpt{Cwd: f.tools, Env: env, Timeout: 30 * time.Second},
		"/bin/bash", filepath.Join(f.root, "scripts/check-source.sh"))
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
	fixture := newSourceFixture(t)
	result := fixture.run(t, "")
	Require(t, result.Code == 0, "check-source failed: %s", result.Stderr)
	commands := fixture.commands(t)
	Check(t, equalCommands(commandNames(commands), expectedSourceCommands), "commands = %v", commandNames(commands))
	for _, entry := range commands {
		Check(t, entry.Cwd == fixture.root, "cwd = %q", entry.Cwd)
		want := map[string]any{"GOWORK": "off", "GOFLAGS": "-mod=readonly", "CGO_ENABLED": "0", "GOTOOLCHAIN": "local"}
		Check(t, len(entry.Env) == len(want), "env = %v", entry.Env)
		for key, value := range want {
			Check(t, entry.Env[key] == value, "env[%s] = %v", key, entry.Env[key])
		}
	}
	Check(t, strings.Contains(result.Stdout, "Local source checks executed"), "stdout=%q", result.Stdout)
}

func TestSourceEachFailureStopsRemainingSuites(t *testing.T) {
	fixture := newSourceFixture(t)
	for index, command := range expectedSourceCommands {
		t.Run(strings.Join(command, " "), func(t *testing.T) {
			WriteFile(t, fixture.log, []byte{}, 0o644)
			result := fixture.run(t, strings.Join(command, " "))
			Check(t, result.Code == 7, "exit = %d", result.Code)
			Check(t, equalCommands(commandNames(fixture.commands(t)), expectedSourceCommands[:index+1]),
				"commands = %v", commandNames(fixture.commands(t)))
			Check(t, !strings.Contains(result.Stdout, "Local source checks executed"), "stdout=%q", result.Stdout)
		})
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
		"Pinned native source-check tools required",
		"Check requires a clean exact-revision checkout",
		"soda-build candidate artifacts directory required (payload.json, candidate.json, host.oci)",
		"verifier=$artifacts/tools/soda-artifacts",
		`"$verifier" verify --source "$artifacts" --arch "$arch" --revision "$revision"`,
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
