package api

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/levitateos/factory/pkg/factory"
)

func (s *API) factoryRepairPreview(ctx context.Context, cap factory.MCPCapability, report json.RawMessage) (factory.RepairPreview, error) {
	if s == nil || s.Coordinator == nil {
		return factory.RepairPreview{}, errors.New("repair preview is unavailable")
	}
	return s.Coordinator.PreviewRepair(ctx, cap, report)
}
