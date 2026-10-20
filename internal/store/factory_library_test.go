package store

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
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

// TestRecordDecisionSeparatesOwnerFromLogger proves the R32.5
// logging/authority distinction against real PostgreSQL: a worker
// records its own choice by leaving the owner blank, or references a
// choice made by an authorized owner; the revision actor always stays
// the logging worker either way.
func TestRecordDecisionSeparatesOwnerFromLogger(t *testing.T) {
	s, _ := postgresFixture(t, nil)
	ctx := context.Background()
	worker := activateMCPCapability(t, ctx, s, testMCPCapabilityRow())
	record := func(payload map[string]any) (factory.LibraryItem, factory.LibraryRevision, error) {
		raw, err := json.Marshal(payload)
		if err != nil {
			t.Fatal(err)
		}
		submission := testMCPSubmissionRow(worker, factory.MCPToolRecordDecision, raw)
		receipt, err := s.RecordMCPSubmission(ctx, submission, testMCPRecordedReceipt(submission))
		if err != nil {
			return factory.LibraryItem{}, factory.LibraryRevision{}, err
		}
		read, err := s.LibraryRead(ctx, worker.Repository, receipt.LibraryItemID, false)
		if err != nil {
			t.Fatal(err)
		}
		if len(read.History) != 1 {
			t.Fatalf("decision history=%d", len(read.History))
		}
		return read.Item, read.History[0], nil
	}
	base := map[string]any{"title": "Task visibility", "summary": "The task policy was recorded.", "scope": "task access", "decision": "Owners only", "rationale": "Product request"}
	own, revision, err := record(base)
	if err != nil {
		t.Fatalf("own choice: %v", err)
	}
	if own.OwnerRole != string(worker.Profile) {
		t.Fatalf("own choice owner=%q, want logger %q", own.OwnerRole, worker.Profile)
	}
	if revision.Actor != "worker:"+worker.AssignmentID {
		t.Fatalf("own choice actor=%q", revision.Actor)
	}
	referenced := map[string]any{"title": "Task visibility", "summary": "Controller approved owners-only.", "scope": "task access", "decision": "Owners only", "rationale": "Controller approval", "decision_owner": "controller"}
	other, orevision, err := record(referenced)
	if err != nil {
		t.Fatalf("referenced choice: %v", err)
	}
	if other.OwnerRole != "controller" {
		t.Fatalf("referenced owner=%q, want controller", other.OwnerRole)
	}
	if orevision.Actor != "worker:"+worker.AssignmentID {
		t.Fatalf("referenced actor=%q", orevision.Actor)
	}
	if other.ID == own.ID {
		t.Fatal("referenced choice reused the logger's identity")
	}
	for _, bad := range []struct {
		name    string
		payload map[string]any
	}{
		{"blank owner", map[string]any{"title": "T", "summary": "S", "scope": "task access", "decision": "D", "rationale": "R", "decision_owner": "   "}},
		{"long owner", map[string]any{"title": "T", "summary": "S", "scope": "task access", "decision": "D", "rationale": "R", "decision_owner": strings.Repeat("o", 161)}},
	} {
		if _, _, err := record(bad.payload); err == nil {
			t.Fatalf("%s admitted", bad.name)
		}
	}
	learned := map[string]any{"title": "T", "summary": "S", "scope": "task access", "evidence_refs": []string{"src/x.go:1"}, "decision_owner": "controller"}
	raw, _ := json.Marshal(learned)
	learnedSubmission := testMCPSubmissionRow(worker, factory.MCPToolLearned, raw)
	if _, err := s.RecordMCPSubmission(ctx, learnedSubmission, testMCPRecordedReceipt(learnedSubmission)); err == nil {
		t.Fatal("learned post named a decision owner")
	}
}

