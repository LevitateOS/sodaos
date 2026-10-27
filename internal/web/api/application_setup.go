package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/levitateos/sodaos/internal/application"
	"github.com/levitateos/sodaos/internal/store"
	"github.com/levitateos/sodaos/internal/web/auth"
)

// setupClaimRefusal maps the host reconcile gate's fixed error literals to
// the existing setup codes. The host emits only fixed prefixes (the text
// after the first ';' is a static sentence, never host or stderr free
// text), so the match is on the prefix before the first ';'. It returns
// false for every other error, which keeps the existing unconfirmed
// behavior. Refusals mapped here return before any renewal-audit mutation:
// no caller may write a renewal row on a refused claim.
func setupClaimRefusal(err error) (code int, name, message string, ok bool) {
	prefix := err.Error()
	if i := strings.IndexByte(prefix, ';'); i >= 0 {
		prefix = prefix[:i]
	}
	switch prefix {
	case "setup claim is committed":
		return 409, "setup_renewal_refused_committed", "This setup already has a committed owner. Renewal keeps the existing owner credential.", true
	case "setup claim conflicts":
		return 409, "setup_renewal_refused_conflict", "This setup conflicts with a committed application claim. Renewal keeps the existing credential.", true
	case "setup claim is unconfirmed":
		return 502, "setup_unconfirmed", "Setup outcome is unconfirmed. Refresh and inspect this same setup before continuing.", true
	}
	return 0, "", "", false
}

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
	if (input.Action == "complete" || input.Action == "abort" || input.Action == "renew") && !input.Confirm {
		auth.JSONError(w, 400, "setup_confirmation_required", "Confirm completing, cancelling, or renewing this setup.")
		return
	}
	if record.Observation != nil && input.ExpectedContainer != record.Observation.Container {
		auth.JSONError(w, 409, "setup_generation_changed", "Inspect this setup and use its current generation.")
		return
	}
	request := application.SetupRequest{Setup: record, Action: input.Action, ExpectedContainer: input.ExpectedContainer}
	if input.Action != "renew" && request.Validate() != nil {
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
	if input.Action == "renew" {
		// Translate first: a malformed re-begin must never consume the
		// bounded grace window or renewal count committed below.
		probe := application.SetupRequest{Setup: record, Action: "begin", ExpectedContainer: input.ExpectedContainer}
		if probe.Validate() != nil {
			auth.JSONError(w, 400, "invalid_setup_action", "Choose a setup action with its exact generation.")
			return
		}
		// Locally provable gates first: these need no owner-claim proof and
		// keep their existing behavior. An unexpired setup needs no
		// renewal; a setup past the bounded grace window is closed.
		now := time.Now().Unix()
		if now < record.ExpiresAt {
			auth.JSONError(w, 409, "setup_renewal_unneeded", "This setup has not expired. Begin or inspect it.")
			return
		}
		if now > record.ExpiresAt+store.SetupRenewGraceSeconds {
			auth.JSONError(w, 410, "setup_renewal_closed", "Setup renewal is closed for this expired setup.")
			return
		}
		// Reconcile-before-rotation: the durable owner claim decides before
		// any token rotation or renewal-audit mutation. The check binds the
		// exact generation this renew acts on (setup identity, observed
		// generation, administrator, and application context, already pinned
		// above) against the committed application claims, under the
		// per-application mutation guard. Interim fail-closed: Soda-side
		// state alone cannot prove the durable owner claim (the native
		// receipt/owner state lives outside Soda's database), so the
		// Soda-derived `unclaimed` outcome authorizes nothing. Committed,
		// conflicting, AND unclaimed reads all keep the old token, consume
		// no renewal, and return a fixed allowlisted code; only locally
		// provable states (unexpired, malformed, confirm/session gates)
		// keep their existing behavior. Host text is never trusted here;
		// every refusal names only a fixed server-contract state. Rotation
		// returns with the queued native reconciliation seam, never here.
		claim, err := s.Store.InspectSetupRenewClaim(r.Context(), record)
		if err != nil {
			auth.JSONError(w, 503, "store_unavailable", "Application setup records are unavailable.")
			return
		}
		switch claim {
		case store.SetupRenewClaimCommitted:
			auth.JSONError(w, 409, "setup_renewal_refused_committed", "This setup already has a committed owner. Renewal keeps the existing owner credential.")
			return
		case store.SetupRenewClaimConflict:
			auth.JSONError(w, 409, "setup_renewal_refused_conflict", "This setup conflicts with a committed application claim. Renewal keeps the existing credential.")
			return
		case store.SetupRenewClaimUnclaimed:
			// Fail closed: no host call and no renewal-audit mutation follow.
			auth.JSONError(w, 409, "setup_renewal_unavailable", "This setup cannot be renewed yet: its owner claim is unconfirmed. Renewal keeps the existing credential.")
			return
		default:
			auth.JSONError(w, 502, "setup_unconfirmed", "Setup outcome is unconfirmed. Refresh and inspect this same setup before continuing.")
			return
		}
		// Every rotation-adjacent outcome returned above, so no path
		// reaches a renewal-audit mutation or host call from here. The
		// store RenewApplicationSetup primitive is retained for the queued
		// native reconciliation seam, not as a currently reachable path.
		// FJ-862 gate order: these refusals structurally precede any audit
		// write — a refused claim can never consume a renewal, and the
		// HTTP path performs no renewal-audit mutation at all.
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
		// FJ-862: the host reconcile gate's fixed literals map to
		// truthful codes (a committed or conflicting claim refuses with
		// its 409, an unconfirmed one with the existing 502) instead of
		// the blanket 502. Every other host error keeps the existing
		// unconfirmed behavior. The refusal returns here, before any
		// settlement or observation write further down this path.
		if code, name, message, ok := setupClaimRefusal(err); ok {
			auth.JSONError(w, code, name, message)
			return
		}
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
