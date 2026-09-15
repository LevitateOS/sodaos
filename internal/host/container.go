package host

import (
	"context"
)

func (d *Daemon) terminalContainer(ctx context.Context, id string) (string, error) {
	return d.Project.TerminalContainer(ctx, id)
}

func (d *Daemon) projectContainer(ctx context.Context, id string, requireRunning bool) (string, error) {
	return d.Project.ProjectContainer(ctx, id, requireRunning)
}
