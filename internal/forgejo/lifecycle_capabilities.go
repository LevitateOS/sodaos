package forgejo

import (
	"context"
	"errors"
	"sync"
	"time"

	extensions "forgejo.org/extension-sdk"

	"github.com/levitateos/sodaos/internal/factory"
)

// CapabilityHandshakeSource adapts a native background client to the
// lifecycle registrar's handshake. It records the latest answer (or
// failure) for current-identity logs; availability itself is fenced by
// registrar rows, never by this record.
type CapabilityHandshakeSource struct {
	client extensions.BackgroundClient

	mu           sync.Mutex
	installation string
	generation   string
	handshakeErr error
	at           time.Time
}

// NewCapabilityHandshakeSource builds a registrar handshake source over
// the shared service transport.
func NewCapabilityHandshakeSource(client extensions.BackgroundClient) *CapabilityHandshakeSource {
	return &CapabilityHandshakeSource{client: client}
}

// Handshake reads one authenticated native capability answer.
func (s *CapabilityHandshakeSource) Handshake(ctx context.Context) (factory.LifecycleHandshake, error) {
	answer, err := s.client.ReadOperations(ctx)
	s.mu.Lock()
	defer s.mu.Unlock()
	if err != nil {
		s.installation, s.generation, s.handshakeErr, s.at = "", "", err, time.Now()
		return factory.LifecycleHandshake{}, err
	}
	s.installation, s.generation, s.handshakeErr, s.at = answer.Installation, answer.Generation, nil, time.Now()
	return factory.LifecycleHandshake{Installation: answer.Installation, Generation: answer.Generation, Kinds: answer.Kinds}, nil
}

// LastHandshake reports the latest handshake outcome for refresh logs.
func (s *CapabilityHandshakeSource) LastHandshake() (installation, generation string, err error, at time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.installation, s.generation, s.handshakeErr, s.at
}

// ReadOperations queries the read-only native operations handshake:
// installation, stable generation and supported kinds. The answer
// installation must equal the bootstrapped pinned installation.
func (b *ServiceBackground) ReadOperations(ctx context.Context) (extensions.OperationsHandshake, error) {
	var handshake extensions.OperationsHandshake
	if err := b.call(ctx, extensions.BackgroundOperationsPath, struct{}{}, &handshake); err != nil {
		return extensions.OperationsHandshake{}, err
	}
	if err := extensions.ValidateOperationsHandshake(handshake); err != nil {
		return extensions.OperationsHandshake{}, err
	}
	b.mu.Lock()
	pinned := b.pinned
	b.mu.Unlock()
	if pinned == "" || handshake.Installation != pinned {
		return extensions.OperationsHandshake{}, errors.New("invalid background response")
	}
	return handshake, nil
}
