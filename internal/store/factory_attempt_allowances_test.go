package store

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
)

func TestAttemptAllowanceSurvivesRetryAndPolicyChange(t *testing.T) {
	ctx := context.Background()
	db := dispatchStoreFixture(t)
	now := time.Now().UTC()
	a, reservation, run, view := dispatchTestPacket(t, now)
	if err := recordDispatchTestPacket(t, ctx, db, dispatchTestRegistration(a), a, reservation, run, view); err != nil {
		t.Fatal(err)
	}

	allowance, err := db.AttemptAllowance(ctx, a.Repository, a.Issue)
	if err != nil || allowance.RootAssignment != a.ID || allowance.Limits != factory.DefaultAttemptLimits() || !allowance.Active {
		t.Fatalf("initial allowance = %+v, %v", allowance, err)
	}

	policy, err := db.RepositoryPolicy(ctx, a.Repository)
	if err != nil {
		t.Fatal(err)
	}
	policy.AttemptLimits.ActiveMinutes += 60
	if err = db.SaveRepositoryPolicy(ctx, policy); err != nil {
		t.Fatal(err)
	}

	control, err := db.IssueControl(ctx, a.Repository, a.Issue)
	if err != nil {
		t.Fatal(err)
	}
	retry, retryView := retryTestRun(t, a, run)
	if _, err = db.RecordRetryPacket(ctx, a, control, retry, retryView, 30); err != nil {
		t.Fatal(err)
	}
	allowance, err = db.AttemptAllowance(ctx, a.Repository, a.Issue)
	if err != nil || allowance.RootAssignment != a.ID || allowance.Limits != factory.DefaultAttemptLimits() {
		t.Fatalf("retry replenished the allowance: %+v, %v", allowance, err)
	}
	tx, err := db.db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err = admitAttemptAllowanceTx(ctx, tx, a.Repository, a.Issue, a.ID, nil, time.Now().Add(time.Hour), time.Now()); err != nil {
		_ = tx.Rollback()
		t.Fatal(err)
	}
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
	allowance, err = db.AttemptAllowance(ctx, a.Repository, a.Issue)
	if err != nil || allowance.RootAssignment != a.ID || allowance.Limits != factory.DefaultAttemptLimits() {
		t.Fatalf("new assignment identity replenished the allowance: %+v, %v", allowance, err)
	}
}

