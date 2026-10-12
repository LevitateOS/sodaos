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
// unattributable handshake deletes every marker before returning: prior
// proof is never left usable for new enables or new effects, so
// availability fails closed until a verified answer arrives. Saved
// configurations and the action journal are preserved untouched, and
// in-flight operations keep reconciling by operation identity; only new
// admission closes.
func ReconcileLifecycleCapabilities(ctx context.Context, s *store.Store, src CapabilityHandshakeSource) error {
	h, err := src.Handshake(ctx)
	if err != nil {
		if clearErr := s.ClearLifecycleCapabilities(ctx); clearErr != nil {
			return clearErr
		}
		return err
	}
	if !factory.ValidHandshake(h) {
		if clearErr := s.ClearLifecycleCapabilities(ctx); clearErr != nil {
			return clearErr
		}
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

// RequireCurrentCapability admits new enables and new effects for one
// event only while its capability marker verifies against the latest
// successful handshake. Every producer of new optional effects must call
// this immediately before the first attempt; reconciliation of
// already-submitted operations by identity is exempt. Stored
// configurations and history are never deleted by admission loss.
func RequireCurrentCapability(ctx context.Context, s *store.Store, event string) error {
	proved, err := s.CapabilityProved(ctx, event)
	if err != nil {
		return err
	}
	if !proved {
		return store.ErrCapabilityUnproved
	}
	return nil
}
