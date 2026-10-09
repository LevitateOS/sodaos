package control

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
)

func TestDispatchWaitReasons(t *testing.T) {
	ctx := context.Background()
	setup := func(t *testing.T, mutate func(fx *dispatchFixture, db *store.Store)) (dispatchFixture, DispatchReport) {
		t.Helper()
		db, _ := dispatchTestDB(t)
		fx := dispatchSeed(t, db)
		head := fx.accept(t, 3, "d333333333333333333333333")
		fx.queue(t, 3, head.ID)
		mutate(&fx, db)
		return fx, DispatchPass(ctx, fx.deps())
	}
	cases := map[string]struct {
		mutate   func(fx *dispatchFixture, db *store.Store)
		reason   string
		launched int64
		waited   int64
	}{
		"reads unwired": {
			mutate: func(fx *dispatchFixture, db *store.Store) { fx.reads = nil },
			reason: WaitInputsUnavailable,
		},
		"native busy": {
			mutate: func(fx *dispatchFixture, db *store.Store) {
				fx.reads.err = &AcceptanceRefusal{Reason: RefusalNativeBusy}
			},
			reason: WaitInputsBusy,
		},
		"stale read": {
			mutate: func(fx *dispatchFixture, db *store.Store) {
				fx.reads.err = &AcceptanceRefusal{Reason: RefusalStaleEvidence}
			},
			reason: WaitInputsStale,
		},
		"objective changed": {
			mutate: func(fx *dispatchFixture, db *store.Store) {
				in := fx.reads.inputs["7/3"]
				in.Issue.ContentDigest = strings.Repeat("9", 64)
				fx.reads.inputs["7/3"] = in
			},
			reason: WaitInputsChanged,
		},
		"source changed": {
			mutate: func(fx *dispatchFixture, db *store.Store) {
				in := fx.reads.inputs["7/3"]
				in.Comments[0].ContentVer = 7
				fx.reads.inputs["7/3"] = in
			},
			reason: WaitInputsChanged,
		},
		"hidden issue": {
			mutate: func(fx *dispatchFixture, db *store.Store) {
				in := fx.reads.inputs["7/3"]
				in.Issue.Visible = false
				fx.reads.inputs["7/3"] = in
			},
			reason: WaitInputsHidden,
		},
		"closed issue": {
			mutate: func(fx *dispatchFixture, db *store.Store) {
				in := fx.reads.inputs["7/3"]
				in.Issue.Closed = true
				fx.reads.inputs["7/3"] = in
			},
			reason: WaitIssueClosed,
		},
		"locked issue": {
			mutate: func(fx *dispatchFixture, db *store.Store) {
				in := fx.reads.inputs["7/3"]
				in.Issue.Locked = true
				fx.reads.inputs["7/3"] = in
			},
			reason: WaitIssueLocked,
		},
		"missing tip": {
			mutate: func(fx *dispatchFixture, db *store.Store) {
				in := fx.reads.inputs["7/3"]
				in.Tip = "short"
				fx.reads.inputs["7/3"] = in
			},
			reason: WaitSource,
		},
		"oversized prompt": {
			mutate: func(fx *dispatchFixture, db *store.Store) {
				in := fx.reads.inputs["7/3"]
				big := strings.Repeat("x", project.MaxFactoryPrompt)
				in.Issue.Body, in.Issue.ContentDigest = big, dispatchDigest(big)
				decision := fx.decision[3]
				decision.ContentDigest = dispatchDigest(big)
				decision.Predecessor, decision.ID = decision.ID, "d777777777777777777777777"
				if err := db.AdmitAcceptanceDecision(ctx, decision); err != nil {
					t.Fatal(err)
				}
				fx.decision[3] = decision
				control, err := db.IssueControl(ctx, fx.repo, 3)
				if err != nil {
					t.Fatal(err)
				}
				control.Acceptance, control.Fingerprint = decision.ID, strings.Repeat("4", 64)
				if _, _, err := db.RecordIssueAssessment(ctx, control, time.Now()); err != nil {
					t.Fatal(err)
				}
				fx.reads.inputs["7/3"] = in
			},
			reason: WaitPrompt,
			waited: 3,
		},
		"harness mismatch": {
			mutate: func(fx *dispatchFixture, db *store.Store) {
				fx.host.harness = func(family string) (project.FactoryHarnessPin, error) {
					return project.FactoryHarnessPin{Harness: project.FactoryHarnessCodex, Version: "9.9.9", SHA256: strings.Repeat("a", 64), Image: "sha256:" + strings.Repeat("b", 64)}, nil
				}
			},
			reason: WaitHarness,
		},
		"harness unpinned": {
			mutate: func(fx *dispatchFixture, db *store.Store) {
				fx.host.harness = func(family string) (project.FactoryHarnessPin, error) {
					return project.FactoryHarnessPin{}, nil
				}
			},
			reason: WaitHarness,
		},
		"reviewer-only sponsorship": {
			mutate: func(fx *dispatchFixture, db *store.Store) {
				sp, err := db.Sponsorship(ctx, fx.repo, "conn")
				if err != nil {
					t.Fatal(err)
				}
				sp.Roles = []string{project.RoleReviewer}
				if err := db.SaveSponsorship(ctx, sp); err != nil {
					t.Fatal(err)
				}
			},
			reason: WaitSponsorshipRole,
		},
		"stale preparation": {
			mutate: func(fx *dispatchFixture, db *store.Store) {
				head, err := db.RequirementHead(ctx, fx.proj)
				if err != nil {
					t.Fatal(err)
				}
				commit, digest := strings.Repeat("e", 40), strings.Repeat("d", 64)
				if err := db.AdmitRequirementDecision(ctx, project.RequirementDecision{
					ID: "d999999999999999999999999", Project: fx.proj, Predecessor: head,
					Approver: 7, SourceCommit: commit, SetupDigest: digest, InputsDigest: digest,
				}); err != nil {
					t.Fatal(err)
				}
			},
			reason: WaitPreparation,
			waited: 3,
		},
		"repository full": {
			mutate: func(fx *dispatchFixture, db *store.Store) {
				policy, err := db.RepositoryPolicy(ctx, fx.repo)
				if err != nil {
					t.Fatal(err)
				}
				policy.MaxConcurrent = 1
				if err := db.SaveRepositoryPolicy(ctx, policy); err != nil {
					t.Fatal(err)
				}
				human := factory.Run{
					ID: factory.NewID(), ProjectID: fx.proj, Role: project.RoleCoder, InputSHA: strings.Repeat("c", 40),
					Started: time.Now(), Deadline: time.Now().Add(time.Hour),
					Image: "sha256:" + strings.Repeat("b", 64), Harness: "codex-1.2.3", Model: "m",
				}
				if err := db.RecordFactoryRun(ctx, human); err != nil {
					t.Fatal(err)
				}
			},
			reason: WaitRepository,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if tc.waited == 0 {
				tc.waited = 3
			}
			fx, report := setup(t, tc.mutate)
			if tc.launched == 0 && len(report.Launched) != 0 {
				t.Fatalf("launched = %+v", report.Launched)
			}
			if tc.launched != 0 && (len(report.Launched) != 1 || report.Launched[0].Issue != tc.launched) {
				t.Fatalf("launched = %+v", report.Launched)
			}
			if got := waitReason(report, tc.waited); got != tc.reason {
				t.Fatalf("wait = %q, want %q (waits %+v errors %+v)", got, tc.reason, report.Waits, report.Errors)
			}
			if tc.launched == 0 && len(fx.host.launches) != 0 {
				t.Fatal("host launched while waiting")
			}
		})
	}
}

