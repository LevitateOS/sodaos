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
	if err := db.SaveCapacity(ctx, factory.Capacity{Revision: 1, UpdatedBy: 7, MaxConcurrentRuns: 1}); err != nil {
		t.Fatal(err)
	}
	const acceptedBody = "Implement app.can and client.jsx; app.cannot and notes.mdbackup are prose, not source files."
	oldHead := fx.accept(t, 3, "d333333333333333333333333", acceptedBody)
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
	if a.AcceptedContext == nil || a.AcceptedContext.Title != "objective 3; inspect README.md" || a.AcceptedContext.Body != acceptedBody || len(a.AcceptedContext.Sources) != 1 || a.AcceptedContext.Sources[0].Content != "answer 3" || a.AcceptedContext.RepositoryContext == nil || a.AcceptedContext.ValidateAcceptance(oldHead) != nil {
		t.Fatal("atomic dispatch lost exact structured accepted context")
	}
	for _, want := range []string{
		"Prompt template: " + factory.DispatchPromptTemplate, "Recorded attempt active-time limit: 120 minutes",
		factory.LibraryReadGuidance,
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
			!contextRead.NotAfter.Equal(run.Deadline) || strings.Join(contextRead.Paths, ",") != "README.md,app.can,client.jsx" {
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
	// The context classifier runs once at the launch boundary with the
	// actual packet and current facts; its choice is then frozen.
	guarded := fx.deps()
	guarded.MCPKey = []byte(strings.Repeat("k", 32))
	beforeLaunch := len(fx.host.launches)
	// Simulate a restart after the pending marker commits but before its
	// deterministic capability production. No identity or attempt changes.
	run.ContextPending = true
	if err := db.SaveFactoryRun(ctx, run); err != nil {
		t.Fatal(err)
	}
	chooser := &bundleGapChooser{pick: factory.ContextBundleCurrent}
	guarded.Priority = &PriorityJudgments{Journal: db, Chooser: chooser}
	qualified := RecoverDispatch(ctx, guarded)
	if len(qualified.Launched) != 1 || !chooser.sawFacts || chooser.calls != 1 {
		t.Fatalf("actual current facts did not qualify launch: %+v facts=%t calls=%d", qualified, chooser.sawFacts, chooser.calls)
	}
	run, err = db.FactoryRun(ctx, run.ID)
	if err != nil || run.ContextPending {
		t.Fatalf("native launch did not clear durable context hold: %+v %v", run, err)
	}
	if chooser.sawAttempt || chooser.history != 0 {
		t.Fatal("first staged run fabricated execution history for context selection")
	}
	capability, err := db.GetMCPCapability(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.ActivateObservedMCPCapability(ctx, capability); err != nil {
		t.Fatal(err)
	}
	capability, err = db.GetMCPCapability(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	state, attempt, _, err := loadContextBundleState(ctx, db, capability, a, nil)
	if err != nil || !attempt || len(state.HistoryPreview) != 1 || state.HistoryPreview[0].ID != run.ID {
		t.Fatalf("observed native execution lost its actual continuity: %+v attempt=%t err=%v", state.HistoryPreview, attempt, err)
	}
	if bundle, reason, err := PrepareContextBundleSelection(ctx, db, nil, a, run, nil); err != nil || bundle != factory.ContextBundleCurrent || reason != "" || chooser.calls != 1 {
		t.Fatalf("frozen actual context choice was consulted again: %s %s %v", bundle, reason, err)
	}

	// Settlement releases the executing session. The reported
	// investigation advances to its implementation child, which retains
	// the slot; completed planning never names a candidate.
	run.Outcome, run.Summary, run.Reconciled = factory.Succeeded, "done", true
	if err := db.SaveFactoryRun(ctx, run); err != nil {
		t.Fatal(err)
	}
	plan := phaseTestPlan(t, a.AttemptRoot, a.SourceCommit, a.Preparation)
	output := fencedResult(t, factory.PhaseResult{
		Version: factory.PhaseResultVersion, Phase: project.ExecutionInvestigate, Status: "completed",
		Summary: "plan ready", Output: &factory.PhaseOutput{Plan: &plan},
	})
	finished, ok := AccountSettledRun(ctx, db, run, output, time.Now())
	if !ok || finished.Outcome != factory.Succeeded || finished.Result == nil || !finished.Result.Reported {
		t.Fatalf("accounted = %+v %v", finished, ok)
	}
	if finished.Result.Candidate != "" {
		t.Fatalf("candidate = %q", finished.Result.Candidate)
	}
	if total, _ := db.UsageTotal(ctx, fx.repo, "conn"); total < 1 {
		t.Fatalf("usage total = %d", total)
	}
	// The reported investigation advances to its implementation child,
	// which retains the short-limit slot: the younger issue still waits.
	second := DispatchPass(ctx, fx.deps())
	if len(second.Launched) != 1 || second.Launched[0].Issue != 3 || len(second.Errors) != 0 || waitReason(second, 5) != WaitCapacity {
		t.Fatalf("reported investigation did not advance under its retained slot: %+v", second)
	}
	child, err := db.Assignment(ctx, second.Launched[0].AssignmentID)
	if err != nil || child.ParentAssignment != a.ID || child.Phase != project.ExecutionImplement || child.AttemptRoot != a.AttemptRoot {
		t.Fatalf("advanced child = %+v %v", child, err)
	}
	if child.AcceptedContext == nil || child.AcceptedContext.ValidateFor(child) != nil || child.AcceptedContext.ValidateAcceptance(oldHead) != nil {
		t.Fatal("implementation child lost its verified accepted context")
	}
	childRun, err := db.FactoryRun(ctx, child.Run)
	if err != nil {
		t.Fatal(err)
	}
	uncertain := &bundleGapChooser{pick: factory.ContextBundleCurrent, confidence: 0.6}
	guarded.Priority = &PriorityJudgments{Journal: db, Chooser: uncertain}
	beforeLaunch = len(fx.host.launches)
	unsure := newDispatchReport()
	launchTail(ctx, guarded, child, childRun, fx.host.launches[beforeLaunch-1], &unsure)
	if len(unsure.Launched) != 1 || len(fx.host.launches) != beforeLaunch+1 || uncertain.calls != 1 {
		t.Fatalf("low confidence selection held work instead of serving current context: %+v calls=%d", unsure, uncertain.calls)
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
	// The moved tip has no ready preparation, so the preparation gate
	// refuses before any source-context read; nothing launches.
	if len(report.Launched) != 0 || len(report.Errors) != 0 || waitReason(report, 3) != WaitPreparation ||
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
