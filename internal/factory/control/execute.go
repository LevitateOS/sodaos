package control

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
)

type Task struct {
	Run       factory.Run       `json:"run"`
	Work      factory.WorkItem  `json:"work"`
	Candidate string            `json:"candidate"`
	Findings  []string          `json:"findings,omitempty"`
	CI        *factory.Evidence `json:"ci,omitempty"`
}

func (c *Controller) Execute(ctx context.Context, id string) error {
	lease, err := c.lock(ctx, id, "execution")
	if err != nil {
		return err
	}
	defer func() { _ = lease.Close() }()
	a, err := c.Store.FactoryAttempt(ctx, id)
	if err != nil {
		return err
	}
	if a.Phase == factory.Finished {
		return c.report(ctx, a)
	}
	deadline, stop := context.WithDeadline(ctx, a.Deadline)
	defer stop()
	err = c.loop(deadline, &a)
	if err != nil {
		return c.intervene(id, err)
	}
	return c.report(ctx, a)
}

func (c *Controller) loop(ctx context.Context, a *factory.Attempt) error {
	for a.Phase != factory.Finished {
		switch a.Phase {
		case factory.Implement, factory.Fix:
			if err := c.executeRun(ctx, a, implementationRole(a.Phase)); err != nil {
				return err
			}
		case factory.Verify, factory.Reverify:
			if err := c.verify(ctx, a); err != nil {
				return err
			}
		default:
			return errors.New("unknown factory phase")
		}
	}
	return nil
}

func implementationRole(phase factory.Phase) factory.Role {
	if phase == factory.Fix {
		return factory.Repair
	}
	return factory.Implementation
}

func (c *Controller) executeRun(ctx context.Context, a *factory.Attempt, role factory.Role) error {
	if _, err := c.live(ctx, a.ID); err != nil {
		return err
	}
	copy := *a
	r, err := copy.BeginRun(role, time.Now())
	if err != nil {
		return err
	}
	c.Config.Workspace.Bind(&r)
	if err = c.Store.StartFactoryRun(ctx, a, r); err != nil {
		return err
	}
	runctx, stop := context.WithDeadline(ctx, r.Deadline)
	defer stop()
	runctx, cancel := context.WithCancelCause(runctx)
	defer cancel(nil)
	go c.watch(runctx, a.ID, cancel)
	if err = c.Workspace.CheckCapacity(); err != nil {
		return c.finishRun(a, &r, factory.Result{}, err)
	}
	if err = c.acquireCredential(runctx, a.Work.HumanID, &r); err != nil {
		return c.finishRun(a, &r, factory.Result{}, err)
	}
	result, err := c.worker(runctx, a, &r)
	if runctx.Err() != nil {
		err = context.Cause(runctx)
	}
	return c.finishRun(a, &r, result, err)
}

func (c *Controller) worker(ctx context.Context, a *factory.Attempt, r *factory.Run) (factory.Result, error) {
	if err := c.prepareWorkspace(ctx, *a, r); err != nil {
		return factory.Result{}, err
	}
	secrets, err := c.delegateCredential(ctx, r)
	if err != nil {
		return factory.Result{}, err
	}
	result, err := c.Workspace.Launch(ctx, *r)
	if err != nil {
		return result, err
	}
	return c.completeWorker(ctx, a, r, result, secrets)
}

func (c *Controller) completeWorker(ctx context.Context, a *factory.Attempt, r *factory.Run, result factory.Result, secrets []string) (factory.Result, error) {
	if result.Status != "completed" {
		return result, errors.New("agent could not complete the assigned task")
	}
	if r.Role == factory.Review {
		head, err := c.Workspace.Revision(ctx, *r)
		if err != nil || head != r.InputSHA {
			return result, errors.New("review did not retain assigned HEAD")
		}
		after, err := c.returnCredential(ctx, r)
		if err != nil {
			return result, err
		}
		secrets = append(secrets, after...)
		if err := c.retainResult(*r, result, secrets); err != nil {
			return result, err
		}
		return result, c.submitReview(ctx, *a, *r, result)
	}
	return c.publishWorker(ctx, a, r, result, secrets)
}

