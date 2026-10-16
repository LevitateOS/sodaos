package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	extensions "forgejo.org/extension-sdk"
	_ "modernc.org/sqlite"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/factory/control"
	"github.com/levitateos/sodaos/internal/forgejo"
	"github.com/levitateos/sodaos/internal/store"
)

// This test proves the FW31 REPAIR serving path against file-backed
// native truth, without PostgreSQL or Fountain: a sqlite database
// holds the open issues, a SnapshotReader serves them through the
// REAL production adapter and bracket reads, the REAL collector
// seals inventory plus bodies, the REAL decision constructor carries
// that substance to an asserting chooser that returns continue, and
// the REAL release re-observation allows the fresh release while
// refusing native edits and insertions after the seal. Store-backed
// charging, fencing and selection ride the PG-gated supplement; what
// runs here is every Store-free serving joint with real file I/O.
//
// sqliteNativeDB stages the native truth: open non-PR issues with
// exact titles, bodies and edges, plus one atomic revision bracket.
type sqliteNativeDB struct {
	db *sql.DB
}

func openSQLiteNative(t *testing.T) *sqliteNativeDB {
	t.Helper()
	db, err := sql.Open("sqlite", "file:"+t.TempDir()+"/native.db?_pragma=busy_timeout(10000)")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	for _, ddl := range []string{
		`CREATE TABLE native_issue(issue_index INTEGER PRIMARY KEY, issue_id TEXT NOT NULL, title TEXT NOT NULL, body TEXT NOT NULL, dep_ids TEXT NOT NULL DEFAULT '[]', closed INTEGER NOT NULL DEFAULT 0, is_pull INTEGER NOT NULL DEFAULT 0, visible INTEGER NOT NULL DEFAULT 1)`,
		`CREATE TABLE native_revision(rev INTEGER NOT NULL, idle INTEGER NOT NULL)`,
		`INSERT INTO native_revision(rev,idle) VALUES(41,1)`,
	} {
		if _, err := db.Exec(ddl); err != nil {
			t.Fatal(err)
		}
	}
	return &sqliteNativeDB{db: db}
}

func (n *sqliteNativeDB) seedIssue(t *testing.T, index int64, title, body string, deps ...string) {
	t.Helper()
	raw, err := json.Marshal(deps)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := n.db.Exec(`INSERT INTO native_issue(issue_index,issue_id,title,body,dep_ids) VALUES(?,?,?,?,?)`,
		index, fmt.Sprintf("%d0", index), title, body, string(raw)); err != nil {
		t.Fatal(err)
	}
}

func (n *sqliteNativeDB) setBody(t *testing.T, index int64, body string) {
	t.Helper()
	if _, err := n.db.Exec(`UPDATE native_issue SET body=? WHERE issue_index=?`, body, index); err != nil {
		t.Fatal(err)
	}
}

func (n *sqliteNativeDB) setRevision(t *testing.T, revision int64) {
	t.Helper()
	if _, err := n.db.Exec(`UPDATE native_revision SET rev=?`, revision); err != nil {
		t.Fatal(err)
	}
}

func (n *sqliteNativeDB) truth(t *testing.T, index int64) (title, body string, deps []string) {
	t.Helper()
	var raw string
	if err := n.db.QueryRow(`SELECT title,body,dep_ids FROM native_issue WHERE issue_index=?`, index).Scan(&title, &body, &raw); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(raw), &deps); err != nil {
		t.Fatal(err)
	}
	return title, body, deps
}

// sqliteNativeReader serves the file-backed native truth through the
// SnapshotReader seam: paged open-issue enumeration plus exact
// per-issue bodies, with one atomic revision observation.
type sqliteNativeReader struct {
	native *sqliteNativeDB
	limit  int
}

func (r *sqliteNativeReader) ReadNativeRevision(context.Context) (extensions.NativeRevisionObservation, error) {
	var observation extensions.NativeRevisionObservation
	var idle int
	if err := r.native.db.QueryRow(`SELECT rev,idle FROM native_revision`).Scan(&observation.Revision, &idle); err != nil {
		return extensions.NativeRevisionObservation{}, err
	}
	observation.Idle = idle == 1
	return observation, nil
}

