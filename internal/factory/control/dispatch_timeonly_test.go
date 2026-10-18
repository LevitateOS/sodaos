package control

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
)

// Serving admission/launch for Issue-zero time-only roots (review-018
// finding 2) against real PostgreSQL: DispatchTimeOnlyRoot plans,
// records, funds and launches intent decompose and completion
// reassess roots through the production route — never by calling the
// packet directly. A bounded host double proves persistence and
// funding precede the host call (the double reads the committed rows
// from inside the launch), the host call runs outside the transaction,
// and replay cannot create a second run or charge.

// timeOnlyOrderHost installs a launch double that proves ordering: it
// reads the committed assignment and funded allowance rows from inside
// the host call, which only the post-commit launch can observe.
func timeOnlyOrderHost(fx dispatchFixture, observed *bool, check func(ctx context.Context, in project.FactoryLaunch)) {
	fx.host.launch = func(in project.FactoryLaunch) (project.FactoryState, error) {
		check(context.Background(), in)
		*observed = true
		return project.FactoryState{ID: in.Run.ID, Project: in.Run.Project, Role: in.Run.Role, Phase: project.FactoryCompleted}, nil
	}
}

func timeOnlyIntentSubject() factory.ExecutionSubject {
	return factory.ExecutionSubject{
		Kind:   factory.SubjectIntent,
		Intent: &factory.IntentSubject{IntentID: "d777777777777777777777777", RevisionID: "d888888888888888888888888", ScopeDigest: strings.Repeat("d", 64), OwnerID: 7},
	}
}

func recordTimeOnlyIntent(t *testing.T, ctx context.Context, db *store.Store) {
	t.Helper()
	now := time.Now().UTC().Truncate(time.Second)
	revision := factory.IntentRevision{
		ID: "d888888888888888888888888", Intent: "d777777777777777777777777", Owner: 7, Repository: 7,
		Objective: "plan the recorded draft", State: "draft",
		MaxIssues: 10, CreatedUnix: now.Unix(),
	}
	if err := db.SaveIntentRevision(ctx, revision); err != nil {
		t.Fatalf("intent revision not recorded: %v", err)
	}
}

func fundTimeOnlyPolicy(t *testing.T, ctx context.Context, db *store.Store, repo int64) {
	t.Helper()
	policy, err := db.RepositoryPolicy(ctx, repo)
	if err != nil {
		t.Fatal(err)
	}
	policy.IntentMinutes = 45
	policy.ReassessmentMinutes = 30
	if err := db.SaveRepositoryPolicy(ctx, policy); err != nil {
		t.Fatal(err)
	}
}

