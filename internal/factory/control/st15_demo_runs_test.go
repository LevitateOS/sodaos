package control_test

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/factory/control"
	"github.com/levitateos/sodaos/internal/forgejo"
)

// publishChildRun drives the existing publication producer and resolves its
// launch only from the matching persisted assignment/run bindings. The
// allowance deadline bounds preparation retries without starting a new root
// or replenishing the owner's budget; the child run's own recorded deadline
// is checked as soon as the run appears.
func (fx *st15Fixture) publishChildRun(role, candidate string) (factory.Assignment, factory.Run, error) {
	fx.t.Helper()
	allowance, err := fx.db.AttemptAllowance(fx.ctx, fx.cfg.Repository, fx.issueAIndex)
	if err != nil || allowance.RootAssignment != fx.assignA || allowance.Closed || !allowance.Active {
		return factory.Assignment{}, factory.Run{}, fmt.Errorf("ST15 child attempt unavailable: %+v %v", allowance, err)
	}
	remaining := allowance.RemainingSeconds(time.Now())
	if remaining <= 0 {
		return factory.Assignment{}, factory.Run{}, fmt.Errorf("ST15 child attempt budget exhausted")
	}
	deadline := time.Now().Add(time.Duration(remaining) * time.Second)
	ctx, cancel := context.WithDeadline(fx.ctx, deadline)
	defer cancel()
	for time.Now().Before(deadline) {
		report := fx.coord.PublishPass(ctx)
		for _, entry := range report.Errors {
			if entry.Reason != "native_unavailable" {
				return factory.Assignment{}, factory.Run{}, fmt.Errorf("ST15 publication child errors: %+v", report)
			}
		}
		runs, err := fx.db.FactoryRuns(ctx, 1000)
		if err != nil {
			return factory.Assignment{}, factory.Run{}, err
		}
		for _, run := range runs {
			if run.Role != role || run.InputSHA != candidate || !time.Now().Before(run.Deadline) {
				continue
			}
			assignment, err := fx.db.AssignmentByRun(ctx, run.ID)
			if err != nil || assignment.ID == fx.assignA || assignment.PublicationAssignment != fx.assignA ||
				assignment.AttemptRoot != allowance.RootAssignment || assignment.Role != role || assignment.SourceCommit != candidate {
				continue
			}
			return assignment, run, nil
		}
		select {
		case <-ctx.Done():
			return factory.Assignment{}, factory.Run{}, ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
	return factory.Assignment{}, factory.Run{}, fmt.Errorf("ST15 %s child was not recorded before the existing attempt budget elapsed", role)
}

// stopSettled retires one run through the production stop path, retrying
// while retirement reports uncertain: the product contract converges a
// repeated stop once racing retirement settles, so a bounded retry follows
// production semantics. Anything still uncertain after the bound fails.
func (fx *st15Fixture) stopSettled(runID string) control.StopReceipt {
	fx.t.Helper()
	if _, err := fx.db.FactoryRun(fx.ctx, runID); err != nil {
		fx.t.Fatalf("ST15 run %s record: %v", runID, err)
	}
	// Retirement remains available after execution expiry. Keep one bounded
	// cleanup window across all retries, while preserving an earlier caller
	// deadline; this context grants no new execution time.
	ctx, cancel := context.WithTimeout(fx.ctx, 2*time.Minute)
	defer cancel()
	var receipt control.StopReceipt
	for attempt := 0; ; attempt++ {
		cmd := factory.Command{
			ID: factory.NewID(), Type: factory.CommandStop, Target: runID,
			Principal: "soda-maintainer", Digest: factory.CommandDigest(factory.CommandStop, runID),
		}
		var err error
		receipt, err = fx.coord.Stop(ctx, cmd)
		if err != nil {
			fx.t.Fatal(err)
		}
		if !receipt.Uncertain || attempt >= 5 {
			break
		}
		fx.t.Logf("ST15 run %s stop uncertain (attempt %d), retrying", runID, attempt+1)
		wait := time.NewTimer(5 * time.Second)
		select {
		case <-ctx.Done():
			wait.Stop()
			fx.t.Fatalf("ST15 run %s stop cleanup deadline: %v", runID, ctx.Err())
		case <-wait.C:
		}
	}
	if !receipt.Confirmed {
		fx.t.Fatalf("ST15 run %s retired uncertain: %s", runID, receipt.Reason)
	}
	return receipt
}

// settleDispatched settles one production-dispatched run through the
// production stop path and returns its recorded assignment and summary.
// The fixture only observes the result.
func (fx *st15Fixture) settleDispatched(runID string) (factory.Assignment, string) {
	fx.t.Helper()
	receipt := fx.stopSettled(runID)
	assignment, err := fx.db.AssignmentByRun(fx.ctx, runID)
	if err != nil {
		fx.t.Fatalf("ST15 run %s assignment: %v", runID, err)
	}
	if assignment.Stage != factory.AssignmentFinished {
		fx.t.Fatalf("ST15 run %s assignment unsettled: %+v", runID, assignment)
	}
	st15Receipt(fx.t, "run-"+runID, map[string]any{
		"outcome": receipt.Outcome, "assignment": assignment.ID, "attempts": assignment.Attempts,
	})
	return assignment, receipt.Reason
}

// submitReview submits one settled reviewer run's genuine verdict through
// the production review path and returns the adopted outcome and event.
func (fx *st15Fixture) submitReview(runID string) (factory.ReviewOutcome, string) {
	fx.t.Helper()
	adopted, err := fx.coord.SubmitReviewForRun(fx.ctx, runID)
	if err != nil {
		fx.t.Fatalf("ST15 review submit: %v", err)
	}
	if adopted.ReviewerID != fx.cfg.ReviewerID {
		fx.t.Fatalf("ST15 review adopted by a foreign reviewer: %+v", adopted)
	}
	return adopted, adopted.Event
}

// seedCIStatus records one native commit status (the fixture CI verdict)
// through the ordinary REST path, exactly like ST11.
func (fx *st15Fixture) seedCIStatus(head, contextName, state, description string) {
	fx.t.Helper()
	var status struct {
		ID int64 `json:"id"`
	}
	fx.api(fx.cfg.CreatorTokenFile, http.MethodPost, "/api/v1/repos/"+fx.cfg.Owner+"/"+fx.cfg.Repo+"/statuses/"+head,
		map[string]any{"state": state, "context": contextName, "description": description}, &status)
	if status.ID <= 0 {
		fx.t.Fatalf("ST15 CI status %s/%s unrecorded", contextName, state)
	}
}

var _ = forgejo.ContentDigest
