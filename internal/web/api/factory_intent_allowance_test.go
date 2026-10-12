package api

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/store"
)

func allowanceTestAPI(t *testing.T) (*API, *store.Store) {
	t.Helper()
	admin, ok := store.TestFixtureDSN()
	if !ok {
		t.Skip("SODA_PG_* fixture unavailable")
	}
	s, _, cleanup, err := store.OpenEphemeral(context.Background(), admin, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	s.PriorityCallBudget = 32
	return &API{Store: s}, s
}

func allowanceTestIntent(t *testing.T, s *store.Store, id, intent string) factory.IntentRevision {
	t.Helper()
	r := factory.IntentRevision{ID: id, Intent: intent, Owner: 1, Objective: "Tracker", State: "draft", MaxIssues: 10, CreatedUnix: 1}
	if err := s.SaveIntentRevision(context.Background(), r); err != nil {
		t.Fatal(err)
	}
	return r
}

// TestAllowanceSaveGroundsBrowserPayloads proves the endpoint core accepts
// the exact browser-emitted allowance shape: V's first save carries no
// ceiling and no usage echo, so the server grounds the ceiling in the
// operator budget and counters in live records instead of failing the
// client for values only the server can source.
func TestAllowanceSaveGroundsBrowserPayloads(t *testing.T) {
	a, s := allowanceTestAPI(t)
	ctx := context.Background()
	r := allowanceTestIntent(t, s, "d111111111111111111111111", "d222222222222222222222222")
	first := factory.PlanningAllowance{ID: "d333333333333333333333333", Intent: r.Intent, Owner: 1, Amount: 5, Unit: factory.PlanningAllowanceUnit, CreatedUnix: 1}
	out, err := a.storeIntentAllowance(ctx, 1, first)
	if err != nil {
		t.Fatalf("browser first save refused: %v", err)
	}
	if out.Ceiling != 32 || out.Reserved != 0 || out.Consumed != 0 {
		t.Fatalf("first save not grounded: %+v", out)
	}
	explicit := factory.PlanningAllowance{ID: "d444444444444444444444444", Intent: "d555555555555555555555555", Owner: 1, Amount: 5, Unit: factory.PlanningAllowanceUnit, CreatedUnix: 1}
	allowanceTestIntent(t, s, "d666666666666666666666666", explicit.Intent)
	explicit.Ceiling = 5
	kept, err := a.storeIntentAllowance(ctx, 1, explicit)
	if err != nil {
		t.Fatalf("explicit ceiling save refused: %v", err)
	}
	if kept.Ceiling != 5 {
		t.Fatalf("explicit ceiling overwritten: %+v", kept)
	}
	if _, err := a.storeIntentAllowance(ctx, 1, first); err != nil {
		t.Fatalf("exact replay refused: %v", err)
	}
	second := factory.PlanningAllowance{ID: "d777777777777777777777777", Intent: r.Intent, Predecessor: first.ID, Owner: 1, Amount: 8, Unit: factory.PlanningAllowanceUnit, Effect: "Authorize 3 additional planning calls (8 total).", CreatedUnix: 1}
	if _, err := a.storeIntentAllowance(ctx, 1, second); err != nil {
		t.Fatalf("stated increase refused: %v", err)
	}
	silent := factory.PlanningAllowance{ID: "d888888888888888888888888", Intent: r.Intent, Predecessor: second.ID, Owner: 1, Amount: 9, Unit: factory.PlanningAllowanceUnit, CreatedUnix: 1}
	if _, err := a.storeIntentAllowance(ctx, 1, silent); err == nil || !strings.Contains(err.Error(), "allowance increase must state its effect") {
		t.Fatalf("silent increase accepted: %v", err)
	}
	reserved, err := s.ReserveIntentJudgment(ctx, r, strings.Repeat("a", 64))
	if err != nil || !reserved {
		t.Fatalf("reservation=%v %v", reserved, err)
	}
	afterUse := factory.PlanningAllowance{ID: "d999999999999999999999999", Intent: r.Intent, Predecessor: second.ID, Owner: 1, Amount: 8, Unit: factory.PlanningAllowanceUnit, CreatedUnix: 1}
	charged, err := a.storeIntentAllowance(ctx, 1, afterUse)
	if err != nil {
		t.Fatalf("save after usage refused: %v", err)
	}
	if charged.Reserved != 1 || charged.Consumed != 0 {
		t.Fatalf("live usage not filled: %+v", charged)
	}
	var clientErr *allowanceWriteError
	missing := factory.PlanningAllowance{ID: "d000000000000000000000001", Intent: "d000000000000000000000002", Owner: 1, Amount: 1, Unit: factory.PlanningAllowanceUnit, CreatedUnix: 1}
	if _, err := a.storeIntentAllowance(ctx, 1, missing); !errors.As(err, &clientErr) || clientErr.code != 404 {
		t.Fatalf("missing intent did not 404: %v", err)
	}
}
