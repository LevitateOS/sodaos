package runners

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// Only native commands are doubled; descriptors and state use the real owned filesystem.
type runnerCommandFunc func(context.Context, Command) (CommandResult, error)

func (f runnerCommandFunc) Run(ctx context.Context, command Command) (CommandResult, error) {
	return f(ctx, command)
}

func TestListRefusesIncompleteNativeObservations(t *testing.T) {
	const complete = "LoadState=loaded\nActiveState=active\nSubState=running\nUnitFileState=enabled\n"
	type observation struct {
		name, service, version string
		failure                string
	}
	cases := []observation{
		{"complete", complete, "forgejo-runner fixture\n", ""},
		{"service failure", complete, "fixture", "systemctl"},
		{"version failure", complete, "fixture", "forgejo-runner"},
		{"empty version", complete, " \n", ""},
		{"empty service", "", "fixture", ""},
		{"duplicate property", complete + "ActiveState=inactive\n", "fixture", ""},
	}
	for _, field := range []string{"LoadState=loaded", "ActiveState=active", "SubState=running", "UnitFileState=enabled"} {
		key, _, _ := strings.Cut(field, "=")
		cases = append(cases,
			observation{"missing " + key, strings.Replace(complete, field+"\n", "", 1), "fixture", ""},
			observation{"blank " + key, strings.Replace(complete, field, key+"= ", 1), "fixture", ""},
		)
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			native, _, prepared := runnerFixture(t)
			require.NoError(t, native.recordRunner(prepared.account, forgejoRequest()))
			before, err := os.ReadFile(native.descriptorPath("one"))
			require.NoError(t, err)
			native.Runner = runnerCommandFunc(func(_ context.Context, command Command) (CommandResult, error) {
				if command.Name == tc.failure {
					return CommandResult{}, errors.New("observation unavailable")
				}
				switch command.Name {
				case "systemctl":
					require.Equal(t, []string{"show", "--no-pager", "--property=LoadState,ActiveState,SubState,UnitFileState", "soda-runner@one.service"}, command.Args)
					return CommandResult{Stdout: tc.service}, nil
				case "forgejo-runner":
					require.Equal(t, []string{"--version"}, command.Args)
					return CommandResult{Stdout: tc.version}, nil
				default:
					t.Fatalf("unexpected mutation: %s", command.Name)
					return CommandResult{}, errors.New("unexpected mutation")
				}
			})
			views, err := native.List(t.Context())
			if tc.name == "complete" {
				require.NoError(t, err)
				require.Len(t, views, 1)
				require.Equal(t, "forgejo-runner fixture", views[0].Version)
				require.Equal(t, ServiceState{Load: "loaded", Active: "active", Sub: "running", Enabled: "enabled"}, views[0].Service)
			} else {
				require.Error(t, err)
				require.Nil(t, views, "unavailable is not an empty or partial inventory")
			}
			after, err := os.ReadFile(native.descriptorPath("one"))
			require.NoError(t, err)
			require.Equal(t, before, after)
		})
	}
}

