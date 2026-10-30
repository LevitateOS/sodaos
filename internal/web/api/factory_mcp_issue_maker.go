package api

import (
	"context"
	"errors"
	"strings"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/levitateos/sodaos/internal/factory"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func isIssueMakerTool(tool factory.MCPTool) bool {
	return tool == factory.MCPToolCreateIssue || tool == factory.MCPToolRefineIssue
}

func issueMakerPayloadSchema() *jsonschema.Schema {
	one, maxTitle, maxBody := 1, 255, 16384
	return &jsonschema.Schema{Type: "object", Properties: map[string]*jsonschema.Schema{
		"key": {Type: "string", Pattern: "^[a-z][a-z0-9_-]{0,31}$"}, "title": {Type: "string", MinLength: &one, MaxLength: &maxTitle}, "body": {Type: "string", MinLength: &one, MaxLength: &maxBody},
	}, Required: []string{"key", "title", "body"}, AdditionalProperties: &jsonschema.Schema{Not: &jsonschema.Schema{}}}
}

func checkIssueMakerPayload(payload map[string]any) error {
	key, _ := payload["key"].(string)
	title, _ := payload["title"].(string)
	body, _ := payload["body"].(string)
	if len(payload) != 3 || !factory.ValidPlanKey(key) || strings.TrimSpace(title) == "" || len(title) > 255 || strings.TrimSpace(body) == "" || len(body) > 16384 {
		return errors.New("supply only a stable key, title (255 bytes) and body (16384 bytes)")
	}
	return nil
}

func (s *factoryCatalogServer) serveIssueMaker(ctx context.Context, tool factory.MCPTool, args mcpCallArgs) (*mcp.CallToolResult, any, error) {
	cap, submission, release, err := s.admitMutation(ctx, tool, args)
	if err != nil {
		return nil, nil, err
	}
	defer release()
	if receipt, err := s.deps.Submissions.MCPReceipt(ctx, cap.AssignmentID, submission.SubmissionID); err == nil {
		if receipt.InputDigest != submission.PayloadHash {
			return nil, nil, errors.New("submission identity already records different issue content")
		}
		return mcpJSONResult(s.receiptBody(ctx, cap.AssignmentID, receipt))
	}
	if s.deps.IssueMaker == nil {
		return nil, nil, errors.New("Issue Maker native tools are unavailable")
	}
	result, err := s.deps.IssueMaker(ctx, cap, tool, args.Payload, submission.SubmissionID)
	if err != nil {
		return nil, nil, err
	}
	post, _ := result["post_id"].(string)
	receipt := factory.MCPReceipt{SubmissionID: submission.SubmissionID, Tool: tool, State: factory.MCPReceiptRecorded, InputDigest: submission.PayloadHash, FeedPostID: post}
	recorded, err := s.deps.Submissions.RecordMCPSubmission(ctx, submission, receipt)
	if err != nil {
		return nil, nil, err
	}
	result["receipt"] = s.receiptBody(ctx, cap.AssignmentID, recorded)
	return mcpJSONResult(result)
}

func (s *API) factoryIssueMakerTool(ctx context.Context, cap factory.MCPCapability, tool factory.MCPTool, payload map[string]any, submission string) (map[string]any, error) {
	if cap.Profile != factory.MCPProfileIssueMaker || !isIssueMakerTool(tool) {
		return nil, errors.New("issue creation belongs to the assigned Issue Maker role")
	}
	if err := s.Store.CheckMCPWorkActive(ctx, cap); err != nil {
		return nil, err
	}
	a, err := s.Store.Assignment(ctx, cap.AssignmentID)
	if err != nil {
		return nil, err
	}
	if a.PlanningContext == nil || a.PlanningContext.IssueMaker == nil || a.Repository != cap.Repository || a.Run != cap.RunID {
		return nil, errors.New("issue maker scope is unavailable")
	}
	scope := *a.PlanningContext.IssueMaker
	if err = scope.Validate(a.PlanningContext.Head); err != nil {
		return nil, err
	}
	comment, err := s.Store.FeedPost(ctx, scope.SourcePost)
	if err != nil || comment.Record.Repository != cap.Repository {
		return nil, errors.New("issue maker request is outside this repository")
	}
	key, title, body := payload["key"].(string), payload["title"].(string), payload["body"].(string)
	action, operation := "create", "create-"+key
	if tool == factory.MCPToolRefineIssue {
		action, operation = "refine", "refine-"+submission
	}
	index, err := s.mutateIssueMakerIssue(ctx, scope, action, key, title, body, operation)
	if err != nil {
		return nil, err
	}
	return map[string]any{"issue": index, "key": key, "post_id": factory.IssueMakerReplyID(scope.Post, key), "parent_id": scope.Post}, nil
}
