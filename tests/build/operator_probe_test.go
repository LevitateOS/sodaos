// Port of test_operator_probe.py: a missing welcome hook must not pass.
package build

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestOperatorProbeMissingFailedNoisyAndQuietHooks(t *testing.T) {
	var probe, assertion string
	for _, line := range strings.Split(ReadFile(t, "tests/installed/operator.sh"), "\n") {
		if probe == "" && strings.HasPrefix(line, "quiet=$(") {
			probe = line
		}
		if assertion == "" && strings.HasPrefix(line, `[[ "$quiet"`) {
			assertion = line
		}
	}
	Require(t, probe != "", "probe line not found in operator.sh")
	Require(t, assertion != "", "assertion line not found in operator.sh")
	contents := []string{"\x00", "false\n", "echo noise\n", ":\n"}
	want := []bool{false, false, false, true}
	for i, content := range contents {
		name := "missing"
		if content != "\x00" {
			name = "content=" + strings.TrimSuffix(content, "\n")
		}
		t.Run(name, func(t *testing.T) {
			dir := TempDir(t)
			hook := filepath.Join(dir, "hook")
			if content != "\x00" {
				WriteFile(t, hook, []byte(content), 0o644)
			}
			command := strings.ReplaceAll(probe, "/etc/profile.d/soda-console-welcome.sh", hook) + "\n" + assertion
			result := Run(t, RunOpt{Env: os.Environ(), Timeout: 5 * time.Second}, "bash", "-ec", command)
			Check(t, result.Code == 0 == want[i], "exit %d, want success=%v", result.Code, want[i])
		})
	}
}
