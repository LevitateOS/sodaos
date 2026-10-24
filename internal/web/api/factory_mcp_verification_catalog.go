package api

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
)

// The transport has authenticated the capability; repeat its domain bindings
// before exposing the preparation-owned catalog for a new planning proposal.
func (s *API) factoryApprovedVerificationCatalog(ctx context.Context, cap factory.MCPCapability) (factory.ApprovedCatalog, error) {
	if s.Store == nil || s.Coordinator == nil || cap.Profile != factory.MCPProfileProductLead || cap.State != factory.MCPCapabilityActive || cap.Expired(time.Now()) {
		return nil, errors.New("approved check catalog is unavailable for this worker")
	}
	a, err := s.Store.Assignment(ctx, cap.AssignmentID)
	if err != nil {
		return nil, err
	}
	if a.ID != cap.AssignmentID || a.Run != cap.RunID || a.ProjectID != cap.Project || a.Repository != cap.Repository || a.SourceCommit != cap.SourceDigest || a.Stage != factory.AssignmentAssigned || a.Subject.Kind != factory.SubjectIntent || a.Subject.Intent == nil || a.Subject.SubjectKey(a.Repository) != cap.Subject || strconv.FormatInt(a.NativeRev, 10) != cap.AcceptedRevision {
		return nil, errors.New("planning catalog assignment differs from capability")
	}
	run, err := s.Store.FactoryRun(ctx, cap.RunID)
	if err != nil {
		return nil, err
	}
	if run.ProjectID != a.ProjectID || run.Role != a.Role || run.Outcome != "" || run.Reconciled {
		return nil, errors.New("planning catalog run is no longer live")
	}
	prep, err := s.Store.Preparation(ctx, a.Preparation)
	if err != nil {
		return nil, err
	}
	if prep.Preparation.SetupDigest != cap.PreparationDigest {
		return nil, errors.New("planning catalog preparation differs from capability")
	}
	head, err := s.Store.IntentRevision(ctx, a.Subject.Intent.IntentID, a.Subject.Intent.OwnerID)
	if err != nil {
		return nil, err
	}
	if head.ID != a.Subject.Intent.RevisionID || head.Repository != a.Repository || head.State != "draft" || head.Digest() != a.Subject.Intent.ScopeDigest {
		return nil, errors.New("planning catalog draft changed")
	}
	return s.Coordinator.ApprovedProjectCheckCatalog(ctx, a)
}
