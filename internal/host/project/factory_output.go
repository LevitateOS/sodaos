package project

import (
	"context"
	"encoding/base64"
	"errors"
	"time"

	"github.com/levitateos/sodaos/internal/identity"
	domain "github.com/levitateos/sodaos/internal/project"
)

// Output reports one bounded slice of a run's recorded CLI output with its
// run/process binding. It never starts, resumes or mutates. A run without a
// recorded binding reports its phase with no bytes; a run whose recorded
// container incarnation no longer resolves refuses with identity.ErrStale
// rather than serving another execution's bytes.
func (f *Factory) Output(ctx context.Context, req domain.FactoryOutput) (domain.FactoryOutputState, error) {
	var empty domain.FactoryOutputState
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
	state := domain.FactoryOutputState{
		ID: req.ID, Project: req.Project, Phase: receipt.Phase,
		Terminal: receiptTerminal(receipt.Phase), Reason: receipt.Reason,
		ExitCode: receipt.ExitCode,
	}
	if receipt.Binding == nil {
		return state, nil
	}
	// Container, unit and invocation describe the recorded process binding
	// together; before a binding exists all three stay empty.
	state.Container, state.Unit, state.Invocation = receipt.Binding.Project, domain.FactoryUnitName(req.ID), receipt.Binding.InvocationID
	if receipt.Phase == domain.FactoryApproved || receipt.Phase == domain.FactoryRunning {
		liveCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		state.Live = f.terminal.FactoryCodexLive(liveCtx, *receipt.Binding)
		cancel()
	}
	slice, err := f.terminal.FactoryCodexOutput(ctx, receipt.Run.Project, receipt.Binding, req.Offset, req.Limit)
	if err != nil {
		if errors.Is(err, identity.ErrStale) {
			return empty, identity.ErrStale
		}
		return empty, err
	}
	state.Total, state.Offset, state.Truncated, state.Gap = slice.Total, slice.Offset, slice.Truncated, slice.Gap
	state.Next = slice.Offset + int64(len(slice.Data))
	if len(slice.Data) > 0 {
		state.Data = base64.StdEncoding.EncodeToString(slice.Data)
	}
	return state, nil
}
