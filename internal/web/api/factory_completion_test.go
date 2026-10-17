package api

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"

	extensions "forgejo.org/extension-sdk"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/factory/control"
	"github.com/levitateos/sodaos/internal/forgejo"
)

// completionReaderFake serves one scripted revision bracket, one
// repository_issues page per cursor, and one body per issue index.
// Tests supply deterministic fixtures; production serves the
// background channel.
type completionReaderFake struct {
	revision int64
	idle     bool
	pages    map[string][]string
	bodies   map[string]forgejo.NativeSnapshot
	revErr   error
	snapErr  error
	moveRev  func(calls int) int64
	calls    int
}

func (f *completionReaderFake) ReadNativeRevision(context.Context) (extensions.NativeRevisionObservation, error) {
	if f.revErr != nil {
		return extensions.NativeRevisionObservation{}, f.revErr
	}
	revision := f.revision
	if f.moveRev != nil {
		revision = f.moveRev(f.calls)
	}
	return extensions.NativeRevisionObservation{Revision: revision, Idle: f.idle}, nil
}

func (f *completionReaderFake) ReadSnapshot(_ context.Context, _ extensions.CredentialFile, req forgejo.SnapshotRequest) (forgejo.NativeSnapshot, error) {
	f.calls++
	if f.snapErr != nil {
		return forgejo.NativeSnapshot{}, f.snapErr
	}
	revision := f.revision
	if f.moveRev != nil {
		revision = f.moveRev(f.calls)
	}
	if len(req.Families) == 1 && req.Families[0] == forgejo.FamilyRepositoryIssues {
		indexes, ok := f.pages[req.Cursor]
		if !ok {
			return forgejo.NativeSnapshot{}, errors.New("unexpected cursor")
		}
		items := make([]forgejo.RepositoryIssueItem, 0, len(indexes))
		for _, index := range indexes {
			items = append(items, forgejo.RepositoryIssueItem{IssueID: index + "0", IssueIndex: index, Visible: true})
		}
		page := &forgejo.RepositoryIssuesPage{Items: items, Complete: true}
		if next, ok := completionNextCursor[req.Cursor]; ok {
			page.Complete = false
			page.NextCursor = next
		}
		return forgejo.NativeSnapshot{Revision: revision, RepositoryID: req.RepositoryID, RepositoryIssues: page}, nil
	}
	snapshot, ok := f.bodies[req.IssueIndex]
	if !ok {
		return forgejo.NativeSnapshot{}, errors.New("unexpected issue")
	}
	snapshot.Revision = revision
	snapshot.RepositoryID = req.RepositoryID
	return snapshot, nil
}

// completionNextCursor chains the scripted pages: cursor "" holds two
// issues with continuation "2", cursor "2" terminates.
var completionNextCursor = map[string]string{"": "2"}

func completionBody(index, title, body string, deps ...string) forgejo.NativeSnapshot {
	edges := make([]forgejo.DependencyEvidence, 0, len(deps))
	for i, dep := range deps {
		edges = append(edges, forgejo.DependencyEvidence{
			OccurrenceID: strconv.Itoa(700 + i), IssueID: index + "0",
			DependencyID: dep, Visible: true, Complete: true,
		})
	}
	return forgejo.NativeSnapshot{
		Issue: &forgejo.IssueEvidence{
			ID: index + "0", Index: index, Title: title, Content: body,
			TitleDigest: forgejo.ContentDigest(title), ContentDigest: forgejo.ContentDigest(body),
			ContentVersion: 1, Visible: true, Complete: true,
		},
		Dependencies: &forgejo.DependencyPage{
			IssueID: index + "0", Items: edges, Total: len(edges), Complete: true,
		},
	}
}

func completionTwoPageFake() *completionReaderFake {
	return &completionReaderFake{
		revision: 41, idle: true,
		pages: map[string][]string{"": {"1", "2"}, "2": {"3"}},
		bodies: map[string]forgejo.NativeSnapshot{
			"1": completionBody("1", "title 1", "body 1"),
			"2": completionBody("2", "title 2", "body 2", "10"),
			"3": completionBody("3", "title 3", "body 3"),
		},
	}
}

func TestCompletionSnapshotSourceEnumeratesBodies(t *testing.T) {
	source := NewCompletionSnapshotSource(nil)
	source.Reader = completionTwoPageFake()
	ctx := context.Background()
	revision, idle, err := source.ObserveRevision(ctx)
	if err != nil || revision != 41 || !idle {
		t.Fatalf("revision=%d idle=%v err=%v, want the stable bracket", revision, idle, err)
	}
	first, err := source.EnumerateIssues(ctx, 7, "")
	if err != nil {
		t.Fatalf("first page: %v", err)
	}
	if first.Complete || first.NextCursor != "2" || len(first.Issues) != 2 || first.Revision != 41 || !first.Idle {
		t.Fatalf("first=%+v, want the bracketed nonterminal page", first)
	}
	second, err := source.EnumerateIssues(ctx, 7, "2")
	if err != nil {
		t.Fatalf("second page: %v", err)
	}
	if !second.Complete || second.NextCursor != "" || len(second.Issues) != 1 {
		t.Fatalf("second=%+v, want the terminal page", second)
	}
	// Bodies carry substance: the exact titles, bodies and edges the
	// funded decision assesses, in canonical form.
	parsed, err := factory.ParseCompletionIssueEvidence(first.Issues[1].Data)
	if err != nil {
		t.Fatalf("body: %v", err)
	}
	if parsed.Index != 2 || parsed.Title != "title 2" || parsed.Body != "body 2" ||
		len(parsed.Dependencies) != 1 || parsed.Dependencies[0] != "10" {
		t.Fatalf("parsed=%+v, want the exact issue substance", parsed)
	}
	if parsed.TitleDigest != forgejo.ContentDigest("title 2") {
		t.Fatal("body digest divorced from its content")
	}
}

