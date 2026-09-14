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
	"github.com/levitateos/sodaos/internal/testoci"
	"github.com/stretchr/testify/require"
)

func TestCandidateSharedLayoutAuthenticatesAllImagesWithoutArchives(t *testing.T) {
	root := t.TempDir()
	rev, hash := strings.Repeat("a", 40), strings.Repeat("b", 64)
	p := appliancerelease.Payload{Format: 3, CoreOS: "44.20260817.3.2", ID: "44.20260817.3.2.soda-" + rev[:12], Revision: rev, Architecture: architecture(), Base: "quay.io/fedora/fedora-coreos@sha256:" + hash, Schema: 10, RepositoryPrefix: "ghcr.io/example/sodaos", PresentationSHA256: hash, HostPackagesSHA256: hash, Images: map[string]appliancerelease.Image{}}
	images := filepath.Join(root, appliancerelease.ImagesPath)
	arch, err := nativebuild.OCIArchitecture(p.Architecture)
	require.NoError(t, err)
	for _, name := range appliancerelease.Names {
		archive := filepath.Join(t.TempDir(), name+".oci")
		im := testoci.Archive(t, archive, arch, rev)
		testoci.Add(t, archive, images)
		p.Images[name] = appliancerelease.Image{Config: im.Config, Manifest: im.Manifest, ArchiveSHA256: im.ArchiveSHA256, Reference: p.RepositoryPrefix + "-" + name + "@" + im.Manifest}
	}
	raw, err := json.Marshal(p)
	require.NoError(t, err)
	path := filepath.Join(root, appliancerelease.Path)
	require.NoError(t, os.WriteFile(path, raw, 0644))
	payloadHash, err := nativebuild.HashFile(path)
	require.NoError(t, err)
	console := filepath.Join(root, candidateInstallerBinary)
	require.NoError(t, os.MkdirAll(filepath.Dir(console), 0755))
	require.NoError(t, os.WriteFile(console, []byte("prebuilt fixture"), 0755))
	consoleHash, err := nativebuild.HashFile(console)
	require.NoError(t, err)
	m := mediaIdentity{Format: 2, Architecture: p.Architecture, Release: p.CoreOS, Revision: rev, InstallerVersion: "coreos-installer 0.26.0", HostManifest: "sha256:" + hash, PayloadSHA256: payloadHash, ConsoleSHA256: consoleHash}
	_, uniqueBytes, err := appliancerelease.VerifyContent(p, images)
	require.NoError(t, err)
	size, err := candidateRequirement(m, root)
	require.NoError(t, err)
	require.Equal(t, uniqueBytes, size)
	live, err := CandidateLiveConfig(raw, []byte(`{"ignition":{"version":"3.5.0"},"storage":{"files":[]}}`), m.HostManifest, consoleHash)
	require.NoError(t, err)
	require.Contains(t, string(live), "ExecStart="+candidateInstallerBinary+" disk")
	for _, forbidden := range []string{"/run/media/iso", "http", "--dest-device"} {
		require.NotContains(t, string(live), forbidden)
	}
	for _, mutate := range []func(*mediaIdentity){func(m *mediaIdentity) { m.Format = 1 }, func(m *mediaIdentity) { m.BundleSHA256 = hash }, func(m *mediaIdentity) { m.HostManifest = "latest" }, func(m *mediaIdentity) { m.Revision = strings.Repeat("c", 40) }, func(m *mediaIdentity) { m.PayloadSHA256 = hash }, func(m *mediaIdentity) { m.ConsoleSHA256 = hash }} {
		bad := m
		mutate(&bad)
		_, err = candidateRequirement(bad, root)
		require.Error(t, err)
	}
	_, err = CandidateLiveConfig(raw, []byte(`{"ignition":{"version":"3.5.0"},"passwd":{}}`), m.HostManifest, consoleHash)
	require.Error(t, err)
	_, err = CandidateLiveConfig(raw, []byte(`{"ignition":{"version":"3.5.0"}}`), "latest", consoleHash)
	require.Error(t, err)
	require.NoError(t, os.Remove(filepath.Join(images, "blobs/sha256", strings.TrimPrefix(p.Images["tailnet"].Config, "sha256:"))))
	_, err = candidateRequirement(m, root)
	require.Error(t, err)
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
