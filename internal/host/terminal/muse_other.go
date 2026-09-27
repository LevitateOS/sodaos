//go:build !linux

package terminal

import (
	"context"
	"errors"
	"net"
	"os"

	"github.com/levitateos/sodaos/internal/identity"
)

func (*MuseLaunch) Serve(context.Context, *net.UnixListener) error {
	return errors.New("Muse launch requires native Linux")
}

func (*MuseRuntime) Start(context.Context, MusePeer, identity.LaunchRequest, [3]*os.File) (MuseInvocation, error) {
	return MuseInvocation{}, identity.ErrDenied
}

func (*MuseRuntime) RegisterNested(context.Context, MusePeer, string, int64, string) error {
	return identity.ErrDenied
}

func (*MuseRuntime) ValidateMuseBinding(context.Context, identity.Binding) error {
	return identity.ErrDenied
}

func (*MuseRuntime) StartFactory(context.Context, MusePeer, identity.LaunchRequest, [3]*os.File) (MuseInvocation, error) {
	return MuseInvocation{}, identity.ErrDenied
}

func (*MuseRuntime) Muse(context.Context, string, identity.Delivery) (identity.Delivery, error) {
	return identity.Delivery{}, identity.ErrDenied
}
