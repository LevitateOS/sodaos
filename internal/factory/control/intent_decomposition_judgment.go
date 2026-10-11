package control

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/store"
)

var ErrIntentDecompositionReview = errors.New("accepted task decomposition needs reconsideration against product scope")

// AdviseIntentDecomposition consults provided accepted facts before a native
// graph packet. Advice cannot extend accepted scope or authorize native effects.
// Three fresh phrasings share exact evidence; disagreement is a durable refusal.
func (c *Coordinator) AdviseIntentDecomposition(ctx context.Context, r factory.IntentRevision, previous *factory.IntentRevision) error {
	if c.Planning == nil || c.Planning.Chooser == nil || c.Store == nil {
		return ErrPriorityUnavailable
	}
	if r.Validate() != nil || r.State != "accepted" {
		return ErrIntentDecompositionReview
	}
	questions := []string{
		"Do the accepted task objectives, outcomes and prerequisite edges form a useful decomposition of the accepted product direction? Consider overlap, unnecessary splits and missing material scope. A declined assumption grants no constraint; if its rejection or an accepted correction conflicts with the objective or leaves consequential scope ambiguous, choose review before any native effect. Judge only supplied facts; do not invent research, estimates, requirements or permission. A prior graph is comparison evidence, not additional authority.",
		"Assess whether the approved task graph sensibly implements the accepted product intent using the provided objectives, acceptance outcomes and dependencies. Look for material ambiguity, duplicated work and needless fragmentation. Respect rejected assumptions as rejected; a contradiction between the objective and reviewed corrections or declined assumptions requires review. Unknown facts stay unknown; the earlier graph supplies context without expanding scope or granting effects.",
		"Which verdict fits the reviewed decomposition when comparing its stated work outcomes and dependency relations to current accepted product choices and, if supplied, the previous graph? Evaluate redundant or fragmented work and consequential unresolved intent from this evidence alone. Do not carry declined constraints into work: choose review if declining or correcting an assumption makes the objective contradictory or materially unclear. No speculation can add objectives, resource estimates or native authority.",
	}
	choices := []map[string]any{
		{"keep": "The approved task graph is coherent within accepted scope; retain it.", "revise": "Material overlap or fragmentation warrants a bounded split/consolidation and dependency refresh while retaining accepted product outcomes.", "review": "A consequential product assumption or scope gap remains unresolved; clarify that product choice before native mutation."},
		{"keep": "Retain the accepted decomposition because its tasks and edges form a sensible scoped implementation.", "revise": "Consolidate duplicated work or split overloaded work and refresh affected dependencies without changing approved outcomes.", "review": "Missing material product intent prevents safe decomposition and needs a concrete product decision."},
		{"keep": "Preserve the reviewed graph: it appropriately divides the accepted outcomes and relationships.", "revise": "Equivalent scoped restructuring of task boundaries or prerequisite links is needed to remove important redundancy or fragmentation.", "review": "An unresolved consequential assumption changes the intended product; obtain that missing direction first."},
	}
	type evidence struct {
		Current  factory.IntentRevision  `json:"current"`
		Previous *factory.IntentRevision `json:"previous,omitempty"`
	}
	state := evidence{r, previous}
	stateBytes, err := json.Marshal(state)
	if err != nil {
		return err
	}
	var requests [3]factory.PriorityRequest
	var answers [3]factory.PriorityAnswer
	var cached [3]bool
	for i := range requests {
		data, err := json.Marshal(struct {
			Kind     string
			State    evidence
			Question string
			Choices  map[string]any
		}{"intent-decomposition-v1", state, questions[i], choices[i]})
		if err != nil {
			return err
		}
		sum := sha256.Sum256(data)
		request := factory.PriorityRequest{Snapshot: hex.EncodeToString(sum[:]), Rubric: questions[i], Candidates: []factory.PriorityCandidate{{ID: "keep", Objective: r.Objective}, {ID: "revise", Objective: r.Objective}, {ID: "review", Objective: r.Objective}}}
		request.State, request.Criteria = stateBytes, choices[i]
		requests[i] = request
		prior, err := c.Store.PriorityJudgment(ctx, request.Snapshot)
		if err == nil {
			if prior.Outcome != "decomposition" || len(prior.Calls) != 1 || prior.Calls[0].Answer.Validate(request) != nil {
				return ErrIntentDecompositionReview
			}
			answers[i], cached[i] = prior.Calls[0].Answer, true
			continue
		}
		if !errors.Is(err, store.ErrNotFound) {
			return err
		}
		reserved, err := c.Store.ReserveIntentJudgment(ctx, r, request.Snapshot)
		if err != nil {
			return err
		}
		if !reserved {
			return ErrPriorityUnavailable
		}
	}
	for i, request := range requests {
		if cached[i] {
			continue
		}
		bounded, cancel := context.WithTimeout(ctx, 30*time.Second)
		answer, callErr := c.Planning.Chooser.Select(bounded, state, questions[i], choices[i])
		cancel()
		if callErr == nil {
			callErr = answer.Validate(request)
		}
		call := factory.PriorityCall{Request: request, Answer: answer}
		receipt := factory.PriorityReceipt{Snapshot: request.Snapshot, Candidates: request.Candidates, Calls: []factory.PriorityCall{call}, Selected: answer.Selected, Outcome: "decomposition", CreatedUnix: time.Now().Unix()}
		if callErr != nil {
			receipt.Calls[0].Failure = "consultation_failed"
			receipt.Outcome = "unavailable"
		}
		if err := c.Store.RecordPriorityJudgment(ctx, receipt); err != nil {
			return err
		}
		if callErr != nil {
			return ErrPriorityUnavailable
		}
		answers[i] = answer
	}
	for _, answer := range answers {
		if answer.Selected != "keep" || answer.Confidence < 0.5 {
			return ErrIntentDecompositionReview
		}
	}
	return nil
}
