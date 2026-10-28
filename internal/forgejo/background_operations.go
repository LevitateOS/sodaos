package forgejo

import (
	"context"
	"errors"

	extensions "forgejo.org/extension-sdk"
)

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
