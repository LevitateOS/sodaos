package control

import (
	"context"
	"errors"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
)

func (c *Controller) intervene(id string, cause error) error {
	ctx, stop := context.WithTimeout(context.Background(), 45*time.Second)
	defer stop()
	lease, err := c.lock(ctx, id, "publication")
	if err != nil {
		return err
	}
	a, err := c.Store.FactoryAttempt(ctx, id)
	if err == nil && a.Phase != factory.Finished {
		outcome := factory.NeedsHuman
		if errors.Is(cause, ErrClosed) {
			outcome = factory.Cancelled
		}
		a.Finish(outcome, interventionSummary(cause))
		err = c.save(ctx, &a)
	}
	_ = lease.Close()
	if err != nil {
		return err
	}
	if err = c.report(ctx, a); err != nil {
		return err
	}
	return cause
}

func (c *Controller) Cancel(ctx context.Context, id string) error {
	lease, err := c.lock(ctx, id, "publication")
	if err != nil {
		return err
	}
	a, err := c.Store.FactoryAttempt(ctx, id)
	if err == nil && a.Phase != factory.Finished {
		a.Finish(factory.Cancelled, "cancelled by human")
		err = c.save(ctx, &a)
	}
	_ = lease.Close()
	if err != nil {
		return err
	}
	// The running controller observes withdrawal, kills the entire workspace and
	// releases this lease. If it crashed, cancellation performs that cleanup here.
	execution, err := c.lock(ctx, id, "execution")
	if err != nil {
		return err
	}
	defer func() { _ = execution.Close() }()
	if err = c.clean(ctx, id); err != nil {
		return err
	}
	a, err = c.Store.FactoryAttempt(ctx, id)
	if err != nil {
		return err
	}
	return c.report(ctx, a)
}

// Recover does not resume conversations or infer work from container names.
// It withdraws interrupted attempts and reconciles only their durable resources.
func (c *Controller) Recover(ctx context.Context) error {
	attempts, err := c.Store.FactoryAttempts(ctx)
	if err != nil {
		return err
	}
	for _, a := range attempts {
		if err = c.recoverAttempt(ctx, a.ID); err != nil {
			return err
		}
	}
	return nil
}

func (c *Controller) recoverAttempt(ctx context.Context, id string) error {
	execution, err := c.lock(ctx, id, "execution")
	if err != nil {
		return err
	}
	defer func() { _ = execution.Close() }()
	publication, err := c.lock(ctx, id, "publication")
	if err != nil {
		return err
	}
	a, err := c.Store.FactoryAttempt(ctx, id)
	if err == nil && a.Phase != factory.Finished {
		a.Finish(factory.NeedsHuman, "controller restarted; explicit human attempt required")
		err = c.save(ctx, &a)
	}
	_ = publication.Close()
	if err != nil {
		return err
	}
	if err = c.clean(ctx, id); err != nil {
		return err
	}
	a, err = c.Store.FactoryAttempt(ctx, id)
	if err != nil {
		return err
	}
	return c.report(ctx, a)
}

func (c *Controller) clean(ctx context.Context, id string) error {
	runs, err := c.Store.FactoryRuns(ctx, id)
	if err != nil {
		return err
	}
	for _, r := range runs {
		if !r.CleanupComplete {
			if err := c.cleanExecution(ctx, r); err != nil {
				return err
			}
		}
	}

	a, err := c.Store.FactoryAttempt(ctx, id)
	if err != nil {
		return err
	}
	a.CleanupComplete = true
	return c.save(ctx, &a)
}

func (c *Controller) cleanExecution(ctx context.Context, r factory.Run) error {
	if r.Outcome == "" {
		r.Outcome = factory.NeedsHuman
		r.Summary = "interrupted execution"
	}
	if err := c.Store.SaveFactoryRun(ctx, r); err != nil {
		return err
	}
	var credentialErr error
	if r.IdentityLeaseID != "" && !r.CredentialReturned {
		credentialErr = c.Identity.ReconcileLease(ctx, r.IdentityLeaseID)
	}
	cleanupErr := c.Workspace.Cleanup(ctx, &r)
	if credentialErr != nil {
		r.CleanupComplete = false
	}
	if err := c.Store.SaveFactoryRun(ctx, r); err != nil {
		return err
	}
	if err := errors.Join(credentialErr, cleanupErr); err != nil {
		return err
	}
	return nil
}
