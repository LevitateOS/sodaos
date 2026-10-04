// D3: setup, build filing, and the persistent server share one rootfs-only
// directory.
//
// The installer ISO is only the boot menu; the guest fetches the hash-named
// rootfs image over HTTP after the builder exits. These checks read the real
// setup source and the staged systemd unit against this package's served
// directory: the three must agree on one directory that holds only rootfs
// images, served without listings, QCOW2, ISO, traversal, subpaths,
// symlinks, or non-GET/HEAD methods. No services, units, or host paths are
// touched.
package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// liveGuestImages is the live guest-disk directory: QCOW2s and installer
// ISOs live here and must never be a served document root again.
const liveGuestImages = "/home/libvirt/images"

var allowNameFixture = strings.Repeat("f", 64) + "-rootfs.img"

func repoPath(t *testing.T, elems ...string) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok, "caller unavailable")
	return filepath.Join(append([]string{filepath.Dir(file), "..", ".."}, elems...)...)
}

func setupRootfsDir(t *testing.T, text string) string {
	t.Helper()
	match := regexp.MustCompile(`(?m)^const ROOTFS_DIR: &str = "([^"]+)";`).FindStringSubmatch(text)
	require.Len(t, match, 2, "setup source names no ROOTFS_DIR")
	return match[1]
}

func TestSetupAndServerConvergeOnOneHomeDirectory(t *testing.T) {
	setup, err := os.ReadFile(repoPath(t, "rust", "soda-candidate-setup", "src", "main.rs"))
	require.NoError(t, err)
	served := rootDir
	require.Equal(t, served, setupRootfsDir(t, string(setup)),
		"setup and the persistent server must file and serve the same directory")
	require.True(t, strings.HasPrefix(served, "/home/"),
		"rootfs images are gigabytes; %s must stay off the small root filesystem", served)
	require.NotEqual(t, liveGuestImages, served, "the guest-disk directory must not be served")
}

func TestUnitRunsTheSingleServerSource(t *testing.T) {
	raw, err := os.ReadFile(repoPath(t, "scripts", "ops", "soda-rootfs-server.service"))
	require.NoError(t, err)
	unit := string(raw)
	require.NotContains(t, unit, "b64decode", "the unit must not embed a duplicate encoded server")
	for _, line := range strings.Split(unit, "\n") {
		require.NotRegexp(t, `^#S`, line, "the unit must not embed a duplicate commented server")
	}
	var installs []string
	for _, line := range strings.Split(unit, "\n") {
		if strings.HasPrefix(line, "ExecStart=") {
			installs = append(installs, line)
		}
	}
	require.Len(t, installs, 1, "the unit must run exactly one server command")
	require.Contains(t, installs[0], "soda-rootfs-server",
		"ExecStart must run the single server binary built from cmd/soda-rootfs-server")
	require.NotContains(t, installs[0], "python",
		"ExecStart must run the Go binary directly, not through an interpreter")
	require.Contains(t, installs[0]+"\n"+unit, "192.168.122.1",
		"guests keep fetching the same bridge address")
}

func TestExactFileServing(t *testing.T) {
	work := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(work, allowNameFixture), []byte("installer-rootfs-bytes"), 0o644))
	for name, body := range map[string]string{
		"soda-iso29.qcow2":         "guest-disk",
		"soda-iso29-installer.iso": "installer-iso",
		"notes.txt":                "unrelated",
	} {
		require.NoError(t, os.WriteFile(filepath.Join(work, name), []byte(body), 0o644))
	}
	require.NoError(t, os.Mkdir(filepath.Join(work, "sub"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(work, "sub", allowNameFixture), []byte("nested"), 0o644))
	linkName := strings.Repeat("e", 64) + "-rootfs.img"
	if err := os.Symlink(allowNameFixture, filepath.Join(work, linkName)); err != nil {
		t.Skip("symlinks unavailable")
	}

	oldRoot := rootDir
	rootDir = work
	t.Cleanup(func() { rootDir = oldRoot })

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	server := &http.Server{Handler: http.HandlerFunc(serveRootfs)}
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() { _ = server.Close() })
	addr := listener.Addr().String()

	client := &http.Client{Timeout: 10 * time.Second}
	fetch := func(method, path string) (int, http.Header, []byte) {
		request, err := http.NewRequest(method, "http://"+addr+path, nil)
		require.NoError(t, err)
		response, err := client.Do(request)
		require.NoError(t, err)
		defer func() { _ = response.Body.Close() }()
		body, err := io.ReadAll(response.Body)
		require.NoError(t, err)
		return response.StatusCode, response.Header, body
	}

	status, header, body := fetch(http.MethodGet, "/"+allowNameFixture)
	require.Equal(t, http.StatusOK, status, "the exact installer rootfs must be fetchable")
	require.Equal(t, []byte("installer-rootfs-bytes"), body)
	require.Equal(t, "no-store", header.Get("Cache-Control"))
	require.Equal(t, "application/octet-stream", header.Get("Content-Type"))
	require.Contains(t, header.Get("Server"), "soda-rootfs/1")
	require.Equal(t, fmt.Sprint(len("installer-rootfs-bytes")), header.Get("Content-Length"))

	status, _, body = fetch(http.MethodHead, "/"+allowNameFixture)
	require.Equal(t, http.StatusOK, status)
	require.Empty(t, body)

	for _, path := range []string{
		"/",
		"/soda-iso29.qcow2",
		"/soda-iso29-installer.iso",
		"/notes.txt",
		"/sub/" + allowNameFixture,
		"/%2e%2e/etc/passwd",
		"/" + linkName,
	} {
		t.Run(path, func(t *testing.T) {
			status, _, _ := fetch(http.MethodGet, path)
			require.Equal(t, http.StatusNotFound, status)
		})
	}

	raw, err := net.DialTimeout("tcp", addr, 10*time.Second)
	require.NoError(t, err)
	defer func() { _ = raw.Close() }()
	_, err = fmt.Fprintf(raw, "POST /%s HTTP/1.0\r\nContent-Length: 0\r\n\r\n", allowNameFixture)
	require.NoError(t, err)
	reply, err := bufio.NewReader(raw).ReadString('\n')
	require.NoError(t, err)
	require.Contains(t, reply, "501", "only GET/HEAD are served")
}
