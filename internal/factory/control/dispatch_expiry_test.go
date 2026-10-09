package control

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
)

func TestDispatchClosesExpiredReportedAttemptBeforeEarlyReturn(t *testing.T) {
	ctx := context.Background()
	db, dsn := dispatchTestDB(t)
	fx := dispatchSeed(t, db)
	head := fx.accept(t, 3, "d333333333333333333333333")
	fx.queue(t, 3, head.ID)

	first := DispatchPass(ctx, fx.deps())
	if len(first.Launched) != 1 || len(first.Errors) != 0 {
		t.Fatalf("initial pass = %+v", first)
	}
	assignment, err := db.Assignment(ctx, first.Launched[0].AssignmentID)
	if err != nil {
		t.Fatal(err)
	}
	run, err := db.FactoryRun(ctx, assignment.Run)
	if err != nil {
		t.Fatal(err)
	}
	run.Outcome, run.Summary, run.Reconciled = factory.Succeeded, "settled", true
	if err = db.SaveFactoryRun(ctx, run); err != nil {
		t.Fatal(err)
	}
	candidate := strings.Repeat("d", 40)
	output := "```result-json\n" +
		`{"status":"completed","summary":"ready for review","candidate":"` + candidate + `","review_passed":false,"findings":[]}` +
		"\n```"
	finished, ok := AccountSettledRun(ctx, db, run, output, time.Now())
	if !ok || finished.ID != assignment.ID || finished.Stage != factory.AssignmentFinished || finished.Result == nil || !finished.Result.Reported || finished.Result.Candidate != candidate {
		t.Fatalf("reported assignment = %+v, %v", finished, ok)
	}
	if total, usageErr := db.UsageTotal(ctx, fx.repo, "conn"); usageErr != nil || total != 1 {
		t.Fatalf("reported run usage = %d, %v", total, usageErr)
	}
	usageBefore, err := db.UsageTotal(ctx, fx.repo, "conn")
	if err != nil {
		t.Fatal(err)
	}
	prior, err := db.AttemptAllowance(ctx, fx.repo, 3)
	if err != nil || prior.Closed || !prior.Active {
		t.Fatalf("reported root before expiry = %+v, %v", prior, err)
	}

	// Expire the genuine recorded root in this disposable PG fixture. The
	// production dispatch pass below performs closure and all wait decisions.
	fixtureDB, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = fixtureDB.Close() }()
	prior.ActiveSeconds = int64(prior.Limits.ActiveMinutes * 60)
	prior.CheckpointUnix = time.Now().Unix()
	prior.Revision++
	data, err := json.Marshal(prior)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = fixtureDB.ExecContext(ctx, `UPDATE factory_attempt_allowances SET revision=$1,data=$2 WHERE root_assignment=$3`, prior.Revision, string(data), prior.RootAssignment); err != nil {
		t.Fatal(err)
	}

	launches := len(fx.host.launches)
	expired := DispatchPass(ctx, fx.deps())
	if len(expired.Launched) != 0 || len(expired.Errors) != 0 || len(expired.Waits) != 1 || expired.Waits[0].Issue != 3 || expired.Waits[0].Reason != WaitAttemptRecorded || expired.Waits[0].Detail != "attempt is closed; maintainer intervention is required" {
		t.Fatalf("expired pass = %+v", expired)
	}
	if len(fx.host.launches) != launches {
		t.Fatalf("expired reported attempt launched more work: launches=%d want=%d", len(fx.host.launches), launches)
	}
	closed, err := db.AttemptAllowance(ctx, fx.repo, 3)
	if err != nil || !closed.Closed || closed.Active || closed.ActiveSeconds != int64(closed.Limits.ActiveMinutes*60) {
		t.Fatalf("expired allowance was not closed and released: %+v, %v", closed, err)
	}
	afterAssignment, err := db.Assignment(ctx, finished.ID)
	if err != nil || afterAssignment.PromptSHA != finished.PromptSHA || afterAssignment.Result == nil ||
		afterAssignment.Result.Candidate != finished.Result.Candidate || afterAssignment.Result.Summary != finished.Result.Summary {
		t.Fatalf("expiry changed the reported assignment: %+v, %v", afterAssignment, err)
	}
	usageAfter, err := db.UsageTotal(ctx, fx.repo, "conn")
	if err != nil || usageAfter != usageBefore {
		t.Fatalf("expiry changed recorded usage: %d -> %d, %v", usageBefore, usageAfter, err)
	}

	replayed := DispatchPass(ctx, fx.deps())
	if len(replayed.Launched) != 0 || len(replayed.Errors) != 0 || len(replayed.Waits) != 1 ||
		replayed.Waits[0].Reason != WaitAttemptRecorded || replayed.Waits[0].Detail != expired.Waits[0].Detail ||
		len(fx.host.launches) != launches {
		t.Fatalf("expired replay changed dispatch result: %+v", replayed)
	}
	replayAllowance, err := db.AttemptAllowance(ctx, fx.repo, 3)
	if err != nil || replayAllowance.Revision != closed.Revision || !replayAllowance.Closed || replayAllowance.Active {
		t.Fatalf("expiry replay changed allowance: %+v, %v", replayAllowance, err)
	}
	afterReplayUsage, err := db.UsageTotal(ctx, fx.repo, "conn")
	if err != nil || afterReplayUsage != usageBefore {
		t.Fatalf("expiry replay changed usage: %d -> %d, %v", usageBefore, afterReplayUsage, err)
	}
}

