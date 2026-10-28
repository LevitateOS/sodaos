package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"testing"

	extensions "forgejo.org/extension-sdk"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/factory/control"
	"github.com/levitateos/sodaos/internal/forgejo"
	"github.com/levitateos/sodaos/internal/store"
)

// scriptedBudgetReader serves a deterministic synthetic issue set
// through the REAL production snapshot adapter and bracket reads
// while counting every native call, plus the served current-target
// tip for the production serving capture. It proves budgeted
// multi-pass resumption, not native/provider/semantic capture.
type scriptedBudgetReader struct {
	total int
	calls int
	tip   string
}

func (r *scriptedBudgetReader) ReadNativeRevision(context.Context) (extensions.NativeRevisionObservation, error) {
	r.calls++
	return extensions.NativeRevisionObservation{Revision: 41, Idle: true}, nil
}

func (r *scriptedBudgetReader) ReadSnapshot(_ context.Context, _ extensions.CredentialFile, req forgejo.SnapshotRequest) (forgejo.NativeSnapshot, error) {
	r.calls++
	if len(req.Families) == 1 && req.Families[0] == forgejo.FamilyRefs {
		refs := []forgejo.RefEvidence{}
		for _, want := range req.Refs {
			if r.tip == "" {
				continue
			}
			refs = append(refs, forgejo.RefEvidence{Ref: want, OID: r.tip, Exists: true, Visible: true, Complete: true})
		}
		return forgejo.NativeSnapshot{Revision: 41, RepositoryID: req.RepositoryID, Refs: refs}, nil
	}
	if len(req.Families) == 1 && req.Families[0] == forgejo.FamilyRepositoryIssues {
		start := 0
		if req.Cursor != "" {
			v, err := strconv.Atoi(req.Cursor)
			if err != nil {
				return forgejo.NativeSnapshot{}, err
			}
			start = v
		}
		limit := req.Limit
		if limit < 1 || limit > 50 {
			return forgejo.NativeSnapshot{}, errors.New("invalid requested limit")
		}
		end := start + limit
		if end > r.total {
			end = r.total
		}
		items := []forgejo.RepositoryIssueItem{}
		for n := start + 1; n <= end; n++ {
			v := strconv.Itoa(n)
			items = append(items, forgejo.RepositoryIssueItem{IssueID: v + "0", IssueIndex: v, Visible: true})
		}
		page := &forgejo.RepositoryIssuesPage{Items: items, Complete: end == r.total}
		if !page.Complete {
			page.NextCursor = strconv.Itoa(end)
		}
		return forgejo.NativeSnapshot{Revision: 41, RepositoryID: req.RepositoryID, RepositoryIssues: page}, nil
	}
	snapshot := completionBody(req.IssueIndex, "title "+req.IssueIndex, "body "+req.IssueIndex)
	snapshot.Revision = 41
	snapshot.RepositoryID = req.RepositoryID
	return snapshot, nil
}

