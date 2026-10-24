package api

import (
	"context"
	"errors"

	"github.com/levitateos/sodaos/internal/factory"
)

func (s *API) factoryVerificationEvidence(ctx context.Context, capability factory.MCPCapability) (map[string]any, error) {
	if capability.Profile != factory.MCPProfileTestPilot || s.Store == nil || s.Coordinator == nil {
		return nil, errors.New("application verification evidence is unavailable for this worker")
	}
	assignment, err := s.Store.Assignment(ctx, capability.AssignmentID)
	if err != nil {
		return nil, err
	}
	if assignment.Run != capability.RunID || assignment.ProjectID != capability.Project || assignment.Repository != capability.Repository {
		return nil, errors.New("verification assignment does not match this worker")
	}
	evidence, err := s.Coordinator.ApplicationVerificationEvidence(ctx, assignment)
	if err != nil {
		return map[string]any{"available": false, "reason": err.Error(), "next_step": "Keep the adopted approved check definitions current, or supply the fixed check.sh soda-app-verification declaration; prepare the exact integrated source for the reviewer and resolve missing delivered work or failed checks."}, nil
	}
	outcome := "native_check_passed"
	if evidence.ExecutionRequired {
		outcome = "native_preparation_check_passed"
		if evidence.ExecutionReceipt != nil {
			outcome = "native_manifest_passed"
		}
	}
	return map[string]any{"available": true, "evidence": evidence, "delivery_id": assignment.Subject.Delivery.DeliveryID, "input_digest": capability.ScopeDigest(), "outcome": outcome}, nil
}
