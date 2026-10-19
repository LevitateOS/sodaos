package store

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
)

func TestLibraryMCPMaintenanceAndDiscovery(t *testing.T) {
	s, _ := postgresFixture(t, nil)
	ctx := context.Background()
	worker := activateMCPCapability(t, ctx, s, testMCPCapabilityRow())
	submit := func(cap factory.MCPCapability, tool factory.MCPTool, payload any) (factory.MCPReceipt, error) {
		raw, err := json.Marshal(payload)
		if err != nil {
			t.Fatal(err)
		}
		submission := testMCPSubmissionRow(cap, tool, raw)
		receipt, err := s.RecordMCPSubmission(ctx, submission, testMCPRecordedReceipt(submission))
		if err == nil {
			replay, e := s.RecordMCPSubmission(ctx, submission, testMCPRecordedReceipt(submission))
			if e != nil || replay != receipt {
				t.Fatalf("operation replay changed: %+v %v", replay, e)
			}
		}
		return receipt, err
	}
	learn := func(title string) factory.LibraryItem {
		receipt, err := submit(worker, factory.MCPToolLearned, map[string]any{"title": title, "summary": "Keep task ownership checks at the boundary.", "scope": "task access", "evidence_refs": []string{"src/access.go:12"}})
		if err != nil {
			t.Fatal(err)
		}
		out, err := s.LibraryRead(ctx, worker.Repository, receipt.LibraryItemID, false)
		if err != nil {
			t.Fatal(err)
		}
		if out.Item.SourcePost != receipt.FeedPostID || out.Item.Revision != 1 || out.Item.State != "observed" {
			t.Fatalf("lost learned identity: %+v", out)
		}
		return out.Item
	}
	a, b := learn("Task ownership"), learn("Task ownership duplicate")
	page, err := s.LibrarySearch(ctx, worker.Repository, "ownership", "knowledge", "", false)
	if err != nil || len(page.Items) != 2 {
		t.Fatalf("search=%+v %v", page, err)
	}
	foreign, err := s.LibrarySearch(ctx, worker.Repository+1, "", "", "", false)
	if err != nil || len(foreign.Items) != 0 {
		t.Fatalf("cross-project search=%+v %v", foreign, err)
	}
	if _, err = s.LibraryRead(ctx, worker.Repository+1, a.ID, false); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-project read=%v", err)
	}
	admin := testMCPCapabilityRow()
	admin.ID = strings.Repeat("d", 32)
	admin.AssignmentID = strings.Repeat("e", 32)
	admin.RunID = strings.Repeat("f", 32)
	admin.Profile = factory.MCPProfileLibraryAdmin
	admin = activateMCPCapability(t, ctx, s, admin)
	seen := map[string]bool{}
	apply := func(tool factory.MCPTool, change factory.LibraryChange) factory.MCPReceipt {
		t.Helper()
		receipt, err := submit(admin, tool, change)
		if err != nil {
			t.Fatalf("%s: %v", tool, err)
		}
		detail, err := s.FeedComment(ctx, receipt.FeedPostID)
		if err != nil {
			t.Fatal(err)
		}
		component := detail.Payload.Communication.ComponentID
		if component != factory.LibraryComponent(tool) || detail.Actor != "library_admin" {
			t.Fatalf("wrong maintenance post: %+v", detail)
		}
		seen[component] = true
		return receipt
	}
	apply(factory.MCPToolBeginLibraryMaintenance, factory.LibraryChange{Title: "Maintain task policy notes", Summary: "Consolidate duplicates and preserve sources."})
	edit := factory.LibraryChange{ID: a.ID, Revision: 1, Reason: "Clarify the lesson", Content: &factory.LibraryContent{Title: "Task ownership boundary", Body: "Check ownership before reading a task.", Scope: "task access"}}
	preview, err := s.LibraryPreview(ctx, admin, factory.MCPToolEditLibraryItem, edit)
	if err != nil || preview["applied"] != false {
		t.Fatalf("preview=%+v %v", preview, err)
	}
	original, _ := s.LibraryRead(ctx, worker.Repository, a.ID, false)
	if original.Item.Revision != 1 {
		t.Fatal("preview changed item")
	}
	apply(factory.MCPToolEditLibraryItem, edit)
	if _, err = submit(admin, factory.MCPToolEditLibraryItem, edit); err == nil {
		t.Fatal("stale revision accepted")
	}
	apply(factory.MCPToolConsolidateLibraryItems, factory.LibraryChange{Sources: []factory.LibraryRef{{ID: b.ID, Revision: 1}}, Target: &factory.LibraryRef{ID: a.ID, Revision: 2}, Reason: "Same task policy", Content: &factory.LibraryContent{Title: "Task ownership boundary", Body: "Check ownership at the task boundary.", Scope: "task access"}})
	if _, err = s.LibraryRead(ctx, worker.Repository, b.ID, false); !errors.Is(err, ErrNotFound) {
		t.Fatal("consolidated source remains in ordinary retrieval")
	}
	apply(factory.MCPToolRestoreLibraryItem, factory.LibraryChange{ID: b.ID, Revision: 2, Reason: "Keep the supporting observation visible"})
	apply(factory.MCPToolSplitLibraryItem, factory.LibraryChange{ID: a.ID, Revision: 3, Reason: "Separate read and write guidance", Items: []factory.LibraryContent{{Title: "Task read policy", Body: "Check ownership before reading.", Scope: "task access"}, {Title: "Task write policy", Body: "Check ownership before writing.", Scope: "task access"}}})
	split, _ := s.LibraryRead(ctx, worker.Repository, a.ID, true)
	if len(split.Item.Replacements) != 2 || len(split.History) != 4 {
		t.Fatalf("split lost history: %+v", split)
	}
	first, second := split.Item.Replacements[0], split.Item.Replacements[1]
	apply(factory.MCPToolRelateLibraryItems, factory.LibraryChange{ID: first, Revision: 1, Target: &factory.LibraryRef{ID: second, Revision: 1}, Relation: "explains", Reason: "Same ownership boundary", EvidenceRefs: []string{"src/access.go:12"}})
	apply(factory.MCPToolSupersedeLibraryItem, factory.LibraryChange{ID: first, Revision: 2, Target: &factory.LibraryRef{ID: second, Revision: 1}, Reason: "The second note includes the correction", EvidenceRefs: []string{"src/access.go:18"}})
	apply(factory.MCPToolRemoveLibraryItem, factory.LibraryChange{ID: second, Revision: 1, Reason: "Revisit evidence"})
	apply(factory.MCPToolRestoreLibraryItem, factory.LibraryChange{ID: second, Revision: 2, Reason: "Evidence checked"})
	apply(factory.MCPToolReportLibraryImpact, factory.LibraryChange{Title: "Affected work", Summary: "Task policy notes now have focused references.", Completed: []string{"Preserved all source links"}, Remaining: []string{"Recheck consuming plans"}})
	apply(factory.MCPToolFinishLibraryMaintenance, factory.LibraryChange{Title: "Library maintained", Summary: "Duplicate notes consolidated and focused guidance recorded."})
	if len(seen) != 9 {
		t.Fatalf("missing actual Library Admin components: %v", seen)
	}
	current, err := s.LibraryRead(ctx, worker.Repository, second, false)
	if err != nil || current.Item.Revision != 3 {
		t.Fatalf("restore=%+v %v", current, err)
	}
}

