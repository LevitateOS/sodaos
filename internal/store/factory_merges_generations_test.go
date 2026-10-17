package store

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
)

// genTestAuth builds one valid non-union generation authorization for a
// row's exact head and base. It stands in for a fresh review and
// judgment generation behind the store-level replacement contract.
func genTestAuth(m factory.Merge, n int, head string) factory.MergeAuthorization {
	fingerprint := strings.Repeat("e", 64)
	answer := factory.DeliveryAnswer{
		Probabilities: map[string]float64{factory.DeliveryReviewAgain: 0.1, factory.DeliveryReadyToMerge: 0.9},
		Fingerprint:   fingerprint,
		Selected:      factory.DeliveryReadyToMerge,
		Model:         "jev-test",
		Probability:   0.9,
		Confidence:    0.85,
	}
	generation := factory.MergeGeneration{
		Authority: m.Authority, PublicationID: m.PublicationID,
		DecisionID:        strings.Repeat("d", 32),
		NativeOperationID: factory.MergeGenerationOperationID(m.PublicationID, n),
		Fingerprint:       fingerprint, Head: head, Base: m.BaseOID, DiffBase: m.BaseOID,
		ChecksDigest:       strings.Repeat("c", 64),
		Generation:         n,
		ReviewRound:        1,
		AssessmentRevision: 2, ObservedNativeRev: 12, NotAfter: 1300,
	}
	return factory.MergeAuthorization{
		Generation:           generation,
		Decision:             factory.DeliveryDecision{Answer: answer, Effective: factory.DeliveryEffectiveReady, ReviewsFit: true},
		ResolutionsConfirmed: true,
	}
}

// genTestIntent builds one driven intent for a generation's exact head
// and derived operation identity.
func genTestIntent(m factory.Merge, head, operationID string) *factory.MergeIntent {
	return &factory.MergeIntent{
		OperationID: operationID, AuthRevision: "accepted-revision",
		HeadRef: m.HeadRef, BaseRef: m.BaseRef,
		HeadOID: head, BaseOID: m.BaseOID,
		Repository: m.Repository, ActorID: 8, PRNumber: m.PRNumber, PRID: m.PRID, IssueID: m.IssueID,
		PRAuthorID: m.PRAuthorID, ReviewerID: m.ReviewerID,
		NativeRev: 12, NotAfter: time.Now().Add(10 * time.Minute).Unix(), AssessmentRevision: 2, ReviewID: 21,
	}
}

