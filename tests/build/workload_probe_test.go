// Port of test_workload_probe.py: readiness polling replays no mutations.
package build

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// fakeCommand is the native-command double from test_workload_probe.py,
// transcribed verbatim (the Python source escapes \n as \\n).
const fakeCommand = `import json,os,pathlib,sys
p=pathlib.Path(os.environ['CALL_LOG']);calls=[json.loads(x) for x in p.read_text().splitlines()] if p.exists() else []
call=[pathlib.Path(sys.argv[0]).name,*sys.argv[1:]]
with p.open('a') as f:f.write(json.dumps(call)+'\n')
if call[0]=='curl':
 sys.exit(7 if os.environ['FAIL_HTTP']=='1' or not any(x[0]=='curl' for x in calls) else 0)
if 'pg_isready' in call:sys.exit(0 if any('pg_isready' in x for x in calls) else 1)
if call[0]=='sleep' or call[:3] in [['podman','compose','ps'],['podman','compose','up'],['podman','secret','inspect']] or 'psql' in call:sys.exit(0)
sys.exit(99)
`

func invokeWorkloads(t *testing.T, mode string, failHTTP bool) (ProcResult, [][]string) {
	t.Helper()
	dir := TempDir(t)
	fake := filepath.Join(dir, "command")
	WriteFile(t, fake, []byte("#!"+Python3(t)+"\n"+fakeCommand), 0o755)
	for _, command := range []string{"podman", "curl", "sleep"} {
		if err := os.Symlink(fake, filepath.Join(dir, command)); err != nil {
			t.Fatalf("symlink %s: %v", command, err)
		}
	}
	log := filepath.Join(dir, "calls.jsonl")
	fail := "0"
	if failHTTP {
		fail = "1"
	}
	env := SetEnv(os.Environ(), "PATH", dir)
	env = SetEnv(env, "CALL_LOG", log)
	env = SetEnv(env, "SODA_NATIVE_VALIDATE", "soda-test")
	env = SetEnv(env, "FAIL_HTTP", fail)
	result := Run(t, RunOpt{Env: env, Timeout: 15 * time.Second},
		"/bin/sh", filepath.Join(RepoRoot, "tests/installed/workloads.sh"), mode)
	var calls [][]string
	if data, err := os.ReadFile(log); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			if strings.TrimSpace(line) == "" {
				continue
			}
			var call []string
			if err := json.Unmarshal([]byte(line), &call); err != nil {
				t.Fatalf("parse call log: %v", err)
			}
			calls = append(calls, call)
		}
	}
	return result, calls
}

func countCalls(calls [][]string, match func([]string) bool) int {
	count := 0
	for _, call := range calls {
		if match(call) {
			count++
		}
	}
	return count
}

func hasWord(call []string, words ...string) bool {
	for _, entry := range call {
		for _, word := range words {
			if entry == word {
				return true
			}
		}
	}
	return false
}

func TestWorkloadStartOnceThenReadiness(t *testing.T) {
	result, calls := invokeWorkloads(t, "start", false)
	Require(t, result.Code == 0, "start failed: %s", result.Stderr)
	up := func(call []string) bool {
		return len(call) >= 3 && call[0] == "podman" && call[1] == "compose" && call[2] == "up"
	}
	Check(t, countCalls(calls, up) == 1, "compose up ran %d times", countCalls(calls, up))
	curl := func(call []string) bool { return len(call) > 0 && call[0] == "curl" }
	Check(t, countCalls(calls, curl) == 2, "curl ran %d times", countCalls(calls, curl))
	Check(t, countCalls(calls, func(call []string) bool { return hasWord(call, "pg_isready") }) == 2, "pg_isready count wrong")
	Check(t, countCalls(calls, func(call []string) bool { return hasWord(call, "psql") }) == 1, "psql count wrong")
}

func TestWorkloadCheckNeverStartsOrBuilds(t *testing.T) {
	result, calls := invokeWorkloads(t, "check", false)
	Require(t, result.Code == 0, "check failed: %s", result.Stderr)
	for _, call := range calls {
		Check(t, !hasWord(call, "up", "build", "secret"), "check mutated: %v", call)
	}
}

func TestWorkloadFailedReadinessIsBoundedAndNotSuccess(t *testing.T) {
	result, calls := invokeWorkloads(t, "check", true)
	Check(t, result.Code == 1, "exit = %d", result.Code)
	curl := func(call []string) bool { return len(call) > 0 && call[0] == "curl" }
	Check(t, countCalls(calls, curl) == 30, "curl ran %d times", countCalls(calls, curl))
	for _, call := range calls {
		Check(t, !hasWord(call, "up", "psql"), "failed check mutated: %v", call)
	}
}

func TestWorkloadInvalidModeNeverInvokesNativeCommands(t *testing.T) {
	for _, mode := range []string{"replace", ""} {
		t.Run("mode="+mode, func(t *testing.T) {
			result, calls := invokeWorkloads(t, mode, false)
			Check(t, result.Code == 2, "exit = %d", result.Code)
			Check(t, len(calls) == 0, "calls = %v", calls)
		})
	}
}
