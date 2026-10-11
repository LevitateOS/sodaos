package control

import (
	"context"
	"errors"

	"github.com/levitateos/sodaos/internal/factory"
)

var ErrIntentDecompositionReview = errors.New("accepted task decomposition needs reconsideration against product scope")

func decompositionQuestions() []string {
	return []string{
		"Do the accepted task objectives, outcomes and prerequisite edges form a useful decomposition of the accepted product direction? Consider overlap, unnecessary splits and missing material scope. A declined assumption grants no constraint; if its rejection or an accepted correction conflicts with the objective or leaves consequential scope ambiguous, choose review before any native effect. Judge only supplied facts; do not invent research, estimates, requirements or permission. A prior graph is comparison evidence, not additional authority.",
		"Assess whether the approved task graph sensibly implements the accepted product intent using the provided objectives, acceptance outcomes and dependencies. Look for material ambiguity, duplicated work and needless fragmentation. Respect rejected assumptions as rejected; a contradiction between the objective and reviewed corrections or declined assumptions requires review. Unknown facts stay unknown; the earlier graph supplies context without expanding scope or granting effects.",
		"Which verdict fits the reviewed decomposition when comparing its stated work outcomes and dependency relations to current accepted product choices and, if supplied, the previous graph? Evaluate redundant or fragmented work and consequential unresolved intent from this evidence alone. Do not carry declined constraints into work: choose review if declining or correcting an assumption makes the objective contradictory or materially unclear. No speculation can add objectives, resource estimates or native authority.",
	}
}

func decompositionChoices() []map[string]any {
	return []map[string]any{
		{"keep": "The approved task graph is coherent within accepted scope; retain it.", "revise": "Material overlap or fragmentation warrants a bounded split/consolidation and dependency refresh while retaining accepted product outcomes.", "review": "A consequential product assumption or scope gap remains unresolved; clarify that product choice before native mutation."},
		{"keep": "Retain the accepted decomposition because its tasks and edges form a sensible scoped implementation.", "revise": "Consolidate duplicated work or split overloaded work and refresh affected dependencies without changing approved outcomes.", "review": "Missing material product intent prevents safe decomposition and needs a concrete product decision."},
		{"keep": "Preserve the reviewed graph: it appropriately divides the accepted outcomes and relationships.", "revise": "Equivalent scoped restructuring of task boundaries or prerequisite links is needed to remove important redundancy or fragmentation.", "review": "An unresolved consequential assumption changes the intended product; obtain that missing direction first."},
	}
}

type decompositionEvidence struct {
	Current  factory.IntentRevision  `json:"current"`
	Previous *factory.IntentRevision `json:"previous,omitempty"`
}

func (c *Coordinator) judgeDecompositionTriplet(ctx context.Context, r factory.IntentRevision, previous *factory.IntentRevision) ([3]factory.PriorityAnswer, [3]string, error) {
	var answers [3]factory.PriorityAnswer
	var fingerprints [3]string
	questions, choices := decompositionQuestions(), decompositionChoices()
	state := decompositionEvidence{r, previous}
	candidates := []factory.PriorityCandidate{{ID: "keep", Objective: r.Objective}, {ID: "revise", Objective: r.Objective}, {ID: "review", Objective: r.Objective}}
	for i := range questions {
		answer, _, err := c.AdvisePurpose(ctx, PurposeAdvice{Purpose: factory.JudgmentDecomposition, State: state, Question: questions[i], Options: choices[i], Outcome: "decomposition", Candidates: candidates, Reserve: func(ctx context.Context, snapshot string) (bool, error) {
			return c.Store.ReserveIntentJudgment(ctx, r, snapshot)
		}})
		if err != nil {
			return answers, fingerprints, err
		}
		fingerprint, err := factory.JudgmentFingerprint(factory.JudgmentDecomposition, state, questions[i], choices[i])
		if err != nil {
			return answers, fingerprints, err
		}
		answers[i], fingerprints[i] = answer, fingerprint
	}
	return answers, fingerprints, nil
}

