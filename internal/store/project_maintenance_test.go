package store

import (
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
)

func TestProjectMaintenanceWorkKeepsUncertainWorkVisible(t *testing.T) {
	s, ctx := prepTestStore(t)
	preparation, _, err := s.AdmitPreparation(ctx, prepTestPreparation())
	if err != nil {
		t.Fatal(err)
	}
	run := inventoryRun(time.Now(), prepTestProject)
	if err = s.RecordFactoryRun(ctx, run); err != nil {
		t.Fatal(err)
	}
	for _, state := range []project.PrepareState{
		{},
		{Phase: project.PrepareRunning},
		{Phase: project.PrepareInterrupted},
		{Phase: project.PrepareFailed},
		{Phase: project.PrepareReady}, // Incomplete readiness is uncertain.
		{Phase: project.PrepareReady, Ready: true},
		{Phase: project.PrepareStopped, Stopped: true, Retirement: "uncertain"},
		{Phase: project.PrepareStopped, Stopped: true, Retirement: "confirmed"},
	} {
		if state.Phase != "" {
			state.ID, state.Project, state.Role = preparation.Preparation.ID, preparation.Preparation.Project, preparation.Preparation.Role
			preparation.State = state
			if err = s.ObservePreparation(ctx, preparation); err != nil {
				t.Fatal(err)
			}
			preparation, err = s.Preparation(ctx, preparation.Preparation.ID)
			if err != nil {
				t.Fatal(err)
			}
		}
		want := 1
		if state.Ready || state.Stopped && state.Retirement == "confirmed" {
			want = 0
		}
		factoryWork, preparationWork, err := s.ProjectMaintenanceWork(ctx)
		if err != nil || factoryWork != 1 || preparationWork != want {
			t.Fatalf("phase %q counts = %d/%d, want 1/%d: %v", state.Phase, factoryWork, preparationWork, want, err)
		}
	}
	run.Outcome = factory.Cancelled
	run.Summary = "native work ended, reconciliation pending"
	if err = s.SaveFactoryRun(ctx, run); err != nil {
		t.Fatal(err)
	}
	if count, _, err := s.ProjectMaintenanceWork(ctx); err != nil || count != 1 {
		t.Fatalf("unsettled completion disappeared: %d %v", count, err)
	}
	run.Reconciled = true
	if err = s.SaveFactoryRun(ctx, run); err != nil {
		t.Fatal(err)
	}
	if count, preparations, err := s.ProjectMaintenanceWork(ctx); err != nil || count != 0 || preparations != 0 {
		t.Fatalf("settled work still blocks: %d/%d %v", count, preparations, err)
	}
}

func TestProjectMaintenanceWorkReportsSaturatedCount(t *testing.T) {
	s, ctx := inventoryFixture(t)
	for range 130 {
		if err := s.RecordFactoryRun(ctx, inventoryRun(time.Now(), prepTestProject)); err != nil {
			t.Fatal(err)
		}
	}
	if count, _, err := s.ProjectMaintenanceWork(ctx); err != nil || count != 129 {
		t.Fatalf("bounded count lost sentinel: %d %v", count, err)
	}
}
