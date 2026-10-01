package project

import (
	"context"
	"encoding/base64"
	"errors"

	"github.com/levitateos/sodaos/internal/identity"
	domain "github.com/levitateos/sodaos/internal/project"
)

// Export reads one settled run's exact candidate as a bounded Git bundle
// from its recorded role checkout. Role and preparation must match the
// receipt, the run must be terminal, and its container incarnation must
// still resolve. The controller supplies the candidate from its recorded
// assignment result. Only Git objects accompany the bundle; publisher
// validation remains responsible for candidate ancestry and policy.
func (f *Factory) Export(ctx context.Context, req domain.FactoryExport) (domain.FactoryExportState, error) {
	var empty domain.FactoryExportState
	if err := req.Validate(); err != nil {
		return empty, err
	}
	file, err := f.lockRun(ctx, req.Project, req.ID)
	if err != nil {
		return empty, err
	}
	receipt, exists, err := f.loadReceipt(req.Project, req.ID)
	_ = file.Close()
	if err != nil {
		return empty, err
	}
	if !exists {
		return empty, identity.ErrNotFound
	}
	if !receiptTerminal(receipt.Phase) || receipt.Run.Validate() != nil || receipt.Binding == nil {
		return empty, errors.New("factory run is not settled for export")
	}
	if receipt.Run.Role != req.Role || receipt.Run.Preparation != req.Preparation {
		return empty, identity.ErrDenied
	}
	bundle, err := f.terminal.FactoryExportBundle(ctx, req.Project, receipt.Binding.Project, req.Role, req.Preparation, req.Candidate)
	if err != nil {
		return empty, err
	}
	if len(bundle) == 0 || len(bundle) > domain.MaxFactoryExportBundle {
		return empty, domain.ErrFactoryExportBounds
	}
	return domain.FactoryExportState{
		ID: req.ID, Project: req.Project, Phase: receipt.Phase,
		Container: receipt.Binding.Project, Candidate: req.Candidate,
		Bundle: base64.StdEncoding.EncodeToString(bundle),
	}, nil
}
