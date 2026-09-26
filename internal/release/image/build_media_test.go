package image

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/release/build"
	"github.com/stretchr/testify/require"
)

func TestMediaToolAdmissionKeepsNativeVersionPlatformAndReadOnlyScope(t *testing.T) {
	for _, platform := range []string{"linux/amd64", "linux/arm64"} {
		t.Run(platform, func(t *testing.T) {
			source := sourceRoot(t)
			evidence := t.TempDir()
			var calls []string
			p := build.Production{Arch: "x86_64", Execute: func(dir, name string, args ...string) error {
				require.Equal(t, source, dir)
				require.Equal(t, "podman", name)
				calls = append(calls, strings.Join(args, " "))
				return nil
			}, Capture: func(dir, name string, args ...string) (string, error) {
				require.Equal(t, source, dir)
				require.Equal(t, "podman", name)
				calls = append(calls, strings.Join(args, " "))
				if args[1] == "image" {
					return platform, nil
				}
				joined := strings.Join(args, " ")
				for _, flag := range []string{"--pull=never", "--cidfile", "--network=none", "--read-only", "--cap-drop=all", "--security-opt=no-new-privileges"} {
					require.Contains(t, joined, flag)
				}
				require.NotContains(t, joined, "--rm")
				return "Butane v2.27.0\n", nil
			}}
			tools, err := admitMediaTools(source, evidence, p)
			if platform == "linux/amd64" {
				require.NoError(t, err)
				require.Len(t, calls, 3)
				require.Equal(t, "quay.io/coreos/butane:latest", tools.Butane)
				recorded, err := os.ReadFile(filepath.Join(evidence, "butane-version.txt"))
				require.NoError(t, err)
				require.Equal(t, "Butane v2.27.0\n", string(recorded))
			} else {
				require.ErrorContains(t, err, "platform mismatch")
				require.Len(t, calls, 2)
			}
		})
	}
	p := build.Production{Arch: "x86_64", Capture: func(dir, name string, args ...string) (string, error) {
		if len(args) > 1 && args[1] == "image" {
			return "linux/amd64", nil
		}
		return "not-a-butane-version\n", nil
	}, Execute: func(dir, name string, args ...string) error { return nil }}
	_, err := admitMediaTools(sourceRoot(t), t.TempDir(), p)
	require.ErrorContains(t, err, "unrecognized Butane version")
}
