package api

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/levitateos/sodaos/internal/factory"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// factoryMCPResearchSource serves the researcher's progressive scope.
type factoryMCPResearchSource interface {
	ResearchScope(context.Context, factory.MCPCapability) (factory.ResearchScopeView, error)
}

func researchToolDescription(tool factory.MCPTool) string {
	switch string(tool) {
	case "factory.research_scope":
		return "Read your research assignment's declared subject, source refs and recorded questions with per-question coverage. Reads record nothing."
	case "factory.record_research":
		return "Save one bounded batch of declared questions, findings with evidence, experiments, unanswered questions and inspected sources to your assignment log."
	}
	return ""
}

func researchPayloadSchema(tool factory.MCPTool) *jsonschema.Schema {
	if tool != factory.MCPToolRecordResearch {
		return nil
	}
	text := func() *jsonschema.Schema { return &jsonschema.Schema{Type: "string"} }
	texts := func(max int) *jsonschema.Schema {
		return &jsonschema.Schema{Type: "array", MaxItems: ptrInt(max), Items: text()}
	}
	finding := func() *jsonschema.Schema {
		return &jsonschema.Schema{Type: "object", Properties: map[string]*jsonschema.Schema{
			"question": {Type: "integer"},
			"finding":  text(),
			"evidence": texts(32),
		}, Required: []string{"question", "finding"}, AdditionalProperties: &jsonschema.Schema{Not: &jsonschema.Schema{}}}
	}
	o := &jsonschema.Schema{Type: "object", Properties: map[string]*jsonschema.Schema{
		"questions":   texts(16),
		"findings":    {Type: "array", MaxItems: ptrInt(64), Items: finding()},
		"experiments": texts(32),
		"unanswered":  {Type: "array", MaxItems: ptrInt(16), Items: &jsonschema.Schema{Type: "integer"}},
		"sources":     texts(32),
	}, AdditionalProperties: &jsonschema.Schema{Not: &jsonschema.Schema{}}}
	return o
}

func checkResearchPayload(tool factory.MCPTool, payload map[string]any) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = factory.DecodeResearchRecord(tool, raw)
	return err
}

func (s *factoryCatalogServer) serveResearchScope(ctx context.Context, tool factory.MCPTool, args mcpCallArgs) (*mcp.CallToolResult, any, error) {
	cap, release, err := s.admitCall(ctx, tool, args)
	if err != nil {
		return nil, nil, err
	}
	defer release()
	source, ok := s.deps.Submissions.(factoryMCPResearchSource)
	if !ok {
		return nil, nil, errors.New("research scope is unavailable")
	}
	view, err := source.ResearchScope(ctx, cap)
	if err != nil {
		return nil, nil, err
	}
	return mcpJSONResult(view)
}
