package host

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	projectexec "github.com/levitateos/sodaos/internal/host/project"
	"github.com/levitateos/sodaos/internal/host/terminal"
	"github.com/levitateos/sodaos/internal/identity"
	identityclient "github.com/levitateos/sodaos/internal/identity/client"
	"github.com/levitateos/sodaos/internal/project"
)

type factoryNoExec struct{ called bool }

func (n *factoryNoExec) Run(context.Context, []byte, string, ...string) ([]byte, error) {
	n.called = true
	return nil, errors.New("unexpected command")
}

func (n *factoryNoExec) RunReader(context.Context, io.Reader, string, ...string) ([]byte, error) {
	n.called = true
	return nil, errors.New("unexpected command")
}

func testFactoryDaemon(t *testing.T, exec terminal.Executor) *Daemon {
	t.Helper()
	dir := t.TempDir()
	if err := os.Chmod(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	service := &terminal.Service{Exec: exec}
	factory, err := projectexec.OpenFactory(dir, service, identityclient.New(filepath.Join(dir, "broker.sock")))
	if err != nil {
		t.Fatal(err)
	}
	return &Daemon{Terminal: service, Factory: factory, Identity: identityclient.New(filepath.Join(dir, "broker.sock"))}
}

func TestFactoryRoutesUnavailableWithoutRuntime(t *testing.T) {
	d := &Daemon{}
	body, _ := json.Marshal(project.FactoryInspect{Project: "p123456789012345678901234", ID: strings.Repeat("a", 32)})
	r := httptest.NewRequest(http.MethodPost, "/factory-inspect", bytes.NewReader(body))
	w := httptest.NewRecorder()
	d.ServeHTTP(w, r)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatal("factory route without runtime reported", w.Code)
	}
}

func TestFactoryInspectUnknownRunIsNotFound(t *testing.T) {
	exec := &factoryNoExec{}
	d := testFactoryDaemon(t, exec)
	body, _ := json.Marshal(project.FactoryInspect{Project: "p123456789012345678901234", ID: strings.Repeat("a", 32)})
	r := httptest.NewRequest(http.MethodPost, "/factory-inspect", bytes.NewReader(body))
	w := httptest.NewRecorder()
	d.ServeHTTP(w, r)
	if w.Code != http.StatusNotFound || exec.called {
		t.Fatal("unknown run was not a clean miss", w.Code)
	}
}

func TestFactoryStopValidationNeverExecutes(t *testing.T) {
	exec := &factoryNoExec{}
	d := testFactoryDaemon(t, exec)
	r := httptest.NewRequest(http.MethodPost, "/factory-stop", strings.NewReader(`{"project":"../../other","id":"short"}`))
	w := httptest.NewRecorder()
	d.ServeHTTP(w, r)
	if w.Code == http.StatusOK || exec.called {
		t.Fatal("untrusted run address reached native execution")
	}
}

func factoryCallback(t *testing.T, d *Daemon, action string, lease identity.Lease) error {
	t.Helper()
	body, err := json.Marshal(identity.DeliveryWire{Lease: lease})
	if err != nil {
		t.Fatal(err)
	}
	_, err = d.identityOperation(context.Background(), "/identity/"+action, bytes.NewReader(body))
	return err
}

func TestFactoryCallbackDeniesWithoutBinding(t *testing.T) {
	exec := &factoryNoExec{}
	d := testFactoryDaemon(t, exec)
	lease := identity.Lease{ProviderID: identity.Codex, Kind: identity.Factory, ExecutionID: strings.Repeat("a", 32)}
	for _, action := range []string{"validate", "stop", "finish", "start"} {
		if err := factoryCallback(t, d, action, lease); !errors.Is(err, identity.ErrDenied) || exec.called {
			t.Fatalf("unbound factory %s was not denied: %v", action, err)
		}
	}
}

func TestFactoryCallbackDeniesUnprovedProvider(t *testing.T) {
	exec := &factoryNoExec{}
	d := testFactoryDaemon(t, exec)
	lease := identity.Lease{ProviderID: identity.Muse, Kind: identity.Factory, ExecutionID: strings.Repeat("a", 32), Binding: &identity.Binding{Kind: identity.Factory, ID: strings.Repeat("a", 32)}}
	if err := factoryCallback(t, d, "validate", lease); !errors.Is(err, identity.ErrDenied) || exec.called {
		t.Fatal("unproved factory provider was not denied", err)
	}
}
