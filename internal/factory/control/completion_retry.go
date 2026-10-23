package control

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/store"
)

// RetryCompletion exposes the existing explicit D13 successor admission to
// the current maintainer. It never changes a decision, releases the fence,
// reopens the completed issue allowance or funds a call itself.
func (c *Coordinator) RetryCompletion(ctx context.Context, commandID, principal string, repository int64, request factory.CompletionRetryRequest) (factory.CompletionRetryDecision, error) {
	var zero factory.CompletionRetryDecision
	if c.Store == nil || repository <= 0 || request.Validate() != nil {
		return zero, store.ErrCompletionRetryIneligible
	}
	bounded, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	payload, _ := json.Marshal(request)
	cmd := factory.Command{ID: commandID, Type: factory.CommandCompletionRetry, Principal: principal,
		Target: fmt.Sprintf("repository/%d/completion/%s", repository, request.EventID), Payload: string(payload)}
	cmd.Digest = factory.SettingsDigest(cmd.Type, cmd.Target, cmd.Payload)
	if err := cmd.Validate(); err != nil {
		return zero, err
	}
	bounded, unlock := c.progressPass(bounded)
	defer unlock()
	// Immutable lost-reply recovery remains available after the successor
	// advances. Current native repository/session authority is checked by API.
	if saved, err := c.Store.FactoryCommand(bounded, cmd.ID); err == nil {
		if saved.Digest != cmd.Digest || saved.Principal != cmd.Principal || saved.Type != cmd.Type || saved.Target != cmd.Target {
			return zero, store.ErrCommandConflict
		}
		if json.Unmarshal([]byte(saved.Outcome), &zero) != nil || zero.Validate() != nil || zero.Repository != repository || zero.EventID != request.EventID || zero.PriorCycle != request.ExpectedPriorCycle || zero.CommandID != cmd.ID {
			return factory.CompletionRetryDecision{}, store.ErrCompletionRetryIneligible
		}
		return zero, nil
	} else if !errors.Is(err, store.ErrNotFound) {
		return zero, err
	}
	prior, err := c.Store.ReassessmentRecord(bounded, request.ExpectedPriorCycle)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return zero, store.ErrCompletionRetryIneligible
		}
		return zero, err
	}
	if prior.Cycle.Repository != repository || prior.Cycle.EventID != request.EventID {
		return zero, store.ErrCompletionRetryIneligible
	}
	effective, err := c.EffectiveAuthority(bounded, repository)
	if err != nil {
		return zero, err
	}
	if !effective.Effective && !onlyDispatchClosed(effective) {
		return zero, ErrIneffectiveAuthority
	}
	retry := factory.ReassessmentRetry{CommandID: commandID, SuccessorCycleID: commandID, EventID: request.EventID, ExpectedPriorCycle: request.ExpectedPriorCycle}
	if err := c.checkCompletionRetry(bounded, retry); err != nil {
		return zero, err
	}
	return c.Store.RecordCompletionRetryCommand(bounded, cmd, repository, request, time.Now())
}
