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

func TestResumePreservesProjectStopUntilVerifiedStart(t *testing.T) {
	c := coordinatorFixture(t, &stubHost{}, &stubBroker{})
	lifecycleProjectFixture(t, c, 42)
	grantFullAuthority(t, c)
	readyPreparations(t, c, lifecycleProject)
	if _, err := c.Store.WithdrawDispatch(context.Background(), 42, factory.CauseProjectStop, "native:7"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.PauseRepository(context.Background(), factory.NewID(), "native:7", 42); err != nil {
		t.Fatal(err)
	}
	resumed, err := c.ResumeRepository(context.Background(), factory.NewID(), "native:7", 42)
	if err != nil || resumed.Reopened || resumed.Effective.DispatchOpen {
		t.Fatalf("resume cleared Project stop: %+v %v", resumed, err)
	}
	if err = resumed.Validate(); err != nil {
		t.Fatal(err)
	}
	open, _, withdrawal, err := c.Store.DispatchState(context.Background(), 42)
	if err != nil || open || len(withdrawal.ActiveCauses) != 1 || withdrawal.ActiveCauses[0] != factory.CauseProjectStop {
		t.Fatalf("Project stop cause not retained: %v %+v %v", open, withdrawal, err)
	}
	open, err = c.ClearProjectStopAfterStart(context.Background(), lifecycleProject, factory.StartVerification{Started: true})
	if err != nil || !open {
		t.Fatalf("verified start did not clear final cause: %v %v", open, err)
	}
}

func TestStartKeepsProjectStopWhileRunIsUnsettled(t *testing.T) {
	c := coordinatorFixture(t, &stubHost{}, &stubBroker{})
	lifecycleProjectFixture(t, c, 42)
	if _, err := c.Store.WithdrawDispatch(context.Background(), 42, factory.CauseProjectStop, "native:7"); err != nil {
		t.Fatal(err)
	}
	if r := recordRun(t, c, nil); r.ID == "" {
		t.Fatal("fixture did not record run")
	}
	open, err := c.ClearProjectStopAfterStart(context.Background(), lifecycleProject, factory.StartVerification{Started: true})
	if err != nil || open {
		t.Fatalf("clean verification cleared a stop with unsettled work: %v %v", open, err)
	}
	open, _, withdrawal, err := c.Store.DispatchState(context.Background(), 42)
	if err != nil || open || len(withdrawal.ActiveCauses) != 1 || withdrawal.ActiveCauses[0] != factory.CauseProjectStop {
		t.Fatalf("unsettled run lost its Project stop fence: %v %+v %v", open, withdrawal, err)
	}
}
