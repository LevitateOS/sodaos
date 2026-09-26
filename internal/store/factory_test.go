package store

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
)

func factoryFixture(t *testing.T) (*Store, factory.Attempt, time.Time) {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), "factory.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := s.Close(); err != nil {
			t.Error(err)
		}
	})
	now := time.Date(2026, 9, 26, 0, 0, 0, 0, time.UTC)
	a, err := factory.New(factory.WorkItem{RepositoryID: 7, Issue: 1, HumanID: 3, Objective: "repair the check", BaseSHA: strings.Repeat("a", 40), PolicySHA: strings.Repeat("a", 64)}, "delivery-1", now)
	if err != nil {
		t.Fatal(err)
	}
	a, created, err := s.AdmitFactory(context.Background(), a)
	if err != nil || !created {
		t.Fatal(a, created, err)
	}
	return s, a, now
}

func factoryExecution(t *testing.T, a factory.Attempt, role factory.Role, now time.Time) factory.Run {
	t.Helper()
	r, err := a.BeginRun(role, now)
	if err != nil {
		t.Fatal(err)
	}
	r.Image = "sha256:" + strings.Repeat("d", 64)
	r.Harness, r.Model = "codex-0.153.4", "test-model"
	r.Resources = []factory.Resource{{Kind: "workspace", Name: factory.ResourceName(r.ID, "workspace")}}
	return r
}

func TestFactoryDuplicateDeliveryAndExplicitRestart(t *testing.T) {
	s, a, now := factoryFixture(t)
	ctx := context.Background()
	duplicate, err := factory.New(a.Work, a.Delivery, now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	got, created, err := s.AdmitFactory(ctx, duplicate)
	if err != nil || created || got.ID != a.ID || !got.Deadline.Equal(a.Deadline) {
		t.Fatal(got, created, err)
	}
	restart, err := factory.New(a.Work, "human-restart", now)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.AdmitFactory(ctx, restart); err == nil {
		t.Fatal("restart overlapped active work")
	}
	a.Finish(factory.Cancelled, "human cancellation")
	ok, err := s.SaveFactoryAttempt(ctx, &a)
	if err != nil || !ok {
		t.Fatal(ok, err)
	}
	got, created, err = s.AdmitFactory(ctx, duplicate)
	if err != nil || created || got.Outcome != factory.Cancelled {
		t.Fatal(got, created, err)
	}
	if _, created, err := s.AdmitFactory(ctx, restart); err != nil || !created {
		t.Fatal(created, err)
	}
}

func TestFactoryCancellationRejectsStaleWriterAndChangedAdmission(t *testing.T) {
	s, a, _ := factoryFixture(t)
	ctx := context.Background()
	stale := a
	a.Finish(factory.Cancelled, "cancelled")
	if ok, err := s.SaveFactoryAttempt(ctx, &a); err != nil || !ok {
		t.Fatal(ok, err)
	}
	if ok, err := s.SaveFactoryAttempt(ctx, &stale); err != nil || ok {
		t.Fatal("stale writer", ok, err)
	}
	a.Work.BaseSHA = strings.Repeat("b", 40)
	if _, err := s.SaveFactoryAttempt(ctx, &a); err == nil {
		t.Fatal("admitted revision changed")
	}
	got, err := s.FactoryAttempt(ctx, a.ID)
	if err != nil || got.Outcome != factory.Cancelled || got.Work.BaseSHA != stale.Work.BaseSHA {
		t.Fatal(got, err)
	}
}

func TestFactoryResourceOwnershipSurvivesReopenAndBlocksLeakedCleanup(t *testing.T) {
	s, a, now := factoryFixture(t)
	ctx := context.Background()
	r := factoryExecution(t, a, factory.Implementation, now)
	if err := s.StartFactoryRun(ctx, &a, r); err != nil {
		t.Fatal(err)
	}
	if a.Executions != 1 || a.CleanupComplete {
		t.Fatal(a)
	}
	r.Resources[0].ID = strings.Repeat("c", 64)
	if err := s.SaveFactoryRun(ctx, r); err != nil {
		t.Fatal(err)
	}
	var path string
	if err := s.db.QueryRow(`SELECT file FROM pragma_database_list WHERE name='main'`).Scan(&path); err != nil {
		t.Fatal(err)
	}
	other, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := other.Close(); err != nil {
			t.Error(err)
		}
	}()
	runs, err := other.FactoryRuns(ctx, a.ID)
	if err != nil || len(runs) != 1 || runs[0].Resources[0].ID != r.Resources[0].ID {
		t.Fatal(runs, err)
	}
	if err := a.CandidateFrom(r, strings.Repeat("b", 40), now); err != nil {
		t.Fatal(err)
	}
	if ok, err := s.SaveFactoryAttempt(ctx, &a); err != nil || !ok {
		t.Fatal(ok, err)
	}
	r.Outcome = factory.Succeeded
	if err := s.SaveFactoryRun(ctx, r); err != nil {
		t.Fatal(err)
	}
	a.CleanupComplete = true
	if ok, err := s.SaveFactoryAttempt(ctx, &a); err != nil || ok {
		t.Fatal("leaked run marked clean", ok, err)
	}
	a.CleanupComplete = false
	reviewer := factoryExecution(t, a, factory.Review, now)
	if err := s.StartFactoryRun(ctx, &a, reviewer); err == nil {
		t.Fatal("new run overlapped leaked workspace")
	}
	got, err := s.FactoryAttempt(ctx, a.ID)
	if err != nil || got.Executions != 1 {
		t.Fatal("failed launch consumed budget", got, err)
	}
	r.CleanupComplete = true
	if err := s.SaveFactoryRun(ctx, r); err != nil {
		t.Fatal(err)
	}
	a.CleanupComplete = true
	if ok, err := s.SaveFactoryAttempt(ctx, &a); err != nil || !ok {
		t.Fatal(ok, err)
	}
	if err := s.StartFactoryRun(ctx, &a, reviewer); err != nil {
		t.Fatal(err)
	}
}

