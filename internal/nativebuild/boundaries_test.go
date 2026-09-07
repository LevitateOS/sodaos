package nativebuild

import (
	"archive/tar"
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBundleRejectsLinkedVerifierParent(t *testing.T) {
	root := fixtureBundle(t)
	tools := filepath.Join(root, "tools")
	outside := filepath.Join(t.TempDir(), "retained-tools")
	if err := os.Rename(tools, outside); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, tools); err != nil {
		t.Fatal(err)
	}
	if _, err := tree(root); err == nil {
		t.Fatal("followed outside verifier parent")
	}
}

func TestBundleRejectsRetiredReactPayload(t *testing.T) {
	for _, name := range []string{"rootfs/usr/local/share/soda/dashboard/index.html", "inputs/dashboard-package.json", "inputs/dashboard-pnpm-lock.yaml"} {
		t.Run(name, func(t *testing.T) {
			root := fixtureBundle(t)
			file := filepath.Join(root, name)
			if err := os.MkdirAll(filepath.Dir(file), 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(file, []byte("retired React payload"), 0644); err != nil {
				t.Fatal(err)
			}
			if _, err := tree(root); err == nil {
				t.Fatal("accepted retired React payload")
			}
		})
	}
}

func TestBuildInputIdentityIsChecked(t *testing.T) {
	root := fixtureBundle(t)
	var inv Inventory
	if err := ReadJSON(filepath.Join(root, inventoryName), &inv); err != nil {
		t.Fatal(err)
	}
	if err := verifyBuildInputs(root, "x86_64", fixtureRevision, inv.Images); err != nil {
		t.Fatal(err)
	}
	if err := verifyBuildInputs(root, "aarch64", fixtureRevision, inv.Images); err == nil {
		t.Fatal("accepted mismatched input platform")
	}
	inv.Images["dashboard"] = Image{Config: "sha256:" + strings.Repeat("0", 64)}
	if err := verifyBuildInputs(root, "x86_64", fixtureRevision, inv.Images); err == nil {
		t.Fatal("accepted mismatched input image")
	}
	if err := os.WriteFile(filepath.Join(root, "inputs/native-build.json"), []byte("not JSON"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := verifyBuildInputs(root, "x86_64", fixtureRevision, inv.Images); err == nil {
		t.Fatal("accepted malformed build inputs")
	}
}

func TestOCIRejectsIndexSchemaTypeAndDuplicateDirectories(t *testing.T) {
	for _, mode := range []string{"schema", "type", "duplicate-directory", "external-descriptor"} {
		t.Run(mode, func(t *testing.T) {
			file := filepath.Join(t.TempDir(), "fixture.oci")
			fixtureOCI(t, file, "amd64", false)
			raw, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}
			tr := tar.NewReader(bytes.NewReader(raw))
			var out bytes.Buffer
			tw := tar.NewWriter(&out)
			for {
				h, err := tr.Next()
				if err == io.EOF {
					break
				}
				if err != nil {
					t.Fatal(err)
				}
				body, err := io.ReadAll(tr)
				if err != nil {
					t.Fatal(err)
				}
				if h.Name == "index.json" {
					var index map[string]any
					if err := json.Unmarshal(body, &index); err != nil {
						t.Fatal(err)
					}
					descriptor := index["manifests"].([]any)[0].(map[string]any)
					switch mode {
					case "schema":
						index["schemaVersion"] = 1
					case "type":
						descriptor["mediaType"] = "application/vnd.oci.image.index.v1+json"
					case "external-descriptor":
						descriptor["urls"] = []string{"https://example.test/blob"}
					}
					body, err = json.Marshal(index)
					if err != nil {
						t.Fatal(err)
					}
				}
				h.Size = int64(len(body))
				if err := tw.WriteHeader(h); err != nil {
					t.Fatal(err)
				}
				if _, err := tw.Write(body); err != nil {
					t.Fatal(err)
				}
			}
			if mode == "duplicate-directory" {
				for i := 0; i < 2; i++ {
					if err := tw.WriteHeader(&tar.Header{Name: "blobs/", Typeflag: tar.TypeDir, Mode: 0755}); err != nil {
						t.Fatal(err)
					}
				}
			}
			if err := tw.Close(); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(file, out.Bytes(), 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := InspectOCI(file, "x86_64", fixtureRevision); err == nil {
				t.Fatal("accepted invalid OCI structure")
			}
		})
	}
}

func TestCoreOSLockAndDownloadBoundaries(t *testing.T) {
	for _, raw := range []string{"http://example.test/base", "https://user@example.test/base", "https://example.test/base?", "https://example.test/base#", "https://example.test/base?token=x"} {
		if httpsURL(raw) {
			t.Fatalf("unsafe public input URL %q", raw)
		}
	}
	path := filepath.Join(t.TempDir(), "lock.json")
	image := CoreOSImage{URL: "https://example.test/base.xz", SignatureURL: "https://example.test/base.xz.sig", SHA256: strings.Repeat("a", 64), UncompressedSHA256: strings.Repeat("b", 64)}
	data, _ := json.Marshal(CoreOSLock{MetadataURL: "https://example.test/stream.json", Release: "fixture", Architectures: map[string]CoreOSImage{"x86_64": image}})
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := ReadCoreOS(path, "x86_64"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := ReadCoreOS(path, "aarch64"); err == nil {
		t.Fatal("accepted absent platform")
	}
	var out bytes.Buffer
	w := &limitWriter{w: &out, remaining: 3}
	if _, err := w.Write([]byte("abc")); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte("d")); err == nil || out.String() != "abc" {
		t.Fatal("input limit bypassed")
	}
}
