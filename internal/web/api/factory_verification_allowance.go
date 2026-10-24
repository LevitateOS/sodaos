package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/store"
	"github.com/levitateos/sodaos/internal/web/auth"
)

func verificationAllowanceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, factory.ErrVerificationAllowanceOwner):
		auth.JSONError(w, 403, "intent_owner_required", "Only this build request's owner can read or increase verification limits.")
	case errors.Is(err, store.ErrNotFound):
		auth.JSONError(w, 404, "not_found", "Build request not found.")
	case errors.Is(err, store.ErrStaleRevision):
		auth.JSONError(w, 409, "stale_revision", "Verification limits changed; refresh before submitting another increase.")
	case errors.Is(err, factory.ErrVerificationAllowanceIncrease):
		auth.JSONError(w, 409, "increase_required", "Verification limits can only increase; prior usage remains charged.")
	case errors.Is(err, factory.ErrVerificationAllowanceState):
		auth.JSONError(w, 409, "accepted_intent_required", "Accept this build request for a repository before increasing verification limits.")
	default:
		factoryCommandError(w, err)
	}
}

func (s *API) apiIntentVerificationAllowance(w http.ResponseWriter, r *http.Request, v store.Session) {
	if r.URL.RawQuery != "" || r.URL.ForceQuery {
		auth.JSONError(w, 400, "invalid_request", "No query parameters are accepted.")
		return
	}
	if r.Method == http.MethodPost && !s.checkLifecycleSession(w, r, v) {
		return
	}
	intent, now := r.PathValue("intentID"), time.Now()
	if r.Method == http.MethodGet {
		current, err := s.Store.AppVerificationAllowance(r.Context(), intent, v.User.ID, now)
		if err != nil {
			verificationAllowanceError(w, err)
			return
		}
		if current.Repository > 0 && !s.factoryOwner(w, r, v, current.Repository) {
			return
		}
		auth.JSONResponse(w, 200, current)
		return
	}
	repository, err := s.Store.AppVerificationAllowanceScope(r.Context(), intent, v.User.ID)
	if err != nil {
		verificationAllowanceError(w, err)
		return
	}
	if repository > 0 && !s.factoryOwner(w, r, v, repository) {
		return
	}
	var change factory.AppVerificationAllowanceChange
	if !auth.DecodeAPIObject(w, r, &change) {
		return
	}
	if err := change.Validate(); err != nil {
		auth.JSONError(w, 400, "invalid_verification_allowance", err.Error())
		return
	}
	raw, err := json.Marshal(change)
	if err != nil {
		verificationAllowanceError(w, err)
		return
	}
	cmd := factory.Command{ID: change.CommandID, Type: factory.CommandVerificationAllowance, Target: factory.AppVerificationAllowanceTarget(intent), Principal: factoryPrincipal(v), Payload: string(raw)}
	cmd.Digest = factory.SettingsDigest(cmd.Type, cmd.Target, cmd.Payload)
	result, err := s.Store.IncreaseAppVerificationAllowance(r.Context(), intent, v.User.ID, cmd, change, now)
	if err != nil {
		verificationAllowanceError(w, err)
		return
	}
	auth.JSONResponse(w, 200, result)
}
