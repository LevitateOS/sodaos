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

	"github.com/stretchr/testify/require"
)

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
	for _, url := range []string{"", "file:///tmp/rootfs", "https://user:secret@example.invalid/", "https://example.invalid/?token=secret", "https://example.invalid/#fragment", "https://example.invalid/ bad"} {
		require.Error(t, mediaBaseURL(url))
	}
}
