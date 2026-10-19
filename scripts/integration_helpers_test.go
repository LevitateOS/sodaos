// Helpers retained for the kept integration tests after unit-test removal.
package scripts

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func buildRustPortBinary(t *testing.T, crate string) string {
	t.Helper()
	return buildRustPortBinaryAs(t, crate, crate)
}

// buildRustPortBinaryAs builds one binary of a multi-binary Cargo package.
// TAILNET-19-001: the tailnet CLI lives in the soda-host package, so the
// package==bin assumption no longer holds for every ported caller.
func buildRustPortBinaryAs(t *testing.T, crate string, bin string) string {
	t.Helper()
	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "cargo", "build", "-p", crate, "--bin", bin)
	cmd.Dir = root
	if combined, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build rust/%s bin %s: %v\n%s", crate, bin, err, combined)
	}
	out := filepath.Join(root, "target", "debug", bin)
	if _, err := os.Stat(out); err != nil {
		t.Fatalf("rust %s binary %s missing: %v", crate, bin, err)
	}
	return out
}
