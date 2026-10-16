package factory

import (
	"strings"
	"testing"
	"time"
)

func stallTestDigest(seed string) string {
	return strings.Repeat(seed, 64)[:64]
}

func validStallEvent() StallEvent {
	return StallEvent{
		ID:                 "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Repository:         7,
		ProgressGeneration: 3,
		InputFingerprint:   stallTestDigest("1"),
		InventoryDigest:    stallTestDigest("2"),
		SourceCommit:       strings.Repeat("c", 40),
		PreparationDigest:  stallTestDigest("3"),
		Reason:             StallReasonCycle,
		EpisodeID:          "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		State:              StallObserved,
	}
}

func TestStallTaxonomyValidators(t *testing.T) {
	for _, reason := range []string{StallReasonCycle, StallReasonMissingProducer, StallReasonUnexplainedNoReady} {
		if !ValidStallReason(reason) {
			t.Fatalf("reason %q rejected", reason)
		}
	}
	if ValidStallReason("repository-wide") || ValidStallReason("") {
		t.Fatal("unbounded reason accepted")
	}
	for _, kind := range []string{
		StallWaitActiveWork, StallWaitCIChecks, StallWaitCapacity, StallWaitQuota,
		StallWaitPreparation, StallWaitPermissions, StallWaitExplicitPause,
		StallWaitProductChoice, StallWaitPendingCompletion,
	} {
		if !ValidStallWaitKind(kind) {
			t.Fatalf("wait kind %q rejected", kind)
		}
	}
	if ValidStallWaitKind("investigate") || ValidStallWaitKind("") {
		t.Fatal("unbounded wait kind accepted")
	}
	if !ValidStallState(StallClassifiedWait) || !ValidStallState(StallReleased) || ValidStallState("repairing") {
		t.Fatal("stall state validation wrong")
	}
}

func TestStallEventValidation(t *testing.T) {
	if err := validStallEvent().Validate(); err != nil {
		t.Fatalf("valid event: %v", err)
	}
	bad := validStallEvent()
	bad.Reason = "repository-wide"
	if err := bad.Validate(); err == nil {
		t.Fatal("repository-wide boolean accepted as stall reason")
	}
	bad = validStallEvent()
	bad.InputFingerprint = "short"
	if err := bad.Validate(); err == nil {
		t.Fatal("unfingerprinted event accepted")
	}
	subject := StallSubject{
		StallID:         "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		CycleID:         "cccccccccccccccccccccccccccccccc",
		InventoryDigest: stallTestDigest("4"),
	}
	if err := subject.Validate(); err != nil {
		t.Fatalf("valid subject: %v", err)
	}
	if StallKey(subject.StallID) != "stall:"+subject.StallID {
		t.Fatal("stall subject key outside its namespace")
	}
}

func validRepairGrant() RepairGrant {
	grant := DefaultRepairGrant()
	grant.ID = "dddddddddddddddddddddddddddddddd"
	grant.Repository = 7
	grant.Sponsor = 5
	grant.CreatedUnix = 1000
	return grant
}

func stallCharge(id, episode string, cycle int, kind, evidence string, minutes int) RepairCharge {
	return RepairCharge{
		ID: id, GrantID: "dddddddddddddddddddddddddddddddd",
		Kind: kind, EpisodeID: episode, Cycle: cycle,
		EvidenceDigest: stallTestDigest(evidence), ActiveMinutes: minutes, SpentUnix: 2000,
	}
}

