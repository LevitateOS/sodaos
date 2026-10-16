package control

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
)

func testLocal() factory.CompletionLocal {
	return factory.CompletionLocal{
		SourceCommit:         strings.Repeat("a", 40),
		PreparationDigest:    strings.Repeat("b", 64),
		ScopeDigest:          strings.Repeat("c", 64),
		AuthorityFingerprint: "authority-fingerprint",
	}
}

// fixtureEnumerator serves scripted pages with a stable revision
// bracket unless told to move it.
type fixtureEnumerator struct {
	pages    map[string]NativeInventoryPage
	revision int64
	idle     bool
	calls    int
	moveRev  func(call int) int64
	err      error
}

func (f *fixtureEnumerator) ObserveRevision(context.Context) (int64, bool, error) {
	if f.err != nil {
		return 0, false, f.err
	}
	revision := f.revision
	if f.moveRev != nil {
		revision = f.moveRev(f.calls)
	}
	idle := f.idle
	if !idle {
		return revision, false, nil
	}
	return revision, true, nil
}

func (f *fixtureEnumerator) EnumerateIssues(_ context.Context, _ int64, cursor string) (NativeInventoryPage, error) {
	f.calls++
	if f.err != nil {
		return NativeInventoryPage{}, f.err
	}
	page, ok := f.pages[cursor]
	if !ok {
		return NativeInventoryPage{}, errors.New("unexpected cursor")
	}
	revision := f.revision
	if f.moveRev != nil {
		revision = f.moveRev(f.calls)
	}
	page.Revision = revision
	page.Idle = f.idle
	return page, nil
}

func issueDigest(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:])
}

// inventoryIssue builds one canonical sealed payload through the real
// producer encoding, so the fixtures prove the exact bytes the seal
// persists and the decision assesses.
func inventoryIssue(t *testing.T, index int64) NativeInventoryIssue {
	t.Helper()
	title, body := fmt.Sprintf("title %d", index), fmt.Sprintf("body %d", index)
	data, err := factory.CompletionIssueEvidence{
		Index: index, Title: title, Body: body,
		TitleDigest: issueDigest(title), ContentDigest: issueDigest(body),
	}.CanonicalData()
	if err != nil {
		t.Fatal(err)
	}
	return NativeInventoryIssue{Index: index, Data: data}
}

func twoPageEnumerator(t *testing.T) *fixtureEnumerator {
	t.Helper()
	return &fixtureEnumerator{
		revision: 41,
		idle:     true,
		pages: map[string]NativeInventoryPage{
			"":  {Issues: []NativeInventoryIssue{inventoryIssue(t, 1), inventoryIssue(t, 2)}, NextCursor: "2"},
			"2": {Issues: []NativeInventoryIssue{inventoryIssue(t, 3)}, Complete: true},
		},
	}
}

func TestInventoryCollectorSealsCompleteMultiPageSet(t *testing.T) {
	enum := twoPageEnumerator(t)
	sealed, _, err := (&InventoryCollector{}).Collect(context.Background(), enum, 7, 1, testLocal())
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	if sealed.Issues != 3 || len(sealed.Indexes) != 3 || sealed.Indexes[0] != 1 || sealed.Indexes[2] != 3 {
		t.Fatalf("sealed=%+v, want ordered indexes 1..3", sealed)
	}
	if !factory.ValidDigest(sealed.Digest) {
		t.Fatalf("digest=%q, want a valid digest", sealed.Digest)
	}
	// The seal persists bodies, not just the digest: every index
	// carries its exact title and body for the funded decision.
	if len(sealed.Bodies) != 3 {
		t.Fatalf("bodies=%+v, want one substance per sealed issue", sealed.Bodies)
	}
	for i, body := range sealed.Bodies {
		wantTitle := fmt.Sprintf("title %d", sealed.Indexes[i])
		if body.Index != sealed.Indexes[i] || body.Title != wantTitle || body.Body != fmt.Sprintf("body %d", sealed.Indexes[i]) {
			t.Fatalf("body=%+v, want the exact substance of issue %d", body, sealed.Indexes[i])
		}
		if err := body.Validate(); err != nil {
			t.Fatalf("sealed body invalid: %v", err)
		}
	}
	again, _, err := (&InventoryCollector{}).Collect(context.Background(), twoPageEnumerator(t), 7, 1, testLocal())
	if err != nil || again.Digest != sealed.Digest {
		t.Fatalf("reseal digest=%q err=%v, want deterministic %q", again.Digest, err, sealed.Digest)
	}
}

