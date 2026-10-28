package factory

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// LibraryReadGuidance is included in newly rendered phase prompts and the
// admitted context read. Live evidence never rewrites the accepted work packet.
const LibraryReadGuidance = "Before choosing an approach or implementing a change, read factory.search_library with payload {\"kind\":\"decision\",\"current_only\":true}; follow next_cursor until complete. Search relevant knowledge with payload {\"kind\":\"knowledge\",\"current_only\":true,\"query\":\"<terms from your admitted task>\"}, broadening the query when no evidence matches. Read relevant items through factory.read_library_item and retain their item ID, exact revision, source_post and evidence in your consultation and decision. current_only excludes removed, superseded and retracted entries but preserves contested items: treat those as unresolved evidence, not settled policy. These are live advisory reads, not new scope or approval; do not replace your frozen accepted inputs. If a current record conflicts with accepted scope, surface the exact conflict to its owner before adopting a change. Read newly maintained revisions for future decisions without silently rewriting prior work.\n"

// LibraryItem is a project-scoped observation or recorded decision. Its source
// identity and decision content survive maintenance; editing never applies work.
type LibraryItem struct {
	ID           string            `json:"id"`
	Repository   int64             `json:"repository"`
	Revision     int64             `json:"revision"`
	Kind         string            `json:"kind"`
	Title        string            `json:"title"`
	Body         string            `json:"body"`
	Scope        string            `json:"scope"`
	EvidenceRefs []string          `json:"evidence_refs"`
	Decision     string            `json:"decision,omitempty"`
	Rationale    string            `json:"rationale,omitempty"`
	Alternatives []string          `json:"alternatives,omitempty"`
	OwnerRole    string            `json:"owner_role"`
	Assignment   string            `json:"assignment"`
	SourcePost   string            `json:"source_post"`
	State        string            `json:"state"`
	Removed      bool              `json:"removed"`
	Replacements []string          `json:"replacements,omitempty"`
	Relations    []LibraryRelation `json:"relations,omitempty"`
	DerivedFrom  []string          `json:"derived_from,omitempty"`
	UpdatedAt    int64             `json:"updated_at"`
}
type LibraryRelation struct {
	Kind     string   `json:"kind"`
	Target   string   `json:"target"`
	Evidence []string `json:"evidence,omitempty"`
}
type LibraryRevision struct {
	Item       LibraryItem `json:"item"`
	Reason     string      `json:"reason"`
	Actor      string      `json:"actor"`
	Submission string      `json:"submission"`
}
type LibraryRef struct {
	ID       string `json:"id"`
	Revision int64  `json:"revision"`
}

// LibraryAffectedRef names one relation another item holds toward a
// changed item: the maintenance operation's blast radius. FromID may
// itself be a changed item when an operation links its own results.
type LibraryAffectedRef struct {
	ItemID   string `json:"item_id"`
	FromID   string `json:"from_id"`
	Relation string `json:"relation"`
}
type LibraryContent struct {
	Title        string   `json:"title"`
	Body         string   `json:"body"`
	Scope        string   `json:"scope"`
	EvidenceRefs []string `json:"evidence_refs,omitempty"`
}
type LibraryPage struct {
	Items      []LibraryItem `json:"items"`
	NextCursor string        `json:"next_cursor,omitempty"`
	Complete   bool          `json:"complete"`
}
type LibraryRead struct {
	Item    LibraryItem       `json:"item"`
	History []LibraryRevision `json:"history"`
}

// LibraryIncomingRef names one relation another inventoried item holds
// toward the keyed item.
type LibraryIncomingRef struct {
	FromID       string `json:"from_id"`
	FromRevision int64  `json:"from_revision"`
	Relation     string `json:"relation"`
}

// LibraryOrphanRef names a relation whose target is missing or removed
// from ordinary retrieval in this repository.
type LibraryOrphanRef struct {
	FromID   string `json:"from_id"`
	Relation string `json:"relation"`
	TargetID string `json:"target_id"`
}

// LibraryInventory is one bounded collection-scoped inventory page. Items
// carries the current entries in ID order; Revisions counts recorded
// history entries per returned item. Incoming, Duplicates, Contradictions
// and Orphans analyze exactly the inventoried items: page through while
// Complete is false for repository-wide coverage. Total names the scope
// size so a partial page is never mistaken for a complete inventory.
type LibraryInventory struct {
	Items          []LibraryItem                   `json:"items"`
	Revisions      map[string]int64                `json:"revisions,omitempty"`
	Incoming       map[string][]LibraryIncomingRef `json:"incoming,omitempty"`
	Duplicates     [][]string                      `json:"duplicates,omitempty"`
	Contradictions [][]string                      `json:"contradictions,omitempty"`
	Orphans        []LibraryOrphanRef              `json:"orphans,omitempty"`
	Total          int64                           `json:"total"`
	NextCursor     string                          `json:"next_cursor,omitempty"`
	Complete       bool                            `json:"complete"`
}

type LibraryChange struct {
	Title          string           `json:"title,omitempty"`
	Summary        string           `json:"summary,omitempty"`
	Reason         string           `json:"reason,omitempty"`
	ID             string           `json:"id,omitempty"`
	Revision       int64            `json:"revision,omitempty"`
	Content        *LibraryContent  `json:"content,omitempty"`
	Sources        []LibraryRef     `json:"sources,omitempty"`
	Target         *LibraryRef      `json:"target,omitempty"`
	Items          []LibraryContent `json:"items,omitempty"`
	Relation       string           `json:"relation,omitempty"`
	RemoveRelation bool             `json:"remove_relation,omitempty"`
	EvidenceRefs   []string         `json:"evidence_refs,omitempty"`
	Completed      []string         `json:"completed,omitempty"`
	Remaining      []string         `json:"remaining,omitempty"`

	// LegacyComponentID accepts the retired presentation ID and ignores it.
	LegacyComponentID string `json:"component_id,omitempty"`
}

