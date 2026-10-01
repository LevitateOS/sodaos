package control

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/identity"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
)

const lifecycleProject = "p123456789012345678901234"

func lifecycleProjectFixture(t *testing.T, c *Coordinator, repository int64) {
	t.Helper()
	ctx := context.Background()
	if err := c.Store.UpsertUser(ctx, store.User{ID: 7, Login: "soda-tester"}); err != nil {
		t.Fatal(err)
	}
	if err := c.Store.CreateProject(ctx, store.Project{ID: lifecycleProject, Name: "repo", RepositoryID: repository, OwnerID: 7, Repository: "soda-tester/repo"}); err != nil {
		t.Fatal(err)
	}
}

func settleStubs(host *stubHost, broker *stubBroker) {
	host.stop = func(in project.FactoryStop) (project.FactoryState, error) {
		return project.FactoryState{ID: in.ID, Project: in.Project, Phase: project.FactoryStopped, Retirement: "confirmed", Reason: "stopped", LeaseID: "lease-" + in.ID, Generation: 3, CredentialReturned: true}, nil
	}
	broker.close = func(string, string) error { return nil }
	broker.get = func(kind, id string) (identity.Execution, error) {
		return identity.Execution{
			Kind: kind, ExecutionID: id, State: identity.ExecutionTerminal, Digest: "digest", LeaseID: "lease-" + id,
			Binding: &identity.Binding{Kind: identity.Factory, ID: id, Generation: 3},
		}, nil
	}
}

func readyPreparations(t *testing.T, c *Coordinator, projectID string) {
	t.Helper()
	ctx := context.Background()
	for i, role := range []string{project.RoleCoder, project.RoleReviewer} {
		id := "f0123456789abcdef0123456" + string(rune('0'+i))
		stored, _, err := c.Store.AdmitPreparation(ctx, project.StoredPreparation{Preparation: project.Preparation{
			ID: id, Project: projectID, Role: role,
			Requirements: project.RequirementAcceptance{ID: "d0123456789abcdef01234567", Revision: 1, Approver: 7, SourceCommit: strings.Repeat("a", 40), Digest: strings.Repeat("b", 64)},
			Approval:     project.AdminApproval{ID: "d123456789abcdef012345678", Revision: 1, Approver: 9, EffectsDigest: strings.Repeat("b", 64)},
			SourceCommit: strings.Repeat("a", 40), SetupDigest: strings.Repeat("b", 64), Tools: []string{"python3"},
		}})
		if err != nil {
			t.Fatal(err)
		}
		stored.State.Ready = true
		if err = c.Store.ObservePreparation(ctx, stored); err != nil {
			t.Fatal(err)
		}
	}
}

func TestPauseWithdrawsBeforeStopping(t *testing.T) {
	host, broker := &stubHost{}, &stubBroker{}
	c := coordinatorFixture(t, host, broker)
	lifecycleProjectFixture(t, c, 42)
	grantFullAuthority(t, c)
	settleStubs(host, broker)
	r := recordRun(t, c, nil)
	dispatch := factory.NewID()
	if err := c.Store.RegisterDispatch(context.Background(), factory.DispatchRegistration{ID: dispatch, Repository: 42}); err != nil {
		t.Fatal(err)
	}
	var order []string
	next := host.stop
	host.stop = func(in project.FactoryStop) (project.FactoryState, error) {
		open, _, _, err := c.Store.DispatchState(context.Background(), 42)
		if err != nil || open {
			t.Error("run stopped before dispatch closed")
		}
		order = append(order, in.ID)
		return next(in)
	}
	receipt, err := c.PauseRepository(context.Background(), factory.NewID(), "native:7", 42)
	if err != nil {
		t.Fatal(err)
	}
	if !receipt.Paused || len(order) != 1 || len(receipt.Runs) != 1 || !receipt.Runs[0].Confirmed {
		t.Fatalf("pause: %+v", receipt)
	}
	if len(receipt.Withdrawal.Captured) != 1 || receipt.Withdrawal.Captured[0] != dispatch {
		t.Fatalf("withdrawal: %+v", receipt.Withdrawal)
	}
	if err = receipt.Validate(); err != nil {
		t.Fatal(err)
	}
	if err = c.Store.RegisterDispatch(context.Background(), factory.DispatchRegistration{ID: factory.NewID(), Repository: 42}); !errors.Is(err, store.ErrDispatchClosed) {
		t.Fatalf("delayed registration escaped withdrawal: %v", err)
	}
	replay, err := c.PauseRepository(context.Background(), receipt.CommandID, "native:7", 42)
	if err != nil || replay.CommandID != receipt.CommandID || host.calls != 1 {
		t.Fatal("pause replay re-executed", replay, err)
	}
	settled, err := c.Store.FactoryRun(context.Background(), r.ID)
	if err != nil || !settled.Reconciled {
		t.Fatal("paused run left unsettled", settled, err)
	}
}

