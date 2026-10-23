package api

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
)

type factoryMCPCurrentContextSource interface {
	FactoryCurrentContext(context.Context, factory.MCPCapability, time.Time) (factory.CurrentContext, error)
}

// addCurrentContext joins required current facts without consulting a model or
// changing the selected optional bundle. Explicit gaps stay visible on S0 hold.
func (s *factoryCatalogServer) addCurrentContext(ctx context.Context, read *readContext, body map[string]any, args mcpCallArgs) error {
	source, ok := s.deps.Submissions.(factoryMCPCurrentContextSource)
	if !ok {
		body["current"] = factory.CurrentContext{
			Version: "context-current-v1", Role: factory.CurrentRoleContext{Kind: read.capability.Profile},
			Evidence: factory.CurrentContextEvidence{RequiredStreams: []string{"required_current"}, Availability: "unknown", MissingReasons: []string{"required_current_source_unwired"}},
		}
		markRequiredContextHold(body)
		return nil
	}
	current, err := source.FactoryCurrentContext(ctx, read.capability, s.now())
	if err != nil {
		return err
	}
	if read.capability.Profile == factory.MCPProfileTestPilot {
		if err = s.addCurrentVerification(ctx, read.capability, &current); err != nil {
			return err
		}
	}
	current.Role.Tools = factory.MCPDiscovery(read.capability.Profile)
	for _, tool := range current.Role.Tools {
		if !factory.MCPIsTerminal(tool) || tool == factory.MCPToolFinishIncomplete {
			continue
		}
		shape := rolePayloadSchema(tool)
		if shape == nil {
			shape = libraryPayloadSchema(tool)
		}
		if shape != nil {
			current.Role.ResultTool = tool
			current.Role.ResultSchema, err = json.Marshal(shape)
			if err != nil {
				return err
			}
		}
	}
	page, err := pageCurrentRole(read.capability, current.Role, args, s.deps.ContextCursorKey, s.now())
	if errors.Is(err, errCurrentContextBound) {
		current.Role = factory.CurrentRoleContext{Kind: read.capability.Profile}
		current.Evidence.Availability = "incomplete"
		current.Evidence.MissingReasons = append(current.Evidence.MissingReasons, "required_current_collection_bound")
		page.Availability, page.MissingReason = "unavailable", "required_current_collection_bound"
	} else if err != nil {
		return err
	}
	if current.Run != nil && s.deps.Observed != nil {
		current.Run.ObservedState = "unavailable"
		observation, observeErr := s.observed(read.capability)
		if observeErr == nil && !observation.Stopped && observation.RunID == read.capability.RunID && observation.AssignmentID == read.capability.AssignmentID && observation.Authority == read.capability.Authority {
			current.Run.ObservedState = "confirmed"
		}
	}
	encoded, err := json.Marshal(current)
	if err != nil {
		return err
	}
	if len(encoded) > 8192 {
		current.Role = factory.CurrentRoleContext{Kind: read.capability.Profile, RecordSource: "required_current"}
	}
	body["current"] = current
	body["current_role_page"] = page
	if current.Evidence.Availability != "complete" {
		markRequiredContextHold(body)
	}
	return nil
}

func markRequiredContextHold(body map[string]any) {
	if selection, ok := body["context_selection"].(map[string]any); ok && selection["action"] != "stop" {
		selection["action"], selection["reason_code"] = "hold", "required_current_unavailable"
	}
}

func (s *factoryCatalogServer) addCurrentVerification(ctx context.Context, cap factory.MCPCapability, current *factory.CurrentContext) error {
	missing := func() {
		current.Evidence.Availability = "incomplete"
		current.Evidence.MissingReasons = append(current.Evidence.MissingReasons, "current_native_verification_unavailable")
	}
	if current.Role.Delivery == nil || s.deps.VerificationEvidence == nil {
		missing()
		return nil
	}
	body, err := s.deps.VerificationEvidence(ctx, cap)
	if err != nil || body["available"] != true {
		missing()
		return nil
	}
	raw, err := json.Marshal(body["evidence"])
	if err != nil {
		return err
	}
	var evidence factory.CurrentVerificationEvidence
	if json.Unmarshal(raw, &evidence) != nil || evidence.ValidateFor(*current.Role.Delivery) != nil {
		return errors.New("current native verification is malformed")
	}
	current.Role.Verification = &evidence
	missingReasons := []string{}
	for _, reason := range current.Evidence.MissingReasons {
		if reason != "current_native_verification_unavailable" {
			missingReasons = append(missingReasons, reason)
		}
	}
	current.Evidence.MissingReasons = missingReasons
	if len(missingReasons) == 0 {
		current.Evidence.Availability = "complete"
	}
	return nil
}
