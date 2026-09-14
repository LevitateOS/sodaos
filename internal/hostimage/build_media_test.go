package hostimage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/nativebuild"
	"github.com/stretchr/testify/require"
)

func TestMediaToolAdmissionKeepsNativeVersionPlatformAndReadOnlyScope(t *testing.T) {
	for _, platform := range []string{"linux/amd64", "linux/arm64"} {
		t.Run(platform, func(t *testing.T) {
			source := sourceRoot(t)
			var calls []string
			p := nativebuild.Production{Execute: func(dir, name string, args ...string) error {
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
				return "Butane v2.27.0", nil
			}}
			_, err := admitMediaTools(source, t.TempDir(), "x86_64", p)
			if platform == "linux/amd64" {
				require.NoError(t, err)
				require.Len(t, calls, 3)
			} else {
				require.ErrorContains(t, err, "platform mismatch")
				require.Len(t, calls, 2)
			}
		})
	}
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "appliance/locks"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "appliance/locks/installer-tools.json"), []byte(`{"Butane":"quay.io/coreos/butane:latest","Architecture":"x86_64","Version":"Butane v2.27.0"}`), 0644))
	_, err := admitMediaTools(root, t.TempDir(), "x86_64", nativebuild.Production{})
	require.ErrorContains(t, err, "unreviewed native media tool")
}
