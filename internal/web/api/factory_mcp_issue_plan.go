package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/factory/control"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/strictjson"
)

// checkNewFinish refuses a new terminal finish that settlement could not
// settle, before its immutable proposal freezes the assignment. It runs the
// settler's own mapping, so the worker sees the exact defect while it can
// still correct the payload and finish again. Retained proposals replay
// through the store without reaching this check.
func (s *factoryCatalogServer) checkNewFinish(ctx context.Context, capability factory.MCPCapability, submission factory.MCPSubmission, payload map[string]any) error {
	if !factory.MCPIsTerminal(submission.Tool) {
		return nil
	}
	if submission.Tool == factory.MCPToolFinishProductPlan {
		if err := s.checkManifestProposal(ctx, capability, payload); err != nil {
			return err
		}
	}
	if s.deps.Assignments == nil {
		return errors.New("assignment unavailable; no terminal proposal recorded")
	}
	assigned, err := s.deps.Assignments.Assignment(ctx, submission.AssignmentID)
	if err != nil {
		return err
	}
	return control.CheckMCPTerminalFinish(assigned, submission, s.now())
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

// bindIssuePlan fills the plan fields this assignment already records,
// then its digest, so a worker never hand-copies 64-digit identities.
// Workers dropped or changed digits copying them from factory.context
// and spent their finish calls on the refusals. A plan that does not
// decode is left untouched for the validation refusal to name.
func (s *factoryCatalogServer) bindIssuePlan(ctx context.Context, capability factory.MCPCapability, payload map[string]any) {
	tasks, ok := payload["tasks"].(map[string]any)
	if !ok || s.deps.Assignments == nil {
		return
	}
	assigned, err := s.deps.Assignments.Assignment(ctx, capability.AssignmentID)
	if err != nil || assigned.Phase != project.ExecutionInvestigate {
		return
	}
	tasks["attempt_root"] = assigned.AttemptRoot
	tasks["source_commit"] = assigned.SourceCommit
	tasks["preparation"] = assigned.Preparation
	tasks["scope_digest"] = capability.ScopeDigest()
	tasks["input_digest"] = assigned.PromptSHA
	tasks["digest"] = ""
	encoded, err := json.Marshal(tasks)
	if err != nil {
		return
	}
	var plan factory.InvestigationPlan
	if err := strictjson.Decode(bytes.NewReader(encoded), &plan); err != nil {
		return
	}
	if plan.Generation == 0 {
		plan.IntegrationCommit = plan.SourceCommit
		tasks["integration_commit"] = plan.SourceCommit
	}
	if digest, err := factory.PlanDigest(plan); err == nil {
		tasks["digest"] = digest
	}
}
