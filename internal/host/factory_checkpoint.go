package host

import (
	"context"

	"github.com/levitateos/factory/pkg/project"
)

func (c *Client) FactoryTaskCheckpoint(ctx context.Context, in project.FactoryTaskCheckpoint) (project.FactoryTaskCheckpointState, error) {
	return c.factoryCheckpoint(ctx, in, "/factory-task-checkpoint")
}

func (c *Client) FactoryFinalCandidate(ctx context.Context, in project.FactoryTaskCheckpoint) (project.FactoryTaskCheckpointState, error) {
	return c.factoryCheckpoint(ctx, in, "/factory-final-candidate")
}

func (c *Client) FactoryFinalCandidateReceipt(ctx context.Context, in project.FactoryTaskCheckpoint) (project.FactoryTaskCheckpointState, error) {
	return c.factoryCheckpoint(ctx, in, "/factory-final-candidate-receipt")
}

func (c *Client) factoryCheckpoint(ctx context.Context, in project.FactoryTaskCheckpoint, path string) (project.FactoryTaskCheckpointState, error) {
	var out project.FactoryTaskCheckpointState
	err := c.callLimit(ctx, path, in, &out, 64<<10)
	return out, factoryNotFound(err)
}
