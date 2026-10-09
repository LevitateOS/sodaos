package control

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/identity"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
)

type stubHost struct {
	stop     func(project.FactoryStop) (project.FactoryState, error)
	inspect  func(project.FactoryInspect) (project.FactoryState, error)
	takeover func(project.FactoryTakeover) (project.TakeoverResult, error)
	launch   func(project.FactoryLaunch) (project.FactoryState, error)
	harness  func() (project.FactoryHarnessPin, error)
	calls    int
}

func (s *stubHost) FactoryLaunch(ctx context.Context, in project.FactoryLaunch) (project.FactoryState, error) {
	if s.launch == nil {
		return project.FactoryState{}, errors.New("unexpected host launch")
	}
	return s.launch(in)
}

func (s *stubHost) FactoryStop(ctx context.Context, in project.FactoryStop) (project.FactoryState, error) {
	s.calls++
	return s.stop(in)
}

func (s *stubHost) FactoryInspect(ctx context.Context, in project.FactoryInspect) (project.FactoryState, error) {
	if s.inspect == nil {
		return project.FactoryState{}, errors.New("unexpected host inspect")
	}
	return s.inspect(in)
}

func (s *stubHost) FactoryTakeover(ctx context.Context, in project.FactoryTakeover) (project.TakeoverResult, error) {
	if s.takeover == nil {
		return project.TakeoverResult{}, errors.New("unexpected host takeover")
	}
	return s.takeover(in)
}

func (s *stubHost) FactoryExport(ctx context.Context, in project.FactoryExport) (project.FactoryExportState, error) {
	return project.FactoryExportState{}, errors.New("unexpected host export")
}

func (s *stubHost) FactoryHarness(ctx context.Context, _ string) (project.FactoryHarnessPin, error) {
	if s.harness == nil {
		return project.FactoryHarnessPin{}, errors.New("unexpected host harness query")
	}
	return s.harness()
}

func (*stubHost) PrepareCandidate(context.Context, project.FactoryCandidate) (project.PrepareState, error) {
	return project.PrepareState{}, errors.New("candidate preparation unavailable in coordinator stub")
}

func (*stubHost) InspectPreparation(context.Context, project.PrepareInspect) (project.PrepareState, error) {
	return project.PrepareState{}, errors.New("preparation inspection unavailable in coordinator stub")
}

func (*stubHost) StopPreparation(context.Context, project.PrepareStop) (project.PrepareState, error) {
	return project.PrepareState{}, errors.New("preparation stop unavailable in coordinator stub")
}

func (*stubHost) ReadPreparationContext(context.Context, project.FactoryPreparationContextRequest, string) (project.FactoryPreparationContext, error) {
	return project.FactoryPreparationContext{}, errors.New("prepared repository context unavailable in coordinator stub")
}

type stubBroker struct {
	get   func(kind, id string) (identity.Execution, error)
	close func(kind, id string) error
}

func (s *stubBroker) GetExecution(ctx context.Context, kind, id string) (identity.Execution, error) {
	return s.get(kind, id)
}

func (s *stubBroker) CloseExecution(ctx context.Context, kind, id string) error {
	return s.close(kind, id)
}

func coordinatorFixture(t *testing.T, host HostFactory, broker BrokerExecution) *Coordinator {
	t.Helper()
	db, _ := postgresFixture(t, nil)
	if host == nil {
		host = &stubHost{stop: func(project.FactoryStop) (project.FactoryState, error) {
			return project.FactoryState{}, errors.New("unexpected host call")
		}}
	}
	if broker == nil {
		broker = &stubBroker{get: func(string, string) (identity.Execution, error) {
			return identity.Execution{}, errors.New("unexpected broker call")
		}, close: func(string, string) error { return errors.New("unexpected broker call") }}
	}
	return NewCoordinator(db, host, broker)
}

func recordRun(t *testing.T, c *Coordinator, mutate func(*factory.Run)) factory.Run {
	t.Helper()
	now := time.Now()
	r := factory.Run{ID: factory.NewID(), ProjectID: "p123456789012345678901234", Role: "coder", InputSHA: strings.Repeat("a", 40), Started: now, Deadline: now.Add(time.Hour), Image: "sha256:" + strings.Repeat("b", 64), Harness: "codex-0.157.1", Model: "test"}
	if mutate != nil {
		mutate(&r)
	}
	if err := c.Store.RecordFactoryRun(context.Background(), r); err != nil {
		t.Fatal(err)
	}
	return r
}

