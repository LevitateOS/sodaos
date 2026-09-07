package acceptance

import (
	"context"
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
		if err := os.WriteFile(path, []byte(strconv.Itoa(os.Getpid())), 0600); err != nil {
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
				if os.IsNotExist(err) {
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