// A stable inventory larger than one pass worth of native calls
// persists its prefix cursor across DriveCompletions passes and
// eventually seals every exact body with per-pass actual native RPC
// at or under the readiness cap — never a subset seal, never a paid
// choice before complete. Both local captures run the REAL production
// serving capture (refs-only current-target read plus the declared
// NATIVE96 fake source store), so the per-pass call counter includes
// every refs call plus every inventory call.
func TestCompletionDriveServingCallsAndResume(t *testing.T) {
	ctx := context.Background()
	admin, ok := store.TestFixtureDSN()
	if !ok {
		t.Skip("SODA_PG fixture unavailable")
	}
	db, dsn, cleanup, err := store.OpenEphemeral(ctx, admin)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	raw, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer raw.Close()
	event := factory.CompletionEvent{
		ID: factory.NewID(), PublicationID: factory.NewID(),
		MergedCommit: strings.Repeat("c", 40), ClosingOccurrence: "budget-resume-130",
		AcceptanceID: "d" + strings.Repeat("a", 24), Repository: 7, Issue: 3, Generation: 1,
	}
	rec := store.ReassessmentRecord{
		Cycle: factory.ReassessmentCycle{
			EventID: event.ID, CycleID: factory.NewID(),
			State: factory.ReassessmentCollecting, Repository: 7, Generation: 1,
		},
		Completion: event, Issue: 3,
	}
	if err = rec.Validate(); err != nil {
		t.Fatal(err)
	}
	eventRaw, _ := json.Marshal(event)
	recordRaw, _ := json.Marshal(rec)
	if _, err = raw.Exec(`INSERT INTO factory_completions(repository,generation,event,publication,occurrence,data) VALUES(7,1,$1,$2,$3,$4)`,
		event.ID, event.PublicationID, event.ClosingOccurrence, string(eventRaw)); err != nil {
		t.Fatal(err)
	}
	if _, err = raw.Exec(`INSERT INTO factory_reassessments(cycle,repository,generation,event,state,data) VALUES($1,7,1,$2,'collecting',$3)`,
		rec.Cycle.CycleID, event.ID, string(recordRaw)); err != nil {
		t.Fatal(err)
	}
	reader := &scriptedBudgetReader{total: 130, tip: strings.Repeat("c", 40)}
	source := &CompletionSnapshotSource{Reader: reader, Store: newFakeCompletionSourceStoreNATIVE96(completionTestLocal())}
	coord := &control.Coordinator{Store: db}
	prior := 0
	for pass := 0; pass < 10; pass++ {
		before := reader.calls
		report := coord.DriveCompletions(ctx, control.CompletionDriveDeps{
			Enumerator: source, CaptureLocal: source.CaptureLocal,
		})
		used := reader.calls - before
		t.Logf("pass%d actual nativeRPC=%d", pass, used)
		if used > 156 {
			t.Fatalf("production completion drive used %d actual native RPC in one pass, cap 156", used)
		}
		stored, err := db.ReassessmentRecord(ctx, rec.Cycle.CycleID)
		if err != nil {
			t.Fatal(err)
		}
		if len(report.Sealed) > 0 {
			if len(stored.Evidence) != 130 || stored.EvidenceIssues != 130 {
				t.Fatalf("sealed incomplete set: bodies %d count %d", len(stored.Evidence), stored.EvidenceIssues)
			}
			for i, v := range stored.Evidence {
				if v.Index != int64(i+1) || v.Title != "title "+strconv.Itoa(i+1) || v.Body != "body "+strconv.Itoa(i+1) {
					t.Fatal("sealed body/index lost or changed")
				}
			}
			if _, err := control.DecisionStateFromRecord(stored); err != nil {
				t.Fatalf("resumed seal does not validate: %v", err)
			}
			if pass == 0 {
				t.Fatal("130 issues improperly sealed in a single bounded pass")
			}
			t.Logf("all 130 exact bodies persisted/sealed after %d passes, no chooser invoked", pass+1)
			return
		}
		if stored.Cycle.State != factory.ReassessmentCollecting || len(stored.CollectorProgress) == 0 {
			t.Fatalf("no resumable collecting prefix: %+v", report)
		}
		var progress control.InventoryCollector
		if err = json.Unmarshal(stored.CollectorProgress, &progress); err != nil {
			t.Fatal(err)
		}
		if len(progress.Issues) <= prior {
			t.Fatalf("prefix did not advance: prior %d current %d", prior, len(progress.Issues))
		}
		prior = len(progress.Issues)
	}
	t.Fatal("stable 130-issue inventory never finished within bounded passes")
}

// continueReleaseChooser answers exactly one deterministic funded continue
// per cycle after asserting the sealed substance: 130 judged bodies with
// sealed local bindings, never hashes. It counts every consultation so the
// release test proves no duplicated paid decision across continuation and
// replay.
type continueReleaseChooser struct {
	calls int
}

