package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInvalidActionsDoNotCreateEvidence(t *testing.T) {
	for _, action := range []string{"publish", "exec", "native", "transfer", "vm", "probe-ssh"} {
		t.Run(action, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "evidence")
			args := []string{action, "--owner", "P07", "--revision", strings.Repeat("a", 40), "--arch", "x86_64", "--target", "fixture", "--evidence", path}
			if err := run(context.Background(), args); err == nil {
				t.Fatal("accepted unsupported owner/action")
			}
			if _, err := os.Lstat(path); !os.IsNotExist(err) {
				t.Fatal("invalid request created state")
			}
		})
	}
}

func TestCancelledExecutionRecordsFailure(t *testing.T) {
	path := filepath.Join(t.TempDir(), "evidence")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	args := []string{"exec", "--owner", "P02", "--revision", strings.Repeat("a", 40), "--arch", "x86_64", "--target", "fixture", "--evidence", path, "--", "must-not-be-started"}
	if err := run(ctx, args); err == nil {
		t.Fatal("cancelled execution succeeded")
	}
	raw, err := os.ReadFile(filepath.Join(path, "observation.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"Outcome": "cancelled"`) || !strings.Contains(string(raw), `"Execution": "not-started"`) {
		t.Fatal("cancellation presented as execution or success")
	}
}
