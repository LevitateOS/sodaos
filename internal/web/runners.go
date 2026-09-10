package web

import (
	"context"
	"net/http"
	"time"

	"github.com/levitateos/sodaos/internal/forgejo"
	"github.com/levitateos/sodaos/internal/runners"
	"github.com/levitateos/sodaos/internal/store"
)

// An operator is a configured stable ID, never a Forgejo site-admin flag. Recheck
// provider identity and the original Soda context after external authorization IO.
func (s *Server) authorizeOperator(w http.ResponseWriter, r *http.Request, v store.Session) bool {
	if s.Config.OperatorID <= 0 || v.User.ID != s.Config.OperatorID {
		jsonError(w, 403, "operator_required", "Only the configured Soda operator can manage this appliance.")
		return false
	}
	grant, err := s.userGrant(r, v)
	if err != nil {
		providerError(w, err)
		return false
	}
	if !forgejo.HasScope(grant.Scopes, "read:user") {
		providerError(w, errRepositoryConsent)
		return false
	}
	actor, err := s.Forgejo.Current(r.Context(), grant.Access)
	if err != nil {
		providerError(w, err)
		return false
	}
	if actor.ID != v.User.ID {
		providerError(w, errProviderIdentity)
		return false
	}
	cookie, err := requestCookie(r, sessionCookie)
	if err != nil {
		providerError(w, store.ErrGrantUnavailable)
		return false
	}
	if err := s.requireCurrentSession(r.Context(), cookie.Value, v); err != nil {
		providerError(w, store.ErrGrantUnavailable)
		return false
	}
	return true
}
func (s *Server) runnerRoutes() {
	s.mux.HandleFunc("/api/settings/runners", s.apiProtected(s.apiRunners, http.MethodGet, http.MethodPost))
	s.mux.HandleFunc("/api/settings/runners/{runner}/{action}", s.apiProtected(s.apiRunnerAction, http.MethodPost))
	s.mux.HandleFunc("GET /settings/runners", s.runnersPage)
}
func (s *Server) apiRunners(w http.ResponseWriter, r *http.Request, v store.Session) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Minute)
	defer cancel()
	r = r.WithContext(ctx)
	if !s.authorizeOperator(w, r, v) {
		return
	}
	if r.URL.RawQuery != "" || r.URL.ForceQuery {
		jsonError(w, 400, "invalid_query", "Runner operations do not accept query parameters.")
		return
	}
	if r.Method == http.MethodPost {
		var in runners.CreateRequest
		if !decodeAPIObject(w, r, &in) {
			return
		}
		if in.Provider == runners.ProviderForgejo {
			in.RegistrationURL = s.Config.ForgejoInternalURL
		}
		if in.Validate() != nil {
			jsonError(w, 400, "invalid_runner", "Check the runner ID, provider, registration ID, labels and token.")
			return
		}
		err := s.Host.RunnerCreate(ctx, in)
		in.RegistrationToken = ""
		if err != nil {
			runnerUnconfirmed(w)
			return
		}
		jsonResponse(w, 200, runners.MutationResponse{OK: true})
		return
	}
	views, err := s.Host.RunnersList(ctx)
	if err != nil || views == nil || len(views) > 64 {
		jsonError(w, 503, "runners_unavailable", "Local runner inventory is unavailable; no empty or provider-available state was inferred.")
		return
	}
	result := runners.ListResponse{ForgejoURL: s.Config.ForgejoURL, Runners: views, RunnerCount: len(views), TotalCapacity: len(views) * runners.RunnerCapacity}
	for i := range result.Runners {
		row := &result.Runners[i]
		if runners.ValidateID(row.ID) != nil || row.Capacity != runners.RunnerCapacity {
			jsonError(w, 503, "runners_unavailable", "Invalid local runner observation.")
			return
		}
		if row.Provider == runners.ProviderForgejo {
			row.RegistrationURL = s.Config.ForgejoURL
		}
		if row.Service.Active == "active" && row.Service.Sub == "running" {
			result.ActiveListeners++
		}
	}
	jsonResponse(w, 200, result)
}
func (s *Server) apiRunnerAction(w http.ResponseWriter, r *http.Request, v store.Session) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Minute)
	defer cancel()
	r = r.WithContext(ctx)
	if !s.authorizeOperator(w, r, v) {
		return
	}
	id, action := r.PathValue("runner"), r.PathValue("action")
	if runners.ValidateID(id) != nil || r.URL.RawQuery != "" || r.URL.ForceQuery {
		jsonError(w, 400, "invalid_runner", "Invalid runner operation.")
		return
	}
	switch action {
	case "start", "stop", "restart", "remove":
	default:
		jsonError(w, 404, "not_found", "Runner operation not found.")
		return
	}
	var in struct {
		ConfirmID string `json:"confirm_id"`
	}
	if !decodeAPIObject(w, r, &in) {
		return
	}
	if in.ConfirmID != id {
		jsonError(w, 400, "confirmation_required", "Confirm the exact runner ID and operation effects.")
		return
	}
	if s.Host.RunnerAction(ctx, action, runners.RunnerRequest{ID: id}) != nil {
		runnerUnconfirmed(w)
		return
	}
	jsonResponse(w, 200, runners.MutationResponse{OK: true})
}
func runnerUnconfirmed(w http.ResponseWriter) {
	jsonError(w, 502, "runner_unconfirmed", "Operation unconfirmed. Local account, files or listener may have changed; provider registration may remain. Refresh and inspect native provider state before retrying. No automatic rollback or retry occurred.")
}
