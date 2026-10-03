package project

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/host/terminal"
	"github.com/levitateos/sodaos/internal/identity"
	identityclient "github.com/levitateos/sodaos/internal/identity/client"
	domain "github.com/levitateos/sodaos/internal/project"
)

type factoryFakeExec struct {
	calls int
	run   func(ctx context.Context, in []byte, command string, args ...string) ([]byte, error)
}

func (f *factoryFakeExec) Run(ctx context.Context, in []byte, command string, args ...string) ([]byte, error) {
	f.calls++
	return f.run(ctx, in, command, args...)
}

func (f *factoryFakeExec) RunReader(ctx context.Context, in io.Reader, command string, args ...string) ([]byte, error) {
	f.calls++
	body, _ := io.ReadAll(in)
	return f.run(ctx, body, command, args...)
}

func factoryTestRun() domain.FactoryRun {
	prompt := []byte("trivial task")
	return domain.FactoryRun{
		Deadline:     time.Now().Add(time.Hour),
		Actor:        1001,
		ID:           strings.Repeat("a", 32),
		Project:      "p" + strings.Repeat("b", 24),
		Role:         domain.RoleCoder,
		Preparation:  "f" + strings.Repeat("c", 24),
		Harness:      domain.FactoryHarnessCodex,
		HarnessVers:  "0.157.1",
		Assignment:   domain.FactoryPromptDigest(prompt),
		SourceCommit: strings.Repeat("e", 40),
		Connection:   "subscription",
	}
}

