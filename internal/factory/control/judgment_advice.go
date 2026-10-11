package control

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/store"
)

// PurposeAdvice is one bounded typed judgment call. Reserve must consume
// finite allowance for the fingerprint before any external request; cached
// completed results reuse without a new reservation. Answers validate
// against the option identities; Candidates only enriches the receipt audit
// trail and defaults to those identities.
type PurposeAdvice struct {
	Purpose    factory.JudgmentPurpose
	State      any
	Question   string
	Options    map[string]any
	Outcome    string
	Candidates []factory.PriorityCandidate
	Reserve    func(context.Context, string) (bool, error)
}

// AdvisePurpose reuses a matching completed receipt, otherwise reserves
// finite allowance, consults the injected chooser once and records the
// durable receipt. Failures retain consumption: a reserved fingerprint
// without a completed answer stays charged and unavailable, with no
// refund, upstream lookup or silent repeat of unchanged input.
func (c *Coordinator) AdvisePurpose(ctx context.Context, advice PurposeAdvice) (factory.PriorityAnswer, bool, error) {
	if c.Planning == nil || c.Planning.Chooser == nil || c.Store == nil || advice.Reserve == nil || advice.Outcome == "" {
		return factory.PriorityAnswer{}, false, ErrPriorityUnavailable
	}
	fingerprint, err := factory.JudgmentFingerprint(advice.Purpose, advice.State, advice.Question, advice.Options)
	if err != nil {
		return factory.PriorityAnswer{}, false, err
	}
	stateBytes, err := json.Marshal(advice.State)
	if err != nil {
		return factory.PriorityAnswer{}, false, err
	}
	request := factory.PriorityRequest{Snapshot: fingerprint, Rubric: advice.Question, State: stateBytes, Criteria: advice.Options}
	for id := range advice.Options {
		request.Candidates = append(request.Candidates, factory.PriorityCandidate{ID: id})
	}
	prior, err := c.Store.PriorityJudgment(ctx, fingerprint)
	if err == nil {
		if prior.Purpose != advice.Purpose || prior.Outcome != advice.Outcome || len(prior.Calls) != 1 || prior.Calls[0].Failure != "" || prior.Calls[0].Answer.Validate(request) != nil {
			return factory.PriorityAnswer{}, false, ErrPriorityUnavailable
		}
		return prior.Calls[0].Answer, true, nil
	}
	if !errors.Is(err, store.ErrNotFound) {
		return factory.PriorityAnswer{}, false, err
	}
	reserved, err := advice.Reserve(ctx, fingerprint)
	if err != nil {
		return factory.PriorityAnswer{}, false, err
	}
	if !reserved {
		return factory.PriorityAnswer{}, false, ErrPriorityUnavailable
	}
	bounded, cancel := context.WithTimeout(ctx, 30*time.Second)
	answer, callErr := c.Planning.Chooser.Select(bounded, advice.State, advice.Question, advice.Options)
	cancel()
	if callErr == nil {
		callErr = answer.Validate(request)
	}
	call := factory.PriorityCall{Request: request, Answer: answer}
	candidates := request.Candidates
	if len(advice.Candidates) > 0 {
		candidates = advice.Candidates
	}
	receipt := factory.PriorityReceipt{Snapshot: fingerprint, Purpose: advice.Purpose, Candidates: candidates, Calls: []factory.PriorityCall{call}, Selected: answer.Selected, Outcome: advice.Outcome, CreatedUnix: time.Now().Unix()}
	if callErr != nil {
		receipt.Calls[0].Failure = "consultation_failed"
		receipt.Outcome = "unavailable"
	}
	if err := c.Store.RecordPriorityJudgment(ctx, receipt); err != nil {
		return factory.PriorityAnswer{}, false, err
	}
	if callErr != nil {
		return factory.PriorityAnswer{}, false, ErrPriorityUnavailable
	}
	return answer, false, nil
}
