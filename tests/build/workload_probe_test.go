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

// fakeCommand is the native-command double: POSIX shell that appends its
// argv as a JSON array to CALL_LOG, then replays the probe's exit policy
// against previous calls (curl needs a prior curl unless FAIL_HTTP forces
// 7; pg_isready passes only after a prior pg_isready; sleep, the podman
// compose/secret reads and psql pass; anything else exits 99).
const fakeCommand = `name=${0##*/}
prev=
if [ -f "$CALL_LOG" ]; then prev=$(<"$CALL_LOG"); fi
json="\"$name\""
for a in "$@"; do
	e=${a//\\/\\\\}
	e=${e//\"/\\\"}
	e=${e//$'\n'/\\n}
	json="$json,\"$e\""
done
printf '[%s]\n' "$json" >> "$CALL_LOG"
prior_curl=0; prior_pg=0
while IFS= read -r line || [ -n "$line" ]; do
	case "$line" in
		'["curl"'*) prior_curl=1;;
	esac
	case "$line" in
		*'"pg_isready"'*) prior_pg=1;;
	esac
done <<< "$prev"
in_call() {
	w=$1; shift
	[ "$name" = "$w" ] && return 0
	for a in "$@"; do [ "$a" = "$w" ] && return 0; done
	return 1
}
if [ "$name" = "curl" ]; then
	if [ "$FAIL_HTTP" = "1" ] || [ "$prior_curl" = 0 ]; then exit 7; fi
	exit 0
fi
if in_call pg_isready "$@"; then
	if [ "$prior_pg" = 1 ]; then exit 0; fi
	exit 1
fi
if [ "$name" = "sleep" ]; then exit 0; fi
if [ "$name" = "podman" ] && [ "${1:-}" = "compose" ] && { [ "${2:-}" = "ps" ] || [ "${2:-}" = "up" ]; }; then exit 0; fi
if [ "$name" = "podman" ] && [ "${1:-}" = "secret" ] && [ "${2:-}" = "inspect" ]; then exit 0; fi
if in_call psql "$@"; then exit 0; fi
exit 99
`

func invokeWorkloads(t *testing.T, mode string, failHTTP bool) (ProcResult, [][]string) {
	t.Helper()
	dir := TempDir(t)
	fake := filepath.Join(dir, "command")
	WriteFile(t, fake, []byte("#!/bin/bash\n"+fakeCommand), 0o755)
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
