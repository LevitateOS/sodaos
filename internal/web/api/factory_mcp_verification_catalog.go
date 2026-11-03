package api

import (
	"context"

	"github.com/levitateos/factory/pkg/factory"
	factoryservice "github.com/levitateos/factory/pkg/factory/service"
)

func (s *API) factoryApprovedVerificationCatalog(ctx context.Context, cap factory.MCPCapability) (factory.ApprovedCatalog, error) {
	views := factoryservice.WorkerVerification{Store: s.Store}
	if s.Coordinator != nil {
		views.Catalog = s.Coordinator.ApprovedProjectCheckCatalog
	}
	return views.ApprovedCatalog(ctx, cap)
}
