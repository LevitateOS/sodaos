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

// Typed completion-root ACTIVE TIME proof (D13/FW32) against the
// coordinator-owned schema 54 format: factory_assignments and
// factory_attempt_allowances admit Issue 0 only with the canonical
// completion subject, reassess phase and coder role. Every test below
// opens its own owned disposable database; nothing is borrowed.

// reassessmentRootFixture builds one valid completion reassessment root
// assignment: repository slot, Issue zero, fresh attempt root, reassess
// phase, coder role, no parent, no publication, and the canonical
// completion subject.
func reassessmentRootFixture(t *testing.T, now time.Time, repository int64, completion, cycle, digest string) factory.Assignment {
	t.Helper()
	selection := grantTestPolicy().Roles[project.RoleCoder]
	prompt := []byte("reassessment active-time probe " + cycle)
	sum := sha256.Sum256(prompt)
	root := factory.NewID()
	run := factory.NewID()
	a := factory.Assignment{
		ID: root, AttemptRoot: root, PublicationAssignment: "", ParentAssignment: "",
		ProjectID: dispatchTestProjectID(), Role: project.RoleCoder, Phase: project.ExecutionReassess,
		Subject: factory.ExecutionSubject{
			Kind:       factory.SubjectCompletion,
			Completion: &factory.CompletionSubject{CompletionID: completion, CycleID: cycle, InventoryDigest: digest},
		},
		ActorID:    7,
		Repository: repository, Issue: 0, Revision: 0, NativeRev: 5,
		Acceptance: "", Preparation: "f" + factory.NewID()[:24],
		Harness: selection.Harness + "-" + selection.HarnessVers, HarnessVers: selection.HarnessVers, Model: selection.Model,
		Connection: "conn", SourceCommit: strings.Repeat("c", 40),
		Prompt: prompt, PromptSHA: hex.EncodeToString(sum[:]),
		Run: run, RunHistory: []string{run}, Stage: factory.AssignmentAssigned, Attempts: 1,
		CreatedUnix: now.Unix(), Authority: dispatchTestAuthority(),
	}
	if err := a.Validate(); err != nil {
		t.Fatalf("reassessment root fixture invalid: %v", err)
	}
	return a
}

func insertReassessmentRootRow(t *testing.T, ctx context.Context, db *Store, a factory.Assignment) {
	t.Helper()
	data, err := json.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.db.ExecContext(ctx, `INSERT INTO factory_assignments(id,repository,issue,run,stage,revision,data) VALUES($1,$2,$3,$4,$5,$6,$7)`,
		a.ID, a.Repository, a.Issue, a.Run, a.Stage, a.Revision, string(data)); err != nil {
		t.Fatal(err)
	}
}

func commitReassessmentInsert(t *testing.T, ctx context.Context, db *Store, a factory.Assignment, minutes int, deadline, now time.Time) {
	t.Helper()
	tx, err := db.db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback() }()
	if err = insertReassessmentAttemptAllowanceTx(ctx, tx, a, minutes, deadline, now); err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
}

func reassessmentSubjectOf(a factory.Assignment) factory.CompletionSubject {
	return *a.Subject.Completion
}

func TestReassessmentAllowanceTypedRootInsertRead(t *testing.T) {
	ctx := context.Background()
	db, _ := postgresFixture(t, nil)
	now := time.Now().UTC()
	a := reassessmentRootFixture(t, now, 7, factory.NewID(), factory.NewID(), strings.Repeat("d", 64))
	insertReassessmentRootRow(t, ctx, db, a)
	commitReassessmentInsert(t, ctx, db, a, 45, now.Add(30*time.Minute), now)
	got, err := db.ReassessmentAttemptAllowance(ctx, 7, reassessmentSubjectOf(a), a.ID)
	if err != nil {
		t.Fatalf("typed read refused: %v", err)
	}
	if got.Repository != 7 || got.Issue != 0 || got.RootAssignment != a.ID || got.Revision != 0 {
		t.Fatalf("typed scope = %+v", got)
	}
	if got.Limits != (factory.AttemptLimits{ActiveMinutes: 45}) {
		t.Fatalf("typed limits = %+v", got.Limits)
	}
	if got.Completion == nil || *got.Completion != reassessmentSubjectOf(a) {
		t.Fatalf("typed subject = %+v", got.Completion)
	}
	if !got.Active || got.Closed || got.ActiveSeconds != 0 || len(got.Corrections) != 0 ||
		len(got.DeliveryCalls) != 0 || len(got.Reviews) != 0 {
		t.Fatalf("fresh typed consumption = %+v", got)
	}
	// Omitted minutes resolve to the admitted default at construction.
	omitted := reassessmentRootFixture(t, now, 7, factory.NewID(), factory.NewID(), strings.Repeat("e", 64))
	insertReassessmentRootRow(t, ctx, db, omitted)
	commitReassessmentInsert(t, ctx, db, omitted, 0, now.Add(20*time.Minute), now)
	resolved, err := db.ReassessmentAttemptAllowance(ctx, 7, reassessmentSubjectOf(omitted), omitted.ID)
	if err != nil || resolved.Limits.ActiveMinutes != factory.DefaultReassessmentMinutes {
		t.Fatalf("omitted minutes = %+v, %v", resolved, err)
	}
}

