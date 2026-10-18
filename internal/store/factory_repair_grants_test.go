package store

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
)

// repairTestDigest returns a deterministic 64-hex evidence digest.
func repairTestDigest(seed byte) string {
	return strings.Repeat(string([]byte{seed}), 64)
}

// repairGrantFixture builds one grant command pair for tests.
func repairGrantFixture(id, predecessor string, revision int64, repo, sponsor int64, runs, judgments, minutes int, effect string) (factory.Command, factory.RepairGrant) {
	grant := factory.RepairGrant{
		ID: id, Repository: repo, Sponsor: sponsor, Revision: revision, Predecessor: predecessor,
		InvestigatorRuns: runs, ProposalJudgments: judgments, ActiveMinutes: minutes,
		Effect: effect, Active: true, CreatedUnix: 1000,
	}
	raw, err := json.Marshal(grant)
	if err != nil {
		panic(err)
	}
	cmd := factory.Command{
		ID: id, Type: factory.CommandRepairGrant, Target: factory.RepairTarget(repo),
		Principal: "soda-tester", Payload: string(raw),
	}
	cmd.Digest = factory.SettingsDigest(cmd.Type, cmd.Target, cmd.Payload)
	return cmd, grant
}

func repairChargeFixture(id, grant, episode string, cycle int, kind, evidence string, minutes int, failed bool) factory.RepairCharge {
	return factory.RepairCharge{
		ID: id, GrantID: grant, Kind: kind, EpisodeID: episode, Cycle: cycle,
		EvidenceDigest: repairTestDigest(evidence[0]), ActiveMinutes: minutes,
		SpentUnix: 2000, Failed: failed,
	}
}

