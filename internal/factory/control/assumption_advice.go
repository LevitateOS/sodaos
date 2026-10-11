package control

import (
	"context"
	"errors"

	"github.com/levitateos/sodaos/internal/factory"
)

const assumptionRubric = "For the supplied assumption, decide between applying a supported or harmless reversible within-scope default and interrupting the person. Prefer the default when supplied evidence resolves the uncertainty or the assumption is harmless and reversible within the accepted direction. Choose challenge only for a consequential unsupported assumption that still needs the person's intent or authority. Judge only supplied facts; grant no authority, access or spending."

func assumptionOptions() map[string]any {
	return map[string]any{
		"apply_default": "A supported or harmless reversible within-scope default resolves this assumption; do not interrupt.",
		"challenge":     "This consequential unsupported assumption needs the person's intent or authority; interrupt with a focused challenge.",
	}
}

type assumptionEvidence struct {
	Objective string              `json:"objective"`
	Facts     []string            `json:"facts"`
	Unknowns  []string            `json:"unknowns"`
	Idea      factory.ProductIdea `json:"idea"`
}

// JudgeAssumptions batches challenges for pending assumption ideas in a draft
// revision. Decided ideas keep their answers without another judgment, and
// ordinary non-assumption choices never become challenges. Each unresolved
// assumption consumes one finite reservation; an uncertain or default verdict
// proceeds without interruption. Challenges hold only their own idea's
// dependents; declining one guess changes no other decision. Permission and
// spending stay code-enforced: this advice touches neither.
func (c *Coordinator) JudgeAssumptions(ctx context.Context, r factory.IntentRevision, content factory.ProposalContent) ([]factory.AssumptionChallenge, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}
	if err := content.Validate(); err != nil {
		return nil, err
	}
	if r.State != "draft" || content.Revision != r.ID || content.Intent != r.Intent {
		return nil, errors.New("assumption review needs the matching draft revision")
	}
	var challenges []factory.AssumptionChallenge
	for _, idea := range r.Ideas {
		if !idea.Assumption || idea.Decision != "pending" {
			continue
		}
		state := assumptionEvidence{Objective: r.Objective, Facts: content.Facts, Unknowns: content.Unknowns, Idea: idea}
		answer, _, err := c.AdvisePurpose(ctx, PurposeAdvice{
			Purpose:  factory.JudgmentAssumptions,
			State:    state,
			Question: assumptionRubric,
			Options:  assumptionOptions(),
			Outcome:  "assumptions",
			Reserve: func(ctx context.Context, snapshot string) (bool, error) {
				return c.Store.ReserveIntentJudgment(ctx, r, snapshot)
			},
		})
		if err != nil {
			return nil, err
		}
		if answer.Selected != "challenge" || answer.Confidence < 0.5 {
			continue
		}
		challenge := factory.AssumptionChallenge{Idea: idea.ID, Assumption: idea.Description, Consequence: idea.Consequence, Recommendation: factory.RecommendAssumptionResolution(idea)}
		if err := challenge.Validate(); err != nil {
			return nil, err
		}
		challenges = append(challenges, challenge)
	}
	return challenges, nil
}
