package store

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
)

func dispatchStoreFixture(t *testing.T) *Store {
	t.Helper()
	db := grantStoreFixture(t)
	seedDispatchLimits(t, db)
	return db
}

// seedDispatchLimits records the capacity, policy, operator grant and
// sponsorship the atomic admission gate requires: repository 7 on
// connection "conn" with room for two concurrent runs.
func seedDispatchLimits(t *testing.T, db *Store) {
	t.Helper()
	ctx := context.Background()
	policy := grantTestPolicy()
	policy.Repository = 7
	for _, err := range []error{
		db.SaveCapacity(ctx, factory.Capacity{UpdatedBy: 7, MaxConcurrentRuns: 2, MaxQueued: 10}),
		db.SaveRepositoryPolicy(ctx, policy),
		db.SaveOperatorGrant(ctx, factory.OperatorGrant{Repository: 7, GrantedBy: 7, MaxConcurrent: 2, Active: true}),
		db.SaveSponsorship(ctx, factory.Sponsorship{Repository: 7, GrantedBy: 7, Generation: 1, Connection: "conn", GrantID: "grant", Roles: []string{project.RoleCoder}, AllowanceMinutes: 120, MaxConcurrent: 2, Active: true}),
	} {
		if err != nil {
			t.Fatal(err)
		}
	}
}

func dispatchTestRegistration(a factory.Assignment) factory.DispatchRegistration {
	return factory.DispatchRegistration{ID: a.ID, Repository: a.Repository, Authority: a.Authority}
}

func dispatchTestPrompt(t *testing.T) []byte {
	t.Helper()
	prompt, err := factory.BuildDispatchPrompt(factory.PromptInputs{
		Repository: 7, Issue: 3, NativeRev: 5,
		AcceptanceID: "d" + strings.Repeat("a", 24),
		TargetBranch: "refs/heads/main", SourceCommit: strings.Repeat("c", 40),
		Preparation: "f" + strings.Repeat("b", 24),
		Harness:     "codex", Model: "m", Role: project.RoleCoder,
		Title: "objective",
	})
	if err != nil {
		t.Fatal(err)
	}
	return prompt
}

func dispatchTestPacket(t *testing.T, now time.Time) (factory.Assignment, factory.Reservation, factory.Run, factory.RunView) {
	t.Helper()
	prompt := dispatchTestPrompt(t)
	sum := sha256.Sum256(prompt)
	runID := factory.NewID()
	a := factory.Assignment{
		ID: factory.NewID(), ProjectID: "p" + strings.Repeat("d", 24), Role: project.RoleCoder,
		Repository: 7, Issue: 3, Revision: 0, NativeRev: 5,
		Acceptance: "d" + strings.Repeat("a", 24), Preparation: "f" + strings.Repeat("b", 24),
		Harness: "codex-1.2.3", HarnessVers: "1.2.3", Model: "m",
		Connection: "conn", SourceCommit: strings.Repeat("c", 40),
		Prompt: prompt, PromptSHA: hex.EncodeToString(sum[:]),
		Run: runID, RunHistory: []string{runID}, Stage: factory.AssignmentAssigned, Attempts: 1,
		CreatedUnix: now.Unix(),
	}
	r := factory.Reservation{AssignmentID: a.ID, Repository: 7, Connection: "conn", State: factory.ReservationHeld, PlannedMinutes: 30}
	run := factory.Run{
		ID: runID, ProjectID: a.ProjectID, Role: project.RoleCoder, InputSHA: strings.Repeat("c", 40),
		Started: now, Deadline: now.Add(30 * time.Minute),
		Image: "sha256:" + strings.Repeat("b", 64), Harness: "codex-1.2.3", Model: "m",
	}
	view := factory.RunView{RunID: runID, Repository: 7, Issue: 3, Attempt: a.ID}
	return a, r, run, view
}

func TestRecordDispatchPacket(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	db := dispatchStoreFixture(t)
	a, r, run, view := dispatchTestPacket(t, now)
	if err := db.RecordDispatchPacket(ctx, dispatchTestRegistration(a), a, r, run, view); err != nil {
		t.Fatalf("packet refused: %v", err)
	}
	got, err := db.Assignment(ctx, a.ID)
	if err != nil || got.Run != run.ID || got.Stage != factory.AssignmentAssigned {
		t.Fatalf("assignment unreadable: %+v %v", got, err)
	}
	byRun, err := db.AssignmentByRun(ctx, run.ID)
	if err != nil || byRun.ID != a.ID {
		t.Fatalf("assignment by run unreadable: %+v %v", byRun, err)
	}
	stored, err := db.FactoryRun(ctx, run.ID)
	if err != nil || stored.ProjectID != a.ProjectID {
		t.Fatalf("packet run unreadable: %+v %v", stored, err)
	}
	held, err := db.HeldReservations(ctx, 10)
	if err != nil || len(held) != 1 || held[0].AssignmentID != a.ID {
		t.Fatalf("reservation not held: %+v %v", held, err)
	}
	again, r2, run2, view2 := dispatchTestPacket(t, now)
	again.ID, again.Run, again.RunHistory = factory.NewID(), run2.ID, []string{run2.ID}
	r2.AssignmentID = again.ID
	view2.Attempt = again.ID
	if err := db.RecordDispatchPacket(ctx, dispatchTestRegistration(again), again, r2, run2, view2); err == nil {
		t.Fatal("second unfinished assignment for the issue accepted")
	} else if !isAssignmentActive(err) {
		t.Fatalf("wrong conflict error: %v", err)
	}
}

