package api

import (
	"context"
	"errors"

	"github.com/levitateos/sodaos/internal/factory"
)

// requireCurrentWork checks the required work inputs directly. It does not
// render context pages or validate the optional context-selection receipt.
func (s *factoryCatalogServer) requireCurrentWork(ctx context.Context, cap factory.MCPCapability) error {
	source, ok := s.deps.Submissions.(factoryMCPCurrentContextSource)
	if !ok {
		return errors.New("required work inputs are unavailable")
	}
	current, err := source.FactoryCurrentContext(ctx, cap, s.now())
	if err != nil {
		return err
	}
	if cap.Profile == factory.MCPProfileTestPilot {
		if err := s.addCurrentVerification(ctx, cap, &current); err != nil {
			return err
		}
	}
	if current.Evidence.Availability != "complete" {
		return errors.New("required work inputs are unavailable; read factory.context for the missing inputs")
	}
	return nil
}
