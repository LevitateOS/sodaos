// soda-rootfs-server is the exact-file installer-rootfs HTTP server for
// libvirt guests. It serves ONLY ^[0-9a-f]{64}-rootfs\.img$ names from the
// rootfs-only directory on the guest-reachable bridge address.
//
// Deny by construction: no directory listing, no QCOW2, no ISO, no
// traversal, no symlinks, no subpaths, GET/HEAD only. Anything else is
// 404/501.
//
// Single source for soda-rootfs-server.service: the unit runs the installed
// binary built from this package, and soda-candidate-setup
// creates the served directory below. There is no second copy; behavior
// checks live in main_test.go.
package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
)

const (
	serverName = "soda-rootfs/1"
	// listenAddr is the loopback-independent fixed bridge bind: guests
	// fetch 192.168.122.1:8080.
	listenAddr = "192.168.122.1:8080"
	// chunkSize streams 1 MiB at a time, matching the retired Python server.
	chunkSize = 1048576
)

// rootDir holds only *-rootfs.img files on the roomy disk. Setup
// (soda-candidate-setup) creates it; tests repoint it.
var rootDir = "/home/soda-rootfs"

var allowName = regexp.MustCompile(`^[0-9a-f]{64}-rootfs\.img$`)

// openAllowed resolves a request path to a served file. Anything outside
// the exact-file contract reports false and the caller answers 404.
func openAllowed(path string) (*os.File, bool) {
	if !strings.HasPrefix(path, "/") || strings.Contains(path[1:], "/") {
		return nil, false
	}
	token := path[1:]
	if !allowName.MatchString(token) {
		return nil, false
	}
	// O_NOFOLLOW refuses symlinks; the regular-file check refuses anything
	// else that is not a plain file.
	file, err := os.OpenFile(filepath.Join(rootDir, token), os.O_RDONLY|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return nil, false
	}
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		_ = file.Close()
		return nil, false
	}
	return file, true
}

func serveRootfs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Server", serverName)
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "Unsupported method", http.StatusNotImplemented)
		return
	}
	file, ok := openAllowed(r.URL.Path)
	if !ok {
		http.NotFound(w, r)
		return
	}
	defer func() { _ = file.Close() }()
	info, err := file.Stat()
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", strconv.FormatInt(info.Size(), 10))
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	if r.Method == http.MethodGet {
		// A torn client connection mid-stream has no status left to send.
		_, _ = io.CopyBuffer(w, file, make([]byte, chunkSize))
	}
}

func run() error {
	return http.ListenAndServe(listenAddr, http.HandlerFunc(serveRootfs))
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
