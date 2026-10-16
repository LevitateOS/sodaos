package factory

import (
	"testing"
)

// TestDeliveryD08BothChoicesNeedEvidence proves the T44 choice leg: either
// actionable D08 choice requires its selected probability at or above
// 0.80, the returned confidence at or above 0.80, and a strict lead over
// the other option. Uncertainty waits with judgment_attention, never an
// automatic review; review_again additionally needs remaining allowance.
func TestDeliveryD08BothChoicesNeedEvidence(t *testing.T) {
	fingerprint, err := DeliveryFingerprint(testDeliveryFingerprintInput())
	if err != nil {
		t.Fatal(err)
	}
	strong := func(selected string, forAgain float64) DeliveryAnswer {
		probs := map[string]float64{DeliveryReviewAgain: 1 - forAgain, DeliveryReadyToMerge: forAgain}
		if selected == DeliveryReviewAgain {
			probs = map[string]float64{DeliveryReviewAgain: forAgain, DeliveryReadyToMerge: 1 - forAgain}
		}
		return DeliveryAnswer{
			Probabilities: probs, Fingerprint: fingerprint, Selected: selected,
			Model: "jev-test", Probability: forAgain, Confidence: 0.9,
		}
	}
	// Strong review_again with allowance buys exactly one more inspection.
	if got := strong(DeliveryReviewAgain, 0.85).EffectiveAction(true); got != DeliveryEffectiveReview {
		t.Fatalf("funded review_again routed %q", got)
	}
	// The same evidence without allowance exhausts instead of reviewing.
	if got := strong(DeliveryReviewAgain, 0.85).EffectiveAction(false); got != DeliveryEffectiveExhausted {
		t.Fatalf("unfunded review_again routed %q", got)
	}
	// review_again below its probability cutoff waits: positive evidence
	// is needed to buy a review.
	weak := strong(DeliveryReviewAgain, 0.79)
	if got := weak.EffectiveAction(true); got != DeliveryEffectiveAttention {
		t.Fatalf("below-threshold review_again routed %q", got)
	}
	// review_again below its confidence cutoff waits too.
	unconfident := strong(DeliveryReviewAgain, 0.9)
	unconfident.Confidence = 0.79
	if got := unconfident.EffectiveAction(true); got != DeliveryEffectiveAttention {
		t.Fatalf("unconfident review_again routed %q", got)
	}
	// A tied review_again at high confidence still waits: a tie is not a
	// strict lead.
	tie := strong(DeliveryReviewAgain, 0.5)
	tie.Probability, tie.Confidence = 0.9, 0.9
	if got := tie.EffectiveAction(true); got != DeliveryEffectiveAttention {
		t.Fatalf("tied review_again routed %q", got)
	}
	// A tied review_again without allowance waits as attention, not as
	// exhaustion: exhaustion names only a warranted-but-unfunded review.
	if got := tie.EffectiveAction(false); got != DeliveryEffectiveAttention {
		t.Fatalf("unfunded tied review_again routed %q", got)
	}
	// Attention retains the raw answer and validates as a pair.
	answer := weak
	decision := DeliveryDecision{Answer: answer, Effective: DeliveryEffectiveAttention, ReviewsFit: true}
	if err := decision.Validate(); err != nil {
		t.Fatalf("attention decision refused: %v", err)
	}
	// Attention never authorizes merge: only ready_to_merge passes the
	// common decision gate's effective check.
	if decision.Effective == DeliveryEffectiveReady {
		t.Fatal("attention authorizes merge")
	}
}
