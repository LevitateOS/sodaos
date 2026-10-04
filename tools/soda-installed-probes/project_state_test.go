package main

// Port of tests/installed/project-state.py into the installed-probes suite:
// CLI refusal, command() failure shape and output bound. The entry()
// contracts live in tests/build/u08_state_test.go (port of
// test_u08_state.py) and are not duplicated here.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestProjectStateRefusesOutsideRootContainer(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root enters the snapshot path")
	}
	script := filepath.Join(probeRoot, "tests/installed/project-state.py")
	result := runProbe(t, nil, 30*time.Second, "python3", script)
	checkProbe(t, result.code == 1, "exit = %d", result.code)
	checkProbe(t, result.stdout == "", "stdout=%q", result.stdout)
	checkProbe(t, result.stderr == "Project snapshot failed: AssertionError \n", "stderr=%q", result.stderr)
}

// stateCommandDriver calls command(*argv) in project-state.py and reports
// the result or the failure shape as JSON.
const stateCommandDriver = `
import importlib.util, json, sys
spec = importlib.util.spec_from_file_location('subject', sys.argv[1])
subject = importlib.util.module_from_spec(spec)
spec.loader.exec_module(subject)
try:
    print(json.dumps({'ok': subject.command(*sys.argv[2:])}))
except Exception as failure:
    print(json.dumps({'error': type(failure).__name__,
                      'mro': [c.__name__ for c in type(failure).__mro__],
                      'message': str(failure)}))
`

type stateCommandOutcome struct {
	Ok      *string
	Error   string
	Mro     []string
	Message string
}

func runStateCommand(t *testing.T, argv ...string) stateCommandOutcome {
	t.Helper()
	script := filepath.Join(probeRoot, "tests/installed/project-state.py")
	full := append([]string{"-c", stateCommandDriver, script}, argv...)
	result := runProbe(t, nil, 60*time.Second, "python3", full...)
	requireProbe(t, result.code == 0, "driver failed: %s", result.stderr)
	var decoded struct {
		Ok      *string  `json:"ok"`
		Error   string   `json:"error"`
		Mro     []string `json:"mro"`
		Message string   `json:"message"`
	}
	requireProbe(t, json.Unmarshal([]byte(result.stdout), &decoded) == nil, "parse driver output %q", result.stdout)
	return stateCommandOutcome(decoded)
}

func TestProjectStateCommandPassesOutputThrough(t *testing.T) {
	outcome := runStateCommand(t, "/bin/echo", "snapshot-ok")
	requireProbe(t, outcome.Error == "", "command raised %s %s", outcome.Error, outcome.Message)
	requireProbe(t, outcome.Ok != nil, "no output")
	checkProbe(t, *outcome.Ok == "snapshot-ok", "output=%q", *outcome.Ok)
}

func TestProjectStateCommandFailureNamesOnlyFirstArgs(t *testing.T) {
	outcome := runStateCommand(t, "/bin/false", "a", "b", "c", "d", "e")
	requireProbe(t, outcome.Error == "RuntimeError", "error = %s", outcome.Error)
	checkProbe(t, outcome.Message == "Required snapshot command failed: /bin/false a b c d",
		"message=%q", outcome.Message)
}

func TestProjectStateCommandOutputBound(t *testing.T) {
	python, err := filepath.EvalSymlinks("/usr/bin/python3")
	if err != nil {
		python = "python3"
	}
	outcome := runStateCommand(t, python, "-c", "print('x' * 5000000)")
	requireProbe(t, outcome.Error == "RuntimeError", "error = %s", outcome.Error)
	checkProbe(t, outcome.Message == "Snapshot output exceeded bound", "message=%q", outcome.Message)
}

func TestProjectStateSnapshotContracts(t *testing.T) {
	source := readProbeSource(t, "tests/installed/project-state.py")
	for _, want := range []string{
		"SODA_EXPECT_WORKLOADS",
		"Caller must declare required workload observations",
		"Snapshot output exceeded bound",
		"Snapshot file too large",
		"4 * 1024 * 1024",
		"512 * 1024 * 1024",
		"entry(f, contents=False)",
		"Project snapshot failed: ",
	} {
		checkProbe(t, strings.Contains(source, want), "missing contract %q", want)
	}
}
