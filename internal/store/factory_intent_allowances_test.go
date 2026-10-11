package store

import (
	"context"
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
)

func ensureAllowanceTables(t *testing.T, s *Store) {
	t.Helper()
	ctx := context.Background()
	for _, ddl := range []string{
		`CREATE TABLE IF NOT EXISTS factory_intent_allowance_heads(intent TEXT PRIMARY KEY REFERENCES factory_intent_heads(id), owner_id BIGINT NOT NULL CHECK(owner_id>0), revision TEXT NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS factory_intent_allowances(id TEXT PRIMARY KEY, intent TEXT NOT NULL REFERENCES factory_intent_allowance_heads(intent), data JSONB NOT NULL)`,
	} {
		if _, err := s.db.ExecContext(ctx, ddl); err != nil {
			t.Fatal(err)
		}
	}
}

func TestPlanningAllowanceCASAndEffect(t *testing.T) {
	s, _ := postgresFixture(t, nil)
	ensureAllowanceTables(t, s)
	ctx := context.Background()
	intent := factory.IntentRevision{ID: "d111111111111111111111111", Intent: "d222222222222222222222222", Owner: 1, Objective: "Tracker", State: "draft", MaxIssues: 10, CreatedUnix: 1}
	if err := s.SaveIntentRevision(ctx, intent); err != nil {
		t.Fatal(err)
	}
	a := factory.PlanningAllowance{ID: "d333333333333333333333333", Intent: intent.Intent, Owner: 1, Amount: 5, Unit: factory.PlanningAllowanceUnit, Consumed: 1, Ceiling: 10, CreatedUnix: 1}
	if err := s.SavePlanningAllowance(ctx, a); err != nil {
		t.Fatal(err)
	}
	if err := s.SavePlanningAllowance(ctx, a); err != nil {
		t.Fatal(err)
	}
	b := a
	b.ID = "d444444444444444444444444"
	b.Predecessor = a.ID
	b.Amount = 8
	if s.SavePlanningAllowance(ctx, b) == nil {
		t.Fatal("allowance increase without effect accepted")
	}
	b.Effect = "Add 3 calls to compare two more alternatives"
	b.Consumed = 1
	if err := s.SavePlanningAllowance(ctx, b); err != nil {
		t.Fatal(err)
	}
	got, err := s.PlanningAllowance(ctx, intent.Intent, 1)
	if err != nil || got.ID != b.ID || got.Available() != 7 {
		t.Fatalf("allowance read %v %v", got, err)
	}
	if _, err = s.PlanningAllowance(ctx, intent.Intent, 2); err == nil {
		t.Fatal("allowance visible to another owner")
	}
}
