package api

import (
	"errors"
	"net/http"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/store"
	"github.com/levitateos/sodaos/internal/web/auth"
)

type connectionUsageBudgetRequest struct {
	CommandID        string `json:"command_id"`
	ExpectedRevision int64  `json:"expected_revision"`
	RollingMinutes   int64  `json:"rolling_minutes"`
}

func (s *API) apiFactoryConnectionUsageBudget(w http.ResponseWriter, r *http.Request, v store.Session) {
	if !s.identityAdmission(w, r, v) {
		return
	}
	connectionID := r.PathValue("connection")
	connections, err := s.Identity.Connections(r.Context(), v.User.ID)
	if err != nil {
		identityError(w, err)
		return
	}
	found := false
	for _, connection := range connections {
		if connection.ID == connectionID && connection.OwnerID == v.User.ID {
			found = true
			break
		}
	}
	if !found {
		auth.JSONError(w, http.StatusNotFound, "not_found", "Provider connection not found.")
		return
	}
	if r.Method == http.MethodGet {
		budget, err := s.Store.ConnectionUsageBudget(r.Context(), connectionID)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				auth.JSONError(w, http.StatusNotFound, "not_found", "Connection usage budget not found.")
				return
			}
			auth.JSONError(w, http.StatusServiceUnavailable, "store_unavailable", "Connection usage budget is unavailable.")
			return
		}
		if s.checkLifecycleSession(w, r, v) {
			auth.JSONResponse(w, http.StatusOK, budget)
		}
		return
	}
	var in connectionUsageBudgetRequest
	if !auth.DecodeAPIObject(w, r, &in) || !settingsCommandID(w, in.CommandID) {
		return
	}
	budget := factory.ConnectionUsageBudget{
		Connection: connectionID, Revision: in.ExpectedRevision, RollingMinutes: in.RollingMinutes,
	}
	if err := budget.Validate(); err != nil {
		auth.JSONError(w, http.StatusBadRequest, "invalid_connection_usage_budget", "Connection usage limit is invalid.")
		return
	}
	_, err = s.Coordinator.ApplyConnectionUsageBudget(r.Context(), in.CommandID, factoryPrincipal(v), in.ExpectedRevision, budget)
	if err != nil {
		factoryCommandError(w, err)
		return
	}
	if s.checkLifecycleSession(w, r, v) {
		budget.Revision++
		auth.JSONResponse(w, http.StatusOK, budget)
	}
}
