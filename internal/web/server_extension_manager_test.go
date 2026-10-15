package web

import (
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	extensions "forgejo.org/extension-sdk"
	"github.com/levitateos/sodaos/internal/config"
)

func TestExtensionManagerAvailableWithoutFactoryCredentials(t *testing.T) {
	dir, err := os.MkdirTemp("", "soda-manager-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	socket := filepath.Join(dir, "manager.sock")
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/extensions/manager/state" || r.Header.Get("Authorization") != "" {
			t.Error("manager control must use the service callback without actor credentials")
		}
		_, _ = io.WriteString(w, `{"admission_open":false}`)
	})}
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() { _ = server.Close() })
	t.Setenv(extensions.ServiceCallbackEnv, socket)
	t.Setenv(extensions.ServiceCallbackPeerEnv, strconv.Itoa(os.Geteuid()))
	s := New(config.Config{}, nil)
	r := httptest.NewRequest(http.MethodPost, "/operator/extensions/state", strings.NewReader(`{"package_id":"tailnet"}`))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.API.OperatorHandler().ServeHTTP(w, r)
	if w.Code != http.StatusOK || w.Body.String() != `{"admission_open":false}` {
		t.Fatalf("ordinary activation lost manager control: %d %s", w.Code, w.Body.String())
	}
}

func TestExtensionManagerRequiresPinnedServiceCallback(t *testing.T) {
	for _, input := range []struct{ socket, peer string }{
		{}, {"relative.sock", "1000"}, {"/ipc/host.sock", ""}, {"/ipc/host.sock", "wrong"}, {"/ipc/host.sock", "4294967296"},
	} {
		t.Run(input.socket+"/"+input.peer, func(t *testing.T) {
			t.Setenv(extensions.ServiceCallbackEnv, input.socket)
			t.Setenv(extensions.ServiceCallbackPeerEnv, input.peer)
			if s := New(config.Config{}, nil); s.API.ExtensionManager != nil {
				t.Fatal("unpinned service callback enabled manager control")
			}
		})
	}
}