func TestDispatchWaitsForAttributedUnsettledRepositorySession(t *testing.T) {
	ctx := context.Background()
	db, _ := dispatchTestDB(t)
	fx := dispatchSeed(t, db)
	firstHead := fx.accept(t, 3, "d333333333333333333333333")
	secondHead := fx.accept(t, 4, "d444444444444444444444444")
	fx.queue(t, 3, firstHead.ID)
	fx.queue(t, 4, secondHead.ID)
	firstPass := DispatchPass(ctx, fx.deps())
	if len(firstPass.Launched) != 1 || firstPass.Launched[0].Issue != 3 {
		t.Fatalf("first pass did not launch one session: %+v", firstPass)
	}
	firstRunID := firstPass.Launched[0].RunID
	fx.host.inspect = func(in project.FactoryInspect) (project.FactoryState, error) {
		if in.ID != firstRunID {
			return project.FactoryState{}, errors.New("unexpected run inspection")
		}
		return project.FactoryState{ID: in.ID, Project: in.Project, Role: project.RoleCoder, Phase: project.FactoryRunning}, nil
	}
	assignment, err := db.LatestIssueAssignment(ctx, fx.repo, 3)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.ReleaseReservation(ctx, assignment.ID); err != nil {
		t.Fatal(err)
	}
	secondPass := DispatchPass(ctx, fx.deps())
	if len(secondPass.Launched) != 0 || len(fx.host.launches) != 1 || waitReason(secondPass, 4) != WaitRepository {
		t.Fatalf("unsettled attributed session did not block the next issue: %+v", secondPass)
	}
}