func TestReassessmentSameRootReplayNoRefill(t *testing.T) {
	ctx := context.Background()
	db, _ := postgresFixture(t, nil)
	now := time.Now().UTC()
	a := reassessmentRootFixture(t, now, 7, factory.NewID(), factory.NewID(), strings.Repeat("d", 64))
	insertReassessmentRootRow(t, ctx, db, a)
	commitReassessmentInsert(t, ctx, db, a, 45, now.Add(30*time.Minute), now)
	// Consume ten active minutes through the shared CAS joint.
	current, err := db.ReassessmentAttemptAllowance(ctx, 7, reassessmentSubjectOf(a), a.ID)
	if err != nil {
		t.Fatal(err)
	}
	advanced, err := current.Checkpoint(now.Add(10*time.Minute), true)
	if err != nil {
		t.Fatal(err)
	}
	advanced.Revision++
	tx, err := db.db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err = saveAttemptAllowanceTx(ctx, tx, advanced); err != nil {
		_ = tx.Rollback()
		t.Fatal(err)
	}
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
	// Replay the same root: no error, no refill, consumption unchanged.
	commitReassessmentInsert(t, ctx, db, a, 45, now.Add(30*time.Minute), now.Add(10*time.Minute))
	replayed, err := db.ReassessmentAttemptAllowance(ctx, 7, reassessmentSubjectOf(a), a.ID)
	if err != nil {
		t.Fatal(err)
	}
	if replayed.ActiveSeconds != 600 || replayed.Revision != 1 || !replayed.Active {
		t.Fatalf("replay refilled or lost consumption: %+v", replayed)
	}
	// A conflicting admitted value on the same root refuses instead of
	// overwriting the recorded allowance.
	tx, err = db.db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback() }()
	if err = insertReassessmentAttemptAllowanceTx(ctx, tx, a, 60, now.Add(30*time.Minute), now); err == nil {
		t.Fatal("conflicting replay minutes accepted")
	}
}

func TestReassessmentTwoCyclesDistinct(t *testing.T) {
	ctx := context.Background()
	db, _ := postgresFixture(t, nil)
	now := time.Now().UTC()
	first := reassessmentRootFixture(t, now, 7, factory.NewID(), factory.NewID(), strings.Repeat("d", 64))
	second := reassessmentRootFixture(t, now, 7, first.Subject.Completion.CompletionID, factory.NewID(), strings.Repeat("d", 64))
	insertReassessmentRootRow(t, ctx, db, first)
	insertReassessmentRootRow(t, ctx, db, second)
	commitReassessmentInsert(t, ctx, db, first, 45, now.Add(30*time.Minute), now)
	commitReassessmentInsert(t, ctx, db, second, 60, now.Add(30*time.Minute), now)
	var count int
	if err := db.db.QueryRowContext(ctx, `SELECT count(*) FROM factory_attempt_allowances WHERE repository=7 AND issue=0`).Scan(&count); err != nil || count != 2 {
		t.Fatalf("typed rows = %d, %v", count, err)
	}
	for _, a := range []factory.Assignment{first, second} {
		got, err := db.ReassessmentAttemptAllowance(ctx, 7, reassessmentSubjectOf(a), a.ID)
		if err != nil || got.RootAssignment != a.ID || *got.Completion != reassessmentSubjectOf(a) {
			t.Fatalf("cycle read = %+v, %v", got, err)
		}
	}
}

