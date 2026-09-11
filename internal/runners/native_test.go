package runners

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/stretchr/testify/require"
)

type recordingCommandRunner struct {
	commands []Command
}

func (runner *recordingCommandRunner) Run(_ context.Context, command Command) (CommandResult, error) {
	runner.commands = append(runner.commands, command)
	return CommandResult{}, nil
}

func TestForgejoConfigurationUsesNativeTokenFileAndOneHostSlot(t *testing.T) {
	state := t.TempDir()
	native := NewNative()
	request := CreateRequest{
		ID: "forgejo-one", Provider: ProviderForgejo,
		RegistrationURL: "http://soda.example.test:30000",
		RegistrationID:  "33834eef-e758-48c4-a676-1745426747aa",
		Labels:          "soda-arm64:host", RegistrationToken: "provider-input",
	}
	require.NoError(t, native.configureForgejo(state, identity{uint32(os.Getuid()), uint32(os.Getgid())}, request))

	contents, err := os.ReadFile(filepath.Join(state, "forgejo-runner.yml"))
	require.NoError(t, err)
	require.NotContains(t, string(contents), request.RegistrationToken)
	var configuration map[string]any
	require.NoError(t, json.Unmarshal(contents, &configuration))
	runner := configuration["runner"].(map[string]any)
	require.Equal(t, float64(1), runner["capacity"])
	require.Equal(t, []any{"soda-arm64:host"}, runner["labels"])
	require.Equal(t, map[string]any{"enabled": false}, configuration["cache"])

	token, err := os.ReadFile(filepath.Join(state, "forgejo-token"))
	require.NoError(t, err)
	require.Equal(t, request.RegistrationToken+"\n", string(token))
	info, err := os.Stat(filepath.Join(state, "forgejo-token"))
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o600), info.Mode().Perm())
}

func TestLifecycleActionsPersistListenerStateAcrossBoot(t *testing.T) {
	root := t.TempDir()
	runner := &recordingCommandRunner{}
	native := &Native{RootPath: root, LockPath: filepath.Join(root, "runners.lock"), Runner: runner}
	id := "one"
	require.NoError(t, os.MkdirAll(filepath.Join(root, id), 0o755))
	require.NoError(t, native.writeDescriptor(Descriptor{ID: id, Provider: ProviderForgejo, Account: "soda-runner-one"}))

	require.NoError(t, native.Start(context.Background(), id))
	require.NoError(t, native.Stop(context.Background(), id))
	require.NoError(t, native.Restart(context.Background(), id))
	require.Equal(t, []Command{
		{Name: "systemctl", Args: []string{"enable", "--now", "soda-runner@one.service"}},
		{Name: "systemctl", Args: []string{"disable", "--now", "soda-runner@one.service"}},
		{Name: "systemctl", Args: []string{"enable", "soda-runner@one.service"}},
		{Name: "systemctl", Args: []string{"restart", "soda-runner@one.service"}},
	}, runner.commands)
}

func TestPrepareAccountParentSurvivesRestrictiveUmask(t *testing.T) {
	if os.Getenv("SODA_RUNNER_UMASK_TEST") != "1" {
		cmd := exec.Command(os.Args[0], "-test.run=^TestPrepareAccountParentSurvivesRestrictiveUmask$")
		cmd.Env = append(os.Environ(), "SODA_RUNNER_UMASK_TEST=1")
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "%s", output)
		return
	}
	// Only this isolated test process changes its umask. No real account command runs.
	syscall.Umask(0o077)
	root := t.TempDir()
	called := false
	native := &Native{RootPath: root, Runner: runnerCommandFunc(func(_ context.Context, command Command) (CommandResult, error) {
		called = true
		require.Equal(t, "useradd", command.Name)
		info, err := os.Stat(filepath.Join(root, "umask-probe"))
		require.NoError(t, err)
		require.Equal(t, os.FileMode(0o755), info.Mode().Perm())
		return CommandResult{}, errors.New("fixture refuses real account creation")
	})}
	_, err := native.prepareAccount(context.Background(), "umask-probe")
	require.ErrorContains(t, err, "create runner Linux account")
	require.True(t, called)
	require.Equal(t, 0o077, syscall.Umask(0o077), "do not weaken the helper's process-wide umask")
}

func TestPrepareAccountRemovesLinuxAccountWhenIdentityLookupFails(t *testing.T) {
	root := t.TempDir()
	runner := &recordingCommandRunner{}
	native := &Native{RootPath: root, Runner: runner}

	_, err := native.prepareAccount(context.Background(), "uncreated")
	require.ErrorContains(t, err, "look up new runner account")
	require.Len(t, runner.commands, 2)
	require.Equal(t, "useradd", runner.commands[0].Name)
	require.Equal(t, Command{Name: "userdel", Args: []string{"soda-runner-uncreated"}}, runner.commands[1])
	require.NoDirExists(t, filepath.Join(root, "uncreated"))
}
