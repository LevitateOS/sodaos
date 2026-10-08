package control

import (
	"context"
	"errors"
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
	a := factory.Assignment{Repository: fx.repo, Connection: "conn"}
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
	list, err := db.IssueAssignments(ctx, fx.repo, 3)
	if err != nil || len(list) != 1 {
		t.Fatalf("assignments = %+v %v", list, err)
	}
	if list[0].Attempts != 1 || list[0].Stage != factory.AssignmentAssigned {
		t.Fatalf("assignment = %+v", list[0])
	}
	run, err := db.FactoryRun(ctx, list[0].Run)
	if err != nil || !run.Reconciled || run.Outcome != factory.Failed {
		t.Fatalf("run = %+v %v", run, err)
	}
	reservation, err := db.Reservation(ctx, list[0].ID)
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
	done, err := db.Assignment(ctx, list[0].ID)
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
	list, err := db.IssueAssignments(ctx, fx.repo, 3)
	if err != nil || len(list) != 1 {
		t.Fatalf("assignments = %+v %v", list, err)
	}
	if reservation, _ := db.Reservation(ctx, list[0].ID); reservation.State != factory.ReservationReleased {
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
	list, err := db.IssueAssignments(ctx, fx.repo, 3)
	if err != nil || len(list) != 1 {
		t.Fatalf("assignments = %+v %v", list, err)
	}
	if reservation, _ := db.Reservation(ctx, list[0].ID); reservation.State != factory.ReservationHeld {
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
	list, err := db.IssueAssignments(ctx, fx.repo, 3)
	if err != nil || len(list) != 1 {
		t.Fatalf("assignments = %+v %v", list, err)
	}
	run, err := db.FactoryRun(ctx, list[0].Run)
	if err != nil || run.Reconciled {
		t.Fatalf("fenced run settled: %+v %v", run, err)
	}
	reservation, err := db.Reservation(ctx, list[0].ID)
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
}