// TestLearnedAmendAndChallengeCAS proves the R32.4 knowledge
// lifecycle against real PostgreSQL: a learned entry amends under
// compare-and-swap with full revision history, a stale expected
// revision refuses instead of overwriting, a challenge marks the
// entry contested without touching its body, and a later amendment
// restores observed. One identity spans work, feed and retrieval.
func TestLearnedAmendAndChallengeCAS(t *testing.T) {
	s, _ := postgresFixture(t, nil)
	ctx := context.Background()
	worker := activateMCPCapability(t, ctx, s, testMCPCapabilityRow())
	submit := func(tool factory.MCPTool, payload map[string]any) (factory.MCPReceipt, error) {
		raw, err := json.Marshal(payload)
		if err != nil {
			t.Fatal(err)
		}
		submission := testMCPSubmissionRow(worker, tool, raw)
		return s.RecordMCPSubmission(ctx, submission, testMCPRecordedReceipt(submission))
	}
	learned := map[string]any{"title": "Task ownership", "summary": "Keep ownership checks at the boundary.", "scope": "task access", "evidence_refs": []string{"src/access.go:12"}}
	receipt, err := submit(factory.MCPToolLearned, learned)
	if err != nil {
		t.Fatalf("learn: %v", err)
	}
	id := receipt.LibraryItemID
	read := func() factory.LibraryItem {
		out, err := s.LibraryRead(ctx, worker.Repository, id, false)
		if err != nil {
			t.Fatal(err)
		}
		return out.Item
	}
	if item := read(); item.Revision != 1 || item.State != "observed" {
		t.Fatalf("created: %+v", item)
	}
	amend := map[string]any{"title": "Task ownership", "summary": "Keep ownership checks at the boundary and in review.", "scope": "task access", "evidence_refs": []string{"src/access.go:12", "src/review.go:3"}, "revises_item": id, "expected_revision": 1}
	areceipt, err := submit(factory.MCPToolLearned, amend)
	if err != nil {
		t.Fatalf("amend: %v", err)
	}
	if areceipt.LibraryItemID != id || areceipt.FeedPostID == "" {
		t.Fatalf("amend lost identity: %+v", areceipt)
	}
	if item := read(); item.Revision != 2 || item.State != "observed" || !strings.Contains(item.Body, "and in review") {
		t.Fatalf("amended: %+v", item)
	}
	if _, err := submit(factory.MCPToolLearned, amend); err == nil {
		t.Fatal("stale amendment overwrote")
	}
	challenge := map[string]any{"title": "Ownership check disputed", "summary": "Review already covers this.", "scope": "task access", "evidence_refs": []string{"src/review.go:3"}, "revises_item": id, "expected_revision": 2, "challenge": true}
	if _, err := submit(factory.MCPToolLearned, challenge); err != nil {
		t.Fatalf("challenge: %v", err)
	}
	if item := read(); item.Revision != 3 || item.State != "contested" || !strings.Contains(item.Body, "and in review") {
		t.Fatalf("challenged: %+v", item)
	}
	restore := map[string]any{"title": "Task ownership", "summary": "Boundary plus review, confirmed.", "scope": "task access", "evidence_refs": []string{"src/access.go:12"}, "revises_item": id, "expected_revision": 3}
	if _, err := submit(factory.MCPToolLearned, restore); err != nil {
		t.Fatalf("restore: %v", err)
	}
	out, err := s.LibraryRead(ctx, worker.Repository, id, false)
	if err != nil {
		t.Fatal(err)
	}
	if out.Item.Revision != 4 || out.Item.State != "observed" || len(out.History) != 4 {
		t.Fatalf("restored: %+v history=%d", out.Item, len(out.History))
	}
	badID := map[string]any{"title": "T", "summary": "S", "scope": "task access", "evidence_refs": []string{"src/x.go:1"}, "revises_item": strings.Repeat("f", 32), "expected_revision": 1}
	if _, err := submit(factory.MCPToolLearned, badID); err == nil {
		t.Fatal("unknown entry amended")
	}
	noRev := map[string]any{"title": "T", "summary": "S", "scope": "task access", "evidence_refs": []string{"src/x.go:1"}, "revises_item": id}
	if _, err := submit(factory.MCPToolLearned, noRev); err == nil {
		t.Fatal("amendment without revision admitted")
	}
	bare := map[string]any{"title": "T", "summary": "S", "scope": "task access", "evidence_refs": []string{"src/x.go:1"}, "challenge": true}
	if _, err := submit(factory.MCPToolLearned, bare); err == nil {
		t.Fatal("challenge without target admitted")
	}
	flag := map[string]any{"title": "T", "summary": "S", "severity": "info", "revises_item": id, "expected_revision": 4}
	if _, err := submit(factory.MCPToolFlag, flag); err == nil {
		t.Fatal("non-learned amendment admitted")
	}
}