func MCPIsLibraryRead(tool MCPTool) bool {
	switch string(tool) {
	case "factory.search_library", "factory.read_library_item", "factory.library_inventory", "factory.preview_library_changes":
		return true
	}
	return false
}
func MCPIsLibraryMutation(tool MCPTool) bool {
	switch string(tool) {
	case "factory.begin_library_maintenance", "factory.edit_library_item", "factory.consolidate_library_items", "factory.split_library_item", "factory.remove_library_item", "factory.restore_library_item", "factory.supersede_library_item", "factory.relate_library_items", "factory.report_library_impact", "factory.finish_library_maintenance":
		return true
	}
	return false
}
// LibraryOperationTitle names a library operation for its feed post when
// the report supplies no title of its own.
func LibraryOperationTitle(tool MCPTool) string {
	switch tool {
	case MCPToolBeginLibraryMaintenance:
		return "Maintenance scope"
	case MCPToolEditLibraryItem:
		return "Item edited"
	case MCPToolConsolidateLibraryItems:
		return "Consolidation"
	case MCPToolSplitLibraryItem:
		return "Split"
	case MCPToolRemoveLibraryItem, MCPToolRestoreLibraryItem:
		return "Removal and restoration"
	case MCPToolSupersedeLibraryItem:
		return "Replacement"
	case MCPToolRelateLibraryItems:
		return "Relationship and conflict"
	case MCPToolReportLibraryImpact:
		return "Affected work"
	case MCPToolFinishLibraryMaintenance:
		return "Maintenance result"
	}
	return ""
}

// libraryReport is true for the operations that only report: they change
// no item and need a title and summary instead of a reason.
func libraryReport(tool MCPTool) bool {
	return tool == MCPToolBeginLibraryMaintenance || tool == MCPToolReportLibraryImpact || tool == MCPToolFinishLibraryMaintenance
}

func DecodeLibraryChange(tool MCPTool, raw []byte) (LibraryChange, error) {
	var c LibraryChange
	if !MCPIsLibraryMutation(tool) {
		return c, errors.New("unknown library operation")
	}
	if len(raw) > MCPMaxSubmissionPayload {
		return c, errors.New("library operation exceeds 64 KiB")
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if err := d.Decode(&c); err != nil {
		return c, err
	}
	if len(c.Sources) > 24 || len(c.Items) > 24 || len(c.EvidenceRefs) > 24 || len(c.Completed) > 24 || len(c.Remaining) > 24 {
		return c, errors.New("library list exceeds 24 entries")
	}
	for _, v := range []string{c.Title, c.Summary, c.Reason} {
		if len(v) > 2048 {
			return c, errors.New("library report text exceeds 2048 bytes")
		}
	}
	for _, content := range append(c.Items, func() []LibraryContent {
		if c.Content != nil {
			return []LibraryContent{*c.Content}
		}
		return nil
	}()...) {
		if strings.TrimSpace(content.Title) == "" || strings.TrimSpace(content.Body) == "" || strings.TrimSpace(content.Scope) == "" || len(content.Title) > 160 || len(content.Body) > 8192 || len(content.Scope) > 2048 || len(content.EvidenceRefs) > 24 {
			return c, errors.New("library content needs bounded title, body and scope")
		}
	}
	if strings.TrimSpace(c.Reason) == "" && !libraryReport(tool) {
		return c, errors.New("library change needs a reason")
	}
	if c.ID != "" && (!ValidID(c.ID) || c.Revision < 1) {
		return c, errors.New("library item needs exact id and revision")
	}
	for _, ref := range c.Sources {
		if !ValidID(ref.ID) || ref.Revision < 1 {
			return c, errors.New("source needs exact id and revision")
		}
	}
	if c.Target != nil && (!ValidID(c.Target.ID) || c.Target.Revision < 1) {
		return c, errors.New("target needs exact id and revision")
	}
	switch tool {
	case MCPToolBeginLibraryMaintenance, MCPToolReportLibraryImpact, MCPToolFinishLibraryMaintenance:
		if strings.TrimSpace(c.Title) == "" || strings.TrimSpace(c.Summary) == "" {
			return c, errors.New("maintenance report needs title and summary")
		}
	case MCPToolEditLibraryItem:
		if c.ID == "" || c.Content == nil {
			return c, errors.New("edit needs item and content")
		}
	case MCPToolConsolidateLibraryItems:
		if len(c.Sources) < 1 || c.Target == nil || c.Content == nil {
			return c, errors.New("consolidation needs sources, target and content")
		}
	case MCPToolSplitLibraryItem:
		if c.ID == "" || len(c.Items) < 2 {
			return c, errors.New("split needs item and at least two parts")
		}
	case MCPToolRemoveLibraryItem, MCPToolRestoreLibraryItem:
		if c.ID == "" {
			return c, errors.New("removal/restoration needs item")
		}
	case MCPToolSupersedeLibraryItem:
		if c.ID == "" || c.Target == nil || len(c.EvidenceRefs) == 0 {
			return c, errors.New("replacement needs item, replacement and evidence")
		}
	case MCPToolRelateLibraryItems:
		if c.ID == "" || c.Target == nil {
			return c, errors.New("relationship needs both items")
		}
		switch c.Relation {
		case "contradicts", "depends_on", "duplicate_of", "explains":
		default:
			return c, fmt.Errorf("unknown relationship %q", c.Relation)
		}
	}
	return c, nil
}
