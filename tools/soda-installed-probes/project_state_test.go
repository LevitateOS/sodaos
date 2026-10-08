package main

// Project-state snapshot coverage for the installed-probes suite: CLI
// refusal plus the behavior contracts owned by the Rust probe. Command
// and entry semantics live in the Rust test suite (command passthrough,
// failure shape, output bound, hashes, sizes, link/missing fail-closed);
// the retired tests/installed/project-state.py asserted the same.

import (
	"os"
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
