package host

import (
	"context"
	"errors"

	"github.com/levitateos/sodaos/internal/project"
)

const factoryContextResponseLimit = 96 << 10

// ReadPreparationContext reads bounded repository instructions, selected
// files and the exact candidate diff from the prepared native checkout.
func (c *Client) ReadPreparationContext(ctx context.Context, in project.FactoryPreparationContextRequest, role string) (project.FactoryPreparationContext, error) {
	var out project.FactoryPreparationContext
	if err := in.Validate(); err != nil || !project.ValidFactoryRole(role) {
		return out, errors.New("invalid factory preparation context selection")
	}
	ctx, cancel := context.WithDeadline(ctx, in.NotAfter)
	defer cancel()
	if err := ctx.Err(); err != nil {
		return out, err
	}
	in.NotAfter, _ = ctx.Deadline()
	err := c.callLimit(ctx, "/prepare-context", in, &out, factoryContextResponseLimit)
	if ctxErr := ctx.Err(); ctxErr != nil {
		return project.FactoryPreparationContext{}, ctxErr
	}
	if err == nil {
		err = out.ValidateFor(in, role)
	}
	if err != nil {
		return project.FactoryPreparationContext{}, err
	}
	return out, nil
}
