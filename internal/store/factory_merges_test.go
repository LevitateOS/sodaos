package store

import (
	"context"
	"errors"
	"strings"
	"testing"

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
		NativeRev: 12, NotAfter: 1900, AssessmentRevision: 2, ReviewID: 21,
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
