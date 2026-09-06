package acceptance

import (
	"archive/tar"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/levitateos/sodaos/internal/nativebuild"
)

func TestStreamBundleVerifiesChangingBytesAndLinks(t *testing.T) {
	for _, mode := range []string{"valid", "changed", "link"} {
		t.Run(mode, func(t *testing.T) {
			root := t.TempDir()
			if err := os.Mkdir(filepath.Join(root, "inputs"), 0755); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(root, "inputs/go.mod")
			for _, file := range []string{path, filepath.Join(root, "build-info.json"), filepath.Join(root, "SHA256SUMS")} {
				if err := os.WriteFile(file, []byte("synthetic bytes; no executable"), 0644); err != nil {
					t.Fatal(err)
				}
			}
			st, err := os.Stat(path)
			if err != nil {
				t.Fatal(err)
			}
			hash, err := nativebuild.HashFile(path)
			if err != nil {
				t.Fatal(err)
			}
			inv := nativebuild.Inventory{Files: map[string]nativebuild.File{"inputs/go.mod": {SHA256: hash, Mode: uint32(st.Mode().Perm())}}}
			if mode == "changed" {
				if err := os.WriteFile(path, []byte("changed bytes"), st.Mode().Perm()); err != nil {
					t.Fatal(err)
				}
			}
			if mode == "link" {
				outside := filepath.Join(t.TempDir(), "outside")
				if err := os.WriteFile(outside, []byte("not transport payload"), 0600); err != nil {
					t.Fatal(err)
				}
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(outside, path); err != nil {
					t.Fatal(err)
				}
			}
			var out bytes.Buffer
			err = streamBundle(&out, root, inv)
			if mode != "valid" {
				if err == nil {
					t.Fatal("stream accepted changed/symlinked input")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			reader := tar.NewReader(&out)
			seen := map[string]bool{}
			for {
				header, err := reader.Next()
				if err == io.EOF {
					break
				}
				if err != nil {
					t.Fatal(err)
				}
				seen[header.Name] = true
				if header.Uid != 0 || header.Gid != 0 {
					t.Fatal("host ownership leaked into transport")
				}
			}
			if len(seen) != 3 || !seen["inputs/go.mod"] || !seen["SHA256SUMS"] {
				t.Fatal("unexpected transport file set")
			}
		})
	}
}
