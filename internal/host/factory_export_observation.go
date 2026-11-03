package host

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/levitateos/factory/pkg/factory/execution"
	"github.com/levitateos/factory/pkg/project"
)

func (c *Client) ExportCandidate(ctx context.Context, request execution.CandidateRequest) (execution.CandidateBundle, error) {
	return NewFactoryObserver(c, nil).ExportCandidate(ctx, request)
}

func (o *FactoryObserver) ExportCandidate(ctx context.Context, request execution.CandidateRequest) (execution.CandidateBundle, error) {
	var result execution.CandidateBundle
	native := project.FactoryExport{Project: request.Address.Workspace, ID: request.Address.Run, Role: request.Role, Preparation: request.Preparation, Candidate: request.Candidate}
	if err := native.Validate(); err != nil {
		return result, err
	}
	exporter, ok := o.host.(interface {
		FactoryExport(context.Context, project.FactoryExport) (project.FactoryExportState, error)
	})
	if !ok {
		return result, errors.New("appliance candidate exporter is unavailable")
	}
	state, err := exporter.FactoryExport(ctx, native)
	if err != nil {
		switch {
		case errors.Is(err, project.ErrFactoryExportCandidate):
			return result, fmt.Errorf("%w: %w", execution.ErrCandidateInvalid, err)
		case errors.Is(err, project.ErrFactoryExportBounds):
			return result, fmt.Errorf("%w: %w", execution.ErrCandidateBounds, err)
		default:
			return result, runObservationError(err)
		}
	}
	if state.ID != request.Address.Run || state.Project != request.Address.Workspace || state.Candidate != request.Candidate {
		return result, execution.ErrObservation
	}
	if state.Bundle == "" || len(state.Bundle) > 8<<20 {
		return result, execution.ErrCandidateBounds
	}
	bundle, err := base64.StdEncoding.DecodeString(state.Bundle)
	if err != nil || len(bundle) == 0 || len(bundle) > 4<<20 {
		return result, execution.ErrCandidateBounds
	}
	return execution.CandidateBundle{Address: request.Address, Candidate: state.Candidate, Bundle: bundle}, nil
}
