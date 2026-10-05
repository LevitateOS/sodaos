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

// IdentityValidate attests the managed terminal before broker credential delivery.
func (c *Client) IdentityValidate(ctx context.Context, l identity.Lease) error {
	return c.call(ctx, "/identity/validate", identity.DeliveryWire{Lease: l}, nil)
}

func (c *Client) IdentityStop(ctx context.Context, l identity.Lease) error {
	return c.call(ctx, "/identity/stop", identity.DeliveryWire{Lease: l}, nil)
}

func (c *Client) IdentityStart(ctx context.Context, d identity.Delivery) error {
	return c.call(ctx, "/identity/start", identity.DeliveryWire(d), nil)
}

// IdentityFinish returns a private credential stream after freezing and ending
// the entire managed terminal cgroup. It is never a browser response.
func (c *Client) IdentityFinish(ctx context.Context, l identity.Lease) (identity.Delivery, error) {
	var out identity.DeliveryWire
	err := c.callLimit(ctx, "/identity/finish", identity.DeliveryWire{Lease: l}, &out, 384<<10)
	return identity.Delivery(out), err
}
