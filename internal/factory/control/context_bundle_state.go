package control

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/store"
)

type contextSelectionCurrent struct {
	Availability   string   `json:"availability"`
	MissingReasons []string `json:"missing_reasons"`
	RequiredDigest string   `json:"required_digest"`
	RequiredBytes  int      `json:"required_bytes"`
	Tasks          int      `json:"tasks"`
	Findings       int      `json:"findings"`
	Submissions    int      `json:"submissions"`
	Stage          string   `json:"stage"`
	Active         *bool    `json:"active"`
	Settled        *bool    `json:"settled"`
}

type contextHistoryPreview struct {
	Relation   string          `json:"relation"`
	ID         string          `json:"id"`
	Phase      string          `json:"phase,omitempty"`
	Outcome    factory.Outcome `json:"outcome,omitempty"`
	Reason     string          `json:"reason,omitempty"`
	Reconciled *bool           `json:"reconciled,omitempty"`
}

type contextBundleDescription struct {
	Eligible        bool     `json:"eligible"`
	RequiredStreams []string `json:"required_streams"`
	HistoryRecords  int      `json:"history_records"`
}

type currentVerificationSource func(context.Context, factory.Assignment) (factory.CurrentVerificationEvidence, error)

func loadContextBundleState(ctx context.Context, st *store.Store, cap factory.MCPCapability, a factory.Assignment, verify currentVerificationSource) (contextBundleState, bool, bool, error) {
	state := contextBundleStateFor(cap, a)
	current, err := st.FactoryCurrentContext(ctx, cap, time.Now())
	if err != nil {
		return state, false, false, err
	}
	if cap.Profile == factory.MCPProfileTestPilot && verify != nil && current.Role.Delivery != nil {
		evidence, err := verify(ctx, a)
		if err == nil && evidence.ValidateFor(*current.Role.Delivery) == nil {
			current.Role.Verification = &evidence
			missing := []string{}
			for _, reason := range current.Evidence.MissingReasons {
				if reason != "current_native_verification_unavailable" {
					missing = append(missing, reason)
				}
			}
			current.Evidence.MissingReasons = missing
			if len(missing) == 0 {
				current.Evidence.Availability = "complete"
			}
		}
	}
	raw, err := json.Marshal(current.Role)
	if err != nil {
		return state, false, false, err
	}
	if len(raw) > 1024*1024 {
		return state, false, false, errors.New("required current context exceeds its bound")
	}
	digest := sha256.Sum256(raw)
	state.Current = contextSelectionCurrent{Availability: current.Evidence.Availability, MissingReasons: current.Evidence.MissingReasons, RequiredDigest: hex.EncodeToString(digest[:]), RequiredBytes: len(raw), Submissions: len(current.Role.Submissions), Stage: a.Stage}
	if current.Role.Plan != nil {
		state.Current.Tasks = len(current.Role.Plan.Tasks)
	}
	if current.Role.Review != nil {
		state.Current.Findings = len(current.Role.Review.CarriedFindings)
	}
	if current.Run != nil {
		state.Current.Active, state.Current.Settled = &current.Run.Active, &current.Run.Settled
	}
	attempt := false
	for _, id := range a.RunHistory {
		if len(state.HistoryPreview) >= 16 {
			break
		}
		view, err := st.FactoryRunView(ctx, id)
		if err != nil || view.Attempt != a.ID || view.Repository != a.Repository {
			continue
		}
		run, err := st.FactoryRun(ctx, id)
		if err != nil || run.ID != id || run.ProjectID != a.ProjectID {
			continue
		}
		// Packet staging records RunHistory and Started before any native
		// launch. Neither that bookkeeping nor a confirmed-unused terminal
		// run provides execution continuity. Require a durable native
		// observation (retained after capability revocation), delegation or
		// checkpoint before offering the attempt bundle.
		liveObserved, observedErr := st.MCPRunWasObserved(ctx, run.ID, a.ID, a.ProjectID)
		if observedErr != nil {
			return state, false, false, observedErr
		}
		if !liveObserved && !(run.CredentialDelegated && run.IdentityBinding != nil) && run.FinalCheckpoint == nil {
			continue
		}
		attempt = true
		state.HistoryPreview = append(state.HistoryPreview, contextHistoryPreview{Relation: "own_assignment", ID: run.ID, Phase: a.Phase, Outcome: run.Outcome, Reconciled: &run.Reconciled})
	}
	lineage := false
	if a.ParentAssignment != "" {
		parent, err := st.Assignment(ctx, a.ParentAssignment)
		if err == nil && factory.ContextParentVisible(a, parent) {
			lineage = true
			state.HistoryPreview = append(state.HistoryPreview, contextHistoryPreview{Relation: "parent", ID: parent.ID, Outcome: parent.Outcome, Reason: parent.Reason})
		}
	}
	state.Bundles = describeContextBundles(attempt, lineage, len(state.HistoryPreview))
	return state, attempt, lineage, nil
}

func describeContextBundles(attempt, lineage bool, history int) map[string]contextBundleDescription {
	return map[string]contextBundleDescription{
		factory.ContextBundleCurrent:      {Eligible: true, RequiredStreams: []string{"required_current"}},
		factory.ContextBundleAttempt:      {Eligible: attempt, RequiredStreams: []string{"required_current"}, HistoryRecords: history},
		factory.ContextBundleLineage:      {Eligible: lineage, RequiredStreams: []string{"required_current"}, HistoryRecords: history},
		factory.ContextBundleSnapshotHold: {Eligible: true, RequiredStreams: []string{"required_current"}},
	}
}