func TestReassessmentMismatchRefuses(t *testing.T) {
	ctx := context.Background()
	db, _ := postgresFixture(t, nil)
	now := time.Now().UTC()
	a := reassessmentRootFixture(t, now, 7, factory.NewID(), factory.NewID(), strings.Repeat("d", 64))
	insertReassessmentRootRow(t, ctx, db, a)
	commitReassessmentInsert(t, ctx, db, a, 45, now.Add(30*time.Minute), now)
	subject := reassessmentSubjectOf(a)
	wrongEvent := subject
	wrongEvent.CompletionID = factory.NewID()
	wrongCycle := subject
	wrongCycle.CycleID = factory.NewID()
	wrongDigest := subject
	wrongDigest.InventoryDigest = strings.Repeat("e", 64)
	for _, tc := range []struct {
		name       string
		repository int64
		subject    factory.CompletionSubject
		root       string
	}{
		{"unknown root", 7, subject, factory.NewID()},
		{"wrong event", 7, wrongEvent, a.ID},
		{"wrong cycle", 7, wrongCycle, a.ID},
		{"wrong digest", 7, wrongDigest, a.ID},
		{"wrong repository", 8, subject, a.ID},
	} {
		if _, err := db.ReassessmentAttemptAllowance(ctx, tc.repository, tc.subject, tc.root); err == nil {
			t.Fatalf("%s accepted", tc.name)
		}
	}
	// A positive-issue root with the same repository never answers the
	// typed completion lookup.
	positive := dispatchStoreFixture(t)
	packet, reservation, run, view := dispatchTestPacket(t, now)
	if err := recordDispatchTestPacket(t, ctx, positive, dispatchTestRegistration(packet), packet, reservation, run, view); err != nil {
		t.Fatal(err)
	}
	if _, err := positive.ReassessmentAttemptAllowance(ctx, packet.Repository, subject, packet.ID); err == nil {
		t.Fatal("positive-issue root answered the typed completion lookup")
	}
	// Unscoped issue-zero lookups refuse explicitly instead of falling back
	// to a latest-per-issue row.
	if _, err := positive.AttemptAllowance(ctx, packet.Repository, 0); err == nil ||
		!strings.Contains(err.Error(), "issue scope") {
		t.Fatalf("unscoped issue-zero lookup = %v", err)
	}
	tx, err := positive.db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback() }()
	if err = admitAttemptAllowanceTx(ctx, tx, packet.Repository, 0, packet.ID, nil, now.Add(time.Hour), now); err == nil ||
		!strings.Contains(err.Error(), "issue scope") {
		t.Fatalf("unscoped issue-zero admission = %v", err)
	}
}

func TestReassessmentMalformedInputsRefuse(t *testing.T) {
	ctx := context.Background()
	db, _ := postgresFixture(t, nil)
	now := time.Now().UTC()
	a := reassessmentRootFixture(t, now, 7, factory.NewID(), factory.NewID(), strings.Repeat("d", 64))
	subject := reassessmentSubjectOf(a)
	badSubject := subject
	badSubject.CompletionID = "short"
	badCycle := subject
	badCycle.CycleID = "short"
	badDigest := subject
	badDigest.InventoryDigest = "short"
	for _, tc := range []struct {
		name       string
		repository int64
		subject    factory.CompletionSubject
		root       string
	}{
		{"zero repository", 0, subject, a.ID},
		{"negative repository", -1, subject, a.ID},
		{"empty root", 7, subject, ""},
		{"short root", 7, subject, "short"},
		{"short completion", 7, badSubject, a.ID},
		{"short cycle", 7, badCycle, a.ID},
		{"short digest", 7, badDigest, a.ID},
		{"empty subject", 7, factory.CompletionSubject{}, a.ID},
	} {
		if _, err := db.ReassessmentAttemptAllowance(ctx, tc.repository, tc.subject, tc.root); err == nil {
			t.Fatalf("%s accepted", tc.name)
		}
	}
	tryInsert := func(name string, mutate func(*factory.Assignment), minutes int, deadline time.Time) {
		t.Helper()
		candidate := a
		cloned := *a.Subject.Completion
		candidate.Subject.Completion = &cloned
		mutate(&candidate)
		tx, err := db.db.BeginTx(ctx, nil)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = tx.Rollback() }()
		if err = insertReassessmentAttemptAllowanceTx(ctx, tx, candidate, minutes, deadline, now); err == nil {
			t.Fatalf("%s accepted", name)
		}
	}
	insertReassessmentRootRow(t, ctx, db, a)
	goodDeadline := now.Add(30 * time.Minute)
	tryInsert("minutes above bound", func(c *factory.Assignment) {}, 121, goodDeadline)
	tryInsert("minutes negative", func(c *factory.Assignment) {}, -5, goodDeadline)
	tryInsert("deadline not after now", func(c *factory.Assignment) {}, 45, now)
	tryInsert("deadline beyond remaining", func(c *factory.Assignment) {}, 45, now.Add(10*time.Hour))
	tryInsert("parented root", func(c *factory.Assignment) { c.ParentAssignment = factory.NewID() }, 45, goodDeadline)
	tryInsert("publishing root", func(c *factory.Assignment) { c.PublicationAssignment = factory.NewID() }, 45, goodDeadline)
	tryInsert("wrong phase", func(c *factory.Assignment) { c.Phase = project.ExecutionImplement }, 45, goodDeadline)
	tryInsert("wrong role", func(c *factory.Assignment) { c.Role = project.RoleReviewer }, 45, goodDeadline)
	tryInsert("positive issue", func(c *factory.Assignment) { c.Issue = 3 }, 45, goodDeadline)
	tryInsert("subject mismatch", func(c *factory.Assignment) { c.Subject.Completion.CycleID = factory.NewID() }, 45, goodDeadline)
	// A deadline beyond the admitted remaining time exhausts deterministically.
	beyondTx, err := db.db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err = insertReassessmentAttemptAllowanceTx(ctx, beyondTx, a, 45, now.Add(10*time.Hour), now); !errors.Is(err, factory.ErrAttemptTimeExhausted) {
		_ = beyondTx.Rollback()
		t.Fatalf("deadline beyond remaining = %v", err)
	}
	_ = beyondTx.Rollback()
	// A valid assignment with no recorded row refuses before insertion.
	unrecorded := reassessmentRootFixture(t, now, 7, factory.NewID(), factory.NewID(), strings.Repeat("d", 64))
	tryInsertOn := func() {
		t.Helper()
		tx, err := db.db.BeginTx(ctx, nil)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = tx.Rollback() }()
		if err = insertReassessmentAttemptAllowanceTx(ctx, tx, unrecorded, 45, goodDeadline, now); err == nil {
			t.Fatal("unrecorded root accepted")
		}
	}
	tryInsertOn()
	// Out-of-bound explicit minutes refuse even on a recorded root.
	tx, err := db.db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback() }()
	if err = insertReassessmentAttemptAllowanceTx(ctx, tx, a, 200, goodDeadline, now); err == nil {
		t.Fatal("two-hundred minutes accepted")
	}
}

