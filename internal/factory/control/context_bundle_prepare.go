package control

import (
	"context"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/store"
)

// PrepareContextBundleSelection records one funded bundle selection
// for a staged capability at preparation time, before the first read.
// Funding is once per selection fingerprint from the priority call
// budget: a recorded selection reuses its receipt without new spend,
// and an exhausted budget holds unfunded. Best-effort like capability
// staging: any refusal or unavailable consultation leaves no receipt
// and never fails the launch — reads fall back to the widest eligible
// bundle.
func PrepareContextBundleSelection(ctx context.Context, st *store.Store, journal PriorityJournal, chooser ProductJudgment, a factory.Assignment, run factory.Run) (bundle, reason string, err error) {
	if st == nil || journal == nil || chooser == nil {
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
	parentLoaded := false
	if a.ParentAssignment != "" {
		_, err := st.Assignment(ctx, a.ParentAssignment)
		parentLoaded = err == nil
	}
	attemptEligible, lineageEligible := factory.BundleEligibility(a, parentLoaded)
	coord := &Coordinator{Store: st, Planning: &PlanningJudgments{Chooser: chooser}}
	reserve := func(ctx context.Context, snapshot string) (bool, error) {
		return journal.ReservePriorityJudgment(ctx, snapshot, 1)
	}
	return coord.SelectContextBundle(ctx, capability, a, attemptEligible, lineageEligible, reserve)
}
