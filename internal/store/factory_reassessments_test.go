package store

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
)

// FW31 persists D13 completion events and bounded cycles, fences packet
// admission per repository and charges funded Jev calls. Pure checks run
// everywhere; the transaction proof needs the disposable SODA_PG_*
// fixture and skips without it.
func TestReassessmentRecordValidation(t *testing.T) {
	cycle := factory.ReassessmentCycle{
		EventID: factory.NewID(), CycleID: factory.NewID(),
		State: factory.ReassessmentCollecting, Repository: 7, Generation: 1,
	}
	record := ReassessmentRecord{Cycle: cycle, Issue: 31}
	if err := record.Validate(); err != nil {
		t.Fatalf("valid record refused: %v", err)
	}
	bad := record
	bad.InventoryDigest = "not-a-digest"
	if err := bad.Validate(); err == nil {
		t.Fatal("malformed inventory digest accepted")
	}
	bad = record
	bad.WaitReason = strings.Repeat("r", factory.MaxReassessmentReasonChars+1)
	if err := bad.Validate(); err == nil {
		t.Fatal("oversize wait reason accepted")
	}
}

func TestCompletionFenceErrorsDistinct(t *testing.T) {
	if ErrCompletionFenced == nil || ErrReassessmentBudgetExhausted == nil {
		t.Fatal("fence errors missing")
	}
	if ErrCompletionFenced.Error() == ErrReassessmentBudgetExhausted.Error() {
		t.Fatal("fence errors collide")
	}
}

// mergedCompletionRecord builds a valid merged merge value: the exact
// shape the first UpdateMerge recording a completion carries, without
// needing the merge journey itself.
func mergedCompletionRecord(publication, occurrence string) factory.Merge {
	m := mergeTestRecord()
	now := time.Now().Unix()
	m.PublicationID = publication
	m.Stage, m.Outcome, m.Reason = factory.MergeMerged, factory.Succeeded, factory.MergeReasonMerged
	m.FinishedUnix, m.MergedUnix, m.ClosedUnix = now, now, now
	m.OriginalClosingOccurrence, m.OriginalClosedUnix = occurrence, now
	m.MergedCommit = m.HeadOID
	return m
}

func recordTestCompletion(t *testing.T, db *Store, m factory.Merge) bool {
	t.Helper()
	tx, err := db.db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback() }()
	inserted, err := recordCompletionTx(context.Background(), tx, m)
	if err != nil {
		t.Fatal(err)
	}
	if inserted {
		if err := tx.Commit(); err != nil {
			t.Fatal(err)
		}
	}
	return inserted
}