func (c *Controller) publishWorker(ctx context.Context, a *factory.Attempt, r *factory.Run, result factory.Result, secrets []string) (factory.Result, error) {
	bundle, err := c.Workspace.ExportCandidate(ctx, *r)
	if err != nil {
		return result, err
	}
	after, err := c.returnCredential(ctx, r)
	if err != nil {
		return result, err
	}
	secrets = append(secrets, after...)
	if err := c.retainResult(*r, result, secrets); err != nil {
		return result, err
	}
	return result, c.publish(ctx, a, *r, result, bundle, secrets)
}

func (c *Controller) prepareWorkspace(ctx context.Context, a factory.Attempt, r *factory.Run) error {
	source, err := c.Publisher.Source(ctx, r.InputSHA)
	if err != nil {
		return err
	}
	task, err := c.task(ctx, a, *r)
	if err != nil {
		return err
	}
	if err = c.Workspace.Prepare(*r, task, source); err != nil {
		return err
	}
	if err = c.Workspace.Create(ctx, r, func(run factory.Run) error { return c.Store.SaveFactoryRun(ctx, run) }); err != nil {
		return err
	}
	if err = c.Workspace.Initialize(ctx, *r); err != nil {
		return fmt.Errorf("initialize workspace: %w", err)
	}
	return nil
}

func (c *Controller) task(ctx context.Context, a factory.Attempt, r factory.Run) (Task, error) {
	task := Task{Run: r, Work: a.Work, Candidate: a.Candidate, CI: a.CI}
	task.Run.IdentityLeaseID, task.Run.IdentityGeneration, task.Run.IdentityBinding = "", 0, nil
	task.Run.CredentialDelegated, task.Run.CredentialReturned = false, false
	// Resource names and image provenance are useful; host paths and credentials
	// are deliberately absent from immutable task input.
	if r.Role != factory.Repair {
		return task, nil
	}
	reviews, err := c.Forgejo.WorkReviews(ctx, c.humanToken, c.Repository, a.Pull)
	if err != nil {
		return task, err
	}
	for _, review := range reviews {
		if review.Commit == a.Candidate && review.User.ID == c.Reviewer.ID {
			task.Findings = append(task.Findings, review.Body)
		}
	}
	return task, nil
}

func (c *Controller) finishRun(a *factory.Attempt, r *factory.Run, result factory.Result, workErr error) error {
	ctx, stop := context.WithTimeout(context.Background(), 45*time.Second)
	defer stop()
	credentialErr := c.returnDelegatedCredential(ctx, r)
	if credentialErr != nil {
		workErr = credentialErr
	}
	current, err := c.Store.FactoryAttempt(ctx, a.ID)
	if err != nil {
		return err
	}
	completeRun(r, result, workErr, current.Outcome)
	if err = c.cleanupRun(ctx, r, credentialErr); err != nil {
		return err
	}
	current, err = c.Store.FactoryAttempt(ctx, a.ID)
	if err != nil {
		return err
	}
	current.CleanupComplete = true
	if err = c.save(ctx, &current); err != nil {
		return err
	}
	*a = current
	if workErr != nil {
		return workErr
	}
	if r.Role == factory.Review {
		if err = a.RecordReview(*r, result.ReviewPassed, time.Now()); err != nil {
			return err
		}
		return c.save(ctx, a)
	}
	return nil
}

func (c *Controller) cleanupRun(ctx context.Context, r *factory.Run, credentialErr error) error {
	if err := c.Store.SaveFactoryRun(ctx, *r); err != nil {
		return err
	}
	cleanupErr := c.Workspace.Cleanup(ctx, r)
	if credentialErr != nil {
		r.CleanupComplete = false
	}
	if err := c.Store.SaveFactoryRun(ctx, *r); err != nil {
		return err
	}
	return errors.Join(cleanupErr, credentialErr)
}

func completeRun(r *factory.Run, result factory.Result, workErr error, parent factory.Outcome) {
	r.Outcome = factory.Succeeded
	r.Summary = result.Summary
	if workErr != nil {
		r.Outcome = factory.NeedsHuman
		r.Summary = "execution requires human intervention"
	}
	if parent == factory.Cancelled {
		r.Outcome = factory.Cancelled
		r.Summary = "execution cancelled"
	}
}