func TestCorrectionAdmissionConsumesRootAllowanceOnce(t *testing.T) {
	ctx := context.Background()
	db := publicationStoreFixture(t)
	now := time.Now().UTC()
	root := recordFinishedAssignment(t, db, now, 3, true, "completed")
	p := publicationTestRecord()
	p.AssignmentID, p.Run, p.Preparation, p.BaseSHA = root.ID, root.Run, root.Preparation, root.SourceCommit
	if err := db.RecordPublication(ctx, p); err != nil {
		t.Fatal(err)
	}
	p.NativeRev, p.ObservedUnix, p.Comparison = 9, 1150, strings.Repeat("c", 40)
	p.Publish = publishedStoreOperation(p, factory.OpRefPublish, 1, "branch-receipt")
	p.Publish.Effect, p.Publish.Completion, p.Publish.Receipt = factory.OpEffectPending, factory.OpCompletionPending, ""
	p.Revision++
	if err := db.UpdatePublication(ctx, p); err != nil {
		t.Fatal(err)
	}
	p.Publish.Effect, p.Publish.Completion, p.Publish.Receipt = factory.OpEffectCommitted, factory.OpCompletionComplete, "branch-receipt"
	p.Revision++
	if err := db.UpdatePublication(ctx, p); err != nil {
		t.Fatal(err)
	}
	p.PRCreate = publishedStoreOperation(p, factory.OpPRCreate, 1, `{"pr_number":9}`)
	p.PRCreate.Effect, p.PRCreate.Completion, p.PRCreate.Receipt = factory.OpEffectPending, factory.OpCompletionPending, ""
	p.PRCreate.PRNumber, p.PRCreate.PRID, p.PRCreate.IssueID = 0, 0, 0
	p.PRCreate.HeadRef, p.PRCreate.BaseRef, p.PRCreate.HeadOID, p.PRCreate.BaseOID = "", "", "", ""
	p.Revision++
	if err := db.UpdatePublication(ctx, p); err != nil {
		t.Fatal(err)
	}
	p.PRCreate.Effect, p.PRCreate.Completion, p.PRCreate.Receipt = factory.OpEffectCommitted, factory.OpCompletionComplete, `{"pr_number":9}`
	p.PRNumber, p.PRID = 9, 8
	p.PRCreate.PRNumber, p.PRCreate.PRID, p.PRCreate.IssueID = 9, 8, p.Issue
	p.PRCreate.HeadRef, p.PRCreate.BaseRef = factory.PublishBranchName(p.AssignmentID), p.TargetBranch
	p.PRCreate.HeadOID, p.PRCreate.BaseOID = p.Candidate, p.Comparison
	p.Stage, p.Outcome, p.Reason, p.FinishedUnix = factory.PublicationPublished, factory.Succeeded, factory.PublishReasonLinked, 1200
	p.Revision++
	if err := db.UpdatePublication(ctx, p); err != nil {
		t.Fatal(err)
	}

	newChild := func(source string) (factory.Assignment, factory.Reservation, factory.Run, factory.RunView, factory.IssueControl) {
		child := root
		child.ID, child.AttemptRoot, child.PublicationAssignment = factory.NewID(), root.AttemptRoot, root.ID
		child.Run, child.RunHistory, child.Stage, child.Revision = factory.NewID(), nil, factory.AssignmentAssigned, 0
		child.Attempts, child.Outcome, child.Reason, child.Result, child.FinishedUnix = 1, "", "", nil, 0
		child.SourceCommit = source
		child.RunHistory = []string{child.Run}
		run := factory.Run{
			ID: child.Run, ProjectID: child.ProjectID, Role: child.Role, InputSHA: source,
			Started: time.Now().UTC(), Deadline: time.Now().UTC().Add(30 * time.Minute), Image: "sha256:" + strings.Repeat("b", 64), Harness: child.Harness, Model: child.Model,
		}
		reservation := factory.Reservation{AssignmentID: child.ID, Repository: child.Repository, Connection: child.Connection, State: factory.ReservationHeld, PlannedMinutes: 30}
		view := factory.RunView{RunID: run.ID, Repository: child.Repository, Issue: child.Issue, Attempt: child.ID}
		return child, reservation, run, view, dispatchTestControlFor(t, db, child)
	}
	allowance, err := db.AttemptAllowance(ctx, root.Repository, root.Issue)
	if err != nil {
		t.Fatal(err)
	}
	allowance.Corrections = []string{factory.NewID(), factory.NewID(), factory.NewID()}
	allowance.Revision++
	tx, err := db.db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err = saveAttemptAllowanceTx(ctx, tx, allowance); err != nil {
		_ = tx.Rollback()
		t.Fatal(err)
	}
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
	exhausted, reservation, exhaustedRun, exhaustedView, control := newChild(p.Candidate)
	if err = db.RecordDispatchPacket(ctx, dispatchTestRegistration(exhausted), control, exhausted, reservation, exhaustedRun, exhaustedView); !errors.Is(err, factory.ErrCorrectionExhausted) {
		t.Fatalf("exhausted correction packet = %v", err)
	}
	if _, err = db.Assignment(ctx, exhausted.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("exhausted correction left assignment: %v", err)
	}

	allowance.Corrections = nil
	allowance.Revision++
	tx, err = db.db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err = saveAttemptAllowanceTx(ctx, tx, allowance); err != nil {
		_ = tx.Rollback()
		t.Fatal(err)
	}
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}

	bad, badReservation, badRun, badView, badControl := newChild(strings.Repeat("4", 40))
	if err = db.RecordDispatchPacket(ctx, dispatchTestRegistration(bad), badControl, bad, badReservation, badRun, badView); err == nil {
		t.Fatal("noncurrent published candidate packet accepted")
	}
	if _, err = db.Assignment(ctx, bad.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("candidate mismatch left assignment: %v", err)
	}

	child, childReservation, childRun, childView, control := newChild(p.Candidate)
	if err = db.RecordDispatchPacket(ctx, dispatchTestRegistration(child), control, child, childReservation, childRun, childView); err != nil {
		t.Fatalf("published correction packet refused: %v", err)
	}
	allowance, err = db.AttemptAllowance(ctx, root.Repository, root.Issue)
	if err != nil || allowance.RootAssignment != root.ID || len(allowance.Corrections) != 1 || allowance.Corrections[0] != child.ID {
		t.Fatalf("correction packet charge = %+v, %v", allowance, err)
	}

	childRun.Outcome, childRun.Reconciled = factory.Failed, true
	if err = db.SaveFactoryRun(ctx, childRun); err != nil {
		t.Fatal(err)
	}
	if err = db.ReleaseReservation(ctx, child.ID); err != nil {
		t.Fatal(err)
	}
	retryRun, retryView := retryTestRun(t, child, childRun)
	if _, err = db.RecordRetryPacket(ctx, child, control, retryRun, retryView, 30); err != nil {
		t.Fatalf("correction retry refused: %v", err)
	}
	allowance, err = db.AttemptAllowance(ctx, root.Repository, root.Issue)
	if err != nil || len(allowance.Corrections) != 1 || allowance.Corrections[0] != child.ID {
		t.Fatalf("correction retry charged another cycle: %+v, %v", allowance, err)
	}

	wrongRoot, wrongReservation, wrongRun, wrongView, wrongControl := newChild(p.Candidate)
	wrongRoot.AttemptRoot = factory.NewID()
	if err = db.RecordDispatchPacket(ctx, dispatchTestRegistration(wrongRoot), wrongControl, wrongRoot, wrongReservation, wrongRun, wrongView); err == nil {
		t.Fatal("child with mismatched attempt root accepted")
	}
	if _, err = db.Assignment(ctx, wrongRoot.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("root mismatch left assignment: %v", err)
	}
	allowance, err = db.AttemptAllowance(ctx, root.Repository, root.Issue)
	if err != nil || len(allowance.Corrections) != 1 || allowance.Corrections[0] != child.ID {
		t.Fatalf("rejected child changed correction history: %+v, %v", allowance, err)
	}
}

