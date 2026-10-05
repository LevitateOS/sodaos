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

// StopReceipt is the durable outcome of one stop command. Uncertain stays
// fenced: the run keeps its unsettled state for reconcile to settle.
type StopReceipt struct {
	RunID     string `json:"run_id"`
	Outcome   string `json:"outcome,omitempty"`
	Reason    string `json:"reason,omitempty"`
	Confirmed bool   `json:"confirmed"`
	Uncertain bool   `json:"uncertain"`
}

// FencedRun names a run reconcile could not settle and why.
type FencedRun struct {
	ID     string `json:"id"`
	Reason string `json:"reason"`
}

// ReconcileReceipt is the durable outcome of one reconcile command.
type ReconcileReceipt struct {
	Settled     []string         `json:"settled"`
	Fenced      []FencedRun      `json:"fenced,omitempty"`
	Readiness   *ReadinessReport `json:"readiness,omitempty"`
	Dispatch    *DispatchReport  `json:"dispatch,omitempty"`
	Publication *PublishReport   `json:"publication,omitempty"`
	Checks      *CheckReport     `json:"checks,omitempty"`
	Merge       *MergeReport     `json:"merge,omitempty"`
}

// Stop retires one recorded run and settles it when retirement and broker
// closure confirm. A lost reply replays the durable outcome; it never
// re-executes the stop.
func (c *Coordinator) Stop(ctx context.Context, cmd factory.Command) (StopReceipt, error) {
	if cmd.Type != factory.CommandStop {
		return StopReceipt{}, errors.New("command is not a stop")
	}
	bounded, stop := context.WithTimeout(ctx, 3*time.Minute)
	defer stop()
	run, err := c.Store.FactoryRun(bounded, cmd.Target)
	if err != nil {
		return StopReceipt{}, ErrNotFound
	}
	stored, created, err := c.Store.RecordFactoryCommand(bounded, cmd, time.Now())
	if err != nil {
		return StopReceipt{}, err
	}
	if !created {
		return replayStop(stored)
	}
	receipt := c.settleRun(bounded, run)
	outcome, err := json.Marshal(receipt)
	if err != nil {
		return StopReceipt{}, err
	}
	if err = c.Store.FinishFactoryCommand(bounded, cmd.ID, string(outcome), time.Now()); err != nil {
		return StopReceipt{}, err
	}
	return receipt, nil
}

func replayStop(stored factory.Command) (StopReceipt, error) {
	if stored.Finished == "" {
		return StopReceipt{}, ErrCommandRunning
	}
	var receipt StopReceipt
	if err := json.Unmarshal([]byte(stored.Outcome), &receipt); err != nil {
		return StopReceipt{}, err
	}
	return receipt, nil
}

// Reconcile retires every outstanding recorded run and settles each one
// whose retirement, credential return and broker closure confirm. Runs with
// unresolved effects stay fenced for a later command.
func (c *Coordinator) Reconcile(ctx context.Context, cmd factory.Command) (ReconcileReceipt, error) {
	if cmd.Type != factory.CommandReconcile {
		return ReconcileReceipt{}, errors.New("command is not a reconcile")
	}
	bounded, stop := context.WithTimeout(ctx, 10*time.Minute)
	defer stop()
	stored, created, err := c.Store.RecordFactoryCommand(bounded, cmd, time.Now())
	if err != nil {
		return ReconcileReceipt{}, err
	}
	if !created {
		return replayReconcile(stored)
	}
	receipt, err := c.reconcileRuns(bounded)
	if err != nil {
		return ReconcileReceipt{}, err
	}
	report := c.reconcileReadinessAll(bounded)
	receipt.Readiness = &report
	if c.DispatchReads != nil {
		dispatch := c.Dispatch(bounded)
		receipt.Dispatch = &dispatch
	}
	if c.Publication != nil {
		published := c.PublishPass(bounded)
		receipt.Publication = &published
	}
	if c.Checks != nil {
		assessed := c.CheckPass(bounded)
		receipt.Checks = &assessed
	}
	if c.Merges != nil {
		merged := c.MergePass(bounded)
		receipt.Merge = &merged
	}
	outcome, err := json.Marshal(receipt)
	if err != nil {
		return ReconcileReceipt{}, err
	}
	if err = c.Store.FinishFactoryCommand(bounded, cmd.ID, string(outcome), time.Now()); err != nil {
		return ReconcileReceipt{}, err
	}
	return receipt, nil
}

