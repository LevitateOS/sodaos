package control

import (
	"context"
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/store"
)

// reassessChooser is the deterministic adapter behind the D13 reassess
// branches: sealed substance judges reassess exactly like the completion
// supplement judges continue, with confidence above the decision bound.
// The decision the drive records, holds and refuses to re-charge is what
// these tests prove; step materialization behind it belongs to FW32.
type reassessChooser struct {
	calls int
}

func (c *reassessChooser) Select(_ context.Context, state any, _ string, choices map[string]any) (factory.PriorityAnswer, error) {
	c.calls++
	decision, ok := state.(ReassessmentDecisionState)
	if !ok {
		return factory.PriorityAnswer{}, errReassessmentTestState
	}
	if len(decision.Evidence) == 0 || !decision.Local.Sealed() || decision.Completion.ID == "" {
		return factory.PriorityAnswer{}, errReassessmentTestBodiless
	}
	for _, body := range decision.Evidence {
		if err := body.Validate(); err != nil {
			return factory.PriorityAnswer{}, err
		}
	}
	answer, ok := choices[factory.ReassessmentOptionReassess].(map[string]any)
	if !ok {
		return factory.PriorityAnswer{}, errReassessmentTestState
	}
	if answer["effect"] == nil {
		return factory.PriorityAnswer{}, errReassessmentTestState
	}
	share := 0.87
	decided := factory.PriorityAnswer{
		Selected: factory.ReassessmentOptionReassess, Confidence: share,
		Probabilities: map[string]float64{}, Model: "test-reassessment-model",
	}
	for id := range choices {
		decided.Probabilities[id] = (1 - share) / float64(len(choices)-1)
	}
	decided.Probabilities[factory.ReassessmentOptionReassess] = share
	return decided, nil
}

// driveReassessmentBranch merges one fixture completion and drives it to
// its funded D13 decision with the given chooser. It returns the decided
// cycle identity for branch assertions.
func driveReassessmentBranch(t *testing.T, chooser ProductJudgment) (context.Context, *store.Store, *Coordinator, string) {
	t.Helper()
	ctx := context.Background()
	fx, p := mergeSeed(t, 3)
	db := fx.publish.db
	coord := fx.publish.coord
	fx.wire(happyMerger())
	fx.pass(t)
	authorizeMergeFixture(t, ctx, db, coord, p)
	if report := fx.pass(t); len(report.Merged) != 1 {
		t.Fatalf("branch merge did not record: %+v", report)
	}
	db.PriorityCallBudget = 100
	coord.Planning = &PlanningJudgments{Journal: db, Funding: db, Chooser: chooser}
	deps := CompletionDriveDeps{
		Enumerator: twoPageEnumerator(t), CaptureLocal: db.CaptureCompletionLocal,
	}
	// One drive collects and seals; the next decides; a decided reassess
	// then holds awaiting its bounded coder run.
	coord.DriveCompletions(ctx, deps)
	report := coord.DriveCompletions(ctx, deps)
	if len(report.Decided) != 1 {
		t.Fatalf("branch decision missing: %+v", report)
	}
	return ctx, db, coord, report.Decided[0]
}

// TestServingReassessmentChangeAwaitsCoderWithoutRerun (T40
// reassess-with-change): the funded reassess decision records on the
// cycle, the drive holds it awaiting the bounded coder run, the fence
// stays pending, and no automatic extra agent call follows — the ledger
// holds exactly the one charged decision call.
func TestServingReassessmentChangeAwaitsCoderWithoutRerun(t *testing.T) {
	ctx, db, coord, cycleID := driveReassessmentBranch(t, &reassessChooser{})
	record, err := db.ReassessmentRecord(ctx, cycleID)
	if err != nil {
		t.Fatal(err)
	}
	if record.Cycle.Decision != factory.ReassessmentOptionReassess {
		t.Fatalf("change did not decide reassess: %+v", record.Cycle)
	}
	// The funded decision is charged exactly once: its shared receipt
	// carries the single call, so a repeated drive re-reads instead of
	// spending again.
	deciding, err := DecisionStateFromRecord(record)
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := db.PriorityJudgment(ctx, fingerprintOf(deciding))
	if err != nil {
		t.Fatal(err)
	}
	if len(receipt.Calls) != 1 || receipt.Calls[0].Failure != "" {
		t.Fatalf("decision receipt is not its single funded call: %+v", receipt)
	}
	again := coord.DriveCompletions(ctx, CompletionDriveDeps{
		Enumerator: twoPageEnumerator(t), CaptureLocal: db.CaptureCompletionLocal,
	})
	if len(again.AwaitingCoder) != 1 || again.AwaitingCoder[0] != cycleID {
		t.Fatalf("decided change left coder wait: %+v", again)
	}
	if len(again.Released) != 0 {
		t.Fatalf("decided change released behind its fence: %+v", again)
	}
	if pending, err := db.CompletionPending(ctx, 7); err != nil || len(pending) == 0 {
		t.Fatalf("fence opened under a decided reassess: %v %v", pending, err)
	}
	head, err := db.AcceptanceHead(ctx, 7, 3)
	if err != nil {
		t.Fatal(err)
	}
	chain, _, err := db.AcceptanceAssignmentsAfter(ctx, 7, 3, head, "", 256)
	if err != nil {
		t.Fatal(err)
	}
	if len(chain) != 1 {
		t.Fatalf("decided change re-ran its attempt: %d assignments", len(chain))
	}
}

// TestServingReassessmentNoChangeStagesEmptyJournal (T40
// reassess-with-no-change): the funded reassess decision records on a
// stable seal, and the staged adoption journal carries zero steps, which
// releases without graph mutations. The drive itself still holds the
// cycle awaiting FW32 adoption — nothing here releases or re-runs on its
// own.
func TestServingReassessmentNoChangeStagesEmptyJournal(t *testing.T) {
	ctx, db, coord, cycleID := driveReassessmentBranch(t, &reassessChooser{})
	record, err := db.ReassessmentRecord(ctx, cycleID)
	if err != nil {
		t.Fatal(err)
	}
	if record.Cycle.Decision != factory.ReassessmentOptionReassess {
		t.Fatalf("stable seal did not decide reassess: %+v", record.Cycle)
	}
	journal := stageReassessmentJournal(factory.ReassessmentReport{
		CompletionID: record.Completion.ID, CycleID: record.Cycle.CycleID,
		InventoryDigest: record.InventoryDigest,
	}, nil)
	if len(journal.Steps) != 0 {
		t.Fatalf("no-change report staged steps: %+v", journal.Steps)
	}
	if err := journal.Releasable(); err != nil {
		t.Fatalf("no-change report does not release: %v", err)
	}
	again := coord.DriveCompletions(ctx, CompletionDriveDeps{
		Enumerator: twoPageEnumerator(t), CaptureLocal: db.CaptureCompletionLocal,
	})
	if len(again.AwaitingCoder) != 1 || len(again.Released) != 0 {
		t.Fatalf("no-change branch left its coder wait or released: %+v", again)
	}
	if pending, err := db.CompletionPending(ctx, 7); err != nil || len(pending) == 0 {
		t.Fatalf("fence opened under a decided reassess: %v %v", pending, err)
	}
}
