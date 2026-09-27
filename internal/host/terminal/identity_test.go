package terminal

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/identity"
)

type identityExecutor struct {
	container string
	stopped   bool
	existsErr error
	requests  []identityRequest
	calls     [][]string
	response  identity.Delivery
}

func (e *identityExecutor) RunReader(_ context.Context, body io.Reader, command string, args ...string) ([]byte, error) {
	e.calls = append(e.calls, append([]string{command}, args...))
	_, err := io.Copy(io.Discard, body)
	return nil, err
}

func (e *identityExecutor) Run(_ context.Context, body []byte, command string, args ...string) ([]byte, error) {
	e.calls = append(e.calls, append([]string{command}, args...))
	if strings.Contains(strings.Join(args, " "), "container exists") {
		return nil, e.existsErr
	}
	if strings.Contains(strings.Join(args, " "), " inspect ") {
		return []byte(`{"id":"` + e.container + `","running":` + fmt.Sprint(!e.stopped) + `,"project":"p` + strings.Repeat("a", 24) + `","owner":"41","privileged":false,"userns":"private","mappings":{"UidMap":["0:100000:262144"],"GidMap":["0:100000:262144"]}}`), nil
	}
	if len(body) == 0 {
		return nil, nil
	}
	var request identityRequest
	if err := json.Unmarshal(body, &request); err != nil {
		return nil, err
	}
	e.requests = append(e.requests, request)
	result := e.response
	if result.Lease.ID == "" && request.Action != "lookup" {
		result.Lease = request.Delivery.Lease
		if request.Action == "prepare" {
			result.Lease.Binding = &identity.Binding{Kind: identity.Terminal, ID: result.Lease.ExecutionID, Project: e.container, Login: request.Login, Generation: result.Lease.Generation}
		}
	}
	return json.Marshal(identity.DeliveryWire(result))
}

func testTerminalLease() identity.Lease {
	return identity.Lease{ID: "lease-one", ConnectionID: "connection-one", Generation: 3, ActorID: 41, ProjectID: "p" + strings.Repeat("a", 24), ExecutionID: strings.Repeat("b", 32), Kind: identity.Terminal, Deadline: time.Now().Add(time.Hour), Binding: &identity.Binding{Kind: identity.Terminal, ID: strings.Repeat("b", 32), Project: strings.Repeat("c", 64), Login: "soda-tester", Generation: 3}}
}

func TestPrepareIdentityRecordsNativeContainer(t *testing.T) {
	lease := testTerminalLease()
	lease.Binding = nil
	exec := &identityExecutor{container: strings.Repeat("c", 64)}
	service := Service{Exec: exec}
	binding, err := service.PrepareIdentity(context.Background(), lease, "soda-tester", strings.Repeat("d", 64), 100, 30)
	if err != nil || binding.Project != exec.container || binding.ID != lease.ExecutionID || binding.Generation != lease.Generation {
		t.Fatalf("native binding: %v", err)
	}
	if len(exec.requests) != 1 || exec.requests[0].Container != exec.container || len(exec.requests[0].Delivery.Credential) != 0 {
		t.Fatal("reservation did not bind exact native target without credentials")
	}
}

func TestIdentityRefusesReplacementBeforeGuestOperation(t *testing.T) {
	exec := &identityExecutor{container: strings.Repeat("e", 64)}
	service := Service{Exec: exec}
	_, err := service.Identity(context.Background(), "stop", identity.Delivery{Lease: testTerminalLease()})
	if !errors.Is(err, identity.ErrStale) || len(exec.requests) != 0 {
		t.Fatal("replacement received guest operation")
	}
}

func TestIdentityStartDeliversOnlyOverStdin(t *testing.T) {
	exec := &identityExecutor{container: strings.Repeat("c", 64)}
	harness := t.TempDir()
	if err := os.Mkdir(filepath.Join(harness, "bin"), 0o700); err != nil {
		t.Fatal(err)
	}
	binary := []byte("soda-synthetic-harness")
	if err := os.WriteFile(filepath.Join(harness, "bin", "codex"), binary, 0o700); err != nil {
		t.Fatal(err)
	}
	service := Service{Exec: exec, CodexHarness: harness, CodexHarnessSHA256: fmt.Sprintf("%x", sha256.Sum256(binary))}
	secret := []byte(`{"refresh_token":"soda-synthetic-secret"}`)
	result, err := service.Identity(context.Background(), "start", identity.Delivery{Lease: testTerminalLease(), Credential: secret})
	if err != nil || len(result.Credential) != 0 {
		t.Fatalf("start: %v", err)
	}
	if len(exec.requests) != 2 || exec.requests[0].Action != "stage" || exec.requests[1].Action != "start" || string(exec.requests[0].Delivery.Credential) != string(secret) {
		t.Fatal("missing fixed native seed/start operations")
	}
	for _, call := range exec.calls {
		if strings.Contains(strings.Join(call, " "), "soda-synthetic-secret") {
			t.Fatal("credential in arguments")
		}
	}
}

func TestManagedEndRejectsWrongNativeReceipt(t *testing.T) {
	lease := testTerminalLease()
	lease.Binding.Project = strings.Repeat("e", 64)
	exec := &identityExecutor{container: strings.Repeat("c", 64), response: identity.Delivery{Lease: lease}}
	called := false
	service := Service{Exec: exec, EndIdentity: func(context.Context, int64, string) error { called = true; return nil }}
	request := TerminalRequest{Action: "end", ID: lease.ExecutionID, Login: "soda-tester", Identity: 41}
	if !errors.Is(service.managedEnd(context.Background(), exec.container, request), identity.ErrStale) || called {
		t.Fatal("wrong container receipt ended a broker lease")
	}
}

func TestTerminalGenerationAndDeadlineAdmission(t *testing.T) {
	lease := testTerminalLease()
	lease.Binding.Generation++
	if terminalLease(lease, false) {
		t.Fatal("changed connection generation admitted")
	}
	lease = testTerminalLease()
	lease.Binding = nil
	lease.Deadline = time.Now().Add(13 * time.Hour)
	if terminalLease(lease, true) {
		t.Fatal("terminal lifetime beyond twelve hours admitted")
	}
}

type nativeExit int

func (e nativeExit) Error() string { return "native failure" }
func (e nativeExit) ExitCode() int { return int(e) }
func TestIdentityStopObservesExactAbsentOrStoppedContainer(t *testing.T) {
	for _, test := range []struct {
		name      string
		exit      error
		stopped   bool
		uncertain bool
	}{
		{name: "absent", exit: nativeExit(1)}, {name: "stopped", stopped: true}, {name: "engine failure", exit: nativeExit(125), uncertain: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			lease := testTerminalLease()
			exec := &identityExecutor{container: lease.Binding.Project, stopped: test.stopped, existsErr: test.exit}
			_, err := (&Service{Exec: exec}).Identity(context.Background(), "stop", identity.Delivery{Lease: lease})
			if test.uncertain {
				if !errors.Is(err, identity.ErrUncertain) {
					t.Fatal(err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
			if len(exec.requests) != 0 {
				t.Fatal("terminated resource received guest mutation")
			}
			for _, args := range exec.calls {
				if args[len(args)-1] != lease.Binding.Project {
					t.Fatal("stop resolved a replacement name")
				}
			}
		})
	}
}