func readTestMerge(t *testing.T, db *Store, publicationID string) factory.Merge {
	t.Helper()
	m, err := db.MergeByPublication(context.Background(), publicationID)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func updateTestMerge(t *testing.T, db *Store, m factory.Merge) factory.Merge {
	t.Helper()
	m.Revision++
	if err := db.UpdateMerge(context.Background(), m); err != nil {
		t.Fatal(err)
	}
	return readTestMerge(t, db, m.PublicationID)
}

// driveTestGeneration records one generation's intent, observes its
// confirmed cancellation and finishes it withdrawn: the finalized
// predecessor a replacement must snapshot exactly.
func driveTestGeneration(t *testing.T, db *Store, m factory.Merge, head string) factory.Merge {
	t.Helper()
	n := 1
	if m.Authorization.Bound() {
		n = m.Authorization.Generation.Generation
	}
	operationID := factory.MergeOperationIDForGeneration(m.PublicationID, n)
	m.Operation = factory.MergeOperation{Kind: factory.OpMerge, Attempts: 1, OperationID: operationID, UpdatedUnix: 1200, Work: genTestIntent(m, head, operationID)}
	m = updateTestMerge(t, db, m)
	now := time.Now().Unix()
	m.Operation.Effect, m.Operation.Cancellation = factory.OpEffectNotCommitted, factory.OpCancelCancelled
	m.Operation.Reason, m.Operation.Receipt, m.Operation.UpdatedUnix = "cancelled_before_admission", `{"cancelled":true}`, now
	m = updateTestMerge(t, db, m)
	m.Stage, m.Outcome, m.Reason, m.FinishedUnix = factory.MergeWithdrawn, factory.Cancelled, factory.MergeReasonWithdrawn, now
	return updateTestMerge(t, db, m)
}

// replaceTestGeneration builds the exact replacement onto a fresh
// generation: immutable prefix, exact predecessor snapshot, next
// generation number, fresh tuple and authority, undriven operation and
// cleared admission.
func replaceTestGeneration(m factory.Merge, auth factory.MergeAuthorization) factory.Merge {
	next := m
	next.History = append(append([]factory.MergeGenerationSnapshot(nil), m.History...),
		factory.SnapshotMergeGeneration(m.Authorization, m.Operation, m.Stage, m.Outcome, m.Reason, m.FinishedUnix))
	next.Authorization = auth
	next.Operation = factory.MergeOperation{Kind: factory.OpMerge}
	next.Authority = auth.Generation.Authority
	next.HeadOID, next.BaseOID = auth.Generation.Head, auth.Generation.Base
	next.WithdrawRequested = false
	next.Stage, next.Outcome, next.Reason, next.FinishedUnix = factory.MergeOpen, "", "", 0
	next.MergedCommit, next.MergedUnix, next.ClosedUnix = "", 0, 0
	next.OriginalClosingOccurrence, next.OriginalClosedUnix = "", 0
	next.NativeRev, next.ObservedUnix = 0, 0
	return next
}

// TestMergeTwoGenerationSamePRFlow drives the real two-generation
// same-PR history on PostgreSQL: intent, confirmed cancel, withdrawal,
// then a fresh generation. The predecessor snapshots exactly once, the
// PR identity never moves, the receipt survives in history and the new
// generation drives its own derived operation identity.
func TestMergeTwoGenerationSamePRFlow(t *testing.T) {
	db := publicationStoreFixture(t)
	ctx := context.Background()
	m := mergeTestRecord()
	seedMergeAssignment(t, db, m)
	if err := db.EnqueueReadinessWork(ctx, "root:7/3", "", factory.DependenceRef{Repository: 7, Issue: 3}); err != nil {
		t.Fatal("seed prior root for completion fence:", err)
	}
	if err := db.RecordMerge(ctx, m); err != nil {
		t.Fatal(err)
	}
	head1 := m.HeadOID
	stored := readTestMerge(t, db, m.PublicationID)
	stored.Authorization = mergeTestAuthorization(stored)
	stored = updateTestMerge(t, db, stored)
	finalized := driveTestGeneration(t, db, stored, head1)
	if finalized.Stage != factory.MergeWithdrawn || finalized.Operation.Effect != factory.OpEffectNotCommitted {
		t.Fatalf("predecessor not finalized: %+v", finalized)
	}
	head2 := strings.Repeat("3", 40)
	auth2 := genTestAuth(finalized, 2, head2)
	replaced := replaceTestGeneration(finalized, auth2)
	replaced = updateTestMerge(t, db, replaced)
	if len(replaced.History) != 1 || replaced.Authorization.Generation.Generation != 2 {
		t.Fatalf("replacement misplaced: history=%d current=%+v", len(replaced.History), replaced.Authorization.Generation)
	}
	snapshot := replaced.History[0]
	if snapshot.Authorization.Generation.Generation != 1 || snapshot.Operation.OperationID != finalized.Operation.OperationID ||
		snapshot.Operation.Receipt != `{"cancelled":true}` || snapshot.Stage != factory.MergeWithdrawn ||
		snapshot.Outcome != factory.Cancelled || snapshot.Reason != factory.MergeReasonWithdrawn || snapshot.FinishedUnix != finalized.FinishedUnix {
		t.Fatalf("predecessor not snapshotted exactly: %+v", snapshot)
	}
	if replaced.PRNumber != m.PRNumber || replaced.PRID != m.PRID || replaced.IssueID != m.IssueID ||
		replaced.Repository != m.Repository || replaced.Issue != m.Issue || replaced.PublicationID != m.PublicationID {
		t.Fatal("replacement moved its PR identity")
	}
	if replaced.HeadOID != head2 || replaced.Operation.Attempts != 0 || replaced.Operation.Work != nil || replaced.WithdrawRequested {
		t.Fatalf("live fields are not the fresh generation: %+v", replaced)
	}
	// The fresh generation drives its own derived operation identity:
	// a lost reply reconciles by exact lookup, never by resubmission.
	driven := replaced
	operationID := factory.MergeOperationIDForGeneration(replaced.PublicationID, 2)
	if operationID != factory.MergeGenerationOperationID(replaced.PublicationID, 2) {
		t.Fatalf("generation two identity %q differs from its derivation", operationID)
	}
	driven.Operation = factory.MergeOperation{Kind: factory.OpMerge, Attempts: 1, OperationID: operationID, UpdatedUnix: 1200, Work: genTestIntent(driven, head2, operationID)}
	driven = updateTestMerge(t, db, driven)
	if driven.Operation.Work.OperationID != operationID || driven.Operation.Work.HeadOID != head2 {
		t.Fatalf("fresh generation intent misbound: %+v", driven.Operation)
	}
}

// TestMergeReplacementRefusals pins the store-level refusal matrix on
// PostgreSQL: history rewrites and deletions, inexact snapshots,
// unknown and committed predecessors, stale revisions and generation
// gaps all refuse, and a refused replacement changes nothing.
func TestMergeReplacementRefusals(t *testing.T) {
	setup := func(t *testing.T) (*Store, factory.Merge) {
		db := publicationStoreFixture(t)
		ctx := context.Background()
		m := mergeTestRecord()
		seedMergeAssignment(t, db, m)
		if err := db.RecordMerge(ctx, m); err != nil {
			t.Fatal(err)
		}
		stored := readTestMerge(t, db, m.PublicationID)
		stored.Authorization = mergeTestAuthorization(stored)
		stored = updateTestMerge(t, db, stored)
		return db, driveTestGeneration(t, db, stored, stored.HeadOID)
	}
	validReplacement := func(m factory.Merge) factory.Merge {
		return replaceTestGeneration(m, genTestAuth(m, 2, strings.Repeat("3", 40)))
	}
	// drivenOpen binds and drives one generation without finalizing it:
	// the predecessor stays non-terminal for the unknown-predecessor
	// cases.
	drivenOpen := func(t *testing.T) (*Store, factory.Merge) {
		db := publicationStoreFixture(t)
		ctx := context.Background()
		m := mergeTestRecord()
		seedMergeAssignment(t, db, m)
		if err := db.RecordMerge(ctx, m); err != nil {
			t.Fatal(err)
		}
		stored := readTestMerge(t, db, m.PublicationID)
		stored.Authorization = mergeTestAuthorization(stored)
		stored = updateTestMerge(t, db, stored)
		operationID := factory.MergeOperationIDForGeneration(stored.PublicationID, 1)
		stored.Operation = factory.MergeOperation{Kind: factory.OpMerge, Attempts: 1, OperationID: operationID, UpdatedUnix: 1200, Work: genTestIntent(stored, stored.HeadOID, operationID)}
		return db, updateTestMerge(t, db, stored)
	}
	t.Run("rewrite history", func(t *testing.T) {
		db, finalized := setup(t)
		next := validReplacement(finalized)
		next.History[0].FinishedUnix++
		next.Revision++
		if err := db.UpdateMerge(context.Background(), next); !errors.Is(err, ErrMergeConflict) {
			t.Fatal("rewritten history accepted")
		}
	})
	t.Run("delete history", func(t *testing.T) {
		db, finalized := setup(t)
		next := validReplacement(finalized)
		next.History = nil
		next.Authorization = genTestAuth(next, 1, next.HeadOID)
		next.Revision++
		if err := db.UpdateMerge(context.Background(), next); !errors.Is(err, ErrMergeConflict) {
			t.Fatal("deleted history accepted")
		}
	})
	t.Run("inexact snapshot", func(t *testing.T) {
		db, finalized := setup(t)
		next := validReplacement(finalized)
		next.History[0].Operation.Receipt = `{"tampered":true}`
		next.Revision++
		if err := db.UpdateMerge(context.Background(), next); !errors.Is(err, ErrMergeConflict) {
			t.Fatal("inexact snapshot accepted")
		}
	})
	t.Run("unknown predecessor", func(t *testing.T) {
		db, driven := drivenOpen(t)
		pending := driven
		pending.Operation.Effect, pending.Operation.Cancellation = factory.OpEffectPending, factory.OpCancelNone
		pending = updateTestMerge(t, db, pending)
		next := replaceTestGeneration(pending, genTestAuth(pending, 2, strings.Repeat("3", 40)))
		next.Revision++
		// The snapshot of a non-finalized predecessor is itself
		// invalid, so the row refuses before the transition check;
		// either way nothing persists.
		if err := db.UpdateMerge(context.Background(), next); err == nil {
			t.Fatal("pending predecessor authorized replacement")
		}
		if got := readTestMerge(t, db, pending.PublicationID); len(got.History) != 0 {
			t.Fatal("refused replacement changed history")
		}
	})
	t.Run("committed predecessor", func(t *testing.T) {
		db, driven := drivenOpen(t)
		committed := driven
		committed.Operation.Effect, committed.Operation.Cancellation = factory.OpEffectCommitted, factory.OpCancelNone
		committed.Operation.Completion = factory.OpCompletionComplete
		committed.Operation.MergedCommit = committed.HeadOID
		committed.Operation.PRNumber, committed.Operation.PRID, committed.Operation.IssueID = committed.PRNumber, committed.PRID, committed.IssueID
		committed.Operation.HeadRef, committed.Operation.BaseRef = committed.HeadRef, committed.BaseRef
		committed.Operation.HeadOID, committed.Operation.BaseOID = committed.HeadOID, committed.BaseOID
		committed = updateTestMerge(t, db, committed)
		next := replaceTestGeneration(committed, genTestAuth(committed, 2, strings.Repeat("3", 40)))
		next.Revision++
		// A committed predecessor proves nothing replaceable: the row
		// refuses and the committed effect rests untouched for
		// recovery and adoption.
		if err := db.UpdateMerge(context.Background(), next); err == nil {
			t.Fatal("committed predecessor authorized replacement")
		}
		if got := readTestMerge(t, db, committed.PublicationID); len(got.History) != 0 || got.Operation.Effect != factory.OpEffectCommitted {
			t.Fatal("refused replacement touched its committed effect")
		}
	})
	t.Run("stale revision", func(t *testing.T) {
		db, finalized := setup(t)
		next := validReplacement(finalized)
		next.Revision++
		if err := db.UpdateMerge(context.Background(), next); err != nil {
			t.Fatal("first replacement:", err)
		}
		retry := validReplacement(finalized)
		retry.Revision++
		if err := db.UpdateMerge(context.Background(), retry); !errors.Is(err, ErrStaleRevision) {
			t.Fatal("stale replacement accepted")
		}
	})
	t.Run("generation gap", func(t *testing.T) {
		db, finalized := setup(t)
		next := validReplacement(finalized)
		next.Authorization = genTestAuth(next, 3, strings.Repeat("3", 40))
		next.HeadOID = strings.Repeat("3", 40)
		next.Revision++
		if err := db.UpdateMerge(context.Background(), next); err == nil {
			t.Fatal("generation gap accepted")
		}
	})
	t.Run("changed PR identity", func(t *testing.T) {
		db, finalized := setup(t)
		next := validReplacement(finalized)
		next.PRNumber++
		next.Revision++
		if err := db.UpdateMerge(context.Background(), next); !errors.Is(err, ErrMergeConflict) {
			t.Fatal("replacement on a new PR accepted")
		}
	})
}

// TestMergeConcurrentReplacementsRefuse stores one finalized predecessor
// on PostgreSQL and races two replacements: exactly one wins the
// revision compare-and-swap, the loser refuses, and no second native
// effect exists — one history entry, one current generation.
func TestMergeConcurrentReplacementsRefuse(t *testing.T) {
	db := publicationStoreFixture(t)
	ctx := context.Background()
	m := mergeTestRecord()
	seedMergeAssignment(t, db, m)
	if err := db.RecordMerge(ctx, m); err != nil {
		t.Fatal(err)
	}
	stored := readTestMerge(t, db, m.PublicationID)
	stored.Authorization = mergeTestAuthorization(stored)
	stored = updateTestMerge(t, db, stored)
	finalized := driveTestGeneration(t, db, stored, stored.HeadOID)
	first := replaceTestGeneration(finalized, genTestAuth(finalized, 2, strings.Repeat("3", 40)))
	second := replaceTestGeneration(finalized, genTestAuth(finalized, 2, strings.Repeat("4", 40)))
	second.HeadOID = strings.Repeat("4", 40)
	first.Revision++
	second.Revision++
	if err := db.UpdateMerge(ctx, first); err != nil {
		t.Fatal("first replacement:", err)
	}
	if err := db.UpdateMerge(ctx, second); !errors.Is(err, ErrStaleRevision) {
		t.Fatal("concurrent replacement accepted")
	}
	got := readTestMerge(t, db, m.PublicationID)
	if len(got.History) != 1 || got.Authorization.Generation.Generation != 2 || got.HeadOID != strings.Repeat("3", 40) {
		t.Fatalf("concurrent replacement duplicated its effect: %+v", got)
	}
}

// TestMergeReplacementAtCeiling fills the bounded history on PostgreSQL
// and proves the 33rd generation is a visible intervention: refused with
// the ledger intact, never a silent drop or a new table.
func TestMergeReplacementAtCeiling(t *testing.T) {
	db := publicationStoreFixture(t)
	ctx := context.Background()
	m := mergeTestRecord()
	seedMergeAssignment(t, db, m)
	if err := db.RecordMerge(ctx, m); err != nil {
		t.Fatal(err)
	}
	stored := readTestMerge(t, db, m.PublicationID)
	stored.Authorization = mergeTestAuthorization(stored)
	stored = updateTestMerge(t, db, stored)
	current := stored
	headOf := func(n int) string {
		if n == 1 {
			return stored.HeadOID
		}
		return fmt.Sprintf("%040d", 2000+n)
	}
	for n := 1; n <= factory.MaxMergeGenerations; n++ {
		current = driveTestGeneration(t, db, current, headOf(n))
		if n == factory.MaxMergeGenerations {
			break
		}
		nextHead := headOf(n + 1)
		current = updateTestMerge(t, db, replaceTestGeneration(current, genTestAuth(current, n+1, nextHead)))
		if len(current.History) != n || current.Authorization.Generation.Generation != n+1 {
			t.Fatalf("generation %d misplaced: history=%d", n+1, len(current.History))
		}
	}
	// Thirty-one finalized priors plus the finalized 32nd generation
	// fill the ledger: the 33rd registration is a visible intervention,
	// refused with the ledger intact, never a silent drop.
	overflowHead := strings.Repeat("8", 40)
	overflow := replaceTestGeneration(current, genTestAuth(current, factory.MaxMergeGenerations+1, overflowHead))
	overflow.HeadOID = overflowHead
	overflow.Revision++
	if err := db.UpdateMerge(ctx, overflow); err == nil {
		t.Fatal("generation above the history ceiling accepted")
	}
	got := readTestMerge(t, db, m.PublicationID)
	if len(got.History) != factory.MaxMergeGenerations-1 || got.Authorization.Generation.Generation != factory.MaxMergeGenerations {
		t.Fatalf("breach changed its ledger: history=%d current=%+v", len(got.History), got.Authorization.Generation)
	}
}
