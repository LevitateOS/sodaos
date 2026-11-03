package host

import (
	"context"
	"errors"

	"github.com/levitateos/factory/pkg/collaboration"
	"github.com/levitateos/factory/pkg/factory/execution"
	"github.com/levitateos/factory/pkg/project"
)

func (c *Client) FactoryNativeTask(ctx context.Context, in project.FactoryNativeTask) (project.FactoryNativeTaskState, error) {
	var out project.FactoryNativeTaskState
	if err := in.Validate(); err != nil {
		return out, err
	}
	err := c.callLimit(ctx, "/factory-native-task", in, &out, 64<<10)
	if err == nil && (out.Result == nil || out.Result["task_id"] != in.Arguments.TaskID) {
		err = errors.New("native task result does not match the requested task")
	}
	return out, factoryNotFound(err)
}

// CommandTask preserves the existing appliance task-owner protocol. A native
// process requires its own commander; it cannot fall through to this client.
func (c *Client) CommandTask(ctx context.Context, request execution.TaskRequest) (execution.TaskReceipt, error) {
	var empty execution.TaskReceipt
	if request.Validate() != nil || request.Kind != collaboration.Appliance {
		return empty, execution.ErrTask
	}
	result, err := c.FactoryNativeTask(ctx, request.Input)
	if err != nil {
		return empty, err
	}
	receipt := execution.TaskReceipt{Request: request, Result: result.Result}
	if err := receipt.ValidateFor(request); err != nil {
		return empty, err
	}
	return receipt, nil
}
