package api

import (
	"encoding/json"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/store"
)

// communicationPayloadSchema exposes the usable structured payload in tool
// discovery; server decoding enforces the same bounds on direct calls.
func communicationPayloadSchema(tool factory.MCPTool) *jsonschema.Schema {
	if !factory.MCPIsCommunication(tool) {
		return nil
	}
	props := map[string]*jsonschema.Schema{}
	for _, key := range []string{"title", "summary", "question", "resolution", "scope", "decision", "rationale", "parent_id", "before", "after"} {
		props[key] = &jsonschema.Schema{Type: "string"}
	}
	props["severity"] = &jsonschema.Schema{Type: "string", Enum: []any{"info", "warning", "critical"}}
	props["commits"] = &jsonschema.Schema{Type: "array", MaxItems: ptrInt(8), Description: "Actual saved commits, in chronological order. Full revision and a short change summary. Recording commits does not claim they passed review.", Items: &jsonschema.Schema{Type: "object", Properties: map[string]*jsonschema.Schema{"revision": {Type: "string"}, "summary": {Type: "string"}}, Required: []string{"revision", "summary"}, AdditionalProperties: &jsonschema.Schema{Not: &jsonschema.Schema{}}}}
	props["source"] = &jsonschema.Schema{Type: "object", Description: "Optional actual source evidence: path, revision, starting line and a short code excerpt. highlight_lines uses absolute line numbers. Supply after for a correction comparison. Reported evidence does not establish verification.", Properties: map[string]*jsonschema.Schema{
		"path": {Type: "string"}, "revision": {Type: "string"}, "start_line": {Type: "integer"}, "code": {Type: "string"}, "after": {Type: "string"}, "highlight_lines": {Type: "array", Items: &jsonschema.Schema{Type: "integer"}},
	}, Required: []string{"path", "start_line", "code"}, AdditionalProperties: &jsonschema.Schema{Not: &jsonschema.Schema{}}}
	for _, key := range []string{"evidence_refs", "options", "completed", "remaining", "attempts", "alternatives"} {
		props[key] = &jsonschema.Schema{Type: "array", Items: &jsonschema.Schema{Type: "string"}}
	}
	props["items"] = &jsonschema.Schema{Type: "array", Items: &jsonschema.Schema{Type: "object", Properties: map[string]*jsonschema.Schema{"id": {Type: "string"}, "depends_on": {Type: "array", Items: &jsonschema.Schema{Type: "string"}}, "label": {Type: "string"}, "detail": {Type: "string"}, "state": {Type: "string"}}, Required: []string{"label"}, AdditionalProperties: &jsonschema.Schema{Not: &jsonschema.Schema{}}}}
	props["artifacts"] = &jsonschema.Schema{Type: "array", Items: &jsonschema.Schema{Type: "object", Properties: map[string]*jsonschema.Schema{"id": {Type: "string"}}, Required: []string{"id"}, AdditionalProperties: &jsonschema.Schema{Not: &jsonschema.Schema{}}}}
	required := []string{"title", "summary"}
	switch tool {
	case factory.MCPToolAsk:
		required = append(required, "question")
	case factory.MCPToolAnswer, factory.MCPToolReply:
		required = append(required, "parent_id")
	case factory.MCPToolFlag:
		required = append(required, "severity")
	case factory.MCPToolLearned:
		required = append(required, "scope", "evidence_refs")
	case factory.MCPToolSuggest:
		required = append(required, "rationale")
	case factory.MCPToolBlocked:
		required = append(required, "resolution")
	case factory.MCPToolStuck:
		required = append(required, "attempts")
	case factory.MCPToolRecordDecision:
		required = append(required, "decision", "rationale")
	}
	return &jsonschema.Schema{Type: "object", Properties: props, Required: required, AdditionalProperties: &jsonschema.Schema{Not: &jsonschema.Schema{}}}
}

func checkCommunicationPayload(tool factory.MCPTool, payload map[string]any) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = factory.DecodeMCPCommunication(tool, raw)
	return err
}

// Worker communications are recorded content; no human comment routing request
// was created, so their reads must not invent a pending JEV pipeline.
func feedWorkerPostResponse(detail store.FeedPostDetail) map[string]any {
	var communication *factory.FeedCommunication
	if detail.Payload != nil {
		communication = detail.Payload.Communication
	}
	return map[string]any{
		"post_id": detail.Record.ID, "revision": detail.Record.Revision, "repository": detail.Record.Repository,
		"author": detail.Record.Author, "author_name": detail.AuthorName, "actor": detail.Actor, "at": detail.At,
		"issue_scope": detail.Record.Issue, "parent_id": detail.Record.ParentID, "body": postBodyFor(detail),
		"status": detail.Status, "deleted": detail.Record.Deleted, "revoked": detail.Record.Revoked, "edited": false,
		"can_retry": false, "communication": communication, "consultation": detail.Consultation,
		"assignment": detail.Assignment, "source": detail.Source,
		"contribution": detail.Contribution,
	}
}
