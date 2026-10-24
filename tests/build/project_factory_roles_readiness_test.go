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
	// The same unstarted preparation may refresh prerequisite observations.
	refreshed := rolesOK(t, fenv, rolesRecord(t, rolesPID, "other-tool", ""))
	Check(t, refreshed["waiting"] == true, "refreshed prerequisite not waiting")
	state = rolesOK(t, fenv, rolesOp(t, "inspect", rolesPID))
	Check(t, state["phase"] == "waiting" && state["missing"] == "other-tool" && state["ready"] == false,
		"refreshed state = %v/%v/%v", state["phase"], state["missing"], state["ready"])
	rolesRefused(t, fenv, rolesOp(t, "start", rolesPID))
	resolved := rolesOK(t, fenv, rolesRecord(t, rolesPID, "", ""))
	Check(t, resolved["waiting"] == false, "resolved prerequisites still waiting")
	rolesRefused(t, fenv, rolesRecord(t, rolesPID, "other-tool", ""))
	state = rolesOK(t, fenv, rolesOp(t, "inspect", rolesPID))
	Check(t, state["phase"] == "approved" && state["missing"] == "" && state["ready"] == false,
		"resolved evidence changed after refusal: %v/%v/%v", state["phase"], state["missing"], state["ready"])
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
