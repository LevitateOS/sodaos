package control

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
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
	decision := factory.RetryDecision{
		Prior: runID, CommandID: cmd.ID, Queued: true,
		Reason: "new attempt queued; dispatch will revalidate current policy and provider budget",
	}
	if err = decision.Validate(); err != nil {
		return factory.RetryDecision{}, err
	}
	view, err := c.Store.FactoryRunView(bounded, run.ID)
	if err != nil || view.RunID != run.ID || view.Repository <= 0 || view.Issue <= 0 {
		return factory.RetryDecision{}, store.ErrNotFound
	}
	assignment, err := c.Store.Assignment(bounded, view.Attempt)
	if err != nil || assignment.Run != run.ID || assignment.AttemptRoot == "" {
		return factory.RetryDecision{}, store.ErrNotFound
	}
	owner, err := c.Store.LatestIssueAssignment(bounded, view.Repository, view.Issue)
	if err != nil || owner.AttemptRoot != assignment.AttemptRoot || assignment.PublicationAssignment != owner.ID {
		return factory.RetryDecision{}, store.ErrNotFound
	}
	head, err := c.Store.AcceptanceHead(bounded, view.Repository, view.Issue)
	if err != nil {
		return factory.RetryDecision{}, err
	}
	if owner.Acceptance != head || assignment.Acceptance != head {
		return factory.RetryDecision{}, store.ErrNotFound
	}
	stored, created, err := c.Store.RecordExplicitRetryCommand(bounded, cmd, decision, view.Repository, view.Issue, head, time.Now())
	if err != nil {
		return factory.RetryDecision{}, err
	}
	if !created {
		return replayRetry(stored)
	}
	return decision, nil
}

func onlyDispatchClosed(effective factory.EffectiveAuthority) bool {
	return len(effective.Missing) == 1 && effective.Missing[0] == factory.MissingDispatch
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
