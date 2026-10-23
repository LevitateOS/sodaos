package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
	"github.com/levitateos/sodaos/internal/strictjson"
)

// An invalid terminal must remain correctable before its immutable proposal
// freezes the assignment. Retained proposals still replay through the store;
// this validation never replaces or repairs their original bytes.
func (s *factoryCatalogServer) validateIssuePlanSubmission(ctx context.Context, submission factory.MCPSubmission) error {
	if submission.Tool != factory.MCPToolFinishIssuePlan {
		return nil
	}
	if _, err := s.deps.Submissions.MCPReceipt(ctx, submission.AssignmentID, submission.SubmissionID); err == nil {
		return nil
	} else if !errors.Is(err, store.ErrNotFound) {
		return err
	}
	if terminal, frozen, err := s.deps.Submissions.TerminalMCPProposal(ctx, submission.AssignmentID); err != nil {
		return err
	} else if frozen && terminal.Tool == submission.Tool && terminal.InputDigest == submission.PayloadHash {
		return nil
	}
	if s.deps.Assignments == nil {
		return errors.New("issue plan assignment unavailable; no terminal proposal recorded")
	}
	assigned, err := s.deps.Assignments.Assignment(ctx, submission.AssignmentID)
	if err != nil {
		return err
	}
	if assigned.Phase != project.ExecutionInvestigate {
		return errors.New("issue plan does not answer an investigation assignment; no terminal proposal recorded")
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(submission.Payload, &fields); err != nil {
		return err
	}
	var plan factory.InvestigationPlan
	if err := strictjson.Decode(bytes.NewReader(fields["tasks"]), &plan); err != nil {
		return fmt.Errorf("issue plan tasks must contain the complete typed InvestigationPlan: %w; no terminal proposal recorded", err)
	}
	expectedDigest, err := factory.PlanDigest(plan)
	if err != nil {
		return err
	}
	refuse := func(reason string) error {
		return fmt.Errorf("issue plan refused: %s; expected_plan_digest=%s for these exact typed plan fields (recompute after editing); no terminal proposal recorded", reason, expectedDigest)
	}
	if err := plan.Validate(); err != nil {
		return refuse(err.Error())
	}
	if plan.AttemptRoot != assigned.AttemptRoot || plan.SourceCommit != assigned.SourceCommit || plan.Preparation != assigned.Preparation {
		return refuse("attempt_root, source_commit and preparation must equal this assignment's recorded bindings")
	}
	var approach string
	if err := json.Unmarshal(fields["approach"], &approach); err != nil || approach != plan.Approach {
		return refuse("approach must equal tasks.approach")
	}
	return nil
}

// Accepted means the immutable proposal was consumed. The result explicitly
// distinguishes successful work from a blocked or invalid consumed report.
func (s *factoryCatalogServer) receiptBody(ctx context.Context, assignment string, receipt factory.MCPReceipt) map[string]any {
	body := map[string]any{
		"submission_id": receipt.SubmissionID, "state": receipt.State, "reason": receipt.Reason,
		"feed_post_id": receipt.FeedPostID, "library_item_id": receipt.LibraryItemID, "artifact_id": receipt.ArtifactID,
	}
	if factory.MCPIsTerminal(receipt.Tool) && receipt.State == factory.MCPReceiptAccepted && s.deps.Assignments != nil {
		body["settlement"] = map[string]any{"availability": "unavailable"}
		if a, err := s.deps.Assignments.Assignment(ctx, assignment); err == nil && a.Stage == factory.AssignmentFinished {
			settlement := map[string]any{"availability": "complete", "outcome": a.Outcome, "reason": a.Reason, "reported": a.Result != nil && a.Result.Reported}
			if a.Result != nil {
				settlement["status"] = a.Result.Status
				settlement["summary"] = a.Result.Summary
			}
			body["settlement"] = settlement
		}
	}
	return body
}
