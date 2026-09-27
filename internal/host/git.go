package host

import (
	"context"
	"errors"
	"net"
	"os"
	"path/filepath"

	"github.com/levitateos/sodaos/internal/host/terminal"
	"github.com/levitateos/sodaos/internal/identity"
)

func validateGitRuntime(c Config) error {
	if c.GitSocket == "" {
		return nil
	}
	if !filepath.IsAbs(c.GitSocket) || filepath.Base(c.GitSocket) != "launch.sock" || !filepath.IsAbs(c.IdentitySocket) {
		return errors.New("explicit git launch and private broker sockets required")
	}
	if filepath.Dir(c.GitSocket) == filepath.Dir(c.IdentitySocket) || filepath.Dir(c.GitSocket) == filepath.Dir(c.MuseSocket) {
		return errors.New("git launch interface must be separate from other sockets")
	}
	return nil
}

func (d *Daemon) selectGit(ctx context.Context, actor int64, project, selected string) (string, error) {
	available, err := d.Identity.Available(ctx, actor, project)
	if err != nil {
		return "", err
	}
	return identity.SelectForgejoConnection(available, selected)
}

func (d *Daemon) gitRuntime() *terminal.GitRuntime {
	return &terminal.GitRuntime{
		Exec: d.Terminal.Exec, Acquire: d.Identity.AcquireGit,
		Register: d.Identity.RegisterGit, End: d.Identity.EndLease,
		Proxy: d.Identity.GitProxy, Select: d.selectGit,
		Authorize: func(ctx context.Context, actor int64, project string) error {
			_, err := d.selectGit(ctx, actor, project, "")
			return err
		},
	}
}

// OpenGitListener supplies launch only; the private broker transport remains
// outside project and nested container mount namespaces.
func (d *Daemon) OpenGitListener() (*net.UnixListener, error) {
	if d.Git == nil {
		return nil, nil
	}
	path := d.Config.GitSocket
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil || len(entries) != 0 {
		return nil, errors.New("git interface directory must be empty at service startup")
	}
	listener, err := net.ListenUnix("unixpacket", &net.UnixAddr{Name: path, Net: "unixpacket"})
	if err != nil {
		return nil, err
	}
	if err := os.Chmod(path, 0o666); err != nil {
		_ = listener.Close()
		return nil, err
	}
	return listener, nil
}

func (d *Daemon) ServeGit(ctx context.Context, listener *net.UnixListener) error {
	return d.Git.Serve(ctx, listener)
}

func (d *Daemon) gitOperation(ctx context.Context, action string, l identity.Lease) error {
	if d.Git == nil || l.Binding == nil || l.ProviderID != identity.Forgejo || l.Kind != identity.Terminal || l.Binding.Project != l.ProjectID {
		return identity.ErrDenied
	}
	switch action {
	case "validate":
		return d.Git.Validate(ctx, *l.Binding)
	case "stop":
		return d.Git.Stop(ctx, *l.Binding)
	default:
		return identity.ErrDenied
	}
}
