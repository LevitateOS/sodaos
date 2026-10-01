package main

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/levitateos/sodaos/internal/factory/control"
)

// operatorHTTPServer listens for private operator commands on a Unix socket
// outside Projects and the Fountain extension host. Only the configured OS
// peer UID may connect; its identity becomes the command principal, never a
// native human author.
func operatorHTTPServer(socket string, uid uint32, coordinator *control.Coordinator) (*http.Server, net.Listener, error) {
	if !filepath.IsAbs(socket) {
		return nil, nil, errors.New("operator socket must be an absolute path")
	}
	if coordinator == nil {
		return nil, nil, errors.New("operator endpoint requires its coordinator")
	}
	// The operator prepares the containing directory. A stale socket from an
	// earlier server generation is unlinked so restarts rebind the same path.
	if err := os.Remove(socket); err != nil && !os.IsNotExist(err) {
		return nil, nil, err
	}
	listener, err := net.Listen("unix", socket)
	if err != nil {
		return nil, nil, err
	}
	if err := os.Chmod(socket, 0o600); err != nil {
		_ = listener.Close()
		return nil, nil, err
	}
	gated, err := gateOperatorPeers(listener, uid)
	if err != nil {
		_ = listener.Close()
		return nil, nil, err
	}
	principal := fmt.Sprintf("os-uid:%d", uid)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		coordinator.OperatorHandler().ServeHTTP(w, r.WithContext(control.WithOperatorPrincipal(r.Context(), principal)))
	})
	return &http.Server{Handler: handler, ReadHeaderTimeout: 10 * time.Second, IdleTimeout: 12 * time.Minute, MaxHeaderBytes: 8192}, gated, nil
}
