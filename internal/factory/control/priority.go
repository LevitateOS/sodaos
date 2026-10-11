package control

import (
	"context"
	"errors"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
)

// PriorityJournal reserves finite independently authorized consultation credit
// before an external request and prevents repeating unchanged selections.
type PriorityJournal interface {
	ReservePriorityJudgment(context.Context, string, int) (bool, error)
	RecordPriorityJudgment(context.Context, factory.PriorityReceipt) error
}

type PriorityChooser interface {
	Choose(context.Context, factory.PriorityRequest) (factory.PriorityAnswer, error)
}

type PriorityJudgments struct {
	Journal       PriorityJournal
	Chooser       PriorityChooser
	MaxCalls      int
	StarvationAge time.Duration
}

var ErrPriorityUnavailable = errors.New("priority consultation unavailable or exhausted")

// selectPriority covers every candidate through a deterministic tournament.
// Groups contain at most 255 alternatives; each winner advances until a final
// choice. Required calls are reserved up front; an oversized set is refused,
// never truncated. Uncertain preference falls back to oldest within its group.
func selectPriority(ctx context.Context, deps *PriorityJudgments, candidates []factory.PriorityCandidate, now time.Time) (string, error) {
	if len(candidates) == 0 {
		return "", nil
	}
	if len(candidates) == 1 {
		return candidates[0].ID, nil
	}
	if deps == nil || deps.Journal == nil {
		return "", ErrPriorityUnavailable
	}
	snapshot, err := factory.PrioritySnapshot(candidates)
	if err != nil {
		return "", err
	}
	oldest := candidates[0]
	for _, c := range candidates {
		if c.Control.FirstSeenUnix < oldest.Control.FirstSeenUnix {
			oldest = c
		}
	}
	age := deps.StarvationAge
	if age <= 0 {
		age = 24 * time.Hour
	}
	receipt := factory.PriorityReceipt{Snapshot: snapshot, Candidates: candidates, CreatedUnix: now.Unix(), Calls: []factory.PriorityCall{}}
	if oldest.Control.FirstSeenUnix > 0 && now.Sub(time.Unix(oldest.Control.FirstSeenUnix, 0)) >= age {
		receipt.Selected, receipt.Outcome = oldest.ID, "fairness"
		if err := deps.Journal.RecordPriorityJudgment(ctx, receipt); err != nil {
			return "", err
		}
		return oldest.ID, nil
	}
	maxCalls := deps.MaxCalls
	if maxCalls <= 0 {
		maxCalls = 16
	}
	required := priorityCallCount(len(candidates))
	if required > maxCalls || deps.Chooser == nil {
		return "", ErrPriorityUnavailable
	}
	reserved, err := deps.Journal.ReservePriorityJudgment(ctx, snapshot, required)
	if err != nil {
		return "", err
	}
	if !reserved {
		return "", ErrPriorityUnavailable
	}
	remaining := candidates
	for len(remaining) > 1 {
		var winners []factory.PriorityCandidate
		for offset := 0; offset < len(remaining); offset += 255 {
			end := offset + 255
			if end > len(remaining) {
				end = len(remaining)
			}
			group := remaining[offset:end]
			if len(group) == 1 {
				winners = append(winners, group[0])
				continue
			}
			request := factory.PriorityRequest{Snapshot: snapshot, Candidates: group, Rubric: factory.PriorityRubric}
			bounded, cancel := context.WithTimeout(ctx, 30*time.Second)
			answer, callErr := deps.Chooser.Choose(bounded, request)
			cancel()
			if callErr == nil {
				callErr = answer.Validate(request)
			}
			call := factory.PriorityCall{Request: request, Answer: answer}
			if callErr != nil {
				call.Failure = "consultation_failed"
				receipt.Calls = append(receipt.Calls, call)
				receipt.Outcome = "unavailable"
				return "", errors.Join(ErrPriorityUnavailable, deps.Journal.RecordPriorityJudgment(ctx, receipt))
			}
			receipt.Calls = append(receipt.Calls, call)
			chosen := group[0]
			for _, c := range group {
				if answer.Confidence < 0.5 {
					if c.Control.FirstSeenUnix < chosen.Control.FirstSeenUnix {
						chosen = c
					}
				} else if c.ID == answer.Selected {
					chosen = c
				}
			}
			winners = append(winners, chosen)
		}
		remaining = winners
	}
	receipt.Selected, receipt.Outcome = remaining[0].ID, "selected"
	if err := deps.Journal.RecordPriorityJudgment(ctx, receipt); err != nil {
		return "", err
	}
	return receipt.Selected, nil
}

func priorityCallCount(count int) int {
	calls := 0
	for count > 1 {
		calls += count / 255
		if count%255 > 1 {
			calls++
		}
		count = (count + 254) / 255
	}
	return calls
}
