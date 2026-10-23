package control

import (
	"context"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/store"
)

// PrepareContextBundleSelection records one funded bundle selection
// for a staged capability at preparation time, before the first read.
// Funding is once per immutable capability from the priority call
// budget: a recorded selection reuses its receipt without new spend,
// and a missing or uncertain answer holds the authenticated launch. Failed
// consultations keep their reservation; retries reuse the frozen selection
// without another call. Reads retain required facts and an explicit hold.
func PrepareContextBundleSelection(ctx context.Context, st *store.Store, chooser ProductJudgment, a factory.Assignment, run factory.Run, verify currentVerificationSource) (bundle, reason string, err error) {
	if st == nil {
		return factory.ContextBundleSnapshotHold, "unavailable", ErrPriorityUnavailable
	}
	preparation, _, err := recordedMCPPreparation(ctx, st, a.Preparation)
	if err != nil {
		return factory.ContextBundleSnapshotHold, "unavailable", err
	}
	capability, err := BuildMCPCapability(a, run, preparation)
	if err != nil {
		return factory.ContextBundleSnapshotHold, "unavailable", err
	}
	recorded, err := st.GetMCPCapability(ctx, capability.ID)
	if err != nil || recorded.ScopeDigest() != capability.ScopeDigest() {
		return factory.ContextBundleSnapshotHold, "context_selection_unavailable", ErrPriorityUnavailable
	}
	capability = recorded
	state, attemptEligible, lineageEligible, err := loadContextBundleState(ctx, st, capability, a, verify)
	if err != nil {
		return factory.ContextBundleSnapshotHold, "required_current_unavailable", err
	}
	if state.Current.Availability != "complete" {
		return factory.ContextBundleSnapshotHold, "required_current_unavailable", nil
	}
	if resolution := capability.ContextCurrentResolution; resolution != nil {
		current, currentErr := st.FactoryCurrentContext(ctx, capability, time.Now())
		if currentErr != nil {
			return factory.ContextBundleSnapshotHold, "required_current_unavailable", currentErr
		}
		digest, digestErr := factory.CurrentRequiredSemanticDigest(current.Role)
		if resolution.ValidateFor(capability) != nil || digestErr != nil || digest != resolution.RequiredDigest {
			return factory.ContextBundleSnapshotHold, "context_selection_stale", nil
		}
		return factory.ContextBundleCurrent, "", nil
	}
	if factory.ValidDigest(capability.ContextSelectionSnapshot) {
		receipt, err := st.PriorityJudgment(ctx, capability.ContextSelectionSnapshot)
		if err != nil {
			return factory.ContextBundleSnapshotHold, "context_selection_unavailable", err
		}
		bundle, reason := ContextBundleFromReceipt(receipt, capability, a, attemptEligible, lineageEligible)
		return bundle, reason, nil
	}
	if chooser == nil {
		return factory.ContextBundleSnapshotHold, "context_selection_unavailable", ErrPriorityUnavailable
	}
	coord := &Coordinator{Store: st, Planning: &PlanningJudgments{Chooser: chooser}}
	reserve := func(ctx context.Context, snapshot string) (bool, error) {
		return st.ReserveContextBundleSelection(ctx, capability, snapshot)
	}
	return coord.selectContextBundleState(ctx, state, attemptEligible, lineageEligible, reserve)
}