func TestRepairAccountSpendOnceAndCaps(t *testing.T) {
	account := RepairAccount{Grant: validRepairGrant()}
	episode := "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"
	first := stallCharge("f0000000000000000000000000000001", episode, 1, RepairChargeRun, "a", 30)
	next, spent, err := account.AdmitCharge(first)
	if err != nil || !spent {
		t.Fatalf("first charge: spent=%v err=%v", spent, err)
	}
	account = next
	replay, spent, err := account.AdmitCharge(first)
	if err != nil || spent {
		t.Fatalf("replay: spent=%v err=%v, unchanged facts must spend once", spent, err)
	}
	if len(replay.Charges) != 1 {
		t.Fatal("replay recorded a duplicate charge")
	}
	// One cycle holds one run and one judgment; the second run of the
	// same cycle is refused even though the grant totals show room.
	second := stallCharge("f0000000000000000000000000000002", episode, 1, RepairChargeRun, "a", 10)
	if _, _, err = account.AdmitCharge(second); err == nil {
		t.Fatal("second run inside one cycle admitted")
	}
	judgment := stallCharge("f0000000000000000000000000000003", episode, 1, RepairChargeJudgment, "a", 0)
	next, spent, err = account.AdmitCharge(judgment)
	if err != nil || !spent {
		t.Fatalf("cycle judgment: spent=%v err=%v", spent, err)
	}
	account = next
	// Exhaust the four investigator runs across episodes, then prove
	// the exhausted replay still spends nothing instead of erroring.
	episode2 := "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeef"
	runs := []RepairCharge{
		stallCharge("f0000000000000000000000000000004", episode, 2, RepairChargeRun, "b", 30),
		stallCharge("f0000000000000000000000000000005", episode2, 1, RepairChargeRun, "a", 10),
		stallCharge("f0000000000000000000000000000006", episode2, 2, RepairChargeRun, "b", 10),
	}
	for i, charge := range runs {
		next, spent, err = account.AdmitCharge(charge)
		if err != nil || !spent {
			t.Fatalf("run %d: spent=%v err=%v", i, spent, err)
		}
		account = next
	}
	episode3 := "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeee0"
	if _, spent, err = account.AdmitCharge(stallCharge("f0000000000000000000000000000009", episode3, 1, RepairChargeRun, "a", 1)); err == nil || spent {
		t.Fatal("fifth investigator run admitted past the finite bound")
	}
	if _, _, err = account.AdmitCharge(first); err != nil {
		t.Fatalf("exhausted replay must stay free, got %v", err)
	}
}

func TestRepairAccountSecondCycleNeedsNewEvidence(t *testing.T) {
	account := RepairAccount{Grant: validRepairGrant()}
	episode := "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"
	next, _, err := account.AdmitCharge(stallCharge("f0000000000000000000000000000011", episode, 1, RepairChargeRun, "a", 30))
	if err != nil {
		t.Fatal(err)
	}
	account = next
	// A reshuffled second cycle over identical facts is refused.
	if _, _, err = account.AdmitCharge(stallCharge("f0000000000000000000000000000012", episode, 2, RepairChargeRun, "a", 30)); err == nil {
		t.Fatal("second cycle over unchanged facts admitted")
	}
	next, spent, err := account.AdmitCharge(stallCharge("f0000000000000000000000000000013", episode, 2, RepairChargeRun, "b", 30))
	if err != nil || !spent {
		t.Fatalf("evidenced second cycle: spent=%v err=%v", spent, err)
	}
	account = next
	// A third cycle never exists: two unsuccessful cycles need owner attention.
	if _, _, err = account.AdmitCharge(stallCharge("f0000000000000000000000000000014", episode, 3, RepairChargeRun, "c", 30)); err == nil {
		t.Fatal("third automatic cycle admitted")
	}
	// Judgment bounds hold independently of cycles: one judgment per
	// cycle across episodes, four total for the grant.
	full := RepairAccount{Grant: validRepairGrant()}
	judgments := []RepairCharge{
		stallCharge("f0000000000000000000000000000021", "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeef", 1, RepairChargeJudgment, "a", 0),
		stallCharge("f0000000000000000000000000000022", "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeef", 2, RepairChargeJudgment, "b", 0),
		stallCharge("f0000000000000000000000000000023", "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeee0", 1, RepairChargeJudgment, "a", 0),
		stallCharge("f0000000000000000000000000000024", "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeee0", 2, RepairChargeJudgment, "b", 0),
	}
	for i, charge := range judgments {
		next, spent, err := full.AdmitCharge(charge)
		if err != nil || !spent {
			t.Fatalf("judgment %d: spent=%v err=%v", i, spent, err)
		}
		full = next
	}
	if _, _, err = full.AdmitCharge(stallCharge("f0000000000000000000000000000030", "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeee1", 1, RepairChargeJudgment, "a", 0)); err == nil {
		t.Fatal("fifth proposal judgment admitted")
	}
	_ = account
}

