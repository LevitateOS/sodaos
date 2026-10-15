package api

import (
	"context"
	"errors"

	extensions "forgejo.org/extension-sdk"
)

var errLifecycleNotAdmitted = errors.New("lifecycle tuple is not admitted for this controller")

// SodaServicePeerUID is the fixed UID the Soda service presents on
// Fountain's service callback (SERVICE_BRIDGE_PEERS=2000:soda). The
// controller pins its forward transport against this identity.
const SodaServicePeerUID = 2000

// LifecycleController implements the required soda controller for the
// optional tailnet package (RCX-2 T01). The manager admits exactly one tuple
// per token on the SDK-owned endpoint; the handler verifies that authority
// independently, scopes the tuple to this controller and target, then
// forwards the exact call with its opaque authority to the Soda service
// over the pinned Unix transport. The socket and service peer UID come
// from trusted local service configuration, never request parameters.
// Rejected, replayed, wrong-variant or unbound authority returns an error
// before any forward, so it causes zero host effects.
func LifecycleController(socket string, servicePeerUID uint32) extensions.LifecycleHandler {
	return func(ctx context.Context, authority extensions.LifecycleAuthority, call extensions.LifecycleCall) (extensions.LifecycleObservation, error) {
		if _, err := authority.Verify(ctx, call); err != nil {
			return extensions.LifecycleObservation{}, err
		}
		if call.ControllerID != extensionLifecycleController || call.PackageID != extensionLifecyclePackage {
			return extensions.LifecycleObservation{}, errLifecycleNotAdmitted
		}
		return authority.ForwardUnix(ctx, socket, servicePeerUID, extensionLifecycleRoute, call)
	}
}