func TestReassessmentLifecyclePreservesConsumption(t *testing.T) {
	ctx := context.Background()
	db := dispatchStoreFixture(t)
	now := time.Now().UTC()
	a := reassessmentRootFixture(t, now, 7, factory.NewID(), factory.NewID(), strings.Repeat("d", 64))
	insertReassessmentRootRow(t, ctx, db, a)
	commitReassessmentInsert(t, ctx, db, a, 45, now.Add(30*time.Minute), now)
	// A no-op close on a root with remaining time preserves the record.
	if closed, err := db.CloseExpiredAttempt(ctx, a.ID); err != nil || closed {
		t.Fatalf("fresh completion root closed = %v, %v", closed, err)
	}
	// The generic ROOT-scoped checkpoint joint consumes the valid
	// completion record: five active minutes accrue, nothing else moves.
	tx, err := db.db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err = transitionAttemptRootTx(ctx, tx, 7, 0, a.ID, false, now.Add(5*time.Minute)); err != nil {
		_ = tx.Rollback()
		t.Fatal(err)
	}
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
	checkpointed, err := db.ReassessmentAttemptAllowance(ctx, 7, reassessmentSubjectOf(a), a.ID)
	if err != nil || checkpointed.Active || checkpointed.ActiveSeconds != 300 || checkpointed.Revision != 1 ||
		len(checkpointed.Corrections) != 0 || len(checkpointed.Reviews) != 0 || len(checkpointed.DeliveryCalls) != 0 {
		t.Fatalf("checkpoint lost consumption: %+v, %v", checkpointed, err)
	}
	// The generic repository freeze preserves the same consumption while
	// stopping the clock.
	withdrawal, err := db.WithdrawDispatch(ctx, 7, factory.CauseControlPaused, "soda-tester")
	if err != nil {
		t.Fatal(err)
	}
	if err = db.FreezeAttemptAllowances(ctx, 7, withdrawal.Revision, now.Add(5*time.Minute)); err != nil {
		t.Fatal(err)
	}
	frozen, err := db.ReassessmentAttemptAllowance(ctx, 7, reassessmentSubjectOf(a), a.ID)
	if err != nil || frozen.Active || frozen.ActiveSeconds != 300 ||
		*checkpointed.Completion != *frozen.Completion || frozen.Limits != checkpointed.Limits {
		t.Fatalf("freeze lost consumption: %+v, %v", frozen, err)
	}
}

