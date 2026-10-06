package store

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
)

func retryTestRun(t *testing.T, a factory.Assignment, run factory.Run) (factory.Run, factory.RunView) {
	t.Helper()
	id := factory.NewID()
	next := factory.Run{
		ID: id, ProjectID: a.ProjectID, Role: a.Role, InputSHA: run.InputSHA,
		Started: run.Started, Deadline: run.Deadline,
		Image: run.Image, Harness: run.Harness, Model: run.Model,
	}
	return next, factory.RunView{RunID: id, Repository: a.Repository, Issue: a.Issue, Attempt: a.ID}
}

func TestRecordRetryPacketBoundsAttemptsAndFinishes(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	db := dispatchStoreFixture(t)
	a, r, run, view := dispatchTestPacket(t, now)
	if err := db.RecordDispatchPacket(ctx, dispatchTestRegistration(a), a, r, run, view); err != nil {
		t.Fatal(err)
	}
	settle := func(id string) {
		t.Helper()
		stored, err := db.FactoryRun(ctx, id)
		if err != nil {
			t.Fatal(err)
		}
		stored.Outcome, stored.Reconciled, stored.Summary = factory.Failed, true, "superseded"
		if err := db.SaveFactoryRun(ctx, stored); err != nil {
			t.Fatal(err)
		}
	}
	second, secondView := retryTestRun(t, a, run)
	next, err := db.RecordRetryPacket(ctx, a, second, secondView, 30)
	if err != nil || next.Attempts != 2 || next.Run != second.ID || len(next.RunHistory) != 2 {
		t.Fatalf("attempt not recorded: %+v %v", next, err)
	}
	if _, err = db.FactoryRun(ctx, second.ID); err != nil {
		t.Fatalf("retry run missing: %v", err)
	}
	if _, err = db.RecordRetryPacket(ctx, a, second, secondView, 30); !errors.Is(err, ErrStaleRevision) {
		t.Fatalf("stale retry accepted: %v", err)
	}
	if _, err = db.RecordRetryPacket(ctx, next, second, secondView, 30); err == nil {
		t.Fatal("reused run identity accepted")
	}
	// Recovery settles the superseded run before the next attempt, so it
	// stops counting against capacity.
	settle(run.ID)
	third, thirdView := retryTestRun(t, a, run)
	current, err := db.RecordRetryPacket(ctx, next, third, thirdView, 30)
	if err != nil {
		t.Fatalf("third attempt refused: %v", err)
	}
	fourth, fourthView := retryTestRun(t, a, run)
	if _, err = db.RecordRetryPacket(ctx, current, fourth, fourthView, 30); !errors.Is(err, ErrAssignmentActive) {
		t.Fatalf("fourth attempt accepted: %v", err)
	}
	finished, err := db.Assignment(ctx, a.ID)
	if err != nil {
		t.Fatal(err)
	}
	finished.Stage, finished.Outcome, finished.Reason = factory.AssignmentFinished, factory.Failed, factory.AssignReasonRunFailed
	finished.FinishedUnix = now.Add(time.Hour).Unix()
	result := factory.ResultSynthesized(finished.ID, finished.Run, "failed", "run failed", finished.FinishedUnix)
	finished.Result = &result
	if err = db.FinishAssignment(ctx, finished); err != nil {
		t.Fatalf("finish refused: %v", err)
	}
	if err = db.FinishAssignment(ctx, finished); err == nil {
		t.Fatal("second finish accepted")
	}
	list, err := db.IssueAssignments(ctx, 7, 3)
	if err != nil || len(list) != 1 || list[0].Stage != factory.AssignmentFinished {
		t.Fatalf("issue assignments wrong: %+v %v", list, err)
	}
}

func TestRecordRetryPacketReholdsAndRefusesLimits(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	db := dispatchStoreFixture(t)
	a, r, run, view := dispatchTestPacket(t, now)
	if err := db.RecordDispatchPacket(ctx, dispatchTestRegistration(a), a, r, run, view); err != nil {
		t.Fatal(err)
	}
	if err := db.ReleaseReservation(ctx, a.ID); err != nil {
		t.Fatal(err)
	}
	second, secondView := retryTestRun(t, a, run)
	next, err := db.RecordRetryPacket(ctx, a, second, secondView, 45)
	if err != nil {
		t.Fatalf("retry refused: %v", err)
	}
	held, err := db.Reservation(ctx, a.ID)
	if err != nil || held.State != factory.ReservationHeld || held.PlannedMinutes != 45 {
		t.Fatalf("reservation not reheld: %+v %v", held, err)
	}
	if err := db.SaveCapacity(ctx, factory.Capacity{Revision: 1, UpdatedBy: 7, MaxConcurrentRuns: 1, MaxQueued: 10}); err != nil {
		t.Fatal(err)
	}
	b, r2, run2, view2 := dispatchTestPacket(t, now)
	b.ID, b.Issue, b.Run, b.RunHistory = factory.NewID(), 4, run2.ID, []string{run2.ID}
	r2.AssignmentID = b.ID
	view2.Issue, view2.Attempt = 4, b.ID
	if err := db.RecordDispatchPacket(ctx, dispatchTestRegistration(b), b, r2, run2, view2); !errors.Is(err, ErrCapacityFull) {
		t.Fatalf("over-capacity packet accepted: %v", err)
	}
	if _, err := db.Assignment(ctx, b.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("refused packet left an assignment: %v", err)
	}
	// A refused retry consumes no attempt either: the assignment still
	// retries once room frees.
	third, thirdView := retryTestRun(t, a, run)
	if _, err = db.RecordRetryPacket(ctx, next, third, thirdView, 45); !errors.Is(err, ErrCapacityFull) {
		t.Fatalf("over-capacity retry accepted: %v", err)
	}
	stored, err := db.Assignment(ctx, a.ID)
	if err != nil || stored.Attempts != next.Attempts {
		t.Fatalf("retry attempts wrong: %+v %v", stored, err)
	}
}
