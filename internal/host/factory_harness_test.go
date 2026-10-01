package host

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/project"
)

func TestFactoryHarnessRouteServesPin(t *testing.T) {
	exec := &factoryNoExec{}
	d := testFactoryDaemon(t, exec)
	d.Terminal.CodexHarnessVersion = "0.157.1"
	d.Terminal.CodexHarnessSHA256 = strings.Repeat("a", 64)
	d.Config.Image = "sha256:" + strings.Repeat("b", 64)
	r := httptest.NewRequest(http.MethodPost, "/factory-harness", strings.NewReader(`{}`))
	w := httptest.NewRecorder()
	d.ServeHTTP(w, r)
	if w.Code != http.StatusOK || exec.called {
		t.Fatal("harness pin not served cleanly", w.Code)
	}
	var pin project.FactoryHarnessPin
	if err := json.Unmarshal(w.Body.Bytes(), &pin); err != nil {
		t.Fatal(err)
	}
	if err := pin.Validate(); err != nil {
		t.Fatalf("served pin invalid: %v", err)
	}
	if pin.Harness != project.FactoryHarnessCodex || pin.Version != "0.157.1" ||
		pin.SHA256 != strings.Repeat("a", 64) || pin.Image != "sha256:"+strings.Repeat("b", 64) {
		t.Fatalf("pin = %+v", pin)
	}
}

func TestFactoryHarnessRouteRefusesUnpinned(t *testing.T) {
	exec := &factoryNoExec{}
	d := testFactoryDaemon(t, exec)
	d.Terminal.CodexHarnessSHA256 = strings.Repeat("a", 64)
	r := httptest.NewRequest(http.MethodPost, "/factory-harness", strings.NewReader(`{}`))
	w := httptest.NewRecorder()
	d.ServeHTTP(w, r)
	if w.Code != http.StatusServiceUnavailable || exec.called {
		t.Fatal("unversioned harness not refused", w.Code)
	}
}
