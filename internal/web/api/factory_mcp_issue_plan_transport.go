package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/strictjson"
)

// Advertise the complete current plan without putting new plan validation
// ahead of immutable receipt recovery. Generic SDK AddTool validates the
// advertised schema before calling us, which refuses older retained reports.
// The raw registration keeps envelope validation here; serveMutation checks
// the current typed plan only after its retained-submission lookup.
func (s *factoryCatalogServer) addIssuePlanTool(server *mcp.Server, definition *mcp.Tool) {
	envelope := definition.InputSchema.(*jsonschema.Schema).CloneSchemas()
	envelope.Properties["payload"] = &jsonschema.Schema{Type: "object"}
	resolved, err := envelope.Resolve(nil)
	if err != nil {
		panic(fmt.Sprintf("issue plan envelope schema: %v", err))
	}
	server.AddTool(definition, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		refuse := func(err error) (*mcp.CallToolResult, error) {
			result := &mcp.CallToolResult{}
			result.SetError(err)
			return result, nil
		}
		var input map[string]any
		if err := strictjson.Decode(bytes.NewReader(req.Params.Arguments), &input); err != nil {
			return refuse(err)
		}
		if err := resolved.Validate(input); err != nil {
			return refuse(fmt.Errorf("validating arguments: %w", err))
		}
		var args mcpCallArgs
		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
			return refuse(err)
		}
		args.SubmissionID = mcpSubmissionID(args.SubmissionID)
		result, _, err := s.serveMutation(ctx, factory.MCPToolFinishIssuePlan, args)
		if err != nil {
			return refuse(err)
		}
		return result, nil
	})
}
