package control

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/identity"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
)

func TestDispatchAccountingNeverRefreshes(t *testing.T) {
	ctx := context.Background()
	db, _ := dispatchTestDB(t)
	fx := dispatchSeed(t, db)
	sp, err := db.Sponsorship(ctx, fx.repo, "conn")
	if err != nil {
		t.Fatal(err)
	}
	sp.AllowanceMinutes = 1
	if err := db.SaveSponsorship(ctx, sp); err != nil {
		t.Fatal(err)
	}
	headA := fx.accept(t, 3, "d333333333333333333333333")
	headB := fx.accept(t, 5, "d555555555555555555555555")
	fx.queue(t, 3, headA.ID)
	fx.queue(t, 5, headB.ID)

	first := DispatchPass(ctx, fx.deps())
	if len(first.Launched) != 1 {
		t.Fatalf("first pass = %+v %+v", first.Launched, first.Waits)
	}
	run, err := db.FactoryRun(ctx, first.Launched[0].RunID)
	if err != nil {
		t.Fatal(err)
	}
	run.Outcome, run.Summary, run.Reconciled = factory.Succeeded, "done", true
	if err := db.SaveFactoryRun(ctx, run); err != nil {
		t.Fatal(err)
	}
	if _, ok := AccountSettledRun(ctx, db, run, "", time.Now()); !ok {
		t.Fatal("settled run unaccounted")
	}
	if total, _ := db.UsageTotal(ctx, fx.repo, "conn"); total != 1 {
		t.Fatalf("usage total = %d", total)
	}
	second := DispatchPass(ctx, fx.deps())
	if len(second.Launched) != 0 || waitReason(second, 5) != WaitAllowance {
		t.Fatalf("second pass = %+v %+v", second.Launched, second.Waits)
	}

	// An acceptance edit authorizes new work but refreshes no budget: the
	// re-queued issue still waits on the exhausted allowance.
	edited := headB
	edited.ID = "d666666666666666666666666"
	edited.Predecessor = headB.ID
	if err := db.AdmitAcceptanceDecision(ctx, edited); err != nil {
		t.Fatal(err)
	}
	control, err := db.IssueControl(ctx, fx.repo, 5)
	if err != nil {
		t.Fatal(err)
	}
	control.Acceptance = edited.ID
	control.Fingerprint = strings.Repeat("3", 64)
	if _, changed, err := db.RecordIssueAssessment(ctx, control, time.Now()); err != nil || !changed {
		t.Fatalf("re-queue = %v %v", changed, err)
	}
	third := DispatchPass(ctx, fx.deps())
	if len(third.Launched) != 0 || waitReason(third, 5) != WaitAllowance {
		t.Fatalf("third pass = %+v %+v", third.Launched, third.Waits)
	}
	if total, _ := db.UsageTotal(ctx, fx.repo, "conn"); total != 1 {
		t.Fatalf("usage total after edit = %d", total)
	}

	// A withdrawal and authorized resume likewise change no usage.
	coord := &Coordinator{Store: db}
	if _, _, err := coord.withdrawAndStopRuns(ctx, fx.repo, "test", "tester"); err != nil {
		t.Fatal(err)
	}
	_, revision, _, err := db.DispatchState(ctx, fx.repo)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := coord.ReopenDispatch(ctx, factory.NewID(), "tester", fx.repo, revision); err != nil {
		t.Fatal(err)
	}
	fourth := DispatchPass(ctx, fx.deps())
	if len(fourth.Launched) != 0 || waitReason(fourth, 5) != WaitAllowance {
		t.Fatalf("fourth pass = %+v %+v", fourth.Launched, fourth.Waits)
	}
	if total, _ := db.UsageTotal(ctx, fx.repo, "conn"); total != 1 {
		t.Fatalf("usage total after resume = %d", total)
	}
}

