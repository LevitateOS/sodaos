package api

import (
	"context"
	"errors"
	"net/http"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/store"
)

// spaceAuthorityView exposes the effective factory verdict for the row's
// repository: whether dispatch may proceed and which authorization is
// missing or withdrawn. It carries reason codes only, never credentials.
type spaceAuthorityView struct {
	Missing      []string `json:"missing"`
	Effective    bool     `json:"effective"`
	DispatchOpen bool     `json:"dispatch_open"`
}

func (s *API) resolveSpaceAuthority(request *http.Request, v store.Session, p store.Project) (environmentReader, bool, bool) {
	reader, err := s.readEnvironmentAuthority(request, v, p)
	if err != nil {
		complete := errors.Is(err, errRepositoryDenied)
		return reader, false, complete
	}
	return reader, true, !reader.authorityUnavailable
}

func (s *API) inspectSpaceAuthority(ctx context.Context, repository int64) *spaceAuthorityView {
	if s.Coordinator == nil {
		return nil
	}
	effective, err := s.Coordinator.EffectiveAuthority(ctx, repository)
	if err != nil {
		return nil
	}
	return &spaceAuthorityView{Missing: effective.Missing, Effective: effective.Effective, DispatchOpen: effective.DispatchOpen}
}

// spaceControlView exposes the intervention state for the row's factory:
// whether dispatch is open and why it closed, whether the owner paused,
// and how many recorded runs are unsettled. It carries reason codes and
// counts only, never credentials or run contents.
type spaceControlView struct {
	WithdrawalCause string `json:"withdrawal_cause,omitempty"`
	UnsettledRuns   int    `json:"unsettled_runs"`
	Paused          bool   `json:"paused"`
	DispatchOpen    bool   `json:"dispatch_open"`
}

func (s *API) inspectSpaceControl(ctx context.Context, runs []factory.Run, runsKnown bool, projectID string, repository int64) *spaceControlView {
	// An unreadable run listing leaves the unsettled count unknown: no
	// control rather than an authoritative zero.
	if !runsKnown {
		return nil
	}
	view := &spaceControlView{}
	if policy, err := s.Store.RepositoryPolicy(ctx, repository); err == nil {
		view.Paused = policy.Paused
	} else if !errors.Is(err, store.ErrNotFound) {
		return nil
	}
	open, _, withdrawal, err := s.Store.DispatchState(ctx, repository)
	if err != nil {
		return nil
	}
	view.DispatchOpen = open
	if !open {
		if len(withdrawal.ActiveCauses) > 0 {
			view.WithdrawalCause = withdrawal.ActiveCauses[0]
		}
	}
	for _, run := range runs {
		if run.ProjectID == projectID && !run.Reconciled {
			view.UnsettledRuns++
		}
	}
	return view
}