func TestCompletionSnapshotSourceSealsThroughCollector(t *testing.T) {
	source := NewCompletionSnapshotSource(nil)
	source.Reader = completionTwoPageFake()
	sealed, _, err := (&control.InventoryCollector{}).Collect(context.Background(), source, 7, 1, completionTestLocal())
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	if sealed.Issues != 3 || len(sealed.Bodies) != 3 || sealed.Revision != 41 {
		t.Fatalf("sealed=%+v, want the complete bracketed set with bodies", sealed)
	}
}

func completionTestLocal() factory.CompletionLocal {
	acceptance := factory.Acceptance{
		ID: "d" + strings.Repeat("0", 24), Repository: 7, IssueIndex: "3",
		Approver: 5, NativeRev: 41,
		TitleDigest: strings.Repeat("a", 64), TitleOccurrence: factory.OriginalOccurrence,
		ContentDigest: strings.Repeat("b", 64), ContentVersion: 2,
		Sources: []factory.SelectedSource{{ID: "11", ContentVersion: 0, Digest: strings.Repeat("c", 64)}},
	}
	material := factory.CompletionMaterial{
		Intents:     []factory.CompletionIntentScope{},
		Acceptances: []factory.Acceptance{acceptance},
		Withdrawals: []factory.CompletionWithdrawal{},
		Controls: []factory.IssueControl{{
			Repository: 7, Issue: 7,
			Readiness: factory.ReadinessQueued, Reason: factory.ReasonEligible,
			Fingerprint: strings.Repeat("1", 64), Authority: strings.Repeat("2", 64),
		}},
		Preparations: []factory.CompletionPreparation{},
		Plans:        []factory.InvestigationPlan{},
		PlanHeads:    []factory.CompletionPlanHead{},
		PlanTasks:    []factory.CompletionPlanTask{},
		Assignments:  []factory.CompletionAssignment{},
		Publications: []factory.CompletionPublication{},
		Merges:       []factory.CompletionMerge{},
		Checks:       []factory.CompletionCheck{},
		Authority: factory.CompletionAuthority{
			Effective:           factory.EffectiveAuthority{Missing: []string{}, Effective: true, DispatchOpen: true},
			Roles:               []string{"soda-coder", "soda-reviewer"},
			ReassessmentMinutes: factory.DefaultReassessmentMinutes,
		},
		Repository: 7,
	}
	digest, err := material.Digest()
	if err != nil {
		panic(err)
	}
	return factory.CompletionLocal{
		Material:             &material,
		SourceCommit:         strings.Repeat("c", 40),
		PreparationDigest:    strings.Repeat("d", 64),
		ScopeDigest:          digest,
		AuthorityFingerprint: "authority",
	}
}

func TestCompletionSnapshotSourceRefusesHiddenBody(t *testing.T) {
	fake := completionTwoPageFake()
	hidden := completionBody("1", "title 1", "body 1")
	hidden.Issue.Visible = false
	hidden.Issue.Title, hidden.Issue.Content = "", ""
	hidden.Issue.TitleDigest, hidden.Issue.ContentDigest = "", ""
	fake.bodies["1"] = hidden
	source := NewCompletionSnapshotSource(nil)
	source.Reader = fake
	if _, err := source.EnumerateIssues(context.Background(), 7, ""); err == nil {
		t.Fatal("hidden body enumerated instead of refusing")
	}
}

func TestCompletionSnapshotSourceRefusesClosedBody(t *testing.T) {
	fake := completionTwoPageFake()
	closed := completionBody("2", "title 2", "body 2")
	closed.Issue.IsClosed = true
	fake.bodies["2"] = closed
	source := NewCompletionSnapshotSource(nil)
	source.Reader = fake
	if _, err := source.EnumerateIssues(context.Background(), 7, ""); err == nil {
		t.Fatal("closed issue enumerated instead of refusing")
	}
}

func TestCompletionSnapshotSourceReportsRevisionMove(t *testing.T) {
	fake := completionTwoPageFake()
	fake.moveRev = func(calls int) int64 {
		if calls >= 2 {
			return 42
		}
		return 41
	}
	source := NewCompletionSnapshotSource(nil)
	source.Reader = fake
	_, err := source.EnumerateIssues(context.Background(), 7, "")
	if err == nil || err.Error() != control.HoldInventoryRevision {
		t.Fatalf("err=%v, want the revision hold for the restart path", err)
	}
}

func TestCompletionSnapshotSourceRequiresReader(t *testing.T) {
	source := NewCompletionSnapshotSource(nil)
	if _, _, err := source.ObserveRevision(context.Background()); err == nil {
		t.Fatal("revision observed without a reader")
	}
	if _, err := source.EnumerateIssues(context.Background(), 7, ""); err == nil {
		t.Fatal("issues enumerated without a reader")
	}
}