func TestRecordConfirmedUsagePreservesFractionalIntervalAndFirstWrite(t *testing.T) {
	ctx := context.Background()
	db, _ := dispatchTestDB(t)
	fx := dispatchSeed(t, db)
	start := time.Date(2026, 10, 1, 0, 0, 0, 123456789, time.FixedZone("test", 2*60*60))
	run := factory.Run{ID: factory.NewID(), Started: start}
	a := factory.Assignment{Repository: fx.repo, Connection: "conn", ActorID: 7}
	end := start.Add(time.Minute + time.Nanosecond)
	if err := recordConfirmedUsage(ctx, db, a, run, end); err != nil {
		t.Fatal(err)
	}
	if err := recordConfirmedUsage(ctx, db, a, run, end.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	got, err := db.RunUsage(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Minutes != 2 || !got.StartedAt.Equal(start) || !got.EndedAt.Equal(end) || got.StartedAt.Location() != time.UTC || got.EndedAt.Location() != time.UTC {
		t.Fatalf("usage interval lost precision or replay changed first write: %+v", got)
	}
}

func TestDispatchInterruptionReleasesOnlyUnused(t *testing.T) {
	ctx := context.Background()
	db, _ := dispatchTestDB(t)
	fx := dispatchSeed(t, db)
	head := fx.accept(t, 3, "d333333333333333333333333")
	fx.queue(t, 3, head.ID)
	fx.host.launch = func(in project.FactoryLaunch) (project.FactoryState, error) {
		return project.FactoryState{}, errors.New("host down")
	}

	first := DispatchPass(ctx, fx.deps())
	if len(first.Launched) != 0 || len(first.Waits) != 1 || first.Waits[0].Reason != WaitLaunchRefused {
		t.Fatalf("first pass = %+v %+v %+v", first.Launched, first.Waits, first.Errors)
	}
	assignment, err := db.LatestIssueAssignment(ctx, fx.repo, 3)
	if err != nil {
		t.Fatalf("latest assignment: %v", err)
	}
	if assignment.Attempts != 1 || assignment.Stage != factory.AssignmentAssigned {
		t.Fatalf("assignment = %+v", assignment)
	}
	run, err := db.FactoryRun(ctx, assignment.Run)
	if err != nil || !run.Reconciled || run.Outcome != factory.Failed {
		t.Fatalf("run = %+v %v", run, err)
	}
	reservation, err := db.Reservation(ctx, assignment.ID)
	if err != nil || reservation.State != factory.ReservationReleased {
		t.Fatalf("reservation = %+v %v", reservation, err)
	}
	if total, _ := db.UsageTotal(ctx, fx.repo, "conn"); total != 0 {
		t.Fatalf("usage total = %d", total)
	}

	second := DispatchPass(ctx, fx.deps())
	if len(second.Launched) != 0 || waitReason(second, 3) != WaitLaunchRefused {
		t.Fatalf("second pass = %+v %+v", second.Launched, second.Waits)
	}
	third := DispatchPass(ctx, fx.deps())
	if len(third.Launched) != 0 || waitReason(third, 3) != WaitLaunchExhausted {
		t.Fatalf("third pass = %+v %+v %+v", third.Launched, third.Waits, third.Errors)
	}
	done, err := db.Assignment(ctx, assignment.ID)
	if err != nil || done.Stage != factory.AssignmentFinished || done.Outcome != factory.Failed || done.Attempts != 3 {
		t.Fatalf("assignment = %+v %v", done, err)
	}
	if len(done.RunHistory) != 3 {
		t.Fatalf("run history = %v", done.RunHistory)
	}
	// A fourth pass finds the terminal attempt recorded, not new work.
	fourth := DispatchPass(ctx, fx.deps())
	if len(fourth.Waits) != 1 || fourth.Waits[0].Reason != WaitAttemptRecorded {
		t.Fatalf("fourth pass = %+v %+v", fourth.Launched, fourth.Waits)
	}
}

func TestReleasedLaunchRetryKeepsPromptAllowanceWhenUsageShortensDeadline(t *testing.T) {
	ctx := context.Background()
	db, _ := dispatchTestDB(t)
	fx := dispatchSeed(t, db)
	head := fx.accept(t, 3, "d333333333333333333333333")
	fx.queue(t, 3, head.ID)

	launchCalls := 0
	fx.host.launch = func(in project.FactoryLaunch) (project.FactoryState, error) {
		launchCalls++
		if launchCalls == 1 {
			return project.FactoryState{}, errors.New("host down")
		}
		return project.FactoryState{ID: in.Run.ID, Project: in.Run.Project, Role: in.Run.Role, Phase: project.FactoryCompleted}, nil
	}
	first := DispatchPass(ctx, fx.deps())
	if len(first.Launched) != 0 || waitReason(first, 3) != WaitLaunchRefused {
		t.Fatalf("first pass = %+v %+v %+v", first.Launched, first.Waits, first.Errors)
	}
	assignment, err := db.LatestIssueAssignment(ctx, fx.repo, 3)
	if err != nil {
		t.Fatal(err)
	}
	firstRun, err := db.FactoryRun(ctx, assignment.Run)
	if err != nil {
		t.Fatal(err)
	}
	firstPromptSHA := assignment.PromptSHA
	firstSpan := firstRun.Deadline.Sub(firstRun.Started)
	if firstSpan < 119*time.Minute || firstSpan > 120*time.Minute {
		t.Fatalf("first deadline span = %v", firstSpan)
	}
	rootAllowance, err := db.AttemptAllowance(ctx, fx.repo, 3)
	if err != nil {
		t.Fatal(err)
	}

	// Record a separate settled run's provider charge between retries.
	now := time.Now().UTC()
	otherRun := firstRun
	otherRun.ID = factory.NewID()
	otherRun.Admission = nil // This usage-control run is standalone, not a dispatch packet.
	otherRun.Started = now.Add(-2 * time.Minute)
	otherRun.Outcome, otherRun.Summary, otherRun.Reconciled = "", "", false
	if err := db.RecordFactoryRun(ctx, otherRun); err != nil {
		t.Fatal(err)
	}
	otherRun.Outcome, otherRun.Summary, otherRun.Reconciled = factory.Failed, "settled", true
	if err := db.SaveFactoryRun(ctx, otherRun); err != nil {
		t.Fatal(err)
	}
	if err := db.RecordRunUsage(ctx, factory.Usage{
		RunID: otherRun.ID, Repository: fx.repo, Connection: "conn", Minutes: 2,
		StartedAt: otherRun.Started, EndedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	second := DispatchPass(ctx, fx.deps())
	if len(second.Launched) != 1 || second.Launched[0].Issue != 3 {
		t.Fatalf("second pass = launched %+v waits %+v errors %+v", second.Launched, second.Waits, second.Errors)
	}
	updated, err := db.Assignment(ctx, assignment.ID)
	if err != nil {
		t.Fatal(err)
	}
	secondRun, err := db.FactoryRun(ctx, second.Launched[0].RunID)
	if err != nil {
		t.Fatal(err)
	}
	sent := fx.host.launches[len(fx.host.launches)-1]
	if updated.PromptSHA != firstPromptSHA || sent.Run.Assignment != firstPromptSHA {
		t.Fatalf("retry prompt SHA changed: first=%s assignment=%s launch=%s", firstPromptSHA, updated.PromptSHA, sent.Run.Assignment)
	}
	if !strings.Contains(string(updated.Prompt), "120 minutes") {
		t.Fatalf("retry prompt lost configured 120-minute allowance: %s", updated.Prompt)
	}
	if span := secondRun.Deadline.Sub(secondRun.Started); span >= firstSpan-time.Minute || span < 117*time.Minute {
		t.Fatalf("retry deadline span = %v, first = %v", span, firstSpan)
	}
	afterAllowance, err := db.AttemptAllowance(ctx, fx.repo, 3)
	if err != nil {
		t.Fatal(err)
	}
	if afterAllowance.RootAssignment != rootAllowance.RootAssignment || afterAllowance.Limits != rootAllowance.Limits {
		t.Fatalf("root allowance changed: before=%+v after=%+v", rootAllowance, afterAllowance)
	}
}

func TestExplicitRetryCreatesFreshAttempt(t *testing.T) {
	ctx := context.Background()
	db, _ := dispatchTestDB(t)
	fx := dispatchSeed(t, db)
	head := fx.accept(t, 3, "d333333333333333333333333")
	fx.queue(t, 3, head.ID)
	fx.host.launch = func(project.FactoryLaunch) (project.FactoryState, error) {
		return project.FactoryState{}, errors.New("host down")
	}
	for i := 0; i < factory.MaxDispatchAttempts; i++ {
		report := DispatchPass(ctx, fx.deps())
		if len(report.Errors) != 0 {
			t.Fatalf("failed unused attempt %d: %+v", i+1, report.Errors)
		}
	}
	old, err := db.LatestIssueAssignment(ctx, fx.repo, 3)
	if err != nil {
		t.Fatalf("settled assignment: %v", err)
	}
	if old.Stage != factory.AssignmentFinished || old.Outcome != factory.Failed || old.Attempts != factory.MaxDispatchAttempts || len(old.RunHistory) != factory.MaxDispatchAttempts {
		t.Fatalf("old attempts did not settle at their bound: %+v", old)
	}
	fourth := DispatchPass(ctx, fx.deps())
	if len(fourth.Waits) != 1 || fourth.Waits[0].Reason != WaitAttemptRecorded {
		t.Fatalf("settled retry remained dispatchable before explicit retry: %+v %+v", fourth.Launched, fourth.Waits)
	}
	oldHistory := append([]string(nil), old.RunHistory...)
	priorRun, err := db.FactoryRun(ctx, old.Run)
	if err != nil || !priorRun.Reconciled || priorRun.Outcome != factory.Failed {
		t.Fatalf("latest failed run: %+v %v", priorRun, err)
	}
	oldAllowance, err := db.AttemptAllowance(ctx, fx.repo, 3)
	if err != nil {
		t.Fatalf("previous attempt allowance: %v", err)
	}
	policy, err := db.RepositoryPolicy(ctx, fx.repo)
	if err != nil {
		t.Fatalf("repository policy: %v", err)
	}
	policy.AttemptLimits.ActiveMinutes = 45
	if err := db.SaveRepositoryPolicy(ctx, policy); err != nil {
		t.Fatalf("lower current attempt allowance: %v", err)
	}

	var retriedRun string
	fx.host.launch = func(in project.FactoryLaunch) (project.FactoryState, error) {
		retriedRun = in.Run.ID
		return project.FactoryState{ID: in.Run.ID, Project: in.Run.Project, Role: in.Run.Role, Phase: project.FactoryCompleted}, nil
	}
	coord := &Coordinator{Store: db}
	commandID := factory.NewID()
	decision, err := coord.RetryRun(ctx, commandID, "native:7", priorRun.ID)
	if err != nil || !decision.Queued || decision.Prior != priorRun.ID {
		t.Fatalf("explicit retry: %+v %v", decision, err)
	}

	launchesBeforeRetry := len(fx.host.launches)
	report := DispatchPass(ctx, fx.deps())
	if len(report.Launched) != 1 || len(fx.host.launches) != launchesBeforeRetry+1 {
		t.Fatalf("fresh explicit retry was not launched once: launched=%+v waits=%+v errors=%+v calls=%d", report.Launched, report.Waits, report.Errors, len(fx.host.launches))
	}
	fresh, err := db.Assignment(ctx, commandID)
	if err != nil {
		t.Fatalf("fresh retry assignment: %v", err)
	}
	if fresh.ID != commandID || fresh.AttemptRoot != fresh.ID || fresh.PublicationAssignment != fresh.ID || fresh.Run == old.Run || fresh.Acceptance != head.ID || fresh.Repository != fx.repo || fresh.Issue != 3 || fresh.Attempts != 1 || len(fresh.RunHistory) != 1 || fresh.RunHistory[0] != fresh.Run {
		t.Fatalf("fresh retry assignment did not bind current accepted work: %+v", fresh)
	}
	freshRun, err := db.FactoryRun(ctx, fresh.Run)
	if err != nil || freshRun.Reconciled || freshRun.Outcome != "" {
		t.Fatalf("fresh retry run: %+v %v", freshRun, err)
	}
	reservation, err := db.Reservation(ctx, fresh.ID)
	if err != nil || reservation.State != factory.ReservationHeld || reservation.Connection != "conn" || reservation.PlannedMinutes <= 0 {
		t.Fatalf("fresh retry allowance reservation: %+v %v", reservation, err)
	}
	freshAllowance, err := db.AttemptAllowance(ctx, fx.repo, 3)
	if err != nil || freshAllowance.RootAssignment != commandID || freshAllowance.Limits.ActiveMinutes != 45 || !freshAllowance.Active || freshAllowance.Closed {
		t.Fatalf("fresh retry root allowance: %+v %v", freshAllowance, err)
	}
	if oldAllowance.RootAssignment == freshAllowance.RootAssignment || oldAllowance.Limits.ActiveMinutes != factory.DefaultAttemptLimits().ActiveMinutes {
		t.Fatalf("explicit retry did not create a fresh policy allowance: old=%+v fresh=%+v", oldAllowance, freshAllowance)
	}
	if currentOld, err := db.Assignment(ctx, old.ID); err != nil || currentOld.Stage != factory.AssignmentFinished || len(currentOld.RunHistory) != len(oldHistory) {
		t.Fatalf("old attempt history changed: %+v %v", currentOld, err)
	} else {
		for i := range oldHistory {
			if currentOld.RunHistory[i] != oldHistory[i] {
				t.Fatalf("old run history changed: %v", currentOld.RunHistory)
			}
		}
	}
	publication := factory.Publication{AssignmentID: old.ID, Repository: fx.repo, Issue: 3}
	linked, err := coord.assignmentForPublication(ctx, publication)
	if err != nil || linked.ID != old.ID {
		t.Fatalf("publication lookup did not return its exact older assignment: %+v %v", linked, err)
	}
	publication.Issue++
	if _, err = coord.assignmentForPublication(ctx, publication); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("publication lookup accepted mismatched issue: %v", err)
	}
	if total, err := db.UsageTotal(ctx, fx.repo, "conn"); err != nil || total != 0 {
		t.Fatalf("confirmed-unused attempts consumed allowance: total=%d err=%v", total, err)
	}

	fx.host.inspect = func(in project.FactoryInspect) (project.FactoryState, error) {
		if in.ID != retriedRun {
			return project.FactoryState{}, errors.New("unexpected run inspection")
		}
		return project.FactoryState{ID: in.ID, Project: in.Project, Role: project.RoleCoder, Phase: project.FactoryRunning}, nil
	}
	launchCount := len(fx.host.launches)
	replay, err := coord.RetryRun(ctx, commandID, "native:7", priorRun.ID)
	if err != nil || replay != decision || len(fx.host.launches) != launchCount {
		t.Fatalf("retry replay duplicated fresh launch: replay=%+v err=%v launches=%d/%d", replay, err, len(fx.host.launches), launchCount)
	}
	latest, err := db.LatestIssueAssignment(ctx, fx.repo, 3)
	if err != nil || latest.ID != commandID {
		t.Fatalf("retry replay replaced the consumed assignment: %+v %v", latest, err)
	}
	again := DispatchPass(ctx, fx.deps())
	if len(again.Launched) != 0 || len(fx.host.launches) != launchCount {
		t.Fatalf("replayed retry launched again: %+v calls=%d", again, len(fx.host.launches))
	}

	// Grow real command and assignment history beyond the former issue-list
	// page. Each retry uses the ordinary command, dispatch and accounting
	// paths, so targeted reads still work after substantial history growth.
	currentRun := freshRun
	currentCommand := commandID
	retainedAssignmentIDs := []string{old.ID, commandID}
	for i := 0; i < factory.MaxDispatchAttempts+8; i++ {
		settledRun := currentRun
		settledRun.Outcome, settledRun.Summary, settledRun.Reconciled = factory.Failed, "worker failed", true
		if err := db.SaveFactoryRun(ctx, settledRun); err != nil {
			t.Fatalf("settle history run %d: %v", i, err)
		}
		if _, ok := AccountSettledRun(ctx, db, settledRun, "", time.Now()); !ok {
			t.Fatalf("account history run %d", i)
		}
		currentCommand = factory.NewID()
		retainedAssignmentIDs = append(retainedAssignmentIDs, currentCommand)
		if _, err := coord.RetryRun(ctx, currentCommand, "native:7", settledRun.ID); err != nil {
			t.Fatalf("queue history retry %d: %v", i, err)
		}
		retryReport := DispatchPass(ctx, fx.deps())
		if len(retryReport.Launched) != 1 {
			t.Fatalf("launch history retry %d: %+v %+v", i, retryReport.Launched, retryReport.Waits)
		}
		currentRun, err = db.FactoryRun(ctx, retryReport.Launched[0].RunID)
		if err != nil || currentRun.ID == settledRun.ID {
			t.Fatalf("history retry run %d: %+v %v", i, currentRun, err)
		}
	}
	latest, err = db.LatestIssueAssignment(ctx, fx.repo, 3)
	if err != nil || latest.ID != currentCommand {
		t.Fatalf("latest assignment after bounded history growth: %+v %v", latest, err)
	}
	if len(retainedAssignmentIDs) != 13 {
		t.Fatalf("history rows = %d, want 13", len(retainedAssignmentIDs))
	}
	for _, id := range retainedAssignmentIDs {
		assignment, err := db.Assignment(ctx, id)
		if err != nil || assignment.ID != id {
			t.Fatalf("historical assignment %s was not retained: %+v %v", id, assignment, err)
		}
	}
	publication = factory.Publication{AssignmentID: latest.ID, Repository: fx.repo, Issue: 3}
	linked, err = coord.assignmentForPublication(ctx, publication)
	if err != nil || linked.ID != latest.ID {
		t.Fatalf("publication lookup missed assignment beyond bounded page: %+v %v", linked, err)
	}
	publication.Repository++
	if _, err = coord.assignmentForPublication(ctx, publication); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("publication lookup accepted mismatched repository: %v", err)
	}
}

func TestExplicitRetryRejectsPartialAccounting(t *testing.T) {
	ctx := context.Background()
	db, _ := dispatchTestDB(t)
	fx := dispatchSeed(t, db)
	head := fx.accept(t, 3, "d333333333333333333333333")
	fx.queue(t, 3, head.ID)
	launched := DispatchPass(ctx, fx.deps())
	if len(launched.Launched) != 1 {
		t.Fatalf("initial attempt: %+v %+v", launched.Launched, launched.Waits)
	}
	prior, err := db.FactoryRun(ctx, launched.Launched[0].RunID)
	if err != nil {
		t.Fatal(err)
	}
	assignment, err := db.AssignmentByRun(ctx, prior.ID)
	if err != nil {
		t.Fatal(err)
	}
	prior.Outcome, prior.Summary, prior.Reconciled = factory.Failed, "worker failed", true
	if err = db.SaveFactoryRun(ctx, prior); err != nil {
		t.Fatal(err)
	}
	if err = db.RecordRunUsage(ctx, factory.Usage{
		RunID: prior.ID, Repository: fx.repo, Connection: "conn", StartedAt: prior.Started, EndedAt: prior.Started,
	}); err != nil {
		t.Fatal(err)
	}
	coord := &Coordinator{Store: db}
	commandID := factory.NewID()
	if _, err = coord.RetryRun(ctx, commandID, "native:7", prior.ID); err == nil {
		t.Fatal("explicit retry accepted an unsettled linked assignment")
	}
	if _, err = db.FactoryCommand(ctx, commandID); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("unsettled retry left a durable command: %v", err)
	}
	if err = finishFromRun(ctx, db, assignment, prior, "", time.Now()); err != nil {
		t.Fatal(err)
	}
	reservation, err := db.Reservation(ctx, assignment.ID)
	if err != nil || reservation.State != factory.ReservationHeld {
		t.Fatalf("partial accounting setup: %+v %v", reservation, err)
	}
	commandID = factory.NewID()
	if _, err = coord.RetryRun(ctx, commandID, "native:7", prior.ID); err == nil {
		t.Fatal("explicit retry accepted usage without the matching consumed reservation")
	}
	if _, err = db.FactoryCommand(ctx, commandID); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("partial retry left a durable command: %v", err)
	}
}

