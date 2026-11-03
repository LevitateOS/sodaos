package api

import (
	"context"

	"github.com/levitateos/factory/pkg/factory"
)

func (s *API) factoryIssueMakerTool(ctx context.Context, cap factory.MCPCapability, tool factory.MCPTool, payload map[string]any, submission string) (map[string]any, error) {
	return s.capturedIssueEffects().Tool(ctx, cap, tool, payload, submission)
}
