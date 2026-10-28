package host

import (
	"context"

	"github.com/levitateos/sodaos/internal/identity"
)

func (c *Client) IdentityLaunch(ctx context.Context, in identity.TerminalStart) (identity.Lease, error) {
	var out identity.Lease
	err := c.call(ctx, "/identity/launch", in, &out)
	return out, err
}
