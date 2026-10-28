package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/store"
	"github.com/levitateos/sodaos/internal/web/auth"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type factoryMCPLibrarySource interface {
	LibrarySearch(context.Context, int64, string, string, string, bool, bool) (factory.LibraryPage, error)
	LibraryRead(context.Context, int64, string, bool) (factory.LibraryRead, error)
}

func libraryToolDescription(tool factory.MCPTool) string {
	switch tool {
	case factory.MCPToolSearchLibrary:
		return "Search this project's reusable knowledge and recorded decisions. Use current_only for current evidence: excludes removed, superseded and retracted entries, while contested entries retain their unresolved state. Returns exact revisions, sources and a continuation cursor. Without current_only, superseded history remains discoverable."
	case factory.MCPToolReadLibraryItem:
		return "Read one project library item, its current revision and recent attributed revision history."
	}
	return ""
}

func libraryPayloadSchema(tool factory.MCPTool) *jsonschema.Schema {
	if !factory.MCPIsLibraryRead(tool) {
		return nil
	}
	text := func() *jsonschema.Schema { return &jsonschema.Schema{Type: "string"} }
	o := &jsonschema.Schema{Type: "object", Properties: map[string]*jsonschema.Schema{}, AdditionalProperties: &jsonschema.Schema{Not: &jsonschema.Schema{}}}
	if tool == factory.MCPToolReadLibraryItem {
		o.Properties["id"] = text()
		o.Required = []string{"id"}
		return o
	}
	for _, key := range []string{"query", "kind", "cursor"} {
		o.Properties[key] = text()
	}
	o.Properties["kind"].Enum = []any{"knowledge", "decision"}
	o.Properties["current_only"] = &jsonschema.Schema{Type: "boolean", Description: "Exclude removed, superseded and retracted entries before pagination. Contested records remain visible as unresolved evidence."}
	return o
}

func ptrInt(v int) *int { return &v }

func (s *factoryCatalogServer) serveLibraryRead(ctx context.Context, tool factory.MCPTool, args mcpCallArgs) (*mcp.CallToolResult, any, error) {
	cap, release, err := s.admitCall(ctx, tool, args)
	if err != nil {
		return nil, nil, err
	}
	defer release()
	source, ok := s.deps.Submissions.(factoryMCPLibrarySource)
	if !ok {
		return nil, nil, errors.New("library is unavailable")
	}
	var in struct {
		Query       string `json:"query"`
		Kind        string `json:"kind"`
		Cursor      string `json:"cursor"`
		ID          string `json:"id"`
		CurrentOnly bool   `json:"current_only"`
	}
	raw, err := json.Marshal(args.Payload)
	if err != nil {
		return nil, nil, err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(&in); err != nil {
		return nil, nil, err
	}
	if tool == factory.MCPToolReadLibraryItem {
		if !factory.ValidID(in.ID) {
			return nil, nil, errors.New("library item id required")
		}
		item, err := source.LibraryRead(ctx, cap.Repository, in.ID, false)
		if err != nil {
			return nil, nil, err
		}
		return mcpJSONResult(item)
	}
	page, err := source.LibrarySearch(ctx, cap.Repository, in.Query, in.Kind, in.Cursor, false, in.CurrentOnly)
	if err != nil {
		return nil, nil, err
	}
	return mcpJSONResult(page)
}

// HTTP browsing uses the same repository visibility owner as the project feed.
func (s *API) apiFactoryLibrary(w http.ResponseWriter, r *http.Request, v store.Session) {
	repository, ok := auth.PositiveID(r.URL.Query().Get("repository"))
	if !ok {
		auth.JSONError(w, 400, "invalid_request", "A project repository is required.")
		return
	}
	if _, err := s.visibleRepository(r, v, repository); err != nil {
		auth.JSONError(w, 404, "repository_not_found", "Repository is not visible.")
		return
	}
	page, err := s.Store.LibrarySearch(r.Context(), repository, r.URL.Query().Get("query"), r.URL.Query().Get("kind"), r.URL.Query().Get("cursor"), r.URL.Query().Get("history") == "true", r.URL.Query().Get("current_only") == "true")
	if err != nil {
		auth.JSONError(w, 400, "invalid_request", "Could not search the project library.")
		return
	}
	auth.JSONResponse(w, 200, page)
}

func (s *API) apiFactoryLibraryItem(w http.ResponseWriter, r *http.Request, v store.Session) {
	repository, ok := auth.PositiveID(r.URL.Query().Get("repository"))
	if !ok {
		auth.JSONError(w, 400, "invalid_request", "A project repository is required.")
		return
	}
	if _, err := s.visibleRepository(r, v, repository); err != nil {
		auth.JSONError(w, 404, "repository_not_found", "Repository is not visible.")
		return
	}
	item, err := s.Store.LibraryRead(r.Context(), repository, r.URL.Query().Get("id"), r.URL.Query().Get("history") == "true")
	if err != nil {
		auth.JSONError(w, 404, "item_not_found", "Library item is not available.")
		return
	}
	auth.JSONResponse(w, 200, item)
}
