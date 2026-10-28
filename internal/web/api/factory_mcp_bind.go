package api

import (
	"context"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
)

// bindPayload fills the bindings SodaOS already records for this
// assignment, so a worker never copies them back: the issue plan's recorded
// fields, the review cycle tuple and clear coverage, the correction's finding
// digest, the task manifest digest, and the app-verification delivery and
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
	case factory.MCPToolSubmitTaskOutput, factory.MCPToolRequestHandoff:
		if s.deps.Runs == nil {
			return
		}
		if run, err := s.deps.Runs.FactoryRun(ctx, capability.RunID); err == nil {
			payload["manifest_digest"] = project.ExecutionManifestDigest(run.ExecutionManifest)
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
