package installer

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/appliancerelease"
	"github.com/levitateos/sodaos/internal/nativebuild"
	"github.com/stretchr/testify/require"
)

func TestCandidateMediaRequiresEveryExactLocalArchive(t *testing.T) {
	root := t.TempDir()
	rev := strings.Repeat("a", 40)
	hash := strings.Repeat("b", 64)
	p := appliancerelease.Payload{Format: 2, CoreOS: "44.20260817.3.2", ID: "44.20260817.3.2.soda-" + rev[:12], Revision: rev, Architecture: architecture(), Base: "quay.io/fedora/fedora-coreos@sha256:" + hash, Schema: 10, RepositoryPrefix: "ghcr.io/example/sodaos", PresentationSHA256: hash, HostPackagesSHA256: hash, Images: map[string]appliancerelease.Image{}}
	images := filepath.Join(root, appliancerelease.ImagesPath)
	require.NoError(t, os.MkdirAll(images, 0755))
	for _, name := range appliancerelease.Names {
		path := filepath.Join(images, name+".oci")
		require.NoError(t, os.WriteFile(path, []byte(name), 0644))
		h, e := nativebuild.HashFile(path)
		require.NoError(t, e)
		p.Images[name] = appliancerelease.Image{Storage: "podman", Config: "sha256:" + hash, Manifest: "sha256:" + hash, ArchiveSHA256: h, Reference: p.RepositoryPrefix + "-" + name + "@sha256:" + hash}
	}
	raw, e := json.Marshal(p)
	require.NoError(t, e)
	path := filepath.Join(root, appliancerelease.Path)
	require.NoError(t, os.WriteFile(path, raw, 0644))
	h, e := nativebuild.HashFile(path)
	require.NoError(t, e)
	console := filepath.Join(root, candidateInstallerBinary)
	require.NoError(t, os.MkdirAll(filepath.Dir(console), 0755))
	require.NoError(t, os.WriteFile(console, []byte("prebuilt fixture"), 0755))
	consoleHash, e := nativebuild.HashFile(console)
	require.NoError(t, e)
	m := mediaIdentity{Format: 2, Architecture: architecture(), Release: p.CoreOS, Revision: rev, InstallerVersion: "coreos-installer 0.26.0", HostManifest: "sha256:" + hash, PayloadSHA256: h, ConsoleSHA256: consoleHash}
	size, e := candidateRequirement(m, root)
	require.NoError(t, e)
	require.Positive(t, size)
	live, e := CandidateLiveConfig(raw, []byte(`{"ignition":{"version":"3.5.0"},"storage":{"files":[]}}`), m.HostManifest, consoleHash)
	require.NoError(t, e)
	require.Contains(t, string(live), "ExecStart="+candidateInstallerBinary+" disk")
	require.NotContains(t, string(live), "/run/media/iso")
	require.NotContains(t, string(live), "http")
	require.NotContains(t, string(live), "--dest-device")
	_, e = CandidateLiveConfig(raw, []byte(`{"ignition":{"version":"3.5.0"},"passwd":{}}`), m.HostManifest, consoleHash)
	require.Error(t, e)
	_, e = CandidateLiveConfig(raw, []byte(`{"ignition":{"version":"3.5.0"}}`), "latest", consoleHash)
	require.Error(t, e)
	for _, mutate := range []func(*mediaIdentity){func(m *mediaIdentity) { m.Format = 1 }, func(m *mediaIdentity) { m.BundleSHA256 = hash }, func(m *mediaIdentity) { m.HostManifest = "latest" }, func(m *mediaIdentity) { m.Revision = strings.Repeat("c", 40) }, func(m *mediaIdentity) { m.PayloadSHA256 = hash }, func(m *mediaIdentity) { m.ConsoleSHA256 = hash }} {
		bad := m
		mutate(&bad)
		_, e = candidateRequirement(bad, root)
		require.Error(t, e)
	}
	for _, name := range appliancerelease.Names {
		path := filepath.Join(images, name+".oci")
		require.NoError(t, os.WriteFile(path, []byte("changed"), 0644))
		_, e = candidateRequirement(m, root)
		require.Error(t, e)
		require.NoError(t, os.WriteFile(path, []byte(name), 0644))
	}
	require.NoError(t, os.Remove(filepath.Join(images, "tailnet.oci")))
	_, e = candidateRequirement(m, root)
	require.Error(t, e)
	require.NoError(t, os.Symlink(filepath.Join(images, "proxy.oci"), filepath.Join(images, "tailnet.oci")))
	_, e = candidateRequirement(m, root)
	require.Error(t, e)
}

func TestCandidateDestinationKeepsPasswordOnlyNativeProvisioning(t *testing.T) {
	template := []byte(`{"ignition":{"version":"3.5.0"},"storage":{"files":[]}}`)
	factory := []byte(`{"network":"soda-projects","bridge":"soda0","subnet":"","tailnet_management":false}`)
	choices := diskInstallChoices{hostname: "soda-tester", passwordHash: "$6$salt$" + strings.Repeat("a", 86), subnet: "10.89.0.0/24"}
	b, e := candidateDestination(template, factory, choices)
	require.NoError(t, e)
	var cfg struct {
		Passwd  struct{ Users []map[string]any }
		Storage struct {
			Files []struct {
				Path     string
				Mode     int
				Contents struct{ Source string }
			}
		}
	}
	require.NoError(t, json.Unmarshal(b, &cfg))
	require.Len(t, cfg.Passwd.Users, 1)
	require.Equal(t, "root", cfg.Passwd.Users[0]["name"])
	require.Equal(t, choices.passwordHash, cfg.Passwd.Users[0]["passwordHash"])
	require.NotContains(t, cfg.Passwd.Users[0], "sshAuthorizedKeys")
	seen := false
	for _, f := range cfg.Storage.Files {
		require.NotEqual(t, "/etc/soda-installer/project-subnet", f.Path)
		if f.Path == "/etc/soda/host.json" {
			seen = true
			require.Equal(t, 0600, f.Mode)
			raw, e := base64.StdEncoding.DecodeString(strings.TrimPrefix(f.Contents.Source, "data:;base64,"))
			require.NoError(t, e)
			var machine map[string]any
			require.NoError(t, json.Unmarshal(raw, &machine))
			require.Equal(t, choices.subnet, machine["subnet"])
			require.NotContains(t, machine, "image")
			require.NotContains(t, machine, "tailnet_image")
		}
	}
	require.True(t, seen)
	require.NotContains(t, string(b), "rpm-ostree install")
	require.NotContains(t, string(b), "soda-install continue")
	_, e = candidateDestination([]byte(`{"ignition":{"version":"3.5.0"},"storage":{"files":[{"path":"/etc/soda/host.json"}]}}`), factory, choices)
	require.Error(t, e)
	_, e = candidateDestination(template, []byte(`{"subnet":"10.0.0.0/24"}`), choices)
	require.Error(t, e)
}
