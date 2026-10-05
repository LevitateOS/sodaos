// The muse-code factory runner (rust/soda-host/src/tmuse.rs) drives headless
// `muse exec` with a fixed argv and a stream contract: stdout carries exactly
// the final answer, diagnostics go to stderr, exit 0 marks success, and a
// rejected META_API_KEY fails fast with exit 1. This test pins that contract
// against the real CLI using the zero-spend echo provider (plus one
// invalid-key rejection), so a CLI behavior change fails loudly instead of
// silently breaking supervised runs. Skips when muse is not installed.
package build

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func museBinary(t *testing.T) string {
	t.Helper()
	if path, err := exec.LookPath("muse"); err == nil {
		return path
	}
	if _, err := os.Stat("/home/vince/.local/bin/muse"); err == nil {
		return "/home/vince/.local/bin/muse"
	}
	t.Skip("muse CLI unavailable")
	return ""
}

func runMuseExec(t *testing.T, ctx context.Context, env []string, args ...string) (int, string, string) {
	t.Helper()
	cmd := exec.CommandContext(ctx, museBinary(t), args...)
	workspace := t.TempDir()
	prompt := filepath.Join(workspace, "prompt")
	if err := os.WriteFile(prompt, []byte("say hello"), 0o600); err != nil {
		t.Fatal(err)
	}
	full := []string{"exec", "--prompt-file", prompt, "--workspace", workspace,
		"--trust-workspace", "--no-session-log", "--disable-approval", "--disable-sandbox"}
	full = append(full, args...)
	cmd.Args = append([]string{cmd.Args[0]}, full...)
	cmd.Env = append(os.Environ(), env...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	code := 0
	if err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			code = exit.ExitCode()
		} else {
			t.Fatalf("muse exec failed to start: %v", err)
		}
	}
	return code, stdout.String(), stderr.String()
}

func TestMuseExecStreamContract(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	code, stdout, stderr := runMuseExec(t, ctx, nil, "--provider", "echo")
	if code != 0 {
		t.Fatalf("echo run exit = %d, stderr = %q", code, stderr)
	}
	if !strings.Contains(stdout, "say hello") {
		t.Fatalf("stdout lost the answer: %q", stdout)
	}
	for _, line := range strings.Split(stdout, "\n") {
		if strings.HasPrefix(line, "muse: ") {
			t.Fatalf("diagnostic leaked to stdout: %q", line)
		}
	}
	if !strings.Contains(stderr, "muse: ") {
		t.Fatalf("stderr lost diagnostics: %q", stderr)
	}
}

func TestMuseExecRejectsBadKey(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	start := time.Now()
	code, _, stderr := runMuseExec(t, ctx,
		[]string{"META_API_KEY=invalid-garbage-key"},
		"--provider", "meta", "--model", "muse-spark-1.3", "--reasoning-effort", "low")
	if elapsed := time.Since(start); elapsed > 100*time.Second {
		t.Fatalf("key rejection took %v; refusing slow hangs", elapsed)
	}
	if code == 0 {
		t.Fatal("invalid META_API_KEY exited 0")
	}
	if !strings.Contains(stderr, "API key") && !strings.Contains(stderr, "rejected") {
		t.Fatalf("rejection unexplained: %q", stderr)
	}
}