func TestDispatchApprovedReceiptReleasesWhenBrokerUnknown(t *testing.T) {
	ctx := context.Background()
	db, _ := dispatchTestDB(t)
	fx := dispatchSeed(t, db)
	head := fx.accept(t, 3, "d333333333333333333333333")
	fx.queue(t, 3, head.ID)
	fx.host.launch = func(in project.FactoryLaunch) (project.FactoryState, error) {
		return project.FactoryState{}, errors.New("acquire refused")
	}
	fx.host.inspect = func(in project.FactoryInspect) (project.FactoryState, error) {
		return project.FactoryState{ID: in.ID, Project: in.Project, Phase: project.FactoryApproved}, nil
	}
	report := DispatchPass(ctx, fx.deps())
	if len(report.Launched) != 0 || waitReason(report, 3) != WaitLaunchRefused {
		t.Fatalf("report = %+v %+v %+v", report.Launched, report.Waits, report.Errors)
	}
	assignment, err := db.LatestIssueAssignment(ctx, fx.repo, 3)
	if err != nil {
		t.Fatalf("latest assignment: %v", err)
	}
	if reservation, _ := db.Reservation(ctx, assignment.ID); reservation.State != factory.ReservationReleased {
		t.Fatalf("reservation = %+v", reservation)
	}
	if total, _ := db.UsageTotal(ctx, fx.repo, "conn"); total != 0 {
		t.Fatalf("phantom usage = %d", total)
	}
}

