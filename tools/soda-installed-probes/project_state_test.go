package main

// Project-state snapshot coverage for the installed-probes suite: CLI
// refusal plus the behavior contracts owned by the Rust probe. Command
// and entry semantics live in the Rust test suite (command passthrough,
// failure shape, output bound, hashes, sizes, link/missing fail-closed);
// the retired tests/installed/project-state.py asserted the same.

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestProjectStateRefusesOutsideRootContainer(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root enters the snapshot path")
	}
	probe := remoteProbeBinary(t)
	result := runProbe(t, nil, 30*time.Second, probe, "project-state")
	checkProbe(t, result.code == 1, "exit = %d", result.code)
	checkProbe(t, result.stdout == "", "stdout=%q", result.stdout)
	checkProbe(t, result.stderr == "Project snapshot failed: AssertionError \n", "stderr=%q", result.stderr)
}

func TestProjectStateSnapshotContracts(t *testing.T) {
	source := readProbeSource(t, "tools/acceptance/src/project_state.rs")
	for _, want := range []string{
		"SODA_EXPECT_WORKLOADS",
		"Caller must declare required workload observations",
		"Snapshot output exceeded bound",
		"Snapshot file too large",
		"4 * 1024 * 1024",
		"512 * 1024 * 1024",
		"Project snapshot failed: ",
		`name == "config" || name == "known_hosts"`,
	} {
		checkProbe(t, strings.Contains(source, want), "missing contract %q", want)
	}
}
