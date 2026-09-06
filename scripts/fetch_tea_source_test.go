package scripts

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFetchTeaSourceAcceptsSafeGzipArchive(t *testing.T) {
	root := t.TempDir()
	for _, directory := range []string{
		filepath.Join(root, "scripts"),
		filepath.Join(root, "project-os", "locks"),
		filepath.Join(root, "project-os", "licenses"),
		filepath.Join(root, ".artifacts", "tools"),
	} {
		require.NoError(t, os.MkdirAll(directory, 0o755))
	}

	script, err := os.ReadFile("fetch-tea-source.sh")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(root, "scripts", "fetch-tea-source.sh"), script, 0o755))
	license := []byte("Tea license fixture\n")
	require.NoError(t, os.WriteFile(filepath.Join(root, "project-os", "licenses", "tea-LICENSE"), license, 0o644))
	archive := gzipTarball(t)
	archiveDigest := sha256.Sum256(archive)
	licenseDigest := sha256.Sum256(license)
	lock := "version = \"0.15.1\"\n" +
		"commit = \"f34697c5ed65928e265d6f48e16928819ce0f332\"\n" +
		"source_archive = \"tea.gz\"\n" +
		"source_url = \"https://mirror.example/tea.gz\"\n" +
		"source_sha256 = \"" + fmt.Sprintf("%x", archiveDigest) + "\"\n" +
		"license_url = \"https://mirror.example/LICENSE\"\n" +
		"license_sha256 = \"" + fmt.Sprintf("%x", licenseDigest) + "\"\n"
	require.NoError(t, os.WriteFile(filepath.Join(root, "project-os", "locks", "tea-source.toml"), []byte(lock), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".artifacts", "tools", "tea.gz"), archive, 0o644))

	bin := filepath.Join(root, "bin")
	require.NoError(t, os.MkdirAll(bin, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(bin, "curl"), []byte("#!/bin/sh\necho 'network must not be used by this fixture' >&2\nexit 99\n"), 0755))
	command := exec.Command("sh", filepath.Join(root, "scripts", "fetch-tea-source.sh"))
	command.Env = append(os.Environ(), "PATH="+bin+":"+os.Getenv("PATH"))
	output, err := command.CombinedOutput()
	require.NoErrorf(t, err, "fetch Tea source: %s", output)
	// License drift must fail before accepting the cached archive or making a request.
	require.NoError(t, os.WriteFile(filepath.Join(root, "project-os", "licenses", "tea-LICENSE"), []byte("different"), 0644))
	command = exec.Command("sh", filepath.Join(root, "scripts", "fetch-tea-source.sh"))
	command.Env = append(os.Environ(), "PATH="+bin+":"+os.Getenv("PATH"))
	output, err = command.CombinedOutput()
	require.Error(t, err)
	require.Contains(t, string(output), "Tea license checksum differs")
}

func gzipTarball(t *testing.T) []byte {
	t.Helper()
	var output bytes.Buffer
	gzipWriter := gzip.NewWriter(&output)
	tarWriter := tar.NewWriter(gzipWriter)
	require.NoError(t, tarWriter.WriteHeader(&tar.Header{Name: "tea/README", Mode: 0o644, Size: 3}))
	_, err := tarWriter.Write([]byte("tea"))
	require.NoError(t, err)
	require.NoError(t, tarWriter.Close())
	require.NoError(t, gzipWriter.Close())
	return output.Bytes()
}
