package host

import (
	"context"

	"github.com/levitateos/sodaos/internal/project"
)

// FactoryVerificationChecks drives or observes only the already admitted
// manifest. Its identity never accepts an executable, fixture path or script.
func (c *Client) FactoryVerificationChecks(ctx context.Context, in project.FactoryVerificationChecks) (project.FactoryVerificationChecksState, error) {
	var out project.FactoryVerificationChecksState
	if err := in.Validate(); err != nil {
		return out, err
	}
	err := c.callLimit(ctx, "/factory-verification-checks", in, &out, 64<<10)
	if err == nil {
		err = out.ValidateFor(in)
	}
	return out, factoryNotFound(err)
}