func TestDispatchTimeOnlyRootLaunchesIntent(t *testing.T) {
	ctx := context.Background()
	db, _ := dispatchTestDB(t)
	fx := dispatchSeed(t, db)
	fundTimeOnlyPolicy(t, ctx, db, fx.repo)
	recordTimeOnlyIntent(t, ctx, db)
	deps := fx.deps()
	occupancy, err := snapshotOccupancy(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	var observed bool
	timeOnlyOrderHost(fx, &observed, func(ctx context.Context, in project.FactoryLaunch) {
		a, err := db.AssignmentByRun(ctx, in.Run.ID)
		if err != nil || a.Phase != project.ExecutionDecompose || a.Issue != 0 || a.AttemptRoot != a.ID || a.PublicationAssignment != "" {
			t.Fatalf("host call precedes its packet: assignment=%+v err=%v", a, err)
		}
		allowance, err := db.IntentAttemptAllowance(ctx, fx.repo, factory.IntentAttemptScope{IntentID: "d777777777777777777777777", OwnerID: 7}, a.ID)
		if err != nil || allowance.Limits.ActiveMinutes != 45 || !allowance.Active {
			t.Fatalf("host call precedes its funding: allowance=%+v err=%v", allowance, err)
		}
	})
	report := newDispatchReport()
	DispatchTimeOnlyRoot(ctx, deps, &occupancy, fx.repo, timeOnlyIntentSubject(), strings.Repeat("c", 40), []byte("plan the recorded draft"), &report)
	if len(report.Errors) != 0 || len(report.Waits) != 0 || len(report.Launched) != 1 {
		t.Fatalf("route report: %+v", report)
	}
	// The report phase echoes the host-observed run phase; the
	// assignment phase was already bound inside the order proof.
	if launched := report.Launched[0]; launched.Issue != 0 || launched.AssignmentID == "" || launched.RunID == "" {
		t.Fatalf("launched = %+v", launched)
	}
	if !observed {
		t.Fatal("host double never ran")
	}
	if len(fx.host.launches) != 1 {
		t.Fatalf("host launches = %d", len(fx.host.launches))
	}
}

func TestDispatchTimeOnlyRootReplaySingleCharge(t *testing.T) {
	ctx := context.Background()
	db, _ := dispatchTestDB(t)
	fx := dispatchSeed(t, db)
	fundTimeOnlyPolicy(t, ctx, db, fx.repo)
	recordTimeOnlyIntent(t, ctx, db)
	deps := fx.deps()
	occupancy, err := snapshotOccupancy(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	first := newDispatchReport()
	DispatchTimeOnlyRoot(ctx, deps, &occupancy, fx.repo, timeOnlyIntentSubject(), strings.Repeat("c", 40), []byte("plan the recorded draft"), &first)
	if len(first.Launched) != 1 {
		t.Fatalf("first route report: %+v", first)
	}
	// Settle the first run so the replay reaches the packet instead of
	// the unsettled-session wait: the finished assignment still blocks
	// any second root for the same subject.
	run, err := db.FactoryRun(ctx, first.Launched[0].RunID)
	if err != nil {
		t.Fatal(err)
	}
	run.Outcome, run.Summary, run.Reconciled = factory.Succeeded, "planning done", true
	if err := db.SaveFactoryRun(ctx, run); err != nil {
		t.Fatal(err)
	}
	second := newDispatchReport()
	DispatchTimeOnlyRoot(ctx, deps, &occupancy, fx.repo, timeOnlyIntentSubject(), strings.Repeat("c", 40), []byte("plan the recorded draft"), &second)
	// The held first reservation waits at limits before any packet: no
	// second run launches and no second charge records. The cross-pass
	// race underneath is covered at the packet by the unfinished
	// completion assignment uniqueness (store concurrent test).
	if len(second.Launched) != 0 || len(second.Waits) == 0 {
		t.Fatalf("replay routed a second run: %+v", second)
	}
	root := first.Launched[0].AssignmentID
	allowance, err := db.IntentAttemptAllowance(ctx, fx.repo, factory.IntentAttemptScope{IntentID: "d777777777777777777777777", OwnerID: 7}, root)
	if err != nil || allowance.Limits.ActiveMinutes != 45 || !allowance.Active {
		t.Fatalf("replay disturbed the first charge: %+v %v", allowance, err)
	}
}

func TestDispatchTimeOnlyRootUnfundedWaits(t *testing.T) {
	ctx := context.Background()
	db, _ := dispatchTestDB(t)
	fx := dispatchSeed(t, db)
	recordTimeOnlyIntent(t, ctx, db)
	deps := fx.deps()
	occupancy, err := snapshotOccupancy(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	report := newDispatchReport()
	DispatchTimeOnlyRoot(ctx, deps, &occupancy, fx.repo, timeOnlyIntentSubject(), strings.Repeat("c", 40), []byte("plan the recorded draft"), &report)
	if len(report.Launched) != 0 || len(report.Errors) != 0 || len(report.Waits) != 1 || report.Waits[0].Reason != WaitAllowance {
		t.Fatalf("unfunded route report: %+v", report)
	}
	if len(fx.host.launches) != 0 {
		t.Fatal("unfunded root reached the host")
	}
}

func TestDispatchTimeOnlyRootRefusesBadSubjects(t *testing.T) {
	ctx := context.Background()
	db, _ := dispatchTestDB(t)
	fx := dispatchSeed(t, db)
	fundTimeOnlyPolicy(t, ctx, db, fx.repo)
	deps := fx.deps()
	occupancy, err := snapshotOccupancy(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	subjects := map[string]factory.ExecutionSubject{
		"unrecorded intent": {
			Kind:   factory.SubjectIntent,
			Intent: &factory.IntentSubject{IntentID: "d111111111111111111111111", RevisionID: "d222222222222222222222222", ScopeDigest: strings.Repeat("d", 64), OwnerID: 7},
		},
		"issue subject": {
			Kind:  factory.SubjectIssue,
			Issue: &factory.IssueSubject{Issue: 3, Acceptance: "d333333333333333333333333"},
		},
		"unstaged completion": {
			Kind:       factory.SubjectCompletion,
			Completion: &factory.CompletionSubject{CompletionID: factory.NewID(), CycleID: factory.NewID(), InventoryDigest: strings.Repeat("f", 64)},
		},
	}
	for name, subject := range subjects {
		report := newDispatchReport()
		DispatchTimeOnlyRoot(ctx, deps, &occupancy, fx.repo, subject, strings.Repeat("c", 40), []byte("plan the recorded draft"), &report)
		if len(report.Launched) != 0 || len(report.Errors) == 0 {
			t.Fatalf("%s routed: %+v", name, report)
		}
	}
	if len(fx.host.launches) != 0 {
		t.Fatal("refused subjects reached the host")
	}
}

// timeOnlyEvidenceBody mirrors the sealed evidence shape with
// computed digests so the sealed cycle validates.
func timeOnlyEvidenceBody(t *testing.T) factory.CompletionIssueEvidence {
	t.Helper()
	sum := func(text string) string {
		digest := sha256.Sum256([]byte(text))
		return hex.EncodeToString(digest[:])
	}
	parsed, err := factory.ParseCompletionIssueEvidence(mustCanonicalBody(t, factory.CompletionIssueEvidence{
		Index: 9, Title: "completion title", Body: "completion body",
		TitleDigest: sum("completion title"), ContentDigest: sum("completion body"),
	}))
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}

func mustCanonicalBody(t *testing.T, body factory.CompletionIssueEvidence) string {
	t.Helper()
	data, err := body.CanonicalData()
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// sealTimeOnlyCycle drives one collecting cycle to a sealed, decided
// reassess outcome through the production CAS joint with a captured
// native local, returning the sealed inventory digest and source.
func sealTimeOnlyCycle(t *testing.T, ctx context.Context, db *store.Store, repo int64) (eventID, cycleID, digest, source string) {
	t.Helper()
	cycles, err := db.PendingReassessmentCycles(ctx, repo)
	if err != nil || len(cycles) != 1 {
		t.Fatalf("pending cycles: %+v %v", cycles, err)
	}
	eventID, cycleID = cycles[0].Cycle.EventID, cycles[0].Cycle.CycleID
	local, err := db.CaptureCompletionLocal(ctx, repo)
	if err != nil {
		t.Fatal(err)
	}
	digest, source = strings.Repeat("d", 64), local.SourceCommit
	record, err := db.ReassessmentRecord(ctx, cycleID)
	if err != nil {
		t.Fatal(err)
	}
	record.Local = local
	record.InventoryDigest = digest
	record.Evidence = []factory.CompletionIssueEvidence{timeOnlyEvidenceBody(t)}
	record.EvidenceIssues = 1
	record.EvidenceRevision = 41
	if err := db.CasReassessmentRecord(ctx, cycleID, factory.ReassessmentCollecting, record); err != nil {
		t.Fatalf("seal not recorded: %v", err)
	}
	decided, err := db.ReassessmentRecord(ctx, cycleID)
	if err != nil {
		t.Fatal(err)
	}
	decided.Cycle.Decision = factory.ReassessmentOptionReassess
	decided.Cycle.State = factory.ReassessmentDeciding
	if err := db.CasReassessmentRecord(ctx, cycleID, factory.ReassessmentCollecting, decided); err != nil {
		t.Fatalf("decision not recorded: %v", err)
	}
	advancing, err := db.ReassessmentRecord(ctx, cycleID)
	if err != nil {
		t.Fatal(err)
	}
	advancing.Cycle.State = factory.ReassessmentReassessing
	if err := db.CasReassessmentRecord(ctx, cycleID, factory.ReassessmentDeciding, advancing); err != nil {
		t.Fatalf("reassessing not recorded: %v", err)
	}
	return eventID, cycleID, digest, source
}

// mergeTimeOnlyCompletion records one merged merge through the
// exported merge joints, returning the fixture holding its recorded
// completion event and staged reassessment cycle. It authorizes the
// union generation, records the committed native effect, then finishes
// the merge as merged: UpdateMerge atomically queues the completion
// root exactly like the production reconcile path.
func mergeTimeOnlyCompletion(t *testing.T, ctx context.Context) *mergeFixture {
	t.Helper()
	fx, p, m, acceptance, decision, fingerprint := unionSeedForBind(t, ctx, 63)
	if err := fx.publish.coord.AuthorizeMerge(ctx, p.ID, unionAuthForBind(t, m, acceptance, decision, fingerprint, true)); err != nil {
		t.Fatal(err)
	}
	driven := recordGenerationIntent(t, ctx, fx.publish.db, m)
	driven.Operation.Effect, driven.Operation.Cancellation = factory.OpEffectCommitted, factory.OpCancelNone
	driven.Operation.Completion = factory.OpCompletionComplete
	driven.Operation.MergedCommit = driven.HeadOID
	driven.Operation.PRNumber, driven.Operation.PRID, driven.Operation.IssueID = driven.PRNumber, driven.PRID, driven.IssueID
	driven.Operation.HeadRef, driven.Operation.BaseRef = driven.HeadRef, driven.BaseRef
	driven.Operation.HeadOID, driven.Operation.BaseOID = driven.HeadOID, driven.BaseOID
	driven.Revision++
	if err := fx.publish.db.UpdateMerge(ctx, driven); err != nil {
		t.Fatal("record commit:", err)
	}
	committed, err := fx.publish.db.MergeByPublication(ctx, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().Unix()
	committed.Stage, committed.Outcome, committed.Reason = factory.MergeMerged, factory.Succeeded, factory.MergeReasonMerged
	committed.MergedCommit = committed.HeadOID
	committed.FinishedUnix, committed.MergedUnix, committed.ClosedUnix = now, now, now
	committed.OriginalClosingOccurrence, committed.OriginalClosedUnix = "501", now
	committed.Revision++
	if err := fx.publish.db.UpdateMerge(ctx, committed); err != nil {
		t.Fatal("record merged:", err)
	}
	stored, err := fx.publish.db.MergeByPublication(ctx, p.ID)
	if err != nil || stored.Stage != factory.MergeMerged {
		t.Fatalf("merge not completed: %+v %v", stored, err)
	}
	return fx
}

func TestDispatchTimeOnlyRootLaunchesReassessment(t *testing.T) {
	ctx := context.Background()
	fx := mergeTimeOnlyCompletion(t, ctx)
	db := fx.publish.db
	repo := fx.publish.seed.repo
	fundTimeOnlyPolicy(t, ctx, db, repo)
	eventID, cycleID, digest, source := sealTimeOnlyCycle(t, ctx, db, repo)
	deps := fx.publish.seed.deps()
	occupancy, err := snapshotOccupancy(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	subject := factory.ExecutionSubject{
		Kind:       factory.SubjectCompletion,
		Completion: &factory.CompletionSubject{CompletionID: eventID, CycleID: cycleID, InventoryDigest: digest},
	}
	var observed bool
	fx.publish.seed.host.launch = func(in project.FactoryLaunch) (project.FactoryState, error) {
		a, err := db.AssignmentByRun(context.Background(), in.Run.ID)
		if err != nil || a.Phase != project.ExecutionReassess || a.Issue != 0 || a.AttemptRoot != a.ID || a.PublicationAssignment != "" {
			t.Fatalf("host call precedes its packet: assignment=%+v err=%v", a, err)
		}
		allowance, err := db.ReassessmentAttemptAllowance(context.Background(), repo, *a.Subject.Completion, a.ID)
		if err != nil || allowance.Limits.ActiveMinutes != 30 || !allowance.Active {
			t.Fatalf("host call precedes its funding: allowance=%+v err=%v", allowance, err)
		}
		observed = true
		return project.FactoryState{ID: in.Run.ID, Project: in.Run.Project, Role: in.Run.Role, Phase: project.FactoryCompleted}, nil
	}
	report := newDispatchReport()
	DispatchTimeOnlyRoot(ctx, deps, &occupancy, repo, subject, source, []byte("reassess the sealed inventory"), &report)
	if len(report.Errors) != 0 || len(report.Waits) != 0 || len(report.Launched) != 1 {
		t.Fatalf("route report: %+v", report)
	}
	// The report phase echoes the host-observed run phase; the
	// assignment phase was already bound inside the order proof.
	if launched := report.Launched[0]; launched.Issue != 0 || launched.AssignmentID == "" || launched.RunID == "" {
		t.Fatalf("launched = %+v", launched)
	}
	if !observed {
		t.Fatal("host double never ran")
	}
}
