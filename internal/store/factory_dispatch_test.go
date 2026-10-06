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
