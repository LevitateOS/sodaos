package store

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
)

func deliveryStoreInput(round int) factory.DeliveryFingerprintInput {
	return factory.DeliveryFingerprintInput{
		Reviews:        []string{strings.Repeat("1", 32)},
		Conversations:  []string{strings.Repeat("2", 64)},
		Constraints:    []string{"protected-files:docs/public"},
		AttemptRoot:    strings.Repeat("3", 32),
		PublicationID:  strings.Repeat("4", 32),
		ScopeDigest:    strings.Repeat("5", 64),
		FixDigest:      strings.Repeat("6", 64),
		DiffDigest:     strings.Repeat("7", 64),
		CheckDigest:    strings.Repeat("8", 64),
		TreeDigest:     strings.Repeat("9", 64),
		Head:           strings.Repeat("a", 40),
		Base:           strings.Repeat("b", 40),
		Model:          "jev-test",
		Rubric:         "ready_to_merge only with full evidence cover",
		PolicyRevision: 3,
		Round:          round,
	}
}

func deliveryStoreAnswer(fingerprint string) factory.DeliveryAnswer {
	return factory.DeliveryAnswer{
		Probabilities: map[string]float64{factory.DeliveryReviewAgain: 0.1, factory.DeliveryReadyToMerge: 0.9},
		Fingerprint:   fingerprint,
		Selected:      factory.DeliveryReadyToMerge,
		Model:         "jev-test",
		Probability:   0.9,
		Confidence:    0.85,
	}
}

func deliveryStoreFixture(t *testing.T) (*Store, factory.Assignment) {
	t.Helper()
	db := dispatchStoreFixture(t)
	now := time.Now().UTC()
	a, reservation, run, view := dispatchTestPacket(t, now)
	if err := recordDispatchTestPacket(t, context.Background(), db, dispatchTestRegistration(a), a, reservation, run, view); err != nil {
		t.Fatal(err)
	}
	return db, a
}

func TestDeliveryJudgmentReserveRecordRead(t *testing.T) {
	db, a := deliveryStoreFixture(t)
	ctx := context.Background()
	in := deliveryStoreInput(1)
	fingerprint, err := factory.DeliveryFingerprint(in)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.DeliveryJudgment(ctx, fingerprint); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unreserved read error=%v", err)
	}
	ok, err := db.ReserveDeliveryJudgment(ctx, a.Repository, a.Issue, a.AttemptRoot, fingerprint)
	if err != nil || !ok {
		t.Fatalf("reserve=%v err=%v", ok, err)
	}
	// A call that left no receipt retries under the same fingerprint.
	ok, err = db.ReserveDeliveryJudgment(ctx, a.Repository, a.Issue, a.AttemptRoot, fingerprint)
	if err != nil || !ok {
		t.Fatalf("retry reserve=%v err=%v", ok, err)
	}
	receipt, err := factory.NewDeliveryReceipt(in, fingerprint, deliveryStoreAnswer(fingerprint), true, 41)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.RecordDeliveryJudgment(ctx, receipt); err != nil {
		t.Fatalf("record: %v", err)
	}
	// A recorded receipt is reused instead of asking again.
	if ok, err = db.ReserveDeliveryJudgment(ctx, a.Repository, a.Issue, a.AttemptRoot, fingerprint); err != nil || ok {
		t.Fatalf("decided reserve=%v err=%v", ok, err)
	}
	if err := db.RecordDeliveryJudgment(ctx, receipt); !errors.Is(err, ErrCommandConflict) {
		t.Fatalf("double record error=%v", err)
	}
	got, err := db.DeliveryJudgment(ctx, fingerprint)
	if err != nil || !got.CurrentFor(fingerprint) || got.Decision.Effective != factory.DeliveryEffectiveReady {
		t.Fatalf("read=%+v err=%v", got, err)
	}
	// Recording without a reservation conflicts instead of inventing funding.
	other := deliveryStoreInput(2)
	otherPrint, err := factory.DeliveryFingerprint(other)
	if err != nil {
		t.Fatal(err)
	}
	unreserved, err := factory.NewDeliveryReceipt(other, otherPrint, deliveryStoreAnswer(otherPrint), true, 41)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.RecordDeliveryJudgment(ctx, unreserved); !errors.Is(err, ErrCommandConflict) {
		t.Fatalf("unreserved record error=%v", err)
	}
}

func TestDeliveryJudgmentAllowsIndependentInputs(t *testing.T) {
	db, a := deliveryStoreFixture(t)
	ctx := context.Background()
	first, err := factory.DeliveryFingerprint(deliveryStoreInput(1))
	if err != nil {
		t.Fatal(err)
	}
	ok, err := db.ReserveDeliveryJudgment(ctx, a.Repository, a.Issue, a.AttemptRoot, first)
	if err != nil || !ok {
		t.Fatalf("first reserve=%v err=%v", ok, err)
	}
	second, err := factory.DeliveryFingerprint(deliveryStoreInput(2))
	if err != nil {
		t.Fatal(err)
	}
	if ok, err := db.ReserveDeliveryJudgment(ctx, a.Repository, a.Issue, a.AttemptRoot, second); err != nil || !ok {
		t.Fatalf("independent reserve=%v err=%v", ok, err)
	}
}
