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

func (d *Daemon) museRuntime() *terminal.MuseRuntime {
	return &terminal.MuseRuntime{
		Exec: d.Terminal.Exec, BinarySHA256: d.Config.MuseSHA256, BinaryVersion: d.Config.MuseVersion,
		Acquire: d.Identity.Acquire, Attach: d.Identity.Register, End: d.Identity.EndLease,
		Authorize: func(ctx context.Context, actor int64, project string) error {
			available, err := d.Identity.Available(ctx, actor, project)
			if err != nil {
				return err
			}
			for _, connection := range available {
				if connection.ProviderID == identity.Muse && connection.State == identity.Ready {
					return nil
				}
			}
			return identity.ErrDenied
		},
		Select: func(ctx context.Context, actor int64, project, selected string) (string, error) {
			available, err := d.Identity.Available(ctx, actor, project)
			if err != nil {
				return "", err
			}
			return identity.SelectMuseConnection(available, selected)
		},
	}
}

// OpenMuseListener exposes only kernel-attested launch and trusted registration.
// Broker administration and credential-delivery sockets stay private.
func (d *Daemon) OpenMuseListener() (*net.UnixListener, error) {
	if d.Muse == nil {
		return nil, nil
	}
	path := d.Config.MuseSocket
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil || len(entries) != 0 {
		return nil, errors.New("muse interface directory must be empty before launch service startup")
	}
	if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
		return nil, errors.New("muse launch socket is occupied")
	}
	listener, err := net.ListenUnix("unixpacket", &net.UnixAddr{Name: path, Net: "unixpacket"})
	if err != nil {
		return nil, err
	}
	if err = os.Chmod(path, 0o666); err != nil {
		_ = listener.Close()
		return nil, err
	}
	return listener, nil
}

func (d *Daemon) ServeMuse(ctx context.Context, listener *net.UnixListener) error {
	service := terminal.MuseLaunch{Start: d.Muse.Start, Register: func(ctx context.Context, peer terminal.MusePeer, in identity.NestedRegistration) error {
		return d.Muse.RegisterNested(ctx, peer, in.ChildID, in.ActorID, in.RegistrationID)
	}}
	return service.Serve(ctx, listener)
}