func TestRepairGrantBoundsAndInactive(t *testing.T) {
	grant := validRepairGrant()
	grant.InvestigatorRuns = 5
	if err := grant.Validate(); err == nil {
		t.Fatal("grant above the frozen run bound accepted")
	}
	grant = validRepairGrant()
	grant.Active = false
	account := RepairAccount{Grant: grant}
	if _, _, err := account.AdmitCharge(stallCharge("f0000000000000000000000000000041", "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee", 1, RepairChargeRun, "a", 1)); err == nil {
		t.Fatal("charge admitted without an active explicit grant")
	}
}

func validIntentAggregate() IntentExecutionAllowance {
	aggregate := DefaultIntentExecutionAllowance()
	aggregate.IntentID = "intent-1"
	aggregate.OwnerID = 5
	aggregate.RepositoryID = 7
	aggregate.CheckpointUnix = 1000
	return aggregate
}

func TestIntentAggregateChargeOnceAndNeverRefill(t *testing.T) {
	aggregate := validIntentAggregate()
	if aggregate.ActiveLimitMs != int64(DefaultIntentActiveMinutes)*60*1000 {
		t.Fatal("default aggregate is not 480 active minutes")
	}
	var intervals []intentInterval
	next, nextIntervals, spent, err := aggregate.ChargeInterval(intervals, "run-1", 1000, 1300)
	if err != nil || !spent || next.ConsumedMs != 300*1000 {
		t.Fatalf("first parent interval: spent=%v err=%v %+v", spent, err, next)
	}
	// The same run identity replays free: retries never refill or double spend.
	replay, _, spent, err := next.ChargeInterval(nextIntervals, "run-1", 1000, 1300)
	if err != nil || spent || replay.ConsumedMs != next.ConsumedMs {
		t.Fatalf("replay: spent=%v err=%v", spent, err)
	}
	// Overlapping native-worker reports of one parent interval charge once.
	union, unionIntervals, spent, err := next.ChargeInterval(nextIntervals, "run-2", 1200, 1500)
	if err != nil || !spent {
		t.Fatalf("overlap: spent=%v err=%v", spent, err)
	}
	if union.ConsumedMs != 500*1000 {
		t.Fatalf("consumed=%d, want the 500s union charged once", union.ConsumedMs)
	}
	// Queued work consumes no active time.
	queued, _, spent, err := union.ChargeInterval(unionIntervals, "run-3", 2000, 2000)
	if err != nil || spent || queued.ConsumedMs != union.ConsumedMs {
		t.Fatalf("queued: spent=%v err=%v", spent, err)
	}
	// An explicit increase retains spent time instead of refilling.
	raised, err := union.ApplyIncrease(union.ActiveLimitMs + 60*1000)
	if err != nil || raised.ConsumedMs != union.ConsumedMs {
		t.Fatalf("increase: err=%v", err)
	}
	if _, err = union.ApplyIncrease(union.ActiveLimitMs - 1); err == nil {
		t.Fatal("limit decrease accepted as an increase")
	}
}

func TestIntentAggregatePauseNeedsObservedStop(t *testing.T) {
	aggregate := validIntentAggregate()
	now := time.Unix(1100, 0)
	running, err := aggregate.Checkpoint(now, true, true)
	if err != nil || !running.Active {
		t.Fatalf("activate: %v", err)
	}
	// Pausing while runs are still unretired keeps the clock running.
	held, err := running.Checkpoint(now.Add(time.Minute), false, false)
	if err != nil || !held.Active {
		t.Fatalf("unretired pause stopped the clock: %+v %v", held, err)
	}
	if held.ConsumedMs != 0 {
		t.Fatalf("held pause consumed %d", held.ConsumedMs)
	}
	stopped, err := running.Checkpoint(now.Add(time.Minute), false, true)
	if err != nil || stopped.Active {
		t.Fatalf("retired pause: %+v %v", stopped, err)
	}
	if stopped.ConsumedMs != 60*1000 {
		t.Fatalf("retired pause consumed %d, want 60000", stopped.ConsumedMs)
	}
}

func TestIntentAggregateBounds(t *testing.T) {
	aggregate := validIntentAggregate()
	aggregate.ActiveLimitMs = 0
	if err := aggregate.Validate(); err == nil {
		t.Fatal("zero limit accepted")
	}
	aggregate = validIntentAggregate()
	aggregate.ActiveLimitMs = int64(MaxIntentActiveMinutes+1) * 60 * 1000
	if err := aggregate.Validate(); err == nil {
		t.Fatal("limit above 10,080 minutes accepted")
	}
}
