package web

import (
	"context"
	"strconv"

	"github.com/levitateos/factory/pkg/collaboration"
	"github.com/levitateos/factory/pkg/factory/control"
	"github.com/levitateos/factory/pkg/factory/execution"
	"github.com/levitateos/sodaos/internal/forgejo"
)

// Appliance composition translates the execution actor at the native credential
// boundary. The coordinator's captured launch and assignments remain canonical;
// the host and its broker consume native Fountain user identities.
type applianceFactoryHost struct {
	control.HostFactory
	identities *forgejo.FountainIdentities
}

var _ control.HostFactory = (*applianceFactoryHost)(nil)

func (h *applianceFactoryHost) LaunchRun(ctx context.Context, request execution.LaunchRequest) (execution.RunObservation, error) {
	if h == nil || h.HostFactory == nil || h.identities == nil || request.Validate() != nil || request.Kind != collaboration.Appliance {
		return execution.RunObservation{}, execution.ErrLaunch
	}
	native, err := h.identities.NativeActor(ctx, collaboration.ActorID(request.Input.Run.Actor))
	if err != nil {
		return execution.RunObservation{}, err
	}
	actor, err := strconv.ParseInt(native, 10, 64)
	if err != nil || actor < 1 {
		return execution.RunObservation{}, execution.ErrLaunch
	}
	// FactoryLaunch.Run is a value: translate the outgoing copy without changing
	// prompt/manifest bytes, custody references or the captured canonical packet.
	request.Input.Run.Actor = actor
	return h.HostFactory.LaunchRun(ctx, request)
}
