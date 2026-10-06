package build

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestRolesEnsureProvisionsLockedRolesWithoutExtraGroups(t *testing.T) {
	fenv := rolesSetup(t)
	decoded := rolesOK(t, fenv, rolesOp(t, "ensure", ""))
	roles, ok := decoded["roles"].([]any)
	Require(t, ok && len(roles) == 2, "roles = %v", decoded["roles"])
	Check(t, roles[0] == "soda-coder" && roles[1] == "soda-reviewer", "roles = %v", roles)
	for _, login := range []string{"soda-coder", "soda-reviewer"} {
		home := filepath.Join(fenv.factory, "test-homes", login)
		Check(t, rolesMode(t, home) == 0o700, "%s home mode = %o", login, rolesMode(t, home))
		Check(t, rolesMode(t, filepath.Join(home, "checkouts")) == 0o755, "%s checkouts", login)
		Check(t, rolesMode(t, filepath.Join(fenv.factory, "credentials", login)) == 0o755, "%s creds", login)
	}
	// Idempotent rerun with no git traffic.
	again := rolesOK(t, fenv, rolesOp(t, "ensure", ""))
	Check(t, fmt.Sprintf("%v", again["roles"]) == fmt.Sprintf("%v", roles), "rerun = %v", again)
	_, err := os.Stat(fenv.record)
	Check(t, os.IsNotExist(err), "git ran during ensure")
}

func TestRolesEnsureRefusesInteractiveOrGroupedAccounts(t *testing.T) {
	fenv := rolesSetup(t)
	overlay := filepath.Join(fenv.factory, "test-accounts")
	Require(t, os.MkdirAll(overlay, 0o755) == nil, "mkdir overlay")
	WriteFile(t, filepath.Join(overlay, "soda-coder.json"), []byte(`{"shell": "/bin/bash"}`), 0o644)
	rolesRefused(t, fenv, rolesOp(t, "ensure", ""))
}
