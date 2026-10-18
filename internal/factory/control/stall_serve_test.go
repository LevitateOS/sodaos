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
