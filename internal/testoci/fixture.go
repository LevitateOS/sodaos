// Package testoci supplies inert OCI fixtures for payload, staging and installer
// tests. No fixture layer is executed and no native engine is required.
package testoci

import (
	"archive/tar"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type Image struct{ Config, Manifest, ArchiveSHA256 string }

func sum(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }

func Archive(t testing.TB, path, arch, revision string) Image {
	t.Helper()
	blobs := map[string][]byte{}
	desc := func(data []byte, media string) map[string]any {
		s := sum(data)
		blobs["blobs/sha256/"+s] = data
		return map[string]any{"mediaType": media, "digest": "sha256:" + s, "size": len(data)}
	}
	var layer bytes.Buffer
	tw := tar.NewWriter(&layer)
	body := []byte(strings.Repeat("shared inert fixture content\n", 100))
	require.NoError(t, tw.WriteHeader(&tar.Header{Name: "fixture.txt", Mode: 0644, Size: int64(len(body))}))
	_, err := tw.Write(body)
	require.NoError(t, err)
	require.NoError(t, tw.Close())
	ld := desc(layer.Bytes(), "application/vnd.oci.image.layer.v1.tar")
	cfg, err := json.Marshal(map[string]any{"os": "linux", "architecture": arch, "rootfs": map[string]any{"type": "layers", "diff_ids": []any{ld["digest"]}}, "config": map[string]any{"Labels": map[string]string{"org.opencontainers.image.revision": revision, "org.opencontainers.image.source": "https://github.com/LevitateOS/sodaos", "org.opencontainers.image.base.name": "synthetic-base", "org.opencontainers.image.base.digest": "sha256:" + strings.Repeat("b", 64), "io.soda.fixture": filepath.Base(path)}}})
	require.NoError(t, err)
	cd := desc(cfg, "application/vnd.oci.image.config.v1+json")
	manifest, err := json.Marshal(map[string]any{"schemaVersion": 2, "mediaType": "application/vnd.oci.image.manifest.v1+json", "config": cd, "layers": []any{ld}})
	require.NoError(t, err)
	md := desc(manifest, "application/vnd.oci.image.manifest.v1+json")
	md["annotations"] = map[string]string{"org.opencontainers.image.ref.name": cd["digest"].(string)}
	blobs["index.json"], err = json.Marshal(map[string]any{"schemaVersion": 2, "manifests": []any{md}})
	require.NoError(t, err)
	blobs["oci-layout"] = []byte(`{"imageLayoutVersion":"1.0.0"}`)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0755))
	f, err := os.Create(path)
	require.NoError(t, err)
	tw = tar.NewWriter(f)
	for _, name := range []string{"blobs", "blobs/sha256"} {
		require.NoError(t, tw.WriteHeader(&tar.Header{Name: name, Typeflag: tar.TypeDir, Mode: 0755, Uid: os.Getuid(), Gid: os.Getgid()}))
	}
	for name, b := range blobs {
		require.NoError(t, tw.WriteHeader(&tar.Header{Name: name, Mode: 0644, Uid: os.Getuid(), Gid: os.Getgid(), Size: int64(len(b))}))
		_, err = tw.Write(b)
		require.NoError(t, err)
	}
	require.NoError(t, tw.Close())
	require.NoError(t, f.Close())
	b, err := os.ReadFile(path)
	require.NoError(t, err)
	return Image{cd["digest"].(string), md["digest"].(string), sum(b)}
}

// Add models only the standard directory result of a Skopeo copy for tiny trusted
// fixtures. Real Skopeo behavior is checked separately against retained exports.
func Add(t testing.TB, archive, layout string) {
	t.Helper()
	f, err := os.Open(archive)
	require.NoError(t, err)
	defer f.Close()
	var descriptors []json.RawMessage
	indexPath := filepath.Join(layout, "index.json")
	if b, err := os.ReadFile(indexPath); err == nil {
		var index struct {
			Manifests []json.RawMessage `json:"manifests"`
		}
		require.NoError(t, json.Unmarshal(b, &index))
		descriptors = index.Manifests
	} else {
		require.True(t, os.IsNotExist(err))
	}
	tr := tar.NewReader(f)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		if h.Typeflag == tar.TypeDir {
			continue
		}
		require.False(t, filepath.IsAbs(h.Name))
		require.NotContains(t, h.Name, "..")
		b, err := io.ReadAll(tr)
		require.NoError(t, err)
		if h.Name == "index.json" {
			var index struct {
				Manifests []json.RawMessage `json:"manifests"`
			}
			require.NoError(t, json.Unmarshal(b, &index))
			descriptors = append(descriptors, index.Manifests...)
			continue
		}
		dest := filepath.Join(layout, h.Name)
		require.NoError(t, os.MkdirAll(filepath.Dir(dest), 0755))
		if previous, err := os.ReadFile(dest); err == nil {
			require.Equal(t, b, previous)
		} else {
			require.True(t, os.IsNotExist(err))
			require.NoError(t, os.WriteFile(dest, b, 0644))
		}
	}
	b, err := json.Marshal(map[string]any{"schemaVersion": 2, "mediaType": "application/vnd.oci.image.index.v1+json", "manifests": descriptors})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(indexPath, b, 0644))
}
