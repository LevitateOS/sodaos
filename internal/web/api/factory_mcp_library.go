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
	LibrarySearch(context.Context, int64, string, string, string, bool) (factory.LibraryPage, error)
	LibraryRead(context.Context, int64, string, bool) (factory.LibraryRead, error)
	LibraryPreview(context.Context, factory.MCPCapability, factory.MCPTool, factory.LibraryChange) (map[string]any, error)
}

func libraryToolDescription(tool factory.MCPTool) string {
	switch string(tool) {
	case "factory.search_library":
		return "Search this project's reusable knowledge and recorded decisions. Returns bounded current entries, sources and a continuation cursor; removed items are hidden."
	case "factory.read_library_item":
		return "Read one project library item, its current revision and recent attributed revision history. Use its exact revision when proposing maintenance."
	case "factory.library_inventory":
		return "Read a bounded project library inventory, including removed entries when requested. complete and next_cursor describe the returned coverage."
	case "factory.preview_library_changes":
		return "Preview an exact typed library operation against current revisions without applying it. Changed, missing, foreign or conflicting decision records refuse."
	case "factory.begin_library_maintenance":
		return "Post the maintenance objective and scope before working on this project's knowledge and decision records."
	case "factory.edit_library_item":
		return "Edit an exact knowledge or decision item revision and record the reason. Preserves decision identity, choice, authority and scope; emits the confirmed edit."
	case "factory.remove_library_item":
		return "Reversibly remove an exact item revision from ordinary library search and retrieval, preserving attributed history and references."
	case "factory.restore_library_item":
		return "Restore an exact removed item revision to library discovery. Superseded validity and replacement links remain unchanged."
	case "factory.consolidate_library_items":
		return "Consolidate exact source revisions into an existing target revision atomically, preserving sources and history. Conflicting decisions refuse."
	case "factory.split_library_item":
		return "Split an exact item revision into focused records, preserving original authority, evidence and provenance; the original links to every replacement."
	case "factory.relate_library_items":
		return "Add, replace or remove an evidenced relationship between exact revisions: contradicts, depends_on, duplicate_of or explains. Linking a conflict does not resolve it."
	case "factory.supersede_library_item":
		return "Mark an exact record superseded by an existing replacement with evidence. Decision replacements must come from the same role authority and scope."
	case "factory.report_library_impact":
		return "Post the known affected work and remaining follow-up from this maintenance assignment; does not modify any issue, plan or decision."
	case "factory.finish_library_maintenance":
		return "Finish this maintenance assignment with a summary and remaining work. Previously recorded library operations remain applied and are not replayed."
	}
	return ""
}

