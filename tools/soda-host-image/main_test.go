package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Exercise the actual entrypoint and timing bridge with refusal-only inputs.
// No build, Podman operation, fixture lifecycle, publication or source mutation.
func TestBuildCLIChild(t *testing.T) {
	if os.Getenv("SODA_BUILD_CLI_TEST") != "1" {
		return
	}
	args := strings.Split(os.Getenv("SODA_BUILD_CLI_ARGS"), "\n")
	os.Args = append([]string{os.Args[0]}, args...)
	main()
}
func TestBuildCLIRejectsMixedLayoutsWithFailureTiming(t *testing.T) {
	for _, args := range [][]string{
		{"--legacy-native", "--build"},
		{"--legacy-native", "--complete"},
		{"--legacy-native", "--repository-prefix", "ghcr.io/levitateos/sodaos"},
		{"--complete"},
		{"--repository-prefix", "ghcr.io/levitateos/sodaos"},
	} {
		out := filepath.Join(t.TempDir(), "must-not-exist")
		args = append(args, "--out", out)
		cmd := exec.Command(os.Args[0], "-test.run=^TestBuildCLIChild$")
		env := []string{}
		for _, entry := range os.Environ() {
			if !strings.HasPrefix(entry, "SODA_BUILD_") {
				env = append(env, entry)
			}
		}
		cmd.Env = append(env, "SODA_BUILD_SUPERVISED=1", "SODA_BUILD_CLI_TEST=1", "SODA_BUILD_CLI_ARGS="+strings.Join(args, "\n"))
		b, e := cmd.CombinedOutput()
		if e == nil || !strings.Contains(string(b), "FAILED") || strings.Contains(string(b), "SUCCESS") {
			t.Fatalf("%v: %v\n%s", args, e, b)
		}
		if _, e = os.Stat(out); !os.IsNotExist(e) {
			t.Fatal("refused request created output", e)
		}
	}
}
