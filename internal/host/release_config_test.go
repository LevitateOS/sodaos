package host

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/release/deliver"
	"github.com/stretchr/testify/require"
)

func TestImageDefaultsAreImmutableAndNeverRewriteMachineConfig(t *testing.T) {
	p := deliver.Payload{Format: 3, ID: "44.20260817.3.2.soda-" + strings.Repeat("a", 12), Revision: strings.Repeat("a", 40), Architecture: "x86_64", CoreOS: "44.20260817.3.2", Base: "quay.io/fedora/fedora-coreos@sha256:" + strings.Repeat("b", 64), RepositoryPrefix: "ghcr.io/example/sodaos", Schema: 10, PresentationSHA256: strings.Repeat("c", 64), HostPackagesSHA256: strings.Repeat("d", 64), Images: map[string]deliver.Image{}}
	if runtime.GOARCH == "arm64" {
		p.Architecture = "aarch64"
	}
	for _, n := range deliver.Names {
		p.Images[n] = deliver.Image{Reference: p.RepositoryPrefix + "-" + n + "@sha256:" + strings.Repeat("e", 64), Manifest: "sha256:" + strings.Repeat("e", 64), Config: "sha256:" + strings.Repeat("f", 64), ArchiveSHA256: strings.Repeat("1", 64)}
	}
	root := t.TempDir()
	release := filepath.Join(root, "release.json")
	config := filepath.Join(root, "host.json")
	data, err := json.Marshal(p)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(release, data, 0o644))
	for _, tc := range []struct {
		body           string
		valid, managed bool
	}{
		{`{"network":"soda-projects","bridge":"soda0","subnet":"10.89.0.0/24"}`, true, false},
		{`{"network":"soda-projects","bridge":"soda0","subnet":"10.89.0.0/24","tailnet_management":true}`, true, true},
		{`{"network":"soda-projects","bridge":"soda0","subnet":"10.89.0.0/24","image":"localhost/soda-project-os:dev"}`, false, false},
		{`{"network":"soda-projects","bridge":"soda0","subnet":"10.89.0.0/24","tailnet_management":true,"tailnet_image":"sha256:` + strings.Repeat("0", 64) + `"}`, false, false},
	} {
		require.NoError(t, os.WriteFile(config, []byte(tc.body), 0o600))
		c, err := loadConfig(config, release)
		if tc.valid {
			require.NoError(t, err)
			require.Equal(t, p.Images["project-os"].Config, c.Image)
			require.Equal(t, tc.managed, c.TailnetManagement)
			if tc.managed {
				require.Equal(t, p.Images["tailnet"].Config, c.TailnetImage)
			} else {
				require.Empty(t, c.TailnetImage)
			}
		} else {
			require.ErrorContains(t, err, "explicit migration")
		}
		after, e := os.ReadFile(config)
		require.NoError(t, e)
		require.Equal(t, tc.body, string(after))
	}
	_, err = loadConfig(config, filepath.Join(root, "missing.json"))
	require.ErrorContains(t, err, "defaults unavailable")
}
