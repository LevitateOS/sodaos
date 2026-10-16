package factory

import (
	"reflect"
	"strings"
	"testing"
)

func testDeliveryReceipt(t *testing.T, reviewsFit bool) (DeliveryFingerprintInput, string, DeliveryReceipt) {
	t.Helper()
	in := testDeliveryFingerprintInput()
	fingerprint, err := DeliveryFingerprint(in)
	if err != nil {
		t.Fatal(err)
	}
	answer := testDeliveryAnswer(fingerprint)
	receipt, err := NewDeliveryReceipt(in, fingerprint, answer, reviewsFit, 41)
	if err != nil {
		t.Fatal(err)
	}
	return in, fingerprint, receipt
}

func TestDeliveryOptionsAreExactlyTwo(t *testing.T) {
	options := DeliveryOptions()
	if len(options) != 2 {
		t.Fatalf("options=%d, want exactly review_again and ready_to_merge", len(options))
	}
	if _, ok := options[DeliveryReviewAgain]; !ok {
		t.Fatal("review_again option missing")
	}
	if _, ok := options[DeliveryReadyToMerge]; !ok {
		t.Fatal("ready_to_merge option missing")
	}
	if strings.TrimSpace(DeliveryRubric) == "" {
		t.Fatal("delivery rubric is empty")
	}
}

func TestDeliveryReceiptBindsEvidence(t *testing.T) {
	in, fingerprint, receipt := testDeliveryReceipt(t, true)
	if err := receipt.Validate(); err != nil {
		t.Fatalf("decided receipt refused: %v", err)
	}
	if !receipt.CurrentFor(fingerprint) {
		t.Fatal("current receipt not reusable for its fingerprint")
	}
	if receipt.Decision.Effective != DeliveryEffectiveReady {
		t.Fatalf("effective=%q, want ready_to_merge", receipt.Decision.Effective)
	}
	// Any stale semantic evidence invalidates the receipt: a changed check
	// digest yields a different fingerprint the receipt is not current for.
	stale := in
	stale.CheckDigest = strings.Repeat("0", 64)
	other, err := DeliveryFingerprint(stale)
	if err != nil || other == fingerprint {
		t.Fatal("changed check evidence kept its fingerprint")
	}
	if receipt.CurrentFor(other) {
		t.Fatal("stale evidence reused a prior decision")
	}
	// Tampering with the bound input breaks the receipt itself.
	tampered := receipt
	tampered.Input.CheckDigest = strings.Repeat("0", 64)
	if err := tampered.Validate(); err == nil {
		t.Fatal("tampered evidence kept a valid receipt")
	}
	// A decision over different evidence is not recordable.
	foreign := testDeliveryAnswer(strings.Repeat("0", 64))
	if _, err := NewDeliveryReceipt(in, fingerprint, foreign, true, 41); err == nil {
		t.Fatal("answer over different evidence recorded")
	}
}

func TestDeliveryBothChoicesExercised(t *testing.T) {
	_, _, ready := testDeliveryReceipt(t, true)
	if ready.Decision.Effective != DeliveryEffectiveReady {
		t.Fatalf("ready routed %q", ready.Decision.Effective)
	}
	in := testDeliveryFingerprintInput()
	fingerprint, err := DeliveryFingerprint(in)
	if err != nil {
		t.Fatal(err)
	}
	again := testDeliveryAnswer(fingerprint)
	again.Selected = DeliveryReviewAgain
	again.Probabilities = map[string]float64{DeliveryReviewAgain: 0.9, DeliveryReadyToMerge: 0.1}
	review, err := NewDeliveryReceipt(in, fingerprint, again, true, 41)
	if err != nil {
		t.Fatal(err)
	}
	if review.Decision.Effective != DeliveryEffectiveReview {
		t.Fatalf("review routed %q", review.Decision.Effective)
	}
	exhausted, err := NewDeliveryReceipt(in, fingerprint, again, false, 41)
	if err != nil {
		t.Fatal(err)
	}
	if exhausted.Decision.Effective != DeliveryEffectiveExhausted {
		t.Fatalf("exhausted routed %q", exhausted.Decision.Effective)
	}
}

func TestDeliveryUnavailableCarriesNoDecision(t *testing.T) {
	in := testDeliveryFingerprintInput()
	fingerprint, err := DeliveryFingerprint(in)
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := NewDeliveryUnavailable(in, fingerprint, "consultation_failed", 41)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.CurrentFor(fingerprint) {
		t.Fatal("unavailable receipt reusable as a decision")
	}
	if _, err := NewDeliveryUnavailable(in, fingerprint, "", 41); err == nil {
		t.Fatal("reasonless failure recorded")
	}
	_, _, decided := testDeliveryReceipt(t, true)
	smeared := decided
	smeared.Outcome = DeliveryOutcomeUnavailable
	if err := smeared.Validate(); err == nil {
		t.Fatal("unavailable receipt carrying a decision accepted")
	}
}

func TestDeliveryExpectedResolutionReadbackAddsNoCall(t *testing.T) {
	_, fingerprint, receipt := testDeliveryReceipt(t, true)
	advanced, err := receipt.ConfirmExpectedResolution(42)
	if err != nil {
		t.Fatalf("expected readback refused: %v", err)
	}
	if advanced.ObservedNativeRevision != 42 {
		t.Fatalf("revision=%d, want 42", advanced.ObservedNativeRevision)
	}
	if !advanced.CurrentFor(fingerprint) {
		t.Fatal("exact expected readback invalidated its acceptance")
	}
	// The readback is pure evidence handling: fingerprint, decision and
	// outcome are untouched, so no funded call could have been consumed.
	if advanced.Fingerprint != receipt.Fingerprint || !reflect.DeepEqual(advanced.Decision, receipt.Decision) || advanced.Outcome != receipt.Outcome {
		t.Fatal("readback changed the acceptance it only observed")
	}
	if _, err := receipt.ConfirmExpectedResolution(40); err == nil {
		t.Fatal("backwards readback advanced")
	}
	unavailable, err := NewDeliveryUnavailable(receipt.Input, fingerprint, "consultation_failed", 41)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := unavailable.ConfirmExpectedResolution(42); err == nil {
		t.Fatal("unavailable judgment confirmed a resolution")
	}
}
