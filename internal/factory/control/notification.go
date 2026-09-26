package control

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/levitateos/sodaos/internal/factory"
)

func (c *Controller) report(ctx context.Context, a factory.Attempt) error {
	if c.Forgejo == nil {
		return nil
	}
	if a.Work.RepositoryID != c.Config.RepositoryID && c.Config.RepositoryID != 0 {
		return errors.New("attempt belongs to a different repository")
	}
	body := fmt.Sprintf("Soda attempt `%s`: **%s**.\n\n%s\n\nCandidate: `%s`. Executions: %d/4. CI evaluations: %d/2. Cleanup complete: %t.", a.ID, a.Outcome, a.Summary, a.Candidate, a.Executions, a.CIEvaluations, a.CleanupComplete)
	if err := c.Forgejo.WorkComment(ctx, c.implementationToken, c.Repository, a.Work.Issue, body); err != nil {
		return err
	}
	if a.Phase != factory.Finished {
		return nil
	}
	return c.labelOutcome(ctx, a)
}

func outcomeLabel(outcome factory.Outcome) (string, string) {
	switch outcome {
	case "":
		return "soda:running", "58a6ff"
	case factory.Succeeded:
		return "soda:ready", "2ea043"
	case factory.Cancelled:
		return "soda:cancelled", "8b949e"
	case factory.Failed:
		return "soda:failed", "f85149"
	default:
		return "soda:needs-human", "dbab09"
	}
}

func (c *Controller) labelOutcome(ctx context.Context, a factory.Attempt) error {
	name, color := outcomeLabel(a.Outcome)
	id, err := c.outcomeLabelID(ctx, name, color)
	if err != nil {
		return err
	}
	latest, err := c.Store.LatestFactoryAttempt(ctx, a.Work.RepositoryID, a.Work.Issue)
	if err != nil {
		return err
	}
	if latest.ID == a.ID {
		if err = c.replaceOutcomeLabel(ctx, a.Work.Issue, id); err != nil {
			return err
		}
	}
	if a.Pull > 0 {
		return c.replaceOutcomeLabel(ctx, a.Pull, id)
	}
	return nil
}

func (c *Controller) replaceOutcomeLabel(ctx context.Context, number, desired int64) error {
	labels, err := c.Forgejo.WorkIssueLabels(ctx, c.humanToken, c.Repository, number)
	if err != nil {
		return err
	}
	for _, label := range labels {
		if label.ID != desired && sodaOutcomeLabel(label.Name) {
			if err = c.Forgejo.RemoveWorkLabel(ctx, c.implementationToken, c.Repository, number, label.ID); err != nil {
				return err
			}
		}
	}
	return c.Forgejo.AddWorkLabel(ctx, c.implementationToken, c.Repository, number, desired)
}

func sodaOutcomeLabel(name string) bool {
	switch name {
	case "soda:running", "soda:ready", "soda:cancelled", "soda:failed", "soda:needs-human":
		return true
	}
	return false
}

func (c *Controller) outcomeLabelID(ctx context.Context, name, color string) (int64, error) {
	for page := 1; ; page++ {
		labels, err := c.Forgejo.WorkLabels(ctx, c.humanToken, c.Repository, page)
		if err != nil {
			return 0, err
		}
		for _, label := range labels {
			if label.Name == name {
				return label.ID, nil
			}
		}
		if len(labels) < 50 {
			break
		}
	}
	label, err := c.Forgejo.CreateWorkLabel(ctx, c.humanToken, c.Repository, name, color)
	return label.ID, err
}

// Transport and process boundaries return opaque errors. Suppress local paths
// in public summaries while retaining actionable fixed system reasons.
func interventionSummary(cause error) string {
	reason := cause.Error()
	if len(reason) > 256 || strings.ContainsAny(reason, "/\\\r\n") {
		reason = "execution failed; check private operator diagnostics"
	}
	return "Execution stopped: " + reason + ". Human intervention is required."
}

func (c *Controller) Report(ctx context.Context, id string) error {
	a, err := c.Store.FactoryAttempt(ctx, id)
	if err != nil {
		return err
	}
	return c.report(ctx, a)
}
