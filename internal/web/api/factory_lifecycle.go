package api

import (
	"errors"
	"net/http"

	"github.com/levitateos/sodaos/internal/web/auth"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/factory/control"
	"github.com/levitateos/sodaos/internal/store"
)

// lifecycleControlError maps intervention failures to status codes. Unknown
// runs and commands report 404; unsettled runs refuse with 409 and name
// the blocking runs; unconfirmed native effects report 502, never success.
func lifecycleControlError(w http.ResponseWriter, err error) {
	var pending control.ErrPendingRuns
	switch {
	case errors.Is(err, control.ErrNotFound):
		auth.JSONError(w, 404, "not_found", "Factory run not found.")
	case errors.As(err, &pending):
		auth.JSONResponse(w, 409, map[string]any{"error": "pending_runs", "runs": pending.Runs, "message": "Recorded runs are not settled; stop or reconcile them first."})
	case errors.Is(err, control.ErrTakeoverFailed):
		auth.JSONError(w, 502, "takeover_unconfirmed", "Takeover was not confirmed; refresh and retry.")
	case errors.Is(err, store.ErrTakeoverConflict):
		auth.JSONError(w, 409, "takeover_conflict", "Takeover identity reused for different content.")
	case errors.Is(err, store.ErrExplicitRetryIneligible):
		auth.JSONError(w, 409, "retry_ineligible", "The prior attempt is not eligible for retry; refresh its settlement and current inputs.")
	case errors.Is(err, store.ErrContextResolutionIneligible):
		auth.JSONError(w, 409, "context_resolution_ineligible", "The held current context changed or cannot be selected; refresh this run.")
	case errors.Is(err, store.ErrCompletionRetryIneligible):
		auth.JSONError(w, 409, "completion_retry_ineligible", "The completion cycle changed or its prior work has not settled; refresh its details.")
	default:
		factoryCommandError(w, err)
	}
}

type factoryActionsRequest struct {
	CommandID  string                          `json:"command_id"`
	Action     string                          `json:"action"`
	Completion *factory.CompletionRetryRequest `json:"completion,omitempty"`
}

// apiFactoryActions admits repository pause/resume and completion Retry from
// the current code-write maintainer. Pause stops outstanding runs; resume
// reopens dispatch; completion Retry retains its prior decision history.
func (s *API) apiFactoryActions(w http.ResponseWriter, r *http.Request, v store.Session) {
	repository, ok := s.factoryRepository(w, r, v)
	if !ok {
		return
	}
	if _, err := s.executionRepository(r, v, repository); err != nil {
		reportExecutionAuthorityError(w, err)
		return
	}
	if !s.checkLifecycleSession(w, r, v) {
		return
	}
	var in factoryActionsRequest
	if !auth.DecodeAPIObject(w, r, &in) {
		return
	}
	if !factory.ValidRepositoryAction(in.Action) {
		auth.JSONError(w, 400, "invalid_action", "Choose pause, resume or retry completion review.")
		return
	}
	if in.Action == factory.ActionCompletionRetry {
		if !settingsCommandID(w, in.CommandID) {
			return
		}
	} else if in.CommandID != "" {
		auth.JSONError(w, 400, "invalid_request", "Pause and resume do not use command IDs.")
		return
	}
	if (in.Action == factory.ActionCompletionRetry) != (in.Completion != nil) ||
		(in.Completion != nil && in.Completion.Validate() != nil) {
		auth.JSONError(w, 400, "invalid_request", "Completion retry requires its exact event and prior cycle.")
		return
	}
	if s.Coordinator == nil {
		auth.JSONError(w, 500, "coordinator_unavailable", "Factory coordination is unavailable.")
		return
	}
	switch in.Action {
	case factory.ActionCompletionRetry:
		receipt, err := s.Coordinator.RetryCompletion(r.Context(), in.CommandID, factoryPrincipal(v), repository, *in.Completion)
		if err != nil {
			lifecycleControlError(w, err)
			return
		}
		auth.JSONResponse(w, 200, receipt)
	case factory.ActionPause:
		receipt, err := s.Coordinator.PauseRepository(r.Context(), factoryPrincipal(v), repository)
		if err != nil {
			lifecycleControlError(w, err)
			return
		}
		auth.JSONResponse(w, 200, receipt)
	case factory.ActionResume:
		receipt, err := s.Coordinator.ResumeRepository(r.Context(), repository)
		if err != nil {
			lifecycleControlError(w, err)
			return
		}
		auth.JSONResponse(w, 200, receipt)
	}
}

