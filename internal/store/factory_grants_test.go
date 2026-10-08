package store

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
)

func grantTestTime() time.Time { return time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC) }

func grantStoreFixture(t *testing.T) *Store {
	t.Helper()
	db, _ := postgresFixture(t, nil)
	return db
}

func grantTestPolicy() factory.RepositoryPolicy {
	return factory.RepositoryPolicy{
		Repository: 42, GrantedBy: 7, Enabled: true,
		TargetBranch: "refs/heads/main",
		Roles: map[string]factory.RoleSelection{
			project.RoleCoder:    {Harness: "codex-0.157.1", Model: "test"},
			project.RoleReviewer: {Harness: "codex-0.157.1", Model: "test"},
		},
		Checks:        []string{"native-ci/build"},
		MergeMethod:   factory.MergeFastForward,
		Publish:       factory.ActorBindingRef{TokenID: 11, ActorID: 12, Kind: factory.OpRefPublish},
		Create:        factory.ActorBindingRef{TokenID: 11, ActorID: 12, Kind: factory.OpPRCreate},
		Review:        factory.ActorBindingRef{TokenID: 13, ActorID: 14, Kind: factory.OpReviewSubmit},
		Merge:         factory.ActorBindingRef{TokenID: 15, ActorID: 16, Kind: factory.OpMerge},
		MaxConcurrent: 2,
	}
}

func TestPolicySaveIsCAS(t *testing.T) {
	db := grantStoreFixture(t)
	ctx := context.Background()
	if err := db.SaveRepositoryPolicy(ctx, grantTestPolicy()); err != nil {
		t.Fatal(err)
	}
	stored, err := db.RepositoryPolicy(ctx, 42)
	if err != nil || stored.Revision != 1 || !stored.Enabled {
		t.Fatalf("policy: %+v %v", stored, err)
	}
	stale := grantTestPolicy()
	stale.Paused = true
	if err = db.SaveRepositoryPolicy(ctx, stale); !errors.Is(err, ErrStaleRevision) {
		t.Fatal("stale policy revision saved", err)
	}
	stored.Paused = true
	if err = db.SaveRepositoryPolicy(ctx, stored); err != nil {
		t.Fatal(err)
	}
	if _, err = db.RepositoryPolicy(ctx, 43); !errors.Is(err, ErrNotFound) {
		t.Fatal("missing policy reported", err)
	}
}

func TestCapacityAndOperatorGrant(t *testing.T) {
	db := grantStoreFixture(t)
	ctx := context.Background()
	if _, err := db.Capacity(ctx); !errors.Is(err, ErrNotFound) {
		t.Fatal("missing capacity reported", err)
	}
	if err := db.SaveCapacity(ctx, factory.Capacity{UpdatedBy: 1, MaxConcurrentRuns: 2, MaxQueued: 4}); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveCapacity(ctx, factory.Capacity{UpdatedBy: 1, MaxConcurrentRuns: 2}); !errors.Is(err, ErrStaleRevision) {
		t.Fatal("stale capacity saved", err)
	}
	grant := factory.OperatorGrant{Repository: 42, GrantedBy: 1, Active: true, MaxConcurrent: 2}
	if err := db.SaveOperatorGrant(ctx, grant); err != nil {
		t.Fatal(err)
	}
	grant.Active = false
	if err := db.SaveOperatorGrant(ctx, grant); !errors.Is(err, ErrStaleRevision) {
		t.Fatal("stale operator grant saved", err)
	}
}