func TestInvalidDescriptorMakesWholeInventoryUnavailableWithoutRewrites(t *testing.T) {
	const descriptor = `{"id":"one","provider":"forgejo","registration_url":"http://old-internal:3000","account":"soda-runner-one","architecture":"x86-64"}`
	for _, bad := range []string{"missing", "directory", "malformed", "unknown field", "duplicate field", "trailing object", "wrong account", "wrong id", "unsupported provider"} {
		t.Run(bad, func(t *testing.T) {
			native, _, prepared := runnerFixture(t)
			require.NoError(t, os.WriteFile(filepath.Join(prepared.state, "work-data"), []byte("keep later writes"), 0600))
			bytes := descriptor
			switch bad {
			case "missing":
			case "directory":
				require.NoError(t, os.Mkdir(native.descriptorPath("one"), 0700))
			case "malformed":
				bytes = "{"
			case "unknown field":
				bytes = strings.Replace(descriptor, `"id":`, `"labels":"old", "id":`, 1)
			case "duplicate field":
				bytes = strings.Replace(descriptor, `"id":`, `"id":"one", "id":`, 1)
			case "trailing object":
				bytes += " {}"
			case "wrong account":
				bytes = strings.Replace(descriptor, "soda-runner-one", "root", 1)
			case "wrong id":
				bytes = strings.Replace(descriptor, `"id":"one"`, `"id":"other"`, 1)
			case "unsupported provider":
				bytes = strings.Replace(descriptor, "forgejo", "github", 1)
			}
			if bad != "missing" && bad != "directory" {
				require.NoError(t, os.WriteFile(native.descriptorPath("one"), []byte(bytes), 0644))
			}
			commands := &recordingCommandRunner{}
			native.Runner = commands
			views, err := native.List(t.Context())
			require.Error(t, err)
			require.Nil(t, views)
			require.Empty(t, commands.commands)
			for _, action := range []func(context.Context, string) error{native.Start, native.Stop, native.Restart, native.Remove} {
				require.Error(t, action(t.Context(), "one"))
			}
			_, err = native.Launch("one")
			require.Error(t, err)
			require.Empty(t, commands.commands)
			if bad != "missing" && bad != "directory" {
				after, err := os.ReadFile(native.descriptorPath("one"))
				require.NoError(t, err)
				require.Equal(t, bytes, string(after))
			}
			work, err := os.ReadFile(filepath.Join(prepared.state, "work-data"))
			require.NoError(t, err)
			require.Equal(t, "keep later writes", string(work))
		})
	}
}

func TestListDoesNotPublishEarlierRowsWhenAnotherRunnerIsUnreadable(t *testing.T) {
	native, _, prepared := runnerFixture(t)
	require.NoError(t, native.recordRunner(prepared.account, forgejoRequest()))
	// The first sorted row is fully readable; the next is a retained partial
	// directory without a descriptor, as can remain after failed creation.
	require.NoError(t, os.MkdirAll(native.statePath("two"), 0700))
	views, err := native.List(t.Context())
	require.Error(t, err)
	require.Nil(t, views)
	coordinator := Coordinator{Authorizer: fakeAuthorizer{}, Local: native}
	result, err := coordinator.Execute(t.Context(), testAdministrator, "list", strings.NewReader(`{}`))
	require.Error(t, err)
	require.Equal(t, ListResponse{}, result)
	operations := Operations{Local: native, Lifecycle: native}
	result, err = operations.Execute(t.Context(), "list", strings.NewReader(`{}`))
	require.Error(t, err)
	require.Nil(t, result)
}

func TestLegacyDescriptorReadsDoNotRewriteStateOrCredentials(t *testing.T) {
	native, _, prepared := runnerFixture(t)
	// Historical descriptor: no labels field; deliberately retain original formatting/origin.
	descriptor := []byte("{\n\"id\":\"one\",\"provider\":\"forgejo\",\"registration_url\":\"http://old-internal:3000\",\"account\":\"soda-runner-one\",\"architecture\":\"x86-64\"\n}\n")
	require.NoError(t, os.WriteFile(native.descriptorPath("one"), descriptor, 0644))
	for _, file := range []string{"forgejo-token", "forgejo-runner.yml", "work-data"} {
		require.NoError(t, os.WriteFile(filepath.Join(prepared.state, file), []byte("synthetic preserved "+file), 0600))
	}
	views, err := native.List(t.Context())
	require.NoError(t, err)
	require.Len(t, views, 1)
	require.Equal(t, "http://old-internal:3000", views[0].RegistrationURL)
	_, err = native.Launch("one")
	require.NoError(t, err)
	after, err := os.ReadFile(native.descriptorPath("one"))
	require.NoError(t, err)
	require.Equal(t, descriptor, after)
	for _, file := range []string{"forgejo-token", "forgejo-runner.yml", "work-data"} {
		path := filepath.Join(prepared.state, file)
		contents, err := os.ReadFile(path)
		require.NoError(t, err)
		require.Equal(t, "synthetic preserved "+file, string(contents))
		info, err := os.Stat(path)
		require.NoError(t, err)
		require.Equal(t, os.FileMode(0600), info.Mode().Perm())
	}
}