// T57 on the real store path: duplicate ticks, restarts, hash changes,
// issue replacement, graph reshuffles and retries never replenish
// room; unchanged facts spend once; at most two automatic cycles with
// materially new evidence; repository bounds hold with a separate
// sponsor scope.
func TestRepairGrantAdmissionAndNoRefillPG(t *testing.T) {
	ctx := context.Background()
	db := dispatchStoreFixture(t)
	now := time.Now().UTC()
	const repo = 7
	grantID := "d0000000000000000000000000000001"
	cmd, grant := repairGrantFixture(grantID, "", 0, repo, 7, 4, 4, 120, "")
	if _, created, err := db.ApplyRepairGrantCommand(ctx, cmd, grant, now); err != nil || !created {
		t.Fatalf("apply grant: created=%v err=%v", created, err)
	}
	// Replaying the identical command replays instead of duplicating.
	if _, created, err := db.ApplyRepairGrantCommand(ctx, cmd, grant, now); err != nil || created {
		t.Fatalf("grant replay: created=%v err=%v", created, err)
	}
	// A revision that skips the predecessor chain is stale.
	badCmd, badGrant := repairGrantFixture("d0000000000000000000000000000002", "d0000000000000000000000000000099", 1, repo, 7, 4, 4, 120, "raise")
	if _, _, err := db.ApplyRepairGrantCommand(ctx, badCmd, badGrant, now); !errors.Is(err, ErrStaleRevision) {
		t.Fatalf("stale grant chain: err=%v", err)
	}
	episode := "e0000000000000000000000000000001"
	first := repairChargeFixture("c0000000000000000000000000000001", grantID, episode, 1, factory.RepairChargeRun, "a", 30, false)
	receipt, spent, err := db.AdmitStallInvestigation(ctx, repo, first, now)
	if err != nil || !spent || receipt.ChargeID != first.ID {
		t.Fatalf("first charge: spent=%v err=%v", spent, err)
	}
	// A restart replaying the same run identity spends nothing.
	if _, spent, err := db.AdmitStallInvestigation(ctx, repo, first, now); err != nil || spent {
		t.Fatalf("restart replay: spent=%v err=%v", spent, err)
	}
	// A reshuffled second cycle over identical facts is refused.
	reshuffle := repairChargeFixture("c0000000000000000000000000000003", grantID, episode, 2, factory.RepairChargeRun, "a", 30, false)
	if _, _, err := db.AdmitStallInvestigation(ctx, repo, reshuffle, now); !errors.Is(err, factory.ErrRepairNeedsEvidence) {
		t.Fatalf("reshuffle without new facts: err=%v", err)
	}
	// Materially new factual evidence opens exactly one second cycle;
	// failure stays charged and counts against the bounds.
	second := repairChargeFixture("c0000000000000000000000000000004", grantID, episode, 2, factory.RepairChargeRun, "b", 30, true)
	if _, spent, err := db.AdmitStallInvestigation(ctx, repo, second, now); err != nil || !spent {
		t.Fatalf("evidenced second cycle: spent=%v err=%v", spent, err)
	}
	judgment := repairChargeFixture("c0000000000000000000000000000041", grantID, episode, 2, factory.RepairChargeJudgment, "b", 0, false)
	if _, spent, err := db.AdmitStallInvestigation(ctx, repo, judgment, now); err != nil || !spent {
		t.Fatalf("second-cycle judgment: spent=%v err=%v", spent, err)
	}
	// Two unsuccessful cycles need owner attention: no further
	// automatic charge fits the episode, whatever its evidence.
	third := repairChargeFixture("c0000000000000000000000000000005", grantID, episode, 2, factory.RepairChargeRun, "c", 30, false)
	if _, _, err := db.AdmitStallInvestigation(ctx, repo, third, now); !errors.Is(err, factory.ErrRepairCyclesExhausted) {
		t.Fatalf("third charge: err=%v", err)
	}
	charges, err := db.RepairCharges(ctx, repo)
	if err != nil || len(charges) != 3 {
		t.Fatalf("charges=%d err=%v, want the three spent charges", len(charges), err)
	}
	// The remaining runs fill the grant totals across episodes; the
	// fifth exceeds the bound while the exhausted replay stays free.
	episode2 := "e0000000000000000000000000000002"
	for _, charge := range []factory.RepairCharge{
		repairChargeFixture("c0000000000000000000000000000006", grantID, episode2, 1, factory.RepairChargeRun, "a", 30, false),
		repairChargeFixture("c0000000000000000000000000000007", grantID, episode2, 2, factory.RepairChargeRun, "b", 30, false),
	} {
		if _, spent, err := db.AdmitStallInvestigation(ctx, repo, charge, now); err != nil || !spent {
			t.Fatalf("charge %s: spent=%v err=%v", charge.ID, spent, err)
		}
	}
	fifth := repairChargeFixture("c0000000000000000000000000000042", grantID, "e0000000000000000000000000000003", 1, factory.RepairChargeRun, "a", 1, false)
	if _, _, err := db.AdmitStallInvestigation(ctx, repo, fifth, now); !errors.Is(err, factory.ErrRepairRunsExhausted) {
		t.Fatalf("fifth run: err=%v", err)
	}
	if _, spent, err := db.AdmitStallInvestigation(ctx, repo, first, now); err != nil || spent {
		t.Fatalf("exhausted replay: spent=%v err=%v", spent, err)
	}
	// Judgment calls hold their own finite bound: one per cycle across
	// episodes, four total for the grant.
	episode4 := "e0000000000000000000000000000004"
	for _, charge := range []factory.RepairCharge{
		repairChargeFixture("c0000000000000000000000000000031", grantID, episode4, 1, factory.RepairChargeJudgment, "a", 0, false),
		repairChargeFixture("c0000000000000000000000000000032", grantID, episode4, 2, factory.RepairChargeJudgment, "b", 0, false),
		repairChargeFixture("c0000000000000000000000000000033", grantID, episode2, 1, factory.RepairChargeJudgment, "a", 0, false),
	} {
		if _, spent, err := db.AdmitStallInvestigation(ctx, repo, charge, now); err != nil || !spent {
			t.Fatalf("judgment %s: spent=%v err=%v", charge.ID, spent, err)
		}
	}
	extra := repairChargeFixture("c0000000000000000000000000000019", grantID, episode2, 2, factory.RepairChargeJudgment, "b", 0, false)
	if _, _, err := db.AdmitStallInvestigation(ctx, repo, extra, now); !errors.Is(err, factory.ErrRepairJudgmentsExhausted) {
		t.Fatalf("fifth judgment: err=%v", err)
	}
}

