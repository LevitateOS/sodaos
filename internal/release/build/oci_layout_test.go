package build_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/release/build"
	"github.com/levitateos/sodaos/internal/testoci"
	"github.com/stretchr/testify/require"
)

func layoutFixture(t *testing.T) (string, map[string]string) {
	t.Helper()
	root := t.TempDir()
	layout := filepath.Join(root, "layout")
	revisions := map[string]string{}
	for _, name := range []string{"first", "second"} {
		archive := filepath.Join(root, name+".oci")
		im := testoci.Archive(t, archive, "amd64", strings.Repeat("a", 40))
		testoci.Add(t, archive, layout)
		revisions[im.Config] = strings.Repeat("a", 40)
	}
	return layout, revisions
}
func TestSharedOCILayoutPreservesIdentitiesAndCountsBlobsOnce(t *testing.T) {
	dir, revisions := layoutFixture(t)
	got, err := build.InspectOCILayout(dir, "x86_64", revisions)
	require.NoError(t, err)
	require.Len(t, got.Images, 2)
	require.Len(t, got.Files, 7)
	var total uint64
	for name, hash := range got.Files {
		b, err := os.ReadFile(filepath.Join(dir, name))
		require.NoError(t, err)
		sum := sha256.Sum256(b)
		require.Equal(t, hex.EncodeToString(sum[:]), hash)
		total += uint64(len(b))
	}
	require.Equal(t, total, got.Bytes)
	for ref, image := range got.Images {
		require.Equal(t, ref, image.Config)
		require.Equal(t, "amd64", image.Architecture)
		require.Equal(t, revisions[ref], image.Revision)
	}
	_, err = build.InspectOCILayout(dir, "aarch64", revisions)
	require.ErrorContains(t, err, "linux/arm64")
	for ref := range revisions {
		revisions[ref] = strings.Repeat("b", 40)
	}
	_, err = build.InspectOCILayout(dir, "x86_64", revisions)
	require.ErrorContains(t, err, "revision mismatch")
}
func TestSharedOCILayoutRefusesSubstitution(t *testing.T) {
	for _, kind := range []string{"missing", "corrupt", "symlink-file", "symlink-dir", "symlink-root", "duplicate-ref", "wrong-ref", "wrong-size", "external-url", "empty-index", "nested-index"} {
		t.Run(kind, func(t *testing.T) {
			dir, revisions := layoutFixture(t)
			before, err := build.InspectOCILayout(dir, "x86_64", revisions)
			require.NoError(t, err)
			var blob string
			for name := range before.Files {
				if strings.HasPrefix(name, "blobs/") {
					blob = filepath.Join(dir, name)
					break
				}
			}
			switch kind {
			case "missing":
				require.NoError(t, os.Remove(blob))
			case "corrupt":
				require.NoError(t, os.WriteFile(blob, []byte("corrupted"), 0o644))
			case "symlink-file":
				outside := filepath.Join(filepath.Dir(dir), "outside-blob")
				require.NoError(t, os.Rename(blob, outside))
				require.NoError(t, os.Symlink(outside, blob))
			case "symlink-dir":
				require.NoError(t, os.Rename(filepath.Join(dir, "blobs"), filepath.Join(filepath.Dir(dir), "outside")))
				require.NoError(t, os.Symlink(filepath.Join(filepath.Dir(dir), "outside"), filepath.Join(dir, "blobs")))
			case "symlink-root":
				link := dir + "-link"
				require.NoError(t, os.Symlink(dir, link))
				dir = link
			default:
				path := filepath.Join(dir, "index.json")
				b, err := os.ReadFile(path)
				require.NoError(t, err)
				var index map[string]any
				require.NoError(t, json.Unmarshal(b, &index))
				ds := index["manifests"].([]any)
				first := ds[0].(map[string]any)
				switch kind {
				case "duplicate-ref":
					ds[1] = first
				case "wrong-ref":
					first["annotations"] = map[string]string{"org.opencontainers.image.ref.name": "latest"}
				case "wrong-size":
					first["size"] = float64(1)
				case "external-url":
					first["urls"] = []string{"https://example.invalid/layer"}
				case "empty-index":
					index["manifests"] = []any{}
				case "nested-index":
					first["mediaType"] = "application/vnd.oci.image.index.v1+json"
				}
				b, err = json.Marshal(index)
				require.NoError(t, err)
				require.NoError(t, os.WriteFile(path, b, 0o644))
			}
			_, err = build.InspectOCILayout(dir, "x86_64", revisions)
			require.Error(t, err)
		})
	}
}
