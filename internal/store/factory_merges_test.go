package store

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
)

func mergeTestRecord() factory.Merge {
	return factory.Merge{
		Authority:     dispatchTestAuthority(),
		Operation:     factory.MergeOperation{Kind: factory.OpMerge},
		ID:            factory.NewID(),
		PublicationID: factory.NewID(),
		AssignmentID:  factory.NewID(),
		ProjectID:     "p" + strings.Repeat("d", 24),
		Role:          project.RoleCoder,
		Acceptance:    "d" + strings.Repeat("a", 24),
		HeadRef:       "refs/heads/soda/factory/candidate",
		BaseRef:       "refs/heads/main",
		HeadOID:       strings.Repeat("2", 40),
		BaseOID:       strings.Repeat("1", 40),
		Stage:         factory.MergeOpen,
		Repository:    7,
		Issue:         3,
		PRNumber:      9,
		PRID:          11,
		IssueID:       13,
		PRAuthorID:    5,
		ReviewerID:    6,
		CreatedUnix:   1100,
	}
}

func mergeTestIntent(operationID string) *factory.MergeIntent {
	return &factory.MergeIntent{
		OperationID: operationID, AuthRevision: "accepted-revision",
		HeadRef: "refs/heads/soda/factory/candidate", BaseRef: "refs/heads/main",
		HeadOID: strings.Repeat("2", 40), BaseOID: strings.Repeat("1", 40),
		Repository: 7, ActorID: 8, PRNumber: 9, PRID: 11, IssueID: 13,
		PRAuthorID: 5, ReviewerID: 6,
		NativeRev: 12, NotAfter: time.Now().Add(10 * time.Minute).Unix(), AssessmentRevision: 2, ReviewID: 21,
	}
}

func TestRecordMergeRoundTrip(t *testing.T) {
	db := publicationStoreFixture(t)
	ctx := context.Background()
	m := mergeTestRecord()
	if err := db.RecordMerge(ctx, m); err != nil {
		t.Fatalf("record: %v", err)
	}
	got, err := db.MergeByPublication(ctx, m.PublicationID)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if got.ID != m.ID || got.Stage != factory.MergeOpen {
		t.Fatalf("round trip differs: %+v", got)
	}
	if err := db.RecordMerge(ctx, m); err == nil {
		t.Fatal("second merge for one publication accepted")
	}
	if _, err := db.MergeByPublication(ctx, factory.NewID()); !errors.Is(err, ErrNotFound) {
		t.Fatalf("absent merge: %v", err)
	}
}

func seedMergeAssignment(t *testing.T, db *Store, m factory.Merge) {
	t.Helper()
	p := publicationTestRecord()
	p.AssignmentID, p.Repository, p.Issue, p.ProjectID, p.Acceptance, p.Authority = m.AssignmentID, m.Repository, m.Issue, m.ProjectID, m.Acceptance, m.Authority
	seedPublicationAssignment(t, db, p)
}

func TestUpdateMergeRegistersOnce(t *testing.T) {
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
	m.Revision++
	m.Operation = factory.MergeOperation{Kind: factory.OpMerge, Attempts: 1, OperationID: "soda-x-merge-1", UpdatedUnix: 1200, Work: mergeTestIntent("soda-x-merge-1")}
	if err := db.UpdateMerge(ctx, m); err != nil {
		t.Fatalf("register: %v", err)
	}
	stale := m
	stale.Revision = 1
	if err := db.UpdateMerge(ctx, stale); !errors.Is(err, ErrStaleRevision) {
		t.Fatalf("stale update: %v", err)
	}
	m.Revision++
	m.ReviewerID = 9
	if err := db.UpdateMerge(ctx, m); !errors.Is(err, ErrMergeConflict) {
		t.Fatalf("rewritten intent: %v", err)
	}
}

func TestUpdateMergeGateRefusesWithdrawnDispatch(t *testing.T) {
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
	if _, err := db.WithdrawDispatch(ctx, 7, "test withdrawal", "soda-tester"); err != nil {
		t.Fatal(err)
	}
	m.Revision++
	m.Operation = factory.MergeOperation{Kind: factory.OpMerge, Attempts: 1, OperationID: "soda-x-merge-1", UpdatedUnix: 1200, Work: mergeTestIntent("soda-x-merge-1")}
	if err := db.UpdateMerge(ctx, m); !errors.Is(err, ErrDispatchClosed) {
		t.Fatalf("registration past withdrawal: %v", err)
	}
}

