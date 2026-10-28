package api

import (
	"context"
	"encoding/json"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
)

// bindPayload fills the bindings SodaOS already records for this
// assignment, so a worker never copies them back: the issue plan's recorded
// fields, the review cycle tuple and clear coverage, the correction's finding
// digest, the task manifest digest with the observed task output and child,
// the committed final candidate, and the app-verification delivery and
// input digest. Any value the worker sent is overwritten. A missing source
// leaves the payload as it is, and settlement reports the gap.
func (s *factoryCatalogServer) bindPayload(ctx context.Context, capability factory.MCPCapability, tool factory.MCPTool, payload map[string]any) {
	if payload == nil {
		return
	}
	switch tool {
	case factory.MCPToolFinishIssuePlan:
		s.bindIssuePlan(ctx, capability, payload)
	case factory.MCPToolFinishReview:
		source, ok := s.deps.Submissions.(factoryMCPCurrentContextSource)
		if !ok {
			return
		}
		current, err := source.FactoryCurrentContext(ctx, capability, s.now())
		if err != nil || current.Role.Review == nil {
			return
		}
		review := current.Role.Review
		payload["tuple"] = map[string]any{"cycle_id": review.Cycle.ID, "head": review.Cycle.Head, "base_oid": review.Cycle.BaseOID}
		if review.ClearCoverageDigest != "" {
			payload["coverage"] = review.ClearCoverageDigest
		}
	case factory.MCPToolFinishCorrection:
		if s.deps.Assignments == nil {
			return
		}
		if a, err := s.deps.Assignments.Assignment(ctx, capability.AssignmentID); err == nil && a.ReviewContext != nil {
			payload["finding_digest"] = a.ReviewContext.Cycle.FindingDigest
		}
	case factory.MCPToolSubmitTaskOutput:
		if s.deps.Runs == nil {
			return
		}
		if run, err := s.deps.Runs.FactoryRun(ctx, capability.RunID); err == nil {
			payload["manifest_digest"] = project.ExecutionManifestDigest(run.ExecutionManifest)
			s.bindTaskOutput(ctx, run, payload)
		}
	case factory.MCPToolFinishImplement:
		if s.deps.Runs == nil {
			return
		}
		if run, err := s.deps.Runs.FactoryRun(ctx, capability.RunID); err == nil && run.FinalCheckpoint != nil &&
			run.FinalCheckpoint.Status == "committed" && factory.ValidCommit(run.FinalCheckpoint.Candidate) {
			payload["candidate"] = run.FinalCheckpoint.Candidate
		}
	case factory.MCPToolFinishAppVerify, factory.MCPToolRecordJourneyResult:
		payload["input_digest"] = capability.ScopeDigest()
		if s.deps.Assignments == nil {
			return
		}
		if a, err := s.deps.Assignments.Assignment(ctx, capability.AssignmentID); err == nil && a.Subject.Delivery != nil {
			payload["delivery_id"] = a.Subject.Delivery.DeliveryID
			if tool == factory.MCPToolRecordJourneyResult {
				payload["source_commit"] = a.SourceCommit
				payload["preparation"] = a.Preparation
				payload["manifest_digest"] = a.Subject.Delivery.VerificationManifestDigest
			}
		}
	}
}

// bindTaskOutput fills the observed output digest and native child of the
// named task from its output-ready plan-task receipt: the same record the
// checkpoint drive compares the submission against.
func (s *factoryCatalogServer) bindTaskOutput(ctx context.Context, run factory.Run, payload map[string]any) {
	taskID, _ := payload["task_id"].(string)
	source, ok := s.deps.Submissions.(checkpointContextSource)
	if !ok || taskID == "" || len(run.ExecutionManifest) == 0 {
		return
	}
	manifest, err := project.ParseExecutionManifest(run.ExecutionManifest)
	if err != nil {
		return
	}
	var plan factory.InvestigationPlan
	if json.Unmarshal(manifest.Plan, &plan) != nil {
		return
	}
	receipts, err := source.PlanTaskReceipts(ctx, plan.AttemptRoot, plan.ID)
	if err != nil {
		return
	}
	for _, receipt := range receipts {
		if receipt.TaskID != taskID || receipt.State != store.PlanTaskOutputReady {
			continue
		}
		payload["output_digest"] = receipt.OutputDigest
		if receipt.Child != "" {
			payload["native_child_ref"] = receipt.Child
		} else {
			delete(payload, "native_child_ref")
		}
	}
}
