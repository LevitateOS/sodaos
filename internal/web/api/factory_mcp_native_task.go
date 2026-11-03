package api

import (
	"context"

	"github.com/levitateos/factory/pkg/factory"
	factoryservice "github.com/levitateos/factory/pkg/factory/service"
)

func (s *API) factoryNativeTask(ctx context.Context, cap factory.MCPCapability, action, taskID string) (map[string]any, error) {
	tasks := factoryservice.WorkerTasks{Store: s.Store}
	if s.Host != nil {
		tasks.Inspect = s.Host.InspectRun
		tasks.Command = s.Host.CommandTask
	}
	return tasks.Execute(ctx, cap, action, taskID)
}