func (c *continueReleaseChooser) Select(_ context.Context, state any, _ string, choices map[string]any) (factory.PriorityAnswer, error) {
	c.calls++
	decision, ok := state.(control.ReassessmentDecisionState)
	if !ok || len(decision.Evidence) != 130 || !decision.Local.Sealed() {
		return factory.PriorityAnswer{}, errors.New("decision judges without the 130 sealed bodies")
	}
	for _, body := range decision.Evidence {
		if err := body.Validate(); err != nil {
			return factory.PriorityAnswer{}, err
		}
	}
	return factory.PriorityAnswer{
		Selected: factory.ReassessmentOptionContinue, Model: "lane-continue-088", Confidence: 0.9,
		Probabilities: map[string]float64{factory.ReassessmentOptionContinue: 0.9, factory.ReassessmentOptionReassess: 0.1},
	}, nil
}

// A decided continue must re-observe its 130-issue inventory through the
// REAL production snapshot adapter across bounded passes and eventually
// release: the fresh prefix persists via the deciding CAS (never a subset
// seal, never a second paid decision), per-pass actual native RPC stays at
// or under the readiness cap, and continuation/replay never duplicates the
// paid receipt. Both local captures run the REAL production serving
// capture (refs-only current-target read plus the declared NATIVE96 fake
// source store) — this proves budgeted multi-pass resumption with
// production capture costs, not native/provider or semantic capture.
func TestCompletionDriveContinueReleaseResumesAcrossPasses(t *testing.T) {
	ctx := context.Background()
	admin, ok := store.TestFixtureDSN()
	if !ok {
		t.Skip("SODA_PG fixture unavailable")
	}
	db, dsn, cleanup, err := store.OpenEphemeral(ctx, admin)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	raw, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer raw.Close()
	event := factory.CompletionEvent{
		ID: factory.NewID(), PublicationID: factory.NewID(),
		MergedCommit: strings.Repeat("c", 40), ClosingOccurrence: "continue-release-130",
		AcceptanceID: "d" + strings.Repeat("a", 24), Repository: 7, Issue: 3, Generation: 1,
	}
	rec := store.ReassessmentRecord{
		Cycle: factory.ReassessmentCycle{
			EventID: event.ID, CycleID: factory.NewID(),
			State: factory.ReassessmentCollecting, Repository: 7, Generation: 1,
		},
		Completion: event, Issue: 3,
	}
	if err = rec.Validate(); err != nil {
		t.Fatal(err)
	}
	eventRaw, _ := json.Marshal(event)
	recordRaw, _ := json.Marshal(rec)
	if _, err = raw.Exec(`INSERT INTO factory_completions(repository,generation,event,publication,occurrence,data) VALUES(7,1,$1,$2,$3,$4)`,
		event.ID, event.PublicationID, event.ClosingOccurrence, string(eventRaw)); err != nil {
		t.Fatal(err)
	}
	if _, err = raw.Exec(`INSERT INTO factory_reassessments(cycle,repository,generation,event,state,data) VALUES($1,7,1,$2,'collecting',$3)`,
		rec.Cycle.CycleID, event.ID, string(recordRaw)); err != nil {
		t.Fatal(err)
	}
	reader := &scriptedBudgetReader{total: 130, tip: strings.Repeat("c", 40)}
	source := &CompletionSnapshotSource{Reader: reader, Store: newFakeCompletionSourceStoreNATIVE96(completionTestLocal())}
	chooser := &continueReleaseChooser{}
	coord := &control.Coordinator{Store: db, Planning: &control.PlanningJudgments{Chooser: chooser}}
	deps := control.CompletionDriveDeps{Enumerator: source, CaptureLocal: source.CaptureLocal}
	sealed := false
	for pass := 0; pass < 14; pass++ {
		before := reader.calls
		report := coord.DriveCompletions(ctx, deps)
		used := reader.calls - before
		stored, err := db.ReassessmentRecord(ctx, rec.Cycle.CycleID)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("pass%d actual nativeRPC=%d state=%s decision=%s prefixBytes=%d chooserCalls=%d report=%+v",
			pass, used, stored.Cycle.State, stored.Cycle.Decision, len(stored.CollectorProgress), chooser.calls, report)
		if used > 156 {
			t.Fatalf("production completion drive used %d actual native RPC in one pass, cap 156", used)
		}
		if len(report.Sealed) > 0 {
			if len(stored.Evidence) != 130 || stored.EvidenceIssues != 130 {
				t.Fatalf("sealed incomplete set: bodies %d count %d", len(stored.Evidence), stored.EvidenceIssues)
			}
			if _, err := control.DecisionStateFromRecord(stored); err != nil {
				t.Fatalf("resumed seal does not validate: %v", err)
			}
			if pass == 0 {
				t.Fatal("130 issues improperly sealed in a single bounded pass")
			}
			sealed = true
		}
		if len(report.Released) > 0 {
			if !sealed {
				t.Fatal("released without a complete seal")
			}
			if stored.Cycle.State != factory.ReassessmentReleased || stored.Cycle.Decision != factory.ReassessmentOptionContinue {
				t.Fatalf("record=%+v, want the released continue", stored)
			}
			if len(stored.CollectorProgress) != 0 {
				t.Fatalf("prefixBytes=%d, want the deciding prefix cleared on release", len(stored.CollectorProgress))
			}
			if chooser.calls != 1 {
				t.Fatalf("calls=%d, want exactly one paid continue decision", chooser.calls)
			}
			if len(stored.Cycle.Ledger.DecisionCalls) != 1 {
				t.Fatalf("decisionCalls=%d, want exactly one paid decision call across continuation",
					len(stored.Cycle.Ledger.DecisionCalls))
			}
			if _, err := control.DecisionStateFromRecord(stored); err != nil {
				t.Fatalf("released record does not validate: %v", err)
			}
			// Replay after release must not charge again: the released
			// cycle rests and the receipt is never duplicated.
			for replay := 0; replay < 2; replay++ {
				before := reader.calls
				report := coord.DriveCompletions(ctx, deps)
				if used := reader.calls - before; used > 156 {
					t.Fatalf("replay used %d actual native RPC in one pass, cap 156", used)
				}
				if len(report.Released) != 0 || len(report.Decided) != 0 {
					t.Fatalf("replay report=%+v, want no second release or decision", report)
				}
			}
			if chooser.calls != 1 {
				t.Fatalf("calls=%d after replay, want still exactly one paid decision", chooser.calls)
			}
			stored, err := db.ReassessmentRecord(ctx, rec.Cycle.CycleID)
			if err != nil {
				t.Fatal(err)
			}
			if stored.Cycle.State != factory.ReassessmentReleased {
				t.Fatalf("record=%+v, want the release to stand across replay", stored)
			}
			t.Logf("130-issue continue released after %d passes with one paid decision", pass+1)
			return
		}
	}
	t.Fatal("stable 130-issue inventory never released its continue across bounded passes")
}