func TestPauseKeepsUncertainRunsFenced(t *testing.T) {
	host, broker := &stubHost{}, &stubBroker{}
	c := coordinatorFixture(t, host, broker)
	lifecycleProjectFixture(t, c, 42)
	host.stop = func(project.FactoryStop) (project.FactoryState, error) {
		return project.FactoryState{}, errors.New("host unreachable")
	}
	broker.close = func(string, string) error { return nil }
	broker.get = func(string, string) (identity.Execution, error) { return identity.Execution{}, identity.ErrNotFound }
	r := recordRun(t, c, nil)
	receipt, err := c.PauseRepository(context.Background(), factory.NewID(), "native:7", 42)
	if err != nil {
		t.Fatal(err)
	}
	if !receipt.Paused || len(receipt.Runs) != 1 || !receipt.Runs[0].Uncertain || receipt.Runs[0].Confirmed {
		t.Fatalf("uncertain run reported settled: %+v", receipt)
	}
	left, err := c.Store.FactoryRun(context.Background(), r.ID)
	if err != nil || left.Reconciled {
		t.Fatal("uncertain run settled without confirmation", left, err)
	}
}

func TestResumeReopensOnlyAfterSettledRuns(t *testing.T) {
	host, broker := &stubHost{}, &stubBroker{}
	c := coordinatorFixture(t, host, broker)
	lifecycleProjectFixture(t, c, 42)
	grantFullAuthority(t, c)
	readyPreparations(t, c, lifecycleProject)
	settleStubs(host, broker)
	r := recordRun(t, c, nil)
	if _, err := c.PauseRepository(context.Background(), factory.NewID(), "native:7", 42); err != nil {
		t.Fatal(err)
	}
	resumed, err := c.ResumeRepository(context.Background(), factory.NewID(), "native:7", 42)
	if err != nil || !resumed.Reopened || !resumed.Effective.Effective {
		t.Fatal("settled resume refused", resumed, err)
	}
	if err = resumed.Validate(); err != nil {
		t.Fatal(err)
	}
	already, err := c.ResumeRepository(context.Background(), factory.NewID(), "native:7", 42)
	if err != nil || already.Reopened || !already.Effective.Effective {
		t.Fatal("open resume misreported", already, err)
	}
	if _, err = c.PauseRepository(context.Background(), factory.NewID(), "native:7", 42); err != nil {
		t.Fatal(err)
	}
	// A fresh unsettled run behind the closed gate: resume must refuse.
	_ = r
	unsettled := recordRun(t, c, nil)
	if _, err = c.ResumeRepository(context.Background(), factory.NewID(), "native:7", 42); err == nil {
		t.Fatal("resume admitted with unsettled runs")
	} else {
		var pending ErrPendingRuns
		if !errors.As(err, &pending) || len(pending.Runs) != 1 || pending.Runs[0] != unsettled.ID {
			t.Fatalf("pending runs: %v", err)
		}
	}
}

func TestResumeRefusesWithdrawnGrants(t *testing.T) {
	c := coordinatorFixture(t, nil, nil)
	if _, err := c.Store.WithdrawDispatch(context.Background(), 43, "test", "native:7"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.ResumeRepository(context.Background(), factory.NewID(), "native:7", 43); !errors.Is(err, ErrIneffectiveAuthority) {
		t.Fatalf("resume without grants: %v", err)
	}
}

func TestRetryStaysQueuedAfterReconciledRun(t *testing.T) {
	host, broker := &stubHost{}, &stubBroker{}
	c := coordinatorFixture(t, host, broker)
	lifecycleProjectFixture(t, c, 42)
	grantFullAuthority(t, c)
	readyPreparations(t, c, lifecycleProject)
	settleStubs(host, broker)
	r := recordRun(t, c, nil)
	if _, err := c.RetryRun(context.Background(), factory.NewID(), "native:7", r.ID); err == nil {
		t.Fatal("retry admitted for an unsettled run")
	} else if _, ok := err.(ErrPendingRuns); !ok {
		t.Fatalf("retry pending: %v", err)
	}
	if _, err := c.Stop(context.Background(), stopCommand(r.ID)); err != nil {
		t.Fatal(err)
	}
	decision, err := c.RetryRun(context.Background(), factory.NewID(), "native:7", r.ID)
	if err != nil || !decision.Queued || decision.Prior != r.ID {
		t.Fatal("reconciled retry refused", decision, err)
	}
	if err = decision.Validate(); err != nil {
		t.Fatal(err)
	}
	replay, err := c.RetryRun(context.Background(), decision.CommandID, "native:7", r.ID)
	if err != nil || replay != decision {
		t.Fatal("retry replay diverged", replay, err)
	}
	if _, err = c.RetryRun(context.Background(), factory.NewID(), "native:7", factory.NewID()); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown retry: %v", err)
	}
}

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
