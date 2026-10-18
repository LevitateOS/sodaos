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

// BundleEligibility reports which usable bundles have backing content:
// attempt needs own-attempt run continuity on the recorded assignment,
// lineage needs a linked parent plus its loaded record. Code enforces
// eligibility; Jev only foregrounds among eligible bundles.
func BundleEligibility(assignment Assignment, parentLoaded bool) (attempt, lineage bool) {
	attempt = len(assignment.RunHistory) > 0
	lineage = assignment.ParentAssignment != "" && parentLoaded
	return attempt, lineage
}

// AttemptContinuity returns the own-attempt run identities for the
// attempt bundle: opaque IDs only, never prompt bytes, history content
// or actor identity. Nil when the assignment carries no continuity.
func AttemptContinuity(assignment Assignment) []string {
	if len(assignment.RunHistory) == 0 {
		return nil
	}
	return append([]string(nil), assignment.RunHistory...)
}

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
