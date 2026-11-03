package api

import (
	"context"

	"github.com/levitateos/factory/pkg/factory"
	factoryservice "github.com/levitateos/factory/pkg/factory/service"
)

// prepareFactoryWorkerMCP activates a staged interface only after the host sees
// its real supervised process. Every request repeats that observation and the
// stored stop checks, so a retained credential cannot outlive the worker.
func (s *API) prepareFactoryWorkerMCP(ctx context.Context, capability factory.MCPCapability) (factory.MCPCapability, error) {
	admission := factoryservice.WorkerAdmission{Store: s.Store}
	if s.Host != nil {
		admission.Inspect = s.Host.InspectRun
	}
	return admission.Prepare(ctx, capability)
}