func replayReconcile(stored factory.Command) (ReconcileReceipt, error) {
	if stored.Finished == "" {
		return ReconcileReceipt{}, ErrCommandRunning
	}
	var receipt ReconcileReceipt
	if err := json.Unmarshal([]byte(stored.Outcome), &receipt); err != nil {
		return ReconcileReceipt{}, err
	}
	return receipt, nil
}

// settleAbandonedCommands finishes stop and reconcile commands that a
// crash left recorded but unfinished. Start just reconciled every run,
// so each command settles from durable target state through the existing
// settle path: an established outcome replays from the record without
// host calls, and an unconfirmed effect finishes uncertain instead of
// timing out into success. Other command types stay running; their
// effects have no durable derivation here. Visited identities bound the
// walk so commands that stay running never spin recovery.
func (c *Coordinator) settleAbandonedCommands(ctx context.Context, settle ReconcileReceipt) error {
	visited := map[string]bool{}
	for {
		batch, err := c.Store.UnfinishedCommands(ctx, 100)
		if err != nil {
			return err
		}
		progress := false
		for _, cmd := range batch {
			if visited[cmd.ID] {
				continue
			}
			visited[cmd.ID] = true
			progress = true
			if err := c.settleAbandonedCommand(ctx, cmd, settle); err != nil {
				return err
			}
		}
		if !progress || len(batch) < 100 {
			return nil
		}
	}
}

func (c *Coordinator) settleAbandonedCommand(ctx context.Context, cmd factory.Command, settle ReconcileReceipt) error {
	switch cmd.Type {
	case factory.CommandStop:
		run, err := c.Store.FactoryRun(ctx, cmd.Target)
		if err != nil {
			if !errors.Is(err, store.ErrNotFound) {
				return err
			}
			return c.finishAbandonedCommand(ctx, cmd.ID, StopReceipt{RunID: cmd.Target, Uncertain: true, Reason: "stop target missing"})
		}
		bounded, stop := context.WithTimeout(ctx, 2*time.Minute)
		receipt := c.settleRun(bounded, run)
		stop()
		return c.finishAbandonedCommand(ctx, cmd.ID, receipt)
	case factory.CommandReconcile:
		return c.finishAbandonedCommand(ctx, cmd.ID, ReconcileReceipt{Settled: settle.Settled, Fenced: settle.Fenced})
	default:
		return nil
	}
}

func (c *Coordinator) finishAbandonedCommand(ctx context.Context, id string, receipt any) error {
	outcome, err := json.Marshal(receipt)
	if err != nil {
		return err
	}
	if err = c.Store.FinishFactoryCommand(ctx, id, string(outcome), time.Now()); err != nil {
		// A concurrent executor finished first; its receipt stands.
		if errors.Is(err, store.ErrNotFound) {
			return nil
		}
		return err
	}
	return nil
}

func (c *Coordinator) reconcileRuns(ctx context.Context) (ReconcileReceipt, error) {
	receipt := ReconcileReceipt{Settled: []string{}}
	// Outstanding work first, in progressing batches: settled history
	// never displaces an unresolved run no matter how many settled rows
	// exist. Fenced runs stay unsettled, so visited identities bound the
	// walk instead of batch fullness alone.
	visited := map[string]bool{}
	for {
		batch, err := c.Store.FactoryUnsettledRuns(ctx, 1000)
		if err != nil {
			return receipt, err
		}
		progress := false
		for _, run := range batch {
			if visited[run.ID] {
				continue
			}
			visited[run.ID] = true
			progress = true
			bounded, stop := context.WithTimeout(ctx, 2*time.Minute)
			settled := c.settleRun(bounded, run)
			stop()
			if settled.Uncertain {
				receipt.Fenced = append(receipt.Fenced, FencedRun{ID: run.ID, Reason: settled.Reason})
				continue
			}
			receipt.Settled = append(receipt.Settled, run.ID)
		}
		if !progress || len(batch) < 1000 {
			return receipt, nil
		}
	}
}