func TestLibraryDecisionAuthorityAndCreationAtomicity(t *testing.T) {
	s, _ := postgresFixture(t, nil)
	ctx := context.Background()
	cap := activateMCPCapability(t, ctx, s, testMCPCapabilityRow())
	decision := func(choice string) factory.LibraryItem {
		raw, _ := json.Marshal(map[string]any{"title": "Task visibility", "summary": "The task policy was recorded.", "scope": "task access", "decision": choice, "rationale": "Product request"})
		submission := testMCPSubmissionRow(cap, factory.MCPToolRecordDecision, raw)
		receipt, err := s.RecordMCPSubmission(ctx, submission, testMCPRecordedReceipt(submission))
		if err != nil {
			t.Fatal(err)
		}
		item, err := s.LibraryRead(ctx, cap.Repository, receipt.LibraryItemID, false)
		if err != nil {
			t.Fatal(err)
		}
		return item.Item
	}
	a, b := decision("Owners only"), decision("All team members")
	admin := cap
	admin.Profile = factory.MCPProfileLibraryAdmin
	bad := factory.LibraryChange{Sources: []factory.LibraryRef{{ID: b.ID, Revision: 1}}, Target: &factory.LibraryRef{ID: a.ID, Revision: 1}, Reason: "Pretend these decisions agree", Content: &factory.LibraryContent{Title: a.Title, Body: a.Body, Scope: a.Scope}}
	if _, err := s.LibraryPreview(ctx, admin, factory.MCPToolConsolidateLibraryItems, bad); err == nil {
		t.Fatal("conflicting decisions consolidated")
	}
	if _, err := s.LibraryPreview(ctx, admin, factory.MCPToolEditLibraryItem, factory.LibraryChange{ID: a.ID, Revision: 1, Reason: "Expand scope", Content: &factory.LibraryContent{Title: a.Title, Body: a.Body, Scope: "all applications"}}); err == nil {
		t.Fatal("decision scope rewritten")
	}
	before, _ := s.LibrarySearch(ctx, cap.Repository, "", "", "", false)
	raw := []byte(`{"title":"Unscoped learning","summary":"Missing source evidence","scope":"all tasks"}`)
	submission := testMCPSubmissionRow(cap, factory.MCPToolLearned, raw)
	if _, err := s.RecordMCPSubmission(ctx, submission, testMCPRecordedReceipt(submission)); err == nil {
		t.Fatal("invalid learning accepted")
	}
	after, _ := s.LibrarySearch(ctx, cap.Repository, "", "", "", false)
	if len(before.Items) != len(after.Items) {
		t.Fatal("rejected learning leaked library item")
	}
}
