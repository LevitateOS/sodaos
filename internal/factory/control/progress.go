package control

import (
	"context"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
)

const progressInterval = 10 * time.Second

type progressPassKey struct{}

// Nested dispatch/publication/check/merge passes share one owner. Native
// withdrawal remains independent so a running CLI never delays revocation.
func (c *Coordinator) progressPass(ctx context.Context) (context.Context, func()) {
	if ctx.Value(progressPassKey{}) == c {
		return ctx, func() {}
	}
	c.progressMu.Lock()
	return context.WithValue(ctx, progressPassKey{}, c), c.progressMu.Unlock
}

// Publication and Actions activity advances the native revision without an
// issue delivery. Refresh the existing readiness record from authoritative
// evidence; an unchanged acceptance keeps its semantic revision and allowance.
// Store authority can reopen without advancing the native revision.
func (c *Coordinator) refreshReadiness(ctx context.Context, control factory.IssueControl) (factory.IssueControl, error) {
	if c.Readiness == nil {
		return control, nil
	}
	authorityChanged := false
	if control.Readiness == factory.ReadinessNotAuthorized && control.Reason == factory.BlockerAuthorityMissing {
		authority, err := c.EffectiveAuthority(ctx, control.Repository)
		if err != nil {
			return factory.IssueControl{}, err
		}
		authorityChanged = factory.AuthorityFingerprint(authority) != control.Authority
		if !authorityChanged {
			return control, nil
		}
	}
	budget := readinessBudgetFrom(ctx)
	if budget != nil && !budget.reserveNativeRPC(1) {
		return factory.IssueControl{}, ErrReadinessPassPending
	}
	revision, idle, err := c.Readiness.ObserveNativeRevision(ctx)
	if err != nil {
		return factory.IssueControl{}, err
	}
	if !idle {
		return factory.IssueControl{}, ErrReadinessPassPending
	}
	if revision == control.NativeRev && !authorityChanged {
		return control, nil
	}
	reserved, err := c.reserveIssueAssessment(ctx, factory.DependenceRef{Repository: control.Repository, Issue: control.Issue})
	if err != nil {
		return factory.IssueControl{}, err
	}
	if !reserved {
		return factory.IssueControl{}, ErrReadinessPassPending
	}
	observed, err := c.assessOne(ctx, control.Repository, control.Issue)
	return observed.control, err
}

// The database owner revisits durable work when no new issue delivery arrives:
// explicit Retry, delayed checks and settled publication children all use the
// same bounded, idempotent passes as intake. This never creates new authority.
func (c *Coordinator) startProgress(ctx context.Context) {
	c.ownerMu.Lock()
	defer c.ownerMu.Unlock()
	if c.closing.Load() || c.lock == nil {
		return
	}
	ctx, c.progressCancel = context.WithCancel(ctx)
	c.progressDone = make(chan struct{})
	go c.runProgress(ctx, c.progressDone)
}

func (c *Coordinator) runProgress(ctx context.Context, done chan struct{}) {
	defer close(done)
	ticker := time.NewTicker(progressInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.requestGate.RLock()
			if !c.closing.Load() && ctx.Err() == nil {
				c.advanceProgress(ctx)
			}
			c.requestGate.RUnlock()
		}
	}
}

func (c *Coordinator) advanceProgress(ctx context.Context) {
	ctx, unlock := c.progressPass(ctx)
	defer unlock()
	ctx, ownsPass, cancelPass := c.readinessPass(ctx)
	defer cancelPass()
	budget := readinessBudgetFrom(ctx)
	runs, err := c.Store.FactoryUnsettledRuns(ctx, MaxDispatchRecovery)
	if err != nil {
		return
	}
	for _, run := range runs {
		if ctx.Err() != nil || !budget.reserveNativeRPC(1) {
			break
		}
		state, err := c.Host.FactoryInspect(ctx, project.FactoryInspect{Project: run.ProjectID, ID: run.ID})
		if err != nil || state.ID != run.ID || state.Project != run.ProjectID || state.Live {
			continue
		}
		if _, terminal := settleOutcome(state); terminal {
			// Stop, at most three broker close calls, and its metadata read.
			if !budget.reserveNativeRPC(5) {
				break
			}
			c.settleRun(ctx, run)
		}
	}
	c.Dispatch(ctx)
	if ownsPass {
		_, _ = c.drainReadinessWork(ctx, "")
	}
}
