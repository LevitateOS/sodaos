package hostimage

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/appliancerelease"
	"github.com/levitateos/sodaos/internal/nativebuild"
	"github.com/levitateos/sodaos/internal/testoci"
	"github.com/stretchr/testify/require"
)

func stagingImages(t *testing.T) (appliancerelease.Payload, string) {
	t.Helper()
	root := t.TempDir()
	rev := strings.Repeat("a", 40)
	hash := strings.Repeat("b", 64)
	p := appliancerelease.Payload{Format: 3, ID: "44.20260817.3.2.soda-" + rev[:12], Revision: rev, Architecture: "x86_64", CoreOS: "44.20260817.3.2", Base: "quay.io/fedora/fedora-coreos@sha256:" + hash, RepositoryPrefix: "ghcr.io/example/sodaos", Schema: 10, PresentationSHA256: hash, HostPackagesSHA256: hash, Images: map[string]appliancerelease.Image{}}
	for _, name := range appliancerelease.Names {
		im := testoci.Archive(t, filepath.Join(root, name+".oci"), "amd64", rev)
		p.Images[name] = appliancerelease.Image{Reference: p.RepositoryPrefix + "-" + name + "@" + im.Manifest, Config: im.Config, Manifest: im.Manifest, ArchiveSHA256: im.ArchiveSHA256, Storage: "podman"}
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
				require.NoError(t, os.WriteFile(filepath.Join(archives, "tailnet.oci"), []byte("changed"), 0644))
			case "occupied":
				require.NoError(t, os.Mkdir(dest, 0755))
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
			dir := filepath.Join(context, "rootfs", appliancerelease.ImagesPath)
			for _, name := range appliancerelease.Names {
				testoci.Add(t, filepath.Join(archives, name+".oci"), dir)
			}
			files, _, err := appliancerelease.VerifyContent(p, dir)
			require.NoError(t, err)
			raw, err := json.Marshal(p)
			require.NoError(t, err)
			require.NoError(t, os.WriteFile(filepath.Join(out, "payload.json"), raw, 0644))
			require.NoError(t, ownedWrite(filepath.Join(out, "tools/soda-installer"), []byte("console"), 0755))
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
						if name, ok := strings.CutPrefix(arg, appliancerelease.ImagesPath+"/"); ok {
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

// Opt-in filesystem-only proof against admitted existing application exports.
// This neither builds a candidate nor starts/imports an application or VM.
func TestNativeSharedLayoutRetainedExports(t *testing.T) {
	output, artifacts := os.Getenv("SODA_LAYOUT_NATIVE_OUT"), os.Getenv("SODA_LAYOUT_NATIVE_INPUT")
	if output == "" && artifacts == "" {
		t.Skip("set fresh SODA_LAYOUT_NATIVE_OUT and retained SODA_LAYOUT_NATIVE_INPUT")
	}
	require.True(t, filepath.IsAbs(output) && filepath.IsAbs(artifacts))
	require.NoError(t, os.Mkdir(output, 0700))
	p, err := appliancerelease.Load(filepath.Join(artifacts, "payload.json"))
	require.NoError(t, err)
	require.Equal(t, 2, p.Format)
	p.Format = 3
	run := func(dir, cmd string, args ...string) error {
		c := exec.CommandContext(context.Background(), cmd, args...)
		c.Dir = dir
		b, e := c.CombinedOutput()
		if e != nil {
			return errors.New(string(b))
		}
		return nil
	}
	version, err := exec.Command("skopeo", "--version").Output()
	require.NoError(t, err)
	require.Contains(t, string(version), "skopeo version 1.22.2")
	layout := filepath.Join(output, "layout")
	require.NoError(t, stageImages(filepath.Join(artifacts, "images"), layout, p, run))
	files, total, err := appliancerelease.VerifyContent(p, layout)
	require.NoError(t, err)
	var archives uint64
	for _, name := range appliancerelease.Names {
		original := filepath.Join(artifacts, "images", name+".oci")
		st, err := os.Stat(original)
		require.NoError(t, err)
		archives += uint64(st.Size())
		hash, err := nativebuild.HashFile(original)
		require.NoError(t, err)
		require.Equal(t, p.Images[name].ArchiveSHA256, hash)
		reference := "oci:" + layout + ":" + p.Images[name].Config
		raw, err := exec.Command("skopeo", "inspect", "--raw", reference).Output()
		require.NoError(t, err)
		require.Equal(t, p.Images[name].Manifest, "sha256:"+hashBytes(raw))
	}
	require.Less(t, total, archives)
	// Native OCI export must also preserve valid uncompressed layer manifests,
	// rather than letting Skopeo silently select a different representation.
	tiny, tinyArchives := stagingImages(t)
	require.NoError(t, stageImages(tinyArchives, filepath.Join(output, "uncompressed-layout"), tiny, run))
	result, err := json.MarshalIndent(map[string]any{"input_revision": p.Revision, "payload_format": 3, "archives_bytes": archives, "layout_bytes": total, "reduction_bytes": archives - total, "unique_files": len(files), "native_tool": strings.TrimSpace(string(version)), "candidate_rebuilt": false, "native_import_or_install": false}, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(output, "result.json"), append(result, '\n'), 0644))
}
