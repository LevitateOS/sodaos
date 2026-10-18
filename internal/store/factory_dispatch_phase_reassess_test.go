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

func countReassessAssignmentRows(t *testing.T, db *Store, id string) int {
	t.Helper()
	var count int
	if err := db.db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM factory_assignments WHERE id=$1`, id).Scan(&count); err != nil {
		t.Fatal(err)
	}
	return count
}

// Typed completion-root packet admission (D13/FW32, R11) against real
// PostgreSQL: one reassessment fresh root binds its verified completion
// event and staged open cycle, funds its time-only allowance in the same
// packet transaction, and refuses unrecorded completions, unstaged
// cycles and foreign bindings. Every test below opens its own owned
// disposable database; nothing is borrowed.

// reassessCycleFixture records one verified completion on the fixture
// repository and returns its event ID, first staged cycle ID and merged
// commit.
func reassessCycleFixture(t *testing.T, db *Store) (eventID, cycleID, merged string) {
	t.Helper()
	m := mergedCompletionRecord(factory.NewID(), "501")
	if !recordTestCompletion(t, db, m) {
		t.Fatal("completion not recorded")
	}
	ctx := context.Background()
	row := db.db.QueryRowContext(ctx, `SELECT event,cycle FROM factory_reassessments WHERE repository=$1 ORDER BY generation DESC LIMIT 1`, m.Repository)
	if err := row.Scan(&eventID, &cycleID); err != nil {
		t.Fatalf("staged cycle not recorded: %v", err)
	}
	return eventID, cycleID, m.MergedCommit
}

// reassessRootPacket builds one completion reassessment fresh-root
// packet: repository slot, Issue zero, fresh attempt root, reassess
// phase, coder role, no parent, no publication, and the canonical
// completion subject.
func reassessRootPacket(t *testing.T, ctx context.Context, db *Store, now time.Time, eventID, cycleID, source string) (factory.DispatchRegistration, factory.Assignment, factory.Reservation, factory.Run, factory.RunView) {
	t.Helper()
	selection, ok := grantTestPolicy().Roles[project.RoleCoder]
	if !ok {
		t.Fatal("coder fixture role has no policy selection")
	}
	prompt := []byte("reassessment root packet " + cycleID)
	sum := sha256.Sum256(prompt)
	rootID, runID := factory.NewID(), factory.NewID()
	a := factory.Assignment{
		Authority: reviewCorrectAuthority(t, ctx, db, dispatchTestProjectID(), 7, "conn"),
		ID:        rootID, AttemptRoot: rootID, PublicationAssignment: "", ParentAssignment: "",
		Subject: factory.ExecutionSubject{
			Kind:       factory.SubjectCompletion,
			Completion: &factory.CompletionSubject{CompletionID: eventID, CycleID: cycleID, InventoryDigest: strings.Repeat("c", 64)},
		},
		Phase:     project.ExecutionReassess,
		ProjectID: dispatchTestProjectID(), ActorID: 7, Role: project.RoleCoder,
		Repository: 7, Issue: 0, NativeRev: 5, Acceptance: "", Preparation: "f" + strings.Repeat("e", 24),
		Harness: selection.Harness + "-" + selection.HarnessVers, HarnessVers: selection.HarnessVers, Model: selection.Model,
		Connection: "conn", SourceCommit: source,
		Prompt: prompt, PromptSHA: hex.EncodeToString(sum[:]),
		Run: runID, RunHistory: []string{runID}, Stage: factory.AssignmentAssigned,
		Attempts: 1, CreatedUnix: now.Unix(),
	}
	if err := a.Validate(); err != nil {
		t.Fatalf("reassessment root invalid: %v", err)
	}
	r := factory.Reservation{AssignmentID: a.ID, Repository: 7, Connection: "conn", State: factory.ReservationHeld, PlannedMinutes: 30}
	run := factory.Run{
		ID: runID, ProjectID: a.ProjectID, Role: project.RoleCoder, InputSHA: source,
		Started: now, Deadline: now.Add(30 * time.Minute),
		Image: "sha256:" + strings.Repeat("b", 64), Harness: a.Harness, Model: a.Model,
	}
	view := factory.RunView{RunID: runID, Repository: 7, Issue: 0, Attempt: a.ID}
	d := dispatchTestRegistration(a)
	return d, a, r, run, view
}

func reassessTestControl(a factory.Assignment) factory.IssueControl {
	return factory.IssueControl{
		Repository: a.Repository, Issue: a.Issue, Acceptance: a.Acceptance, NativeRev: a.NativeRev,
		Readiness: factory.ReadinessQueued, Reason: factory.ReasonEligible,
		Fingerprint: strings.Repeat("1", 64), Authority: strings.Repeat("2", 64),
	}
}

func TestReassessRootAdmitsThroughReassessGuard(t *testing.T) {
	db := dispatchStoreFixture(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	eventID, cycleID, merged := reassessCycleFixture(t, db)
	d, a, r, run, view := reassessRootPacket(t, ctx, db, now, eventID, cycleID, merged)
	if err := db.RecordDispatchPacket(ctx, d, reassessTestControl(a), a, r, run, view); err != nil {
		t.Fatalf("reassessment root refused: %v", err)
	}
	allowance, err := db.ReassessmentAttemptAllowance(ctx, 7, *a.Subject.Completion, a.ID)
	if err != nil {
		t.Fatalf("reassessment allowance not funded: %v", err)
	}
	if allowance.Limits.ActiveMinutes != factory.DefaultReassessmentMinutes || !allowance.Active || allowance.Closed {
		t.Fatalf("reassessment allowance mis-funded: %+v", allowance)
	}
}

func TestReassessRootRefusesUnrecordedCompletion(t *testing.T) {
	db := dispatchStoreFixture(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	d, a, r, run, view := reassessRootPacket(t, ctx, db, now, factory.NewID(), factory.NewID(), strings.Repeat("c", 40))
	err := db.RecordDispatchPacket(ctx, d, reassessTestControl(a), a, r, run, view)
	if !errors.Is(err, ErrAttemptRootMismatch) {
		t.Fatalf("unrecorded completion error = %v, want root mismatch", err)
	}
	assertDispatchPacketAbsent(t, db, a)
}

func TestReassessRootRefusesForeignBindings(t *testing.T) {
	db := dispatchStoreFixture(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	eventID, cycleID, merged := reassessCycleFixture(t, db)
	cases := map[string]func(*factory.Assignment){
		"parent":      func(a *factory.Assignment) { a.ParentAssignment = factory.NewID() },
		"publication": func(a *factory.Assignment) { a.PublicationAssignment = factory.NewID() },
		"issue":       func(a *factory.Assignment) { a.Issue = 3 },
		"role":        func(a *factory.Assignment) { a.Role = project.RoleReviewer },
		"cycle":       func(a *factory.Assignment) { a.Subject.Completion.CycleID = factory.NewID() },
	}
	for name, mutate := range cases {
		d, a, r, run, view := reassessRootPacket(t, ctx, db, now, eventID, cycleID, merged)
		mutate(&a)
		if err := db.RecordDispatchPacket(ctx, d, reassessTestControl(a), a, r, run, view); err == nil {
			t.Fatalf("%s binding admitted", name)
		}
		assertDispatchPacketAbsent(t, db, a)
	}
}

func TestReassessRootReplayWritesNoRows(t *testing.T) {
	db := dispatchStoreFixture(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	eventID, cycleID, merged := reassessCycleFixture(t, db)
	d, a, r, run, view := reassessRootPacket(t, ctx, db, now, eventID, cycleID, merged)
	if err := db.RecordDispatchPacket(ctx, d, reassessTestControl(a), a, r, run, view); err != nil {
		t.Fatalf("first reassessment packet refused: %v", err)
	}
	if err := db.RecordDispatchPacket(ctx, d, reassessTestControl(a), a, r, run, view); err == nil {
		t.Fatal("replayed reassessment packet admitted")
	}
	count := countReassessAssignmentRows(t, db, a.ID)
	if count != 1 {
		t.Fatalf("replay left %d assignment rows, want exactly one", count)
	}
}