// settleRun drives one recorded run to its settled state: host stop, broker
// closure, then the durable record. A settled run never reopens; anything
// unconfirmed stays fenced with its observed facts saved.
func (c *Coordinator) settleRun(ctx context.Context, run factory.Run) StopReceipt {
	receipt := StopReceipt{RunID: run.ID}
	if run.Reconciled {
		receipt.Confirmed, receipt.Outcome, receipt.Reason = true, string(run.Outcome), run.Summary
		return receipt
	}
	state, err := c.Host.FactoryStop(ctx, project.FactoryStop{Project: run.ProjectID, ID: run.ID})
	if err != nil {
		receipt.Uncertain, receipt.Reason = true, "host stop unconfirmed"
		return receipt
	}
	// The broker fence is idempotent: ride out a transient close
	// failure before fencing the run.
	var closeErr error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				break
			case <-time.After(time.Second):
			}
		}
		if closeErr = c.Broker.CloseExecution(ctx, identity.Factory, run.ID); closeErr == nil {
			break
		}
	}
	if closeErr != nil {
		receipt.Uncertain, receipt.Reason = true, "broker closure unconfirmed"
		return receipt
	}
	execution, err := c.Broker.GetExecution(ctx, identity.Factory, run.ID)
	if err != nil && !errors.Is(err, identity.ErrNotFound) {
		receipt.Uncertain, receipt.Reason = true, "broker execution unreadable"
		return receipt
	}
	outcome, ok := settleOutcome(state)
	if !ok {
		receipt.Uncertain, receipt.Reason = true, settleReason(state)
		return receipt
	}
	if err = adoptExecutionFacts(&run, state, execution); err != nil {
		receipt.Uncertain, receipt.Reason = true, "recorded execution conflicts with observed facts"
		return receipt
	}
	run.Outcome, run.Summary, run.Reconciled = outcome, settleReason(state), true
	if err = c.Store.SaveFactoryRun(ctx, run); err != nil {
		receipt.Uncertain, receipt.Reason = true, "run record unsettled"
		return receipt
	}
	// Best-effort dispatch accounting: confirmed usage, reservation
	// consume and the recorded result. Recovery replays anything missed;
	// a completed issue retriggers its dependants.
	if finished, ok := AccountSettledRun(ctx, c.Store, run, state.Output, time.Now()); ok && finished.Outcome == factory.Succeeded {
		c.assessDispatchDependants(ctx, finished.Repository, finished.Issue)
		c.publishAfterSettle(ctx, finished)
	}
	// A completed run beyond its assignment's finishing one advances the
	// same PR as a correction; the finishing run published above.
	c.correctAfterSettle(ctx, run, state.Output)
	// A completed reviewer run submits its genuine verdict through the
	// separate reviewer actor. Runs without an assignment (like reviews)
	// never reach the accounting path above.
	if run.Role == project.RoleReviewer && outcome == factory.Succeeded {
		c.reviewAfterSettle(ctx, run, state.Output)
	}
	receipt.Confirmed, receipt.Outcome, receipt.Reason = true, string(outcome), run.Summary
	return receipt
}

// settleOutcome maps confirmed host retirement to its run outcome. Uncertain
// retirement never settles: a pending native effect cannot appear cancelled.
func settleOutcome(state project.FactoryState) (factory.Outcome, bool) {
	if state.Retirement == "uncertain" {
		return "", false
	}
	switch state.Phase {
	case project.FactoryCompleted:
		return factory.Succeeded, true
	case project.FactoryFailed:
		return factory.Failed, true
	case project.FactoryStopped:
		return factory.Cancelled, true
	default:
		return "", false
	}
}

func settleReason(state project.FactoryState) string {
	if state.Phase == project.FactoryCompleted && state.Output != "" {
		if len(state.Output) > 4096 {
			return state.Output[:4096]
		}
		return state.Output
	}
	if state.Reason != "" {
		return state.Reason
	}
	return "run " + state.Phase
}

// adoptExecutionFacts fills recorded lease/binding/credential facts from host
// and broker observations. First-recorded facts never change; a conflict
// fences the run instead of rewriting its identity.
func adoptExecutionFacts(run *factory.Run, state project.FactoryState, execution identity.Execution) error {
	if run.IdentityLeaseID == "" {
		run.IdentityLeaseID, run.IdentityGeneration = state.LeaseID, state.Generation
	} else if run.IdentityLeaseID != state.LeaseID || run.IdentityGeneration != state.Generation {
		return errors.New("lease differs from recorded execution")
	}
	if execution.Binding != nil {
		if run.IdentityBinding == nil {
			binding := *execution.Binding
			run.IdentityBinding, run.CredentialDelegated = &binding, true
		} else if *run.IdentityBinding != *execution.Binding {
			return errors.New("binding differs from recorded execution")
		}
	}
	if state.CredentialReturned {
		run.CredentialReturned = true
	}
	return run.Validate()
}
