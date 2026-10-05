// The muse factory runner (rust/soda-host/src/tmuse.rs) drives headless
// `muse exec` with a fixed argv and a stream contract: stdout carries exactly
// the final answer, diagnostics go to stderr, exit 0 marks success, and a
// bogus staged auth.json fails fast with nonzero exit. This test pins that
// contract against the real CLI using the zero-spend echo provider (plus one
// invalid-auth rejection through the runner's own file-backend env), so a
// CLI behavior change fails loudly instead of silently breaking supervised
// runs. Skips when muse is not installed.
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
	// Hermetic auth: no ambient key may smuggle past the staged file,
	// exactly like the supervisor's `unset META_API_KEY`.
	var base []string
	for _, kv := range os.Environ() {
		if strings.HasPrefix(kv, "META_API_KEY=") {
			continue
		}
		base = append(base, kv)
	}
	cmd.Env = append(base, env...)
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

func TestMuseExecRejectsBadAuthFile(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	// Stage a bogus auth.json under an isolated HOME through the
	// runner's own file-backend env: the rejection below must come from
	// the file, never from ambient environment.
	home := t.TempDir()
	authDir := filepath.Join(home, ".config", "muse")
	if err := os.MkdirAll(authDir, 0o700); err != nil {
		t.Fatal(err)
	}
	auth := []byte(`{"schema_version":1,"providers":{"meta":{"api_key":"invalid-garbage-key","mechanism":"api_key"}}}`)
	if err := os.WriteFile(filepath.Join(authDir, "auth.json"), auth, 0o600); err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	code, _, stderr := runMuseExec(t, ctx,
		[]string{
			"HOME=" + home,
			"XDG_CONFIG_HOME=" + filepath.Join(home, ".config"),
			"TBH_CREDENTIAL_BACKEND=file",
			"MUSE_NO_AUTO_UPDATE=1",
		},
		"--provider", "meta", "--model", "muse-spark-1.3", "--reasoning-effort", "low")
	if elapsed := time.Since(start); elapsed > 100*time.Second {
		t.Fatalf("auth rejection took %v; refusing slow hangs", elapsed)
	}
	if code == 0 {
		t.Fatal("bogus auth.json exited 0")
	}
	if !strings.Contains(stderr, "rejected") && !strings.Contains(stderr, "authentication failed") {
		t.Fatalf("rejection unexplained: %q", stderr)
	}
}
