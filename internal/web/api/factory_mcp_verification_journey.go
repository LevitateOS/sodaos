package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"slices"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/levitateos/sodaos/internal/factory"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type factoryMCPVerificationJourneySource interface {
	VerificationScope(context.Context, factory.MCPCapability, int64) (factory.VerificationScopeView, error)
}

func verificationJourneyPayloadSchema(tool factory.MCPTool) *jsonschema.Schema {
	switch tool {
	case factory.MCPToolRecordJourneyResult:
		schema := mcpCarrier[factory.VerificationJourneyResult]()
		filled := map[string]bool{"delivery_id": true, "input_digest": true, "source_commit": true, "preparation": true, "manifest_digest": true}
		schema.Required = slices.DeleteFunc(schema.Required, func(field string) bool { return filled[field] })
		for field := range filled {
			if property := schema.Properties[field]; property != nil {
				property.Description = serverFilled
			}
		}
		return schema
	case factory.MCPToolVerificationScope:
		return &jsonschema.Schema{Type: "object", Properties: map[string]*jsonschema.Schema{"after": {Type: "integer"}}, AdditionalProperties: &jsonschema.Schema{Not: &jsonschema.Schema{}}}
	default:
		return nil
	}
}

func checkVerificationJourneyPayload(payload map[string]any) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = factory.DecodeVerificationJourney(raw)
	return err
}

func (s *factoryCatalogServer) serveVerificationScope(ctx context.Context, args mcpCallArgs) (*mcp.CallToolResult, any, error) {
	cap, release, err := s.admitCall(ctx, factory.MCPToolVerificationScope, args)
	if err != nil {
		return nil, nil, err
	}
	defer release()
	source, ok := s.deps.Submissions.(factoryMCPVerificationJourneySource)
	if !ok {
		return nil, nil, errors.New("verification scope is unavailable")
	}
	var page struct {
		After int64 `json:"after"`
	}
	if args.Payload != nil {
		raw, err := json.Marshal(args.Payload)
		if err != nil {
			return nil, nil, err
		}
		d := json.NewDecoder(bytes.NewReader(raw))
		d.DisallowUnknownFields()
		if d.Decode(&page) != nil || page.After < 0 {
			return nil, nil, errors.New("invalid verification scope cursor")
		}
	}
	view, err := source.VerificationScope(ctx, cap, page.After)
	if err != nil {
		return nil, nil, err
	}
	return mcpJSONResult(view)
}
