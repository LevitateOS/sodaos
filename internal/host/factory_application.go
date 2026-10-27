package host

import (
	"context"

	"github.com/levitateos/sodaos/internal/application"
)

// FactoryApplicationBuild packages an already verified bundle through the
// native owner. No caller can supply a build command, image, path or digest.
func (c *Client) FactoryApplicationBuild(ctx context.Context, in application.BuildRequest) (application.BuildArtifact, error) {
	var out application.BuildArtifact
	if err := in.Validate(); err != nil {
		return out, err
	}
	err := c.callLimit(ctx, "/factory-application-build", in, &out, 16<<10)
	if err == nil {
		err = out.ValidateFor(in)
	}
	return out, factoryNotFound(err)
}
