package store

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
)

func saveExpiredAttemptAllowance(t *testing.T, db *Store, allowance factory.AttemptAllowance) {
	t.Helper()
	tx, err := db.db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback() }()
	if err = saveAttemptAllowanceTx(context.Background(), tx, allowance); err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
}

func TestCloseExpiredAttemptClosesSettledRootAndReplays(t *testing.T) {
	ctx := context.Background()
	db := dispatchStoreFixture(t)
	now := time.Now().UTC()
	a, reservation, run, view := dispatchTestPacket(t, now)
	if err := recordDispatchTestPacket(t, ctx, db, dispatchTestRegistration(a), a, reservation, run, view); err != nil {
		t.Fatal(err)
	}

	initial, err := db.AttemptAllowance(ctx, a.Repository, a.Issue)
	if err != nil {
		t.Fatal(err)
	}
	closed, err := db.CloseExpiredAttempt(ctx, a.ID)
	if err != nil || closed {
		t.Fatalf("unexpired close = %v, %v", closed, err)
	}
	unchanged, err := db.AttemptAllowance(ctx, a.Repository, a.Issue)
	if err != nil || unchanged.Revision != initial.Revision || unchanged.Closed || !unchanged.Active {
		t.Fatalf("unexpired attempt changed: %+v, %v", unchanged, err)
	}

	future := unchanged
	future.CheckpointUnix = time.Now().Add(time.Minute).Unix()
	future.Revision++
	saveExpiredAttemptAllowance(t, db, future)
	if _, err = db.CloseExpiredAttempt(ctx, a.ID); !errors.Is(err, factory.ErrAllowanceClock) {
		t.Fatalf("future checkpoint close = %v", err)
	}
	afterClockError, err := db.AttemptAllowance(ctx, a.Repository, a.Issue)
	if err != nil || afterClockError.Revision != future.Revision || afterClockError.CheckpointUnix != future.CheckpointUnix || afterClockError.Closed {
		t.Fatalf("clock error changed allowance: %+v, %v", afterClockError, err)
	}

	paused := afterClockError
	paused.CheckpointUnix = time.Now().Unix()
	paused.Active = false
	paused.Revision++
	saveExpiredAttemptAllowance(t, db, paused)
	closed, err = db.CloseExpiredAttempt(ctx, a.ID)
	pausedAfter, readErr := db.AttemptAllowance(ctx, a.Repository, a.Issue)
	if err != nil || readErr != nil || closed || pausedAfter.Revision != paused.Revision || pausedAfter.Closed || pausedAfter.Active {
		t.Fatalf("paused allowance changed on expiry scan: closed=%v allowance=%+v err=%v read=%v", closed, pausedAfter, err, readErr)
	}

	settled, err := db.FactoryRun(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	settled.Outcome, settled.Reconciled = factory.Failed, true
	if err = db.SaveFactoryRun(ctx, settled); err != nil {
		t.Fatal(err)
	}
	exhausted, err := db.AttemptAllowance(ctx, a.Repository, a.Issue)
	if err != nil {
		t.Fatal(err)
	}
	exhausted.Active = true
	exhausted.ActiveSeconds = int64(exhausted.Limits.ActiveMinutes * 60)
	exhausted.CheckpointUnix = time.Now().Unix()
	exhausted.Revision++
	saveExpiredAttemptAllowance(t, db, exhausted)

	closed, err = db.CloseExpiredAttempt(ctx, a.ID)
	if err != nil || !closed {
		t.Fatalf("expired settled close = %v, %v", closed, err)
	}
	after, err := db.AttemptAllowance(ctx, a.Repository, a.Issue)
	if err != nil || !after.Closed || after.Active || after.Revision != exhausted.Revision+1 {
		t.Fatalf("expired settled allowance = %+v, %v", after, err)
	}
	closed, err = db.CloseExpiredAttempt(ctx, a.ID)
	replayed, readErr := db.AttemptAllowance(ctx, a.Repository, a.Issue)
	if err != nil || readErr != nil || !closed || replayed.Revision != after.Revision || replayed.Active || !replayed.Closed {
		t.Fatalf("close replay changed settled root: closed=%v allowance=%+v err=%v read=%v", closed, replayed, err, readErr)
	}
}

func TestCloseExpiredAttemptRetainsLiveRootUntilRecordedRunSettles(t *testing.T) {
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
	allowance.ActiveSeconds = int64(allowance.Limits.ActiveMinutes * 60)
	allowance.CheckpointUnix = time.Now().Unix()
	allowance.Revision++
	saveExpiredAttemptAllowance(t, db, allowance)

	closed, err := db.CloseExpiredAttempt(ctx, a.ID)
	if err != nil || !closed {
		t.Fatalf("expired live close = %v, %v", closed, err)
	}
	retained, err := db.AttemptAllowance(ctx, a.Repository, a.Issue)
	if err != nil || !retained.Closed || !retained.Active {
		t.Fatalf("live run did not retain active slot: %+v, %v", retained, err)
	}
	closed, err = db.CloseExpiredAttempt(ctx, a.ID)
	replayed, readErr := db.AttemptAllowance(ctx, a.Repository, a.Issue)
	if err != nil || readErr != nil || !closed || replayed.Revision != retained.Revision || !replayed.Active || !replayed.Closed {
		t.Fatalf("live close replay changed allowance: closed=%v allowance=%+v err=%v read=%v", closed, replayed, err, readErr)
	}

	settled, err := db.FactoryRun(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	settled.Outcome, settled.Reconciled = factory.Failed, true
	if err = db.SaveFactoryRun(ctx, settled); err != nil {
		t.Fatal(err)
	}
	released, err := db.AttemptAllowance(ctx, a.Repository, a.Issue)
	if err != nil || !released.Closed || released.Active || released.Revision != retained.Revision+1 {
		t.Fatalf("settlement did not release closed root: allowance=%+v, %v", released, err)
	}
	closed, err = db.CloseExpiredAttempt(ctx, a.ID)
	replayed, readErr = db.AttemptAllowance(ctx, a.Repository, a.Issue)
	if err != nil || readErr != nil || !closed || replayed.Revision != released.Revision || replayed.Active {
		t.Fatalf("settlement replay changed allowance: closed=%v allowance=%+v err=%v read=%v", closed, replayed, err, readErr)
	}
}

func TestCloseExpiredAttemptIgnoresStalePublicationOwnerWithinRoot(t *testing.T) {
	ctx := context.Background()
	db := publicationStoreFixture(t)
	now := time.Now().UTC()
	a := recordFinishedAssignment(t, db, now, 3, true, "completed")
	allowance, err := db.AttemptAllowance(ctx, a.Repository, a.Issue)
	if err != nil {
		t.Fatal(err)
	}
	allowance.ActiveSeconds = int64(allowance.Limits.ActiveMinutes * 60)
	allowance.CheckpointUnix = time.Now().Unix()
	allowance.Revision++
	saveExpiredAttemptAllowance(t, db, allowance)

	// A newer same-root assignment now owns a distinct publication. The old
	// publication's assignment must not close that current owner's root.
	latest := a
	latest.ID, latest.Run = factory.NewID(), factory.NewID()
	latest.PublicationAssignment = latest.ID
	latest.RunHistory = []string{latest.Run}
	latest.Revision = 0
	latest.CreatedUnix = time.Now().Unix()
	result := *latest.Result
	result.AssignmentID, result.RunID = latest.ID, latest.Run
	latest.Result = &result
	if err = latest.Validate(); err != nil {
		t.Fatalf("stale-owner assignment fixture: %v", err)
	}
	data, err := json.Marshal(latest)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.db.ExecContext(ctx, `INSERT INTO factory_assignments(id,repository,issue,run,stage,revision,data) VALUES($1,$2,$3,$4,$5,$6,$7)`,
		latest.ID, latest.Repository, latest.Issue, latest.Run, latest.Stage, latest.Revision, string(data)); err != nil {
		t.Fatal(err)
	}

	closed, err := db.CloseExpiredAttempt(ctx, a.ID)
	after, readErr := db.AttemptAllowance(ctx, a.Repository, a.Issue)
	if err != nil || readErr != nil || closed || after.Revision != allowance.Revision || after.Closed || !after.Active {
		t.Fatalf("stale publication owner changed current root: closed=%v allowance=%+v err=%v read=%v", closed, after, err, readErr)
	}
}
