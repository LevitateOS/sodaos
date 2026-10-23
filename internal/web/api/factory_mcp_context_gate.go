package api

import (
	"context"
	"errors"

	"github.com/levitateos/sodaos/internal/factory"
)

func (s *factoryCatalogServer) requireCurrentWork(ctx context.Context, cap factory.MCPCapability) error {
	a, err := s.deps.Assignments.Assignment(ctx, cap.AssignmentID)
	if err != nil {
		return err
	}
	read := &readContext{capability: cap, assignment: a}
	if a.ParentAssignment != "" {
		parent, err := s.deps.Assignments.Assignment(ctx, a.ParentAssignment)
		if err == nil && factory.ContextParentVisible(a, parent) {
			read.parent, read.parentLoaded = parent, true
		}
	}
	attempt, lineage := factory.BundleEligibility(a, read.parentLoaded)
	bundle, reason, _ := s.foregroundReadBundle(ctx, read, attempt, lineage)
	if bundle == factory.ContextBundleSnapshotHold || reason != "" {
		return errors.New("context selection holds new work; read factory.context and report incomplete or blocked")
	}
	body := map[string]any{}
	if err = s.addCurrentContext(ctx, read, body, mcpCallArgs{}); err != nil {
		return err
	}
	current, ok := body["current"].(factory.CurrentContext)
	if !ok || current.Evidence.Availability != "complete" {
		return errors.New("required current context holds new work; read factory.context and report incomplete or blocked")
	}
	return nil
}

func (s *factoryCatalogServer) requireNewSubmissionContext(ctx context.Context, cap factory.MCPCapability, submission factory.MCPSubmission) error {
	if submission.Tool == factory.MCPToolFinishIncomplete || factory.MCPIsCommunication(submission.Tool) || submission.Tool == factory.MCPToolResolvePost {
		return nil
	}
	// Persistence remains authoritative for byte conflicts. These lookups
	// exempt only recovery of existing immutable actions, never a new effect.
	if _, err := s.deps.Submissions.MCPReceipt(ctx, submission.AssignmentID, submission.SubmissionID); err == nil {
		return nil
	}
	if terminal, frozen, err := s.deps.Submissions.TerminalMCPProposal(ctx, submission.AssignmentID); err != nil {
		return err
	} else if frozen && terminal.Tool == submission.Tool && terminal.InputDigest == submission.PayloadHash {
		return nil
	}
	return s.requireCurrentWork(ctx, cap)
}
