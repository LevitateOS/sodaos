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

func contentDigest(body []byte) string {
	return strings.TrimPrefix(Hash(body), "sha256:")
}

func TestVerifyCandidateImagesChecksRecordedImageMembers(t *testing.T) {
	rootPath := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(rootPath, "images"), 0o700))
	p := fixture()
	content := map[string][]byte{
		"forgejo:/usr/local/bin/gitea":                                []byte("patched Forgejo binary"),
		"extension:/usr/local/bin/gitea":                              []byte("patched Forgejo binary"),
		"extension:/usr/share/soda/extension/extension.json":          []byte("extension manifest"),
		"extension:/usr/share/soda/extension/backend":                 []byte("extension backend"),
		"extension:/usr/share/soda/extension/run":                     []byte("extension runner"),
		"extension:/usr/share/soda/extension/assets/entry.js":         []byte("extension browser asset"),
		"extension:/usr/share/soda/extension/assets/entry.css":        []byte("extension browser style"),
		"host:/usr/share/containers/systemd/forgejo.container":        []byte("host unit"),
		"host:/usr/lib/systemd/system/soda-extension-install.service": []byte("extension install unit"),
	}
	for _, name := range Names {
		files := map[string][]byte{"fixture.txt": []byte(name)}
		if name == "forgejo" || name == "extension" {
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
