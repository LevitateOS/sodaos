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

// Serving settlement for Issue-zero time-only roots (R11 acceptance)
// against real PostgreSQL: a succeeded intent decompose run settles
// end to end through AccountSettledRun — the same serving chain as
// every other phase — closing its typed allowance with zero readiness
// work, and a second settlement refuses. Failed and unparseable runs
// stay charged through the shared joint. This proves the shared
// registration/assignment/reservation/run/view/result core serves
// intent subjects with no separate intent engine.

// settleIntentPacket records one funded intent decompose packet through
// the production packet boundary and settles its run as succeeded,
// returning the recorded assignment and settled run.
// settleIntentPacket records one funded intent decompose packet through
// the production packet boundary and settles its run under the given
// outcome, returning the recorded assignment and settled run. A
// reconciled run is immutable, so the outcome is settled exactly once.
func settleIntentPacket(t *testing.T, ctx context.Context, db *store.Store, fx dispatchFixture, coord *Coordinator, intentID, revisionID string, outcome factory.Outcome, now time.Time) (factory.Assignment, factory.Run) {
	t.Helper()
	revision := factory.IntentRevision{
		ID: revisionID, Intent: intentID, Owner: 7, Repository: fx.repo,
		Objective: "plan the recorded draft", State: "draft",
		MaxIssues: 10, CreatedUnix: now.Unix(),
		Ideas: []factory.ProductIdea{
			{ID: "d333333333333333333333333", Description: "Capture leave requests", Consequence: "Requests can be recorded", Decision: "accept", Source: "requested", Provenance: "human", Actor: 7},
		},
	}
	if err := db.SaveIntentRevision(ctx, revision); err != nil {
		t.Fatalf("intent revision not recorded: %v", err)
	}
	selection := fx.policy.Roles[project.RoleCoder]
	content, err := factory.ComposeProposalContent(revision)
	if err != nil {
		t.Fatalf("proposal content: %v", err)
	}
	prompt, err := StageIntentPlanningPrompt(revision, factory.SemanticPlannerInput{
		Operation: factory.IntentPlanningPropose, Head: revision, Content: content,
		Alternatives: factory.DescribeAlternatives(revision.Objective),
		Inspected: []factory.InspectedSource{
			{Path: "internal/factory/intent.go", Commit: strings.Repeat("c", 40), Evidence: "recorded investigation"},
		},
	})
	if err != nil {
		t.Fatalf("staged prompt: %v", err)
	}
	sum := sha256.Sum256(prompt)
	rootID, runID := factory.NewID(), factory.NewID()
	a := factory.Assignment{
		ID: rootID, AttemptRoot: rootID, PublicationAssignment: "", ParentAssignment: "",
		Subject: factory.ExecutionSubject{
			Kind:   factory.SubjectIntent,
			Intent: &factory.IntentSubject{IntentID: intentID, RevisionID: revisionID, ScopeDigest: revision.Digest(), OwnerID: 7},
		},
		Phase:     project.ExecutionDecompose,
		ProjectID: fx.proj, ActorID: 7, Role: project.RoleCoder,
		Repository: fx.repo, Issue: 0, NativeRev: 5, Acceptance: "", Preparation: "f111111111111111111111111",
		Harness: selection.Harness + "-" + selection.HarnessVers, HarnessVers: selection.HarnessVers, Model: selection.Model,
		Connection: "conn", SourceCommit: strings.Repeat("c", 40),
		Prompt: prompt, PromptSHA: hex.EncodeToString(sum[:]),
		Run: runID, RunHistory: []string{runID}, Stage: factory.AssignmentAssigned,
		Attempts: 1, CreatedUnix: now.Unix(),
	}
	authority, err := coord.EffectiveAuthority(ctx, a.Repository)
	if err != nil || !authority.Effective {
		t.Fatalf("fixture authority: %+v %v", authority, err)
	}
	a.Authority = authority.Authority
	if a.Authority.RequirementsID, err = db.RequirementHead(ctx, a.ProjectID); err != nil {
		t.Fatal(err)
	}
	if a.Authority.ApprovalID, err = db.ApprovalHead(ctx, a.ProjectID); err != nil {
		t.Fatal(err)
	}
	if err := a.Validate(); err != nil {
		t.Fatalf("settling intent root invalid: %v", err)
	}
	expected := factory.IssueControl{
		Repository: fx.repo, Issue: 0, NativeRev: a.NativeRev,
		Readiness: factory.ReadinessQueued, Reason: factory.ReasonEligible,
		Fingerprint: strings.Repeat("1", 64), Authority: strings.Repeat("2", 64),
	}
	r := factory.Reservation{AssignmentID: a.ID, Repository: fx.repo, Connection: "conn", State: factory.ReservationHeld, PlannedMinutes: 45}
	run := factory.Run{
		ID: runID, ProjectID: a.ProjectID, Role: project.RoleCoder, InputSHA: a.SourceCommit,
		Started: now, Deadline: now.Add(45 * time.Minute),
		Image: "sha256:" + strings.Repeat("b", 64), Harness: a.Harness, Model: a.Model,
	}
	view := factory.RunView{RunID: runID, Repository: fx.repo, Issue: 0, Attempt: a.ID}
	if err := db.RecordDispatchPacket(ctx, factory.DispatchRegistration{ID: a.ID, Repository: a.Repository, Authority: a.Authority}, expected, a, r, run, view); err != nil {
		t.Fatalf("settling intent packet refused: %v", err)
	}
	summary := "planning done"
	if outcome != factory.Succeeded {
		summary = "provider failed"
	}
	run.Outcome, run.Summary, run.Reconciled = outcome, summary, true
	if err := db.SaveFactoryRun(ctx, run); err != nil {
		t.Fatal(err)
	}
	return a, run
}

