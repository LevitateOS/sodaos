package host

import (
	"context"

	"github.com/levitateos/sodaos/internal/tailnet"
)

func (d *Daemon) StartTailnet(ctx context.Context, id string) (string, error) {
	return d.Companion.StartTailnet(ctx, id)
}

func (d *Daemon) StopTailnet(ctx context.Context, id string) error {
	return d.Companion.StopTailnet(ctx, id)
}

func (d *Daemon) WaitTailnet(ctx context.Context, cid string) error {
	return d.Companion.WaitTailnet(ctx, cid)
}

func (d *Daemon) observeProjectTailnet(ctx context.Context, in tailnet.ProjectRequest, cid string) (tailnet.ProjectView, error) {
	return d.Companion.ObserveProjectTailnet(ctx, in, cid)
}