func TestFactoryRunBindingAndTerminalOutcomeCannotBeRewritten(t *testing.T) {
	s, a, now := factoryFixture(t)
	ctx := context.Background()
	r := factoryExecution(t, a, factory.Implementation, now)
	if err := s.StartFactoryRun(ctx, &a, r); err != nil {
		t.Fatal(err)
	}
	changed := r
	changed.InputSHA = strings.Repeat("b", 40)
	if err := s.SaveFactoryRun(ctx, changed); err == nil {
		t.Fatal("run input changed")
	}
	changed = r
	changed.Resources = append([]factory.Resource(nil), r.Resources...)
	changed.Resources[0].Name = "persistent-human-project"
	if err := s.SaveFactoryRun(ctx, changed); err == nil {
		t.Fatal("unowned resource entered ledger")
	}
	r.Outcome = factory.Cancelled
	if err := s.SaveFactoryRun(ctx, r); err != nil {
		t.Fatal(err)
	}
	r.Outcome = ""
	if err := s.SaveFactoryRun(ctx, r); err == nil {
		t.Fatal("terminal execution reactivated")
	}
}

func TestLatestFactoryAttemptFollowsExplicitAdmission(t *testing.T) {
	s, a, now := factoryFixture(t)
	a.Finish(factory.NeedsHuman, "needs clarification")
	saved, err := s.SaveFactoryAttempt(t.Context(), &a)
	if err != nil || !saved {
		t.Fatal(err)
	}
	next, err := factory.New(a.Work, "human-next", now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	next, _, err = s.AdmitFactory(t.Context(), next)
	if err != nil {
		t.Fatal(err)
	}
	latest, err := s.LatestFactoryAttempt(t.Context(), a.Work.RepositoryID, a.Work.Issue)
	if err != nil || latest.ID != next.ID {
		t.Fatal("older result replaced current work state")
	}
}