func TestSponsorships(t *testing.T) {
	db := grantStoreFixture(t)
	ctx := context.Background()
	sp := factory.Sponsorship{
		Repository: 42, GrantedBy: 9, Connection: "conn-1", GrantID: "grant-1",
		Generation: 3, Roles: []string{project.RoleCoder}, AllowanceMinutes: 60, MaxConcurrent: 1, Active: true,
	}
	if err := db.SaveSponsorship(ctx, sp); err != nil {
		t.Fatal(err)
	}
	stored, err := db.Sponsorship(ctx, 42, "conn-1")
	if err != nil || stored.Revision != 1 {
		t.Fatalf("sponsorship: %+v %v", stored, err)
	}
	list, err := db.Sponsorships(ctx, 42)
	if err != nil || len(list) != 1 {
		t.Fatalf("sponsorships: %+v %v", list, err)
	}
	sp.Active = false
	if err = db.SaveSponsorship(ctx, sp); !errors.Is(err, ErrStaleRevision) {
		t.Fatal("stale sponsorship saved", err)
	}
}

func TestDispatchWithdrawalOrdering(t *testing.T) {
	db := grantStoreFixture(t)
	ctx := context.Background()
	open, revision, _, err := db.DispatchState(ctx, 42)
	if err != nil || !open || revision != 0 {
		t.Fatal("fresh dispatch is not open", open, revision, err)
	}
	first := factory.DispatchRegistration{ID: factory.NewID(), Repository: 42, Authority: factory.AuthorityRef{Policy: 1}}
	second := factory.DispatchRegistration{ID: factory.NewID(), Repository: 42, Authority: factory.AuthorityRef{Policy: 1}}
	if err = db.RegisterDispatch(ctx, first); err != nil {
		t.Fatal(err)
	}
	if err = db.RegisterDispatch(ctx, first); err != nil {
		t.Fatal("identical registration refused", err)
	}
	changed := first
	changed.Revision = 9
	if err = db.RegisterDispatch(ctx, changed); !errors.Is(err, ErrDispatchConflict) {
		t.Fatal("changed registration accepted", err)
	}
	if err = db.RegisterDispatch(ctx, second); err != nil {
		t.Fatal(err)
	}
	withdrawal, err := db.WithdrawDispatch(ctx, 42, "policy_paused", "native:7")
	if err != nil || len(withdrawal.Captured) != 2 || withdrawal.Captured[0] != first.ID || withdrawal.Revision != 1 {
		t.Fatalf("withdrawal: %+v %v", withdrawal, err)
	}
	late := factory.DispatchRegistration{ID: factory.NewID(), Repository: 42}
	if err = db.RegisterDispatch(ctx, late); !errors.Is(err, ErrDispatchClosed) {
		t.Fatal("late registration escaped withdrawal", err)
	}
	replay, err := db.WithdrawDispatch(ctx, 42, "policy_paused", "native:7")
	if err != nil || len(replay.Captured) != 2 || replay.Revision != 1 {
		t.Fatalf("withdrawal replay: %+v %v", replay, err)
	}
	if _, _, err = db.ReopenDispatch(ctx, 42, 0); !errors.Is(err, ErrStaleRevision) {
		t.Fatal("stale reopen accepted", err)
	}
	if opened, revision, reopenErr := db.ReopenDispatch(ctx, 42, 1); reopenErr != nil || !opened || revision != 2 {
		t.Fatal(opened, revision, reopenErr)
	}
	if err = db.RegisterDispatch(ctx, late); err != nil {
		t.Fatal("registration refused after reopen", err)
	}
}

