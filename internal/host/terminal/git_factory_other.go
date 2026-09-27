//go:build !linux

package terminal

import (
	"context"
	"net"
	"os"

	"github.com/levitateos/sodaos/internal/identity"
)

func (f *FactoryGitRuntime) StartFactory(context.Context, MusePeer, identity.GitLaunchRequest, *os.File) (GitInvocation, error) {
	return GitInvocation{}, identity.ErrDenied
}

func (f *FactoryGitRuntime) ServeFactory(context.Context, *net.UnixListener) error {
	return identity.ErrDenied
}

func (f *FactoryGitRuntime) Validate(context.Context, identity.Lease, int) error {
	return identity.ErrDenied
}

func (f *FactoryGitRuntime) Stop(context.Context, identity.Binding) error {
	return identity.ErrDenied
}
