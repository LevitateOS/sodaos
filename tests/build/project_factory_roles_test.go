// Behavioral coverage for the project-factory-roles helper. Every case
// drives the compiled Rust binary with the factory redirected to a
// test-owned directory plus a recording stand-in for git; the embedded
// interpreter driver is gone and no interpreter runs anywhere here.
package build

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"testing"
	"time"
)

const (
	rolesPID    = "f0123456789abcdef01234567"
	rolesPID2   = "f123456789abcdef012345678"
	rolesCommit = "cccccccccccccccccccccccccccccccccccccccc"
	// Fixed refusal contract: exit 1, empty stdout, this stderr line.
	rolesFixedStderr = "factory preparation unconfirmed; inspect native state and managed files\n"
)

// rolesEnv is one isolated helper world: a scratch factory plus a
// recording git stand-in.
type rolesEnv struct {
	factory string
	git     string
	record  string
}

func rolesBinary(t *testing.T) string {
	t.Helper()
	return CargoBinary(t, "soda-project-factory-roles", "project-factory-roles")
}

func rolesSetup(t *testing.T) rolesEnv {
	t.Helper()
	root := TempDir(t)
	record := filepath.Join(root, "git.record")
	git := filepath.Join(root, "git")
	script := fmt.Sprintf("#!/bin/sh\n{ echo '---'; printf '<%%s>\\n' \"$@\"; } >> '%s'\nexit 0\n", record)
	WriteFile(t, git, []byte(script), 0o755)
	return rolesEnv{factory: filepath.Join(root, "factory"), git: git, record: record}
}

// rolesRun pipes one request body to the helper and captures the result.
func rolesRun(t *testing.T, fenv rolesEnv, stdin []byte) ProcResult {
	t.Helper()
	cmd := exec.Command(rolesBinary(t))
	cmd.Env = SetEnv(SetEnv(os.Environ(), "SODA_FACTORY_DIR", fenv.factory), "SODA_FACTORY_GIT", fenv.git)
	cmd.Stdin = bytes.NewReader(stdin)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	code := 0
	if err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			code = exit.ExitCode()
		} else {
			t.Fatalf("run helper: %v", err)
		}
	}
	return ProcResult{Code: code, Stdout: stdout.String(), Stderr: stderr.String()}
}

// rolesOK runs a request that must succeed and returns its decoded JSON.
func rolesOK(t *testing.T, fenv rolesEnv, body []byte) map[string]any {
	t.Helper()
	result := rolesRun(t, fenv, body)
	Require(t, result.Code == 0, "helper failed: code=%d stderr=%q", result.Code, result.Stderr)
	Require(t, result.Stderr == "", "stderr = %q", result.Stderr)
	var decoded map[string]any
	Require(t, json.Unmarshal([]byte(result.Stdout), &decoded) == nil, "parse %q", result.Stdout)
	return decoded
}

// rolesRefused runs a request that must fail with the fixed contract.
func rolesRefused(t *testing.T, fenv rolesEnv, body []byte) {
	t.Helper()
	result := rolesRun(t, fenv, body)
	Check(t, result.Code == 1, "code = %d (stdout=%q stderr=%q)", result.Code, result.Stdout, result.Stderr)
	Check(t, result.Stdout == "", "stdout = %q", result.Stdout)
	Check(t, result.Stderr == rolesFixedStderr, "stderr = %q", result.Stderr)
}

// rolesDigest recomputes the canonical approved-inputs digest.
func rolesDigest(files map[string][]byte) string {
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	sum := sha256.New()
	for _, name := range names {
		sum.Write([]byte(name))
		sum.Write([]byte{0})
		sum.Write(files[name])
	}
	return fmt.Sprintf("%x", sum.Sum(nil))
}

func rolesMarshal(t *testing.T, value map[string]any) []byte {
	t.Helper()
	body, err := json.Marshal(value)
	Require(t, err == nil, "marshal: %v", err)
	return body
}

func rolesApprove(t *testing.T, pid, role string, files map[string][]byte, bundle []byte, credential string) []byte {
	t.Helper()
	encoded := map[string]string{}
	for name, contents := range files {
		encoded[name] = base64.StdEncoding.EncodeToString(contents)
	}
	return rolesMarshal(t, map[string]any{
		"op": "approve", "id": pid, "role": role,
		"setup_digest": rolesDigest(files), "source_commit": rolesCommit,
		"files": encoded, "bundle": base64.StdEncoding.EncodeToString(bundle), "credential": credential,
	})
}

func rolesFixture(t *testing.T, pid string) []byte {
	t.Helper()
	return rolesApprove(t, pid, "soda-coder",
		map[string][]byte{"setup.sh": []byte("true\n"), "check.sh": []byte("true\n")},
		[]byte("bundle"), "")
}

func rolesRecord(t *testing.T, pid, missing, refusal string) []byte {
	t.Helper()
	verified := map[string]any{"uid": "1", "login": "soda-coder", "groups": "soda-coder"}
	if refusal != "" {
		verified["refusal"] = refusal
	}
	return rolesMarshal(t, map[string]any{
		"op": "record", "id": pid, "tools": []any{}, "missing": missing, "verified": verified,
	})
}

func rolesOp(t *testing.T, op, pid string) []byte {
	t.Helper()
	value := map[string]any{"op": op}
	if pid != "" {
		value["id"] = pid
	}
	return rolesMarshal(t, value)
}

func rolesMode(t *testing.T, path string) os.FileMode {
	t.Helper()
	info, err := os.Stat(path)
	Require(t, err == nil, "stat %s: %v", path, err)
	return info.Mode().Perm()
}

func rolesWaitFile(t *testing.T, path string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		if _, err := os.Stat(path); err == nil {
			return
		}
		Require(t, time.Now().Before(deadline), "timed out waiting for %s", path)
		time.Sleep(50 * time.Millisecond)
	}
}

// rolesDeadPGID returns a process group that just exited: a group leader
// is spawned, reaped, and proven gone before its pgid is handed out.
func rolesDeadPGID(t *testing.T) int {
	t.Helper()
	cmd := exec.Command("/bin/true")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	Require(t, cmd.Start() == nil, "spawn true")
	pgid := cmd.Process.Pid
	Require(t, cmd.Wait() == nil, "reap true")
	Require(t, !rolesGroupAlive(pgid), "group %d unexpectedly alive", pgid)
	return pgid
}

// rolesGroupAlive reports whether any non-zombie process keeps pgid.
func rolesGroupAlive(pgid int) bool {
	want := fmt.Sprintf("%d", pgid)
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return false
	}
	for _, entry := range entries {
		name := entry.Name()
		if name == "" {
			continue
		}
		digits := true
		for i := 0; i < len(name); i++ {
			if name[i] < '0' || name[i] > '9' {
				digits = false
				break
			}
		}
		if !digits {
			continue
		}
		raw, err := os.ReadFile(filepath.Join("/proc", name, "stat"))
		if err != nil {
			continue
		}
		text := string(raw)
		idx := strings.LastIndexByte(text, ')')
		if idx < 0 {
			continue
		}
		fields := strings.Fields(text[idx+1:])
		if len(fields) < 3 || fields[0] == "Z" || fields[2] != want {
			continue
		}
		return true
	}
	return false
}

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
