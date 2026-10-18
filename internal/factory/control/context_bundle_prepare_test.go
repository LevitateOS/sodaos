package control

import (
	"context"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
)

// TestPrepareContextBundleSelectionRecordsFundedChoice proves the
// preparation-time selection: one funded Choice per capability records
// its receipt, and a second preparation reuses the receipt without a
// new consultation. Requires the disposable PG fixture like the other
// producer-lane tests.
func TestPrepareContextBundleSelectionRecordsFundedChoice(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	db := producerTestDB(t)
	db.PriorityCallBudget = 10
	assignment, run := producerDispatch(t, ctx, db)

	chooser := &bundleChooser{selected: factory.ContextBundleCurrent, confidence: 0.9}
	bundle, reason, err := PrepareContextBundleSelection(ctx, db, db, chooser, assignment, run)
	if err != nil || bundle != factory.ContextBundleCurrent || reason != "" {
		t.Fatalf("bundle=%q reason=%q err=%v", bundle, reason, err)
	}
	if chooser.calls != 1 {
		t.Fatalf("calls=%d, want exactly one funded consultation", chooser.calls)
	}
	again, _, err := PrepareContextBundleSelection(ctx, db, db, chooser, assignment, run)
	if err != nil || again != factory.ContextBundleCurrent || chooser.calls != 1 {
		t.Fatalf("reuse bundle=%q calls=%d err=%v", again, chooser.calls, err)
	}
}

// TestPrepareContextBundleSelectionHoldsWhenChooserDown proves a
// failed consultation holds with attention and never fails the
// launch it prepares.
func TestPrepareContextBundleSelectionHoldsWhenChooserDown(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	db := producerTestDB(t)
	db.PriorityCallBudget = 10
	assignment, run := producerDispatch(t, ctx, db)

	chooser := &bundleChooser{fail: true}
	bundle, _, err := PrepareContextBundleSelection(ctx, db, db, chooser, assignment, run)
	if err == nil || bundle != factory.ContextBundleSnapshotHold {
		t.Fatalf("bundle=%q err=%v", bundle, err)
	}
}

// TestPrepareContextBundleSelectionRefusesWithoutInputs proves the
// nil guard without any fixture: no store, journal or chooser means
// hold, never a consultation.
func TestPrepareContextBundleSelectionRefusesWithoutInputs(t *testing.T) {
	chooser := &bundleChooser{selected: factory.ContextBundleCurrent, confidence: 0.9}
	if _, _, err := PrepareContextBundleSelection(context.Background(), nil, nil, chooser, factory.Assignment{}, factory.Run{}); err == nil {
		t.Fatal("nil store prepared a selection")
	}
	if _, _, err := PrepareContextBundleSelection(context.Background(), nil, nil, nil, factory.Assignment{}, factory.Run{}); err == nil {
		t.Fatal("nil chooser prepared a selection")
	}
	if chooser.calls != 0 {
		t.Fatal("missing inputs attempted a consultation")
	}
}