func stopCommand(runID string) factory.Command {
	return factory.Command{ID: factory.NewID(), Type: factory.CommandStop, Target: runID, Principal: "os-uid:0", Digest: factory.CommandDigest(factory.CommandStop, runID)}
}

func TestStatusRedactsCredentialPaths(t *testing.T) {
	c := coordinatorFixture(t, nil, nil)
	r := recordRun(t, c, func(r *factory.Run) {
		r.IdentityLeaseID, r.IdentityGeneration, r.CredentialDelegated = "lease", 3, true
		r.IdentityBinding = &identity.Binding{Kind: identity.Factory, ID: r.ID, Generation: 3, CredentialRoot: "/home/coder/checkouts/p/.soda-home/runs/" + r.ID}
	})
	runs, err := c.Status(context.Background(), r.ID)
	if err != nil || len(runs) != 1 {
		t.Fatal(runs, err)
	}
	if runs[0].IdentityBinding == nil || runs[0].IdentityBinding.CredentialRoot != "" {
		t.Fatal("status exposed a credential-adjacent host path")
	}
	if _, err = c.Status(context.Background(), factory.NewID()); !errors.Is(err, ErrNotFound) {
		t.Fatal("unknown run reported status")
	}
}

func TestStopSettlesAndReplays(t *testing.T) {
	host := &stubHost{}
	broker := &stubBroker{}
	c := coordinatorFixture(t, host, broker)
	r := recordRun(t, c, nil)
	binding := identity.Binding{Kind: identity.Factory, ID: r.ID, Generation: 3, CredentialRoot: "/home/coder/runs/" + r.ID}
	host.stop = func(in project.FactoryStop) (project.FactoryState, error) {
		if in.ID != r.ID || in.Project != r.ProjectID {
			return project.FactoryState{}, errors.New("stop addressed another run")
		}
		return project.FactoryState{ID: r.ID, Project: r.ProjectID, Phase: project.FactoryStopped, Retirement: "confirmed", Reason: "stopped", LeaseID: "lease", Generation: 3, CredentialReturned: true}, nil
	}
	broker.close = func(kind, id string) error {
		if kind != identity.Factory || id != r.ID {
			return errors.New("close addressed another execution")
		}
		return nil
	}
	broker.get = func(kind, id string) (identity.Execution, error) {
		return identity.Execution{Kind: identity.Factory, ExecutionID: r.ID, State: identity.ExecutionTerminal, Digest: "digest", LeaseID: "lease", Binding: &binding}, nil
	}
	cmd := stopCommand(r.ID)
	receipt, err := c.Stop(context.Background(), cmd)
	if err != nil || !receipt.Confirmed || receipt.Uncertain || receipt.Outcome != string(factory.Cancelled) {
		t.Fatal(receipt, err)
	}
	settled, err := c.Store.FactoryRun(context.Background(), r.ID)
	if err != nil || !settled.Reconciled || settled.Outcome != factory.Cancelled || !settled.CredentialDelegated || !settled.CredentialReturned || settled.IdentityBinding == nil {
		t.Fatal(settled, err)
	}
	replay, err := c.Stop(context.Background(), cmd)
	if err != nil || replay != receipt || host.calls != 1 {
		t.Fatal("duplicate command re-executed instead of replaying", replay, err)
	}
	again, err := c.Stop(context.Background(), stopCommand(r.ID))
	if err != nil || !again.Confirmed || host.calls != 1 {
		t.Fatal("settled stop re-executed its host call", again, err)
	}
}

func TestStopReplaysDurableOutcome(t *testing.T) {
	host := &stubHost{}
	c := coordinatorFixture(t, host, &stubBroker{
		get:   func(string, string) (identity.Execution, error) { return identity.Execution{}, identity.ErrNotFound },
		close: func(string, string) error { return nil },
	})
	r := recordRun(t, c, nil)
	host.stop = func(project.FactoryStop) (project.FactoryState, error) {
		return project.FactoryState{ID: r.ID, Phase: project.FactoryStopped, Retirement: "confirmed", Reason: "stop-before-start"}, nil
	}
	cmd := stopCommand(r.ID)
	first, err := c.Stop(context.Background(), cmd)
	if err != nil || !first.Confirmed {
		t.Fatal(first, err)
	}
	second, err := c.Stop(context.Background(), cmd)
	if err != nil || second != first || host.calls != 1 {
		t.Fatal("duplicate command re-executed instead of replaying", second, err)
	}
	other := recordRun(t, c, nil)
	changed := factory.Command{ID: cmd.ID, Type: factory.CommandStop, Target: other.ID, Principal: "os-uid:0", Digest: factory.CommandDigest(factory.CommandStop, other.ID)}
	if _, err = c.Stop(context.Background(), changed); !errors.Is(err, store.ErrCommandConflict) {
		t.Fatal("command identity reused for different content", err)
	}
}

