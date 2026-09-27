//go:build !linux

package terminal

import (
	"context"
	"net"
	"os"

	"github.com/levitateos/sodaos/internal/identity"
)

func (g *GitRuntime) Start(context.Context, MusePeer, identity.GitLaunchRequest, *os.File) (GitInvocation, error) {
	return GitInvocation{}, identity.ErrDenied
}
func (g *GitRuntime) Serve(context.Context, *net.UnixListener) error   { return identity.ErrDenied }
func (g *GitRuntime) Validate(context.Context, identity.Binding) error { return identity.ErrDenied }
func (g *GitRuntime) Stop(context.Context, identity.Binding) error     { return identity.ErrDenied }