// staleRefusalChooser answers one deterministic funded continue per cycle
// after asserting the small sealed substance: 3 judged bodies with sealed
// local bindings, never hashes.
type staleRefusalChooser struct {
	calls int
}

func (c *staleRefusalChooser) Select(_ context.Context, state any, _ string, choices map[string]any) (factory.PriorityAnswer, error) {
	c.calls++
	decision, ok := state.(control.ReassessmentDecisionState)
	if !ok || len(decision.Evidence) != 3 || !decision.Local.Sealed() {
		return factory.PriorityAnswer{}, errors.New("decision judges without the 3 sealed bodies")
	}
	return factory.PriorityAnswer{
		Selected: factory.ReassessmentOptionContinue, Model: "lane-continue-088", Confidence: 0.9,
		Probabilities: map[string]float64{factory.ReassessmentOptionContinue: 0.9, factory.ReassessmentOptionReassess: 0.1},
	}, nil
}

// completionTestLocalForRepo clones the synthetic fixture local for one
// scenario repository: every repo-bearing section follows the cycle
// repository and the scope digest is recomputed, so the seal CAS accepts
// the bindings it persists.
func completionTestLocalForRepo(t *testing.T, repository int64) factory.CompletionLocal {
	t.Helper()
	local := completionTestLocal()
	material := *local.Material
	material.Repository = repository
	for i := range material.Acceptances {
		material.Acceptances[i].Repository = repository
	}
	for i := range material.Controls {
		material.Controls[i].Repository = repository
	}
	digest, err := material.Digest()
	if err != nil {
		t.Fatal(err)
	}
	local.Material = &material
	local.ScopeDigest = digest
	if err := local.Validate(); err != nil {
		t.Fatalf("scenario local invalid: %v", err)
	}
	return local
}