func TestStopUnknownRunRecordsNothing(t *testing.T) {
	host := &stubHost{}
	c := coordinatorFixture(t, host, nil)
	cmd := stopCommand(factory.NewID())
	if _, err := c.Stop(context.Background(), cmd); !errors.Is(err, ErrNotFound) {
		t.Fatal("unknown run stopped")
	}
	if host.calls != 0 {
		t.Fatal("unknown run reached the host")
	}
	if _, err := c.Store.FactoryCommand(context.Background(), cmd.ID); !errors.Is(err, store.ErrNotFound) {
		t.Fatal("unknown run recorded a command")
	}
}

func TestStopFencesUncertainRetirement(t *testing.T) {
	host := &stubHost{stop: func(project.FactoryStop) (project.FactoryState, error) {
		return project.FactoryState{Phase: project.FactoryStopped, Retirement: "uncertain", Reason: "stop-uncertain"}, nil
	}}
	c := coordinatorFixture(t, host, &stubBroker{
		get:   func(string, string) (identity.Execution, error) { return identity.Execution{}, identity.ErrNotFound },
		close: func(string, string) error { return nil },
	})
	r := recordRun(t, c, nil)
	receipt, err := c.Stop(context.Background(), stopCommand(r.ID))
	if err != nil || !receipt.Uncertain || receipt.Confirmed {
		t.Fatal(receipt, err)
	}
	current, err := c.Store.FactoryRun(context.Background(), r.ID)
	if err != nil || current.Reconciled || current.Outcome != "" {
		t.Fatal("uncertain retirement settled the run", current, err)
	}
}

func TestReconcileSettlesCompletedRun(t *testing.T) {
	host := &stubHost{}
	c := coordinatorFixture(t, host, &stubBroker{
		get:   func(string, string) (identity.Execution, error) { return identity.Execution{}, identity.ErrNotFound },
		close: func(string, string) error { return nil },
	})
	r := recordRun(t, c, nil)
	host.stop = func(project.FactoryStop) (project.FactoryState, error) {
		return project.FactoryState{ID: r.ID, Phase: project.FactoryCompleted, Retirement: "confirmed", Output: "candidate ready"}, nil
	}
	cmd := factory.Command{ID: factory.NewID(), Type: factory.CommandReconcile, Principal: "os-uid:0", Digest: factory.CommandDigest(factory.CommandReconcile, "")}
	receipt, err := c.Reconcile(context.Background(), cmd)
	if err != nil || len(receipt.Settled) != 1 || len(receipt.Fenced) != 0 {
		t.Fatal(receipt, err)
	}
	settled, err := c.Store.FactoryRun(context.Background(), r.ID)
	if err != nil || !settled.Reconciled || settled.Outcome != factory.Succeeded || settled.Summary != "candidate ready" {
		t.Fatal(settled, err)
	}
}

func TestReconcileFencesBrokerFailure(t *testing.T) {
	host := &stubHost{stop: func(project.FactoryStop) (project.FactoryState, error) {
		return project.FactoryState{Phase: project.FactoryCompleted, Retirement: "confirmed"}, nil
	}}
	c := coordinatorFixture(t, host, &stubBroker{
		get:   func(string, string) (identity.Execution, error) { return identity.Execution{}, identity.ErrNotFound },
		close: func(string, string) error { return errors.New("broker unavailable") },
	})
	r := recordRun(t, c, nil)
	cmd := factory.Command{ID: factory.NewID(), Type: factory.CommandReconcile, Principal: "os-uid:0", Digest: factory.CommandDigest(factory.CommandReconcile, "")}
	receipt, err := c.Reconcile(context.Background(), cmd)
	if err != nil || len(receipt.Settled) != 0 || len(receipt.Fenced) != 1 || receipt.Fenced[0].ID != r.ID {
		t.Fatal(receipt, err)
	}
	current, err := c.Store.FactoryRun(context.Background(), r.ID)
	if err != nil || current.Reconciled {
		t.Fatal("broker failure settled the run", current, err)
	}
}

