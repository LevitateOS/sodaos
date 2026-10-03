package main

import (
	"errors"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

func extensionSocketLive(socket string) bool {
	conn, err := net.DialUnix("unix", nil, &net.UnixAddr{Name: socket, Net: "unix"})
	if err != nil {
		return !errors.Is(err, syscall.ECONNREFUSED)
	}
	_ = conn.Close()
	return true
}

func extensionListener(socket string, handler http.Handler) (*http.Server, net.Listener, error) {
	if !filepath.IsAbs(socket) {
		return nil, nil, errors.New("extension socket must be an absolute path")
	}
	if info, err := os.Lstat(socket); err == nil {
		// The operator prepares the containing directory and group. A live
		// socket or any non-socket file is never replaced; only a stale
		// socket from an unclean shutdown is reclaimed.
		if info.Mode().Type() != os.ModeSocket || extensionSocketLive(socket) {
			return nil, nil, errors.New("extension socket path is occupied")
		}
		if err := os.Remove(socket); err != nil {
			return nil, nil, err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, nil, err
	}
	listener, err := net.Listen("unix", socket)
	if err != nil {
		return nil, nil, err
	}
	// The Forgejo extension process runs as a bare container uid: its
	// entrypoint drops the supplemental IPC group, so group membership
	// cannot reach it. Other-write admits its connect(); connecting alone
	// grants nothing because every request still needs a live Forgejo
	// native authority (see auth.ExtensionAuthority).
	if err := os.Chmod(socket, 0o662); err != nil {
		_ = listener.Close()
		return nil, nil, err
	}
	return &http.Server{Handler: handler, ReadHeaderTimeout: 10 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16384}, listener, nil
}