func TestExplicitRetryCreatesFreshRootFromSettledReviewerChildAfterExpiry(t *testing.T) {
	ctx := context.Background()
	fx, producer, owner, _ := publishedChildProducerSeed(t, 27)
	if len(producer.launches) != 1 {
		t.Fatalf("initial production did not launch one reviewer: %+v", producer.launches)
	}
	child, priorRun := settleProducedChild(t, fx, producer.launches[0].Run.ID,
		"```review-json\n{\"verdict\":\"request-changes\",\"summary\":\"fix\",\"body\":\"handle empty input\",\"findings\":[\"empty input\"]}\n```")
	if child.Role != project.RoleReviewer || child.PublicationAssignment != owner.ID || child.AttemptRoot != owner.AttemptRoot ||
		priorRun.Outcome != factory.Succeeded || !priorRun.Reconciled {
		t.Fatalf("settled production child lost owner linkage: child=%+v run=%+v", child, priorRun)
	}
	usageBefore, err := fx.db.UsageTotal(ctx, owner.Repository, owner.Connection)
	if err != nil {
		t.Fatal(err)
	}
	allowance, err := fx.db.AttemptAllowance(ctx, owner.Repository, owner.Issue)
	if err != nil || allowance.RootAssignment != owner.AttemptRoot || !allowance.Active || allowance.Closed {
		t.Fatalf("reviewer result did not remain on active root: %+v %v", allowance, err)
	}

	// Expire this genuine root in the disposable PostgreSQL fixture; DispatchPass
	// performs the production closure and makes the explicit-retry wait visible.
	fixtureDB, err := sql.Open("pgx", fx.dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = fixtureDB.Close() }()
	allowance.ActiveSeconds = int64(allowance.Limits.ActiveMinutes * 60)
	allowance.CheckpointUnix = time.Now().Unix()
	allowance.Revision++
	data, err := json.Marshal(allowance)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = fixtureDB.ExecContext(ctx, `UPDATE factory_attempt_allowances SET revision=$1,data=$2 WHERE root_assignment=$3`, allowance.Revision, string(data), allowance.RootAssignment); err != nil {
		t.Fatal(err)
	}

	deps := fx.seed.deps()
	deps.Host = producer
	fx.seed.queue(t, owner.Issue, owner.Acceptance)
	launchesBeforeExpiry := len(producer.launches)
	expired := DispatchPass(ctx, deps)
	if len(expired.Launched) != 0 || len(expired.Errors) != 0 || len(expired.Waits) != 1 ||
		expired.Waits[0].Reason != WaitAttemptRecorded || expired.Waits[0].Detail != "attempt is closed; maintainer intervention is required" ||
		len(producer.launches) != launchesBeforeExpiry {
		t.Fatalf("expired root did not fence automatic work: report=%+v launches=%d", expired, len(producer.launches))
	}
	closed, err := fx.db.AttemptAllowance(ctx, owner.Repository, owner.Issue)
	if err != nil || !closed.Closed || closed.Active {
		t.Fatalf("expired root was not closed and released: %+v %v", closed, err)
	}

	commandID := factory.NewID()
	decision, err := fx.coord.RetryRun(ctx, commandID, "native:7", priorRun.ID)
	if err != nil || !decision.Queued || decision.Prior != priorRun.ID || decision.CommandID != commandID {
		t.Fatalf("explicit retry of settled child: %+v %v", decision, err)
	}
	launchCount := len(producer.launches)
	replay, err := fx.coord.RetryRun(ctx, commandID, "native:7", priorRun.ID)
	if err != nil || replay != decision || len(producer.launches) != launchCount {
		t.Fatalf("retry command replay diverged or launched: replay=%+v err=%v launches=%d", replay, err, len(producer.launches))
	}

	fresh := DispatchPass(ctx, deps)
	if len(fresh.Launched) != 1 || len(fresh.Errors) != 0 || len(producer.launches) != launchCount+1 {
		t.Fatalf("explicit retry did not produce one fresh launch: %+v launches=%d", fresh, len(producer.launches))
	}
	assignment, err := fx.db.Assignment(ctx, commandID)
	if err != nil || assignment.ID != commandID || assignment.AttemptRoot != commandID || assignment.PublicationAssignment != commandID || assignment.Run == priorRun.ID {
		t.Fatalf("explicit retry did not create a fresh root: %+v %v", assignment, err)
	}
	freshAllowance, err := fx.db.AttemptAllowance(ctx, owner.Repository, owner.Issue)
	if err != nil || freshAllowance.RootAssignment != commandID || !freshAllowance.Active || freshAllowance.Closed {
		t.Fatalf("fresh retry allowance: %+v %v", freshAllowance, err)
	}
	usageAfter, err := fx.db.UsageTotal(ctx, owner.Repository, owner.Connection)
	if err != nil || usageAfter != usageBefore {
		t.Fatalf("fresh retry changed recorded prior usage: %d -> %d, %v", usageBefore, usageAfter, err)
	}
}
