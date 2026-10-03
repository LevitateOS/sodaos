package api

import (
	"context"
	"errors"
	"net/http"

	"github.com/levitateos/sodaos/internal/web/auth"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/factory/control"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
)

func (s *API) loadLifecycleEnvironment(w http.ResponseWriter, r *http.Request) (store.Project, bool) {
	if r.URL.RawQuery != "" || r.URL.ForceQuery {
		auth.JSONError(w, 400, "invalid_request", "No query parameters are accepted.")
		return store.Project{}, false
	}
	return s.loadEnvironment(w, r)
}

type lifecycleRequest struct {
	Action      string `json:"action"`
	ConfirmStop bool   `json:"confirm_stop"`
}

func decodeLifecycleRequest(w http.ResponseWriter, r *http.Request) (string, bool) {
	var in lifecycleRequest
	if !auth.DecodeAPIObject(w, r, &in) {
		return "", false
	}
	if (in.Action != "start" && in.Action != "stop") || (in.Action == "stop" && !in.ConfirmStop) || (in.Action == "start" && in.ConfirmStop) {
		auth.JSONError(w, 400, "invalid_action", "Choose Start or explicitly confirm Stop for everyone.")
		return "", false
	}
	return in.Action, true
}

func (s *API) authorizeLifecycleOperator(w http.ResponseWriter, r *http.Request, v store.Session, repositoryID int64) bool {
	if v.User.ID == s.Config.OperatorID {
		return true
	}
	access, err := s.visibleRepository(r, v, repositoryID)
	if err != nil {
		auth.ProviderError(w, err)
		return false
	}
	allowed, err := s.environmentAdministrator(r, access)
	if err != nil {
		auth.ProviderError(w, err)
		return false
	}
	if !allowed {
		auth.JSONError(w, 403, "administrator_required", "Only the current project administrator or Soda operator can start/stop it.")
		return false
	}
	return true
}

func (s *API) checkLifecycleSession(w http.ResponseWriter, r *http.Request, v store.Session) bool {
	if !s.extensionSessionCurrent(r.Context(), r, v) {
		auth.JSONError(w, 401, "unauthorized", "Soda context changed. Reconnect before acting.")
		return false
	}
	return true
}

func (s *API) acquireLifecycleStop(w http.ResponseWriter, projectID string) (func(), bool) {
	s.terminalMu.Lock()
	if s.TerminalStopping == nil {
		s.TerminalStopping = make(map[string]bool)
	}
	if s.TerminalStopping[projectID] {
		s.terminalMu.Unlock()
		auth.JSONError(w, 409, "stop_pending", "A Stop is already pending; inspect its outcome.")
		return nil, false
	}
	s.TerminalStopping[projectID] = true
	for _, peer := range s.TerminalPeers {
		if peer.Project == projectID {
			peer.Cancel()
		}
	}
	s.terminalMu.Unlock()
	cleanup := func() {
		s.terminalMu.Lock()
		delete(s.TerminalStopping, projectID)
		s.terminalMu.Unlock()
	}
	return cleanup, true
}

func (s *API) handleLifecycleMutation(w http.ResponseWriter, r *http.Request, v store.Session, p store.Project) (string, func(), bool) {
	action, ok := decodeLifecycleRequest(w, r)
	if !ok {
		return "", nil, false
	}
	// Start on a retained reservation finishes an interrupted creation;
	// stopping unconfirmed provisioning stays refused.
	if action == "stop" && !p.Ready {
		auth.JSONError(w, 409, "not_provisioned", "Provisioning is incomplete; do not repair or recreate it.")
		return "", nil, false
	}
	if !s.authorizeLifecycleOperator(w, r, v, p.RepositoryID) {
		return "", nil, false
	}
	if !s.checkLifecycleSession(w, r, v) {
		return "", nil, false
	}
	if action == "stop" {
		cleanup, ok := s.acquireLifecycleStop(w, p.ID)
		if !ok {
			return "", nil, false
		}
		return action, cleanup, true
	}
	return action, nil, true
}

func (s *API) apiLifecycle(w http.ResponseWriter, r *http.Request, v store.Session) {
	p, ok := s.loadLifecycleEnvironment(w, r)
	if !ok {
		return
	}
	if r.Method == "POST" {
		action, cleanup, ok := s.handleLifecycleMutation(w, r, v, p)
		if !ok {
			return
		}
		if cleanup != nil {
			defer cleanup()
		}
		if action == "stop" {
			s.apiLifecycleStop(w, r, v, p)
		} else {
			s.apiLifecycleStart(w, r, v, p)
		}
		return
	}
	if _, ok := s.authorizeEnvironmentRead(w, r, v, p); !ok {
		return
	}
	// Inspection stays available while provisioning is incomplete: it
	// reports host evidence and finishes the retained reservation once
	// that evidence proves the admitted creation.
	result, err := s.Host.Lifecycle(r.Context(), project.Lifecycle{Project: p.ID, Action: "inspect"})
	if err != nil {
		auth.JSONError(w, 502, "native_outcome_unconfirmed", "Native state was not confirmed. Refresh or ask the operator to inspect; do not repeat or repair blindly.")
		return
	}
	if !p.Ready {
		_, _ = s.reconcileProvisioning(r.Context(), p, result.Environment)
	}
	auth.JSONResponse(w, 200, result)
}

