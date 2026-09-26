package runners

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type fixtureRunnerState struct{ account, state string }

func runnerFixture(t *testing.T) (*Native, *recordingCommandRunner, fixtureRunnerState) {
	t.Helper()
	root := t.TempDir()
	commands := &recordingCommandRunner{}
	native := &Native{RootPath: root, LockPath: filepath.Join(filepath.Dir(root), "runners.lock"), Runner: commands}
	prepared := fixtureRunnerState{account: "soda-runner-one", state: native.statePath("one")}
	require.NoError(t, os.MkdirAll(prepared.state, 0o700))
	return native, commands, prepared
}

func forgejoRequest() CreateRequest {
	return CreateRequest{ID: "one", Provider: ProviderForgejo, RegistrationURL: BundledForgejoURL, RegistrationID: "33834eef-e758-48c4-a676-1745426747aa", Labels: "soda:host", RegistrationToken: "test-token"}
}

func fixtureDescriptor() Descriptor {
	return Descriptor{ID: "one", Provider: ProviderForgejo, RegistrationURL: BundledForgejoURL, Account: "soda-runner-one", Architecture: "x86-64"}
}

func TestExecutionUnavailableWithoutNativeOrFilesystemEffects(t *testing.T) {
	root := t.TempDir()
	commands := &recordingCommandRunner{}
	native := &Native{RootPath: filepath.Join(root, "absent"), LockPath: filepath.Join(root, "absent.lock"), Runner: commands}
	require.ErrorIs(t, native.Create(t.Context(), forgejoRequest()), ErrUnavailable)
	require.ErrorIs(t, native.Start(t.Context(), "one"), ErrUnavailable)
	require.ErrorIs(t, native.Restart(t.Context(), "one"), ErrUnavailable)
	operations := Operations{Local: native, Lifecycle: native}
	for _, action := range []string{"start", "restart"} {
		_, err := operations.Execute(t.Context(), action, strings.NewReader(`{"id":"one"}`))
		require.ErrorIs(t, err, ErrUnavailable)
	}
	require.Empty(t, commands.commands)
	entries, err := os.ReadDir(root)
	require.NoError(t, err)
	require.Empty(t, entries)
}

func TestStopRetainsCleanupAuthority(t *testing.T) {
	native, commands, _ := runnerFixture(t)
	require.NoError(t, native.writeDescriptor(fixtureDescriptor()))
	require.NoError(t, native.Stop(context.Background(), "one"))
	require.Equal(t, []Command{{Name: "systemctl", Args: []string{"disable", "--now", "soda-runner@one.service"}}}, commands.commands)
}
