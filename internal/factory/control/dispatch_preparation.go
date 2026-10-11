package control

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
)

// Retire expired or invalidated setup even when the issue is no longer queued
// or planning now waits before reaching the preparation step. This runs under
// the same progress owner and budget as dispatch, without creating new work.
func (c *Coordinator) reconcileRetryPreparations(ctx context.Context) error {
	records, err := c.Store.PendingRetryPreparations(ctx, MaxDispatchRecovery)
	if err != nil {
		return err
	}
	for _, record := range records {
		p := record.Preparation
		if record.NotAfter > time.Now().Unix() {
			p.State = project.PrepareState{}
			if _, _, err := c.Store.AdmitCandidatePreparation(ctx, p, record.Admission); err == nil {
				continue
			} else if !retryPreparationInvalidated(err) {
				return err
			}
		}
		// A committed packet transfers ownership to its recorded run; a lost
		// reply or concurrent request must not retire that run's checkout.
		if _, err := c.Store.Assignment(ctx, record.Admission.OwnerAssignment); !errors.Is(err, store.ErrNotFound) {
			if err != nil {
				return err
			}
			continue
		}
		if err := c.retireRetryCoder(ctx, &attemptPlan{projectID: p.Preparation.Project, prep: record.Preparation}); err != nil {
			return err
		}
	}
	return nil
}

func retryPreparationInvalidated(err error) bool {
	return errors.Is(err, store.ErrAdmissionChanged) || errors.Is(err, store.ErrDispatchClosed) ||
		errors.Is(err, store.ErrStaleRevision) || errors.Is(err, store.ErrDispatchControlStale) ||
		errors.Is(err, store.ErrExplicitRetryIneligible) || errors.Is(err, store.ErrNotFound) ||
		errors.Is(err, factory.ErrAttemptClosed) || errors.Is(err, factory.ErrAttemptTimeExhausted)
}

// Explicit Retry starts a new private checkout. The preceding run's checkout
// and protected source snapshot remain inspectable; neither is reset in place.
func (c *Coordinator) prepareRetryCoder(ctx context.Context, plan *attemptPlan, command string) (bool, error) {
	source := plan.prep.Preparation
	if source.SourceCommit != plan.inputs.Tip {
		return false, errors.New("retry target differs from the approved source")
	}
	sum := sha256.Sum256([]byte("retry-coder\x00" + command + "\x00" + source.SourceCommit))
	prep := source
	prep.ID, prep.Revision = "f"+hex.EncodeToString(sum[:12]), 0
	requested := time.Now().Add(10 * time.Minute).Unix()
	if plan.deadline.Unix() < requested {
		requested = plan.deadline.Unix()
	}
	registration := store.CandidatePreparationRegistration{
		Repository: plan.control.Repository, Issue: plan.control.Issue, ActorID: plan.sponsorship.ActorID,
		Project: plan.projectID, OwnerAssignment: command, ChildAssignment: command, AttemptRoot: command,
		Connection: plan.sponsorship.Connection, Authority: plan.effective.Authority,
		GateRevision: plan.gateRev, Control: plan.control, RequestedNotAfter: requested,
	}
	// The gate is read before preparation admission, rather than after prompt
	// construction, so setup is fenced by the same revision as dispatch.
	open, revision, _, err := c.Store.DispatchState(ctx, plan.control.Repository)
	if err != nil || !open {
		return false, errors.New("retry dispatch gate is unavailable or closed")
	}
	registration.GateRevision = revision
	record, created, err := c.Store.AdmitCandidatePreparation(ctx, project.StoredPreparation{Preparation: prep}, registration)
	if err != nil {
		if prior, readErr := c.Store.CandidatePreparation(ctx, prep.ID); retryPreparationInvalidated(err) && readErr == nil &&
			prior.Admission.OwnerAssignment == command && prior.Admission.PublicationID == "" && !prior.Stopped {
			plan.prep = prior.Preparation
			if stopErr := c.retireRetryCoder(ctx, plan); stopErr != nil {
				return false, stopErr
			}
		}
		return false, err
	}
	stored := record.Preparation
	if record.Stopped || record.Retirement == "confirmed" {
		return false, errors.New("retry preparation was retired")
	}
	if stored.State.Ready {
		if !candidatePreparationStateMatches(stored.State, prep) {
			return false, errors.New("retry preparation ready identity differs")
		}
		plan.prep = stored
		return true, nil
	}
	if !created {
		if budget := readinessBudgetFrom(ctx); budget != nil && !budget.reserveNativeRPC(1) {
			return false, ErrReadinessPassPending
		}
		state, inspectErr := c.Host.InspectPreparation(ctx, project.PrepareInspect{Project: plan.projectID, ID: prep.ID})
		if inspectErr == nil {
			if !candidatePreparationIdentityMatches(state, prep) {
				return false, errors.New("retry preparation native identity differs")
			}
			if err := c.observeCandidatePreparation(ctx, stored, state); err != nil {
				return false, err
			}
			if state.Ready {
				plan.prep, err = c.Store.Preparation(ctx, prep.ID)
				return err == nil, err
			}
			if state.Phase == project.PrepareRunning {
				return false, nil
			}
			if state.Phase == project.PrepareFailed || state.Phase == project.PrepareStopped || state.Phase == project.PrepareInterrupted {
				return false, errors.New("retry preparation requires intervention")
			}
		} else if !errors.Is(inspectErr, project.ErrPreparationNotFound) {
			return false, inspectErr
		}
	}
	notAfter, err := c.Store.AuthorizeCandidatePreparation(ctx, prep.ID)
	if err != nil {
		if retryPreparationInvalidated(err) {
			plan.prep = stored
			if stopErr := c.retireRetryCoder(ctx, plan); stopErr != nil {
				return false, stopErr
			}
		}
		return false, err
	}
	if budget := readinessBudgetFrom(ctx); budget != nil && !budget.reserveNativeRPC(1) {
		return false, ErrReadinessPassPending
	}
	state, err := c.Host.PrepareCandidate(ctx, project.FactoryCandidate{
		Preparation: prep, SourcePreparation: source.ID, Deadline: time.Unix(notAfter, 0),
	})
	if err != nil {
		return false, err
	}
	if !candidatePreparationIdentityMatches(state, prep) {
		return false, errors.New("retry preparation native receipt differs")
	}
	if err := c.observeCandidatePreparation(ctx, stored, state); err != nil {
		return false, err
	}
	if !state.Ready {
		return false, nil
	}
	plan.prep, err = c.Store.Preparation(ctx, prep.ID)
	return err == nil, err
}

func (c *Coordinator) retireRetryCoder(ctx context.Context, plan *attemptPlan) error {
	if budget := readinessBudgetFrom(ctx); budget != nil && !budget.reserveNativeRPC(1) {
		return ErrReadinessPassPending
	}
	state, err := c.Host.StopPreparation(ctx, project.PrepareStop{Project: plan.projectID, ID: plan.prep.Preparation.ID})
	if err != nil {
		return err
	}
	if err := c.Store.RecordCandidatePreparationStop(ctx, plan.prep.Preparation.ID, state); err != nil {
		return err
	}
	if state.Retirement != "confirmed" {
		return store.ErrCandidatePreparationPending
	}
	return nil
}
