// Package build ports tests/build/*.py to Go: same assertions, same
// fixtures, same subprocesses. The Python files are untouched; a final
// cleanup lane owns their deletion.
package build

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

// RepoRoot is the checkout root derived from this file's location
// (tests/build/helpers.go -> two levels up), like ROOT in the Python tests.
var RepoRoot = func() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		panic("cannot locate helpers.go")
	}
	return filepath.Dir(filepath.Dir(filepath.Dir(file)))
}()

// Check fails the test with a formatted message unless cond holds.
func Check(t *testing.T, cond bool, format string, args ...any) {
	t.Helper()
	if !cond {
		t.Errorf(format, args...)
	}
}

// Require fails the test immediately unless cond holds.
func Require(t *testing.T, cond bool, format string, args ...any) {
	t.Helper()
	if !cond {
		t.Fatalf(format, args...)
	}
}

// ReadFile reads a repo-relative file as text.
func ReadFile(t *testing.T, rel string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(RepoRoot, rel))
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	return string(data)
}

// ReadJSON reads a repo-relative JSON file.
func ReadJSON(t *testing.T, rel string) any {
	t.Helper()
	var value any
	if err := json.Unmarshal([]byte(ReadFile(t, rel)), &value); err != nil {
		t.Fatalf("parse %s: %v", rel, err)
	}
	return value
}

// TempDir makes a test-owned temporary directory.
func TempDir(t *testing.T) string {
	t.Helper()
	return t.TempDir()
}

// WriteFile writes data to path, creating parents, like pathlib write_bytes.
func WriteFile(t *testing.T, path string, data []byte, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir for %s: %v", path, err)
	}
	if err := os.WriteFile(path, data, mode); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// ProcResult is a completed subprocess, like CompletedProcess.
type ProcResult struct {
	Code   int
	Stdout string
	Stderr string
}

// RunOpt tunes Run; zero values mirror subprocess.run defaults used by the
// Python tests (capture output, no check, inherit environment).
type RunOpt struct {
	Cwd     string
	Env     []string // full environment; nil inherits os.Environ()
	Timeout time.Duration
}

// Run executes name with args and captures output and exit code.
func Run(t *testing.T, opt RunOpt, name string, args ...string) ProcResult {
	t.Helper()
	ctx := context.Background()
	var cancel context.CancelFunc
	if opt.Timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, opt.Timeout)
		defer cancel()
	}
	cmd := exec.CommandContext(ctx, name, args...)
	if opt.Cwd != "" {
		cmd.Dir = opt.Cwd
	}
	if opt.Env != nil {
		cmd.Env = opt.Env
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	code := 0
	if err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			code = exit.ExitCode()
		} else {
			t.Fatalf("run %s %v: %v", name, args, err)
		}
	}
	return ProcResult{Code: code, Stdout: stdout.String(), Stderr: stderr.String()}
}

// SetEnv returns env with key set to val, replacing any existing entry
// (Python's {**os.environ, key: val} replacement semantics).
func SetEnv(env []string, key, val string) []string {
	prefix := key + "="
	out := make([]string, 0, len(env)+1)
	for _, entry := range env {
		if !strings.HasPrefix(entry, prefix) {
			out = append(out, entry)
		}
	}
	return append(out, prefix+val)
}

// Python3 resolves the python3 binary (the Go spelling of sys.executable).
func Python3(t *testing.T) string {
	t.Helper()
	path, err := exec.LookPath("python3")
	if err != nil {
		t.Fatalf("python3 required: %v", err)
	}
	abs, err := filepath.EvalSymlinks(path)
	if err != nil {
		return path
	}
	if !filepath.IsAbs(abs) {
		return path
	}
	return abs
}

var (
	cargoMu    sync.Mutex
	cargoBuilt = map[string]string{} // build key -> "" on success, stderr tail on failure
)

// CargoBinary builds a Rust binary once per key (like setUpClass) and returns
// its target/debug path. extra holds cargo build flags after -p pkg.
func CargoBinary(t *testing.T, pkg, bin string, extra ...string) string {
	t.Helper()
	args := append([]string{"build", "-p", pkg}, extra...)
	key := strings.Join(args, "\x00")
	cargoMu.Lock()
	failure, seen := cargoBuilt[key]
	cargoMu.Unlock()
	if !seen {
		cmd := exec.Command("cargo", args...)
		cmd.Dir = RepoRoot
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		failure = ""
		if err := cmd.Run(); err != nil {
			failure = tail(stderr.String(), 2000)
		}
		cargoMu.Lock()
		cargoBuilt[key] = failure
		cargoMu.Unlock()
	}
	if failure != "" {
		t.Fatalf("cannot build %s: %s", bin, failure)
	}
	return filepath.Join(RepoRoot, "target", "debug", bin)
}

func tail(s string, n int) string {
	if len(s) > n {
		return s[len(s)-n:]
	}
	return s
}