func seedStaleRefusalCycle(t *testing.T, raw *sql.DB, repository int64, occurrence string) string {
	t.Helper()
	event := factory.CompletionEvent{
		ID: factory.NewID(), PublicationID: factory.NewID(),
		MergedCommit: strings.Repeat("c", 40), ClosingOccurrence: occurrence,
		AcceptanceID: "d" + strings.Repeat("a", 24), Repository: repository, Issue: 3, Generation: 1,
	}
	rec := store.ReassessmentRecord{
		Cycle: factory.ReassessmentCycle{
			EventID: event.ID, CycleID: factory.NewID(),
			State: factory.ReassessmentCollecting, Repository: repository, Generation: 1,
		},
		Completion: event, Issue: 3,
	}
	if err := rec.Validate(); err != nil {
		t.Fatal(err)
	}
	eventRaw, _ := json.Marshal(event)
	recordRaw, _ := json.Marshal(rec)
	if _, err := raw.Exec(`INSERT INTO factory_completions(repository,generation,event,publication,occurrence,data) VALUES($1,1,$2,$3,$4,$5)`,
		repository, event.ID, event.PublicationID, event.ClosingOccurrence, string(eventRaw)); err != nil {
		t.Fatal(err)
	}
	if _, err := raw.Exec(`INSERT INTO factory_reassessments(cycle,repository,generation,event,state,data) VALUES($1,$2,1,$3,'collecting',$4)`,
		rec.Cycle.CycleID, repository, event.ID, string(recordRaw)); err != nil {
		t.Fatal(err)
	}
	return rec.Cycle.CycleID
}