func TestInventoryCollectorSealsCompleteEmptyInventory(t *testing.T) {
	enum := &fixtureEnumerator{
		revision: 41,
		idle:     true,
		pages:    map[string]NativeInventoryPage{"": {Complete: true}},
	}
	sealed, _, err := (&InventoryCollector{}).Collect(context.Background(), enum, 7, 1, testLocal())
	if err != nil {
		t.Fatalf("empty collect: %v", err)
	}
	if sealed.Issues != 0 || len(sealed.Indexes) != 0 || !factory.ValidDigest(sealed.Digest) {
		t.Fatalf("sealed=%+v, want a valid empty seal", sealed)
	}
	// Each completion gets its own decision cache entry even when none
	// remain: the fingerprint binds the distinct completion.
	first := SealEvidenceFingerprint(sealed.Digest, strings.Repeat("1", 32), strings.Repeat("2", 32))
	second := SealEvidenceFingerprint(sealed.Digest, strings.Repeat("3", 32), strings.Repeat("4", 32))
	if first == second || !factory.ValidDigest(first) {
		t.Fatal("empty-inventory fingerprints collide or malformed")
	}
}

func TestInventoryCollectorRefusesNonCanonicalBody(t *testing.T) {
	enum := &fixtureEnumerator{
		revision: 41,
		idle:     true,
		pages: map[string]NativeInventoryPage{
			"": {Issues: []NativeInventoryIssue{{Index: 1, Data: "not-canonical"}}, Complete: true},
		},
	}
	if _, _, err := (&InventoryCollector{}).Collect(context.Background(), enum, 7, 1, testLocal()); err == nil || err.Error() != HoldInventoryPage {
		t.Fatalf("err=%v, want %q", err, HoldInventoryPage)
	}
}

func TestInventoryCollectorRefusesMismatchedBodyIndex(t *testing.T) {
	title, body := "title 9", "body 9"
	data, err := factory.CompletionIssueEvidence{
		Index: 9, Title: title, Body: body,
		TitleDigest: issueDigest(title), ContentDigest: issueDigest(body),
	}.CanonicalData()
	if err != nil {
		t.Fatal(err)
	}
	enum := &fixtureEnumerator{
		revision: 41,
		idle:     true,
		pages: map[string]NativeInventoryPage{
			"": {Issues: []NativeInventoryIssue{{Index: 1, Data: data}}, Complete: true},
		},
	}
	if _, _, err := (&InventoryCollector{}).Collect(context.Background(), enum, 7, 1, testLocal()); err == nil || err.Error() != HoldInventoryPage {
		t.Fatalf("err=%v, want %q", err, HoldInventoryPage)
	}
}

func TestInventoryCollectorRefusesEmptyIncompletePage(t *testing.T) {
	enum := &fixtureEnumerator{
		revision: 41,
		idle:     true,
		pages:    map[string]NativeInventoryPage{"": {}},
	}
	if _, _, err := (&InventoryCollector{}).Collect(context.Background(), enum, 7, 1, testLocal()); err == nil || err.Error() != HoldInventoryPage {
		t.Fatalf("err=%v, want %q", err, HoldInventoryPage)
	}
}

func TestInventoryCollectorRefusesCursorMismatch(t *testing.T) {
	enum := &fixtureEnumerator{
		revision: 41,
		idle:     true,
		pages: map[string]NativeInventoryPage{
			"": {Issues: []NativeInventoryIssue{inventoryIssue(t, 1), inventoryIssue(t, 2)}, NextCursor: "9"},
		},
	}
	if _, _, err := (&InventoryCollector{}).Collect(context.Background(), enum, 7, 1, testLocal()); err == nil || err.Error() != HoldInventoryPage {
		t.Fatalf("err=%v, want %q", err, HoldInventoryPage)
	}
}

func TestInventoryCollectorRefusesTerminalCursor(t *testing.T) {
	enum := &fixtureEnumerator{
		revision: 41,
		idle:     true,
		pages:    map[string]NativeInventoryPage{"": {Issues: []NativeInventoryIssue{inventoryIssue(t, 1)}, NextCursor: "1", Complete: true}},
	}
	if _, _, err := (&InventoryCollector{}).Collect(context.Background(), enum, 7, 1, testLocal()); err == nil || err.Error() != HoldInventoryPage {
		t.Fatalf("err=%v, want %q", err, HoldInventoryPage)
	}
}

