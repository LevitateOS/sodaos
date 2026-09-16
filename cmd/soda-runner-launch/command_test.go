package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/runners"
	"github.com/stretchr/testify/require"
)

func stubRunnerCaller(t *testing.T, username string) {
	t.Helper()
	original := currentUsername
	currentUsername = func() (string, error) { return username, nil }
	t.Cleanup(func() { currentUsername = original })
}

func TestExecuteRejectsForeignCaller(t *testing.T) {
	stubRunnerCaller(t, "soda-runner-other")
	err := execute([]string{"id"}, nil, func(string) (runners.LaunchCommand, error) {
		t.Fatal("foreign caller must not look up a runner")
		return runners.LaunchCommand{}, nil
	}, nil)
	require.ErrorContains(t, err, `must be launched by "soda-runner-id"`)
}

func TestExecuteRejectsInvalidRunnerIDBeforeCaller(t *testing.T) {
	err := execute([]string{"INVALID"}, nil, func(string) (runners.LaunchCommand, error) {
		t.Fatal("invalid runner id must not look up a runner")
		return runners.LaunchCommand{}, nil
	}, nil)
	require.Error(t, err)
}

func TestExecuteRejectsArgumentsBeforeLaunch(t *testing.T) {
	for _, args := range [][]string{nil, {"id", "extra"}} {
		err := execute(args, nil, func(string) (runners.LaunchCommand, error) {
			t.Fatal("invalid arguments must not look up a runner")
			return runners.LaunchCommand{}, nil
		}, nil)
		require.ErrorIs(t, err, errUsage)
	}
}

func TestExecutePropagatesLaunchFailure(t *testing.T) {
	stubRunnerCaller(t, "soda-runner-id")
	failure := errors.New("unknown runner")
	err := execute([]string{"id"}, nil, func(string) (runners.LaunchCommand, error) {
		return runners.LaunchCommand{}, failure
	}, func(runners.LaunchCommand, []string) error {
		t.Fatal("failed lookup must not replace the process")
		return nil
	})
	require.ErrorIs(t, err, failure)
}

func TestExecutePassesPlanAndReplacesHomeWithoutMutatingInput(t *testing.T) {
	stubRunnerCaller(t, "soda-runner-my-runner")
	inherited := []string{"HOME=/old", "PATH=/usr/bin", "HOME=/duplicate", "TOKEN=preserve"}
	original := append([]string(nil), inherited...)
	plan := runners.LaunchCommand{Path: "/provider", Arguments: []string{"provider", "daemon"}, Directory: "/state/app", Home: "/state"}
	failure := errors.New("exec failed")
	err := execute([]string{"my-runner"}, inherited, func(id string) (runners.LaunchCommand, error) {
		require.Equal(t, "my-runner", id)
		return plan, nil
	}, func(command runners.LaunchCommand, environment []string) error {
		require.Equal(t, plan, command)
		require.Equal(t, []string{"PATH=/usr/bin", "TOKEN=preserve", "HOME=/state"}, environment)
		return failure
	})
	require.ErrorIs(t, err, failure)
	require.Equal(t, original, inherited)
}

func TestNativeExecReplacesPIDAndPreservesExit(t *testing.T) {
	if directory := os.Getenv("SODA_TEST_EXEC_DIRECTORY"); directory != "" {
		stubRunnerCaller(t, "soda-runner-fixture")
		plan := runners.LaunchCommand{
			Path:      "/bin/sh",
			Arguments: []string{"sh", "-c", `printf '%s\n%s\n%s\n%s\n' "$$" "$PWD" "$HOME" "$1"; exit 37`, "fixture", "argument"},
			Directory: directory,
			Home:      directory,
		}
		require.NoError(t, execute([]string{"fixture"}, os.Environ(), func(string) (runners.LaunchCommand, error) { return plan, nil }, replaceProcess))
		t.Fatal("successful exec must never return")
	}
	directory := t.TempDir()
	binary, err := os.Executable()
	require.NoError(t, err)
	child := exec.CommandContext(t.Context(), binary, "-test.run=^TestNativeExecReplacesPIDAndPreservesExit$")
	child.Env = append(os.Environ(), "SODA_TEST_EXEC_DIRECTORY="+directory)
	output, err := child.CombinedOutput()
	var exit *exec.ExitError
	require.ErrorAs(t, err, &exit)
	require.Equal(t, 37, exit.ExitCode(), string(output))
	require.Equal(t, []string{fmt.Sprint(child.Process.Pid), directory, directory, "argument"}, strings.Split(strings.TrimSpace(string(output)), "\n"))
}
