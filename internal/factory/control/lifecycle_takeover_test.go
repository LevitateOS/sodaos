package control

import (
	"context"
	"errors"
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/identity"
	"github.com/levitateos/sodaos/internal/project"
)

func TestTakeoverCopiesOnlyReconciledRuns(t *testing.T) {
	host, broker := &stubHost{}, &stubBroker{}
	c := coordinatorFixture(t, host, broker)
	lifecycleProjectFixture(t, c, 42)
	settleStubs(host, broker)
	takeovers := 0
	host.takeover = func(in project.FactoryTakeover) (project.TakeoverResult, error) {
		takeovers++
		return project.TakeoverResult{ID: in.ID, Project: in.Project, Member: in.Member, Destination: project.TakeoverDestination(in.Member, in.ID)}, nil
	}
	r := recordRun(t, c, nil)
	if _, err := c.TakeoverRun(context.Background(), factory.NewID(), "native:7", r.ID, "alice"); err == nil {
		t.Fatal("takeover finished for an unsettled run")
	}
	if _, err := c.Stop(context.Background(), stopCommand(r.ID)); err != nil {
		t.Fatal(err)
	}
	record, err := c.TakeoverRun(context.Background(), factory.NewID(), "native:7", r.ID, "alice")
	if err != nil || record.Run != r.ID || record.Member != "alice" || record.Dest != project.TakeoverDestination("alice", r.ID) {
		t.Fatal("settled takeover refused", record, err)
	}
	if err = record.Validate(); err != nil {
		t.Fatal(err)
	}
	again, err := c.TakeoverRun(context.Background(), factory.NewID(), "native:7", r.ID, "alice")
	if err != nil || again != record || takeovers != 1 {
		t.Fatal("takeover copied twice", again, err)
	}
	host.takeover = func(project.FactoryTakeover) (project.TakeoverResult, error) {
		return project.TakeoverResult{}, errors.New("copy unconfirmed")
	}
	other := recordRun(t, c, nil)
	if _, err = c.Stop(context.Background(), stopCommand(other.ID)); err != nil {
		t.Fatal(err)
	}
	if _, err = c.TakeoverRun(context.Background(), factory.NewID(), "native:7", other.ID, "alice"); !errors.Is(err, ErrTakeoverFailed) {
		t.Fatalf("unconfirmed takeover: %v", err)
	}
}

func TestStopProjectScopesWithdrawalAndRuns(t *testing.T) {
	host, broker := &stubHost{}, &stubBroker{}
	c := coordinatorFixture(t, host, broker)
	lifecycleProjectFixture(t, c, 42)
	settleStubs(host, broker)
	mine := recordRun(t, c, nil)
	if _, _, err := c.StopProject(context.Background(), "native:7", lifecycleProject); err != nil {
		t.Fatal(err)
	}
	open, _, _, err := c.Store.DispatchState(context.Background(), 42)
	if err != nil || open {
		t.Fatal("project stop left dispatch open")
	}
	settled, err := c.Store.FactoryRun(context.Background(), mine.ID)
	if err != nil || !settled.Reconciled {
		t.Fatal("project run left unsettled", settled, err)
	}
	if _, _, err = c.StopProject(context.Background(), "native:7", "p999999999999999999999999"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown project: %v", err)
	}
}

func TestVerifyProjectStartClassifiesRuns(t *testing.T) {
	host, broker := &stubHost{}, &stubBroker{}
	c := coordinatorFixture(t, host, broker)
	lifecycleProjectFixture(t, c, 42)
	settleStubs(host, broker)
	ctx := context.Background()
	settled := recordRun(t, c, nil)
	if _, err := c.Stop(ctx, stopCommand(settled.ID)); err != nil {
		t.Fatal(err)
	}
	live := recordRun(t, c, nil)
	unknown := recordRun(t, c, nil)
	broker.get = func(kind, id string) (identity.Execution, error) {
		switch id {
		case live.ID:
			return identity.Execution{Kind: kind, ExecutionID: id, State: identity.ExecutionLive, Digest: "digest"}, nil
		case unknown.ID:
			return identity.Execution{}, identity.ErrNotFound
		default:
			return identity.Execution{Kind: kind, ExecutionID: id, State: identity.ExecutionTerminal, Digest: "digest"}, nil
		}
	}
	host.inspect = func(in project.FactoryInspect) (project.FactoryState, error) {
		if in.ID == unknown.ID {
			return project.FactoryState{}, errors.New("no receipt")
		}
		return project.FactoryState{ID: in.ID, Project: in.Project, Phase: project.FactoryStopped}, nil
	}
	verification, err := c.VerifyProjectStart(ctx, lifecycleProject)
	if err != nil {
		t.Fatal(err)
	}
	if !verification.Started || len(verification.Revived) != 1 || verification.Revived[0] != live.ID {
		t.Fatalf("revived: %+v", verification)
	}
	if len(verification.Unverified) != 1 || verification.Unverified[0] != unknown.ID {
		t.Fatalf("unverified: %+v", verification)
	}
	if verification.Hold {
		t.Fatalf("hold misreported: %+v", verification)
	}
	if err = verification.Validate(); err != nil {
		t.Fatal(err)
	}
	if _, err = c.VerifyProjectStart(ctx, "p999999999999999999999999"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown project: %v", err)
	}
}
