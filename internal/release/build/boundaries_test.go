package build

import (
	"archive/tar"
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"
)

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
					if err := tw.WriteHeader(&tar.Header{Name: "blobs/", Typeflag: tar.TypeDir, Mode: 0o755}); err != nil {
						t.Fatal(err)
					}
				}
			}
			if err := tw.Close(); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(file, out.Bytes(), 0o600); err != nil {
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
	var out bytes.Buffer
	w := &limitWriter{w: &out, remaining: 3}
	if _, err := w.Write([]byte("abc")); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte("d")); err == nil || out.String() != "abc" {
		t.Fatal("input limit bypassed")
	}
}
