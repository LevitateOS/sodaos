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

func TestBundleEligibilityNeedsBackingContent(t *testing.T) {
	if attempt, lineage := BundleEligibility(Assignment{}, false); attempt || lineage {
		t.Fatal("bare assignment reports eligible bundles")
	}
	if attempt, _ := BundleEligibility(Assignment{RunHistory: []string{"run-1"}}, false); !attempt {
		t.Fatal("own-attempt continuity reports ineligible")
	}
	parent := Assignment{ID: "parent-1"}
	if _, lineage := BundleEligibility(Assignment{ParentAssignment: parent.ID}, false); lineage {
		t.Fatal("unloadable parent reports eligible lineage")
	}
	if _, lineage := BundleEligibility(Assignment{ParentAssignment: parent.ID}, true); !lineage {
		t.Fatal("loaded parent reports ineligible lineage")
	}
}

func TestAttemptContinuityCopiesOpaqueIDs(t *testing.T) {
	runs := []string{"run-1", "run-2"}
	got := AttemptContinuity(Assignment{RunHistory: runs})
	if len(got) != 2 || got[0] != "run-1" || got[1] != "run-2" {
		t.Fatalf("continuity=%v", got)
	}
	got[0] = "mutated"
	if runs[0] != "run-1" {
		t.Fatal("continuity aliases the stored history")
	}
	if AttemptContinuity(Assignment{}) != nil {
		t.Fatal("bare assignment reports continuity")
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
