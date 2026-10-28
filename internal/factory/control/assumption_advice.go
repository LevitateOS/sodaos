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

// JudgeAssumptions reviews supplied proposal ideas against a draft revision.
// Decided ideas keep their answers without another judgment. Attributable
// autonomous and evidenced supported choices settle without a human stamp and
// without spending a call; unclassified legacy choices stay pending for the
// person. Each pending assumption consumes one finite reservation: a confident
// challenge verdict interrupts, while an uncertain or default verdict settles
// a non-material assumption with autonomous basis. Material assumptions never
// settle; a material assumption without a confident challenge is still asked
// as a challenge, since stranding it pending (no question, no vote) blocks
// the revision while design keeps clarifying. Challenges hold only their own
// idea's dependents; declining one guess changes no other decision.
// Permission and spending stay code-enforced: this advice touches neither.
func (c *Coordinator) JudgeAssumptions(ctx context.Context, r factory.IntentRevision, content factory.ProposalContent, ideas []factory.ProductIdea) (factory.AssumptionReview, error) {
	if err := r.Validate(); err != nil {
		return factory.AssumptionReview{}, err
	}
	if err := content.Validate(); err != nil {
		return factory.AssumptionReview{}, err
	}
	if r.State != "draft" || content.Revision != r.ID || content.Intent != r.Intent {
		return factory.AssumptionReview{}, errors.New("assumption review needs the matching draft revision")
	}
	var review factory.AssumptionReview
	for _, idea := range ideas {
		if idea.Decision != "pending" {
			continue
		}
		if !idea.Assumption {
			if settlement, ok := settleChoice(idea); ok {
				review.Settled = append(review.Settled, settlement)
			}
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
			return factory.AssumptionReview{}, err
		}
		if answer.Selected != "challenge" || answer.Confidence < 0.5 {
			if !idea.Material {
				review.Settled = append(review.Settled, factory.AssumptionSettlement{Idea: idea.ID, Source: idea.Source, Provenance: "autonomous"})
				continue
			}
			// A material assumption still needs the person even when the
			// judge does not confidently interrupt: leaving it pending with
			// no challenge strands the revision (no question to answer, no
			// work-linked vote) while design keeps clarifying. Ask the
			// standard question so the person can confirm or correct it.
			challenge := factory.AssumptionChallenge{Idea: idea.ID, Assumption: idea.Description, Consequence: idea.Consequence, Recommendation: factory.RecommendAssumptionResolution(idea)}
			if err := challenge.Validate(); err != nil {
				return factory.AssumptionReview{}, err
			}
			review.Challenges = append(review.Challenges, challenge)
			continue
		}
		challenge := factory.AssumptionChallenge{Idea: idea.ID, Assumption: idea.Description, Consequence: idea.Consequence, Recommendation: factory.RecommendAssumptionResolution(idea)}
		if err := challenge.Validate(); err != nil {
			return factory.AssumptionReview{}, err
		}
		review.Challenges = append(review.Challenges, challenge)
	}
	return review, review.Validate()
}

// settleChoice accepts attributable autonomous and evidenced supported
// choices without a judgment call. Material, legacy-unclassified and
// human-claimed choices stay pending for the person.
func settleChoice(idea factory.ProductIdea) (factory.AssumptionSettlement, bool) {
	if idea.Material || idea.Actor != 0 || idea.Provenance == "human" {
		return factory.AssumptionSettlement{}, false
	}
	if idea.Source == "autonomous" && (idea.Provenance == "autonomous" || idea.Provenance == "") {
		return factory.AssumptionSettlement{Idea: idea.ID, Source: "autonomous", Provenance: "autonomous"}, true
	}
	if idea.Source == "supported" && (idea.Provenance == "autonomous" || idea.Provenance == "") && idea.Evidence != "" {
		return factory.AssumptionSettlement{Idea: idea.ID, Source: "supported", Provenance: "autonomous"}, true
	}
	return factory.AssumptionSettlement{}, false
}
