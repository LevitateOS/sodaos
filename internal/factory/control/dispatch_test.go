package control

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
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
	if a.AcceptedContext == nil || a.AcceptedContext.Title != "objective 3; inspect README.md" || a.AcceptedContext.Body != "body 3" || len(a.AcceptedContext.Sources) != 1 || a.AcceptedContext.Sources[0].Content != "answer 3" || a.AcceptedContext.RepositoryContext == nil || a.AcceptedContext.ValidateAcceptance(oldHead) != nil {
		t.Fatal("atomic dispatch lost exact structured accepted context")
	}
	for _, want := range []string{
		"Prompt template: " + factory.DispatchPromptTemplate, "Recorded attempt active-time limit: 120 minutes",
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
	// Actual packet/current inputs are mandatory at the authenticated
	// launch boundary. An unavailable choice holds without a host call;
	// the qualified frozen choice then admits exact delivery only once.
	guarded := fx.deps()
	guarded.MCPKey = []byte(strings.Repeat("k", 32))
	beforeLaunch := len(fx.host.launches)
	// Simulate a restart after the pending marker commits but before its
	// deterministic capability production. No identity or attempt changes.
	run.ContextPending = true
	if err := db.SaveFactoryRun(ctx, run); err != nil {
		t.Fatal(err)
	}
	held := RecoverDispatch(ctx, guarded)
	if len(held.Launched) != 0 || waitReason(held, a.Issue) != "context_selection_hold" || len(fx.host.launches) != beforeLaunch {
		t.Fatal("unavailable context selection started native work")
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
	held = newDispatchReport()
	launchTail(ctx, guarded, child, childRun, fx.host.launches[beforeLaunch-1], &held)
	if len(held.Launched) != 0 || waitReason(held, child.Issue) != "context_selection_hold" || len(fx.host.launches) != beforeLaunch || uncertain.calls != 1 {
		t.Fatalf("low confidence selection started work: %+v calls=%d", held, uncertain.calls)
	}
	childCapability, err := db.GetMCPCapability(ctx, childRun.ID)
	if err != nil {
		t.Fatal(err)
	}
	childState, ownEligible, lineageEligible, err := loadContextBundleState(ctx, db, childCapability, child, nil)
	if err != nil || ownEligible || !lineageEligible || childState.Current.Required == nil || childState.Current.Required.Acceptance == nil || childState.Current.Required.Plan == nil {
		t.Fatalf("builder comparison lost its actual required inputs: %+v %v", childState, err)
	}
	coverage := childState.Current.Required
	if coverage.Acceptance.ID != oldHead.ID || !coverage.Acceptance.ObjectivePresent || !coverage.Acceptance.RepositoryContext || coverage.Plan.ID != plan.ID || coverage.Plan.Digest != plan.Digest || coverage.Plan.Tasks != len(plan.Tasks) || coverage.Plan.Checks != len(plan.Checks) || coverage.Plan.CurrentHead == nil || coverage.Plan.CurrentHead.PlanID != plan.ID {
		t.Fatalf("builder comparison differs from its admitted current plan: %+v", coverage)
	}
	if len(childState.HistoryPreview) != 1 || childState.HistoryPreview[0].Phase != project.ExecutionInvestigate || childState.HistoryPreview[0].Profile != factory.MCPProfileArchitect || childState.HistoryPreview[0].PlanInCurrent == nil || !*childState.HistoryPreview[0].PlanInCurrent || childState.Bundles[factory.ContextBundleAttempt].HistoryRecords != 0 || len(childState.Bundles[factory.ContextBundleLineage].AdditionalFields) != 3 {
		t.Fatalf("builder comparison blurred parent disposition with required plan: %+v", childState)
	}
	comparison, err := json.Marshal(childState)
	if err != nil || strings.Contains(string(comparison), child.AcceptedContext.Body) || strings.Contains(string(comparison), "plan ready") {
		t.Fatal("context comparison disclosed private objective or parent result prose")
	}
	for pass := 0; pass < 4; pass++ {
		recovered := RecoverDispatch(ctx, guarded)
		current, err := db.Assignment(ctx, child.ID)
		currentRun, runErr := db.FactoryRun(ctx, childRun.ID)
		if err != nil || runErr != nil || current.Run != childRun.ID || current.Attempts != 1 || len(current.RunHistory) != 1 || current.Stage != factory.AssignmentAssigned || !currentRun.ContextPending || currentRun.Reconciled || len(recovered.Launched) != 0 || len(fx.host.launches) != beforeLaunch || uncertain.calls != 1 {
			t.Fatalf("context hold consumed a native attempt or refilled choice on pass %d: assignment=%+v run=%+v report=%+v calls=%d errors=%v/%v", pass, current, currentRun, recovered, uncertain.calls, err, runErr)
		}
	}
	// A separate explicit owner decision can select complete current
	// requirements on this same held packet. It never changes the model's
	// uncertain answer, buys another call, creates a run or extends time.
	heldRun, _ := db.FactoryRun(ctx, childRun.ID)
	selectionView, err := ContextSelectionStatus(ctx, db, heldRun)
	if err != nil || selectionView == nil || !selectionView.CanUseCurrent || selectionView.Action != "hold" {
		t.Fatalf("owner cannot inspect the exact context hold: %+v %v", selectionView, err)
	}
	request := factory.ContextCurrentRequest{Fingerprint: selectionView.Fingerprint, ScopeDigest: selectionView.ScopeDigest, RequiredDigest: selectionView.RequiredDigest}
	ownerCoord := NewCoordinator(db, fx.host, fx.broker)
	ownerCoord.DispatchReads = fx.reads
	ownerCoord.MCPKey = guarded.MCPKey
	// The actual coordinator startup must preserve this never-launched
	// held packet before normal RecoverDispatch can recheck its receipt.
	startup := NewCoordinator(db, fx.host, fx.broker)
	startup.DispatchReads, startup.MCPKey = fx.reads, guarded.MCPKey
	if err := startup.Start(ctx, filepath.Join(t.TempDir(), "context-owner.lock")); err != nil {
		t.Fatal("startup refused held packet", err)
	}
	if err := startup.Close(); err != nil {
		t.Fatal(err)
	}
	retained, _ := db.FactoryRun(ctx, childRun.ID)
	retainedCap, _ := db.GetMCPCapability(ctx, childRun.ID)
	if retained.Reconciled || retained.Outcome != "" || !retained.ContextPending || retainedCap.State != factory.MCPCapabilityStaged || retainedCap.ContextSelectionSnapshot != childCapability.ContextSelectionSnapshot || uncertain.calls != 1 || len(fx.host.launches) != beforeLaunch {
		t.Fatalf("startup retired or refilled the unstarted hold: %+v %+v", retained, retainedCap)
	}
	commandID := factory.NewID()
	wrong := request
	wrong.RequiredDigest = strings.Repeat("e", 64)
	if _, err := ownerCoord.UseCurrentContext(ctx, commandID, "native:7", childRun.ID, wrong); !errors.Is(err, store.ErrContextResolutionIneligible) {
		t.Fatalf("changed required context admitted: %v", err)
	}
	beforeReceipt, _ := db.PriorityJudgment(ctx, selectionView.Fingerprint)
	beforeJSON, _ := json.Marshal(beforeReceipt)
	resolution, err := ownerCoord.UseCurrentContext(ctx, commandID, "native:7", childRun.ID, request)
	if err != nil || resolution.Bundle != factory.ContextBundleCurrent || resolution.RunID != childRun.ID {
		t.Fatalf("owner current choice refused: %+v %v", resolution, err)
	}
	if replay, err := ownerCoord.UseCurrentContext(ctx, commandID, "native:7", childRun.ID, request); err != nil || replay != resolution {
		t.Fatalf("owner choice replay changed: %+v %v", replay, err)
	}
	if _, err := ownerCoord.UseCurrentContext(ctx, commandID, "native:7", childRun.ID, wrong); !errors.Is(err, store.ErrCommandConflict) {
		t.Fatalf("owner choice identity accepted changed bytes: %v", err)
	}
	afterReceipt, _ := db.PriorityJudgment(ctx, selectionView.Fingerprint)
	afterJSON, _ := json.Marshal(afterReceipt)
	if string(beforeJSON) != string(afterJSON) || uncertain.calls != 1 {
		t.Fatal("owner resolution rewrote or re-funded the original uncertain judgment")
	}
	resolved := RecoverDispatch(ctx, guarded)
	if len(resolved.Launched) != 1 || len(fx.host.launches) != beforeLaunch+1 || uncertain.calls != 1 {
		t.Fatalf("owner choice did not resume the exact existing run: %+v calls=%d", resolved, uncertain.calls)
	}
	afterRun, _ := db.FactoryRun(ctx, childRun.ID)
	if afterRun.ContextPending || !afterRun.Deadline.Equal(heldRun.Deadline) {
		t.Fatal("owner context choice changed its deadline or pending identity")
	}
	if _, err := ownerCoord.UseCurrentContext(ctx, factory.NewID(), "native:7", childRun.ID, request); !errors.Is(err, store.ErrContextResolutionIneligible) {
		t.Fatalf("active context accepted a new owner choice: %v", err)
	}
	allowance, err := db.AttemptAllowance(ctx, fx.repo, 3)
	if err != nil || !allowance.Active || allowance.Closed || allowance.RootAssignment != a.AttemptRoot {
		t.Fatalf("settled candidate attempt = %+v, %v", allowance, err)
	}
	t.Run("owner replaces a cancelled unlaunched implementation", func(t *testing.T) {
		db, _ := dispatchTestDB(t)
		fx := dispatchSeed(t, db)
		head := fx.accept(t, 3, "d333333333333333333333333")
		fx.queue(t, 3, head.ID)
		first := DispatchPass(ctx, fx.deps())
		if len(first.Launched) != 1 {
			t.Fatalf("planning launch: %+v", first)
		}
		parent, _ := db.Assignment(ctx, first.Launched[0].AssignmentID)
		parentRun, _ := db.FactoryRun(ctx, parent.Run)
		parentRun.Outcome, parentRun.Summary, parentRun.Reconciled = factory.Succeeded, "plan ready", true
		if err := db.SaveFactoryRun(ctx, parentRun); err != nil {
			t.Fatal(err)
		}
		plan := phaseTestPlan(t, parent.AttemptRoot, parent.SourceCommit, parent.Preparation)
		output := fencedResult(t, factory.PhaseResult{Version: factory.PhaseResultVersion, Phase: project.ExecutionInvestigate, Status: "completed", Summary: "plan ready", Output: &factory.PhaseOutput{Plan: &plan}})
		if _, ok := AccountSettledRun(ctx, db, parentRun, output, time.Now()); !ok {
			t.Fatal("planning settlement refused")
		}
		deps := fx.deps()
		deps.MCPKey = []byte(strings.Repeat("k", 32))
		chooser := &bundleGapChooser{pick: factory.ContextBundleCurrent, confidence: 0.6}
		deps.Priority = &PriorityJudgments{Journal: db, Chooser: chooser}
		occupancy, err := snapshotOccupancy(ctx, db)
		if err != nil {
			t.Fatal(err)
		}
		control, _ := db.IssueControl(ctx, fx.repo, 3)
		held := newDispatchReport()
		dispatchOne(ctx, deps, &occupancy, control, &held)
		if len(held.Launched) != 0 || waitReason(held, 3) != "context_selection_hold" || chooser.calls != 1 {
			t.Fatalf("implementation did not hold: %+v calls=%d", held, chooser.calls)
		}
		child, err := db.LatestIssueAttempt(ctx, fx.repo, 3)
		if err != nil || child.Phase != project.ExecutionImplement {
			t.Fatalf("held child: %+v %v", child, err)
		}
		run, _ := db.FactoryRun(ctx, child.Run)
		beforeCap, _ := db.GetMCPCapability(ctx, run.ID)
		beforeJudgment, _ := db.PriorityJudgment(ctx, beforeCap.ContextSelectionSnapshot)
		// Reproduce the old startup's native stop/settlement through its
		// real coordinator path, without rewriting any cancelled record.
		fx.host.stop = func(in project.FactoryStop) (project.FactoryState, error) {
			return project.FactoryState{ID: in.ID, Project: in.Project, Phase: project.FactoryStopped, Retirement: "confirmed", Reason: "stop-before-start"}, nil
		}
		coord := NewCoordinator(db, fx.host, fx.broker)
		coord.DispatchReads, coord.MCPKey = fx.reads, deps.MCPKey
		if stopped := coord.settleRun(ctx, run); !stopped.Confirmed || stopped.Outcome != string(factory.Cancelled) {
			t.Fatalf("unused stop not settled: %+v", stopped)
		}
		beforeAllowance, _ := db.AttemptAllowance(ctx, fx.repo, 3)
		beforeInventory, _ := db.CapturePriorityInventory(ctx)
		command := factory.NewID()
		decision, err := coord.RetryRun(ctx, command, "native:7", run.ID)
		if err != nil || !decision.Queued || decision.CommandID != command {
			t.Fatalf("owner continuation refused: %+v %v", decision, err)
		}
		if replay, err := coord.RetryRun(ctx, command, "native:7", run.ID); err != nil || replay != decision {
			t.Fatalf("owner continuation replay: %+v %v", replay, err)
		}
		afterInventory, _ := db.CapturePriorityInventory(ctx)
		if beforeInventory.LocalDigest == afterInventory.LocalDigest {
			t.Fatal("owner continuation did not invalidate the cached exclusion")
		}
		occupancy, _ = snapshotOccupancy(ctx, db)
		control, _ = db.IssueControl(ctx, fx.repo, 3)
		resumed := newDispatchReport()
		deps.Priority = &PriorityJudgments{Journal: db, Chooser: &bundleGapChooser{pick: factory.ContextBundleCurrent}}
		dispatchOne(ctx, deps, &occupancy, control, &resumed)
		if len(resumed.Launched) != 1 || resumed.Launched[0].AssignmentID != command {
			t.Fatalf("continuation not launched: %+v", resumed)
		}
		next, _ := db.Assignment(ctx, command)
		old, _ := db.FactoryRun(ctx, run.ID)
		afterAllowance, _ := db.AttemptAllowance(ctx, fx.repo, 3)
		afterJudgment, _ := db.PriorityJudgment(ctx, beforeCap.ContextSelectionSnapshot)
		beforeRaw, _ := json.Marshal(beforeJudgment)
		afterRaw, _ := json.Marshal(afterJudgment)
		if next.AttemptRoot != parent.ID || next.ParentAssignment != parent.ID || next.Preparation != child.Preparation || next.SourceCommit != child.SourceCommit || next.Run == old.ID || !old.Reconciled || old.Outcome != factory.Cancelled || string(beforeRaw) != string(afterRaw) || afterAllowance.Limits != beforeAllowance.Limits || afterAllowance.ActiveSeconds < beforeAllowance.ActiveSeconds || afterAllowance.RootAssignment != beforeAllowance.RootAssignment {
			t.Fatal("continuation replaced old history, plan lineage, paid judgment or cumulative allowance")
		}
		parent, _ = db.Assignment(ctx, parent.ID)
		if _, _, _, _, ready := coord.resolvePublicationOwner(ctx, parent); ready {
			t.Fatal("unfinished continuation supplied a publication candidate")
		}
		nextRun, _ := db.FactoryRun(ctx, next.Run)
		nextRun.Outcome, nextRun.Summary, nextRun.Reconciled = factory.Succeeded, "implementation complete", true
		if err := db.SaveFactoryRun(ctx, nextRun); err != nil {
			t.Fatal(err)
		}
		candidate := strings.Repeat("2", 40)
		implementation := fencedResult(t, factory.PhaseResult{Version: factory.PhaseResultVersion, Phase: project.ExecutionImplement, Status: "completed", Summary: "implementation complete", Output: &factory.PhaseOutput{Implementation: &factory.ImplementationReceipt{Candidate: candidate}}})
		if _, ok := AccountSettledRun(ctx, db, nextRun, implementation, time.Now()); !ok {
			t.Fatal("continuation implementation settlement refused")
		}
		publicationOwner, gotCandidate, gotRun, gotPreparation, ready := coord.resolvePublicationOwner(ctx, parent)
		if !ready || publicationOwner.ID != next.ID || gotCandidate != candidate || gotRun != next.Run || gotPreparation != next.Preparation {
			t.Fatalf("publication ignored the completed current continuation: owner=%s candidate=%s run=%s preparation=%s ready=%t", publicationOwner.ID, gotCandidate, gotRun, gotPreparation, ready)
		}
		old, _ = db.FactoryRun(ctx, old.ID)
		if old.Outcome != factory.Cancelled || !old.Reconciled || old.Summary != "stop-before-start" {
			t.Fatal("publication resolution rewrote the cancelled first child")
		}
	})
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