// TestLibraryInventoryReadsReferencesAndCompleteness proves the R32.7
// inventory read against real PostgreSQL: one bounded page names its
// scope total and completeness, counts recorded history per item, and
// analyzes exactly the inventoried items — duplicate title candidates,
// contradicts pairs, incoming references and relations dangling at a
// removed target. Stale inputs refuse instead of returning a silently
// partial page.
func TestLibraryInventoryReadsReferencesAndCompleteness(t *testing.T) {
	s, _ := postgresFixture(t, nil)
	ctx := context.Background()
	worker := activateMCPCapability(t, ctx, s, testMCPCapabilityRow())
	admin := testMCPCapabilityRow()
	admin.ID = strings.Repeat("d", 32)
	admin.AssignmentID = strings.Repeat("e", 32)
	admin.RunID = strings.Repeat("f", 32)
	admin.Profile = factory.MCPProfileLibraryAdmin
	admin = activateMCPCapability(t, ctx, s, admin)
	submit := func(cap factory.MCPCapability, tool factory.MCPTool, payload any) factory.MCPReceipt {
		t.Helper()
		raw, err := json.Marshal(payload)
		if err != nil {
			t.Fatal(err)
		}
		submission := testMCPSubmissionRow(cap, tool, raw)
		receipt, err := s.RecordMCPSubmission(ctx, submission, testMCPRecordedReceipt(submission))
		if err != nil {
			t.Fatalf("%s: %v", tool, err)
		}
		return receipt
	}
	learn := func(title, body string) string {
		t.Helper()
		receipt := submit(worker, factory.MCPToolLearned, map[string]any{"title": title, "summary": body, "scope": "task access", "evidence_refs": []string{"src/access.go:12"}})
		if receipt.LibraryItemID == "" {
			t.Fatal("learned entry recorded no item")
		}
		return receipt.LibraryItemID
	}
	a := learn("Task ownership boundary", "Check ownership before reading a task.")
	b := learn("Task ownership boundary", "Check ownership before writing a task.")
	c := learn("Unrelated note", "Nothing to do with ownership.")
	submit(admin, factory.MCPToolRelateLibraryItems, factory.LibraryChange{ID: a, Revision: 1, Target: &factory.LibraryRef{ID: b, Revision: 1}, Relation: "explains", Reason: "Same ownership boundary", EvidenceRefs: []string{"src/access.go:12"}})
	submit(admin, factory.MCPToolRelateLibraryItems, factory.LibraryChange{ID: a, Revision: 2, Target: &factory.LibraryRef{ID: c, Revision: 1}, Relation: "contradicts", Reason: "Ownership applies after all", EvidenceRefs: []string{"src/access.go:12"}})
	submit(admin, factory.MCPToolRemoveLibraryItem, factory.LibraryChange{ID: b, Revision: 1, Reason: "Revisit evidence"})
	sortedPair := func(x, y string) []string {
		if x > y {
			return []string{y, x}
		}
		return []string{x, y}
	}
	inv, err := s.LibraryInventory(ctx, worker.Repository, "knowledge", "", false)
	if err != nil {
		t.Fatal(err)
	}
	if inv.Total != 2 || !inv.Complete || inv.NextCursor != "" || len(inv.Items) != 2 {
		t.Fatalf("default inventory hid its scope: %+v", inv)
	}
	if inv.Revisions[a] != 3 {
		t.Fatalf("inventory lost history depth: %+v", inv.Revisions)
	}
	if len(inv.Duplicates) != 0 {
		t.Fatalf("partial page reported duplicates outside its coverage: %+v", inv.Duplicates)
	}
	want := sortedPair(a, c)
	found := false
	for _, pair := range inv.Contradictions {
		if len(pair) == 2 && pair[0] == want[0] && pair[1] == want[1] {
			found = true
		}
	}
	if !found {
		t.Fatalf("inventory missed contradictions: %+v", inv.Contradictions)
	}
	refs := inv.Incoming[c]
	if len(refs) != 1 || refs[0].FromID != a || refs[0].Relation != "contradicts" {
		t.Fatalf("inventory lost incoming references: %+v", inv.Incoming)
	}
	if _, ok := inv.Incoming[b]; ok {
		t.Fatalf("inventory reported incoming for an item outside the page: %+v", inv.Incoming)
	}
	if len(inv.Orphans) != 1 || inv.Orphans[0].FromID != a || inv.Orphans[0].TargetID != b || inv.Orphans[0].Relation != "explains" {
		t.Fatalf("inventory missed the removed-target orphan: %+v", inv.Orphans)
	}
	full, err := s.LibraryInventory(ctx, worker.Repository, "", "", true)
	if err != nil {
		t.Fatal(err)
	}
	if full.Total != 3 || !full.Complete || len(full.Items) != 3 {
		t.Fatalf("unfiltered inventory hid its scope: %+v", full)
	}
	wantFull := sortedPair(a, b)
	foundFull := false
	for _, pair := range full.Duplicates {
		if len(pair) == 2 && pair[0] == wantFull[0] && pair[1] == wantFull[1] {
			foundFull = true
		}
	}
	if !foundFull {
		t.Fatalf("full inventory missed duplicate candidates: %+v", full.Duplicates)
	}
	if refs := full.Incoming[b]; len(refs) != 1 || refs[0].FromID != a || refs[0].Relation != "explains" {
		t.Fatalf("full inventory lost the explains reference: %+v", full.Incoming)
	}
	if _, err := s.LibraryInventory(ctx, worker.Repository, "bogus", "", false); err == nil {
		t.Fatal("bogus kind inventoried")
	}
	if _, err := s.LibraryInventory(ctx, worker.Repository, "", "nope", false); err == nil {
		t.Fatal("bogus cursor inventoried")
	}
	if _, err := s.LibraryInventory(ctx, 0, "", "", false); err == nil {
		t.Fatal("bogus repository inventoried")
	}
}

