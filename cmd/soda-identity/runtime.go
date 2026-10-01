package main

import (
	"context"

	"github.com/levitateos/sodaos/internal/host"
	"github.com/levitateos/sodaos/internal/identity"
)

// nativeRuntime adapts broker lease callbacks to the fixed host process
// contract. The daemon attests terminal sessions and supervised factory runs
// by their exact bindings; the broker never shells out to runtimes itself.
type nativeRuntime struct {
	Host *host.Client
}

func (n nativeRuntime) Finish(ctx context.Context, l identity.Lease) ([]byte, error) {
	if l.Binding == nil {
		return nil, identity.ErrDenied
	}
	out, err := n.Host.IdentityFinish(ctx, l)
	return out.Credential, err
}

func (n nativeRuntime) Validate(ctx context.Context, l identity.Lease) error {
	if l.Binding == nil {
		return identity.ErrDenied
	}
	return n.Host.IdentityValidate(ctx, l)
}

func (n nativeRuntime) Stop(ctx context.Context, l identity.Lease) error {
	if l.Binding == nil {
		return nil
	}
	return n.Host.IdentityStop(ctx, l)
}
