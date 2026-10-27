package host

import (
	"context"
	"errors"

	"github.com/levitateos/sodaos/internal/project"
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
