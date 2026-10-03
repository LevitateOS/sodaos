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

func prepareCandidateContent() map[string][]byte {
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

func stagePrepareCandidate(t *testing.T, rootPath string) (Payload, Candidate) {
	t.Helper()
	require.NoError(t, os.Mkdir(filepath.Join(rootPath, "images"), 0o700))
	p := fixture()
	content := prepareCandidateContent()
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
		archive := filepath.Join(rootPath, "images", name+".oci")
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
		ForgejoRevision:  strings.Repeat("2", 40),
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
	hostArchive := filepath.Join(rootPath, "host.oci")
	host := testoci.ArchiveFiles(t, hostArchive, "amd64", p.Revision, hostFiles)
	c.Host = build.Image{Config: host.Config, Manifest: host.Manifest, Architecture: "amd64", Revision: p.Revision, Source: "https://github.com/LevitateOS/sodaos", BaseName: p.Base, BaseDigest: "sha256:" + strings.Repeat("b", 64)}
	c.HostReference = p.RepositoryPrefix + "-host@" + c.Host.Manifest
	c.HostArchiveSHA256 = host.ArchiveSHA256
	return p, c
}

func writePrepareMetadata(t *testing.T, rootPath string, p Payload, c Candidate) {
	t.Helper()
	forgejoSource := []byte("synthetic archived fork source")
	packages := []byte("cockpit-ostree 1:225-1.fc44.noarch\n")
	presentation := []byte("{\"templates/custom/header.tmpl\":\"synthetic\"}\n")
	p.PresentationSHA256 = strings.TrimPrefix(Hash(presentation), "sha256:")
	p.HostPackagesSHA256 = strings.TrimPrefix(Hash(packages), "sha256:")
	c.ForgejoSourceSHA256 = strings.TrimPrefix(Hash(forgejoSource), "sha256:")
	pb, err := marshal(p)
	require.NoError(t, err)
	c.PayloadSHA256 = strings.TrimPrefix(Hash(pb), "sha256:")
	cb, err := marshal(c)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(rootPath, "payload.json"), pb, 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(rootPath, "candidate.json"), cb, 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(rootPath, "source.tar"), []byte("synthetic archived soda source"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(rootPath, "forgejo-source.tar"), forgejoSource, 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(rootPath, "app-inputs.json"), []byte("[]"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(rootPath, "packages.txt"), packages, 0o600))
	require.NoError(t, os.Mkdir(filepath.Join(rootPath, "forgejo-context"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(rootPath, "forgejo-context/presentation.json"), presentation, 0o600))
}

func TestPrepareAcceptsAlignedProducerOutput(t *testing.T) {
	tr := testTrust(t)
	rootPath := privateDir(t)
	candidate := filepath.Join(rootPath, "candidate")
	require.NoError(t, os.Mkdir(candidate, 0o700))
	p, c := stagePrepareCandidate(t, candidate)
	writePrepareMetadata(t, candidate, p, c)
	require.NoError(t, decode(mustRead(t, filepath.Join(candidate, "payload.json")), &p))
	require.NoError(t, decode(mustRead(t, filepath.Join(candidate, "candidate.json")), &c))
	mediaPath := filepath.Join(rootPath, "media.json")
	require.NoError(t, os.WriteFile(mediaPath, testMediaBytes(t, p, c), 0o600))
	q := Qualification{Serial: 1, Class: "normal", Scope: "local-only", Notes: "synthetic aligned-candidate evidence", Evidence: map[string]string{"qualification.json": Hash([]byte("synthetic"))}}
	digest, err := Prepare(tr, candidate, mediaPath, q, filepath.Join(rootPath, "prepared"))
	require.NoError(t, err)
	require.True(t, Digest(digest))
	stored, err := documentDigest(filepath.Join(rootPath, "prepared"))
	require.NoError(t, err)
	require.Equal(t, digest, stored)

	unaligned := filepath.Join(rootPath, "unaligned")
	require.NoError(t, os.Mkdir(unaligned, 0o700))
	p, c = stagePrepareCandidate(t, unaligned)
	writePrepareMetadata(t, unaligned, p, c)
	require.NoError(t, os.Remove(filepath.Join(unaligned, "source.tar")))
	_, err = Prepare(tr, unaligned, mediaPath, q, filepath.Join(rootPath, "prepared-unaligned"))
	require.Error(t, err, "candidate archives outside the candidate directory are not finalizer input")
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	return raw
}

func contentDigest(body []byte) string {
	return strings.TrimPrefix(Hash(body), "sha256:")
}

func TestVerifyCandidateImagesChecksRecordedImageMembers(t *testing.T) {
	rootPath := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(rootPath, "images"), 0o700))
	p := fixture()
	content := map[string][]byte{
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
		archive := filepath.Join(rootPath, "images", name+".oci")
		im := testoci.ArchiveFiles(t, archive, "amd64", p.Revision, files)
		p.Images[name] = Image{Reference: p.RepositoryPrefix + "-" + name + "@" + im.Manifest, Config: im.Config, Manifest: im.Manifest, ArchiveSHA256: im.ArchiveSHA256}
	}
	hostFiles := map[string][]byte{}
	for key, body := range content {
		if strings.HasPrefix(key, "host:") {
			hostFiles[strings.TrimPrefix(key, "host:")] = body
		}
	}
	c := Candidate{ContentSHA256: map[string]string{}}
	for name, body := range content {
		c.ContentSHA256[name] = contentDigest(body)
	}
	inventory, err := json.MarshalIndent(c.ContentSHA256, "", "  ")
	require.NoError(t, err)
	hostFiles["usr/share/soda/host-image/content.json"] = append(inventory, '\n')
	hostArchive := filepath.Join(rootPath, "host.oci")
	host := testoci.ArchiveFiles(t, hostArchive, "amd64", p.Revision, hostFiles)
	c.Host = build.Image{Config: host.Config, Manifest: host.Manifest}
	c.HostArchiveSHA256 = host.ArchiveSHA256
	root, err := os.OpenRoot(rootPath)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, root.Close()) })
	_, err = VerifyCandidateImages(root, rootPath, p, c)
	require.NoError(t, err)

	service := "dashboard:/usr/local/bin/soda-dashboard"
	serviceDigest := c.ContentSHA256[service]
	c.ContentSHA256[service] = strings.Repeat("0", 64)
	_, err = VerifyCandidateImages(root, rootPath, p, c)
	require.Error(t, err, "the Soda service archive must match its recorded executable")
	c.ContentSHA256[service] = serviceDigest

	asset := "extension:/usr/share/soda/extension/assets/entry.css"
	assetDigest := c.ContentSHA256[asset]
	delete(c.ContentSHA256, asset)
	require.True(t, ValidCandidateContent(c.ContentSHA256), "one remaining asset still passes the shape check")
	_, err = VerifyCandidateImages(root, rootPath, p, c)
	require.Error(t, err, "the embedded host inventory must reject an omitted asset")
	c.ContentSHA256[asset] = assetDigest

	c.ContentSHA256["extension:/usr/share/soda/extension/backend"] = strings.Repeat("0", 64)
	_, err = VerifyCandidateImages(root, rootPath, p, c)
	require.Error(t, err)
	c.ContentSHA256["extension:/usr/share/soda/extension/backend"] = contentDigest(content["extension:/usr/share/soda/extension/backend"])
	c.ContentSHA256["extension:/usr/local/bin/gitea"] = strings.Repeat("0", 64)
	_, err = VerifyCandidateImages(root, rootPath, p, c)
	require.Error(t, err)
	c.ContentSHA256["extension:/usr/local/bin/gitea"] = contentDigest(content["extension:/usr/local/bin/gitea"])
	c.ContentSHA256["host:/usr/lib/systemd/system/soda-extension-install.service"] = strings.Repeat("0", 64)
	_, err = VerifyCandidateImages(root, rootPath, p, c)
	require.Error(t, err)
}