func TestStartTakesExclusiveOwnership(t *testing.T) {
	c := coordinatorFixture(t, nil, nil)
	lock := filepath.Join(t.TempDir(), "factory-coordinator.lock")
	ctx := context.Background()
	if err := c.Start(ctx, lock); err != nil {
		t.Fatal(err)
	}
	second := coordinatorFixture(t, nil, nil)
	second.Store = c.Store
	if err := second.Start(ctx, lock); err == nil {
		t.Fatal("second coordinator owned the database")
	}
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
	if err := second.Start(ctx, lock); err != nil {
		t.Fatal(err)
	}
	_ = second.Close()
}

func TestCloseRetainsCoordinatorOwnershipUntilAdmittedOperatorStops(t *testing.T) {
	entered := make(chan struct{})
	release := make(chan struct{})
	var enteredOnce sync.Once
	var releaseOnce sync.Once
	defer releaseOnce.Do(func() { close(release) })
	host := &stubHost{stop: func(in project.FactoryStop) (project.FactoryState, error) {
		enteredOnce.Do(func() { close(entered) })
		<-release
		return project.FactoryState{ID: in.ID, Phase: project.FactoryStopped, Retirement: "confirmed", Reason: "stopped"}, nil
	}}
	broker := &stubBroker{
		get:   func(string, string) (identity.Execution, error) { return identity.Execution{}, identity.ErrNotFound },
		close: func(string, string) error { return nil },
	}
	c := coordinatorFixture(t, host, broker)
	lock := filepath.Join(t.TempDir(), "factory-coordinator.lock")
	if err := c.Start(context.Background(), lock); err != nil {
		t.Fatal(err)
	}
	run := recordRun(t, c, nil)
	server := &http.Server{Handler: c.AdmitHTTP(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.OperatorHandler().ServeHTTP(w, r.WithContext(WithOperatorPrincipal(r.Context(), "os-uid:0")))
	}))}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	serveDone := make(chan error, 1)
	go func() { serveDone <- server.Serve(listener) }()
	requestDone := make(chan struct{})
	go func() {
		defer close(requestDone)
		body := fmt.Sprintf(`{"type":"stop","command_id":%q,"target":%q}`, factory.NewID(), run.ID)
		response, requestErr := http.Post("http://"+listener.Addr().String()+OperatorPath, "application/json", strings.NewReader(body))
		if requestErr == nil {
			_ = response.Body.Close()
		}
	}()
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		releaseOnce.Do(func() { close(release) })
		t.Fatal("operator stop did not reach the held host callback")
	}
	c.StopAdmission()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	err = server.Shutdown(shutdownCtx)
	cancel()
	if err == nil {
		t.Fatal("shutdown completed while the admitted operator callback was held")
	}
	if err := server.Close(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		t.Fatal(err)
	}

	closed := make(chan error, 1)
	go func() { closed <- c.Close() }()
	second := NewCoordinator(c.Store, host, broker)
	tryCtx, cancelTry := context.WithTimeout(context.Background(), 100*time.Millisecond)
	err = second.Start(tryCtx, lock)
	cancelTry()
	if err == nil {
		t.Fatal("second coordinator acquired ownership while admitted callback remained active")
	}
	releaseOnce.Do(func() { close(release) })
	select {
	case <-requestDone:
	case <-time.After(5 * time.Second):
		t.Fatal("operator request did not finish after host callback release")
	}
	select {
	case err := <-closed:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("coordinator close did not drain the admitted request")
	}
	if err := <-serveDone; err != nil && !errors.Is(err, http.ErrServerClosed) {
		t.Fatal(err)
	}
	if err := second.Start(context.Background(), lock); err != nil {
		t.Fatalf("second coordinator could not acquire released ownership: %v", err)
	}
	if err := second.Close(); err != nil {
		t.Fatal(err)
	}
}

// restartCoordinatorAfterCrash reopens the database and starts a fresh
// coordinator over the crash prefix: recorded but unfinished commands
// plus whatever run state the crash left behind.
func restartCoordinatorAfterCrash(t *testing.T, dsn string, host HostFactory, broker BrokerExecution) *Coordinator {
	t.Helper()
	reopened, err := store.Open(dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = reopened.Close() })
	restarted := NewCoordinator(reopened, host, broker)
	if err := restarted.Start(context.Background(), filepath.Join(t.TempDir(), "coordinator.lock")); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = restarted.Close() })
	return restarted
}

