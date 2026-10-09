package control

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
)

func TestDispatchPassLaunchesOldestWithinShortLimit(t *testing.T) {
	ctx := context.Background()
	db, _ := dispatchTestDB(t)
	fx := dispatchSeed(t, db)
	if err := db.SaveCapacity(ctx, factory.Capacity{Revision: 1, UpdatedBy: 7, MaxConcurrentRuns: 1, MaxQueued: 10}); err != nil {
		t.Fatal(err)
	}
	oldHead := fx.accept(t, 3, "d333333333333333333333333")
	youngHead := fx.accept(t, 5, "d555555555555555555555555")
	now := time.Now()
	fx.queueAt(t, 3, oldHead.ID, now.Add(-2*time.Hour))
	fx.queueAt(t, 5, youngHead.ID, now.Add(-time.Hour))

	report := DispatchPass(ctx, fx.deps())
	if len(report.Launched) != 1 || report.Launched[0].Issue != 3 {
		t.Fatalf("launched = %+v", report.Launched)
	}
	if len(report.Waits) != 1 || report.Waits[0].Issue != 5 || report.Waits[0].Reason != WaitCapacity {
		t.Fatalf("waits = %+v", report.Waits)
	}
	if len(report.Errors) != 0 {
		t.Fatalf("errors = %+v", report.Errors)
	}
	launched := report.Launched[0]
	a, err := db.Assignment(ctx, launched.AssignmentID)
	if err != nil {
		t.Fatal(err)
	}
	if a.Stage != factory.AssignmentAssigned || a.Attempts != 1 || a.Run != launched.RunID ||
		a.SourceCommit != fx.tip || a.Acceptance != oldHead.ID || a.Connection != "conn" {
		t.Fatalf("assignment = %+v", a)
	}
	if a.Authority.Sponsorship != 1 || a.Authority.RequirementsID == "" || a.Authority.ApprovalID == "" {
		t.Fatalf("assignment authority = %+v", a.Authority)
	}
	for _, want := range []string{
		"Prompt template: soda-f07-f2-v6", "Recorded attempt active-time limit: 120 minutes",
		"synthetic approved setup instructions", "synthetic exact candidate diff",
		"Required evidence checks: ci", "Required evidence:", "report blocked with the concrete reason",
		`Provider connection: "conn"`, "appliance concurrency is 1", "repository concurrency is 1",
		"sponsorship concurrency for this connection in this repository is 2",
		"Preparation requirements: " + a.Authority.RequirementsID + " approval: " + a.Authority.ApprovalID,
	} {
		if !strings.Contains(string(a.Prompt), want) {
			t.Errorf("recorded prompt lacks %q", want)
		}
	}
	if len(fx.host.launches) != 1 {
		t.Fatalf("host launches = %d", len(fx.host.launches))
	}
	sent := fx.host.launches[0]
	if string(sent.Prompt) != string(a.Prompt) || sent.Run.Assignment != a.PromptSHA ||
		sent.Run.SourceCommit != fx.tip || sent.Run.Actor != 7 || sent.Run.Connection != "conn" ||
		sent.HarnessSHA256 != strings.Repeat("a", 64) {
		t.Fatalf("launch = %+v", sent.Run)
	}
	if sent.Run.Preparation != "f111111111111111111111111" {
		t.Fatalf("launch preparation = %+v", sent.Run)
	}
	if run, err := db.FactoryRun(ctx, launched.RunID); err != nil {
		t.Fatal(err)
	} else if span := run.Deadline.Sub(run.Started); span <= 119*time.Minute || span > 120*time.Minute {
		t.Fatalf("run deadline span = %v", span)
	} else if len(fx.host.contextRequests) != 1 || len(fx.host.contextRoles) != 1 {
		t.Fatalf("prepared repository context reads = %+v roles=%+v", fx.host.contextRequests, fx.host.contextRoles)
	} else {
		contextRead := fx.host.contextRequests[0]
		if fx.host.contextRoles[0] != project.RoleCoder || contextRead.ID != a.Preparation ||
			contextRead.SourceCommit != a.SourceCommit || contextRead.ApprovedBase != a.SourceCommit ||
			contextRead.DiffBase != a.SourceCommit || contextRead.Candidate != fx.tip ||
			!contextRead.NotAfter.Equal(run.Deadline) || len(contextRead.Paths) != 1 || contextRead.Paths[0] != "README.md" {
			t.Fatalf("initial context changed approved source, role or run deadline: request=%+v role=%q run=%+v", contextRead, fx.host.contextRoles[0], run)
		}
	}
	run, err := db.FactoryRun(ctx, launched.RunID)
	if err != nil || run.InputSHA != fx.tip || run.Reconciled {
		t.Fatalf("run = %+v %v", run, err)
	}
	view, err := db.FactoryRunView(ctx, launched.RunID)
	if err != nil || view.Attempt != launched.AssignmentID || view.Issue != 3 {
		t.Fatalf("view = %+v %v", view, err)
	}
	reservation, err := db.Reservation(ctx, launched.AssignmentID)
	if err != nil || reservation.State != factory.ReservationHeld || reservation.PlannedMinutes != 120 {
		t.Fatalf("reservation = %+v %v", reservation, err)
	}

	// Settlement releases the executing session. The successful attempt
	// retains its slot while its candidate still needs review and CI.
	run.Outcome, run.Summary, run.Reconciled = factory.Succeeded, "done", true
	if err := db.SaveFactoryRun(ctx, run); err != nil {
		t.Fatal(err)
	}
	candidate := strings.Repeat("d", 40)
	output := "work\n```result-json\n" +
		`{"status":"completed","summary":"fixed","candidate":"` + candidate + `","review_passed":false,"findings":[]}` +
		"\n```"
	finished, ok := AccountSettledRun(ctx, db, run, output, time.Now())
	if !ok || finished.Outcome != factory.Succeeded || finished.Result == nil || !finished.Result.Reported {
		t.Fatalf("accounted = %+v %v", finished, ok)
	}
	if finished.Result.Candidate != candidate {
		t.Fatalf("candidate = %q", finished.Result.Candidate)
	}
	if total, _ := db.UsageTotal(ctx, fx.repo, "conn"); total < 1 {
		t.Fatalf("usage total = %d", total)
	}
	second := DispatchPass(ctx, fx.deps())
	if len(second.Launched) != 0 || len(second.Errors) != 0 || waitReason(second, 5) != WaitCapacity {
		t.Fatalf("settled successful attempt released capacity: %+v", second)
	}
	allowance, err := db.AttemptAllowance(ctx, fx.repo, 3)
	if err != nil || !allowance.Active || allowance.Closed || allowance.RootAssignment != a.AttemptRoot {
		t.Fatalf("settled candidate attempt = %+v, %v", allowance, err)
	}
}

