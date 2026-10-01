package control

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
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
	calls    int
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
	db, err := store.Open(filepath.Join(t.TempDir(), "coordinator.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
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
