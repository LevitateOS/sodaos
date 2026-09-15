package host

import (
	"context"
	"errors"
	"runtime"

	"github.com/levitateos/sodaos/internal/project"
)

// ResolveProfile inspects only the configured installed image. It never pulls,
// runs a container, changes a tag or accepts an image/architecture from the caller.
func (c *Client) ResolveProfile(ctx context.Context) (project.Profile, error) {
	var p project.Profile
	err := c.call(ctx, "/profile", struct{}{}, &p)
	if err == nil {
		err = p.Validate()
		if p.Architecture != runtime.GOARCH {
			err = errors.New("project image is not native to this backend")
		}
	}
	return p, err
}
