package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/levitateos/sodaos/internal/factory"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func repairPreviewPayloadSchema() *jsonschema.Schema {
	return &jsonschema.Schema{Type: "object", Properties: map[string]*jsonschema.Schema{
		"report": {AnyOf: []*jsonschema.Schema{mcpCarrier[factory.ReassessmentReport](), mcpCarrier[factory.StallRepairReport](), mcpCarrier[factory.DeliveryRepairReport]()}},
	}, Required: []string{"report"}, AdditionalProperties: &jsonschema.Schema{Not: &jsonschema.Schema{}}}
}

func (s *factoryCatalogServer) serveRepairPreview(ctx context.Context, args mcpCallArgs) (*mcp.CallToolResult, any, error) {
	cap, release, err := s.admitCall(ctx, factory.MCPToolPreviewRepair, args)
	if err != nil {
		return nil, nil, err
	}
	defer release()
	if s.deps.RepairPreview == nil {
		return nil, nil, errors.New("repair preview is unavailable")
	}
	raw, err := canonicalPayload(args.Payload)
	if err != nil {
		return nil, nil, err
	}
	var payload struct {
		Report json.RawMessage `json:"report"`
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if d.Decode(&payload) != nil || len(payload.Report) == 0 || bytes.Equal(payload.Report, []byte("null")) {
		return nil, nil, errors.New("repair preview needs a typed report")
	}
	preview, err := s.deps.RepairPreview(ctx, cap, payload.Report)
	if err != nil {
		return nil, nil, err
	}
	return mcpJSONResult(preview)
}

func (s *API) factoryRepairPreview(ctx context.Context, cap factory.MCPCapability, report json.RawMessage) (factory.RepairPreview, error) {
	if s == nil || s.Coordinator == nil {
		return factory.RepairPreview{}, errors.New("repair preview is unavailable")
	}
	return s.Coordinator.PreviewRepair(ctx, cap, report)
}
