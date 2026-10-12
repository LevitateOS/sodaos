package factory

import (
	"errors"
	"regexp"
	"strings"
	"unicode"

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

// exclusionStopwords drops closed-class words so decline overlap needs
// shared substantive vocabulary rather than shared grammar.
var exclusionStopwords = map[string]bool{
	"a": true, "an": true, "the": true, "and": true, "or": true, "to": true,
	"of": true, "in": true, "on": true, "for": true, "with": true, "without": true,
	"is": true, "are": true, "was": true, "were": true, "be": true, "been": true,
	"being": true, "by": true, "at": true, "as": true, "it": true, "its": true,
	"this": true, "that": true, "these": true, "those": true, "they": true,
	"them": true, "their": true, "theirs": true, "from": true, "into": true,
	"over": true, "under": true, "than": true, "then": true, "when": true,
	"while": true, "what": true, "who": true, "whom": true, "whose": true,
	"which": true, "how": true, "not": true, "no": true, "nor": true, "but": true,
	"if": true, "else": true, "unless": true, "until": true, "can": true,
	"cannot": true, "could": true, "should": true, "would": true, "will": true,
	"shall": true, "may": true, "might": true, "must": true, "do": true,
	"does": true, "did": true, "done": true, "have": true, "has": true,
	"had": true, "having": true, "each": true, "every": true, "all": true,
	"any": true, "some": true, "such": true, "own": true, "other": true,
	"otherwise": true, "only": true, "same": true, "also": true, "both": true,
	"either": true, "neither": true, "between": true, "through": true,
	"during": true, "before": true, "after": true, "again": true, "once": true,
	"about": true,
}

// distinctiveWords lowers text into substantive tokens: letter/digit runs
// of at least four characters with stopwords removed. Single shared words
// prove nothing; overlap needs two distinct shared tokens.
func distinctiveWords(text string) []string {
	var words []string
	for _, field := range strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	}) {
		if len([]rune(field)) < 4 || exclusionStopwords[field] {
			continue
		}
		words = append(words, field)
	}
	return words
}

// shortQuote bounds a declined description inside a mismatch detail so the
// finding always fits its limit while naming the specific contradiction.
func shortQuote(text string) string {
	const limit = 160
	runes := []rune(text)
	if len(runes) <= limit {
		return text
	}
	return string(runes[:limit]) + "..."
}

// CheckProposalEffects scans the recommended walkthrough for effect words
// with no sourced basis in the person's facts. Person-sourced effects in
// facts are allowed; silent authored additions are specific mismatches.
// A walkthrough that shares substantive vocabulary with a declined idea
// reintroduces explicitly rejected behavior and reports the same mismatch
// class with the decline named, without asking for renewed approval.
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
	walkthrough := map[string]bool{}
	for _, word := range distinctiveWords(rendered.String()) {
		walkthrough[word] = true
	}
	for _, exclusion := range content.Exclusions {
		matched := map[string]bool{}
		for _, word := range distinctiveWords(exclusion.Description + " " + exclusion.Consequence) {
			if walkthrough[word] {
				matched[word] = true
			}
		}
		if len(matched) < 2 {
			continue
		}
		mismatches = append(mismatches, Mismatch{Class: MismatchEffectAdded, Detail: "The walkthrough reintroduces declined '" + shortQuote(exclusion.Description) + "' (idea " + exclusion.Idea + "); recompose without it."})
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
