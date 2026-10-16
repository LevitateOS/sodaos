package web

import (
	"net/http"
	"path/filepath"

	"github.com/levitateos/sodaos/internal/config"
)

// Fixed Factory MCP IPC location (FW38).
//
// The dedicated Factory-only listener serves the fourteen-tool
// catalog at factory-ipc/mcp.sock under the service private data
// root (the parent of the publication workspace root). The IPC
// directory contains only this socket: no secrets, no extension,
// operator or host-operation sockets. Clients cannot choose a
// socket, URL or listener; there is no flag, request input or
// configuration key that moves it. Directory mounting lets
// reconnects observe a recreated socket inode after service
// restart; a single socket-file bind mount does not satisfy the
// recovery contract.
const (
	// FactoryMCPIPCDir is the fixed IPC directory relative name.
	FactoryMCPIPCDir = "factory-ipc"
	// FactoryMCPSocketName is the fixed socket file name.
	FactoryMCPSocketName = "mcp.sock"
)

// FactoryMCPDataRoot resolves the service private data root: the
// parent of the publication workspace root.
func FactoryMCPDataRoot(c config.Config) string {
	return filepath.Dir(publicationRoot(c))
}

// FactoryMCPSocketPath resolves the fixed Factory listener socket.
// Operator, extension and test deployments share this derivation,
// so the path is identical wherever the data root is identical.
func FactoryMCPSocketPath(c config.Config) string {
	return filepath.Join(FactoryMCPDataRoot(c), FactoryMCPIPCDir, FactoryMCPSocketName)
}

// FactoryMCPHandler exposes the catalog handler for the dedicated
// private listener. Only that listener serves it; the public
// dashboard mux never mounts it.
func (s *Server) FactoryMCPHandler() http.Handler {
	if s == nil || s.API == nil {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "factory MCP unavailable", http.StatusServiceUnavailable)
		})
	}
	return s.API.FactoryMCPHandler()
}