func (r *sqliteNativeReader) ReadSnapshot(_ context.Context, _ extensions.CredentialFile, req forgejo.SnapshotRequest) (forgejo.NativeSnapshot, error) {
	for _, family := range req.Families {
		if family == forgejo.FamilyRepositoryIssues {
			return r.enumerate(req)
		}
	}
	return r.readIssue(req)
}

func (r *sqliteNativeReader) enumerate(req forgejo.SnapshotRequest) (forgejo.NativeSnapshot, error) {
	bound := 0
	if req.Cursor != "" {
		var err error
		if bound, err = atoi(req.Cursor); err != nil {
			return forgejo.NativeSnapshot{}, err
		}
	}
	rows, err := r.native.db.Query(`SELECT issue_index,issue_id,visible FROM native_issue
		WHERE closed=0 AND is_pull=0 AND issue_index>? ORDER BY issue_index LIMIT ?`, bound, r.limit+1)
	if err != nil {
		return forgejo.NativeSnapshot{}, err
	}
	defer func() { _ = rows.Close() }()
	var items []forgejo.RepositoryIssueItem
	for rows.Next() {
		var index int
		var item forgejo.RepositoryIssueItem
		var visible int
		if err := rows.Scan(&index, &item.IssueID, &visible); err != nil {
			return forgejo.NativeSnapshot{}, err
		}
		item.IssueIndex = fmt.Sprintf("%d", index)
		item.Visible = visible == 1
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return forgejo.NativeSnapshot{}, err
	}
	page := &forgejo.RepositoryIssuesPage{Items: items, Complete: true}
	if len(items) > r.limit {
		items = items[:r.limit]
		page.Items = items
		page.Complete = false
		page.NextCursor = items[len(items)-1].IssueIndex
	}
	return forgejo.NativeSnapshot{RepositoryID: req.RepositoryID, RepositoryIssues: page}, nil
}

func (r *sqliteNativeReader) readIssue(req forgejo.SnapshotRequest) (forgejo.NativeSnapshot, error) {
	var issue forgejo.IssueEvidence
	var rawDeps string
	var closed, isPull, visible int
	index, err := atoi(req.IssueIndex)
	if err != nil {
		return forgejo.NativeSnapshot{}, err
	}
	err = r.native.db.QueryRow(`SELECT issue_id,title,body,dep_ids,closed,is_pull,visible FROM native_issue WHERE issue_index=?`,
		index).Scan(&issue.ID, &issue.Title, &issue.Content, &rawDeps, &closed, &isPull, &visible)
	if err != nil {
		return forgejo.NativeSnapshot{}, err
	}
	var depIDs []string
	if err := json.Unmarshal([]byte(rawDeps), &depIDs); err != nil {
		return forgejo.NativeSnapshot{}, err
	}
	issue.Index = req.IssueIndex
	issue.TitleDigest = forgejo.ContentDigest(issue.Title)
	issue.ContentDigest = forgejo.ContentDigest(issue.Content)
	issue.ContentVersion = 1
	issue.Visible = visible == 1
	issue.IsClosed = closed == 1
	issue.IsPull = isPull == 1
	issue.Complete = true
	edges := make([]forgejo.DependencyEvidence, 0, len(depIDs))
	for i, dep := range depIDs {
		edges = append(edges, forgejo.DependencyEvidence{
			OccurrenceID: fmt.Sprintf("%d", 900+i), IssueID: issue.ID,
			DependencyID: dep, Visible: true, Complete: true,
		})
	}
	return forgejo.NativeSnapshot{
		RepositoryID: req.RepositoryID, Issue: &issue,
		Dependencies: &forgejo.DependencyPage{IssueID: issue.ID, Items: edges, Total: len(edges), Complete: true},
	}, nil
}

func atoi(text string) (int, error) {
	var value int
	for _, c := range text {
		if c < '0' || c > '9' || value > 1<<30 {
			return 0, fmt.Errorf("invalid decimal index %q", text)
		}
		value = value*10 + int(c-'0')
	}
	if text == "" || (len(text) > 1 && text[0] == '0') {
		return 0, fmt.Errorf("invalid decimal index %q", text)
	}
	return value, nil
}

// substanceChooser is the funded-Jev seam: it asserts the decision
// input carries the ACTUAL sealed bodies — not digests alone — before
// answering continue.
type substanceChooser struct {
	native *sqliteNativeDB
	t      *testing.T
	calls  int
}