// TestLibraryDecisionEditRemoveRestore proves the R32.8 lifecycle for
// decision representations against real PostgreSQL: an edit revises
// the presentation while the recorded choice, owner, rationale and
// scope survive; a stale edit refuses; remove hides the decision from
// ordinary retrieval while keeping its relations and history, with the
// receipt naming the removed revision for recovery; restore brings the
// same decision back with its authority intact and a full history.
func TestLibraryDecisionEditRemoveRestore(t *testing.T) {
	s, _ := postgresFixture(t, nil)
	ctx := context.Background()
	worker := activateMCPCapability(t, ctx, s, testMCPCapabilityRow())
	admin := testMCPCapabilityRow()
	admin.ID = strings.Repeat("d", 32)
	admin.AssignmentID = strings.Repeat("e", 32)
	admin.RunID = strings.Repeat("f", 32)
	admin.Profile = factory.MCPProfileLibraryAdmin
	admin = activateMCPCapability(t, ctx, s, admin)
	record := func(payload map[string]any) factory.MCPReceipt {
		t.Helper()
		raw, err := json.Marshal(payload)
		if err != nil {
			t.Fatal(err)
		}
		submission := testMCPSubmissionRow(worker, factory.MCPToolRecordDecision, raw)
		receipt, err := s.RecordMCPSubmission(ctx, submission, testMCPRecordedReceipt(submission))
		if err != nil {
			t.Fatalf("record decision: %v", err)
		}
		return receipt
	}
	apply := func(tool factory.MCPTool, change factory.LibraryChange) factory.MCPReceipt {
		t.Helper()
		raw, err := json.Marshal(change)
		if err != nil {
			t.Fatal(err)
		}
		submission := testMCPSubmissionRow(admin, tool, raw)
		receipt, err := s.RecordMCPSubmission(ctx, submission, testMCPRecordedReceipt(submission))
		if err != nil {
			t.Fatalf("%s: %v", tool, err)
		}
		return receipt
	}
	receipt := record(map[string]any{"title": "Task visibility", "summary": "The task policy was recorded.", "scope": "task access", "decision": "Owners only", "rationale": "Product request"})
	id := receipt.LibraryItemID
	edit := factory.LibraryChange{ID: id, Revision: 1, Reason: "Clarify the title", Content: &factory.LibraryContent{Title: "Task visibility rule", Body: "The task policy was recorded.", Scope: "task access"}}
	apply(factory.MCPToolEditLibraryItem, edit)
	if _, err := s.RecordMCPSubmission(ctx, testMCPSubmissionRow(admin, factory.MCPToolEditLibraryItem, mustMarshal(t, edit)), testMCPRecordedReceipt(testMCPSubmissionRow(admin, factory.MCPToolEditLibraryItem, mustMarshal(t, edit)))); err == nil {
		t.Fatal("stale decision edit accepted")
	}
	edited, err := s.LibraryRead(ctx, worker.Repository, id, false)
	if err != nil {
		t.Fatal(err)
	}
	if edited.Item.Revision != 2 || edited.Item.Title != "Task visibility rule" || edited.Item.Decision != "Owners only" || edited.Item.OwnerRole != string(worker.Profile) || edited.Item.Rationale != "Product request" || edited.Item.Scope != "task access" || len(edited.History) != 2 {
		t.Fatalf("decision edit lost authority: %+v", edited.Item)
	}
	removeReceipt := apply(factory.MCPToolRemoveLibraryItem, factory.LibraryChange{ID: id, Revision: 2, Reason: "Revisit evidence"})
	if removeReceipt.LibraryItemID != id || removeReceipt.LibraryRevision != 3 {
		t.Fatalf("remove receipt not recoverable: %+v", removeReceipt)
	}
	if _, err := s.LibraryRead(ctx, worker.Repository, id, false); !errors.Is(err, ErrNotFound) {
		t.Fatalf("removed decision still retrievable: %v", err)
	}
	hidden, err := s.LibraryRead(ctx, worker.Repository, id, true)
	if err != nil || !hidden.Item.Removed || hidden.Item.Decision != "Owners only" || len(hidden.History) != 3 {
		t.Fatalf("removed decision lost history: %+v %v", hidden, err)
	}
	apply(factory.MCPToolRestoreLibraryItem, factory.LibraryChange{ID: id, Revision: 3, Reason: "Evidence checked"})
	restored, err := s.LibraryRead(ctx, worker.Repository, id, false)
	if err != nil {
		t.Fatal(err)
	}
	if restored.Item.Removed || restored.Item.Revision != 4 || restored.Item.Decision != "Owners only" || restored.Item.OwnerRole != string(worker.Profile) || len(restored.History) != 4 {
		t.Fatalf("restore lost decision authority: %+v", restored.Item)
	}
}

