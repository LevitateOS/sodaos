package host

import (
	"context"
	"errors"
	"fmt"

	"github.com/levitateos/factory/pkg/factory/execution"
	"github.com/levitateos/factory/pkg/project"
)

var _ execution.PreparationContextReader = (*FactoryObserver)(nil)

func (c *Client) ReadWorkspaceContext(ctx context.Context, request execution.PreparationContextRequest) (execution.PreparationContext, error) {
	return NewFactoryObserver(c, nil).ReadWorkspaceContext(ctx, request)
}

func (o *FactoryObserver) ReadWorkspaceContext(ctx context.Context, request execution.PreparationContextRequest) (execution.PreparationContext, error) {
	if o == nil || request.Validate() != nil || appliancePreparationAddress(request.Address()) != nil {
		return execution.PreparationContext{}, execution.ErrPreparationContext
	}
	bounded, cancel := context.WithDeadline(ctx, request.Input.NotAfter)
	defer cancel()
	if err := bounded.Err(); err != nil {
		return execution.PreparationContext{}, err
	}
	reader, ok := o.host.(interface {
		ReadPreparationContext(context.Context, project.FactoryPreparationContextRequest, string) (project.FactoryPreparationContext, error)
	})
	if !ok {
		return execution.PreparationContext{}, errors.New("appliance preparation context reader unavailable")
	}
	material, err := reader.ReadPreparationContext(bounded, request.Input, request.Role)
	if err != nil {
		if preparationContextBoundsExceeded(err) {
			return execution.PreparationContext{}, fmt.Errorf("%w: %w", execution.ErrPreparationContextBounds, err)
		}
		return execution.PreparationContext{}, err
	}
	if err := bounded.Err(); err != nil {
		return execution.PreparationContext{}, err
	}
	observed := execution.PreparationContext{Material: material}
	if err := observed.ValidateFor(request); err != nil {
		return execution.PreparationContext{}, err
	}
	return observed, nil
}