func (c *substanceChooser) Select(_ context.Context, state any, _ string, _ map[string]any) (factory.PriorityAnswer, error) {
	c.calls++
	decision, ok := state.(control.ReassessmentDecisionState)
	if !ok {
		return factory.PriorityAnswer{}, fmt.Errorf("decision carries %T, want issue substance", state)
	}
	if len(decision.Evidence) == 0 {
		return factory.PriorityAnswer{}, fmt.Errorf("decision judges hashes without bodies")
	}
	for _, body := range decision.Evidence {
		title, content, deps := c.native.truth(c.t, body.Index)
		if body.Title != title || body.Body != content {
			return factory.PriorityAnswer{}, fmt.Errorf("issue %d judges %q/%q instead of the sealed %q/%q",
				body.Index, body.Title, body.Body, title, content)
		}
		if len(body.Dependencies) != len(deps) {
			return factory.PriorityAnswer{}, fmt.Errorf("issue %d judges %v instead of the sealed edges %v",
				body.Index, body.Dependencies, deps)
		}
		if body.TitleDigest != forgejo.ContentDigest(title) || body.ContentDigest != forgejo.ContentDigest(content) {
			return factory.PriorityAnswer{}, fmt.Errorf("issue %d digests divorced from content", body.Index)
		}
	}
	if decision.Completion.MergedCommit == "" || decision.Completion.Issue <= 0 {
		return factory.PriorityAnswer{}, fmt.Errorf("decision lacks its completed change")
	}
	if !decision.Local.Sealed() {
		return factory.PriorityAnswer{}, fmt.Errorf("decision lacks its local bindings")
	}
	return factory.PriorityAnswer{
		Selected: factory.ReassessmentOptionContinue, Confidence: 0.9, Model: "serving-proof",
		Probabilities: map[string]float64{},
	}, nil
}

