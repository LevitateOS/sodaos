package runners

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// This subprocess is the Go test executable, not a production CLI with test flags.
// Both processes call Native with the same temporary lock and command doubles.
func TestNativeRunnerLockContender(t *testing.T) {
	action := os.Getenv("SODA_TEST_RUNNER_LOCK_ACTION")
	if action == "" {
		return
	}
	root := os.Getenv("SODA_TEST_RUNNER_LOCK_ROOT")
	require.NotEmpty(t, root)
	blocked := os.Getenv("SODA_TEST_RUNNER_LOCK_BLOCKED") == "1"
	ctx, cancel := context.WithTimeout(t.Context(), 250*time.Millisecond)
	defer cancel()
	calls := 0
	native := &Native{RootPath: root, LockPath: filepath.Join(filepath.Dir(root), "runners.lock"), Runner: runnerCommandFunc(func(_ context.Context, command Command) (CommandResult, error) {
		calls++
		require.False(t, blocked, "cancelled waiter dispatched a native command")
		require.Equal(t, Command{Name: "systemctl", Args: []string{"enable", "--now", "soda-runner@one.service"}}, command)
		return CommandResult{}, nil
	})}
	var err error
	switch action {
	case "list":
		_, err = native.List(ctx)
	case "create":
		err = native.Create(ctx, forgejoRequest())
	case "start":
		err = native.Start(ctx, "one")
	case "stop":
		err = native.Stop(ctx, "one")
	case "restart":
		err = native.Restart(ctx, "one")
	case "remove":
		err = native.Remove(ctx, "one")
	default:
		t.Fatal("unknown test operation")
	}
	if blocked {
		require.ErrorIs(t, err, context.DeadlineExceeded)
		require.Zero(t, calls)
	} else {
		require.NoError(t, err)
		require.Equal(t, 1, calls)
	}
}

func runRunnerLockContender(t *testing.T, native *Native, action string, blocked bool) {
	t.Helper()
	executable, err := os.Executable()
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, executable, "-test.run=^TestNativeRunnerLockContender$", "-test.count=1")
	wait := "0"
	if blocked {
		wait = "1"
	}
	command.Env = append(os.Environ(), "SODA_TEST_RUNNER_LOCK_ROOT="+native.RootPath, "SODA_TEST_RUNNER_LOCK_ACTION="+action, "SODA_TEST_RUNNER_LOCK_BLOCKED="+wait)
	output, err := command.CombinedOutput()
	require.NoError(t, err, "contender %s: %s", action, output)
}

func TestEveryReadAndMutationUsesTheCrossProcessLock(t *testing.T) {
	native, _, prepared := runnerFixture(t)
	require.NoError(t, native.recordRunner(prepared.account, forgejoRequest()))
	lock, err := native.lock(t.Context())
	require.NoError(t, err)
	defer lock.Close()
	for _, action := range []string{"list", "create", "start", "stop", "restart", "remove"} {
		t.Run(action, func(t *testing.T) { runRunnerLockContender(t, native, action, true) })
	}
	require.NoError(t, lock.Close())
	// A fresh call after the cancelled processes have exited can acquire the lock.
	// None of the timed-out requests is replayed on release.
	runRunnerLockContender(t, native, "start", false)
}

func TestRestartKeepsCrossProcessAdmissionAcrossEnableAndRestart(t *testing.T) {
	for _, failure := range []string{"", "enable", "restart"} {
		t.Run("failure-"+failure, func(t *testing.T) {
			native, _, prepared := runnerFixture(t)
			require.NoError(t, native.recordRunner(prepared.account, forgejoRequest()))
			var actions []string
			native.Runner = runnerCommandFunc(func(_ context.Context, command Command) (CommandResult, error) {
				require.Equal(t, "systemctl", command.Name)
				require.Len(t, command.Args, 2)
				require.Equal(t, "soda-runner@one.service", command.Args[1])
				action := command.Args[0]
				actions = append(actions, action)
				// Actual second-process Native calls must remain excluded at both command
				// boundaries, including when enable/restart subsequently reports failure.
				runRunnerLockContender(t, native, "start", true)
				if action == failure {
					return CommandResult{}, errors.New("simulated service failure")
				}
				return CommandResult{}, nil
			})
			err := native.Restart(t.Context(), "one")
			if failure == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, failure+" local runner listener")
			}
			expected := []string{"enable"}
			if failure != "enable" {
				expected = append(expected, "restart")
			}
			require.Equal(t, expected, actions)
			runRunnerLockContender(t, native, "start", false)
		})
	}
}
