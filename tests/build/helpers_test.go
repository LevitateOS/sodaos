package build

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func writeFakeCargo(t *testing.T, dir, body string) {
	t.Helper()
	path := filepath.Join(dir, "cargo")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body), 0o700); err != nil {
		t.Fatal(err)
	}
}

func cargoTestEnv(t *testing.T, fakeBin, log, mode, target string) {
	t.Helper()
	t.Setenv("PATH", fakeBin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("CARGO_FIXTURE_LOG", log)
	t.Setenv("CARGO_FIXTURE_MODE", mode)
	t.Setenv("CARGO_TARGET_DIR", target)
}

func TestCargoBinaryHelperProcess(t *testing.T) {
	if os.Getenv("SODA_TEST_CARGO_HELPER") != "1" {
		return
	}
	path := CargoBinary(t,
		os.Getenv("SODA_TEST_CARGO_PKG"),
		os.Getenv("SODA_TEST_CARGO_BIN"),
		"--luna-fixture",
	)
	if err := os.WriteFile(os.Getenv("SODA_TEST_CARGO_MARKER"), []byte(path), 0o600); err != nil {
		t.Fatal(err)
	}
}

func runCargoBinaryHelper(t *testing.T, pkg, bin, marker string) *exec.Cmd {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=^TestCargoBinaryHelperProcess$")
	cmd.Dir = RepoRoot
	env := os.Environ()
	env = SetEnv(env, "SODA_TEST_CARGO_HELPER", "1")
	env = SetEnv(env, "SODA_TEST_CARGO_PKG", pkg)
	env = SetEnv(env, "SODA_TEST_CARGO_BIN", bin)
	cmd.Env = SetEnv(env, "SODA_TEST_CARGO_MARKER", marker)
	return cmd
}

func TestCargoBinaryEmptyStderrFailureIsNotCachedAsSuccess(t *testing.T) {
	dir := t.TempDir()
	fakeBin := filepath.Join(dir, "bin")
	if err := os.MkdirAll(fakeBin, 0o755); err != nil {
		t.Fatal(err)
	}
	log := filepath.Join(dir, "cargo.log")
	writeFakeCargo(t, fakeBin, `printf 'called\n' >>"$CARGO_FIXTURE_LOG"
if [ "$CARGO_FIXTURE_MODE" = fail ]; then exit 1; fi
exit 0
`)
	relativeTarget, err := filepath.Rel(RepoRoot, filepath.Join(dir, "target"))
	if err != nil {
		t.Fatal(err)
	}
	cargoTestEnv(t, fakeBin, log, "fail", relativeTarget)
	stale := filepath.Join(RepoRoot, relativeTarget, "debug", "fixture-bin")
	if err := os.MkdirAll(filepath.Dir(stale), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(stale, []byte("stale"), 0o755); err != nil {
		t.Fatal(err)
	}

	for attempt := 0; attempt < 2; attempt++ {
		path, result := cargoBinary("empty-stderr-fixture", "fixture-bin", "--luna-fixture")
		if path != "" || result.processError == "" {
			t.Fatalf("attempt %d returned %q with result %+v", attempt+1, path, result)
		}
	}
	if got := countLines(t, log); got != 1 {
		t.Fatalf("cached failure invoked Cargo %d times", got)
	}

	marker := filepath.Join(dir, "returned-path")
	cmd := runCargoBinaryHelper(t, "empty-stderr-fixture", "fixture-bin", marker)
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("CargoBinary accepted empty-stderr failure; output=%s", output)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("failed build returned a path, marker stat err=%v", err)
	}
	if got := countLines(t, log); got != 2 {
		t.Fatalf("CargoBinary subprocess did not execute fake Cargo once: calls=%d", got)
	}
}

func TestCargoBinaryUsesAndKeysSelectedTargetDirectory(t *testing.T) {
	dir := t.TempDir()
	fakeBin := filepath.Join(dir, "bin")
	if err := os.MkdirAll(fakeBin, 0o755); err != nil {
		t.Fatal(err)
	}
	log := filepath.Join(dir, "cargo.log")
	writeFakeCargo(t, fakeBin, `printf 'called\n' >>"$CARGO_FIXTURE_LOG"
exit 0
`)
	cargoTestEnv(t, fakeBin, log, "success", filepath.Join(dir, "target-absolute"))
	relativeTarget, err := filepath.Rel(RepoRoot, filepath.Join(dir, "target-relative"))
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		name, target string
	}{
		{name: "absolute", target: filepath.Join(dir, "target-absolute")},
		{name: "relative", target: relativeTarget},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("CARGO_TARGET_DIR", tc.target)
			marker := filepath.Join(dir, "returned-"+tc.name)
			cmd := runCargoBinaryHelper(t, "explicit-target-fixture", "fixture-bin", marker)
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("CargoBinary failed: %v: %s", err, output)
			}
			got, err := os.ReadFile(marker)
			if err != nil {
				t.Fatalf("CargoBinary did not return a path: %v", err)
			}
			expectedTarget := tc.target
			if !filepath.IsAbs(expectedTarget) {
				expectedTarget = filepath.Join(RepoRoot, expectedTarget)
			}
			want := filepath.Join(expectedTarget, "debug", "fixture-bin")
			if string(got) != want {
				t.Fatalf("CargoBinary path = %q, want %q", got, want)
			}
		})
	}

	// The same build arguments in one process must still invoke Cargo when the
	// selected target directory changes.
	t.Setenv("CARGO_TARGET_DIR", filepath.Join(dir, "cache-target-a"))
	first, firstResult := cargoBinary("target-cache-fixture", "fixture-bin", "--luna-fixture")
	t.Setenv("CARGO_TARGET_DIR", filepath.Join(dir, "cache-target-b"))
	second, secondResult := cargoBinary("target-cache-fixture", "fixture-bin", "--luna-fixture")
	if firstResult.processError != "" || secondResult.processError != "" || first == second {
		t.Fatalf("target selection was not reflected in build result: %q (%+v), %q (%+v)", first, firstResult, second, secondResult)
	}
	if got := countLines(t, log); got != 4 {
		t.Fatalf("expected two CLI builds and two distinct target builds, got %d", got)
	}
}

func countLines(t *testing.T, path string) int {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return strings.Count(string(data), "\n")
}