// A continue never releases from a stale world: a changed body, a moved
// native revision, or a changed local-head context each refuse with
// attention (never a release, never a second paid decision for the same
// cycle). Capture Local is the synthetic completionTestLocal fixture here
// — this proves the refusal dimensions, not native/provider capture.
func TestCompletionDriveRefusesStaleContinueRelease(t *testing.T) {
	ctx := context.Background()
	admin, ok := store.TestFixtureDSN()
	if !ok {
		t.Skip("SODA_PG fixture unavailable")
	}
	db, dsn, cleanup, err := store.OpenEphemeral(ctx, admin)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	raw, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer raw.Close()
	chooser := &staleRefusalChooser{}
	coord := &control.Coordinator{Store: db, Planning: &control.PlanningJudgments{Chooser: chooser}}
	// One completion per repository-generation row: each scenario seals
	// exactly its own cycle with no cross-talk between worlds.
	scenario := func(repository int64, occurrence string, mutate func(fake *completionReaderFake, current *factory.CompletionLocal), wantReason string) {
		t.Helper()
		fake := completionTwoPageFake()
		source := NewCompletionSnapshotSource(nil)
		source.Reader = fake
		current := completionTestLocalForRepo(t, repository)
		deps := control.CompletionDriveDeps{
			Enumerator: source,
			CaptureLocal: func(context.Context, int64) (factory.CompletionLocal, error) {
				return current, nil
			},
		}
		drive := func() control.CompletionDriveReport {
			before := fake.calls + fake.revs
			report := coord.DriveCompletions(ctx, deps)
			if used := fake.calls + fake.revs - before; used > 156 {
				t.Fatalf("used %d actual native RPC in one pass, cap 156", used)
			}
			return report
		}
		cycleID := seedStaleRefusalCycle(t, raw, repository, occurrence)
		if report := drive(); len(report.Sealed) != 1 || report.Sealed[0] != cycleID {
			t.Fatalf("report=%+v, want the 3-issue seal", report)
		}
		mutate(fake, &current)
		if err := current.Validate(); err != nil {
			t.Fatalf("mutated world invalid: %v", err)
		}
		beforeCalls := chooser.calls
		report := drive()
		want := cycleID + ":" + wantReason
		found := false
		for _, held := range report.Held {
			found = found || held == want
		}
		if !found || len(report.Released) != 0 {
			t.Fatalf("report=%+v, want %q with no release", report, want)
		}
		stored, err := db.ReassessmentRecord(ctx, cycleID)
		if err != nil {
			t.Fatal(err)
		}
		if stored.Cycle.State != factory.ReassessmentAttention {
			t.Fatalf("record=%+v, want attention on the stale world", stored)
		}
		if stored.Cycle.Decision != factory.ReassessmentOptionContinue {
			t.Fatalf("record=%+v, want the recorded continue to stand", stored)
		}
		if chooser.calls != beforeCalls+1 {
			t.Fatalf("calls=%d, want exactly one paid decision for the refused cycle", chooser.calls)
		}
	}

	// Changed body after the seal refuses on the evidence dimension.
	scenario(7, "stale-body", func(fake *completionReaderFake, current *factory.CompletionLocal) {
		fake.bodies["2"] = completionBody("2", "title 2 changed", "body 2 changed")
	}, "completion evidence changed during reassessment")
	// Changed local-head context after the seal refuses on the source
	// dimension; the native world is untouched so only the local moves.
	scenario(8, "stale-local", func(_ *completionReaderFake, current *factory.CompletionLocal) {
		current.SourceCommit = strings.Repeat("e", 40)
	}, "completion source changed during reassessment")
	// Moved native revision after the seal never mixes revisions: the
	// fresh bracket seals at the new revision and its digest refuses on
	// the evidence dimension.
	scenario(9, "stale-revision", func(fake *completionReaderFake, current *factory.CompletionLocal) {
		fake.revision = 42
	}, "completion evidence changed during reassessment")
}

// continue48Chooser answers exactly one deterministic funded continue
// per cycle after asserting the sealed substance: 48 judged bodies with
// sealed local bindings, never hashes. It counts every consultation so
// the release test proves no duplicated paid decision across
// continuation and replay.
type continue48Chooser struct {
	calls int
}

func (c *continue48Chooser) Select(_ context.Context, state any, _ string, choices map[string]any) (factory.PriorityAnswer, error) {
	c.calls++
	decision, ok := state.(control.ReassessmentDecisionState)
	if !ok || len(decision.Evidence) != 48 || !decision.Local.Sealed() {
		return factory.PriorityAnswer{}, errors.New("decision judges without the 48 sealed bodies")
	}
	for _, body := range decision.Evidence {
		if err := body.Validate(); err != nil {
			return factory.PriorityAnswer{}, err
		}
	}
	return factory.PriorityAnswer{
		Selected: factory.ReassessmentOptionContinue, Model: "lane-continue-096", Confidence: 0.9,
		Probabilities: map[string]float64{factory.ReassessmentOptionContinue: 0.9, factory.ReassessmentOptionReassess: 0.1},
	}, nil
}

