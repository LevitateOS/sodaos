// Optional live CLI coverage for the muse factory runner. Running these cases
// is an explicit development action: SODA_TEST_MUSE_CLI must name the binary.
// The ordinary build suite tests selection behavior without discovering a
// user's ambient CLI.
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

func selectedMuseCLI(value string) (string, bool) {
	if value == "" || !filepath.IsAbs(value) {
		return "", false
	}
	return value, true
}

func museBinary(t *testing.T) string {
	t.Helper()
	path, selected := selectedMuseCLI(os.Getenv("SODA_TEST_MUSE_CLI"))
	if !selected {
		t.Skip("set SODA_TEST_MUSE_CLI to an absolute Muse binary path to run live CLI checks")
	}
	info, err := os.Stat(path)
	if err != nil || info.IsDir() || info.Mode()&0o111 == 0 {
		t.Fatalf("SODA_TEST_MUSE_CLI must name an executable file: %s", path)
	}
	return path
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

func TestMuseExecCLISelectionIsExplicit(t *testing.T) {
	if path, ok := selectedMuseCLI(""); ok || path != "" {
		t.Fatalf("empty selector chose a CLI: %q", path)
	}
	if path, ok := selectedMuseCLI("relative/muse"); ok || path != "" {
		t.Fatalf("relative selector was accepted: %q", path)
	}
	want := "/opt/muse/bin/muse"
	if path, ok := selectedMuseCLI(want); !ok || path != want {
		t.Fatalf("explicit selector = %q, %v", path, ok)
	}
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
