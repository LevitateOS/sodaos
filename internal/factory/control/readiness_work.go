package control

import (
	"context"
	"errors"
	"sync/atomic"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/store"
)

const (
	readinessPassLimit       = 120 * time.Second
	readinessCleanupReserve  = 2 * time.Second
	readinessMaxPages        = 8
	readinessMaxAssessments  = 32
	readinessMaxEvidenceRead = 52
	// Each production evidence bracket makes two revision observations and
	// one snapshot request. Reuse polls spend from the same absolute-call cap.
	readinessMaxNativeRPC  = 156
	readinessMaxSelections = 64
)

var ErrReadinessPassPending = errors.New("readiness work remains pending")

type readinessPassKey struct{}

type readinessPass struct {
	outerDeadline time.Time
	evidence      atomic.Int32
	nativeRPC     atomic.Int32
	reserved      atomic.Int32
	pages         atomic.Int32
	assessments   atomic.Int32
	drained       atomic.Bool
	assessed      map[readinessAssessmentKey]readinessCachedAssessment
}

type readinessAssessmentKey struct {
	generation int64
	ref        factory.DependenceRef
}

type readinessCachedAssessment struct {
	revision      int64
	authorityHash string
	outcome       assessOutcome
}

type readinessDrainResult struct {
	targetDone    bool
	targetChanged bool
	failed        bool
}

func (c *Coordinator) readinessPass(ctx context.Context) (context.Context, bool, context.CancelFunc) {
	if _, ok := ctx.Value(readinessPassKey{}).(*readinessPass); ok {
		return ctx, false, func() {}
	}
	outerDeadline := time.Now().Add(readinessPassLimit)
	if deadline, ok := ctx.Deadline(); ok && deadline.Before(outerDeadline) {
		outerDeadline = deadline
	}
	remaining := time.Until(outerDeadline)
	cleanupReserve := time.Duration(0)
	if remaining > 0 {
		cleanupReserve = min(readinessCleanupReserve, remaining/10)
	}
	workDeadline := outerDeadline.Add(-cleanupReserve)
	budget := &readinessPass{
		outerDeadline: outerDeadline,
		assessed:      make(map[readinessAssessmentKey]readinessCachedAssessment),
	}
	bounded, cancel := context.WithDeadline(ctx, workDeadline)
	return context.WithValue(bounded, readinessPassKey{}, budget), true, cancel
}

func readinessBudgetFrom(ctx context.Context) *readinessPass {
	budget, _ := ctx.Value(readinessPassKey{}).(*readinessPass)
	return budget
}

func (b *readinessPass) reserveEvidence(count int) bool {
	for {
		current := b.reserved.Load()
		if count < 1 || current+int32(count) > readinessMaxEvidenceRead {
			return false
		}
		if b.reserved.CompareAndSwap(current, current+int32(count)) {
			return true
		}
	}
}

func (b *readinessPass) recordEvidenceRead() error {
	if b == nil {
		return nil
	}
	if !b.reserveNativeRPC(3) {
		return ErrReadinessPassPending
	}
	if b.evidence.Add(1) > readinessMaxEvidenceRead {
		b.nativeRPC.Add(-3)
		return ErrReadinessPassPending
	}
	return nil
}

func (b *readinessPass) reserveNativeRPC(count int) bool {
	for {
		current := b.nativeRPC.Load()
		if count < 1 || current+int32(count) > readinessMaxNativeRPC {
			return false
		}
		if b.nativeRPC.CompareAndSwap(current, current+int32(count)) {
			return true
		}
	}
}

func (c *Coordinator) reserveIssueAssessment(ctx context.Context, ref factory.DependenceRef) (bool, error) {
	budget := readinessBudgetFrom(ctx)
	if budget == nil {
		return false, ErrReadinessPassPending
	}
	cost := 1
	head, err := c.Store.AcceptanceHead(ctx, ref.Repository, ref.Issue)
	if err == nil {
		decision, decisionErr := c.Store.AcceptanceDecision(ctx, head)
		if decisionErr != nil {
			return false, decisionErr
		}
		cost += len(decision.Prerequisites)
	} else if !errors.Is(err, store.ErrNotFound) {
		return false, err
	}
	if cost > readinessMaxEvidenceRead {
		return false, ErrReadinessPassPending
	}
	for {
		current := budget.assessments.Load()
		if current >= readinessMaxAssessments {
			return false, ErrReadinessPassPending
		}
		if budget.assessments.CompareAndSwap(current, current+1) {
			break
		}
	}
	if !budget.reserveEvidence(cost) {
		budget.assessments.Add(-1)
		return false, ErrReadinessPassPending
	}
	return true, nil
}

func (c *Coordinator) recordReadinessEvidence(ctx context.Context) error {
	budget := readinessBudgetFrom(ctx)
	if budget == nil {
		return nil
	}
	return budget.recordEvidenceRead()
}

