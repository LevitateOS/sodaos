package factory

import (
	"errors"
	"regexp"
	"strings"

	"github.com/levitateos/sodaos/internal/project"
)

// Mismatch classes name specific recomposition targets. "unverified" means
// the check could not confirm the walkthrough; it is not a clean verdict.
const (
	MismatchEffectAdded  = "effect_added"
	MismatchOutcomeGap   = "outcome_gap"
	MismatchSummaryDrift = "summary_drift"
	MismatchUnverified   = "unverified"
)

// Mismatch is one specific consistency finding for recomposition.
type Mismatch struct {
	Class  string `json:"class"`
	Detail string `json:"detail"`
}

// ConsistencyReport binds verification findings to one exact proposal
// revision. Consumers treat findings under any other revision as stale.
// Empty Mismatches with a confident check means the walkthrough verified.
type ConsistencyReport struct {
	Producer   string     `json:"producer"`
	Intent     string     `json:"intent"`
	Revision   string     `json:"revision"`
	Mismatches []Mismatch `json:"mismatches"`
	Confidence float64    `json:"confidence"`
	Advice     string     `json:"advice"`
}

var effectPattern = regexp.MustCompile(`(?i)\b(email|calendar|payment|sms|access)\b`)

// CheckProposalEffects scans the recommended walkthrough for effect words
// with no sourced basis in the person's facts. Person-sourced effects in
// facts are allowed; silent authored additions are specific mismatches.
func CheckProposalEffects(content ProposalContent, proposal ConsolidatedProposal) []Mismatch {
	var mismatches []Mismatch
	if len(proposal.Outcomes) == 0 || proposal.Outcomes[0] != proposalObjective(content.Objective) {
		mismatches = append(mismatches, Mismatch{Class: MismatchOutcomeGap, Detail: "The first outcome does not preserve the requested objective."})
	}
	var rendered strings.Builder
	alternative := proposal.Alternative
	rendered.WriteString(alternative.Name + " " + alternative.Summary + " " + alternative.Visibility + " " + alternative.Consequences)
	for _, step := range alternative.Steps {
		rendered.WriteString(" " + step.Actor + " " + step.Action + " " + step.Result)
		for _, branch := range step.Branches {
			rendered.WriteString(" " + branch.Outcome + " " + branch.Result)
		}
	}
	facts := strings.ToLower(strings.Join(append([]string{content.Objective}, content.Facts...), " "))
	seen := map[string]bool{}
	for _, found := range effectPattern.FindAllString(strings.ToLower(rendered.String()), -1) {
		word := strings.ToLower(found)
		if seen[word] || strings.Contains(facts, word) {
			seen[word] = true
			continue
		}
		seen[word] = true
		mismatches = append(mismatches, Mismatch{Class: MismatchEffectAdded, Detail: "The walkthrough adds " + word + " without a sourced basis in the request."})
	}
	return mismatches
}

// Validate checks report binding, mismatch classes and advice provenance.
func (r ConsistencyReport) Validate() error {
	if r.Producer != AuthoredProducer || !project.ValidDecisionID(r.Intent) || !project.ValidDecisionID(r.Revision) {
		return errors.New("invalid consistency report identity")
	}
	if len(r.Mismatches) > 16 || r.Confidence < 0 || r.Confidence > 1 || !ValidDigest(r.Advice) {
		return errors.New("invalid consistency report bounds or advice")
	}
	for _, mismatch := range r.Mismatches {
		switch mismatch.Class {
		case MismatchEffectAdded, MismatchOutcomeGap, MismatchSummaryDrift, MismatchUnverified:
		default:
			return errors.New("unknown mismatch class")
		}
		if !intentText(mismatch.Detail, 4000) {
			return errors.New("invalid mismatch detail")
		}
	}
	return nil
}
