package control

import (
	"context"
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
				fx.host.harness = func() (project.FactoryHarnessPin, error) {
					return project.FactoryHarnessPin{Harness: project.FactoryHarnessCodex, Version: "9.9.9", SHA256: strings.Repeat("a", 64), Image: "sha256:" + strings.Repeat("b", 64)}, nil
				}
			},
			reason: WaitHarness,
		},
		"harness unpinned": {
			mutate: func(fx *dispatchFixture, db *store.Store) {
				fx.host.harness = func() (project.FactoryHarnessPin, error) {
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
		"sponsorship full": {
			mutate: func(fx *dispatchFixture, db *store.Store) {
				sp, err := db.Sponsorship(ctx, fx.repo, "conn")
				if err != nil {
					t.Fatal(err)
				}
				sp.MaxConcurrent = 1
				if err := db.SaveSponsorship(ctx, sp); err != nil {
					t.Fatal(err)
				}
				head := fx.accept(t, 9, "d999999999999999999999999")
				fx.queue(t, 9, head.ID)
			},
			reason:   WaitSponsorship,
			launched: 3,
			waited:   9,
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
		Repository: fx.repo, GrantedBy: 7, Generation: 1, Connection: "zzz-rich", GrantID: "grant2",
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
	list, err := db.IssueAssignments(ctx, fx.repo, 3)
	if err != nil || len(list) != 0 {
		t.Fatalf("assignments = %+v %v", list, err)
	}
	if len(fx.host.launches) != 0 {
		t.Fatal("host launched behind a closed gate")
	}
}
