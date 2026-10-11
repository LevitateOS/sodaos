package control

import (
	"context"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/store"
)

// CapabilityHandshakeSource produces one authenticated native capability
// answer. Lane C supplies the production source backed by the native
// read-only operations handshake; tests supply scripted answers. The
// answer must bind the current installation and a stable native
// operations generation.
type CapabilityHandshakeSource interface {
	Handshake(ctx context.Context) (factory.LifecycleHandshake, error)
}

// ReconcileLifecycleCapabilities registers one installation-bound marker
// per supported lifecycle kind from an authenticated handshake. Markers
// for unsupported events are deleted, as are markers bound to any other
// installation; a changed native generation re-registers. A failed or
// unattributable handshake writes nothing and leaves prior markers
// untouched, failing closed: effects stay unavailable until a verified
// answer arrives. In-flight actions are not revoked here; they keep
// their own per-attempt revalidation and fail safely against native
// refusal.
func ReconcileLifecycleCapabilities(ctx context.Context, s *store.Store, src CapabilityHandshakeSource) error {
	h, err := src.Handshake(ctx)
	if err != nil {
		return err
	}
	if !factory.ValidHandshake(h) {
		return factory.ErrLifecycleAction
	}
	if err := s.DeleteForeignCapabilities(ctx, h.Installation); err != nil {
		return err
	}
	supported := map[string]bool{}
	for _, kind := range h.Kinds {
		supported[kind] = true
	}
	for _, event := range []string{factory.LifecycleEventIssueCreated, factory.LifecycleEventPRCreated, factory.LifecycleEventVerifiedMerge} {
		if supported[factory.LifecycleEventKind(event)] {
			if _, err := s.RegisterCapability(ctx, event, h.Generation, h.Installation); err != nil {
				return err
			}
			continue
		}
		if err := s.DeleteCapability(ctx, event); err != nil {
			return err
		}
	}
	return nil
}
