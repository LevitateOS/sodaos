package main

// Dedicated Factory-only Unix HTTP listener (FW38).
//
// The role-filtered catalog serves at the fixed factory-ipc/mcp.sock
// relative location under the service private data root (see
// web.FactoryMCPSocketPath: no flag, request input or configuration
// key moves it). The IPC directory is service-owned mode 0755 and
// the socket accepts connections across Project user namespaces, visible
// only in admitted Project mounts; connecting alone grants nothing
// because every request must authenticate its live Factory capability. A
// live socket or any non-socket file is never replaced; only a stale
// socket from an unclean shutdown is reclaimed, so dashboard socket
// unlink/rebind reconnects the live Project without new run
// authority: reinitialization after restart never recreates
// capability or submission authority.

import (
	"context"
	"errors"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/levitateos/sodaos/internal/web"
)

func factoryMCPSocketLive(socket string) bool {
	conn, err := net.DialUnix("unix", nil, &net.UnixAddr{Name: socket, Net: "unix"})
	if err != nil {
		return !errors.Is(err, syscall.ECONNREFUSED)
	}
	_ = conn.Close()
	return true
}

func factoryMCPListener(socket string, handler http.Handler) (*http.Server, net.Listener, error) {
	if socket == "" || !filepath.IsAbs(socket) {
		return nil, nil, errors.New("factory MCP socket must be an absolute path")
	}
	if err := os.MkdirAll(filepath.Dir(socket), 0o755); err != nil {
		return nil, nil, err
	}
	if info, err := os.Lstat(socket); err == nil {
		if info.Mode().Type() != os.ModeSocket || factoryMCPSocketLive(socket) {
			return nil, nil, errors.New("factory MCP socket path is occupied")
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
	// Project user namespaces have distinct UID/GID mappings. Connecting to
	// this dedicated socket grants no authority: each request must prove its
	// run-specific HMAC secret, and only this directory is mounted in workers.
	if err := os.Chmod(socket, 0o666); err != nil {
		_ = listener.Close()
		return nil, nil, err
	}
	return &http.Server{Handler: handler, ReadHeaderTimeout: 10 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16384}, listener, nil
}

func factoryMCPHTTPServer(app *web.Server) (*http.Server, net.Listener, error) {
	return factoryMCPListener(web.FactoryMCPSocketPath(app.Config), app.FactoryMCPHandler())
}

func serveFactoryMCPSocket(server *http.Server, listener net.Listener, stop context.CancelFunc, serveErrors chan<- error) {
	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			serveErrors <- err
			stop()
		}
	}()
}
