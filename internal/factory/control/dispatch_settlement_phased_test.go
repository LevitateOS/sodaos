package control

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
)

func settleSucceededRunFixture(t *testing.T) (context.Context, *store.Store, factory.Assignment, factory.Run) {
	t.Helper()
	return settleRunFixture(t, factory.Succeeded, "host says so")
}

func assertBlockedHuman(t *testing.T, finished factory.Assignment, ok bool) {
	t.Helper()
	if !ok || finished.Outcome != factory.NeedsHuman || finished.Result.Status != "blocked" {
		t.Fatalf("settled = %+v %v, want blocked human", finished, ok)
	}
	if finished.Result.Candidate != "" {
		t.Fatalf("candidate = %q, want empty", finished.Result.Candidate)
	}
}

func assertSingleChargeStable(t *testing.T, ctx context.Context, db *store.Store, a factory.Assignment, run factory.Run) {
	t.Helper()
	total, err := db.UsageTotal(ctx, a.Repository, a.Connection)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.RunUsage(ctx, run.ID); err != nil {
		t.Fatalf("usage row: %v", err)
	}
	if again, ok := AccountSettledRun(ctx, db, run, "", time.Now()); ok || again.Stage != "" {
		t.Fatalf("replay = %+v %v, want silent refusal", again, ok)
	}
	if replayed, err := db.UsageTotal(ctx, a.Repository, a.Connection); err != nil || replayed != total {
		t.Fatalf("usage total = %d, want stable %d", replayed, total)
	}
}

// TestPhasedSettlementRefusalsOnPG proves the R10 refusal matrix end to
// end on real PG: a candidate-bearing planning result and a cross-phase
// result both finish blocked-human with no recorded candidate and a
// single charge, and a stale direct finish writes nothing twice.
func TestPhasedSettlementRefusalsOnPG(t *testing.T) {
	t.Run("cross-phase finishes blocked", func(t *testing.T) {
		ctx, db, a, run := settleSucceededRunFixture(t)
		payload, err := json.Marshal(factory.PhaseResult{
			Version: 2, Phase: project.ExecutionImplement, Status: "completed",
			Summary: "x", Output: &factory.PhaseOutput{Implementation: &factory.ImplementationReceipt{Candidate: strings.Repeat("e", 40)}},
		})
		if err != nil {
			t.Fatal(err)
		}
		finished, ok := AccountSettledRun(ctx, db, run, fencedResult(t, json.RawMessage(payload)), time.Now())
		assertBlockedHuman(t, finished, ok)
		assertSingleChargeStable(t, ctx, db, a, run)
	})

	t.Run("legacy candidate finishes blocked", func(t *testing.T) {
		ctx, db, a, run := settleSucceededRunFixture(t)
		legacy := `{"status":"completed","summary":"done","candidate":"` + strings.Repeat("d", 40) + `","review_passed":false,"findings":[]}`
		output := "log\n```" + factory.ResultFence + "\n" + legacy + "\n```\n"
		finished, ok := AccountSettledRun(ctx, db, run, output, time.Now())
		assertBlockedHuman(t, finished, ok)
		assertSingleChargeStable(t, ctx, db, a, run)
	})

	t.Run("stale direct finish writes once", func(t *testing.T) {
		ctx, db, a, run := settleSucceededRunFixture(t)
		plan := phaseTestPlan(t, a.AttemptRoot, a.SourceCommit, a.Preparation)
		output := fencedResult(t, factory.PhaseResult{
			Version: factory.PhaseResultVersion, Phase: project.ExecutionInvestigate, Status: "completed",
			Summary: "plan ready", Output: &factory.PhaseOutput{Plan: &plan},
		})
		now := time.Now()
		built, err := finishAssignmentFromRun(a, run, output, now)
		if err != nil {
			t.Fatal(err)
		}
		if built.Outcome != factory.Succeeded {
			t.Fatalf("good-path outcome = %v, want succeeded", built.Outcome)
		}
		if _, ok := AccountSettledRun(ctx, db, run, output, now); !ok {
			t.Fatal("good settlement unaccounted")
		}
		if err := db.FinishAssignment(ctx, built); !errors.Is(err, store.ErrNotFound) {
			t.Fatalf("stale finish error = %v, want %v", err, store.ErrNotFound)
		}
		stored, err := db.Assignment(ctx, a.ID)
		if err != nil || stored.Stage != factory.AssignmentFinished || stored.Outcome != factory.Succeeded {
			t.Fatalf("stored = %+v %v, want one finished success", stored, err)
		}
	})
}