func openTestFactory(t *testing.T, pin string) *Factory {
	t.Helper()
	stateDir := filepath.Join(t.TempDir(), "runs")
	if err := os.Mkdir(stateDir, 0o700); err != nil {
		t.Fatal(err)
	}
	term := &terminal.Service{Exec: &factoryFakeExec{run: func(context.Context, []byte, string, ...string) ([]byte, error) {
		return nil, errors.New("native execution unavailable in unit test")
	}}, CodexHarnessSHA256: pin}
	broker := identityclient.New(filepath.Join(t.TempDir(), "dead.sock"))
	f, err := OpenFactory(stateDir, term, broker)
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func TestFactoryOpenRefusesSharedState(t *testing.T) {
	// Exercise the worker-like private umask: Mkdir alone would leave the
	// shared fixture private, so its mode is set explicitly below.
	oldMask := syscall.Umask(0o077)
	defer syscall.Umask(oldMask)
	term := &terminal.Service{}
	broker := identityclient.New(filepath.Join(t.TempDir(), "dead.sock"))
	if _, err := OpenFactory("relative/path", term, broker); err == nil {
		t.Fatal("relative receipt directory admitted")
	}
	shared := filepath.Join(t.TempDir(), "shared")
	if err := os.Mkdir(shared, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(shared, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenFactory(shared, term, broker); err == nil {
		t.Fatal("shared receipt directory admitted")
	}
	if _, err := OpenFactory(filepath.Join(t.TempDir(), "missing"), term, broker); err == nil {
		t.Fatal("missing receipt directory admitted")
	}
	if _, err := OpenFactory(t.TempDir(), nil, broker); err == nil {
		t.Fatal("missing terminal admitted")
	}
}

func TestFactoryLaunchRefusesBeforeNativeWork(t *testing.T) {
	pin := strings.Repeat("f", 64)
	f := openTestFactory(t, pin)
	ctx := t.Context()
	run := factoryTestRun()
	launch := domain.FactoryLaunch{Run: run, Prompt: []byte("trivial task"), HarnessSHA256: pin}
	launch.Prompt = []byte("changed")
	if _, err := f.Launch(ctx, launch); err == nil {
		t.Fatal("changed prompt launched")
	}
	launch.Prompt = []byte("trivial task")
	launch.HarnessSHA256 = strings.Repeat("0", 64)
	if _, err := f.Launch(ctx, launch); err == nil {
		t.Fatal("unproved harness launched")
	}
	launch.HarnessSHA256 = pin
	launch.Run.Deadline = time.Now().Add(-time.Minute)
	if _, err := f.Launch(ctx, launch); err == nil {
		t.Fatal("expired run launched")
	}
	launch.Run.Deadline = time.Now().Add(4 * time.Hour)
	if _, err := f.Launch(ctx, launch); err == nil {
		t.Fatal("unbounded run launched")
	}
}

func TestFactoryStopTombstoneBarsLaterLaunch(t *testing.T) {
	pin := strings.Repeat("f", 64)
	f := openTestFactory(t, pin)
	ctx := t.Context()
	run := factoryTestRun()
	stopped, err := f.Stop(ctx, domain.FactoryStop{Project: run.Project, ID: run.ID})
	if err != nil {
		t.Fatal(err)
	}
	// The test broker socket is dead, so the broker fence cannot confirm;
	// the host tombstone still bars the run.
	if stopped.Phase != domain.FactoryStopped {
		t.Fatal("stop did not tombstone", stopped.Phase)
	}
	launch := domain.FactoryLaunch{Run: run, Prompt: []byte("trivial task"), HarnessSHA256: pin}
	late, err := f.Launch(ctx, launch)
	if err != nil {
		t.Fatal(err)
	}
	if late.Phase != domain.FactoryStopped {
		t.Fatal("launch escaped its stop tombstone", late.Phase)
	}
	again, err := f.Stop(ctx, domain.FactoryStop{Project: run.Project, ID: run.ID})
	if err != nil || again.Phase != domain.FactoryStopped {
		t.Fatal("stop tombstone not idempotent", again.Phase, err)
	}
	observed, err := f.Inspect(ctx, domain.FactoryInspect{Project: run.Project, ID: run.ID})
	if err != nil || observed.Phase != domain.FactoryStopped || observed.Live {
		t.Fatal("tombstone misreported", observed, err)
	}
}

func TestFactoryStopWithoutBrokerStaysUncertain(t *testing.T) {
	pin := strings.Repeat("f", 64)
	f := openTestFactory(t, pin)
	ctx := t.Context()
	run := factoryTestRun()
	// Seed an approved receipt directly: the dead broker cannot acquire.
	seed := factoryReceipt{Run: run, Phase: domain.FactoryApproved}
	if err := f.storeReceipt(seed); err != nil {
		t.Fatal(err)
	}
	stopped, err := f.Stop(ctx, domain.FactoryStop{Project: run.Project, ID: run.ID})
	if err != nil {
		t.Fatal(err)
	}
	// Native retirement cannot confirm against the fake executor and the
	// broker fence cannot confirm over the dead socket: the run must stay
	// fenced rather than report a clean stop.
	if stopped.Phase != domain.FactoryUncertain {
		t.Fatal("unconfirmed stop reported clean", stopped.Phase)
	}
	if stopped.Reason != "stop-uncertain" {
		t.Fatal("stop reason lost", stopped.Reason)
	}
}

func TestFactoryInspectMissingIsNotFound(t *testing.T) {
	f := openTestFactory(t, strings.Repeat("f", 64))
	_, err := f.Inspect(t.Context(), domain.FactoryInspect{Project: "p" + strings.Repeat("b", 24), ID: strings.Repeat("a", 32)})
	if !errors.Is(err, identity.ErrNotFound) {
		t.Fatal("missing run misreported", err)
	}
}

// factoryBrokerStub records broker cleanup calls over a Unix socket.
type factoryBrokerStub struct {
	mu   sync.Mutex
	hits []string
}

func (s *factoryBrokerStub) handler(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	s.hits = append(s.hits, r.URL.Path)
	s.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{}`))
}

func (s *factoryBrokerStub) count(path string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	for _, hit := range s.hits {
		if hit == path {
			n++
		}
	}
	return n
}

type factoryExitError int

func (e factoryExitError) Error() string { return "native exit status" }

func (e factoryExitError) ExitCode() int { return int(e) }

// TestFactoryTimeoutCleanupSurvivesExpiredParent is the N-RUN1 host-side
// regression: when the run deadline or the caller already expired the
// parent, retirement must still stop the boundary, reconcile broker
// custody and record the outcome under a usable bounded context.
func TestFactoryTimeoutCleanupSurvivesExpiredParent(t *testing.T) {
	pin := strings.Repeat("f", 64)
	stub := &factoryBrokerStub{}
	sock := filepath.Join(t.TempDir(), "broker.sock")
	listener, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatal(err)
	}
	server := &http.Server{Handler: http.HandlerFunc(stub.handler)}
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() { _ = server.Close() })
	exec := &factoryFakeExec{run: func(_ context.Context, _ []byte, command string, args ...string) ([]byte, error) {
		joined := command + " " + strings.Join(args, " ")
		if strings.Contains(joined, "systemctl") {
			for _, arg := range args {
				if arg == "show" {
					return []byte("ActiveState=inactive\nInvocationID=" + strings.Repeat("e", 32) + "\n"), nil
				}
			}
			return nil, nil
		}
		if strings.Contains(joined, "container") && strings.Contains(joined, "exists") {
			return nil, factoryExitError(1)
		}
		return nil, errors.New("native execution unavailable in unit test")
	}}
	stateDir := filepath.Join(t.TempDir(), "runs")
	if err := os.Mkdir(stateDir, 0o700); err != nil {
		t.Fatal(err)
	}
	term := &terminal.Service{Exec: exec, CodexHarnessSHA256: pin}
	f, err := OpenFactory(stateDir, term, identityclient.New(sock))
	if err != nil {
		t.Fatal(err)
	}
	run := factoryTestRun()
	_, runDir, _, _ := domain.FactoryRunPaths(domain.RoleCoder, run.Preparation, run.ID)
	binding := identity.Binding{
		Kind: identity.Factory, ID: run.ID, Project: strings.Repeat("d", 64),
		Login: domain.RoleCoder, UID: 2001, GID: 2001, Scope: domain.FactoryScopeCodex,
		InvocationID: strings.Repeat("e", 32), CredentialRoot: runDir, Generation: 3, ChildID: run.Preparation,
	}
	lease := identity.Lease{
		ProviderID: identity.Codex, ID: "lease-1", ConnectionID: "subscription",
		Generation: 3, ActorID: 1001, ProjectID: run.Project, ExecutionID: run.ID,
		Kind: identity.Factory, Binding: &binding,
	}
	seed := factoryReceipt{Run: run, Lease: &lease, Binding: &binding, Generation: 3, Phase: domain.FactoryRunning, Started: true, Delivered: true}
	if err := f.storeReceipt(seed); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	state, err := f.stopTimedOut(ctx, seed, lease, binding)
	if err != nil {
		t.Fatalf("expired parent lost the timeout outcome: %v", err)
	}
	if state.Phase != domain.FactoryUncertain || state.Reason != "deadline-exceeded" {
		t.Fatalf("timeout outcome misrecorded: %+v", state)
	}
	if exec.calls == 0 {
		t.Fatal("native retirement never ran")
	}
	if stub.count("/reconcile-lease") == 0 || stub.count("/execution/close") == 0 {
		t.Fatal("broker custody cleanup never ran")
	}
	observed, err := f.Inspect(context.Background(), domain.FactoryInspect{Project: run.Project, ID: run.ID})
	if err != nil || observed.Phase != domain.FactoryUncertain {
		t.Fatalf("timeout receipt not durable: %+v %v", observed, err)
	}
}
