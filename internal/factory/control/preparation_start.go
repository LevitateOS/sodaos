package control

import (
	"context"
	"errors"
	"time"

	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
)

// ErrPreparationIntervention preserves a failed or retired identity. The
// administrator must inspect its effects and choose a new preparation identity.
var ErrPreparationIntervention = errors.New("preparation requires administrator intervention and a new identity")

type standingPreparationHost interface {
	Prepare(context.Context, project.Prepare) (project.PrepareState, error)
}

// StartPreparation admits the exact human-selected standing setup before any
// native effect. authorize checks the current caller and both recorded approvers;
// it grants no provider lease or privileged installation capability.
func (c *Coordinator) StartPreparation(ctx context.Context, in project.Prepare, authorize func(context.Context) error) (project.StoredPreparation, error) {
	if err := in.Validate(); err != nil {
		return project.StoredPreparation{}, err
	}
	setupFiles := make([]string, 0, len(in.Setup.Files))
	for name := range in.Setup.Files {
		setupFiles = append(setupFiles, name)
	}
	if project.PreparationInputsDigestOf(setupFiles, in.Preparation.Requirements.SourceInputs, in.Preparation.Tools) != in.Preparation.Requirements.Digest {
		return project.StoredPreparation{}, store.ErrCandidatePreparationConflict
	}
	if authorize == nil || c.Store == nil || c.Host == nil {
		return project.StoredPreparation{}, errors.New("standing preparation authority is unavailable")
	}
	host, ok := c.Host.(standingPreparationHost)
	if !ok {
		return project.StoredPreparation{}, errors.New("standing preparation host is unavailable")
	}
	deadline := time.Now().Add(30 * time.Second)
	bounded, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()
	if err := authorize(bounded); err != nil {
		return project.StoredPreparation{}, err
	}
	stored, _, err := c.Store.AdmitStandingPreparation(bounded, in.Preparation)
	if err != nil {
		return project.StoredPreparation{}, err
	}
	if preparationNeedsIntervention(stored.State) {
		return stored, ErrPreparationIntervention
	}
	inspectCtx, stopInspect := context.WithTimeout(bounded, 5*time.Second)
	state, inspectErr := c.Host.InspectPreparation(inspectCtx, project.PrepareInspect{Project: in.Preparation.Project, ID: in.Preparation.ID})
	stopInspect()
	if inspectErr == nil {
		stored, err = c.adoptStandingPreparation(bounded, stored, state)
		if err != nil || state.Phase == project.PrepareRunning || state.Phase == project.PrepareReady {
			return stored, err
		}
	} else if !errors.Is(inspectErr, project.ErrPreparationNotFound) {
		return stored, inspectErr
	}
	// Missing can start; an approved or waiting identity can resume after its
	// prerequisite is repaired. Recheck authority before the identical request.
	if err := authorize(bounded); err != nil {
		return stored, err
	}
	if err := c.Store.AuthorizeStandingPreparation(bounded, in.Preparation); err != nil {
		return stored, err
	}
	callCtx, stopCall := context.WithDeadline(bounded, deadline.Add(-5*time.Second))
	state, err = host.Prepare(callCtx, in)
	stopCall()
	if err == nil {
		return c.adoptStandingPreparation(bounded, stored, state)
	}
	// The native worker outlives an HTTP observation. A lost or canceled reply
	// neither removes admission nor authorizes another worker identity.
	recoveryCtx, stopRecovery := context.WithDeadline(context.WithoutCancel(ctx), deadline)
	defer stopRecovery()
	recoveryCtx, stopRecoveryInspect := context.WithTimeout(recoveryCtx, 5*time.Second)
	defer stopRecoveryInspect()
	state, inspectErr = c.Host.InspectPreparation(recoveryCtx, project.PrepareInspect{Project: in.Preparation.Project, ID: in.Preparation.ID})
	if inspectErr != nil {
		return stored, errors.Join(err, inspectErr)
	}
	return c.adoptStandingPreparation(recoveryCtx, stored, state)
}

func (c *Coordinator) adoptStandingPreparation(ctx context.Context, stored project.StoredPreparation, state project.PrepareState) (project.StoredPreparation, error) {
	if !candidatePreparationIdentityMatches(state, stored.Preparation) || !project.ValidPreparePhase(state.Phase) || !project.ValidContainerID(state.Container) ||
		state.Ready != (state.Phase == project.PrepareReady) || state.Ready && state.Stopped {
		return stored, errors.New("native standing preparation identity was not confirmed")
	}
	observed := stored
	observed.State = state
	// Keep the revision read before the native observation. Reloading it here
	// would let a late Ready reply overwrite a newer terminal observation.
	if err := c.Store.ObservePreparation(ctx, observed); err != nil {
		current, readErr := c.Store.Preparation(ctx, stored.Preparation.ID)
		if readErr != nil {
			return project.StoredPreparation{}, errors.Join(err, readErr)
		}
		return current, err
	}
	current, err := c.Store.Preparation(ctx, stored.Preparation.ID)
	if err != nil {
		return stored, err
	}
	if preparationNeedsIntervention(current.State) {
		return current, ErrPreparationIntervention
	}
	return current, nil
}

func preparationNeedsIntervention(state project.PrepareState) bool {
	return state.Stopped || state.Phase == project.PrepareFailed || state.Phase == project.PrepareStopped || state.Phase == project.PrepareInterrupted
}
