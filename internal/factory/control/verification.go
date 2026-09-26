package control

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/forgejo"
)

func (c *Controller) verify(ctx context.Context, a *factory.Attempt) error {
	if err := c.reviewCandidate(ctx, a); err != nil {
		return err
	}
	evidence, err := c.waitCI(ctx, *a)
	if err != nil {
		return err
	}
	lease, err := c.lock(ctx, a.ID, "publication")
	if err != nil {
		return err
	}
	defer func() { _ = lease.Close() }()
	current, err := c.live(ctx, a.ID)
	if err != nil {
		return err
	}
	if err = c.checkPull(ctx, current); err != nil {
		return err
	}
	if err = current.RecordCI(evidence, time.Now()); err != nil {
		return err
	}
	if err = current.Evaluate(time.Now()); err != nil {
		return err
	}
	if err = c.save(ctx, &current); err != nil {
		return err
	}
	*a = current
	return nil
}

func (c *Controller) waitCI(ctx context.Context, a factory.Attempt) (factory.Evidence, error) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		if _, err := c.live(ctx, a.ID); err != nil {
			return factory.Evidence{}, err
		}
		if err := c.checkPull(ctx, a); err != nil {
			return factory.Evidence{}, err
		}
		runs, err := c.Forgejo.WorkActions(ctx, c.humanToken, c.Repository, a.Candidate, c.Config.Workflow, 1)
		if err != nil {
			return factory.Evidence{}, err
		}
		evidence, done, err := ciResult(runs)
		if err != nil || done {
			return evidence, err
		}
		select {
		case <-ctx.Done():
			return factory.Evidence{}, ctx.Err()
		case <-ticker.C:
		}
	}
}

func (c *Controller) reviewCandidate(ctx context.Context, a *factory.Attempt) error {
	if a.Review != nil {
		return nil
	}
	return c.executeRun(ctx, a, factory.Review)
}

func ciResult(runs []forgejo.ActionRun) (factory.Evidence, bool, error) {
	if len(runs) > 1 {
		return factory.Evidence{}, false, errors.New("candidate has multiple CI evaluations; human reconciliation required")
	}
	if len(runs) == 0 {
		return factory.Evidence{}, false, nil
	}
	run := runs[0]
	if run.Event != "pull_request" {
		return factory.Evidence{}, false, errors.New("CI did not evaluate the assigned PR")
	}
	e := factory.Evidence{ID: strconv.FormatInt(run.ID, 10), Commit: run.Commit, Passed: run.Status == "success"}
	switch run.Status {
	case "success", "failure":
		return e, true, nil
	case "waiting", "running":
		return e, false, nil
	default:
		return e, false, errors.New("CI needs human intervention")
	}
}
