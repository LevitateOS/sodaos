package control

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
)

// RetryRun records an explicit retry request after the prior run is
// reconciled and current authority revalidates. The next dispatch pass
// rechecks current policy, accepted inputs and allowance before launching.
func (c *Coordinator) RetryRun(ctx context.Context, commandID, principal, runID string) (factory.RetryDecision, error) {
	bounded, stop := context.WithTimeout(ctx, 30*time.Second)
	defer stop()
	run, err := c.Store.FactoryRun(bounded, runID)
	if err != nil {
		return factory.RetryDecision{}, ErrNotFound
	}
	if !run.Reconciled {
		return factory.RetryDecision{}, ErrPendingRuns{Runs: []string{runID}}
	}
	p, err := c.Store.Project(bounded, run.ProjectID)
	if err != nil {
		return factory.RetryDecision{}, ErrNotFound
	}
	effective, err := c.EffectiveAuthority(bounded, p.RepositoryID)
	if err != nil {
		return factory.RetryDecision{}, err
	}
	if !effective.Effective && !onlyDispatchClosed(effective) {
		return factory.RetryDecision{}, ErrIneffectiveAuthority
	}
	payload, err := json.Marshal(map[string]string{"prior_run": runID, "prior_outcome": string(run.Outcome)})
	if err != nil {
		return factory.RetryDecision{}, err
	}
	target := "run/" + runID
	cmd := factory.Command{
		ID: commandID, Type: factory.CommandRetry, Target: target, Principal: principal,
		Payload: string(payload), Digest: factory.SettingsDigest(factory.CommandRetry, target, string(payload)),
	}
	if err = cmd.Validate(); err != nil {
		return factory.RetryDecision{}, err
	}
	prior, err := c.Store.FactoryCommand(bounded, cmd.ID)
	if err == nil {
		if prior.Digest != cmd.Digest {
			return factory.RetryDecision{}, store.ErrCommandConflict
		}
		return replayRetry(prior)
	}
	if !errors.Is(err, store.ErrNotFound) {
		return factory.RetryDecision{}, err
	}
	// The retry enters the real dispatch lifecycle when it can: a failed
	// coder run re-queues its issue for the next dispatch pass. Succeeded
	// runs relaunch nothing (corrections advance through new runs), and
	// review runs resubmit through a new review; both record truthfully
	// instead of pretending to queue work no consumer reads.
	decision := factory.RetryDecision{
		Prior: runID, CommandID: cmd.ID, Queued: true,
		Reason: c.retryQueueReason(bounded, run),
	}
	if err = decision.Validate(); err != nil {
		return factory.RetryDecision{}, err
	}
	if run.Role == project.RoleCoder && (run.Outcome == factory.Failed || run.Outcome == factory.Cancelled) {
		view, viewErr := c.Store.FactoryRunView(bounded, run.ID)
		if viewErr != nil && !errors.Is(viewErr, store.ErrNotFound) {
			return factory.RetryDecision{}, viewErr
		}
		if viewErr == nil {
			assignment, assignmentErr := c.Store.Assignment(bounded, view.Attempt)
			if assignmentErr != nil && !errors.Is(assignmentErr, store.ErrNotFound) {
				return factory.RetryDecision{}, assignmentErr
			}
			if assignmentErr != nil || assignment.Run != run.ID || assignment.Role != project.RoleCoder || view.RunID != run.ID || view.Repository <= 0 || view.Issue <= 0 {
				return factory.RetryDecision{}, store.ErrNotFound
			}
			head, headErr := c.Store.AcceptanceHead(bounded, view.Repository, view.Issue)
			if headErr != nil {
				return factory.RetryDecision{}, headErr
			}
			stored, created, retryErr := c.Store.RecordExplicitRetryCommand(bounded, cmd, decision, view.Repository, view.Issue, head, time.Now())
			if retryErr != nil {
				return factory.RetryDecision{}, retryErr
			}
			if !created {
				return replayRetry(stored)
			}
			return decision, nil
		}
	}
	stored, created, err := c.Store.RecordFactoryCommand(bounded, cmd, time.Now())
	if err != nil {
		return factory.RetryDecision{}, err
	}
	if !created {
		return replayRetry(stored)
	}
	outcome, err := json.Marshal(decision)
	if err != nil {
		return factory.RetryDecision{}, err
	}
	if err = c.Store.FinishFactoryCommand(bounded, cmd.ID, string(outcome), time.Now()); err != nil {
		return factory.RetryDecision{}, err
	}
	return decision, nil
}

func onlyDispatchClosed(effective factory.EffectiveAuthority) bool {
	return len(effective.Missing) == 1 && effective.Missing[0] == factory.MissingDispatch
}

// retryQueueReason re-queues a failed coder run's issue for dispatch and
// reports what actually happened. Reassessment is best effort: when the
// issue cannot be re-read, the recorded retry waits for the next intake
// or reconcile trigger instead of failing the operator command.
func (c *Coordinator) retryQueueReason(ctx context.Context, run factory.Run) string {
	if run.Role == project.RoleReviewer {
		return "retry recorded; review runs resubmit through a new review"
	}
	if run.Outcome == factory.Succeeded {
		return "retry recorded; the prior run succeeded so nothing relaunches"
	}
	view, err := c.Store.FactoryRunView(ctx, run.ID)
	if err != nil {
		return "retry recorded; the prior run names no dispatchable issue"
	}
	outcome, err := c.assessCascade(ctx, view.Repository, view.Issue, make(map[factory.DependenceRef]bool))
	if err != nil {
		return "retry recorded; reassessment unavailable so dispatch waits for the next trigger"
	}
	if outcome.control.Readiness == factory.ReadinessQueued {
		return "retry queued for dispatch"
	}
	return "retry recorded; issue " + outcome.control.Readiness
}

func replayRetry(stored factory.Command) (factory.RetryDecision, error) {
	if stored.Finished == "" {
		return factory.RetryDecision{}, ErrCommandRunning
	}
	var decision factory.RetryDecision
	if err := json.Unmarshal([]byte(stored.Outcome), &decision); err != nil {
		return factory.RetryDecision{}, err
	}
	return decision, nil
}
