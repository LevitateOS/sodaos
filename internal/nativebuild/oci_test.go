package nativebuild

import (
	"archive/tar"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
	if err := lw.WriteHeader(&tar.Header{Name: "fixture.txt", Mode: 0644, Size: int64(len(body))}); err != nil {
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
		if err := writer.WriteHeader(&tar.Header{Name: name, Mode: 0644, Size: int64(len(data))}); err != nil {
			t.Fatal(err)
		}
		if _, err := writer.Write(data); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, buf.Bytes(), 0600); err != nil {
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
	if err := os.WriteFile(file, []byte("not OCI despite extension"), 0600); err != nil {
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
	if err = os.WriteFile(file, data, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = InspectOCI(file, "x86_64", fixtureRevision); err == nil {
		t.Fatal("changed blob accepted")
	}
}