// A naive single pass over 48 issues would spend 7+147+3=157 actual
// native RPC against the 156 cap (two outer plus two collector plus
// the opening capture, one page bracket plus 48 three-call bodies,
// the closing capture). The fixed-8 prepayment plus the closing-3
// hold before the limit calc instead progresses exactly 47+1 across
// passes: the first pass holds collecting with the 47-prefix, the
// second seals, the decided continue releases fresh — every pass at
// or under 156 actual RPC with exactly one paid decision, one ledger
// call, and a replay-free release. Both captures run the REAL
// production serving capture, so refs reads ride the same counter as
// inventory reads.
func TestCompletionDriveCompletes48AcrossTwoPasses(t *testing.T) {
	ctx := context.Background()
	admin, ok := store.TestFixtureDSN()
	if !ok {
		t.Skip("SODA_PG fixture unavailable")
	}
	db, dsn, cleanup, err := store.OpenEphemeral(ctx, admin)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	raw, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer raw.Close()
	cycleID := seedStaleRefusalCycle(t, raw, 7, "budget-48")
	reader := &scriptedBudgetReader{total: 48, tip: strings.Repeat("c", 40)}
	source := &CompletionSnapshotSource{Reader: reader, Store: newFakeCompletionSourceStoreNATIVE96(completionTestLocal())}
	chooser := &continue48Chooser{}
	coord := &control.Coordinator{Store: db, Planning: &control.PlanningJudgments{Chooser: chooser}}
	deps := control.CompletionDriveDeps{Enumerator: source, CaptureLocal: source.CaptureLocal}
	sealed := false
	for pass := 0; pass < 10; pass++ {
		before := reader.calls
		report := coord.DriveCompletions(ctx, deps)
		used := reader.calls - before
		if used > 156 {
			t.Fatalf("pass%d used %d actual native RPC, cap 156", pass, used)
		}
		stored, err := db.ReassessmentRecord(ctx, cycleID)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("pass%d actual nativeRPC=%d state=%s decision=%s prefixBytes=%d chooserCalls=%d report=%+v",
			pass, used, stored.Cycle.State, stored.Cycle.Decision, len(stored.CollectorProgress), chooser.calls, report)
		if pass == 0 {
			if len(report.Sealed) != 0 {
				t.Fatalf("report=%+v, want no seal on the first pass: 48 exceeds one bounded pass", report)
			}
			if stored.Cycle.State != factory.ReassessmentCollecting || len(stored.CollectorProgress) == 0 {
				t.Fatalf("no resumable collecting prefix after the first pass: %+v", report)
			}
			var progress control.InventoryCollector
			if err = json.Unmarshal(stored.CollectorProgress, &progress); err != nil {
				t.Fatal(err)
			}
			if len(progress.Issues) != 47 || progress.Cursor != "47" {
				t.Fatalf("prefix issues=%d cursor=%q, want exactly the 47-prefix with its cursor", len(progress.Issues), progress.Cursor)
			}
			continue
		}
		if len(report.Sealed) > 0 {
			if len(stored.Evidence) != 48 || stored.EvidenceIssues != 48 {
				t.Fatalf("sealed incomplete set: bodies %d count %d", len(stored.Evidence), stored.EvidenceIssues)
			}
			if pass != 1 {
				t.Fatalf("sealed on pass %d, want the exact 47+1 shape across two passes", pass)
			}
			sealed = true
		}
		if len(report.Released) > 0 {
			if !sealed {
				t.Fatal("released without a complete seal")
			}
			if stored.Cycle.State != factory.ReassessmentReleased || stored.Cycle.Decision != factory.ReassessmentOptionContinue {
				t.Fatalf("record=%+v, want the released continue", stored)
			}
			if len(stored.CollectorProgress) != 0 {
				t.Fatalf("prefixBytes=%d, want the deciding prefix cleared on release", len(stored.CollectorProgress))
			}
			if chooser.calls != 1 {
				t.Fatalf("calls=%d, want exactly one paid continue decision", chooser.calls)
			}
			if len(stored.Cycle.Ledger.DecisionCalls) != 1 {
				t.Fatalf("decisionCalls=%d, want exactly one paid decision call across continuation",
					len(stored.Cycle.Ledger.DecisionCalls))
			}
			for replay := 0; replay < 2; replay++ {
				before := reader.calls
				report := coord.DriveCompletions(ctx, deps)
				if used := reader.calls - before; used > 156 {
					t.Fatalf("replay used %d actual native RPC in one pass, cap 156", used)
				}
				if len(report.Released) != 0 || len(report.Decided) != 0 {
					t.Fatalf("replay report=%+v, want no second release or decision", report)
				}
			}
			if chooser.calls != 1 {
				t.Fatalf("calls=%d after replay, want still exactly one paid decision", chooser.calls)
			}
			t.Logf("48-issue inventory sealed 47+1 and released with one paid decision")
			return
		}
	}
	t.Fatal("stable 48-issue inventory never released its continue across bounded passes")
}