// lifecycleStopControlView reports the factory coordination around an
// explicit Project stop: the withdrawal that closed dispatch first, every
// outstanding run's stop outcome, and the maintenance hold state. Uncertain
// runs stay fenced inside the receipt; the stop itself still proceeds.
type lifecycleStopControlView struct {
	Withdrawal factory.Withdrawal       `json:"withdrawal"`
	Runs       []factory.RunStopOutcome `json:"runs"`
	Hold       bool                     `json:"hold"`
	HoldSynced bool                     `json:"hold_synced"`
}

type lifecycleStopView struct {
	project.LifecycleState
	Control lifecycleStopControlView `json:"control"`
}

// apiLifecycleStop coordinates an explicit Project stop: withdraw dispatch
// and stop outstanding runs first, set the maintenance hold, then stop the
// host unit. A hold failure never wedges the stop; it is reported instead.
// The admitted sequence runs detached from the request so response
// cancellation cannot strand it between factory and host effects.
func (s *API) apiLifecycleStop(w http.ResponseWriter, r *http.Request, v store.Session, p store.Project) {
	if s.Coordinator == nil {
		auth.JSONError(w, 500, "coordinator_unavailable", "Factory coordination is unavailable.")
		return
	}
	opCtx, cancel := operationContext(r.Context(), stopOperationTimeout)
	defer cancel()
	withdrawal, outcomes, err := s.Coordinator.StopProject(opCtx, factoryPrincipal(v), p.ID)
	if err != nil {
		if errors.Is(err, control.ErrNotFound) {
			auth.JSONError(w, 404, "not_found", "Environment not found.")
		} else {
			auth.JSONError(w, 503, "store_unavailable", "Could not withdraw factory dispatch.")
		}
		return
	}
	held, synced := s.lifecycleStopHold(opCtx, p.ID)
	result, err := s.Host.Lifecycle(opCtx, project.Lifecycle{Project: p.ID, Action: "stop"})
	if err != nil {
		auth.JSONError(w, 502, "native_outcome_unconfirmed", "Native state was not confirmed. Refresh or ask the operator to inspect; do not repeat or repair blindly.")
		return
	}
	auth.JSONResponse(w, 200, lifecycleStopView{LifecycleState: result, Control: lifecycleStopControlView{Withdrawal: withdrawal, Runs: outcomes, Hold: held, HoldSynced: synced}})
}

// lifecycleStopHold sets the maintenance hold marker-first, following the
// preparation hold protocol. An already held project needs no marker
// write; any failure reports unsynced and lets the stop proceed.
func (s *API) lifecycleStopHold(ctx context.Context, projectID string) (held, synced bool) {
	current, err := s.Store.MaintenanceHold(ctx, projectID)
	if err == nil && current.Hold {
		return true, true
	}
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		return false, false
	}
	var revision int64
	if err == nil {
		revision = current.Revision
	}
	if _, err = s.Host.HoldPreparation(ctx, project.PrepareHold{Project: projectID, Hold: true, Revision: revision + 1}); err != nil {
		return false, false
	}
	if err = s.Store.SaveMaintenanceHold(ctx, project.MaintenanceHold{Project: projectID, Revision: revision, Hold: true}); err != nil {
		return false, false
	}
	return true, true
}

type lifecycleStartView struct {
	project.LifecycleState
	Verification factory.StartVerification `json:"verification"`
}

// apiLifecycleStart starts the host unit and then proves no old run
// revived. Leases stay closed and the maintenance hold keeps its state;
// releasing it stays an explicit hold control. Starting a retained
// reservation finishes an interrupted creation from the same evidence.
func (s *API) apiLifecycleStart(w http.ResponseWriter, r *http.Request, v store.Session, p store.Project) {
	if s.Coordinator == nil {
		auth.JSONError(w, 500, "coordinator_unavailable", "Factory coordination is unavailable.")
		return
	}
	opCtx, cancel := operationContext(r.Context(), startOperationTimeout)
	defer cancel()
	result, err := s.Host.Lifecycle(opCtx, project.Lifecycle{Project: p.ID, Action: "start"})
	if err != nil {
		auth.JSONError(w, 502, "native_outcome_unconfirmed", "Native state was not confirmed. Refresh or ask the operator to inspect; do not repeat or repair blindly.")
		return
	}
	if !p.Ready {
		if updated, finished := s.reconcileProvisioning(opCtx, p, result.Environment); finished {
			p = updated
		} else if provisioningConfirmed(p, result.Environment) {
			auth.JSONResponse(w, 503, struct {
				Error       apiError        `json:"error"`
				Environment EnvironmentView `json:"environment"`
			}{apiError{"result_not_saved", "Native start was confirmed, but its result was not saved. Inspect the retained environment; do not recreate it."}, EnvironmentDTO(p)})
			return
		}
	}
	verification, err := s.Coordinator.VerifyProjectStart(opCtx, p.ID)
	if err != nil {
		if errors.Is(err, control.ErrNotFound) {
			auth.JSONError(w, 404, "not_found", "Environment not found.")
		} else {
			auth.JSONError(w, 503, "store_unavailable", "Project started but revival could not be verified; refresh and reconcile.")
		}
		return
	}
	auth.JSONResponse(w, 200, lifecycleStartView{LifecycleState: result, Verification: verification})
}
