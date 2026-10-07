package acceptance

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// Executed only as a child of the tests below, with synthetic private paths.
func TestOwnedChildFixture(t *testing.T) {
	mode := os.Getenv("SODA_OWNED_CHILD_TEST")
	if mode == "" {
		return
	}
	path := os.Getenv("SODA_OWNED_CHILD_PID")
	if mode == "descendant" {
		signal.Ignore(syscall.SIGTERM)
		if err := os.WriteFile(path, []byte(strconv.Itoa(os.Getpid())), 0o600); err != nil {
			os.Exit(71)
		}
		for {
			time.Sleep(time.Hour)
		}
	}
	if mode == "leader-resistant" {
		signal.Ignore(syscall.SIGTERM)
	}
	child := exec.Command(os.Args[0], "-test.run=^TestOwnedChildFixture$")
	child.Env = append(os.Environ(), "SODA_OWNED_CHILD_TEST=descendant")
	if err := child.Start(); err != nil {
		os.Exit(72)
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		if raw, err := os.ReadFile(path); err == nil && len(raw) > 0 {
			break
		}
		if time.Now().After(deadline) {
			os.Exit(73)
		}
		time.Sleep(10 * time.Millisecond)
	}
	if mode == "leader-exit" {
		os.Exit(0)
	}
	for {
		time.Sleep(time.Hour)
	}
}

func TestRunBoundedOutputFixture(t *testing.T) {
	mode := os.Getenv("SODA_BOUNDED_OUTPUT_TEST")
	if mode == "" {
		return
	}
	var stream *os.File
	size := probeStdoutLimit + 1
	if mode == "exact-stdout" {
		size = probeStdoutLimit
	}
	if mode == "stderr" || mode == "exact-stderr" {
		stream = os.Stderr
		size = probeStderrLimit + 1
		if mode == "exact-stderr" {
			size = probeStderrLimit
		}
	} else {
		stream = os.Stdout
	}
	chunk := make([]byte, 32*1024)
	for size > 0 {
		write := len(chunk)
		if size < write {
			write = size
		}
		if _, err := stream.Write(chunk[:write]); err != nil {
			os.Exit(0)
		}
		size -= write
	}
	os.Exit(0)
}

func TestOwnedLeaderExitAndCancellationStopResistantDescendant(t *testing.T) {
	for _, mode := range []string{"leader-exit", "leader-term", "leader-resistant"} {
		t.Run(mode, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "descendant.pid")
			p, err := StartProcess(context.Background(), Command{Name: os.Args[0], Args: []string{"-test.run=^TestOwnedChildFixture$"}, Env: []string{"SODA_OWNED_CHILD_TEST=" + mode, "SODA_OWNED_CHILD_PID=" + path}}, io.Discard, io.Discard)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = p.Stop() })
			deadline := time.Now().Add(5 * time.Second)
			var pid string
			for pid == "" {
				raw, _ := os.ReadFile(path)
				pid = string(raw)
				if time.Now().After(deadline) {
					t.Fatal("child not ready")
				}
				time.Sleep(10 * time.Millisecond)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
			defer cancel()
			if mode != "leader-exit" {
				stopErr := p.Stop()
				if mode == "leader-resistant" && stopErr == nil {
					t.Fatal("forced termination reported as orderly cleanup")
				}
			}
			err = p.Wait(ctx)
			if mode == "leader-exit" && err != nil {
				t.Fatal(err)
			}
			select {
			case <-p.Done():
			default:
				t.Fatal("cleanup not complete")
			}
			// A killed orphan can remain a zombie until init reaps it; never signal a
			// PID read from disk. Assert it no longer executes rather than adopting it.
			deadline = time.Now().Add(2 * time.Second)
			for {
				raw, err := os.ReadFile("/proc/" + pid + "/stat")
				// ESRCH means the descendant exited between open and
				// read: it no longer executes, same as ENOENT.
				if os.IsNotExist(err) || errors.Is(err, syscall.ESRCH) {
					break
				}
				if err != nil {
					t.Fatal(err)
				}
				tail := strings.Fields(string(raw)[strings.LastIndexByte(string(raw), ')')+1:])
				if len(tail) > 0 && tail[0] == "Z" {
					break
				}
				if time.Now().After(deadline) {
					t.Fatal("TERM-resistant descendant survived")
				}
				time.Sleep(10 * time.Millisecond)
			}
		})
	}
}

func TestRunBoundedCapsBeforeRetentionAndRejectsOverflow(t *testing.T) {
	for _, test := range []struct {
		name  string
		limit int
	}{
		{name: "stdout", limit: probeStdoutLimit},
		{name: "stderr", limit: probeStderrLimit},
	} {
		t.Run(test.name, func(t *testing.T) {
			outcome, err := runBoundedEnv(os.Args[0], []string{"-test.run=^TestRunBoundedOutputFixture$"}, nil, []string{"SODA_BOUNDED_OUTPUT_TEST=" + test.name}, 10*time.Second)
			if err == nil || len(outcome.stdout)+len(outcome.stderr) > test.limit {
				t.Fatalf("overflow was not bounded and rejected: retained stdout=%d stderr=%d err=%v", len(outcome.stdout), len(outcome.stderr), err)
			}
		})
	}
}

func TestRunBoundedAcceptsExactCaptureLimit(t *testing.T) {
	for _, test := range []struct {
		name  string
		limit int
	}{
		{name: "exact-stdout", limit: probeStdoutLimit},
		{name: "exact-stderr", limit: probeStderrLimit},
	} {
		t.Run(test.name, func(t *testing.T) {
			outcome, err := runBoundedEnv(os.Args[0], []string{"-test.run=^TestRunBoundedOutputFixture$"}, nil, []string{"SODA_BOUNDED_OUTPUT_TEST=" + test.name}, 10*time.Second)
			if err != nil || len(outcome.stdout)+len(outcome.stderr) != test.limit {
				t.Fatalf("exact-limit output rejected or miscounted: stdout=%d stderr=%d err=%v", len(outcome.stdout), len(outcome.stderr), err)
			}
		})
	}
}

func TestRunBoundedReapsDescendantHoldingOutputPipes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "descendant.pid")
	started := time.Now()
	outcome, err := runBoundedEnv(os.Args[0], []string{"-test.run=^TestOwnedChildFixture$"}, nil, []string{
		"SODA_OWNED_CHILD_TEST=leader-exit",
		"SODA_OWNED_CHILD_PID=" + path,
	}, 5*time.Second)
	if err != nil || outcome.exitCode != 0 {
		t.Fatalf("bounded command outcome=%#v err=%v", outcome, err)
	}
	if time.Since(started) > 5*time.Second {
		t.Fatal("descendant-held output pipes exceeded command allowance")
	}
}

func TestRunBoundedCancellationReportsForcedCleanup(t *testing.T) {
	path := filepath.Join(t.TempDir(), "descendant.pid")
	started := time.Now()
	_, err := runBoundedEnv(os.Args[0], []string{"-test.run=^TestOwnedChildFixture$"}, nil, []string{
		"SODA_OWNED_CHILD_TEST=leader-resistant",
		"SODA_OWNED_CHILD_PID=" + path,
	}, 100*time.Millisecond)
	if err == nil || !strings.Contains(err.Error(), "forced termination") {
		t.Fatalf("timeout cleanup was not reported: %v", err)
	}
	if time.Since(started) > 20*time.Second {
		t.Fatal("cancellation cleanup exceeded bounded allowance")
	}
}
