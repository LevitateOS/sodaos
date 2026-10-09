package control

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
)

func TestFindPrereqCycle(t *testing.T) {
	c := coordinatorFixture(t, nil, nil)
	ctx := context.Background()
	chain := func(id, issue string, endpoint int64) {
		decision := readinessDecision(id, 42, issue, 12)
		decision.Prerequisites = []factory.AcceptedPrerequisite{{
			Occurrence: "21", DependsOn: "8", EndpointRepo: 42, EndpointIssue: endpoint,
			Outcome: factory.PrereqResult,
		}}
		mustAdmit(t, c, decision)
	}
	chain("d"+strings.Repeat("1", 24), "3", 4)
	chain("d"+strings.Repeat("2", 24), "4", 5)
	if cycle, exceeded, err := c.findPrereqCycle(ctx, 42, 3); err != nil || exceeded || cycle != "" {
		t.Fatal("acyclic walk reported a cycle:", cycle, exceeded, err)
	}
	self := readinessDecision("d"+strings.Repeat("3", 24), 42, "6", 12)
	self.Prerequisites = []factory.AcceptedPrerequisite{{
		Occurrence: "21", DependsOn: "8", EndpointRepo: 42, EndpointIssue: 6,
		Outcome: factory.PrereqResult,
	}}
	mustAdmit(t, c, self)
	if cycle, _, err := c.findPrereqCycle(ctx, 42, 6); err != nil || cycle != "42/6 -> 42/6" {
		t.Fatal("self cycle wrong:", cycle, err)
	}
}

func TestFindPrereqCycleExceedsBound(t *testing.T) {
	db, dsn := postgresFixture(t, nil)
	c := NewCoordinator(db, nil, nil)
	fixtureDB, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = fixtureDB.Close() }()
	ctx := context.Background()
	for i := int64(0); i <= MaxCycleNodes+1; i++ {
		issue := 1000 + i
		decision := readinessDecision(fmt.Sprintf("d%024x", i+1), 42, fmt.Sprint(issue), 12)
		decision.Prerequisites = []factory.AcceptedPrerequisite{{
			Occurrence: "21", DependsOn: "8", EndpointRepo: 42, EndpointIssue: issue + 1,
			Outcome: factory.PrereqResult,
		}}
		mustAdmit(t, c, decision)
		// This test shapes the graph directly; emitted root headers are not
		// under test and would otherwise hit the independent source-page cap.
		if _, err := fixtureDB.ExecContext(ctx, `DELETE FROM factory_readiness_sources WHERE id=$1`, fmt.Sprintf("root:42/%d", issue)); err != nil {
			t.Fatalf("remove emitted fixture header for issue %d: %v", issue, err)
		}
	}
	if _, exceeded, err := c.findPrereqCycle(ctx, 42, 1000); err != nil || !exceeded {
		t.Fatal("over-bound walk not exceeded:", exceeded, err)
	}
}
