package build

import (
	"testing"
)

func TestRolesRecordReportsWaitingAndFailedPhases(t *testing.T) {
	fenv := rolesSetup(t)
	rolesOK(t, fenv, rolesOp(t, "ensure", ""))
	rolesOK(t, fenv, rolesFixture(t, rolesPID))
	recorded := rolesOK(t, fenv, rolesRecord(t, rolesPID, "node22", ""))
	Check(t, recorded["waiting"] == true, "not waiting")
	state := rolesOK(t, fenv, rolesOp(t, "inspect", rolesPID))
	Check(t, state["phase"] == "waiting" && state["missing"] == "node22" && state["ready"] == false,
		"state = %v/%v/%v", state["phase"], state["missing"], state["ready"])
	rolesRefused(t, fenv, rolesRecord(t, rolesPID, "other-tool", ""))
}

func TestRolesLauncherRefusalFailsWithoutStart(t *testing.T) {
	fenv := rolesSetup(t)
	rolesOK(t, fenv, rolesOp(t, "ensure", ""))
	rolesOK(t, fenv, rolesFixture(t, rolesPID))
	rolesOK(t, fenv, rolesRecord(t, rolesPID, "", "role holds unexpected groups"))
	state := rolesOK(t, fenv, rolesOp(t, "inspect", rolesPID))
	Check(t, state["phase"] == "failed", "phase = %v", state["phase"])
	rolesRefused(t, fenv, rolesOp(t, "start", rolesPID))
}