// T57 time bound in isolation: active minutes exhaust independently of
// the run count.
func TestRepairActiveTimeBoundPG(t *testing.T) {
	ctx := context.Background()
	db := dispatchStoreFixture(t)
	now := time.Now().UTC()
	const repo = 77
	grantID := "d0000000000000000000000000000011"
	cmd, grant := repairGrantFixture(grantID, "", 0, repo, 7, 4, 4, 80, "")
	if _, _, err := db.ApplyRepairGrantCommand(ctx, cmd, grant, now); err != nil {
		t.Fatal(err)
	}
	episode := "e0000000000000000000000000000011"
	first := repairChargeFixture("c0000000000000000000000000000011", grantID, episode, 1, factory.RepairChargeRun, "a", 30, false)
	if _, spent, err := db.AdmitStallInvestigation(ctx, repo, first, now); err != nil || !spent {
		t.Fatalf("first run: spent=%v err=%v", spent, err)
	}
	second := repairChargeFixture("c0000000000000000000000000000014", grantID, episode, 2, factory.RepairChargeRun, "b", 30, false)
	if _, spent, err := db.AdmitStallInvestigation(ctx, repo, second, now); err != nil || !spent {
		t.Fatalf("second run: spent=%v err=%v", spent, err)
	}
	episode2 := "e0000000000000000000000000000015"
	over := repairChargeFixture("c0000000000000000000000000000012", grantID, episode2, 1, factory.RepairChargeRun, "a", 30, false)
	if _, _, err := db.AdmitStallInvestigation(ctx, repo, over, now); !errors.Is(err, factory.ErrRepairTimeExhausted) {
		t.Fatalf("over-time run: err=%v", err)
	}
	fitting := repairChargeFixture("c0000000000000000000000000000013", grantID, episode2, 1, factory.RepairChargeRun, "a", 20, false)
	if _, spent, err := db.AdmitStallInvestigation(ctx, repo, fitting, now); err != nil || !spent {
		t.Fatalf("fitting run: spent=%v err=%v", spent, err)
	}
}

// T57 grant increase: an explicit authorized raise opens new room while
// preserving every spent charge and the prior history.
func TestRepairGrantIncreasePreservesConsumptionPG(t *testing.T) {
	ctx := context.Background()
	db := dispatchStoreFixture(t)
	now := time.Now().UTC()
	const repo = 78
	grantID := "d0000000000000000000000000000021"
	cmd, grant := repairGrantFixture(grantID, "", 0, repo, 7, 2, 1, 60, "")
	if _, _, err := db.ApplyRepairGrantCommand(ctx, cmd, grant, now); err != nil {
		t.Fatal(err)
	}
	episode := "e0000000000000000000000000000021"
	for _, charge := range []factory.RepairCharge{
		repairChargeFixture("c0000000000000000000000000000021", grantID, episode, 1, factory.RepairChargeRun, "a", 10, false),
		repairChargeFixture("c0000000000000000000000000000022", grantID, episode, 2, factory.RepairChargeRun, "b", 10, false),
	} {
		if _, spent, err := db.AdmitStallInvestigation(ctx, repo, charge, now); err != nil || !spent {
			t.Fatalf("charge %s: spent=%v err=%v", charge.ID, spent, err)
		}
	}
	capped := repairChargeFixture("c0000000000000000000000000000023", grantID, "e0000000000000000000000000000025", 1, factory.RepairChargeRun, "a", 10, false)
	if _, _, err := db.AdmitStallInvestigation(ctx, repo, capped, now); !errors.Is(err, factory.ErrRepairRunsExhausted) {
		t.Fatalf("capped run: err=%v", err)
	}
	// An unexplained raise is refused; a stated raise preserves history.
	silentCmd, silent := repairGrantFixture("d0000000000000000000000000000022", grantID, 1, repo, 7, 4, 1, 60, "")
	if _, _, err := db.ApplyRepairGrantCommand(ctx, silentCmd, silent, now); err == nil {
		t.Fatal("silent limit raise admitted")
	}
	raisedCmd, raised := repairGrantFixture("d0000000000000000000000000000022", grantID, 1, repo, 7, 4, 1, 60, "owner authorizes two further investigator runs")
	if _, created, err := db.ApplyRepairGrantCommand(ctx, raisedCmd, raised, now); err != nil || !created {
		t.Fatalf("stated raise: created=%v err=%v", created, err)
	}
	// New charges bind the current head; the two pre-raise charges
	// stay valid history against the same scope. The refused capped
	// charge never persisted, so its identity is reusable.
	rebound := repairChargeFixture("c0000000000000000000000000000023", raisedCmd.ID, "e0000000000000000000000000000025", 1, factory.RepairChargeRun, "a", 10, false)
	if _, spent, err := db.AdmitStallInvestigation(ctx, repo, rebound, now); err != nil || !spent {
		t.Fatalf("post-raise charge: spent=%v err=%v", spent, err)
	}
	charges, err := db.RepairCharges(ctx, repo)
	if err != nil || len(charges) != 3 {
		t.Fatalf("charges=%d err=%v, want all three preserved", len(charges), err)
	}
}

