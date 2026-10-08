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

// ErrTakeoverFailed reports a takeover the host did not confirm. The
// command stays unfinished so the caller refreshes instead of assuming a
// copy that may not exist.
var ErrTakeoverFailed = errors.New("factory takeover unconfirmed")

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
	verification.DispatchOpen, _, _, err = c.Store.DispatchState(bounded, p.RepositoryID)
	if err != nil {
		return factory.StartVerification{}, err
	}
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