func libraryPayloadSchema(tool factory.MCPTool) *jsonschema.Schema {
	if !factory.MCPIsLibraryRead(tool) && !factory.MCPIsLibraryMutation(tool) {
		return nil
	}
	object := func() *jsonschema.Schema {
		return &jsonschema.Schema{Type: "object", Properties: map[string]*jsonschema.Schema{}, AdditionalProperties: &jsonschema.Schema{Not: &jsonschema.Schema{}}}
	}
	text := func() *jsonschema.Schema { return &jsonschema.Schema{Type: "string"} }
	texts := func() *jsonschema.Schema {
		return &jsonschema.Schema{Type: "array", MaxItems: ptrInt(24), Items: text()}
	}
	ref := func() *jsonschema.Schema {
		o := object()
		o.Properties["id"] = text()
		o.Properties["revision"] = &jsonschema.Schema{Type: "integer"}
		o.Required = []string{"id", "revision"}
		return o
	}
	content := func() *jsonschema.Schema {
		o := object()
		for _, key := range []string{"title", "body", "scope"} {
			o.Properties[key] = text()
		}
		o.Properties["evidence_refs"] = texts()
		o.Required = []string{"title", "body", "scope"}
		return o
	}
	o := object()
	if factory.MCPIsLibraryRead(tool) {
		switch string(tool) {
		case "factory.read_library_item":
			o.Properties["id"] = text()
			o.Required = []string{"id"}
		case "factory.preview_library_changes":
			o.Properties["tool"] = text()
			o.Properties["change"] = &jsonschema.Schema{Type: "object", Description: "The exact payload of the maintenance tool to preview; validates revisions and shows resulting items without writing."}
			o.Required = []string{"tool", "change"}
		default:
			for _, key := range []string{"query", "kind", "cursor"} {
				o.Properties[key] = text()
			}
			o.Properties["kind"].Enum = []any{"knowledge", "decision"}
		}
		if string(tool) != "factory.preview_library_changes" {
			o.Properties["include_removed"] = &jsonschema.Schema{Type: "boolean", Description: "Only Library Admin can include removed items."}
		}
		return o
	}
	for _, key := range []string{"component_id", "title", "summary", "reason", "id", "relation"} {
		o.Properties[key] = text()
	}
	o.Properties["revision"] = &jsonschema.Schema{Type: "integer"}
	o.Properties["content"] = content()
	o.Properties["target"] = ref()
	o.Properties["sources"] = &jsonschema.Schema{Type: "array", Items: ref(), MaxItems: ptrInt(24)}
	o.Properties["items"] = &jsonschema.Schema{Type: "array", Items: content(), MaxItems: ptrInt(24)}
	o.Properties["remove_relation"] = &jsonschema.Schema{Type: "boolean"}
	for _, key := range []string{"evidence_refs", "completed", "remaining"} {
		o.Properties[key] = texts()
	}
	switch factory.LibraryComponent(tool) {
	case "LA01", "LA08", "LA09":
		o.Required = []string{"title", "summary"}
	case "LA02":
		o.Required = []string{"id", "revision", "reason", "content"}
	case "LA03":
		o.Required = []string{"sources", "target", "reason", "content"}
	case "LA04":
		o.Required = []string{"id", "revision", "reason", "items"}
	case "LA05":
		o.Required = []string{"id", "revision", "reason"}
	case "LA06":
		o.Required = []string{"id", "revision", "reason", "target", "evidence_refs"}
	case "LA07":
		o.Required = []string{"id", "revision", "reason", "target", "relation"}
	}
	return o
}

func ptrInt(v int) *int { return &v }

func checkLibraryPayload(tool factory.MCPTool, payload map[string]any) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = factory.DecodeLibraryChange(tool, raw)
	return err
}

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
		Query          string          `json:"query"`
		Kind           string          `json:"kind"`
		Cursor         string          `json:"cursor"`
		ID             string          `json:"id"`
		IncludeRemoved bool            `json:"include_removed"`
		Tool           factory.MCPTool `json:"tool"`
		Change         json.RawMessage `json:"change"`
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
	if in.IncludeRemoved && string(cap.Profile) != "library_admin" {
		return nil, nil, errors.New("removed entries require Library Admin")
	}
	if string(tool) == "factory.preview_library_changes" {
		change, err := factory.DecodeLibraryChange(in.Tool, in.Change)
		if err != nil {
			return nil, nil, err
		}
		preview, err := source.LibraryPreview(ctx, cap, in.Tool, change)
		if err != nil {
			return nil, nil, err
		}
		return mcpJSONResult(preview)
	}
	if string(tool) == "factory.read_library_item" {
		if !factory.ValidID(in.ID) {
			return nil, nil, errors.New("library item id required")
		}
		item, err := source.LibraryRead(ctx, cap.Repository, in.ID, in.IncludeRemoved)
		if err != nil {
			return nil, nil, err
		}
		return mcpJSONResult(item)
	}
	page, err := source.LibrarySearch(ctx, cap.Repository, in.Query, in.Kind, in.Cursor, in.IncludeRemoved)
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
	page, err := s.Store.LibrarySearch(r.Context(), repository, r.URL.Query().Get("query"), r.URL.Query().Get("kind"), r.URL.Query().Get("cursor"), false)
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
	item, err := s.Store.LibraryRead(r.Context(), repository, r.URL.Query().Get("id"), false)
	if err != nil {
		auth.JSONError(w, 404, "item_not_found", "Library item is not available.")
		return
	}
	auth.JSONResponse(w, 200, item)
}