// intentRevisionFixture builds one valid accepted revision for aggregate tests.
func intentRevisionFixture(id, predecessor, taskKey string, repo int64) factory.IntentRevision {
	task := factory.IntentTask{Key: taskKey, Objective: "Build request form", Outcomes: []string{"Submit a request"}, IdeaIDs: []string{"d333333333333333333333333"}}
	return factory.IntentRevision{
		ID: id, Intent: "d222222222222222222222222", Predecessor: predecessor,
		Owner: 1, Repository: repo, Objective: "Build a request form", State: "accepted",
		MaxIssues: 4, CreatedUnix: 1,
		Ideas: []factory.ProductIdea{{ID: "d333333333333333333333333", Description: "Request form", Consequence: "Requests are recorded", Decision: "accept", Source: "requested", Provenance: "human", Actor: 1}},
		Tasks: []factory.IntentTask{task},
	}
}

// bumpAttemptSeconds records consumed active seconds through the
// production allowance saver, mirroring settled execution.
func bumpAttemptSeconds(t *testing.T, ctx context.Context, db *Store, repo, issue int64, seconds int64) {
	t.Helper()
	allowance, err := db.AttemptAllowance(ctx, repo, issue)
	if err != nil {
		t.Fatal(err)
	}
	allowance.ActiveSeconds = seconds
	allowance.CheckpointUnix = time.Now().Unix()
	allowance.Revision++
	tx, err := db.db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback() }()
	if err = saveAttemptAllowanceTx(ctx, tx, allowance); err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
}

