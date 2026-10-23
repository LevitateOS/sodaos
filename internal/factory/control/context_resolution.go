package control

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/store"
)

type ContextSelectionView struct {
	Fingerprint    string                            `json:"fingerprint"`
	ScopeDigest    string                            `json:"scope_digest"`
	RequiredDigest string                            `json:"required_digest,omitempty"`
	Availability   string                            `json:"availability"`
	Reason         string                            `json:"reason"`
	Action         string                            `json:"action"`
	CanUseCurrent  bool                              `json:"can_use_current"`
	Resolution     *factory.ContextCurrentResolution `json:"resolution,omitempty"`
}

// ContextSelectionStatus is a spend-free owner projection of the retained
// selection and complete current records, including its next permitted action.
func ContextSelectionStatus(ctx context.Context, st *store.Store, run factory.Run) (*ContextSelectionView, error) {
	cap, err := st.GetMCPCapability(ctx, run.ID)
	if errors.Is(err, store.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	a, err := st.Assignment(ctx, cap.AssignmentID)
	if err != nil {
		return nil, err
	}
	state, attempt, lineage, err := loadContextBundleState(ctx, st, cap, a, nil)
	if err != nil {
		return nil, err
	}
	current, err := st.FactoryCurrentContext(ctx, cap, time.Now())
	if err != nil {
		return nil, err
	}
	digest, err := factory.CurrentRequiredSemanticDigest(current.Role)
	if err != nil {
		return nil, err
	}
	view := &ContextSelectionView{Fingerprint: cap.ContextSelectionSnapshot, ScopeDigest: cap.ScopeDigest(), RequiredDigest: digest, Availability: state.Current.Availability, Action: "hold", Reason: "context_selection_unavailable"}
	if cap.ContextCurrentResolution != nil && cap.ContextCurrentResolution.ValidateFor(cap) == nil && cap.ContextCurrentResolution.RequiredDigest == digest && state.Current.Availability == "complete" {
		view.Action, view.Reason, view.Resolution = "serve", "", cap.ContextCurrentResolution
		return view, nil
	}
	receipt, err := st.PriorityJudgment(ctx, cap.ContextSelectionSnapshot)
	if err != nil {
		return view, nil
	}
	bundle, reason := ContextBundleFromReceipt(receipt, cap, a, attempt, lineage)
	view.Reason = reason
	if bundle != factory.ContextBundleSnapshotHold && reason == "" {
		view.Action = "serve"
	}
	view.CanUseCurrent = run.ContextPending && !run.Reconciled && run.Outcome == "" && !run.CredentialDelegated && run.IdentityBinding == nil && cap.State == factory.MCPCapabilityStaged && !cap.Expired(time.Now()) && state.Current.Availability == "complete" && reason == "context_selection_attention" && cap.ContextCurrentResolution == nil
	return view, nil
}

func (c *Coordinator) UseCurrentContext(ctx context.Context, commandID, principal, runID string, request factory.ContextCurrentRequest) (factory.ContextCurrentResolution, error) {
	var zero factory.ContextCurrentResolution
	run, err := c.Store.FactoryRun(ctx, runID)
	if err != nil {
		return zero, err
	}
	cap, err := c.Store.GetMCPCapability(ctx, runID)
	if err != nil {
		return zero, err
	}
	payload, _ := json.Marshal(request)
	cmd := factory.Command{ID: commandID, Type: factory.CommandContextCurrent, Target: "run/" + runID, Principal: principal, Payload: string(payload)}
	cmd.Digest = factory.SettingsDigest(cmd.Type, cmd.Target, cmd.Payload)
	// Exact lost-reply recovery remains available after activation/expiry.
	if saved, err := c.Store.FactoryCommand(ctx, commandID); err == nil {
		if saved.Digest != cmd.Digest || saved.Principal != principal || saved.Type != cmd.Type || saved.Target != cmd.Target {
			return zero, store.ErrCommandConflict
		}
		if json.Unmarshal([]byte(saved.Outcome), &zero) != nil || zero.ValidateFor(cap) != nil {
			return zero, store.ErrContextResolutionIneligible
		}
		return zero, nil
	} else if !errors.Is(err, store.ErrNotFound) {
		return zero, err
	}
	view, err := ContextSelectionStatus(ctx, c.Store, run)
	if err != nil {
		return zero, err
	}
	if view == nil || !view.CanUseCurrent || view.Fingerprint != request.Fingerprint || view.ScopeDigest != request.ScopeDigest || view.RequiredDigest != request.RequiredDigest {
		return zero, store.ErrContextResolutionIneligible
	}
	a, err := c.Store.Assignment(ctx, cap.AssignmentID)
	if err != nil {
		return zero, err
	}
	_, wait, terminal := pendingContextLaunch(ctx, c.dispatchDeps(), a, run)
	if wait != nil || terminal {
		return zero, store.ErrContextResolutionIneligible
	}
	return c.Store.RecordContextCurrentCommand(ctx, cmd, cap, request, time.Now())
}
