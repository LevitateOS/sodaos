package api

import (
	"errors"
	"net/http"
	"strings"

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
	default:
		factoryCommandError(w, err)
	}
}

type factoryActionsRequest struct {
	CommandID string `json:"command_id"`
	Action    string `json:"action"`
}

// apiFactoryActions admits repository pause/resume from the current
// code-write maintainer. Pause latches dispatch closed and stops
// outstanding runs; resume reopens the gate after every run settles.
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
	if !auth.DecodeAPIObject(w, r, &in) || !settingsCommandID(w, in.CommandID) {
		return
	}
	if !factory.ValidRepositoryAction(in.Action) {
		auth.JSONError(w, 400, "invalid_action", "Choose pause or resume.")
		return
	}
	if s.Coordinator == nil {
		auth.JSONError(w, 500, "coordinator_unavailable", "Factory coordination is unavailable.")
		return
	}
	switch in.Action {
	case factory.ActionPause:
		receipt, err := s.Coordinator.PauseRepository(r.Context(), in.CommandID, factoryPrincipal(v), repository)
		if err != nil {
			lifecycleControlError(w, err)
			return
		}
		auth.JSONResponse(w, 200, receipt)
	case factory.ActionResume:
		receipt, err := s.Coordinator.ResumeRepository(r.Context(), in.CommandID, factoryPrincipal(v), repository)
		if err != nil {
			lifecycleControlError(w, err)
			return
		}
		auth.JSONResponse(w, 200, receipt)
	}
}

type factoryRunActionsRequest struct {
	CommandID string `json:"command_id"`
	Action    string `json:"action"`
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
	if !auth.DecodeAPIObject(w, r, &in) || !settingsCommandID(w, in.CommandID) {
		return
	}
	if !factory.ValidRunAction(in.Action) {
		auth.JSONError(w, 400, "invalid_action", "Choose stop, retry or takeover.")
		return
	}
	if s.Coordinator == nil {
		auth.JSONError(w, 500, "coordinator_unavailable", "Factory coordination is unavailable.")
		return
	}
	switch in.Action {
	case factory.ActionStop:
		cmd := factory.Command{ID: in.CommandID, Type: factory.CommandStop, Target: run.ID, Principal: factoryPrincipal(v), Digest: factory.CommandDigest(factory.CommandStop, run.ID)}
		receipt, err := s.Coordinator.Stop(r.Context(), cmd)
		if err != nil {
			lifecycleControlError(w, err)
			return
		}
		auth.JSONResponse(w, 200, receipt)
	case factory.ActionRetry:
		decision, err := s.Coordinator.RetryRun(r.Context(), in.CommandID, factoryPrincipal(v), run.ID)
		if err != nil {
			lifecycleControlError(w, err)
			return
		}
		auth.JSONResponse(w, 200, decision)
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
		record, err := s.Coordinator.TakeoverRun(r.Context(), in.CommandID, factoryPrincipal(v), run.ID, login)
		if err != nil {
			lifecycleControlError(w, err)
			return
		}
		auth.JSONResponse(w, 200, record)
	}
}

// commandRepository resolves the repository authorizing one durable
// command for read-back. Operator stop targets carry a bare run identity;
// settings and lifecycle targets name their repository, run or project.
// Targets without a repository stay undisclosed.
func (s *API) commandRepository(w http.ResponseWriter, r *http.Request, v store.Session, cmd factory.Command) (int64, bool) {
	target := cmd.Target
	if rest, ok := strings.CutPrefix(target, "repository/"); ok {
		id, _, _ := strings.Cut(rest, "/")
		repository, valid := auth.PositiveID(id)
		if !valid {
			auth.JSONError(w, 404, "not_found", "Command not found.")
			return 0, false
		}
		return repository, true
	}
	if target == "capacity" {
		if v.User.ID != s.Config.OperatorID {
			auth.JSONError(w, 403, "operator_required", "Only the configured Soda operator can read appliance capacity commands.")
			return 0, false
		}
		return 0, true
	}
	runID := target
	if rest, ok := strings.CutPrefix(target, "run/"); ok {
		runID = rest
	}
	projectID := ""
	if rest, ok := strings.CutPrefix(target, "project/"); ok {
		projectID, _, _ = strings.Cut(rest, "/")
	}
	if projectID == "" {
		if !factory.ValidID(runID) {
			auth.JSONError(w, 404, "not_found", "Command not found.")
			return 0, false
		}
		run, err := s.Store.FactoryRun(r.Context(), runID)
		if err != nil {
			auth.JSONError(w, 404, "not_found", "Command not found.")
			return 0, false
		}
		projectID = run.ProjectID
	}
	p, err := s.Store.Project(r.Context(), projectID)
	if err != nil {
		auth.JSONError(w, 404, "not_found", "Command not found.")
		return 0, false
	}
	return p.RepositoryID, true
}

// apiFactoryCommand reads back one durable command outcome after a pending
// command or lost reply. Visibility of the bound repository authorizes
// the read; the outcome carries receipts and reason codes, never secrets.
func (s *API) apiFactoryCommand(w http.ResponseWriter, r *http.Request, v store.Session) {
	if r.URL.RawQuery != "" || r.URL.ForceQuery {
		auth.JSONError(w, 400, "invalid_request", "No query parameters are accepted.")
		return
	}
	commandID := r.PathValue("commandID")
	if !factory.ValidID(commandID) {
		auth.JSONError(w, 404, "not_found", "Command not found.")
		return
	}
	cmd, err := s.Store.FactoryCommand(r.Context(), commandID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			auth.JSONError(w, 404, "not_found", "Command not found.")
		} else {
			auth.JSONError(w, 503, "store_unavailable", "Could not read command.")
		}
		return
	}
	repository, ok := s.commandRepository(w, r, v, cmd)
	if !ok {
		return
	}
	if repository > 0 {
		if _, err = s.visibleRepository(r, v, repository); err != nil {
			auth.ProviderError(w, err)
			return
		}
	}
	auth.JSONResponse(w, 200, cmd)
}
