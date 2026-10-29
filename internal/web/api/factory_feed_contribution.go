package api

import (
	"context"
	"sync"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/factory/control"
)

type contributionAnswer struct {
	call factory.ContributionCall
	err  error
}

// driveContributions is part of the existing service continuation, not a feed
// GET, browser timer or separate worker loop. It judges one source per pass and
// only considers roles already running under native admission.
func (s *API) driveContributions(ctx context.Context) {
	if err := s.Store.ExpireContributionReviews(ctx); err != nil {
		return
	}
	if !s.jevConnected() {
		return
	}
	ids, err := s.Store.ContributionSources(ctx)
	if err != nil {
		return
	}
	for _, id := range ids {
		review, err := s.Store.ContributionContext(ctx, id)
		if err != nil || len(review.Roles) == 0 {
			continue
		}
		claimed, err := s.Store.ClaimContributionReview(ctx, review)
		if err != nil || !claimed {
			continue
		}
		s.assessContribution(ctx, &review)
		return
	}
}

func (s *API) assessContribution(ctx context.Context, review *factory.ContributionReview) {
	funded, err := s.Store.ReservePriorityJudgment(ctx, review.Snapshot, len(review.Roles)*len(factory.ContributionQuestions))
	if err != nil || !funded {
		review.State = "unavailable"
		_ = s.Store.SettleContributionReview(ctx, *review)
		return
	}
	judge := &control.TypeSafePriority{APIKeyFile: s.Config.TypeSafeAPIKeyFile, Model: s.Config.TypeSafeModel}
	callctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	review.Calls, err = judgeContributions(callctx, judge, *review)
	if err == nil {
		factory.SelectContribution(review)
	} else {
		review.State = "unavailable"
	}
	receipt := factory.PriorityReceipt{Snapshot: review.Snapshot, Outcome: review.State, CreatedUnix: time.Now().Unix(), Selected: review.Assignment}
	for _, call := range review.Calls {
		receipt.Calls = append(receipt.Calls, factory.PriorityCall{Request: contributionRequest(review.Snapshot, call.Question), Answer: call.Answer})
	}
	settlectx, settleCancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer settleCancel()
	if err := s.Store.RecordPriorityJudgment(settlectx, receipt); err != nil {
		review.State, review.Assignment = "unavailable", ""
	}
	_ = s.Store.SettleContributionReview(settlectx, *review)
}

func contributionRequest(snapshot, question string) factory.PriorityRequest {
	return factory.PriorityRequest{Snapshot: snapshot, Rubric: question, Candidates: []factory.PriorityCandidate{{ID: "contribute"}, {ID: "no_contribution"}}}
}

func judgeContributions(ctx context.Context, judge *control.TypeSafePriority, review factory.ContributionReview) ([]factory.ContributionCall, error) {
	// All prompts and state are fixed before launching the independent calls.
	// Each role gets the same binary judgment shape used in evaluation; role
	// selection afterward cannot dilute its probability across other candidates.
	type request struct {
		role     factory.ContributionRole
		question string
		state    any
	}
	requests := []request{}
	for _, role := range review.Roles {
		state := map[string]any{"source": map[string]any{"body": review.Body, "actor": review.SourceActor, "context": review.Context, "revision": review.Revision}, "existing_replies": review.Replies, "eligible_running_role": role}
		for _, question := range factory.ContributionQuestions {
			requests = append(requests, request{role, question, state})
		}
	}
	results := make([]contributionAnswer, len(requests))
	var wg sync.WaitGroup
	for index, input := range requests {
		wg.Add(1)
		go func(index int, input request) {
			defer wg.Done()
			answer, err := judge.Select(ctx, input.state, input.question, map[string]any{
				"contribute":      "This role can add a specific useful missing correction, risk or evidence within its admitted work.",
				"no_contribution": "A reply would be redundant, courtesy, ambiguous, already answered, irrelevant or outside its admitted work.",
			})
			if err == nil {
				err = answer.Validate(contributionRequest(review.Snapshot, input.question))
			}
			results[index] = contributionAnswer{factory.ContributionCall{Assignment: input.role.Assignment, Question: input.question, Answer: answer}, err}
		}(index, input)
	}
	wg.Wait()
	calls := []factory.ContributionCall{}
	var failure error
	for _, result := range results {
		if result.err != nil {
			failure = result.err
		} else {
			calls = append(calls, result.call)
		}
	}
	return calls, failure
}