// T60 on the real store path: new attempts, decomposition revisions,
// split or replacement issues and verification cannot refill stable
// intent usage; an explicit increase retains spent time.
func TestIntentExecutionAggregatePG(t *testing.T) {
	ctx := context.Background()
	db := dispatchStoreFixture(t)
	now := time.Now().UTC()
	const repo = 7
	rev1 := intentRevisionFixture("d111111111111111111111111", "", "d444444444444444444444444", repo)
	if err := db.SaveIntentRevision(ctx, rev1); err != nil {
		t.Fatalf("save rev1: %v", err)
	}
	a, reservation, run, view := dispatchTestPacket(t, now)
	if err := recordDispatchTestPacket(t, ctx, db, dispatchTestRegistration(a), a, reservation, run, view); err != nil {
		t.Fatalf("packet issue 3: %v", err)
	}
	bumpAttemptSeconds(t, ctx, db, repo, 3, 300)
	if err := db.BindIntentTaskAcceptance(ctx, factory.IntentTaskBinding{
		RevisionID: rev1.ID, TaskKey: rev1.Tasks[0].Key, IssueID: 3, IssueIndex: 3,
		AcceptanceID: "d555555555555555555555555",
	}); err != nil {
		t.Fatalf("bind issue 3: %v", err)
	}
	aggregate, err := db.IntentExecutionAggregateForIntent(ctx, rev1.Intent, 1)
	if err != nil {
		t.Fatalf("aggregate: %v", err)
	}
	if aggregate.ConsumedMs != 300*1000 || len(aggregate.Roots) != 1 || len(aggregate.Issues) != 1 {
		t.Fatalf("aggregate=%+v, want the 300s origin root charged once", aggregate)
	}
	// A decomposition revision keeps the origin aggregate: the revision
	// binds a split issue whose fresh row adds nothing and resets nothing.
	rev2 := intentRevisionFixture("d666666666666666666666666", rev1.ID, "d444444444444444444444444", repo)
	if err := db.SaveIntentRevision(ctx, rev2); err != nil {
		t.Fatalf("save rev2: %v", err)
	}
	if err := db.BindIntentTaskAcceptance(ctx, factory.IntentTaskBinding{
		RevisionID: rev2.ID, TaskKey: rev2.Tasks[0].Key, IssueID: 4, IssueIndex: 4,
		AcceptanceID: "d777777777777777777777777",
	}); err != nil {
		t.Fatalf("bind split issue 4: %v", err)
	}
	aggregate, err = db.IntentExecutionAggregateForIntent(ctx, rev1.Intent, 1)
	if err != nil {
		t.Fatalf("aggregate after split: %v", err)
	}
	if aggregate.ConsumedMs != 300*1000 {
		t.Fatalf("split refilled or dropped usage: %+v", aggregate)
	}
	if len(aggregate.Issues) != 2 {
		t.Fatalf("aggregate issues=%v, want both bound issues", aggregate.Issues)
	}
	// The aggregate bound records beside funding; an explicit increase
	// retains spent time instead of refilling.
	limit := factory.DefaultIntentExecutionAllowance()
	limit.IntentID, limit.OwnerID, limit.RepositoryID = rev1.Intent, 1, repo
	limit.CheckpointUnix = now.Unix()
	raw, err := json.Marshal(limit)
	if err != nil {
		t.Fatal(err)
	}
	limitCmd := factory.Command{
		ID: factory.NewID(), Type: factory.CommandIntentExecutionLimit,
		Target:    factory.IntentExecutionTarget(rev1.Intent),
		Principal: "soda-tester", Payload: string(raw),
	}
	limitCmd.Digest = factory.SettingsDigest(limitCmd.Type, limitCmd.Target, limitCmd.Payload)
	if _, created, err := db.ApplyIntentExecutionLimitCommand(ctx, limitCmd, limit, now); err != nil || !created {
		t.Fatalf("apply limit: created=%v err=%v", created, err)
	}
	raised := limit
	raised.Revision = 1
	raised.Predecessor = limitCmd.ID
	next, err := raised.ApplyIncrease(raised.ActiveLimitMs + 120*60*1000)
	if err != nil {
		t.Fatalf("domain increase: %v", err)
	}
	raised = next
	raw, err = json.Marshal(raised)
	if err != nil {
		t.Fatal(err)
	}
	raisedCmd := factory.Command{
		ID: factory.NewID(), Type: factory.CommandIntentExecutionLimit,
		Target:    factory.IntentExecutionTarget(rev1.Intent),
		Principal: "soda-tester", Payload: string(raw),
	}
	raisedCmd.Digest = factory.SettingsDigest(raisedCmd.Type, raisedCmd.Target, raisedCmd.Payload)
	if _, created, err := db.ApplyIntentExecutionLimitCommand(ctx, raisedCmd, raised, now); err != nil || !created {
		t.Fatalf("raise limit: created=%v err=%v", created, err)
	}
	aggregate, err = db.IntentExecutionAggregateForIntent(ctx, rev1.Intent, 1)
	if err != nil {
		t.Fatalf("aggregate after raise: %v", err)
	}
	if aggregate.ConsumedMs != 300*1000 || aggregate.LimitMs != raised.ActiveLimitMs {
		t.Fatalf("raise reset usage: %+v", aggregate)
	}
	// A shrinking bound is refused.
	shrunk := raised
	shrunk.Revision = 2
	shrunk.Predecessor = raisedCmd.ID
	shrunk.ActiveLimitMs = limit.ActiveLimitMs
	raw, err = json.Marshal(shrunk)
	if err != nil {
		t.Fatal(err)
	}
	shrunkCmd := factory.Command{
		ID: factory.NewID(), Type: factory.CommandIntentExecutionLimit,
		Target:    factory.IntentExecutionTarget(rev1.Intent),
		Principal: "soda-tester", Payload: string(raw),
	}
	shrunkCmd.Digest = factory.SettingsDigest(shrunkCmd.Type, shrunkCmd.Target, shrunkCmd.Payload)
	if _, _, err := db.ApplyIntentExecutionLimitCommand(ctx, shrunkCmd, shrunk, now); err == nil {
		t.Fatal("shrinking aggregate bound admitted")
	}
}