func (c *Coordinator) drainReadinessWork(ctx context.Context, targetDelivery string) (readinessDrainResult, error) {
	workCtx, _, cancelPass := c.readinessPass(ctx)
	defer cancelPass()
	budget := readinessBudgetFrom(workCtx)
	if !budget.drained.CompareAndSwap(false, true) {
		return readinessDrainResult{}, nil
	}
	if !c.cascadeMu.TryLock() {
		return readinessDrainResult{}, ErrReadinessPassPending
	}
	defer c.cascadeMu.Unlock()

	var result readinessDrainResult
	for selected := 0; selected < readinessMaxSelections; selected++ {
		if err := workCtx.Err(); err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				return result, ErrReadinessPassPending
			}
			return result, err
		}
		work, err := c.Store.NextReadinessWork(workCtx, time.Now())
		if errors.Is(err, store.ErrNotFound) {
			return result, nil
		}
		if err != nil {
			return result, err
		}
		current, err := c.Store.BeginReadinessWork(workCtx, work.ID, time.Now())
		if errors.Is(err, store.ErrReadinessPending) {
			continue
		}
		if err != nil {
			return result, err
		}
		done, changed, err := c.processReadinessSource(workCtx, current)
		if err != nil {
			if errors.Is(err, ErrReadinessPassPending) {
				return result, nil
			}
			if errors.Is(workCtx.Err(), context.DeadlineExceeded) {
				return result, ErrReadinessPassPending
			}
			cleanupCtx, cleanupCancel := readinessCleanupContext(workCtx, budget.outerDeadline)
			cleanupErr := c.Store.DeferReadinessWork(cleanupCtx, current.ID, time.Now())
			cleanupCancel()
			if cleanupErr != nil {
				return result, errors.Join(err, cleanupErr)
			}
			result.failed = true
			if workCtx.Err() != nil {
				return result, workCtx.Err()
			}
			if targetDelivery != "" && current.Delivery == targetDelivery {
				return result, err
			}
			continue
		}
		if done && targetDelivery != "" && current.Delivery == targetDelivery {
			result.targetDone = true
			result.targetChanged = changed
		}
	}
	return result, nil
}

func readinessCleanupContext(ctx context.Context, deadline time.Time) (context.Context, context.CancelFunc) {
	cleanupDeadline := time.Now().Add(readinessCleanupReserve)
	if deadline.Before(cleanupDeadline) {
		cleanupDeadline = deadline
	}
	cleanup, cancel := context.WithDeadline(context.WithoutCancel(ctx), cleanupDeadline)
	return cleanup, cancel
}

func (c *Coordinator) processReadinessSource(ctx context.Context, work store.ReadinessWork) (bool, bool, error) {
	budget := readinessBudgetFrom(ctx)
	rootChanged := false
	for {
		if err := ctx.Err(); err != nil {
			return false, false, err
		}
		node, err := c.Store.NextReadinessWorkNode(ctx, work.ID)
		if errors.Is(err, store.ErrNotFound) {
			current, readErr := c.Store.ReadinessWork(ctx, work.ID)
			if readErr != nil {
				return false, false, readErr
			}
			complete, finishErr := c.Store.CompleteReadinessWork(ctx, current, time.Now())
			if finishErr != nil || !complete {
				return false, false, finishErr
			}
			return true, rootChanged, nil
		}
		if err != nil {
			return false, false, err
		}
		if node.State == "queued" {
			outcome, reused, reuseErr := c.reuseReadinessAssessment(ctx, work.Generation, node.Ref)
			if reuseErr != nil {
				return false, false, reuseErr
			}
			if !reused {
				reserved, reserveErr := c.reserveIssueAssessment(ctx, node.Ref)
				if reserveErr != nil {
					return false, false, reserveErr
				}
				if !reserved {
					return false, false, ErrReadinessPassPending
				}
				outcome, err = c.assessOne(ctx, node.Ref.Repository, node.Ref.Issue)
				if err != nil {
					return false, false, err
				}
				if outcome.revision > 0 && outcome.authorityHash != "" {
					budget.assessed[readinessAssessmentKey{generation: work.Generation, ref: node.Ref}] = readinessCachedAssessment{
						revision: outcome.revision, authorityHash: outcome.authorityHash, outcome: outcome,
					}
				}
			}
			if node.Ref == work.Root {
				rootChanged = outcome.changed
			}
			if err = c.Store.CheckpointReadinessAssessment(ctx, work, node, outcome.skip); err != nil {
				return false, false, err
			}
			continue
		}
		if node.State != "scanning" {
			return false, false, ErrReadinessPassPending
		}
		if budget.pages.Add(1) > readinessMaxPages {
			budget.pages.Add(-1)
			return false, false, ErrReadinessPassPending
		}
		if _, err = c.Store.CheckpointReadinessPage(ctx, work, node); err != nil {
			return false, false, err
		}
	}
}

func (c *Coordinator) reuseReadinessAssessment(ctx context.Context, generation int64, ref factory.DependenceRef) (assessOutcome, bool, error) {
	budget := readinessBudgetFrom(ctx)
	cached, ok := budget.assessed[readinessAssessmentKey{generation: generation, ref: ref}]
	if !ok || cached.revision < 1 || c.Readiness == nil {
		return assessOutcome{}, false, nil
	}
	if !budget.reserveNativeRPC(1) {
		return assessOutcome{}, false, nil
	}
	revision, idle, err := c.Readiness.ObserveNativeRevision(ctx)
	if err != nil || !idle || revision != cached.revision {
		return assessOutcome{}, false, nil
	}
	authority, err := c.EffectiveAuthority(ctx, ref.Repository)
	if err != nil {
		return assessOutcome{}, false, err
	}
	if factory.AuthorityFingerprint(authority) != cached.authorityHash {
		return assessOutcome{}, false, nil
	}
	// The originating assessment already recorded the issue control. Reuse
	// only its classification for this same-generation graph checkpoint.
	return cached.outcome, true, nil
}
