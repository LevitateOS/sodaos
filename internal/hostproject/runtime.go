// Package hostproject runs privileged project environment operations. It does
// not own HTTP admission, Tailnet policy, terminal attach or runner protocols.
package hostproject

import (
	"context"
)

type Executor interface {
	Run(context.Context, []byte, string, ...string) ([]byte, error)
}

type Runtime struct {
	Exec   Executor
	Config Config
}

func (r *Runtime) podman(ctx context.Context, in []byte, args ...string) ([]byte, error) {
	return r.Exec.Run(ctx, in, "/usr/bin/podman", args...)
}
