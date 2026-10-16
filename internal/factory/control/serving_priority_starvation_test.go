package control

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
)

// TestServingPriorityDispatchStarvesBeyondReadinessBudget reproduces P3
// (FW04): the priority path demands the complete candidate inventory while
// production refresh spends one native RPC per control from the shared
// 156-RPC readiness budget. Past 156 unassigned ready issues every pass
// reports priority_incomplete and launches nothing, on consecutive passes.
// This is the FW10 regression anchor; it implements no D02 cursor fix.
//
// Seeding follows two committed precedents: the serving coordinator shape
// of serving_planning_priority_test.go (planningCoordinator wiring plus a
// fake evidence source and observer) and the traversal_test.go header
// release (each acceptance emits one intake root capped at 64 sources, so
// the emitted header is forgotten through the disposable fixture handle,
// keeping the acceptance head and queued control).
func TestServingPriorityDispatchStarvesBeyondReadinessBudget(t *testing.T) {
	db, dsn := dispatchTestDB(t)
	fixtureDB, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = fixtureDB.Close() }()
	ctx := context.Background()
	fx := dispatchSeed(t, db)
	forgetAdmissionHeader := func(issue int64) {
		t.Helper()
		if _, err := fixtureDB.ExecContext(ctx, `DELETE FROM factory_readiness_sources WHERE id=$1`, fmt.Sprintf("root:%d/%d", fx.repo, issue)); err != nil {
			t.Fatalf("remove emitted fixture header for issue %d: %v", issue, err)
		}
	}
	source := &fakeEvidenceSource{evidence: map[string]AcceptanceEvidence{}}
	c := NewCoordinator(db, fx.host, fx.broker)
	c.AcceptanceReads, c.DispatchReads = source, fx.reads
	c.Readiness = &fakeObserver{revision: 41, idle: true}
	const issues = 170
	for i := 0; i < issues; i++ {
		issue := int64(100 + i)
		decision := fx.accept(t, issue, fmt.Sprintf("d%024x", 9000+i))
		forgetAdmissionHeader(issue)
		fx.queue(t, issue, decision.ID)
	}
	deps := fx.deps()
	deps.Refresh = c.refreshReadiness
	deps.Authority = c.EffectiveAuthority
	chooser := &priorityFixture{choose: priorityLast}
	deps.Priority = &PriorityJudgments{Journal: c.Store, Chooser: chooser, MaxCalls: 16}
	for pass := 0; pass < 2; pass++ {
		passCtx, _, cancel := c.readinessPass(ctx)
		report := DispatchPass(passCtx, deps)
		cancel()
		if len(report.Launched) != 0 || len(fx.host.launches) != 0 {
			t.Fatalf("pass %d launched during starvation: %+v", pass, report)
		}
		incomplete := false
		for _, failure := range report.Errors {
			if failure.Reason == "priority_incomplete" {
				incomplete = true
			}
		}
		if !incomplete {
			t.Fatalf("pass %d missed priority_incomplete: %+v", pass, report)
		}
	}
}
