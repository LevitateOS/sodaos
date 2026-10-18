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

// Typed intent-root packet admission (D01/R11) against real PostgreSQL:
// one decompose fresh root binds its persisted head intent revision,
// funds its time-only allowance in the same packet transaction, and
// refuses unrecorded, superseded and foreign revisions. Every test below
// opens its own owned disposable database; nothing is borrowed.

// decomposeIntentRevisionFixture persists one draft intent revision on the fixture
// repository and returns its intent and revision IDs.
func decomposeIntentRevisionFixture(t *testing.T, db *Store, intentID, revisionID string, owner int64) {
	t.Helper()
	revision := factory.IntentRevision{
		ID: revisionID, Intent: intentID, Owner: owner, Repository: 7,
		Objective: "plan the recorded draft", State: "draft",
		MaxIssues: 10, CreatedUnix: time.Now().UTC().Truncate(time.Second).Unix(),
	}
	if err := db.SaveIntentRevision(context.Background(), revision); err != nil {
		t.Fatalf("intent revision not recorded: %v", err)
	}
}

// decomposeRootPacket builds one intent decompose fresh-root packet:
// repository slot, Issue zero, fresh attempt root, decompose phase,
// coder role, no parent, no publication, and the canonical intent
// subject bound to the persisted head revision.
func decomposeRootPacket(t *testing.T, ctx context.Context, db *Store, now time.Time, intentID, revisionID string, owner int64, source string) (factory.DispatchRegistration, factory.Assignment, factory.Reservation, factory.Run, factory.RunView) {
	t.Helper()
	selection, ok := grantTestPolicy().Roles[project.RoleCoder]
	if !ok {
		t.Fatal("coder fixture role has no policy selection")
	}
	prompt := []byte("intent root packet " + revisionID)
	sum := sha256.Sum256(prompt)
	rootID, runID := factory.NewID(), factory.NewID()
	a := factory.Assignment{
		Authority: reviewCorrectAuthority(t, ctx, db, dispatchTestProjectID(), 7, "conn"),
		ID:        rootID, AttemptRoot: rootID, PublicationAssignment: "", ParentAssignment: "",
		Subject: factory.ExecutionSubject{
			Kind:   factory.SubjectIntent,
			Intent: &factory.IntentSubject{IntentID: intentID, RevisionID: revisionID, ScopeDigest: strings.Repeat("d", 64), OwnerID: owner},
		},
		Phase:     project.ExecutionDecompose,
		ProjectID: dispatchTestProjectID(), ActorID: 7, Role: project.RoleCoder,
		Repository: 7, Issue: 0, NativeRev: 5, Acceptance: "", Preparation: "f" + strings.Repeat("e", 24),
		Harness: selection.Harness + "-" + selection.HarnessVers, HarnessVers: selection.HarnessVers, Model: selection.Model,
		Connection: "conn", SourceCommit: source,
		Prompt: prompt, PromptSHA: hex.EncodeToString(sum[:]),
		Run: runID, RunHistory: []string{runID}, Stage: factory.AssignmentAssigned,
		Attempts: 1, CreatedUnix: now.Unix(),
	}
	if err := a.Validate(); err != nil {
		t.Fatalf("intent root invalid: %v", err)
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

func decomposeTestControl(a factory.Assignment) factory.IssueControl {
	return factory.IssueControl{
		Repository: a.Repository, Issue: a.Issue, Acceptance: a.Acceptance, NativeRev: a.NativeRev,
		Readiness: factory.ReadinessQueued, Reason: factory.ReasonEligible,
		Fingerprint: strings.Repeat("1", 64), Authority: strings.Repeat("2", 64),
	}
}

func readIntentAllowance(t *testing.T, db *Store, root string) factory.AttemptAllowance {
	t.Helper()
	var data []byte
	if err := db.db.QueryRowContext(context.Background(), `SELECT data FROM factory_attempt_allowances WHERE root_assignment=$1`, root).Scan(&data); err != nil {
		t.Fatalf("intent allowance not funded: %v", err)
	}
	var allowance factory.AttemptAllowance
	if err := json.Unmarshal(data, &allowance); err != nil {
		t.Fatal(err)
	}
	return allowance
}

func TestDecomposeRootAdmitsThroughDecomposeGuard(t *testing.T) {
	db := dispatchStoreFixture(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	intentID, revisionID := "d"+strings.Repeat("b", 24), "d"+strings.Repeat("a", 24)
	decomposeIntentRevisionFixture(t, db, intentID, revisionID, 7)
	d, a, r, run, view := decomposeRootPacket(t, ctx, db, now, intentID, revisionID, 7, strings.Repeat("c", 40))
	if err := db.RecordDispatchPacket(ctx, d, decomposeTestControl(a), a, r, run, view); err != nil {
		t.Fatalf("intent root refused: %v", err)
	}
	allowance := readIntentAllowance(t, db, a.ID)
	if allowance.Intent == nil || allowance.Intent.IntentID != intentID || allowance.Intent.OwnerID != 7 {
		t.Fatalf("intent allowance mis-scoped: %+v", allowance.Intent)
	}
	if allowance.Limits.ActiveMinutes != factory.DefaultIntentMinutes || !allowance.Active || allowance.Closed {
		t.Fatalf("intent allowance mis-funded: %+v", allowance.Limits)
	}
}

func TestDecomposeRootRefusesUnrecordedRevision(t *testing.T) {
	db := dispatchStoreFixture(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	d, a, r, run, view := decomposeRootPacket(t, ctx, db, now, "d"+strings.Repeat("b", 24), "d"+strings.Repeat("a", 24), 7, strings.Repeat("c", 40))
	err := db.RecordDispatchPacket(ctx, d, decomposeTestControl(a), a, r, run, view)
	if !errors.Is(err, ErrAttemptRootMismatch) {
		t.Fatalf("unrecorded revision error = %v, want root mismatch", err)
	}
	assertDispatchPacketAbsent(t, db, a)
}

func TestDecomposeRootRefusesForeignBindings(t *testing.T) {
	db := dispatchStoreFixture(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	intentID, revisionID := "d"+strings.Repeat("b", 24), "d"+strings.Repeat("a", 24)
	decomposeIntentRevisionFixture(t, db, intentID, revisionID, 7)
	advanced := factory.IntentRevision{
		ID: "d" + strings.Repeat("c", 24), Intent: intentID, Owner: 7, Repository: 7,
		Predecessor: revisionID, Objective: "plan the recorded draft", State: "draft",
		MaxIssues: 10, CreatedUnix: now.Unix(),
	}
	if err := db.SaveIntentRevision(ctx, advanced); err != nil {
		t.Fatalf("advanced revision not recorded: %v", err)
	}
	cases := map[string]func(*factory.Assignment){
		"parent":      func(a *factory.Assignment) { a.ParentAssignment = factory.NewID() },
		"publication": func(a *factory.Assignment) { a.PublicationAssignment = factory.NewID() },
		"issue":       func(a *factory.Assignment) { a.Issue = 3 },
		"role":        func(a *factory.Assignment) { a.Role = project.RoleReviewer },
		"superseded":  func(a *factory.Assignment) { a.Subject.Intent.RevisionID = revisionID },
		"owner":       func(a *factory.Assignment) { a.Subject.Intent.OwnerID = 9 },
	}
	for name, mutate := range cases {
		d, a, r, run, view := decomposeRootPacket(t, ctx, db, now, intentID, advanced.ID, 7, strings.Repeat("c", 40))
		mutate(&a)
		if err := db.RecordDispatchPacket(ctx, d, decomposeTestControl(a), a, r, run, view); err == nil {
			t.Fatalf("%s binding admitted", name)
		}
		assertDispatchPacketAbsent(t, db, a)
	}
}

func TestDecomposeRootReplayWritesNoRows(t *testing.T) {
	db := dispatchStoreFixture(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	intentID, revisionID := "d"+strings.Repeat("b", 24), "d"+strings.Repeat("a", 24)
	decomposeIntentRevisionFixture(t, db, intentID, revisionID, 7)
	d, a, r, run, view := decomposeRootPacket(t, ctx, db, now, intentID, revisionID, 7, strings.Repeat("c", 40))
	if err := db.RecordDispatchPacket(ctx, d, decomposeTestControl(a), a, r, run, view); err != nil {
		t.Fatalf("first intent packet refused: %v", err)
	}
	if err := db.RecordDispatchPacket(ctx, d, decomposeTestControl(a), a, r, run, view); err == nil {
		t.Fatal("replayed intent packet admitted")
	}
	if count := countReassessAssignmentRows(t, db, a.ID); count != 1 {
		t.Fatalf("replay left %d assignment rows, want exactly one", count)
	}
}