// AdviseIntentDecomposition consults provided accepted facts before a native
// graph packet. Advice cannot extend accepted scope or authorize native effects.
// Three fresh phrasings share exact evidence; disagreement is a durable refusal.
func (c *Coordinator) AdviseIntentDecomposition(ctx context.Context, r factory.IntentRevision, previous *factory.IntentRevision) error {
	if r.Validate() != nil || r.State != "accepted" {
		return ErrIntentDecompositionReview
	}
	answers, _, err := c.judgeDecompositionTriplet(ctx, r, previous)
	if err != nil {
		return err
	}
	for _, answer := range answers {
		if answer.Selected != "keep" || answer.Confidence < 0.5 {
			return ErrIntentDecompositionReview
		}
	}
	return nil
}

const decompositionOptionsRubric = "Compare the supplied decomposition candidates against the accepted product direction. Every candidate preserves accepted outcomes, idea bindings and bounds; choose the task boundaries and prerequisite edges that best remove overlap or fragmentation. Judge only supplied facts; grant no authority, access or spending."

// RecomposeDecomposition replaces a technical revise verdict with a valid
// scoped alternative for native graph writes. Confident keep verdicts return
// the accepted graph without further spend; confident revise verdicts select
// among validated split/combine candidates, falling back to keep when the
// selection is uncertain. Only consequential product ambiguity or uncertain
// triplet verdicts need a human challenge. I consumes the selected tasks for
// initial submission and later deltas; this advice writes no native state.
func (c *Coordinator) RecomposeDecomposition(ctx context.Context, r factory.IntentRevision, previous *factory.IntentRevision) (factory.ScopedDecomposition, error) {
	if err := r.Validate(); err != nil {
		return factory.ScopedDecomposition{}, err
	}
	if r.State != "accepted" {
		return factory.ScopedDecomposition{}, ErrIntentDecompositionReview
	}
	answers, fingerprints, err := c.judgeDecompositionTriplet(ctx, r, previous)
	if err != nil {
		return factory.ScopedDecomposition{}, err
	}
	revise := false
	for _, answer := range answers {
		if answer.Selected == "review" || answer.Confidence < 0.5 {
			return factory.ScopedDecomposition{}, ErrIntentDecompositionReview
		}
		revise = revise || answer.Selected == "revise"
	}
	advice := append([]string(nil), fingerprints[:]...)
	if !revise {
		decomposition := factory.ScopedDecomposition{Intent: r.Intent, Revision: r.ID, Selected: "keep", Tasks: append([]factory.IntentTask(nil), r.Tasks...), Advice: advice}
		return decomposition, decomposition.Validate()
	}
	candidates := factory.DecompositionCandidates(r)
	byKey := map[string]factory.DecompositionCandidate{}
	options := make(map[string]any, len(candidates))
	for _, candidate := range candidates {
		byKey[candidate.Key] = candidate
		options[candidate.Key] = candidate.Rationale
	}
	type optionsEvidence struct {
		Current    factory.IntentRevision           `json:"current"`
		Candidates []factory.DecompositionCandidate `json:"candidates"`
	}
	state := optionsEvidence{r, candidates}
	answer, _, err := c.AdvisePurpose(ctx, PurposeAdvice{
		Purpose:  factory.JudgmentDecomposition,
		State:    state,
		Question: decompositionOptionsRubric,
		Options:  options,
		Outcome:  "decomposition-options",
		Reserve: func(ctx context.Context, snapshot string) (bool, error) {
			return c.Store.ReserveIntentJudgment(ctx, r, snapshot)
		},
	})
	if err != nil {
		return factory.ScopedDecomposition{}, err
	}
	fingerprint, err := factory.JudgmentFingerprint(factory.JudgmentDecomposition, state, decompositionOptionsRubric, options)
	if err != nil {
		return factory.ScopedDecomposition{}, err
	}
	selected := answer.Selected
	if answer.Confidence < 0.5 {
		selected = "keep"
	}
	candidate, ok := byKey[selected]
	if !ok {
		return factory.ScopedDecomposition{}, ErrIntentDecompositionReview
	}
	decomposition := factory.ScopedDecomposition{Intent: r.Intent, Revision: r.ID, Selected: selected, Tasks: candidate.Tasks, Advice: append(advice, fingerprint)}
	return decomposition, decomposition.Validate()
}