func TestFreshRetryAllowancePreservesConsumedPriorRoot(t *testing.T) {
	ctx := context.Background()
	db := dispatchStoreFixture(t)
	now := time.Now().UTC()
	a, reservation, run, view := dispatchTestPacket(t, now)
	if err := recordDispatchTestPacket(t, ctx, db, dispatchTestRegistration(a), a, reservation, run, view); err != nil {
		t.Fatal(err)
	}
	prior, err := db.AttemptAllowance(ctx, a.Repository, a.Issue)
	if err != nil {
		t.Fatal(err)
	}
	prior.ActiveSeconds = int64(prior.Limits.ActiveMinutes * 60)
	prior.Corrections = []string{factory.NewID(), factory.NewID(), factory.NewID()}
	prior.CheckpointUnix, prior.Active = now.Unix(), true
	prior.Revision++

	root := a
	root.ID, root.Run = factory.NewID(), factory.NewID()
	root.AttemptRoot, root.PublicationAssignment = root.ID, root.ID
	root.Stage, root.Outcome, root.Reason = factory.AssignmentFinished, factory.Failed, factory.AssignReasonRunFailed
	root.Attempts, root.RunHistory, root.Revision = 1, []string{root.Run}, 0
	root.FinishedUnix = now.Unix()
	result := factory.ResultSynthesized(root.ID, root.Run, "failed", "settled", now.Unix())
	root.Result = &result
	if err := root.Validate(); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(root)
	if err != nil {
		t.Fatal(err)
	}
	tx, err := db.db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err = tx.ExecContext(ctx, `INSERT INTO factory_assignments(id,repository,issue,run,stage,revision,data) VALUES($1,$2,$3,$4,$5,$6,$7)`,
		root.ID, root.Repository, root.Issue, root.Run, root.Stage, root.Revision, string(data)); err != nil {
		t.Fatal(err)
	}
	if err = saveAttemptAllowanceTx(ctx, tx, prior); err != nil {
		t.Fatal(err)
	}
	limits := factory.AttemptLimits{ActiveMinutes: 45, CorrectionCycles: 1}
	if err = startFreshAttemptAllowanceTx(ctx, tx, a.Repository, a.Issue, root.ID, limits, now.Add(30*time.Minute), now); err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}

	fresh, err := db.AttemptAllowance(ctx, a.Repository, a.Issue)
	if err != nil || fresh.RootAssignment != root.ID || fresh.Limits != limits || fresh.ActiveSeconds != 0 || len(fresh.Corrections) != 0 || !fresh.Active {
		t.Fatalf("fresh root allowance = %+v, %v", fresh, err)
	}
	var oldData []byte
	if err = db.db.QueryRowContext(ctx, `SELECT data FROM factory_attempt_allowances WHERE root_assignment=$1`, a.ID).Scan(&oldData); err != nil {
		t.Fatal(err)
	}
	var previous factory.AttemptAllowance
	if err = json.Unmarshal(oldData, &previous); err != nil {
		t.Fatal(err)
	}
	if err = validateAttemptAllowanceRow(previous, a.Repository, a.Issue, a.ID, previous.Revision); err != nil {
		t.Fatal(err)
	}
	if previous.Limits != factory.DefaultAttemptLimits() || len(previous.Corrections) != 3 || previous.ActiveSeconds != int64(previous.Limits.ActiveMinutes*60) || previous.Active {
		t.Fatalf("prior root consumption changed: %+v", previous)
	}
}