// A native tip change between the opening and closing serving captures
// defers with the named material hold and a reset cursor: the
// unconfirmed seal never persists, and the next drive reseals clean at
// the new tip. The inventory revision never moves here — only the
// current-target tip does, between snapshot calls 1 (opening refs) and
// 7 (closing refs: one refs plus two pages plus three bodies precede
// it).
func TestCompletionDriveDefersOnTipChangeDuringCapturePair(t *testing.T) {
	ctx := context.Background()
	admin, ok := store.TestFixtureDSN()
	if !ok {
		t.Skip("SODA_PG fixture unavailable")
	}
	db, dsn, cleanup, err := store.OpenEphemeral(ctx, admin)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	raw, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer raw.Close()
	cycleID := seedStaleRefusalCycle(t, raw, 7, "tip-pair-move")
	tipA, tipB := strings.Repeat("c", 40), strings.Repeat("e", 40)
	fake := completionTwoPageFake()
	fake.refTipMove = func(snapCalls int) string {
		if snapCalls >= 7 {
			return tipB
		}
		return tipA
	}
	source := NewCompletionSnapshotSource(nil)
	source.Reader = fake
	source.Store = newFakeCompletionSourceStoreNATIVE96(completionTestLocalForRepo(t, 7))
	coord := &control.Coordinator{Store: db}
	deps := control.CompletionDriveDeps{Enumerator: source, CaptureLocal: source.CaptureLocal}
	before := fake.calls + fake.revs
	report := coord.DriveCompletions(ctx, deps)
	if used := fake.calls + fake.revs - before; used > 156 {
		t.Fatalf("used %d actual native RPC in one pass, cap 156", used)
	}
	want := cycleID + ":" + control.HoldInventoryMaterial
	found := false
	for _, held := range report.Held {
		found = found || held == want
	}
	if !found || len(report.Sealed) != 0 {
		t.Fatalf("report=%+v, want %q with no seal", report, want)
	}
	stored, err := db.ReassessmentRecord(ctx, cycleID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Cycle.State != factory.ReassessmentCollecting || len(stored.CollectorProgress) != 0 {
		t.Fatalf("record=%+v, want collecting with a reset cursor after the tip move", stored)
	}
	fake.refTipMove = nil
	fake.refTip = tipB
	before = fake.calls + fake.revs
	report = coord.DriveCompletions(ctx, deps)
	if used := fake.calls + fake.revs - before; used > 156 {
		t.Fatalf("used %d actual native RPC in one pass, cap 156", used)
	}
	if len(report.Sealed) != 1 || report.Sealed[0] != cycleID {
		t.Fatalf("report=%+v, want the clean reseal at the new tip", report)
	}
	stored, err = db.ReassessmentRecord(ctx, cycleID)
	if err != nil {
		t.Fatal(err)
	}
	if len(stored.Evidence) != 3 || stored.Local.SourceCommit != tipB {
		t.Fatalf("bodies=%d source=%q, want the 3-issue seal bound at the new tip", len(stored.Evidence), stored.Local.SourceCommit)
	}
}
