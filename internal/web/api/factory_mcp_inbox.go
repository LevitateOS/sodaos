package api

import (
	"context"
	"errors"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/levitateos/sodaos/internal/factory"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type factoryMCPDiscussionSource interface {
	MCPInbox(context.Context, factory.MCPCapability, string) (factory.MCPFeedPage, error)
	MCPDiscussion(context.Context, factory.MCPCapability, string, string) (factory.MCPFeedPage, error)
}

func inboxPayloadSchema(tool factory.MCPTool) *jsonschema.Schema {
	if tool != factory.MCPToolInbox && tool != factory.MCPToolDiscussion {
		return nil
	}
	schema := &jsonschema.Schema{Type: "object", Properties: map[string]*jsonschema.Schema{"cursor": {Type: "string"}}, AdditionalProperties: &jsonschema.Schema{Not: &jsonschema.Schema{}}}
	if tool == factory.MCPToolDiscussion {
		schema.Properties["post_id"] = &jsonschema.Schema{Type: "string"}
		schema.Required = []string{"post_id"}
	}
	return schema
}

func (s *factoryCatalogServer) serveInbox(ctx context.Context, tool factory.MCPTool, args mcpCallArgs) (*mcp.CallToolResult, any, error) {
	cap, release, err := s.admitCall(ctx, tool, args)
	if err != nil {
		return nil, nil, err
	}
	defer release()
	source, ok := s.deps.Submissions.(factoryMCPDiscussionSource)
	if !ok {
		return nil, nil, errors.New("worker inbox is unavailable")
	}
	cursor, postID := "", ""
	for key, value := range args.Payload {
		text, ok := value.(string)
		if !ok {
			return nil, nil, errors.New("discussion parameters must be text")
		}
		switch key {
		case "cursor":
			cursor = text
		case "post_id":
			if tool != factory.MCPToolDiscussion {
				return nil, nil, errors.New("unknown inbox field")
			}
			postID = text
		default:
			return nil, nil, errors.New("unknown discussion field")
		}
	}
	var page factory.MCPFeedPage
	if tool == factory.MCPToolInbox {
		page, err = source.MCPInbox(ctx, cap, cursor)
	} else {
		page, err = source.MCPDiscussion(ctx, cap, postID, cursor)
	}
	if err != nil {
		return nil, nil, err
	}
	return mcpJSONResult(page)
}
