package factory

import (
	"errors"

	"github.com/levitateos/sodaos/internal/project"
)

// AssumptionChallenge is one batched consequential question for the person.
// It names the assumed behavior, its consequence and a recommended resolution.
// Dependents of Idea stay held until the person answers; independent work
// continues. A challenge never records approval and never flips another idea.
type AssumptionChallenge struct {
	Idea           string `json:"idea"`
	Assumption     string `json:"assumption"`
	Consequence    string `json:"consequence"`
	Recommendation string `json:"recommendation"`
}

// RecommendAssumptionResolution authors the bounded default resolution offered
// beside a correction input. It restates the assumed behavior as sourced text,
// not as model prose or human approval.
func RecommendAssumptionResolution(idea ProductIdea) string {
	return "Confirm whether '" + idea.Description + "' holds, or explain the correction."
}

// Validate checks idea binding and bounded challenge prose.
func (c AssumptionChallenge) Validate() error {
	if !project.ValidDecisionID(c.Idea) || !intentText(c.Assumption, 4000) || !intentText(c.Consequence, 4000) || !intentText(c.Recommendation, 4000) {
		return errors.New("invalid assumption challenge")
	}
	return nil
}
