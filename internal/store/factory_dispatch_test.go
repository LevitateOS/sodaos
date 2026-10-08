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

// seedDispatchLimits records current dispatch authority and room for two
// concurrent runs on repository 7 and connection "conn".
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
	projectID := dispatchTestProjectID()
	if err := db.UpsertUser(ctx, User{ID: 7, Login: "soda-tester"}); err != nil {
		t.Fatal(err)
	}
	if err := db.CreateProject(ctx, Project{ID: projectID, Name: "factory", RepositoryID: 7, OwnerID: 7, Repository: "soda/factory"}); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveEnvironmentGrant(ctx, project.EnvironmentGrant{
		Project: projectID, Repository: 7, Owner: 7, Active: true,
		Profile: &project.Profile{
			ID: project.RockyHeadless, Distribution: "rocky", Version: "9.6",
			Interface: "headless", Architecture: "amd64",
			Image: "sha256:" + strings.Repeat("b", 64), Revision: strings.Repeat("c", 40),
		},
	}); err != nil {
		t.Fatal(err)
	}
	requirementID := dispatchTestRequirementID()
	if err := db.AdmitRequirementDecision(ctx, project.RequirementDecision{
		ID: requirementID, Project: projectID, Approver: 7,
		SourceCommit: strings.Repeat("1", 40), SetupDigest: strings.Repeat("a", 64), InputsDigest: strings.Repeat("b", 64),
	}); err != nil {
		t.Fatal(err)
	}
	if err := db.AdmitApprovalDecision(ctx, project.ApprovalDecision{
		ID: dispatchTestApprovalID(), Project: projectID, Requirement: requirementID,
		Approver: 7, EffectsDigest: strings.Repeat("a", 64), ReadinessDigest: strings.Repeat("b", 64), Verified: true,
	}); err != nil {
		t.Fatal(err)
	}
}

func dispatchTestProjectID() string     { return "p" + strings.Repeat("d", 24) }
func dispatchTestRequirementID() string { return "d" + strings.Repeat("c", 24) }
func dispatchTestApprovalID() string    { return "d" + strings.Repeat("e", 24) }

func dispatchTestAuthority() factory.AuthorityRef {
	return factory.AuthorityRef{
		Policy: 1, Operator: 1, Capacity: 1, Sponsorship: 1, Environment: 1,
		RequirementsID: dispatchTestRequirementID(), ApprovalID: dispatchTestApprovalID(),
	}
}

func assertDispatchPacketAbsent(t *testing.T, db *Store, a factory.Assignment) {
	t.Helper()
	for _, check := range []struct{ table, key, value string }{
		{"factory_dispatch_regs", "id", a.ID},
		{"factory_assignments", "id", a.ID},
		{"factory_reservations", "assignment", a.ID},
		{"factory_runs", "id", a.Run},
		{"factory_run_views", "run", a.Run},
	} {
		var count int
		query := "SELECT count(*) FROM " + check.table + " WHERE " + check.key + "=$1"
		if err := db.db.QueryRowContext(context.Background(), query, check.value).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Fatalf("dispatch packet left %d rows in %s", count, check.table)
		}
	}
}

func dispatchCurrentAuthority(t *testing.T, db *Store, a factory.Assignment) factory.AuthorityRef {
	t.Helper()
	policy, err := db.RepositoryPolicy(context.Background(), a.Repository)
	if err != nil {
		t.Fatal(err)
	}
	operator, err := db.OperatorGrant(context.Background(), a.Repository)
	if err != nil {
		t.Fatal(err)
	}
	capacity, err := db.Capacity(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	sponsorship, err := db.Sponsorship(context.Background(), a.Repository, a.Connection)
	if err != nil {
		t.Fatal(err)
	}
	environment, err := db.EnvironmentGrant(context.Background(), a.Repository)
	if err != nil {
		t.Fatal(err)
	}
	requirement, err := db.RequirementHead(context.Background(), a.ProjectID)
	if err != nil {
		t.Fatal(err)
	}
	approval, err := db.ApprovalHead(context.Background(), a.ProjectID)
	if err != nil {
		t.Fatal(err)
	}
	return factory.AuthorityRef{
		Policy: policy.Revision, Operator: operator.Revision, Capacity: capacity.Revision,
		Sponsorship: sponsorship.Revision, Environment: environment.Revision,
		RequirementsID: requirement, ApprovalID: approval,
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
		ID: factory.NewID(), ProjectID: dispatchTestProjectID(), Role: project.RoleCoder,
		Repository: 7, Issue: 3, Revision: 0, NativeRev: 5,
		Acceptance: "d" + strings.Repeat("a", 24), Preparation: "f" + strings.Repeat("b", 24),
		Harness: "codex-1.2.3", HarnessVers: "1.2.3", Model: "m",
		Connection: "conn", SourceCommit: strings.Repeat("c", 40),
		Prompt: prompt, PromptSHA: hex.EncodeToString(sum[:]),
		Run: runID, RunHistory: []string{runID}, Stage: factory.AssignmentAssigned, Attempts: 1,
		CreatedUnix: now.Unix(), Authority: dispatchTestAuthority(),
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
		a.Authority.Capacity = 2
		if err := db.RecordDispatchPacket(ctx, dispatchTestRegistration(a), a, r, run, view); err != nil {
			t.Fatal(err)
		}
		b, r2, run2, view2 := dispatchTestPacket(t, now)
		b.ID, b.Issue, b.Run, b.RunHistory = factory.NewID(), 4, run2.ID, []string{run2.ID}
		b.Authority.Capacity = 2
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
		a.Authority.Sponsorship = 2
		if err := db.RecordDispatchPacket(ctx, dispatchTestRegistration(a), a, r, run, view); err != nil {
			t.Fatal(err)
		}
		b, r2, run2, view2 := dispatchTestPacket(t, now)
		b.ID, b.Issue, b.Run, b.RunHistory = factory.NewID(), 4, run2.ID, []string{run2.ID}
		b.Authority.Sponsorship = 2
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
		assertDispatchPacketAbsent(t, db, a)
	})
}
