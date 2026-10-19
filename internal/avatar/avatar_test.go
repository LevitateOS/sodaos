package avatar

import (
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"testing"
)

// TestRenderProcess is the subprocess entry for TestStableAcrossProcesses,
// not a standalone check: a direct run without the process env is a no-op.
func TestRenderProcess(t *testing.T) {
	if os.Getenv("SODA_AVATAR_PROCESS") != "1" {
		return
	}
	svg, err := Render("0123456789abcdef0123456789abcdef", 64)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("%x", sha256.Sum256([]byte(svg)))
	os.Exit(0)
}

// TestStableAcrossProcesses proves the embedded renderer delivers identical
// bytes from fresh processes with no source artwork present.
func TestStableAcrossProcesses(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	var previous string
	for range 2 {
		cmd := exec.Command(exe, "-test.run=^TestRenderProcess$")
		cmd.Dir = t.TempDir() // no source artwork is present; exercise embedded delivery
		cmd.Env = append(os.Environ(), "SODA_AVATAR_PROCESS=1")
		out, err := cmd.Output()
		if err != nil {
			t.Fatal(err)
		}
		if len(out) != 64 {
			t.Fatalf("unexpected child output: %q", out)
		}
		if previous != "" && previous != string(out) {
			t.Fatal("avatar changed on process restart")
		}
		previous = string(out)
	}
}
