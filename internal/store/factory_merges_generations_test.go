package store

import (
	"context"
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