func TestDispatchApprovedReceiptFencesWhenBrokerHolds(t *testing.T) {
	ctx := context.Background()
	db, _ := dispatchTestDB(t)
	fx := dispatchSeed(t, db)
	head := fx.accept(t, 3, "d333333333333333333333333")
	fx.queue(t, 3, head.ID)
	fx.host.launch = func(in project.FactoryLaunch) (project.FactoryState, error) {
		return project.FactoryState{}, errors.New("launch outcome unconfirmed")
	}
	fx.host.inspect = func(in project.FactoryInspect) (project.FactoryState, error) {
		return project.FactoryState{ID: in.ID, Project: in.Project, Phase: project.FactoryApproved}, nil
	}
	fx.broker.get = func(kind, id string) (identity.Execution, error) {
		return identity.Execution{Kind: kind, ExecutionID: id, State: identity.ExecutionLive, Digest: "d"}, nil
	}
	report := DispatchPass(ctx, fx.deps())
	if len(report.Errors) != 1 || report.Errors[0].Reason != DispatchErrFence {
		t.Fatalf("report = %+v %+v %+v", report.Launched, report.Waits, report.Errors)
	}
	assignment, err := db.LatestIssueAssignment(ctx, fx.repo, 3)
	if err != nil {
		t.Fatalf("latest assignment: %v", err)
	}
	if reservation, _ := db.Reservation(ctx, assignment.ID); reservation.State != factory.ReservationHeld {
		t.Fatalf("reservation = %+v", reservation)
	}
}

