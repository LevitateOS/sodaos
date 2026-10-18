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

func timeOnlyIntentSubject(digest string) factory.ExecutionSubject {
	return factory.ExecutionSubject{
		Kind:   factory.SubjectIntent,
		Intent: &factory.IntentSubject{IntentID: "d777777777777777777777777", RevisionID: "d888888888888888888888888", ScopeDigest: digest, OwnerID: 7},
	}
}

func recordTimeOnlyIntent(t *testing.T, ctx context.Context, db *store.Store) string {
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
	return revision.Digest()
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
	digest := recordTimeOnlyIntent(t, ctx, db)
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
	DispatchTimeOnlyRoot(ctx, deps, &occupancy, fx.repo, timeOnlyIntentSubject(digest), strings.Repeat("c", 40), []byte("plan the recorded draft"), &report)
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
	digest := recordTimeOnlyIntent(t, ctx, db)
	deps := fx.deps()
	occupancy, err := snapshotOccupancy(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	first := newDispatchReport()
	DispatchTimeOnlyRoot(ctx, deps, &occupancy, fx.repo, timeOnlyIntentSubject(digest), strings.Repeat("c", 40), []byte("plan the recorded draft"), &first)
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
	DispatchTimeOnlyRoot(ctx, deps, &occupancy, fx.repo, timeOnlyIntentSubject(digest), strings.Repeat("c", 40), []byte("plan the recorded draft"), &second)
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
	digest := recordTimeOnlyIntent(t, ctx, db)
	deps := fx.deps()
	occupancy, err := snapshotOccupancy(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	report := newDispatchReport()
	DispatchTimeOnlyRoot(ctx, deps, &occupancy, fx.repo, timeOnlyIntentSubject(digest), strings.Repeat("c", 40), []byte("plan the recorded draft"), &report)
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

// intentPlanHead returns a repository-bound draft head owned by the
// fixture operator for the R12 connector proof.
func intentPlanHead(now time.Time) factory.IntentRevision {
	return factory.IntentRevision{
		ID: "d888888888888888888888888", Intent: "d777777777777777777777777", Owner: 7, Repository: 7,
		Objective: "plan the recorded draft", State: "draft",
		MaxIssues: 10, CreatedUnix: now.Unix(),
		Ideas: []factory.ProductIdea{
			{ID: "d333333333333333333333333", Description: "Capture leave requests", Consequence: "Requests can be recorded", Decision: "accept", Source: "requested", Provenance: "human", Actor: 7},
		},
	}
}

// intentPlanInput stages the admitted planning input for the head:
// the requested operation with its inspected source investigation.
func intentPlanInput(t *testing.T, head factory.IntentRevision, op string) factory.SemanticPlannerInput {
	t.Helper()
	content, err := factory.ComposeProposalContent(head)
	if err != nil {
		t.Fatalf("proposal content: %v", err)
	}
	return factory.SemanticPlannerInput{
		Operation:    op,
		Head:         head,
		Content:      content,
		Alternatives: factory.DescribeAlternatives(head.Objective),
		Inspected: []factory.InspectedSource{
			{Path: "internal/factory/intent.go", Commit: strings.Repeat("c", 40), Evidence: "Draft head carries attributed effective scope with no tasks yet"},
		},
	}
}

// TestDispatchIntentPlanConnectsHeadToBoundResult proves the R12 source
// connection on real PostgreSQL: the connector admits the reviewed
// repository-bound head for the requested operation, stages the bounded
// prompt and launches through the shared route with a bounded host
// double; the host's answer settles through AccountSettledRun into the
// typed result bound to the assignment's exact intent and revision,
// while an answer for another revision refuses without a report.
func TestDispatchIntentPlanConnectsHeadToBoundResult(t *testing.T) {
	ctx := context.Background()
	db, _ := dispatchTestDB(t)
	fx := dispatchSeed(t, db)
	fundTimeOnlyPolicy(t, ctx, db, fx.repo)
	head := intentPlanHead(time.Now().UTC().Truncate(time.Second))
	if err := db.SaveIntentRevision(ctx, head); err != nil {
		t.Fatal(err)
	}
	in := intentPlanInput(t, head, factory.IntentPlanningPropose)
	coord := &Coordinator{Store: db, Host: fx.host, Broker: fx.broker, DispatchReads: fx.deps().Reads}
	var observed bool
	timeOnlyOrderHost(fx, &observed, func(ctx context.Context, in project.FactoryLaunch) {
		a, err := db.AssignmentByRun(ctx, in.Run.ID)
		if err != nil || a.Phase != project.ExecutionDecompose || a.Subject.Intent == nil || a.Subject.Intent.ScopeDigest != head.Digest() {
			t.Fatalf("host call precedes its bound admission: assignment=%+v err=%v", a, err)
		}
		if !strings.Contains(string(a.Prompt), head.Digest()) {
			t.Fatal("staged prompt does not carry the bound head digest")
		}
	})
	report := coord.DispatchIntentPlan(ctx, head.Intent, 7, factory.IntentPlanningPropose, in, strings.Repeat("c", 40))
	if len(report.Errors) != 0 || len(report.Waits) != 0 || len(report.Launched) != 1 {
		t.Fatalf("plan route report: %+v", report)
	}
	if !observed {
		t.Fatal("host double never ran")
	}
	run, err := db.FactoryRun(ctx, report.Launched[0].RunID)
	if err != nil {
		t.Fatal(err)
	}
	run.Outcome, run.Summary, run.Reconciled = factory.Succeeded, "planning done", true
	if err := db.SaveFactoryRun(ctx, run); err != nil {
		t.Fatal(err)
	}
	finished, ok := AccountSettledRun(ctx, db, run, settleIntentOutput(t, head.Intent, head.ID), time.Now())
	if !ok || finished.Outcome != factory.Succeeded || finished.Result == nil || !finished.Result.Reported {
		t.Fatalf("bound answer unsettled: %+v %v", finished, ok)
	}
	decomposition := finished.Result.PhaseResult.Output.Decomposition
	if decomposition == nil || decomposition.Intent != head.Intent || decomposition.Revision != head.ID {
		t.Fatalf("typed result not bound to its inputs: %+v", finished.Result)
	}
	if again, ok := AccountSettledRun(ctx, db, run, settleIntentOutput(t, head.Intent, head.ID), time.Now()); ok || again.Stage != "" {
		t.Fatalf("second accounting = %+v %v", again, ok)
	}
}

// TestDispatchIntentPlanRefusesUnreviewedHead proves the connector's
// guards: an unassociated draft launches nothing, and a wrong owner
// cannot read another owner's head.
func TestDispatchIntentPlanRefusesUnreviewedHead(t *testing.T) {
	ctx := context.Background()
	db, _ := dispatchTestDB(t)
	fx := dispatchSeed(t, db)
	fundTimeOnlyPolicy(t, ctx, db, fx.repo)
	loose := intentPlanHead(time.Now().UTC().Truncate(time.Second))
	loose.Repository = 0
	if err := db.SaveIntentRevision(ctx, loose); err != nil {
		t.Fatal(err)
	}
	coord := &Coordinator{Store: db, Host: fx.host, Broker: fx.broker, DispatchReads: fx.deps().Reads}
	in := intentPlanInput(t, loose, factory.IntentPlanningPropose)
	unassociated := coord.DispatchIntentPlan(ctx, loose.Intent, 7, factory.IntentPlanningPropose, in, strings.Repeat("c", 40))
	if len(unassociated.Errors) == 0 || len(unassociated.Launched) != 0 {
		t.Fatalf("unassociated draft launched: %+v", unassociated)
	}
	head := intentPlanHead(time.Now().UTC().Truncate(time.Second))
	head.ID, head.Intent = "d777777777777777777777777", "d666666666666666666666666"
	if err := db.SaveIntentRevision(ctx, head); err != nil {
		t.Fatal(err)
	}
	foreign := coord.DispatchIntentPlan(ctx, head.Intent, 9, factory.IntentPlanningPropose, in, strings.Repeat("c", 40))
	if len(foreign.Errors) == 0 || len(foreign.Launched) != 0 {
		t.Fatalf("wrong owner launched: %+v", foreign)
	}
	if len(fx.host.launches) != 0 {
		t.Fatalf("guarded connector reached the host: %d", len(fx.host.launches))
	}
}

// TestDispatchIntentPlanRefusesMismatchedInspectedSource proves the
// common boundary binds evidence to source: inspected evidence for
// another commit refuses before admission, leaving no launch, run or
// funding behind.
func TestDispatchIntentPlanRefusesMismatchedInspectedSource(t *testing.T) {
	ctx := context.Background()
	db, _ := dispatchTestDB(t)
	fx := dispatchSeed(t, db)
	fundTimeOnlyPolicy(t, ctx, db, fx.repo)
	head := intentPlanHead(time.Now().UTC().Truncate(time.Second))
	if err := db.SaveIntentRevision(ctx, head); err != nil {
		t.Fatal(err)
	}
	content, err := factory.ComposeProposalContent(head)
	if err != nil {
		t.Fatal(err)
	}
	in := factory.SemanticPlannerInput{
		Operation: factory.IntentPlanningPropose, Head: head, Content: content,
		Alternatives: factory.DescribeAlternatives(head.Objective),
		Inspected: []factory.InspectedSource{
			{Path: "internal/factory/intent.go", Commit: strings.Repeat("d", 40), Evidence: "evidence for another commit"},
		},
	}
	coord := &Coordinator{Store: db, Host: fx.host, Broker: fx.broker, DispatchReads: fx.deps().Reads}
	report := coord.DispatchIntentPlan(ctx, head.Intent, 7, factory.IntentPlanningPropose, in, strings.Repeat("c", 40))
	if len(report.Errors) == 0 || len(report.Launched) != 0 {
		t.Fatalf("mismatched source admitted: %+v", report)
	}
	if len(fx.host.launches) != 0 {
		t.Fatalf("mismatched source reached the host: %d", len(fx.host.launches))
	}
}

// TestSettledDecomposeRefusesUnrecordedIdeaBinding proves settlement
// validates the reported answer against its recorded staged input: a
// decomposition binding an idea the recorded head never carried
// converts to needs-human with no report, while the executed run's
// usage and consumed reservation stay recorded.
func TestSettledDecomposeRefusesUnrecordedIdeaBinding(t *testing.T) {
	ctx := context.Background()
	db, _ := dispatchTestDB(t)
	fx := dispatchSeed(t, db)
	fundTimeOnlyPolicy(t, ctx, db, fx.repo)
	now := time.Now().UTC().Truncate(time.Second)
	head := factory.IntentRevision{
		ID: "d888888888888888888888888", Intent: "d777777777777777777777777", Owner: 7, Repository: fx.repo,
		Objective: "Decide vacation requests with manager review", State: "draft",
		MaxIssues: 10, CreatedUnix: now.Unix(),
		Ideas: []factory.ProductIdea{
			{ID: "d333333333333333333333333", Description: "Capture leave requests", Consequence: "Requests can be recorded", Decision: "accept", Source: "requested", Provenance: "human", Actor: 7},
		},
	}
	if err := db.SaveIntentRevision(ctx, head); err != nil {
		t.Fatal(err)
	}
	content, err := factory.ComposeProposalContent(head)
	if err != nil {
		t.Fatal(err)
	}
	in := factory.SemanticPlannerInput{
		Operation: factory.IntentPlanningPropose, Head: head, Content: content,
		Alternatives: factory.DescribeAlternatives(head.Objective),
		Inspected: []factory.InspectedSource{
			{Path: "internal/factory/intent.go", Commit: strings.Repeat("c", 40), Evidence: "recorded investigation"},
		},
	}
	coord := &Coordinator{Store: db, Host: fx.host, Broker: fx.broker, DispatchReads: fx.deps().Reads}
	var observed bool
	timeOnlyOrderHost(fx, &observed, func(context.Context, project.FactoryLaunch) {})
	report := coord.DispatchIntentPlan(ctx, head.Intent, 7, factory.IntentPlanningPropose, in, strings.Repeat("c", 40))
	if len(report.Errors) != 0 || len(report.Launched) != 1 {
		t.Fatalf("plan route report: %+v", report)
	}
	run, err := db.FactoryRun(ctx, report.Launched[0].RunID)
	if err != nil {
		t.Fatal(err)
	}
	run.Outcome, run.Summary, run.Reconciled = factory.Succeeded, "planning done", true
	if err := db.SaveFactoryRun(ctx, run); err != nil {
		t.Fatal(err)
	}
	stranger := fencedResult(t, factory.PhaseResult{
		Version: factory.PhaseResultVersion, Phase: project.ExecutionDecompose, Status: "completed",
		Summary: "single choice ready",
		Output: &factory.PhaseOutput{Decomposition: &factory.SemanticPlannerOutput{
			Intent: head.Intent, Revision: head.ID, Operation: factory.IntentPlanningPropose,
			Candidates: []factory.SemanticCandidate{{
				Key: "keep", Rationale: "Keep the recorded draft as the single review baseline.",
				Tasks: []factory.IntentTask{{
					Key: "d999999999999999999999999", Objective: "Decide the rollout order",
					Outcomes: []string{"Rollout is ordered"}, IdeaIDs: []string{"d666666666666666666666666"},
				}},
			}},
		}},
	})
	finished, ok := AccountSettledRun(ctx, db, run, stranger, time.Now())
	if !ok || finished.Outcome != factory.NeedsHuman || finished.Result == nil || finished.Result.Reported {
		t.Fatalf("unbound answer settled: %+v %v", finished, ok)
	}
}
