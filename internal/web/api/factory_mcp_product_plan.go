package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/factory/control"
)

// checkProductPlanWalkthroughs validates authored experience before freezing a
// Product Lead terminal submission. Older recorded scopes remain readable;
// newly staged scopes cannot report completion without their walkthroughs.
func (s *factoryCatalogServer) checkProductPlanWalkthroughs(ctx context.Context, cap factory.MCPCapability, payload map[string]any) error {
	raw, present := payload["walkthroughs"]
	if s.deps.Assignments == nil {
		if present {
			return errors.New("product walkthrough has no recorded planning scope")
		}
		return nil
	}
	a, err := s.deps.Assignments.Assignment(ctx, cap.AssignmentID)
	if err != nil {
		return err
	}
	in, staged, err := control.ParseStagedPlanningInput(a.Prompt)
	if err != nil {
		return err
	}
	if !present && (!staged || !in.RequireWalkthroughs) {
		return nil
	}
	if !present {
		return errors.New("completed product planning requires authored walkthroughs")
	}
	if !staged || a.Subject.Intent == nil || in.Head.ID != a.Subject.Intent.RevisionID || in.Head.Digest() != a.Subject.Intent.ScopeDigest {
		return errors.New("product walkthrough has no matching planning scope")
	}
	encoded, err := json.Marshal(raw)
	if err != nil {
		return err
	}
	var content factory.ProposalContent
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&content); err != nil {
		return err
	}
	return factory.ValidatePlanningWalkthroughs(in.Head, content)
}