func TestDispatchFencedLaunchStaysHeld(t *testing.T) {
	ctx := context.Background()
	db, _ := dispatchTestDB(t)
	fx := dispatchSeed(t, db)
	head := fx.accept(t, 3, "d333333333333333333333333")
	fx.queue(t, 3, head.ID)
	fx.host.launch = func(in project.FactoryLaunch) (project.FactoryState, error) {
		return project.FactoryState{}, errors.New("launch outcome unconfirmed")
	}
	fx.broker.get = func(kind, id string) (identity.Execution, error) {
		return identity.Execution{Kind: kind, ExecutionID: id, State: identity.ExecutionLive, Digest: "d"}, nil
	}
	report := DispatchPass(ctx, fx.deps())
	if len(report.Launched) != 0 || len(report.Errors) != 1 || report.Errors[0].Reason != DispatchErrFence {
		t.Fatalf("report = %+v %+v %+v", report.Launched, report.Waits, report.Errors)
	}
	assignment, err := db.LatestIssueAssignment(ctx, fx.repo, 3)
	if err != nil {
		t.Fatalf("latest assignment: %v", err)
	}
	run, err := db.FactoryRun(ctx, assignment.Run)
	if err != nil || run.Reconciled {
		t.Fatalf("fenced run settled: %+v %v", run, err)
	}
	reservation, err := db.Reservation(ctx, assignment.ID)
	if err != nil || reservation.State != factory.ReservationHeld {
		t.Fatalf("reservation = %+v %v", reservation, err)
	}
}