func mustMarshal(t *testing.T, v any) []byte {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

// TestLibraryConsolidateSplitFinishAtomicity proves the R32.9 finish
// path against real PostgreSQL: revoked (removed) sources refuse
// further operation, restoring an original split into replacements
// keeps its replacement mapping with full history, and the
// maintenance finish durably posts its confirmed operations plus
// unresolved cases on the feed.
func TestLibraryConsolidateSplitFinishAtomicity(t *testing.T) {
	s, _ := postgresFixture(t, nil)
	ctx := context.Background()
	worker := activateMCPCapability(t, ctx, s, testMCPCapabilityRow())
	admin := testMCPCapabilityRow()
	admin.ID = strings.Repeat("d", 32)
	admin.AssignmentID = strings.Repeat("e", 32)
	admin.RunID = strings.Repeat("f", 32)
	admin.Profile = factory.MCPProfileLibraryAdmin
	admin = activateMCPCapability(t, ctx, s, admin)
	learn := func(title string) string {
		t.Helper()
		raw, err := json.Marshal(map[string]any{"title": title, "summary": "Boundary note.", "scope": "task access", "evidence_refs": []string{"src/access.go:12"}})
		if err != nil {
			t.Fatal(err)
		}
		submission := testMCPSubmissionRow(worker, factory.MCPToolLearned, raw)
		receipt, err := s.RecordMCPSubmission(ctx, submission, testMCPRecordedReceipt(submission))
		if err != nil {
			t.Fatalf("learn: %v", err)
		}
		return receipt.LibraryItemID
	}
	apply := func(tool factory.MCPTool, change factory.LibraryChange) factory.MCPReceipt {
		t.Helper()
		raw, err := json.Marshal(change)
		if err != nil {
			t.Fatal(err)
		}
		submission := testMCPSubmissionRow(admin, tool, raw)
		receipt, err := s.RecordMCPSubmission(ctx, submission, testMCPRecordedReceipt(submission))
		if err != nil {
			t.Fatalf("%s: %v", tool, err)
		}
		return receipt
	}
	refuse := func(tool factory.MCPTool, change factory.LibraryChange) {
		t.Helper()
		raw, err := json.Marshal(change)
		if err != nil {
			t.Fatal(err)
		}
		submission := testMCPSubmissionRow(admin, tool, raw)
		if _, err := s.RecordMCPSubmission(ctx, submission, testMCPRecordedReceipt(submission)); err == nil {
			t.Fatalf("%s on revoked source accepted", tool)
		}
	}
	a := learn("Task ownership boundary")
	b := learn("Task write policy")
	apply(factory.MCPToolRemoveLibraryItem, factory.LibraryChange{ID: b, Revision: 1, Reason: "Revisit evidence"})
	refuse(factory.MCPToolEditLibraryItem, factory.LibraryChange{ID: b, Revision: 2, Reason: "Edit a removed item", Content: &factory.LibraryContent{Title: "Task write policy", Body: "Changed.", Scope: "task access"}})
	refuse(factory.MCPToolRelateLibraryItems, factory.LibraryChange{ID: a, Revision: 1, Target: &factory.LibraryRef{ID: b, Revision: 2}, Relation: "explains", Reason: "Link to a removed item", EvidenceRefs: []string{"src/access.go:12"}})
	splitReceipt := apply(factory.MCPToolSplitLibraryItem, factory.LibraryChange{ID: a, Revision: 1, Reason: "Separate read and write guidance", Items: []factory.LibraryContent{{Title: "Task read policy", Body: "Check ownership before reading.", Scope: "task access"}, {Title: "Task write policy", Body: "Check ownership before writing.", Scope: "task access"}}})
	split, err := s.LibraryRead(ctx, worker.Repository, a, true)
	if err != nil {
		t.Fatal(err)
	}
	if !split.Item.Removed || len(split.Item.Replacements) != 2 || len(split.History) != 2 {
		t.Fatalf("split lost its mapping: %+v", split.Item)
	}
	if splitReceipt.LibraryItemID != split.Item.Replacements[0] && splitReceipt.LibraryItemID != split.Item.Replacements[1] {
		t.Fatalf("split receipt names no replacement: %+v", splitReceipt)
	}
	apply(factory.MCPToolRestoreLibraryItem, factory.LibraryChange{ID: a, Revision: 2, Reason: "Keep the original visible too"})
	restored, err := s.LibraryRead(ctx, worker.Repository, a, false)
	if err != nil {
		t.Fatal(err)
	}
	if restored.Item.Removed || len(restored.Item.Replacements) != 2 || restored.Item.Revision != 3 || len(restored.History) != 3 {
		t.Fatalf("restore against replacements lost history: %+v", restored.Item)
	}
	finishReceipt := apply(factory.MCPToolFinishLibraryMaintenance, factory.LibraryChange{Title: "Library maintained", Summary: "Split focused guidance; one item still under review.", Completed: []string{"Split ownership boundary into read/write records"}, Remaining: []string{"Recheck the removed write policy"}})
	detail, err := s.FeedComment(ctx, finishReceipt.FeedPostID)
	if err != nil {
		t.Fatal(err)
	}
	posted := detail.Payload.Communication
	if len(posted.Completed) != 1 || posted.Completed[0] != "Split ownership boundary into read/write records" || len(posted.Remaining) != 1 || posted.Remaining[0] != "Recheck the removed write policy" {
		t.Fatalf("finish lost confirmed/unresolved cases: %+v", posted)
	}
}

func researchPromptSHA() string {
	sum := sha256.Sum256([]byte("research prompt"))
	return hex.EncodeToString(sum[:])
}

// TestResearchRecordsAndScopeCoverage proves the R32.13 progressive
// research interface against real PostgreSQL: a researcher declares
// questions and saves findings, experiments and sources across
// batches; research_scope serves the admitted subject with
// per-question answered coverage; findings citing undeclared
// questions, empty batches and non-researcher recorders refuse; a
// foreign assignment serves no scope.
func TestResearchRecordsAndScopeCoverage(t *testing.T) {
	s, _ := postgresFixture(t, nil)
	ctx := context.Background()
	assignmentID, runID := factory.NewID(), factory.NewID()
	acceptance := "d" + strings.Repeat("a", 24)
	a := factory.Assignment{
		Subject: factory.ExecutionSubject{Kind: factory.SubjectResearch, Research: &factory.ResearchSubject{Issue: 7, Acceptance: acceptance, Sources: []string{"ops.runbooks"}}},
		Prompt:  []byte("research prompt"), RunHistory: []string{runID},
		ID: assignmentID, AttemptRoot: assignmentID,
		ProjectID: "p" + strings.Repeat("1", 24), Role: project.RoleResearcher, Phase: project.ExecutionResearch,
		Acceptance: acceptance, Preparation: "f" + strings.Repeat("d", 24),
		Harness: "codex-1.2.3", HarnessVers: "1.2.3", Model: "test-model", Connection: "conn",
		SourceCommit: strings.Repeat("c", 40), PromptSHA: researchPromptSHA(),
		Run: runID, Stage: factory.AssignmentAssigned, Attempts: 1,
		Repository: 7, Issue: 7, Revision: 0, NativeRev: 1, CreatedUnix: 1700000000, ActorID: 7,
	}
	if err := a.Validate(); err != nil {
		t.Fatalf("research assignment invalid: %v", err)
	}
	raw, err := json.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.ExecContext(ctx, `INSERT INTO factory_assignments(id,repository,issue,run,stage,revision,data) VALUES($1,$2,$3,$4,$5,$6,$7)`,
		a.ID, a.Repository, a.Issue, a.Run, a.Stage, a.Revision, string(raw)); err != nil {
		t.Fatalf("seed assignment: %v", err)
	}
	researcher := testMCPCapabilityRow()
	researcher.AssignmentID = assignmentID
	researcher.Profile = factory.MCPProfileResearcher
	researcher = activateMCPCapability(t, ctx, s, researcher)
	record := func(cap factory.MCPCapability, payload map[string]any) error {
		t.Helper()
		raw, err := json.Marshal(payload)
		if err != nil {
			t.Fatal(err)
		}
		submission := testMCPSubmissionRow(cap, factory.MCPToolRecordResearch, raw)
		_, err = s.RecordMCPSubmission(ctx, submission, testMCPRecordedReceipt(submission))
		return err
	}
	scope, err := s.ResearchScope(ctx, researcher)
	if err != nil {
		t.Fatal(err)
	}
	if scope.Issue != 7 || scope.Acceptance != acceptance || len(scope.Questions) != 0 || scope.Records != 0 {
		t.Fatalf("empty scope invented coverage: %+v", scope)
	}
	first := map[string]any{
		"questions":   []string{"What restarts the worker?", "What supervises it?"},
		"findings":    []any{map[string]any{"question": 0, "finding": "The supervisor restarts it.", "evidence": []string{"ops.runbooks#3"}}},
		"experiments": []string{"Restarted the worker twice"},
		"sources":     []string{"ops.runbooks#3"},
	}
	if err := record(researcher, first); err != nil {
		t.Fatalf("record batch: %v", err)
	}
	second := map[string]any{
		"findings":   []any{map[string]any{"question": 1, "finding": "The controller supervises it.", "evidence": []string{"ops.runbooks#7"}}},
		"unanswered": []int{1},
	}
	if err := record(researcher, second); err != nil {
		t.Fatalf("record findings: %v", err)
	}
	scope, err = s.ResearchScope(ctx, researcher)
	if err != nil {
		t.Fatal(err)
	}
	if scope.Records != 2 || len(scope.Questions) != 2 {
		t.Fatalf("scope lost batches: %+v", scope)
	}
	if !scope.Questions[0].Answered || scope.Questions[0].Findings != 1 || !scope.Questions[1].Answered {
		t.Fatalf("scope misreported coverage: %+v", scope.Questions)
	}
	if len(scope.Unanswered) != 1 || scope.Unanswered[0] != "What supervises it?" {
		t.Fatalf("scope lost unanswered questions: %+v", scope.Unanswered)
	}
	if len(scope.Findings) != 2 || len(scope.Experiments) != 1 {
		t.Fatalf("scope lost records: %+v", scope)
	}
	bad := map[string]any{"findings": []any{map[string]any{"question": 5, "finding": "Nothing cites this.", "evidence": []string{"x"}}}}
	if err := record(researcher, bad); err == nil {
		t.Fatal("finding on undeclared question accepted")
	}
	if err := record(researcher, map[string]any{}); err == nil {
		t.Fatal("empty record accepted")
	}
	intruder := testMCPCapabilityRow()
	intruder.AssignmentID = assignmentID
	intruder.Profile = factory.MCPProfileBuilder
	intruder = activateMCPCapability(t, ctx, s, intruder)
	if err := record(intruder, first); err == nil {
		t.Fatal("non-researcher recorded research")
	}
	if _, err := s.ResearchScope(ctx, intruder); err == nil {
		t.Fatal("non-researcher read research scope")
	}
	scope, err = s.ResearchScope(ctx, researcher)
	if err != nil {
		t.Fatal(err)
	}
	if scope.Records != 2 {
		t.Fatalf("refused batches polluted the log: %+v", scope)
	}
}

// TestLibraryWorkerRetrievalSurfacesConflictAndSupersession proves the
// R31.3 worker retrieval properties against real PostgreSQL: search
// and read serve the same authorized records with state (observed,
// contested, superseded), provenance and relations intact; removed
// entries stay out of ordinary search; another repository sees
// nothing; read carries the attributed history.
func TestLibraryWorkerRetrievalSurfacesConflictAndSupersession(t *testing.T) {
	s, _ := postgresFixture(t, nil)
	ctx := context.Background()
	worker := activateMCPCapability(t, ctx, s, testMCPCapabilityRow())
	admin := testMCPCapabilityRow()
	admin.ID = strings.Repeat("d", 32)
	admin.AssignmentID = strings.Repeat("e", 32)
	admin.RunID = strings.Repeat("f", 32)
	admin.Profile = factory.MCPProfileLibraryAdmin
	admin = activateMCPCapability(t, ctx, s, admin)
	submit := func(cap factory.MCPCapability, tool factory.MCPTool, payload any) factory.MCPReceipt {
		t.Helper()
		raw, err := json.Marshal(payload)
		if err != nil {
			t.Fatal(err)
		}
		submission := testMCPSubmissionRow(cap, tool, raw)
		receipt, err := s.RecordMCPSubmission(ctx, submission, testMCPRecordedReceipt(submission))
		if err != nil {
			t.Fatalf("%s: %v", tool, err)
		}
		return receipt
	}
	learn := func(title string) string {
		t.Helper()
		return submit(worker, factory.MCPToolLearned, map[string]any{"title": title, "summary": "Boundary note.", "scope": "task access", "evidence_refs": []string{"src/access.go:12"}}).LibraryItemID
	}
	a := learn("Task ownership boundary")
	b := learn("Task ownership evidence")
	challenge := map[string]any{"title": "Ownership check disputed", "summary": "Review already covers this.", "scope": "task access", "evidence_refs": []string{"src/review.go:3"}, "revises_item": a, "expected_revision": 1, "challenge": true}
	submit(worker, factory.MCPToolLearned, challenge)
	submit(admin, factory.MCPToolSupersedeLibraryItem, factory.LibraryChange{ID: b, Revision: 1, Target: &factory.LibraryRef{ID: a, Revision: 2}, Reason: "The boundary note carries the correction", EvidenceRefs: []string{"src/access.go:18"}})
	page, err := s.LibrarySearch(ctx, worker.Repository, "", "knowledge", "", false)
	if err != nil {
		t.Fatal(err)
	}
	if !page.Complete || len(page.Items) != 2 {
		t.Fatalf("worker search hid records: %+v", page)
	}
	byID := map[string]factory.LibraryItem{}
	for _, item := range page.Items {
		byID[item.ID] = item
		if item.OwnerRole == "" || item.Assignment == "" || item.SourcePost == "" || item.EvidenceRefs == nil {
			t.Fatalf("search lost provenance: %+v", item)
		}
	}
	if byID[a].State != "contested" {
		t.Fatalf("search hid the conflict state: %+v", byID[a])
	}
	if byID[b].State != "superseded" || len(byID[b].Replacements) != 1 || byID[b].Replacements[0] != a {
		t.Fatalf("search hid the supersession: %+v", byID[b])
	}
	submit(admin, factory.MCPToolRemoveLibraryItem, factory.LibraryChange{ID: b, Revision: 2, Reason: "Revisit evidence"})
	hidden, err := s.LibrarySearch(ctx, worker.Repository, "", "knowledge", "", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(hidden.Items) != 1 || hidden.Items[0].ID != a {
		t.Fatalf("removed entry leaked into search: %+v", hidden)
	}
	foreign, err := s.LibrarySearch(ctx, worker.Repository+1, "", "", "", true)
	if err != nil {
		t.Fatal(err)
	}
	if len(foreign.Items) != 0 || !foreign.Complete {
		t.Fatalf("cross-repository leakage: %+v", foreign)
	}
	if _, err := s.LibraryRead(ctx, worker.Repository+1, a, true); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-repository read admitted: %v", err)
	}
	read, err := s.LibraryRead(ctx, worker.Repository, a, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(read.History) != 2 || read.History[0].Actor == "" {
		t.Fatalf("read lost attributed history: %+v", read)
	}
	removed, err := s.LibraryRead(ctx, worker.Repository, b, true)
	if err != nil || !removed.Item.Removed || removed.Item.State != "superseded" {
		t.Fatalf("privileged read lost the removed record: %+v %v", removed, err)
	}
}

// TestLibraryMaintenancePostCarriesMappingsAndAffectedRefs proves the
// R42.4 maintenance projection against real PostgreSQL: the feed post
// for an atomic multi-item operation names every resulting item with
// its revision (not just the first), and carries the affected
// references other items hold toward the changed items.
func TestLibraryMaintenancePostCarriesMappingsAndAffectedRefs(t *testing.T) {
	s, _ := postgresFixture(t, nil)
	ctx := context.Background()
	worker := activateMCPCapability(t, ctx, s, testMCPCapabilityRow())
	admin := testMCPCapabilityRow()
	admin.ID = strings.Repeat("d", 32)
	admin.AssignmentID = strings.Repeat("e", 32)
	admin.RunID = strings.Repeat("f", 32)
	admin.Profile = factory.MCPProfileLibraryAdmin
	admin = activateMCPCapability(t, ctx, s, admin)
	submit := func(cap factory.MCPCapability, tool factory.MCPTool, payload any) factory.MCPReceipt {
		t.Helper()
		raw, err := json.Marshal(payload)
		if err != nil {
			t.Fatal(err)
		}
		submission := testMCPSubmissionRow(cap, tool, raw)
		receipt, err := s.RecordMCPSubmission(ctx, submission, testMCPRecordedReceipt(submission))
		if err != nil {
			t.Fatalf("%s: %v", tool, err)
		}
		return receipt
	}
	learn := func(title string) string {
		t.Helper()
		return submit(worker, factory.MCPToolLearned, map[string]any{"title": title, "summary": "Boundary note.", "scope": "task access", "evidence_refs": []string{"src/access.go:12"}}).LibraryItemID
	}
	a := learn("Task ownership boundary")
	_ = learn("Task write policy")
	c := learn("Ownership consumer")
	submit(admin, factory.MCPToolRelateLibraryItems, factory.LibraryChange{ID: c, Revision: 1, Target: &factory.LibraryRef{ID: a, Revision: 1}, Relation: "depends_on", Reason: "Consumer follows the boundary", EvidenceRefs: []string{"src/access.go:12"}})
	splitReceipt := submit(admin, factory.MCPToolSplitLibraryItem, factory.LibraryChange{ID: a, Revision: 1, Reason: "Separate read and write guidance", Items: []factory.LibraryContent{{Title: "Task read policy", Body: "Check ownership before reading.", Scope: "task access"}, {Title: "Task write policy", Body: "Check ownership before writing.", Scope: "task access"}}})
	detail, err := s.FeedComment(ctx, splitReceipt.FeedPostID)
	if err != nil {
		t.Fatal(err)
	}
	posted := detail.Payload.Communication
	split, err := s.LibraryRead(ctx, worker.Repository, a, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(posted.LibraryItems) != 3 {
		t.Fatalf("split post lost atomic mapping: %+v", posted.LibraryItems)
	}
	mapped := map[string]int64{}
	for _, ref := range posted.LibraryItems {
		mapped[ref.ID] = ref.Revision
	}
	for _, id := range append([]string{a}, split.Item.Replacements...) {
		read, err := s.LibraryRead(ctx, worker.Repository, id, true)
		if err != nil {
			t.Fatal(err)
		}
		if rev, ok := mapped[id]; !ok || rev != read.Item.Revision {
			t.Fatalf("split post misstates %s: %+v", id, posted.LibraryItems)
		}
	}
	found := false
	for _, ref := range posted.AffectedRefs {
		if ref.ItemID == a && ref.FromID == c && ref.Relation == "depends_on" {
			found = true
		}
	}
	if !found {
		t.Fatalf("split post lost affected references: %+v", posted.AffectedRefs)
	}
}
