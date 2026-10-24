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

type factoryMCPReassessmentScopeSource interface {
	ReassessmentScope(context.Context, factory.MCPCapability, int64) (factory.RepairScopeView, error)
}

func repairScopePayloadSchema() *jsonschema.Schema {
	shape := contextPayloadSchema()
	shape.Properties["after"] = &jsonschema.Schema{Type: "integer", Description: "Continue this assignment's diagnosis and draft references after the previous next_after sequence."}
	return shape
}

func (s *factoryCatalogServer) serveReassessmentScope(ctx context.Context, args mcpCallArgs) (*mcp.CallToolResult, any, error) {
	var payload struct {
		After  int64   `json:"after"`
		Cursor *string `json:"cursor"`
	}
	if args.Payload != nil {
		raw, err := json.Marshal(args.Payload)
		if err != nil {
			return nil, nil, err
		}
		d := json.NewDecoder(bytes.NewReader(raw))
		d.DisallowUnknownFields()
		if d.Decode(&payload) != nil || payload.After < 0 || payload.Cursor != nil && len(*payload.Cursor) > 1024 {
			return nil, nil, errors.New("invalid reassessment scope cursor")
		}
	}
	read, err := s.loadReadContext(ctx, factory.MCPToolReassessmentScope, args)
	if err != nil {
		return nil, nil, err
	}
	defer read.release()
	if args.ExpectedInputDigest != "" && args.ExpectedInputDigest != read.capability.ScopeDigest() {
		return nil, nil, errors.New("reassessment scope input differs from admitted scope")
	}
	source, ok := s.deps.Submissions.(factoryMCPReassessmentScopeSource)
	if !ok {
		return nil, nil, errors.New("reassessment scope is unavailable")
	}
	view, err := source.ReassessmentScope(ctx, read.capability, payload.After)
	if err != nil {
		return nil, nil, err
	}
	attemptEligible, lineageEligible := factory.BundleEligibility(read.assignment, read.parentLoaded)
	foregrounded, reason, fingerprint := s.foregroundReadBundle(ctx, read, attemptEligible, lineageEligible)
	body := buildReadBody(read, attemptEligible, lineageEligible, foregrounded)
	action := "serve"
	if reason != "" {
		action = "hold"
	}
	body["context_selection"] = map[string]any{"bundle": foregrounded, "action": action, "reason_code": reason, "fingerprint": fingerprint}
	body["reassessment_scope"] = view
	contextArgs := args
	contextArgs.Payload = nil
	if payload.Cursor != nil {
		contextArgs.Payload = map[string]any{"cursor": *payload.Cursor}
	}
	if err := s.addCurrentContext(ctx, read, body, contextArgs); err != nil {
		return nil, nil, err
	}
	return mcpJSONResult(body)
}
