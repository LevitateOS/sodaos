package main

import (
	"errors"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

func extensionListener(socket string, handler http.Handler) (*http.Server, net.Listener, error) {
	if !filepath.IsAbs(socket) {
		return nil, nil, errors.New("extension socket must be an absolute path")
	}
	listener, err := net.Listen("unix", socket)
	if err != nil {
		return nil, nil, err
	}
	// The operator prepares the containing directory and group. No existing
	// socket is unlinked and no protected file is replaced on startup.
	if err := os.Chmod(socket, 0o660); err != nil {
		_ = listener.Close()
		return nil, nil, err
	}
	return &http.Server{Handler: handler, ReadHeaderTimeout: 10 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16384}, listener, nil
}