func settledStopBroker() *stubBroker {
	return &stubBroker{
		get:   func(string, string) (identity.Execution, error) { return identity.Execution{}, identity.ErrNotFound },
		close: func(string, string) error { return nil },
	}
}

func TestRestartedStopReportsEstablishedOutcome(t *testing.T) {
	ctx := context.Background()
	db, dsn := postgresFixture(t, nil)
	c := NewCoordinator(db, &stubHost{}, &stubBroker{})
	r := recordRun(t, c, nil)
	cmd := stopCommand(r.ID)
	if _, created, err := db.RecordFactoryCommand(ctx, cmd, time.Now()); err != nil || !created {
		t.Fatalf("command prefix unrecorded: %v", err)
	}
	// Crash before the finish lands: close without finishing.
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	host := &stubHost{}
	host.stop = func(project.FactoryStop) (project.FactoryState, error) {
		return project.FactoryState{ID: r.ID, Phase: project.FactoryStopped, Retirement: "confirmed", Reason: "stop-before-start"}, nil
	}
	restarted := restartCoordinatorAfterCrash(t, dsn, host, settledStopBroker())
	receipt, err := restarted.Stop(ctx, cmd)
	if err != nil || !receipt.Confirmed || receipt.Uncertain {
		t.Fatalf("replay after restart: %+v %v", receipt, err)
	}
	again, err := restarted.Stop(ctx, cmd)
	if err != nil || again != receipt {
		t.Fatalf("second replay after restart: %+v %v", again, err)
	}
}

func TestRestartedStopPreservesUncertainty(t *testing.T) {
	ctx := context.Background()
	db, dsn := postgresFixture(t, nil)
	c := NewCoordinator(db, &stubHost{}, &stubBroker{})
	r := recordRun(t, c, nil)
	cmd := stopCommand(r.ID)
	if _, created, err := db.RecordFactoryCommand(ctx, cmd, time.Now()); err != nil || !created {
		t.Fatalf("command prefix unrecorded: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	host := &stubHost{}
	host.stop = func(project.FactoryStop) (project.FactoryState, error) {
		return project.FactoryState{}, errors.New("synthetic host outage")
	}
	restarted := restartCoordinatorAfterCrash(t, dsn, host, settledStopBroker())
	receipt, err := restarted.Stop(ctx, cmd)
	if err != nil {
		t.Fatalf("uncertain replay errored: %v", err)
	}
	if receipt.Confirmed || !receipt.Uncertain {
		t.Fatalf("uncertain effect reported as established: %+v", receipt)
	}
	run, err := restarted.Store.FactoryRun(ctx, r.ID)
	if err != nil || run.Reconciled {
		t.Fatalf("fenced run settled by an unconfirmed stop: %+v %v", run, err)
	}
	stored, err := restarted.Store.FactoryCommand(ctx, cmd.ID)
	if err != nil || stored.Finished == "" {
		t.Fatalf("abandoned command left running: %+v %v", stored, err)
	}
}

func TestRestartedReconcileReportsSettledRuns(t *testing.T) {
	ctx := context.Background()
	db, dsn := postgresFixture(t, nil)
	c := NewCoordinator(db, &stubHost{}, &stubBroker{})
	r := recordRun(t, c, nil)
	cmd := factory.Command{ID: factory.NewID(), Type: factory.CommandReconcile, Principal: "os-uid:0", Digest: factory.CommandDigest(factory.CommandReconcile, "")}
	if _, created, err := db.RecordFactoryCommand(ctx, cmd, time.Now()); err != nil || !created {
		t.Fatalf("command prefix unrecorded: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	host := &stubHost{}
	host.stop = func(project.FactoryStop) (project.FactoryState, error) {
		return project.FactoryState{ID: r.ID, Phase: project.FactoryCompleted, Reason: "done"}, nil
	}
	restarted := restartCoordinatorAfterCrash(t, dsn, host, settledStopBroker())
	receipt, err := restarted.Reconcile(ctx, cmd)
	if err != nil {
		t.Fatalf("replay after restart: %v", err)
	}
	if len(receipt.Settled) != 1 || receipt.Settled[0] != r.ID {
		t.Fatalf("settled runs after restart: %+v", receipt)
	}
}
