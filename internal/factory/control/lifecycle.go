package control

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/identity"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
)

// ErrPendingRuns reports a resume, retry or takeover refused while recorded
// runs are unsettled. The caller stops or reconciles first; pending native
// effects never appear cancelled, queued or taken over.
type ErrPendingRuns struct {
	Runs []string
}

func (e ErrPendingRuns) Error() string { return "factory runs are not settled" }

// ErrTakeoverFailed reports a takeover the host did not confirm. The
// command stays unfinished so the caller refreshes instead of assuming a
// copy that may not exist.
var ErrTakeoverFailed = errors.New("factory takeover unconfirmed")

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

// projectRuns lists the recorded runs of one project, most recent first.
func (c *Coordinator) projectRuns(ctx context.Context, projectID string) ([]factory.Run, error) {
	runs, err := c.Store.FactoryRuns(ctx, 1000)
	if err != nil {
		return nil, err
	}
	out := []factory.Run{}
	for _, run := range runs {
		if run.ProjectID == projectID {
			out = append(out, run)
		}
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
	receipt := factory.ResumeReceipt{Effective: reopened.Effective, CommandID: cmd.ID, Revision: reopened.Revision, Reopened: true}
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

// RetryRun records an explicit retry as a queued decision after the prior
// run is reconciled and current authority revalidates. Remaining
// allowances cannot be established without usage records, so the retry
// stays queued instead of launching; a later scheduler consumes it.
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
	stored, created, err := c.Store.RecordFactoryCommand(bounded, cmd, time.Now())
	if err != nil {
		return factory.RetryDecision{}, err
	}
	if !created {
		return replayRetry(stored)
	}
	decision := factory.RetryDecision{
		Prior: runID, CommandID: cmd.ID, Queued: true,
		Reason: "dispatch automation unavailable; retry recorded without launching",
	}
	if err = decision.Validate(); err != nil {
		return factory.RetryDecision{}, err
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

// TakeoverRun copies a reconciled run's retained work into the admitted
// member's own checkout. Unsettled runs refuse: takeover cannot finish
// while retirement or credential accounting is unresolved. Provider homes
// and role Git configuration are excluded; no credential is transferred.
func (c *Coordinator) TakeoverRun(ctx context.Context, commandID, principal, runID, member string) (factory.TakeoverRecord, error) {
	bounded, stop := context.WithTimeout(ctx, 5*time.Minute)
	defer stop()
	run, err := c.Store.FactoryRun(bounded, runID)
	if err != nil {
		return factory.TakeoverRecord{}, ErrNotFound
	}
	if !run.Reconciled {
		return factory.TakeoverRecord{}, ErrPendingRuns{Runs: []string{runID}}
	}
	takeover := project.FactoryTakeover{Project: run.ProjectID, ID: runID, Member: member}
	if err = takeover.Validate(); err != nil {
		return factory.TakeoverRecord{}, err
	}
	if existing, err := c.Store.Takeover(bounded, runID, member); err == nil {
		return existing, nil
	} else if !errors.Is(err, store.ErrNotFound) {
		return factory.TakeoverRecord{}, err
	}
	payload, err := json.Marshal(map[string]string{"member": member})
	if err != nil {
		return factory.TakeoverRecord{}, err
	}
	target := "run/" + runID
	cmd := factory.Command{
		ID: commandID, Type: factory.CommandTakeover, Target: target, Principal: principal,
		Payload: string(payload), Digest: factory.SettingsDigest(factory.CommandTakeover, target, string(payload)),
	}
	if err = cmd.Validate(); err != nil {
		return factory.TakeoverRecord{}, err
	}
	stored, created, err := c.Store.RecordFactoryCommand(bounded, cmd, time.Now())
	if err != nil {
		return factory.TakeoverRecord{}, err
	}
	if !created {
		return replayTakeover(stored)
	}
	result, err := c.Host.FactoryTakeover(bounded, takeover)
	if err != nil {
		return factory.TakeoverRecord{}, ErrTakeoverFailed
	}
	if err = result.Validate(); err != nil || result.ID != runID || result.Member != member || result.Project != run.ProjectID {
		return factory.TakeoverRecord{}, ErrTakeoverFailed
	}
	record := factory.TakeoverRecord{
		Run: runID, Member: member, Project: run.ProjectID,
		Dest: result.Destination, CommandID: cmd.ID, Copied: time.Now().UTC().Format(time.RFC3339),
	}
	record, err = c.Store.RecordTakeover(bounded, record)
	if err != nil {
		return factory.TakeoverRecord{}, err
	}
	outcome, err := json.Marshal(record)
	if err != nil {
		return factory.TakeoverRecord{}, err
	}
	if err = c.Store.FinishFactoryCommand(bounded, cmd.ID, string(outcome), time.Now()); err != nil {
		return factory.TakeoverRecord{}, err
	}
	return record, nil
}

func replayTakeover(stored factory.Command) (factory.TakeoverRecord, error) {
	if stored.Finished == "" {
		return factory.TakeoverRecord{}, ErrCommandRunning
	}
	var record factory.TakeoverRecord
	if err := json.Unmarshal([]byte(stored.Outcome), &record); err != nil {
		return factory.TakeoverRecord{}, err
	}
	return record, nil
}

// VerifyProjectStart proves a coordinated start revived no old run: no
// recorded run reports a live broker lease or a live native boundary.
// Runs the host or broker never recorded stay unverified, never clean:
// a future launch or acquisition under that identity could still
// proceed. Fenced runs stay fenced for reconcile; leases stay closed;
// grants, readiness and the maintenance hold are untouched by the start.
func (c *Coordinator) VerifyProjectStart(ctx context.Context, projectID string) (factory.StartVerification, error) {
	bounded, stop := context.WithTimeout(ctx, 5*time.Minute)
	defer stop()
	p, err := c.Store.Project(bounded, projectID)
	if err != nil {
		return factory.StartVerification{}, ErrNotFound
	}
	runs, err := c.projectRuns(bounded, p.ID)
	if err != nil {
		return factory.StartVerification{}, err
	}
	verification := factory.StartVerification{Revived: []string{}, Unverified: []string{}, Started: true}
	for _, run := range runs {
		revived, unverified := c.checkStartRun(bounded, run)
		if revived {
			verification.Revived = append(verification.Revived, run.ID)
		} else if unverified {
			verification.Unverified = append(verification.Unverified, run.ID)
		}
	}
	held, err := c.Store.MaintenanceHold(bounded, p.ID)
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		return factory.StartVerification{}, err
	}
	verification.Hold = err == nil && held.Hold
	return verification, nil
}

// checkStartRun classifies one recorded run after a coordinated start. A
// live lease or live native boundary is revived; an unreadable or
// never-recorded boundary is unverified; anything else is clean.
func (c *Coordinator) checkStartRun(ctx context.Context, run factory.Run) (revived, unverified bool) {
	execution, err := c.Broker.GetExecution(ctx, identity.Factory, run.ID)
	switch {
	case err != nil:
		return false, true
	case execution.State == identity.ExecutionLive:
		return true, false
	case execution.State != identity.ExecutionTerminal:
		return false, true
	}
	state, err := c.Host.FactoryInspect(ctx, project.FactoryInspect{Project: run.ProjectID, ID: run.ID})
	switch {
	case err != nil:
		return false, true
	case state.Live:
		return true, false
	default:
		return false, false
	}
}
