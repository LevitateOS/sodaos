package hostimage

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/nativebuild"
	"github.com/stretchr/testify/require"
)

func TestControllerAdmissionRefusesBeforeProduction(t *testing.T) {
	for _, mode := range []string{"dirty", "revision", "occupied", "outside", "worktree", "architecture"} {
		t.Run(mode, func(t *testing.T) {
			source := t.TempDir()
			require.NoError(t, os.Mkdir(filepath.Join(source, ".git"), 0700))
			parent := filepath.Join(source, ".artifacts/releases")
			require.NoError(t, os.MkdirAll(parent, 0700))
			out := filepath.Join(parent, "run")
			if mode == "occupied" {
				require.NoError(t, os.Mkdir(out, 0700))
				require.NoError(t, os.WriteFile(filepath.Join(out, "retain"), []byte("later writes"), 0600))
			}
			if mode == "worktree" {
				require.NoError(t, os.Rename(filepath.Join(source, ".git"), filepath.Join(source, "retained-git")))
				require.NoError(t, os.WriteFile(filepath.Join(source, ".git"), []byte("gitdir: elsewhere"), 0600))
			}
			r := Request{Source: source, Out: out, Arch: "x86_64", RepositoryPrefix: "ghcr.io/example/sodaos"}
			if mode == "outside" {
				r.Out = filepath.Join(source, "outside")
			}
			if mode == "architecture" {
				r.Arch = "not-native"
			}
			if mode == "revision" {
				r.Revision = strings.Repeat("b", 40)
			}
			capture := func(_, name string, args ...string) (string, error) {
				require.Equal(t, "git", name)
				switch strings.Join(args, " ") {
				case "rev-parse --show-toplevel":
					return source, nil
				case "status --porcelain --untracked-files=normal":
					if mode == "dirty" {
						return " M source.go", nil
					}
					return "", nil
				case "rev-parse HEAD":
					return strings.Repeat("a", 40), nil
				}
				t.Fatalf("unexpected preflight %v", args)
				return "", nil
			}
			p := &nativebuild.BuildProgress{Now: func() time.Duration { return time.Second }, Stderr: io.Discard}
			_, err := build(r, p, func(string, string, ...string) error { t.Fatal("refused input dispatched production"); return nil }, capture, func(string) (func() error, error) { t.Fatal("refused input opened logs"); return nil, nil })
			require.Error(t, err)
			if mode == "occupied" {
				data, e := os.ReadFile(filepath.Join(out, "retain"))
				require.NoError(t, e)
				require.Equal(t, "later writes", string(data))
			} else {
				require.NoDirExists(t, out)
			}
		})
	}
}
func TestPreparedChecksUseExistingOutputsAndStopOnFailure(t *testing.T) {
	root := t.TempDir()
	var calls []string
	sentinel := errors.New("source test failed")
	p := nativebuild.Production{Source: root, Native: filepath.Join(root, ".artifacts/native/x86_64"), Next: func(string) error { return nil }, Execute: func(_, name string, args ...string) error {
		calls = append(calls, name+" "+strings.Join(args, " "))
		if name == "bun" && strings.Contains(strings.Join(args, " "), "test:forgejo") {
			return sentinel
		}
		return nil
	}}
	require.ErrorIs(t, preparedChecks(p), sentinel)
	require.Equal(t, []string{"go test ./...", "bun run typecheck", "bun run test:frontend:prepared", "bun run test:forgejo:prepared"}, calls)
	target, err := os.Readlink(filepath.Join(root, ".artifacts/forgejo-js"))
	require.NoError(t, err)
	require.Equal(t, filepath.Join(p.Native, "forgejo-js"), target)
	require.NotContains(t, strings.Join(calls, "\n"), "build:forgejo")
}
func TestBuildCommandCaptureEnvironmentAndFailure(t *testing.T) {
	t.Setenv("SODA_RELEASE_NATIVE_OUT", "private-signing-sentinel")
	t.Setenv("SODA_FORGEJO_NATIVE_PAGES", "private-fixture-sentinel")
	t.Setenv("GITHUB_TOKEN", "private-token-sentinel")
	var log bytes.Buffer
	got, err := runBuildCommand(t.Context(), &log, nil, t.TempDir(), "sh", "-c", "printf captured-id; test -z \"$GITHUB_TOKEN$SODA_RELEASE_NATIVE_OUT$SODA_FORGEJO_NATIVE_PAGES\"")
	require.NoError(t, err)
	require.Equal(t, "captured-id", got)
	require.NotContains(t, log.String(), "printf")
	_, err = runBuildCommand(t.Context(), &log, nil, t.TempDir(), "sh", "-c", "exit 7")
	require.Equal(t, 7, nativebuild.BuildExitCode(err))
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err = runBuildCommand(ctx, &log, nil, t.TempDir(), "sh", "-c", "exit 0")
	require.ErrorIs(t, err, context.Canceled)
	require.NotContains(t, log.String(), "private-")
}
