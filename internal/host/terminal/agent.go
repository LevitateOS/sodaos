package terminal

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

// AgentProgram is the fixed project-local terminal agent executed inside
// project containers. Both the path and the argument vector are product code,
// never caller-selected.
const AgentProgram = "/usr/libexec/soda/project-terminal"

// agentProgramPath resolves the host-side copy the daemon verifies. Tests
// override it with SODA_PROJECT_TERMINAL; the ownership, mode and size
// requirements still apply.
func agentProgramPath() string {
	if v := os.Getenv("SODA_PROJECT_TERMINAL"); v != "" {
		return v
	}
	return AgentProgram
}

// AgentProgramHash verifies the fixed host-side agent binary and returns its
// lowercase hex SHA-256 digest. It fails closed: a missing, non-regular,
// unsafely owned or unreadable file is an error, never a fallback.
func AgentProgramHash() (string, error) {
	return agentProgramHash(0)
}

// agentProgramHash verifies ownership against uid so tests can use their own
// UID in a private directory; production always passes 0.
func agentProgramHash(uid uint32) (string, error) {
	path := agentProgramPath()
	info, err := os.Lstat(path)
	if err != nil {
		return "", fmt.Errorf("project terminal agent unavailable: %w", err)
	}
	if !info.Mode().IsRegular() {
		return "", errors.New("project terminal agent is not a regular file")
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || stat.Uid != uid {
		return "", errors.New("project terminal agent has unexpected ownership")
	}
	if info.Mode().Perm()&0o022 != 0 {
		return "", errors.New("project terminal agent is group- or world-writable")
	}
	if info.Size() < 1 || info.Size() > 32<<20 {
		return "", errors.New("project terminal agent has unexpected size")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("project terminal agent unreadable: %w", err)
	}
	if len(raw) < 1 || len(raw) > 32<<20 {
		return "", errors.New("project terminal agent changed during verification")
	}
	sum := sha256.Sum256(raw)
	return fmt.Sprintf("%x", sum), nil
}

// AgentExec builds the fixed podman invocation of the project terminal agent.
// Stdin is wired from stdin when non-nil; a nil stdin leaves cmd.Stdin unset
// for callers that stream through StdinPipe.
func AgentExec(container string, stdin []byte, args ...string) *exec.Cmd {
	argv := append([]string{"--remote=false", "exec", "--interactive", container, AgentProgram}, args...)
	cmd := exec.Command("/usr/bin/podman", argv...)
	if stdin != nil {
		cmd.Stdin = bytes.NewReader(stdin)
	}
	return cmd
}
