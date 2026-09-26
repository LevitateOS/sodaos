package image

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/release/build"
	"github.com/stretchr/testify/require"
)

func TestPrepareAssemblerFloatsOnStableUpstream(t *testing.T) {
	work := filepath.Join(t.TempDir(), "media")
	digest := strings.Repeat("e", 64)
	sha := "ab12cd34ef56ab12cd34ef56ab12cd34ef56ab12"
	var runs []string
	p := build.Production{Arch: "x86_64",
		Execute: func(dir, name string, args ...string) error {
			runs = append(runs, name+" "+strings.Join(args, " "))
			if name == "tar" {
				// Simulate the config archive providing build arguments.
				require.NoError(t, os.WriteFile(filepath.Join(dir, "config", "build-args.conf"), []byte("BUILDER_IMG=placeholder\n"), 0o644))
			}
			for _, arg := range args {
				if name == "podman" && arg == "build" {
					require.NoError(t, os.WriteFile(filepath.Join(dir, "builder.iid"), []byte("sha256:"+strings.Repeat("f", 64)+"\n"), 0o644))
				}
			}
			return nil
		},
		Capture: func(dir, name string, args ...string) (string, error) {
			joined := strings.Join(args, " ")
			switch {
			case strings.Contains(joined, "rev-parse"):
				return sha + "\n", nil
			case strings.Contains(joined, "{{.Digest}}"):
				return "sha256:" + digest + "\n", nil
			case strings.Contains(joined, "RootFS.Layers"):
				return `["a","b"]`, nil
			}
			return "", nil
		}}
	lock, err := prepareAssembler(p, work)
	require.NoError(t, err)
	require.Equal(t, "quay.io/coreos-assembler/coreos-assembler@sha256:"+digest, lock.Assembler)
	require.Equal(t, sha, lock.Config)
	require.Equal(t, "x86_64", lock.Architecture)
	joined := strings.Join(runs, "\n")
	require.Contains(t, joined, "fetch --depth=1 https://github.com/coreos/fedora-coreos-config.git stable")
	require.Contains(t, joined, "archive --format=tar --output "+filepath.Join(work, "config.tar")+" "+sha)
	require.Contains(t, joined, "pull quay.io/coreos-assembler/coreos-assembler:latest")
	content, err := os.ReadFile(filepath.Join(work, "Containerfile"))
	require.NoError(t, err)
	require.Contains(t, string(content), "FROM quay.io/coreos-assembler/coreos-assembler@sha256:"+digest)
}

func TestMediaReadbackBindsNativeIgnitionAndRootfs(t *testing.T) {
	expected := []byte(`{"ignition":{"version":"3.5.0"},"storage":{"files":[]}}`)
	var compressed bytes.Buffer
	gz := gzip.NewWriter(&compressed)
	_, err := gz.Write([]byte(`{"ignition":{"version":"3.5.0"},"storage":{"files":[],"directories":null}}`))
	require.NoError(t, err)
	require.NoError(t, gz.Close())
	readback, err := json.Marshal(map[string]any{"ignition": map[string]any{"config": map[string]any{"merge": []any{map[string]string{"source": "data:;base64," + base64.StdEncoding.EncodeToString(compressed.Bytes()), "compression": "gzip"}}}}})
	require.NoError(t, err)
	require.NoError(t, VerifyLiveIgnition(readback, expected))
	require.Error(t, VerifyLiveIgnition(readback, []byte(`{"ignition":{"version":"3.5.0"}}`)))
	require.Error(t, VerifyLiveIgnition(readback, []byte(`{"ignition":{"version":"3.5.0"},"storage":{"files":[],"directories":false}}`)))
	require.Error(t, VerifyLiveIgnition(expected, expected))
	rootfs := bytes.Repeat([]byte("x"), (2<<20)+127)
	var hashes []string
	for offset := 0; offset < len(rootfs); offset += 2 << 20 {
		end := min(offset+(2<<20), len(rootfs))
		sum := sha256.Sum256(rootfs[offset:end])
		hashes = append(hashes, hex.EncodeToString(sum[:]))
	}
	manifest := "stream-hash sha256 2097152\n" + strings.Join(hashes, "\n") + "\n"
	path := filepath.Join(t.TempDir(), "rootfs.img")
	require.NoError(t, os.WriteFile(path, rootfs, 0o600))
	require.NoError(t, VerifyRootfsChunks(path, manifest))
	require.Error(t, VerifyRootfsChunks(path, strings.Join(hashes, "\n")))
	require.Error(t, VerifyRootfsChunks(path, manifest+hashes[0]+"\n"))
	rootfs[len(rootfs)-1] = 'y'
	require.NoError(t, os.WriteFile(path, rootfs, 0o600))
	require.Error(t, VerifyRootfsChunks(path, manifest))
	require.NoError(t, os.WriteFile(path, rootfs[:2<<20], 0o600))
	require.Error(t, VerifyRootfsChunks(path, manifest))
}
func TestMediaURLHasNoCredentialsOrMutableQuery(t *testing.T) {
	for _, url := range []string{"https://example.invalid/releases/candidate", "http://10.0.2.2:19843"} {
		require.NoError(t, mediaBaseURL(url))
	}
	for _, url := range []string{"", "file:///tmp/rootfs", "https://user:secret@example.invalid/", "https://example.invalid/?token=secret", "https://example.invalid/#fragment", "https://example.invalid/ bad", "http://127.0.0.1:8080", "http://localhost:8080", "http://[::1]:8080"} {
		require.Error(t, mediaBaseURL(url))
	}
}
