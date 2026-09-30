package build

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const fixtureRevision = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func fixtureOCI(t *testing.T, file, arch string, unsafe bool) {
	t.Helper()
	blobs := map[string][]byte{}
	add := func(data []byte) descriptor {
		sum := sha256.Sum256(data)
		digest := hex.EncodeToString(sum[:])
		blobs["blobs/sha256/"+digest] = data
		return descriptor{Digest: "sha256:" + digest, Size: int64(len(data))}
	}
	var layerBytes bytes.Buffer
	lw := tar.NewWriter(&layerBytes)
	body := []byte("synthetic layer fixture; never executed")
	if err := lw.WriteHeader(&tar.Header{Name: "fixture.txt", Mode: 0o644, Size: int64(len(body))}); err != nil {
		t.Fatal(err)
	}
	if _, err := lw.Write(body); err != nil {
		t.Fatal(err)
	}
	if err := lw.Close(); err != nil {
		t.Fatal(err)
	}
	layer := add(layerBytes.Bytes())
	layer.MediaType = "application/vnd.oci.image.layer.v1.tar"
	config, _ := json.Marshal(map[string]any{"os": "linux", "architecture": arch, "rootfs": map[string]any{"type": "layers", "diff_ids": []string{layer.Digest}}, "config": map[string]any{"Labels": map[string]string{"org.opencontainers.image.revision": fixtureRevision, "org.opencontainers.image.source": "https://github.com/LevitateOS/sodaos", "org.opencontainers.image.base.name": "synthetic-base", "org.opencontainers.image.base.digest": "sha256:" + strings.Repeat("b", 64)}}})
	cfg := add(config)
	cfg.MediaType = "application/vnd.oci.image.config.v1+json"
	manifest, _ := json.Marshal(map[string]any{"schemaVersion": 2, "config": cfg, "layers": []descriptor{layer}})
	m := add(manifest)
	m.MediaType = "application/vnd.oci.image.manifest.v1+json"
	blobs["index.json"], _ = json.Marshal(map[string]any{"schemaVersion": 2, "manifests": []descriptor{m}})
	blobs["oci-layout"] = []byte(`{"imageLayoutVersion":"1.0.0"}`)
	if unsafe {
		blobs["../escape"] = []byte("unsafe")
	}
	var buf bytes.Buffer
	writer := tar.NewWriter(&buf)
	for name, data := range blobs {
		if err := writer.WriteHeader(&tar.Header{Name: name, Mode: 0o644, Size: int64(len(data))}); err != nil {
			t.Fatal(err)
		}
		if _, err := writer.Write(data); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, buf.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestOCIIdentityAndWrongPlatform(t *testing.T) {
	file := filepath.Join(t.TempDir(), "image.oci")
	fixtureOCI(t, file, "amd64", false)
	image, err := InspectOCI(file, "x86_64", fixtureRevision)
	if err != nil {
		t.Fatal(err)
	}
	if image.Architecture != "amd64" || image.Revision != fixtureRevision || image.BaseName != "synthetic-base" {
		t.Fatalf("%#v", image)
	}
	if _, err = InspectOCI(file, "aarch64", fixtureRevision); err == nil {
		t.Fatal("wrong platform accepted")
	}
	if _, err = InspectOCI(file, "x86_64", strings.Repeat("c", 40)); err == nil {
		t.Fatal("wrong source accepted")
	}
}

func TestOCIRejectsEscapesAndMisnamedFormats(t *testing.T) {
	file := filepath.Join(t.TempDir(), "image.oci")
	fixtureOCI(t, file, "amd64", true)
	if _, err := InspectOCI(file, "x86_64", fixtureRevision); err == nil {
		t.Fatal("escaping archive accepted")
	}
	if err := os.WriteFile(file, []byte("not OCI despite extension"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := InspectOCI(file, "x86_64", fixtureRevision); err == nil {
		t.Fatal("filename became format proof")
	}
}

func TestOCIRejectsChangedBlob(t *testing.T) {
	file := filepath.Join(t.TempDir(), "image.oci")
	fixtureOCI(t, file, "amd64", false)
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	data = bytes.Replace(data, []byte("synthetic layer"), []byte("tampered! layer"), 1)
	if err = os.WriteFile(file, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err = InspectOCI(file, "x86_64", fixtureRevision); err == nil {
		t.Fatal("changed blob accepted")
	}
}

func TestOCIMemberScannerRejectsMalformedDuplicateAndMissingEntries(t *testing.T) {
	layer := func(names ...string) []byte {
		t.Helper()
		var out bytes.Buffer
		tw := tar.NewWriter(&out)
		for _, name := range names {
			body := []byte(name)
			if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o644, Size: int64(len(body))}); err != nil {
				t.Fatal(err)
			}
			if _, err := tw.Write(body); err != nil {
				t.Fatal(err)
			}
		}
		if err := tw.Close(); err != nil {
			t.Fatal(err)
		}
		return out.Bytes()
	}
	wanted := map[string]string{"wanted": "/wanted"}
	if _, err := scanOCILayer(bytes.NewReader(layer("wanted", "wanted")), wanted); err == nil {
		t.Fatal("duplicate layer member accepted")
	}
	if _, err := scanOCILayer(bytes.NewReader(layer("../wanted")), wanted); err == nil {
		t.Fatal("malformed layer member accepted")
	}
	members, err := scanOCILayer(bytes.NewReader(layer("other")), wanted)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := resolveOCIMembers([]map[string]layerMember{members}, []bool{false}, wanted); err == nil {
		t.Fatal("missing layer member accepted")
	}
}

func TestOCIMemberScannerRejectsNonDirectoryAncestor(t *testing.T) {
	layer := func(typeflag byte) []byte {
		t.Helper()
		var out bytes.Buffer
		tw := tar.NewWriter(&out)
		header := &tar.Header{Name: "usr/share/soda", Mode: 0o755, Typeflag: typeflag}
		if typeflag == tar.TypeSymlink {
			header.Linkname = "elsewhere"
		} else if typeflag == tar.TypeReg {
			header.Size = 1
		}
		require.NoError(t, tw.WriteHeader(header))
		if typeflag == tar.TypeReg {
			_, err := tw.Write([]byte("x"))
			require.NoError(t, err)
		}
		require.NoError(t, tw.Close())
		return out.Bytes()
	}
	wanted := map[string]string{"usr/share/soda/extension/run": "/usr/share/soda/extension/run"}
	lower := map[string]layerMember{"usr/share/soda/extension/run": {hash: strings.Repeat("a", 64), present: true}}
	for _, typeflag := range []byte{tar.TypeSymlink, tar.TypeReg} {
		upper, err := scanOCILayer(bytes.NewReader(layer(typeflag)), wanted)
		require.NoError(t, err)
		_, err = resolveOCIMembers([]map[string]layerMember{lower, upper}, []bool{false, false}, wanted)
		require.ErrorContains(t, err, "non-directory ancestor")
	}
	upper, err := scanOCILayer(bytes.NewReader(layer(tar.TypeDir)), wanted)
	require.NoError(t, err)
	resolved, err := resolveOCIMembers([]map[string]layerMember{lower, upper}, []bool{false, false}, wanted)
	require.NoError(t, err)
	require.Equal(t, strings.Repeat("a", 64), resolved["/usr/share/soda/extension/run"])
}

func TestOCIMemberScannerAppliesWhiteouts(t *testing.T) {
	wanted := map[string]string{"usr/share/soda/run": "/usr/share/soda/run"}
	lower := map[string]layerMember{"usr/share/soda/run": {hash: strings.Repeat("a", 64), present: true}}
	for _, whiteout := range []string{"usr/share/soda/.wh.run", "usr/share/soda/.wh..wh..opq"} {
		var data bytes.Buffer
		tw := tar.NewWriter(&data)
		require.NoError(t, tw.WriteHeader(&tar.Header{Name: whiteout, Mode: 0o644, Size: 0}))
		require.NoError(t, tw.Close())
		upper, err := scanOCILayer(bytes.NewReader(data.Bytes()), wanted)
		require.NoError(t, err)
		_, err = resolveOCIMembers([]map[string]layerMember{lower, upper}, []bool{false, false}, wanted)
		require.ErrorContains(t, err, "removed")
	}
}

func TestOCIArchiveLayerCompressionAndDigest(t *testing.T) {
	var layer bytes.Buffer
	layerTar := tar.NewWriter(&layer)
	content := []byte("verified member")
	require.NoError(t, layerTar.WriteHeader(&tar.Header{Name: "wanted", Mode: 0o644, Size: int64(len(content))}))
	_, err := layerTar.Write(content)
	require.NoError(t, err)
	require.NoError(t, layerTar.Close())
	var compressed bytes.Buffer
	gz := gzip.NewWriter(&compressed)
	_, err = gz.Write(layer.Bytes())
	require.NoError(t, err)
	require.NoError(t, gz.Close())
	wanted := map[string]string{"wanted": "/wanted"}
	for _, tc := range []struct {
		mediaType string
		data      []byte
		blocked   bool
	}{
		{"application/vnd.oci.image.layer.v1.tar", layer.Bytes(), false},
		{"application/vnd.oci.image.layer.v1.tar+gzip", compressed.Bytes(), false},
		{"application/vnd.oci.image.layer.v1.tar+zstd", compressed.Bytes(), true},
	} {
		sum := sha256.Sum256(tc.data)
		digest := "sha256:" + hex.EncodeToString(sum[:])
		var archive bytes.Buffer
		archiveTar := tar.NewWriter(&archive)
		name := "blobs/sha256/" + hex.EncodeToString(sum[:])
		require.NoError(t, archiveTar.WriteHeader(&tar.Header{Name: name, Mode: 0o644, Size: int64(len(tc.data))}))
		_, err = archiveTar.Write(tc.data)
		require.NoError(t, err)
		require.NoError(t, archiveTar.Close())
		desc := descriptor{Digest: digest, MediaType: tc.mediaType}
		members, unsupported, err := scanOCIArchiveLayers(bytes.NewReader(archive.Bytes()), []descriptor{desc}, wanted)
		require.NoError(t, err)
		require.Equal(t, tc.blocked, unsupported[0])
		if tc.blocked {
			_, err = resolveOCIMembers(members, unsupported, wanted)
			require.ErrorContains(t, err, "zstd OCI layer blocks")
		} else {
			contentSum := sha256.Sum256(content)
			require.Equal(t, hex.EncodeToString(contentSum[:]), members[0]["wanted"].hash)
		}
		desc.Digest = "sha256:" + strings.Repeat("0", 64)
		_, _, err = scanArchiveLayer(bytes.NewReader(tc.data), desc, wanted)
		require.ErrorContains(t, err, "OCI layer changed")
	}
}