func TestCompletionServingPathSealDecideReleaseEvidence(t *testing.T) {
	ctx := context.Background()
	native := openSQLiteNative(t)
	native.seedIssue(t, 1, "First title", "First body", "20")
	native.seedIssue(t, 2, "Second title", "Second body")
	native.seedIssue(t, 3, "Third title", "Third body", "10", "20")

	adapter := &CompletionSnapshotSource{Reader: &sqliteNativeReader{native: native, limit: 2}}
	local := completionTestLocal()

	// Completion seals its complete inventory plus bodies through
	// the real production adapter and collector.
	sealed, _, err := (&control.InventoryCollector{}).Collect(ctx, adapter, 7, 1, local)
	if err != nil {
		t.Fatalf("seal: %v", err)
	}
	if sealed.Issues != 3 || len(sealed.Bodies) != 3 || sealed.Revision != 41 {
		t.Fatalf("sealed=%+v, want the three file-backed issues with bodies", sealed)
	}
	for _, body := range sealed.Bodies {
		title, content, _ := native.truth(t, body.Index)
		if body.Title != title || body.Body != content {
			t.Fatalf("body=%+v, want the file-backed substance", body)
		}
	}

	// The exact D13 decision request carries that substance.
	record := store.ReassessmentRecord{
		Cycle: factory.ReassessmentCycle{
			EventID: strings.Repeat("1", 32), CycleID: strings.Repeat("2", 32),
			State: factory.ReassessmentDeciding, Repository: 7, Generation: 1,
		},
		Completion: factory.CompletionEvent{
			PublicationID: strings.Repeat("5", 32), MergedCommit: strings.Repeat("c", 40),
			ClosingOccurrence: "501", AcceptanceID: "d" + strings.Repeat("a", 24),
			ID: strings.Repeat("1", 32), Repository: 7, Issue: 3, Generation: 1,
		},
		Local: local, Evidence: sealed.Bodies,
		InventoryDigest: sealed.Digest,
		EvidenceFingerprint: control.SealEvidenceFingerprint(sealed.Digest,
			strings.Repeat("1", 32), strings.Repeat("2", 32)),
		Issue: 3, EvidenceIssues: sealed.Issues, EvidenceRevision: sealed.Revision,
	}
	if err := record.Validate(); err != nil {
		t.Fatalf("sealed record: %v", err)
	}
	state, err := control.DecisionStateFromRecord(record)
	if err != nil {
		t.Fatalf("decision input: %v", err)
	}
	// Replay determinism: the same sealed record encodes the same
	// request, so a charged replay reconciles against its receipt
	// instead of charging again.
	again, err := control.DecisionStateFromRecord(record)
	if err != nil {
		t.Fatalf("replay input: %v", err)
	}
	first, _ := json.Marshal(state)
	second, _ := json.Marshal(again)
	if string(first) != string(second) || !factory.ValidDigest(control.SealEvidenceFingerprint(sealed.Digest, strings.Repeat("1", 32), strings.Repeat("2", 32))) {
		t.Fatal("replayed decision request differs from its sealed input")
	}

	// The chooser asserts the actual bodies, then continues.
	chooser := &substanceChooser{native: native, t: t}
	answer, err := chooser.Select(ctx, state, "", nil)
	if err != nil {
		t.Fatalf("chooser: %v", err)
	}
	if answer.Selected != factory.ReassessmentOptionContinue {
		t.Fatalf("answer=%+v, want the substance-backed continue", answer)
	}

	// A fresh world releases: re-observation matches the seal.
	fresh, err := control.ReobserveCompletionInventory(ctx, adapter, 7, 1, local)
	if err != nil {
		t.Fatalf("re-observe: %v", err)
	}
	if fresh.Digest != sealed.Digest {
		t.Fatal("fresh inventory differs from its seal without any native change")
	}
	sealedHalf := control.CompletionFreshness{
		SourceCommit: local.SourceCommit, PreparationDigest: local.PreparationDigest,
		ScopeDigest: local.ScopeDigest, AuthorityFingerprint: local.AuthorityFingerprint,
		EvidenceDigest: sealed.Digest,
	}
	currentHalf := control.CompletionFreshness{
		SourceCommit: fresh.Local.SourceCommit, PreparationDigest: fresh.Local.PreparationDigest,
		ScopeDigest: fresh.Local.ScopeDigest, AuthorityFingerprint: fresh.Local.AuthorityFingerprint,
		EvidenceDigest: fresh.Digest,
	}
	if err := control.CheckCompletionFreshness(sealedHalf, currentHalf); err != nil {
		t.Fatalf("fresh release refused: %v", err)
	}

	// A native edit after the seal refuses the stale release.
	native.setBody(t, 2, "Second body edited after the seal")
	edited, err := control.ReobserveCompletionInventory(ctx, adapter, 7, 1, local)
	if err != nil {
		t.Fatalf("re-observe after edit: %v", err)
	}
	currentHalf.EvidenceDigest = edited.Digest
	if err := control.CheckCompletionFreshness(sealedHalf, currentHalf); err == nil ||
		!strings.Contains(err.Error(), "evidence") {
		t.Fatalf("err=%v, want the evidence invalidation for the edited issue", err)
	}
	native.setBody(t, 2, "Second body")

	// An insertion after the seal refuses the stale release too.
	native.seedIssue(t, 4, "Fourth title", "Fourth body")
	inserted, err := control.ReobserveCompletionInventory(ctx, adapter, 7, 1, local)
	if err != nil {
		t.Fatalf("re-observe after insertion: %v", err)
	}
	currentHalf.EvidenceDigest = inserted.Digest
	if err := control.CheckCompletionFreshness(sealedHalf, currentHalf); err == nil ||
		!strings.Contains(err.Error(), "evidence") {
		t.Fatalf("err=%v, want the evidence invalidation for the inserted issue", err)
	}
	if _, err := native.db.Exec(`DELETE FROM native_issue WHERE issue_index=4`); err != nil {
		t.Fatal(err)
	}

	// A revision advance refuses too: the seal binds its bracket, so
	// any native mutation after the seal — even one outside the
	// issue bodies — retires the release and waits for explicit
	// Retry, which reseals the successor at the new revision. The
	// bracket is identity, not an observation timestamp.
	native.setRevision(t, 42)
	moved, err := control.ReobserveCompletionInventory(ctx, adapter, 7, 1, local)
	if err != nil {
		t.Fatalf("re-observe after revision move: %v", err)
	}
	if moved.Digest == sealed.Digest {
		t.Fatal("revision move resealed the stale bracket digest")
	}
	currentHalf.EvidenceDigest = moved.Digest
	if err := control.CheckCompletionFreshness(sealedHalf, currentHalf); err == nil ||
		!strings.Contains(err.Error(), "evidence") {
		t.Fatalf("err=%v, want the evidence invalidation for the moved revision", err)
	}
}