// TestAllIssueControlsOverflowFailsClosed proves the inventory bound:
// a set past the limit returns no controls and reports incomplete, so
// no caller can prove deadlock from a truncated set. Bounded cost is
// preserved by the limit+1 probe row.
func TestAllIssueControlsOverflowFailsClosed(t *testing.T) {
	ctx := context.Background()
	db := dispatchStoreFixture(t)
	now := time.Now().UTC()
	for issue := int64(1); issue <= 3; issue++ {
		control := factory.IssueControl{
			Repository: 31, Issue: issue,
			Readiness: factory.ReadinessQueued, Reason: factory.ReasonEligible,
			Fingerprint: repairTestDigest(byte('0' + issue)), Authority: repairTestDigest(byte('a' + issue)),
		}
		if _, _, err := db.RecordIssueAssessment(ctx, control, now); err != nil {
			t.Fatalf("seed control %d: %v", issue, err)
		}
	}
	if _, _, err := db.AllIssueControls(ctx, 0); err == nil {
		t.Fatal("zero inventory bound admitted")
	}
	controls, complete, err := db.AllIssueControls(ctx, 2)
	if err != nil || complete || controls != nil {
		t.Fatalf("overflow: complete=%v controls=%d err=%v, want incomplete with no set", complete, len(controls), err)
	}
	controls, complete, err = db.AllIssueControls(ctx, 3)
	if err != nil || !complete || len(controls) != 3 {
		t.Fatalf("exact bound: complete=%v controls=%d err=%v", complete, len(controls), err)
	}
}

// TestCompletionWatermarkStartsZeroAndAdvances proves the verified-
// delivery generation source behind stall episode binding: empty when
// nothing completed, advancing only through recorded completions.
func TestCompletionWatermarkStartsZeroAndAdvances(t *testing.T) {
	ctx := context.Background()
	db := dispatchStoreFixture(t)
	if mark, err := db.CompletionWatermark(ctx, 31); err != nil || mark != 0 {
		t.Fatalf("fresh watermark=%d err=%v, want zero", mark, err)
	}
	event := factory.CompletionEvent{
		ID: factory.NewID(), PublicationID: factory.NewID(),
		Repository: 31, Issue: 1, Generation: 1,
		AcceptanceID: "d" + strings.Repeat("e", 24),
		MergedCommit: strings.Repeat("f", 40), ClosingOccurrence: "native closed the issue",
	}
	eventData, err := json.Marshal(event)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.db.ExecContext(ctx, `INSERT INTO factory_completions(repository,generation,event,publication,occurrence,data)
		VALUES($1,$2,$3,$4,$5,$6)`, event.Repository, event.Generation, event.ID, event.PublicationID, event.ClosingOccurrence, string(eventData)); err != nil {
		t.Fatalf("seed completion: %v", err)
	}
	if mark, err := db.CompletionWatermark(ctx, 31); err != nil || mark != 1 {
		t.Fatalf("watermark=%d err=%v, want the verified delivery", mark, err)
	}
	if mark, err := db.CompletionWatermark(ctx, 32); err != nil || mark != 0 {
		t.Fatalf("other repository watermark=%d err=%v, want zero", mark, err)
	}
}
