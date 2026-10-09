package control

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/store"
)

// ErrPendingRuns reports a resume, retry or takeover refused while recorded
// runs are unsettled. The caller stops or reconciles first; pending native
// effects never appear cancelled, queued or taken over.
type ErrPendingRuns struct {
	Runs []string
}

func (e ErrPendingRuns) Error() string { return "factory runs are not settled" }

// PauseRepository latches one repository's dispatch closed and then stops
// its outstanding runs. Withdrawal precedes every stop, so a delayed
// registration cannot escape the captured set and a delayed start meets
// the stop tombstone. Uncertain runs stay fenced inside the receipt.
func (c *Coordinator) PauseRepository(ctx context.Context, commandID, principal string, repository int64) (factory.PauseReceipt, error) {
	bounded, stop := context.WithTimeout(ctx, 10*time.Minute)
	defer stop()
	if repository <= 0 {
		return factory.PauseReceipt{}, errors.New("invalid pause target")
	}
	payload := `{"cause":"` + factory.CauseControlPaused + `"}`
	target := grantTarget("dispatch", repository)
	cmd := factory.Command{
		ID: commandID, Type: factory.CommandPause, Target: target, Principal: principal,
		Payload: payload, Digest: factory.SettingsDigest(factory.CommandPause, target, payload),
	}
	if err := cmd.Validate(); err != nil {
		return factory.PauseReceipt{}, err
	}
	stored, created, err := c.Store.RecordFactoryCommand(bounded, cmd, time.Now())
	if err != nil {
		return factory.PauseReceipt{}, err
	}
	if !created {
		return replayPause(stored)
	}
	withdrawal, outcomes, err := c.withdrawAndStopRuns(bounded, repository, factory.CauseControlPaused, principal)
	if err != nil {
		return factory.PauseReceipt{}, err
	}
	if err = c.Store.FreezeAttemptAllowances(bounded, repository, withdrawal.Revision, time.Now()); err != nil {
		return factory.PauseReceipt{}, err
	}
	receipt := factory.PauseReceipt{Withdrawal: withdrawal, Runs: outcomes, CommandID: cmd.ID, Paused: true}
	if err = receipt.Validate(); err != nil {
		return factory.PauseReceipt{}, err
	}
	outcome, err := json.Marshal(receipt)
	if err != nil {
		return factory.PauseReceipt{}, err
	}
	if err = c.Store.FinishFactoryCommand(bounded, cmd.ID, string(outcome), time.Now()); err != nil {
		return factory.PauseReceipt{}, err
	}
	return receipt, nil
}

func replayPause(stored factory.Command) (factory.PauseReceipt, error) {
	if stored.Finished == "" {
		return factory.PauseReceipt{}, ErrCommandRunning
	}
	var receipt factory.PauseReceipt
	if err := json.Unmarshal([]byte(stored.Outcome), &receipt); err != nil {
		return factory.PauseReceipt{}, err
	}
	return receipt, nil
}

// withdrawAndStopRuns closes dispatch first and then settles every
// outstanding run of the repository's project. It is the shared first
// half of pause and coordinated Project stop.
func (c *Coordinator) withdrawAndStopRuns(ctx context.Context, repository int64, cause, principal string) (factory.Withdrawal, []factory.RunStopOutcome, error) {
	withdrawal, err := c.Store.WithdrawDispatch(ctx, repository, cause, principal)
	if err != nil {
		return factory.Withdrawal{}, nil, err
	}
	withdrawal.Publications = c.cancelRepositoryPublications(ctx, repository)
	withdrawal.Merges = c.cancelRepositoryMerges(ctx, repository)
	outcomes := []factory.RunStopOutcome{}
	p, err := c.Store.ProjectByRepository(ctx, repository)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return withdrawal, outcomes, nil
		}
		return factory.Withdrawal{}, nil, err
	}
	runs, err := c.projectRuns(ctx, p.ID)
	if err != nil {
		return factory.Withdrawal{}, nil, err
	}
	for _, run := range runs {
		if run.Reconciled {
			continue
		}
		bounded, stop := context.WithTimeout(ctx, 2*time.Minute)
		settled := c.settleRun(bounded, run)
		stop()
		outcomes = append(outcomes, factory.RunStopOutcome{
			ID: settled.RunID, Outcome: settled.Outcome, Reason: settled.Reason,
			Confirmed: settled.Confirmed, Uncertain: settled.Uncertain,
		})
	}
	return withdrawal, outcomes, nil
}

// projectRuns lists one project's recorded runs: every unsettled run
// first, then recent history. Settled history never hides live work;
// the history tail stays bounded for display-sized decisions.
func (c *Coordinator) projectRuns(ctx context.Context, projectID string) ([]factory.Run, error) {
	outstanding, err := c.Store.ProjectUnsettledRuns(ctx, projectID, 1000)
	if err != nil {
		return nil, err
	}
	history, err := c.Store.ProjectFactoryRuns(ctx, projectID, 1000)
	if err != nil {
		return nil, err
	}
	out := make([]factory.Run, 0, len(outstanding)+len(history))
	seen := make(map[string]bool, len(outstanding)+len(history))
	for _, run := range outstanding {
		if seen[run.ID] {
			continue
		}
		seen[run.ID] = true
		out = append(out, run)
	}
	for _, run := range history {
		if seen[run.ID] {
			continue
		}
		seen[run.ID] = true
		out = append(out, run)
	}
	return out, nil
}