func TestCheckHarnessSelectsPolicyFamily(t *testing.T) {
	fx := dispatchFixture{harness: "0.157.1"}
	available := map[string]project.FactoryHarnessPin{
		project.FactoryHarnessCodex: {Harness: project.FactoryHarnessCodex, Version: "0.157.1", SHA256: strings.Repeat("a", 64), Image: "sha256:" + strings.Repeat("b", 64)},
		project.FactoryHarnessMuse:  {Harness: project.FactoryHarnessMuse, Version: "1.4.0", SHA256: strings.Repeat("c", 64), Image: "sha256:" + strings.Repeat("d", 64)},
	}
	fx.host = &fakeDispatchHost{harness: func(family string) (project.FactoryHarnessPin, error) {
		pin, ok := available[family]
		if !ok {
			return project.FactoryHarnessPin{}, errors.New("family unavailable")
		}
		return pin, nil
	}}
	plan := &attemptPlan{policy: factory.RepositoryPolicy{AttemptLimits: factory.DefaultAttemptLimits(), Roles: map[string]factory.RoleSelection{
		project.RoleCoder:    {Harness: project.FactoryHarnessMuse, HarnessVers: "1.4.0", Model: "muse-spark-1.3"},
		project.RoleReviewer: {Harness: project.FactoryHarnessCodex, HarnessVers: "0.157.1", Model: "codex-model"},
	}}}
	if wait := checkHarnessFor(context.Background(), DispatchDeps{Host: fx.host}, plan, project.RoleCoder); wait != nil {
		t.Fatalf("Muse selection waited: %+v", wait)
	}
	if len(fx.host.families) != 1 || fx.host.families[0] != project.FactoryHarnessMuse {
		t.Fatalf("requested families = %v, want [muse]", fx.host.families)
	}
	if plan.pin.Harness != project.FactoryHarnessMuse || plan.pin.Version != "1.4.0" {
		t.Fatalf("selected pin = %+v", plan.pin)
	}
}

func TestCheckHarnessRejectsFamilyAndVersionMismatch(t *testing.T) {
	for _, tc := range []struct {
		name string
		pin  project.FactoryHarnessPin
	}{
		{name: "wrong family", pin: project.FactoryHarnessPin{Harness: project.FactoryHarnessCodex, Version: "1.4.0", SHA256: strings.Repeat("a", 64), Image: "sha256:" + strings.Repeat("b", 64)}},
		{name: "wrong version", pin: project.FactoryHarnessPin{Harness: project.FactoryHarnessMuse, Version: "9.9.9", SHA256: strings.Repeat("a", 64), Image: "sha256:" + strings.Repeat("b", 64)}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := dispatchFixture{host: &fakeDispatchHost{harness: func(string) (project.FactoryHarnessPin, error) { return tc.pin, nil }}}
			plan := &attemptPlan{policy: factory.RepositoryPolicy{AttemptLimits: factory.DefaultAttemptLimits(), Roles: map[string]factory.RoleSelection{
				project.RoleCoder: {Harness: project.FactoryHarnessMuse, HarnessVers: "1.4.0", Model: "m"},
			}}}
			if wait := checkHarnessFor(context.Background(), DispatchDeps{Host: fx.host}, plan, project.RoleCoder); wait == nil || wait.Reason != WaitHarness {
				t.Fatalf("wait = %+v, want harness wait", wait)
			}
		})
	}
}