type factoryRunActionsRequest struct {
	CommandID string                         `json:"command_id,omitempty"`
	Action    string                         `json:"action"`
	Context   *factory.ContextCurrentRequest `json:"context,omitempty"`
}

// factoryActionRun resolves the path run under current code-write
// authority. Absent runs and projects report 404 without disclosure.
func (s *API) factoryActionRun(w http.ResponseWriter, r *http.Request, v store.Session) (factory.Run, store.Project, bool) {
	runID := r.PathValue("runID")
	if !factory.ValidID(runID) {
		auth.JSONError(w, 404, "not_found", "Factory run not found.")
		return factory.Run{}, store.Project{}, false
	}
	run, err := s.Store.FactoryRun(r.Context(), runID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			auth.JSONError(w, 404, "not_found", "Factory run not found.")
		} else {
			auth.JSONError(w, 503, "store_unavailable", "Could not read factory run.")
		}
		return factory.Run{}, store.Project{}, false
	}
	p, err := s.Store.Project(r.Context(), run.ProjectID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			auth.JSONError(w, 404, "not_found", "Factory run not found.")
		} else {
			auth.JSONError(w, 503, "store_unavailable", "Could not read factory run.")
		}
		return factory.Run{}, store.Project{}, false
	}
	if _, err = s.executionRepository(r, v, p.RepositoryID); err != nil {
		reportExecutionAuthorityError(w, err)
		return factory.Run{}, store.Project{}, false
	}
	if !s.checkLifecycleSession(w, r, v) {
		return factory.Run{}, store.Project{}, false
	}
	return run, p, true
}

// apiFactoryRunActions admits run stop/retry/takeover from the current
// code-write maintainer. Stop retires through the same path as the
// operator command; retry records a queued decision; takeover additionally
// requires project membership and copies into the member's own checkout.
func (s *API) apiFactoryRunActions(w http.ResponseWriter, r *http.Request, v store.Session) {
	run, p, ok := s.factoryActionRun(w, r, v)
	if !ok {
		return
	}
	var in factoryRunActionsRequest
	if !auth.DecodeAPIObject(w, r, &in) {
		return
	}
	if !factory.ValidRunAction(in.Action) && in.Action != factory.ActionContextCurrent {
		auth.JSONError(w, 400, "invalid_action", "Choose stop, retry, delivery repair or takeover.")
		return
	}
	if in.Action == factory.ActionRetry && !settingsCommandID(w, in.CommandID) {
		return
	}
	if (in.Action == factory.ActionContextCurrent) != (in.Context != nil) {
		auth.JSONError(w, 400, "invalid_request", "Current context selection requires its exact saved scope and fingerprint.")
		return
	}
	if s.Coordinator == nil {
		auth.JSONError(w, 500, "coordinator_unavailable", "Factory coordination is unavailable.")
		return
	}
	switch in.Action {
	case factory.ActionContextCurrent:
		receipt, err := s.Coordinator.UseCurrentContext(r.Context(), factoryPrincipal(v), run.ID, *in.Context)
		if err != nil {
			lifecycleControlError(w, err)
			return
		}
		auth.JSONResponse(w, 200, receipt)
	case factory.ActionStop:
		result, err := s.Coordinator.Stop(r.Context(), run.ID)
		if err != nil {
			lifecycleControlError(w, err)
			return
		}
		auth.JSONResponse(w, 200, result)
	case factory.ActionRetry:
		decision, err := s.Coordinator.RetryRun(r.Context(), in.CommandID, factoryPrincipal(v), run.ID)
		if err != nil {
			lifecycleControlError(w, err)
			return
		}
		auth.JSONResponse(w, 200, decision)
	case factory.ActionDeliveryRepair:
		receipt, err := s.Coordinator.RequestDeliveryRepair(r.Context(), factoryPrincipal(v), run.ID)
		if err != nil {
			lifecycleControlError(w, err)
			return
		}
		auth.JSONResponse(w, 200, receipt)
	case factory.ActionTakeover:
		login, err := s.Store.MemberLogin(r.Context(), p.ID, v.User.ID)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				auth.JSONError(w, 403, "membership_required", "Join the project before taking over its work.")
			} else {
				auth.JSONError(w, 503, "store_unavailable", "Could not read membership.")
			}
			return
		}
		record, err := s.Coordinator.TakeoverRun(r.Context(), run.ID, login)
		if err != nil {
			lifecycleControlError(w, err)
			return
		}
		auth.JSONResponse(w, 200, record)
	}
}