// StopProject withdraws dispatch and stops outstanding runs for one
// project without touching holds or host lifecycle. The lifecycle API
// runs the hold and host stop around this in the same order.
func (c *Coordinator) StopProject(ctx context.Context, principal, projectID string) (factory.Withdrawal, []factory.RunStopOutcome, error) {
	bounded, stop := context.WithTimeout(ctx, 10*time.Minute)
	defer stop()
	p, err := c.Store.Project(bounded, projectID)
	if err != nil {
		return factory.Withdrawal{}, nil, ErrNotFound
	}
	return c.withdrawAndStopRuns(bounded, p.RepositoryID, factory.CauseProjectStop, principal)
}

// ClearProjectStopAfterStart releases only the Project lifecycle cause after
// host start verification and confirmation that no unsettled Project run
// remains. Independent repository and grant causes stay closed.
func (c *Coordinator) ClearProjectStopAfterStart(ctx context.Context, projectID string, verification factory.StartVerification) (bool, error) {
	if err := verification.Validate(); err != nil {
		return false, err
	}
	p, err := c.Store.Project(ctx, projectID)
	if err != nil {
		return false, err
	}
	open, revision, withdrawal, err := c.Store.DispatchState(ctx, p.RepositoryID)
	if err != nil || open {
		return open, err
	}
	hasProjectStop := false
	for _, cause := range withdrawal.ActiveCauses {
		if cause == factory.CauseProjectStop {
			hasProjectStop = true
			break
		}
	}
	if !hasProjectStop || len(verification.Revived) != 0 || len(verification.Unverified) != 0 {
		return false, nil
	}
	unsettled, err := c.Store.ProjectUnsettledRuns(ctx, p.ID, 1)
	if err != nil {
		return false, err
	}
	if len(unsettled) != 0 {
		return false, nil
	}
	open, _, err = c.Store.ClearProjectStop(ctx, p.RepositoryID, revision)
	return open, err
}

// ResumeRepository reopens a paused gate after every recorded run settles
// and every grant revalidates. It consumes the coordinator-only reopen
// hook and launches nothing; queued work stays queued.
func (c *Coordinator) ResumeRepository(ctx context.Context, commandID, principal string, repository int64) (factory.ResumeReceipt, error) {
	bounded, stop := context.WithTimeout(ctx, 2*time.Minute)
	defer stop()
	if repository <= 0 {
		return factory.ResumeReceipt{}, errors.New("invalid resume target")
	}
	if p, err := c.Store.ProjectByRepository(bounded, repository); err == nil {
		runs, err := c.projectRuns(bounded, p.ID)
		if err != nil {
			return factory.ResumeReceipt{}, err
		}
		pending := []string{}
		for _, run := range runs {
			if !run.Reconciled {
				pending = append(pending, run.ID)
			}
		}
		if len(pending) > 0 {
			return factory.ResumeReceipt{}, ErrPendingRuns{Runs: pending}
		}
	} else if !errors.Is(err, store.ErrNotFound) {
		return factory.ResumeReceipt{}, err
	}
	open, revision, _, err := c.Store.DispatchState(bounded, repository)
	if err != nil {
		return factory.ResumeReceipt{}, err
	}
	if open {
		effective, err := c.EffectiveAuthority(bounded, repository)
		if err != nil {
			return factory.ResumeReceipt{}, err
		}
		return factory.ResumeReceipt{Effective: effective, CommandID: commandID, Revision: revision}, nil
	}
	payload := `{"cause":"control_resume"}`
	target := grantTarget("dispatch", repository)
	cmd := factory.Command{
		ID: commandID, Type: factory.CommandResume, Target: target, Principal: principal,
		Payload: payload, Digest: factory.SettingsDigest(factory.CommandResume, target, payload),
	}
	if err := cmd.Validate(); err != nil {
		return factory.ResumeReceipt{}, err
	}
	stored, created, err := c.Store.RecordFactoryCommand(bounded, cmd, time.Now())
	if err != nil {
		return factory.ResumeReceipt{}, err
	}
	if !created {
		return replayResume(stored)
	}
	reopened, err := c.ReopenDispatch(bounded, factory.NewID(), principal, repository, revision)
	if err != nil {
		return factory.ResumeReceipt{}, err
	}
	receipt := factory.ResumeReceipt{Effective: reopened.Effective, CommandID: cmd.ID, Revision: reopened.Revision, Reopened: reopened.Effective.DispatchOpen}
	if err = receipt.Validate(); err != nil {
		return factory.ResumeReceipt{}, err
	}
	outcome, err := json.Marshal(receipt)
	if err != nil {
		return factory.ResumeReceipt{}, err
	}
	if err = c.Store.FinishFactoryCommand(bounded, cmd.ID, string(outcome), time.Now()); err != nil {
		return factory.ResumeReceipt{}, err
	}
	return receipt, nil
}

func replayResume(stored factory.Command) (factory.ResumeReceipt, error) {
	if stored.Finished == "" {
		return factory.ResumeReceipt{}, ErrCommandRunning
	}
	var receipt factory.ResumeReceipt
	if err := json.Unmarshal([]byte(stored.Outcome), &receipt); err != nil {
		return factory.ResumeReceipt{}, err
	}
	return receipt, nil
}
