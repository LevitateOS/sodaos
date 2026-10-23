package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
)

var ErrCompletionRetryIneligible = errors.New("completion retry is no longer admissible")

// RecordCompletionRetryCommand journals the owner command and its single
// successor atomically. A lost response cannot strand an unfinished command
// or create another cycle; the previous cycle and paid ledger stay intact.
func (s *Store) RecordCompletionRetryCommand(ctx context.Context, cmd factory.Command, repository int64, request factory.CompletionRetryRequest, now time.Time) (factory.CompletionRetryDecision, error) {
	var zero factory.CompletionRetryDecision
	payload, _ := json.Marshal(request)
	if cmd.Validate() != nil || request.Validate() != nil || cmd.Type != factory.CommandCompletionRetry ||
		cmd.Target != fmt.Sprintf("repository/%d/completion/%s", repository, request.EventID) || cmd.Payload != string(payload) {
		return zero, ErrCompletionRetryIneligible
	}
	decision := factory.CompletionRetryDecision{CommandID: cmd.ID, EventID: request.EventID, PriorCycle: request.ExpectedPriorCycle, SuccessorCycle: cmd.ID, Repository: repository, Queued: true}
	if err := decision.Validate(); err != nil {
		return zero, ErrCompletionRetryIneligible
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return zero, err
	}
	defer tx.Rollback()
	stored, created, err := recordFactoryCommandTx(ctx, tx, cmd, now)
	if err != nil {
		return zero, err
	}
	if !created {
		if stored.Principal != cmd.Principal || stored.Type != cmd.Type || stored.Target != cmd.Target {
			return zero, ErrCommandConflict
		}
		var prior factory.CompletionRetryDecision
		if json.Unmarshal([]byte(stored.Outcome), &prior) != nil || prior != decision {
			return zero, ErrCompletionRetryIneligible
		}
		return prior, tx.Commit()
	}
	retry := factory.ReassessmentRetry{CommandID: cmd.ID, EventID: request.EventID, ExpectedPriorCycle: request.ExpectedPriorCycle, SuccessorCycleID: cmd.ID}
	if _, err = startReassessmentRetryTx(ctx, tx, retry, repository); err != nil {
		return zero, err
	}
	outcome, err := json.Marshal(decision)
	if err != nil {
		return zero, err
	}
	if err := finishFactoryCommandTx(ctx, tx, cmd.ID, string(outcome), now); err != nil {
		return zero, err
	}
	return decision, tx.Commit()
}
