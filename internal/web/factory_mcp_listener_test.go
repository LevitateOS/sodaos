package web

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/config"
)

// FW38 fixes the Factory listener location: factory-ipc/mcp.sock
// under the private data root, identical for every deployment with
// the same data root, and never mounted on the public mux.
func TestFactoryMCPSocketPathFixed(t *testing.T) {
	path := FactoryMCPSocketPath(config.Config{})
	if !filepath.IsAbs(path) {
		t.Fatalf("factory socket path %q is not absolute", path)
	}
	if !strings.HasSuffix(path, filepath.Join(FactoryMCPIPCDir, FactoryMCPSocketName)) {
		t.Fatalf("factory socket path %q is not the fixed relative location", path)
	}
	if FactoryMCPDataRoot(config.Config{}) != filepath.Dir(filepath.Dir(path)) {
		t.Fatal("factory data root does not own the socket path")
	}
}

func TestFactoryMCPHandlerUnavailableWithoutAPI(t *testing.T) {
	var server *Server
	recorder := httptest.NewRecorder()
	server.FactoryMCPHandler().ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/mcp", nil))
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("nil facade serves %d, want 503", recorder.Code)
	}
}
