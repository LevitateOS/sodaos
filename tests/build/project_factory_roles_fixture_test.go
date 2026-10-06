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
	return CargoBinary(t, "soda-project-terminal", "project-factory-roles")
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
