package host

import (
	"context"
	"errors"
	"fmt"

	"github.com/levitateos/factory/pkg/factory/execution"
	"github.com/levitateos/factory/pkg/project"
)

var (
	_ execution.PreparationObserver = (*FactoryObserver)(nil)
	_ execution.PreparationLauncher = (*FactoryObserver)(nil)
)

func (c *Client) InspectWorkspacePreparation(ctx context.Context, a execution.PreparationAddress) (execution.PreparationState, error) {
	return NewFactoryObserver(c, nil).InspectWorkspacePreparation(ctx, a)
}

func (c *Client) RetireWorkspacePreparation(ctx context.Context, a execution.PreparationAddress) (execution.PreparationState, error) {
	return NewFactoryObserver(c, nil).RetireWorkspacePreparation(ctx, a)
}

func (c *Client) PrepareWorkspace(ctx context.Context, r execution.StandingPreparationRequest) (execution.PreparationState, error) {
	return NewFactoryObserver(c, nil).PrepareWorkspace(ctx, r)
}

func (c *Client) PrepareWorkspaceCandidate(ctx context.Context, r execution.CandidatePreparationRequest) (execution.PreparationState, error) {
	return NewFactoryObserver(c, nil).PrepareWorkspaceCandidate(ctx, r)
}

func appliancePreparationAddress(a execution.PreparationAddress) error {
	if a.Native != nil || a.Validate() != nil || (project.PrepareInspect{Project: a.Workspace, ID: a.ID}).Validate() != nil {
		return execution.ErrPreparation
	}
	return nil
}

func preparationError(err error) error {
	if errors.Is(err, project.ErrPreparationNotFound) {
		return fmt.Errorf("%w: %w", execution.ErrPreparationNotObserved, err)
	}
	return err
}

func preparationObservation(s project.PrepareState, a execution.PreparationAddress) (execution.PreparationState, error) {
	state := execution.PreparationState{ID: s.ID, Project: s.Project, Role: s.Role, Phase: s.Phase, Container: s.Container, SourceCommit: s.SourceCommit, SetupDigest: s.SetupDigest, Tools: append([]project.ResolvedTool(nil), s.Tools...), Missing: s.Missing, Output: s.Output, Ready: s.Ready, Stopped: s.Stopped, Retirement: s.Retirement}
	if s.SetupExit != nil {
		code := *s.SetupExit
		state.SetupExit = &code
	}
	if s.CheckExit != nil {
		code := *s.CheckExit
		state.CheckExit = &code
	}
	return state, state.ValidateFor(a)
}

func (o *FactoryObserver) InspectWorkspacePreparation(ctx context.Context, a execution.PreparationAddress) (execution.PreparationState, error) {
	if o == nil || appliancePreparationAddress(a) != nil {
		return execution.PreparationState{}, execution.ErrPreparation
	}
	reader, ok := o.host.(interface {
		InspectPreparation(context.Context, project.PrepareInspect) (project.PrepareState, error)
	})
	if !ok {
		return execution.PreparationState{}, errors.New("appliance preparation observer unavailable")
	}
	state, err := reader.InspectPreparation(ctx, project.PrepareInspect{Project: a.Workspace, ID: a.ID})
	if err != nil {
		return execution.PreparationState{}, preparationError(err)
	}
	return preparationObservation(state, a)
}

func (o *FactoryObserver) RetireWorkspacePreparation(ctx context.Context, a execution.PreparationAddress) (execution.PreparationState, error) {
	if o == nil || appliancePreparationAddress(a) != nil {
		return execution.PreparationState{}, execution.ErrPreparation
	}
	worker, ok := o.host.(interface {
		StopPreparation(context.Context, project.PrepareStop) (project.PrepareState, error)
	})
	if !ok {
		return execution.PreparationState{}, errors.New("appliance preparation retirement unavailable")
	}
	state, err := worker.StopPreparation(ctx, project.PrepareStop{Project: a.Workspace, ID: a.ID})
	if err != nil {
		return execution.PreparationState{}, preparationError(err)
	}
	if !state.Stopped {
		return execution.PreparationState{}, execution.ErrPreparation
	}
	return preparationObservation(state, a)
}

func preparationLaunched(s project.PrepareState, p project.Preparation) (execution.PreparationState, error) {
	state, err := preparationObservation(s, execution.PreparationAddress{Workspace: p.Project, ID: p.ID})
	if err != nil || s.Role != p.Role || s.SourceCommit != p.SourceCommit || s.SetupDigest != p.SetupDigest {
		return execution.PreparationState{}, execution.ErrPreparation
	}
	return state, nil
}

func (o *FactoryObserver) PrepareWorkspace(ctx context.Context, r execution.StandingPreparationRequest) (execution.PreparationState, error) {
	if o == nil || r.Native != nil || r.Validate() != nil {
		return execution.PreparationState{}, execution.ErrPreparation
	}
	worker, ok := o.host.(interface {
		Prepare(context.Context, project.Prepare) (project.PrepareState, error)
	})
	if !ok {
		return execution.PreparationState{}, errors.New("appliance standing preparation unavailable")
	}
	state, err := worker.Prepare(ctx, r.Input)
	if err != nil {
		return execution.PreparationState{}, preparationError(err)
	}
	return preparationLaunched(state, r.Input.Preparation)
}

func (o *FactoryObserver) PrepareWorkspaceCandidate(ctx context.Context, r execution.CandidatePreparationRequest) (execution.PreparationState, error) {
	if o == nil || r.Native != nil || r.Validate() != nil {
		return execution.PreparationState{}, execution.ErrPreparation
	}
	worker, ok := o.host.(interface {
		PrepareCandidate(context.Context, project.FactoryCandidate) (project.PrepareState, error)
	})
	if !ok {
		return execution.PreparationState{}, errors.New("appliance candidate preparation unavailable")
	}
	state, err := worker.PrepareCandidate(ctx, r.Input)
	if err != nil {
		return execution.PreparationState{}, preparationError(err)
	}
	return preparationLaunched(state, r.Input.Preparation)
}
