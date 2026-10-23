package api

import (
	"net/http"
	"time"

	"github.com/levitateos/sodaos/internal/application"
	"github.com/levitateos/sodaos/internal/store"
	"github.com/levitateos/sodaos/internal/web/auth"
)

// Setup is an operator-only provisional admission. Its credential lives at the
// native host and the transient response is never stored in Soda's database.
func (s *API) apiApplicationSetups(w http.ResponseWriter, r *http.Request, v store.Session) {
	w.Header().Set("Cache-Control", "no-store")
	if !applicationNoQuery(w, r) || !s.authorizeOperator(w, r, v) {
		return
	}
	if r.Method == http.MethodGet {
		records, err := s.Store.ApplicationSetups(r.Context())
		if err != nil {
			auth.JSONError(w, 503, "store_unavailable", "Application setup records are unavailable.")
			return
		}
		if !s.checkLifecycleSession(w, r, v) {
			return
		}
		auth.JSONResponse(w, 200, struct {
			Setups []application.SetupRecord `json:"setups"`
		}{records})
		return
	}
	var input struct {
		Application     application.SetupSpec `json:"application"`
		AdministratorID string                `json:"administrator_id"`
	}
	if !auth.DecodeAPIObject(w, r, &input) {
		return
	}
	administrator, ok := auth.PositiveID(input.AdministratorID)
	if !ok || input.Application.Validate() != nil {
		auth.JSONError(w, 400, "invalid_setup", "Use admitted application references and choose its Soda administrator.")
		return
	}
	if !s.applicationOriginAllowed(input.Application.Bind("00000000-0000-0000-0000-000000000000")) {
		auth.JSONError(w, 409, "application_origin_conflict", "Application and Forgejo require distinct hostnames.")
		return
	}
	identity, err := application.NewIdentity()
	if err != nil {
		auth.JSONError(w, 503, "setup_unavailable", "Could not reserve application setup.")
		return
	}
	record := application.SetupRecord{ID: identity, Application: input.Application, AdministratorID: administrator, ExpiresAt: time.Now().Add(30 * time.Minute).Unix(), State: "pending"}
	if !s.checkLifecycleSession(w, r, v) {
		return
	}
	if err := s.Store.CreateApplicationSetup(r.Context(), record); err != nil {
		auth.JSONError(w, 409, "setup_conflict", "These application references or this address are already reserved.")
		return
	}
	auth.JSONResponse(w, 201, record)
}
func (s *API) apiApplicationSetupAction(w http.ResponseWriter, r *http.Request, v store.Session) {
	w.Header().Set("Cache-Control", "no-store")
	if !applicationNoQuery(w, r) || !s.authorizeOperator(w, r, v) {
		return
	}
	var input struct {
		Action            string `json:"action"`
		ExpectedContainer string `json:"expected_container"`
		Confirm           bool   `json:"confirm"`
	}
	if !auth.DecodeAPIObject(w, r, &input) {
		return
	}
	record, err := s.Store.ApplicationSetup(r.Context(), r.PathValue("setup"))
	if err != nil {
		auth.JSONError(w, 404, "setup_not_found", "Application setup was not found.")
		return
	}
	if record.State != "pending" {
		auth.JSONResponse(w, 200, struct {
			Setup application.SetupRecord `json:"setup"`
		}{record})
		return
	}
	if (input.Action == "complete" || input.Action == "abort") && !input.Confirm {
		auth.JSONError(w, 400, "setup_confirmation_required", "Confirm completing or cancelling this setup.")
		return
	}
	if record.Observation != nil && input.ExpectedContainer != record.Observation.Container {
		auth.JSONError(w, 409, "setup_generation_changed", "Inspect this setup and use its current generation.")
		return
	}
	request := application.SetupRequest{Setup: record, Action: input.Action, ExpectedContainer: input.ExpectedContainer}
	if request.Validate() != nil {
		auth.JSONError(w, 400, "invalid_setup_action", "Choose a setup action with its exact generation.")
		return
	}
	release, ok := s.acquireApplicationMutation(record.Application.ID)
	if !ok {
		auth.JSONError(w, 409, "application_pending", "An application operation is in progress.")
		return
	}
	defer release()
	if !s.checkLifecycleSession(w, r, v) {
		return
	}
	if s.Host == nil {
		auth.JSONError(w, 503, "native_unavailable", "Application host is unavailable.")
		return
	}
	// Unknown survives a lost native response. Reconciliation inspects only the
	// admitted provisional generation; it never infers completion from desire.
	if err = s.Store.ObserveApplicationSetup(r.Context(), record, nil); err != nil {
		auth.JSONError(w, 409, "setup_changed", "Setup state changed. Refresh before continuing.")
		return
	}
	record.Observation = nil
	ctx, cancel := operationContext(r.Context(), 4*time.Minute)
	defer cancel()
	result, err := s.Host.ApplicationSetup(ctx, request)
	if err != nil {
		auth.JSONError(w, 502, "setup_unconfirmed", "Setup outcome is unconfirmed. Refresh and inspect this same setup before continuing.")
		return
	}
	if input.Action == "complete" || input.Action == "abort" {
		err = s.Store.SettleApplicationSetup(ctx, record, result, input.Action == "abort")
	} else {
		err = s.Store.ObserveApplicationSetup(ctx, record, &result.Observation)
	}
	if err != nil {
		auth.JSONError(w, 502, "setup_unconfirmed", "Native setup was observed but could not be recorded. Inspect and reconcile this same setup.")
		return
	}
	current, err := s.Store.ApplicationSetup(ctx, record.ID)
	if err != nil {
		auth.JSONError(w, 503, "store_unavailable", "Could not reload application setup.")
		return
	}
	auth.JSONResponse(w, 200, struct {
		Setup  application.SetupRecord `json:"setup"`
		Result application.SetupResult `json:"result"`
	}{current, result})
}