func TestMergeListings(t *testing.T) {
	db := publicationStoreFixture(t)
	ctx := context.Background()
	first, second := mergeTestRecord(), mergeTestRecord()
	second.Issue = 4
	for _, m := range []factory.Merge{first, second} {
		if err := db.RecordMerge(ctx, m); err != nil {
			t.Fatal(err)
		}
	}
	open, err := db.OpenMerges(ctx, 10)
	if err != nil || len(open) != 2 {
		t.Fatalf("open: %+v %v", open, err)
	}
	repository, err := db.OutstandingMerges(ctx, 7, 10)
	if err != nil || len(repository) != 2 {
		t.Fatalf("outstanding: %+v %v", repository, err)
	}
	other, err := db.OutstandingMerges(ctx, 8, 10)
	if err != nil || len(other) != 0 {
		t.Fatalf("other repository: %+v %v", other, err)
	}
	latest, err := db.MergeForIssue(ctx, 7, 3)
	if err != nil || latest.PublicationID != first.PublicationID {
		t.Fatalf("for issue: %+v %v", latest, err)
	}
	if _, err := db.IssueMergeCompletion(ctx, 7, 3); !errors.Is(err, ErrNotFound) {
		t.Fatalf("completion without a merged merge: %v", err)
	}
}

func prepareMergedTestUpdate(t *testing.T, db *Store) factory.Merge {
	t.Helper()
	ctx := context.Background()
	m := mergeTestRecord()
	seedMergeAssignment(t, db, m)
	if err := db.RecordMerge(ctx, m); err != nil {
		t.Fatal(err)
	}
	m.Revision++
	now := time.Now().Unix()
	operationID := factory.MergeOperationID(m.PublicationID, 1)
	m.Operation = factory.MergeOperation{Kind: factory.OpMerge, Attempts: 1, OperationID: operationID, UpdatedUnix: now, Work: mergeTestIntent(operationID)}
	if err := db.UpdateMerge(ctx, m); err != nil {
		t.Fatal("register merge intent:", err)
	}
	m.Revision++
	m.Stage, m.Outcome, m.Reason = factory.MergeMerged, factory.Succeeded, factory.MergeReasonMerged
	m.FinishedUnix, m.MergedUnix, m.ClosedUnix = now, now, now
	m.MergedCommit = m.HeadOID
	m.Operation.Effect = factory.OpEffectCommitted
	m.Operation.Completion = factory.OpCompletionComplete
	m.Operation.MergedCommit = m.HeadOID
	m.Operation.PRNumber, m.Operation.PRID, m.Operation.IssueID = m.PRNumber, m.PRID, m.IssueID
	m.Operation.HeadRef, m.Operation.BaseRef = m.HeadRef, m.BaseRef
	m.Operation.HeadOID, m.Operation.BaseOID = m.HeadOID, m.BaseOID
	return m
}

func TestNewlyMergedCompletionEnqueuesReadinessRootAtomically(t *testing.T) {
	ctx := context.Background()
	t.Run("success", func(t *testing.T) {
		db := publicationStoreFixture(t)
		m := prepareMergedTestUpdate(t, db)
		before := readinessGeneration(t, db)
		if err := db.UpdateMerge(ctx, m); err != nil {
			t.Fatal("finish merge:", err)
		}
		work, err := db.ReadinessWork(ctx, "root:7/3")
		if err != nil || work.Root != (factory.DependenceRef{Repository: 7, Issue: 3}) || work.Generation != before || readinessGeneration(t, db) != before+1 {
			t.Fatal("new merge completion did not coalesce and invalidate prior root:", work, before, readinessGeneration(t, db), err)
		}
		resumed, beginErr := db.BeginReadinessWork(ctx, work.ID, time.Now())
		if beginErr != nil || resumed.Generation != readinessGeneration(t, db) || resumed.RootChanged {
			t.Fatal("new merge evidence did not fence prior root progress:", resumed, beginErr)
		}
		if err = db.UpdateMerge(ctx, m); !errors.Is(err, ErrStaleRevision) || readinessGeneration(t, db) != before+1 {
			t.Fatal("stale terminal merge replay advanced generation:", err, readinessGeneration(t, db))
		}
	})
	t.Run("source-capacity-rolls-back-merge", func(t *testing.T) {
		db := publicationStoreFixture(t)
		m := prepareMergedTestUpdate(t, db)
		// Earlier head-event processing is outside this completion fixture.
		// Remove its task-owned header so completion needs a new source.
		if _, err := db.db.ExecContext(ctx, `DELETE FROM factory_readiness_sources WHERE id='root:7/3'`); err != nil {
			t.Fatal(err)
		}
		fillReadinessSourceHeaders(t, db, "capacity:merge", maxReadinessSources)
		before := readinessGeneration(t, db)
		if err := db.UpdateMerge(ctx, m); !errors.Is(err, ErrReadinessCapacity) {
			t.Fatal("full source table must refuse merged transition:", err)
		}
		stored, err := db.MergeByPublication(ctx, m.PublicationID)
		if err != nil || stored.Stage != factory.MergeOpen || stored.Revision != m.Revision-1 || readinessGeneration(t, db) != before {
			t.Fatal("refused merge partially committed:", stored.Stage, stored.Revision, readinessGeneration(t, db), err)
		}
	})
}
