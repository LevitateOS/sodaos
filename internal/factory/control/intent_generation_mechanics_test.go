package control

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/store"
)

// These helpers isolate graph mechanics from planning, just as the native
// graph fixture isolates them from Forgejo. Production entry points always
// use the real recorded-plan validator. intent_plan_choice_test covers that
// boundary with persisted, settled planner assignments and real PostgreSQL.
func generateGraphMechanics(t *testing.T, c *Coordinator, ctx context.Context, intent string, owner int64) (factory.IntentGeneration, error) {
	t.Helper()
	r, err := c.Store.IntentRevision(ctx, intent, owner)
	if err != nil {
		return factory.IntentGeneration{}, err
	}
	var chosen *ChosenIntentPlan
	if _, err := c.Store.IntentGeneration(ctx, r.ID); errors.Is(err, store.ErrNotFound) {
		var previous *factory.IntentRevision
		prior, e := c.Store.PreviousGeneratedIntent(ctx, intent, r.ID)
		if e == nil {
			g, e := c.Store.IntentGeneration(ctx, prior.ID)
			if e != nil {
				return factory.IntentGeneration{}, e
			}
			prior.Tasks = g.SubmittedTasks
			previous = &prior
		} else if !errors.Is(e, store.ErrNotFound) {
			return factory.IntentGeneration{}, e
		}
		tasks, e := c.scopedIntentTasks(ctx, r, previous)
		if e != nil {
			return factory.IntentGeneration{}, e
		}
		chosen = &ChosenIntentPlan{Intent: intent, Revision: r.ID, Operation: factory.IntentPlanningOptions, Key: "mechanics", Tasks: tasks, Attempt: "mechanics-fixture:" + r.ID}
	}
	return c.generateAcceptedIntent(ctx, intent, owner, chosen, mechanicsPlanStub)
}

func generateChoiceMechanics(t *testing.T, c *Coordinator, ctx context.Context, intent string, owner int64, chosen ChosenIntentPlan) (factory.IntentGeneration, error) {
	t.Helper()
	chosen.Attempt = "mechanics-fixture:" + chosen.Revision
	return c.generateAcceptedIntent(ctx, intent, owner, &chosen, mechanicsPlanStub)
}

func mechanicsPlanStub(_ context.Context, g factory.IntentGeneration) (*DispatchWait, error) {
	if !strings.HasPrefix(g.PlanAssignment, "mechanics-fixture:") {
		return nil, errors.New("mechanics fixture lacks its explicit planning stub")
	}
	return nil, nil
}

// seedUnboundGeneration models persisted invalid provenance without asking a
// production creation API to manufacture it. It may be used to verify refusal
// or receipt reconciliation; it never supplies a valid planning binding.
func seedUnboundGeneration(t *testing.T, c *Coordinator, r factory.IntentRevision) factory.IntentGeneration {
	t.Helper()
	p, err := IntentGraphPayload(r)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	g := factory.IntentGeneration{RevisionID: r.ID, IntentID: r.Intent, OperationID: "intent-" + r.ID, ActorID: c.GraphActorID, NativeRev: 10, AuthRevision: r.ID + ":" + r.Digest(), NotAfter: time.Now().Add(time.Minute).Unix(), Payload: raw, SubmittedTasks: r.Tasks, IntentDigest: r.Digest(), State: "pending"}
	if err := c.Store.BeginIntentGeneration(context.Background(), g); err != nil {
		t.Fatal(err)
	}
	return g
}