func TestDispatchRefusesTargetDifferentFromPreparedCoderSource(t *testing.T) {
	ctx := context.Background()
	db, _ := dispatchTestDB(t)
	fx := dispatchSeed(t, db)
	head := fx.accept(t, 3, "d333333333333333333333333")
	fx.queue(t, 3, head.ID)
	inputs := fx.reads.inputs["7/3"]
	inputs.Tip = strings.Repeat("f", 40)
	fx.reads.inputs["7/3"] = inputs

	report := DispatchPass(ctx, fx.deps())
	if len(report.Launched) != 0 || len(report.Errors) != 0 || waitReason(report, 3) != WaitSource ||
		len(fx.host.launches) != 0 || len(fx.host.contextRequests) != 0 {
		t.Fatalf("mismatched prepared source launched or read context: %+v launches=%d reads=%+v", report, len(fx.host.launches), fx.host.contextRequests)
	}
}

func TestDispatchRefusesRepositoryContextPathOverflowBeforeNativeRead(t *testing.T) {
	ctx := context.Background()
	db, _ := dispatchTestDB(t)
	fx := dispatchSeed(t, db)
	var body strings.Builder
	for i := 0; i < project.MaxFactoryContextPaths; i++ {
		body.WriteString(fmt.Sprintf("see selected-%02d.md\n", i))
	}
	head := fx.accept(t, 3, "d333333333333333333333333", body.String())
	fx.queue(t, 3, head.ID)

	report := DispatchPass(ctx, fx.deps())
	if len(report.Launched) != 0 || len(report.Errors) != 0 || waitReason(report, 3) != WaitSource ||
		len(fx.host.launches) != 0 || len(fx.host.contextRequests) != 0 {
		t.Fatalf("path-reference overflow reached native context read or launch: %+v launches=%d reads=%+v", report, len(fx.host.launches), fx.host.contextRequests)
	}
}

func TestDispatchPassWithoutDepsReports(t *testing.T) {
	report := DispatchPass(context.Background(), DispatchDeps{})
	if len(report.Errors) != 1 || len(report.Launched) != 0 {
		t.Fatalf("report = %+v", report)
	}
}
