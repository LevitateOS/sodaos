package api

import (
	"context"

	"github.com/levitateos/factory/pkg/factory"
	factoryservice "github.com/levitateos/factory/pkg/factory/service"
)

func (s *API) factoryVerificationEvidence(ctx context.Context, cap factory.MCPCapability) (map[string]any, error) {
	views := factoryservice.WorkerVerification{Store: s.Store}
	if s.Coordinator != nil {
		views.Read = s.Coordinator.ApplicationVerificationEvidence
	}
	return views.Evidence(ctx, cap)
}
