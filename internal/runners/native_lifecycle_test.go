package runners

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// Account/service commands are simulated; configuration, descriptor publication
// and cleanup use the real fixture filesystem.
type lifecycleCommands struct {
	t               *testing.T
	cleanupError    error
	startError      error
	deletedAccounts []string
}

func (commands *lifecycleCommands) Run(_ context.Context, command Command) (CommandResult, error) {
	switch command.Name {
	case "forgejo-runner":
		require.Equal(commands.t, []string{"--version"}, command.Args)
		return CommandResult{Stdout: "forgejo-runner fixture\n"}, nil
	case "systemctl":
		if command.Args[0] == "enable" {
			return CommandResult{}, commands.startError
		}
		return CommandResult{Stdout: "LoadState=loaded\nActiveState=inactive\nSubState=dead\nUnitFileState=disabled\n"}, nil
	case "userdel":
		commands.deletedAccounts = append(commands.deletedAccounts, command.Args[0])
		return CommandResult{}, commands.cleanupError
	default:
		commands.t.Fatalf("unexpected native command: %s", command.Name)
		return CommandResult{}, errors.New("unexpected command")
	}
}

func runnerFixture(t *testing.T) (*Native, *lifecycleCommands, preparedRunner) {
	t.Helper()
	root := t.TempDir()
	commands := &lifecycleCommands{t: t}
	native := &Native{RootPath: filepath.Join(root, "runners"), LockPath: filepath.Join(root, "runners.lock"), Runner: commands}
	prepared := preparedRunner{account: "soda-runner-one", state: native.statePath("one"), owner: identity{uint32(os.Getuid()), uint32(os.Getgid())}}
	require.NoError(t, createOwnedDirectory(prepared.state, prepared.owner))
	return native, commands, prepared
}

func forgejoRequest() CreateRequest {
	return CreateRequest{ID: "one", Provider: ProviderForgejo, RegistrationURL: BundledForgejoURL, RegistrationID: "33834eef-e758-48c4-a676-1745426747aa", Labels: "soda:host", RegistrationToken: "test-token"}
}

func TestForgejoRegistrationRetainsStateWhenListenerFailsToStart(t *testing.T) {
	native, commands, prepared := runnerFixture(t)
	commands.startError = errors.New("systemctl failed")
	err := native.registerPrepared(t.Context(), prepared, forgejoRequest())
	require.ErrorContains(t, err, "registered and retained")
	require.Empty(t, commands.deletedAccounts)
	require.FileExists(t, filepath.Join(prepared.state, "forgejo-token"))
	views, err := native.List(t.Context())
	require.NoError(t, err)
	require.Len(t, views, 1)
	require.Equal(t, "forgejo-runner fixture", views[0].Version)
	require.Equal(t, "inactive", views[0].Service.Active)
	require.Equal(t, 1, views[0].Capacity)
	launch, err := native.Launch("one")
	require.NoError(t, err)
	require.Equal(t, LaunchCommand{Path: "/usr/bin/forgejo-runner", Arguments: []string{"forgejo-runner", "daemon", "--config", filepath.Join(prepared.state, "forgejo-runner.yml")}, Directory: prepared.state, Home: prepared.state}, launch)
}

func TestForgejoDescriptorFailureReportsLocalCleanup(t *testing.T) {
	for _, cleanupFails := range []bool{false, true} {
		t.Run(map[bool]string{false: "removed", true: "retained"}[cleanupFails], func(t *testing.T) {
			native, commands, prepared := runnerFixture(t)
			require.NoError(t, os.Mkdir(native.descriptorPath("one"), 0700))
			if cleanupFails {
				commands.cleanupError = errors.New("userdel failed")
			}
			err := native.registerPrepared(t.Context(), prepared, forgejoRequest())
			require.ErrorContains(t, err, "save local runner details")
			require.Equal(t, []string{prepared.account}, commands.deletedAccounts)
			if cleanupFails {
				require.ErrorIs(t, err, commands.cleanupError)
				require.ErrorContains(t, err, "account removal is unconfirmed and state remains")
				require.DirExists(t, prepared.state)
			} else {
				require.ErrorContains(t, err, "were removed")
				require.NoDirExists(t, prepared.state)
			}
		})
	}
}

func TestForgejoConfigurationFailureDoesNotClaimFailedCleanupSucceeded(t *testing.T) {
	native, commands, prepared := runnerFixture(t)
	require.NoError(t, os.Mkdir(filepath.Join(prepared.state, "forgejo-token"), 0700))
	commands.cleanupError = errors.New("userdel failed")
	err := native.registerPrepared(t.Context(), prepared, forgejoRequest())
	require.ErrorContains(t, err, "write provider-owned Forgejo connection token")
	require.ErrorContains(t, err, "account removal is unconfirmed")
	require.NotContains(t, err.Error(), "were removed")
	require.DirExists(t, prepared.state)
}

func TestUnsupportedProvidersNeverDispatchOrRewriteRetainedState(t *testing.T) {
	for _, provider := range []Provider{"github", "unknown"} {
		t.Run(string(provider), func(t *testing.T) {
			root := t.TempDir()
			commands := &recordingCommandRunner{}
			native := &Native{RootPath: root, LockPath: filepath.Join(root, "runners.lock"), Runner: commands}
			request := forgejoRequest()
			request.Provider = provider
			require.ErrorContains(t, native.Create(t.Context(), request), "provider must be forgejo")
			require.NoDirExists(t, filepath.Join(root, "one"))
			require.NoError(t, os.MkdirAll(native.statePath("one"), 0700))
			require.NoError(t, native.writeDescriptor(Descriptor{ID: "one", Provider: provider, Account: "soda-runner-one"}))
			before, err := os.ReadFile(native.descriptorPath("one"))
			require.NoError(t, err)
			state := filepath.Join(native.statePath("one"), "retained-input")
			require.NoError(t, os.WriteFile(state, []byte("preserve fixture state"), 0600))
			_, err = native.List(t.Context())
			require.ErrorContains(t, err, "unsupported provider")
			_, err = native.Launch("one")
			require.ErrorContains(t, err, "unsupported provider")
			for _, action := range []func(context.Context, string) error{native.Start, native.Stop, native.Restart, native.Remove} {
				require.ErrorContains(t, action(t.Context(), "one"), "unsupported provider")
			}
			after, err := os.ReadFile(native.descriptorPath("one"))
			require.NoError(t, err)
			require.Equal(t, before, after)
			data, err := os.ReadFile(state)
			require.NoError(t, err)
			require.Equal(t, "preserve fixture state", string(data))
			require.Empty(t, commands.commands)
		})
	}
}