func TestAttemptAllowanceExhaustionRollsBackRetry(t *testing.T) {
	ctx := context.Background()
	db := dispatchStoreFixture(t)
	now := time.Now().UTC()
	a, reservation, run, view := dispatchTestPacket(t, now)
	if err := recordDispatchTestPacket(t, ctx, db, dispatchTestRegistration(a), a, reservation, run, view); err != nil {
		t.Fatal(err)
	}
	allowance, err := db.AttemptAllowance(ctx, a.Repository, a.Issue)
	if err != nil {
		t.Fatal(err)
	}
	allowance.ActiveSeconds = int64(allowance.Limits.ActiveMinutes*60) - 10
	allowance.CheckpointUnix = time.Now().Unix()
	allowance.Revision++
	data, err := json.Marshal(allowance)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.db.ExecContext(ctx, `UPDATE factory_attempt_allowances SET revision=$1,data=$2 WHERE repository=$3 AND issue=$4`, allowance.Revision, string(data), a.Repository, a.Issue); err != nil {
		t.Fatal(err)
	}

	control, err := db.IssueControl(ctx, a.Repository, a.Issue)
	if err != nil {
		t.Fatal(err)
	}
	retry, retryView := retryTestRun(t, a, run)
	if _, err = db.RecordRetryPacket(ctx, a, control, retry, retryView, 30); !errors.Is(err, factory.ErrAttemptTimeExhausted) {
		t.Fatalf("exhausted retry error = %v", err)
	}
	if _, err = db.FactoryRun(ctx, retry.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("refused retry left a run: %v", err)
	}
	stored, err := db.Assignment(ctx, a.ID)
	if err != nil || stored.Revision != a.Revision || stored.Run != a.Run {
		t.Fatalf("refused retry changed assignment: %+v, %v", stored, err)
	}
}

func TestInitialAttemptDeadlineMustFitRecordedAllowance(t *testing.T) {
	for name, expired := range map[string]bool{"over budget": false, "already expired": true} {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			db := dispatchStoreFixture(t)
			now := time.Now().UTC()
			a, reservation, run, view := dispatchTestPacket(t, now)
			if expired {
				run.Started = now.Add(-time.Minute)
				run.Deadline = now.Add(-time.Second)
			} else {
				run.Deadline = now.Add(3 * time.Hour)
			}
			if err := recordDispatchTestPacket(t, ctx, db, dispatchTestRegistration(a), a, reservation, run, view); !errors.Is(err, factory.ErrAttemptTimeExhausted) {
				t.Fatalf("inadmissible initial deadline error = %v", err)
			}
			for _, check := range []struct {
				table, key string
				id         string
			}{
				{"factory_assignments", "id", a.ID},
				{"factory_runs", "id", run.ID},
				{"factory_reservations", "assignment", a.ID},
				{"factory_dispatch_regs", "id", a.ID},
			} {
				var exists bool
				if err := db.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM `+check.table+` WHERE `+check.key+`=$1)`, check.id).Scan(&exists); err != nil {
					t.Fatal(err)
				}
				if exists {
					t.Fatalf("refused packet left %s row", check.table)
				}
			}
			if _, err := db.AttemptAllowance(ctx, a.Repository, a.Issue); !errors.Is(err, ErrNotFound) {
				t.Fatalf("refused packet left allowance: %v", err)
			}
		})
	}
}

func TestPauseFreezesOnlyAfterSettledViewsAndRechecksGateRevision(t *testing.T) {
	ctx := context.Background()
	db := dispatchStoreFixture(t)
	now := time.Now().UTC()
	a, reservation, run, view := dispatchTestPacket(t, now)
	if err := recordDispatchTestPacket(t, ctx, db, dispatchTestRegistration(a), a, reservation, run, view); err != nil {
		t.Fatal(err)
	}
	withdrawal, err := db.WithdrawDispatch(ctx, a.Repository, factory.CauseControlPaused, "soda-tester")
	if err != nil {
		t.Fatal(err)
	}
	if err = db.FreezeAttemptAllowances(ctx, a.Repository, withdrawal.Revision, time.Now()); !errors.Is(err, ErrAttemptRunsPending) {
		t.Fatalf("pause freeze with unsettled run = %v", err)
	}
	settled, err := db.FactoryRun(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	settled.Outcome, settled.Reconciled = factory.Failed, true
	if err = db.SaveFactoryRun(ctx, settled); err != nil {
		t.Fatal(err)
	}
	if err = db.FreezeAttemptAllowances(ctx, a.Repository, withdrawal.Revision, time.Now()); err != nil {
		t.Fatal(err)
	}
	allowance, err := db.AttemptAllowance(ctx, a.Repository, a.Issue)
	if err != nil || allowance.Active {
		t.Fatalf("pause did not freeze allowance: %+v, %v", allowance, err)
	}
	if _, _, err = db.ReopenDispatch(ctx, a.Repository, withdrawal.Revision); err != nil {
		t.Fatal(err)
	}
	if err = db.FreezeAttemptAllowances(ctx, a.Repository, withdrawal.Revision, time.Now()); !errors.Is(err, ErrStaleRevision) {
		t.Fatalf("freeze accepted stale closed-gate revision: %v", err)
	}
}
