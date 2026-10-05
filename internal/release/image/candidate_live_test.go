package image

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/release/deliver"
	"github.com/stretchr/testify/require"
)

func TestCandidateLiveConfigRendersConsoleHandoff(t *testing.T) {
	rev, hash := strings.Repeat("a", 40), strings.Repeat("b", 64)
	p := deliver.Payload{Format: 3, CoreOS: "44.20260817.3.2", ID: "44.20260817.3.2.soda-" + rev[:12], Revision: rev, Architecture: "x86_64", Base: "quay.io/fedora/fedora-coreos@sha256:" + hash, Schema: 10, RepositoryPrefix: "ghcr.io/example/sodaos", PresentationSHA256: hash, HostPackagesSHA256: hash, Images: map[string]deliver.Image{}}
	for i, name := range deliver.Names {
		manifest := fmt.Sprintf("sha256:%064x", 0x100+i)
		config := fmt.Sprintf("sha256:%064x", 0x200+i)
		p.Images[name] = deliver.Image{Config: config, Manifest: manifest, ArchiveSHA256: strings.Repeat("1", 64), Reference: p.RepositoryPrefix + "-" + name + "@" + manifest}
	}
	require.NoError(t, p.Validate())
	raw, err := json.Marshal(p)
	require.NoError(t, err)
	consoleHash := strings.Repeat("c", 64)
	live, err := candidateLiveConfig(raw, []byte(`{"ignition":{"version":"3.5.0"},"storage":{"files":[]}}`), "sha256:"+hash, consoleHash)
	require.NoError(t, err)
	require.Contains(t, string(live), "ExecStart="+candidateInstallerBinary+" disk")
	require.Contains(t, string(live), "Type=simple")
	require.NotContains(t, string(live), "Type=idle")
	for _, forbidden := range []string{"/run/media/iso", "http", "--dest-device"} {
		require.NotContains(t, string(live), forbidden)
	}
	_, err = candidateLiveConfig(raw, []byte(`{"ignition":{"version":"3.5.0"},"passwd":{}}`), "sha256:"+hash, consoleHash)
	require.Error(t, err)
	_, err = candidateLiveConfig(raw, []byte(`{"ignition":{"version":"3.5.0"}}`), "latest", consoleHash)
	require.Error(t, err)
	_, err = candidateLiveConfig([]byte(`{"Format":3`), []byte(`{"ignition":{"version":"3.5.0"}}`), "sha256:"+hash, consoleHash)
	require.Error(t, err)
}