func TestAccountSettledRunResults(t *testing.T) {
	ctx := context.Background()
	setup := func(t *testing.T) (*store.Store, factory.Assignment, factory.Run) {
		t.Helper()
		db, _ := dispatchTestDB(t)
		fx := dispatchSeed(t, db)
		head := fx.accept(t, 3, "d333333333333333333333333")
		fx.queue(t, 3, head.ID)
		report := DispatchPass(ctx, fx.deps())
		if len(report.Launched) != 1 {
			t.Fatalf("launched = %+v", report.Launched)
		}
		a, err := db.Assignment(ctx, report.Launched[0].AssignmentID)
		if err != nil {
			t.Fatal(err)
		}
		run, err := db.FactoryRun(ctx, report.Launched[0].RunID)
		if err != nil {
			t.Fatal(err)
		}
		return db, a, run
	}
	settle := func(t *testing.T, db *store.Store, run factory.Run, outcome factory.Outcome) factory.Run {
		t.Helper()
		run.Outcome, run.Summary, run.Reconciled = outcome, "host says so", true
		if err := db.SaveFactoryRun(ctx, run); err != nil {
			t.Fatal(err)
		}
		return run
	}
	setupReviewer := func(t *testing.T) (*store.Store, factory.Assignment, factory.Run) {
		t.Helper()
		fx, _, _, run := reviewRunSeed(t, false)
		view, err := fx.db.FactoryRunView(ctx, run.ID)
		if err != nil {
			t.Fatal(err)
		}
		assignment, err := fx.db.Assignment(ctx, view.Attempt)
		if err != nil {
			t.Fatal(err)
		}
		return fx.db, assignment, run
	}

	t.Run("reported candidate", func(t *testing.T) {
		db, a, run := setup(t)
		run = settle(t, db, run, factory.Succeeded)
		candidate := strings.Repeat("d", 40)
		output := "```result-json\n" +
			`{"status":"completed","summary":"fixed it","candidate":"` + candidate + `","review_passed":false,"findings":[]}` + "\n```"
		finished, ok := AccountSettledRun(ctx, db, run, output, time.Now())
		if !ok || finished.Outcome != factory.Succeeded || finished.Reason != factory.AssignReasonReported {
			t.Fatalf("accounted = %+v %v", finished, ok)
		}
		if finished.Result == nil || !finished.Result.Reported || finished.Result.Candidate != candidate {
			t.Fatalf("result = %+v", finished.Result)
		}
		if r, _ := db.Reservation(ctx, a.ID); r.State != factory.ReservationConsumed {
			t.Fatalf("reservation = %+v", r)
		}
		if total, _ := db.UsageTotal(ctx, 7, "conn"); total < 1 {
			t.Fatalf("usage = %d", total)
		}
		if again, ok := AccountSettledRun(ctx, db, run, output, time.Now()); ok || again.Stage != "" {
			t.Fatalf("second accounting = %+v %v", again, ok)
		}
	})
	t.Run("completed without report needs human", func(t *testing.T) {
		db, _, run := setup(t)
		run = settle(t, db, run, factory.Succeeded)
		finished, ok := AccountSettledRun(ctx, db, run, "chatty output, no fence", time.Now())
		if !ok || finished.Outcome != factory.NeedsHuman || finished.Result.Status != "blocked" {
			t.Fatalf("accounted = %+v %v", finished, ok)
		}
	})
	t.Run("failed run", func(t *testing.T) {
		db, _, run := setup(t)
		run = settle(t, db, run, factory.Failed)
		finished, ok := AccountSettledRun(ctx, db, run, "", time.Now())
		if !ok || finished.Outcome != factory.Failed || finished.Result.Status != "failed" {
			t.Fatalf("accounted = %+v %v", finished, ok)
		}
	})
	t.Run("cancelled run", func(t *testing.T) {
		db, _, run := setup(t)
		run = settle(t, db, run, factory.Cancelled)
		finished, ok := AccountSettledRun(ctx, db, run, "", time.Now())
		if !ok || finished.Outcome != factory.Cancelled || finished.Result.Status != "cancelled" {
			t.Fatalf("accounted = %+v %v", finished, ok)
		}
	})
	t.Run("unassigned run ignored", func(t *testing.T) {
		db, _, _ := setup(t)
		human := factory.Run{
			ID: factory.NewID(), ProjectID: "p765432109876543210987654", Role: project.RoleCoder,
			InputSHA: strings.Repeat("c", 40), Started: time.Now(), Deadline: time.Now().Add(time.Hour),
			Image: "sha256:" + strings.Repeat("b", 64), Harness: "codex-1.2.3", Model: "m",
			Outcome: factory.Succeeded, Summary: "ok", Reconciled: true,
		}
		if _, ok := AccountSettledRun(ctx, db, human, "", time.Now()); ok {
			t.Fatal("unassigned run accounted")
		}
	})
	for _, tc := range []struct {
		name, verdict, summary, body string
	}{
		{name: "approve", verdict: "approve", summary: "No blocking findings."},
		{name: "request changes", verdict: "request-changes", summary: "Fix the edge case.", body: "Empty input panics."},
	} {
		t.Run("reviewer "+tc.name, func(t *testing.T) {
			db, assignment, run := setupReviewer(t)
			run = settle(t, db, run, factory.Succeeded)
			output := "```review-json\n" +
				`{"verdict":"` + tc.verdict + `","summary":"` + tc.summary + `","body":"` + tc.body + `","findings":[]}` + "\n```"
			finished, ok := AccountSettledRun(ctx, db, run, output, time.Now())
			if !ok || finished.Outcome != factory.Succeeded || finished.Result == nil || !finished.Result.Reported || finished.Result.Review == nil {
				t.Fatalf("reviewer result = %+v, %v", finished, ok)
			}
			if finished.Result.Candidate != assignment.SourceCommit || finished.Result.Review.Verdict != tc.verdict || finished.Result.Review.Body != tc.body {
				t.Fatalf("reviewer candidate/report = %+v", finished.Result)
			}
			if reservation, err := db.Reservation(ctx, assignment.ID); err != nil || reservation.State != factory.ReservationConsumed {
				t.Fatalf("review reservation = %+v, %v", reservation, err)
			}
			if total, err := db.UsageTotal(ctx, assignment.Repository, assignment.Connection); err != nil || total < 1 {
				t.Fatalf("review usage = %d, %v", total, err)
			}
			if replay, err := db.Assignment(ctx, assignment.ID); err != nil || replay.Result == nil || !reflect.DeepEqual(replay.Result.Review, finished.Result.Review) {
				t.Fatalf("persisted reviewer result = %+v, %v", replay.Result, err)
			}
			if replayed, ok := AccountSettledRun(ctx, db, run, output, time.Now()); ok || replayed.Stage != "" {
				t.Fatalf("review accounting replay = %+v, %v", replayed, ok)
			}
		})
	}
	t.Run("malformed reviewer report needs human", func(t *testing.T) {
		db, assignment, run := setupReviewer(t)
		run = settle(t, db, run, factory.Succeeded)
		output := "```review-json\n{\"verdict\":\"maybe\",\"summary\":\"unclear\",\"body\":\"\",\"findings\":[]}\n```"
		finished, ok := AccountSettledRun(ctx, db, run, output, time.Now())
		if !ok || finished.Outcome != factory.NeedsHuman || finished.Result == nil || finished.Result.Reported || finished.Result.Review != nil {
			t.Fatalf("malformed reviewer result = %+v %v", finished, ok)
		}
		if reservation, err := db.Reservation(ctx, assignment.ID); err != nil || reservation.State != factory.ReservationConsumed {
			t.Fatalf("malformed review reservation = %+v, %v", reservation, err)
		}
	})
	t.Run("review fence is not a coder report", func(t *testing.T) {
		db, _, run := setup(t)
		run = settle(t, db, run, factory.Succeeded)
		output := "```review-json\n" +
			`{"verdict":"approve","summary":"approved","body":"","findings":[]}` + "\n```"
		finished, ok := AccountSettledRun(ctx, db, run, output, time.Now())
		if !ok || finished.Outcome != factory.NeedsHuman || finished.Result == nil || finished.Result.Reported || finished.Result.Review != nil {
			t.Fatalf("coder accepted reviewer fence: %+v %v", finished, ok)
		}
	})
}
