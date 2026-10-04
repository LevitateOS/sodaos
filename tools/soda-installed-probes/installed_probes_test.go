package main

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

// probeRoot is the checkout root derived from this file's location.
var probeRoot = func() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		panic("cannot locate installed_probes_test.go")
	}
	return filepath.Dir(filepath.Dir(filepath.Dir(file)))
}()

func checkProbe(t *testing.T, cond bool, format string, args ...any) {
	t.Helper()
	if !cond {
		t.Errorf(format, args...)
	}
}

func requireProbe(t *testing.T, cond bool, format string, args ...any) {
	t.Helper()
	if !cond {
		t.Fatalf(format, args...)
	}
}

type probeResult struct {
	code   int
	stdout string
	stderr string
}

func runProbe(t *testing.T, env []string, timeout time.Duration, name string, args ...string) probeResult {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	if env != nil {
		cmd.Env = env
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	code := 0
	if err := cmd.Run(); err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			code = exit.ExitCode()
		} else {
			t.Fatalf("run %s: %v", name, err)
		}
	}
	return probeResult{code: code, stdout: stdout.String(), stderr: stderr.String()}
}

func readProbeSource(t *testing.T, rel string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(probeRoot, rel))
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	return string(data)
}
