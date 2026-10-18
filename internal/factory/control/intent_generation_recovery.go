package control

import (
	"context"
	"errors"
	"fmt"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/store"
)

// RecoverIntentGraphs resumes at most one accepted native graph per serving
// pass. The durable packet/receipt and existing task bindings survive restart;
// a fair cursor prevents one unavailable native repository starving another.
// Recovery never mints fresh operations: a head without a generation row has
// nothing to resume (creation belongs to the recorded-plan serving
// connector), and a pending operation without a recorded-plan binding is
// bound first so it cannot bypass submission/adoption validation.
func (c *Coordinator) RecoverIntentGraphs(ctx context.Context) error {
	if c.Graph == nil || c.GraphActorID <= 0 {
		return nil
	}
	revisions, err := c.Store.PendingIntentGenerations(ctx, c.intentGenerationCursor, 1)
	if err != nil {
		return err
	}
	if len(revisions) == 0 {
		c.intentGenerationCursor = ""
		revisions, err = c.Store.PendingIntentGenerations(ctx, "", 1)
		if err != nil || len(revisions) == 0 {
			return err
		}
	}
	revision := revisions[0]
	c.intentGenerationCursor = revision.ID
	g, err := c.Store.IntentGeneration(ctx, revision.ID)
	if errors.Is(err, store.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if g.PlanAssignment == "" {
		if err := c.bindPendingGenerationPlan(ctx, revision, g); err != nil {
			return err
		}
	}
	_, err = c.GenerateAcceptedIntent(ctx, revision.Intent, revision.Owner)
	return err
}

// bindPendingGenerationPlan stamps a pre-binding pending operation with the
// recorded plan behind its accepted head, so the resumed submission and
// adoption validate the durable original binding. A binding the current
// direction no longer honors refuses and cancels the unprovable operation;
// a volatile preparation leaves it pending with a retryable error.
func (c *Coordinator) bindPendingGenerationPlan(ctx context.Context, revision factory.IntentRevision, g factory.IntentGeneration) error {
	head, err := c.Store.IntentRevision(ctx, revision.Intent, revision.Owner)
	if err != nil {
		_, _ = c.cancelIntentGeneration(ctx, g)
		return err
	}
	chosen, wait, err := c.ResolveAcceptedPlanChoice(ctx, revision.Owner, head)
	if err != nil {
		_, _ = c.cancelIntentGeneration(ctx, g)
		return err
	}
	if wait != nil {
		return fmt.Errorf("recorded plan waits: %s", wait.Detail)
	}
	g.PlanAssignment = chosen.Attempt
	return c.Store.RecordIntentGeneration(ctx, g)
}
