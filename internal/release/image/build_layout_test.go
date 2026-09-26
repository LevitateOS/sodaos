package image

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/release/deliver"
	"github.com/levitateos/sodaos/internal/testoci"
	"github.com/stretchr/testify/require"
)

func stagingImages(t *testing.T) (deliver.Payload, string) {
	t.Helper()
	root := t.TempDir()
	rev := strings.Repeat("a", 40)
	hash := strings.Repeat("b", 64)
	p := deliver.Payload{Format: 3, ID: "44.20260817.3.2.soda-" + rev[:12], Revision: rev, Architecture: "x86_64", CoreOS: "44.20260817.3.2", Base: "quay.io/fedora/fedora-coreos@sha256:" + hash, RepositoryPrefix: "ghcr.io/example/sodaos", Schema: 10, PresentationSHA256: hash, HostPackagesSHA256: hash, Images: map[string]deliver.Image{}}
	for _, name := range deliver.Names {
		im := testoci.Archive(t, filepath.Join(root, name+".oci"), "amd64", rev)
		p.Images[name] = deliver.Image{Reference: p.RepositoryPrefix + "-" + name + "@" + im.Manifest, Config: im.Config, Manifest: im.Manifest, ArchiveSHA256: im.ArchiveSHA256}
	}
	return p, root
}
func TestSharedLayoutStagingRefusesBeforeCopiesAndStopsOnFailure(t *testing.T) {
	for _, kind := range []string{"changed-archive", "occupied", "linked", "bad-format", "ambiguous-path", "failed-copy"} {
		t.Run(kind, func(t *testing.T) {
			p, archives := stagingImages(t)
			dest := filepath.Join(t.TempDir(), "layout")
			calls := 0
			switch kind {
			case "changed-archive":
				require.NoError(t, os.WriteFile(filepath.Join(archives, "tailnet.oci"), []byte("changed"), 0o644))
			case "occupied":
				require.NoError(t, os.Mkdir(dest, 0o755))
			case "linked":
				require.NoError(t, os.Symlink(archives, dest))
			case "bad-format":
				p.Format = 2
			case "ambiguous-path":
				dest += ":wrong"
			}
			err := stageImages(archives, dest, p, func(string, string, ...string) error { calls++; return errors.New("copy failed") })
			require.Error(t, err)
			if kind == "failed-copy" {
				require.Equal(t, 1, calls)
			} else {
				require.Zero(t, calls)
			}
		})
	}
}
func TestNativeHostReadbackChecksEverySharedBlob(t *testing.T) {
	for _, fault := range []string{"", "missing", "changed"} {
		t.Run("readback-"+fault, func(t *testing.T) {
			p, archives := stagingImages(t)
			context := t.TempDir()
			out := t.TempDir()
			dir := filepath.Join(context, "rootfs", deliver.ImagesPath)
			for _, name := range deliver.Names {
				testoci.Add(t, filepath.Join(archives, name+".oci"), dir)
			}
			files, _, err := deliver.VerifyContent(p, dir)
			require.NoError(t, err)
			raw, err := json.Marshal(p)
			require.NoError(t, err)
			require.NoError(t, os.WriteFile(filepath.Join(out, "payload.json"), raw, 0o644))
			require.NoError(t, ownedWrite(filepath.Join(out, "tools/soda-installer"), []byte("console"), 0o755))
			consoleHash := hashBytes([]byte("console"))
			checked := 0
			err = inspectComplete(context, out, "candidate", func(_ string, cmd string, args ...string) (string, error) {
				require.Equal(t, "podman", cmd)
				joined := strings.Join(args, " ")
				switch {
				case strings.Contains(joined, "payload-inspect.cid"):
					return string(raw), nil
				case strings.Contains(joined, "content-inspect.cid"):
					var lines []string
					for _, arg := range args {
						if name, ok := strings.CutPrefix(arg, deliver.ImagesPath+"/"); ok {
							hash, exists := files[name]
							require.True(t, exists)
							lines = append(lines, hash+"  "+arg)
							checked++
						}
					}
					require.Equal(t, len(files), checked)
					if fault == "missing" {
						lines = lines[:len(lines)-1]
					}
					if fault == "changed" {
						lines[0] = "wrong"
					}
					return strings.Join(lines, "\n"), nil
				case strings.Contains(joined, "installer-inspect.cid"):
					return consoleHash + "  /usr/libexec/soda/soda-install", nil
				case strings.Contains(joined, "quadlet-inspect.cid"):
					return "soda-image-import.service " + p.Images["forgejo"].Config + " " + p.Images["dashboard"].Config + " " + p.Images["proxy"].Config, nil
				default:
					t.Fatalf("unexpected native inspection: %v", args)
					return "", nil
				}
			})
			if fault == "" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}
