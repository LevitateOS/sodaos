package control

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
)

// TestServeStallRoundAdmitsBoundedEpisode proves the R36 production
// joint on the real store path: a no-launch dispatch pass runs
// detection at the standing generation, persists the bound episode
// durably, and admits exactly one cycle-1 charge under the explicit
// grant. A redetected round replays without spending; a pass that
// launched work skips detection entirely.
func TestServeStallRoundAdmitsBoundedEpisode(t *testing.T) {
	ctx := context.Background()
	db, _ := postgresFixture(t, nil)
	coord := &Coordinator{Store: db}
	const repo = 44
	grantID := "d1000000000000000000000000000044"
	grant := factory.RepairGrant{
		ID: grantID, Repository: repo, Sponsor: 7, Revision: 0,
		InvestigatorRuns: 4, ProposalJudgments: 4, ActiveMinutes: 120,
		Active: true, CreatedUnix: 1000,
	}
	raw, err := json.Marshal(grant)
	if err != nil {
		t.Fatal(err)
	}
	cmd := factory.Command{
		ID: grantID, Type: factory.CommandRepairGrant, Target: factory.RepairTarget(repo),
		Principal: "soda-tester", Payload: string(raw),
	}
	cmd.Digest = factory.SettingsDigest(cmd.Type, cmd.Target, cmd.Payload)
	if _, created, err := db.ApplyRepairGrantCommand(ctx, cmd, grant, time.Now().UTC()); err != nil || !created {
		t.Fatalf("apply grant: created=%v err=%v", created, err)
	}
	stallSeedControl(t, db, repo, 1, factory.ReadinessBlocked,
		factory.BlockerCycle, stallBlocker(factory.BlockerCycle, 0, 0))

	round, err := coord.ServeStallRound(ctx, DispatchReport{})
	if err != nil {
		t.Fatalf("serve: %v", err)
	}
	if len(round.Repositories) != 1 || round.Repositories[0].Kind != StallRepositoryStall {
		t.Fatalf("round=%+v, want one bound stall repository", round)
	}
	standing, err := db.StandingStallEvents(ctx)
	if err != nil {
		t.Fatalf("standing: %v", err)
	}
	entry, ok := standing[repo]
	if !ok || entry.Generation != 0 || entry.EventID == "" || entry.Digest == "" {
		t.Fatalf("standing=%+v, want the durable generation-0 observation", standing)
	}
	if charges, err := db.RepairCharges(ctx, repo); err != nil || len(charges) != 1 {
		t.Fatalf("charges=%d err=%v, want exactly one admitted cycle", len(charges), err)
	}

	again, err := coord.ServeStallRound(ctx, DispatchReport{})
	if err != nil {
		t.Fatalf("reserve: %v", err)
	}
	if len(again.Repositories) != 1 {
		t.Fatalf("round=%+v, want the replayed episode", again)
	}
	if charges, err := db.RepairCharges(ctx, repo); err != nil || len(charges) != 1 {
		t.Fatalf("charges=%d err=%v, want redetection to replay without spending", len(charges), err)
	}

	skipped, err := coord.ServeStallRound(ctx, DispatchReport{
		Launched: []DispatchLaunch{{Repository: repo, Issue: 9}},
	})
	if err != nil {
		t.Fatalf("skip: %v", err)
	}
	if len(skipped.Repositories) != 0 {
		t.Fatalf("round=%+v, want no detection when the pass launched work", skipped)
	}
}

// TestServeStallRoundGenerationReuseAndIsolation proves the R36 serving
// joints across rounds and repositories on the real store path: the
// detection generation reuses until verified delivery advances it, and a
// stalled repository's bound episode and single charge never leak into a
// sibling repository that still holds ready work.
func TestServeStallRoundGenerationReuseAndIsolation(t *testing.T) {
	ctx := context.Background()
	db, _ := postgresFixture(t, nil)
	coord := &Coordinator{Store: db}
	const stalled, ready = 61, 62
	grantID := "d1000000000000000000000000000061"
	grant := factory.RepairGrant{
		ID: grantID, Repository: stalled, Sponsor: 7, Revision: 0,
		InvestigatorRuns: 4, ProposalJudgments: 4, ActiveMinutes: 120,
		Active: true, CreatedUnix: 1000,
	}
	raw, err := json.Marshal(grant)
	if err != nil {
		t.Fatal(err)
	}
	cmd := factory.Command{
		ID: grantID, Type: factory.CommandRepairGrant, Target: factory.RepairTarget(stalled),
		Principal: "soda-tester", Payload: string(raw),
	}
	cmd.Digest = factory.SettingsDigest(cmd.Type, cmd.Target, cmd.Payload)
	if _, created, err := db.ApplyRepairGrantCommand(ctx, cmd, grant, time.Now().UTC()); err != nil || !created {
		t.Fatalf("apply grant: created=%v err=%v", created, err)
	}
	stallSeedControl(t, db, stalled, 1, factory.ReadinessBlocked,
		factory.BlockerCycle, stallBlocker(factory.BlockerCycle, 0, 0))
	stallSeedControl(t, db, ready, 1, factory.ReadinessQueued, factory.ReasonEligible)

	byRepo := func(round StallClassification) map[int64]StallRepositoryResult {
		out := map[int64]StallRepositoryResult{}
		for _, result := range round.Repositories {
			out[result.Repository] = result
		}
		return out
	}
	for round := 1; round <= 2; round++ {
		served, err := coord.ServeStallRound(ctx, DispatchReport{})
		if err != nil {
			t.Fatalf("round %d: %v", round, err)
		}
		if served.Generation != 0 {
			t.Fatalf("round %d generation = %d, want reused 0", round, served.Generation)
		}
		repos := byRepo(served)
		stalledResult, ok := repos[stalled]
		if !ok || stalledResult.Kind != StallRepositoryStall || !stalledResult.Investigate || stalledResult.Event == nil {
			t.Fatalf("round %d stalled=%+v, want one bound episode", round, stalledResult)
		}
		readyResult, ok := repos[ready]
		if !ok || readyResult.Kind != StallRepositoryReady || readyResult.Investigate || readyResult.Event != nil {
			t.Fatalf("round %d ready=%+v, want resumed selection without an episode", round, readyResult)
		}
		if charges, err := db.RepairCharges(ctx, stalled); err != nil || len(charges) != 1 {
			t.Fatalf("round %d stall charges=%d err=%v, want exactly one", round, len(charges), err)
		}
		if charges, err := db.RepairCharges(ctx, ready); err != nil || len(charges) != 0 {
			t.Fatalf("round %d ready charges=%d err=%v, want none", round, len(charges), err)
		}
	}
	standing, err := db.StandingStallEvents(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if entry, ok := standing[stalled]; !ok || entry.Generation != 0 {
		t.Fatalf("standing=%+v, want the stalled generation-0 observation", standing)
	}
	if _, ok := standing[ready]; ok {
		t.Fatalf("standing=%+v, want no observation for the ready repository", standing)
	}
}
