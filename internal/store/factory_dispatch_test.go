package store

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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

func dispatchTestControlFor(t *testing.T, db *Store, a factory.Assignment) factory.IssueControl {
	t.Helper()
	control := factory.IssueControl{
		Repository: a.Repository, Issue: a.Issue, Acceptance: a.Acceptance, NativeRev: a.NativeRev,
		Readiness: factory.ReadinessQueued, Reason: factory.ReasonEligible,
		Fingerprint: strings.Repeat("1", 64), Authority: strings.Repeat("2", 64),
	}
	stored, _, err := db.RecordIssueAssessment(context.Background(), control, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	return stored
}

func recordDispatchTestPacket(t *testing.T, ctx context.Context, db *Store, d factory.DispatchRegistration, a factory.Assignment, r factory.Reservation, run factory.Run, view factory.RunView) error {
	t.Helper()
	return db.RecordDispatchPacket(ctx, d, dispatchTestControlFor(t, db, a), a, r, run, view)
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
	if err := db.SaveConnectionUsageBudget(ctx, factory.ConnectionUsageBudget{
		Connection: "conn", RollingMinutes: factory.DefaultConnectionUsageBudgetMinutes,
	}); err != nil {
		t.Fatal(err)
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
		SponsorshipConnection: "conn", ConnectionUsageBudget: 1,
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
	budget, err := db.ConnectionUsageBudget(context.Background(), a.Connection)
	if err != nil {
		t.Fatal(err)
	}
	return factory.AuthorityRef{
		Policy: policy.Revision, Operator: operator.Revision, Capacity: capacity.Revision,
		Sponsorship: sponsorship.Revision, SponsorshipConnection: sponsorship.Connection,
		ConnectionUsageBudget: budget.Revision, Environment: environment.Revision,
		RequirementsID: requirement, ApprovalID: approval,
	}
}

func dispatchTestRegistration(a factory.Assignment) factory.DispatchRegistration {
	return factory.DispatchRegistration{ID: a.ID, Repository: a.Repository, Authority: a.Authority}
}

func dispatchTestPrompt(t *testing.T) []byte {
	t.Helper()
	selection := grantTestPolicy().Roles[project.RoleCoder]
	prompt, err := factory.BuildDispatchPrompt(factory.PromptInputs{
		Repository: 7, Issue: 3, NativeRev: 5,
		AcceptanceID: "d" + strings.Repeat("a", 24),
		TargetBranch: "refs/heads/main", SourceCommit: strings.Repeat("c", 40),
		Preparation: "f" + strings.Repeat("b", 24),
		Harness:     selection.Harness, Model: selection.Model, Role: project.RoleCoder,
		ProviderConnection: "conn", RequiredChecks: []string{"ci"},
		ApplianceConcurrent: 2, RepositoryConcurrent: 2, SponsorshipConcurrent: 2, AttemptLimits: factory.DefaultAttemptLimits(),
		RequirementsID: "d" + strings.Repeat("e", 24), ApprovalID: "d" + strings.Repeat("f", 24),
		Title: "objective",
	})
	if err != nil {
		t.Fatal(err)
	}
	return prompt
}

func dispatchTestPacket(t *testing.T, now time.Time) (factory.Assignment, factory.Reservation, factory.Run, factory.RunView) {
	t.Helper()
	selection := grantTestPolicy().Roles[project.RoleCoder]
	prompt := dispatchTestPrompt(t)
	sum := sha256.Sum256(prompt)
	assignmentID, runID := factory.NewID(), factory.NewID()
	a := factory.Assignment{
		ID: assignmentID, AttemptRoot: assignmentID, PublicationAssignment: assignmentID,
		ProjectID: dispatchTestProjectID(), Role: project.RoleCoder,
		Repository: 7, Issue: 3, Revision: 0, NativeRev: 5,
		Acceptance: "d" + strings.Repeat("a", 24), Preparation: "f" + strings.Repeat("b", 24),
		Harness: selection.Harness + "-" + selection.HarnessVers, HarnessVers: selection.HarnessVers, Model: selection.Model,
		Connection: "conn", SourceCommit: strings.Repeat("c", 40),
		Prompt: prompt, PromptSHA: hex.EncodeToString(sum[:]),
		Run: runID, RunHistory: []string{runID}, Stage: factory.AssignmentAssigned, Attempts: 1,
		CreatedUnix: now.Unix(), Authority: dispatchTestAuthority(),
	}
	r := factory.Reservation{AssignmentID: a.ID, Repository: 7, Connection: "conn", State: factory.ReservationHeld, PlannedMinutes: 30}
	run := factory.Run{
		ID: runID, ProjectID: a.ProjectID, Role: project.RoleCoder, InputSHA: strings.Repeat("c", 40),
		Started: now, Deadline: now.Add(30 * time.Minute),
		Image: "sha256:" + strings.Repeat("b", 64), Harness: selection.Harness + "-" + selection.HarnessVers, Model: selection.Model,
	}
	view := factory.RunView{RunID: runID, Repository: 7, Issue: 3, Attempt: a.ID}
	return a, r, run, view
}

func TestRecordDispatchPacket(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC()
	db := dispatchStoreFixture(t)
	a, r, run, view := dispatchTestPacket(t, now)
	if err := recordDispatchTestPacket(t, ctx, db, dispatchTestRegistration(a), a, r, run, view); err != nil {
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
	again.AttemptRoot, again.PublicationAssignment = again.ID, again.ID
	r2.AssignmentID = again.ID
	view2.Attempt = again.ID
	if err := recordDispatchTestPacket(t, ctx, db, dispatchTestRegistration(again), again, r2, run2, view2); err == nil {
		t.Fatal("second unfinished assignment for the issue accepted")
	} else if !isAssignmentActive(err) {
		t.Fatalf("wrong conflict error: %v", err)
	}
}

func TestRecordDispatchPacketEnforcesOneRepositorySession(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		name             string
		releasePriorSlot bool
	}{
		{name: "held reservation"},
		{name: "active attributed run after reservation release", releasePriorSlot: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := dispatchStoreFixture(t)
			now := time.Now().UTC()
			first, reservation, run, view := dispatchTestPacket(t, now)
			if err := recordDispatchTestPacket(t, ctx, db, dispatchTestRegistration(first), first, reservation, run, view); err != nil {
				t.Fatal(err)
			}
			if tc.releasePriorSlot {
				if err := db.ReleaseReservation(ctx, first.ID); err != nil {
					t.Fatal(err)
				}
			}
			second, nextReservation, nextRun, nextView := dispatchTestPacket(t, now)
			second.ID, second.Issue, second.Run, second.RunHistory = factory.NewID(), 4, nextRun.ID, []string{nextRun.ID}
			second.AttemptRoot, second.PublicationAssignment = second.ID, second.ID
			nextReservation.AssignmentID = second.ID
			nextView.Issue, nextView.Attempt = second.Issue, second.ID
			if err := recordDispatchTestPacket(t, ctx, db, dispatchTestRegistration(second), second, nextReservation, nextRun, nextView); !errors.Is(err, ErrRepositoryFull) {
				t.Fatalf("second same-repository session error = %v, want %v", err, ErrRepositoryFull)
			}
			assertDispatchPacketAbsent(t, db, second)
		})
	}
}

func TestRecordDispatchPacketRejectsChangedQueuedControlAtomically(t *testing.T) {
	ctx := context.Background()
	db := dispatchStoreFixture(t)
	a, r, run, view := dispatchTestPacket(t, time.Now())
	expected := dispatchTestControlFor(t, db, a)
	changed := expected
	changed.NativeRev++
	changed.Fingerprint = strings.Repeat("3", 64)
	if _, _, err := db.RecordIssueAssessment(ctx, changed, time.Now()); err != nil {
		t.Fatal(err)
	}
	err := db.RecordDispatchPacket(ctx, dispatchTestRegistration(a), expected, a, r, run, view)
	if !errors.Is(err, ErrDispatchControlStale) {
		t.Fatalf("stale queued control error = %v", err)
	}
	assertDispatchPacketAbsent(t, db, a)
}

func isAssignmentActive(err error) bool {
	return err != nil && (err == ErrAssignmentActive || strings.Contains(err.Error(), "unfinished assignment"))
}

func TestRecordDispatchPacketEnforcesLimits(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC()
	t.Run("capacity", func(t *testing.T) {
		db := grantStoreFixture(t)
		seedDispatchLimits(t, db)
		if err := db.SaveCapacity(ctx, factory.Capacity{Revision: 1, UpdatedBy: 7, MaxConcurrentRuns: 1, MaxQueued: 10}); err != nil {
			t.Fatal(err)
		}
		a, r, run, view := dispatchTestPacket(t, now)
		a.Authority.Capacity = 2
		if err := recordDispatchTestPacket(t, ctx, db, dispatchTestRegistration(a), a, r, run, view); err != nil {
			t.Fatal(err)
		}
		b, r2, run2, view2 := dispatchTestPacket(t, now)
		b.ID, b.Issue, b.Run, b.RunHistory = factory.NewID(), 4, run2.ID, []string{run2.ID}
		b.AttemptRoot, b.PublicationAssignment = b.ID, b.ID
		b.Authority.Capacity = 2
		r2.AssignmentID = b.ID
		view2.Issue, view2.Attempt = 4, b.ID
		if err := recordDispatchTestPacket(t, ctx, db, dispatchTestRegistration(b), b, r2, run2, view2); !errors.Is(err, ErrCapacityFull) {
			t.Fatalf("over-capacity packet accepted: %v", err)
		}
		if _, err := db.Assignment(ctx, b.ID); !errors.Is(err, ErrNotFound) {
			t.Fatalf("refused packet left an assignment: %v", err)
		}
		settled := run
		settled.Outcome, settled.Reconciled, settled.Summary = factory.Succeeded, true, "completed"
		if err := db.SaveFactoryRun(ctx, settled); err != nil {
			t.Fatal(err)
		}
		finished, err := db.Assignment(ctx, a.ID)
		if err != nil {
			t.Fatal(err)
		}
		result := factory.Result{
			Status: "completed", Summary: "completed", Candidate: a.SourceCommit, Findings: []string{},
		}
		finished.Stage, finished.Outcome, finished.Reason = factory.AssignmentFinished, factory.Succeeded, factory.AssignReasonReported
		bound := factory.ResultFromHarness(a.ID, run.ID, result, now.Unix())
		finished.Result = &bound
		finished.FinishedUnix = now.Unix()
		if err := db.FinishAssignment(ctx, finished); err != nil {
			t.Fatal(err)
		}
		if err := db.ConsumeReservation(ctx, a.ID); err != nil {
			t.Fatal(err)
		}
		allowance, err := db.AttemptAllowance(ctx, a.Repository, a.Issue)
		if err != nil || !allowance.Active || allowance.Closed {
			t.Fatalf("settled reported attempt allowance = %+v, %v", allowance, err)
		}
		if err := recordDispatchTestPacket(t, ctx, db, dispatchTestRegistration(b), b, r2, run2, view2); !errors.Is(err, ErrCapacityFull) {
			t.Fatalf("packet bypassed appliance limit after session settled: %v", err)
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
		if err := recordDispatchTestPacket(t, ctx, db, dispatchTestRegistration(a), a, r, run, view); err != nil {
			t.Fatal(err)
		}
		settled, err := db.FactoryRun(ctx, run.ID)
		if err != nil {
			t.Fatal(err)
		}
		settled.Outcome, settled.Reconciled, settled.Summary = factory.Failed, true, "completed"
		if err := db.SaveFactoryRun(ctx, settled); err != nil {
			t.Fatal(err)
		}
		if err := db.ConsumeReservation(ctx, a.ID); err != nil {
			t.Fatal(err)
		}
		if err := db.RecordRunUsage(ctx, factory.Usage{
			RunID: run.ID, Repository: a.Repository, Connection: a.Connection,
			Minutes: 30, StartedAt: now.Add(-30 * time.Minute), EndedAt: now,
		}); err != nil {
			t.Fatal(err)
		}
		finished, err := db.Assignment(ctx, a.ID)
		if err != nil {
			t.Fatal(err)
		}
		finished.Stage, finished.Outcome, finished.Reason = factory.AssignmentFinished, factory.Failed, factory.AssignReasonRunFailed
		bound := factory.ResultSynthesized(a.ID, run.ID, "failed", "completed", now.Unix())
		finished.Result = &bound
		finished.FinishedUnix = now.Unix()
		if err := db.FinishAssignment(ctx, finished); err != nil {
			t.Fatal(err)
		}
		b, r2, run2, view2 := dispatchTestPacket(t, now)
		b.ID, b.Issue, b.Run, b.RunHistory = factory.NewID(), 4, run2.ID, []string{run2.ID}
		b.AttemptRoot, b.PublicationAssignment = b.ID, b.ID
		b.Authority.Sponsorship = 2
		r2.AssignmentID = b.ID
		view2.Issue, view2.Attempt = 4, b.ID
		if err := recordDispatchTestPacket(t, ctx, db, dispatchTestRegistration(b), b, r2, run2, view2); !errors.Is(err, ErrAllowanceExhausted) {
			t.Fatalf("over-budget packet accepted: %v", err)
		}
	})
	t.Run("connection rolling window spans repositories", func(t *testing.T) {
		now := time.Now().UTC()
		for _, tc := range []struct {
			name string
			end  time.Time
			want error
		}{
			{"in window", now.Add(-time.Minute), ErrConnectionUsageBudget},
			{"outside window", now.Add(-24*time.Hour - time.Minute), nil},
		} {
			t.Run(tc.name, func(t *testing.T) {
				db := dispatchStoreFixture(t)
				start := tc.end.Add(-360 * time.Minute)
				usage := factory.Usage{RunID: factory.NewID(), Repository: 8, Connection: "conn", Minutes: 360, StartedAt: start, EndedAt: tc.end}
				if err := db.RecordRunUsage(ctx, usage); err != nil {
					t.Fatal(err)
				}
				a, r, run, view := dispatchTestPacket(t, now)
				err := recordDispatchTestPacket(t, ctx, db, dispatchTestRegistration(a), a, r, run, view)
				if !errors.Is(err, tc.want) {
					t.Fatalf("rolling-window packet error = %v, want %v", err, tc.want)
				}
				if tc.want != nil {
					assertDispatchPacketAbsent(t, db, a)
				}
			})
		}
	})
	t.Run("held session keeps growing past its plan", func(t *testing.T) {
		now := time.Now().UTC()
		db := dispatchStoreFixture(t)
		firstAt := now.Add(-4 * time.Hour)
		a, reservation, run, _ := dispatchTestPacket(t, firstAt)
		// This is a different repository sharing the same provider connection;
		// the product allows cross-repository use of available appliance slots.
		a.Repository, reservation.Repository = 8, 8
		a.ProjectID, run.ProjectID = "p888888888888888888888888", "p888888888888888888888888"
		// Seed an existing held process with its original immutable binding.
		// Fresh admission correctly refuses its now-expired deadline.
		if err := db.RecordFactoryRun(ctx, run); err != nil {
			t.Fatal(err)
		}
		assignmentData, err := json.Marshal(a)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := db.db.ExecContext(ctx, `INSERT INTO factory_assignments(id,repository,issue,run,stage,revision,data) VALUES($1,$2,$3,$4,$5,$6,$7)`, a.ID, a.Repository, a.Issue, a.Run, a.Stage, a.Revision, string(assignmentData)); err != nil {
			t.Fatal(err)
		}
		reservationData, err := json.Marshal(reservation)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := db.db.ExecContext(ctx, `INSERT INTO factory_reservations(assignment,repository,connection,state,data) VALUES($1,$2,$3,$4,$5)`, a.ID, a.Repository, a.Connection, reservation.State, string(reservationData)); err != nil {
			t.Fatal(err)
		}
		budget, err := db.ConnectionUsageBudget(ctx, "conn")
		if err != nil {
			t.Fatal(err)
		}
		budget.RollingMinutes = 200
		if err := db.SaveConnectionUsageBudget(ctx, budget); err != nil {
			t.Fatal(err)
		}
		second, r2, run2, view2 := dispatchTestPacket(t, now)
		second.ID, second.Issue, second.Run, second.RunHistory = factory.NewID(), 4, run2.ID, []string{run2.ID}
		second.AttemptRoot, second.PublicationAssignment = second.ID, second.ID
		second.Authority.ConnectionUsageBudget = budget.Revision + 1
		r2.AssignmentID = second.ID
		view2.Issue, view2.Attempt = 4, second.ID
		if err := recordDispatchTestPacket(t, ctx, db, dispatchTestRegistration(second), second, r2, run2, view2); !errors.Is(err, ErrConnectionUsageBudget) {
			t.Fatalf("over-plan held usage was not charged: %v", err)
		}
		assertDispatchPacketAbsent(t, db, second)
	})
	t.Run("held session with missing run start fails closed", func(t *testing.T) {
		now := time.Now().UTC()
		db := dispatchStoreFixture(t)
		a, reservation, run, view := dispatchTestPacket(t, now)
		if err := recordDispatchTestPacket(t, ctx, db, dispatchTestRegistration(a), a, reservation, run, view); err != nil {
			t.Fatal(err)
		}
		missingRun := factory.NewID()
		if _, err := db.db.ExecContext(ctx, `UPDATE factory_assignments SET run=$2,
			data=jsonb_set(data,'{run}',to_jsonb($2::text)) WHERE id=$1`, a.ID, missingRun); err != nil {
			t.Fatal(err)
		}
		second, r2, run2, view2 := dispatchTestPacket(t, now)
		second.ID, second.Issue, second.Run, second.RunHistory = factory.NewID(), 4, run2.ID, []string{run2.ID}
		second.AttemptRoot, second.PublicationAssignment = second.ID, second.ID
		r2.AssignmentID = second.ID
		view2.Issue, view2.Attempt = 4, second.ID
		if err := recordDispatchTestPacket(t, ctx, db, dispatchTestRegistration(second), second, r2, run2, view2); err == nil {
			t.Fatal("held session without a run start was ignored")
		}
		assertDispatchPacketAbsent(t, db, second)
	})
	t.Run("missing grants", func(t *testing.T) {
		db := grantStoreFixture(t)
		a, r, run, view := dispatchTestPacket(t, now)
		if err := recordDispatchTestPacket(t, ctx, db, dispatchTestRegistration(a), a, r, run, view); !errors.Is(err, ErrAdmissionChanged) {
			t.Fatalf("unlimited packet accepted: %v", err)
		}
		assertDispatchPacketAbsent(t, db, a)
	})
}

func TestRecordRunUsageStoresOutwardMicrosecondProjections(t *testing.T) {
	ctx := context.Background()
	db := grantStoreFixture(t)
	started := time.Now().UTC().Truncate(time.Second).Add(123456789 * time.Nanosecond)
	ended := started.Add(121*time.Second + time.Nanosecond)
	usage := factory.Usage{
		RunID: factory.NewID(), Repository: 7, Connection: "conn", Minutes: 3,
		StartedAt: started, EndedAt: ended,
	}
	if err := db.RecordRunUsage(ctx, usage); err != nil {
		t.Fatal(err)
	}
	var storedStart, storedEnd time.Time
	if err := db.db.QueryRowContext(ctx, `SELECT started_at,ended_at FROM factory_usage WHERE run=$1`, usage.RunID).Scan(&storedStart, &storedEnd); err != nil {
		t.Fatal(err)
	}
	if !storedStart.Equal(floorUsageMicro(started)) || !storedEnd.Equal(ceilUsageMicro(ended)) {
		t.Fatalf("SQL interval projections = %s..%s, want %s..%s", storedStart, storedEnd, floorUsageMicro(started), ceilUsageMicro(ended))
	}
	canonical, err := db.RunUsage(ctx, usage.RunID)
	if err != nil || !canonical.StartedAt.Equal(started) || !canonical.EndedAt.Equal(ended) {
		t.Fatalf("canonical interval lost precision: %+v %v", canonical, err)
	}
}

func TestConnectionUsageBudgetAdmitsExactPlannedMinutes(t *testing.T) {
	ctx := context.Background()
	db := dispatchStoreFixture(t)
	budget, err := db.ConnectionUsageBudget(ctx, "conn")
	if err != nil {
		t.Fatal(err)
	}
	budget.RollingMinutes = 30
	if err = db.SaveConnectionUsageBudget(ctx, budget); err != nil {
		t.Fatal(err)
	}
	started := time.Now().UTC().Truncate(time.Second).Add(-time.Second + time.Nanosecond)
	assignment, reservation, run, view := dispatchTestPacket(t, started)
	assignment.Authority.ConnectionUsageBudget = budget.Revision + 1
	if reservation.PlannedMinutes != 30 {
		t.Fatalf("fixture planned %d minutes", reservation.PlannedMinutes)
	}
	if err = recordDispatchTestPacket(t, ctx, db, dispatchTestRegistration(assignment), assignment, reservation, run, view); err != nil {
		t.Fatalf("unused exact-minute connection budget refused its complete reservation: %v", err)
	}
}
