package acceptance

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMalformedQMPIsNotReadiness(t *testing.T) {
	for _, body := range []string{`{"id":"wanted"}`, `{"id":"wanted","return":`, `{"event":"ignored"}`} {
		if err := decodeQMPResponse(json.NewDecoder(strings.NewReader(body)), "wanted", nil); err == nil {
			t.Fatalf("accepted %s", body)
		}
	}
	if err := decodeQMPResponse(json.NewDecoder(strings.NewReader(`{"event":"RESET"}{"id":"wanted","return":{}}`)), "wanted", nil); err != nil {
		t.Fatal(err)
	}
}
func TestCancelledQMPCannotDial(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := (QMPClient{Socket: "/nonexistent"}).Execute(ctx, "query-status", "test", nil, nil); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}
func TestBadVMPreflightCreatesNoDisk(t *testing.T) {
	e := fixtureEvidence(t)
	work := filepath.Join(t.TempDir(), "new-work")
	vm, err := LaunchVM(context.Background(), VMConfig{Name: "soda-native-fixture", Architecture: "not-native", Work: work}, e)
	if vm != nil || err == nil {
		t.Fatal("invalid native target accepted")
	}
	if _, err = os.Stat(work); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("created VM state before preflight")
	}
}
func TestRemoteUnknownPhaseFailsBeforeSSH(t *testing.T) {
	path := filepath.Join(t.TempDir(), "request.json")
	revision := strings.Repeat("a", 40)
	raw, _ := json.Marshal(RemoteRequest{Revision: revision, Architecture: "x86_64", Target: "fixture", Work: "/new/work", Phase: "publish"})
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	// No identity/pins are supplied: local request validation must win first.
	_, err := (Remote{}).NativePhase(path, revision, "x86_64", "fixture")
	if err == nil || !strings.Contains(err.Error(), "unknown native phase") {
		t.Fatal(err)
	}
}