func settleIntentOutput(t *testing.T, intentID, revisionID string) string {
	t.Helper()
	decomposition := factory.ScopedDecomposition{
		Intent: intentID, Revision: revisionID, Selected: "keep",
		Tasks: []factory.IntentTask{{
			Key: "d999999999999999999999999", Objective: "Plan the recorded draft",
			Outcomes: []string{"Draft is planned"}, IdeaIDs: []string{"d333333333333333333333333"},
		}},
		Advice: []string{strings.Repeat("a", 64)},
	}
	return fencedResult(t, factory.PhaseResult{
		Version: factory.PhaseResultVersion, Phase: project.ExecutionDecompose, Status: "completed",
		Summary: "single choice ready", Output: &factory.PhaseOutput{Decomposition: &decomposition},
	})
}

func TestAccountSettledRunSettlesIntentRoot(t *testing.T) {
	ctx := context.Background()
	db, _ := dispatchTestDB(t)
	fx := dispatchSeed(t, db)
	policy, err := db.RepositoryPolicy(ctx, fx.repo)
	if err != nil {
		t.Fatal(err)
	}
	policy.IntentMinutes = 45
	if err := db.SaveRepositoryPolicy(ctx, policy); err != nil {
		t.Fatal(err)
	}
	coord := &Coordinator{Store: db, Host: fx.host}
	now := time.Now().UTC().Truncate(time.Second)
	intentID, revisionID := "d777777777777777777777777", "d888888888888888888888888"
	a, run := settleIntentPacket(t, ctx, db, fx, coord, intentID, revisionID, factory.Succeeded, now)
	finished, ok := AccountSettledRun(ctx, db, run, settleIntentOutput(t, intentID, revisionID), time.Now())
	if !ok || finished.Outcome != factory.Succeeded || finished.Reason != factory.AssignReasonReported {
		t.Fatalf("accounted = %+v %v", finished, ok)
	}
	if finished.Result == nil || !finished.Result.Reported || finished.Result.Phase != project.ExecutionDecompose ||
		finished.Result.Candidate != "" || finished.Result.PhaseResult.Output.Decomposition == nil {
		t.Fatalf("intent result not exact: %+v", finished.Result)
	}
	if r, _ := db.Reservation(ctx, a.ID); r.State != factory.ReservationConsumed {
		t.Fatalf("reservation = %+v", r)
	}
	if total, _ := db.UsageTotal(ctx, fx.repo, "conn"); total < 1 {
		t.Fatalf("usage = %d", total)
	}
	allowance, err := db.IntentAttemptAllowance(ctx, fx.repo, factory.IntentAttemptScope{IntentID: intentID, OwnerID: 7}, a.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !allowance.Closed || allowance.Active {
		t.Fatalf("settled intent did not close+stop: %+v", allowance)
	}
	if again, ok := AccountSettledRun(ctx, db, run, settleIntentOutput(t, intentID, revisionID), time.Now()); ok || again.Stage != "" {
		t.Fatalf("second accounting = %+v %v", again, ok)
	}
}

func TestAccountSettledRunChargesFailedIntentRun(t *testing.T) {
	ctx := context.Background()
	db, _ := dispatchTestDB(t)
	fx := dispatchSeed(t, db)
	policy, err := db.RepositoryPolicy(ctx, fx.repo)
	if err != nil {
		t.Fatal(err)
	}
	policy.IntentMinutes = 45
	if err := db.SaveRepositoryPolicy(ctx, policy); err != nil {
		t.Fatal(err)
	}
	coord := &Coordinator{Store: db, Host: fx.host}
	now := time.Now().UTC().Truncate(time.Second)
	a, run := settleIntentPacket(t, ctx, db, fx, coord, "d777777777777777777777777", "d888888888888888888888888", factory.Failed, now)
	finished, ok := AccountSettledRun(ctx, db, run, "", time.Now())
	if !ok || finished.Outcome != factory.Failed || finished.Reason != factory.AssignReasonRunFailed {
		t.Fatalf("accounted = %+v %v", finished, ok)
	}
	// Failed provider work stays charged: the typed allowance still
	// answers for this root, never refilled.
	if _, err := db.IntentAttemptAllowance(ctx, fx.repo, factory.IntentAttemptScope{IntentID: "d777777777777777777777777", OwnerID: 7}, a.ID); err != nil {
		t.Fatalf("failed intent lost its allowance: %v", err)
	}
}

func TestAccountSettledRunRefusesUnparseableIntentOutput(t *testing.T) {
	ctx := context.Background()
	db, _ := dispatchTestDB(t)
	fx := dispatchSeed(t, db)
	policy, err := db.RepositoryPolicy(ctx, fx.repo)
	if err != nil {
		t.Fatal(err)
	}
	policy.IntentMinutes = 45
	if err := db.SaveRepositoryPolicy(ctx, policy); err != nil {
		t.Fatal(err)
	}
	coord := &Coordinator{Store: db, Host: fx.host}
	now := time.Now().UTC().Truncate(time.Second)
	_, run := settleIntentPacket(t, ctx, db, fx, coord, "d777777777777777777777777", "d888888888888888888888888", factory.Succeeded, now)
	finished, ok := AccountSettledRun(ctx, db, run, "chatty output, no fence", time.Now())
	if !ok || finished.Outcome != factory.NeedsHuman || finished.Result.Status != "blocked" {
		t.Fatalf("accounted = %+v %v", finished, ok)
	}
}
