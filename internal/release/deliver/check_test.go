package deliver

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/release/build"
	"github.com/levitateos/sodaos/internal/testoci"
	"github.com/stretchr/testify/require"
)

func checkCandidateContent() map[string][]byte {
	return map[string][]byte{
		"dashboard:/usr/local/bin/soda-dashboard":                     []byte("Soda service"),
		"forgejo:/usr/local/bin/gitea":                                []byte("patched Forgejo binary"),
		"extension:/usr/local/bin/gitea":                              []byte("patched Forgejo binary"),
		"extension:/usr/share/soda/extension/extension.json":          []byte("extension manifest"),
		"extension:/usr/share/soda/extension/backend":                 []byte("extension backend"),
		"extension:/usr/share/soda/extension/run":                     []byte("extension runner"),
		"extension:/usr/share/soda/extension/assets/entry.js":         []byte("extension browser asset"),
		"extension:/usr/share/soda/extension/assets/entry.css":        []byte("extension browser style"),
		"host:/usr/share/containers/systemd/forgejo.container":        []byte("host unit"),
		"host:/usr/share/containers/systemd/soda-dashboard.container": []byte("Soda service unit"),
		"host:/usr/lib/systemd/system/soda-extension-install.service": []byte("extension install unit"),
	}
}

func writeCheckCandidate(t *testing.T) (string, Payload, Candidate) {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(dir, "images"), 0o700))
	p := fixture()
	content := checkCandidateContent()
	for _, name := range Names {
		files := map[string][]byte{"fixture.txt": []byte(name)}
		if name == "dashboard" || name == "forgejo" || name == "extension" {
			files = map[string][]byte{}
			for key, body := range content {
				if strings.HasPrefix(key, name+":") {
					files[strings.TrimPrefix(key, name+":")] = body
				}
			}
		}
		archive := filepath.Join(dir, "images", name+".oci")
		im := testoci.ArchiveFiles(t, archive, "amd64", p.Revision, files)
		p.Images[name] = Image{Reference: p.RepositoryPrefix + "-" + name + "@" + im.Manifest, Config: im.Config, Manifest: im.Manifest, ArchiveSHA256: im.ArchiveSHA256}
	}
	hostFiles := map[string][]byte{}
	for key, body := range content {
		if strings.HasPrefix(key, "host:") {
			hostFiles[strings.TrimPrefix(key, "host:")] = body
		}
	}
	c := Candidate{
		Format: 1, Architecture: p.Architecture,
		ForgejoRevision: strings.Repeat("f", 40), ForgejoSourceSHA256: strings.Repeat("e", 64),
		ForgejoToolchain: build.ForgejoToolchain{CompilerImage: build.ForgejoCompilerImage, APKPackages: []string{"build-base-0.5-r4", "gcc-14.2.0-r6", "musl-dev-1.2.5-r10"}},
		Migration:        "no upgrade qualified", Notes: "synthetic fixture",
		ContentSHA256: map[string]string{},
	}
	for name, body := range content {
		c.ContentSHA256[name] = contentDigest(body)
	}
	inventory, err := json.MarshalIndent(c.ContentSHA256, "", "  ")
	require.NoError(t, err)
	hostFiles["usr/share/soda/host-image/content.json"] = append(inventory, '\n')
	host := testoci.ArchiveFiles(t, filepath.Join(dir, "host.oci"), "amd64", p.Revision, hostFiles)
	c.Host = build.Image{
		Config: host.Config, Manifest: host.Manifest, Architecture: "amd64", Revision: p.Revision,
		Source: "https://github.com/LevitateOS/sodaos", BaseName: p.Base, BaseDigest: "sha256:" + strings.Repeat("b", 64),
	}
	c.HostReference = p.RepositoryPrefix + "-host@" + c.Host.Manifest
	c.HostArchiveSHA256 = host.ArchiveSHA256
	payloadBytes, err := json.MarshalIndent(p, "", "  ")
	require.NoError(t, err)
	payloadBytes = append(payloadBytes, '\n')
	require.NoError(t, os.WriteFile(filepath.Join(dir, "payload.json"), payloadBytes, 0o600))
	c.PayloadSHA256 = strings.TrimPrefix(Hash(payloadBytes), "sha256:")
	candidateBytes, err := json.MarshalIndent(c, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "candidate.json"), append(candidateBytes, '\n'), 0o600))
	require.NoError(t, c.Validate(p, payloadBytes))
	return dir, p, c
}

func TestCheckCandidateAcceptsMatchingIdentitiesAndArchives(t *testing.T) {
	dir, p, c := writeCheckCandidate(t)
	require.NoError(t, CheckCandidate(dir, p.Architecture, p.Revision, c.ForgejoRevision))
}

func TestCheckCandidateRejectsStaleIdentities(t *testing.T) {
	dir, p, c := writeCheckCandidate(t)
	for _, tc := range []struct {
		name, arch, soda, forgejo string
	}{
		{"stale soda revision", p.Architecture, strings.Repeat("b", 40), c.ForgejoRevision},
		{"stale fountain revision", p.Architecture, p.Revision, strings.Repeat("0", 40)},
		{"wrong architecture", "aarch64", p.Revision, c.ForgejoRevision},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Error(t, CheckCandidate(dir, tc.arch, tc.soda, tc.forgejo))
		})
	}
}

func TestCheckCandidateRejectsChangedPayloadBinding(t *testing.T) {
	dir, p, c := writeCheckCandidate(t)
	c.PayloadSHA256 = strings.Repeat("0", 64)
	rebound, err := json.MarshalIndent(c, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "candidate.json"), append(rebound, '\n'), 0o600))
	require.Error(t, CheckCandidate(dir, p.Architecture, p.Revision, c.ForgejoRevision))
}

func TestCheckCandidateRejectsMalformedArchives(t *testing.T) {
	dir, p, c := writeCheckCandidate(t)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "images", "tailnet.oci"), []byte("not an OCI archive\n"), 0o600))
	require.Error(t, CheckCandidate(dir, p.Architecture, p.Revision, c.ForgejoRevision))
}

func TestCheckCandidateRejectsChangedArchives(t *testing.T) {
	dir, p, c := writeCheckCandidate(t)
	archive := filepath.Join(dir, "images", "dashboard.oci")
	data, err := os.ReadFile(archive)
	require.NoError(t, err)
	data[len(data)/2] ^= 0x01
	require.NoError(t, os.WriteFile(archive, data, 0o600))
	require.Error(t, CheckCandidate(dir, p.Architecture, p.Revision, c.ForgejoRevision))
}