func TestDispatchWaitsWhenPolicyChangesAfterPlanning(t *testing.T) {
	ctx := context.Background()
	db, _ := dispatchTestDB(t)
	fx := dispatchSeed(t, db)
	head := fx.accept(t, 3, "d333333333333333333333333")
	fx.queue(t, 3, head.ID)

	harnessRead := false
	fx.host.harness = func(family string) (project.FactoryHarnessPin, error) {
		// Planning has already captured authority and policy by this point;
		// withdraw dispatch authority before the Store records the packet.
		harnessRead = true
		policy, err := db.RepositoryPolicy(ctx, fx.repo)
		if err != nil {
			return project.FactoryHarnessPin{}, err
		}
		policy.Paused = true
		if err := db.SaveRepositoryPolicy(ctx, policy); err != nil {
			return project.FactoryHarnessPin{}, err
		}
		return project.FactoryHarnessPin{
			Harness: family, Version: fx.harness,
			SHA256: strings.Repeat("a", 64), Image: "sha256:" + strings.Repeat("b", 64),
		}, nil
	}

	report := DispatchPass(ctx, fx.deps())
	if !harnessRead {
		t.Fatal("dispatch did not reach the post-capture harness boundary")
	}
	if len(report.Waits) != 1 || report.Waits[0].Issue != 3 || report.Waits[0].Reason != WaitAuthority {
		t.Fatalf("waits = %+v, want issue 3 waiting for authority", report.Waits)
	}
	if len(report.Errors) != 0 || len(report.Launched) != 0 {
		t.Fatalf("report errors/launched = %+v / %+v", report.Errors, report.Launched)
	}
	if len(fx.host.launches) != 0 {
		t.Fatalf("host launches = %d, want 0", len(fx.host.launches))
	}

	if _, err := db.LatestIssueAssignment(ctx, fx.repo, 3); err != store.ErrNotFound {
		t.Fatalf("latest issue assignment = %v; want no packet", err)
	}
	assigned, err := db.AssignedAssignments(ctx, 10)
	if err != nil || len(assigned) != 0 {
		t.Fatalf("assigned packets = %+v, %v; want none", assigned, err)
	}
	runs, err := db.FactoryRuns(ctx, 10)
	if err != nil || len(runs) != 0 {
		t.Fatalf("factory runs = %+v, %v; want none", runs, err)
	}
	views, err := db.FactoryRunViews(ctx, 10)
	if err != nil || len(views) != 0 {
		t.Fatalf("factory run views = %+v, %v; want none", views, err)
	}
}

func TestDispatchNeverFallsBackToAnotherConnection(t *testing.T) {
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
	rich := factory.Sponsorship{
		Repository: fx.repo, GrantedBy: 7, ActorID: 7, ProjectID: fx.proj, Generation: 1, Connection: "zzz-rich", GrantID: "grant2",
		Roles: []string{project.RoleCoder}, AllowanceMinutes: 10080, MaxConcurrent: 8, Active: true,
	}
	if err := db.SaveSponsorship(ctx, rich); err != nil {
		t.Fatal(err)
	}
	head := fx.accept(t, 3, "d333333333333333333333333")
	fx.queue(t, 3, head.ID)
	first := DispatchPass(ctx, fx.deps())
	if len(first.Launched) != 1 {
		t.Fatalf("first pass = %+v %+v", first.Launched, first.Waits)
	}
	a, err := db.Assignment(ctx, first.Launched[0].AssignmentID)
	if err != nil || a.Connection != "conn" {
		t.Fatalf("assignment connection = %+v %v", a, err)
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
	head2 := fx.accept(t, 5, "d555555555555555555555555")
	fx.queue(t, 5, head2.ID)
	second := DispatchPass(ctx, fx.deps())
	if len(second.Launched) != 0 || waitReason(second, 5) != WaitAllowance {
		t.Fatalf("second pass = %+v %+v", second.Launched, second.Waits)
	}
	if total, _ := db.UsageTotal(ctx, fx.repo, "zzz-rich"); total != 0 {
		t.Fatalf("fallback connection consumed %d", total)
	}
}

func TestDispatchWithdrawnGateRecordsNothing(t *testing.T) {
	ctx := context.Background()
	db, _ := dispatchTestDB(t)
	fx := dispatchSeed(t, db)
	head := fx.accept(t, 3, "d333333333333333333333333")
	fx.queue(t, 3, head.ID)
	if _, err := db.WithdrawDispatch(ctx, fx.repo, "test", "tester"); err != nil {
		t.Fatal(err)
	}
	report := DispatchPass(ctx, fx.deps())
	if len(report.Launched) != 0 || len(report.Waits) != 1 || report.Waits[0].Reason != WaitDispatchClosed {
		t.Fatalf("report = %+v %+v %+v", report.Launched, report.Waits, report.Errors)
	}
	if _, err := db.LatestIssueAssignment(ctx, fx.repo, 3); err != store.ErrNotFound {
		t.Fatalf("latest issue assignment = %v; want no packet", err)
	}
	if len(fx.host.launches) != 0 {
		t.Fatal("host launched behind a closed gate")
	}
}
