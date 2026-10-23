package host

import (
	"context"

	"github.com/levitateos/sodaos/internal/project"
)

func (c *Client) FactoryTaskCheckpoint(ctx context.Context, in project.FactoryTaskCheckpoint) (project.FactoryTaskCheckpointState, error) {
	return c.factoryCheckpoint(ctx, in, false, false)
}

func (c *Client) FactoryFinalCandidate(ctx context.Context, in project.FactoryTaskCheckpoint) (project.FactoryTaskCheckpointState, error) {
	return c.factoryCheckpoint(ctx, in, true, false)
}

func (c *Client) FactoryTaskCheckpointReceipt(ctx context.Context, in project.FactoryTaskCheckpoint) (project.FactoryTaskCheckpointState, error) {
	return c.factoryCheckpoint(ctx, in, false, true)
}

func (c *Client) FactoryFinalCandidateReceipt(ctx context.Context, in project.FactoryTaskCheckpoint) (project.FactoryTaskCheckpointState, error) {
	return c.factoryCheckpoint(ctx, in, true, true)
}

func (c *Client) factoryCheckpoint(ctx context.Context, in project.FactoryTaskCheckpoint, final, receipt bool) (project.FactoryTaskCheckpointState, error) {
	var out project.FactoryTaskCheckpointState
	if err := in.Validate(final); err != nil {
		return out, err
	}
	path := "/factory-task-checkpoint"
	if final {
		path = "/factory-final-candidate"
	}
	if receipt {
		path += "-receipt"
	}
	err := c.callLimit(ctx, path, in, &out, 64<<10)
	if err == nil {
		err = out.ValidateFor(in, final)
	}
	return out, factoryNotFound(err)
}
