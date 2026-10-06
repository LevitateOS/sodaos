// Behavioral coverage for the project-factory-roles helper. Every case
// drives the compiled Rust binary with the factory redirected to a
// test-owned directory plus a recording stand-in for git; the embedded
// interpreter driver is gone and no interpreter runs anywhere here.
package build

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRolesRunAsRoleCapturesBoundedOutputAndExit(t *testing.T) {
	fenv := rolesSetup(t)
	rolesOK(t, fenv, rolesOp(t, "ensure", ""))
	rolesOK(t, fenv, rolesApprove(t, rolesPID, "soda-coder",
		map[string][]byte{"setup.sh": []byte("echo out; exit 3"), "check.sh": []byte("echo ran")},
		[]byte("bundle"), ""))
	rolesOK(t, fenv, rolesRecord(t, rolesPID, "", ""))
	rolesOK(t, fenv, rolesOp(t, "start", rolesPID))
	rolesWaitFile(t, filepath.Join(fenv.factory, "preparations", rolesPID, "finished.json"), 30*time.Second)
	state := rolesOK(t, fenv, rolesOp(t, "inspect", rolesPID))
	Check(t, state["phase"] == "failed", "phase = %v", state["phase"])
	Check(t, state["setup_exit"] == float64(3), "setup_exit = %v", state["setup_exit"])
	Check(t, state["check_exit"] == nil, "check ran after setup failure: %v", state["check_exit"])
	Check(t, state["setup_log"] == "out\n", "log = %q", state["setup_log"])
	// Output past the cap truncates with the marker.
	rolesOK(t, fenv, rolesApprove(t, rolesPID2, "soda-coder",
		map[string][]byte{"setup.sh": []byte("yes | head -c 70000"), "check.sh": []byte("true\n")},
		[]byte("bundle"), ""))
	rolesOK(t, fenv, rolesRecord(t, rolesPID2, "", ""))
	rolesOK(t, fenv, rolesOp(t, "start", rolesPID2))
	rolesWaitFile(t, filepath.Join(fenv.factory, "preparations", rolesPID2, "finished.json"), 30*time.Second)
	state = rolesOK(t, fenv, rolesOp(t, "inspect", rolesPID2))
	log, ok := state["setup_log"].(string)
	Require(t, ok, "setup_log = %T", state["setup_log"])
	Check(t, strings.HasSuffix(log, "\n[output truncated]\n"), "log not truncated: %q", log[len(log)-30:])
	Check(t, len(log) == 65536+len("\n[output truncated]\n"), "log length = %d", len(log))
}

func TestRolesRejectsMalformedRequests(t *testing.T) {
	fenv := rolesSetup(t)
	for _, body := range []string{
		"not json",
		"",
		"[1, 2]",
		"null",
		`{"op": "frobnicate"}`,
		`{"id": "x"}`,
		`{"op": "ensure", "extra": 1}`,
		`{"op": "hold", "revision": true}`,
		`{"op": "approve"}`,
		`{"op": "stop", "id": "../escape"}`,
	} {
		rolesRefused(t, fenv, []byte(body))
	}
	rolesRefused(t, fenv, bytes.Repeat([]byte("x"), 4*1024*1024+65536+1))
}