func TestInventoryCollectorHoldsOversizeEvidence(t *testing.T) {
	big := strings.Repeat("x", factory.MaxReassessmentEvidenceBytes)
	enum := &fixtureEnumerator{
		revision: 41,
		idle:     true,
		pages:    map[string]NativeInventoryPage{"": {Issues: []NativeInventoryIssue{{Index: 1, Data: big}, {Index: 2, Data: "y"}}, NextCursor: "2"}},
	}
	if _, _, err := (&InventoryCollector{}).Collect(context.Background(), enum, 7, 1, testLocal()); err == nil || err.Error() != HoldInventoryOversize {
		t.Fatalf("err=%v, want %q", err, HoldInventoryOversize)
	}
}

func TestInventoryCollectorRestartsOnRevisionMoveWithoutOmission(t *testing.T) {
	enum := twoPageEnumerator(t)
	enum.moveRev = func(call int) int64 {
		if call >= 2 {
			return 42
		}
		return 41
	}
	sealed, progress, err := (&InventoryCollector{}).Collect(context.Background(), enum, 7, 1, testLocal())
	if err != nil {
		t.Fatalf("collect after move: %v", err)
	}
	if progress != nil {
		t.Fatalf("progress=%+v, want no resumable progress after a seal", progress)
	}
	if sealed.Revision != 42 {
		t.Fatalf("sealed=%+v, want the post-move revision bracket", sealed)
	}
	if sealed.Issues != 3 {
		t.Fatalf("sealed=%+v, want the full set after restart, never a subset", sealed)
	}
	again, _, err := (&InventoryCollector{}).Collect(context.Background(), twoPageEnumerator(t), 7, 1, testLocal())
	if err != nil || again.Digest == sealed.Digest {
		t.Fatal("a moved revision must seal a different digest, never the stale set")
	}
}

func TestInventoryCollectorResumesFromSavedProgress(t *testing.T) {
	first := twoPageEnumerator(t)
	_, progress, err := (&InventoryCollector{}).Collect(context.Background(), &cappedEnumerator{inner: first, maxPages: 1}, 7, 1, testLocal())
	// The page budget cuts the scan mid-set: the failure holds with the
	// observed prefix persisted, so the next drive resumes instead of
	// rescanning.
	if err == nil || err.Error() != HoldInventoryUnavailable {
		t.Fatalf("err=%v, want %q", err, HoldInventoryUnavailable)
	}
	if progress == nil || progress.Cursor != "2" || len(progress.Issues) != 2 {
		t.Fatalf("progress=%+v, want the first page persisted", progress)
	}
	sealed, after, err := progress.Collect(context.Background(), twoPageEnumerator(t), 7, 1, testLocal())
	if err != nil || sealed.Issues != 3 {
		t.Fatalf("resumed=%+v err=%v, want the complete set", sealed, err)
	}
	if after != nil {
		t.Fatalf("progress=%+v, want no resumable progress after a seal", after)
	}
	fresh, _, err := (&InventoryCollector{}).Collect(context.Background(), twoPageEnumerator(t), 7, 1, testLocal())
	if err != nil || fresh.Digest != sealed.Digest {
		t.Fatal("resumed seal differs from a clean seal over the same set")
	}
}

// cappedEnumerator stops enumeration after maxPages pages to prove
// resumable progress.
type cappedEnumerator struct {
	inner    *fixtureEnumerator
	maxPages int
	pages    int
}

func (c *cappedEnumerator) ObserveRevision(ctx context.Context) (int64, bool, error) {
	return c.inner.ObserveRevision(ctx)
}

func (c *cappedEnumerator) EnumerateIssues(ctx context.Context, repository int64, cursor string) (NativeInventoryPage, error) {
	if c.pages >= c.maxPages {
		return NativeInventoryPage{}, errors.New("page budget exhausted")
	}
	c.pages++
	return c.inner.EnumerateIssues(ctx, repository, cursor)
}

func TestInventoryCollectorRefusesInvalidLocal(t *testing.T) {
	if _, _, err := (&InventoryCollector{}).Collect(context.Background(), twoPageEnumerator(t), 7, 1, factory.CompletionLocal{}); err == nil || err.Error() != HoldInventoryUnavailable {
		t.Fatalf("err=%v, want %q", err, HoldInventoryUnavailable)
	}
}

func TestInventoryCollectorHoldsWhenNativeBusy(t *testing.T) {
	enum := twoPageEnumerator(t)
	enum.idle = false
	if _, _, err := (&InventoryCollector{}).Collect(context.Background(), enum, 7, 1, testLocal()); err == nil || err.Error() != HoldInventoryRevision {
		t.Fatalf("err=%v, want %q", err, HoldInventoryRevision)
	}
}
