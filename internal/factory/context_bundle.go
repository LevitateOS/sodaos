package factory

// Context bundle identities for R31 bundle selection. Code builds the
// fixed bundles; Jev returns exactly one eligible ID. snapshot_hold is
// the deterministic fallback, never a usable bundle.
const (
	ContextBundleCurrent      = "current"
	ContextBundleAttempt      = "attempt"
	ContextBundleLineage      = "lineage"
	ContextBundleSnapshotHold = "snapshot_hold"
)

// Context bundle decision bounds. Only an eligible selected option at
// or above both bounds serves; anything else holds with attention.
// Proposed bound pending context-specific evaluation.
const (
	MinContextBundleProbability = 0.80
	MinContextBundleConfidence  = 0.80
)

// Context bundle hold reasons. Empty reason means serve.
const (
	ContextBundleHoldLowConfidence = "low_confidence"
	ContextBundleHoldIneligible    = "ineligible_selection"
)

// ApplyContextBundleAnswer maps one validated answer to its bundle or
// a hold: only an eligible selected option at or above both bounds
// serves the bundle, everything else holds snapshot_hold for
// attention. It never infers a bundle.
func ApplyContextBundleAnswer(answer PriorityAnswer, eligible map[string]bool) (bundle, reason string) {
	if !eligible[answer.Selected] {
		return ContextBundleSnapshotHold, ContextBundleHoldIneligible
	}
	if answer.Confidence < MinContextBundleConfidence || answer.Probabilities[answer.Selected] < MinContextBundleProbability {
		return ContextBundleSnapshotHold, ContextBundleHoldLowConfidence
	}
	return answer.Selected, ""
}
