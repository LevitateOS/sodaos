package api

import (
	"github.com/levitateos/sodaos/internal/web/auth"
	"net/http"

	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
)

func (s *API) checkLifecycleEnvironment(w http.ResponseWriter, r *http.Request) (store.Project, bool) {
	if r.URL.RawQuery != "" || r.URL.ForceQuery {
		auth.JSONError(w, 400, "invalid_request", "No query parameters are accepted.")
		return store.Project{}, false
	}
	p, ok := s.loadEnvironment(w, r)
	if !ok {
		return store.Project{}, false
	}
	if !p.Ready {
		auth.JSONError(w, 409, "not_provisioned", "Provisioning is incomplete; do not repair or recreate it.")
		return store.Project{}, false
	}
	return p, true
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
	cookie, err := auth.RequestCookie(r, auth.SessionCookie)
	if err != nil || s.Auth.RequireCurrentSession(r.Context(), cookie.Value, v) != nil {
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
	p, ok := s.checkLifecycleEnvironment(w, r)
	if !ok {
		return
	}
	action := "inspect"
	if r.Method == "POST" {
		var cleanup func()
		var ok bool
		action, cleanup, ok = s.handleLifecycleMutation(w, r, v, p)
		if !ok {
			return
		}
		if cleanup != nil {
			defer cleanup()
		}
	} else if _, ok := s.authorizeEnvironmentRead(w, r, v, p); !ok {
		return
	}
	result, err := s.Host.Lifecycle(r.Context(), project.Lifecycle{Project: p.ID, Action: action})
	if err != nil {
		auth.JSONError(w, 502, "native_outcome_unconfirmed", "Native state was not confirmed. Refresh or ask the operator to inspect; do not repeat or repair blindly.")
		return
	}
	auth.JSONResponse(w, 200, result)
}
