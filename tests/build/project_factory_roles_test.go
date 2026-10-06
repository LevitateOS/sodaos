// Behavioral coverage for the project-factory-roles helper. Every case
// drives the compiled Rust binary with the factory redirected to a
// test-owned directory plus a recording stand-in for git; the embedded
// interpreter driver is gone and no interpreter runs anywhere here.
package build

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
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

func TestRolesApproveWritesProtectedSnapshotAndVerifiesBundle(t *testing.T) {
	fenv := rolesSetup(t)
	rolesOK(t, fenv, rolesOp(t, "ensure", ""))
	decoded := rolesOK(t, fenv, rolesFixture(t, rolesPID))
	Check(t, decoded["approved"] == rolesPID, "approved = %v", decoded["approved"])
	Check(t, decoded["repeated"] == false, "repeated = %v", decoded["repeated"])
	snapshot := filepath.Join(fenv.factory, "preparations", rolesPID, "snapshot")
	setup, err := os.ReadFile(filepath.Join(snapshot, "setup.sh"))
	Require(t, err == nil, "read setup: %v", err)
	Check(t, string(setup) == "true\n", "setup = %q", setup)
	Check(t, rolesMode(t, filepath.Join(snapshot, "setup.sh")) == 0o644, "setup mode")
	Check(t, rolesMode(t, filepath.Join(snapshot, "source.bundle")) == 0o644, "bundle mode")
	// Exact verification argv, in order.
	record, err := os.ReadFile(fenv.record)
	Require(t, err == nil, "read git record: %v", err)
	bundle := filepath.Join(snapshot, "source.bundle")
	repo := filepath.Join(snapshot, "verify-tmp", "repo")
	Check(t, strings.Contains(string(record),
		"---\n<clone>\n<-q>\n<--no-checkout>\n<"+bundle+">\n<"+repo+">\n"), "clone argv:\n%s", record)
	Check(t, strings.Contains(string(record),
		"---\n<-C>\n<"+repo+">\n<cat-file>\n<-e>\n<"+rolesCommit+">\n"), "cat-file argv:\n%s", record)
	_, err = os.Stat(filepath.Join(snapshot, "verify-tmp"))
	Check(t, os.IsNotExist(err), "verify-tmp remains")
	repeated := rolesOK(t, fenv, rolesFixture(t, rolesPID))
	Check(t, repeated["repeated"] == true, "repeat not reported")
	conflict := map[string]any{}
	Require(t, json.Unmarshal(rolesFixture(t, rolesPID), &conflict) == nil, "decode fixture")
	conflict["setup_digest"] = strings.Repeat("e", 64)
	rolesRefused(t, fenv, rolesMarshal(t, conflict))
}

func TestRolesApproveRejectsUntrustedInputsBeforeEffects(t *testing.T) {
	fenv := rolesSetup(t)
	rolesOK(t, fenv, rolesOp(t, "ensure", ""))
	var base map[string]any
	Require(t, json.Unmarshal(rolesFixture(t, rolesPID), &base) == nil, "decode fixture")
	clone := func() map[string]any {
		out := map[string]any{}
		for key, value := range base {
			out[key] = value
		}
		return out
	}
	single := clone()
	single["files"] = map[string]any{"setup.sh": base64.StdEncoding.EncodeToString([]byte("x"))}
	cases := []struct {
		name  string
		value map[string]any
	}{
		{"role", func() map[string]any { v := clone(); v["role"] = "root"; return v }()},
		{"id", func() map[string]any { v := clone(); v["id"] = "../escape"; return v }()},
		{"digest", func() map[string]any { v := clone(); v["setup_digest"] = "zz"; return v }()},
		{"commit", func() map[string]any { v := clone(); v["source_commit"] = "short"; return v }()},
		{"credential", func() map[string]any { v := clone(); v["credential"] = "../x"; return v }()},
		{"files", single},
		{"bundle", func() map[string]any { v := clone(); v["bundle"] = "!!!"; return v }()},
		{"extra", func() map[string]any { v := clone(); v["extra"] = 1; return v }()},
	}
	Require(t, len(cases) == 8, "cases = %d", len(cases))
	for _, tc := range cases {
		rolesRefused(t, fenv, rolesMarshal(t, tc.value))
	}
	_, err := os.Stat(fenv.record)
	Check(t, os.IsNotExist(err), "git ran before refusal")
	_, err = os.Stat(filepath.Join(fenv.factory, "preparations", rolesPID))
	Check(t, os.IsNotExist(err), "effects before refusal")
}

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
