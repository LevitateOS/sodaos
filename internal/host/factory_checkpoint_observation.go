package host

import (
	"context"
	"errors"

	"github.com/levitateos/factory/pkg/factory/execution"
	"github.com/levitateos/factory/pkg/project"
)

var (
	_ execution.CheckpointWriter      = (*FactoryObserver)(nil)
	_ execution.FinalCheckpointReader = (*FactoryObserver)(nil)
)

func (c *Client) CheckpointTask(ctx context.Context, r execution.CheckpointRequest) (execution.CheckpointReceipt, error) {
	return NewFactoryObserver(c, nil).CheckpointTask(ctx, r)
}

func (c *Client) CheckpointFinal(ctx context.Context, r execution.CheckpointRequest) (execution.CheckpointReceipt, error) {
	return NewFactoryObserver(c, nil).CheckpointFinal(ctx, r)
}

func (c *Client) ReadFinalCheckpoint(ctx context.Context, r execution.CheckpointRequest) (execution.CheckpointReceipt, error) {
	return NewFactoryObserver(c, nil).ReadFinalCheckpoint(ctx, r)
}

func applianceCheckpoint(r execution.CheckpointRequest, final bool) (project.FactoryTaskCheckpoint, error) {
	if r.Native != nil || r.Validate(final) != nil {
		return project.FactoryTaskCheckpoint{}, execution.ErrCheckpoint
	}
	return project.FactoryTaskCheckpoint{Project: r.Project, ID: r.ID, Container: r.Container, Invocation: r.Invocation, ManifestDigest: r.ManifestDigest, TaskKey: r.TaskKey, OutputDigest: r.OutputDigest, ExpectedHead: r.ExpectedHead}, nil
}

func checkpointObservation(result project.FactoryTaskCheckpointState, request project.FactoryTaskCheckpoint, final bool) (execution.CheckpointReceipt, error) {
	if err := result.ValidateFor(request, final); err != nil {
		return execution.CheckpointReceipt{}, err
	}
	r := result.Request
	observed := execution.CheckpointReceipt{Request: execution.CheckpointRequest{Project: r.Project, ID: r.ID, Container: r.Container, Invocation: r.Invocation, ManifestDigest: r.ManifestDigest, TaskKey: r.TaskKey, OutputDigest: r.OutputDigest, ExpectedHead: r.ExpectedHead}, FinalCandidate: result.FinalCandidate, Status: result.Status, Candidate: result.Candidate, Tree: result.Tree, ReceiptDigest: result.ReceiptDigest, Checks: result.Checks, Reason: result.Reason}
	return observed, observed.ValidateFor(observed.Request, final)
}

func (o *FactoryObserver) CheckpointTask(ctx context.Context, r execution.CheckpointRequest) (execution.CheckpointReceipt, error) {
	if o == nil {
		return execution.CheckpointReceipt{}, execution.ErrCheckpoint
	}
	request, err := applianceCheckpoint(r, false)
	if err != nil {
		return execution.CheckpointReceipt{}, err
	}
	writer, ok := o.host.(interface {
		FactoryTaskCheckpoint(context.Context, project.FactoryTaskCheckpoint) (project.FactoryTaskCheckpointState, error)
	})
	if !ok {
		return execution.CheckpointReceipt{}, errors.New("appliance task checkpoint unavailable")
	}
	result, err := writer.FactoryTaskCheckpoint(ctx, request)
	if err != nil {
		return execution.CheckpointReceipt{}, runObservationError(err)
	}
	return checkpointObservation(result, request, false)
}

func (o *FactoryObserver) CheckpointFinal(ctx context.Context, r execution.CheckpointRequest) (execution.CheckpointReceipt, error) {
	if o == nil {
		return execution.CheckpointReceipt{}, execution.ErrCheckpoint
	}
	request, err := applianceCheckpoint(r, true)
	if err != nil {
		return execution.CheckpointReceipt{}, err
	}
	writer, ok := o.host.(interface {
		FactoryFinalCandidate(context.Context, project.FactoryTaskCheckpoint) (project.FactoryTaskCheckpointState, error)
	})
	if !ok {
		return execution.CheckpointReceipt{}, errors.New("appliance final checkpoint unavailable")
	}
	result, err := writer.FactoryFinalCandidate(ctx, request)
	if err != nil {
		return execution.CheckpointReceipt{}, runObservationError(err)
	}
	return checkpointObservation(result, request, true)
}

func (o *FactoryObserver) ReadFinalCheckpoint(ctx context.Context, r execution.CheckpointRequest) (execution.CheckpointReceipt, error) {
	if o == nil {
		return execution.CheckpointReceipt{}, execution.ErrCheckpoint
	}
	request, err := applianceCheckpoint(r, true)
	if err != nil {
		return execution.CheckpointReceipt{}, err
	}
	reader, ok := o.host.(interface {
		FactoryFinalCandidateReceipt(context.Context, project.FactoryTaskCheckpoint) (project.FactoryTaskCheckpointState, error)
	})
	if !ok {
		return execution.CheckpointReceipt{}, errors.New("appliance final receipt unavailable")
	}
	result, err := reader.FactoryFinalCandidateReceipt(ctx, request)
	if err != nil {
		return execution.CheckpointReceipt{}, runObservationError(err)
	}
	return checkpointObservation(result, request, true)
}
