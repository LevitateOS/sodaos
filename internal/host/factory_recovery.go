package host

import (
	"context"
	"errors"

	"github.com/levitateos/factory/pkg/factory/execution"
	"github.com/levitateos/factory/pkg/project"
)

var _ execution.WorkRecoverer = (*FactoryObserver)(nil)

func (c *Client) RecoverWork(ctx context.Context, request execution.WorkRecoveryRequest) (execution.WorkRecoveryReceipt, error) {
	return NewFactoryObserver(c, nil).RecoverWork(ctx, request)
}

func (o *FactoryObserver) RecoverWork(ctx context.Context, request execution.WorkRecoveryRequest) (execution.WorkRecoveryReceipt, error) {
	if o == nil || request.Native != nil || request.Validate() != nil {
		return execution.WorkRecoveryReceipt{}, execution.ErrRecovery
	}
	worker, ok := o.host.(interface {
		FactoryTakeover(context.Context, project.FactoryTakeover) (project.TakeoverResult, error)
	})
	if !ok {
		return execution.WorkRecoveryReceipt{}, errors.New("appliance work recovery unavailable")
	}
	wire := project.FactoryTakeover{Project: request.Address.Workspace, ID: request.Address.Run, Member: request.Member}
	result, err := worker.FactoryTakeover(ctx, wire)
	if err != nil {
		return execution.WorkRecoveryReceipt{}, runObservationError(err)
	}
	if result.Validate() != nil || result.Project != wire.Project || result.ID != wire.ID || result.Member != wire.Member {
		return execution.WorkRecoveryReceipt{}, execution.ErrRecovery
	}
	receipt := execution.WorkRecoveryReceipt{Request: request, Destination: result.Destination, Reused: result.Reused}
	return receipt, receipt.ValidateFor(request)
}
