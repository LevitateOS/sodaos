package store

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
)

// setupImmutableHistoryRow records, binds and finalizes generation one,
// then replaces onto an open generation two: one immutable snapshot
// behind live fields. Every subtest below starts from this row.
func setupImmutableHistoryRow(t *testing.T) (*Store, factory.Merge) {
	t.Helper()
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
	head2 := strings.Repeat("3", 40)
	return db, updateTestMerge(t, db, replaceTestGeneration(finalized, genTestAuth(finalized, 2, head2)))
}

// refuseSameLengthRewrite attempts one ordinary same-generation update
// carrying a rewritten snapshot and proves zero writes: the update
// refuses and the stored revision and history rest untouched.
func refuseSameLengthRewrite(t *testing.T, mutate func(*factory.Merge)) {
	t.Helper()
	db, current := setupImmutableHistoryRow(t)
	if len(current.History) != 1 {
		t.Fatalf("fixture history=%d, want one snapshot", len(current.History))
	}
	before := current
	next := current
	next.History = append([]factory.MergeGenerationSnapshot(nil), current.History...)
	mutate(&next)
	next.Revision++
	if err := db.UpdateMerge(context.Background(), next); !errors.Is(err, ErrMergeConflict) {
		t.Fatalf("same-length history rewrite accepted: %+v", next.History[0])
	}
	got := readTestMerge(t, db, current.PublicationID)
	if got.Revision != before.Revision {
		t.Fatalf("refused rewrite moved revision %d -> %d", before.Revision, got.Revision)
	}
	if !reflect.DeepEqual(got.History, before.History) {
		t.Fatalf("refused rewrite touched stored history: %+v", got.History)
	}
}

// TestMergeSameLengthHistoryImmutable pins the C17 equal-length guard on
// PostgreSQL: an ordinary UpdateMerge that changes ONLY a prior snapshot
// — receipt (022), authorization evidence, or generation tuple — refuses
// with ErrMergeConflict and zero writes.
func TestMergeSameLengthHistoryImmutable(t *testing.T) {
	t.Run("receipt", func(t *testing.T) {
		refuseSameLengthRewrite(t, func(m *factory.Merge) {
			m.History[0].Operation.Receipt = `{"tampered":true}`
		})
	})
	t.Run("authorization evidence", func(t *testing.T) {
		refuseSameLengthRewrite(t, func(m *factory.Merge) {
			m.History[0].Authorization.Generation.ChecksDigest = strings.Repeat("d", 64)
		})
	})
	t.Run("generation tuple", func(t *testing.T) {
		refuseSameLengthRewrite(t, func(m *factory.Merge) {
			m.History[0].Authorization.Generation.Head = strings.Repeat("4", 40)
		})
	})
}

// TestMergeUnchangedHistoryProgresses proves the guard admits ordinary
// work: driving generation two behind an untouched snapshot passes and
// the snapshot rests exactly as stored.
func TestMergeUnchangedHistoryProgresses(t *testing.T) {
	db, current := setupImmutableHistoryRow(t)
	head2 := strings.Repeat("3", 40)
	driven := current
	operationID := factory.MergeOperationIDForGeneration(current.PublicationID, 2)
	driven.Operation = factory.MergeOperation{Kind: factory.OpMerge, Attempts: 1, OperationID: operationID, UpdatedUnix: 1200, Work: genTestIntent(driven, head2, operationID)}
	driven.Revision++
	if err := db.UpdateMerge(context.Background(), driven); err != nil {
		t.Fatalf("ordinary progression behind unchanged history refused: %v", err)
	}
	got := readTestMerge(t, db, current.PublicationID)
	if len(got.History) != 1 || !reflect.DeepEqual(got.History, current.History) {
		t.Fatalf("ordinary progression touched history: %+v", got.History)
	}
	if got.Operation.Work == nil || got.Operation.Work.HeadOID != head2 {
		t.Fatalf("ordinary progression lost its intent: %+v", got.Operation)
	}
}
