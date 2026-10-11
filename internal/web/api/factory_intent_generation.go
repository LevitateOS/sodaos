package api

import (
	"net/http"

	"github.com/levitateos/sodaos/internal/store"
	"github.com/levitateos/sodaos/internal/web/auth"
)

func (s *API) apiIntentGenerate(w http.ResponseWriter, r *http.Request, v store.Session) {
	if !s.checkLifecycleSession(w, r, v) {
		return
	}
	intent, err := s.Store.IntentRevision(r.Context(), r.PathValue("intentID"), v.User.ID)
	if err != nil {
		factoryCommandError(w, err)
		return
	}
	if intent.State != "accepted" {
		auth.JSONError(w, 409, "intent_not_accepted", "Review the product direction before generating work.")
		return
	}
	if !s.factoryOwner(w, r, v, intent.Repository) {
		return
	}
	if s.Coordinator == nil || s.Coordinator.Graph == nil {
		auth.JSONError(w, 503, "native_graph_unavailable", "Native issue creation needs an enrolled graph actor.")
		return
	}
	if _, err = s.Coordinator.GenerateAcceptedIntent(r.Context(), intent.Intent, v.User.ID); err != nil {
		factoryCommandError(w, err)
		return
	}
	generation, err := s.Store.IntentGeneration(r.Context(), intent.ID)
	if err != nil {
		factoryCommandError(w, err)
		return
	}
	auth.JSONResponse(w, 200, generation)
}

func (s *API) apiIntentGeneration(w http.ResponseWriter, r *http.Request, v store.Session) {
	intent, err := s.Store.IntentRevision(r.Context(), r.PathValue("intentID"), v.User.ID)
	if err != nil {
		factoryCommandError(w, err)
		return
	}
	generation, err := s.Store.IntentGeneration(r.Context(), intent.ID)
	if err == store.ErrNotFound {
		auth.JSONResponse(w, 200, map[string]any{"state": "pending", "revision_id": intent.ID})
		return
	}
	if err != nil {
		factoryCommandError(w, err)
		return
	}
	auth.JSONResponse(w, 200, generation)
}