func TestReassessmentScopeImmutableOnCAS(t *testing.T) {
	ctx := context.Background()
	db, _ := postgresFixture(t, nil)
	now := time.Now().UTC()
	a := reassessmentRootFixture(t, now, 7, factory.NewID(), factory.NewID(), strings.Repeat("d", 64))
	insertReassessmentRootRow(t, ctx, db, a)
	commitReassessmentInsert(t, ctx, db, a, 45, now.Add(30*time.Minute), now)
	current, err := db.ReassessmentAttemptAllowance(ctx, 7, reassessmentSubjectOf(a), a.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name   string
		mutate func(*factory.AttemptAllowance)
	}{
		{"admitted minutes", func(x *factory.AttemptAllowance) { x.Limits.ActiveMinutes = 60 }},
		{"forged budget", func(x *factory.AttemptAllowance) { x.Limits.ReviewRounds = 1 }},
		{"swapped subject", func(x *factory.AttemptAllowance) {
			cloned := *x.Completion
			cloned.CycleID = factory.NewID()
			x.Completion = &cloned
		}},
		{"swapped root", func(x *factory.AttemptAllowance) { x.RootAssignment = factory.NewID() }},
	} {
		forged := current
		tc.mutate(&forged)
		forged.Revision++
		tx, err := db.db.BeginTx(ctx, nil)
		if err != nil {
			t.Fatal(err)
		}
		err = saveAttemptAllowanceTx(ctx, tx, forged)
		_ = tx.Rollback()
		if err == nil {
			t.Fatalf("%s saved over immutable scope", tc.name)
		}
	}
	// The untouched record still reads back identically.
	untouched, err := db.ReassessmentAttemptAllowance(ctx, 7, reassessmentSubjectOf(a), a.ID)
	if err != nil || untouched.Revision != 0 || untouched.ActiveSeconds != 0 {
		t.Fatalf("refused CAS changed the record: %+v, %v", untouched, err)
	}
}

func TestReassessmentPositiveIssueAllowanceRegression(t *testing.T) {
	ctx := context.Background()
	db := dispatchStoreFixture(t)
	now := time.Now().UTC()
	a, reservation, run, view := dispatchTestPacket(t, now)
	if err := recordDispatchTestPacket(t, ctx, db, dispatchTestRegistration(a), a, reservation, run, view); err != nil {
		t.Fatal(err)
	}
	allowance, err := db.AttemptAllowance(ctx, a.Repository, a.Issue)
	if err != nil || allowance.RootAssignment != a.ID || allowance.Limits != factory.DefaultAttemptLimits() ||
		allowance.Completion != nil || !allowance.Active {
		t.Fatalf("positive-issue allowance = %+v, %v", allowance, err)
	}
	// Root-scoped lock, checkpoint and CAS save still round-trip positive
	// roots with no completion subject.
	tx, err := db.db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	locked, err := lockAttemptAllowanceRootTx(ctx, tx, a.Repository, a.Issue, a.ID)
	if err != nil {
		_ = tx.Rollback()
		t.Fatal(err)
	}
	advanced, err := locked.Checkpoint(now.Add(time.Minute), true)
	if err != nil {
		_ = tx.Rollback()
		t.Fatal(err)
	}
	advanced.Revision++
	if err = saveAttemptAllowanceTx(ctx, tx, advanced); err != nil {
		_ = tx.Rollback()
		t.Fatal(err)
	}
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
	reread, err := db.AttemptAllowance(ctx, a.Repository, a.Issue)
	if err != nil || reread.ActiveSeconds != 60 || reread.Revision != 1 {
		t.Fatalf("positive-issue round trip = %+v, %v", reread, err)
	}
	// Spending an issue correction on the positive root still works, and
	// the typed completion getter never sees it.
	if err = func() error {
		inner, err := db.db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		defer func() { _ = inner.Rollback() }()
		if err = consumeCorrectionAttemptTx(ctx, inner, a, now.Add(2*time.Minute)); err != nil {
			return err
		}
		return inner.Commit()
	}(); err != nil {
		t.Fatal(err)
	}
	if _, err = db.ReassessmentAttemptAllowance(ctx, a.Repository, factory.CompletionSubject{
		CompletionID: factory.NewID(), CycleID: factory.NewID(), InventoryDigest: strings.Repeat("d", 64),
	}, a.ID); err == nil {
		t.Fatal("typed completion lookup answered a positive-issue root")
	}
}
