package host

import (
	"context"

	"github.com/levitateos/sodaos/internal/host/project"
	"github.com/levitateos/sodaos/internal/project"
)

// NewProject builds the privileged project runtime from host executor and config.
func NewProject(exec Executor, c Config) *hostproject.Runtime {
	return &hostproject.Runtime{
		Exec: exec,
		Config: hostproject.Config{
			Image:   c.Image,
			Network: c.Network,
			Subnet:  c.Subnet,
			Bridge:  c.Bridge,
		},
	}
}

// The daemon decodes the Unix-socket wire directly into project domain types
// and passes them to privileged execution unchanged. There is exactly one
// type per concept; no translation layer remains here.

func (d *Daemon) create(ctx context.Context, in project.Create) (project.Environment, error) {
	if err := in.Validate(); err != nil {
		return project.Environment{}, err
	}
	if err := d.acquireAdmission(ctx); err != nil {
		return project.Environment{}, err
	}
	defer func() { <-d.admission }()
	return d.Project.Create(ctx, in)
}

func (d *Daemon) inspect(ctx context.Context, id string) (project.Environment, int64, error) {
	return d.Project.Inspect(ctx, id)
}

func (d *Daemon) account(ctx context.Context, in project.Account) error {
	return d.Project.Account(ctx, in)
}

func (d *Daemon) connection(ctx context.Context, id string) (project.Connection, error) {
	return d.Project.Connection(ctx, id)
}

func (d *Daemon) lifecycle(ctx context.Context, in project.Lifecycle) (project.LifecycleState, error) {
	return d.Project.Lifecycle(ctx, in)
}

func (d *Daemon) accessKeys(ctx context.Context, in project.AccessKeys) (project.AccessKeyState, error) {
	return d.Project.AccessKeys(ctx, in)
}

func (d *Daemon) resolveProfile(ctx context.Context) (project.Profile, error) {
	return d.Project.ResolveProfile(ctx)
}

func (d *Daemon) observeOS(ctx context.Context, id string) (project.OSObservation, error) {
	return d.Project.ObserveOS(ctx, id)
}
