package host

import (
	"context"

	"github.com/levitateos/sodaos/internal/hostproject"
	"github.com/levitateos/sodaos/internal/projectos"
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

func toProjectCreate(in Create) hostproject.Create {
	return hostproject.Create{Profile: in.Profile, ID: in.ID, Owner: in.Owner}
}

func toProjectAccount(in Account) hostproject.Account {
	return hostproject.Account{Project: in.Project, Login: in.Login, Identity: in.Identity, Keys: in.Keys}
}

func toProjectLifecycle(in Lifecycle) hostproject.Lifecycle {
	return hostproject.Lifecycle{Project: in.Project, Action: in.Action}
}

func toProjectAccessKeys(in AccessKeys) hostproject.AccessKeys {
	return hostproject.AccessKeys{Project: in.Project, Login: in.Login, Identity: in.Identity, Revision: in.Revision, Keys: in.Keys, Apply: in.Apply}
}

func fromProjectEnvironment(e hostproject.Environment) Environment {
	return Environment{Image: e.Image, Profile: e.Profile, ID: e.ID, IP: e.IP, Running: e.Running}
}

func fromProjectConnection(c hostproject.Connection) Connection {
	return Connection{Environment: fromProjectEnvironment(c.Environment), HostKey: c.HostKey, Fingerprint: c.Fingerprint}
}

func fromProjectLifecycleState(s hostproject.LifecycleState) LifecycleState {
	return LifecycleState{Environment: fromProjectEnvironment(s.Environment), BootEnabled: s.BootEnabled}
}

func fromProjectAccessKeyState(s hostproject.AccessKeyState) AccessKeyState {
	return AccessKeyState{Revision: s.Revision, Keys: s.Keys}
}

func fromProjectOSObservation(o hostproject.OSObservation) OSObservation {
	out := OSObservation{Environment: fromProjectEnvironment(o.Environment), Unavailable: o.Unavailable}
	if o.Release != nil {
		out.Release = &OSRelease{ID: o.Release.ID, Version: o.Release.Version, Name: o.Release.Name}
	}
	return out
}

func (d *Daemon) create(ctx context.Context, in Create) (Environment, error) {
	req := toProjectCreate(in)
	if err := req.Validate(); err != nil {
		return Environment{}, err
	}
	if err := d.acquireAdmission(ctx); err != nil {
		return Environment{}, err
	}
	defer func() { <-d.admission }()
	out, err := d.Project.Create(ctx, req)
	return fromProjectEnvironment(out), err
}

func (d *Daemon) inspect(ctx context.Context, id string) (Environment, int64, error) {
	env, owner, err := d.Project.Inspect(ctx, id)
	return fromProjectEnvironment(env), owner, err
}

func (d *Daemon) account(ctx context.Context, in Account) error {
	return d.Project.Account(ctx, toProjectAccount(in))
}

func (d *Daemon) connection(ctx context.Context, id string) (Connection, error) {
	out, err := d.Project.Connection(ctx, id)
	return fromProjectConnection(out), err
}

func (d *Daemon) lifecycle(ctx context.Context, in Lifecycle) (LifecycleState, error) {
	out, err := d.Project.Lifecycle(ctx, toProjectLifecycle(in))
	return fromProjectLifecycleState(out), err
}

func (d *Daemon) accessKeys(ctx context.Context, in AccessKeys) (AccessKeyState, error) {
	out, err := d.Project.AccessKeys(ctx, toProjectAccessKeys(in))
	return fromProjectAccessKeyState(out), err
}

func (d *Daemon) resolveProfile(ctx context.Context) (projectos.Profile, error) {
	return d.Project.ResolveProfile(ctx)
}

func (d *Daemon) observeOS(ctx context.Context, id string) (OSObservation, error) {
	out, err := d.Project.ObserveOS(ctx, id)
	return fromProjectOSObservation(out), err
}
