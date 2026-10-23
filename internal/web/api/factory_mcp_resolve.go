package api

import (
	"encoding/json"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/levitateos/sodaos/internal/factory"
)

func resolvePostToolDescription(tool factory.MCPTool) string {
	if string(tool) != "factory.resolve_post" {
		return ""
	}
	return "Record one disposition for a suggestion or answer post: applied or refused for suggestions, incorporated for adopted answers. One disposition per post; it rides beside the immutable post."
}

func resolvePostPayloadSchema(tool factory.MCPTool) *jsonschema.Schema {
	if tool != factory.MCPToolResolvePost {
		return nil
	}
	return &jsonschema.Schema{Type: "object", Properties: map[string]*jsonschema.Schema{
		"post":       {Type: "string"},
		"resolution": {Type: "string", Enum: []any{"applied", "refused", "incorporated"}},
		"reason":     {Type: "string"},
	}, Required: []string{"post", "resolution"}, AdditionalProperties: &jsonschema.Schema{Not: &jsonschema.Schema{}}}
}

func checkResolvePostPayload(tool factory.MCPTool, payload map[string]any) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = factory.DecodePostResolution(tool, raw)
	return err
}
