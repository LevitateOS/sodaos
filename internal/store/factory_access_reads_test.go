package store

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
)

func TestRepositoryIntentAllowancesValidation(t *testing.T) {
	empty := &Store{}
	ctx := context.Background()
	if _, _, err := empty.RepositoryIntentAllowances(ctx, 0, 1, 5); err == nil {
		t.Fatal("accepted invalid scope")
	}
	if _, _, err := empty.RepositoryIntentAllowances(ctx, 7, 1, 0); err == nil {
		t.Fatal("accepted invalid bound")
	}
	if _, _, err := empty.RepositoryIntentAllowances(ctx, 7, 1, MaxRepositoryAllowances+1); err == nil {
		t.Fatal("accepted oversized bound")
	}
}

func TestRepositoryIntentAllowancesListsBoundIntents(t *testing.T) {
	s, _ := postgresFixture(t, nil)
	ctx := context.Background()
	save := func(intent, revision string, owner int64, repository string) {
		t.Helper()
		if _, err := s.db.ExecContext(ctx, `INSERT INTO factory_intent_heads(id,owner_id,revision) VALUES($1,$2,$3) ON CONFLICT(id) DO UPDATE SET revision=excluded.revision`, intent, owner, revision); err != nil {
			t.Fatal(err)
		}
		if _, err := s.db.ExecContext(ctx, `INSERT INTO factory_intent_revisions(id,intent,data) VALUES($1,$2,$3) ON CONFLICT(id) DO NOTHING`, revision, intent, `{"repository":"`+repository+`"}`); err != nil {
			t.Fatal(err)
		}
		raw, err := json.Marshal(factory.PlanningAllowance{ID: "allowance-" + revision, Intent: intent, Owner: owner, Amount: 5, Unit: "planning_calls", Ceiling: 10})
		if err != nil {
			t.Fatal(err)
		}
		if _, err = s.db.ExecContext(ctx, `INSERT INTO factory_intent_allowance_heads(intent,owner_id,revision) VALUES($1,$2,$3) ON CONFLICT(intent) DO UPDATE SET revision=excluded.revision`, intent, owner, "allowance-"+revision); err != nil {
			t.Fatal(err)
		}
		if _, err = s.db.ExecContext(ctx, `INSERT INTO factory_intent_allowances(id,intent,data) VALUES($1,$2,$3) ON CONFLICT(id) DO NOTHING`, "allowance-"+revision, intent, string(raw)); err != nil {
			t.Fatal(err)
		}
	}
	save("intent-a", "rev-a", 1, "7")
	save("intent-b", "rev-b", 1, "7")
	save("intent-c", "rev-c", 1, "8")
	save("intent-d", "rev-d", 2, "7")
	page, truncated, err := s.RepositoryIntentAllowances(ctx, 7, 1, 1)
	if err != nil || !truncated || len(page) != 1 || page[0].Intent != "intent-a" {
		t.Fatal(page, truncated, err)
	}
	page, truncated, err = s.RepositoryIntentAllowances(ctx, 7, 1, MaxRepositoryAllowances)
	if err != nil || truncated || len(page) != 2 {
		t.Fatal(page, truncated, err)
	}
	for _, allowance := range page {
		if allowance.Owner != 1 || allowance.Unit != "planning_calls" {
			t.Fatalf("allowance: %+v", allowance)
		}
	}
	empty, truncated, err := s.RepositoryIntentAllowances(ctx, 9, 1, MaxRepositoryAllowances)
	if err != nil || truncated || len(empty) != 0 {
		t.Fatal(empty, truncated, err)
	}
}