func TestDispatchCausesComposeAndClearIndependently(t *testing.T) {
	db := grantStoreFixture(t)
	ctx := context.Background()
	first, err := db.WithdrawDispatch(ctx, 42, factory.CauseProjectStop, "native:7")
	if err != nil || first.Revision != 1 || len(first.ActiveCauses) != 1 {
		t.Fatalf("first cause: %+v %v", first, err)
	}
	second, err := db.WithdrawDispatch(ctx, 42, factory.CauseControlPaused, "native:8")
	if err != nil || second.Revision != first.Revision || second.Cause != first.Cause || second.ClosedBy != first.ClosedBy || len(second.ActiveCauses) != 2 {
		t.Fatalf("second cause changed first-closure receipt: %+v %v", second, err)
	}
	open, revision, state, err := db.DispatchState(ctx, 42)
	if err != nil || open || revision != 2 || len(state.Captured) != 0 || len(state.ActiveCauses) != 2 {
		t.Fatalf("composed state: %v %d %+v %v", open, revision, state, err)
	}
	duplicate, err := db.WithdrawDispatch(ctx, 42, factory.CauseControlPaused, "different-principal")
	if err != nil || len(duplicate.ActiveCauses) != 2 {
		t.Fatalf("duplicate cause changed state: %+v %v", duplicate, err)
	}
	open, revision, err = db.ClearProjectStop(ctx, 42, revision)
	if err != nil || open || revision != 3 {
		t.Fatalf("clearing Project stop cleared independent pause: %v %d %v", open, revision, err)
	}
	open, revision, err = db.ReopenDispatch(ctx, 42, revision)
	if err != nil || !open || revision != 4 {
		t.Fatalf("resume did not clear final cause: %v %d %v", open, revision, err)
	}
	open, revision, state, err = db.DispatchState(ctx, 42)
	if err != nil || !open || revision != 4 || len(state.ActiveCauses) != 0 || state.Cause != factory.CauseProjectStop {
		t.Fatalf("open state lost first-closure receipt or retained causes: %v %d %+v %v", open, revision, state, err)
	}
}

func TestProjectStopCauseComposesAfterExistingPause(t *testing.T) {
	db := grantStoreFixture(t)
	ctx := context.Background()
	first, err := db.WithdrawDispatch(ctx, 42, factory.CauseControlPaused, "native:7")
	if err != nil {
		t.Fatal(err)
	}
	second, err := db.WithdrawDispatch(ctx, 42, factory.CauseProjectStop, "native:8")
	if err != nil || second.Cause != first.Cause || second.ClosedBy != first.ClosedBy || len(second.ActiveCauses) != 2 {
		t.Fatalf("Project stop overwrote the first pause receipt: %+v %v", second, err)
	}
	open, revision, err := db.ClearProjectStop(ctx, 42, 2)
	if err != nil || open || revision != 3 {
		t.Fatalf("Start cleared the independent pause: %v %d %v", open, revision, err)
	}
	open, revision, _, err = db.DispatchState(ctx, 42)
	if err != nil || open || revision != 3 {
		t.Fatalf("pause did not remain active: %v %d %v", open, revision, err)
	}
	open, revision, err = db.ReopenDispatch(ctx, 42, revision)
	if err != nil || !open || revision != 4 {
		t.Fatalf("authorized Resume did not clear the final pause: %v %d %v", open, revision, err)
	}
}

func TestSettingsCommandReplay(t *testing.T) {
	db := grantStoreFixture(t)
	ctx := context.Background()
	payload := `{"enabled":true}`
	cmd := factory.Command{
		ID: factory.NewID(), Type: factory.CommandPolicy, Target: "repository/42/policy",
		Principal: "native:7", Payload: payload, Digest: factory.SettingsDigest(factory.CommandPolicy, "repository/42/policy", payload),
	}
	stored, created, err := db.RecordFactoryCommand(ctx, cmd, grantTestTime())
	if err != nil || !created || stored.Payload != payload {
		t.Fatalf("command: %+v %v %v", stored, created, err)
	}
	again, created, err := db.RecordFactoryCommand(ctx, cmd, grantTestTime())
	if err != nil || created || again.ID != cmd.ID {
		t.Fatalf("replay: %+v %v %v", again, created, err)
	}
	changed := cmd
	changed.Payload = `{"enabled":false}`
	changed.Digest = factory.SettingsDigest(changed.Type, changed.Target, changed.Payload)
	if _, _, err = db.RecordFactoryCommand(ctx, changed, grantTestTime()); !errors.Is(err, ErrCommandConflict) {
		t.Fatal("changed payload reused the command identity", err)
	}
}
