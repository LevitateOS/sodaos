package factory

import "testing"

func TestApplyContextBundleAnswerServesConfidentEligible(t *testing.T) {
	for _, bundle := range []string{ContextBundleCurrent, ContextBundleAttempt, ContextBundleLineage} {
		answer := PriorityAnswer{
			Selected: bundle, Confidence: 0.95,
			Probabilities: map[string]float64{bundle: 0.9, ContextBundleSnapshotHold: 0.1},
		}
		eligible := map[string]bool{bundle: true, ContextBundleSnapshotHold: true}
		if got, reason := ApplyContextBundleAnswer(answer, eligible); got != bundle || reason != "" {
			t.Fatalf("%s selected (%q, %q)", bundle, got, reason)
		}
	}
}

func TestApplyContextBundleAnswerHoldsBelowBounds(t *testing.T) {
	eligible := map[string]bool{ContextBundleCurrent: true, ContextBundleSnapshotHold: true}
	low := PriorityAnswer{
		Selected: ContextBundleCurrent, Confidence: 0.5,
		Probabilities: map[string]float64{ContextBundleCurrent: 0.9},
	}
	if got, reason := ApplyContextBundleAnswer(low, eligible); got != ContextBundleSnapshotHold || reason != ContextBundleHoldLowConfidence {
		t.Fatalf("low confidence selected (%q, %q)", got, reason)
	}
	thin := PriorityAnswer{
		Selected: ContextBundleCurrent, Confidence: 0.95,
		Probabilities: map[string]float64{ContextBundleCurrent: 0.4},
	}
	if got, reason := ApplyContextBundleAnswer(thin, eligible); got != ContextBundleSnapshotHold || reason != ContextBundleHoldLowConfidence {
		t.Fatalf("thin probability selected (%q, %q)", got, reason)
	}
}

func TestApplyContextBundleAnswerHoldsIneligible(t *testing.T) {
	eligible := map[string]bool{ContextBundleCurrent: true, ContextBundleSnapshotHold: true}
	answer := PriorityAnswer{
		Selected: ContextBundleLineage, Confidence: 0.95,
		Probabilities: map[string]float64{ContextBundleLineage: 0.9},
	}
	if got, reason := ApplyContextBundleAnswer(answer, eligible); got != ContextBundleSnapshotHold || reason != ContextBundleHoldIneligible {
		t.Fatalf("ineligible selected (%q, %q)", got, reason)
	}
	empty := PriorityAnswer{Selected: "", Confidence: 0.95, Probabilities: map[string]float64{}}
	if got, reason := ApplyContextBundleAnswer(empty, eligible); got != ContextBundleSnapshotHold || reason != ContextBundleHoldIneligible {
		t.Fatalf("empty selection selected (%q, %q)", got, reason)
	}
}
