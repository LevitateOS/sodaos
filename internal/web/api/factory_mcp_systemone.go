package api

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/google/jsonschema-go/jsonschema"
	"github.com/levitateos/sodaos/internal/factory"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"strings"
)

type FactorySystemOne interface {
	Evaluate(context.Context, string, any, any, any) (map[string]any, error)
}
type systemOneArgs struct {
	State        any    `json:"state"`
	Instructions any    `json:"instructions"`
	Criteria     any    `json:"criteria,omitempty"`
	CapabilityID string `json:"capability_id,omitempty"`
}

func isSystemOneTool(tool factory.MCPTool) bool {
	return tool == factory.MCPToolSystemOneNoul || tool == factory.MCPToolSystemOneChoice || tool == factory.MCPToolSystemOneScore
}
func (s *factoryCatalogServer) addSystemOneTool(server *mcp.Server, tool factory.MCPTool) {
	kind := strings.TrimPrefix(string(tool), "systemone.")
	schema := &jsonschema.Schema{Type: "object", Properties: map[string]*jsonschema.Schema{"state": {}, "instructions": {Types: []string{"string", "object", "array"}}, "criteria": {}}, Required: []string{"state", "instructions"}, AdditionalProperties: &jsonschema.Schema{Not: &jsonschema.Schema{}}}
	description := "Ask a generic System One " + kind + " judgment using supplied state. Returns typed probabilities; does not change Factory or library state. Never include credentials."
	switch kind {
	case "noul":
		schema.Properties["criteria"] = &jsonschema.Schema{Type: "object", Properties: map[string]*jsonschema.Schema{"true": {}, "false": {}}, AdditionalProperties: &jsonschema.Schema{Not: &jsonschema.Schema{}}}
		description += " Noul is probability of yes from 0 to 1."
	case "choice":
		schema.Properties["criteria"] = &jsonschema.Schema{Type: "object", Description: "2–255 named alternatives with their descriptions."}
		schema.Required = append(schema.Required, "criteria")
	case "score":
		schema.Properties["criteria"] = &jsonschema.Schema{Type: "array", Items: &jsonschema.Schema{Types: []string{"string", "object", "array"}}, MinItems: ptrInt(2), MaxItems: ptrInt(10), Description: "Ordered descriptions from lowest to highest; response score is between 0 and level count minus one."}
		schema.Required = append(schema.Required, "criteria")
	}
	if s.deps.BoundCapability == nil {
		schema.Properties["capability_id"] = &jsonschema.Schema{Type: "string"}
		schema.Required = append(schema.Required, "capability_id")
	}
	mcp.AddTool(server, &mcp.Tool{Name: string(tool), Description: description, InputSchema: schema}, func(ctx context.Context, _ *mcp.CallToolRequest, args systemOneArgs) (*mcp.CallToolResult, any, error) {
		_, release, err := s.admitCall(ctx, tool, mcpCallArgs{CapabilityID: args.CapabilityID})
		if err != nil {
			return nil, nil, err
		}
		defer release()
		if s.deps.SystemOne == nil {
			return nil, nil, errors.New("System One provider is not configured")
		}
		raw, err := json.Marshal(args)
		if err != nil || len(raw) > factory.MCPMaxSubmissionPayload {
			return nil, nil, errors.New("System One question exceeds 64 KiB")
		}
		if args.Instructions == nil {
			return nil, nil, errors.New("instructions are required")
		}
		if text, ok := args.Instructions.(string); ok && strings.TrimSpace(text) == "" {
			return nil, nil, errors.New("instructions are empty")
		}
		if kind == "choice" {
			criteria, ok := args.Criteria.(map[string]any)
			if !ok || len(criteria) < 2 || len(criteria) > 255 {
				return nil, nil, errors.New("choice requires 2–255 alternatives")
			}
		}
		result, err := s.deps.SystemOne.Evaluate(ctx, kind, args.State, args.Instructions, args.Criteria)
		if err != nil {
			return nil, nil, err
		}
		return mcpJSONResult(result)
	})
}