func TestCompletionPersistFenceAndFund(t *testing.T) {
	db := dispatchStoreFixture(t)
	db.PriorityCallBudget = 100
	ctx := context.Background()
	const repository = int64(7)

	if pending, err := db.CompletionPending(ctx, repository); err != nil || len(pending) != 0 {
		t.Fatalf("pending=%v err=%v, want an open fence", pending, err)
	}

	publication := factory.NewID()
	first := mergedCompletionRecord(publication, "501")
	if !recordTestCompletion(t, db, first) {
		t.Fatal("first completion not recorded")
	}
	// Recovery re-observation replays the same effect: exact-once.
	if recordTestCompletion(t, db, first) {
		t.Fatal("replayed completion recorded twice")
	}
	second := mergedCompletionRecord(factory.NewID(), "502")
	if !recordTestCompletion(t, db, second) {
		t.Fatal("second completion not recorded")
	}

	pending, err := db.CompletionPending(ctx, repository)
	if err != nil || len(pending) != 2 || pending[0] != 1 || pending[1] != 2 {
		t.Fatalf("pending=%v err=%v, want ordered generations 1,2", pending, err)
	}
	cycles, err := db.PendingReassessmentCycles(ctx, repository)
	if err != nil || len(cycles) != 2 {
		t.Fatalf("cycles=%v err=%v, want two collecting cycles", cycles, err)
	}
	for _, record := range cycles {
		if record.Cycle.State != factory.ReassessmentCollecting || record.Issue != 3 {
			t.Fatalf("cycle=%+v, want collecting cycle for issue 3", record.Cycle)
		}
	}
	if cycles[0].Cycle.Generation != 1 || cycles[1].Cycle.Generation != 2 {
		t.Fatal("cycles out of generation order")
	}

	// A wrong-state compare-and-swap refuses instead of overwriting.
	stale := cycles[0]
	stale.Cycle.State = factory.ReassessmentDeciding
	if err := db.CasReassessmentRecord(ctx, stale.Cycle.CycleID, factory.ReassessmentDeciding, stale); !errors.Is(err, ErrStaleRevision) {
		t.Fatalf("wrong-state CAS: %v", err)
	}
	// Sealed evidence is immutable once recorded.
	sealed := cycles[0]
	sealed.InventoryDigest = strings.Repeat("a", 64)
	sealed.EvidenceFingerprint = strings.Repeat("b", 64)
	sealed.Local = factory.CompletionLocal{
		SourceCommit: strings.Repeat("c", 40), PreparationDigest: strings.Repeat("d", 64),
		ScopeDigest: strings.Repeat("e", 64), AuthorityFingerprint: "authority",
	}
	sealed.Cycle.State = factory.ReassessmentDeciding
	if err := db.CasReassessmentRecord(ctx, sealed.Cycle.CycleID, factory.ReassessmentCollecting, sealed); err != nil {
		t.Fatalf("seal CAS: %v", err)
	}
	rewritten := sealed
	rewritten.InventoryDigest = strings.Repeat("f", 64)
	if err := db.CasReassessmentRecord(ctx, rewritten.Cycle.CycleID, factory.ReassessmentDeciding, rewritten); err == nil {
		t.Fatal("sealed inventory rewritten")
	}

	// One funded call reserves against the ledger and the global
	// ceiling; repeats reconcile instead of charging again.
	call := factory.NewID()
	fingerprint := strings.Repeat("0", 64)
	reserved, err := db.ReserveReassessmentCall(ctx, sealed.Cycle.CycleID, factory.ReassessmentDeciding, call, factory.ReassessmentDecisionPurpose, "", fingerprint)
	if err != nil || !reserved {
		t.Fatalf("reserved=%v err=%v, want a charged reservation", reserved, err)
	}
	if reserved, err := db.ReserveReassessmentCall(ctx, sealed.Cycle.CycleID, factory.ReassessmentDeciding, call, factory.ReassessmentDecisionPurpose, "", fingerprint); err != nil || reserved {
		t.Fatalf("reserved=%v err=%v, want reconcile without recharge", reserved, err)
	}
	if _, err := db.db.ExecContext(ctx, `UPDATE factory_judgment_budget SET spent=1000 WHERE id=1`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ReserveReassessmentCall(ctx, sealed.Cycle.CycleID, factory.ReassessmentDeciding, factory.NewID(), factory.ReassessmentDecisionPurpose, "", strings.Repeat("1", 64)); !errors.Is(err, ErrReassessmentBudgetExhausted) {
		t.Fatalf("exhausted budget: %v", err)
	}

	// The fenced repository refuses packet admission while the open
	// fence stands: no early admission through any packet path.
	a, r, run, view := dispatchTestPacket(t, time.Now().UTC())
	a.Issue, view.Issue = 4, 4
	if err := recordDispatchTestPacket(t, ctx, db, dispatchTestRegistration(a), a, r, run, view); !errors.Is(err, ErrCompletionFenced) {
		t.Fatalf("packet past the fence: %v", err)
	}
}

func TestCompletionRetrySuccession(t *testing.T) {
	db := dispatchStoreFixture(t)
	ctx := context.Background()
	m := mergedCompletionRecord(factory.NewID(), "601")
	if !recordTestCompletion(t, db, m) {
		t.Fatal("completion not recorded")
	}
	cycles, err := db.PendingReassessmentCycles(ctx, 7)
	if err != nil || len(cycles) != 1 {
		t.Fatalf("cycles=%v err=%v", cycles, err)
	}
	prior := cycles[0].Cycle.CycleID
	attention := cycles[0]
	attention.Cycle.State = factory.ReassessmentAttention
	attention.WaitReason = "decision_unavailable"
	if err := db.CasReassessmentRecord(ctx, prior, factory.ReassessmentCollecting, attention); err != nil {
		t.Fatalf("attention CAS: %v", err)
	}
	retry := factory.ReassessmentRetry{
		SuccessorCycleID: factory.NewID(), ExpectedPriorCycle: prior,
		CommandID: factory.NewID(), EventID: cycles[0].Cycle.EventID,
	}
	successor, err := db.StartReassessmentRetry(ctx, retry)
	if err != nil || successor != retry.SuccessorCycleID {
		t.Fatalf("successor=%q err=%v", successor, err)
	}
	// Command replay reuses the recorded successor.
	again, err := db.StartReassessmentRetry(ctx, retry)
	if err != nil || again != successor {
		t.Fatalf("replay=%q err=%v, want the recorded successor", again, err)
	}
	foreign := retry
	foreign.SuccessorCycleID = factory.NewID()
	if _, err := db.StartReassessmentRetry(ctx, foreign); err == nil {
		t.Fatal("second successor admitted for one prior cycle")
	}
}
