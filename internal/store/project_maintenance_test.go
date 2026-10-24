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
	// Retained candidate failures keep their original check result. Their
	// separately recorded native stop receipt owns retirement, not that result.
	checkExit := 124
	preparation.State = project.PrepareState{ID: prepTestID, Project: prepTestProject, Role: preparation.Preparation.Role, Phase: project.PrepareFailed, CheckExit: &checkExit, Output: "retained failed check"}
	meta := candidatePreparationRegistration{Repository: 7, Issue: 1, ActorID: 3, Project: prepTestProject, OwnerAssignment: factory.NewID(), ChildAssignment: factory.NewID(), AttemptRoot: factory.NewID(), PublicationID: factory.NewID(), Connection: "candidate-connection", NotAfter: time.Now().Add(time.Minute).Unix()}
	retained, err := marshalCandidatePreparation(preparation, meta)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.db.ExecContext(ctx, `UPDATE project_preparations SET data=$1 WHERE id=$2`, string(retained), prepTestID); err != nil {
		t.Fatal(err)
	}
	stop := project.PrepareState{ID: prepTestID, Project: prepTestProject, Role: preparation.Preparation.Role, Phase: project.PrepareStopped, Stopped: true, Retirement: "uncertain"}
	if err = s.RecordCandidatePreparationStop(ctx, prepTestID, stop); err != nil {
		t.Fatal(err)
	}
	if _, count, err := s.ProjectMaintenanceWork(ctx); err != nil || count != 1 {
		t.Fatalf("uncertain candidate retirement disappeared: %d %v", count, err)
	}
	stop.Retirement = "confirmed"
	if err = s.RecordCandidatePreparationStop(ctx, prepTestID, stop); err != nil {
		t.Fatal(err)
	}
	if _, count, err := s.ProjectMaintenanceWork(ctx); err != nil || count != 0 {
		t.Fatalf("confirmed candidate retirement still blocks maintenance: %d %v", count, err)
	}
	kept, err := s.CandidatePreparation(ctx, prepTestID)
	if err != nil || !kept.Stopped || kept.Retirement != "confirmed" || kept.Preparation.State.Phase != project.PrepareFailed || kept.Preparation.State.Output != "retained failed check" || kept.Preparation.State.CheckExit == nil || *kept.Preparation.State.CheckExit != 124 {
		t.Fatalf("native retirement rewrote the retained failure: %+v %v", kept, err)
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