func isAssignmentActive(err error) bool {
	return err != nil && (err == ErrAssignmentActive || strings.Contains(err.Error(), "unfinished assignment"))
}

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

func TestRecordDispatchPacketEnforcesLimits(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	t.Run("capacity", func(t *testing.T) {
		db := grantStoreFixture(t)
		seedDispatchLimits(t, db)
		if err := db.SaveCapacity(ctx, factory.Capacity{Revision: 1, UpdatedBy: 7, MaxConcurrentRuns: 1, MaxQueued: 10}); err != nil {
			t.Fatal(err)
		}
		a, r, run, view := dispatchTestPacket(t, now)
		if err := db.RecordDispatchPacket(ctx, dispatchTestRegistration(a), a, r, run, view); err != nil {
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
	})
	t.Run("allowance", func(t *testing.T) {
		db := grantStoreFixture(t)
		seedDispatchLimits(t, db)
		sponsorship, err := db.Sponsorship(ctx, 7, "conn")
		if err != nil {
			t.Fatal(err)
		}
		sponsorship.AllowanceMinutes = 30
		if err := db.SaveSponsorship(ctx, sponsorship); err != nil {
			t.Fatal(err)
		}
		a, r, run, view := dispatchTestPacket(t, now)
		if err := db.RecordDispatchPacket(ctx, dispatchTestRegistration(a), a, r, run, view); err != nil {
			t.Fatal(err)
		}
		b, r2, run2, view2 := dispatchTestPacket(t, now)
		b.ID, b.Issue, b.Run, b.RunHistory = factory.NewID(), 4, run2.ID, []string{run2.ID}
		r2.AssignmentID = b.ID
		view2.Issue, view2.Attempt = 4, b.ID
		if err := db.RecordDispatchPacket(ctx, dispatchTestRegistration(b), b, r2, run2, view2); !errors.Is(err, ErrAllowanceExhausted) {
			t.Fatalf("over-budget packet accepted: %v", err)
		}
	})
	t.Run("missing grants", func(t *testing.T) {
		db := grantStoreFixture(t)
		a, r, run, view := dispatchTestPacket(t, now)
		if err := db.RecordDispatchPacket(ctx, dispatchTestRegistration(a), a, r, run, view); !errors.Is(err, ErrAdmissionChanged) {
			t.Fatalf("unlimited packet accepted: %v", err)
		}
	})
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

func TestReservationTransitions(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	db := dispatchStoreFixture(t)
	a, r, run, view := dispatchTestPacket(t, now)
	if err := db.RecordDispatchPacket(ctx, dispatchTestRegistration(a), a, r, run, view); err != nil {
		t.Fatal(err)
	}
	if err := db.ConsumeReservation(ctx, a.ID); err != nil {
		t.Fatalf("consume refused: %v", err)
	}
	if err := db.ConsumeReservation(ctx, a.ID); err == nil {
		t.Fatal("double consume accepted")
	}
	if err := db.ReleaseReservation(ctx, a.ID); err == nil {
		t.Fatal("release after consume accepted")
	}
	held, err := db.HeldReservations(ctx, 10)
	if err != nil || len(held) != 0 {
		t.Fatalf("consumed reservation still held: %+v %v", held, err)
	}

	b, r2, run2, view2 := dispatchTestPacket(t, now)
	b.ID, b.Issue, b.Run, b.RunHistory = factory.NewID(), 4, run2.ID, []string{run2.ID}
	r2.AssignmentID = b.ID
	view2.Issue, view2.Attempt = 4, b.ID
	if err := db.RecordDispatchPacket(ctx, dispatchTestRegistration(b), b, r2, run2, view2); err != nil {
		t.Fatal(err)
	}
	if err := db.ReleaseReservation(ctx, b.ID); err != nil {
		t.Fatalf("release refused: %v", err)
	}
	rehold := factory.Reservation{AssignmentID: b.ID, Repository: 7, Connection: "conn", State: factory.ReservationHeld, PlannedMinutes: 12, Revision: 2}
	if err := db.ReholdReservation(ctx, rehold); err != nil {
		t.Fatalf("rehold refused: %v", err)
	}
	if err := db.ReholdReservation(ctx, rehold); err == nil {
		t.Fatal("rehold while held accepted")
	}
	got, err := db.Reservation(ctx, b.ID)
	if err != nil || got.State != factory.ReservationHeld || got.PlannedMinutes != 12 {
		t.Fatalf("reheld reservation wrong: %+v %v", got, err)
	}
}

func TestRunUsageFirstWriteWins(t *testing.T) {
	ctx := context.Background()
	db := dispatchStoreFixture(t)
	u := factory.Usage{RunID: factory.NewID(), Repository: 7, Connection: "conn", Minutes: 4, RecordedUnix: 1700000000}
	if err := db.RecordRunUsage(ctx, u); err != nil {
		t.Fatal(err)
	}
	dup := u
	dup.Minutes = 9
	if err := db.RecordRunUsage(ctx, dup); err != nil {
		t.Fatalf("usage replay refused: %v", err)
	}
	other := factory.Usage{RunID: factory.NewID(), Repository: 7, Connection: "conn", Minutes: 3, RecordedUnix: 1700000001}
	if err := db.RecordRunUsage(ctx, other); err != nil {
		t.Fatal(err)
	}
	got, err := db.RunUsage(ctx, u.RunID)
	if err != nil || got.Minutes != 4 {
		t.Fatalf("run usage = %+v %v", got, err)
	}
	total, err := db.UsageTotal(ctx, 7, "conn")
	if err != nil || total != 7 {
		t.Fatalf("usage total = %d, %v", total, err)
	}
	empty, err := db.UsageTotal(ctx, 7, "other")
	if err != nil || empty != 0 {
		t.Fatalf("missing usage total = %d, %v", empty, err)
	}
}

func queuedTestControl(repository, issue int64, readiness, reason string, firstSeen int64) factory.IssueControl {
	return factory.IssueControl{
		Repository: repository, Issue: issue,
		Readiness: readiness, Reason: reason,
		Acceptance:  "d" + strings.Repeat("a", 24),
		Fingerprint: strings.Repeat("1", 64), Authority: strings.Repeat("2", 64),
		FirstSeenUnix: firstSeen, AssessedUnix: firstSeen,
		Blockers: func() []factory.Blocker {
			if readiness == factory.ReadinessQueued {
				return nil
			}
			return []factory.Blocker{{Code: factory.BlockerAcceptanceMissing, Resolution: factory.BlockerResolution(factory.BlockerAcceptanceMissing)}}
		}(),
	}
}

func TestQueuedControlsOldestFirst(t *testing.T) {
	ctx := context.Background()
	db := dispatchStoreFixture(t)
	now := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	young := queuedTestControl(7, 10, factory.ReadinessQueued, factory.ReasonEligible, now.Add(-time.Hour).Unix())
	old := queuedTestControl(8, 2, factory.ReadinessQueued, factory.ReasonEligible, now.Add(-2*time.Hour).Unix())
	tied := queuedTestControl(7, 3, factory.ReadinessQueued, factory.ReasonEligible, now.Add(-2*time.Hour).Unix())
	blocked := queuedTestControl(7, 4, factory.ReadinessBlocked, factory.BlockerAcceptanceMissing, now.Add(-3*time.Hour).Unix())
	for _, c := range []factory.IssueControl{young, old, tied, blocked} {
		if _, _, err := db.RecordIssueAssessment(ctx, c, now); err != nil {
			t.Fatal(err)
		}
	}
	// FirstSeenUnix is assigned by the store clock; set distinct stamps
	// directly so the test proves oldest-first visiting with a
	// deterministic repository/issue tie-break.
	for _, stamp := range []struct {
		repository, issue, seen int64
	}{
		{7, 10, now.Add(-time.Hour).Unix()},
		{8, 2, now.Add(-2 * time.Hour).Unix()},
		{7, 3, now.Add(-2 * time.Hour).Unix()},
	} {
		if _, err := db.exec(ctx, `UPDATE issue_controls
			SET data=jsonb_set(data,'{first_seen_unix}',to_jsonb(?::bigint)) WHERE repository=? AND issue=?`,
			stamp.seen, stamp.repository, stamp.issue); err != nil {
			t.Fatal(err)
		}
	}
	got, err := db.QueuedControls(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	want := [][2]int64{{7, 3}, {8, 2}, {7, 10}}
	if len(got) != len(want) {
		t.Fatalf("queued count = %d", len(got))
	}
	for i, w := range want {
		if got[i].Repository != w[0] || got[i].Issue != w[1] {
			t.Fatalf("queued[%d] = %d/%d, want %d/%d", i, got[i].Repository, got[i].Issue, w[0], w[1])
		}
	}
}

func TestActiveRunCountsSkipAttributed(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	db := dispatchStoreFixture(t)
	a, r, run, view := dispatchTestPacket(t, now)
	if err := db.RecordDispatchPacket(ctx, dispatchTestRegistration(a), a, r, run, view); err != nil {
		t.Fatal(err)
	}
	human := factory.Run{
		ID: factory.NewID(), ProjectID: a.ProjectID, Role: project.RoleCoder, InputSHA: strings.Repeat("c", 40),
		Started: now, Deadline: now.Add(time.Hour),
		Image: "sha256:" + strings.Repeat("b", 64), Harness: "codex-1.2.3", Model: "m",
	}
	if err := db.RecordFactoryRun(ctx, human); err != nil {
		t.Fatal(err)
	}
	total, byProject, err := db.ActiveRunCounts(ctx)
	if err != nil || total != 1 || byProject[a.ProjectID] != 1 {
		t.Fatalf("counts = %d %+v, %v", total, byProject, err)
	}
}
