package control

import "context"

// RecoverIntentGraphs resumes at most one accepted native graph per serving
// pass. The durable packet/receipt and existing task bindings survive restart;
// a fair cursor prevents one unavailable native repository starving another.
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
	_, err = c.GenerateAcceptedIntent(ctx, revision.Intent, revision.Owner)
	return err
}
