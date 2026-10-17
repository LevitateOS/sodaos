package control

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/forgejo"
)

// testMaterialLocal builds one valid material-bearing local for
// repository 7: explicit empty sections plus one acceptance, its
// control and the standing authority, with ScopeDigest exactly the
// material digest. It panics on construction failure like the other
// deterministic fixtures: the shape is fixed and always valid.
func testMaterialLocal() factory.CompletionLocal {
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
		SourceCommit:         strings.Repeat("a", 40),
		PreparationDigest:    strings.Repeat("b", 64),
		ScopeDigest:          digest,
		AuthorityFingerprint: "authority-fingerprint",
	}
}

func testLocal() factory.CompletionLocal {
	return testMaterialLocal()
}

// digestOnlyLocal is the pre-077 shape: valid four bindings but no
// material body. Every consumer must refuse it before any spend.
func digestOnlyLocal() factory.CompletionLocal {
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
	observes int
	moveRev  func(call int) int64
	err      error
}

func (f *fixtureEnumerator) ObserveRevision(context.Context) (int64, bool, error) {
	f.observes++
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

// bracketCapture serves a scripted local per capture call and counts
// its reads, proving the bracket's extra reads stay bounded.
type bracketCapture struct {
	locals []factory.CompletionLocal
	calls  int
	err    error
}

func (b *bracketCapture) capture(context.Context, int64) (factory.CompletionLocal, error) {
	b.calls++
	if b.err != nil {
		return factory.CompletionLocal{}, b.err
	}
	if b.calls-1 < len(b.locals) {
		return b.locals[b.calls-1], nil
	}
	return b.locals[len(b.locals)-1], nil
}

func bracketDeps(enum CompletionIssueEnumerator, capture *bracketCapture) CompletionDriveDeps {
	return CompletionDriveDeps{Enumerator: enum, CaptureLocal: capture.capture}
}

// A stable world seals through the full bracket: idle revision
// observed before capture, collection under it, recapture and confirm
// after, with exactly two bounded local reads and the clean-seal
// digest — never a combined snapshot.
func TestBracketMaterialCollectSealsStableWorld(t *testing.T) {
	capture := &bracketCapture{locals: []factory.CompletionLocal{testLocal(), testLocal()}}
	out := bracketMaterialCollect(context.Background(), bracketDeps(twoPageEnumerator(t), capture), 7, 1, nil)
	if out.hold != "" || out.sealed == nil {
		t.Fatalf("out=%+v, want a confirmed seal", out)
	}
	if capture.calls != 2 {
		t.Fatalf("capture calls=%d, want exactly the bounded capture plus recapture", capture.calls)
	}
	fresh, _, err := (&InventoryCollector{}).Collect(context.Background(), twoPageEnumerator(t), 7, 1, testLocal())
	if err != nil || fresh.Digest != out.sealed.Digest {
		t.Fatalf("bracket digest=%q err=%v, want the clean-seal digest %q", out.sealed.Digest, err, fresh.Digest)
	}
}

// A local that moves between capture and recapture defers with the
// named material hold and a reset cursor: the unconfirmed seal never
// persists.
func TestBracketMaterialCollectDefersOnLocalMove(t *testing.T) {
	moved := testLocal()
	moved.AuthorityFingerprint = "changed-under-the-seal"
	capture := &bracketCapture{locals: []factory.CompletionLocal{testLocal(), moved}}
	out := bracketMaterialCollect(context.Background(), bracketDeps(twoPageEnumerator(t), capture), 7, 1, nil)
	if out.hold != HoldInventoryMaterial || out.sealed != nil || out.progress != nil {
		t.Fatalf("out=%+v, want the named material hold with a reset cursor", out)
	}
}

// A native revision that moves across the bracket defers with the
// revision hold and a reset cursor, even when the collection itself
// resealed under the new revision.
func TestBracketMaterialCollectDefersOnRevisionMove(t *testing.T) {
	enum := twoPageEnumerator(t)
	enum.moveRev = func(call int) int64 {
		if call >= 2 {
			return 42
		}
		return 41
	}
	capture := &bracketCapture{locals: []factory.CompletionLocal{testLocal()}}
	out := bracketMaterialCollect(context.Background(), bracketDeps(enum, capture), 7, 1, nil)
	if out.hold != HoldInventoryRevision || out.sealed != nil || out.progress != nil {
		t.Fatalf("out=%+v, want the revision hold with a reset cursor", out)
	}
}

// A busy native state holds with the saved cursor kept for resume:
// nothing moved, so nothing resets.
func TestBracketMaterialCollectHoldsBusyNative(t *testing.T) {
	enum := twoPageEnumerator(t)
	enum.idle = false
	saved := &InventoryCollector{Revision: 41, Cursor: "2"}
	capture := &bracketCapture{locals: []factory.CompletionLocal{testLocal()}}
	out := bracketMaterialCollect(context.Background(), bracketDeps(enum, capture), 7, 1, saved)
	if out.hold != HoldInventoryRevision || out.progress != saved {
		t.Fatalf("out=%+v, want the revision hold with the saved cursor kept", out)
	}
	if capture.calls != 0 {
		t.Fatalf("capture calls=%d, want zero reads before an idle revision", capture.calls)
	}
}

// A stale saved cursor from a moved revision resets instead of
// resuming: the seal matches a clean seal exactly, proving snapshots
// never combine.
func TestBracketMaterialCollectResetsStaleCursor(t *testing.T) {
	saved := &InventoryCollector{
		Revision: 99, Cursor: "2",
		Issues: []NativeInventoryIssue{{Index: 9, Data: "stale-snapshot"}},
	}
	capture := &bracketCapture{locals: []factory.CompletionLocal{testLocal()}}
	out := bracketMaterialCollect(context.Background(), bracketDeps(twoPageEnumerator(t), capture), 7, 1, saved)
	if out.hold != "" || out.sealed == nil {
		t.Fatalf("out=%+v, want a seal after the cursor reset", out)
	}
	fresh, _, err := (&InventoryCollector{}).Collect(context.Background(), twoPageEnumerator(t), 7, 1, testLocal())
	if err != nil || fresh.Digest != out.sealed.Digest {
		t.Fatal("reset seal combined its stale snapshot instead of recollecting clean")
	}
}

func TestBracketMaterialCollectRefusesUnwired(t *testing.T) {
	out := bracketMaterialCollect(context.Background(), CompletionDriveDeps{}, 7, 1, nil)
	if out.hold != HoldCompletionWiring || out.sealed != nil {
		t.Fatalf("out=%+v, want the wiring hold without sealing", out)
	}
}

func TestBracketMaterialCollectHoldsOnCaptureFailure(t *testing.T) {
	saved := &InventoryCollector{Revision: 41}
	capture := &bracketCapture{err: errors.New("local capture failed")}
	out := bracketMaterialCollect(context.Background(), bracketDeps(twoPageEnumerator(t), capture), 7, 1, saved)
	if out.hold != HoldInventoryUnavailable || out.progress != saved {
		t.Fatalf("out=%+v, want the unavailable hold with the saved cursor kept", out)
	}
}

// The pre-077 digest-only local (valid four bindings, nil material)
// refuses before any native spend: the collector validates the local
// before its first observation, so zero revision or page reads happen.
// pagedEnumerator serves the ordered issues 1..total split into pages
// of the given sizes through the real canonical encoding, so digest
// comparisons prove pagination independence over identical substance.
func pagedEnumerator(t *testing.T, sizes ...int) *fixtureEnumerator {
	t.Helper()
	pages := map[string]NativeInventoryPage{}
	cursor := ""
	index := int64(1)
	for i, size := range sizes {
		issues := make([]NativeInventoryIssue, 0, size)
		for j := 0; j < size; j++ {
			issues = append(issues, inventoryIssue(t, index))
			index++
		}
		fetched := cursor
		cursor = strconv.FormatInt(index-1, 10)
		if i == len(sizes)-1 {
			pages[fetched] = NativeInventoryPage{Issues: issues, Complete: true}
		} else {
			pages[fetched] = NativeInventoryPage{Issues: issues, NextCursor: cursor}
		}
	}
	return &fixtureEnumerator{revision: 41, idle: true, pages: pages}
}

func collectDigest(t *testing.T, enum CompletionIssueEnumerator) string {
	t.Helper()
	sealed, progress, err := (&InventoryCollector{}).Collect(context.Background(), enum, 7, 1, testLocal())
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	if progress != nil {
		t.Fatalf("progress=%+v, want no resumable progress after a seal", progress)
	}
	if sealed.Issues != 3 || len(sealed.Bodies) != 3 {
		t.Fatalf("sealed=%+v, want the three ordered issues with bodies", sealed)
	}
	return sealed.Digest
}

// The fold is associative over any page partition: one whole-list fold
// matches every per-page chained fold, so collection and reseal agree
// no matter where page boundaries fall. An empty page is the identity.
func TestFoldInventoryPageMatchesAnyPartition(t *testing.T) {
	var items []NativeInventoryIssue
	for i := int64(1); i <= 4; i++ {
		items = append(items, inventoryIssue(t, i))
	}
	var zero [32]byte
	whole := foldInventoryPage(zero, 7, 1, items)
	if got := foldInventoryPage(zero, 7, 1, nil); got != zero {
		t.Fatal("an empty page must leave the running digest unchanged")
	}
	split := foldInventoryPage(foldInventoryPage(foldInventoryPage(zero, 7, 1, items[:1]), 7, 1, items[1:3]), 7, 1, items[3:])
	if split != whole {
		t.Fatal("1+2+1 page folds differ from the whole-list fold over the same order")
	}
	single := zero
	for _, item := range items {
		single = foldInventoryPage(single, 7, 1, []NativeInventoryIssue{item})
	}
	if single != whole {
		t.Fatal("per-issue folds differ from the whole-list fold over the same order")
	}
}

// The chain stays order-sensitive on purpose: folding a reordered
// sequence must differ, so no caller can sort a malformed persisted
// sequence into admission through this fold.
func TestFoldInventoryPageIsOrderSensitive(t *testing.T) {
	var ordered, reversed []NativeInventoryIssue
	for i := int64(1); i <= 3; i++ {
		ordered = append(ordered, inventoryIssue(t, i))
		reversed = append([]NativeInventoryIssue{inventoryIssue(t, i)}, reversed...)
	}
	var zero [32]byte
	if foldInventoryPage(zero, 7, 1, reversed) == foldInventoryPage(zero, 7, 1, ordered) {
		t.Fatal("reordered items fold identically, losing the order binding")
	}
}

// The same ordered inventory seals the identical digest across one
// page, two pages, three single-item pages and a resumed cursor: page
// boundaries carry no semantic authority.
func TestInventoryCollectorSealsIdenticalDigestAcrossPaginations(t *testing.T) {
	one := collectDigest(t, pagedEnumerator(t, 3))
	two := collectDigest(t, pagedEnumerator(t, 2, 1))
	three := collectDigest(t, pagedEnumerator(t, 1, 1, 1))
	if one != two || two != three {
		t.Fatalf("digests %q/%q/%q differ across paginations of the same order", one, two, three)
	}
	first := pagedEnumerator(t, 1, 1, 1)
	_, progress, err := (&InventoryCollector{}).Collect(context.Background(), &cappedEnumerator{inner: first, maxPages: 1}, 7, 1, testLocal())
	if err == nil || err.Error() != HoldInventoryUnavailable {
		t.Fatalf("err=%v, want %q", err, HoldInventoryUnavailable)
	}
	resumed, after, err := progress.Collect(context.Background(), pagedEnumerator(t, 1, 1, 1), 7, 1, testLocal())
	if err != nil || resumed.Digest != one {
		t.Fatalf("resumed digest=%q err=%v, want the clean-seal digest %q", resumed.Digest, err, one)
	}
	if after != nil {
		t.Fatalf("progress=%+v, want no resumable progress after a seal", after)
	}
}

// A page whose issues run out of order refuses: increasing order is
// enforced, never sorted into admission.
func TestInventoryCollectorRefusesReorderedPage(t *testing.T) {
	enum := &fixtureEnumerator{
		revision: 41,
		idle:     true,
		pages: map[string]NativeInventoryPage{
			"": {Issues: []NativeInventoryIssue{inventoryIssue(t, 2), inventoryIssue(t, 1)}, NextCursor: "1"},
		},
	}
	if _, _, err := (&InventoryCollector{}).Collect(context.Background(), enum, 7, 1, testLocal()); err == nil || err.Error() != HoldInventoryPage {
		t.Fatalf("err=%v, want %q", err, HoldInventoryPage)
	}
}

// A duplicated issue index refuses, within one page and across the
// page boundary alike.
func TestInventoryCollectorRefusesDuplicateIndex(t *testing.T) {
	within := &fixtureEnumerator{
		revision: 41,
		idle:     true,
		pages: map[string]NativeInventoryPage{
			"": {Issues: []NativeInventoryIssue{inventoryIssue(t, 2), inventoryIssue(t, 2)}, NextCursor: "2"},
		},
	}
	if _, _, err := (&InventoryCollector{}).Collect(context.Background(), within, 7, 1, testLocal()); err == nil || err.Error() != HoldInventoryPage {
		t.Fatalf("err=%v, want %q", err, HoldInventoryPage)
	}
	across := &fixtureEnumerator{
		revision: 41,
		idle:     true,
		pages: map[string]NativeInventoryPage{
			"":  {Issues: []NativeInventoryIssue{inventoryIssue(t, 1), inventoryIssue(t, 2)}, NextCursor: "2"},
			"2": {Issues: []NativeInventoryIssue{inventoryIssue(t, 2), inventoryIssue(t, 3)}, Complete: true},
		},
	}
	if _, _, err := (&InventoryCollector{}).Collect(context.Background(), across, 7, 1, testLocal()); err == nil || err.Error() != HoldInventoryPage {
		t.Fatalf("err=%v, want %q", err, HoldInventoryPage)
	}
}

func TestInventoryCollectorRefusesDigestOnlyLocal(t *testing.T) {
	enum := twoPageEnumerator(t)
	if _, _, err := (&InventoryCollector{}).Collect(context.Background(), enum, 7, 1, digestOnlyLocal()); err == nil || err.Error() != HoldInventoryUnavailable {
		t.Fatalf("err=%v, want %q", err, HoldInventoryUnavailable)
	}
	if enum.calls != 0 || enum.observes != 0 {
		t.Fatalf("calls=%d observes=%d, want zero native spend on a digest-only local", enum.calls, enum.observes)
	}
}

// The stable bracket charges exactly its two outer revision
// observations plus the collector's own two (opening + confirming)
// through the caller's readiness pass, reusing the existing budget:
// five pre-reserved units plus the four observation charges read
// back as nine, proving no new counter, no uncharged exemption and
// no double charge of either bracket.
func TestBracketMaterialCollectChargesTwoObservations(t *testing.T) {
	coord := &Coordinator{}
	ctx, _, cancel := coord.readinessPass(context.Background())
	defer cancel()
	budget := readinessBudgetFrom(ctx)
	if budget == nil {
		t.Fatal("drive did not create its readiness pass")
	}
	if !budget.reserveNativeRPC(5) {
		t.Fatal("fixture could not prime its budget")
	}
	enum := twoPageEnumerator(t)
	capture := &bracketCapture{locals: []factory.CompletionLocal{testLocal(), testLocal()}}
	out := bracketMaterialCollect(ctx, bracketDeps(enum, capture), 7, 1, nil)
	if out.hold != "" || out.sealed == nil {
		t.Fatalf("out=%+v, want a confirmed seal", out)
	}
	if got := budget.nativeRPC.Load(); got != 9 {
		t.Fatalf("nativeRPC=%d, want 5 primed plus two outer plus two collector observations", got)
	}
	if enum.observes != 4 {
		t.Fatalf("observes=%d, want two outer plus two collector observations", enum.observes)
	}
}

// scriptedRevEnumerator serves twoPageEnumerator pages while
// reporting scripted observation revisions, so confirmation movement
// can be isolated from page movement (the shared fixture keys both
// off fetch counts).
type scriptedRevEnumerator struct {
	inner             *fixtureEnumerator
	observes          int
	fetches           int
	moveFromObserve   int
	movedRev          int64
	flipAfterFetches  int
	flipTo            int64
	flipped           bool
}

func (e *scriptedRevEnumerator) ObserveRevision(ctx context.Context) (int64, bool, error) {
	e.observes++
	if e.moveFromObserve > 0 && e.observes >= e.moveFromObserve {
		_, idle, err := e.inner.ObserveRevision(ctx)
		return e.movedRev, idle, err
	}
	return e.inner.ObserveRevision(ctx)
}

func (e *scriptedRevEnumerator) EnumerateIssues(ctx context.Context, repository int64, cursor string) (NativeInventoryPage, error) {
	e.fetches++
	if e.flipAfterFetches > 0 && e.fetches > e.flipAfterFetches && !e.flipped {
		e.inner.revision = e.flipTo
		e.flipped = true
	}
	return e.inner.EnumerateIssues(ctx, repository, cursor)
}

// A rebracket quota hold carries actually-resumable progress: the
// cursor tracks the folded prefix, and a fresh pass seals exactly the
// ordered bodies with the identical digest of a clean single-pass
// seal — no duplicate, no gap.
func TestInventoryCollectorResumesPrefixAfterBudgetHold(t *testing.T) {
	coord := &Coordinator{}
	ctx, _, cancel := coord.readinessPass(context.Background())
	defer cancel()
	budget := readinessBudgetFrom(ctx)
	if budget == nil {
		t.Fatal("missing readiness pass")
	}
	if !budget.reserveNativeRPC(readinessMaxNativeRPC - 2) {
		t.Fatal("fixture could not prime its budget")
	}
	enum := twoPageEnumerator(t)
	enum.moveRev = func(call int) int64 {
		if call >= 2 {
			return 42
		}
		return 41
	}
	_, progress, err := (&InventoryCollector{}).Collect(ctx, enum, 7, 1, testLocal())
	if err == nil || err.Error() != HoldInventoryInProgress {
		t.Fatalf("err=%v, want %q", err, HoldInventoryInProgress)
	}
	if progress == nil || len(progress.Issues) != 2 || progress.Cursor != "2" {
		t.Fatalf("progress=%+v, want the one-page prefix with its advanced cursor", progress)
	}
	if enum.observes != 1 {
		t.Fatalf("observes=%d, want only the opening call: the re-bracket reserves its own unit and fails", enum.observes)
	}
	if got := budget.nativeRPC.Load(); got != readinessMaxNativeRPC {
		t.Fatalf("nativeRPC=%d, want the upfront two spent and the refused re-bracket spending nothing", got)
	}
	clean, _, err := (&InventoryCollector{}).Collect(context.Background(), twoPageEnumerator(t), 7, 1, testLocal())
	if err != nil {
		t.Fatalf("clean seal: %v", err)
	}
	ctx2, _, cancel2 := coord.readinessPass(context.Background())
	defer cancel2()
	resumed, _, err := progress.Collect(ctx2, twoPageEnumerator(t), 7, 1, testLocal())
	if err != nil {
		t.Fatalf("resume: %v", err)
	}
	if resumed.Issues != 3 {
		t.Fatalf("resumed issues=%d, want exactly the three ordered bodies", resumed.Issues)
	}
	for i, want := range []int64{1, 2, 3} {
		if resumed.Indexes[i] != want || resumed.Bodies[i].Index != want {
			t.Fatalf("resumed=%+v, want exact ordered bodies without duplicate or gap", resumed.Indexes)
		}
	}
	if resumed.Digest != clean.Digest {
		t.Fatal("resumed digest differs from the clean single-pass seal")
	}
}

// A confirmation-observed move resets (the old-rev prefix is invalid)
// with a named hold; resume seals the new-revision set exactly once.
func TestInventoryCollectorResetsOnConfirmMoveAndResumes(t *testing.T) {
	coord := &Coordinator{}
	ctx, _, cancel := coord.readinessPass(context.Background())
	defer cancel()
	budget := readinessBudgetFrom(ctx)
	if budget == nil {
		t.Fatal("missing readiness pass")
	}
	if !budget.reserveNativeRPC(readinessMaxNativeRPC - 2) {
		t.Fatal("fixture could not prime its budget")
	}
	enum := &scriptedRevEnumerator{inner: twoPageEnumerator(t), moveFromObserve: 2, movedRev: 42}
	_, progress, err := (&InventoryCollector{}).Collect(ctx, enum, 7, 1, testLocal())
	if err == nil || err.Error() != HoldInventoryInProgress {
		t.Fatalf("err=%v, want %q", err, HoldInventoryInProgress)
	}
	if progress == nil || len(progress.Issues) != 0 {
		t.Fatalf("progress=%+v, want the valid reset: moved revision invalidates the prefix", progress)
	}
	ctx2, _, cancel2 := coord.readinessPass(context.Background())
	defer cancel2()
	fresh := twoPageEnumerator(t)
	fresh.revision = 42
	resumed, _, err := progress.Collect(ctx2, fresh, 7, 1, testLocal())
	if err != nil {
		t.Fatalf("resume: %v", err)
	}
	if resumed.Issues != 3 || resumed.Revision != 42 {
		t.Fatalf("resumed=%+v, want the three new-revision bodies sealed exactly once", resumed)
	}
	for i, want := range []int64{1, 2, 3} {
		if resumed.Indexes[i] != want {
			t.Fatalf("resumed=%+v, want exact ordered bodies", resumed.Indexes)
		}
	}
}

// A funded confirmation move retries transparently within the call:
// the replacement reservation arms the next confirmation and the
// recollected set seals at the new revision.
func TestInventoryCollectorRetriesConfirmMoveWhenFunded(t *testing.T) {
	coord := &Coordinator{}
	ctx, _, cancel := coord.readinessPass(context.Background())
	defer cancel()
	budget := readinessBudgetFrom(ctx)
	if budget == nil {
		t.Fatal("missing readiness pass")
	}
	if !budget.reserveNativeRPC(readinessMaxNativeRPC - 3) {
		t.Fatal("fixture could not prime its budget")
	}
	enum := &scriptedRevEnumerator{
		inner: twoPageEnumerator(t), moveFromObserve: 2, movedRev: 42,
		flipAfterFetches: 2, flipTo: 42,
	}
	sealed, _, err := (&InventoryCollector{}).Collect(ctx, enum, 7, 1, testLocal())
	if err != nil {
		t.Fatalf("funded retry: %v", err)
	}
	if sealed.Issues != 3 || sealed.Revision != 42 {
		t.Fatalf("sealed=%+v, want the recollected new-revision set", sealed)
	}
	if got := budget.nativeRPC.Load(); got != readinessMaxNativeRPC {
		t.Fatalf("nativeRPC=%d, want upfront two plus the replacement confirmation spent", got)
	}
}

// An exhausted pass holds pending before the first observation: no
// seal, the saved cursor kept, and zero native or local reads spent.
func TestBracketMaterialCollectHoldsPendingOnExhaustedBudget(t *testing.T) {
	coord := &Coordinator{}
	ctx, _, cancel := coord.readinessPass(context.Background())
	defer cancel()
	budget := readinessBudgetFrom(ctx)
	if budget == nil {
		t.Fatal("drive did not create its readiness pass")
	}
	if !budget.reserveNativeRPC(readinessMaxNativeRPC) {
		t.Fatal("fixture could not exhaust its budget")
	}
	saved := &InventoryCollector{Revision: 41, Cursor: "2"}
	enum := twoPageEnumerator(t)
	capture := &bracketCapture{locals: []factory.CompletionLocal{testLocal(), testLocal()}}
	out := bracketMaterialCollect(ctx, bracketDeps(enum, capture), 7, 1, saved)
	if out.hold != HoldInventoryInProgress || out.sealed != nil {
		t.Fatalf("out=%+v, want the pending hold with no seal", out)
	}
	if out.progress != saved {
		t.Fatalf("progress=%+v, want the saved cursor kept on budget exhaustion", out.progress)
	}
	if enum.observes != 0 {
		t.Fatalf("observes=%d, want zero native reads on an exhausted budget", enum.observes)
	}
	if capture.calls != 0 {
		t.Fatalf("capture calls=%d, want zero local reads on an exhausted budget", capture.calls)
	}
}

// One free unit cannot cover the upfront two-unit bracket
// reservation: the drive holds pending immediately with the saved
// cursor kept and zero observes, captures, or page reads — no
// expensive collection is started that a late quota failure could
// discard.
func TestBracketMaterialCollectHoldsPendingWhenConfirmUncharged(t *testing.T) {
	coord := &Coordinator{}
	ctx, _, cancel := coord.readinessPass(context.Background())
	defer cancel()
	budget := readinessBudgetFrom(ctx)
	if budget == nil {
		t.Fatal("drive did not create its readiness pass")
	}
	if !budget.reserveNativeRPC(readinessMaxNativeRPC - 1) {
		t.Fatal("fixture could not prime its budget")
	}
	saved := &InventoryCollector{Revision: 41, Cursor: "2"}
	enum := twoPageEnumerator(t)
	capture := &bracketCapture{locals: []factory.CompletionLocal{testLocal(), testLocal()}}
	out := bracketMaterialCollect(ctx, bracketDeps(enum, capture), 7, 1, saved)
	if out.hold != HoldInventoryInProgress || out.sealed != nil {
		t.Fatalf("out=%+v, want the pending hold with no seal", out)
	}
	if out.progress != saved {
		t.Fatalf("progress=%+v, want the saved cursor kept on the upfront preflight refusal", out.progress)
	}
	if enum.observes != 0 {
		t.Fatalf("observes=%d, want zero native reads on the upfront preflight refusal", enum.observes)
	}
	if capture.calls != 0 {
		t.Fatalf("capture calls=%d, want zero local reads on the upfront preflight refusal", capture.calls)
	}
	if enum.calls != 0 {
		t.Fatalf("page calls=%d, want zero page reads on the upfront preflight refusal", enum.calls)
	}
	if got := budget.nativeRPC.Load(); got != readinessMaxNativeRPC-1 {
		t.Fatalf("nativeRPC=%d, want the primed count unchanged (failed reserve consumes nothing)", got)
	}
}

// The collector prepays exactly its two revision observations
// (opening + confirming) upfront: a stable two-page seal over a
// fresh budget charges two units and observes exactly twice, with
// no page calling into an uncharged observation.
func TestInventoryCollectorPrepaysTwoObservations(t *testing.T) {
	coord := &Coordinator{}
	ctx, _, cancel := coord.readinessPass(context.Background())
	defer cancel()
	budget := readinessBudgetFrom(ctx)
	if budget == nil {
		t.Fatal("missing readiness pass")
	}
	enum := twoPageEnumerator(t)
	sealed, progress, err := (&InventoryCollector{}).Collect(ctx, enum, 7, 1, testLocal())
	if err != nil || sealed == nil || sealed.Issues != 3 {
		t.Fatalf("sealed=%+v progress=%+v err=%v, want the complete seal", sealed, progress, err)
	}
	if progress != nil {
		t.Fatalf("progress=%+v, want no resumable progress after a seal", progress)
	}
	if got := budget.nativeRPC.Load(); got != 2 {
		t.Fatalf("nativeRPC=%d, want exactly the two prepaid collector observations", got)
	}
	if enum.observes != 2 {
		t.Fatalf("observes=%d, want the opening plus the confirming observation", enum.observes)
	}
}

// An exhausted budget holds collecting before any native read: the
// incoming prefix/cursor returns untouched with the collecting hold,
// and zero observations or pages are spent.
func TestInventoryCollectorHoldsCollectingOnExhaustedBudget(t *testing.T) {
	coord := &Coordinator{}
	ctx, _, cancel := coord.readinessPass(context.Background())
	defer cancel()
	budget := readinessBudgetFrom(ctx)
	if budget == nil {
		t.Fatal("missing readiness pass")
	}
	if !budget.reserveNativeRPC(readinessMaxNativeRPC) {
		t.Fatal("fixture could not exhaust its budget")
	}
	saved := &InventoryCollector{Revision: 41, Cursor: "2"}
	enum := twoPageEnumerator(t)
	_, progress, err := saved.Collect(ctx, enum, 7, 1, testLocal())
	if err == nil || err.Error() != HoldInventoryInProgress {
		t.Fatalf("err=%v, want %q", err, HoldInventoryInProgress)
	}
	if progress != saved {
		t.Fatalf("progress=%+v, want the saved prefix/cursor untouched", progress)
	}
	if enum.observes != 0 || enum.calls != 0 {
		t.Fatalf("observes=%d calls=%d, want zero native spend on the upfront refusal", enum.observes, enum.calls)
	}
	if got := budget.nativeRPC.Load(); got != readinessMaxNativeRPC {
		t.Fatalf("nativeRPC=%d, want the exhausted count unchanged", got)
	}
}

// A moved revision restarts under a tight budget: the re-bracket
// spends the confirming prepayment and the later confirmation
// reserves anew, sealing exactly at the cap with three observations
// and no uncharged call.
func TestInventoryCollectorRestartReservesConfirmationAnew(t *testing.T) {
	coord := &Coordinator{}
	ctx, _, cancel := coord.readinessPass(context.Background())
	defer cancel()
	budget := readinessBudgetFrom(ctx)
	if budget == nil {
		t.Fatal("missing readiness pass")
	}
	if !budget.reserveNativeRPC(readinessMaxNativeRPC - 3) {
		t.Fatal("fixture could not prime its budget")
	}
	enum := twoPageEnumerator(t)
	enum.moveRev = func(call int) int64 {
		if call >= 2 {
			return 42
		}
		return 41
	}
	sealed, progress, err := (&InventoryCollector{}).Collect(ctx, enum, 7, 1, testLocal())
	if err != nil || sealed == nil {
		t.Fatalf("sealed=%+v progress=%+v err=%v, want the post-move seal", sealed, progress, err)
	}
	if progress != nil {
		t.Fatalf("progress=%+v, want no resumable progress after a seal", progress)
	}
	if sealed.Revision != 42 || sealed.Issues != 3 {
		t.Fatalf("sealed=%+v, want the full post-move set, never a subset", sealed)
	}
	if got := budget.nativeRPC.Load(); got != readinessMaxNativeRPC {
		t.Fatalf("nativeRPC=%d, want the upfront two plus the re-reserved confirmation", got)
	}
	if enum.observes != 3 {
		t.Fatalf("observes=%d, want opening plus re-bracket plus confirmation", enum.observes)
	}
}

// When the re-reserved confirmation finds no free unit, the
// collection holds collecting with resumable progress: the revision
// move already reset the cursor (snapshots never combine across
// revisions), the confirming observation never runs, and nothing
// seals or charges a decision.
/*
 * TestInventoryCollectorHoldsPrefixWhenConfirmationUncharged was
 * removed: its "3 final issues / cursor 2 is resumable" assertion was
 * false — resuming that prefix re-appended the final page and sealed
 * 4 bodies. Replaced by
 * TestInventoryCollectorResumesPrefixAfterBudgetHold (actual
 * fresh-pass resume, digest-equal to a clean seal),
 * TestInventoryCollectorResetsOnConfirmMoveAndResumes, and
 * TestInventoryCollectorRetriesConfirmMoveWhenFunded.
 */

// The completion page cap mirrors the native transport bound: the
// control boundary stays adapter-free while the limit math cannot
// drift from the producer it protects.
func TestCompletionPageCapMatchesNativePageLimit(t *testing.T) {
	if completionNativePageCap != forgejo.SnapshotPageLimit {
		t.Fatalf("completionNativePageCap=%d, want the native page bound %d", completionNativePageCap, forgejo.SnapshotPageLimit)
	}
}
