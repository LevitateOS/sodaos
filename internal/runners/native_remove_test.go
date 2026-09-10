package runners

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRemoveReportsPartialOutcomeAndPreservesOtherRunners(t *testing.T) {
	for _, failure := range []string{"none", "stop", "account", "state"} {
		t.Run(failure, func(t *testing.T) {
			native, _, prepared := runnerFixture(t)
			require.NoError(t, native.recordRunner(prepared.account, forgejoRequest()))
			descriptor, err := os.ReadFile(native.descriptorPath("one"))
			require.NoError(t, err)
			require.NoError(t, os.WriteFile(filepath.Join(prepared.state, "forgejo-token"), []byte("synthetic-private-token"), 0600))
			require.NoError(t, os.WriteFile(filepath.Join(prepared.state, "work-data"), []byte("uncommitted fixture work"), 0600))
			other := filepath.Join(native.rootPath(), "other", "state")
			require.NoError(t, os.MkdirAll(other, 0700))
			require.NoError(t, os.WriteFile(filepath.Join(other, "work-data"), []byte("unrelated fixture work"), 0600))
			var commands []Command
			native.Runner = runnerCommandFunc(func(_ context.Context, command Command) (CommandResult, error) {
				commands = append(commands, command)
				switch command.Name {
				case "systemctl":
					require.Equal(t, []string{"disable", "--now", "soda-runner@one.service"}, command.Args)
					if failure == "stop" {
						return CommandResult{}, errors.New("synthetic-private-token")
					}
				case "userdel":
					require.Equal(t, []string{prepared.account}, command.Args)
					if failure == "account" {
						return CommandResult{}, errors.New("synthetic-private-token")
					}
					if failure == "state" {
						// A real filesystem failure even as root, without mounts, privilege changes
						// or an injected production RemoveAll hook. Preserve the tree, then make
						// its parent unresolvable only after simulated account deletion succeeds.
						root := native.rootPath()
						saved := root + "-preserved"
						require.NoError(t, os.Rename(root, saved))
						require.NoError(t, os.Symlink(filepath.Base(root), root))
						t.Cleanup(func() { require.NoError(t, os.Remove(root)); require.NoError(t, os.Rename(saved, root)) })
					}
				default:
					t.Fatalf("unexpected native command: %s", command.Name)
				}
				return CommandResult{}, nil
			})
			err = native.Remove(t.Context(), "one")
			expected := []Command{{Name: "systemctl", Args: []string{"disable", "--now", "soda-runner@one.service"}}}
			if failure != "stop" {
				expected = append(expected, Command{Name: "userdel", Args: []string{prepared.account}})
			}
			require.Equal(t, expected, commands)
			switch failure {
			case "none":
				require.NoError(t, err)
				require.NoDirExists(t, filepath.Dir(prepared.state))
			case "stop":
				require.ErrorContains(t, err, "listener stop is unconfirmed; account and state removal were not attempted")
			case "account":
				require.ErrorContains(t, err, "account removal is unconfirmed; local state was not removed")
			case "state":
				require.ErrorContains(t, err, "account was removed, but local state was not fully removed")
			}
			if err != nil {
				require.NotContains(t, err.Error(), "synthetic-private-token")
			}
			root := native.rootPath()
			if failure == "state" {
				root += "-preserved"
			}
			if failure != "none" {
				after, err := os.ReadFile(filepath.Join(root, "one", "descriptor.json"))
				require.NoError(t, err)
				require.Equal(t, descriptor, after)
				for file, contents := range map[string]string{"forgejo-token": "synthetic-private-token", "work-data": "uncommitted fixture work"} {
					after, err := os.ReadFile(filepath.Join(root, "one", "state", file))
					require.NoError(t, err)
					require.Equal(t, contents, string(after))
				}
			}
			contents, err := os.ReadFile(filepath.Join(root, "other", "state", "work-data"))
			require.NoError(t, err)
			require.Equal(t, "unrelated fixture work", string(contents))
		})
	}
}
