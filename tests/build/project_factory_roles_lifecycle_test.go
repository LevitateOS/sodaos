package build

import (
	"fmt"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func TestRolesStartSpawnsSupervisorAndReportsRunning(t *testing.T) {
	fenv := rolesSetup(t)
	rolesOK(t, fenv, rolesOp(t, "ensure", ""))
	rolesOK(t, fenv, rolesApprove(t, rolesPID, "soda-coder",
		map[string][]byte{"setup.sh": []byte("sleep 120"), "check.sh": []byte("true\n")},
		[]byte("bundle"), ""))
	rolesOK(t, fenv, rolesRecord(t, rolesPID, "", ""))
	started := rolesOK(t, fenv, rolesOp(t, "start", rolesPID))
	pgid, ok := started["pgid"].(float64)
	Require(t, ok && pgid > 0, "pgid = %v", started["pgid"])
	t.Cleanup(func() { _ = syscall.Kill(-int(pgid), syscall.SIGKILL) })
	state := rolesOK(t, fenv, rolesOp(t, "inspect", rolesPID))
	Check(t, state["phase"] == "running", "phase = %v", state["phase"])
	repeated := rolesOK(t, fenv, rolesOp(t, "start", rolesPID))
	Check(t, repeated["repeated"] == true, "repeat not reported")
	_, hasPGID := repeated["pgid"]
	Check(t, !hasPGID, "repeat carries pgid = %v", repeated["pgid"])
	stopped := rolesOK(t, fenv, rolesOp(t, "stop", rolesPID))
	Check(t, stopped["known"] == true && stopped["retirement"] == "confirmed", "stop = %v", stopped)
	state = rolesOK(t, fenv, rolesOp(t, "inspect", rolesPID))
	Check(t, state["phase"] == "stopped" && state["stopped"] == true, "state = %v", state["phase"])
}

func TestRolesDeadSupervisorReportsInterrupted(t *testing.T) {
	fenv := rolesSetup(t)
	rolesOK(t, fenv, rolesOp(t, "ensure", ""))
	rolesOK(t, fenv, rolesFixture(t, rolesPID))
	rolesOK(t, fenv, rolesRecord(t, rolesPID, "", ""))
	pgid := rolesDeadPGID(t)
	WriteFile(t, filepath.Join(fenv.factory, "preparations", rolesPID, "started.json"),
		[]byte(fmt.Sprintf(`{"pid": %d, "pgid": %d}`, pgid, pgid)), 0o644)
	state := rolesOK(t, fenv, rolesOp(t, "inspect", rolesPID))
	Check(t, state["phase"] == "interrupted", "phase = %v", state["phase"])
	rolesRefused(t, fenv, rolesOp(t, "start", rolesPID))
}

func TestRolesStopBarsUnknownIdentityAndRetiresKnown(t *testing.T) {
	fenv := rolesSetup(t)
	stopped := rolesOK(t, fenv, rolesOp(t, "stop", rolesPID))
	Check(t, stopped["known"] == false && stopped["retirement"] == "confirmed", "unknown = %v", stopped)
	rolesRefused(t, fenv, rolesFixture(t, rolesPID))
	rolesOK(t, fenv, rolesFixture(t, rolesPID2))
	rolesOK(t, fenv, rolesRecord(t, rolesPID2, "", ""))
	stopped = rolesOK(t, fenv, rolesOp(t, "stop", rolesPID2))
	Check(t, stopped["known"] == true && stopped["retirement"] == "confirmed", "known = %v", stopped)
	state := rolesOK(t, fenv, rolesOp(t, "inspect", rolesPID2))
	Check(t, state["phase"] == "stopped" && state["stopped"] == true, "state = %v", state["phase"])
}

func TestRolesHoldDeniesApproveAndReleaseNeedsRevisionAndQuiescence(t *testing.T) {
	fenv := rolesSetup(t)
	rolesOK(t, fenv, rolesOp(t, "ensure", ""))
	rolesOK(t, fenv, rolesFixture(t, rolesPID))
	held := rolesOK(t, fenv, rolesMarshal(t, map[string]any{"op": "hold", "revision": 1}))
	Check(t, held["hold"].(map[string]any)["active"] == true, "hold not active")
	rolesRefused(t, fenv, rolesFixture(t, rolesPID2))
	rolesRefused(t, fenv, rolesMarshal(t, map[string]any{"op": "release", "revision": 2}))
	WriteFile(t, filepath.Join(fenv.factory, "preparations", rolesPID, "started.json"),
		[]byte(`{"pid": 999, "pgid": 999}`), 0o644)
	rolesRefused(t, fenv, rolesMarshal(t, map[string]any{"op": "release", "revision": 1}))
	rolesOK(t, fenv, rolesOp(t, "stop", rolesPID))
	released := rolesOK(t, fenv, rolesMarshal(t, map[string]any{"op": "release", "revision": 1}))
	Check(t, released["hold"].(map[string]any)["active"] == false, "hold still active")
}

func TestRolesSupervisorLivenessTracksProcessGroup(t *testing.T) {
	fenv := rolesSetup(t)
	rolesOK(t, fenv, rolesOp(t, "ensure", ""))
	rolesOK(t, fenv, rolesApprove(t, rolesPID, "soda-coder",
		map[string][]byte{"setup.sh": []byte("sleep 120"), "check.sh": []byte("true\n")},
		[]byte("bundle"), ""))
	rolesOK(t, fenv, rolesRecord(t, rolesPID, "", ""))
	started := rolesOK(t, fenv, rolesOp(t, "start", rolesPID))
	pgid := int(started["pgid"].(float64))
	t.Cleanup(func() { _ = syscall.Kill(-pgid, syscall.SIGKILL) })
	Require(t, rolesGroupAlive(pgid), "supervisor group %d not alive", pgid)
	// A crashed supervisor (group gone, no completion) reads interrupted,
	// and the identity refuses any restart.
	Require(t, syscall.Kill(-pgid, syscall.SIGKILL) == nil, "kill group %d", pgid)
	deadline := time.Now().Add(10 * time.Second)
	for rolesGroupAlive(pgid) && time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
	}
	Require(t, !rolesGroupAlive(pgid), "group %d survives SIGKILL", pgid)
	state := rolesOK(t, fenv, rolesOp(t, "inspect", rolesPID))
	Check(t, state["phase"] == "interrupted", "phase = %v", state["phase"])
	rolesRefused(t, fenv, rolesOp(t, "start", rolesPID))
}
