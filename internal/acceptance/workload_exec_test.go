package acceptance

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeExecFixture(t *testing.T, target string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.Chmod(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if target != "" {
		if err := os.WriteFile(filepath.Join(dir, "target.json"), []byte(target), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// stubSSH installs a fake ssh that emulates the member sessions: alice prints
// the database name, bob is denied engine access. It logs argv for assertions.
func stubSSH(t *testing.T, aliceStdout string, bobExit int) string {
	t.Helper()
	dir := t.TempDir()
	log := filepath.Join(dir, "argv.log")
	inputDir := filepath.Join(dir, "stdin")
	if err := os.Mkdir(inputDir, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("STUBSSH_INPUT_DIR", inputDir)
	script := "#!/bin/sh\n" +
		"printf '%s\\n' \"$@\" >> " + log + "\n" +
		"login=\"\"; for arg in \"$@\"; do case \"$arg\" in u08-alice-8417@*) login=alice;; u08-bob-8417@*) echo 'permission denied: engine socket' >&2; exit " + bobExitString(bobExit) + ";; esac; done\n" +
		"cat > \"$STUBSSH_INPUT_DIR/$login\"\n" +
		"printf '%s' '" + aliceStdout + "'\n"
	path := filepath.Join(dir, "ssh")
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return log
}

func bobExitString(code int) string {
	if code == 0 {
		return "0"
	}
	return "1"
}

func TestRunWorkloadExec(t *testing.T) {
	t.Setenv("SODA_NATIVE_VALIDATE", "soda-test")
	fixture := writeExecFixture(t, `{"ip": "10.89.0.2"}`)
	log := stubSSH(t, "soda_example", 1)
	var stdout bytes.Buffer
	if err := RunWorkloadExec([]string{fixture}, &stdout); err != nil {
		t.Fatalf("exec probe failed: %v", err)
	}
	if !strings.Contains(stdout.String(), "ordinary member denied engine access") {
		t.Errorf("unexpected success line: %q", stdout.String())
	}
	argv, err := os.ReadFile(log)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"-F", "BatchMode=yes", "sh -se", "u08-alice-8417@10.89.0.2", "u08-bob-8417@10.89.0.2"} {
		if !strings.Contains(string(argv), want) {
			t.Errorf("argv log misses %q:\n%s", want, argv)
		}
	}
	probe, err := os.ReadFile(filepath.Join(filepath.Dir(log), "stdin", "alice"))
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{
		"set -eu",
		"grep -Eq '^Uid:[[:space:]]+999[[:space:]]+999[[:space:]]+999[[:space:]]+999$' /proc/1/status",
		`test "$(podman exec workload_database_1 id -u)" = 0`,
		`test "$(podman exec --user postgres workload_database_1 id -u)" = 999`,
		"psql -X -U developer -d soda_example -At -c 'select current_database()'",
		"exec -t --user postgres workload_database_1 true",
	} {
		if !strings.Contains(string(probe), required) {
			t.Errorf("piped sh -se probe misses required check %q:\n%s", required, probe)
		}
	}
}

func TestRunWorkloadExecRefusals(t *testing.T) {
	t.Setenv("SODA_NATIVE_VALIDATE", "soda-test")
	t.Run("wrong database", func(t *testing.T) {
		t.Setenv("SODA_NATIVE_VALIDATE", "soda-test")
		stubSSH(t, "wrong_database", 1)
		err := RunWorkloadExec([]string{writeExecFixture(t, `{"ip": "10.89.0.2"}`)}, &bytes.Buffer{})
		if err == nil || !strings.HasPrefix(err.Error(), "Workload exec check incomplete (") {
			t.Errorf("wrong database error = %v", err)
		}
	})
	t.Run("missing denial", func(t *testing.T) {
		t.Setenv("SODA_NATIVE_VALIDATE", "soda-test")
		stubSSH(t, "soda_example", 0)
		err := RunWorkloadExec([]string{writeExecFixture(t, `{"ip": "10.89.0.2"}`)}, &bytes.Buffer{})
		if err == nil {
			t.Error("missing denial accepted")
		}
	})
	t.Run("bad fixture", func(t *testing.T) {
		t.Setenv("SODA_NATIVE_VALIDATE", "soda-test")
		stubSSH(t, "soda_example", 1)
		for _, target := range []string{`{"ip": "10.90.0.2"}`, `{"ip": "not-an-ip"}`, `{"host": "10.89.0.2"}`, `not json`, ""} {
			if err := RunWorkloadExec([]string{writeExecFixture(t, target)}, &bytes.Buffer{}); err == nil {
				t.Errorf("bad target accepted: %q", target)
			}
		}
		if err := RunWorkloadExec([]string{writeExecFixture(t, `{"ip": "10.89.0.2"}`), "extra"}, &bytes.Buffer{}); err == nil {
			t.Error("extra argv accepted")
		}
	})
	t.Run("wrong env", func(t *testing.T) {
		t.Setenv("SODA_NATIVE_VALIDATE", "wrong-target")
		if err := RunWorkloadExec([]string{writeExecFixture(t, `{"ip": "10.89.0.2"}`)}, &bytes.Buffer{}); err == nil {
			t.Error("wrong env accepted")
		}
	})
}
