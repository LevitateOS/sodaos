package web

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/levitateos/sodaos/internal/forgejo"
	"github.com/levitateos/sodaos/internal/runners"
	"github.com/levitateos/sodaos/internal/store"
)

// An operator is a configured stable ID, never a Forgejo site-admin flag. Recheck
// provider identity and the original Soda context after external authorization IO.
var errOperatorRequired = errors.New("configured Soda operator required")

func (s *Server) operatorAuthorization(r *http.Request, v store.Session) error {
	if s.Config.OperatorID <= 0 || v.User.ID != s.Config.OperatorID {
		return errOperatorRequired
	}
	grant, err := s.userGrant(r, v)
	if err != nil {
		return err
	}
	if !forgejo.HasScope(grant.Scopes, "read:user") {
		return errRepositoryConsent
	}
	actor, err := s.Forgejo.Current(r.Context(), grant.Access)
	if err != nil {
		return err
	}
	if actor.ID != v.User.ID {
		return errProviderIdentity
	}
	cookie, err := requestCookie(r, sessionCookie)
	if err != nil {
		return store.ErrGrantUnavailable
	}
	if err := s.requireCurrentSession(r.Context(), cookie.Value, v); err != nil {
		return store.ErrGrantUnavailable
	}
	return nil
}

func (s *Server) authorizeOperator(w http.ResponseWriter, r *http.Request, v store.Session) bool {
	if err := s.operatorAuthorization(r, v); err != nil {
		if errors.Is(err, errOperatorRequired) {
			jsonError(w, 403, "operator_required", "Only the configured Soda operator can manage this appliance.")
		} else {
			providerError(w, err)
		}
		return false
	}
	return true
}

func (s *Server) runnerRoutes() {
	s.mux.HandleFunc("/api/settings/runners", s.apiProtected(s.apiRunners, http.MethodGet, http.MethodPost))
	s.mux.HandleFunc("/api/settings/runners/{runner}/{action}", s.apiProtected(s.apiRunnerAction, http.MethodPost))
	s.mux.HandleFunc("GET /settings/runners", s.runnersPage)
}

func (s *Server) createRunner(w http.ResponseWriter, r *http.Request, v store.Session, ctx context.Context) {
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
	cookie, err := requestCookie(r, sessionCookie)
	if err != nil || s.requireCurrentSession(ctx, cookie.Value, v) != nil {
		providerError(w, store.ErrGrantUnavailable)
		return
	}
	err = s.Host.RunnerCreate(ctx, in)
	in.RegistrationToken = ""
	if err != nil {
		runnerUnconfirmed(w)
		return
	}
	jsonResponse(w, 200, runners.MutationResponse{OK: true})
}

func admitUnavailableRunners(w http.ResponseWriter, ids []string, seen map[string]bool) bool {
	for _, id := range ids {
		if runners.ValidateID(id) != nil || seen[id] {
			jsonError(w, 503, "runners_unavailable", "Invalid unavailable runner locator.")
			return false
		}
		seen[id] = true
	}
	return true
}

func admitObservedRunners(w http.ResponseWriter, inventory *runners.Inventory, seen map[string]bool, forgejoURL string) bool {
	for i := range inventory.Runners {
		row := &inventory.Runners[i]
		if runners.ValidateID(row.ID) != nil || seen[row.ID] || row.Capacity != runners.RunnerCapacity {
			jsonError(w, 503, "runners_unavailable", "Invalid local runner observation.")
			return false
		}
		if row.Provider != runners.ProviderForgejo {
			jsonError(w, 503, "runners_unavailable", "Invalid local runner provider.")
			return false
		}
		seen[row.ID] = true
		row.RegistrationURL = forgejoURL
	}
	return true
}

func admitRunnerInventory(w http.ResponseWriter, inventory *runners.Inventory, forgejoURL string) bool {
	if inventory.Runners == nil || inventory.Unavailable == nil || len(inventory.Runners)+len(inventory.Unavailable) > 64 {
		jsonError(w, 503, "runners_unavailable", "Local runner inventory is unavailable; no empty or provider-available state was inferred.")
		return false
	}
	seen := map[string]bool{}
	return admitUnavailableRunners(w, inventory.Unavailable, seen) && admitObservedRunners(w, inventory, seen, forgejoURL)
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
		s.createRunner(w, r, v, ctx)
		return
	}
	inventory, err := s.Host.RunnersList(ctx)
	if err != nil {
		jsonError(w, 503, "runners_unavailable", "Local runner inventory is unavailable; no empty or provider-available state was inferred.")
		return
	}
	if !admitRunnerInventory(w, &inventory, s.Config.ForgejoURL) {
		return
	}
	jsonResponse(w, 200, inventory.Response(s.Config.ForgejoURL))
}

func runnerActionLocator(r *http.Request) bool {
	return runners.ValidateID(r.PathValue("runner")) == nil && r.URL.RawQuery == "" && !r.URL.ForceQuery
}

func knownRunnerAction(action string) bool {
	return action == "start" || action == "stop" || action == "restart" || action == "remove"
}

func confirmRunnerAction(w http.ResponseWriter, r *http.Request, id, action string) bool {
	if action == "remove" {
		var in struct {
			ConfirmID string `json:"confirm_id"`
		}
		if !decodeAPIObject(w, r, &in) {
			return false
		}
		if in.ConfirmID != id {
			jsonError(w, 400, "confirmation_required", "Confirm the exact runner ID and destructive effects.")
			return false
		}
		return true
	}
	var in runners.EmptyRequest
	return decodeAPIObject(w, r, &in)
}

func parseRunnerAction(w http.ResponseWriter, r *http.Request) (string, string, bool) {
	id, action := r.PathValue("runner"), r.PathValue("action")
	if !runnerActionLocator(r) {
		jsonError(w, 400, "invalid_runner", "Invalid runner operation.")
		return "", "", false
	}
	if !knownRunnerAction(action) {
		jsonError(w, 404, "not_found", "Runner operation not found.")
		return "", "", false
	}
	if !confirmRunnerAction(w, r, id, action) {
		return "", "", false
	}
	return id, action, true
}

func (s *Server) apiRunnerAction(w http.ResponseWriter, r *http.Request, v store.Session) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Minute)
	defer cancel()
	r = r.WithContext(ctx)
	if !s.authorizeOperator(w, r, v) {
		return
	}
	id, action, ok := parseRunnerAction(w, r)
	if !ok {
		return
	}
	// Recheck after decoding and confirmation, immediately before dispatch.
	cookie, err := requestCookie(r, sessionCookie)
	if err != nil || s.requireCurrentSession(ctx, cookie.Value, v) != nil {
		providerError(w, store.ErrGrantUnavailable)
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
