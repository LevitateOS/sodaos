package terminal

import (
	"context"
	"net/http"
	"os/exec"

	"github.com/levitateos/sodaos/internal/identity"
)

// GitRuntime executes a native remote helper in an independent project unit.
// Its broker callbacks use private host transport; no broker socket enters OCI.
type GitRuntime struct {
	Exec      Executor
	Authorize func(context.Context, int64, string) error
	Select    func(context.Context, int64, string, string) (string, error)
	Acquire   func(context.Context, identity.GitAcquireRequest) (identity.Lease, error)
	Register  func(context.Context, string, identity.Binding) (identity.GitSession, error)
	End       func(context.Context, int64, string) error
	Proxy     func(string) http.Handler
}

// GitInvocation has the same command lifetime as the existing native launcher,
// but owns a separate relay listener and Git-specific execution boundary.
type GitInvocation struct {
	Command *exec.Cmd
	Prepare func(context.Context) error
	Finish  func(context.Context) error
	Control func(context.Context, identity.LaunchControl) error
}
