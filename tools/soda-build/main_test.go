//go:build linux

package main

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/acceptance"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/unix"
)

func TestControllerSubreaperReapsLeaderFirstDescendant(t *testing.T) {
	// Isolate the process-wide subreaper setting from the parent test runner.
	if os.Getenv("SODA_BUILD_REAPER_FIXTURE") != "1" {
		self, err := os.Executable()
		require.NoError(t, err)
		cmd := exec.CommandContext(t.Context(), self, "-test.run=^TestControllerSubreaperReapsLeaderFirstDescendant$")
		cmd.Env = append(os.Environ(), "SODA_BUILD_REAPER_FIXTURE=1")
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, string(output))
		return
	}
	require.NoError(t, unix.Prctl(unix.PR_SET_CHILD_SUBREAPER, 1, 0, 0, 0))
	pidFile := filepath.Join(t.TempDir(), "descendant.pid")
	cmd := exec.Command("sh", "-c", `sleep 60 & printf '%s' "$!" > "$1"`, "fixture", pidFile)
	cmd.Stdout, cmd.Stderr = io.Discard, io.Discard
	process, err := acceptance.StartCommand(t.Context(), cmd)
	require.NoError(t, err)
	require.NoError(t, process.Wait(t.Context()))
	data, err := os.ReadFile(pidFile)
	require.NoError(t, err)
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	require.NoError(t, err)
	// ESRCH, not an orphaned zombie waiting for init. Signal 0 only observes.
	require.ErrorIs(t, unix.Kill(pid, 0), unix.ESRCH)
}
