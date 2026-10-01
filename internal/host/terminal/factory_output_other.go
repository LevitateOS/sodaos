//go:build !linux

package terminal

import (
	"context"

	"github.com/levitateos/sodaos/internal/identity"
)

func (s *Service) FactoryCodexOutput(context.Context, string, *identity.Binding, int64, int) (FactoryCodexOutputSlice, error) {
	return FactoryCodexOutputSlice{}, errFactoryUnavailable
}
